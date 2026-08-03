# AI Context

## Purpose
Provide future AI assistants with the minimum durable context required to work safely in Project Atlas.

## Scope
Applies to repository-wide planning, documentation, architecture, and implementation assistance.

## Dependencies
- [README](./README.md)
- [Engineering Principles](./docs/engineering/PRINCIPLES.md)
- [Documentation Standards](./docs/engineering/DOCUMENTATION_STANDARDS.md)

## Owner
Atlas Core Engineering

## Status
Bootstrap

## Last Updated
2026-08-03

## Related Documents
- [Architecture Overview](./docs/architecture/OVERVIEW.md)
- [RFC Process](./docs/rfcs/README.md)
- [ADR Index](./docs/adr/README.md)

## Operating Model
Atlas is documentation-first. Do not implement features until requirements, product design, architecture, ADRs, API contracts, data models, tasks, and test plans exist for the relevant scope.

## Core Constraints
- Keep business logic independent from vendors.
- Prefer small documents over large context-heavy files.
- Use service-local documentation for service-specific decisions.
- Do not make silent architectural decisions.
- Update documentation with every meaningful change.

## Recommended Context Loading
For repository-wide work, read only:
1. `README.md`
2. `AI_CONTEXT.md`
3. `docs/engineering/PRINCIPLES.md`
4. The specific service or document being changed

For service work, read only:
1. `services/<service>/AI_CONTEXT.md`
2. `services/<service>/PRD.md`
3. `services/<service>/API.md`
4. `services/<service>/README.md`
