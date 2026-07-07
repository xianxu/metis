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


def parse_model_config(raw) -> tuple[str, dict]:
    """Normalize a `with["model"]` value to `(kind, params)`.

    Accepts BOTH forms the pipeline produces:
    - a bare string (`"logreg"`) — the v0 form → `(kind, {})`.
    - a single-key dict (`{"rf": {"n_estimators": 200, "max_depth": 4}}`) — metis#6's `$any`-map
      (tagged, ex-`$oneof`) labeled-sum bundle → `(kind, params)`.
    Anything else (multi-key dict, empty, non-str kind, non-dict/str) is a loud error — a
    malformed model knob must fail, not silently pick a branch.
    """
    if isinstance(raw, str):
        return raw, {}
    if isinstance(raw, dict) and len(raw) == 1:
        (kind, params), = raw.items()
        if isinstance(kind, str):
            return kind, dict(params or {})
    raise ValueError(
        f"malformed model config {raw!r}; want a kind string (\"logreg\") or a single-key "
        f'$any-map bundle ({{"rf": {{...}}}})'
    )


def make_model(kind: str, seed: int, params: dict | None = None):
    """Construct an unfitted estimator of the given kind, seeded for determinism.

    `params` are the swept hyperparams (from the `$any`-map branch); known keys are applied,
    unknown keys ignored (forward-compatible with shapes carrying extra knobs).
    """
    p = params or {}
    if kind == "logreg":
        return LogisticRegression(C=p.get("C", 1.0), max_iter=1000, random_state=seed)
    if kind == "rf":
        return RandomForestClassifier(
            n_estimators=p.get("n_estimators", 100), max_depth=p.get("max_depth"),
            random_state=seed)
    raise ValueError(f"unknown model {kind!r}; want one of {sorted(MODELS)}")


def train(X, y, kind: str, seed: int, params: dict | None = None):
    """Fit and return an estimator. Pure given (X, y, kind, seed, params)."""
    model = make_model(kind, seed, params)
    model.fit(X, y)
    return model


def predict(estimator, X):
    """Predict labels for X with a fitted estimator."""
    return estimator.predict(X)


def cv_score(X, y, folds, kind: str, seed: int, params: dict | None = None) -> float:
    """Mean validation accuracy over the fold assignment (pure, deterministic).

    For each fold f: train on rows where fold != f, score on rows where fold == f;
    return the mean accuracy. Uses numpy internally so per-fold models are
    name-free and reproducible. `params` are the swept hyperparams (threaded to make_model).
    """
    Xa = np.asarray(X)
    ya = np.asarray(y)
    fa = np.asarray(folds)
    scores = []
    for f in sorted(set(fa.tolist())):
        val = fa == f
        trn = ~val
        model = train(Xa[trn], ya[trn], kind, seed, params)
        scores.append(accuracy_score(ya[val], predict(model, Xa[val])))
    return float(np.mean(scores))
