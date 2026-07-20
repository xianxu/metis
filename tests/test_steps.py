"""Contract tests for the step entrypoints (metis.steps.*) against a temp run dir.

Drives each entrypoint through the M2 step contract (env + with.json + upstream
artifacts) exactly as the Go runner would, chaining cv-split → train → predict so
the upstream-artifact convention (folds via cv-split, model via train) is
exercised end-to-end at the Python level. The Go-side full e2e is in cmd/metis.
"""

import json
from pathlib import Path

import pandas as pd
import pytest

from metis import io
from metis.steps import cv_split, predict, train

# testdata/dataset/ contains toy/ ; used as METIS_EXP_DIR so `with.dataset: toy`
# resolves experiment-relative.
TOY_PARENT = Path(__file__).parents[1] / "testdata" / "dataset"


def _run_step(monkeypatch, run_dir, step_id, with_cfg, main_fn, seed=42, exp_dir=TOY_PARENT):
    step_dir = run_dir / step_id
    step_dir.mkdir(parents=True, exist_ok=True)
    (step_dir / "with.json").write_text(json.dumps(with_cfg))
    monkeypatch.setenv("METIS_STEP_DIR", str(step_dir))
    monkeypatch.setenv("METIS_RUN_DIR", str(run_dir))
    monkeypatch.setenv("METIS_STEP_ID", step_id)
    monkeypatch.setenv("METIS_EXP_DIR", str(exp_dir))
    monkeypatch.setenv("METIS_SEED", str(seed))
    main_fn()
    return step_dir


def test_cv_split_entrypoint(tmp_path, monkeypatch):
    run = tmp_path / "runs" / "r1"
    sd = _run_step(monkeypatch, run, "split", {"dataset": "toy", "k": 3, "stratify": True}, cv_split.main)

    folds = json.loads((sd / "folds.json").read_text())
    assert len(folds) == 60 and set(folds) == {0, 1, 2}
    metrics = json.loads((sd / "metrics.json").read_text())
    assert metrics["k"] == 3 and metrics["n"] == 60


def test_train_then_predict_chain(tmp_path, monkeypatch):
    run = tmp_path / "runs" / "r1"
    _run_step(monkeypatch, run, "split", {"dataset": "toy", "k": 3, "stratify": True}, cv_split.main)

    ts = _run_step(monkeypatch, run, "train", {"dataset": "toy", "folds": "split", "model": "logreg"}, train.main)
    assert (ts / "model.pkl").exists()
    cv = json.loads((ts / "metrics.json").read_text())["cv_score"]
    assert 0.0 < cv <= 1.0

    ps = _run_step(monkeypatch, run, "predict", {"dataset": "toy", "model": "train"}, predict.main)
    preds = pd.read_csv(ps / "predictions.csv")
    assert list(preds.columns) == ["id", "prediction"]
    assert len(preds) == 20  # the toy test split


def test_ship_refit_reads_captured_features_no_folds(tmp_path, monkeypatch):
    """The driver:single ship (metis#18 M1a-5): train + predict refit on the all-rows
    `features` output as a CAPTURED upstream artifact, with NO cv-split/folds.

    Two seams at once: (1) train's all-rows path must not require `folds` (the ship refit
    fits on ALL rows for predict — CV is the sweep's job, not the ship's); (2) predict must
    resolve `dataset: features` via io.dataset_dir (the captured handoff), not io.exp_path.
    """
    run = tmp_path / "runs" / "ship"
    # Simulate the all-rows `features` step: a captured `dataset/` artifact in its step dir.
    ds = io.load_dataset(str(TOY_PARENT / "toy"))
    io.save_dataset(ds, str(run / "features" / "dataset"))

    # train fits on ALL rows (no `_fold`, no `folds`) → model.pkl, no cv_score demanded.
    ts = _run_step(monkeypatch, run, "train", {"dataset": "features", "model": "logreg"}, train.main)
    assert (ts / "model.pkl").exists()
    assert "cv_score" not in json.loads((ts / "metrics.json").read_text())

    # predict reads the SAME captured features dataset (dataset_dir), not an exp-relative path.
    ps = _run_step(monkeypatch, run, "predict", {"dataset": "features", "model": "train"}, predict.main)
    preds = pd.read_csv(ps / "predictions.csv")
    assert list(preds.columns) == ["id", "prediction"]
    assert len(preds) == 20  # the toy test split — predict ran on the captured dataset's test frame


