"""Pure tests for metis.model — train/predict/cv_score (no IO)."""

import numpy as np
import pytest
from sklearn.datasets import make_classification

from metis.model import (apply_offsets, complexity, cv_score, fold_fit, fold_score, make_model,
                         parse_decide, parse_model_config, predict, resolve_scorer,
                         train, tune_class_offsets)
from metis.split import cv_folds


def _separable(n=120, seed=0):
    X, y = make_classification(
        n_samples=n, n_features=4, n_informative=3, n_redundant=0,
        n_clusters_per_class=1, class_sep=2.0, random_state=seed,
    )
    return X, y


@pytest.mark.parametrize("kind", ["logreg", "rf", "hist_gbm"])
def test_train_predict_shapes(kind):
    X, y = _separable()
    model = train(X, y, kind, seed=42)
    preds = predict(model, X)
    assert preds.shape == (len(y),)
    assert set(np.unique(preds)).issubset(set(np.unique(y)))


@pytest.mark.parametrize("kind", ["logreg", "rf", "hist_gbm"])
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
    gbm = make_model("hist_gbm", seed=0,
                     params={"learning_rate": 0.05, "max_iter": 7, "max_leaf_nodes": 15})
    assert gbm.learning_rate == 0.05
    assert gbm.max_iter == 7
    assert gbm.max_leaf_nodes == 15
    # Defaults hold when a param is absent / params is None (backward-compat).
    assert make_model("logreg", seed=0).C == 1.0
    assert make_model("rf", seed=0, params={"max_depth": 3}).n_estimators == 100
    assert make_model("hist_gbm", seed=0, params={"learning_rate": 0.2}).max_iter == 100


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


def test_complexity_rf_mean_leaves():
    """rf complexity = MEAN leaves per tree (metis#19) — the realized capacity, and
    n_estimators-neutral (mean, not total, per Breiman's LLN)."""
    X, y = _separable()
    m = train(X, y, "rf", seed=42, params={"n_estimators": 10, "max_depth": 4})
    expected = float(np.mean([t.tree_.n_leaves for t in m.estimators_]))
    assert complexity(m, "rf") == expected
    # n_estimators-neutral: 10 vs 50 trees on the same data → ~equal mean leaves.
    m10 = train(X, y, "rf", seed=42, params={"n_estimators": 10, "max_depth": 4})
    m50 = train(X, y, "rf", seed=42, params={"n_estimators": 50, "max_depth": 4})
    assert abs(complexity(m10, "rf") - complexity(m50, "rf")) < 1.0


def test_complexity_logreg_is_coef_count():
    """logreg complexity = coefficient count (L2 zeroes nothing → = feature count)."""
    X, y = _separable()
    m = train(X, y, "logreg", seed=42)
    assert complexity(m, "logreg") == float(m.coef_.size)
    assert complexity(m, "logreg") == 4.0  # _separable has 4 features


def test_complexity_hist_gbm_total_leaves():
    """hist_gbm complexity = TOTAL leaves summed across ALL boosted trees (metis#21,
    extending #19's measured-complexity) —
    SUM, not mean, because boosting is ADDITIVE (F(x)=Σ trees; ESL §10.2): each iteration
    adds capacity. The deliberate INVERSE of rf's n_estimators-neutrality (mean-per-tree
    would be max_iter-blind → blind to boosting's primary regularizer)."""
    X, y = _separable()
    m = train(X, y, "hist_gbm", seed=42, params={"max_iter": 20, "max_leaf_nodes": 8})
    # _predictors is a list-of-lists (one inner list per boosting iteration; K predictors
    # per iteration for K-class — binary Titanic → 1), so the count flattens.
    expected = float(sum(t.get_n_leaf_nodes() for stage in m._predictors for t in stage))
    assert complexity(m, "hist_gbm") == expected
    # max_iter-SENSITIVE (the inverse of rf's neutrality): more boosting rounds add trees,
    # so total leaves STRICTLY grows — the parsimony rule can then prefer fewer rounds.
    m10 = train(X, y, "hist_gbm", seed=42, params={"max_iter": 10, "max_leaf_nodes": 8})
    m40 = train(X, y, "hist_gbm", seed=42, params={"max_iter": 40, "max_leaf_nodes": 8})
    assert complexity(m40, "hist_gbm") > complexity(m10, "hist_gbm")


