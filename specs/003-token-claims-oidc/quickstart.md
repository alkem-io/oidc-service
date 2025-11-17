# Quickstart: Alkemio User ID Token Claim

This feature adds an `alkemio_user_id` claim to tokens issued by oidc-service when a corresponding Alkemio user mapping exists for the users Kratos identity.

## Prerequisites

- Alkemio server running and exposing `/rest/internal/identity/resolve` with valid mappings from Kratos IDs to Alkemio user IDs.
- oidc-service configured to reach the Alkemio server (base URL and any required authentication configured via `internal/config`).
- Existing Hydra/Kratos setup for authentication.

## Basic Flow

1. Sign in a user through the usual Kratos-based authentication flow.
2. Ensure the user has a valid mapping from `kratos_id` to `alkemio_user_id` in the Alkemio server (both identifiers must be UUIDs).
3. Request an access token and ID token from oidc-service.
4. Inspect the tokens:
   - Verify that `alkemio_user_id` is present and matches the mapping.

> **Note**: There is no client or flow-based scoping. Every token the service issues includes `alkemio_user_id` when a mapping exists.

## Failure Scenarios

- If no mapping exists:
  - Token issuance fails; investigate and create the mapping in the Alkemio server.
- If multiple mappings exist for the same Kratos ID:
  - Token issuance fails; resolve data inconsistency.
- If the identity resolution endpoint fails (network/5xx):
  - Token issuance follows the defined failure behavior in the implementation (see logs for details).

## Observability

- Check structured logs for resolution attempts and failures (set `OIDC_LOG_LEVEL=debug` when triaging).
