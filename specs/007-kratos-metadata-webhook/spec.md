# Feature: Kratos Identity Metadata Webhook

**Status**: Implemented
**Branch**: `007-kratos-metadata-webhook`

## Summary

Webhook endpoint that resolves Alkemio identity claims (`alkemio_actor_id`, `alkemio_agent_id`) and stores them in Kratos `identity.metadata_public`. Called after email verification and on each login.

## Architecture

```
User verifies email / logs in
        ↓
Kratos (after hook) → POST /webhooks/kratos/post-login
        ↓
OIDC Service: Resolve identity via CompositeResolver
        ↓
PATCH /admin/identities/{id} → Kratos Admin API
        ↓
Session includes metadata_public → Oathkeeper → Backend
```

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Use Admin API PATCH | Kratos `response.parse: true` only works for registration/settings flows, not login/verification |
| Post-verification hook | Alkemio creates user record after email verification, not during registration |
| JSON Patch `op: add` | Works whether `metadata_public` exists or is null |
| HTTP 500 on failure | Sessions MUST have claims; resolution failures block login |

## Endpoints

| Endpoint | Trigger | Action |
|----------|---------|--------|
| `POST /webhooks/kratos/post-login` | Verification complete, Login | PATCH identity via Admin API |

## Breaking Change

Renamed OIDC token claim: `alkemio_user_id` → `alkemio_actor_id`

## Files

- `internal/webhook/handler.go` - Webhook handlers
- `internal/webhook/kratos_admin.go` - Kratos Admin API client
- `internal/webhook/handler_test.go` - Tests
- `configs/kratos/alkemio-claims.jsonnet` - Webhook payload template