def test_complexity_unknown_raises():
    X, y = _separable()
    m = train(X, y, "rf", seed=0)
    with pytest.raises(ValueError, match="unknown model kind"):
        complexity(m, "svm")


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


# --- metis#59: metric knob + class_weight passthrough ---


def test_balanced_accuracy_scorer_on_skewed_labels():
    # majority-argmax: accuracy looks great (0.9), balanced accuracy exposes it (0.5).
    y_true = np.array([0] * 9 + [1])
    y_pred = np.zeros(10, dtype=int)
    assert resolve_scorer("accuracy")(y_true, y_pred) == pytest.approx(0.9)
    assert resolve_scorer("balanced_accuracy")(y_true, y_pred) == pytest.approx(0.5)


def test_unknown_metric_rejected():
    with pytest.raises(ValueError, match="balanced_accuracy"):  # message names the closed set
        resolve_scorer("auc")


def test_fold_score_metric_kwarg_changes_the_score_and_cv_threads():
    import pandas as pd

    # Constant feature + 10/2 skew → the model predicts the majority class everywhere.
    # STRATIFIED k=2 puts one minority row in each assessment fold (unstratified could leave
    # a single-class fold where balanced accuracy degenerates to accuracy).
    X = np.ones((12, 1))
    y = np.array([0] * 10 + [1] * 2)
    folds = cv_folds(pd.DataFrame({"y": y}), 2, 0, stratify_col="y")
    acc = fold_score(X, y, folds, 0, "logreg", 0)
    bal = fold_score(X, y, folds, 0, "logreg", 0, metric="balanced_accuracy")
    assert acc == pytest.approx(5 / 6)
    assert bal == pytest.approx(0.5)
    # cv_score threads the metric to fold_score: it IS the mean of the balanced fold scores.
    per_fold = [fold_score(X, y, folds, f, "logreg", 0, metric="balanced_accuracy") for f in (0, 1)]
    assert cv_score(X, y, folds, "logreg", 0, metric="balanced_accuracy") == pytest.approx(
        float(np.mean(per_fold)))


def test_make_model_class_weight_reaches_estimators():
    for kind in ("rf", "hist_gbm"):
        m = make_model(kind, seed=0, params={"class_weight": "balanced"})
        assert m.class_weight == "balanced"
        assert make_model(kind, seed=0).class_weight is None  # default unchanged


# --- metis#60: decision layer (probabilities + offsets) ---


def _decide_frame(seed=0):
    """40 rows, 30/10 skew, ONE weak-but-informative feature (overlapping normals) — the
    dedicated decide test frame (#60 plan review issue 1: the 12-row constant-feature frame
    is both illegal for the internal stratified holdout and vacuous for tuning)."""
    rng = np.random.default_rng(seed)
    X = np.concatenate([rng.normal(0.0, 1.0, 30), rng.normal(1.5, 1.0, 10)]).reshape(-1, 1)
    y = np.array([0] * 30 + [1] * 10)
    return X, y


def _posterior_matrix():
    """400 rows of hand-built TRUE posteriors: 4 distinct posterior vectors, each tiled
    100x with labels matching the vector's frequencies exactly. Deterministic, RNG-free."""
    blocks = [
        ([0.85, 0.09, 0.06], (85, 9, 6)),
        ([0.70, 0.20, 0.10], (70, 20, 10)),
        ([0.55, 0.15, 0.30], (55, 15, 30)),
        ([0.60, 0.30, 0.10], (60, 30, 10)),
    ]
    proba, y = [], []
    for vec, counts in blocks:
        for cls, c in enumerate(counts):
            proba.extend([vec] * c)
            y.extend([cls] * c)
    return np.array(proba), np.array(y)


