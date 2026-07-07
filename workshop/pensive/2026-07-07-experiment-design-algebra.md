---
status: active
type: pensive
created: 2026-07-07
updated: 2026-07-07
---

# The experiment-design algebra — driver · sweeper · pipeline

Design source-of-truth for the metis-v2 project (`brain/data/project/metis-v2-experiment-algebra.md`).
Product of a long design conversation with the operator (2026-07-07) plus a three-front prior-art
survey (ML frameworks · config/sweep/adaptive · reproducible-pipeline caching). The trigger: the
metis-v1 Titanic winner scored **~0.81 cv → 0.77990 public** — a textbook selection-overfit gap.
Piling on features/models while selecting by raw cv-max just overfits harder. The fix isn't more
knobs; it's making the workbench **find honest, non-overfit performance** — which is an *algebra*
problem, not a pile of loops.

> **Supersedes** the original framing in this file (a `$fold`/`$cv` *resample axis* threaded as a
> `fold: {$cv: cv-split}` coordinate in `train.with`). That was correct in spirit — fold as a
> first-class value, reuse the cache chain — but the layering was wrong. The converged model below
> is simpler and prior-art-validated. See **§Evolution** for what changed and why.

## The shape: three phases, two execution layers, one atom

An experiment shape has **three phases** and the engine wraps the middle one in **two execution
layers**:

```
data:      get-data → adapt         ── produced ONCE, above the resample · shared across folds
                                        (author's home for run-once/invariant work)
  ┌─ DRIVER (outer)  ── evaluate the sweeper HONESTLY · single | cv | nested · reduces by AGGREGATE
  │    └─ SWEEPER    ── BLACK BOX: training-data → winner · owns inner-CV + objective + select rule
  │         └─ PIPELINE  ── the (algorithm × hyperparameter) atom · one config × one fold → a score
ship:      predict → submission     ── runs ONCE on the promoted winner (driver:single, refit on all)
```

Three ideas do the load-bearing work:

1. **The structural cut is the `data│pipeline` phase boundary — nothing else.** Everything in
   `pipeline` runs *per-fold* (it is downstream of the split), which **structurally guarantees
   cross-fold leakage-safety with zero per-step markers**. `data` is the author's home for run-once
   work; if you want something computed once, you place it in `data`. (No `fit_scope`/`over:` knob —
   see §No markers.)

2. **The sweeper is a black box: `training data → winner`.** It *owns* its inner resampling (how it
   scores each candidate config), its objective, and its selection rule. This is mlr3's `AutoTuner`
   (inner resampling is a constructor arg), sklearn's `GridSearchCV(cv=…)`, tidymodels'
   `tune_grid(resamples=…)`. The `(mean, SE)` per config is **internal** to the sweeper's selection —
   it is *not* a channel to an upper layer.

3. **The driver is the outer, honest evaluator — and it is optional.** `single` = fit the sweeper on
   all data and ship the winner (no honest estimate). `cv`/`nested` = run the sweeper on each
   outer-train, hand it a *sealed* outer-test, and aggregate the outer scores → the honest procedure
   estimate. Crucially, the sweeper hands the driver the winner's **reconstructable run-keys** (config
   + the keys that pin the exact run + provenance) — the reproducibility identity, not just abstract
   hyperparameters — so `ship`/assessment can rebuild it faithfully.

`nested-CV` is then just `driver:cv` wrapping a sweeper that already has an inner `resample:cv`:
`driver(sweeper[inner-cv](pipeline))` — isomorphic to mlr3's `resample(AutoTuner(resample(learner)))`,
doctrine included: **the outer resample estimates; only the sweeper selects.**

## Estimation vs selection — the two knobs, in their proper homes

The operator's original distinction, now cleanly located:

