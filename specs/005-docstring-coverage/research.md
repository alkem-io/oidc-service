# Research – Docstring Coverage Uplift

## Decision 1 — Build our own analyzer
- **Decision**: Implement `cmd/docstringcov` on top of `go/ast` + `go/packages` instead of leaning on existing linters.
- **Why**: Gives full control over what “documented” means, supports per-package metrics, and keeps the workflow deterministic. `golint`/`revive` only emit warnings and cannot report coverage percentages without fragile parsing.

## Decision 2 — Single JSON artifact (for now)
- **Decision**: Persist the latest run at `docs/coverage.json`, overwriting it each time.
- **Why**: Keeps the repo noise-free and makes diffs obvious. History would be useful, but the immediate need was a single source of truth that developers and CI can read quickly. Future work can add `docs/coverage/history/` rotation if trend analysis becomes important.
- **Alternatives**: GitHub Action artifacts (expire), external dashboards (new cost/infra), or database storage (operational overhead) were all rejected for this first iteration.

## Decision 3 — Enforce via CLI threshold
- **Decision**: Let the CLI enforce the ≥80% policy using `--threshold` and exit codes; CI integration will call the same command, but adoption starts locally.
- **Why**: Keeps the rule close to the implementation, avoids duplicating logic in linters, and makes failures actionable (package list + missing symbols). Once the team is comfortable with the workflow we can wire it into a dedicated GitHub Actions job.

## Decision 4 — GoDoc-compliant style rules
- **Decision**: Treat any exported symbol without a proper GoDoc sentence (identifier prefix, punctuation) as undocumented.
- **Why**: Aligns with pkg.go.dev rendering expectations and discourages placeholder comments. A looser definition would meet the numeric target but produce low-quality documentation.
