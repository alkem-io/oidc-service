# Contracts: OIDC Service Bootstrap

This feature is governed by `contracts/openapi.yaml` at the repository root.

- Login contract: `POST /v1/oidc/login`
- Consent contract: `POST /v1/oidc/consent`
- Health contracts: `GET /health/live`, `GET /health/ready`
- Maintenance contract: `POST /maintenance`

Contract tests under `test/contract/*.go` assert that the live handlers conform to these schemas. Regenerate supporting fixtures via:

```bash
make contracts
```

Any changes to the Go handlers MUST be reflected in both the OpenAPI file and the corresponding contract tests before merge.
