# Data Model: Alkemio User ID Token Claim

**Feature**: `003-token-claims-oidc`
**Date**: 2025-11-17

## Entities

### Kratos Identity
- **Identifier**: `kratos_id` (string)
- **Description**: Identity ID from the authentication system.
- **Key fields used**:
  - `kratos_id`: unique per user.

### Alkemio User
- **Identifier**: `alkemio_user_id` (string)
- **Description**: Canonical user identifier in the Alkemio platform, surfaced by the Alkemio server as `userId` in the response of `POST /rest/internal/identity/resolve`.
- **Relationships**:
  - Linked 1:1 to a `kratos_id` for fully onboarded users.

### Token Claims Set
- **Identifier**: n/a (logical structure inside a token)
- **Description**: Collection of claims attached to access tokens and ID tokens.
- **Key fields**:
  - Existing identity and session claims as defined today.
  - New field: `alkemio_user_id` (string) present only when the identity resolution endpoint returns 200 and token issuance is allowed.

## Relationships & Constraints

- Each `Kratos Identity` **must** have exactly one `Alkemio User` mapping (as resolved via `/rest/internal/identity/resolve` using the authentication ID) to receive a token; otherwise token issuance fails.
- `alkemio_user_id` in the Token Claims Set must match the `userId` value returned by the Alkemio server for the given authentication ID.
- Inconsistent mappings (e.g., more than one `alkemio_user_id` for the same underlying Kratos identity) MUST cause token issuance to fail.

## Lifecycle Considerations

- Mapping creation and maintenance are handled outside oidc-service (in Alkemio server); this feature only consumes the mapping.
- If mappings change, subsequent token issuance will reflect the current mapping returned by the Alkemio server.
