"""train / predict / cv_score — the pure modeling core (metis#1 M3).

Thin, deterministic wrappers over sklearn estimators (logreg / random forest) plus
a cross-validated scorer. All deterministic given a seed and IO-free, so they are
unit-tested directly on in-memory arrays (ARCH-PURE); the train/predict step
entrypoints (metis.steps.*) are the only place these meet the filesystem.
"""

from __future__ import annotations

import numpy as np
from sklearn.ensemble import RandomForestClassifier
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import accuracy_score

MODELS = frozenset({"logreg", "rf"})


def make_model(kind: str, seed: int):
    """Construct an unfitted estimator of the given kind, seeded for determinism."""
    if kind == "logreg":
        return LogisticRegression(max_iter=1000, random_state=seed)
    if kind == "rf":
        return RandomForestClassifier(n_estimators=100, random_state=seed)
    raise ValueError(f"unknown model {kind!r}; want one of {sorted(MODELS)}")


def train(X, y, kind: str, seed: int):
    """Fit and return an estimator. Pure given (X, y, kind, seed)."""
    model = make_model(kind, seed)
    model.fit(X, y)
    return model


def predict(estimator, X):
    """Predict labels for X with a fitted estimator."""
    return estimator.predict(X)


def cv_score(X, y, folds, kind: str, seed: int) -> float:
    """Mean validation accuracy over the fold assignment (pure, deterministic).

    For each fold f: train on rows where fold != f, score on rows where fold == f;
    return the mean accuracy. Uses numpy internally so per-fold models are
    name-free and reproducible.
    """
    Xa = np.asarray(X)
    ya = np.asarray(y)
    fa = np.asarray(folds)
    scores = []
    for f in sorted(set(fa.tolist())):
        val = fa == f
        trn = ~val
        model = train(Xa[trn], ya[trn], kind, seed)
        scores.append(accuracy_score(ya[val], predict(model, Xa[val])))
    return float(np.mean(scores))
