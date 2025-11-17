# Tasks: Alkemio User ID Token Claim

**Input**: Design documents from `/specs/003-token-claims-oidc/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: The spec does not explicitly require a TDD approach, but given the security- and behavior-sensitive nature of token issuance, we will include targeted test tasks per user story.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm existing project wiring and docs for this feature.

- [X] T001 Verify existing Go module and build work for oidc-service using `go test ./...` at repository root
- [X] T002 [P] Skim existing token claims implementation and challenge flow in `internal/challenge/` and `internal/server/` to identify extension points for the new claim
- [X] T003 [P] Review existing Hydra/Kratos client patterns in `internal/hydra/` and `internal/kratos/` for timeout and error handling conventions

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core wiring and configuration that MUST be complete before any user story can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Add any required Alkemio server configuration fields (e.g., base URL) to `internal/config/config.go` and ensure they are populated from environment/sample config in `configs/env.sample`
- [X] T005 [P] Update configuration tests in `internal/config/config_test.go` to cover new Alkemio server settings
- [X] T006 [P] Ensure existing maintenance middleware in `internal/middleware/maintenance.go` short-circuits requests before any new identity resolution logic is invoked

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel.

---

## Phase 3: User Story 1 - Include Alkemio user ID in tokens (Priority: P1) 🎯 MVP

**Goal**: When a user with a valid Kratos identity and Alkemio user mapping receives tokens from oidc-service, those tokens contain an `alkemio_user_id` claim matching the mapping, and no tokens are issued if the mapping cannot be resolved.

**Independent Test**: Authenticate a user with a valid Kratos→Alkemio mapping, request access and ID tokens from oidc-service, and verify that both tokens contain the `alkemio_user_id` claim with the expected value. Authenticate a user whose mapping cannot be resolved (404 or failure) and verify that token issuance fails.

### Tests for User Story 1

- [X] T007 [P] [US1] Add or extend contract test in `test/contract/` to assert presence of `alkemio_user_id` in both access and ID tokens when a mapping exists
- [X] T008 [P] [US1] Add integration test in `test/integration/` to cover end-to-end token issuance with a stubbed Alkemio identity resolution that returns a single valid `alkemio_user_id` and assert that both access and ID tokens include the claim
-- [X] T011a [US1] Add integration or unit test in `test/integration/` or `test/unit/` that verifies token issuance fails when the identity resolution endpoint returns 404 (no Alkemio user mapping)

### Implementation for User Story 1

- [X] T009 [P] [US1] Introduce a small Alkemio identity resolution helper or client in `internal/challenge/` or a new dedicated package (e.g., `internal/alkemio/`) that calls `/rest/internal/identity/resolve` reusing the existing HTTP client abstraction
- [X] T010 [US1] Wire the Alkemio identity resolver into the challenge service in `internal/challenge/service.go` (or equivalent) so that, before finalizing token claims, it resolves `alkemio_user_id` from the current `kratos_id`
- [X] T011 [US1] Extend the token claims construction logic in `internal/challenge/identity_mapper.go` (or equivalent file) to include the `alkemio_user_id` claim in both access tokens and ID tokens when resolution succeeds
- [X] T012 [US1] Ensure the new claim name and semantics are documented or reflected in any existing token model structs in `internal/challenge/model.go` (or related files)
- [X] T013 [US1] Run `go test ./...` and fix any issues introduced by the new claim wiring

**Checkpoint**: User Story 1 should be fully functional and testable independently.

---

## Phase 4: User Story 2 - Handle identity resolution failures gracefully (Priority: P2)

**Goal**: Ensure that failures in the Alkemio identity resolution call are handled predictably and are observable without leaking sensitive details.

**Independent Test**: Simulate network errors, timeouts, and 5xx responses from `/rest/internal/identity/resolve` and verify that token issuance follows the defined failure behavior and structured logs capture the failures.

### Tests for User Story 2

- [X] T014 [P] [US2] Add integration test in `test/integration/` simulating a temporary failure (e.g., 5xx) from the Alkemio identity endpoint and asserting that the service behavior and responses match the specification
- [X] T015 [P] [US2] Add unit or integration test in `test/integration/` or `test/unit/` to verify timeout behavior and structured logging when the identity resolution call exceeds configured timeouts

### Implementation for User Story 2

- [X] T016 [US2] Implement robust error handling in the Alkemio identity resolver (e.g., in `internal/challenge/` or `internal/alkemio/`) for network errors, timeouts, and non-success HTTP statuses
- [X] T017 [US2] Add structured logging (using Zap) for resolution attempts and failures in the relevant files (e.g., `internal/challenge/service.go` or the resolver helper), including correlation IDs but avoiding full identifiers where not necessary
- [X] T018 [US2] ~~Expose Prometheus metrics (e.g., counters or histograms) for identity resolution outcomes in `pkg/telemetry/` or appropriate middleware, and ensure they are incremented from the resolver logic~~ (removed Nov 2025; resolver now relies exclusively on structured logs)
- [X] T019 [US2] Confirm that maintenance mode logic still short-circuits requests before any external Alkemio calls by verifying control flow in `internal/middleware/maintenance.go` and related wiring
- [X] T020 [US2] Run `go test ./...` and update tests as needed to accommodate failure handling and observability changes

**Checkpoint**: User Stories 1 and 2 should both work independently and provide clear operational signals.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements and validation that affect multiple user stories.

- [X] T027 [P] Update feature-specific documentation in `specs/003-token-claims-oidc/quickstart.md` and, if needed, global docs in `docs/operations.md` to describe the new claim and failure behaviors
- [X] T028 Review and refactor any duplicated logic in `internal/challenge/` or a potential `internal/alkemio/` package to keep the implementation simple and maintainable
- [X] T029 [P] Add any additional unit tests in `test/unit/` for edge cases (e.g., rapid repeated resolution calls, unexpected response shapes) if they are not covered by existing integration tests
- [X] T030 Run `go test ./...` and any linting script (e.g., `scripts/lint.sh`) to ensure the codebase remains healthy

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories.
- **User Stories (Phases 3–5)**: All depend on Foundational phase completion.
  - User stories can proceed in priority order (P1 → P2 → P3).
  - Some tasks marked [P] can be run in parallel when they touch different files.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2); does not depend on other user stories.
- **User Story 2 (P2)**: Can start after Foundational (Phase 2); depends logically on the presence of the resolver and claim wiring from US1 but is still testable via its own failure scenarios.
- **User Story 3 (P3)**: Can start after Foundational (Phase 2); builds on the claim from US1 but focuses on scoping behavior and is testable independently via configuration and token inspection.

### Parallel Opportunities

- Phase 1 tasks T002 and T003 are marked [P] and can run in parallel once T001 has confirmed basic project health.
- Phase 2 tasks T005 and T006 are marked [P] and can be executed in parallel after T004.
- Within each user story phase, tasks marked [P] (primarily test additions) can be developed concurrently with implementation tasks that do not touch the same files.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1 (resolver, claim wiring, and basic tests).
4. STOP and validate by running the US1 tests and verifying tokens for a mapped user.

### Incremental Delivery

1. Deliver MVP (User Story 1) as above.
2. Add User Story 2 to harden failure handling and observability.
3. Add User Story 3 to refine privacy and scoping behavior.
4. Finish with Phase 6 polish tasks and documentation updates.
