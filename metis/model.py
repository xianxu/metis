"""train / predict / cv_score — the pure modeling core (metis#1 M3).

Thin, deterministic wrappers over sklearn estimators (logreg / random forest) plus
a cross-validated scorer (metric-parameterized, metis#59: accuracy | balanced_accuracy). All deterministic given a seed and IO-free, so they are
unit-tested directly on in-memory arrays (ARCH-PURE); the train/predict step
entrypoints (metis.steps.*) are the only place these meet the filesystem.
"""

from __future__ import annotations

import numpy as np
from sklearn.ensemble import (HistGradientBoostingClassifier, HistGradientBoostingRegressor,
                              RandomForestClassifier, RandomForestRegressor, VotingClassifier)
from sklearn.linear_model import LogisticRegression, Ridge
from sklearn.metrics import accuracy_score, balanced_accuracy_score, root_mean_squared_error

from metis.split import cv_folds

# The regressor kinds (metis#36 M0): rogii-wellbore is RMSE regression. They share the pure
# train/predict/fold path with the classifiers; predict + complexity + the decide guard branch on
# is_regression (a regressor has no classes_/predict_proba, so it carries no decision layer).
_REGRESSORS = frozenset({"rf_reg", "hist_gbm_reg", "ridge"})
MODELS = frozenset({"logreg", "rf", "hist_gbm", "ensemble", "catboost"}) | _REGRESSORS


def is_regression(kind: str) -> bool:
    """True iff `kind` is a regressor (metis#36 M0) — the branch predict/complexity/fold_fit use
    to route (regressors have no classes_/predict_proba → no decision layer)."""
    return kind in _REGRESSORS


# The closed scorer set (metis#59; +rmse metis#36 M0). The ONE place a metric name resolves to a
# scorer — fold_fit consumes it; the train step ALSO resolves eagerly at entry so an unknown metric
# fails loudly on every path (incl. the foldless ship refit, which never scores). rmse is an ERROR
# metric (lower is better) — the shape declares objective.direction: minimize; the offset-tuner
# (which maximizes) never sees it (regression has no decision layer).
_SCORERS = {"accuracy": accuracy_score, "balanced_accuracy": balanced_accuracy_score,
            "rmse": root_mean_squared_error}


def resolve_scorer(metric: str):
    """Resolve a metric name to its sklearn scorer; loud on unknown (parse_model_config pattern)."""
    scorer = _SCORERS.get(metric)
    if scorer is None:
        raise ValueError(f"unknown metric {metric!r}; want one of {sorted(_SCORERS)}")
    return scorer


# ── the decision layer (metis#60): cost-sensitive plug-in rule, leaf-local ──────
# INFERENCE (estimate p(y|x); metric-independent) vs DECISION (choose labels to
# maximize the declared objective). Offsets are additive in log-probability space
# = multiplicative prior reweighting — the Bayes-correct family for a prior-shifted
# metric (balanced accuracy = accuracy under a uniform prior = the diagonal cost
# matrix 1/π_k). A future full cost-matrix rule generalizes without touching the
# sweeper: it only ever sees a scalar per-fold score of the declared procedure.

# ±4 covers −log-prior optima down to ~1.8% class priors (metis#60 review issue 2).
_OFFSET_GRID = np.linspace(-4.0, 4.0, 41)


def parse_decide(raw):
    """Normalize a `with["decide"]` value to `(rule, params)` — loud misuse pattern.

    Accepts "argmax"/None (default: plain argmax, today's behavior) or the single-key
    bundle {"offsets": {"holdout": 0.2}} (holdout ∈ (0, 0.5]; round(1/holdout) quantizes
    the effective internal split, e.g. 0.3 → k=3 → 1/3)."""
    if raw is None or raw == "argmax":
        return "argmax", {}
    if isinstance(raw, dict) and len(raw) == 1 and "offsets" in raw:
        p = dict(raw["offsets"] or {})
        # Closed config, exactly one legal key today — unlike parse_model_config's params
        # (an open sweep surface), a typo here must be LOUD, not a silent default.
        unknown = set(p) - {"holdout"}
        if unknown:
            raise ValueError(
                f"decide offsets bundle has unknown key(s) {sorted(unknown)}; known: ['holdout']")
        holdout = p.get("holdout", 0.2)
        if not (isinstance(holdout, (int, float)) and not isinstance(holdout, bool)
                and 0 < holdout <= 0.5):
            raise ValueError(f"decide offsets.holdout must be in (0, 0.5], got {holdout!r}")
        p["holdout"] = float(holdout)
        return "offsets", p
    raise ValueError(
        f'malformed decide config {raw!r}; want "argmax" or a single-key bundle '
        f'{{"offsets": {{"holdout": 0.2}}}}')


