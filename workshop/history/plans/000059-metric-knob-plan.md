# Metric Knob + class_weight Passthrough Implementation Plan (metis#59)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox (`- [ ]`) syntax.

**Goal:** `metis/train` selects on the competition's objective: a `with.metric ∈ {accuracy, balanced_accuracy}` knob (default `accuracy` — zero re-keying for shapes that don't set it) threaded to the pure scorers, plus `class_weight` accepted by `make_model` for rf/hist_gbm.

**Architecture:** Scorer resolution lives in ONE place (`metis.model._SCORERS` + `resolve_scorer`, the `parse_model_config` loud-misuse pattern — `ARCH-DRY`/`ARCH-PURE`); `metric` threads as a pure keyword param (default `"accuracy"`) through `fold_fit`/`fold_score`/`cv_score`, so every existing caller is source-compatible. `train.py` reads `w.get("metric", "accuracy")` on BOTH paths (per-fold and all-rows refit). `class_weight` is just another `p.get(...)` in `make_model` — swept via the `$any`-map like any hyperparam. Cache semantics: a shape that SETS `metric` gets a new leaf address (new key in `with` → re-key, correct); absent key → addresses unchanged (titanic cohorts untouched).

**Tech Stack:** Python (sklearn `balanced_accuracy_score`, `class_weight="balanced"` on both estimators — HistGradientBoostingClassifier since sklearn 1.2); pytest (`tests/test_model.py`, `tests/test_steps.py`).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `_SCORERS` / `resolve_scorer` | `metis/model.py` | new |
| `fold_fit`/`fold_score`/`cv_score` (+`metric=` kwarg) | `metis/model.py` | modified |
| `make_model` (+`class_weight` passthrough) | `metis/model.py` | modified |

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `with.metric` read | `metis/steps/train.py` | modified | step contract |
| docs | `metis/steps/train.py` docstring, `atlas/experiment.md` | modified | operator surface |

- The shape's `objective.metric: train.fold_score` is a ledger NAME, not the scorer — untouched.
- kbench s6e7 shapes adopt the knob in kbench#12 M2, NOT here (`ARCH-PURPOSE`: that adoption commit also drops the METRIC GAP comment, per the M1 review).

### Task 1: pure core (TDD, `tests/test_model.py`)

