# Implementation Plan: Token Claims Enhancement

**Branch**: `002-token-claims` | **Date**: October 31, 2025 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-token-claims/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Enhance OIDC token generation to include standardized user profile claims in both Access tokens (given_name, family_name) and ID tokens (given_name, family_name, email_verified, accepted_terms). Claims will be sourced from Kratos identity traits and follow OpenID Connect standards, with graceful handling of missing data and robust error handling.

## Technical Context

**Language/Version**: Go 1.25 (from copilot instructions and constitution)  
**Primary Dependencies**: Chi router, Zap logging, Hydra client, Kratos client  
**Storage**: N/A (reads from existing Kratos identity system)  
**Testing**: `go test ./...`, golangci-lint run (from copilot instructions)  
**Target Platform**: Linux server (containerized backend service)  
**Project Type**: Single backend service (OIDC token enhancement)  
**Performance Goals**: N/A (lightweight claim addition with negligible impact)  
**Constraints**: Maintain existing token generation performance, token size limits, secure handling of user data  
**Scale/Scope**: Enhancement to existing OIDC service, affects all token generation flows

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Initial Check (Pre-Research)**:
✅ **Domain-Oriented Packages First**: Token claim logic will live in `internal/challenge` as pure Go structs, following existing patterns  
✅ **Deterministic Configuration**: No new configuration needed, uses existing Kratos client configuration  
✅ **Secure External Integrations**: Leverages existing Kratos client with timeout and error handling  
✅ **Operational Observability**: Will extend existing Zap logging for token claim operations (Prometheus metrics were removed Nov 2025)  
✅ **Fail-Fast Maintenance Controls**: Existing maintenance mode will cover new functionality  
✅ **Test-Driven Delivery**: Will add contract tests for token claim validation and integration tests for error paths  
✅ **Reproducible Containers**: No changes to Docker build process required  
✅ **CI Integrity**: Existing pipeline will validate new functionality  

**Re-Check (Post-Design)**:
✅ **Domain-Oriented Packages First**: Design confirms extension of `internal/challenge/identity_mapper.go` following existing patterns  
✅ **Deterministic Configuration**: No new config required, uses existing structured config pattern  
✅ **Secure External Integrations**: Design leverages existing Kratos client with proper error handling and no secret logging  
✅ **Operational Observability**: Design includes structured logging for claim operations; metrics requirement removed Nov 2025  
✅ **Fail-Fast Maintenance Controls**: Existing maintenance middleware covers new token generation paths  
✅ **Test-Driven Delivery**: Detailed contract test specifications in `/contracts/` and integration test scenarios defined  
✅ **Reproducible Containers**: No Docker changes needed, existing build pipeline remains valid  
✅ **CI Integrity**: Existing linting and testing pipeline will validate new code  

**GATE STATUS**: ✅ PASS - All constitutional principles satisfied in design

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/server/                    # Application entrypoint
internal/
├── challenge/                 # Domain logic for Hydra/Kratos orchestration (MODIFY)
│   ├── identity_mapper.go     # Map Kratos traits to token claims (MODIFY)
│   ├── service.go            # Token generation orchestration (MODIFY)
│   └── model.go              # Token claim data structures (MODIFY)
├── config/                   # Configuration management (NO CHANGE)
├── hydra/                    # Hydra client integration (NO CHANGE)
├── kratos/                   # Kratos client integration (NO CHANGE)
├── server/                   # HTTP handlers (MINIMAL CHANGE)
│   ├── consent_handler.go    # May need token claim integration (REVIEW)
│   └── login_handler.go      # May need token claim integration (REVIEW)
└── middleware/               # Request middleware (NO CHANGE)

pkg/telemetry/                # Shared instrumentation (NO CHANGE)

test/
├── contract/                 # OpenAPI behavior tests (ADD)
└── integration/              # Hydra/Kratos integration tests (ADD)

contracts/                    # OpenAPI definitions (UPDATE)
```

**Structure Decision**: Single backend service enhancement. Primary changes will be in `internal/challenge` package following domain-oriented architecture. Existing HTTP handlers and client integrations remain largely unchanged.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
