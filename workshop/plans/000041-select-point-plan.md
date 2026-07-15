# metis#41 `select --point` Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote an operator-chosen config by ledger row id — `metis select <shape> --point
<point_addr> [--fingerprint F] [--promote]` — the v1 of the ledger-publish track.

**Architecture:** A row-resolve in front of the EXISTING reconstruct path (ARCH-DRY: no new
promote machinery — `promotedExperiment(sh, row.FreeParams)` + `runResolvedExperiment` as
`promoteSelected` uses; the cohort guard applies unchanged). Run id prefix `point-` (not `best-`)
so provenance shows operator-chosen. Prefix matching like git SHAs; ambiguity/absence/wrong-cohort
are loud errors. Without `--promote`, print the config's board line (pooled inner mean±SE + any
outer rows) — the single-config inspect.

**Tech Stack:** Go (`cmd/metis/select_cmd.go` + tests). Pure resolve logic, injected exec seam
for the promote test (existing pattern in `select_cmd_test.go`).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `resolvePointRows` | `cmd/metis/select_cmd.go` | new |

- **`resolvePointRows(led ledger.Ledger, prefix string) ([]ledger.Row, error)`** — rows whose
  `PointAddr` has the prefix, grouped to distinct configs (by FreeParams identity). 0 configs →
  "no row" error; >1 → ambiguous error listing candidate addrs+configs; 1 → its rows (all folds).
  Pure over the already-filtered (cohort-pinned) ledger — the guard stays where it is.
  - **DRY rationale:** the ONLY new logic; reconstruct/run/report all reuse #32's code.
  - **Future extensions:** `--where` predicates resolve to the same "rows → one config" contract.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `--point` flag + dispatch in `cmdSelect`/`runSelect` | `cmd/metis/select_cmd.go` | modified | CLI |

- Mutually exclusive with `--best`/`--best-per-model-class` (sharp error). With `--point`, skip
  the family/config selection entirely: resolve → report line → (if `--promote`)
  `promotedExperiment` + run as `point-{family}-{hash}`.

## Tasks (single pass, no Mx)

### Task 1: resolve + errors (TDD)

**Files:** `cmd/metis/select_cmd.go`, `cmd/metis/select_cmd_test.go`

- [ ] **Step 1: failing tests** (mirror the existing fixture-ledger tests in `select_cmd_test.go`):
  - `TestSelectPoint_ResolvesPrefixAndPrintsBoardLine` — fixture ledger, unique prefix, no
    `--promote`: output contains the config's free params + pooled inner mean; exit nil.
  - `TestSelectPoint_AmbiguousPrefixListsCandidates` — two configs sharing a prefix → error names
    both addrs.
  - `TestSelectPoint_NoMatchErrors` — unknown addr → error.
  - `TestSelectPoint_WrongCohortErrors` — addr exists only outside the pinned `--fingerprint` →
    error (the filtered ledger simply has no row; error text should hint the cohort pin).
  - `TestSelectPoint_ConflictsWithBest` — `--point` + `--best` → sharp usage error.
- [ ] **Step 2: run — all FAIL** (`go test ./cmd/metis -run TestSelectPoint`).
- [ ] **Step 3: implement** — flag, exclusivity check, `resolvePointRows`, report line (reuse the
  board-line rendering helpers used by the --best path where possible).
- [ ] **Step 4: run — PASS + full `go test ./...` green.**
- [ ] **Step 5: commit** `#41: select --point — resolve + inspect (TDD)`.

### Task 2: promote path (TDD)

- [ ] **Step 1: failing test** — `TestSelectPoint_PromoteReconstructsRowConfig`: injected exec
  seam (as existing promote tests); assert the promoted run id has the `point-` prefix and the
  reconstructed experiment's resolved `with` matches the fixture row's FreeParams exactly.
- [ ] **Step 2: run — FAIL.**
- [ ] **Step 3: implement** — `--promote` branch: `promotedExperiment(sh, row.FreeParams)` →
  `pointAddressOf` → runID `point-{familyTag}-{short(addr)}` → `runResolvedExperiment`.
- [ ] **Step 4: PASS + full suite + `go vet`.**
- [ ] **Step 5: commit** `#41: select --point --promote — ship an operator-chosen config`.

### Task 3: real-ledger verification + close

- [ ] **Step 1:** rebuild `bin/metis`; on the real kbench ledger run
  `metis select titanic-sweep.md --point <addr of rf md=8 n=200 all-6+tickets> --fingerprint
  b7aee3de… ` (inspect), then `--promote`. Operator runs `kaggle submit --run point-rf-…`.
- [ ] **Step 2:** atlas touch (select command surface — one line in `atlas/experiment.md`'s select
  paragraph), RUNBOOK §2 note (point-select exists).
- [ ] **Step 3:** `sdlc close --issue 41 --verified '<tests + real promote evidence>'` (actual
  measured), then `sdlc pr` + `sdlc merge --yes`.

## Notes

- Plan review: proportionality — this plan relies on `sdlc change-code`'s plan-quality judge as
  the fresh-eyes gate (a separate agent review pass would exceed the feature's size; the #35
  precedent used one because that plan spanned two repos).
- ARCH-PURE: `resolvePointRows` is pure over parsed ledger rows; IO stays in the existing shell.
- ARCH-PURPOSE: the purpose is "publish ANY ledger row" — v1 scope is row-id only by operator
  decision; `--where` is explicitly the same surface later, so the flag parsing/dispatch should
  not hard-code point-only assumptions (resolve returns the shared "one config's rows" contract).
