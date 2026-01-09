# Research: Kratos Webhooks

## Key Finding

**Kratos `response.parse: true` only works for registration and settings flows.**

For login and verification flows, the webhook response is NOT parsed to update identity. Must use Kratos Admin API instead.

## Solution

Use `PATCH /admin/identities/{id}` with JSON Patch (RFC 6902):

```json
[{
  "op": "add",
  "path": "/metadata_public",
  "value": {
    "alkemio_actor_id": "uuid",
    "alkemio_agent_id": "uuid"
  }
}]
```

Using `op: add` works whether `metadata_public` exists or is null.

## Kratos Configuration

Hooks must be configured **per method** (password, oidc), not just under `after:`.

```yaml
selfservice:
  flows:
    verification:
      after:
        hooks:
          - hook: web_hook
            config:
              url: http://oidc-service:8080/webhooks/kratos/post-login
              response:
                parse: false  # We update via Admin API
    login:
      after:
        password:
          hooks:
            - hook: web_hook
              ...
        oidc:
          hooks:
            - hook: web_hook
              ...
```

## Jsonnet Payload

```jsonnet
function(ctx) {
  identity_id: ctx.identity.id
}
```
