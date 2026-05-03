package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunLoop_CallsReconcileOnInterval(t *testing.T) {
	var count atomic.Int32
	reconcileFn := func(_ context.Context) error {
		count.Add(1)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	runLoop(ctx, 80*time.Millisecond, reconcileFn)

	if count.Load() < 2 {
		t.Errorf("reconcile called %d times, want >= 2", count.Load())
	}
}

func TestRunLoop_StopsOnContextCancel(t *testing.T) {
	var count atomic.Int32
	reconcileFn := func(_ context.Context) error {
		count.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runLoop(ctx, 10*time.Millisecond, reconcileFn)
	// should return immediately without calling reconcile repeatedly
	if count.Load() > 1 {
		t.Errorf("reconcile called %d times after cancel, want <= 1", count.Load())
	}
}
