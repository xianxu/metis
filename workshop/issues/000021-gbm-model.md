---
id: 000021
status: codecomplete
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-11
estimate_hours: 0.6
started: 2026-07-11T21:50:20-07:00
actual_hours: 0.62
---

# GBM model branch — HistGradientBoosting model step-type

## Problem

The model set is only `logreg` + `rf`. Gradient boosting is usually the strongest model on tabular
data like Titanic, and the workbench can't sweep it.

## Spec

metis-v2 M4a (project + pensive). Add **HistGradientBoosting** (sklearn, no extra dep; XGBoost/LightGBM
optional later) to `metis/model.py` as a `$any`-map model branch:
`model: {$any: {hist_gbm: {learning_rate: …, max_iter: …, max_leaf_nodes: …}}}`. **Independent of
M1** — a clean quick win + a real exercise of the metis#17 tagged-`$any` model set. Known hyperparams
applied, unknowns ignored (the existing forward-compatible contract).

**Recon (Explore digest):** model kinds are single-sourced in `metis/model.py` — exactly three touch
points enumerate kinds (`MODELS` frozenset, `make_model`, `complexity`); everything downstream
(`train.py`, the whole `pkg/` Go layer, ledger, select rule) dispatches by opaque string and derives
families **structurally** from the `$any`-map tag (`sampler.FamilyOf`). So this is a **Python-only**
change — **zero Go edits**. `parse_model_config` is already kind-agnostic (no change). One-fit-feeds-both
is load-bearing: `fold_fit` returns `(score, fitted_model)` so `complexity` reads the *same* fit.

