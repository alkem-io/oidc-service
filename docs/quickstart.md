# OIDC Service Quickstart

This guide walks through running the standalone OIDC Challenge Service locally
alongside the Synapse + Hydra + Kratos stack used in Alkemio development.

## Prerequisites

- Docker Desktop 4.30+ (Docker Engine 24+) with Compose v2
- Access to pull `docker.io/alkemio/oidc-service`
- The Alkemio server quickstart stack (`quickstart-services.yml`) from the
  root of the `alkem-io/server` repository
- A configuration file based on `configs/env.sample`

## 1. Pull required images

> The following Docker Compose commands assume you are running them from the
> `alkem-io/server` repository root where `quickstart-services.yml` lives. If
> you're inside `oidc-service/`, run `cd ..` first or adjust the `-f` path.

```sh
docker compose -f quickstart-services.yml pull hydra kratos mailslurper synapse
docker pull docker.io/alkemio/oidc-service:latest
```

## 2. Prepare environment configuration

```sh
cp configs/env.sample .env.oidc
# Edit .env.oidc with minimal required variables.
#
# Minimal required (internal addresses):
#   OIDC_HYDRA_ADMIN_URL=http://hydra:4445
#   OIDC_KRATOS_ADMIN_URL=http://kratos:4434
#   OIDC_KRATOS_PUBLIC_URL=http://kratos:4433
#   OIDC_WEB_BASE_URL=http://localhost:3000  # public origin for return_to
#
# Optional:
#   # If unset, the service infers scheme/host from X-Forwarded-* and falls back safely.
#   # Set only when you must force a specific external auth host.
#   # OIDC_KRATOS_BROWSER_URL=http://localhost:3000/ory/kratos/public
#
#   # Override return base (defaults to ${OIDC_WEB_BASE_URL}/oidc/login)
#   # OIDC_LOGIN_RETURN_BASE_URL=http://localhost:3000/oidc/login
```

## 3. Launch supporting services

```sh
docker compose -f quickstart-services.yml up -d hydra kratos mailslurper synapse
```

## 4. Run the OIDC Service container

```sh
docker run --rm \
  --env-file .env.oidc \
  --name oidc-service \
  -p 8080:8080 \
  docker.io/alkemio/oidc-service:latest
```

## 5. Verify readiness

```sh
curl -sSf http://localhost:8080/health/ready | jq
```

A healthy service returns a payload similar to:

```json
{"status":"ready","hydra":"ok","kratos":"ok"}
```

## 6. Exercise the login flow

1. Mint a real Hydra login challenge. With the quickstart stack the Hydra
   public endpoint is `http://localhost:4444`, Synapse is
   `http://localhost:8008`, and the client ID is exposed via the
   `SYNAPSE_OIDC_CLIENT_ID` environment variable:

    ```sh
    LOGIN_CHALLENGE=$(curl -s -D - -o /dev/null \
      -G "${HYDRA_PUBLIC_URL:-http://localhost:4444}/oauth2/auth" \
      --data-urlencode client_id="${SYNAPSE_OIDC_CLIENT_ID}" \
      --data-urlencode redirect_uri="${SYNAPSE_PUBLIC_URL:-http://localhost:8008}/_synapse/client/oidc/callback" \
      --data-urlencode response_type=code \
      --data-urlencode scope="openid profile email" \
      --data-urlencode state=local-cli-test \
      --data-urlencode prompt=login \
      | awk -F'login_challenge=' '/^Location:/ {split($2,v,"&"); print v[1]; exit}')
    echo "LOGIN_CHALLENGE=${LOGIN_CHALLENGE}"
    ```

   Hydra responds with `302 Found`; the inline `awk` command extracts the
   `login_challenge` query value from the `Location` header and stores it in the
   shell variable shown in the final `echo`.

2. Call the OIDC service with the captured challenge (public endpoints are under `/oidc`):

    ```sh
  curl -i "http://localhost:8080/oidc/login?login_challenge=${LOGIN_CHALLENGE}"
    ```

Expect an HTTP `302` redirect when Hydra accepts the challenge. If your browser
has no Kratos session, the service redirects you to the Kratos browser login at
`/ory/kratos/public/self-service/login/browser` with `return_to` back to
`${OIDC_WEB_BASE_URL}/oidc/login?...`. Structured `4xx` errors return only to
API clients (e.g., curl) for invalid challenges.

### Traefik & Hydra wiring (dev stack)

- Traefik: route `/oidc/*` directly to the OIDC service.
- Hydra: set `URLS_LOGIN=${HYDRA_PUBLIC_URL}/oidc/login` and
  `URLS_CONSENT=${HYDRA_PUBLIC_URL}/oidc/consent`.

## 7. Toggle maintenance mode

```sh
docker run --rm \
  --env-file .env.oidc \
  -e OIDC_MAINTENANCE_MODE=true \
  -p 8080:8080 \
  docker.io/alkemio/oidc-service:latest
# regenerate LOGIN_CHALLENGE by rerunning the Step 6 command
curl -i "http://localhost:8080/oidc/login?login_challenge=${LOGIN_CHALLENGE}"
```

During maintenance the service responds with HTTP `503` and a JSON body
explaining the outage window. Hydra invalidates challenges after use, so rerun
the Step 6 command to obtain a fresh token before each invocation.

## 8. Validate Token Claims Implementation

The service now includes comprehensive token claims support with `given_name`, `family_name`, `email_verified`, and `accepted_terms` claims in both Access and ID tokens.

### Quick Token Claims Test

```sh
# Run the comprehensive validation script
./scripts/validate-token-claims.sh
```

This script validates:
- Complete user profiles with all claims
- Email verification status handling
- Terms acceptance status handling  
- Unicode name support

### Manual Token Claims Verification

For manual testing, examine the token claims after completing the login flow:

```sh
# After step 6, decode the returned tokens to verify claims:
# Access Token will include: given_name, family_name
# ID Token will include: given_name, family_name, email_verified, accepted_terms
```

Instead of scraping metrics, rely on structured zap logs (set `OIDC_LOG_LEVEL=debug`
when needed) to observe identity resolution and token claim extraction outcomes.

## 9. GitHub Actions overview

The legacy `ci.yml` and `release.yml` jobs have been removed. The service now
ships with the following workflows under `.github/workflows/`:

- `pr-build.yml` — builds the container image for pull requests against any path.
- `build-release-docker-hub.yml` — publishes `docker.io/alkemio/oidc-service`
  when a release is published.
- `build-deploy-k8s-dev-hetzner.yml` — builds the image and deploys to the dev
  Hetzner cluster on pushes to `develop`.
- `build-deploy-k8s-sandbox-hetzner.yml` — manual deployment to the sandbox
  Hetzner environment via `workflow_dispatch`.
- `build-deploy-k8s-test-hetzner.yml` — manual deployment to the test Hetzner
  environment via `workflow_dispatch`.
- `schema-contract.yml` — lints the OpenAPI contract and diff-checks pull
  requests touching `contracts/` or documentation.
- `trigger-e2e-tests.yml` — notifies the external `alkem-io/test-suites`
  repository to execute end-to-end verification on release publication.

Consult the repository secrets to ensure Docker registry and Travis API tokens
are available before invoking the release or deployment workflows.

## Troubleshooting

- Readiness showing `unreachable` indicates Hydra or Kratos is misconfigured—
verify URLs and admin tokens in `.env.oidc`.
- `INVALID_CHALLENGE` 404 responses mean the challenge has expired in Hydra; use
`hydra tokens introspect` to confirm state.
- Set `OIDC_LOG_LEVEL=debug` to gain additional zap logs when investigating
failures.
