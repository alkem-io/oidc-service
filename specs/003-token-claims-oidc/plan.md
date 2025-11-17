# Implementation Plan: Alkemio User ID Token Claim

**Branch**: `003-token-claims-oidc` | **Date**: 2025-11-17 | **Spec**: `specs/003-token-claims-oidc/spec.md`
**Input**: Feature specification from `/specs/003-token-claims-oidc/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add a new Alkemio user ID claim to tokens issued by oidc-service by resolving the Alkemio user ID from the user's Kratos authentication ID via the Alkemio server endpoint `/rest/internal/identity/resolve`. Token issuance must fail when no Alkemio user mapping exists or when the identity resolution call fails, and the claim must be included in both access tokens and ID tokens for every successful issuance (no client/flow gating) with strong observability of resolution outcomes.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.25  
**Primary Dependencies**: internal Hydra/Kratos clients, internal HTTP server/middleware, Zap logging  
**Storage**: N/A (feature reads from existing Kratos and Alkemio server; no new storage)  
**Testing**: `go test ./...` with existing unit, integration, and contract tests  
**Target Platform**: Linux container runtime for oidc-service
**Project Type**: Single backend service  
**Performance Goals**: Preserve existing token issuance behavior (no new performance targets introduced by this feature).  
**Constraints**: Must respect a total timeout budget of 30 seconds for identity resolution with up to 5 retries, reuse the existing HTTP client abstraction, ensure both Kratos authentication IDs and Alkemio user IDs are valid UUIDs, and avoid logging sensitive identifiers; follow constitution observability and security principles.  
**Scale/Scope**: Limited to token issuance paths that require the Alkemio user ID claim; no changes to other services.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Domain-Oriented Packages First**: Plan keeps orchestration logic in `internal/challenge` and ensures HTTP handlers stay thin and delegate to the challenge/claim service. No business logic will be added directly in HTTP handlers.
- **Deterministic Configuration**: Any new configuration (e.g., Alkemio server base URL or timeouts, if needed) will be added to `internal/config` and wired through typed structs; no direct `os.Getenv` reads will be introduced.
- **Secure External Integrations**: The identity resolution call to the Alkemio server will reuse the existing HTTP client abstraction, enforce a total timeout budget of 30 seconds with up to 5 retries, and will not log tokens, secrets, or full identifiers.
- **Operational Observability**: Identity resolution attempts will emit structured Zap logs (success, not-found, failure) with correlation and challenge IDs, following existing middleware patterns. Prometheus metrics were removed in Nov 2025.
- **Fail-Fast Maintenance Controls**: The new behavior will respect existing maintenance middleware, i.e., no external calls will be made when maintenance mode short-circuits requests.
- **Test-Driven Delivery**: New contract and integration tests will be added to cover successful resolution, not-found (404), and failure cases before or alongside implementation; `go test ./...` must remain green.
- **Reproducible Containers**: No changes to Docker pipeline are required; build remains deterministic.
- **CI Integrity**: No changes to CI workflows are required; existing pipelines will validate the feature.

**Gate Evaluation**: All relevant principles can be satisfied with this plan; no constitution violations are anticipated, so no entries are required in Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/003-token-claims-oidc/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── server/               # Entrypoint wiring configuration and HTTP server

internal/
├── challenge/            # Domain orchestration for token challenges and claims
├── config/               # Typed configuration for external services
├── hydra/                # Hydra client integration
├── kratos/               # Kratos client integration
├── middleware/           # Request ID, maintenance, error handling
└── server/               # HTTP server setup and routing

pkg/
└── telemetry/            # Shared logging utilities

test/
├── contract/
├── e2e/
├── integration/
└── unit/
```

**Structure Decision**: Use the existing single-service layout with new or extended functionality in `internal/challenge` for claim resolution orchestration, a dedicated client or helper for calling the Alkemio server if needed, and minimal glue changes in `internal/server` or middleware to wire the new claim into token issuance.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
