# Releases

## Purpose
Define how Atlas will version and release artifacts once implementation exists.

## Scope
Release philosophy for a pre-runtime monorepo. No binary releases required for M0.

## Dependencies
- [Changelog](./CHANGELOG.md)
- [Milestones](./MILESTONES.md)

## Owner
Atlas Core Engineering

## Status
Draft

## Last Updated
2026-08-03

## Related Documents
- [Deployment Architecture](../docs/architecture/deployment.md)

## Policy (pre-1.0)
- Repository remains the source of truth; documentation milestones (M0) are not versioned as product releases.
- When services ship, use semantic versioning for modules/images as appropriate.
- Every release references changelog entries and known constraints (provider defaults, breaking contract changes).

## Artifacts (future)
- Container images per service
- Compose / Helm packages
- Contract version tags
- Benchmark reports attached to release notes when eval gates exist

## M0 Note
M0 completion is a documentation foundation checkpoint, not a versioned customer release.
