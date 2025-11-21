# Data Snapshot – Agent ID Token Claim

- **Resolver Request**: oidc-service posts `{ "authenticationId": <kratos auth UUID> }` to `/rest/internal/identity/resolve`.
- **Resolver Response**: `{ "userId": <uuid>, "agentId": <uuid> }` — both fields are required canonical UUIDs; any deviation is treated as a failure.
- **Token Claims**:
  - `alkemio_user_id` ← `userId`
  - `agent_id` ← `agentId` (added by this feature)
  - ID tokens also continue to carry `email_verified`, `accepted_terms`, and optional profile traits.
  - Access tokens mirror `agent_id` plus `identity_id` for downstream lookups.
- **Persistence**: No new storage; all identifiers flow through existing in-memory profiles before Hydra finalizes tokens.
