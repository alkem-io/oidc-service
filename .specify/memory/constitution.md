# Alkemio OIDC Service Engineering Constitution

## Core Principles

1. **Domain-Oriented Packages First**
   - Business logic for Hydra/Kratos orchestration lives under `internal/challenge` as pure Go structs.
   - HTTP handlers in `internal/server` remain thin and delegate to orchestration services.
2. **Deterministic Configuration**
   - All runtime settings flow through typed structs in `internal/config`; no package reads from `os.Getenv` outside bootstrap.
3. **Secure External Integrations**
   - Hydra and Kratos clients enforce request-scoped timeouts, structured error translation, and never log tokens or secrets.
4. **Operational Observability**
   - Zap logging includes correlation IDs, challenge IDs, and outcome status. Prometheus metrics were intentionally removed in Nov 2025; structured logs (shipped via the platform stack) are the single required observability signal.
5. **Fail-Fast Maintenance Controls**
   - Maintenance mode immediately returns HTTP 503 with Retry-After headers without touching external services.
6. **Test-Driven Delivery**
   - Tests exist only when they defend a real invariant or observable behaviour. Contract tests exercise OpenAPI behaviour before handler implementation, integration tests cover Hydra/Kratos error paths and maintenance toggles, and we explicitly avoid superficial "coverage padding" or placeholder tests that do not catch regressions.
7. **Reproducible Containers**
   - Docker builds pin base image digests, run `go test` and `go build` in multi-stage pipeline, and produce distroless runtime images.
8. **CI Integrity**
   - GitHub Actions pipelines run linting, unit tests, integration smoke, SBOM generation, and signed Docker pushes before releases.
9. **Evidence-Based Performance Goals**
   - Do not invent ad-hoc SLAs (e.g., "p95 < 500 ms") for thin orchestration layers whose latency is dominated by external systems. Only codify performance targets when the service owns the bottleneck and can meaningfully improve it; otherwise focus on timeout budgets, dependency health checks, and log-based diagnostics.
10. **Meaningful Success Criteria**
   - Specifications must express success criteria that are directly testable within oidc-service (e.g., contract or integration tests) and under this feature's control; avoid vanity metrics or external business outcomes that cannot be validated during development.
   - Never invent arbitrary measurable outcomes pulled from thin air (e.g., "50ms response time", "40% improvement") unless backed by baseline measurements or explicit stakeholder requirements.
11. **No Busywork**
   - Every task, test, and artifact must deliver demonstrable value. Reject make-work activities that exist only to satisfy process checkboxes.
   - Do not create documentation, comments, or abstractions "just in case" — only when they solve a real problem or prevent a real mistake.
   - Specifications should be lean: include only what is necessary to communicate intent and validate correctness.
12. **Meaningful Tests Only**
   - Tests must defend real invariants or catch real regressions. Never write tests for the sake of coverage metrics.
   - Avoid testing implementation details, trivial getters/setters, or scenarios that cannot fail in practice.
   - If a test does not help catch bugs or document critical behaviour, delete it.
13. **Latest Dependencies Always**
   - When adding or updating any dependency, always verify the latest stable version by checking online sources (pkg.go.dev, GitHub releases, npm registry, etc.). Never rely on AI training data for version numbers.
   - Pin dependencies to specific versions in go.mod/go.sum, but ensure those versions are current at time of addition.
   - Document rationale when intentionally using older versions (compatibility, stability concerns).
14. **Always Root Cause Analysis**
   - Never apply opportunistic or speculative fixes hoping they might resolve an issue.
   - Before any bug fix, identify and document the actual root cause with evidence (logs, traces, reproduction steps).
   - If the root cause is unclear, invest time in debugging and analysis first — guessing wastes more time than investigating.
   - Fixes must directly address the identified root cause, not symptoms.
15. **No Legacy Code**
   - Never silently assume backward compatibility is required. We control the full stack and all consumers.
   - Do not leave code "just in case" — dead, deprecated, or unused code has no right to stay in the codebase unless explicitly requested.
   - When a feature requires changes across multiple services, coordinate those changes rather than maintaining compatibility shims.
   - Aim for a minimalistic, well-maintainable codebase. Every line of code must justify its existence.
   - Remove backward-compatibility hacks, unused exports, commented-out code, and defensive code for scenarios that no longer apply.
16. **Single Source of Truth**
   - No two methods should implement the same logic in different modules. If duplication exists, extract to a single shared utility.
   - When methods share partial logic, extract the common part to a shared helper.
   - There is always exactly one piece of code for any given logic — find it, use it, or create it once.
   - Before implementing new logic, search for existing implementations. Extend rather than duplicate.
   - Configuration, constants, and type definitions live in one canonical location.
17. **No Assumptions**
   - Never assume requirements, behavior, or implementation details that are not explicitly defined.
   - If something is unclear or unknown, ask the user for clarification before proceeding.
   - If factual information is needed (versions, API specs, library behavior), search online to verify.
   - Do not guess — guessing leads to rework. Asking or searching takes less time than fixing wrong assumptions.

## Architecture Standards

- Directory layout:
  - `cmd/server`: entrypoint wiring configuration and HTTP server start.
  - `internal/*`: non-exported packages encapsulating config, clients, middleware, and domain orchestration.
   - `pkg/telemetry`: shared logging utilities safe for reuse.
  - `configs/`: sample environment files and documentation.
  - `contracts/`: OpenAPI definitions and generated fixtures.
  - `docs/`: operational runbooks and quickstarts.
- Interfaces to external systems (`internal/hydra`, `internal/kratos`) expose narrow methods consumed by the challenge service.
- Health endpoints (`/health/live`, `/health/ready`) MUST satisfy constitution observability principle with structured JSON responses.
- Maintenance middleware short-circuits request handling while still recording structured logs.
- Performance requirements appear only when the service directly controls the critical path; otherwise specifications MUST document dependency expectations instead of arbitrary p95 quotas.

## Engineering Workflow

- Execute Spec Kit phases sequentially: research → design → tasks → implementation → validation.
- Tests (`go test ./...`) MUST pass before merging. Contract tests run under `test/contract`; integration tests use `httptest` with mocked Hydra/Kratos clients.
- Any change to OpenAPI contracts requires regenerating associated fixtures and reviewing downstream consumers.
- Release bumps require update to `docs/operations.md` and changelog entry summarizing risk.
- Feature numbering is global. Every new feature branch, Spec Kit directory, and checklist MUST use the next available integer across the repository (e.g., `004-agent-claim`), never resetting numbering per initiative or contributor.

## Governance

- Deviations from this constitution are documented in Spec Kit plans with mitigation timelines.
- Major changes (new external dependency, architecture shift) require constitution update and review.
- Operability incidents result in follow-up tasks captured in Spec Kit memory.

**Version**: 1.6.0 | **Ratified**: 2025-10-29 | **Last Amended**: 2026-01-08
