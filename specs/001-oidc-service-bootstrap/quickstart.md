# Quickstart: OIDC Service Bootstrap

This guide demonstrates how to run the OIDC service locally, execute the contract tests, and simulate the Hydra/Kratos login journey end-to-end.

## Prerequisites

- Go 1.22+
- Docker Desktop (for Hydra/Kratos dependencies)
- `make`, `curl`, and `jq`

## 1. Start Dependencies

```bash
# From the repository root
make dev-up
# or run docker compose manually
# docker compose -f deployments/docker-compose/compose.yaml up -d
```

Services started:
- Hydra Admin/API on `http://localhost:4445`
- Kratos public API on `http://localhost:4433`
- MailSlurper, Redis, MySQL for supporting flows

## 2. Run the OIDC Service

```bash
cd cmd/server
GOENV=production go run .
# Service listens on :8080 by default
```

Override configuration using environment variables, e.g.

```bash
OIDC_HYDRA_ADMIN_URL=http://localhost:4445 \
OIDC_KRATOS_PUBLIC_URL=http://localhost:4433 \
OIDC_KRATOS_BROWSER_URL=http://localhost:4433 \
OIDC_LOGIN_RETURN_BASE_URL=http://localhost:8080/oidc/login \
OIDC_SESSION_COOKIE=ory_kratos_session \
OIDC_MAINTENANCE_SHARED_SECRET=dev-secret \
go run .
```

## 3. Acquire a Real Hydra Login Challenge

```bash
LOGIN_CHALLENGE=$(curl -s -L \
  "http://localhost:3000/oidc/login" \
  -c /tmp/kratos.cookie | \
  awk -F'login_challenge=' 'NF>1 {split($2, a, "&"); print a[1]}' )

echo "$LOGIN_CHALLENGE"
```

## 4. Call the Login Endpoint

```bash
curl -s \
  -X POST "http://localhost:8080/oidc/login" \
  -H "Content-Type: application/json" \
  -H "Cookie: ory_kratos_session=$(grep ory_kratos_session /tmp/kratos.cookie | awk '{print $7}')" \
  -d "{\"challengeId\": \"$LOGIN_CHALLENGE\", \"accept\": true}" | jq
```

Expected response:

```json
{
  "redirect_to": "https://...",
  "remember": true,
  "remember_for": 3600
}
```

## 5. Toggle Maintenance Mode

```bash
curl -X POST "http://localhost:8080/maintenance" \
  -H "X-Maintenance-Secret: dev-secret" \
  -d '{"active": true, "reason": "Rolling restart"}' | jq
```

Subsequent login attempts will yield HTTP 503 with code `MAINTENANCE_MODE_ENABLED`.

## 6. Health & Metrics

```bash
curl -s http://localhost:8080/health/live | jq
curl -s http://localhost:8080/health/ready | jq
curl -s http://localhost:8080/metrics | head -n 20
```

## 7. Run Tests

```bash
go test ./...
# Contract tests
GO_TEST_CONTRACT=1 go test ./test/contract -run TestLoginContract
```

## 8. Tear Down

```bash
make dev-down
# or docker compose down
```

## Troubleshooting

- **INVALID_CHALLENGE**: Ensure you generated a fresh challenge via the Hydra login endpoint (`/oidc/login`).
- **KRATOS_SESSION_INVALID**: Verify the session cookie is present and not expired (`/sessions/whoami`).
- **Hydra 500 errors**: Check Hydra logs (`docker logs alkemio_dev_hydra`).
- **Metrics missing**: Confirm `OIDC_METRICS_ENABLED=true` and scrape path `/metrics` exposed.
