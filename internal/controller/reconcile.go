package controller

import (
	"sort"
	"time"
)

// BlockingIssue represents a GitHub issue referenced by "Blocked by #N".
type BlockingIssue struct {
	Number int
	Closed bool
}

// JobStatus describes a single Agent Job's current state.
type JobStatus struct {
	Name           string
	CreationTime   time.Time
	Suspended      bool
	BlockingIssues []BlockingIssue
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

// Reconcile inspects current Job states and returns a list of actions.
// It is a pure function — no k8s or GitHub API calls.
func Reconcile(state State) []Action {
	running := 0
	var suspended []JobStatus

	for _, j := range state.Jobs {
		if j.Suspended && !jobHasOpenBlocker(j) {
			suspended = append(suspended, j)
		} else if !j.Suspended {
			running++
		}
	}

	slotsFree := state.MaxParallel - running
	if slotsFree <= 0 || len(suspended) == 0 {
		return nil
	}

	sort.Slice(suspended, func(i, j int) bool {
		return suspended[i].CreationTime.Before(suspended[j].CreationTime)
	})

	if slotsFree > len(suspended) {
		slotsFree = len(suspended)
	}

	actions := make([]Action, 0, slotsFree)
	for i := 0; i < slotsFree; i++ {
		actions = append(actions, UnsuspendJob{Name: suspended[i].Name})
	}

	return actions
}

// jobHasOpenBlocker returns true if any of the job's blocking issues are still open.
func jobHasOpenBlocker(j JobStatus) bool {
	for _, b := range j.BlockingIssues {
		if !b.Closed {
			return true
		}
	}
	return false
}
