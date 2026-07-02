"""Pure tests for metis.split.cv_folds — determinism, coverage, stratification."""

import numpy as np
import pandas as pd
import pytest

from metis.split import cv_folds


def _df(n=60, seed=0):
    rng = np.random.default_rng(seed)
    y = (rng.random(n) < 0.4).astype(int)  # ~40% positives
    return pd.DataFrame({"f": rng.random(n), "target": y})


def test_disjoint_and_covering():
    df = _df()
    folds = cv_folds(df, k=5, seed=42)
    assert len(folds) == len(df)               # one assignment per row
    assert set(folds) == {0, 1, 2, 3, 4}       # all k folds populated
    assert all(0 <= f < 5 for f in folds)      # in range → exactly one fold each


def test_deterministic_under_seed():
    df = _df()
    assert cv_folds(df, k=5, seed=42) == cv_folds(df, k=5, seed=42)


def test_different_seed_changes_assignment():
    df = _df()
    assert cv_folds(df, k=5, seed=1) != cv_folds(df, k=5, seed=2)


def test_stratified_preserves_class_balance():
    df = _df(n=100)
    overall = df["target"].mean()
    folds = np.array(cv_folds(df, k=5, seed=7, stratify_col="target"))
    for f in range(5):
        fold_rate = df["target"].to_numpy()[folds == f].mean()
        assert abs(fold_rate - overall) < 0.12  # balanced within a tight band


@pytest.mark.parametrize("k", [1, 0])
def test_k_too_small_rejected(k):
    with pytest.raises(ValueError, match="k must be >= 2"):
        cv_folds(_df(n=10), k=k, seed=0)


def test_k_exceeds_rows_rejected():
    with pytest.raises(ValueError, match="cannot exceed"):
        cv_folds(_df(n=3), k=5, seed=0)
