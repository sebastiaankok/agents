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
			{Name: "job-a", CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseRunning},
			{Name: "job-b", CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseRunning},
			{Name: "job-c", CreationTime: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseRunning},
			{Name: "job-d", CreationTime: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
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
			{Name: "job-a", CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseRunning},
			{Name: "job-b", CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseRunning},
			{Name: "job-c", CreationTime: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
			{Name: "job-d", CreationTime: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
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
			{Name: "job-a", CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseRunning},
			{Name: "job-b", CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
			{Name: "job-c", CreationTime: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
			{Name: "job-d", CreationTime: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
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
			{Name: "job-a", CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
			{Name: "job-b", CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
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

func TestReconcile_MixedPhases_AllCorrectActions(t *testing.T) {
	state := controller.State{
		Jobs: []controller.JobStatus{
			{Name: "agent-issue-1", IssueNumber: 1, CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseRunning},
			{Name: "agent-issue-2", IssueNumber: 2, CreationTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSucceeded},
			{Name: "agent-issue-3", IssueNumber: 3, CreationTime: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseFailed},
			{Name: "agent-issue-4", IssueNumber: 4, CreationTime: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSuspended},
		},
		MaxParallel: 2,
	}

	actions := controller.Reconcile(state)

	// Expect: UpdateLabel(1, in-progress), UpdateLabel(2, waiting-for-review), MarkFailed(3), UnsuspendJob(4)
	if len(actions) != 4 {
		t.Fatalf("Reconcile() returned %d actions, want 4: %v", len(actions), actions)
	}

	ul1, ok := actions[0].(controller.UpdateLabel)
	if !ok || ul1.IssueNumber != 1 || ul1.Add != "in-progress" {
		t.Errorf("action[0] = %v, want UpdateLabel{1, in-progress, ready-for-agent}", actions[0])
	}
	ul2, ok := actions[1].(controller.UpdateLabel)
	if !ok || ul2.IssueNumber != 2 || ul2.Add != "waiting-for-review" {
		t.Errorf("action[1] = %v, want UpdateLabel{2, waiting-for-review, in-progress}", actions[1])
	}
	mf, ok := actions[2].(controller.MarkFailed)
	if !ok || mf.IssueNumber != 3 {
		t.Errorf("action[2] = %v, want MarkFailed{3}", actions[2])
	}
	us, ok := actions[3].(controller.UnsuspendJob)
	if !ok || us.Name != "agent-issue-4" {
		t.Errorf("action[3] = %v, want UnsuspendJob{agent-issue-4}", actions[3])
	}
}

func TestReconcile_FailedJob_EmitsMarkFailed(t *testing.T) {
	state := controller.State{
		Jobs: []controller.JobStatus{
			{Name: "agent-issue-5", IssueNumber: 5, CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseFailed},
		},
		MaxParallel: 3,
	}

	actions := controller.Reconcile(state)

	if len(actions) != 1 {
		t.Fatalf("Reconcile() returned %d actions, want 1", len(actions))
	}
	mf, ok := actions[0].(controller.MarkFailed)
	if !ok {
		t.Fatalf("action is %T, want MarkFailed", actions[0])
	}
	if mf.IssueNumber != 5 {
		t.Errorf("MarkFailed.IssueNumber = %d, want 5", mf.IssueNumber)
	}
}

func TestReconcile_SucceededJob_EmitsUpdateLabel(t *testing.T) {
	state := controller.State{
		Jobs: []controller.JobStatus{
			{Name: "agent-issue-7", IssueNumber: 7, CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseSucceeded},
		},
		MaxParallel: 3,
	}

	actions := controller.Reconcile(state)

	if len(actions) != 1 {
		t.Fatalf("Reconcile() returned %d actions, want 1", len(actions))
	}
	ul, ok := actions[0].(controller.UpdateLabel)
	if !ok {
		t.Fatalf("action is %T, want UpdateLabel", actions[0])
	}
	if ul.IssueNumber != 7 {
		t.Errorf("UpdateLabel.IssueNumber = %d, want 7", ul.IssueNumber)
	}
	if ul.Add != "waiting-for-review" {
		t.Errorf("UpdateLabel.Add = %q, want %q", ul.Add, "waiting-for-review")
	}
	if ul.Remove != "in-progress" {
		t.Errorf("UpdateLabel.Remove = %q, want %q", ul.Remove, "in-progress")
	}
}

func TestReconcile_RunningJob_EmitsUpdateLabel(t *testing.T) {
	state := controller.State{
		Jobs: []controller.JobStatus{
			{Name: "agent-issue-42", IssueNumber: 42, CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Phase: controller.PhaseRunning},
		},
		MaxParallel: 3,
	}

	actions := controller.Reconcile(state)

	if len(actions) != 1 {
		t.Fatalf("Reconcile() returned %d actions, want 1", len(actions))
	}
	ul, ok := actions[0].(controller.UpdateLabel)
	if !ok {
		t.Fatalf("action is %T, want UpdateLabel", actions[0])
	}
	if ul.IssueNumber != 42 {
		t.Errorf("UpdateLabel.IssueNumber = %d, want 42", ul.IssueNumber)
	}
	if ul.Add != "in-progress" {
		t.Errorf("UpdateLabel.Add = %q, want %q", ul.Add, "in-progress")
	}
	if ul.Remove != "ready-for-agent" {
		t.Errorf("UpdateLabel.Remove = %q, want %q", ul.Remove, "ready-for-agent")
	}
}
