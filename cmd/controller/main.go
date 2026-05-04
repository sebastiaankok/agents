package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sebastiaankok/agents/internal/controller"
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
	flag.Parse()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("failed to load in-cluster config: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %v", err)
	}

	client := k8s.NewClient(kubeClient)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("controller starting, interval=%s, namespace=%s, max-parallel=%d", *interval, *namespace, *maxParallel)
	runLoop(ctx, *interval, func(ctx context.Context) error {
		return reconcile(ctx, client, *namespace, *maxParallel)
	})
	log.Println("controller stopped")
}

func reconcile(ctx context.Context, client *k8s.Client, namespace string, maxParallel int) error {
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
		}
	}

	return nil
}

func buildState(jobs []batchv1.Job, maxParallel int) controller.State {
	var statuses []controller.JobStatus
	for _, j := range jobs {
		statuses = append(statuses, controller.JobStatus{
			Name:         j.Name,
			CreationTime: j.CreationTimestamp.Time,
			Suspended:    j.Spec.Suspend != nil && *j.Spec.Suspend,
		})
	}
	return controller.State{
		Jobs:        statuses,
		MaxParallel: maxParallel,
	}
}
