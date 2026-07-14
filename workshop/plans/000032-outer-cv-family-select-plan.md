# Outer-CV Family Selection Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `metis run` on a sweep *record* the full nested CV (inner + outer rows) to the ledger, and add `metis select` to read it — picking the model **family** by the honest outer estimate (lowest-SE-within-1-SE) and the **config** within it by inner CV, then `--promote` reconstructs-and-ships on all data — so the honest estimate *steers* selection instead of just reporting.

**Architecture:** Two review boundaries. **M1 (measure/record):** delete the `driver:` block, derive the run mode by config-count (`>1`→nested, `1`→flat single-level CV), add `--fast` (one outer fold), and extend the ledger to record inner + outer rows with a `Level`-keyed schema so they never collide. **M2 (choose/ship):** the new `metis select` command — a `FamilyOf`-keyed family reducer + the config-within-family pick, a dry report with the honesty caveat, and `--promote` reconstruct-and-ship via the existing `shapeConfigToExperiment` + `runResolvedExperiment`; retire `metis ledger select` + `metis promote` and migrate the surface.

**Tech Stack:** Go (`pkg/ledger`, `pkg/sampler`, `pkg/experiment`, `cmd/metis`), the metis#18 sampler fold algebra, CUE (`construct/vocabulary/experiment.cue`).

**Spec:** `workshop/issues/000032-outer-cv-family-select.md` `## Spec` (twice fresh-eyes reviewed). Read it first — this plan decomposes it.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ledger.Row` (+ `Level`, outer-fold coord) | `pkg/ledger/ledger.go` | modified |
| `ledger.AggregateView` (Level-aware key) | `pkg/ledger/ledger.go` | modified |
| `FamilyEstimate` reducer | `pkg/ledger/ledger.go` (or `cmd/metis/select_cmd.go`) | new |
| `runModeFor(shape)` (config-count dispatch) | `cmd/metis/sweep.go` | new |
| `familySelect` (lowest-SE-within-1-SE) | `pkg/sampler/select.go` | new |
| `experiment.Shape` (drop `Driver`) | `pkg/experiment/shape.go` | modified |

- **`ledger.Row` + `Level`/outer-fold** — a row gains a `Level` discriminant (`inner`|`outer`) and, for the nested run, an **outer-fold** coordinate (inner rows: `outer-fold × config × inner-fold`; outer rows: `outer-fold × family-winner-config`). Both `Level` and outer-fold **must enter `dedupKey` and the `AggregateView` group key** — a column alone still collides (`ledger.go:53,216`).
  - **DRY rationale:** one ledger, two levels, distinguished by the key — no second sidecar file.
  - **Future extensions:** a third level (e.g. an ensemble blend, metis#22) slots in as another `Level` value.
- **`FamilyEstimate` reducer** — groups the **outer** rows by `FamilyOf(free-params)` (NOT exact free-params — a family's winner differs across outer folds) and reduces the outer score over the outer folds → per-family `mean ± SE`. Distinct from `AggregateView` (which stays for the config/inner side).
  - **DRY rationale:** the family estimate is a genuinely different grouping; reusing `AggregateView` would silently blend/fail (spec C-1).
- **`runModeFor(shape)`** — pure decision: `len(shape.Expand()) > 1` → nested (outerK = `sweeper.resample.cv.k`, or 1 under `--fast`); `== 1` → flat single-level CV on all data; no sweeper → run steps as-is. Replaces the deleted `sh.Driver.CV != nil` branch (`sweep.go:143,175`).
- **`familySelect`** — pure: over per-family `(mean, SE)`, the families within 1 SE of the best mean → the lowest-SE one. Does **NOT** reuse `SweepResult.Ship` (cross-family inner-argmax = the overfitter #32 replaces). Degrades to argmax-mean under one outer fold (`--fast`, no SE).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `metis select` command | `cmd/metis/select_cmd.go`, `main.go` | new/modified | CLI + ledger read + ship |
| `--fast` / outer-fold knob | `cmd/metis/main.go`, `sweep.go` | new | run config |
| nested-run ledger writer | `cmd/metis/sweep.go` | modified | the ledger sidecar |
| `--promote` reconstruct-and-ship | `cmd/metis/select_cmd.go` | new | `runResolvedExperiment` |

- **`metis select`** — reads the shape + its ledger; reduces (family + config); dry-prints or (`--promote`) ships. Injected: the pure `FamilyEstimate`/`familySelect` do the deciding; the command is the thin IO seam. **Retires** `metis ledger select` + `metis promote` (removed from `main.go`).
- **`--promote` reconstruct-and-ship** — reuses the pure `shapeConfigToExperiment(shape, freeParams)` (`cmd/metis/ledger.go:76`, all-data fit = no `_fold`) → `runResolvedExperiment` (the shared engine). Run id `best-{family}-{hash}`. Errors on empty `ship:`.
  - **Test surface:** a fixture ledger (inner+outer rows) drives `select` with no subprocess; `--promote` uses the injected fake exec to assert an assembled all-data ship experiment (+ a real-step smoke where feasible).

---

## Chunk 1 — M1: the measure/record side

### Task 1.1: extend `ledger.Row` with a `Level` + outer-fold coordinate in the key

**Files:** Modify `pkg/ledger/ledger.go`; Test `pkg/ledger/ledger_test.go`.

- [ ] **Step 1: failing collision test** — an inner row and an outer row for the same config+fold must NOT merge in `AggregateView`.

```go
func TestAggregateView_LevelKeyedNoCollision(t *testing.T) {
	// same free-params + same fold index, different Level: inner fold_score 0.80,
	// outer held-out 0.72. AggregateView must keep them in SEPARATE groups (not avg to 0.76).
}
```

- [ ] **Step 2: run → FAIL** (`go test ./pkg/ledger/ -run TestAggregateView_LevelKeyedNoCollision`) — today they merge.
- [ ] **Step 3: implement** — add `Level string` (+ `OuterFold *int`) to `Row`; fold both into `dedupKey` (`ledger.go:53`) and the `AggregateView` group key (`ledger.go:216`); extend `Encode`/`Decode` header + codec (a new column, ragged so a v1 ledger stays byte-identical when the column is absent). Level empty ⇒ legacy/flat (back-compat).
- [ ] **Step 4: run → PASS** + the existing ledger suite green (back-compat).
- [ ] **Step 5: commit** — `#32 M1: ledger Row gains Level + outer-fold in the group/dedup key`.

