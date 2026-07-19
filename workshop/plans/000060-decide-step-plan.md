# Probabilities + Decide Implementation Plan (metis#60)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Steps use checkbox (`- [ ]`) syntax.

**Goal:** The decision layer (arena2 M4): per-class threshold tuning as a swept, sealed, honest part of the training procedure, plus probability outputs everywhere they're needed — with family blending (M2) reduced to a small verb over those outputs.

**Architecture — the load-bearing decision: tuning is LEAF-LOCAL, the engine is untouched.** The naive design (a cross-fold tuning step) needs per-config OOF aggregation *above* the leaf level — engine surgery, and a new honesty seam to defend. Instead, the tuned offsets are treated exactly like any other fitted parameter (the impute-median precedent): learned INSIDE each leaf from its analysis rows only. Mechanically, `fold_fit` under `decide=offsets` does: (1) split the fold's TRAINING rows 80/20 (seeded, stratified); (2) fit an auxiliary model on the 80, produce held-out probabilities on the 20; (3) grid-tune per-class log-offsets maximizing the declared metric on that held-out slice; (4) fit the main model on ALL training rows (unchanged from today); (5) score the assessment fold with `argmax(log_proba + offsets)`. The assessment fold honestly measures the *whole* procedure (fit + tune) because the tuning never saw it; the nested-CV seal covers everything with zero changes; cost is 2 fits per leaf (documented; the #58 sampling dial exists precisely to price such things). The ship refit does the same on all rows and persists `offsets.json`; `predict` applies it. Why an internal HOLDOUT and not in-sample tuning: an overfit model's training-row probabilities are overconfident, so offsets tuned on them are garbage — the aux-fit holdout is the cheapest honest OOF source (a full internal k-fold would be k× cost for 2 parameters' worth of tuning).

**Tech Stack:** Python only (sklearn `predict_proba` on both families — verified this session incl. class_weight+NaN; numpy vectorized grid search). No Go changes (`decide` is a `with` key like `metric` — Kpre re-keys shapes that set it, absent key leaves cohorts untouched).

## Milestones

- **M1 (this plan's tasks): probabilities + `decide: offsets`** — pure core, train/predict wiring, tests, docs. Boundary: `sdlc milestone-close`.
- **M2: `metis blend`** — DETAILED (post-M1, 2026-07-19):
  - **CLI:** `metis blend <shape.md> --runs <id1,id2,...> [--weights w1,w2,...]` (equal weights default; weights normalized, loud on count mismatch or non-positive).
  - **Combination rule (the settled design question #1):** average in **tilted log-space** — member i contributes `w_i · (log(clip(p_i)) + o_i)` where `o_i` is ITS persisted offsets (zeros when the run has no `offsets.json`) — then argmax. Each member's tuned decision layer is respected without re-tuning ("weights only" holds); rf's load-bearing tilt and gbm's ≈0 tilt both carry through. Validation: identical id sets/order and identical `proba_*` column sets across runs (loud otherwise); classes from the columns.
  - **Materialization (the settled design question #2):** write `runs/blend-<short-hash-of-inputs>/` containing: `predict/predictions.csv` (id + decided prediction — the SAME artifact shape the ship predict step emits), `record.json` (a blend-flavored record: member run ids + weights + the shape's get-data step `with` carried over so `kaggle submit --run` resolves the slug via runref), then **exec the shape's ship `submission` step** via the existing single-step exec path (env contract + step-path discovery exactly as `metis run` does — reuse, don't reimplement) with `with: {predictions: predict}` → `submission/submission.csv`. `kaggle submit --run blend-...` then works unchanged.
  - **Honesty (accepted, recorded in Revisions):** no in-sweep OOF for blends — leaderboard-measured only; the verb prints this caveat.
  - Tests: pure combine fn (tilted-log averaging: hand-built 2-member case where the blend flips a boundary row the argmax-average would not; zero-offset members ≡ plain log-average; weight normalization; mismatched ids/columns loud). Step-level: materialized dir has the right artifacts; submission step runs on the fixture workspace (reuse the in-process harness where possible; the exec path may need the e2e-style built binary — implementer judges, mirroring how promote is tested today).
  - **SDLC note:** M1 merged early (cross-repo dep, logged); M2 works on a FRESH branch name (#148 no-reuse) — `git checkout -b 000060-m2-blend` declared loudly as the change-code-already-ran continuation (the gates passed once for this issue; do not re-run change-code).

## Core concepts

### Pure entities (all `metis/model.py` unless noted)

| Name | Lives in | Status |
|------|----------|--------|
| `predict_proba(estimator, X)` | `metis/model.py` | new |
| `tune_class_offsets(proba, y, metric) → offsets` | `metis/model.py` | new |
| `apply_offsets(proba, offsets) → labels` | `metis/model.py` | new |
| `parse_decide(raw) → (rule, params)` | `metis/model.py` | new |
| `fold_fit(..., decide=...)` | `metis/model.py` | modified |

- **`tune_class_offsets`** — vectorized grid search over per-class additive offsets in log-probability space (class 0 pinned to 0 → 2 free params for 3 classes; grid `np.linspace(-4, 4, 41)` per free class — ±4 covers −log-prior optima down to ~1.8% priors (review issue 2: ±3 could NOT reach the 90/6/4 test prior's log(22.5)≈3.11); coarse-to-fine NOT needed at 41² points × vectorized argmax). Additive-in-log = multiplicative prior reweighting — the Bayes-correct family for a prior-shifted metric. Deterministic (no RNG; ties broken by first-max). Score via `resolve_scorer(metric)` (#59 — ONE scorer source). Returns `np.zeros(K)` when tuning cannot improve on no-op (grid includes 0).
- **`apply_offsets`** — `argmax(log(clip(proba)) + offsets, axis=1)`; clip guards log(0). Used by fold scoring AND predict.
- **`parse_decide`** — `"argmax"` (default) | `{"offsets": {"holdout": 0.2}}` — the `parse_model_config` loud-misuse pattern; holdout ∈ (0,0.5] validated.
- **`fold_fit(X, y, folds, fold_idx, kind, seed, params, metric, decide)`** — under `("offsets", p)`: internal split of the TRAINING rows via `cv_folds(k=round(1/holdout), seed, stratify)` taking fold 0 as the tuning holdout (reuses the existing deterministic splitter), **wrapped in a loud metis-voiced error** naming the constraint when stratification is illegal ("decide=offsets needs ≥ k rows of every class among the fold's training rows; got …" — review issue 4; the raw sklearn error is opaque, and small leaves are exactly where the sampling dial pushes). Doc note: `round(1/holdout)` quantizes the effective holdout (0.3 → k=3 → 1/3). Aux `train()` on the rest; `tune_class_offsets` on the holdout's proba; main `train()` on all training rows (exactly today's model); score assessment via `apply_offsets`. **Return decision (review issue 3): `fold_fit` ALWAYS returns `(score, model, offsets)`** with `offsets=None` under argmax — and Task 1's commit updates BOTH existing unpack sites in the same motion (`fold_score` at model.py:148, train.py per-fold at :69) so no intermediate commit is broken. `cv_score` threads `decide` through to `fold_score` (review issue 6 — a v1 plain experiment setting decide must NOT get an argmax cv_score beside an offsets ship; uniform threading, no silent divergence).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `with.decide` read + eager parse | `metis/steps/train.py` | modified | step contract |
| `offsets.json` persist (all-rows path) | `metis/steps/train.py` | modified | run-dir artifact |
| `probabilities.csv` + offsets apply | `metis/steps/predict.py` | modified | run-dir artifact |
| docs | train/predict docstrings, `atlas/experiment.md` (model.py + step bullets) | modified | operator surface |

- **train.py:** parse `decide` EAGERLY in `main()` (the #59 lesson: every path validates, incl. the foldless ship refit). Per-fold: thread `decide` into `fold_fit`; emitted `fold_score` is the after-decide score (the ledger metric NAME is unchanged — `train.fold_score` now measures the declared procedure). All-rows: fit + tune on all training rows (same aux-holdout mechanics), persist `model.pkl` + `offsets.json` (`{"offsets": [...], "classes": [...], "rule": ...}`); when `decide=argmax`, NO offsets.json (absence = argmax, backward-compatible with every existing run dir).
- **predict.py:** ALWAYS write `probabilities.csv` (id + one column per class: suffix = the ACTUAL class label from `model.classes_`, e.g. `proba_0`/`proba_1`/`proba_2` for int-coded targets — the suffix IS the class label, not a positional index; M2's cross-run averaging and offsets application both key on it — review issue 7) alongside `predictions.csv`. If the upstream train dir has `offsets.json`, **validate `offsets.json["classes"] == list(model.classes_)` (loud on mismatch — that's why classes is persisted)**, then apply for `predictions.csv`; else argmax (today's behavior byte-identical). `n_predictions` unchanged; add `has_offsets` 0/1.
- **Cache/compat:** `decide` absent → no re-key (existing cohorts untouched); `probabilities.csv` is additive output (existing consumers unaffected). kbench adoption (the M4 sweep: `decide {argmax, offsets} × cw {None, balanced}`) is a kbench issue, NOT here (ARCH-PURPOSE division as with #59).

## Tasks

### Task 1: pure core (TDD, `tests/test_model.py`)
- [ ] **The decide test frame (review issue 1 — the #59 constant-feature 12-row frame is BOTH illegal and vacuous here):** per-fold its training set is 6 rows/1 minority (internal StratifiedKFold(k≥4) → sklearn ValueError), and constant X gives identical probabilities on every row, so ANY offset yields an all-one-class prediction and balanced accuracy 0.5 regardless — assertions would pass without exercising tuning. Build a dedicated `_decide_frame()`: ~40 rows, 30/10 two-class skew, outer k=2 stratified (→ 15+5 training rows per fold; holdout 0.2 → internal k=5 → min class 5, legal), one WEAK-but-informative feature (e.g. minority rows drawn higher on x with overlap) so probabilities vary by row and tuned offsets genuinely flip specific boundary rows.
- [ ] Failing tests: (a) `tune_class_offsets` on a hand-built skewed proba matrix — 3-class **80/12/8** prior (−log-prior optima ≈ 1.90/2.30, comfortably inside the ±4 grid — review issue 2), probabilities = true posteriors; assert tuned score ≥ prior-corrected score − ε and > argmax score; (b) offsets deterministic across calls; grid includes no-op (uniform proba → zeros); (c) `apply_offsets` zero-offsets ≡ plain argmax incl. proba=0 columns (clip path); (d) `parse_decide` table (default, dict form, malformed loud, holdout range); (e) `fold_fit` with `decide=offsets` on `_decide_frame()`: returned offsets non-null AND assessment score ≥ argmax score − 0.1 on the same fold (the no-op grid point bounds the tuning slice; assessment noise gets the tolerance); ALSO the illegal-split loud error on a too-small frame (the wrapped constraint message); (f) the main model under decide=offsets is fitted on ALL training rows (prediction-equality with the argmax model — same seed, same rows → same model).
- [ ] Implement; PASS; commit `#60 M1: decision core — proba, offset tuning, parse_decide`.

### Task 2: step wiring (TDD, `tests/test_steps.py`)
- [ ] Failing tests (in-process `_run_step`, a step-level `_decide_frame()` dataset saved via the #59 captured-artifact pattern — NOT the 12-row frame, per review issue 1): (a) per-fold train with `decide: {offsets: {holdout: 0.2}}` on the 40-row frame emits a fold_score ≥ the argmax run's − tolerance, metrics carry `fold_score`; (b) foldless ship refit persists `offsets.json` alongside `model.pkl` (and does NOT with argmax); (c) malformed `decide` refuses eagerly on the foldless path; (d) predict: probabilities.csv columns/rows correct, offsets applied when present (hand-check one row where offsets flip the argmax), byte-identical `predictions.csv` when no offsets.json (the compat anchor).
- [ ] Implement train.py + predict.py; full `uv run pytest -q` + `go test ./cmd/metis` green; commit `#60 M1: train/predict wire decide + probabilities`.

### Task 3: docs + boundary
- [ ] `atlas/experiment.md`: model.py bullet (decision layer: leaf-local tuning, why holdout-not-in-sample, 2-fits cost, **+ the two honest costs (review issue 8): (i) SE inflation — the procedure's variance now includes tuning variance, so decide=offsets configs carry wider SEs and the 1-SE band widens (legitimate: the estimate measures the whole procedure); (ii) the aux/main mismatch — offsets are tuned against the 80%-fit aux model's probabilities and applied to the 100%-fit main model's (no leakage; standard CV-style pessimism, assumed not measured)**) + train/predict step bullets (decide knob, offsets.json + classes validation, probabilities.csv labeling, re-key semantics). train/predict docstring `with:` tables. Commit.
- [ ] Issue tick/Log; `sdlc milestone-close --issue 60 --milestone M1` (measured actual; findings per protocol). M2 (blend) plans after this closes.

## Execution notes
- **Before Task 1:** populate issue Done-when + Plan (M1/M2 rows) + ```estimate block, **AND add the issue `## Revisions` entry recording the leaf-local pivot** (review issue 5 / artifacts-lie-by-aspiration: the Spec's `metis/decide` step-type, full-OOF "no data sacrificed" tuning, and per-fold probability emission are superseded — record what changed and why, incl. the accepted consequence that blend has no in-sweep OOF material and will be leaderboard-measured only). Then `sdlc change-code`.
- The 2-fits-per-leaf cost note goes in the train docstring AND the atlas (an `out1in2` iteration run prices it before any decision run).
- sklearn facts already verified this session: `predict_proba` on rf + hist_gbm incl. class_weight + NaN inputs, sklearn 1.9.0.