- [ ] **Step 1 — failing tests** (match the file's existing style; `make_classification`/handwritten arrays):

```python
def test_balanced_accuracy_scorer_on_skewed_labels():
    # majority-argmax: accuracy looks great (0.9), balanced accuracy exposes it (0.5).
    y_true = np.array([0]*9 + [1])
    y_pred = np.zeros(10, dtype=int)
    from metis.model import resolve_scorer
    assert resolve_scorer("accuracy")(y_true, y_pred) == pytest.approx(0.9)
    assert resolve_scorer("balanced_accuracy")(y_true, y_pred) == pytest.approx(0.5)


def test_unknown_metric_rejected():
    from metis.model import resolve_scorer
    with pytest.raises(ValueError, match="balanced_accuracy"):  # message names the closed set
        resolve_scorer("auc")


def test_fold_score_metric_kwarg_changes_the_score():
    # a skewed frame where the majority class dominates folds: accuracy ≠ balanced accuracy
    # Build with cv_folds(k=2, stratify_col=<target>) on a 12-row 10/2-skewed frame —
    # STRATIFIED (review issue 4): unstratified k=2 can put both minority rows in one fold,
    # leaving the other single-class, where balanced accuracy degenerates to accuracy and
    # the "metrics differ" assertion goes flaky. 2 minority rows >= k=2, so StratifiedKFold
    # is legal. Assert: the two metrics differ, AND
    # cv_score(..., metric="balanced_accuracy") == mean of the per-fold
    # fold_score(..., metric="balanced_accuracy") values (pins the cv_score->fold_score
    # threading — review issue 3).
    ...


def test_make_model_class_weight_reaches_estimators():
    from metis.model import make_model
    for kind in ("rf", "hist_gbm"):
        m = make_model(kind, seed=0, params={"class_weight": "balanced"})
        assert m.class_weight == "balanced"
        assert make_model(kind, seed=0).class_weight is None  # default unchanged
```

- [ ] **Step 2** — `uv run pytest tests/test_model.py -q` → FAIL (no `resolve_scorer`).
- [ ] **Step 3 — implement in `metis/model.py`:** import `balanced_accuracy_score`; `_SCORERS = {"accuracy": accuracy_score, "balanced_accuracy": balanced_accuracy_score}`; `resolve_scorer(name)` → loud `ValueError` naming `sorted(_SCORERS)` on unknown. Add `metric: str = "accuracy"` kwarg to `fold_fit`/`fold_score`/`cv_score` (threaded `fold_score → fold_fit`; the one `accuracy_score(...)` call site in `fold_fit` becomes `resolve_scorer(metric)(...)`). `make_model`: `class_weight=p.get("class_weight")` on rf + hist_gbm constructors (logreg untouched). Update the module docstring's scorer sentence.
- [ ] **Step 4** — tests PASS; whole file green. **Step 5 — commit** `#59: metric knob + class_weight in the pure core`.

### Task 2: train step + docs

- [ ] **Step 1 — failing step-level tests FIRST** (`tests/test_steps.py`, in-process `_run_step` harness — it monkeypatches env and calls `main_fn()` directly, so `pytest.raises` works):
  - **The skewed dataset does NOT exist as a fixture** (review issue 1 — `testdata/dataset/toy` is 33/27 balanced): build it IN-TEST via `io.save_dataset` into the run dir and reference it as an upstream step-id — the exact pattern `test_ship_refit_reads_captured_features_no_folds` uses (tests/test_steps.py:69-75). ~12 rows, 10/2 skew.
  - Test A: `with.metric: balanced_accuracy` per-fold run emits a `fold_score` equal to the balanced computation and ≠ the accuracy run's.
  - Test B: unknown `with.metric` raises `ValueError` naming the closed set on the **foldless ship path** (no `folds` in `with`) — the path that would otherwise silently accept it (review issue 2).
- [ ] **Step 2 — implement:** `metis/steps/train.py` — read `metric = w.get("metric", "accuracy")` once in `main()` and **eagerly `resolve_scorer(metric)` right there** (review issue 2: validation fires on EVERY path — per-fold, flat-with-folds, and the foldless ship refit that never scores — and before any fit is wasted); pass `metric` to `fold_fit(...)` (per-fold) and `cv_score(...)` (all-rows-with-folds). Docstring `with:` table gains the `metric` row (optional, default accuracy, closed set, loud eager misuse).
- [ ] **Step 3:** `atlas/experiment.md` — **BOTH stale bullets** (review issue 6): the `model.py` bullet (~:126-131 — "cv_score averages per-fold validation *accuracy*" becomes metric-parameterized; the hyperparam list gains `class_weight`) AND the train step-entrypoint bullet (~:287 — the metric knob + the re-key note: setting the key re-keys, absent key leaves existing cohorts untouched).
- [ ] **Step 4** — `uv run pytest -q` (full python suite) green; `go test ./cmd/metis` untouched-green (no Go surface). **Step 5 — commit** `#59: train step reads with.metric eagerly; docs both atlas bullets`.

### Task 3: close

- [ ] Log + tick; `sdlc pr` → `sdlc merge` → `sdlc close --verified '<evidence>'` (single boundary; review runs at close).

## Execution notes

- Estimate derived in the issue's ```estimate block; set before `sdlc change-code`.
- Callers of the retyped signatures are all keyword-default-compatible — but grep `fold_score\|cv_score\|fold_fit` across `metis/` + `tests/` anyway (the #58 lesson: `go build` doesn't see tests; here, pytest compiles everything, but grep is cheap).
