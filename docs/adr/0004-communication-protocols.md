# ADR 0004: Communication Protocols

## Purpose
Record how clients and internal services communicate in early Atlas milestones.

## Scope
External client protocols and internal service-to-service communication for M1–M3 design. Streaming UX choices included at a high level.

## Dependencies
- [Architecture: Gateway](../architecture/gateway.md)
- [ADR 0003](./0003-architecture-style.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Contracts Index](../../contracts/README.md)
- [NFR Performance](../product/NON_FUNCTIONAL_REQUIREMENTS.md)

## Context
Atlas needs a clear public API for conversations and runs, plus internal calls among gateway, orchestrator, inference, and tools. Streaming improves agent UX. Premature multi-protocol support increases cost.

## Problem
Which protocols should Atlas standardize on for public and internal communication?

## Options
1. **HTTP/JSON only** everywhere, including internal.
2. **gRPC only** public and internal.
3. **HTTP/JSON (and SSE or WebSocket) at the edge**; **gRPC or HTTP internally** with explicit versioned contracts; prefer simple HTTP between services until scale demands gRPC.

## Decision
- **Public edge:** versioned **HTTP JSON** APIs. For streaming assistant output, prefer **SSE** for simplicity in v1; evaluate **WebSocket** if bidirectional client events (cancel, tool approval from UI) need a single long-lived channel—document the choice in a follow-up ADR when implementing the client.
- **Internal:** start with **HTTP JSON** between services for debuggability; introduce **gRPC** for high-chatter internal paths when profiling justifies it (not day one).
- **Async (later):** message broker behind an interface (NATS first if needed)—not required for M0.

## Trade-offs
- **Benefit:** Easy debugging, browser-friendly public API, low early complexity.
- **Cost:** May revisit streaming and internal RPC under load.

## Consequences
- Contracts are OpenAPI-oriented (or equivalent markdown schemas) first.
- Clients must not assume WebSocket-only.
- Protocol changes require ADR updates.

## Alternatives Considered
gRPC-only public API rejected for browser and simple client friction. Broker-first architecture rejected until async requirements are proven.
