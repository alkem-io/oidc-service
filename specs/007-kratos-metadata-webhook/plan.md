# Implementation Plan: Kratos Identity Metadata Webhook

**Status**: Complete

## Phases

### Phase 1: Breaking Change
- Renamed `alkemio_user_id` → `alkemio_actor_id` in TokenClaims and tests

### Phase 2: Webhook Handler
- Created `internal/webhook/` package with handler, types, Kratos admin client
- Implemented identity resolution via CompositeResolver
- Added UUID validation for resolved claims

### Phase 3: Integration
- Registered route in `internal/server/router.go`
- Wired Kratos admin client in `internal/app/app.go`

### Phase 4: Validation
- All tests passing
- Lint clean
- Docstring coverage 100% for webhook package
- Manual integration test with Kratos successful

## Key Learning

Kratos `response.parse: true` only works for registration/settings flows. For login and verification flows, must use Kratos Admin API to PATCH identity metadata directly.
