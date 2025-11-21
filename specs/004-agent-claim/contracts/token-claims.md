# Contract: ID & Access Token Claims

## Overview
Both ID and access tokens issued by oidc-service MUST include the `agent_id` claim whenever the Alkemio resolver returns an `{ userId, agentId }` pair for the authenticated Kratos session. Tokens are rejected if the resolver fails or returns invalid identifiers.

## Claim Schema
| Claim      | Type  | Source                                 | Required | Description |
|------------|-------|-----------------------------------------|----------|-------------|
| `agent_id` | UUID  | `agentId` from `/rest/internal/identity/resolve` | Yes (when resolver succeeds) | Identifies the Alkemio agent representing the authenticated user. |
| `alkemio_user_id`  | UUID  | `userId` from resolver                   | Yes      | Existing user identifier (unchanged). |
| `sub`      | UUID  | Kratos identity ID                      | Yes      | Existing subject claim. |
| `aud`      | Array | Hydra client                            | Yes      | Existing OIDC audience claim. |
| `exp`      | Int   | Hydra-issued expiry                      | Yes      | Token expiration timestamp. |
| `iat`      | Int   | Hydra-issued issued-at                   | Yes      | Token issuance timestamp. |

## Example Payloads

### ID Token (truncated)
```json
{
  "iss": "https://oidc-service/",
  "sub": "8c0b7f5a-4dce-4d13-b6ad-6b2df2c1d10c",
  "aud": ["alkemio-client"],
  "exp": 1732125100,
  "iat": 1732121500,
  "alkemio_user_id": "1bcf5bd1-5f3e-4f01-9125-0edc93e5f5b1",
  "agent_id": "6f4ed2d2-0ad0-4b83-8d43-8d9b9b4970b3"
}
```

### Access Token (truncated)
```json
{
  "aud": ["alkemio-api"],
  "sub": "8c0b7f5a-4dce-4d13-b6ad-6b2df2c1d10c",
  "scope": "openid profile",
  "exp": 1732123300,
  "alkemio_user_id": "1bcf5bd1-5f3e-4f01-9125-0edc93e5f5b1",
  "agent_id": "6f4ed2d2-0ad0-4b83-8d43-8d9b9b4970b3"
}
```

## Failure Modes
- Resolver returns error/404 → token issuance aborted; contract guarantees `agent_id` never omitted silently.
- Resolver returns invalid UUID → issuance aborted with structured log entry citing validation failure.
