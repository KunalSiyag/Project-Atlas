# Changelog

## Purpose
Record notable changes to the Atlas repository for humans and release notes.

## Scope
Repository-level changes. Prefer ADRs for architectural decisions; this file summarizes.

## Dependencies
- [Releases](./RELEASES.md)
- [README](../README.md)

## Owner
Atlas Core Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Milestones](./MILESTONES.md)

## Format
Keep a human-readable list. When versioned releases begin, adopt Keep a Changelog style under version headings.

## Unreleased

### Added
- Product constitution under `docs/product/` (vision, goals, non-goals, personas, FR, NFR, success metrics, glossary).
- Domain language under `docs/domain/` (workspace, agent, conversation, memory, tool, provider, evaluation, prompt, workflow).
- Subsystem architecture docs under `docs/architecture/`.
- Foundation ADRs 0002–0008 (provider independence, architecture style, protocols, storage, configuration, testing, OpenTelemetry).
- Management layer under `management/`.
- Indexes for `contracts/`, `schemas/`, `tasks/`.
- Updated root README and AI_CONTEXT for M0-complete navigation.

### Previously
- Bootstrap documentation-first monorepo, ADR 0001, engineering standards, agent-orchestrator service doc shell.