def test_tune_class_offsets_recovers_balanced_optimum():
    proba, y = _posterior_matrix()
    scorer = resolve_scorer("balanced_accuracy")
    argmax_score = scorer(y, proba.argmax(axis=1))            # all-majority: 1/3
    # the Bayes plug-in for balanced accuracy: argmax p(k|x)/prior_k
    priors = np.bincount(y) / len(y)
    prior_offsets = -np.log(priors) + np.log(priors[0])       # pinned to class 0
    prior_score = scorer(y, apply_offsets(proba, prior_offsets))
    offsets = tune_class_offsets(proba, y, metric="balanced_accuracy")
    tuned_score = scorer(y, apply_offsets(proba, offsets))
    assert offsets[0] == 0.0                                  # class 0 pinned
    assert tuned_score > argmax_score
    assert tuned_score >= prior_score - 0.02                  # 270/74/56 (≈67.5/18.5/14%) priors — optima ≈1.29/1.57, inside the ±4 grid: optima inside the ±4 grid


def test_tune_class_offsets_deterministic_and_noop_on_uniform():
    proba, y = _posterior_matrix()
    o1 = tune_class_offsets(proba, y, metric="balanced_accuracy")
    o2 = tune_class_offsets(proba, y, metric="balanced_accuracy")
    assert np.array_equal(o1, o2)
    uniform = np.full((60, 3), 1 / 3)
    yu = np.array([0, 1, 2] * 20)
    assert np.array_equal(tune_class_offsets(uniform, yu, metric="balanced_accuracy"),
                          np.zeros(3))                        # no improvement -> no-op zeros


def test_apply_offsets_zero_is_plain_argmax_incl_zero_proba():
    proba = np.array([[0.7, 0.3, 0.0], [0.0, 0.5, 0.5], [0.2, 0.2, 0.6]])
    assert np.array_equal(apply_offsets(proba, np.zeros(3)), proba.argmax(axis=1))


def test_parse_decide_table():
    assert parse_decide(None) == ("argmax", {})
    assert parse_decide("argmax") == ("argmax", {})
    rule, p = parse_decide({"offsets": {}})
    assert rule == "offsets" and p["holdout"] == pytest.approx(0.2)
    rule, p = parse_decide({"offsets": {"holdout": 0.5}})
    assert p["holdout"] == pytest.approx(0.5)
    for bad in ("foo", {"offsets": {"holdout": 0.6}}, {"offsets": {"holdout": 0}},
                {"blend": {}}, {"offsets": {}, "extra": 1}):
        with pytest.raises(ValueError, match="decide"):
            parse_decide(bad)

    with pytest.raises(ValueError, match="holdour"):   # typo'd inner key must be LOUD (close-review)
        parse_decide({"offsets": {"holdour": 0.25}})

def test_fold_fit_offsets_on_decide_frame_and_loud_small_frame():
    import pandas as pd

    X, y = _decide_frame()
    folds = cv_folds(pd.DataFrame({"y": y}), 2, 0, stratify_col="y")
    score_argmax, _, off_none = fold_fit(X, y, folds, 0, "logreg", 0,
                                         metric="balanced_accuracy")
    assert off_none is None
    score_tuned, _, offsets = fold_fit(X, y, folds, 0, "logreg", 0,
                                       metric="balanced_accuracy",
                                       decide={"offsets": {"holdout": 0.2}})
    assert offsets is not None and len(offsets) == 2
    assert score_tuned >= score_argmax - 0.1                  # no-op grid point bounds the tune slice

    # too-small frame: 6 training rows / 1 minority row -> internal k=5 illegal, LOUD
    Xs = np.ones((12, 1))
    ys = np.array([0] * 10 + [1] * 2)
    fs = cv_folds(pd.DataFrame({"y": ys}), 2, 0, stratify_col="y")
    with pytest.raises(ValueError, match="training rows"):
        fold_fit(Xs, ys, fs, 0, "logreg", 0, metric="balanced_accuracy",
                 decide={"offsets": {"holdout": 0.2}})


