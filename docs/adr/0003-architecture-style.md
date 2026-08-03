# ADR 0003: Architecture Style (Modular Hexagonal Services)

## Purpose
Record the chosen application architecture style for Atlas services.

## Scope
Service internal structure and inter-service modularity in the monorepo.

## Dependencies
- [Engineering Principles](../engineering/PRINCIPLES.md)
- [Architecture Overview](../architecture/OVERVIEW.md)
- [ADR 0001](./0001-documentation-first-modular-monorepo.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [ADR 0002: Provider Independence](./0002-provider-independence.md)
- [Documentation Standards](../engineering/DOCUMENTATION_STANDARDS.md)

## Context
Agent platforms sprawl: gateway, planner, tools, memory, eval. Without boundaries, vendor SDKs and transport details leak into domain logic.

## Problem
Which structural style keeps Atlas maintainable for humans and AI assistants while allowing independent evolution of subsystems?

## Options
1. **Modular monolith only** with no ports—packages call each other freely.
2. **Microservices-first** with many deployables from day one.
3. **Hexagonal / clean architecture** within a modular monorepo: domain and use cases at center; adapters at edges; services may start co-located and split later.

## Decision
Adopt **hexagonal (ports and adapters) modular architecture** inside a documentation-first monorepo. Services are logical modules with local docs; physical process split follows operational need (Compose services when implemented). Domain does not depend on frameworks or vendors.

## Trade-offs
- **Benefit:** Testability, provider swaps, clear ownership, AI-loadable service packs.
- **Cost:** More files and indirection than a script-style agent.

## Consequences
- New features land as use cases + ports first; adapters second.
- Service-local PRD/API/data model docs precede implementation.
- Cross-service calls use explicit contracts (ADR 0004).

## Alternatives Considered
Unstructured modular monolith rejected for vendor bleed. Microservices-first rejected as operational overhead before product fundamentals exist.
