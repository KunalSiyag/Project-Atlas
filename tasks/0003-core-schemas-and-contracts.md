# Task 0003: Expand Core Schemas and Provider Contracts

## Purpose
Fill remaining JSON schemas and deepen provider/tool/memory contracts for M1.

## Scope
`schemas/*` and `contracts/*` content expansion.

## Dependencies
- [Task 0001](./0001-repository-bootstrap.md)
- [Domain language](../docs/domain/README.md)

## Owner
Atlas Architecture

## Status
Planned

## Last Updated
2026-08-03

## Related Documents
- [Schemas README](../schemas/README.md)
- [Contracts README](../contracts/README.md)

## Goal
Have machine-readable schemas for tool, memory, prompt, provider and tightened contract docs ready for code generation or validation.

## Inputs
- Domain docs and starter schemas (conversation, workspace, run)

## Outputs
- Additional `*.schema.json` files; richer contract markdown

## Acceptance Criteria
- Listed planned schemas either exist or are explicitly deferred with reason
- Contracts describe error classes and required fields

## Test Requirements
- JSON Schema parse validation in CI when tooling lands
