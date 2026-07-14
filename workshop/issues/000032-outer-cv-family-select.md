---
id: 000032
status: working
deps: [metis#23]
github_issue:
created: 2026-07-13
updated: 2026-07-13
estimate_hours: 4.5
started: 2026-07-13T15:52:26-07:00
---

# outer-CV model-family selection — close the loop (nested-CV selects, not just reports)

## Problem

**metis#23's nested CV (`driver: cv`) is a passive reporter — it produces an honest estimate but ships
NO winner, so the outer CV isn't used in any automated fashion.** That wastes exactly the signal we
need. Demonstrated empirically on the real Titanic honest-beat run (kbench#8):

- The sweeper's cross-family pick is inner-CV **argmax-mean** (metis#19's parsimony is intra-family only).
- Inner CV shipped `hist_gbm [title,family]` (mean **0.846**, cx 1500) → **public 0.749** — a ~0.10 overfit
  gap. The rf robust winner `[title…embarked, ticket_survival]` md=4 (mean 0.839, cx 14.3) would generalize
  far better (that family ~0.78 public historically).
- The outer CV *holds the signal that GBM overfits* (its honest estimate drops toward ~0.75 while rf's
  holds) — and we throw it away. The cross-family choice is made on the optimistic inner CV instead.

## Spec

*(Design converged via brainstorm 2026-07-13. The **principle** below survives from the original capture;
the **mechanism** evolved twice — from "a driver mode that selects+ships inline" → run/select separation →
the final **derived-run-mode** model here. See the [[## Log]] deltas.)*

**The principle (two-level selection, each signal used where it's valid):**
- **Within a family** → inner CV + parsimony (metis#19). Fine choice (dozens of hyperparams); complexity
  discriminates here, where its units are comparable.
- **Across families** → the **outer/nested honest estimate** (metis#23). Coarse choice (few classes).
  **Outer-CV accuracy is cross-family COMPARABLE** (held-out accuracy, same units) — which complexity is
  NOT (14 leaves vs 1500 leaves vs 5 coefs). So the outer CV is the correct cross-family signal.
  **Subsumes** the cross-family-argmax problem AND the (unsound) cross-family-complexity idea.

**Statistical spine — granularity.** Inner-CV selection overfits because it's a FINE choice (best of ~99)
on an OPTIMISTIC estimate; family selection is a COARSE choice (best of ~3) on an HONEST estimate → small
selection bias on a low-dim choice. Fine choice on the cheap estimate; coarse choice on the honest one.

### Mechanism — three commands, a derived run mode, one shared run engine

`run` **measures** and populates a ledger; `select` **chooses** (and, with `--promote`, ships); `kaggle
submit` **uploads**. The outer CV is a pure *measure* of ~3 per-family procedures, **never a config sweep**
(outer-scoring all configs + selecting on them would be "yet another sweep" on the held-out fold →
dishonest; it reintroduces the optimism #23 removed).

**1. `metis run <shape>` — MEASURE (never ships). The mode is DERIVED from the shape (a clean branch on
`len(configs)`, replacing the deleted `driver:` dispatch — NOT "no special-casing"; the paths genuinely differ):**

| Shape (`shape.Expand`) | `metis run` does | records |
|---|---|---|
| **sweep** (`>1` config) | **nested CV** (outer × inner; outer folds reuse `sweeper.resample.cv.k`) | inner + outer rows |
| **1 config** (no free vars, has resample) | **flat single-level CV on all data** — a different, cheaper path (a plain k-fold of the one config on the FULL dataset, not the nested subset-sweep). NB: this inner-CV is on all data, whereas a config inside a multi-config nested sweep is measured on the `analysis_i` subset — different data, so don't claim "identical signal" | inner rows |
| **bare experiment** (no sweeper) | run the steps as-is | — |

- **`metis run --fast`** = one outer fold (outerK=1) instead of `sweeper.resample.cv.k`: one sealed inner
  sweep on `analysis_0` + one holdout scoring ≈ **~1/5 the cost**, and gives an **honest single-point**
  per-family holdout number (no SE — one fold). Replaces a naive `--inner-only`: same cost, but honest.
  For iteration; the full nested pass is what you run before selecting/shipping (where the SE-based family
  rule applies). Mechanically the outer-fold count is a run knob (default = `sweeper.resample.cv.k`,
  `--fast` = 1; room for a general `--outer-folds N` later).

**Ledger recording (C2 — the one substantial new backend piece; today the nested path records NOTHING).**
The `ledger.Row` gains a **`Level` discriminant** (`inner` | `outer`) **and an outer-fold coordinate**, and
BOTH must enter the dedup/group **key** (a column alone does not prevent collision):
- **inner rows**: per `(outer-fold, config, inner-fold)` — the sealed inner-CV fold scores (keyed by
  `config × outer-fold × inner-fold × Level=inner`).
- **outer rows**: per `(outer-fold, family-winner-config)` — that fold's family inner-winner + its
  outer-held-out score (keyed by `config × outer-fold × Level=outer`).

Without the `Level` in the key, an inner row and the outer row for the same winning-config+fold share
`(fingerprint, free-params, fold)` and `AggregateView` would **average the inner subset-score with the
outer held-out score** — the exact collision C2 exists to prevent.

**Two reductions, not one.** `AggregateView` (groups by exact free-params) stays for the **config/inner**
side. The **per-family honest estimate needs a NEW reducer** that groups the *outer* rows by
`FamilyOf(free-params)` (a family's winner *differs* across outer folds → different free-params → they'd
never group under `AggregateView`) and reduces the outer score over the outer folds → per-family
`mean ± SE`. Family is derived (`FamilyOf`), but the *grouping key* is the family, not the raw config.

**2. `metis select <shape> [--best | --best-per-model-class] [--promote]`** — new top-level command;
**retires `metis ledger select` + `metis promote`**.
- Reduces the ledger: **family** from the outer rows, **config-within-family** from the inner rows.
- **Family rule (NEW code over the outer rows — do NOT reuse `SweepResult.Ship`, which is the cross-family
  inner-argmax that overfits, the exact behavior #32 replaces; only `SelectConfigs.PerFamily` is reused,
  for the config-within-family pick):** among families whose honest mean is within **1 SE of the best
  family's mean**, pick the **lowest-SE** one (documented tiebreak; not over-designed — revisit only if real
  runs show families tying). NOT a static family-complexity order — class complexity isn't statically
  orderable (a regularized gbm can use fewer effective params than a deep rf), so a static order just
  smuggles cross-family complexity comparison back in. (Under `--fast`: one fold → no SE → the rule degrades
  to argmax-mean over the single honest holdout; `select` says so.)
- **Config rule (within family):** inner CV + `sweeper.objective.select` (the metis#19 rule, e.g. pct-loss).
- **`--best`** → single ship rec. **`--best-per-model-class`** → one pick per family (the metis#22 seam).
- **Dry (no `--promote`):** print the selected config(s) + per-family honest `mean±SE` + the honesty caveat.
- **`--promote`:** for each selected config, **reconstruct it from the ledger** (config free-params from the
  ledger row; the `data`/`pipeline`/`ship` steps from the shape `.md` `select` already has) via the existing
  pure `shapeConfigToExperiment` (all-data fit = no `_fold` injected) → run it in **ship mode — `data +
  pipeline + ship`, fit on ALL data, no CV → `submission.csv`**, into `runs/best-{family}-{hash}/`. **Prints
  the run ids** (the handles for `kaggle submit`). Config always reconstructed from the ledger → no committed
  winner `.md`, no `--point` selector (retires that friction). **Error** if the shape's `ship:` is empty
  (today `shipWinner` silently no-ops — but `--promote` promises a `submission.csv`). NB: `best-{family}-{hash}`
  makes the run **dir name ≠ point-address** (reversing metis#27's "dir name IS the identity" for auto-named
  single runs) — fine as a human handle; `record.json` still carries the true point_address.

**3. `kaggle submit --run best-{family}-{hash}`** → uploads that run's `submission.csv` (+ slug from
`record.json`). Never runs the pipeline.

**One shared run engine, two assemblies (ARCH-DRY).** Both `metis run`'s measure and `--promote`'s ship
funnel through the SAME per-experiment runner (`runResolvedExperiment` — cache, step executor,
record/run.json), differing only in how the fixed-config experiment is *assembled*: measure = `data +
pipeline + cv-split` (fold-scored); ship = `data + pipeline + ship`, all-data fit. No second engine → the
cache/provenance/determinism guarantees are identical on both paths.

**Shape simplification.** The `driver:` block is **deleted** (the run mode is derived; the outer loop
reuses `sweeper.resample.cv`). `data` / `pipeline` / `ship` / `sweeper` all stay. `ship` is **latent
during measurement** and activated only by `select --promote` (which reconstructs the winner *from this
shape* and needs its `ship` steps) — so it lives in the sweep shape but `metis run` never runs it.

**Workflow:**
```
metis run titanic-sweep.md --fast                       # iterate: one outer fold, ~1/5 cost, honest single-point
metis run titanic-sweep.md                              # decide: full nested CV → ledger (inner + outer rows)
metis select titanic-sweep.md --best                    # dry: print the pick + honest mean±SE + caveat
metis select titanic-sweep.md --best-per-model-class --promote   # reconstruct + ship each on ALL data →
                                                        #   runs/best-{family}-{hash}/submission.csv (prints ids)
kaggle submit --run best-rf-{hash}
```
metis#31's parallelism makes the full nested `run` affordable — the reason it was sequenced first.

**Honesty caveat (reported, not hidden).** The *selected family's* honest estimate is mildly optimistic — a
1-SE pick over ~3 families, ~0.01 upward bias (NOT a per-config claim — the shipped config has no outer
estimate of its own; only the family/procedure does). A large honesty gain over the inner-CV path's ~0.10
gap, not a perfectly-unbiased number (that needs another outer loop; not worth it at ~3 families). `select`
surfaces the caveat alongside the number.

**Join soundness.** The family (outer rows) ↔ config (inner rows) reduction MUST be pinned to one
`code_fingerprint` — an unscoped reduce over a mixed-code ledger silently blends versions (the
`workshop/lessons.md` footgun). `select` errors sharply if the ledger lacks the rows it needs (e.g. a
non-sweep ledger with no outer rows) rather than nil-deref.

**Breaking change (state it, don't hide it).** `metis run` **no longer auto-ships** — today the flat
`driver: single` path selects AND ships the winner (`sweep.go` `shipWinner`); under #32 `run` only measures,
and ALL shipping moves to `select --promote`. Justified: the auto-shipped argmax winner IS the overfitter
#32 exists to stop shipping; `--fast` preserves cheap iteration. **Migration surface** (delete `driver:` +
rewire): the CUE schema `construct/vocabulary/experiment.cue` (the `driver` field); two shape files
(`testdata/experiment/titanic-baseline-shape.md`, `kbench …/titanic-sweep.md`, both `driver: single`); the
four `sh.Driver.CV`/`Driver.Single` reads in `cmd/metis/sweep.go` → `sweeper.resample.cv`; `kbench …
/RUNBOOK-sweep.md` (§§1–4 + the "edit `driver:` to `cv`" honest-estimate flow — all rewritten to
`run`/`select --promote`/`--fast`); and the tests asserting the retired behavior (`TestShapeSweep_ShipsWinner`,
`TestNestedCV_ProducesHonestEstimateNoShip` — which asserts the nested path records *nothing*, the exact
inverse of the new contract — + `ledger_cmd_test.go`/`select_cmd_test.go` for the retired `ledger select`+`promote`).
One-time: deleting `driver:` changes every shape's blob-hash once (invalidates cached run dirs — regenerable).

**Scope.** IN #32: the nested-run ledger recording (inner+outer, `Level`-keyed); the derived run mode +
`--fast` + `driver:`-drop + migration; `metis select` (the select+promote merge, reconstruct-and-ship, the
family reducer); the family rule; the caveat. OUT (separate issues): the **repo-root-relative path key**
(metis#34). COMPOSES with: metis#19 (intra-family, unchanged), metis#23 (the estimator this extends +
records), metis#22 (consumes `--best-per-model-class`), metis#33 (GBM's own overfit — orthogonal).

## Done when

- **`metis run` on a sweep records the full nested CV to the ledger** — per-`(outer-fold, config, inner-fold)`
  inner rows + per-`(outer-fold, family-winner)` outer rows, with a **`Level` discriminant + outer-fold coord
  in the group/dedup key** so an inner row and the outer row for the same config+fold do NOT merge in
  `AggregateView` (a hermetic collision test proves it). Today the nested path records nothing.
- **The run mode is derived by config-count** (`>1` → nested; `==1` → flat single-level CV on all data;
  `--fast` → outerK=1), and a hermetic test covers the 1-config degenerate path. **The `driver:` block is
  removed** from the CUE schema + shapes; migration surface (RUNBOOK, 2 shapes, ~4 tests) updated; `ship` is
  retained and activated by `--promote`.
- **`metis select <shape>`** reduces the outer rows via a **`FamilyOf`-keyed reducer** (distinct from
  `AggregateView`) → per-family honest `mean±SE`; **`--best`** applies the lowest-SE-within-1-SE family rule
  (does NOT reuse `SweepResult.Ship`) + inner-CV config choice; **`--best-per-model-class`** reports per family;
  **`--promote`** reconstructs each config from the ledger and ships it on ALL data (`runs/best-{family}-{hash}/`),
  printing the run ids (errors on empty `ship:`); **retires `metis ledger select` + `metis promote`**; the
  family↔config join is fingerprint-scoped and errors sharply when required rows are absent.
- On a fixture cv+inner ledger, `select --best` picks the **rf** family over the **gbm** overfitter
  (hermetic gate). *(The real-Kaggle numbers — GBM 0.749 vs rf 0.79186 — are recorded evidence, not a
  runnable test.)*
- The max-over-families honesty caveat is surfaced in the `select` report (as the *family's* estimate).

## Estimate

Derived against estimate-logic-v3.1 (impl = ship-wall-clock, AI-paired). The design is resolved (spec
twice fresh-eyes-reviewed + durable plan), but that design was done IN this issue's window (a long
post-claim brainstorm + 2 reviews), so its design hours are counted (part of what `sdlc actual` measures),
not free. Two milestones: M1 (ledger schema + nested recording + driver-drop/dispatch/--fast), M2 (the
`metis select` command + `--promote` + run-no-ship + migration). The migration surface (CUE + 2 shapes +
RUNBOOK + ~4 tests + retiring 2 commands) is real weight.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.3 impl=0.4
item: smaller-go-module      design=0.3 impl=0.4
item: cross-cutting-refactor design=0.2 impl=0.3
item: smaller-go-module      design=0.3 impl=0.4
item: smaller-go-module      design=0.2 impl=0.3
item: cross-cutting-refactor design=0.2 impl=0.3
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.1 impl=0.15
design-buffer: 0.15
total: 4.49
```

- `smaller-go-module` (M1 ledger) — `ledger.Row` `Level`+outer-fold in the key, `AggregateView` group-key + codec, `FamilyEstimate` reducer.
- `smaller-go-module` (M1 run) — nested run records inner + per-family outer rows (`runOuterFold` scores each family's inner-winner; ledger write).
- `cross-cutting-refactor` (M1) — drop `driver:` (shape.go + CUE + fixtures), config-count dispatch, `--fast` knob.
- `smaller-go-module` (M2 select) — `familySelect` + `metis select` (dry): reduce, family+config, report, fingerprint-scoped.
- `smaller-go-module` (M2 promote) — `--promote` reconstruct+ship on all data, `best-{family}-{hash}`, empty-ship guard, retire `promote`.
- `cross-cutting-refactor` (M2) — `metis run` no-auto-ship + migration (RUNBOOK/shapes/tests, retire `ledger select`).
- `milestone-review` ×2 — the M1 boundary + M2 close reviews.
- `atlas-docs` — atlas run/select command model + ledger `Level` schema.

## Plan

- [x] **Brainstorm** (2026-07-13, operator) — converged the mechanism (see Log): derived run mode (nested
  for sweeps, single-level CV degenerate), one nested run recording inner+outer, `metis select`
  reconstruct-and-ship on all data, `driver:` dropped, one shared run engine. Spec above.

Durable plan: `workshop/plans/000032-outer-cv-family-select-plan.md` (twice-reviewed spec decomposed;
core-concepts tables + TDD tasks). **Two review-boundary milestones:**

- [ ] **M1 — measure/record side:** ledger `Row` `Level`+outer-fold in the key + `FamilyEstimate` reducer;
  drop `driver:` + config-count dispatch + `--fast`; nested run records inner + per-family outer rows.
  (closes via `sdlc milestone-close --milestone M1`.)
- [ ] **M2 — choose/ship side:** `familySelect` (lowest-SE-within-1-SE); `metis select` (dry: family+config
  report) + `--promote` (reconstruct+ship all-data, `best-{family}-{hash}`); `metis run` no-auto-ship; retire
  `ledger select`+`promote`; migrate RUNBOOK/atlas/shapes/tests. (closes via `sdlc close --milestone M2`.)

## Log

### 2026-07-13
- 2026-07-13: closed M1 — M1 (measure/record side) complete + all 9 packages GREEN under go test ./... -race (independently re-run in main). Delivered: (1.1) ledger Row Level discriminant (+ outer_fold col) in the AggregateView group key — collision test RED-first; (1.2) FamilyEstimate reducer groups outer rows by FamilyOf (distinct from AggregateView); (1.3) shape driver: field deleted (shape.go + CUE), run mode DERIVED by config-count (>1 nested, ==1 flat single-level CV), metis run --fast (one outer fold), metis run no longer auto-ships; (1.4) nested run records inner rows (Level=inner, outer_fold) + per-family outer rows (each PerFamily winner scored on the held outer-assessment), mutex-guarded under metis#31 ParExec. metis#23 leakage sealing PRESERVED. 10 cmd/metis assertions aligned (Category-B kept multi-config->nested for coverage; no source weakened). Coverage moved: ShipRunIsCodeCaptured -> M2 (ship moved to select --promote). ACTUAL = N/A (--no-actual): sdlc actual attributes 4.06h across #20/#31/#32/#33/#34 by mention-fallback (interleaved-session, no clean per-issue boundary) AND the impl ran in forks the commit-based measure captures imperfectly AND this is a mid-issue partial (M2 remains) — recording N/A rather than polluting calibration; the final #32 actual is reckoned at M2 close.; review verdict: FIX-THEN-SHIP
- Filed from the kbench#8 honest-beat run (operator). THE headline metis-v2 capability — the honest
  estimate STEERING, not just reporting (closes the loop metis#23 left open; deps it). Empirically:
  inner-CV cross-family argmax shipped a GBM overfitter (0.846 inner → 0.749 public) over the rf-ticket
  generalizer. **Subsumes** the earlier "cross-family selection robustness" + "GBM cross-family complexity"
  ideas — the outer CV is the comparable cross-family signal, so no complexity hack is needed there.
  Sibling: metis#33 (GBM's own overfit + intra-family complexity), metis#31 (parallelism to afford the
  nested run), metis#30 (progress). Brainstorm-first — it extends the outer-node contract.
- **Brainstorm 2026-07-13 (operator) — mechanism refined; principle unchanged.** The original capture put
  selection INSIDE a driver mode (the outer node's `Tell/Done` selects+ships inline). The operator reframed
  to **run/select separation**: `run` populates a ledger, choosing is a read-time op over it. Deltas from
  the original spec:
  - **`driver: cv` records** per-`(outer-fold, family)` honest rows (it recorded nothing before); the outer
    node does NOT select — it stays a pure honest *measure* (operator: "the outer is only meant to provide
    honest measure, not yet another sweep"). Outer-scoring all configs + selecting on them = dishonest.
  - **`metis select` merges select+promote** (retires `ledger select` + `promote`); `--promote` materializes
    all-data runs named `best-{family}-{hash}` and prints the ids for `kaggle submit --run`.
  - **Family tie-break = lowest-SE-within-1-SE**, NOT "simplest class" — the operator killed a static
    family-complexity order as unsound (effective params vary; it re-smuggles cross-family complexity). SE
    also penalizes inner-selection instability for free.
  - **Two-run workflow** (single → config ledger + ship; cv → family estimates), ship on all data.
  - **Split out:** the repo-root-relative path key → its own issue (cwd-ergonomics, orthogonal). *(Filed as metis#34.)*
- **Brainstorm 2026-07-13 (operator) — second evolution: DERIVED run mode; supersedes the two-run workflow.**
  The spec review (fresh-eyes) surfaced C1: `driver:` is a shape field with an "exactly one of single|cv"
  validator + no `--driver` flag, so "run under two drivers" means editing the .md → changes its blob-hash →
  shifts all point-addresses + breaks `best-{hash}` naming. Resolving it collapsed the whole model:
  - **ONE `metis run` does the nested CV** and records BOTH inner (per config) + outer (per family) rows —
    no separate `driver: single` run. The two-run workflow is retired.
  - **Run mode is DERIVED, not declared:** sweep (free vars) → nested CV; no free vars → single-level CV (the
    outer selection loop has one candidate, so nested-CV mathematically *degenerates* — no special case);
    bare experiment → run as-is. **The `driver:` block is deleted**; the outer loop reuses `sweeper.resample.cv`.
    This dissolves C1 (no driver toggle → blob constant by construction) — the named-drivers machinery was
    solving a problem the derived mode designs away.
  - **`select --promote` reconstructs the config from the ledger** and runs it in ship mode (all data, no CV,
    → submission) — the winner is NEVER pre-materialized. `metis run` measures + `select --promote` ships
    funnel through the SAME run engine (`runResolvedExperiment`), two experiment assemblies (ARCH-DRY).
  - **`ship:` stays** (latent during measure; activated only by `--promote`, which reconstructs the winner
    from the shape and needs its predict/submission steps). Only `driver:` is obsolete.
  - **Verbs:** `metis run` (kept, not `metis sweep`) / `metis select` / `kaggle submit` — three commands.
  - Spec `## Spec` rewritten to this model; the C2 ledger extension (inner+outer rows, level-marked) is the
    one substantial new backend piece — everything else reassembles existing parts.
- **Spec review round 2 (fresh-eyes) + fixes folded — spec now plannable.** C1 confirmed dissolved. Fixes:
  - **C2 made concrete:** a `Level` (+ outer-fold coord) discriminant enters the ledger group/dedup KEY
    (a column alone still collides); the per-family estimate uses a NEW `FamilyOf`-keyed reducer (each outer
    fold's family-winner is a different config → `AggregateView`'s free-param grouping would never collect them).
  - **Dispatch (I-1):** branch on `len(configs)` (1 → flat all-data single-level CV, a cheaper distinct path;
    >1 → nested). Dropped the false "no special-casing"/"identical inner signal" framing (1-config inner CV
    is on all data; a nested config's is on the `analysis_i` subset).
  - **`--fast` (operator, I-2):** one outer fold — replaces a naive `--inner-only` (same ~1/5 cost, but an
    honest single-point holdout beats the optimistic inner-CV leaderboard for the family read). Full nested
    reserved for select/ship. Outer-fold count = a run knob (default `sweeper.resample.cv.k`, `--fast`=1).
    Cost clarified: the 5× isn't the cheap holdout scoring — it's the outer loop re-running the sealed inner
    sweep per fold (intrinsic to honesty). `--fast` skips the outer loop.
  - **I-3:** the family rule is NEW code over outer rows — do NOT reuse `SweepResult.Ship` (cross-family
    inner-argmax = the overfitter #32 replaces); only `PerFamily` is reused.
  - **Breaking change stated:** `metis run` no longer auto-ships (shipping → `select --promote`); migration
    surface (CUE schema, 2 shapes, RUNBOOK, ~4 tests) enumerated in Scope. Empty-`ship` guard under `--promote`.
