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


def test_step_context_requires_env(monkeypatch):
    for v in ("METIS_STEP_DIR", "METIS_RUN_DIR", "METIS_STEP_ID", "METIS_EXP_DIR", "METIS_SEED"):
        monkeypatch.delenv(v, raising=False)
    with pytest.raises(RuntimeError, match="not set"):
        io.step_context()
