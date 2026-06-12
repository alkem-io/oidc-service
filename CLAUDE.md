# AI Agent Guidelines for Alkemio OIDC Service

> **Workspace context.** This repo is part of the Alkemio polyrepo at
> [alkem-io/agents-hq](https://github.com/alkem-io/agents-hq).
> Cross-repo (vertical) feature specs live there under `specs/NNN-*/`. When
> working on a `feat/NNN-...` branch in this repo, the matching workspace
> spec is the single source of truth.

This document provides essential guidance for AI coding assistants working on this codebase.

## Core Engineering Principles

### Always Root Cause Analysis
- **Never apply opportunistic or speculative fixes** hoping they might resolve an issue
- Before any bug fix, identify and document the actual root cause with evidence
- If the root cause is unclear, invest time in debugging first — guessing wastes more time than investigating
- Fixes must directly address the identified root cause, not symptoms

### No Legacy Code
- **Never silently assume backward compatibility is required** — we control the full stack and all consumers
- Do not leave code "just in case" — dead, deprecated, or unused code must be removed
- When a feature requires changes across multiple services, coordinate those changes rather than maintaining compatibility shims
- Remove backward-compatibility hacks, unused exports, commented-out code, and defensive code for scenarios that no longer apply
- Every line of code must justify its existence

### Single Source of Truth
- **No two methods should implement the same logic** in different modules
- If duplication exists, extract to a single shared utility
- When methods share partial logic, extract the common part to a shared helper
- Before implementing new logic, search for existing implementations — extend rather than duplicate
- Configuration, constants, and type definitions live in one canonical location

### No Busywork
- Every task, test, and artifact must deliver demonstrable value
- Reject make-work activities that exist only to satisfy process checkboxes
- Do not create documentation, comments, or abstractions "just in case"
- Specifications should be lean: only what is necessary to communicate intent

### Meaningful Tests Only
- Tests must defend real invariants or catch real regressions
- Never write tests for the sake of coverage metrics
- Avoid testing implementation details, trivial getters/setters, or scenarios that cannot fail
- If a test does not help catch bugs or document critical behaviour, do not write it

### Meaningful Success Criteria
- Success criteria must be directly testable within this service
- Never invent arbitrary metrics pulled from thin air (e.g., "50ms response time", "40% improvement") without baseline measurements or explicit stakeholder requirements
- Avoid vanity metrics or external business outcomes that cannot be validated during development

### Latest Dependencies Always
- When adding or updating any dependency, **always verify the latest stable version online** (pkg.go.dev, GitHub releases, npm registry, etc.)
- **Never rely on AI training data for version numbers** — it is likely outdated
- Pin dependencies to specific versions, but ensure those versions are current at time of addition

### No Assumptions
- **Never assume** requirements, behavior, or implementation details that are not explicitly defined
- If something is unclear or unknown, **ask the user** for clarification before proceeding
- If factual information is needed (versions, API specs, library behavior), **search online** to verify
- Do not guess — guessing leads to rework; asking or searching takes less time than fixing wrong assumptions

## Active Technologies
- Go 1.25 + chi router, zap logger, existing CompositeResolver (007-kratos-metadata-webhook)
- N/A (uses existing Alkemio DB via CompositeResolver) (007-kratos-metadata-webhook)

- **Language**: Go 1.25
- **Database**: PostgreSQL (Alkemio database)
- **New dependencies** (006-db-claims-lookup): pgx v5.10.0, sqlc v1.30.0

## Architecture Quick Reference

```text
cmd/server/          # Entrypoint wiring
internal/
├── alkemio/         # Identity resolution (API + DB resolvers)
├── challenge/       # Business logic for Hydra/Kratos orchestration
├── config/          # Typed configuration structs
├── hydra/           # Hydra client
├── kratos/          # Kratos client
└── server/          # Thin HTTP handlers
pkg/telemetry/       # Shared logging utilities
test/
├── contract/        # Contract tests
└── integration/     # Integration tests with mocked clients
```

## Key Patterns

1. **Domain-Oriented Packages**: Business logic in `internal/challenge` as pure Go structs
2. **Deterministic Configuration**: All settings via typed structs in `internal/config`
3. **Secure Integrations**: Request-scoped timeouts, structured error translation, never log secrets
4. **Observability**: Zap logging with correlation IDs; no Prometheus (removed Nov 2025)
5. **Fail-Fast Maintenance**: HTTP 503 with Retry-After, no external service calls

## What NOT to Do

- Do not apply speculative fixes — find root cause first
- Do not keep code "just in case" or for backward compatibility unless explicitly requested
- Do not duplicate logic — find or create a single shared implementation
- Do not add superficial tests for coverage padding
- Do not invent performance SLAs without evidence
- Do not create abstractions for hypothetical future needs
- Do not add comments explaining obvious code
- Do not rely on training data for dependency versions — check online
- Do not create documentation files unless explicitly requested
- Do not assume — ask or search when something is unclear

## Full Constitution

See `.specify/memory/constitution.md` for the complete engineering constitution (v1.6.0).

## Recent Changes
- 007-kratos-metadata-webhook: Added Go 1.25 + chi router, zap logger, existing CompositeResolver