- **Selection** (*which config do I ship?*) is the **sweeper's** job. Its inner-CV produces `(mean, SE)`
  per config; its **select rule** turns that into a winner. Today's `cv-max` = `argmax-mean` — biased
  toward the config that got lucky on the fold noise. The lever is the rule (metis#19): **1-standard-
  error rule** (among configs within 1 SE of the best, pick the simplest — Breiman; tidymodels
  `select_by_one_std_err`) or `mean − λ·std`. This is the single capability **no** surveyed framework
  offers — our differentiator — and it is *only possible* because the sweeper carries the `(mean, SE)`
  vector (free from read-time reduction) and the tagged `$any` tree gives a parsimony ordering.
- **Estimation** (*how good is the procedure?*) is the **driver's** job. `single` vs nested `cv` is the
  estimator knob; it attacks *selection optimism* without changing *which* config is selected. Nested-CV
  **produces no winner** — different outer folds may legitimately select different configs; it estimates
  the *tune-then-fit procedure*. The winner exists only when the sweeper runs on all data (the ship path).

They compose: swap the sweeper's select rule to 1-SE, and the driver's nested-CV then honestly
estimates *that* better policy.

## No markers — cross-fold safety is structural; target-safety is the step's own job

We considered a per-step `fit_scope` (`stateless`/`row-fit`/`target-fit`) property. **Dropped** — a
hand-declared marker is error-prone, and it isn't needed:

- **Cross-fold leakage** is handled *structurally*: everything in `pipeline` is per-fold. An author who
  wants run-once work places it in `data`. The phase boundary is the only cut, and it can't be
  mis-set per step.
- **Within-fold target overfit** (e.g. group-survival-mean using the label to build a feature) is the
  **feature step's own responsibility** — it cross-fits/shrinks internally, exactly as sklearn's
  `TargetEncoder` bakes cross-fitting into its `fit_transform` and tidymodels `step_lencode_mixed`
  uses shrinkage. The engine doesn't need to know.
- If we ever want the engine to *enforce* target-safety, we **derive** "this step reads the target
  column" from a **data-read trace** (metis already traces which *code* files a step reads — extend to
  which *columns*), never a hand-typed tag. Derive, don't declare.

Cost of dropping the marker: the engine can't auto-lint a leaky placement (matches sklearn — author
discipline / the feature step owns it). Benefit: no error-prone ceremony, one structural mechanism.

## Cache mechanics — input-addressed identity, fold-as-artifact, fan-in reducer

A step's cache identity is its **input recipe**: `(algorithm/feature-schema + hyperparameters) +
code content-hash + which-rows`. That tuple uniquely determines the output.

- **Fold enters as a distinct artifact, not an integer.** The driver/sweeper materializes k partition
  artifacts; each holds the actual row-assignment and thus a distinct content-hash = "which rows".
  A downstream per-fold step consuming `fold-i` re-keys **for free** via the existing upstream chain,
  and — unlike a bare integer — the identity tracks the *content* (seed/k change → rows change → hash
  changes → correct re-key). This is the Nextflow-channel / Nix-derivation model. (For the fold there
  is no param-vs-artifact tension: the partition's output *is* which-rows.)
- **The shared/per-fold boundary is emergent** (Nextflow/Nix/Make consensus): a step is per-fold **iff**
  it transitively consumes the partition artifact — a reachability query on the DAG, zero declarations.
  `get-data`/`adapt` (no fold input) run once; `features`/`train` run per fold.
- **The reducer is a fan-in (gather) node** keyed on the **sorted set of all k manifested fold row-content
  hashes** → a content-addressed, order-independent CV score. Scatter (1→k partition) and gather (k→1
  mean/SE) are the symmetric bookends of the resample layer. All k per-fold results are cached
  individually (so adding one config recomputes only its folds).

**Two runtime-manifested facts** (the key is *not* fully static from the shape file):
- the **folds** (seed + data → the actual row partitions → their hashes), and
- the **code read-set** (metis's *validating trace* discovers which code files a step read *during* the
  run, not before).

So the key resolves in **two phases**, and the design must keep this: `K_pre` (pre-run, from
config + seed + upstream) → *run* → **validate** against the discovered read-set. Static plannability
is *partial* — the structure (`$fold`, the DAG) is known up front; the exact fold-content and
code-read-set are runtime-established. The reducer specifically can't be keyed before its folds
manifest.

Two cache decisions fell out of the survey and are filed separately:
- **metis#24 — input-addressed vs output-hash-chained interior.** metis today keys on *upstream
  output-hashes* (content-addressed → "early cutoff"). The operator leans **input-addressed** (key =
  the input recipe): statically *plannable* (a sweep planner can compute the cache-hit map + cost up
  front) and robust to upstream output non-determinism (ML outputs are rarely byte-reproducible, so
  early-cutoff rarely fires anyway). Real fork, touches existing cache architecture → its own issue.
- **metis#25 — root gap.** `get-data` keys on the dataset *path string*, not its bytes/size/mtime — a
  same-path data mutation is a silent stale hit that nothing downstream can catch. metis is the weakest
  of six surveyed systems here (below even Make's mtime). Content-address (or size+mtime) the ingested
  dataset. Orthogonal soundness bug.

## The reshaped `titanic-sweep.md` (M1a flat; nested = one-line driver swap)

```yaml
---
type: experiment-shape
id: titanic-sweep
competition: titanic
seed: 42
status: active

