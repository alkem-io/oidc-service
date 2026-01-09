# Tasks: Kratos Identity Metadata Webhook

**Status**: All Complete

## Phase 1: Breaking Change
- [x] Rename `alkemio_user_id` → `alkemio_actor_id` in model.go
- [x] Update service.go logging
- [x] Update all test files

## Phase 2: Webhook Handler
- [x] Create `internal/webhook/handler.go` with PostLogin handler
- [x] Create `internal/webhook/kratos_admin.go` for Admin API PATCH
- [x] Add UUID validation for resolved claims
- [x] Add comprehensive unit tests

## Phase 3: Integration
- [x] Register `/webhooks/kratos/post-login` route
- [x] Wire Kratos admin client in app.go

## Phase 4: Validation
- [x] All tests passing
- [x] Lint clean
- [x] Docstring coverage complete
- [x] Manual integration test successful
