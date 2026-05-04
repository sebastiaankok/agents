package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sebastiaankok/agents/internal/controller"
	"github.com/sebastiaankok/agents/internal/github"
	"github.com/sebastiaankok/agents/internal/k8s"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	defaultInterval := 30 * time.Second
	if v := os.Getenv("RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			defaultInterval = d
		}
	}

	interval := flag.Duration("interval", defaultInterval, "reconcile loop interval")
	namespace := flag.String("namespace", "agent-runners", "kubernetes namespace for agent jobs")
	maxParallel := flag.Int("max-parallel", 3, "maximum number of concurrently running agent jobs")
	repo := flag.String("repo", "", "GitHub repository in owner/name format (required)")
	flag.Parse()

	if *repo == "" {
		log.Fatal("--repo is required")
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("failed to load in-cluster config: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %v", err)
	}

	client := k8s.NewClient(kubeClient)
	ghClient := github.NewClient("")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("controller starting, interval=%s, namespace=%s, max-parallel=%d, repo=%s", *interval, *namespace, *maxParallel, *repo)
	runLoop(ctx, *interval, func(ctx context.Context) error {
		return reconcile(ctx, client, ghClient, *namespace, *maxParallel, *repo)
	})
	log.Println("controller stopped")
}

func reconcile(ctx context.Context, client *k8s.Client, ghClient *github.Client, namespace string, maxParallel int, repo string) error {
	jobs, err := client.ListJobs(ctx, namespace, "app=agent-job")
	if err != nil {
		return err
	}

	state := buildState(jobs, maxParallel)
	actions := controller.Reconcile(state)

	for _, action := range actions {
		switch a := action.(type) {
		case controller.UnsuspendJob:
			log.Printf("unsuspending job %q", a.Name)
			if err := client.UnsuspendJob(ctx, namespace, a.Name); err != nil {
				return err
			}
		case controller.UpdateLabel:
			log.Printf("updating label on issue %d: add=%q remove=%q", a.IssueNumber, a.Add, a.Remove)
			if err := ghClient.UpdateLabel(ctx, repo, a.IssueNumber, a.Add, a.Remove); err != nil {
				return fmt.Errorf("update label issue %d: %w", a.IssueNumber, err)
			}
		case controller.MarkFailed:
			log.Printf("marking issue %d as failed", a.IssueNumber)
			if err := ghClient.UpdateLabel(ctx, repo, a.IssueNumber, "failed", "in-progress"); err != nil {
				return fmt.Errorf("mark failed issue %d: %w", a.IssueNumber, err)
			}
		}
	}

	return nil
}

func buildState(jobs []batchv1.Job, maxParallel int) controller.State {
	var statuses []controller.JobStatus
	for _, j := range jobs {
		statuses = append(statuses, controller.JobStatus{
			Name:         j.Name,
			IssueNumber:  parseIssueNumber(j.Labels),
			CreationTime: j.CreationTimestamp.Time,
			Phase:        detectPhase(j),
		})
	}
	return controller.State{
		Jobs:        statuses,
		MaxParallel: maxParallel,
	}
}

func parseIssueNumber(labels map[string]string) int {
	n, _ := strconv.Atoi(labels["issue-number"])
	return n
}

func detectPhase(j batchv1.Job) controller.Phase {
	if j.Spec.Suspend != nil && *j.Spec.Suspend {
		return controller.PhaseSuspended
	}
	for _, c := range j.Status.Conditions {
		if c.Status != corev1.ConditionTrue {
			continue
		}
		switch c.Type {
		case batchv1.JobComplete:
			return controller.PhaseSucceeded
		case batchv1.JobFailed:
			return controller.PhaseFailed
		}
	}
	return controller.PhaseRunning
}
