# Alkemio OIDC Challenge Service

Standalone Go 1.25 service that resolves Hydra login and consent challenges, fetches identity traits from Kratos, and redirects Synapse clients with the same semantics as the legacy NestJS implementation.

## Endpoints

- Public: `/oidc/login`, `/oidc/consent`
- Health: `/health/ready`, `/health/live`
- Metrics: `/metrics`

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

See `configs/env.sample` for a documented template.