# ── DATA ──  produced ONCE, above the resample · shared across all folds
data:
  - id: get-data
    uses: kaggle/download
    with: {competition: {slug: titanic}}
  - id: adapt
    uses: titanic/adapt
    needs: [get-data]
    with: {raw: get-data, out: ../data/titanic}

# ── SWEEPER ──  black box: training data → winner · owns inner-CV + objective + select
sweeper:
  sampler: grid                    # the degenerate ask/tell sampler (asks for every point); future: bayes
  resample: {cv: {k: 5, stratify: true}}          # INNER CV — how each config is scored
  objective: {metric: accuracy, direction: maximize, select: argmax-mean}
  #                                    select: argmax-mean | one-std-err | pct-loss   (#19)

# ── PIPELINE ──  the swept (algorithm × hyperparameter) atom · per config × per fold
pipeline:
  - id: features
    uses: titanic/features
    needs: [adapt]
    with:
      dataset: adapt               # driver/sweeper rewires to the CURRENT fold's analysis/assessment split
      features: {$any: [[], [title], [title,family], [title,family,age], [title,family,age,fare], [title,family,age,fare,deck,embarked]]}
    # M5 ticket-group-survival feature cross-fits internally (its own responsibility — no marker)
  - id: train
    uses: metis/train              # the model — inherently per-fold; fits analysis, scores assessment → accuracy
    needs: [features]
    with:
      model: {$any: {logreg: {C: {$any: [0.1,1,10]}}, rf: {n_estimators: {$any: [200,500]}, max_depth: {$any: [4,8]}}}}

# ── DRIVER ──  outer honest evaluation of the sweeper · optional
driver:
  single: {}                       # M1a: fit sweeper on all data → ship winner (no honest estimate)
  # cv: {k: 5, stratify: true}     # M1b (metis#23): nested-CV — honest procedure estimate (~5× compute)

# ── SHIP ──  runs ONCE on the promoted winner · driver:single, refit on all data
ship:
  - id: predict
    uses: metis/predict
    needs: [train]
  - id: submission
    uses: titanic/submission
    needs: [predict]
