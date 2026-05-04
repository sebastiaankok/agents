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

// IssueState holds the open/closed status of a blocking GitHub issue.
type IssueState struct {
	Number int
	Open   bool
}

// JobStatus describes a single Agent Job's current state.
type JobStatus struct {
	Name         string
	IssueNumber  int
	CreationTime time.Time
	Phase        Phase
	BlockedBy    []int // issue numbers from "Blocked by #N" lines
}

// State is the input to Reconcile. It contains all observed Jobs, PRs, and configuration.
type State struct {
	Jobs           []JobStatus
	PRs            []PRStatus
	MaxParallel    int
	BlockingIssues []IssueState // open/closed status for all issues referenced in BlockedBy fields
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

// EnableAutoMerge instructs the controller to enable squash auto-merge on a PR.
type EnableAutoMerge struct {
	PRNumber int
}

// SetDone instructs the controller to move a GitHub Issue to done state.
type SetDone struct {
	IssueNumber int
}

// PRStatus describes the observed state of a pull request associated with an issue.
type PRStatus struct {
	IssueNumber int
	PRNumber    int
	Merged      bool
}

// Reconcile inspects current Job states and returns a list of actions.
// It is a pure function — no k8s or GitHub API calls.
func Reconcile(state State) []Action {
	running := 0
	var suspended []JobStatus
	var actions []Action

	prByIssue := make(map[int]PRStatus, len(state.PRs))
	for _, pr := range state.PRs {
		prByIssue[pr.IssueNumber] = pr
	}

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
				if pr, ok := prByIssue[j.IssueNumber]; ok && !pr.Merged {
					actions = append(actions, EnableAutoMerge{PRNumber: pr.PRNumber})
				}
			}
		case PhaseFailed:
			if j.IssueNumber != 0 {
				actions = append(actions, MarkFailed{IssueNumber: j.IssueNumber})
			}
		}
	}

	for _, pr := range state.PRs {
		if pr.Merged {
			actions = append(actions, SetDone{IssueNumber: pr.IssueNumber})
		}
	}

	openIssues := make(map[int]bool, len(state.BlockingIssues))
	for _, s := range state.BlockingIssues {
		if s.Open {
			openIssues[s.Number] = true
		}
	}

	var eligible []JobStatus
	for _, j := range suspended {
		blocked := false
		for _, n := range j.BlockedBy {
			if openIssues[n] {
				blocked = true
				break
			}
		}
		if !blocked {
			eligible = append(eligible, j)
		}
	}

	slotsFree := state.MaxParallel - running
	if slotsFree <= 0 || len(eligible) == 0 {
		return actions
	}

	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].CreationTime.Before(eligible[j].CreationTime)
	})

	if slotsFree > len(eligible) {
		slotsFree = len(eligible)
	}

	for i := 0; i < slotsFree; i++ {
		actions = append(actions, UnsuspendJob{Name: eligible[i].Name})
	}

	return actions
}
