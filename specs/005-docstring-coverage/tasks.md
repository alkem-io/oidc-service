---

description: "Lean task list for Docstring Coverage Uplift"
---

# Tasks: Docstring Coverage Uplift

**Input**: Design documents from `/specs/005-docstring-coverage/`
**Prerequisites**: `plan.md`, `spec.md`, `quickstart.md`

**Organization**: Tasks are grouped by user story so each slice can be delivered and tested independently.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the minimal documentation scaffolding every story relies on.

- [X] T001 Add placeholder `docs/coverage.json` (empty JSON object) so coverage outputs have a tracked destination.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the CLI skeleton and configuration hooks the stories rely on.

- [X] T003 Create `cmd/docstringcov/main.go` with stdlib flag parsing and stubbed `run()` function.
- [X] T004 Add `internal/docstringcov/analyzer/analyzer.go` and `analyzer_test.go` with package docs plus TODO markers for symbol inspection logic.
- [X] T005 Update `internal/config/config.go` + tests with `DocstringCoverage` settings (target percentage) loaded from env/config file.

**Checkpoint**: CLI skeleton, analyzer package, and config plumbing exist; user stories can now build concrete behaviour.

---

## Phase 3: User Story 1 – Run Coverage Check (Priority: P1) 🎯 MVP

**Goal**: Provide a single command that reports current docstring coverage (overall, per-package, missing symbols).

**Independent Test**: Execute `make docstring-coverage` on the default branch; confirm it prints overall coverage, emits `docs/coverage.json` with per-package percentages + missing symbols, and exits successfully.

### Tasks

- [X] T006 [P] [US1] Implement exported-symbol parsing with `go/packages` in `internal/docstringcov/analyzer/analyzer.go` (counts, coverage math, missing list).
- [X] T007 [US1] Add unit tests covering edge cases (no exports, zero documented symbols, docstring format failures) in `internal/docstringcov/analyzer/analyzer_test.go`.
- [X] T008 [US1] Write JSON formatter that serializes overall stats, per-package entries, and missing symbols to `docs/coverage.json` in `internal/docstringcov/formatter/formatter.go`, including tests that ensure packages with zero exported symbols still appear in the output.
- [X] T009 [US1] Wire CLI command to call analyzer + formatter, print a human-friendly summary, and respect coverage thresholds in `cmd/docstringcov/main.go`.
- [X] T010 [US1] Add `make docstring-coverage` target plus `quickstart.md` snippet explaining how to run the tool locally and interpret results.
- [X] T011 [US1] Create a smoke test that runs `cmd/docstringcov`, validates the generated `docs/coverage.json`, and fails if the command exits non-zero.

**Checkpoint**: `make docstring-coverage` gives maintainers accurate, per-package coverage data in one run.

---

## Phase 4: User Story 2 – Raise Coverage (Priority: P2)

**Goal**: Bring the repository to ≥80% overall coverage by applying docstrings to the main exported packages using the new coverage tool.

**Independent Test**: Run the coverage command before and after docstring updates; the second run reports ≥80% coverage overall and shows the touched packages above 80%.

### Tasks

- [X] T012 [P] [US2] Add compliant docstrings for exported types/functions in `internal/challenge/service.go` and `internal/challenge/identity_mapper.go`, then rerun the coverage tool.
- [X] T013 [P] [US2] Document exported HTTP handlers in `internal/server/login_handler.go` and `internal/server/consent_handler.go`, verifying coverage increases after each change.
- [X] T014 [P] [US2] Update docstrings across client packages `internal/hydra/client.go` and `internal/kratos/client.go`, ensuring the coverage report shows those packages ≥80%.
- [X] T015 [US2] Re-run `make docstring-coverage`, confirm overall coverage ≥80%, and attach the final `docs/coverage.json` results to the PR description.

**Checkpoint**: Overall coverage meets or exceeds 80%; maintainers know how to rerun the tool to keep it there.

---

## Phase 5: Polish & Cross-Cutting

**Purpose**: Clean up docs and ensure the workflow is easy to follow.

- [X] T016 [P] Update `README.md` with a short "Docstring Coverage" section referencing `make docstring-coverage` and the JSON report in `docs/coverage.json`.
- [X] T017 Validate `quickstart.md` instructions by running them end-to-end on a clean checkout, adjusting any mismatched commands.

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 → User Stories → Polish.
- US1 is the MVP and must finish before US2 starts (US2 relies on the analyzer + CLI).
- US2 tasks T012–T014 can run in parallel because they touch different packages.

## Parallel Execution Examples

- During US1, run T006 (analyzer logic) and T008 (formatter) in parallel once the data structures are defined; developers can simultaneously add tests (T007) and CLI wiring (T009).
- During US2, assign separate engineers to challenge/server/client packages (T012–T014) while another contributor handles T015 coordination.

## Implementation Strategy

1. Deliver MVP by completing Phases 1–3 (coverage tool + documentation).
2. Use the tool to raise coverage (Phase 4) until the target is met.
3. Apply polish tasks for documentation consistency.
