# Quickstart – Docstring Coverage Uplift

> The CLI overwrites `docs/coverage.json` on every run. Copy it elsewhere if you need to compare historical results.

## Prerequisites
1. Go 1.25 installed (matching repo toolchain).
2. `golangci-lint` available locally (same version as CI).
3. GitHub token with workflow permissions if you plan to trigger CI runs.

## Local Workflow
1. **Generate coverage report**
   ```bash
   make docstring-coverage
   ```
   - Builds `cmd/docstringcov` if needed.
   - Produces `docs/coverage.json` and a human-readable summary in stdout.

2. **Inspect package gaps**
   ```bash
   jq '.packages[] | select(.coverage < 80) | {path, coverage, missing: .missingSymbols | length}' docs/coverage.json
   ```
   - Lists low-performing packages and how many exports need docstrings.

3. **Apply docstrings**
   - Use the `missingSymbols` entries to update the corresponding exported identifiers in source files.

4. **Re-run the tool**
   ```bash
   make docstring-coverage
   ```
   - Confirms the updated package coverage meets or exceeds 80%.

5. **Run tests**
   ```bash
   go test ./...
   golangci-lint run
   ```
   - Ensures the codebase remains stable after documentation changes.

## CI Workflow
1. Push a branch or open a PR.
2. GitHub Actions (optional) can invoke `make docstring-coverage` after lint/tests to publish the same JSON file as an artifact.
3. Developers rerun the command locally until overall and per-package coverage meet thresholds before merging.

## Troubleshooting
- **Formatting issues**: `reason` field explains whether the docstring is missing entirely or violates GoDoc conventions.
