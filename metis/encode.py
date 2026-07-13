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
    Returns enc where fit rows get a cross-fit encoding (own label never enters via the
    group aggregate — a negligible O(1/N) residual persists through the global shrinkage
    prior, as in sklearn TargetEncoder) and non-fit rows get the full-fit shrunk group
    mean over fit rows (prior if unseen).
    """
    if strategy not in ("kfold", "loo"):
        raise ValueError(f"unknown strategy {strategy!r}; known: kfold, loo")
    groups = np.asarray(groups)
    y = np.asarray(y, dtype=float)
    n = len(groups)
    fit_mask = np.ones(n, bool) if fit_mask is None else np.asarray(fit_mask, bool)
    if len(y) != n or len(fit_mask) != n:
        raise ValueError(f"groups/y/fit_mask length mismatch: {n}, {len(y)}, {len(fit_mask)}")

    fit_idx = np.flatnonzero(fit_mask)
    gf, yf = groups[fit_idx], y[fit_idx]
    prior = float(yf.mean()) if len(yf) else 0.0

    full_sum, full_cnt = _group_stats(yf, gf)          # over ALL fit rows
    full_enc = _shrunk(full_sum, full_cnt, prior, m)

    enc = np.full(n, prior, dtype=float)
    for i in np.flatnonzero(~fit_mask):                # non-fit rows: full-fit lookup
        enc[i] = full_enc.get(groups[i], prior)

    if strategy == "kfold":
        if n_folds < 2:                                 # misconfiguration, not a data condition
            raise ValueError(f"n_folds must be >= 2, got {n_folds}")
        k = min(n_folds, len(fit_idx))
        if k < 2:
            enc[fit_idx] = prior                        # too few FIT ROWS to cross-fit → prior (no leak)
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
