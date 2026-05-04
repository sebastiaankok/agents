package controller_test

import (
	"testing"
	"time"

	"github.com/sebastiaankok/agents/internal/controller"
)

func TestReconcile_EmptyState_NoActions(t *testing.T) {
	state := controller.State{
		Jobs:        nil,
		MaxParallel: 3,
	}

	actions := controller.Reconcile(state)

	if len(actions) != 0 {
		t.Errorf("Reconcile() returned %d actions, want 0", len(actions))
	}
}

func TestReconcile_AtLimit_NoUnsuspend(t *testing.T) {
	state := controller.State{
		Jobs: []controller.JobStatus{
			{Name: "job-a", CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Suspended: false},
			{Name: "job-b", CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Suspended: false},
			{Name: "job-c", CreationTime: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Suspended: false},
			{Name: "job-d", CreationTime: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), Suspended: true},
		},
		MaxParallel: 3,
	}

	actions := controller.Reconcile(state)

	if len(actions) != 0 {
		t.Errorf("Reconcile() returned %d actions, want 0 (at limit)", len(actions))
		for i, a := range actions {
			t.Logf("  action[%d]: %v", i, a)
		}
	}
}

func TestReconcile_BelowLimit_UnsuspendOldest(t *testing.T) {
	state := controller.State{
		Jobs: []controller.JobStatus{
			{Name: "job-a", CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Suspended: false},
			{Name: "job-b", CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Suspended: false},
			{Name: "job-c", CreationTime: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Suspended: true},
			{Name: "job-d", CreationTime: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), Suspended: true},
		},
		MaxParallel: 3,
	}

	actions := controller.Reconcile(state)

	if len(actions) != 1 {
		t.Fatalf("Reconcile() returned %d actions, want 1", len(actions))
	}

	unsuspend, ok := actions[0].(controller.UnsuspendJob)
	if !ok {
		t.Fatalf("action is %T, want UnsuspendJob", actions[0])
	}
	if unsuspend.Name != "job-c" {
		t.Errorf("UnsuspendJob.Name = %q, want %q (oldest suspended)", unsuspend.Name, "job-c")
	}
}

func TestReconcile_MultipleSlotsFree_UnsuspendMultiple(t *testing.T) {
	state := controller.State{
		Jobs: []controller.JobStatus{
			{Name: "job-a", CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Suspended: false},
			{Name: "job-b", CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Suspended: true},
			{Name: "job-c", CreationTime: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Suspended: true},
			{Name: "job-d", CreationTime: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), Suspended: true},
		},
		MaxParallel: 3,
	}

	actions := controller.Reconcile(state)

	if len(actions) != 2 {
		t.Fatalf("Reconcile() returned %d actions, want 2", len(actions))
	}

	wantNames := []string{"job-b", "job-c"}
	for i, wantName := range wantNames {
		unsuspend, ok := actions[i].(controller.UnsuspendJob)
		if !ok {
			t.Fatalf("action[%d] is %T, want UnsuspendJob", i, actions[i])
		}
		if unsuspend.Name != wantName {
			t.Errorf("action[%d].Name = %q, want %q", i, unsuspend.Name, wantName)
		}
	}
}

func TestReconcile_MaxParallelOne_UnsuspendOldest(t *testing.T) {
	state := controller.State{
		Jobs: []controller.JobStatus{
			{Name: "job-a", CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Suspended: true},
			{Name: "job-b", CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Suspended: true},
		},
		MaxParallel: 1,
	}

	actions := controller.Reconcile(state)

	if len(actions) != 1 {
		t.Fatalf("Reconcile() returned %d actions, want 1", len(actions))
	}

	unsuspend, ok := actions[0].(controller.UnsuspendJob)
	if !ok {
		t.Fatalf("action is %T, want UnsuspendJob", actions[0])
	}
	if unsuspend.Name != "job-a" {
		t.Errorf("UnsuspendJob.Name = %q, want %q (oldest suspended)", unsuspend.Name, "job-a")
	}
}
