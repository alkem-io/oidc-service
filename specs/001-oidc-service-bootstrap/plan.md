# Implementation Plan: OIDC Service Bootstrap

**Branch**: `[001-oidc-service-bootstrap]` | **Date**: 2025-10-30 | **Spec**: `specs/001-oidc-service-bootstrap/spec.md`

**Input**: Feature specification from `/specs/001-oidc-service-bootstrap/spec.md`

## Summary

Port the Alkemio OIDC orchestration from the legacy NestJS server into a standalone Go service. The service exposes `/v1/oidc/login` and `/v1/oidc/consent`, integrates with Hydra and Kratos using typed clients, enforces maintenance toggles, and surfaces structured health telemetry.

## Technical Context

**Language/Version**: Go 1.22

**Primary Dependencies**: `chi` for routing, `ory/hydra-client-go`, `zap` for logging, `github.com/prometheus/client_golang`, `github.com/kelseyhightower/envconfig`

**Storage**: MySQL (Kratos session database read access), Redis not required by this service

**Testing**: `go test`, httptest-based integration suites, contract tests invoking generated OpenAPI fixtures

**Target Platform**: Linux containers running in Kubernetes and docker-compose

**Project Type**: Headless HTTP service

**Performance Goals**: p95 login latency < 500 ms, readiness checks < 200 ms even under load

**Constraints**: All outbound calls must honour configured timeouts; maintenance must short-circuit without hitting Hydra/Kratos; container image based on distroless runtime

**Scale/Scope**: Supports Alkemio multi-tenant deployments (~10k daily authentications)

## Constitution Check

- Domain-Oriented Packages: `internal/challenge`, `internal/login`, `internal/consent`, `internal/maintenance` encapsulate orchestration logic – compliant.
- Deterministic Configuration: `internal/config` centralises env parsing – compliant.
- Secure External Integrations: Hydra/Kratos clients built with timeouts and error wrapping – compliant.
- Observability: Structured zap logging and Prometheus metrics wired in `cmd/server` – compliant.
- Fail-Fast Maintenance Controls: `internal/maintenance` middleware short-circuits requests – compliant.
- Test-Driven Delivery: Contract, integration, and unit suites defined under `test/` – compliant.
- Reproducible Containers: Multi-stage Dockerfile pinned to go1.22 bookworm and distroless runtime – compliant.
- CI Integrity: `.github/workflows/ci.yml` runs lint/test/build – compliant.

_No deviations from the constitution were required._

## Project Structure

### Documentation (this feature)

```
specs/001-oidc-service-bootstrap/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```
cmd/
└── server/
    └── main.go

internal/
├── challenge/
├── config/
├── consent/
├── health/
├── hydra/
├── kratos/
├── login/
├── maintenance/
├── middleware/
└── server/

pkg/
└── telemetry/

test/
├── contract/
├── integration/
└── e2e/
```

**Structure Decision**: Single headless Go service with clear internal package boundaries. `cmd/server` composes dependencies, `internal/` holds unexported domain orchestration, and `test/` mirrors contract vs integration coverage.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|---------------------------------------|
| _None_ | N/A | N/A |
