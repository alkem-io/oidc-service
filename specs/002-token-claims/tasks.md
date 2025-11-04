# Tasks: Token Claims Enhancement

**Input**: Design documents from `/specs/002-token-claims/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Contract tests are included as specified in the contracts documentation

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure: Go backend service with domain-oriented packages in `internal/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Analyze current identity mapping implementation in internal/challenge/identity_mapper.go
- [ ] T002 [P] Review current token generation flow in internal/challenge/service.go
- [ ] T003 [P] Examine Kratos client integration patterns in internal/kratos/client.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Extend TokenClaims model structure in internal/challenge/model.go to support new claim fields
- [X] T005 [P] Add Kratos identity trait extraction utilities in internal/challenge/identity_mapper.go
- [X] T006 [P] Implement safe claim validation functions for Kratos identity traits in internal/challenge/identity_mapper.go
- [X] T007 Add structured logging support for token claim operations in internal/challenge/service.go
- [X] T008 Update Prometheus metrics to track token claim generation in pkg/telemetry/metrics.go
- [X] T009 [P] Create integration test for token size limit failure in test/integration/token_size_limit_test.go
- [X] T010 [P] Create integration test for Kratos unavailability during token generation in test/integration/kratos_unavailable_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Enhanced Token Claims for User Profile Data (Priority: P1) 🎯 MVP

**Goal**: Client applications can access standardized user profile information (given_name and family_name) from both Access tokens and ID tokens

**Independent Test**: Can be fully tested by examining token contents after authentication and verifying that given_name and family_name claims are present in both token types with correct user data

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T011 [P] [US1] Create contract test for Access token given_name/family_name claims that validates claim presence and accuracy in test/contract/access_token_claims_test.go
- [X] T012 [P] [US1] Create contract test for ID token given_name/family_name claims that validates claim presence and accuracy in test/contract/id_token_claims_test.go
- [X] T013 [P] [US1] Create integration test for missing Kratos identity traits handling that ensures claims are omitted in test/integration/token_claims_missing_data_test.go

### Implementation for User Story 1

- [X] T014 [P] [US1] Implement extractGivenNameClaim function that extracts and validates UTF-8 names in internal/challenge/identity_mapper.go
- [X] T015 [P] [US1] Implement extractFamilyNameClaim function that extracts and validates UTF-8 names in internal/challenge/identity_mapper.go
- [X] T016 [US1] Integrate name claim extraction into Access token generation flow that produces tokens with name claims in internal/challenge/service.go
- [X] T017 [US1] Integrate name claim extraction into ID token generation flow that produces tokens with name claims in internal/challenge/service.go
- [X] T018 [US1] Add error handling for missing Kratos identity traits that ensures graceful omission in internal/challenge/identity_mapper.go
- [X] T019 [US1] Add logging for name claim extraction operations that provides traceability in internal/challenge/service.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Compliance and Security Claims in ID Tokens (Priority: P2)

**Goal**: Client applications can determine user compliance and security status by examining the email_verified and accepted_terms claims in ID tokens

**Independent Test**: Can be fully tested by examining ID token contents and verifying that email_verified and accepted_terms claims accurately reflect the user's verification and terms acceptance status

### Tests for User Story 2

- [X] T020 [P] [US2] Create contract test for ID token email_verified claim that validates boolean accuracy in test/contract/id_token_email_verified_test.go
- [X] T021 [P] [US2] Create contract test for ID token accepted_terms claim that validates boolean accuracy in test/contract/id_token_accepted_terms_test.go
- [X] T022 [P] [US2] Create integration test for email verification status edge cases that ensures proper boolean logic in test/integration/email_verification_edge_cases_test.go
- [X] T023 [P] [US2] Create integration test for missing accepted_terms Kratos identity trait handling that ensures claim omission in test/integration/missing_accepted_terms_test.go

### Implementation for User Story 2

- [X] T024 [P] [US2] Implement extractEmailVerifiedClaim function that determines verification status from verifiable_addresses in internal/challenge/identity_mapper.go
- [X] T025 [P] [US2] Implement extractAcceptedTermsClaim function that extracts boolean terms acceptance in internal/challenge/identity_mapper.go
- [X] T026 [US2] Integrate email verification claim extraction into ID token generation flow that produces tokens with verification status in internal/challenge/service.go
- [X] T027 [US2] Integrate accepted terms claim extraction into ID token generation flow that produces tokens with terms status in internal/challenge/service.go
- [X] T028 [US2] Add error handling for missing compliance Kratos identity traits that ensures graceful omission in internal/challenge/identity_mapper.go
- [X] T029 [US2] Add logging for compliance claim extraction operations that provides audit trail in internal/challenge/service.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T030 [P] Update operational documentation in docs/operations.md with token claim monitoring guidance
- [X] T031 Code review and refactoring of claim extraction logic that ensures consistency across functions
- [X] T032 [P] Add comprehensive unit tests for edge cases that validate unicode handling and boundary conditions in test/unit/identity_mapper_claims_test.go
- [X] T033 Security review of PII handling in token claims that ensures compliance with data protection
- [X] T034 Run quickstart.md validation with complete token claim scenarios that verifies end-to-end functionality

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Integrates with US1 but should be independently testable

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Extraction functions before service integration
- Core implementation before error handling
- Error handling before logging
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, both user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Extraction functions within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Contract test for Access token given_name/family_name claims that validates claim presence and accuracy in test/contract/access_token_claims_test.go"
Task: "Contract test for ID token given_name/family_name claims that validates claim presence and accuracy in test/contract/id_token_claims_test.go"
Task: "Integration test for missing Kratos identity traits handling that ensures claims are omitted in test/integration/token_claims_missing_data_test.go"

# Launch all extraction functions for User Story 1 together:
Task: "Implement extractGivenNameClaim function that extracts and validates UTF-8 names in internal/challenge/identity_mapper.go"
Task: "Implement extractFamilyNameClaim function that extracts and validates UTF-8 names in internal/challenge/identity_mapper.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch all tests for User Story 2 together:
Task: "Contract test for ID token email_verified claim that validates boolean accuracy in test/contract/id_token_email_verified_test.go"
Task: "Contract test for ID token accepted_terms claim that validates boolean accuracy in test/contract/id_token_accepted_terms_test.go"
Task: "Integration test for email verification status edge cases that ensures proper boolean logic in test/integration/email_verification_status_test.go"
Task: "Integration test for missing accepted_terms Kratos identity trait handling that ensures claim omission in test/integration/accepted_terms_missing_test.go"

# Launch all extraction functions for User Story 2 together:
Task: "Implement extractEmailVerifiedClaim function that determines verification status from verifiable_addresses in internal/challenge/identity_mapper.go"
Task: "Implement extractAcceptedTermsClaim function that extracts boolean terms acceptance in internal/challenge/identity_mapper.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
3. Stories complete and integrate independently

---

## Task Summary

**Total Tasks**: 34
- Setup Phase: 3 tasks
- Foundational Phase: 7 tasks (includes error scenario tests)
- User Story 1: 9 tasks (3 tests + 6 implementation)
- User Story 2: 10 tasks (4 tests + 6 implementation)
- Polish Phase: 5 tasks

**Parallel Opportunities**: 22 tasks marked [P] can run in parallel with other tasks
**Independent Test Criteria**: Each user story has specific acceptance criteria from spec.md
**Suggested MVP Scope**: User Story 1 only (given_name and family_name claims in both tokens)

---

## Notes

- [P] tasks = different files, no dependencies within same phase
- [Story] label maps task to specific user story for traceability  
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Follow Go 1.25 and constitutional patterns throughout implementation