# ADR-0001: Kubernetes-only runtime (no local Docker)

## Status
Accepted

## Context
agentctl needs to run opencode containers. The obvious default is local Docker — no cluster setup required. An alternative is Kubernetes with Kind for local development.

## Decision
Kubernetes only. No local Docker support.

## Reasons
- Suspended Jobs + Controller pattern requires a control loop that k8s provides natively
- Parallelism control, Job TTL, and RBAC are k8s primitives — reimplementing them for Docker adds scope
- Consistent environment: local (Kind) and remote clusters behave identically
- Volume mounts don't work uniformly across local Docker and k8s; git clone approach requires no mounts

## Consequences
- Users must set up a k8s cluster (Kind works locally)
- `agentctl install` bootstraps all required cluster resources
- No Podman support in v1 (planned later)
