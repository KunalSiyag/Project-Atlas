# Inference Gateway AI Context

## Purpose

Give contributors the smallest safe context needed to work on the inference gateway.

## Scope

Applies to design and implementation of model and embedding invocation in this service.

## Dependencies

- [Repository AI Context](../../AI_CONTEXT.md)
- [Service README](./README.md)
- [Service PRD](./PRD.md)
- [Service API](./API.md)
- [Service Test Plan](./TEST_PLAN.md)

## Owner

Atlas Core Engineering

## Status

Planning

## Last Updated

2026-08-03

## Related Documents

- [Inference Architecture](../../docs/architecture/inference.md)
- [Chat Provider Contract](../../contracts/chat-provider.md)
- [Embedding Provider Contract](../../contracts/embedding-provider.md)

## Non-negotiable constraints

- Domain and gateway logic must not import a vendor SDK directly.
- Providers are selected by configuration and logical model/task class, not scattered code branches.
- Local/fake providers must support development and tests without commercial credentials.
- Provider failures are classified, bounded, observable, and returned without leaking secrets.
- Usage, provider, model, latency, and correlation metadata are recorded when available.

## Recommended load path

1. `README.md`
2. `PRD.md`
3. `API.md`
4. `DATA_MODEL.md`
5. `TEST_PLAN.md`
6. Repository inference architecture and provider contracts
