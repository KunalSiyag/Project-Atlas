# Project Atlas

## Purpose
Project Atlas is an enterprise-grade AI Agent Platform for building, orchestrating, evaluating, observing, and deploying production AI agents.

## Scope
This repository is the source of truth for Atlas product documentation, architecture, API contracts, data models, operational guidance, and implementation.

## Dependencies
- Git for change history and review.
- Markdown for documentation-first collaboration.
- Future services will declare their own runtime dependencies in service-local documentation.

## Owner
Atlas Core Engineering

## Status
Bootstrap

## Last Updated
2026-08-03

## Related Documents
- [AI Context](./AI_CONTEXT.md)
- [Engineering Principles](./docs/engineering/PRINCIPLES.md)
- [Documentation Standards](./docs/engineering/DOCUMENTATION_STANDARDS.md)
- [Architecture Overview](./docs/architecture/OVERVIEW.md)
- [Product Vision](./docs/product/VISION.md)
- [ADR Index](./docs/adr/README.md)
- [RFC Process](./docs/rfcs/README.md)
- [Agent Orchestrator Service](./services/agent-orchestrator/README.md)

## Mission
Atlas is not another chatbot. It is a modular platform for production agent systems with explicit contracts, observable execution, testable components, and vendor-independent provider adapters.

## Development Order
All meaningful implementation follows this sequence:

1. Requirements
2. Product Design
3. Architecture
4. Architecture Decision Record
5. API Contracts
6. Data Models
7. Tasks
8. Tests
9. Implementation
10. Documentation Update

If prerequisite documentation is missing, write or update the documentation before implementation.

## Repository Map

```text
docs/
  adr/              Architecture decision records
  architecture/     System architecture and diagrams
  benchmarks/       Benchmark plans and results
  engineering/      Engineering standards and workflows
  kb/               Knowledge base notes
  playbooks/        Repeatable engineering procedures
  product/          Product strategy and requirements
  rfcs/             Request for comments process and proposals
  runbooks/         Operational response guides
services/
  agent-orchestrator/  First service documentation shell
```

## Current Bootstrap Deliverables
- Repository-level documentation standards.
- ADR and RFC processes with templates.
- Architecture principles emphasizing clean architecture, dependency inversion, and vendor independence.
- Service-level documentation skeleton for the Agent Orchestrator.
