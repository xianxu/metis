"""Pure tests for metis.model — train/predict/cv_score (no IO)."""

import numpy as np
import pytest
from sklearn.datasets import make_classification

from metis.model import cv_score, make_model, parse_model_config, predict, train
from metis.split import cv_folds


def _separable(n=120, seed=0):
    X, y = make_classification(
        n_samples=n, n_features=4, n_informative=3, n_redundant=0,
        n_clusters_per_class=1, class_sep=2.0, random_state=seed,
    )
    return X, y


@pytest.mark.parametrize("kind", ["logreg", "rf"])
def test_train_predict_shapes(kind):
    X, y = _separable()
    model = train(X, y, kind, seed=42)
    preds = predict(model, X)
    assert preds.shape == (len(y),)
    assert set(np.unique(preds)).issubset(set(np.unique(y)))


@pytest.mark.parametrize("kind", ["logreg", "rf"])
def test_deterministic(kind):
    X, y = _separable()
    p1 = predict(train(X, y, kind, seed=7), X)
    p2 = predict(train(X, y, kind, seed=7), X)
    assert np.array_equal(p1, p2)


def test_unknown_model_rejected():
    with pytest.raises(ValueError, match="unknown model"):
        make_model("svm", seed=0)


def test_make_model_applies_hyperparams():
    """The swept hyperparams must reach the estimator — not be silently dropped (metis#12)."""
    lr = make_model("logreg", seed=0, params={"C": 0.001})
    assert lr.C == 0.001
    rf = make_model("rf", seed=0, params={"n_estimators": 7, "max_depth": 2})
    assert rf.n_estimators == 7
    assert rf.max_depth == 2
    # Defaults hold when a param is absent / params is None (backward-compat).
    assert make_model("logreg", seed=0).C == 1.0
    assert make_model("rf", seed=0, params={"max_depth": 3}).n_estimators == 100


def test_hyperparams_change_the_fit():
    """Two materially-different hyperparam settings must produce different CV scores on the
    same data — proving the sweep isn't a sham (the params actually reach the fit)."""
    import pandas as pd

    X, y = _separable(n=120, seed=2)
    df = pd.DataFrame(X)
    df["target"] = y
    folds = cv_folds(df, k=5, seed=3, stratify_col="target")
    # An rf stump (max_depth=1, few trees) vs a deep forest → different accuracy.
    weak = cv_score(X, y, folds, "rf", seed=3, params={"n_estimators": 1, "max_depth": 1})
    strong = cv_score(X, y, folds, "rf", seed=3, params={"n_estimators": 300, "max_depth": None})
    assert weak != strong


def test_parse_model_config():
    """with['model'] normalizes to (kind, params): a bare string OR the $any-map single-key bundle."""
    assert parse_model_config("logreg") == ("logreg", {})
    assert parse_model_config({"rf": {"n_estimators": 200, "max_depth": 4}}) == (
        "rf", {"n_estimators": 200, "max_depth": 4})
    assert parse_model_config({"logreg": {}}) == ("logreg", {})
    assert parse_model_config({"logreg": None}) == ("logreg", {})
    for bad in ({"a": 1, "b": 2}, 123, {}, None, ["logreg"]):
        with pytest.raises(ValueError):
            parse_model_config(bad)


def test_cv_score_reasonable_and_deterministic():
    import pandas as pd

    X, y = _separable(n=120, seed=1)
    df = pd.DataFrame(X)
    df["target"] = y
    folds = cv_folds(df, k=5, seed=3, stratify_col="target")
    s1 = cv_score(X, y, folds, "logreg", seed=3)
    s2 = cv_score(X, y, folds, "logreg", seed=3)
    assert s1 == s2                 # deterministic
    assert 0.0 <= s1 <= 1.0
    assert s1 > 0.8                 # separable data → a real, good CV score


def test_fold_score_is_single_fold_and_cv_score_is_their_mean():
    """metis#18 M1a: fold_score scores ONE fold (analysis-fit, assessment-score);
    cv_score is exactly the mean over folds (DRY — the engine drives the fold axis)."""
    import pandas as pd

    from metis.model import fold_score

    X, y = _separable(n=120, seed=1)
    df = pd.DataFrame(X)
    df["target"] = y
    folds = cv_folds(df, k=5, seed=3, stratify_col="target")

    per_fold = [fold_score(X, y, folds, f, "logreg", seed=3) for f in sorted(set(folds))]
    assert len(per_fold) == 5
    assert all(0.0 <= s <= 1.0 for s in per_fold)
    # cv_score == mean(fold_score) — the reducer the resample Sampler's Done performs
    assert np.isclose(float(np.mean(per_fold)), cv_score(X, y, folds, "logreg", seed=3))
    # fold_score is deterministic per fold
    assert fold_score(X, y, folds, 0, "logreg", seed=3) == fold_score(X, y, folds, 0, "logreg", seed=3)
