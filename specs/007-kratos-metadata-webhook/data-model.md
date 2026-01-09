# Data Model: Kratos Identity Metadata Webhook

## Request (from Kratos)

```json
{
  "identity_id": "kratos-identity-uuid"
}
```

## Response (success)

```json
{
  "status": "ok"
}
```

## Response (error)

```json
{
  "error": "resolution_failed",
  "message": "failed to resolve Alkemio identity"
}
```

| Status | Condition | Result |
|--------|-----------|--------|
| 200 | Success | Identity patched via Admin API |
| 400 | Malformed request | Login blocked |
| 500 | Resolution failed | Login blocked |

## Internal Flow

```
Webhook Request
      ↓
Extract identity_id
      ↓
CompositeResolver.Resolve(identity_id)
      ↓
Validate UUIDs
      ↓
KratosAdmin.PatchIdentity() → PATCH /admin/identities/{id}
      ↓
Return success/error
```

## Metadata Stored

```json
{
  "metadata_public": {
    "alkemio_actor_id": "alkemio-user-uuid",
    "alkemio_agent_id": "alkemio-agent-uuid"
  }
}
```
