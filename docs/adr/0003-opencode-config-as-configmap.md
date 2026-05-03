# ADR-0003: opencode provider config as user-managed ConfigMap

## Status
Accepted

## Context
opencode supports multiple LLM providers (Anthropic, OpenRouter, LM Studio, etc.), each with different API keys, env var names, and config schema. An abstraction layer in agentctl would need to track opencode's config schema and every provider's requirements.

## Decision
Users pre-create a ConfigMap (`opencode-config`) containing their full `opencode config.json` verbatim. agentctl mounts it into each Job at the opencode config path. agentctl validates the ConfigMap exists at startup and prints an example if missing.

## Reasons
- Zero abstraction lag: any provider opencode supports works immediately
- Users already know their opencode config from local usage
- No translation layer to maintain as opencode adds providers

## Consequences
- Users must understand opencode's config format
- agentctl prints a helpful example (covering Anthropic, OpenRouter, LM Studio) when ConfigMap is absent
