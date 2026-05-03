package main

import (
	"context"
	"log"
	"time"
)

type reconcileFunc func(ctx context.Context) error

func runLoop(ctx context.Context, interval time.Duration, reconcile reconcileFunc) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Println("reconcile: start")
			if err := reconcile(ctx); err != nil {
				log.Printf("reconcile: error: %v", err)
			} else {
				log.Println("reconcile: done")
			}
		}
	}
}