def predict_proba(estimator, X):
    """Class probabilities for X, columns in `estimator.classes_` order."""
    return estimator.predict_proba(X)


def apply_offsets(proba, offsets):
    """Decide class-column INDICES: argmax(log(clip(proba)) + offsets). Zero offsets ≡
    plain argmax (log is monotone; clip guards log(0)). Callers map indices → labels via
    the estimator's classes_."""
    logp = np.log(np.clip(np.asarray(proba, dtype=float), 1e-12, None))
    return (logp + np.asarray(offsets, dtype=float)).argmax(axis=1)


def tune_class_offsets(proba, y, metric: str = "balanced_accuracy", classes=None):
    """Grid-tune per-class log-offsets maximizing `metric` on (proba, y) — pure, no RNG.

    Class 0 is pinned to 0 (only relative tilts matter → K-1 free params); each free class
    sweeps _OFFSET_GRID. Best-so-far initializes at the NO-OP (zeros), and only a STRICT
    improvement replaces it — so an uninformative proba matrix returns zeros, never an
    arbitrary grid corner. Cost: O(grid^(K-1) × n) — priced by the caller (2-fits-per-leaf
    note in the train docstring)."""
    from itertools import product

    proba = np.asarray(proba, dtype=float)
    y = np.asarray(y)
    k = proba.shape[1]
    classes = np.arange(k) if classes is None else np.asarray(classes)
    scorer = resolve_scorer(metric)
    logp = np.log(np.clip(proba, 1e-12, None))

    best = np.zeros(k)
    best_score = scorer(y, classes[(logp + best).argmax(axis=1)])
    for combo in product(_OFFSET_GRID, repeat=k - 1):
        offs = np.concatenate([[0.0], combo])
        score = scorer(y, classes[(logp + offs).argmax(axis=1)])
        if score > best_score:
            best, best_score = offs, score
    return best


