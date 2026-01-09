# Alkemio OIDC Challenge Service

Standalone Go 1.25 service that resolves Hydra login and consent challenges, fetches identity traits from Kratos, and redirects Synapse clients with the same semantics as the legacy NestJS implementation.

## Endpoints

- Public: `/oidc/login`, `/oidc/consent`
- Webhooks: `/webhooks/kratos/post-login`, `/webhooks/kratos/post-registration`
- Health: `/health/ready`, `/health/live`

### Kratos Webhooks for Alkemio Claims

Two webhook endpoints resolve Alkemio identity claims (`alkemio_actor_id`, `alkemio_agent_id`) and store them in `identity.metadata_public`:

- **`/webhooks/kratos/post-registration`** - Returns claims in response; Kratos parses and stores them
- **`/webhooks/kratos/post-login`** - Calls Kratos Admin API to patch identity metadata

**Jsonnet payload** (`configs/kratos/alkemio-claims.jsonnet`):

```jsonnet
function(ctx) {
  identity_id: ctx.identity.id
}
```

**Kratos configuration** (`kratos.yml`):

Alkemio creates the user record after email verification, so webhooks are configured for:
- **Post-verification**: Resolves claims when user completes email verification
- **Post-login**: Refreshes claims on each login

```yaml
selfservice:
  flows:
    verification:
      after:
        hooks:
          - hook: web_hook
            config:
              url: http://oidc-service:8080/webhooks/kratos/post-login
              method: POST
              body: file:///etc/config/kratos/alkemio-claims.jsonnet
              response:
                ignore: false
                parse: false   # We update via Admin API

    login:
      after:
        # Hooks must be configured per login method
        password:
          hooks:
            - hook: web_hook
              config:
                url: http://oidc-service:8080/webhooks/kratos/post-login
                method: POST
                body: file:///etc/config/kratos/alkemio-claims.jsonnet
                response:
                  ignore: false
                  parse: false   # We update via Admin API
        oidc:
          hooks:
            - hook: web_hook
              config:
                url: http://oidc-service:8080/webhooks/kratos/post-login
                method: POST
                body: file:///etc/config/kratos/alkemio-claims.jsonnet
                response:
                  ignore: false
                  parse: false
```

Resolution failures return HTTP 500 to block login/registration (sessions must have claims).

On `session_required`/`session_invalid`, the service redirects browsers to the
Kratos login flow and returns to `/oidc/login` after authentication.

## OpenAPI reference

- Specification source lives in `contracts/openapi.yaml` and stays in sync with the handlers under `internal/server/`.
- Preview the documentation with Swagger UI by running:

	```sh
	docker run --rm -p 8089:8080 \
		-e SWAGGER_JSON=/tmp/openapi.yaml \
		-v "$(pwd)/contracts/openapi.yaml:/tmp/openapi.yaml" \
		swaggerapi/swagger-ui
	```

	Then visit `http://localhost:8089` in your browser.
- When editing the spec, validate it locally with `docker run --rm -v "$(pwd)/contracts:/tmp" redocly/cli lint /tmp/openapi.yaml`.

## Configuration

Minimal required variables (internal addresses):

- `OIDC_HYDRA_ADMIN_URL` (e.g. `http://hydra:4445`)
- `OIDC_KRATOS_ADMIN_URL` (e.g. `http://kratos:4434`)
- `OIDC_KRATOS_PUBLIC_URL` (e.g. `http://kratos:4433`)
- `OIDC_WEB_BASE_URL` (e.g. `http://localhost:3000`)

Optional:

- `OIDC_KRATOS_BROWSER_URL` for browser redirects. If unset, the service uses
  forwarded headers from your reverse proxy.
- `OIDC_LOGIN_RETURN_BASE_URL` to override the default `${OIDC_WEB_BASE_URL}/oidc/login`.

Database (optional, for fast identity resolution):

- `DATABASE_HOST` (default: `localhost`)
- `DATABASE_PORT` (default: `5432`)
- `DATABASE_USERNAME` (default: `synapse`)
- `DATABASE_PASSWORD` (default: `synapse`)
- `DATABASE_NAME` (default: `alkemio`)
- `DATABASE_TIMEOUT` (default: `5s`)

When database is configured, the service queries the Alkemio database directly
for identity mappings (UserID, AgentID) before falling back to the HTTP API.
This reduces latency and load on the Alkemio server. If the database is
unavailable, the service gracefully falls back to API-only mode.

See `configs/env.sample` for a documented template.

## Development

Build and test:

```sh
make build      # Build all binaries
make test       # Run test suite
make lint       # Run linters
make fmt        # Format code
```

Code generation (after modifying SQL queries):

```sh
make sqlc       # Generate type-safe Go code from SQL
make generate   # Run all code generators
```

## Docstring Coverage

Run the internal coverage CLI before pushing docstring changes to keep the
repository above the enforced threshold:

```sh
make docstring-coverage
```

The target builds `cmd/docstringcov`, prints overall/per-package coverage, and
writes the structured report to `docs/coverage.json`. The command exits with a
non-zero status if overall coverage or any package drops below the configured
`OIDC_DOCSTRING_COVERAGE_THRESHOLD` (defaults to 80%). Supply extra flags via
`DOCSTRINGCOV_FLAGS` when you need to override the threshold or scan a different
root, e.g. `make docstring-coverage DOCSTRINGCOV_FLAGS="--threshold 90"`.
