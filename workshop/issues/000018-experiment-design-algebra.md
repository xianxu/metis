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

Durable plan: `workshop/plans/000018-experiment-design-algebra-m1a-plan.md` (5 review boundaries).

- [x] M1a-1 — schema: phase-structured Shape + Sweeper/Driver structs, strict unknown-key parse, combined-DAG ValidateShape, closed `#ExperimentShape` CUE rewrite + drift guard, reshaped titanic-sweep.md. *(`cmd/metis` red until M1a-4 — dependency-forced; green scoped to `pkg/experiment`+CUE.)*
- [x] M1a-2 — pure Sampler core: Sampler interface + generic Run + FixedKFolds/GridConfigs/SingleDriver + Aggregate(mean,SE) + Winner (`pkg/sampler`; zero-IO).
- [x] M1a-3 (foundations landed) — `Entry.TransitiveD`+`MergeTransitiveD` (pure, `bae19a1`) + `model.fold_score` (`b407f3d`). The merge revealed the boundary is a ~1000-line sweep-driver re-architecture (dependency order is IO-first) → **reordered + split** into M1a-3a/M1a-3b (both SHIPPED):
  - [x] M1a-3a — **IO rewire → `cmd/metis` GREEN** (was M1a-4, reordered first): nested-Sampler loop (`run.go`/`sweep.go` off flat `Shape`) + fold-aware `train.py`/`features` + engine partition + per-fold ledger + retire `pkg/sweep`, on the EXISTING output-hash cache. Close: whole-module `go build ./...` green + Titanic sweeper → `(mean,SE)` leaderboard. **SHIPPED (FIX-THEN-SHIP → fixed).**
  - [x] M1a-3b — **cache #24 + soundness gate** (was M1a-3, reordered second — now testable): input-addressed `Kpre` + transitive-`D` snapshot wired into `caching.go` (foundations already committed) + real-executor soundness e2e (edit `features.py` → `train` MISSes). **SHIPPED (review SHIP, 2 Minors fixed).**
- [x] M1a-5 — ship + e2e: driver:single ship (all-rows refit→predict→submission), reconstructable winner run-keys, honest Titanic e2e, atlas. *(M1a-4 label retired — folded into M1a-3a.)*

## Log

