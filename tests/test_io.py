"""Tests for metis.io Dataset serialization: parquet round-trip + CSV fixture load."""

from pathlib import Path

import pandas as pd
import pytest
from pandas.testing import assert_frame_equal

from metis.dataset import Dataset
from metis.io import load_dataset, save_dataset
from metis.schema import Schema

TOY = Path(__file__).parents[1] / "testdata" / "dataset" / "toy"


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
