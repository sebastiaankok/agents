package controller

import (
	"sort"
	"time"
)

// Phase describes a Job's current lifecycle state.
type Phase string

const (
	PhaseSuspended Phase = "Suspended"
	PhaseRunning   Phase = "Running"
	PhaseSucceeded Phase = "Succeeded"
	PhaseFailed    Phase = "Failed"
)

// JobStatus describes a single Agent Job's current state.
type JobStatus struct {
	Name         string
	IssueNumber  int
	CreationTime time.Time
	Phase        Phase
}

// State is the input to Reconcile. It contains all observed Jobs and configuration.
type State struct {
	Jobs        []JobStatus
	MaxParallel int
}

// Action represents a decision made by Reconcile.
type Action interface{}

// UnsuspendJob instructs the controller to unsuspend an Agent Job by name.
type UnsuspendJob struct {
	Name string
}

// UpdateLabel instructs the controller to add one label and remove another on a GitHub Issue.
type UpdateLabel struct {
	IssueNumber int
	Add         string
	Remove      string
}

// MarkFailed instructs the controller to label a GitHub Issue as failed.
type MarkFailed struct {
	IssueNumber int
}

// Reconcile inspects current Job states and returns a list of actions.
// It is a pure function — no k8s or GitHub API calls.
func Reconcile(state State) []Action {
	running := 0
	var suspended []JobStatus
	var actions []Action

	for _, j := range state.Jobs {
		switch j.Phase {
		case PhaseSuspended:
			suspended = append(suspended, j)
		case PhaseRunning:
			running++
			if j.IssueNumber != 0 {
				actions = append(actions, UpdateLabel{
					IssueNumber: j.IssueNumber,
					Add:         "in-progress",
					Remove:      "ready-for-agent",
				})
			}
		case PhaseSucceeded:
			if j.IssueNumber != 0 {
				actions = append(actions, UpdateLabel{
					IssueNumber: j.IssueNumber,
					Add:         "waiting-for-review",
					Remove:      "in-progress",
				})
			}
		case PhaseFailed:
			if j.IssueNumber != 0 {
				actions = append(actions, MarkFailed{IssueNumber: j.IssueNumber})
			}
		}
	}

	slotsFree := state.MaxParallel - running
	if slotsFree <= 0 || len(suspended) == 0 {
		return actions
	}

	sort.Slice(suspended, func(i, j int) bool {
		return suspended[i].CreationTime.Before(suspended[j].CreationTime)
	})

	if slotsFree > len(suspended) {
		slotsFree = len(suspended)
	}

	for i := 0; i < slotsFree; i++ {
		actions = append(actions, UnsuspendJob{Name: suspended[i].Name})
	}

	return actions
}
