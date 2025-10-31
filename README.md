# Alkemio OIDC Challenge Service

Standalone Go 1.25 service that resolves Hydra login and consent challenges, fetches identity traits from Kratos, and redirects Synapse clients with the same semantics as the legacy NestJS implementation.

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
