# ADR-0002: Suspended Jobs + Controller for concurrency

## Status
Accepted

## Context
agentctl is fire-and-forget: CLI exits after creating Jobs. Something must enforce max parallel concurrency and manage Job lifecycle without the CLI present.

Alternatives considered:
- Init container polling k8s Job count (race condition at low concurrency)
- Namespace ResourceQuota (Jobs fail rather than queue)
- CLI-side counting (requires CLI to stay running)

## Decision
All Agent Jobs are created with `spec.suspend: true`. A Controller Deployment in the `agent-runners` namespace watches for suspended Jobs and unsuspends them FIFO when the running Job count is below `--max-parallel` (default 3).

## Reasons
- No race conditions: single controller serialises unsuspend decisions
- CLI is truly fire-and-forget
- Controller already needed for label updates and PR polling — no extra component

## Consequences
- Controller Deployment must be running before Jobs are created (`agentctl install` handles this)
- Controller needs RBAC to get/list/watch/update Jobs and call GitHub API
