# Leakage-Safe Target Features (internal cross-fit) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the workbench a reusable, leakage-safe way to build a *target-based* feature — one whose value derives from other rows' labels (e.g. group-survival) — so a row's own label never leaks into its own feature, proven by a regression test.

**Architecture:** A pure metis primitive `cross_fit_target_encode` (internal K-fold cross-fit + shrinkage; reuses `metis.split.cv_folds`) does the leakage-safe math, competition-agnostic. The kbench Titanic `features` step grows a *target-group* protocol alongside its existing stateless groups: `apply_features` threads `seed` and branches to a `target_encode_group` adapter that calls the primitive. The `data│pipeline` phase cut already makes features per-fold (cross-*fold* safety is free via `fit_mask`); this plan adds only the *within*-fold self-leak fix (the step owns it — no engine marker, per the metis-v2 pensive).

**Tech Stack:** Python 3.12, numpy, pandas, scikit-learn (`StratifiedKFold` via `cv_folds`), pytest. Two repos: `metis` (substrate, the primitive) and `kbench` (Titanic step, editable-depends on metis).

**Scope boundary (vs kbench#8 / M5):** This issue ships the *machinery* + a rigorous leak test + one demonstrator target group keyed on an existing column (`pclass_survival`), NOT wired into the Titanic sweep shape (so the thread stays green by construction). kbench#8 registers the high-value `ticket`-group survival feature on top, wires it into the sweep, and does the Kaggle-gated honest run.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `cross_fit_target_encode` | `metis/metis/encode.py` | new |
| `target_encode_group` | `kbench/kbench/titanic/features.py` | new |
| `apply_features` | `kbench/kbench/titanic/features.py` | modified |

- **`cross_fit_target_encode(groups, y, *, fit_mask, strategy, n_folds, m, loo_noise, seed) -> np.ndarray`** — the reusable leakage-safe target-mean encoder. One float per row: fit rows get an internal cross-fit encoding (own label never used); non-fit rows get the full-fit shrunk group mean over fit rows (prior when the group is unseen). `strategy ∈ {"kfold" (default), "loo"}`.
  - **Relationships:** N/A (a stateless pure function). Depends on `metis.split.cv_folds` for the internal folds (ARCH-DRY — no second fold generator).
  - **DRY rationale:** First occurrence of "target encoding" in the codebase, and the *reusable* unit across competitions — the whole reason it lives in metis, not kbench (operator: "continue to use this setup for other kaggle competitions"). kbench#8's `ticket` group and any future competition's target features all call this one function.
  - **Future extensions:** more strategies (e.g. empirical-Bayes `m="auto"`, hierarchical/mixed-model shrinkage à la tidymodels `step_lencode_mixed`); a `prior=` override; multi-column group keys. The `strategy` dispatch is the widening axis (built now, per operator's "do it right from the start").

- **`target_encode_group(work_train, work_test, *, key, target, analysis_mask, seed, strategy, n_folds, m, loo_noise) -> (work_train, work_test, [col])`** — the kbench adapter that turns the primitive into a feature *group*. Emits one column `<key>_TgtEnc`. Concatenates train+test group keys into one array, marks only analysis rows in the fit-mask (assessment + all test = non-fit → full-fit), calls the primitive once, splits the result back.
  - **Relationships:** 1 adapter → N registered instances (one per group key). `pclass_survival = partial(target_encode_group, key="Pclass")` here; kbench#8 adds `partial(..., key="Ticket")`.
  - **DRY rationale:** Without the adapter, every target group would re-implement the concat/mask/split glue around the primitive. First occurrence of a pattern kbench#8 reuses verbatim.
  - **Future extensions:** the `key` partial is the instance axis; a group could emit companion columns (group size) — kbench#8's `ticket` adds that.

- **`apply_features(...)` (modified)** — gains a `seed: int = 0` parameter and a second loop over `TARGET_CANONICAL_ORDER`/`TARGET_GROUPS` after the existing stateless-group loop. The 6 stateless groups' call sites are **byte-identical** (Done-when: "existing features unchanged").
  - **Relationships:** owns both `GROUPS` (stateless) and `TARGET_GROUPS` (target) registries; unchanged for stateless names.
  - **DRY rationale:** one enrichment entry point for both feature kinds; the branch keys off which registry a name is in (no per-step `fit_scope` marker — dropped as error-prone per pensive).
  - **Future extensions:** target groups run *after* stateless ones, so a future target group could key off a derived column.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `features.main()` | `kbench/kbench/titanic/features.py` | modified | env/IO (step contract) |

- **`features.main()` (modified)** — the step entrypoint. One-line change: pass `seed=ctx.seed` into `apply_features` (determinism — the internal cross-fit folds and any LOO noise must be seed-reproducible; [[feedback_controllable_time]] applies to seeds).
  - **Injected into:** `apply_features` (which is otherwise pure and unit-tested without env).
  - **Future extensions:** target-group knobs (`m`, `strategy`, `n_folds`) could surface in `with.json` when kbench#8 wires the sweep; today they take the primitive's defaults.

**Test surface:** `cross_fit_target_encode` is PURE → unit-tested directly on in-memory arrays in `metis/tests/test_encode.py` (no IO, no mocks). `target_encode_group` + `apply_features` are pure → tested in-memory in `kbench/kbench/titanic/features_test.py`, plus one step-level e2e test through the real `_run_features_step` harness (env + `with.json` + upstream artifacts → `main()` → loaded Dataset) — the "verify through the real chain, not hope" discipline. No external service → no process-level fake needed.

---

## Chunk 1: the metis primitive

### Task 1: `cross_fit_target_encode` (metis)

**Files:**
- Create: `metis/metis/encode.py`
- Test: `metis/tests/test_encode.py`

- [ ] **Step 1: Write the failing no-self-leak test**

`metis/tests/test_encode.py`:
```python
"""Pure tests for metis.encode.cross_fit_target_encode (no IO)."""
import numpy as np
import pandas as pd
import pytest

from metis.encode import cross_fit_target_encode


def _naive_incl_self(groups, y):
    """The LEAKY baseline: group mean INCLUDING the row itself (what #20 must beat)."""
    s = pd.Series(y, dtype=float)
    return s.groupby(np.asarray(groups)).transform("mean").to_numpy()


def _corr(a, b):
    return float(np.corrcoef(a, b)[0, 1])


def test_kfold_no_self_leak_on_random_data():
    """y independent of group ⇒ NO real signal. Naive (incl-self) encoding correlates
    with the row's own label (leak); cross-fit does not."""
    rng = np.random.default_rng(0)
    groups = np.repeat(np.arange(100), 2)          # 100 groups of size 2
    rng.shuffle(groups)
    y = rng.integers(0, 2, size=len(groups))
    naive = _naive_incl_self(groups, y)
    enc = cross_fit_target_encode(groups, y, strategy="kfold", n_folds=5, m=10.0, seed=0)
    assert abs(_corr(naive, y)) > 0.4              # the leak the naive path introduces
    assert abs(_corr(enc, y)) < 0.15               # cross-fit removes it
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/xianxu/workspace/metis && uv run --project . pytest tests/test_encode.py -q`
Expected: FAIL — `ModuleNotFoundError: No module named 'metis.encode'`.

- [ ] **Step 3: Write the primitive**

`metis/metis/encode.py`:
```python
"""cross_fit_target_encode — leakage-safe target (mean) encoding (metis#20).

A target-based feature encodes a categorical group by the mean of the target over that
group. Computed naively (group mean including the row itself) it leaks the row's own
label into its own feature — catastrophic for small groups (a size-1 group's feature IS
its label). This module provides the leakage-safe version feature steps call (the step
owns leakage-safety; the engine has no marker — see the metis-v2 pensive).

Two strategies, both shrinking small groups toward the global prior (m-estimate, the
step_lencode_mixed idea):
  - "kfold" (default): internal K-fold cross-fit. Fit rows split into K folds; each fold's
    rows are encoded from the OTHER folds' group means (out-of-fold), so a row's own label
    never enters its own encoding VIA THE GROUP AGGREGATE. (A negligible O(1/N) residual
    persists through the global shrinkage prior `y.mean()`, exactly as sklearn TargetEncoder —
    accepted, not a leak.) sklearn TargetEncoder's model, and the robust default.
  - "loo": leave-one-out — each row encoded from its group's OTHER members, with optional
    seeded additive noise (the classic defense against LOO's residual invertibility for
    small groups). Kept for reuse/comparison; kfold is safer in small-group regimes.

Pure + deterministic under `seed` (reuses metis.split.cv_folds for the internal folds).
Unit-tested directly on in-memory arrays (ARCH-PURE).
"""
from __future__ import annotations

import numpy as np
import pandas as pd

from metis.split import cv_folds


def _shrunk(sum_by: dict, cnt_by: dict, prior: float, m: float) -> dict:
    """m-estimate shrinkage per group: (sum + m·prior)/(count + m)."""
    return {g: (sum_by[g] + m * prior) / (cnt_by[g] + m) for g in cnt_by}


def _group_stats(y: np.ndarray, groups: np.ndarray):
    s = pd.Series(y, dtype=float).groupby(groups)
    return s.sum().to_dict(), s.size().to_dict()


def cross_fit_target_encode(
    groups,
    y,
    *,
    fit_mask=None,
    strategy: str = "kfold",
    n_folds: int = 5,
    m: float = 10.0,
    loo_noise: float = 0.0,
    seed: int = 0,
) -> np.ndarray:
    """Leakage-safe target-mean encoding of `groups`, one float per row.

    groups   : array of the categorical group key, all rows (fit + non-fit).
    y        : target array; used ONLY on fit rows (non-fit entries may be NaN).
    fit_mask : bool array; True = fit row (trusted label). None ⇒ all rows are fit.
    Returns enc where fit rows get a cross-fit encoding (own label never used) and
    non-fit rows get the full-fit shrunk group mean over fit rows (prior if unseen).
    """
    if strategy not in ("kfold", "loo"):
        raise ValueError(f"unknown strategy {strategy!r}; known: kfold, loo")
    groups = np.asarray(groups)
    y = np.asarray(y, dtype=float)
    n = len(groups)
    fit_mask = np.ones(n, bool) if fit_mask is None else np.asarray(fit_mask, bool)

    fit_idx = np.flatnonzero(fit_mask)
    gf, yf = groups[fit_idx], y[fit_idx]
    prior = float(yf.mean()) if len(yf) else 0.0

    full_sum, full_cnt = _group_stats(yf, gf)          # over ALL fit rows
    full_enc = _shrunk(full_sum, full_cnt, prior, m)

    enc = np.full(n, prior, dtype=float)
    for i in np.flatnonzero(~fit_mask):                # non-fit rows: full-fit lookup
        enc[i] = full_enc.get(groups[i], prior)

    if strategy == "kfold":
        k = min(n_folds, len(fit_idx))
        if k < 2:
            enc[fit_idx] = prior                        # too few to cross-fit → prior (no leak)
        else:
            classes, counts = np.unique(yf, return_counts=True)
            strat = "_y" if (len(classes) > 1 and counts.min() >= k) else None
            inner = np.asarray(cv_folds(pd.DataFrame({"_y": yf}), k=k, seed=seed,
                                        stratify_col=strat))
            for f in range(k):
                out = inner == f                        # rows to encode this pass
                pool = ~out                             # complement: everyone else
                s, c = _group_stats(yf[pool], gf[pool])
                oof = _shrunk(s, c, prior, m)
                for j in np.flatnonzero(out):
                    enc[fit_idx[j]] = oof.get(gf[j], prior)
    else:  # loo
        for j, gi in enumerate(gf):
            n_g = full_cnt.get(gi, 0)
            if n_g <= 1:
                enc[fit_idx[j]] = prior
            else:                                       # leave j out, then shrink
                enc[fit_idx[j]] = (full_sum[gi] - yf[j] + m * prior) / ((n_g - 1) + m)
        if loo_noise > 0:
            rng = np.random.default_rng(seed)
            enc[fit_idx] += rng.normal(0.0, loo_noise, size=len(fit_idx))

    return enc
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/xianxu/workspace/metis && uv run --project . pytest tests/test_encode.py -q`
Expected: PASS.

- [ ] **Step 5: Add the real-signal-preserved counter-test** (guards against a "return prior/constant" cheat passing Step 1)

Append to `test_encode.py`:
```python
def test_kfold_preserves_real_between_group_signal():
    """When y genuinely tracks the group, cross-fit must RECOVER it (not null everything)."""
    rng = np.random.default_rng(1)
    rates = {0: 0.1, 1: 0.4, 2: 0.6, 3: 0.9}
    groups = np.repeat(np.arange(4), 50)               # 4 groups of size 50 (large ⇒ stable OOF)
    y = np.array([rng.random() < rates[g] for g in groups], dtype=float)
    enc = cross_fit_target_encode(groups, y, strategy="kfold", n_folds=5, m=10.0, seed=0)
    assert _corr(enc, y) > 0.3                          # recovers the legitimate signal
    assert enc.std() > 0.1                              # NOT a constant (kills the return-prior cheat)
    for g, p in rates.items():                          # each group's enc ≈ its true rate
        assert abs(enc[groups == g].mean() - p) < 0.15
```

- [ ] **Step 6: Add fit_mask / non-fit / unseen-group / determinism / ship-path / loo tests**

Append to `test_encode.py`:
```python
def test_non_fit_rows_get_full_fit_and_unseen_gets_prior():
    groups = np.array(["a", "a", "a", "b", "b", "zzz"])
    y      = np.array([1.0, 1.0, 0.0, 0.0, 0.0, np.nan])   # last row = non-fit, unseen group
    fit    = np.array([True, True, True, True, True, False])
    enc = cross_fit_target_encode(groups, y, fit_mask=fit, m=0.0, seed=0)   # m=0 ⇒ raw means
    # non-fit 'zzz' never seen among fit rows ⇒ prior = mean(fit y) = 2/5 = 0.4
    assert enc[5] == pytest.approx(0.4)

def test_non_fit_known_group_gets_full_fit_mean():
    groups = np.array(["a", "a", "a", "a"])
    y      = np.array([1.0, 1.0, 0.0, np.nan])
    fit    = np.array([True, True, True, False])
    enc = cross_fit_target_encode(groups, y, fit_mask=fit, m=0.0, seed=0)
    assert enc[3] == pytest.approx(2 / 3)              # full-fit mean of group 'a' over fit rows

def test_deterministic_under_seed():
    rng = np.random.default_rng(2)
    groups = rng.integers(0, 30, size=120)
    y = rng.integers(0, 2, size=120)
    a = cross_fit_target_encode(groups, y, seed=7)
    b = cross_fit_target_encode(groups, y, seed=7)
    assert np.array_equal(a, b)

def test_ship_path_all_fit_no_crash():
    rng = np.random.default_rng(3)
    groups = rng.integers(0, 20, size=80)
    y = rng.integers(0, 2, size=80)
    enc = cross_fit_target_encode(groups, y, fit_mask=None, seed=0)   # all rows fit
    assert enc.shape == (80,) and np.all(np.isfinite(enc))

def test_loo_within_group_structure_is_label_invertible():
    """LOO's residual leak — the reason kfold is the default. Within a REALIZED group, raw-LOO
    encoding enc_i = (S - y_i)/(n-1) is a deterministic function of (group, own label): all
    survivors collapse to one value, all non-survivors to another, separated by exactly 1/(n-1).
    A flexible model that isolates the group can invert this to recover the label. Seeded noise
    (loo_noise>0) blurs it; kfold has no such deterministic per-label structure (Step 1 proves
    kfold doesn't leak). This is NOT visible in marginal corr(enc, y) — it is a within-group
    property — so we assert the structure directly, not a fragile correlation inequality."""
    groups = np.array(["g", "g", "g", "g"])        # one size-4 group, labels [1,1,0,0], S=2
    y = np.array([1.0, 1.0, 0.0, 0.0])
    enc = cross_fit_target_encode(groups, y, strategy="loo", loo_noise=0.0, m=0.0, seed=0)
    assert enc[0] == pytest.approx(1 / 3) and enc[1] == pytest.approx(1 / 3)   # survivors collapse
    assert enc[2] == pytest.approx(2 / 3) and enc[3] == pytest.approx(2 / 3)   # non-survivors collapse
    assert enc[2] - enc[0] == pytest.approx(1 / 3)                             # the 1/(n-1) gap

def test_loo_deterministic_and_finite_with_noise():
    """LOO with seeded additive noise: finite + reproducible under seed (the safe LOO form)."""
    rng = np.random.default_rng(4)
    groups = rng.integers(0, 30, size=120)
    y = rng.integers(0, 2, size=120)
    a = cross_fit_target_encode(groups, y, strategy="loo", loo_noise=0.1, m=1.0, seed=5)
    b = cross_fit_target_encode(groups, y, strategy="loo", loo_noise=0.1, m=1.0, seed=5)
    assert np.all(np.isfinite(a)) and np.array_equal(a, b)

def test_unknown_strategy_rejected():
    with pytest.raises(ValueError, match="unknown strategy"):
        cross_fit_target_encode(np.array([1, 1]), np.array([0.0, 1.0]), strategy="bogus")
```

- [ ] **Step 7: Run the full encode suite**

Run: `cd /Users/xianxu/workspace/metis && uv run --project . pytest tests/test_encode.py -q`
Expected: PASS (all). All assertions are deterministic (the leak/invertibility tests use fixed
constructions, not seed-sensitive correlation inequalities). Never weaken the Step-1 no-leak
assertion — it is the load-bearing proof of the Done-when.

- [ ] **Step 8: Run the WHOLE metis suite (nothing regressed)**

Run: `cd /Users/xianxu/workspace/metis && uv run --project . pytest -q`
Expected: PASS (existing + new).

- [ ] **Step 9: Commit**

```bash
cd /Users/xianxu/workspace/metis
git add metis/encode.py tests/test_encode.py
git commit -m "#20: leakage-safe target encoder (internal cross-fit + shrinkage)

Pure metis primitive cross_fit_target_encode: kfold (default) / loo strategies,
m-estimate shrinkage, reuses metis.split.cv_folds for the internal folds. Fit rows
get an out-of-fold encoding (own label never used); non-fit rows get the full-fit
shrunk group mean. Leak regression test: naive incl-self correlates with own label
on no-signal data, cross-fit does not; real between-group signal still recovered.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Chunk 2: the kbench target-group protocol

### Task 2: `apply_features` extension + `target_encode_group` adapter + demonstrator (kbench)

**Files:**
- Modify: `kbench/kbench/titanic/features.py`
- Test: `kbench/kbench/titanic/features_test.py`

- [ ] **Step 1: Write the failing adapter test** (pure, in-memory)

Append to `kbench/kbench/titanic/features_test.py`:
```python
def test_target_encode_group_oof_train_fullfit_holdout():
    """The adapter: analysis rows get OOF cross-fit; assessment + test rows get the
    full-analysis shrunk mean; emits one <key>_TgtEnc column."""
    from kbench.titanic.features import target_encode_group
    # 6 train rows in 2 ticket-like groups; rows 0-3 analysis, 4-5 assessment.
    work_train = pd.DataFrame({
        "Ticket":   ["A", "A", "B", "B", "A", "B"],
        "Survived": [1, 1, 0, 0, 1, 0],
    })
    work_test = pd.DataFrame({"Ticket": ["A", "B", "ZZZ"]})   # ZZZ unseen ⇒ prior
    analysis_mask = np.array([True, True, True, True, False, False])
    wtr, wte, cols = target_encode_group(
        work_train, work_test, key="Ticket", target="Survived",
        analysis_mask=analysis_mask, seed=0, m=0.0)
    assert cols == ["Ticket_TgtEnc"]
    assert "Ticket_TgtEnc" in wtr.columns and "Ticket_TgtEnc" in wte.columns
    prior = 2 / 4                                        # analysis y = [1,1,0,0]
    assert wte.loc[wte["Ticket"] == "ZZZ", "Ticket_TgtEnc"].iloc[0] == pytest.approx(prior)
    # assessment row 4 (group A) gets the full-analysis mean of A over analysis rows (0,1) = 1.0
    assert wtr.iloc[4]["Ticket_TgtEnc"] == pytest.approx(1.0)
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /Users/xianxu/workspace/kbench && uv run --project . pytest kbench/titanic/features_test.py::test_target_encode_group_oof_train_fullfit_holdout -q`
Expected: FAIL — `ImportError: cannot import name 'target_encode_group'`.

- [ ] **Step 3: Add the adapter + registry + apply_features branch**

In `kbench/kbench/titanic/features.py`, add near the top imports:
```python
from functools import partial
from metis.encode import cross_fit_target_encode
```

Add the adapter (after the stateless groups, before `apply_features`):
```python
# ── target-based feature groups (metis#20) ───────────────────────────────────
# A DIFFERENT protocol from the stateless GROUPS: a target group reads the label, so it
# must cross-fit internally (own label never in own feature). It needs the analysis mask
# (to split OOF-train from full-fit-holdout) + the seed — which the stateless (train,test)
# signature can't carry. So target groups get their own signature + registry; the 6
# stateless groups are untouched. The heavy lifting is metis.encode.cross_fit_target_encode.

def target_encode_group(work_train, work_test, *, key, target, analysis_mask, seed,
                        strategy="kfold", n_folds=5, m=10.0, loo_noise=0.0):
    """Leakage-safe target-encoding feature group → one column `<key>_TgtEnc`.

    Concatenate train+test group keys, mark ONLY analysis rows as fit (assessment + all
    test = non-fit), call the primitive once, split the encoding back. Analysis rows get
    the out-of-fold cross-fit encoding; assessment + test rows get the full-analysis
    shrunk mean (prior for a group unseen among analysis rows)."""
    col = f"{key}_TgtEnc"
    keys = np.concatenate([work_train[key].to_numpy(), work_test[key].to_numpy()])
    ys = np.concatenate([work_train[target].to_numpy(dtype=float),
                         np.full(len(work_test), np.nan)])
    mask = np.concatenate([np.asarray(analysis_mask, bool),
                           np.zeros(len(work_test), bool)])
    enc = cross_fit_target_encode(keys, ys, fit_mask=mask, strategy=strategy,
                                  n_folds=n_folds, m=m, loo_noise=loo_noise, seed=seed)
    work_train, work_test = work_train.copy(), work_test.copy()
    work_train[col] = enc[:len(work_train)]
    work_test[col] = enc[len(work_train):]
    return work_train, work_test, [col]


# Target-group registry (name → callable). A demonstrator keyed on an existing column;
# kbench#8 registers `ticket` (Ticket-group survival) on top. Kept out of the swept shape.
TARGET_GROUPS: dict = {
    "pclass_survival": partial(target_encode_group, key="Pclass"),
}
TARGET_CANONICAL_ORDER = ["pclass_survival"]
```

Modify `apply_features`'s signature and add the target loop. Change the signature line:
```python
def apply_features(base: Dataset, raw_train: pd.DataFrame, raw_test: pd.DataFrame,
                   features: set[str], fit_mask=None, seed: int = 0) -> Dataset:
```
Change the unknown-name guard to accept both registries:
```python
    known = set(GROUPS) | set(TARGET_GROUPS)
    unknown = set(features) - known
    if unknown:
        raise ValueError(f"unknown feature group(s): {sorted(unknown)}; "
                         f"known: {CANONICAL_ORDER + TARGET_CANONICAL_ORDER}")
```
After the existing `for name in CANONICAL_ORDER:` stateless loop (which stays byte-identical), add:
```python
    # Target groups (metis#20): read the label ⇒ cross-fit internally. Run AFTER the
    # stateless groups. analysis_mask = the analysis rows over work_train (all-True off-fold).
    analysis_mask = np.ones(len(work_train), bool) if fit_mask is None else np.asarray(fit_mask, bool)
    for name in TARGET_CANONICAL_ORDER:
        if name in features:
            work_train, work_test, added = TARGET_GROUPS[name](
                work_train, work_test, target=TARGET, analysis_mask=analysis_mask, seed=seed)
            new_cols.extend(added)
```
(`feat_cols = BASE_FEATURES + new_cols` downstream already picks up the new column.)

- [ ] **Step 4: Thread the seed through `main()`**

In `features.py::main()`, change the `apply_features` call:
```python
    ds = apply_features(base, raw_train, raw_test, features, fit_mask, seed=ctx.seed)
```

- [ ] **Step 5: Run the adapter test — verify it passes**

Run: `cd /Users/xianxu/workspace/kbench && uv run --project . pytest kbench/titanic/features_test.py::test_target_encode_group_oof_train_fullfit_holdout -q`
Expected: PASS.

- [ ] **Step 6: Add a step-level e2e test through the real step contract**

Append to `features_test.py`:
```python
def test_pclass_survival_flows_through_the_step(tmp_path, monkeypatch):
    """The target group flows through the real features step (env + with.json → main()
    → loaded Dataset): the encoded column appears with finite values, id/target intact."""
    base = _base_dataset()
    with_cfg = {"dataset": "adapt", "raw": "get-data", "out": "features",
                "features": ["pclass_survival"]}
    ds = _run_features_step(tmp_path, monkeypatch, base, with_cfg)
    assert "Pclass_TgtEnc" in ds.schema.feature_cols()
    assert ds.train["Pclass_TgtEnc"].notna().all()
    assert len(ds.train) == 4 and len(ds.test) == 2       # rows preserved
    assert "Survived" not in ds.test.columns              # target never emitted on test
    assert "Pclass_TgtEnc" in ds.test.columns             # the feature IS emitted on test (full-fit)
```

- [ ] **Step 7: Add the "existing features unchanged" regression assertion**

Append to `features_test.py`:
```python
def test_stateless_features_unchanged_by_target_addition():
    """Adding the target-group machinery must not perturb the 6 stateless groups' output
    (Done-when: existing features unchanged)."""
    from kbench.titanic.features import apply_features, BASE_FEATURES
    base = _base_dataset()
    ds = apply_features(base, _raw_train(), _raw_test(),
                        {"title", "family", "age", "fare", "deck", "embarked"}, seed=42)
    # the stateless feature set is exactly the pre-#20 columns (no Pclass_TgtEnc leaked in)
    assert "Pclass_TgtEnc" not in ds.schema.feature_cols()
    expected = BASE_FEATURES + ["Title", "FamilySize", "IsAlone", "Age", "AgeBin",
                                "FarePerPerson", "Deck", "HasCabin",
                                "Embarked_C", "Embarked_Q", "Embarked_S"]
    assert ds.schema.feature_cols() == expected


def test_target_registry_matches_canonical_order():
    """Drift guard mirroring test_apply_features_registry_matches_canonical_order, for the
    TARGET_* registry — keeps names + order in sync as kbench#8 adds `ticket`."""
    from kbench.titanic.features import TARGET_GROUPS, TARGET_CANONICAL_ORDER
    assert set(TARGET_GROUPS) == set(TARGET_CANONICAL_ORDER)
```
(`apply_features`, `BASE_FEATURES`, and the `TARGET_*` names are imported inside each test — the
existing tests already import `apply_features` inline, so match that convention.)

- [ ] **Step 8: Run the whole kbench features suite**

Run: `cd /Users/xianxu/workspace/kbench && uv run --project . pytest kbench/titanic/features_test.py -q`
Expected: PASS (existing + new). If the pre-existing registry-drift guard (`test_apply_features_registry_matches_canonical_order`) checks only `GROUPS`/`CANONICAL_ORDER`, confirm it still passes (target registry is separate); if it asserts totality over *all* features, extend it to include `TARGET_*`.

- [ ] **Step 9: Commit**

```bash
cd /Users/xianxu/workspace/kbench
git add kbench/titanic/features.py kbench/titanic/features_test.py
git commit -m "#20: target-group protocol + leakage-safe pclass_survival demonstrator

apply_features gains a seed param + a TARGET_GROUPS branch (stateless groups
byte-identical). target_encode_group adapter wraps metis.encode via concat/mask/split;
analysis rows get OOF cross-fit, assessment+test get full-fit. Demonstrator
pclass_survival registered (NOT wired into the sweep shape); kbench#8 adds ticket.
Step-level e2e proves the real chain.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Chunk 3: verification & close

### Task 3: prove the thread is green + update atlas

**Files:**
- Modify: `metis/atlas/experiment.md`, `metis/atlas/index.md` (if the entry is new)
- Modify: `metis/workshop/issues/000020-fold-aware-features.md` (Plan/Log)

- [ ] **Step 1: Full suites, both repos**

Run: `cd /Users/xianxu/workspace/metis && uv run --project . pytest -q`
Run: `cd /Users/xianxu/workspace/kbench && uv run --project . pytest -q`
Expected: PASS (both). Record counts in the issue Log.

- [ ] **Step 2: Confirm the existing Titanic sweep still enumerates unchanged** (thread green — the demonstrator is NOT in the shape)

Run (from the kbench workspace, where the shape lives; `metis run` supports `-dry-run` = list swept configs without running — main.go:41):
```bash
cd /Users/xianxu/workspace/kbench && uv run --project . metis run competition/titanic/pipelines/titanic-sweep.md -dry-run
```
Reference: `kbench/competition/titanic/pipelines/RUNBOOK-sweep.md`.
Expected: the config enumeration is unchanged from pre-#20 (the pre-#21 count was 42; post-#21 GBM it is 33 — use whichever the RUNBOOK/shape currently declares as the baseline). `pclass_survival` must **NOT** appear (it's registered but unwired). Record the observed count in the issue Log and confirm it matches the shape's declared config-count comment.

- [ ] **Step 3: Update the atlas** (per AGENTS.md §8 — new surface at the boundary)

In `metis/atlas/experiment.md`, add a short entry: the leakage-safe target-encode primitive (`metis/encode.py::cross_fit_target_encode`, kfold/loo + shrinkage, reuses `cv_folds`); note the kbench-side `TARGET_GROUPS` protocol (target groups own within-fold cross-fit; no engine marker). Keep it a pointer, not a spec. Ensure `atlas/index.md` links `experiment.md` (already linked — verify).

- [ ] **Step 4: Commit atlas + issue updates**

```bash
cd /Users/xianxu/workspace/metis
git add atlas/experiment.md atlas/index.md workshop/issues/000020-fold-aware-features.md
git commit -m "#20: atlas + issue log — leakage-safe target-encode primitive & protocol

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 5: Close** (single review boundary — one `sdlc close`; the binary auto-dispatches the mandatory fresh-eyes review over the branch point→HEAD window, #69)

Run (background — the review can exceed a 2-min inline timeout):
```bash
cd /Users/xianxu/workspace/metis && sdlc close --issue 20 \
  --verified 'metis+kbench pytest green (record counts); leak regression proves naive-incl-self correlates with own label on no-signal data while cross-fit does not, and real between-group signal is recovered; step-level e2e flows pclass_survival through the real features step; existing stateless features byte-identical; titanic sweep enumeration unchanged (demonstrator not wired into shape).'
```
Fix any Critical/Important the review raises before the boundary; log the `Review-Verdict:` outcome in `## Log`. `--actual` is measured (omit → close computes it, or run `sdlc actual --issue 20` first) — never hand-type hours.

---

## Notes for the executor

- **Two repos, two `uv` projects.** metis code + tests run under `--project /Users/xianxu/workspace/metis`; kbench under `--project /Users/xianxu/workspace/kbench`. kbench depends on metis editable, so the metis primitive is importable from kbench immediately after Task 1.
- **The leak proof lives in metis** (Task 1, synthetic small-group no-signal data) — that's where the assertion is crisp and controllable. The kbench tests prove *plumbing/protocol*, not the leak.
- **Done-when mapping (record in `## Log`):** the issue's Done-when says "the naive whole-train version inflates cv." The plan proves the sharper, more controllable *feature-level* property — `corr(naive-incl-self encoding, own label) ≈ 0.7` vs `corr(cross-fit, own label) ≈ 0` on no-signal data — which is the *cause* of cv inflation, isolated from model/CV noise. A feature that correlates with its own label is exactly what inflates a downstream cv; proving it at the encoding level is a superior operationalization. Note this mapping in the Log so the boundary review doesn't expect a literal cv-delta number.
- **Determinism is the contract** — every random path (internal folds, LOO noise) is seeded from `ctx.seed`. Do not introduce an unseeded RNG (breaks cache/reproduce guards metis#24/#28).
- **Do not wire the demonstrator into the Titanic sweep shape.** That (with `ticket`) is kbench#8's job; keeping it out is what makes "the Titanic thread still green" true by construction.
- **ARCH markers to cite in Log/commits where they shaped a call:** ARCH-DRY (reuse `cv_folds`; one primitive across competitions), ARCH-PURE (the primitive + adapter are pure, tested without IO), ARCH-PURPOSE (ship a *usable* capability + protocol, with kbench#8 as the tracked consumer — not a bare extension point).
