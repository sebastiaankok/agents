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
)

func main() {
	defaultInterval := 30 * time.Second
	if v := os.Getenv("RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			defaultInterval = d
		}
	}

	interval := flag.Duration("interval", defaultInterval, "reconcile loop interval")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("controller starting, interval=%s", *interval)
	runLoop(ctx, *interval, controller.Reconcile)
	log.Println("controller stopped")
}
