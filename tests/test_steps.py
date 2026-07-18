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


def _run_step(monkeypatch, run_dir, step_id, with_cfg, main_fn, seed=42):
    step_dir = run_dir / step_id
    step_dir.mkdir(parents=True, exist_ok=True)
    (step_dir / "with.json").write_text(json.dumps(with_cfg))
    monkeypatch.setenv("METIS_STEP_DIR", str(step_dir))
    monkeypatch.setenv("METIS_RUN_DIR", str(run_dir))
    monkeypatch.setenv("METIS_STEP_ID", step_id)
    monkeypatch.setenv("METIS_EXP_DIR", str(TOY_PARENT))
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
