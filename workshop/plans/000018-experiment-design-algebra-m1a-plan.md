# experiment-design algebra M1a — Sampler fold node Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give metis a first-class *resample/sweep* construct — the **Sampler fold node** (`Init/Ask/Tell/Done`) — so an experiment declares `data│pipeline│ship` phases and the engine drives configs × folds through nested Samplers, producing an honest per-config `(mean, SE)` and shipping the selected winner via `driver:single`. Fold in the **input-addressed cache identity** (#24) as one coherent boundary.

**Architecture:** Resample and sweep collapse into ONE first-class graph node — an ask/tell fold instantiated at each level (driver ⊃ sweeper ⊃ resample); static scatter/gather is the degenerate no-op-`Tell` Sampler. M1a builds the node + the generic driver loop and wires only the **static** Samplers (grid over configs, fixed-k over folds); adaptive Samplers (#19 select, #23 nested, racing, Bayesian) are later impls against the same node. Everything in `pipeline` runs per-fold (the one structural leakage cut); `data` runs once; the partition materializes once above the sweeper; `ship` runs once on the winner. The interior cache becomes **input-addressed** — the key is the input recipe (config + seed + upstream `Kpre`s), computable pre-run — made sound by **transitive-`D`-closure validation** at hit-check (so an upstream code edit still invalidates downstream, while output nondeterminism does not).

**Tech Stack:** Go (pure core: `pkg/experiment`, `pkg/shape`, new `pkg/sampler`, `pkg/cache`, `pkg/ledger`; thin IO shell: `cmd/metis`), CUE (`construct/vocabulary/experiment.cue`), Python (`metis/…` sklearn step-types), `uv`.

**Design source-of-truth:** `workshop/pensive/2026-07-07-experiment-design-algebra.md` (§The Sampler fold node; §Cache mechanics; Evolution #5). Grounded in a verified survey of current metis + a fresh-eyes plan review (issue #18 Log; `workshop/lessons.md` "Plan authoring / cache-key changes").

**Plan altitude (deliberate, per operator's under-specification preference):** tasks fix the **entities, file touch-points, interfaces/signatures, test intent, and the subtle bits** (closed-CUE rewrite, strict parse, combined-DAG validation, input-addressed key + transitive-`D`, per-fold leakage, the all-rows ship signal). They do *not* paste every implementation line — the tier-1 promises are the signatures + the tests; the bodies are the implementer's (human or fork) tier-2 work. Follow TDD: red → green → commit.

**ARCH principles in play:** ARCH-DRY (reuse `$any` Expand, CAS, record/point-address, the trace, `experiment.Step`/`Validate`/`TopoSort`, per-entry `D` storage; generalize `pkg/sweep` rather than fork it) · ARCH-PURE (the Sampler node + driver loop + reducers + `Kpre` are pure; all subprocess/FS IO — incl. partition materialization — stays behind `StepExecutor`/`cmd/metis`) · ARCH-PURPOSE (deliver honest per-config `(mean,SE)` shipped via `driver:single` + the reusable Sampler node + a *sound* input-addressed cache — not a demo; the e2e must reproduce/beat v1 honestly, and the cache must invalidate on upstream code edits).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Shape` | `pkg/experiment/shape.go` | modified |
| `Sweeper` / `Resample` / `Driver` | `pkg/experiment/shape.go` | new |
| `Objective` (`+Select`) | `pkg/experiment/shape.go` | modified |
| `Sampler[S,P,O,R]` | `pkg/sampler/sampler.go` | new |
| `Run` (generic driver loop, `+Ctx`) | `pkg/sampler/run.go` | new |
| `FixedKFolds` (fold Sampler) | `pkg/sampler/folds.go` | new |
| `GridConfigs` (sweeper) | `pkg/sampler/configs.go` | new (moves `sweep.Grid`) |
| `SingleDriver` (degenerate outer Sampler) | `pkg/sampler/driver.go` | new |
| `Aggregate` (`(mean,SE)` reducer + told-set key) | `pkg/sampler/aggregate.go` | new |
| `Winner` (reconstructable run-keys) | `pkg/sampler/winner.go` | new |
| `Kpre` (input-addressed) | `pkg/cache/cache.go` | modified |
| `transitiveD` / `Validate` (closure hit-check) | `pkg/cache/cache.go` | modified |
| `Row` (raw per-fold) + `AggregateView` | `pkg/ledger/ledger.go` | modified |
| `fold_score` | `metis/model.py` | new (replaces `cv_score`) |

- **Shape** — the experiment lifted to a config-space, now **phase-structured**: `Data`, `Pipeline`, `Ship` (each `[]experiment.Step`) + `Sweeper` + `Driver`, replacing the flat `Steps` + `Sweep`.
- **Sampler[S,P,O,R]** — the load-bearing primitive: `Init(ctx)→S`, `Ask(S)→([]P, done)`, `Tell(S,P,O)→S`, `Done(S)→R`. Generic so the SAME loop drives every level (types compose: the sweeper's `runPoint = Run(resample,…)`, whose `R=(mean,SE)` is the sweeper's `O` — the reviewer confirmed this monomorphizes cleanly, no higher-kinded types).
  - **DRY rationale:** one construct subsumes today's Go sweep-loop, Python fold-loop, and `ledger.Best` select. Generalizes `pkg/sweep.Sampler` (adds `Init`/`Done`/batch-`Ask`).
- **FixedKFolds** — the fold Sampler: `Init(ctx)` consumes the **already-materialized** Partition (from `ctx` — materialized once above the sweeper, NOT here; keeps `Init` pure) and enumerates k fold-Points; `Ask` emits all k once (static); `Tell` accumulates a fold score; `Done` = `Aggregate`.
- **GridConfigs** — the sweeper: `Ask` emits config-points from `shape.Expand(Pipeline)`; `Tell` records `config→(mean,SE)`; `Done` = the `Winner` (argmax per `Objective`; `argmax-mean` in M1a — the swappable seam #19 replaces).
- **SingleDriver** — `driver:single`: degenerate outer Sampler — `Ask` one all-data point, `runPoint`=the sweeper, `Done`=passthrough of the winner. The seam #23's k-outer-fold Sampler replaces.
- **Aggregate** — pure `Reduce([]FoldScore) (mean, SE)` (SE = sample-std/√n); order-independent; keyed on the sorted told-set. A *different* `Done` (#19's 1-SE) re-reduces the same cached fold-scores for free.
- **Kpre (input-addressed)** — `hash(step_id, uses, with, seed, sorted upstream **Kpres**)` — the input recipe, computable pre-run from the DAG (no upstream *output* needed). Paired with **transitive-`D` validation** for soundness (below).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| driver-loop wiring | `cmd/metis/run.go`, `cmd/metis/sweep.go` | modified | orchestration |
| partition materialization (engine-synthesized) | `cmd/metis` + `metis/steps/cv_split.py` | new/modified | dataset IO |
| `cachingExecutor` (input-addressed + transitive-`D`) | `cmd/metis/caching.go` | modified | CAS + cache index |
| ledger sidecar (per-fold) | `cmd/metis` + `pkg/ledger` | modified | file IO |
| `#ExperimentShape` CUE | `construct/vocabulary/experiment.cue` | modified | `cue vet` merge-check |
| strict parse | `pkg/experiment/shape.go` | modified | `yaml.v3` |
| fold-aware Python steps + fold-context injection | `metis/steps/{cv_split,features,train}.py`, `metis/model.py` | modified | sklearn / FS |

- **partition materialization** — the Partition is **config-invariant** (depends only on dataset + seed + `resample.cv.{k,stratify}`), so the engine **synthesizes** a `cv-split` step from `sweeper.resample.cv` (single-source — NO explicit `cv-split` in the shape, NO `k`/`stratify` duplication) and runs it **once above the sweeper** (after `data`, before the config loop); its output (the partition artifact, content = which-rows) is cached + shared across all configs/folds and handed to `FixedKFolds.Init` via `ctx`. IO lives here; `Init` stays pure.
- **cachingExecutor (input-addressed + transitive-`D` snapshot)** — the folded-in #24: the executor's `Kpre` upstream term = upstream steps' **`Kpre`** (input identity), accumulated in `c.kpres[stepID]`. **Soundness:** because metis's read-set `D` deliberately excludes data/upstream artifacts (`trace.py` — they were the *only* carrier of upstream-code propagation under output-hashing), each step stores a **transitive-`D` snapshot in its OWN `Entry`** (`entry.TransitiveD` = the topo-fold `ownD ∪ ⋃_{d∈needs} c.transitiveD[d]`), and `isHit` re-hashes that *stored* snapshot against the current tree. NOT a walk of upstreams' live entries (inert — the executor heals an edited upstream's entry before the downstream is checked). Store & validate key on the same snapshot (symmetric); no upstream-entry lookup at validate (eviction-robust); diamond-correct. Robust to output nondeterminism (no output bytes in the key), sound for upstream code edits (the snapshot catches them). The record-provenance `Upstream` (output-hashes, via `buildRecord`/`assembleRecord`) is a SEPARATE path, unchanged. *(get-data keys on the slug string — #25, now load-bearing for data-change propagation; see Notes.)*
- **fold-context injection** — the engine injects a fold context into each pipeline-step run (like `seed` today — reproducibility is a runner concern): per-fold → `{partition, idx:i}`; ship refit → `{mode:all}`. `features` and `train` read it (see M1a-4 T14/T18).

---

## Chunk 1: M1a-1 — the schema (vocabulary)

Build the phase/Sampler vocabulary. Parse+validate ONLY. **Boundary close:** `cue vet` + parser round-trips the reshaped `titanic-sweep.md` + a strict-reject test.

> **Implemented (impl discovery):** removing the flat `Shape.Steps/Sweep/Experiment` fields breaks the 4 `cmd/metis` files (`run.go`/`sweep.go`/`ledger.go`/`ledger_cmd.go`) — the v1 flat-sweep paths **M1a-4 rewires** into the nested Sampler loop. Dependency-forced (the rewire needs `pkg/sampler`+`pkg/cache` first), so **`cmd/metis` stays RED until M1a-4**: intermediate boundaries scope their green to their own packages (`go build/test ./pkg/...`) and confirm breakage stays confined to `cmd/metis` (`go build ./...` names only it). Whole-module green returns at M1a-4.

### Task 1: Phase/Sweeper/Driver Go structs
**Files:** Modify `pkg/experiment/shape.go`; Test `pkg/experiment/shape_test.go`
- [ ] **Step 1 — failing test** `TestParseShape_v2`: unmarshal a `data│pipeline│ship` + `sweeper` + `driver:single` fixture into `Shape{Data,Pipeline,Ship []Step; Sweeper; Driver}`; assert the pipeline `$any` `with` survives untyped, `Sweeper.Resample.CV.K==5`, `Driver.Single != nil`.
- [ ] **Step 2 — run, expect FAIL.**
- [ ] **Step 3 — implement:** replace `Shape{Experiment; Sweep}` with phase fields + `Sweeper{Sampler string; Resample; Objective}`, `Resample{CV struct{K int; Stratify bool}}`, `Driver{Single *struct{}; CV *struct{K int; Stratify bool}}`, `Objective{Metric, Direction, Select string}`. Reuse `Step` per phase.
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-1: phase-structured Shape + Sweeper/Driver structs`

### Task 2: strict unknown-key parse
**Files:** Modify `pkg/experiment/shape.go` (`ParseShape`); Test `shape_test.go`
- [ ] **Step 1 — failing test** `TestParseShape_RejectsUnknownKey`: a shape with `sweeperr:` / an unknown top-level key → an error naming the key (closes the `yaml.v3` silent-drop footgun that would diverge from CUE's closed rejection — ARCH-PURE root-cause).
- [ ] **Step 2 — FAIL** (silently dropped today).
- [ ] **Step 3 — implement:** decode via `yaml.Decoder` + `KnownFields(true)`; surface the error.
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-1: strict unknown-key shape parse`

### Task 3: ValidateShape v2 — combined-DAG validation
**Files:** Modify `pkg/experiment/shape.go` (`ValidateShape`); Test `shape_test.go`
> **Fix (review):** the reshaped shape has **cross-phase** `needs` (`features(pipeline) needs [adapt(data)]`, `predict(ship) needs [train(pipeline)]`), but `experiment.Validate` resolves `needs` within ONE experiment. Validate the phases as **one combined DAG**, not per-phase.
- [ ] **Step 1 — failing tests:** a valid v2 shape passes; each of {cross-phase `needs` resolve, unique ids **across all phases**, acyclicity over the combined DAG, missing `sweeper.resample`, `driver` with neither/both single&cv, `objective.select ∉ {argmax-mean}`, empty `pipeline`} behaves correctly (valid→pass, broken→specific error).
- [ ] **Step 2 — FAIL.**
- [ ] **Step 3 — implement:** concatenate `Data++Pipeline++Ship` into one synthetic `Experiment` and run `experiment.Validate` on it (unique-ids + cross-phase needs + acyclicity + uses-format in ONE pass); then layer the shape-only checks (sweeper/driver/select/non-empty-pipeline). Do NOT validate phases in isolation.
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-1: ValidateShape v2 over the combined phase DAG`

### Task 4: CUE `#ExperimentShape` rewrite + drift guard
**Files:** Modify `construct/vocabulary/experiment.cue`; verify `scripts/merge-checks.d/experiment-validate.sh`; Test `shape_test.go`
- [ ] **Step 1 — failing check:** `validate-instance --type experiment-shape <reshaped shape>` FAILS today (closed schema has no phases/sweeper/driver).
- [ ] **Step 2 — implement:** rewrite `#ExperimentShape` — `data/pipeline/ship: [...#Step]`, `sweeper: {sampler; resample: {cv: {k:int, stratify?:bool}}; objective: {metric, direction, select}}`, `driver: {single?: {} | cv?: {k:int, stratify?:bool}}`. Keep **closed**. Factor a shared `_phase: [...#Step]` (ARCH-DRY, mirrors `_pipeline`).
- [ ] **Step 3 — run** merge-check + `validate-instance` → PASS on the reshaped shape, FAIL on a malformed one.
- [ ] **Step 4 — drift guard:** a Go test marshals a `Shape` and `cue vet`s it against `#ExperimentShape` (renamed/extra field fails); `t.Skip` if `cue` absent.
- [ ] **Step 5 — commit:** `#18 M1a-1: rewrite closed #ExperimentShape for phases/sweeper/driver`

### Task 5: reshape the live `titanic-sweep.md`
**Files:** Modify `kbench/competition/titanic/pipelines/titanic-sweep.md`
> **Note (review):** NO `cv-split` step — the resample is declared ONLY in `sweeper.resample.cv` (the engine materializes the partition, M1a-4 T15).
- [ ] **Step 1:** rewrite — `data:[get-data,adapt]`, `pipeline:[features($any), train(model $any)]`, `ship:[predict,submission]`, `sweeper:{sampler:grid, resample:{cv:{k:5,stratify:true}}, objective:{metric:accuracy,direction:maximize,select:argmax-mean}}`, `driver:{single:{}}`.
- [ ] **Step 2 — verify** parse+validate + `validate-instance` PASS. **Step 3 — commit:** `#18 M1a-1: reshape titanic-sweep.md (resample in sweeper, no cv-split step)`

---

## Chunk 2: M1a-2 — the pure Sampler core (`pkg/sampler`)

The pure fold-node core — interface, generic loop, the two static Samplers, the driver, the reducer, the winner. **Zero IO; every task unit-testable without subprocess/FS** (ARCH-PURE). **Boundary close:** `go test ./pkg/sampler/...` green; the nested-`Run` type-composition demonstrated on fakes.

### Task 6: Sampler interface + generic `Run` loop
**Files:** Create `pkg/sampler/sampler.go`, `pkg/sampler/run.go`; Test `run_test.go`
- [ ] **Step 1 — failing test:** a trivial `Sampler` (Init=0, Ask emits `[1,2,3]` once then done, Tell=+out, Done=sum) through `Run(ctx, smp, id)` returns 6; Ask fires once, Tell 3×, Done once.
- [ ] **Step 2 — FAIL.**
- [ ] **Step 3 — implement:** `type Sampler[S,P,O,R any] interface { Init(ctx Ctx) S; Ask(S) ([]P, bool); Tell(S,P,O) S; Done(S) R }` and `func Run[S,P,O,R any](ctx Ctx, smp Sampler[S,P,O,R], runPoint func(P) O) R`. Pure. **`Ctx`** carries the run-scoped inputs (seed, the materialized partition ref). *(Concrete-triple `RunFolds/RunConfigs/RunDriver` is a fallback only if generics snag — the reviewer confirmed they compose; prefer generic.)*
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-2: Sampler fold-node interface + generic Run loop`

### Task 7: `FixedKFolds` fold Sampler
**Files:** Create `pkg/sampler/folds.go`; Test `folds_test.go`
> **Note (review):** `Init` consumes the **already-materialized** Partition from `ctx` — it does NOT do IO (materialization is M1a-4 T15).
- [ ] **Step 1 — failing test:** `FixedKFolds{k:5}` over a fake Partition in `ctx`: `Ask` emits 5 fold-Points `(partitionRef, idx)` once then done; feeding 5 fake fold-scores, `Done` = `Aggregate` `(mean,SE)`; `Tell` order-independent.
- [ ] **Step 2 — FAIL. Step 3 — implement. Step 4 — PASS.**
- [ ] **Step 5 — commit:** `#18 M1a-2: FixedKFolds static resample Sampler (pure Init over materialized partition)`

### Task 8: `GridConfigs` sweeper + `Winner`
**Files:** Create `pkg/sampler/configs.go`, `pkg/sampler/winner.go`; move `sweep.Grid`; Test `configs_test.go`
- [ ] **Step 1 — failing test:** `GridConfigs` over `shape.Expand(pipeline)`: `Ask` emits all config-points once; feeding fake per-config `(mean,SE)`, `Done` = the `argmax-mean` `Winner` (deterministic tie-break); `Winner` carries config free-params + fold point-addresses + seed.
- [ ] **Step 2 — FAIL. Step 3 — implement** (reuse `shape.Expand`/`Point`; keep `pkg/sweep`'s stop-predicates, renamed into `pkg/sampler`). **Step 4 — PASS.**
- [ ] **Step 5 — commit:** `#18 M1a-2: GridConfigs sweeper + argmax-mean winner`

### Task 9: `Aggregate` reducer
**Files:** Create `pkg/sampler/aggregate.go`; Test `aggregate_test.go`
- [ ] **Step 1 — failing test:** `Reduce([]FoldScore) (mean, SE)` — mean=Σ/n, SE=sample-std/√n; a known 5-vector → known `(mean,SE)`; permuting input doesn't change output; the reduce key = the sorted told point-addresses.
- [ ] **Step 2 — FAIL. Step 3 — implement. Step 4 — PASS.**
- [ ] **Step 5 — commit:** `#18 M1a-2: (mean,SE) reducer keyed on sorted told-set`

### Task 10: `SingleDriver` (degenerate outer Sampler)
**Files:** Create `pkg/sampler/driver.go`; Test `driver_test.go`
- [ ] **Step 1 — failing test:** `SingleDriver`: `Ask` one all-data point; `runPoint`=the sweeper; `Done`=passthrough of the sweeper's `Winner`. Assert it returns exactly the sweeper's winner. (Documents the seam #23's k-outer-fold Sampler slots into.)
- [ ] **Step 2 — FAIL. Step 3 — implement. Step 4 — PASS.**
- [ ] **Step 5 — commit:** `#18 M1a-2: SingleDriver degenerate outer Sampler`

---

## Chunk 3: M1a-3 — cache identity (input-addressed + transitive-`D` snapshot) — folds in #24

The soundness-critical boundary. Swap the interior to input-addressed keying, made sound by storing a **transitive-`D` snapshot in each step's own cache `Entry`**. **NOT** by walking upstream entries at hit-check — that is unsound: the topo executor **heals** an edited upstream's entry (`recordMiss` overwrites `Entry.D` with the new code hash) *before* the downstream is validated, so the walk always re-hashes clean → the downstream serves a stale output (see `workshop/lessons.md`). Store the snapshot in the DOWNSTREAM's own entry so store & validate key on the same bytes (symmetric), it's eviction-robust (no upstream lookup at validate), and diamond-correct (set union). **Boundary-ordering invariant (A3):** M1a-3's soundness gate runs on the EXISTING linear pipeline (`adapt→features→train`) — no fold wiring needed; the per-fold surface arrives in M1a-4. **Boundary close:** the interior key is computable pre-run; an upstream **code edit** re-runs downstream; upstream **output nondeterminism / eviction** does NOT re-key downstream.

### Task 11: input-addressed `Kpre`
**Files:** Modify `pkg/cache/cache.go` (`Kpre`), `cmd/metis/caching.go` (executor key path → `c.kpres`); Test `pkg/cache/cache_test.go`
> **Fix (final review):** spell out the fate of each touch-point — the *executor's* key path switches to `c.kpres`; the record-provenance path is SEPARATE and unchanged; `c.outputs`/`recordOutput` become dead.
- [ ] **Step 1 — failing tests:** (a) a step's `Kpre` is computable **pre-run** from the DAG (recursively from roots, no upstream *output*); (b) a config change re-keys; (c) `Kpre` is sensitive to a per-run **fold coordinate** — assert a `_fold` term in `with` changes the key (forward-wired Kpre-visible in M1a-4 T15/17 — the B2 fix).
- [ ] **Step 2 — FAIL** (today `Kpre.Upstream` = output-hashes).
- [ ] **Step 3 — implement:**
  - `Kpre`'s upstream term = sorted upstream **`Kpre`s**; the executor sets `c.kpres[stepID]` **unconditionally right after `Kpre` is computed** in `Execute` (identical value on hit + miss — no special-cased hit repopulation, so no miss-path-forgets bug).
  - The executor's `c.outputs` + `recordOutput` were read ONLY by the old output-hash `Kpre`; now **dead for keying → drop them** (Simplicity-First: no consumer; provenance does not read them).
  - **Do NOT touch the record-provenance path:** `buildRecord`/`assembleRecord` keep the record's `Upstream` = upstream **output-hashes** via their own independent re-hash (`record.go`; `upstreamHashes` the *function* survives for them). **Reconcile the now-stale doc comments** (`caching.go:18-21`, `record.go:101-103`) that claim the executor's `Kpre` and the record derive an identical upstream term — post-M1a-3 they diverge by design (key = `Kpre`s; provenance = output-hashes).
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-3: input-addressed interior Kpre (metis#24)`

### Task 12: transitive-`D` snapshot in the step's own `Entry`
**Files:** Modify `pkg/cache/cache.go` (`Entry` +`TransitiveD`, `Validate`), `cmd/metis/caching.go` (`recordMiss`, `isHit`, a `c.transitiveD` accumulator); Test `cache_test.go` + `cmd/metis` caching test
> **Fix (re-review — the soundness hole):** validating against upstreams' *live* entries is inert (they heal before the downstream is checked). Store the snapshot in the downstream's OWN entry; validate the current tree against THAT.
- [ ] **Step 1 — failing test:** with `Entry.TransitiveD`, a 3-step chain `R→M→S`: at store time `S.TransitiveD = D_S ∪ D_M ∪ D_R`; moving a file in **R**'s code → re-hashing `S.TransitiveD` mismatches → `S` MISSes. A diamond (`S←A,S←B, both←R`) folds `R` **once** (sorted+deduped by `(repo,path)` → canonical bytes, A1).
- [ ] **Step 2 — FAIL.**
- [ ] **Step 3 — implement:** extend `Entry` with `TransitiveD []record.CodeRef`. In the executor, accumulate a **topo-order fold** over *direct* needs: `c.transitiveD[id] = ownD ∪ ⋃_{d ∈ step.Needs} c.transitiveD[d]` (needs only `step.Needs` + the accumulator — no DAG in the executor, I2 mooted). `recordMiss` stores it. **On a HIT, repopulate `c.transitiveD[id]` (and `c.kpres[id]`) from `entry.TransitiveD` in the `Execute` HIT branch, AFTER `isHit`** — NOT gated by `isHit`'s early-return, so an immutable-leaf / empty-`D` HIT still feeds its (possibly empty) closure to downstream folds. This repopulation is **load-bearing** (a downstream in the same run whose upstream HIT must still fold the upstream's closure — Task 13 pins it). `isHit`/`Validate` re-hash the **stored** `entry.TransitiveD` (reuse `hashDByRepo`; canonical sort+dedup by `(repo,path)`). Store & validate both key on `TransitiveD` (symmetry — `lessons.md`). **Migration guard:** the `Kpre`-term change orphans all non-root entries (an implicit cache-version bump); defensively treat a found **non-leaf** entry with **absent** `TransitiveD` as a MISS, so a nil snapshot can never vacuously HIT. **Note:** the reshaped `get-data` is a plain Go step (empty `D` → vacuous `Kpre`-only HIT), NOT a `cache:{leaf:immutable}` pin — the "leaf" language refers only to the `isImmutableLeaf` path.
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-3: transitive-D snapshot in the step's own Entry (sound input-addressing)`

### Task 13: end-to-end soundness gate + nondeterminism (existing linear pipeline)
**Files:** Test `cmd/metis` caching e2e (drives the current `adapt→features→train` pipeline — NO fold wiring)
> The reducer `Done`'s told-set key (over the fold runs' `Kpre`-identities) is validated in M1a-4 (T16/17) where real fold runs exist; the pure reducer-key unit test is Task 9. (I5.)
- [ ] **Step 1 — failing tests:** **(soundness gate)** run the pipeline warm, edit an upstream step's **code** (`features.py`) → the downstream (`train`) MISSes + re-runs — driven through the REAL topo executor incl. the recordMiss-heal ordering (a pure `Validate` unit test is structurally blind to this — it MUST be a real-executor e2e); **(nondeterminism/eviction)** evict + re-run an upstream so its **output** changes byte-for-byte, same code → downstream does **NOT** re-key (HITs); **(HIT-feeds-downstream — pins the repopulation seam)** warm cache → edit `train.py` (so `features` **HITS**, `train` MISSes + **re-stores** its snapshot from the repopulated `c.transitiveD[features]`) → THEN edit `features.py` → assert `train` MISSes. **Reverting the `c.transitiveD` HIT-repopulation line must make THIS test FAIL** while the all-MISS gate above stays green — the all-MISS test is blind to a dropped repopulation, which only surfaces one edit later (final-review lesson).
- [ ] **Step 2 — run, FAIL → GREEN.**
- [ ] **Step 3 — commit:** `#18 M1a-3: e2e soundness gate (upstream code edit → downstream MISS) + nondeterminism-robustness`

---

## Chunk 4: M1a-4 — IO integration (fold-aware pipeline, partition, ledger, wiring)

Make the pipeline per-fold, materialize the partition once above the sweeper, persist raw fold rows, and wire the nested loop. **Boundary close:** a config × 5-fold run persists 5 raw rows + an aggregated `(mean,SE)`; the `data` steps + partition run once; adding one config recomputes only its folds.

### Task 14: fold-aware Python pipeline (`features`, `train`, `fold_score`)
**Files:** Modify `metis/model.py`, `metis/steps/train.py`, `kbench/steps/titanic/features` (+ its py); Test `tests/test_model.py` + step fixtures
> **Fix (review):** `features` must emit BOTH the analysis rows AND the assessment rows transformed by the **analysis-fitted** transform (else `train` has no assessment set to score). Thread the fold-context.
> **Fix (change-code judge):** the fold loop + `np.mean` live in `metis/model.py:cv_score`, NOT `train.py` (which delegates to `cv_score`). Extract `fold_score` from `model.py`; `train.py`'s change is to call `fold_score` (one fold) instead of `cv_score` (loop→mean).
- [ ] **Step 1 — failing tests:** `fold_score(X_ana,y_ana,X_ass,y_ass,kind,seed,params)` fits on analysis, scores on assessment → one accuracy; `train` given fold-context `{partition,idx}` emits `metrics{fold_score}` + the per-fold model; `features` given `{partition,idx}` fits its transforms on analysis rows, emits analysis+assessment both transformed by the analysis fit.
- [ ] **Step 2 — FAIL.**
- [ ] **Step 3 — implement:** add `model.fold_score` (extract the single-fold body from `model.py:cv_score` — the loop + `np.mean` live there); rewrite `train.py` to read the fold-context, subset analysis/assessment, and call `fold_score` for that one split (replacing the `cv_score` call) → emit ONE `fold_score`; make `features` fold-aware (fit-on-analysis, transform-both). Keep the fit-on-all path for ship (M1a-5).
- [ ] **Step 4 — PASS** (`uv run pytest`). **Step 5 — commit:** `#18 M1a-4: per-fold features/train + fold_score (analysis-fit, assessment-transform)`

### Task 15: engine-synthesized partition (once above the sweeper)
**Files:** Modify `cmd/metis` (partition materialization), `metis/steps/cv_split.py`; Test reuse `tests/test_split.py` (`cv_folds` lives in `metis/split.py`) + `cmd/metis`
> **Fix (review):** the Partition is config-invariant → synthesize a `cv-split` step from `sweeper.resample.cv` (single-source) and run it ONCE above the sweeper; cache + share; hand it to `FixedKFolds.Init` via `ctx`.
- [ ] **Step 1 — failing test:** the engine, given `sweeper.resample.cv:{k:5,stratify:true}`, materializes the partition once (after `data`, before the config loop) via the `cv-split` step-type; content = which-rows; k folds disjoint+covering; deterministic under seed; NOT re-run per config. (M1a: one assignment artifact + fold-index; k-separate-partition granularity is a later optimization — note in `## Log`.)
- [ ] **Step 2 — FAIL.**
- [ ] **Step 3 — implement:** `cv_split.py` stays (reuse `cv_folds`); the engine derives its `with` from `resample.cv` + the `adapt` output, runs it once, threads the partition's **identity** (its `Kpre`/content-hash, not just a path — so it can enter each per-fold `Kpre` in T17) into the Sampler `ctx`.
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-4: engine-materialized partition once above the sweeper`

### Task 16: raw per-fold ledger + aggregate view
**Files:** Modify `pkg/ledger/ledger.go`; Test `ledger_test.go`
- [ ] **Step 1 — failing test:** N configs × k folds → N×k raw rows (`config + foldIdx + fold_score`); `AggregateView(l)` groups by config over the fold coord → N `(mean,SE)` rows; raw rows persist so #19 re-reduces without a re-run.
- [ ] **Step 2 — FAIL** (today: one reduced row per config). **Step 3 — implement:** extend `Row`/CSV with the fold coordinate; add `AggregateView` (pure read-time group-by). Keep append-only + point-address dedup.
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-4: raw per-fold ledger rows + read-time aggregate view`

### Task 17: nested driver-loop wiring (fold-context `Kpre`-visible + per-fold executor scope)
**Files:** Modify `cmd/metis/run.go`, `cmd/metis/sweep.go`; Test `cmd/metis` integration (fake executor)
> **Fix (re-review B2+I1):** the fold coordinate MUST enter `Kpre` (overlay into `with`), and each per-fold point-run needs its OWN executor accumulator scope.
- [ ] **Step 1 — failing tests:** (a) a fixture shape (2 configs × 2 folds) with a **fake** `StepExecutor`: `data`+partition execute once → `Run(SingleDriver, · → Run(GridConfigs, c → Run(FixedKFolds, f → runPipelineFold(c,f))))` → a `Winner` + N×k ledger + per-config `(mean,SE)`; pipeline runs per (config,fold). (b) **two folds of the same step get DISTINCT cache entries** (the B2 collision guard — revert the `with` overlay and this FAILS).
- [ ] **Step 2 — FAIL.**
- [ ] **Step 3 — implement:** `runPipelineFold(c,f)` runs `{data steps (cache-HIT) + pipeline steps}` as its own point-run with a **fresh executor scope** (fresh `c.kpres`/`c.transitiveD`/`c.outputs` — matching today's per-point `newCachingExecutor`; shared `data`/partition steps HIT the on-disk cache and repopulate the in-memory accumulators from their entries, per M1a-3 T11/12). Overlay config `c` **and** the fold-context `{_fold:{partition:<partition-identity>, idx:i}}` into the pipeline steps' `with` (so `Kpre` sees the fold — B2); end at `train` → `fold_score`. Nest via `Run`; persist raw fold rows.
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-4: nested loop — fold-context Kpre-visible + per-fold executor scope`
- [ ] **Step 6 — retire `pkg/sweep`:** once `cmd/metis/sweep.go` is rewired onto `pkg/sampler`, DELETE `pkg/sweep` (its `Sampler`/`Grid`/stop-predicates are superseded by `pkg/sampler`) + fix the stale `atlas/index.md:53-54` lines that call `pkg/sweep` *the* sweep sampler. (M1a-2 built `pkg/sampler` **additively**, leaving `pkg/sweep` intact — deleting it before this rewire would only trade the field-undefined `cmd/metis` red for import-not-found red; the removal is coupled to the wiring, so it lands here.)
- [ ] **Step 7 — commit:** `#18 M1a-4: retire pkg/sweep (superseded by pkg/sampler)`

---

## Chunk 5: M1a-5 — select + driver:single ship + Titanic e2e + atlas

Close the loop. **Boundary close:** the reshaped `titanic-sweep.md` runs end-to-end, selects a winner honestly, ships a `submission.csv`, reproduces/beats v1. `sdlc close --issue 18`.

### Task 18: driver:single ship — refit winner + predict + submission
**Files:** Modify `cmd/metis` (ship phase); reuse `metis/steps/predict.py`, `kbench/steps/titanic/submission`; Test integration
> **Fix (review):** specify the **all-rows signal** — the ship refit injects fold-context `{mode:all}`; BOTH `features` (fit+transform ALL rows, no assessment split — a NEW path for the kbench features step) and `train` (fit on all — its existing path) honor it.
- [ ] **Step 1 — failing test:** given a `Winner`, `ship` runs `pipeline` with `{mode:all}` (features+train fit on ALL training rows) → `predict` on test → `submission.csv` of the expected shape/ids.
- [ ] **Step 2 — FAIL. Step 3 — implement** the `{mode:all}` branch in `features` + thread it through `train`/predict/submission on the winner's config.
- [ ] **Step 4 — PASS.** **Step 5 — commit:** `#18 M1a-5: driver:single ship — all-rows refit, predict, submission`

### Task 19: reconstructable winner run-keys
**Files:** Modify `pkg/sampler/winner.go`, `cmd/metis`; Test `winner_test.go`
- [ ] **Step 1 — failing test:** the `Winner`'s run-keys (config free-params + fold point-addresses + seed + provenance) reconstruct the exact shipped config; `ship` rebuilds from the run-keys, not by re-deriving hyperparameters.
- [ ] **Step 2 — FAIL. Step 3 — implement. Step 4 — PASS.**
- [ ] **Step 5 — commit:** `#18 M1a-5: reconstructable winner run-keys → ship`

### Task 20: Titanic e2e (honest)
**Files:** Create/extend `cmd/metis` e2e; manual run against `kbench/…/titanic-sweep.md`
- [ ] **Step 1 — failing e2e:** `metis run titanic-sweep.md` — `data`+partition once, sweeper over (features×model × 5 folds), per-config `(mean,SE)`, select argmax-mean, `driver:single` ship → `submission.csv`. Assert: a `(mean,SE)` leaderboard; the winner ships; a re-run cache-hits unchanged points; a one-hyperparameter change recomputes only affected folds; **an upstream code edit re-runs downstream** (the M1a-3 soundness gate, end-to-end).
- [ ] **Step 2 — run (deterministic clock + fake git), FAIL → GREEN.**
- [ ] **Step 3 — manual honest check:** run the real shape; confirm the honest per-config `(mean,SE)` and that the promoted winner reproduces/beats v1's `~0.81 cv`/`0.77990 public` without selection-overfit inflation (record numbers in `## Log`). Live Kaggle submission is operator-gated (RUNBOOK), not automated.
- [ ] **Step 4 — commit:** `#18 M1a-5: Titanic e2e through the Sampler algebra (honest)`

### Task 21: atlas + close
**Files:** Modify `atlas/` (+ `atlas/index.md`)
- [ ] **Step 1:** document the driver/sweeper/resample **Sampler fold node** algebra — enumerate the `pkg/sampler` surface (`Sampler[S,P,O,R]` + the generic `Run` loop · `FixedKFolds` · `GridConfigs` · `SingleDriver` · `Aggregate`→`(mean,SE)` · `Winner` run-keys) + the **static-vs-adaptive** plannability line (grid/fixed-k are the feedback-free degenerate Samplers; #19/#23/racing/Bayesian are later impls against the same node) — plus the three-phase shape and the input-addressed + transitive-`D` cache identity; link every new file in `atlas/index.md`.
- [ ] **Step 2 — commit:** `#18 M1a-5: atlas — Sampler algebra + three-phase shape + input-addressed cache`
- [ ] **Step 3 — close:** `sdlc close --issue 18` (measured `--actual`, `--verified` with e2e + soundness-gate evidence).

---

## Revisions

- **2026-07-07 (M1a-2 review):** the Core-concepts "`GridConfigs` **moves** `sweep.Grid`" was realized as **fork-now / remove-at-M1a-4** — M1a-2 built `pkg/sampler` additively and left `pkg/sweep` intact (a transient duplication), because deleting `pkg/sweep` is coupled to the `cmd/metis/sweep.go` rewire (M1a-4 Task 17 Step 6): removing it earlier only trades one flavor of `cmd/metis` red for another. `pkg/sweep`'s stop-predicates were likewise NOT ported into `pkg/sampler` (they're an adaptive-sampler / early-stop feature; M1a wires only static samplers that exhaust naturally — YAGNI), and remain in `pkg/sweep` for reuse when the first adaptive sampler lands (#19+).

## Cross-repo boundary (change-code judge)

Several tasks mutate the **kbench peer repo**, not metis: **T5** (`titanic-sweep.md`), **T14** (`kbench/steps/titanic/features`), **T18** (`kbench/steps/titanic/submission`), **T20** (manual run). metis's `sdlc milestone-close`/`close` mandatory fresh-eyes review diffs the **metis** tree (`BASE_SHA..HEAD`), so kbench-side changes **escape the automated boundary review**. Before touching kbench: read its `AGENTS.local.md` + `MEMORY.md` (per AGENTS.md peer-repo rule). Commit the kbench edits in kbench's tree, and **run a separate fresh-eyes review over the kbench diff** at the M1a-4/M1a-5 boundaries (the metis milestone-close won't cover it). The metis engine changes stay reviewable under metis's own gates.

## Notes / open seams (not M1a)

- **#25 get-data root gap** — get-data keys on the slug string, not the ingested bytes. **Now load-bearing (final review):** input-addressing drops the output-hash term, so get-data's root hash becomes the SOLE propagator of a same-slug data change downstream — an evicted+re-fetched *changed* dataset won't re-key downstream until #25 content-addresses the ingest. The pipeline is deterministic-given-inputs so no new interior bug class appears, but #25 is elevated from "root gap" to "data-change propagation gap." Untouched here.
- **#19 select rule** — M1a's `Objective.Select` is `argmax-mean` only; `one-std-err`/`mean−λ·std` are a swappable `Done` over the same cached fold-scores.
- **#23 nested-CV** — `driver:cv` is the k-outer-fold Sampler replacing `SingleDriver`; result-dependent; built ON this substrate.
- **#20 target cross-fit** — the feature step's own internal cross-fit; no engine marker; built on the per-fold `pipeline` position.
- **k-separate partitions** — M1a uses one assignment artifact + fold-index; per-fold cache granularity is a later optimization, not a correctness property.
- **transitive-`D` cost** — `gitBlobHashes` already batches a repo's paths into one `git hash-object` (`trace.go`), so it's O(steps) batched subprocesses — negligible for M1a's shallow chain. (No per-run hash memoization is implemented; don't assume it. If a deep pipeline ever makes the closure re-hash dominate, add a per-run `(repo,path)→hash` memo.) Not an M1a blocker.
- **estimate_hours** — set on the issue frontmatter at `sdlc change-code` (post-approval), against the calibrated estimate source; not typed from memory.
