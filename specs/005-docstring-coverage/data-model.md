# Data Model – Docstring Coverage Uplift

`cmd/docstringcov` emits a single JSON payload at `docs/coverage.json`. The schema mirrors the Go structs in `internal/docstringcov/analyzer` and should remain stable so downstream tooling (reports, CI checks) can parse it.

## Result (root object)
- `generatedAt` *(RFC3339 string)* – timestamp emitted by the CLI.
- `overallCoverage` *(float, 0–100)* – documented exports divided by total exports.
- `documentedExports` *(int)* – count of exports with valid GoDoc comments.
- `totalExports` *(int)* – number of exported declarations inspected.
- `threshold` *(float)* – policy target applied to the run.
- `packages` *(array<Package>)* – ordered list of package-level metrics.

## Package
- `path` *(string)* – module-relative package path (`internal/challenge`).
- `coverage` *(float)* – documented ÷ total × 100.
- `documented` *(int)* – exports that passed the style checks.
- `undocumented` *(int)* – exports missing or violating docstrings.
- `missingSymbols` *(array<MissingSymbol>)* – detailed breakdown for each uncovered export; empty when coverage is 100%.

## MissingSymbol
- `symbol` *(string)* – identifier name or method receiver notation.
- `file` *(string)* – module-relative `.go` file path.
- `line` *(int)* – 1-based line number of the declaration.
- `reason` *(enum string)* – failure classification (`missing`, `mismatched-prefix`, `format`, etc.).

## Notes
- Numbers are deterministic for a given checkout because the analyzer uses `go/packages` in module mode.
- Schema intentionally avoids versioning; if fields change we will bump the CLI and capture details in this document.
- Historical retention is out of scope for v1; only the latest JSON is tracked.
```}