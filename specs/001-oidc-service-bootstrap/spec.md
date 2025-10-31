# Feature Specification: OIDC Service Bootstrap

**Feature Branch**: `[001-oidc-service-bootstrap]`

**Created**: 2025-10-30

**Status**: Implemented

**Input**: User description: "Replace the legacy NestJS OIDC controllers with a dedicated Go service that owns Hydra/Kratos orchestration, health reporting, and maintenance controls."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete Hydra Login Flow (Priority: P1)

Operations staff trigger the OIDC login journey through the public REST gateway so that end-users can authenticate via Hydra and receive the correct redirect response.

**Why this priority**: Without a working login flow the platform cannot authenticate any interactive user; this is the MVP slice of the service.

**Independent Test**: Execute `POST /v1/oidc/login` with a valid Hydra `login_challenge` and verify the JSON response contains redirect metadata and session identifiers without touching other stories.

**Acceptance Scenarios**:

1. **Given** a valid `login_challenge` issued by Hydra, **When** the client submits it to `/v1/oidc/login`, **Then** the service validates the challenge with Hydra, resolves the Kratos session, and returns the redirect payload defined in the OpenAPI contract.
2. **Given** an invalid or expired `login_challenge`, **When** the client submits it to `/v1/oidc/login`, **Then** the service returns HTTP 400 with error code `INVALID_CHALLENGE` and does not mutate Hydra state.

---

### User Story 2 - Consent Completion (Priority: P2)

As an end-user completing OIDC consent, I can accept granted scopes and be redirected back to the application without manual Hydra intervention.

**Why this priority**: Consent happens after login; delivering it second preserves incremental value while keeping the journey independently deployable.

**Independent Test**: Call `POST /v1/oidc/consent` with a valid `consent_challenge` and verify that Hydra receives the acceptance decision and the client gets the redirect URI specified in the response body.

**Acceptance Scenarios**:

1. **Given** a valid consent challenge and granted scopes, **When** the operator calls `/v1/oidc/consent`, **Then** Hydra records the decision and the response includes the `redirect_to` URL from Hydra.
2. **Given** a challenge marked for denial, **When** the client submits the payload with `accept=false`, **Then** Hydra receives the rejection and the endpoint returns HTTP 200 redirecting to the failure URL while recording the audit event.

---

### User Story 3 - Operational Overlays (Priority: P3)

Platform SREs must observe health and toggle emergency maintenance mode without redeploying the service.

**Why this priority**: Operability is essential but can launch after login/consent once traffic is proven.

**Independent Test**: Use `/health/live`, `/health/ready`, and `/maintenance` endpoints to verify JSON responses, readiness dependencies, and maintenance state transitions without executing login or consent stories.

**Acceptance Scenarios**:

1. **Given** the service is running with all dependencies healthy, **When** `/health/ready` is called, **Then** it returns HTTP 200 with component statuses for Hydra, Kratos, database, and maintenance flag `inactive`.
2. **Given** maintenance mode is toggled on via `/maintenance`, **When** a login request arrives, **Then** the service short-circuits with HTTP 503 and the health endpoint reports `maintenanceActive=true`.

### Edge Cases

- What happens when Hydra Admin API returns a 5xx error mid-flow?
- How does the system handle Kratos session cookies that fail validation?
- What response is emitted if maintenance is enabled while a consent request is in-flight?
- How do we surface configuration errors (e.g., missing Hydra URLs) at startup?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose `/v1/oidc/login` that validates Hydra login challenges and returns redirect directives.
- **FR-002**: System MUST expose `/v1/oidc/consent` supporting acceptance and denial flows with proper audit logging.
- **FR-003**: System MUST call Hydra Admin APIs using request-scoped timeouts and translate failures into structured problem responses.
- **FR-004**: System MUST resolve Kratos sessions using the configured cookie name without reading `os.Getenv` outside bootstrap.
- **FR-005**: System MUST provide `/health/live` and `/health/ready` endpoints returning JSON objects aligned with the OpenAPI schema.
- **FR-006**: System MUST expose `/maintenance` management endpoints protected by shared secret and toggle request short-circuiting.
- **FR-007**: System MUST emit structured logs and Prometheus metrics for every public endpoint invocation.
- **FR-008**: System MUST maintain OpenAPI 3.1 contract files under `contracts/` and keep contract tests green.

### Key Entities

- **HydraChallenge**: Represents login or consent challenge metadata retrieved from Hydra, including redirect URLs and scope grants.
- **KratosSession**: Captures the authenticated identity, session expiry, and traits fetched from Kratos.
- **MaintenanceState**: Describes whether the service is actively rejecting traffic, including the operator who toggled the flag and timestamps.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 99% of Hydra login requests complete in under 500 ms at p95 with Hydra reachable.
- **SC-002**: Maintenance toggles propagate within 100 ms and block subsequent login requests immediately.
- **SC-003**: Contract tests in `test/contract` remain green on every CI run.
- **SC-004**: Health readiness endpoint accurately reports dependency degradation within one polling interval (≤10 seconds).
