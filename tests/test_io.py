"""Tests for metis.io Dataset serialization: parquet round-trip + CSV fixture load."""

import os
from pathlib import Path

import pandas as pd
import pytest
from pandas.testing import assert_frame_equal

from metis.dataset import Dataset
from metis.io import StepContext, dataset_dir, load_dataset, save_dataset
from metis.schema import Schema

TOY = Path(__file__).parents[1] / "testdata" / "dataset" / "toy"


def test_dataset_dir_prefers_captured_upstream_else_exp_relative(tmp_path):
    """metis#18: `dataset` is polymorphic — a captured upstream `<run>/<ref>/dataset/` artifact
    wins (the per-fold features→train handoff); otherwise fall back to the exp-relative path
    (a v1 shared dataset OR a bare-token dir like `toy`). Detected by existence, not a name."""
    run = tmp_path / "runs" / "r1"
    exp = tmp_path / "exp"
    ctx = StepContext(step_dir=str(run / "train"), run_dir=str(run), step_id="train",
                      exp_dir=str(exp), seed=1)
    # No captured artifact yet: a bare token resolves against the EXP dir (not the run dir),
    # so a v1 `dataset: toy` still works.
    assert dataset_dir(ctx, "toy") == str(exp / "toy")
    # An exp-relative PATH also resolves against the exp dir.
    assert dataset_dir(ctx, "../data/x") == os.path.normpath(str(exp / ".." / "data" / "x"))
    # Once `features` has written its captured `dataset/` artifact, the same bare-token ref
    # resolves to the UPSTREAM artifact — the fold handoff.
    (run / "features" / "dataset").mkdir(parents=True)
    assert dataset_dir(ctx, "features") == str(run / "features" / "dataset")


def test_save_load_parquet_round_trip(tmp_path):
    schema = Schema(
        columns={"id": "id", "f0": "feature", "target": "target"},
        dtypes={"id": "int64", "f0": "float64", "target": "int64"},
    )
    train = pd.DataFrame({"id": [1, 2], "f0": [0.5, 1.5], "target": [0, 1]})
    test = pd.DataFrame({"id": [3], "f0": [2.5]})
    save_dataset(Dataset(schema=schema, train=train, test=test), str(tmp_path))

    loaded = load_dataset(str(tmp_path))
    assert loaded.schema == schema
    assert_frame_equal(loaded.train, train)
    assert_frame_equal(loaded.test, test)


def test_load_csv_fixture():
    ds = load_dataset(str(TOY))
    assert ds.train.shape == (60, 6)
    assert ds.test.shape == (20, 5)  # test omits the target column
    assert ds.schema.feature_cols() == ["f0", "f1", "f2", "f3"]
    assert ds.schema.target_col() == "target"
    assert "target" not in ds.test.columns


def test_load_missing_train_raises(tmp_path):
    (tmp_path / "schema.json").write_text('{"columns": {"f0": "feature"}}')
    with pytest.raises(FileNotFoundError, match="no train"):
        load_dataset(str(tmp_path))


def test_source_role_round_trip_object_dtype_with_nan(tmp_path):
    """metis#35: `source` columns may hold strings/NaN (the ROLES contract). Pin the
    parquet round-trip where that contract meets IO: role + dtype survive; a NaN in an
    object column comes back as None (parquet's null) — None/NaN equivalence is the
    contract consumers must honor (isna-safe ops, never `== np.nan`)."""
    import numpy as np
    schema = Schema(
        columns={"id": "id", "f0": "feature", "target": "target", "Ticket": "source"},
        dtypes={"id": "int64", "f0": "float64", "target": "int64", "Ticket": "object"},
    )
    train = pd.DataFrame({"id": [1, 2, 3], "f0": [0.5, 1.5, 2.5], "target": [0, 1, 0],
                          "Ticket": ["A/5 21171", np.nan, "A/5 21171"]})
    test = pd.DataFrame({"id": [4], "f0": [3.5], "Ticket": [np.nan]})
    save_dataset(Dataset(schema=schema, train=train, test=test), str(tmp_path))

    loaded = load_dataset(str(tmp_path))
    assert loaded.schema == schema                       # role + dtype dict survive
    assert loaded.schema.columns["Ticket"] == "source"
    assert loaded.schema.feature_cols() == ["f0"]        # source never a model input
    assert loaded.train["Ticket"].tolist()[0] == "A/5 21171"
    assert loaded.train["Ticket"].isna().tolist() == [False, True, False]  # null survives (as None)
    assert loaded.test["Ticket"].isna().all()
