# Context: agentctl

A Go CLI tool that dispatches `sst/opencode` containers to work on GitHub Issues autonomously, running on Kubernetes.

## Glossary

### agentctl
The CLI binary. Run from a target repository directory. Infers the GitHub repo from `git remote -v`. Subcommands: `install`, `run`.

### Agent Job
A Kubernetes Job that runs the opencode runner container for a single GitHub Issue. Created in suspended state; the Controller unsuspends it when a slot is free.

### Controller
A Kubernetes Deployment (running in the `agent-runners` namespace) that owns the Agent Job lifecycle: unsuspends queued Jobs, tracks state, updates GitHub Issue labels, polls PR merge status, and sets the `failed` label on Job failure.

### opencode
`sst/opencode` — the AI coding agent that runs inside an Agent Job. Receives the issue body + `/tdd` as its prompt. Config is provided via a user-managed ConfigMap mounted at the opencode config path.

### opencode runner image
Custom Docker image (`ghcr.io/sebastiaankok/agents/opencode-runner`) bundling `git`, `gh` CLI, and `opencode`. Configurable per-run via `--job-image` flag.

### ready-for-agent
GitHub Issue label that marks an issue as eligible for automation. `agentctl run` fetches all open issues with this label.

### Blocked by
Convention in an Issue body: `Blocked by #<number>`. Controller checks if the referenced issue is still open; if open, Job stays suspended until it closes.

### agent-runners
Default Kubernetes namespace where all agentctl resources live (Controller Deployment, Agent Jobs, Secrets, ConfigMaps). Configurable via `--namespace`.

### in-progress
GitHub Issue label set by Controller when an Agent Job is unsuspended and starts running.

### waiting-for-review
GitHub Issue label set by Controller when the Agent Job completes and a PR has been opened.

### done
GitHub Issue label set by Controller when the PR is merged (auto-merge fires after CI passes).

### failed
GitHub Issue label set by Controller when an Agent Job exits non-zero. No auto-retry; user re-labels as `ready-for-agent` to retry.

## Key conventions

- Branch per issue: `agent/issue-{number}`
- PR merge strategy: squash via GitHub native auto-merge
- Job TTL after completion: 7 days (`ttlSecondsAfterFinished: 604800`)
- Default max parallel Jobs: 3 (configurable via `--max-parallel`)
- Blocked issue poll interval: 5 minutes
- Skills: agents repo cloned at `/skills` inside each Job (public repo, no auth needed)
- Credentials: user pre-creates a `agentctl-credentials` Secret; `agentctl run` validates it exists
- opencode config: user pre-creates a `opencode-config` ConfigMap with full `config.json`
