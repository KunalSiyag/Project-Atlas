# API and Provider Contracts

## Purpose
Hold versioned interface contracts before and alongside implementation.

## Scope
Public HTTP/API contracts and provider port contracts. JSON schemas for entities may live under `schemas/` and be referenced here.

## Dependencies
- [Documentation Standards](../docs/engineering/DOCUMENTATION_STANDARDS.md)
- [ADR 0002: Provider Independence](../docs/adr/0002-provider-independence.md)
- [ADR 0004: Communication Protocols](../docs/adr/0004-communication-protocols.md)

## Owner
Atlas Architecture

## Status
Bootstrap

## Last Updated
2026-08-03

## Related Documents
- [Schemas](../schemas/README.md)
- [Architecture Overview](../docs/architecture/OVERVIEW.md)

## Planned Contracts (M1+)
| Contract | Intent |
|----------|--------|
| [chat-provider.md](./chat-provider.md) | Model generate/stream port |
| [embedding-provider.md](./embedding-provider.md) | Embedding port |
| [tool-provider.md](./tool-provider.md) | Tool execution port / plugin shape |
| [memory-provider.md](./memory-provider.md) | Memory engine port |
| [planner.md](./planner.md) | Plan and run control plane |

## Rules
- Contracts are written before service implementation for the relevant scope.
- Breaking changes require version bumps and changelog notes.
- Adapters must satisfy contracts via tests when code exists.
