# Project Atlas — Application Overview

## Purpose

This document is the single-file overview of what Project Atlas is, how it is intended to work, what exists in the repository today, and what remains to be implemented.

## Repository status

Atlas is currently a documentation-first foundation for an enterprise-oriented AI agent platform. The repository does not yet contain a complete runnable platform, a live model integration, a Docker Compose runtime, or Kubernetes deployment manifests. A minimal Go implementation slice now exists for the inference gateway; its M0 milestone was explicitly complete without production binaries.

The current repository is therefore the product and architecture source of truth rather than a working chatbot. The first planned runtime service is an inference gateway, followed by the orchestration and intelligence paths.

## What the application is

Atlas is a platform for building, running, evaluating, observing, and deploying production AI agents. It is designed for platform engineers, agent/application developers, operators, evaluators, security reviewers, and humans who approve sensitive actions.

Atlas is not a single-purpose chatbot and is not defined by one model vendor or agent framework. Chat is one client surface. The platform around the model provides:

- Authentication, authorization, and workspace isolation.
- Durable conversations, turns, sessions, and run history.
- Context engineering that selects and budgets history, memory, documents, tool results, preferences, and prompts.
- Planning and workflow execution, including DAGs, parallel steps, retries, cancellation, and human approval gates.
- A common tool runtime for integrations such as GitHub, Slack, SQL, browser, filesystem, REST, and email.
- Model routing for planning, acting, coding, vision, embedding, and evaluation tasks.
- Short-term and long-term memory plus knowledge/RAG capabilities.
- Guardrails for permissions, prompt injection, secrets, dangerous actions, and output policy.
- Deterministic evaluation, optional model judges, regression comparison, cost tracking, and human feedback.
- Structured logs, metrics, and distributed traces across the complete agent run.

The central product idea is that the LLM is a replaceable dependency. The application is the reliable coordination, policy, state, and operational system around it.

## Typical request flow

The intended end-to-end path is:

```text
Client
  → API Gateway and authentication
  → Workspace/conversation resolution
  → Context Builder and Memory/Knowledge retrieval
  → Planner
  → Workflow Engine / Agent Orchestrator
  → Model Router and Inference Gateway
  → Tool Runtime and/or Knowledge Base
  → Guardrails and response generation
  → Evaluation and Observability
  → Response, stream, or run status to client
```

1. A web, desktop, CLI, automation client, or future SDK sends a versioned request.
2. The gateway validates the request, authenticates the principal, resolves the workspace, checks membership/roles, applies rate limits, and attaches correlation identifiers.
3. The conversation layer loads or creates the conversation and records the user turn.
4. The context builder assembles a bounded model context. It ranks sources, estimates tokens, records an inclusion manifest, and reports exclusions or degraded retrieval.
5. The planner decides whether the request needs a direct model response or a structured multi-step plan. A plan is data: steps, dependencies, inputs, and policies—not merely prose.
6. The workflow engine persists a run and executes independent DAG steps in parallel where possible. Steps can be model, tool, approval, or parallel steps and have inspectable states, retries, timeouts, and errors.
7. The model router resolves a logical task class to a configured provider/model. The inference gateway presents a common chat and embedding interface and records provider, model, usage, and latency.
8. The tool runtime discovers registered tools, validates schemas, checks authorization and policy, executes with timeout/retry controls, and emits an auditable result. Side-effecting actions may pause for human approval.
9. The response path applies output guardrails, formats text/structured output/citations, records the assistant turn, and returns a response or run-progress view.
10. Evaluation and observability capture quality, cost, latency, tool success, traces, metrics, and structured logs for the run.

## Core domain model

The main tenant boundary is a **Workspace**. A workspace owns users, permissions, agents, conversations, memories, and knowledge. A **Conversation** contains durable user, assistant, system, and tool turns. An **Agent** is a configured reasoning actor that uses prompts, memory, tools, providers, and policies.

A **Workflow** is a versioned definition of executable steps and execution policy. A **Run** or execution is one attempt to execute that workflow. A run contains steps and immutable lifecycle events and may be pending, running, succeeded, failed, cancelled, or waiting for approval.

Other important entities are:

- **Prompt:** a versioned instruction artifact with metadata and change history.
- **Memory:** scoped, potentially expiring facts or working state for a user, workspace, or conversation.
- **Tool:** a named, schema-defined capability with validation, authorization, timeout, retry, and side-effect semantics.
- **Provider:** an adapter-backed external capability such as chat inference, embeddings, vector storage, object storage, or messaging.
- **Evaluation:** measurements and reports over a run, dataset, provider, prompt, or benchmark.

## Architecture and boundaries

Atlas uses modular services with hexagonal ports-and-adapters boundaries:

- Domain logic owns use cases and business rules.
- Ports define capabilities such as model generation, embeddings, memory, tools, repositories, object storage, and vectors.
- Adapters integrate vendors and local implementations.
- Public contracts, schemas, ADRs, and tests are established before service implementation.

The principal planned subsystems are the gateway, conversation/session service, context builder, memory engine, planner, workflow engine/agent orchestrator, tool runtime, model router, inference gateway, response generator, guardrails, evaluation layer, and observability layer.

The first documented service is `services/agent-orchestrator`. It will own planning, execution state transitions, tool coordination, and execution events while remaining independent of model, tool, storage, and messaging vendors. It is currently documentation-only.

## Providers and infrastructure

Provider selection is configuration-driven and replaceable. Planned model adapters include local Ollama and optional hosted providers such as OpenAI, Anthropic, and Gemini. Embeddings can use local Sentence Transformers/BGE/Nomic-style implementations or an optional hosted adapter.

The intended local-first data plane is polyglot:

| Concern | Development default | Responsibility |
|---|---|---|
| Durable relational state | PostgreSQL | Users, workspaces, permissions, conversations, runs, prompt metadata |
| Cache and hot/session state | Redis | Short-term memory, caching, rate limits |
| Semantic retrieval | Qdrant or another vector store | Embeddings and similarity search |
| Files and large artifacts | MinIO/S3-compatible object storage | Documents, media, and run artifacts |
| Optional asynchronous events | NATS, later if needed | Brokered events |
| Telemetry | OpenTelemetry with Prometheus/Grafana-compatible backends | Traces, metrics, and dashboards |

This stack is meant to run locally at low cost before managed infrastructure or Kubernetes is introduced. Secrets are supplied through environment/secret-manager mechanisms and are not committed to the repository.

## API and interaction model

The accepted protocol direction is versioned HTTP/JSON at the public edge, with SSE preferred for streaming in the first user-facing version. Internal services start with HTTP/JSON for debuggability; gRPC may be introduced for high-chatter paths only when profiling justifies it. Conceptual edge operations include creating conversations, posting messages, fetching conversation history, inspecting runs, and submitting approvals.

The repository’s reference diagrams also illustrate broader future possibilities such as GraphQL, WebSocket, mobile clients, billing, search engines, and additional control-plane registries. Those are architectural illustrations, not evidence that those features are implemented or final protocol decisions.

## Safety, reliability, and operations

Security is enforced at multiple points: gateway authentication, workspace authorization, data-layer isolation, pre-tool permission/schema/injection checks, approval gates for dangerous actions, and post-response secret/policy scanning. The default posture for hard side effects and security failures is deny or fail closed.

Reliability comes from bounded retries, idempotency keys, circuit breakers where appropriate, cancellation, timeouts, persisted partial state, and provider-independent adapters. A provider outage should not corrupt durable conversation state, and a failed run should show which steps completed and why others failed.

Every request should carry `request_id`, `workspace_id`, `conversation_id`, `run_id`, and where applicable `step_id`, provider, and model attributes. OpenTelemetry spans are expected across gateway, planning, retrieval, inference, tools, memory, and response generation. Raw prompt content must not become a high-cardinality metric label or an accidentally exported secret.

Evaluation prioritizes deterministic signals—latency, token/cost usage, tool success, schema/citation checks, and retrieval proxies—before optional LLM-as-judge or human ratings. Benchmarks are intended to compare providers, prompts, and releases rather than rely on subjective “it feels better” testing.

## Current repository contents

The repository currently contains:

- Product vision, goals, non-goals, personas, functional/non-functional requirements, glossary, and success metrics.
- Domain definitions for workspace, agent, conversation, memory, tool, provider, evaluation, prompt, and workflow.
- Architecture documents for every major subsystem.
- ADRs covering documentation-first development, provider independence, architecture style, protocols, storage, configuration, testing, and OpenTelemetry.
- Draft provider/tool/planner contracts and starter JSON schemas for workspace, conversation, and workflow runs.
- Management roadmaps, milestones, backlog, changelog, release guidance, and numbered tasks.
- A documentation verification script: `scripts/verify_m0_foundation.py`.
- An agent-orchestrator documentation pack containing its PRD, API concepts, data model, test plan, decisions, and TODOs.
- An initial `services/inference-gateway` Go module containing the provider port, normalized chat types, deterministic fake provider, and unit tests.
- A minimal HTTP entry point at `services/inference-gateway/cmd/inference-gateway` exposing `/v1/generate`, `/healthz`, and `/readyz` with the fake provider.
- Environment-first gateway configuration documented in `services/inference-gateway/CONFIGURATION.md`.
- An optional Ollama adapter at `services/inference-gateway/ollama_provider.go`, selected with `ATLAS_INFERENCE_PROVIDER=ollama`.
- A development Dockerfile and Compose service for the inference gateway, connected to the local Ollama data-plane service.
- PostgreSQL-backed inference-run persistence with an embedded schema migration and `X-Run-ID` response correlation.
- PostgreSQL-backed conversation identity and ordered message-history storage with create/get endpoints.
- Conversation messages can now invoke the configured model using full stored history; user and assistant turns are linked to the persisted inference run.

There is still no complete platform runtime, database migration set, or generated API specification. The Ollama adapter exists but requires an Ollama process and local model outside this repository. The verification script checks the M0 documentation foundation; the inference gateway also has a local Go unit test suite.

## Roadmap

The planned progression is:

1. **M0 Foundation — complete:** product/domain/architecture documentation, ADRs, navigation, contracts, schemas, and management structure.
2. **M1 Core Platform:** inference gateway, configuration, logging, metrics, and local Compose data plane.
3. **M2 Intelligence:** planner, context builder, memory engine, and stored multi-step plan artifacts.
4. **M3 Tools:** tool runtime, mock tools, optional real integration, workflow timeouts/retries, approvals, and traces.
5. **M4 Knowledge:** document ingestion, embeddings, retrieval, and source metadata/citations.
6. **M5 Production:** evaluation gates, security hardening, deployment documentation, runbooks, dashboards, and release readiness.

Autonomous self-modifying coding agents, unbounded self-improvement loops, large multi-agent swarms, multi-region active-active deployment, and a full enterprise identity stack are explicit later/non-goals for the early milestones.

## One-sentence summary

Project Atlas is a documentation-first, provider-independent platform intended to turn authenticated user goals into observable, policy-controlled, evaluatable agent workflows that can use models, memory, knowledge, and tools without making any one vendor or model the application itself.

## Source documents

This overview consolidates the repository’s `README.md`, `AI_CONTEXT.md`, product constitution, domain language, architecture documents, ADRs, contracts, schemas, agent-orchestrator service pack, management roadmap, and M0 verification script. Those documents remain the detailed sources of truth for individual requirements and design decisions.
