# Functional Requirements

## Purpose
Specify the capabilities Atlas must provide as a platform, independent of a single implementation technology.

## Scope
Platform-level functional requirements for foundation through early production readiness. Service-level PRDs refine these further.

## Dependencies
- [Vision](./VISION.md)
- [Goals](./GOALS.md)
- [Non-Goals](./NON_GOALS.md)
- [Personas](./PERSONAS.md)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Non-Functional Requirements](./NON_FUNCTIONAL_REQUIREMENTS.md)
- [Architecture Overview](../architecture/OVERVIEW.md)
- [Domain Language](../domain/README.md)

## Capability Areas

### FR-AUTH — Authentication and workspace isolation
- FR-AUTH-1: Identify the human user (e.g. JWT or equivalent session credential).
- FR-AUTH-2: Bind requests to a workspace with membership and roles.
- FR-AUTH-3: Authorize tool and knowledge access per workspace and user permissions.
- FR-AUTH-4: Deny cross-workspace data access by default.

### FR-CONV — Conversation and session management
- FR-CONV-1: Create and resume conversations within a workspace.
- FR-CONV-2: Load conversation history, preferences, and agent configuration for a turn.
- FR-CONV-3: Record turn-level artifacts needed for audit and evaluation (inputs, outputs, tool calls, costs).

### FR-CTX — Context building
- FR-CTX-1: Assemble context from conversation history, memories, documents, retrieved tool data, preferences, and system prompts.
- FR-CTX-2: Apply token/budget limits and prioritization rather than always sending everything.
- FR-CTX-3: Expose what was included/excluded for debugging and evaluation (at least at debug verbosity).

### FR-PLAN — Planning
- FR-PLAN-1: Produce a structured plan (tasks/steps) before or instead of a final answer when the request requires multi-step work.
- FR-PLAN-2: Support multi-step goals such as “read sources → summarize → draft → request approval → act.”
- FR-PLAN-3: Persist plan state for inspection and resume after failure or approval.

### FR-WF — Workflow engine
- FR-WF-1: Execute plans as workflows, including DAG structures with parallel independent steps.
- FR-WF-2: Support human-approval gates before side-effecting steps.
- FR-WF-3: Capture per-step status, retries, and errors.
- FR-WF-4: Allow cancellation and timeout of in-flight steps.

### FR-TOOL — Tool runtime
- FR-TOOL-1: Register tools with name, schema, validation, timeout, and retry policy.
- FR-TOOL-2: Execute tools through a common interface (GitHub, Slack, SQL, browser, filesystem, REST, etc.).
- FR-TOOL-3: Support mock/local tool backends for development without cloud credentials.
- FR-TOOL-4: Enforce permissions and policy checks before execution.

### FR-MODEL — Model routing and inference
- FR-MODEL-1: Invoke chat/completion models through a provider interface.
- FR-MODEL-2: Route different task types to different models/providers when configured (planning vs coding vs vision vs embedding).
- FR-MODEL-3: Support at least one free/local provider path for development.
- FR-MODEL-4: Record token usage and estimated cost per call when available.

### FR-MEM — Memory
- FR-MEM-1: Store and retrieve short-term session memory and longer-lived user/workspace memory.
- FR-MEM-2: Support write and query paths used by context building.
- FR-MEM-3: Allow memory entries to be scoped (user, workspace, conversation) and expired or deleted.

### FR-KB — Knowledge and RAG (platform readiness)
- FR-KB-1: Ingest documents into object storage and index embeddings in a vector store via interfaces.
- FR-KB-2: Retrieve relevant chunks for a query with source metadata for citations.
- FR-KB-3: Support free/local embedding and vector implementations for development.

### FR-SAFE — Guardrails
- FR-SAFE-1: Validate permissions before tool execution.
- FR-SAFE-2: Detect and mitigate prompt injection and dangerous tool actions according to policy.
- FR-SAFE-3: Scan outbound responses for secrets and policy violations where configured.
- FR-SAFE-4: Support citation/grounding checks as part of evaluation or response path where configured.

### FR-EVAL — Evaluation
- FR-EVAL-1: Capture latency, cost, tool success rate, and optional quality scores per run.
- FR-EVAL-2: Support deterministic evaluators first (retrieval precision, tool success, citation shape, latency, token usage).
- FR-EVAL-3: Allow optional LLM-as-judge evaluators later without hard dependency.
- FR-EVAL-4: Store benchmark runs for regression comparison across providers.

### FR-OBS — Observability
- FR-OBS-1: Produce structured logs with correlation IDs across a request.
- FR-OBS-2: Emit metrics for request volume, errors, latency, and cost proxies.
- FR-OBS-3: Propagate distributed traces (OpenTelemetry-oriented) across planner, retriever, LLM, tools, and memory.

### FR-CFG — Configuration
- FR-CFG-1: Configure providers, models, and feature flags without code changes where practical.
- FR-CFG-2: Separate secrets from non-secret configuration.
- FR-CFG-3: Document configuration keys per service.

### FR-API — Platform API surface
- FR-API-1: Expose a conversation/agent API suitable for clients (HTTP and/or gRPC as decided in ADRs).
- FR-API-2: Version public contracts.
- FR-API-3: Define contracts before implementation for each public surface.