def test_fold_fit_offsets_main_model_is_all_training_rows():
    import pandas as pd

    X, y = _decide_frame()
    folds = cv_folds(pd.DataFrame({"y": y}), 2, 0, stratify_col="y")
    _, m_argmax, _ = fold_fit(X, y, folds, 0, "logreg", 0, metric="balanced_accuracy")
    _, m_tuned, _ = fold_fit(X, y, folds, 0, "logreg", 0, metric="balanced_accuracy",
                             decide={"offsets": {"holdout": 0.2}})
    # same seed + same (all) training rows -> identical main model predictions
    assert np.array_equal(m_argmax.predict(X), m_tuned.predict(X))


# --- metis#65: seed passthrough (params-level seed override) ---


def test_seed_passthrough_overrides_ctx_seed():
    """A params["seed"] overrides the ctx seed at the estimator; absent = ctx seed (no re-key)."""
    for kind in ("logreg", "rf", "hist_gbm"):
        assert make_model(kind, seed=3).random_state == 3              # default: ctx seed
        assert make_model(kind, seed=3, params={"seed": 7}).random_state == 7   # override wins
        # override is independent of ctx seed (same eff_seed regardless of ctx)
        a = make_model(kind, seed=0, params={"seed": 5}).random_state
        b = make_model(kind, seed=999, params={"seed": 5}).random_state
        assert a == b == 5


def _noisy(n=300, seed=0):
    """Non-separable frame (class overlap + label noise) — different bootstrap seeds yield
    genuinely different forests here, unlike the trivially-separable `_separable`."""
    X, y = make_classification(
        n_samples=n, n_features=6, n_informative=3, n_redundant=0, n_clusters_per_class=2,
        class_sep=0.6, flip_y=0.12, random_state=seed,
    )
    return X, y


def test_seed_passthrough_changes_the_fit():
    """The overridden seed must actually reach the fit — two rf seeds → different forests
    (compared on predict_proba, sensitive to per-tree bootstrap differences)."""
    X, y = _noisy(seed=1)
    m1 = train(X, y, "rf", seed=0, params={"seed": 1, "max_depth": 3})
    m2 = train(X, y, "rf", seed=0, params={"seed": 2, "max_depth": 3})
    assert not np.allclose(m1.predict_proba(X), m2.predict_proba(X))   # distinct bootstraps
    # and the override is deterministic (same seed → same fit, regardless of ctx seed)
    m3 = train(X, y, "rf", seed=555, params={"seed": 1, "max_depth": 3})
    assert np.array_equal(m1.predict_proba(X), m3.predict_proba(X))


# --- metis#65: ensemble model kind (soft-vote blend, scorable in nested CV) ---


def _ensemble_params(seeds=None):
    """Two-member ensemble bundle (rf + hist_gbm). `seeds` (optional) seed-bags the members."""
    rf = {"n_estimators": 20, "max_depth": 3}
    gbm = {"max_iter": 20, "max_leaf_nodes": 7}
    if seeds is not None:
        rf, gbm = {**rf, "seed": seeds[0]}, {**gbm, "seed": seeds[1]}
    return {"members": [{"rf": rf}, {"hist_gbm": gbm}]}


def test_parse_model_config_ensemble_bundle():
    """The ensemble bundle normalizes like any $any-map: {"ensemble": {...}} → ("ensemble", {...})."""
    raw = {"ensemble": _ensemble_params()}
    kind, params = parse_model_config(raw)
    assert kind == "ensemble"
    assert params == _ensemble_params()


def test_make_model_ensemble_requires_members():
    for bad in ({}, {"members": []}, {"members": "rf"}, {"weights": [1, 1]}):
        with pytest.raises(ValueError, match="members"):
            make_model("ensemble", seed=0, params=bad)