def tune_offsets_on_holdout(X, y, kind: str, seed: int, params: dict | None, metric: str,
                            holdout: float):
    """Learn decision offsets from TRAINING rows only (metis#60, leaf-local): an internal
    seeded stratified split (k=round(1/holdout), fold 0 = the tuning slice), an auxiliary
    fit on the rest, offsets tuned on the held-out slice's probabilities. The aux model's
    training-row probabilities are NOT used — an overfit model's in-sample probabilities
    are overconfident and offsets tuned on them are garbage."""
    import pandas as pd

    Xa, ya = np.asarray(X), np.asarray(y)
    k = int(round(1.0 / holdout))
    uniq, counts = np.unique(ya, return_counts=True)
    if counts.min() < k:
        raise ValueError(
            f"decide=offsets needs >= {k} rows of every class among the fold's training rows "
            f"(internal stratified holdout k={k} = round(1/holdout={holdout})); got class "
            f"counts {dict(zip(uniq.tolist(), counts.tolist()))}")
    inner = np.asarray(cv_folds(pd.DataFrame({"_y": ya}), k, seed, stratify_col="_y"))
    tune = inner == 0
    aux = train(Xa[~tune], ya[~tune], kind, seed, params)
    return tune_class_offsets(predict_proba(aux, Xa[tune]), ya[tune], metric=metric,
                              classes=aux.classes_)


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

    **Seed passthrough (metis#65):** a `params["seed"]` OVERRIDES the ctx seed for THIS
    estimator (absent → ctx seed, byte-identical to before; present → re-keys the leaf since
    it rides `with.model` → Kpre). This is what lets a shape sweep seed as a dimension, and —
    composed with the `ensemble` kind — turns "one config × several seeds" into seed-bagging.
    """
    p = params or {}
    eff_seed = int(p.get("seed", seed))
    if kind == "logreg":
        return LogisticRegression(C=p.get("C", 1.0), max_iter=1000, random_state=eff_seed)
    if kind == "rf":
        return RandomForestClassifier(
            n_estimators=p.get("n_estimators", 100), max_depth=p.get("max_depth"),
            class_weight=p.get("class_weight"),
            random_state=eff_seed)
    if kind == "hist_gbm":
        return HistGradientBoostingClassifier(
            learning_rate=p.get("learning_rate", 0.1), max_iter=p.get("max_iter", 100),
            max_leaf_nodes=p.get("max_leaf_nodes", 31), max_depth=p.get("max_depth"),
            class_weight=p.get("class_weight"),
            random_state=eff_seed)
    if kind == "catboost":
        # The M5 mechanism bet (metis#65): per-node ordered target statistics — a sharper
        # test of "cell signal matters conditionally" than the flat global encoding (M3), and
        # the most-different boosting bias → best blend partner. Lazy import: catboost is a
        # heavy dep; only pay its import when a catboost leaf actually runs (keeps the
        # forkserver's third-party preload light for the other kinds).
        from catboost import CatBoostClassifier
        cw = p.get("class_weight")
        kw = dict(
            iterations=p.get("iterations", p.get("max_iter", 200)),
            depth=p.get("depth", 6),
            random_seed=eff_seed,
            # ARCH-PURE + determinism pins: no catboost_info/ FS write, no stdout, and the
            # orchestrator owns parallelism (metis#48 — one thread per leaf, N leaves in
            # flight via the #31 semaphore), so a single-thread fit is both correct and
            # reproducible.
            thread_count=1,
            allow_writing_files=False,
            logging_level="Silent",
        )
        if "learning_rate" in p:
            kw["learning_rate"] = p["learning_rate"]  # else CatBoost auto-selects
        if cw == "balanced":
            kw["auto_class_weights"] = "Balanced"  # CatBoost's inverse-frequency reweighting
        elif cw is not None:
            raise ValueError(f"catboost class_weight: only 'balanced' or None, got {cw!r}")
        return CatBoostClassifier(**kw)
    if kind == "rf_reg":
        return RandomForestRegressor(
            n_estimators=p.get("n_estimators", 100), max_depth=p.get("max_depth"),
            random_state=eff_seed)
    if kind == "hist_gbm_reg":
        return HistGradientBoostingRegressor(
            learning_rate=p.get("learning_rate", 0.1), max_iter=p.get("max_iter", 100),
            max_leaf_nodes=p.get("max_leaf_nodes", 31), max_depth=p.get("max_depth"),
            random_state=eff_seed)
    if kind == "ridge":
        return Ridge(alpha=p.get("alpha", 1.0), random_state=eff_seed)
    if kind == "ensemble":
        # Soft-vote blend (metis#65): the blend made scorable INSIDE nested CV (vs `metis
        # blend`'s post-hoc leaderboard-only combine). members = a list of $any-map bundles
        # ({"rf": {...}}, ...) parsed by the SAME parse_model_config (ARCH-DRY, one level of
        # recursion). Each member is NAMED by its kind (suffixed -<i> for uniqueness) so
        # complexity() recovers the kind from the name — no estimator-type→kind reverse map.
        # eff_seed is each member's BASE seed; a member's own params["seed"] still overrides
        # (seed-bagging). VotingClassifier(soft) averages predict_proba (weighted by weights),
        # exposing fit/predict/predict_proba/classes_ → it composes with decide/metric/seal
        # unchanged.
        members = p.get("members")
        if not isinstance(members, list) or not members:
            raise ValueError(f"ensemble needs a non-empty 'members' list; got {members!r}")
        estimators = [(f"{mk}-{i}", make_model(mk, eff_seed, mp))
                      for i, (mk, mp) in enumerate(parse_model_config(m) for m in members)]
        return VotingClassifier(estimators=estimators, voting="soft", weights=p.get("weights"))
    raise ValueError(f"unknown model {kind!r}; want one of {sorted(MODELS)}")


def train(X, y, kind: str, seed: int, params: dict | None = None):
    """Fit and return an estimator. Pure given (X, y, kind, seed, params)."""
    model = make_model(kind, seed, params)
    model.fit(X, y)
    return model


def predict(estimator, X):
    """Predict labels for X with a fitted estimator, as a 1-D array in the label dtype.

    Two normalizations, single-sourced here (a no-op for the sklearn kinds; both guard
    CatBoost, whose `.predict()` can return shape (n, 1) and — on some versions/configs —
    FLOAT class labels): `reshape(-1)` → 1-D, then `.astype(classes_.dtype)` → the label
    dtype the estimator learned (so `predictions.csv` writes `0`, never `0.0`; the s6e7
    submission step maps INT codes → string labels). Fixes catboost everywhere predict flows
    (fold_score/cv_score + the ship predict step's argmax path)."""
    out = np.asarray(estimator.predict(X)).reshape(-1)
    if not hasattr(estimator, "classes_"):
        return out  # metis#36 M0: a regressor has no classes_ — keep the continuous output
    return out.astype(estimator.classes_.dtype)


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
    if kind in ("rf", "rf_reg"):  # metis#36 M0: the regressor variant shares the measure
        leaves = [t.tree_.n_leaves for t in fitted.estimators_]
        return float(sum(leaves) / len(leaves))
    if kind in ("logreg", "ridge"):  # coef count (both single-source the linear feature count)
        return float(fitted.coef_.size)
    if kind in ("hist_gbm", "hist_gbm_reg"):
        # _predictors is a list-of-lists: one inner list per boosting iteration, holding K
        # TreePredictors for K classes (binary → 1) — flatten and sum realized leaf counts.
        # (Private attr: sklearn 1.9.0 exposes no public per-tree accessor; if an upgrade
        # breaks this, that's the regression site.)
        return float(sum(t.get_n_leaf_nodes() for stage in fitted._predictors for t in stage))
    if kind == "catboost":
        # CatBoost grows OBLIVIOUS (symmetric) trees — full binary at fixed depth → 2^depth
        # leaves each; total leaves = tree_count × 2^depth (the summed-leaves capacity proxy,
        # the additive-boosting analogue of hist_gbm's total leaves).
        depth = int(fitted.get_all_params().get("depth", 6))
        return float(fitted.tree_count_ * (2 ** depth))
    if kind == "ensemble":
        # SUM of member realized complexities (aggregate capacity — the parsimony axis for a
        # blend). Recover each member's kind from its NAME (set in make_model from the
        # parse_model_config label, suffixed -<i>): rsplit on the LAST '-' — no kind name
        # contains '-', so this is unambiguous and derives from the single kind source (DRY).
        return float(sum(complexity(est, name.rsplit("-", 1)[0])
                         for name, est in fitted.named_estimators_.items()))
    raise ValueError(f"complexity: unknown model kind {kind!r}; want one of {sorted(MODELS)}")


def fold_fit(X, y, folds, fold_idx: int, kind: str, seed: int, params: dict | None = None,
             metric: str = "accuracy", decide="argmax"):
    """Fit ONE fold and return `(score, fitted_model, offsets)` — pure, deterministic.

    Train on the analysis rows (fold != fold_idx), score on the assessment rows
    (fold == fold_idx). Under `decide={"offsets": ...}` (metis#60) the decision offsets are
    a FITTED PARAMETER learned inside the fold's training rows (tune_offsets_on_holdout);
    the MAIN model is still fitted on ALL training rows (unchanged from argmax), and the
    assessment fold is scored through the tuned decision — the assessment never enters
    tuning, so the sealed sweep measures fit+tune as ONE procedure. `offsets` is None under
    argmax. The fitted model is returned so a caller can *also* read its realized
    complexity (metis#19) WITHOUT a second fit.
    """
    rule, dparams = parse_decide(decide)
    if rule == "offsets" and is_regression(kind):
        raise ValueError(
            f"decide=offsets is classification-only (it tunes class probabilities); {kind!r} is a "
            f"regressor — use decide='argmax' (the default)")
    Xa = np.asarray(X)
    ya = np.asarray(y)
    fa = np.asarray(folds)
    val = fa == fold_idx
    trn = ~val
    model = train(Xa[trn], ya[trn], kind, seed, params)
    if rule == "offsets":
        offsets = tune_offsets_on_holdout(Xa[trn], ya[trn], kind, seed, params, metric,
                                          dparams["holdout"])
        labels = model.classes_[apply_offsets(predict_proba(model, Xa[val]), offsets)]
        score = float(resolve_scorer(metric)(ya[val], labels))
        return score, model, offsets
    score = float(resolve_scorer(metric)(ya[val], predict(model, Xa[val])))
    return score, model, None


def fold_score(X, y, folds, fold_idx: int, kind: str, seed: int, params: dict | None = None,
               metric: str = "accuracy", decide="argmax") -> float:
    """Validation accuracy for ONE fold (pure, deterministic) — metis#18 M1a.

    The single-fold body cv_score loops over, LIFTED OUT so the engine drives the fold
    axis: each (config, fold) is a cached run emitting one fold_score, and the resample
    Sampler's Done reduces the k fold-scores → (mean, SE) (the ledger keeps the raw fold
    rows, so metis#19's select is a free re-reduction). Delegates to fold_fit (DRY).
    """
    score, _, _ = fold_fit(X, y, folds, fold_idx, kind, seed, params, metric=metric,
                           decide=decide)
    return score


def cv_score(X, y, folds, kind: str, seed: int, params: dict | None = None,
             metric: str = "accuracy", decide="argmax") -> float:
    """Mean validation accuracy over the fold assignment (pure, deterministic).

    The v1 whole-CV reducer, now expressed over fold_score (ARCH-DRY) — retained for
    callers that want the mean in one call; the M1a engine instead runs fold_score
    per (config, fold) and reduces in the resample Sampler's Done.
    """
    fa = np.asarray(folds)
    scores = [fold_score(X, y, folds, f, kind, seed, params, metric=metric, decide=decide)
              for f in sorted(set(fa.tolist()))]
    return float(np.mean(scores))
