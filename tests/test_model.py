"""Pure tests for metis.model — train/predict/cv_score (no IO)."""

import numpy as np
import pytest
from sklearn.datasets import make_classification

from metis.model import cv_score, make_model, predict, train
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