def test_train_step_accepts_any_map_model_config(tmp_path, monkeypatch):
    """The train step must consume the $any-map (ex-$oneof) bundle `{kind: {params}}` — the EXACT
    shape a hyperparam sweep (kbench#4) emits (was: `kind = w["model"]` failed on the dict)."""
    run = tmp_path / "runs" / "r-any-map"
    _run_step(monkeypatch, run, "split", {"dataset": "toy", "k": 3, "stratify": True}, cv_split.main)
    ts = _run_step(monkeypatch, run, "train",
                   {"dataset": "toy", "folds": "split",
                    "model": {"rf": {"n_estimators": 50, "max_depth": 3}}}, train.main)
    assert (ts / "model.pkl").exists()
    cv = json.loads((ts / "metrics.json").read_text())["cv_score"]
    assert 0.0 < cv <= 1.0  # a real CV score — the sweep point trains, no unknown-model error


def test_train_per_fold_emits_score_and_complexity(tmp_path, monkeypatch):
    """metis#19: the per-fold branch emits BOTH fold_score and complexity (the realized
    fitted-model complexity the select rule's parsimony consumes). rf → mean leaves > 0."""
    run = tmp_path / "runs" / "r-fold"
    _run_step(monkeypatch, run, "split", {"dataset": "toy", "k": 3, "stratify": True}, cv_split.main)
    ts = _run_step(monkeypatch, run, "train",
                   {"dataset": "toy", "folds": "split",
                    "model": {"rf": {"n_estimators": 20, "max_depth": 3}},
                    "_fold": {"partition": "p", "idx": 0}}, train.main)
    metrics = json.loads((ts / "metrics.json").read_text())
    assert 0.0 <= metrics["fold_score"] <= 1.0
    assert metrics["complexity"] > 0.0  # rf mean leaves/tree
    assert (ts / "model.pkl").exists() is False  # per-fold does not persist a model


def test_train_per_fold_emits_complexity_for_hist_gbm(tmp_path, monkeypatch):
    """metis#21: the per-fold branch is model-kind-generic — a hist_gbm config flows through
    the same `complexity(model, kind)` seam and emits its total-leaves complexity (> 0)."""
    run = tmp_path / "runs" / "r-gbm"
    _run_step(monkeypatch, run, "split", {"dataset": "toy", "k": 3, "stratify": True}, cv_split.main)
    ts = _run_step(monkeypatch, run, "train",
                   {"dataset": "toy", "folds": "split",
                    "model": {"hist_gbm": {"max_iter": 15, "max_leaf_nodes": 8}},
                    "_fold": {"partition": "p", "idx": 0}}, train.main)
    metrics = json.loads((ts / "metrics.json").read_text())
    assert 0.0 <= metrics["fold_score"] <= 1.0
    assert metrics["complexity"] > 0.0  # hist_gbm total leaves across boosted trees


def test_train_per_fold_ensemble_through_step_path(tmp_path, monkeypatch):
    """metis#65: an `ensemble` bundle flows through the SAME train step → parse_model_config →
    make_model(VotingClassifier) → complexity seam and emits fold_score + a finite complexity
    (the sum-of-members capacity). Covers the step/IO boundary the unit tests don't."""
    run = tmp_path / "runs" / "r-ens"
    _run_step(monkeypatch, run, "split", {"dataset": "toy", "k": 3, "stratify": True}, cv_split.main)
    ts = _run_step(monkeypatch, run, "train",
                   {"dataset": "toy", "folds": "split",
                    "model": {"ensemble": {"members": [
                        {"rf": {"n_estimators": 15, "max_depth": 3}},
                        {"hist_gbm": {"max_iter": 15, "max_leaf_nodes": 8}}]}},
                    "_fold": {"partition": "p", "idx": 0}}, train.main)
    metrics = json.loads((ts / "metrics.json").read_text())
    assert 0.0 <= metrics["fold_score"] <= 1.0
    assert metrics["complexity"] > 0.0  # Σ member complexities (rf mean-leaves + gbm total-leaves)