### 2026-07-07
- 2026-07-07: closed M1a-3a — M1a-3a IO rewire: whole-module go build/vet/test ./... + both experiment merge-checks green (Go rewire 1bb1783; io/train handoff 067870b). Nested-Sampler fold loop wired (sampler.Run(GridConfigs)⊃Run(FixedKFolds)); retired pkg/sweep. Real OFFLINE Titanic smoke sweep (fake-kaggle serving the committed fixture) → honest per-config (mean,SE) leaderboard + argmax-mean winner; ledger holds 4x3=12 raw per-fold rows (fold coord); warm re-run cache-hits everything; metis ledger show --sort reduces to the 4-config aggregate; one-knob change recomputes only affected folds (Go shapesweep_test). kbench full suite green (36) incl. leak-free fold-fit unit test; metis-v1 plain-experiment threads still pass. Scope: EXISTING output-hash cache (#24=M1a-3b); ship+real-data-reproduce=M1a-5.; review verdict: FIX-THEN-SHIP
- 2026-07-07: closed M1a-2 — go test ./pkg/sampler/... green — 14 tests incl. TestRun_NestedComposition (real 3-level driver⊃sweeper⊃resample; generics monomorphize, NO concrete-triple fallback needed), FixedKFolds/GridConfigs(argmax-mean+tie-break)/SingleDriver/Aggregate(mean,SE,told-set key,order-independent). Pure zero-IO greenfield pkg/sampler. pkg/sweep untouched (additive; retired at M1a-4); go build ./... still names ONLY cmd/metis (no new breakage). Deviation: stop-predicates NOT ported (YAGNI — adaptive-sampler surface; M1a wires only static grid/fixed-k). Bypasses: --no-atlas (atlas at M1a-5), --no-actual (contaminated forked actuals; reconcile at full close), --no-project (project tracks at M1a grain).; review verdict: FIX-THEN-SHIP
- 2026-07-07: M1a-2 review fixes (FIX-THEN-SHIP → shipped) — `Run` empty-batch guard (panic on a non-progressing `Ask`, was a silent-hang risk) + test; `Winner` seed single-sourced from `Ctx.Seed` (dropped `GridConfigs.Seed` — two-sources-for-one-fact); scheduled `pkg/sweep` retirement at M1a-4 (plan Task 17 Step 6 + `## Revisions`) — a **transient `pkg/sweep`↔`pkg/sampler` duplication** that resolves when `cmd/metis` rewires off `pkg/sweep`; enriched Task 21 atlas scope. Deferred to M1a-4 (per reviewer, when the wiring feeds real values): `staticAsk` consolidation of the two identical static `Ask`s, a `folds.go` `K<0` guard, and empty-`Done`/minimize-tie-break test coverage.
- 2026-07-07: closed M1a-1 — pkg/... build+vet+test green (independently re-verified). Real-cue drift-guard (TestShapeConformsToCUE) + closedness test + both merge-checks (experiment-schema-selftest, experiment-validate) pass. cmd/metis red is dependency-forced+confined (go build ./... names ONLY cmd/metis — the v1 flat-sweep paths M1a-4 rewires); whole-module green returns at M1a-4. kbench titanic-sweep reshape c55b549 cue-validated. Bypasses: --no-atlas (atlas at M1a-5/Task 21), --no-actual (per-boundary active-time contaminated in forked+interleaved session; reconcile at full close), --no-project (project tracks at M1a grain; row updated at full close).; review verdict: FIX-THEN-SHIP
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

### 2026-07-08 (M1a-3a Go rewire — SHIPPED, whole module green)
- 2026-07-08: closed M1a-5 — driver:single ship wired (Run(SingleDriver)⊃Run(GridConfigs)⊃Run(FixedKFolds)); winner refits all-rows→predict→submission — real smoke run ships a valid PassengerId,Survived submission (no cv-split, capture_status=captured all 6 steps); promote reconstructs a runnable experiment from a per-fold ledger (+honest sweep_estimate); HonestE2E ties the algebra + the metis#24 soundness gate end-to-end (mutation-checked teeth). metis 9 Go pkgs + both merge-checks + 42 pytest, kbench 36 pytest (incl e2e) green.; review verdict: FIX-THEN-SHIP → fixed (two fresh-eyes reviews via the in-harness Agent tool — binary judge network-blocked in sandbox; metis diff over 3535102..HEAD = FIX-THEN-SHIP: 1 Important — the driver:single ship silently skipped metis#14 code-capture, fixed by dropping inSweep for the ship + a teeth test asserting CaptureStatus=captured, re-verified on the real smoke run; 3 minors fixed; kbench diff = SHIP)
- 2026-07-08: closed M1a-3b — input-addressed Kpre + transitive-D snapshot (metis#24): go build/vet/test ./... green; 3 real-executor soundness gates (upstream-code-edit→downstream MISS via the stored closure; output-nondeterminism→HIT; HIT-feeds-downstream repopulation) each mutation-proven to have teeth; migration guard + direct codec test; both experiment merge-checks pass. Fresh-eyes review (in-harness Agent, binary judge network-blocked) = SHIP, 2 Minors fixed.; review verdict: SHIP (fresh-eyes review dispatched via the in-harness Agent tool over 03344c1..HEAD — the binary's nested-claude judge needs Anthropic-API network the sandbox blocks; 0 Critical / 0 Important, 2 Minors fixed: pkg/cache package-header staleness + a direct []≠nil codec test)
- **M1a-3a IO rewire DONE + committed (`1bb1783`).** `cmd/metis` rewired onto the nested-Sampler fold
  loop — `go build/vet/test ./...` + both `experiment` merge-checks all green (the all-or-nothing-compile
  boundary that bounced 3 forks is passed). Changes: `run.go` dispatch (tolerant Parse peeks type →
  experiment-shape is ALWAYS the nested sweep; +`runOpts.exec` test seam); `sweep.go` the nested loop
  (`sampler.Run(GridConfigs) ⊃ sampler.Run(FixedKFolds)`, per-fold experiment = data ++ synthesized
  cv-split ++ pipeline with config+`_fold` overlaid → Kpre fold-distinct; **failing fold is FATAL**,
  replacing v1 record-and-continue; preserves manifest/#14-capture/#8-ledger/detect-abort); ledger (pkg +
  cmd) Row+`Fold` coordinate + `AggregateView` (raw per-fold rows → per-config (mean,SE), #19 re-reduces
  free); `promotedExperiment` from phases; **retired `pkg/sweep`**; dropped `--max-points`. Replaced the
  v1 flat-sweep e2e suite with `shapesweep_test.go` (fake-exec fold-model integration: winner + N×k ledger
  + per-config (mean,SE); fold-distinct cache; warm re-run HITs; one-knob change recomputes only affected;
  dry-run; failing-fold-fatal; detect-abort). **Wired onto the EXISTING output-hash cache** (NOT #24).
- **Discovery — the data-handoff soundness ripple (map under-specified this).** The map framed kbench
  `features` fold-awareness as "the last Python piece"; it is actually a **cross-pipeline data-plane
  change**. `collectArtifacts` only walks the step dir, so `features`' exp-relative output
  (`../data/titanic-feat`) is INVISIBLE to the cache → a per-fold `features` HIT + `train` MISS reads
  stale shared data (fails the "one-hyperparam change recomputes only affected folds" bar). Fix: `features`
  must write its enriched dataset to its STEP dir (captured artifact) and `train` must read it via
  `upstream_path`. That ripples into the **plain experiments** `titanic-baseline.md` / `titanic-features.md`
  (train reads the exp-relative dataset) → `train.py` needs dual-mode reads OR those pipelines migrate too.
  Phase B (features fold-aware + captured output, train dual-read, shape migrations, kbench smoke e2e) is a
  distinct dedicated effort — checkpointed to a fresh session (see the superseding continuation).

### 2026-07-08 (M1a-3a Phase B + boundary review — FIX-THEN-SHIP → fixed, SHIPPED)
- **Phase B done:** fold-aware `features` (analysis-only fit, leak-free) writing a CAPTURED step-dir
  artifact; `train` reads the per-fold handoff via `io.dataset_dir` (upstream-artifact-or-exp-relative,
  dual-mode so the plain experiments stay green); `titanic-sweep(-smoke).md` migrated to the phase model;
  smoke e2e migrated. Verified OFFLINE (fake-kaggle + committed fixture): the nested loop → honest
  per-config `(mean,SE)` leaderboard + winner; 12 raw per-fold ledger rows; warm re-run all-HIT;
  `metis ledger show --sort` → 4-config aggregate. Commits: metis `067870b`, kbench `c8d1d28`.
- **Boundary reviews:** metis fresh-eyes (auto, `sdlc milestone-close`) = **FIX-THEN-SHIP**; kbench
  fresh-eyes (separate — kbench escapes the metis window) = **SHIP** (leakage cut / row-alignment /
  value-identity all confirmed correct + tested). Fixed the metis findings before crossing:
  - **Critical** — `promote --best` emitted a corrupt (NUL-in-frontmatter) + unrunnable experiment on a
    per-fold ledger. Fixed: `AggregateView` identity is NUL-free (`|` sep); `promote` **refuses cleanly**
    on a per-fold ledger (full reconstruction = M1a-5); restored a promote guard test. Confirmed e2e
    (promote on the real smoke ledger → clean error, no file written).
  - **Important** — restored `promote`/`ledger show`/`AggregateView`/`hoistShapePath` test coverage
    (`ledger_cmd_test.go`, `pkg/ledger` unit tests); `objective.metric` → `train.fold_score` (fixture +
    shapes); `atlas/index.md` `pkg/sweep` bullet rewritten to `pkg/sampler` (dangling-ref + reversed
    failure-semantics fix); `io.dataset_dir` unit test.
  - **Minor** — dropped `writeSweepLedger`'s dead `objective` param; stale `M1a-4` comment; kbench body
    prose (`accuracy`→`train.fold_score`); fold-fit test row-order assertion.
- **M1a-5 items surfaced by review:** `predict.py` still resolves `dataset` via `exp_path` (not
  `io.dataset_dir`) → the ship `predict.dataset: features` will fail until M1a-5 reconciles it; the driver
  level is inlined (`SingleDriver` built+tested but not in the call path) → #23 must insert the outer wrap.