---
```

Run-count (flat, features(6) × model(7)): `get-data`/`adapt` = 1 each (shared); `features` = 6×5 = 30;
`train` = 6×7×5 = 210; reducer nodes gather per config; `ship` = 1 each (winner only). Re-running with
one changed hyperparameter cache-hits everything upstream and unaffected.

## Prior art (validated our model; three sharp additions)

- **mlr3 is the structural twin.** `driver(sweeper(pipeline))` = `resample(AutoTuner(resample(learner)))`.
  Use its three-object correspondence (Resampling=driver, AutoTuner=sweeper, Learner=pipeline) as the
  sanity check; adopt its doctrine ("nested resampling estimates a tuned model's performance; it is not
  a selection method").
- **tidymodels** = the three-phase decomposition (rsample data │ workflow+tune │ `fit_best`/`last_fit`);
  the right rejection of caret's monolithic `train()`. `select_by_one_std_err`/`select_by_pct_loss` are
  the metis#19 menu. Preprocessing is refit per fold inside the workflow — the "whole workflow per-fold"
  cut we made structural.
- **sklearn** — `Pipeline` fit-per-fold (leakage-safe by construction); `TargetEncoder`'s internal
  cross-fitting is the model for the target-feature's own-responsibility cross-fit. `refit=True`
  *conflates* sweep and ship — a trap we avoid (ship is a separate phase).
- **Sweeper should be ask/tell (define-by-run), not pre-expand** (Ax/Optuna `study.ask/tell`/Ray
  `Searcher`). Grid is the *degenerate* sampler that asks for every point. Our metis#7 Sampler seam
  (Ask/Tell) is thereby **vindicated** — no rework; grid is a sampler over the space, not a static
  expansion.
- **Our tagged `$any` is best-in-class** — it expands branches *disjointly* and emits *sparse* configs,
  the two things Optuna's `GridSampler` gets wrong. It matches Ax `dependents` / ConfigSpace conditions
  and beats Hydra/W&B. The one primitive we lack is a `ForbiddenClause` (cross-branch prune) — future.
- Keep `$any` **strategy-agnostic data** (declare the space, never bake "grid" in); split discrete
  `$any:[...]` from a future continuous-range construct that's adaptive-native (grid discretizes it).
- **1-SE / robust selection is uncontested across all six systems** — the sharpest differentiator, and
  the operator's own intuition ("prefer the less-overfitting near-winner"), named.

## Milestone map (see the project file for the portfolio view)

- **M1a — metis#18:** the sweeper substrate — three-phase shape (`data│pipeline│ship`), black-box
  sweeper with inner-CV, read-time reduction → `(mean, SE)`, fold-as-artifact scatter + fan-in reducer,
  `driver:single`. Unblocks everything.
- **M1b — metis#23:** nested-CV — the outer `driver:cv` wrapping the sweeper; the result-dependent
  select-then-assess-on-sealed-outer-fold; the honest procedure estimate. Deps #18.
- **M2 — metis#19:** the sweeper's select rule (`one-std-err`/`mean−λ·std`) over `(mean, SE)` + a
  parsimony ordering from the tagged tree.
- **M3 — metis#20:** the target-feature's internal cross-fit (leakage-safe ticket-survival). No engine
  marker; the step owns it. Deps #18.
- **M4 — metis#21 (GBM), metis#22 (ensembling).** Independent; startable now.
- **M5 — kbench#8:** ticket-group survival + the honest Titanic validation (nested-CV estimate tracks
  public within noise). Deps #20.
- **Cross-cutting:** metis#24 (cache addressing decision), metis#25 (get-data root-hash gap).

## Evolution (what changed, and why)

1. **`fold: {$cv: cv-split}` coordinate → driver/sweeper layers.** The fold-in-`with` framing tangled
   resampling into the step-DAG. Prior art (mlr3/tidymodels/sklearn) universally puts resampling *around*
   the workflow, not inside a step. The driver/sweeper split is that, and it dissolved the "where does
   the outer split attach" question entirely.
2. **`over:` cut-knob → structural `data│pipeline` cut + no markers.** No mature framework exposes a
   movable cut (it's a leakage footgun); all draw one structural cut. The operator pushed to drop the
   error-prone per-step marker; cross-fold safety is structural, target-safety is the step's own job.
3. **Inner CV as a peer layer → folded into the sweeper.** "Score a config by CV then pick" *is* tuning;
   the inner resample belongs to the sweeper (mlr3 AutoTuner). This collapsed three peer layers to two,
   and corrected the "sweeper↔driver channel" mis-framing — the `(mean, SE)` is internal to the sweeper;
   only the winner (as reconstructable run-keys) crosses to the driver.
4. **Output-hash-chained cache → (leaning) input-addressed.** Filed as metis#24. The reducer's key must
   incorporate all folds' *manifested* row-content; the code read-set is likewise trace-discovered — so
   the two-phase (`K_pre` → validate) key stays, and full static plannability is only partial.
