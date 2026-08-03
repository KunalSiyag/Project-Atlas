# Engineering Principles

## Purpose
Document the engineering principles that guide Project Atlas decisions.

## Scope
Applies to architecture, implementation, testing, operations, and documentation.

## Dependencies
- [Documentation Standards](./DOCUMENTATION_STANDARDS.md)

## Owner
Atlas Core Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Architecture Overview](../architecture/OVERVIEW.md)
- [ADR Index](../adr/README.md)

## Principles
- Optimize for long-term maintainability over short-term speed.
- Use clean architecture and dependency inversion where boundaries matter.
- Keep domain logic independent from external vendors.
- Define contracts before implementation.
- Prefer explicit dependencies, deterministic behavior, structured logs, metrics, and traces.
- Make decisions visible through ADRs.
- Treat documentation as a first-class deliverable.

## Design Preferences
- Small interfaces.
- Loose coupling.
- High cohesion.
- Composition over inheritance.
- Configuration over hardcoding.
- Tests alongside implementation.