def test_step_context_requires_env(monkeypatch):
    for v in ("METIS_STEP_DIR", "METIS_RUN_DIR", "METIS_STEP_ID", "METIS_EXP_DIR", "METIS_SEED"):
        monkeypatch.delenv(v, raising=False)
    with pytest.raises(RuntimeError, match="not set"):
        io.step_context()


def _save_skewed_dataset(run_dir):
    """A 12-row 10/2-skewed captured dataset (metis#59) — the step-level counterpart of the
    unit tests' skewed frame (testdata/dataset/toy is 33/27 balanced, useless here). Saved
    into a fake upstream step dir and referenced as `dataset: skewed` (the captured-artifact
    pattern of test_ship_refit_reads_captured_features_no_folds)."""
    from metis.dataset import Dataset
    from metis.schema import Schema

    train_df = pd.DataFrame({
        "id": range(12),
        "x": [1.0] * 12,  # constant feature → the model predicts the majority class
        "y": [0] * 10 + [1] * 2,
    })
    schema = Schema(columns={"id": "id", "x": "feature", "y": "target"},
                    dtypes={"id": "int64", "x": "float64", "y": "int64"})
    io.save_dataset(Dataset(schema=schema, train=train_df, test=None), str(run_dir / "skewed" / "dataset"))


def test_train_step_metric_knob_changes_fold_score(tmp_path, monkeypatch):
    """with.metric reaches the per-fold scorer (metis#59): on a 10/2 skew with a constant
    feature, accuracy = 5/6 per stratified k=2 fold while balanced_accuracy = 0.5."""
    from metis.split import cv_folds

    run = tmp_path / "runs" / "r-metric"
    _save_skewed_dataset(run)
    folds = cv_folds(pd.DataFrame({"y": [0] * 10 + [1] * 2}), 2, 42, stratify_col="y")
    split_dir = run / "split"
    split_dir.mkdir(parents=True)
    (split_dir / "folds.json").write_text(json.dumps(folds))

    base = {"dataset": "skewed", "folds": "split", "model": "logreg", "_fold": {"idx": 0}}
    ts = _run_step(monkeypatch, run, "train-acc", base, train.main)
    acc = json.loads((ts / "metrics.json").read_text())["fold_score"]
    ts = _run_step(monkeypatch, run, "train-bal", {**base, "metric": "balanced_accuracy"}, train.main)
    bal = json.loads((ts / "metrics.json").read_text())["fold_score"]
    assert acc == pytest.approx(5 / 6)
    assert bal == pytest.approx(0.5)


def test_train_step_unknown_metric_fails_loud_on_foldless_ship_path(tmp_path, monkeypatch):
    """Eager validation (metis#59): the foldless ship refit never scores, so without the
    eager resolve_scorer at entry an unknown metric would be SILENTLY accepted there."""
    run = tmp_path / "runs" / "r-badmetric"
    _save_skewed_dataset(run)
    with pytest.raises(ValueError, match="balanced_accuracy"):
        _run_step(monkeypatch, run, "train",
                  {"dataset": "skewed", "model": "logreg", "metric": "auc"}, train.main)


def _save_decide_dataset(run_dir):
    """The 40-row 30/10 decide frame (metis#60) as a captured upstream artifact — one weak
    informative feature so probabilities vary by row and tuned offsets flip boundary rows."""
    import numpy as np

    from metis.dataset import Dataset
    from metis.schema import Schema

    rng = np.random.default_rng(0)
    x = np.concatenate([rng.normal(0.0, 1.0, 30), rng.normal(1.5, 1.0, 10)])
    df = pd.DataFrame({"id": range(40), "x": x, "y": [0] * 30 + [1] * 10})
    schema = Schema(columns={"id": "id", "x": "feature", "y": "target"},
                    dtypes={"id": "int64", "x": "float64", "y": "int64"})
    io.save_dataset(Dataset(schema=schema, train=df, test=None), str(run_dir / "decide" / "dataset"))
    return df


