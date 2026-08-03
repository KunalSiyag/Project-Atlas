# Domain: Prompt

## Purpose
Define prompts as versioned artifacts, not anonymous strings buried in code.

## Scope
Prompt identity, versioning, and composition with tools and agents. Prompt engineering craft is not fully specified here.

## Dependencies
- [Agent](./agent.md)
- [Tool](./tool.md)
- [Evaluation](./evaluation.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Context Builder Architecture](../architecture/context-builder.md)
- [Inference Architecture](../architecture/inference.md)

## Definition
A **prompt** is a versioned instruction artifact (system, developer, style, or task template) used when constructing model context. Tool schemas and few-shot examples may be associated prompt assets.

## Boundaries
- In scope: identifier, version, content, ownership, changelog, environment promotion.
- Out of scope: full multi-modal asset pipelines (linked via object storage when needed).

## Core Attributes (conceptual)
- `prompt_id`, `version`, stage (`draft` | `staging` | `production`)
- Role (system / developer / task template)
- Content body and optional structured sections
- Linked **tools** schemas allowed with this prompt
- Author, timestamps, change notes

## Relationships
- Bound to **agents** by reference (not copy-paste only).
- Combined with **memory**, history, and retrieved knowledge during **context building**.
- Versions are compared in **evaluation** and benchmarks.
- Distinct from **memory** (instructions vs state).

## Invariants
- Production agents reference immutable prompt versions (or immutable snapshots) for a run.
- Changes that affect behavior get a new version for auditability.
- Secrets never live inside prompt text.

## Failure Modes
- Drift when code embeds prompts that diverge from versioned store.
- Accidental promotion of draft prompts.
- Context overflow from oversized templates plus history.
