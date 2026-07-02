"""cv_folds — deterministic k-fold cross-validation assignment (metis#1 M3).

Pure: given a DataFrame, k, and a seed, return a positional fold assignment (one
fold index per row, aligned to row order). Stratified on a column when given
(preserves class balance across folds), else plain k-fold. Deterministic under
the seed — the reproducibility guarantee the project's done-when rests on. sklearn
is called, but the function is IO-free and deterministic, so it is unit-tested
directly on in-memory frames (ARCH-PURE).
"""

from __future__ import annotations

import numpy as np
import pandas as pd
from sklearn.model_selection import KFold, StratifiedKFold


def cv_folds(df: pd.DataFrame, k: int, seed: int, stratify_col: str | None = None) -> list[int]:
    """Assign each row of df to one of k validation folds.

    Returns a length-len(df) list of fold indices (0..k-1), positional. Every row
    lands in exactly one fold (disjoint + covering). With stratify_col, folds
    preserve that column's class distribution (StratifiedKFold); otherwise KFold.
    """
    n = len(df)
    if k < 2:
        raise ValueError(f"k must be >= 2, got {k}")
    if k > n:
        raise ValueError(f"k ({k}) cannot exceed number of rows ({n})")

    folds = np.empty(n, dtype=int)
    placeholder_X = np.zeros(n)
    if stratify_col is not None:
        splitter = StratifiedKFold(n_splits=k, shuffle=True, random_state=seed)
        iterator = splitter.split(placeholder_X, df[stratify_col].to_numpy())
    else:
        splitter = KFold(n_splits=k, shuffle=True, random_state=seed)
        iterator = splitter.split(placeholder_X)

    for fold_idx, (_, val_idx) in enumerate(iterator):
        folds[val_idx] = fold_idx
    return folds.tolist()