def _decide_folds(run_dir, df):
    from metis.split import cv_folds

    folds = cv_folds(df[["y"]], 2, 42, stratify_col="y")
    split_dir = run_dir / "split"
    split_dir.mkdir(parents=True, exist_ok=True)
    (split_dir / "folds.json").write_text(json.dumps(folds))
    return folds


def test_train_step_decide_offsets_per_fold(tmp_path, monkeypatch):
    """with.decide reaches the per-fold scorer; the tuned score can't fall far below argmax
    (the no-op grid point bounds the tuning slice; assessment noise gets the tolerance)."""
    run = tmp_path / "runs" / "r-decide"
    df = _save_decide_dataset(run)
    _decide_folds(run, df)
    base = {"dataset": "decide", "folds": "split", "model": "logreg",
            "metric": "balanced_accuracy", "_fold": {"idx": 0}}
    ts = _run_step(monkeypatch, run, "t-argmax", base, train.main)
    argmax_score = json.loads((ts / "metrics.json").read_text())["fold_score"]
    ts = _run_step(monkeypatch, run, "t-offsets",
                   {**base, "decide": {"offsets": {"holdout": 0.2}}}, train.main)
    tuned_score = json.loads((ts / "metrics.json").read_text())["fold_score"]
    assert tuned_score >= argmax_score - 0.1


def test_train_step_ship_persists_offsets_json_only_under_offsets(tmp_path, monkeypatch):
    run = tmp_path / "runs" / "r-ship"
    _save_decide_dataset(run)
    ts = _run_step(monkeypatch, run, "t-off",
                   {"dataset": "decide", "model": "logreg", "metric": "balanced_accuracy",
                    "decide": {"offsets": {"holdout": 0.2}}}, train.main)
    assert (ts / "model.pkl").exists()
    payload = json.loads((ts / "offsets.json").read_text())
    assert payload["rule"] == "offsets" and len(payload["offsets"]) == len(payload["classes"])
    ts2 = _run_step(monkeypatch, run, "t-arg",
                    {"dataset": "decide", "model": "logreg"}, train.main)
    assert not (ts2 / "offsets.json").exists()      # absence = argmax (compat)


def test_train_step_malformed_decide_refuses_eagerly_foldless(tmp_path, monkeypatch):
    run = tmp_path / "runs" / "r-bad"
    _save_decide_dataset(run)
    with pytest.raises(ValueError, match="decide"):
        _run_step(monkeypatch, run, "t-bad",
                  {"dataset": "decide", "model": "logreg",
                   "decide": {"offsets": {"holdout": 0.9}}}, train.main)
    assert not (run / "t-bad" / "model.pkl").exists()  # refused before any fit


def test_predict_probabilities_and_offsets_application(tmp_path, monkeypatch):
    import numpy as np

    from metis.model import apply_offsets

    run = tmp_path / "runs" / "r-pred"
    _save_decide_dataset(run)
    # offsets chain: ship train (persists offsets.json) -> predict applies it
    _run_step(monkeypatch, run, "train",
              {"dataset": "decide", "model": "logreg", "metric": "balanced_accuracy",
               "decide": {"offsets": {"holdout": 0.2}}}, train.main)
    ps = _run_step(monkeypatch, run, "predict", {"dataset": "decide", "model": "train"}, predict.main)
    proba = pd.read_csv(ps / "probabilities.csv")
    assert list(proba.columns) == ["id", "proba_0", "proba_1"] and len(proba) == 40
    assert np.allclose(proba[["proba_0", "proba_1"]].sum(axis=1), 1.0)
    payload = json.loads((run / "train" / "offsets.json").read_text())
    # Anti-vacuity (close-review Important 2, the plan's dropped flip-check): all-zero
    # offsets would make the expectation below plain argmax — assert tuning actually tilted
    # the decision on this frame (deterministic under the fixed seed).
    assert any(o != 0 for o in payload["offsets"])
    preds = pd.read_csv(ps / "predictions.csv")
    expect = np.array(payload["classes"])[
        apply_offsets(proba[["proba_0", "proba_1"]].to_numpy(), np.array(payload["offsets"]))]
    assert np.array_equal(preds["prediction"].to_numpy(), expect)
    assert json.loads((ps / "metrics.json").read_text())["has_offsets"] == 1.0

    # compat anchor: argmax chain -> predictions.csv is the plain argmax labels
    _run_step(monkeypatch, run, "train2", {"dataset": "decide", "model": "logreg"}, train.main)
    ps2 = _run_step(monkeypatch, run, "predict2", {"dataset": "decide", "model": "train2"}, predict.main)
    preds2 = pd.read_csv(ps2 / "predictions.csv")
    proba2 = pd.read_csv(ps2 / "probabilities.csv")
    assert np.array_equal(preds2["prediction"].to_numpy(),
                          proba2[["proba_0", "proba_1"]].to_numpy().argmax(axis=1))
    assert json.loads((ps2 / "metrics.json").read_text())["has_offsets"] == 0.0


