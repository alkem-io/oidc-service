# Feature Tasks: Agent ID Token Claim

## Phase 1 – Setup

- [x] T001 Ensure local resolver tests use latest agent payload fixtures (`test/integration/alkemio_user_id_claim_test.go`)
- [x] T002 Refresh Go dependencies and verify toolchain per plan (`go.mod`, `go.sum`)

## Phase 2 – Foundational Work

- [x] T003 Update shared resolver client models to include `agentId` field (`internal/alkemio/identity_resolver.go`)
- [x] T004 Extend resolver client tests with agent coverage (`internal/alkemio/identity_resolver_test.go`)
- [x] T005 Add agent-aware helpers for test fixtures (`internal/challenge/test_helpers.go`, `test/support`)

## Phase 3 – User Story 1: Tokens expose agent identity (P1)

**Goal**: ID and access tokens include `agent_id` claim populated from resolver response

**Independent Test**: Trigger login/consent flow with a user whose resolver mapping includes `agentId`; decode tokens and assert the `agent_id` claim matches resolver payloads. Contract suite must pass verifying claim schema.

- [x] T006 [US1] Map resolver agent UUID into challenge model (`internal/challenge/model.go`)
- [x] T007 [US1] Validate resolver response rejects missing/malformed agent IDs (`internal/challenge/identity_mapper.go`)
- [x] T008 [US1] Unit tests for mapper validation logic (`internal/challenge/identity_mapper_test.go`)
- [x] T009 [US1] Include `agent_id` in claim assembly path (`internal/challenge/service.go`)
- [x] T010 [US1] Propagate claim through login handler (`internal/server/login_handler.go` & tests)
- [x] T011 [US1] Propagate claim through consent handler (`internal/server/consent_handler.go` & tests)
- [x] T012 [US1] Ensure Hydra adapter copies new claim (`internal/challenge/service.go`, verified pass-through client)
- [x] T013 [US1] Contract tests for ID/access tokens containing `agent_id` (`test/contract/id_token_claims_test.go`, `test/contract/access_token_claims_test.go`)
- [x] T014 [US1] Integration tests covering resolver success plus not-found/invalid-agent failures (`test/integration/alkemio_user_id_claim_test.go`, `test/integration/token_claims_missing_data_test.go`)
- [x] T015 [US1] Structured log entry updates for resolver success/failure (no new signals) (`internal/challenge/service.go`)

## Phase 4 – Polish & Cross-Cutting

- [x] T016 Update quickstart with verification steps already drafted (`specs/004-agent-claim/quickstart.md`)
- [x] T017 Document changelog/operations note referencing new claim (`docs/operations.md`)
- [x] T018 Verify lint/test scripts pass (`scripts/lint.sh`, `scripts/validate-token-claims.sh`)

## Dependencies & Execution Order

1. Setup (Phase 1)
2. Foundational resolver updates (Phase 2)
3. User Story 1 implementation & tests (Phase 3)
4. Polish & documentation (Phase 4)

## Parallel Execution Opportunities

- T003–T005 can run in parallel once setup completes (different files)
- Within User Story 1:
  - T006/T007/T008 can proceed while handler updates (T010/T011) occur
  - Contract tests (T013) may start after mapper + claim assembly (T006–T009)
  - Integration tests (T014) can run once handler paths compile

## Implementation Strategy

Deliverable MVP is Phase 3 completion (User Story 1). Ship incremental commits: foundational resolver updates → claim mapper → handler wiring → test suites. Polish tasks may land post-MVP.
