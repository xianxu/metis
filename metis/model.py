"""train / predict / cv_score — the pure modeling core (metis#1 M3).

Thin, deterministic wrappers over sklearn estimators (logreg / random forest) plus
a cross-validated scorer. All deterministic given a seed and IO-free, so they are
unit-tested directly on in-memory arrays (ARCH-PURE); the train/predict step
entrypoints (metis.steps.*) are the only place these meet the filesystem.
"""

from __future__ import annotations

import numpy as np
from sklearn.ensemble import HistGradientBoostingClassifier, RandomForestClassifier
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import accuracy_score

MODELS = frozenset({"logreg", "rf", "hist_gbm"})


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
    if kind == "hist_gbm":
        return HistGradientBoostingClassifier(
            learning_rate=p.get("learning_rate", 0.1), max_iter=p.get("max_iter", 100),
            max_leaf_nodes=p.get("max_leaf_nodes", 31), max_depth=p.get("max_depth"),
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


def complexity(fitted, kind: str) -> float:
    """Realized complexity of a FITTED model (metis#19) — the select rule's parsimony axis.

    Measured on the fitted object, not predicted from hyperparameters: trees prune and
    regularization sparsifies, so realized structure is the capacity (cost-complexity
    pruning penalizes realized leaf count |T|; `2^max_depth` overstates).
    - rf → MEAN leaves per tree (mean, not total, so it's n_estimators-neutral per
      Breiman's LLN — more trees reduce variance, not overfitting-capacity).
    - logreg → coefficient count (L2 zeroes nothing → all non-zero = feature count).
    - hist_gbm → TOTAL leaves SUMMED across all boosted trees (sum, NOT mean). Boosting is
      ADDITIVE (F(x)=Σ trees; ESL §10.2, Friedman 2001), so capacity SUMS and MORE iterations
      DO overfit (ESL §10.12; Bühlmann–Hothorn df(m)=trace(𝐁ₘ)↑m) — the exact inverse of rf's
      n_estimators-neutral mean. XGBoost's own Ω=γT penalizes total leaves across the ensemble.
      CAVEAT: total leaves is a clean monotone capacity proxy only WITHIN a fixed learning_rate;
      shrinkage (small ν needs more trees yet regularizes better) decouples leaf-count from
      effective DoF across ν, so a ν-sweeping shape would need a ν-weighted measure (deferred).
    """
    if kind == "rf":
        leaves = [t.tree_.n_leaves for t in fitted.estimators_]
        return float(sum(leaves) / len(leaves))
    if kind == "logreg":
        return float(fitted.coef_.size)
    if kind == "hist_gbm":
        # _predictors is a list-of-lists: one inner list per boosting iteration, holding K
        # TreePredictors for K classes (binary → 1) — flatten and sum realized leaf counts.
        return float(sum(t.get_n_leaf_nodes() for stage in fitted._predictors for t in stage))
    raise ValueError(f"complexity: unknown model kind {kind!r}; want one of {sorted(MODELS)}")


def fold_fit(X, y, folds, fold_idx: int, kind: str, seed: int, params: dict | None = None):
    """Fit ONE fold and return `(score, fitted_model)` — pure, deterministic (metis#18 M1a).

    Train on the analysis rows (fold != fold_idx), score on the assessment rows
    (fold == fold_idx). The fitted model is returned so a caller can *also* read its
    realized complexity (metis#19) WITHOUT a second fit — one fit feeds both score and
    complexity. Uses numpy so per-fold models are name-free and reproducible.
    """
    Xa = np.asarray(X)
    ya = np.asarray(y)
    fa = np.asarray(folds)
    val = fa == fold_idx
    trn = ~val
    model = train(Xa[trn], ya[trn], kind, seed, params)
    score = float(accuracy_score(ya[val], predict(model, Xa[val])))
    return score, model


def fold_score(X, y, folds, fold_idx: int, kind: str, seed: int, params: dict | None = None) -> float:
    """Validation accuracy for ONE fold (pure, deterministic) — metis#18 M1a.

    The single-fold body cv_score loops over, LIFTED OUT so the engine drives the fold
    axis: each (config, fold) is a cached run emitting one fold_score, and the resample
    Sampler's Done reduces the k fold-scores → (mean, SE) (the ledger keeps the raw fold
    rows, so metis#19's select is a free re-reduction). Delegates to fold_fit (DRY).
    """
    score, _ = fold_fit(X, y, folds, fold_idx, kind, seed, params)
    return score


def cv_score(X, y, folds, kind: str, seed: int, params: dict | None = None) -> float:
    """Mean validation accuracy over the fold assignment (pure, deterministic).

    The v1 whole-CV reducer, now expressed over fold_score (ARCH-DRY) — retained for
    callers that want the mean in one call; the M1a engine instead runs fold_score
    per (config, fold) and reduces in the resample Sampler's Done.
    """
    fa = np.asarray(folds)
    scores = [fold_score(X, y, folds, f, kind, seed, params) for f in sorted(set(fa.tolist()))]
    return float(np.mean(scores))
