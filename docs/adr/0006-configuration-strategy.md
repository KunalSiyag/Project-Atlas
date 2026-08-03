# ADR 0006: Configuration Strategy

## Purpose
Record how Atlas configures services, providers, and feature flags without code changes.

## Scope
Application configuration and secrets separation for all services.

## Dependencies
- [ADR 0002: Provider Independence](./0002-provider-independence.md)
- [Architecture: Deployment](../architecture/deployment.md)
- [Engineering Principles](../engineering/PRINCIPLES.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Inference Architecture](../architecture/inference.md)
- [Security Architecture](../architecture/security.md)

## Context
Provider selection, model IDs, timeouts, and feature flags must change across environments (local, CI, prod) without rebuilding images or editing domain code.

## Problem
How should configuration and secrets be supplied to Atlas services?

## Options
1. **Hard-coded constants** in source.
2. **Single giant config repo** with custom DSL only.
3. **12-factor style:** environment variables + optional layered config files; secrets from env/secret manager; documented keys per service.

## Decision
Adopt **environment-first configuration** with optional file overlays:
- Non-secret config: env vars and/or versioned example files (e.g. `config.example.yaml`) committed to the repo.
- Secrets: environment or secret manager references—never committed.
- Provider/model selection is configuration, not code branches scattered in domain logic.
- Each service documents its keys in service-local docs when implementation exists.

## Trade-offs
- **Benefit:** Portable containers; safe defaults; easy local overrides.
- **Cost:** Requires disciplined documentation of keys; env sprawl if unchecked.

## Consequences
- `.env` is gitignored; `.env.example` lists names only.
- Feature flags for optional judges, tools, and streaming.
- Invalid config fails fast at process start (readiness).

## Alternatives Considered
Hard-coding rejected. Proprietary DSL-only config rejected as premature until multiple services exist.
