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

	prStatuses, err := fetchPRStatuses(ctx, ghClient, repo, jobs)
	if err != nil {
		return err
	}

	state := buildState(jobs, prStatuses, maxParallel)
	actions := controller.Reconcile(state)

	jobNameByIssue := make(map[int]string, len(jobs))
	for _, j := range jobs {
		if n := parseIssueNumber(j.Labels); n != 0 {
			jobNameByIssue[n] = j.Name
		}
	}

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
			if jobName := jobNameByIssue[a.IssueNumber]; jobName != "" {
				if err := client.SetJobAnnotation(ctx, namespace, jobName, "agents.io/issue-label", a.Add); err != nil {
					return fmt.Errorf("annotate job %s: %w", jobName, err)
				}
			}
		case controller.MarkFailed:
			log.Printf("marking issue %d as failed", a.IssueNumber)
			if err := ghClient.UpdateLabel(ctx, repo, a.IssueNumber, "failed", "in-progress"); err != nil {
				return fmt.Errorf("mark failed issue %d: %w", a.IssueNumber, err)
			}
			if jobName := jobNameByIssue[a.IssueNumber]; jobName != "" {
				if err := client.SetJobAnnotation(ctx, namespace, jobName, "agents.io/issue-label", "failed"); err != nil {
					return fmt.Errorf("annotate job %s: %w", jobName, err)
				}
			}
		case controller.EnableAutoMerge:
			log.Printf("enabling auto-merge on PR %d", a.PRNumber)
			if err := ghClient.EnableAutoMerge(ctx, repo, a.PRNumber); err != nil {
				return fmt.Errorf("enable auto-merge PR %d: %w", a.PRNumber, err)
			}
		case controller.SetDone:
			log.Printf("setting done on issue %d", a.IssueNumber)
			if err := ghClient.UpdateLabel(ctx, repo, a.IssueNumber, "done", "waiting-for-review"); err != nil {
				return fmt.Errorf("set done issue %d: %w", a.IssueNumber, err)
			}
			if jobName := jobNameByIssue[a.IssueNumber]; jobName != "" {
				if err := client.SetJobAnnotation(ctx, namespace, jobName, "agents.io/issue-label", "done"); err != nil {
					return fmt.Errorf("annotate job %s: %w", jobName, err)
				}
			}
		}
	}

	return nil
}

func fetchPRStatuses(ctx context.Context, ghClient *github.Client, repo string, jobs []batchv1.Job) ([]controller.PRStatus, error) {
	var statuses []controller.PRStatus
	for _, j := range jobs {
		issueNum := parseIssueNumber(j.Labels)
		if issueNum == 0 {
			continue
		}
		branch := fmt.Sprintf("agent/issue-%d", issueNum)
		pr, ok, err := ghClient.GetPRByBranch(ctx, repo, branch)
		if err != nil {
			return nil, fmt.Errorf("get PR for issue %d: %w", issueNum, err)
		}
		if !ok {
			continue
		}
		statuses = append(statuses, controller.PRStatus{
			IssueNumber: issueNum,
			PRNumber:    pr.Number,
			Merged:      pr.Merged,
		})
	}
	return statuses, nil
}

func buildState(jobs []batchv1.Job, prStatuses []controller.PRStatus, maxParallel int) controller.State {
	var statuses []controller.JobStatus
	for _, j := range jobs {
		statuses = append(statuses, controller.JobStatus{
			Name:              j.Name,
			IssueNumber:       parseIssueNumber(j.Labels),
			CreationTime:      j.CreationTimestamp.Time,
			Phase:             detectPhase(j),
			AppliedIssueLabel: j.Annotations["agents.io/issue-label"],
		})
	}
	return controller.State{
		Jobs:        statuses,
		PRs:         prStatuses,
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
