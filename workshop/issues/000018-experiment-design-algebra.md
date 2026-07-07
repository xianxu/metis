---
id: 000018
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 7.0
started: 2026-07-07T11:08:31-07:00
---

# experiment-design algebra M1a — three-phase shape + Sampler fold node (static samplers, per-fold pipeline, driver:single)

## Problem

The sweep treats data-splitting (CV) as an internal detail of `train` and selects by raw cv-max on a
single split → selection-overfitting (the metis-v1 gap: ~0.81 cv → 0.78 public). Resampling and
selection aren't first-class; the workbench can't produce an honest per-config mean/std, and the
structure that nested-CV (#23) and leakage-safe features (#20) need doesn't exist.

## Spec

metis-v2 **M1a** — the substrate everything else depends on. Full design:
**`workshop/pensive/2026-07-07-experiment-design-algebra.md`** (read first). The converged model
(supersedes the earlier `fold: {$cv: cv-split}` resample-axis framing):

- **Three-phase shape:** `data` (get-data/adapt — run ONCE, above the resample) │ `pipeline` (the swept
  algorithm×hyperparameter atom — always per-fold) │ `ship` (predict/submission — winner only). The
  `data│pipeline` boundary is the ONE structural cut → cross-fold leakage-safety with no per-step markers.
- **Sampler fold node (the load-bearing construct):** resample + sweep are ONE first-class graph node —
  an ask/tell fold `Init/Ask/Tell/Done` — instantiated at each level (driver ⊃ sweeper ⊃ resample);
  static scatter/gather is the degenerate Sampler (no-op `Tell`, `Ask` emits its whole point-set once).
  **M1a builds the node + the driver loop but wires only the STATIC Samplers** — grid (over configs) +
  fixed-k (over folds); adaptive Samplers (#19 select = a different `Done`, #23 nested = an outer
  Sampler, racing, Bayesian) are later impls against the same node. Generalizes the metis#7 Ask/Tell seam
  to the resample level. Hands the driver the winner's **reconstructable run-keys**. = mlr3 `AutoTuner`.
- **The resample Sampler's `Done` → `(mean, SE)`:** each per-fold Point is a cached run emitting ONE
  fold's score; `Done` reduces the told fold-scores → `(mean, SE)`, keyed on the sorted told-set. #19's
  select is a *different `Done`* over the same cached fold-scores — free, no re-run. (Today `cv_score`
  reduces to a bare mean *inside* `train`, discarding fold rows — M1a lifts that out.)
- **Point/Partition as first-class artifacts:** the fold Sampler materializes partition artifacts
  (content = which-rows); a per-fold Point keys on its partition + config. Shared `data` steps run once
  (emergent from the cache); `pipeline` steps run per-fold.
- **`driver: single`** for M1a — the degenerate outer Sampler (fit sweeper on all → ship winner). The
  outer `driver:cv` (nested-CV, an adaptive-nesting Sampler) is **metis#23**.

Keeps metis's two-phase key (`K_pre` → validate); folds AND code read-set are runtime-manifested.
Prior art: mlr3 (`AutoTuner` = our sweeper), tidymodels (three-phase), sklearn (Pipeline per-fold).
**DESIGN-FIRST:** durable plan (superpowers-writing-plans → `workshop/plans/`) before code.

## Done when

- A shape declares `data│pipeline│ship` + a `sweeper` with `resample: {cv:k}`; the engine drives the
  **Sampler fold node** (grid over configs, fixed-k over folds), runs each per-fold Point as a cached run
  emitting one fold-score, and the resample Sampler's `Done` reduces → per-config `(mean, SE)` (not one
  lucky split); the sweeper selects a winner and ships it via `driver: single`.
- Partition enters the cache key as a first-class artifact; get-data/adapt cache once, features/train per
  fold (emergent from the DAG); the `Done` CV score is content-addressed + order-independent (keyed on
  the sorted told-set).