def test_ensemble_train_predict_shapes_and_classes():
    X, y = _separable()
    m = train(X, y, "ensemble", seed=42, params=_ensemble_params())
    preds = predict(m, X)
    assert preds.shape == (len(y),)
    assert set(np.unique(preds)).issubset(set(np.unique(y)))
    # named by kind (DRY complexity recovery); suffixed -<i> for uniqueness.
    assert list(m.named_estimators_) == ["rf-0", "hist_gbm-1"]


def test_ensemble_predict_proba_is_member_mean():
    """Soft vote (unweighted) = the plain mean of member predict_probas over estimators_."""
    X, y = _separable()
    m = train(X, y, "ensemble", seed=42, params=_ensemble_params())
    member_mean = np.mean([e.predict_proba(X) for e in m.estimators_], axis=0)
    assert np.allclose(m.predict_proba(X), member_mean)


def test_ensemble_weights_tilt_the_average():
    """weights re-weight the soft vote: [3,1] → (3·p_rf + 1·p_gbm)/4."""
    X, y = _separable()
    params = {**_ensemble_params(), "weights": [3, 1]}
    m = train(X, y, "ensemble", seed=42, params=params)
    p_rf, p_gbm = (e.predict_proba(X) for e in m.estimators_)
    assert np.allclose(m.predict_proba(X), (3 * p_rf + 1 * p_gbm) / 4)


def test_ensemble_single_member_matches_bare_model():
    """A one-member ensemble is a degenerate no-op: its proba == the lone member's (a cheap pin)."""
    X, y = _separable()
    m = train(X, y, "ensemble", seed=42, params={"members": [{"rf": {"n_estimators": 20, "max_depth": 3}}]})
    (lone,) = m.estimators_
    assert np.allclose(m.predict_proba(X), lone.predict_proba(X))
    assert complexity(m, "ensemble") == complexity(lone, "rf")


def test_ensemble_complexity_is_sum_of_members():
    """Aggregate capacity = Σ member realized complexities; member kind recovered from the name."""
    X, y = _separable()
    m = train(X, y, "ensemble", seed=42, params=_ensemble_params())
    rf_fit, gbm_fit = m.estimators_
    assert complexity(m, "ensemble") == complexity(rf_fit, "rf") + complexity(gbm_fit, "hist_gbm")


def test_ensemble_seed_bagging_distinct_seeds_distinct_members():
    """Seed-bagging (metis#65 ✕ ensemble): two rf members with DISTINCT seeds → distinct fits.
    (rf, not hist_gbm — hist_gbm's random_state is a no-op below the 10k early-stopping cutoff.)"""
    X, y = _noisy(seed=2)
    params = {"members": [{"rf": {"n_estimators": 20, "max_depth": 3, "seed": 1}},
                          {"rf": {"n_estimators": 20, "max_depth": 3, "seed": 2}}]}
    m = train(X, y, "ensemble", seed=42, params=params)
    a, b = m.estimators_
    assert not np.allclose(a.predict_proba(X), b.predict_proba(X))   # the two seed-bags differ


def test_ensemble_composes_with_decide_offsets():
    """The decision layer tunes offsets on the ENSEMBLE's averaged proba (metis#60 ✕ #65)."""
    import pandas as pd

    X, y = _decide_frame()
    folds = cv_folds(pd.DataFrame({"y": y}), 2, 0, stratify_col="y")
    params = {"members": [{"rf": {"n_estimators": 20, "max_depth": 3}},
                          {"logreg": {}}]}
    score, model, offsets = fold_fit(X, y, folds, 0, "ensemble", 0, params,
                                     metric="balanced_accuracy",
                                     decide={"offsets": {"holdout": 0.2}})
    assert offsets is not None and len(offsets) == 2   # per-class offsets on the blend
    assert isinstance(model, type(train(X, y, "ensemble", 0, params)))  # a VotingClassifier
    assert 0.0 <= score <= 1.0