### Task 1.2: the `FamilyEstimate` reducer (group by `FamilyOf`, reduce over outer folds)

**Files:** Modify `pkg/ledger/ledger.go` (or a new `cmd/metis/family.go`); Test alongside.

- [ ] **Step 1: failing test** — outer rows for `rf`(md4, fold0),`rf`(md8, fold1),`gbm`(…,fold0/1) reduce to per-family `(mean, SE)` grouped by `FamilyOf`, NOT by exact config.
- [ ] **Step 2: run → FAIL.**
- [ ] **Step 3: implement** `FamilyEstimate(rows, familyOf) map[string]MeanSE` — filter `Level==outer`, group by `familyOf(FreeParams)`, `Aggregate` the outer score. (`FamilyOf` lives in `pkg/sampler`; inject it to keep `pkg/ledger` from depending on `sampler`, or site the reducer in `cmd/metis`.)
- [ ] **Step 4: run → PASS.**
- [ ] **Step 5: commit** — `#32 M1: FamilyEstimate reducer (family-keyed, distinct from AggregateView)`.

### Task 1.3: delete `driver:`, derive the run mode by config-count, add `--fast`

**Files:** Modify `pkg/experiment/shape.go` (drop `Driver` struct + validator), `construct/vocabulary/experiment.cue` (drop `driver` field), `cmd/metis/sweep.go` (`runShapeSweep` dispatch), `cmd/metis/main.go` (`--fast`), the two shape fixtures (`testdata/experiment/titanic-baseline-shape.md`, note `kbench …/titanic-sweep.md` is a peer edit — do in M2 migration or coordinate). Test `cmd/metis/*_test.go`, `pkg/experiment/shape_test.go`.

- [ ] **Step 1: failing test** — `runModeFor`: `>1` config → nested (outerK = cv.k); `==1` → flat single-level CV; `--fast` → outerK=1.
- [ ] **Step 2: run → FAIL.**
- [ ] **Step 3: implement** — remove `Driver` from `shape.go` (+ its `ValidateShape` "exactly one" clause) and the CUE `driver` field; replace `sweep.go`'s `sh.Driver.CV != nil` branch (`:143,175,217`) with `runModeFor`; outerK from `sweeper.resample.cv.k` (or 1 under `--fast`); add the `--fast` flag → `runOpts`. Migrate `titanic-baseline-shape.md` (drop `driver:`). Reword any "driver" references.
- [ ] **Step 4: run → PASS**; `pkg/experiment` + `cmd/metis` suites green (fix fixtures that carried `driver:` — they now `ParseShape`-fail under `KnownFields(true)`).
- [ ] **Step 5: commit** — `#32 M1: drop driver:; derive run mode by config-count; --fast (one outer fold)`.

### Task 1.4: the nested run records inner + outer rows (each family's inner-winner)