- Titanic runs through the new shape and reproduces (or beats) v1's promoted winner honestly.
- atlas: the driver/sweeper/pipeline algebra + three-phase shape documented.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.2  impl=0.4
item: typed-data-prototype   design=0.2  impl=0.4
item: milestone-review       design=0.0  impl=0.2
item: greenfield-go-module   design=0.5  impl=0.8
item: milestone-review       design=0.0  impl=0.2
item: smaller-go-module      design=0.2  impl=0.45
item: smaller-go-module      design=0.2  impl=0.45
item: milestone-review       design=0.0  impl=0.2
item: smaller-go-module      design=0.15 impl=0.35
item: smaller-go-module      design=0.2  impl=0.45
item: milestone-review       design=0.0  impl=0.2
item: smaller-go-module      design=0.2  impl=0.4
item: atlas-docs             design=0.05 impl=0.15
item: milestone-review       design=0.0  impl=0.2
design-buffer: 0.15
total: 7.0
```

Item → boundary map: **M1a-1** smaller-go-module (structs+parse+combined-DAG validate) + typed-data-prototype
(closed `#ExperimentShape` CUE rewrite + drift guard) + milestone-review. **M1a-2** greenfield-go-module
(`pkg/sampler`: interface+Run+FixedKFolds+GridConfigs+SingleDriver+Aggregate+Winner) + milestone-review.
**M1a-3** smaller-go-module ×2 (input-addressed `Kpre` + transitive-`D` snapshot in `pkg/cache`; `caching.go`
executor rewire + real-executor soundness-gate e2e) + milestone-review. **M1a-4** smaller-go-module ×2
(fold-aware Python `features`/`train`/`fold_score`; partition materialization + per-fold ledger + nested
wiring) + milestone-review. **M1a-5** smaller-go-module (driver:single ship + run-keys + Titanic e2e) +
atlas-docs + milestone-review. Design near floor (a thorough 3×-reviewed plan doc); impl at 40%-of-v2
(v3.1); +15% thorough-plan buffer; 5 milestone-reviews (5 boundaries). Calibration source stale (#127) →
provisional.

## Plan

- [ ] (spec → durable plan) three-phase shape parse+validate; the Sampler fold node (`Init/Ask/Tell/Done` + driver loop) with STATIC samplers (grid/fixed-k); Point/Partition artifacts + per-fold pipeline; resample `Done` → (mean,SE); input-addressed cache (#24); select + driver:single ship; Titanic e2e.

## Log

### 2026-07-07
- Filed as metis-v2 M1 (the core). Design in the pensive + project (`sources`). The operator's frame:
  resampling & selection are first-class, declarative axes with per-axis reducers — one algebra, not loops.

### 2026-07-07 (design converged)
- Reframed from "M1 + nested CV" to **M1a** (the sweeper substrate); nested-CV split to **metis#23**.
  Converged model: driver/sweeper/pipeline + three-phase shape + no fit_scope marker + input-addressed
  cache leaning (metis#24). Superseded the `fold: {$cv: cv-split}` axis framing after a 3-front prior-art
  survey (mlr3 is the structural twin). Design + reshaped titanic-sweep.md in the pensive.

### 2026-07-07 (Sampler fold node + grounding)
- **Model pivot: scatter/gather → the ask/tell Sampler fold node** (`Init/Ask/Tell/Done`). Resample is
  first-class like a step; driver/sweeper/resample are the SAME node nested. Static scatter/gather = the
  no-op-`Tell` degenerate Sampler. **M1a builds the node + wires only the static samplers** (grid over
  configs, fixed-k over folds); #19 (a different `Done`), #23 (an outer Sampler), racing/Bayesian are
  later impls. Chose Option **A** (first-class graph construct) over **B** (flat expansion axis +
  read-time group-by) — B pushes nested-CV into imperative glue. Full model in the pensive (§The Sampler
  fold node, Evolution #5).
- **Grounding survey of current metis** (reuse-vs-build, verified against code). REUSE-AS-IS — the
  metis#7 Ask/Tell seam (`pkg/sweep`), `$any` expansion (`pkg/shape`), the validating trace, layer/step
  discovery. REUSE-WITH-CHANGE — the sweep loop / ledger / promote; the cache `K_pre` (swap the upstream
  term output-hash→input-recipe for #24: `cache.go:36` + `caching.go:22,311`). BUILD-NEW — the three-phase
  + sweeper/driver structs (+ the CLOSED `#ExperimentShape` CUE rewrite + strict-unknown-key parse); the
  Sampler fold node + driver loop; per-fold persistence (today `model.py:cv_score` returns a bare mean,
  discarding fold rows) + the `Done` reducer; Point/Partition artifacts (the runner `run.go:55` is a
  linear topo-sort, no scatter/gather). Correction to the earlier framing: the "free read-time reduction"
  was NOT reuse — the ledger stores one reduced row per config today; lifting the fold loop out is
  build-new.

### 2026-07-07 (durable plan authored + reviewed)
- Durable plan: `workshop/plans/000018-experiment-design-algebra-m1a-plan.md`. A fresh-eyes review
  verified all code claims and caught two blocking issues + fixes now applied: **(1)** input-addressing
  is unsound without **transitive-`D` closure validation** (the read-set excludes data, so output-hash
  was the only code-propagation carrier); **(2)** the partition-materialization seam (config-invariant →
  engine-synthesized from `sweeper.resample.cv`, once above the sweeper; `FixedKFolds.Init` stays pure);
  plus combined-DAG (not per-phase) validation, `features` must emit analysis+assessment transformed by
  the analysis fit, and the ship all-rows signal. **#24 folded in** as its own `cache identity` boundary.
  Plan restructured into **5 review boundaries**: M1a-1 schema · M1a-2 pure Sampler core · M1a-3 cache
  identity (#24) · M1a-4 IO integration · M1a-5 ship+e2e. Lessons → `workshop/lessons.md`.
