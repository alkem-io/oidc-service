# Tasks: DB-First Claims Resolution

**Status**: Completed
**Date**: 2026-01-08

All tasks completed successfully. Tests pass, build compiles.

## Summary

| Phase | Tasks | Status |
|-------|-------|--------|
| Setup | T001-T004 | Done |
| Foundational | T005-T007 | Done |
| US1: DB Resolution | T008-T010 | Done |
| US2: API Fallback | T011-T014 | Done |
| US3: Error Handling | T015-T017 | Done |
| Polish | T018-T020 | Done |

## Task Details

### Phase 1: Setup
- [x] T001 Add pgx v5 dependency
- [x] T002 Create SQLC config (`sqlc.yaml`)
- [x] T003 Create SQL queries (`internal/alkemio/queries/queries.sql`)
- [x] T004 Run SQLC code generation

### Phase 2: Foundational
- [x] T005 Add database config to `internal/config/config.go`
- [x] T006 Create database pool in `internal/alkemio/db.go`
- [x] T007 Wire database pool in `cmd/server/main.go`

### Phase 3: US1 - Database Resolution
- [x] T008 Implement DatabaseResolver in `internal/alkemio/db_resolver.go`
- [x] T009 Add UUID validation for returned values
- [x] T010 Write integration test for DB hit path

### Phase 4: US2 - API Fallback
- [x] T011 Implement CompositeResolver in `internal/alkemio/composite_resolver.go`
- [x] T012 Handle pgx.ErrNoRows for API fallback
- [x] T013 Update main.go to use CompositeResolver
- [x] T014 Write integration test for DB miss path

### Phase 5: US3 - Error Handling
- [x] T015 Add warning log on DB error before fallback
- [x] T016 Handle NULL/empty values as cache miss
- [x] T017 Write integration test for DB error path

### Phase 6: Polish
- [x] T018 Run existing integration tests (backward compatibility)
- [x] T019 Run all tests
- [x] T020 Verify clean build
