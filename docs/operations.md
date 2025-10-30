# Operations Runbook

## Maintenance Mode Toggle

Use the `OIDC_MAINTENANCE_MODE` environment variable to place the service in
maintenance. The toggle blocks challenge handling and returns HTTP 503 with an
optional `Retry-After` header derived from `OIDC_MAINTENANCE_RETRY`.

1. **Toggle via Docker Compose**
   ```sh
   docker compose -f quickstart-services.yml up -d oidc-service \
     && docker compose exec oidc-service sh -c 'export OIDC_MAINTENANCE_MODE=true'
   ```
   Restart the container to apply the updated environment.

2. **Toggle via Kubernetes**
   - Patch the deployment with `kubectl set env deployment/oidc-service OIDC_MAINTENANCE_MODE=true`.
   - Optionally set `OIDC_MAINTENANCE_RETRY=600` to emit a 10 minute retry hint.

3. **Validate state**
   ```sh
   curl -i http://OIDC_SERVICE_HOST/health/ready
   curl -i "http://OIDC_SERVICE_HOST/v1/oidc/login?login_challenge=test"
   ```
   Expect `HTTP 503` with `{"error":"maintenance_mode"...}` while maintenance is
   enabled.

4. **Disable maintenance** by removing or setting the toggle to `false` and
   redeploying.

## Dependency and CI Audits

- **Go module audit (2025-10-30)**: `go list -u -m` surfaced
  `github.com/ory/client-go v1.22.7` as the latest stable release; the module was
  updated via `go get` and committed.
- **GitHub Actions audit (2025-10-30)**: `.github/workflows/ci.yaml` pins current
  stable actions (`actions/checkout@v4`, `actions/setup-go@v5`,
  `actions/cache@v4`, `golangci/golangci-lint-action@v4`,
  `docker/setup-qemu-action@v3`, `docker/setup-buildx-action@v3`,
  `docker/build-push-action@v5`, `sigstore/cosign-installer@v3`). The syft
  container used for SBOM generation is pinned to `anchore/syft:v1.15.0`.
- **Re-run audit** after each release by executing `scripts/lint.sh`,
  reviewing the output of `go list -u -m all | grep '\['` for upgrades, and
  checking action release notes for new major versions.
