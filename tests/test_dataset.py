"""Pure tests for metis.dataset — feature/target selection (no IO)."""

import pandas as pd
import pytest

from metis.dataset import Dataset
from metis.schema import Schema


def _ds() -> Dataset:
    schema = Schema(
        columns={"id": "id", "f0": "feature", "f1": "feature", "target": "target"},
        dtypes={},
    )
    train = pd.DataFrame({"id": [1, 2], "f0": [0.1, 0.2], "f1": [1.0, 2.0], "target": [0, 1]})
    return Dataset(schema=schema, train=train)


def test_X_selects_features_in_schema_order():
    ds = _ds()
    X = ds.X(ds.train)
    assert list(X.columns) == ["f0", "f1"]  # id and target excluded
    assert X.shape == (2, 2)


def test_y_selects_target():
    ds = _ds()
    assert list(ds.y(ds.train)) == [0, 1]


def test_y_without_target_raises():
    schema = Schema(columns={"id": "id", "f0": "feature"}, dtypes={})
    ds = Dataset(schema=schema, train=pd.DataFrame({"id": [1], "f0": [0.1]}))
    with pytest.raises(ValueError, match="no target"):
        ds.y(ds.train)
