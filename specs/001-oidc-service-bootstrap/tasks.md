# Tasks: OIDC Service Bootstrap

**Input**: Design documents from `/specs/001-oidc-service-bootstrap/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

## Phase 1: Setup (Shared Infrastructure)

- [x] T001 Initialise Go module `github.com/alkem-io/oidc-service` with Go 1.22 (go.mod)
- [x] T002 [P] Configure lint/test GitHub workflow `.github/workflows/ci.yml`
- [x] T003 [P] Pin multi-stage Dockerfile with distroless runtime (`Dockerfile`)

## Phase 2: Foundational (Blocking Prerequisites)

- [x] T010 Implement typed configuration loader in `internal/config/config.go`
- [x] T011 [P] Add structured logging + metrics utilities in `pkg/telemetry`
- [x] T012 [P] Build Hydra and Kratos clients with timeout-aware HTTP transport (`internal/hydra`, `internal/kratos`)
- [x] T013 Wire chi router, middleware, and telemetry bootstrap in `cmd/server/main.go`
- [x] T014 Establish contract + integration test harness under `test/`

**Checkpoint**: Foundation ready – user story implementation can proceed.

## Phase 3: User Story 1 - Complete Hydra Login Flow (Priority: P1) 🎯 MVP

**Goal**: Accept Hydra login challenges, resolve Kratos session, and provide redirect payload.

**Independent Test**: `go test ./test/contract -run TestLoginContract`

### Tests

- [x] T101 [P] [US1] Write contract coverage in `test/contract/login_contract_test.go`
- [x] T102 [P] [US1] Add integration test using httptest mux in `test/integration/login_flow_test.go`

### Implementation

- [x] T110 [P] [US1] Model Hydra login domain types in `internal/challenge/model.go`
- [x] T111 [US1] Implement `ChallengeService.Login` orchestration (`internal/challenge/service.go`)
- [x] T112 [US1] Implement login handler + DTO mapping in `internal/server/login_handler.go`
- [x] T113 [US1] Add maintenance short-circuit middleware path (`internal/middleware/maintenance.go`)
- [x] T114 [US1] Record metrics + logs for login outcomes (`pkg/telemetry/metrics.go`)

**Checkpoint**: Login endpoint independently deployable.

## Phase 4: User Story 2 - Consent Completion (Priority: P2)

**Goal**: Allow acceptance/denial of Hydra consent challenges with redirect payloads.

**Independent Test**: `go test ./test/contract -run TestConsentContract`

### Tests

- [x] T201 [P] [US2] Contract assertions in `test/contract/consent_contract_test.go`
- [x] T202 [US2] Integration path coverage `test/integration/consent_flow_test.go`

### Implementation

- [x] T210 [P] [US2] Extend challenge service with consent orchestration (`internal/challenge/service.go`)
- [x] T211 [US2] Implement consent handler + DTOs in `internal/server/consent_handler.go`
- [x] T212 [US2] Ensure audit logging around granted scopes (`internal/server/redaction.go`)

**Checkpoint**: Consent flow operational alongside login.

## Phase 5: User Story 3 - Operational Overlays (Priority: P3)

**Goal**: Health probes, maintenance toggles, and observability surfaces.

**Independent Test**: `go test ./test/contract -run TestHealthContract`

### Tests

- [x] T301 [P] [US3] Contract coverage for health endpoints (`test/contract/health_contract_test.go`)
- [x] T302 [US3] Integration coverage for maintenance toggles (`test/integration/maintenance_toggle_test.go`)

### Implementation

- [x] T310 [US3] Build readiness checker aggregating Hydra/Kratos/database (`internal/challenge/readiness_http.go`)
- [x] T311 [P] [US3] Implement maintenance state + HTTP handlers (`internal/maintenance/state.go`, `internal/server/router.go`)
- [x] T312 [US3] Expose `/metrics` Prometheus handler in `internal/server/router.go`

**Checkpoint**: Service operationally ready with health + maintenance endpoints.

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T401 [P] Publish OpenAPI 3.1 contract to `contracts/openapi.yaml`
- [x] T402 Update runbooks in `docs/operations.md` and quickstart instructions
- [x] T403 [P] Add zap logger redaction coverage (`internal/server/redaction_test.go`)
- [x] T404 Harden error translation for Hydra/Kratos clients (`internal/challenge/session_errors.go`)
- [x] T405 Ensure docker-compose integration exposes port 8080 (`deployments/docker-compose/compose.yaml`)

## Dependencies & Execution Order

- Phases 1–2 completed sequentially to lay infrastructure.
- User Story phases (3–5) delivered in priority order; each independently testable.
- Polish tasks executed after functional parity to align documentation and deployments.

## Implementation Strategy

1. Deliver login MVP (Phase 3) and verify contract tests.
2. Layer consent flow (Phase 4) while reusing challenge service.
3. Add operational overlays (Phase 5) to satisfy SRE requirements.
4. Complete polish tasks to align docs, contracts, and deployment manifests.
