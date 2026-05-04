package main

import (
	"context"
	"flag"
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
	repo := flag.String("repo", os.Getenv("GITHUB_REPO"), "github repository (owner/name)")
	flag.Parse()

	if *repo == "" {
		log.Fatal("--repo (or GITHUB_REPO env var) is required")
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("failed to load in-cluster config: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %v", err)
	}

	k8sClient := k8s.NewClient(kubeClient)
	ghClient := github.NewClient("")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("controller starting, interval=%s, namespace=%s, max-parallel=%d, repo=%s", *interval, *namespace, *maxParallel, *repo)
	runLoop(ctx, *interval, func(ctx context.Context) error {
		return reconcile(ctx, k8sClient, ghClient, *repo, *namespace, *maxParallel)
	})
	log.Println("controller stopped")
}

func reconcile(ctx context.Context, k8sClient *k8s.Client, ghClient *github.Client, repo, namespace string, maxParallel int) error {
	jobs, err := k8sClient.ListJobs(ctx, namespace, "app=agent-job")
	if err != nil {
		return err
	}

	state := buildState(ctx, ghClient, repo, jobs, maxParallel)
	actions := controller.Reconcile(state)

	for _, action := range actions {
		switch a := action.(type) {
		case controller.UnsuspendJob:
			log.Printf("unsuspending job %q", a.Name)
			if err := k8sClient.UnsuspendJob(ctx, namespace, a.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildState(ctx context.Context, ghClient *github.Client, repo string, jobs []batchv1.Job, maxParallel int) controller.State {
	var statuses []controller.JobStatus
	for _, j := range jobs {
		suspended := j.Spec.Suspend != nil && *j.Spec.Suspend
		js := controller.JobStatus{
			Name:         j.Name,
			CreationTime: j.CreationTimestamp.Time,
			Suspended:    suspended,
		}
		if suspended {
			js.BlockingIssues = fetchBlockingIssues(ctx, ghClient, repo, j)
		}
		statuses = append(statuses, js)
	}
	return controller.State{
		Jobs:        statuses,
		MaxParallel: maxParallel,
	}
}

func fetchBlockingIssues(ctx context.Context, ghClient *github.Client, repo string, job batchv1.Job) []controller.BlockingIssue {
	issueNumberStr, ok := job.Labels["issue-number"]
	if !ok {
		return nil
	}
	issueNumber, err := strconv.Atoi(issueNumberStr)
	if err != nil {
		return nil
	}

	issue, err := ghClient.GetIssue(ctx, repo, issueNumber)
	if err != nil {
		log.Printf("warning: failed to fetch issue %d for job %s: %v", issueNumber, job.Name, err)
		return nil
	}

	blockedBy := github.ParseBlockedBy(issue.Body)
	if len(blockedBy) == 0 {
		return nil
	}

	var blocking []controller.BlockingIssue
	for _, n := range blockedBy {
		closed, err := ghClient.IsIssueClosed(ctx, repo, n)
		if err != nil {
			log.Printf("warning: failed to check blocking issue #%d: %v", n, err)
			continue
		}
		blocking = append(blocking, controller.BlockingIssue{
			Number: n,
			Closed: closed,
		})
	}
	return blocking
}