**Files:** Modify `cmd/metis/sweep.go` (`runNestedCV`/`runOuterFold`/`runPipelineFold` + the ledger write). Test `cmd/metis/nestedcv_e2e_test.go`.

- [ ] **Step 1: failing test** — a fake-exec nested run over 2 families × N configs writes, to the ledger: inner rows per `(outer-fold, config, inner-fold)` AND one outer row per `(outer-fold, family)` (each family's inner-winner, not just the single overall `sres.Ship`). *(This inverts the current `TestNestedCV_ProducesHonestEstimateNoShip`, which asserts the nested path records nothing — update it.)*
- [ ] **Step 2: run → FAIL** (today `runOuterFold` outer-scores only `sres.Ship`, records nothing).
- [ ] **Step 3: implement** — in `runOuterFold`, score **each family's inner-winner** (`sres.PerFamily`, not just `sres.Ship`) on the held outer-assessment; harvest inner rows (from `pass.points`) + the per-family outer rows into `ledger.Row`s with `Level`/outer-fold set; write via the ledger sidecar (extend `rowsFromManifest`/`writeSweepLedger`). Keep the GuardComplexity per fold.
- [ ] **Step 4: run → PASS**; the honest-estimate report still prints (now alongside the recorded rows).
- [ ] **Step 5: commit** — `#32 M1: nested run records inner + per-family outer rows to the ledger`.

### Task 1.5: M1 close boundary

- [ ] `sdlc milestone-close --issue 32 --milestone M1` (the mandatory fresh-context boundary review). Fix Critical/Important before crossing. Verified = the collision test + family-reducer test + dispatch/degenerate test + the nested-recording test, all green under `-race`; a `metis run --dry-run` + a small fake-exec nested run showing the ledger rows.

---

## Chunk 2 — M2: the choose/ship side

### Task 2.1: `familySelect` (lowest-SE-within-1-SE) in the sampler

**Files:** Modify `pkg/sampler/select.go`; Test `pkg/sampler/select_test.go`.

- [ ] **Step 1: failing test** — over per-family `(mean,SE)` where `rf`=(0.80,0.01) and `gbm`=(0.805,0.03): within 1 SE of the best (gbm 0.805 − 0.03 = 0.775) both qualify → pick **lowest-SE** = `rf`. And a one-fold (SE=0) case → argmax-mean.
- [ ] **Step 2: run → FAIL.**
- [ ] **Step 3: implement** `familySelect(map[string]MeanSE) (family string, caveat string)` — 1-SE band on the best mean, lowest-SE among them; degrade to argmax-mean when SE unavailable. Reuse the `withinBand` idiom (`select.go:156`) but over families. Do NOT touch `SweepResult.Ship`.
- [ ] **Step 4: run → PASS.**
- [ ] **Step 5: commit** — `#32 M2: familySelect — lowest-SE-within-1-SE over per-family honest estimates`.

### Task 2.2: `metis select` (dry) — read, reduce, report

**Files:** New `cmd/metis/select_cmd.go`; Modify `cmd/metis/main.go` (register `select`; remove `ledger select`). Test `cmd/metis/select_cmd_test.go`.

- [ ] **Step 1: failing test (the acceptance gate)** — on a **fixture ledger** (inner rows favoring gbm on inner CV; outer rows where rf's honest estimate ≥ gbm's), `metis select <shape> --best` prints the **rf** family (not gbm) + per-family `mean±SE` + the honesty caveat. `--best-per-model-class` prints each family + its config.
- [ ] **Step 2: run → FAIL** (command doesn't exist).
- [ ] **Step 3: implement** `cmdSelect` — load shape + ledger (fingerprint-scoped: pin one `code_fingerprint`, error sharply on a mixed/absent set); `FamilyEstimate` → `familySelect` → family; `SelectConfigs.PerFamily` + `sweeper.objective.select` → config within family; print. Register in `main.go`, remove `ledger select`.
- [ ] **Step 4: run → PASS.**
- [ ] **Step 5: commit** — `#32 M2: metis select (dry) — family by honest estimate, config by inner CV`.

### Task 2.3: `metis select --promote` — reconstruct + ship on all data

**Files:** Modify `cmd/metis/select_cmd.go`; Test `cmd/metis/select_cmd_test.go`.

- [ ] **Step 1: failing test** — `select <shape> --best --promote` (fake exec) assembles an **all-data** ship experiment (data+pipeline+ship, no `_fold`) for the selected config, runs it into `runs/best-{family}-{hash}/`, and prints the run id. A shape with empty `ship:` → a clear error.
- [ ] **Step 2: run → FAIL.**
- [ ] **Step 3: implement** — for each selected config: `shapeConfigToExperiment(shape, freeParams)` → build `runOpts` (`runID = best-{family}-{shorthash}`) → `runResolvedExperiment`; guard empty `ship`. `--best-per-model-class --promote` → one run per family. Print the ids.
- [ ] **Step 4: run → PASS**; retire `metis promote` from `main.go` (+ its `ledger_cmd.go` path if now dead).
- [ ] **Step 5: commit** — `#32 M2: select --promote reconstruct+ship on all data (best-{family}-{hash}); retire promote`.

### Task 2.4: `metis run` stops auto-shipping + migration surface

**Files:** Modify `cmd/metis/sweep.go` (flat path no longer calls `shipWinner`); migrate `kbench …/titanic-sweep.md` (peer repo — read its `AGENTS.local.md`/`MEMORY.md` first) + `RUNBOOK-sweep.md`; update tests asserting the retired behavior. Test: the flat-path test.

- [ ] **Step 1:** update/replace `TestShapeSweep_ShipsWinner` → assert `metis run` measures + records, does NOT ship; add the note that shipping is `select --promote`.
- [ ] **Step 2: implement** — remove the `shipWinner` call from `runShapeSweep`'s flat path (`sweep.go:262-266`); `metis run` now measures only.
- [ ] **Step 3: migrate docs (peer + in-repo)** — `RUNBOOK-sweep.md` §§1–4 rewritten to `run`/`--fast`/`select --promote`; the metis atlas (`atlas/experiment.md` + index) gains the `run`/`select` command model + the ledger `Level` schema.
- [ ] **Step 4: run → whole suite green under `-race`.**
- [ ] **Step 5: commit** — `#32 M2: metis run no longer auto-ships; migrate RUNBOOK/atlas/shapes/tests`.

### Task 2.5: M2 close (issue close)

- [ ] `sdlc close --issue 32 --milestone M2` (the mandatory boundary review). Verified = the `select --best`-picks-rf-over-gbm fixture gate + the `--promote` materialize test + the run-no-ship test, all green under `-race`; a real fixture end-to-end (`metis run` → `metis select --best --promote` → a submission.csv) if hermetically feasible, else the fake-exec chain + a note. `--actual` measured.

---

## change-code plan-quality refinements (fold into implementation)

1. **`--promote` reuses `promotedExperiment` (`ledger.go:59`), not `shapeConfigToExperiment` directly** — `promotedExperiment` is the free-params→experiment inverter `cmdPromote` uses (it `shape.Expand`s + matches the tuple, then calls `shapeConfigToExperiment`). Retiring `promote` must keep its inverter. (Task 2.3.)
2. **`cmd/metis/select_cmd.go` already exists** (holds `cmdLedgerSelect`) — Task 2.2 is a **repurpose/modify in place**, not a new file.
3. **Add an explicit failing-test step for the 1-config degenerate path** (Task 1.3/1.4) — pin its two new behaviors: records inner rows AND does NOT ship. Done-when commits to this hermetic test.
4. **Pin the `(mean, SE)` carrier type** between `FamilyEstimate` and `familySelect` — a plain shared `{mean, se float64}` (not `sampler.MeanSE`'s struct with `ToldSet`), keeping `pkg/ledger` free of a `sampler` import.
5. **`OuterFold` likely only needs to be a recorded COLUMN, not in `dedupKey`** — inner/outer rows already differ by PointAddr (different `baseRef`/`splitK`/`partRef`); the load-bearing fix is **`Level` in the `AggregateView` group key** (`ledger.go:216`). Don't over-constrain the key. Also state explicitly: the config-within-family inner reduction **pools inner folds across all outer folds** into one per-config `(mean, SE)` — that pooled estimate (not a flat all-data inner CV) is the intended config-selection signal.
6. **Migration watch:** under the derived mode, every multi-config shape now defaults to the ~`outerK`× nested path and no longer auto-ships — when migrating fixtures/tests, confirm none silently become 5×+ costlier or assert a now-absent ship line (`--fast` is the escape hatch).

## Open note for the operator / executor

- **The real acceptance** (`select --best` ships rf over the GBM overfitter, and the shipped config's outer estimate tracks public) needs the operator-gated real-Kaggle run (GBM 0.749 vs rf 0.79186 are recorded evidence). The hermetic gate is the fixture-ledger `select` test (Task 2.2 Step 1); the Kaggle confirmation is a separate operator run, not a blocking test.
- **`FamilyEstimate` siting:** if injecting `FamilyOf` into `pkg/ledger` feels wrong (a `ledger`→`sampler` dependency), site the reducer in `cmd/metis` — the plan tolerates either; pick the one that keeps `pkg/ledger` free of `sampler`.
