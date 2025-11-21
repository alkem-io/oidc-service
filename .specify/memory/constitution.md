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
   - Specifications must express success criteria that are directly testable within oidc-service (e.g., contract or integration tests) and under this feature’s control; avoid vanity metrics or external business outcomes that cannot be validated during development.

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

**Version**: 1.3.2 | **Ratified**: 2025-10-29 | **Last Amended**: 2025-11-20