def test_predict_offsets_class_mismatch_fails_loud(tmp_path, monkeypatch):
    """The classes validation is the honesty guarantee for applying a persisted decision
    rule (close-review Important 3): a wrong-model/wrong-offsets pairing must refuse, not
    silently produce garbage labels."""
    run = tmp_path / "runs" / "r-mismatch"
    _save_decide_dataset(run)
    _run_step(monkeypatch, run, "train",
              {"dataset": "decide", "model": "logreg", "metric": "balanced_accuracy",
               "decide": {"offsets": {"holdout": 0.2}}}, train.main)
    payload = json.loads((run / "train" / "offsets.json").read_text())
    payload["classes"] = list(reversed(payload["classes"]))  # mangle the pairing
    (run / "train" / "offsets.json").write_text(json.dumps(payload))
    with pytest.raises(ValueError, match="classes"):
        _run_step(monkeypatch, run, "predict", {"dataset": "decide", "model": "train"}, predict.main)


def _save_reg_dataset(exp_dir):
    """A tiny y = 3x + 0.1 regression dataset (id, feature x, continuous target y) under
    exp_dir/reg/ — the fixture for the regression ship path (metis#36 M1)."""
    d = exp_dir / "reg"
    d.mkdir(parents=True, exist_ok=True)
    n = 40
    xs = [i / n for i in range(n)]
    df = pd.DataFrame({"id": list(range(n)), "x": xs, "y": [3.0 * v + 0.1 for v in xs]})
    (d / "schema.json").write_text(json.dumps({
        "columns": {"id": "id", "x": "feature", "y": "target"},
        "dtypes": {"id": "int64", "x": "float64", "y": "float64"}}))
    df.iloc[:30].to_csv(d / "train.csv", index=False)
    df.iloc[30:].to_csv(d / "test.csv", index=False)


def test_predict_regressor_emits_continuous_no_proba(tmp_path, monkeypatch):
    """metis#36 M1 (the rogii ship path): a regressor has no predict_proba/classes_, so the
    predict step must branch on that — emit continuous predictions.csv and NO probabilities.csv.
    Before the branch, predict.main called predict_proba unconditionally and crashed the whole
    rogii submission flow (AttributeError). The regression fold path (train's cv_score) is already
    covered in test_model; this pins the STEP entrypoint."""
    exp = tmp_path / "exp"
    _save_reg_dataset(exp)
    run = tmp_path / "runs" / "r-reg"
    ts = _run_step(monkeypatch, run, "train",
                   {"dataset": "reg", "model": {"rf_reg": {"n_estimators": 8, "max_depth": 3}},
                    "metric": "rmse"}, train.main, exp_dir=exp)
    assert (ts / "model.pkl").exists()

    ps = _run_step(monkeypatch, run, "predict",
                   {"dataset": "reg", "model": "train"}, predict.main, exp_dir=exp)
    preds = pd.read_csv(ps / "predictions.csv")
    assert list(preds.columns) == ["id", "prediction"]
    assert len(preds) == 10  # the 10 test rows
    assert preds["prediction"].dtype.kind == "f"  # continuous, not int class codes
    # a regressor has no decision layer → no probabilities.csv, no offsets
    assert not (ps / "probabilities.csv").exists()
    assert json.loads((ps / "metrics.json").read_text())["has_offsets"] == 0.0