**`make_model` branch** (mirroring rf's `p.get` forward-compat): `learning_rate` (0.1), `max_iter`
(100), `max_leaf_nodes` (31), `max_depth` (None), `+ random_state=seed`. Model stays fully general
(all knobs sweepable).

**`complexity(fitted, "hist_gbm")` = TOTAL realized leaves summed across ALL boosted trees**
(`sum` over `fitted._predictors` of `get_n_leaf_nodes()`), NOT mean-per-tree like rf. Boosting is
**additive** (F(x)=Σ trees; ESL §10.2 Eq.10.4, Friedman 2001), so capacity SUMS; more iterations DO
increase overfitting-capacity (ESL §10.12 optimal M\*; Bühlmann–Hothorn df(m)=trace(𝐁ₘ)↑m; Google DF:
"unlike random forests, gradient boosted trees *can* overfit") — the sharp contrast with rf's
n_estimators-neutral MEAN (bagging averages → variance-reduction, not capacity). XGBoost's own Ω=**γT**+…
penalizes total leaves summed across the ensemble. Mean-per-tree would be M-neutral → blind to boosting's
primary regularizer → **affirmatively wrong**. Realized (not configured `max_iter × max_leaf_nodes`):
early stopping makes `n_iter_ < max_iter`. (Literature pass, this session — full citations in Log.)

**Caveat + containment:** learning_rate **shrinkage** decouples leaf-count from effective DoF *across* ν
(a low-ν/many-tree config has more leaves yet is often better-regularized). Contained **structurally**
at the shape level (the literature's preferred fixed-ν stratum, zero code cost): the baseline shape
**fixes `learning_rate`** and sweeps `max_iter × max_leaf_nodes`, where total-leaves is a clean monotone
DoF proxy. The `complexity` docstring documents that a ν-sweeping shape would need a ν-weighted measure;
defer that correction until a real sweep shows the misranking (measure-before-rebuild).

## Done when

- `model: {$any: {hist_gbm: {...}}}` trains + cross-validates + sweeps through the ledger like logreg/rf.
- Unit test on the model factory; the titanic sweep can include a gbm branch.

## Plan

Single-pass atomic (one review boundary, closes in one `sdlc close`). TDD.

- [x] `hist_gbm` in `make_model` (learning_rate/max_iter/max_leaf_nodes/max_depth via `p.get` + `random_state=seed`) + add to `MODELS`; import `HistGradientBoostingClassifier`.
- [x] `complexity(fitted, "hist_gbm")` = TOTAL realized leaves summed across boosted trees (`sum` over flattened `_predictors`); docstring: sum-not-mean rationale (additive/boosting) + the fixed-ν caveat.
- [x] Tests (`test_model.py`): parametrize `hist_gbm` into shape/determinism; hyperparam-applies; `test_complexity_hist_gbm_total_leaves` pinning the sum + **max_iter-SENSITIVITY** (inverse of rf's neutrality — more rounds → strictly more total leaves).
- [x] Test (`test_steps.py`): per-fold `hist_gbm` emits `fold_score` + `complexity` > 0 (end-to-end kind flow).
- [x] Titanic baseline shape: add `hist_gbm` branch (fixed `learning_rate`, sweep `max_iter × max_leaf_nodes`); config-count comment updated (33). `metis run -dry-run` confirms 33 configs incl. 12 hist_gbm (real-data ledger run is the operator-gated Kaggle step).

## Estimate

Extend a well-specced module (`model.py`: `make_model` + `complexity` branches + mirrored tests + a
shape-fixture branch) — one real, literature-grounded design decision (the complexity measure, now
pre-resolved, so design trends low) — plus the close-time fresh-eyes review. Zero Go edits (families
derive structurally). `smaller-go-module` is used as the closed vocabulary's **language-agnostic
size anchor** ("extend a well-specced module"); the work is Python, but the reference-class scope matches.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.2 impl=0.2
item: milestone-review    design=0.0 impl=0.15
design-buffer: 0.15
total: 0.6
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Log

### 2026-07-07
- Filed as metis-v2 M4a. Independent additive win (startable in parallel with M1). Design in the project/pensive.

### 2026-07-11 (claimed; design converged — literature-grounded)
- 2026-07-11: closed — Python 50 pass incl. new test_complexity_hist_gbm_total_leaves (pins exact total-leaves sum + max_iter-sensitivity m40>m10) + test_steps per-fold gbm complexity>0; Go 9/9 ok, zero Go edits (FamilyOf structural); metis run -dry-run → 33 configs incl 12 hist_gbm at fixed learning_rate. Real-data ledger run operator-gated (Kaggle).; review verdict: SHIP
- Recon + design in Spec above. Core decision (hist_gbm complexity = total realized leaves, sum not mean)
  validated by a boosting-complexity **literature pass**: Friedman 2001 (stagewise additive expansion);
  ESL §10.2 (boosting = additive model, Eq.10.4), §10.11 (tree size J → interaction order J−1), §10.12
  (optimal M\*, shrinkage ν<0.1 needs larger M), ch.15 (RF = variance reduction, does not overfit as B↑);
  Chen–Guestrin 2016 XGBoost (Ω=γT+½λ‖w‖², T=#leaves per tree, summed across the ensemble);
  Bühlmann–Hothorn 2007 (df(m)=trace(𝐁ₘ)↑ with m; shrinkage slows per-step DoF growth → the ν caveat);
  Efron 2004 (effective DoF = Σ cov(μ̂ᵢ,yᵢ)/σ²); Google DF course ("unlike RF, GBT can overfit"); Breiman
  2001 (RF convergence). Verdict: sum is correct (mean would be M-neutral, blind to boosting's primary
  regularizer); mean-per-tree affirmatively wrong for GBM.
- **ν-shrinkage caveat contained structurally** (fix learning_rate in the baseline shape = fixed-ν
  stratum; model branch stays ν-general); ν-weighted measure deferred to when a real sweep shows the
  misranking. est 0.6h (derived against v3.1; superseded the pre-derivation 1.5h gut number).

### 2026-07-11 (implemented + verified — TDD)
- **model.py** (3 touch points): `MODELS += hist_gbm`; `make_model` branch (learning_rate/max_iter/
  max_leaf_nodes/max_depth via `p.get` + `random_state`); `complexity` branch = `sum(t.get_n_leaf_nodes()
  for stage in fitted._predictors for t in stage)` — flattened (list-of-lists: one inner list per
  iteration, K predictors per K-class; binary→1), per the plan-judge's non-blocking API-shape note.
- **Tests** (RED→GREEN): parametrized `hist_gbm` into shape/determinism; hyperparam-applies; new
  `test_complexity_hist_gbm_total_leaves` pins the exact sum AND max_iter-sensitivity (m40 > m10 total
  leaves — the additive-capacity property, inverse of rf's neutrality); `test_steps` per-fold gbm emits
  complexity > 0. **Python 50 passed; Go 9/9 ok** (zero Go edits — `FamilyOf` picks up `hist_gbm`
  structurally, confirmed by the pre-existing generic `configs_test.go "gbm"` family).
- **Shape:** titanic-baseline gained a `hist_gbm` branch (learning_rate fixed at 0.1, sweep
  max_iter×max_leaf_nodes); `metis run -dry-run` → **33 configs, 12 hist_gbm**, learning_rate absent from
  the swept free-params (fixed-ν stratum confirmed). Full real-data ledger run is operator-gated (Kaggle).
- **Atlas:** `experiment.md` (3-kind model set + "adding a kind is Python-only") + `index.md` (complexity
  now lists hist_gbm total-leaves). Estimate tightened to 0.6h per the estimate-judge (design 0.3→0.2).
