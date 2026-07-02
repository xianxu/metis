"""Pure tests for metis.schema — column-role resolution (no IO)."""

import pytest

from metis.schema import Schema


def _toy_schema() -> Schema:
    return Schema(
        columns={"id": "id", "f0": "feature", "f1": "feature", "target": "target"},
        dtypes={"id": "int64", "f0": "float64", "f1": "float64", "target": "int64"},
    )


def test_role_resolution():
    s = _toy_schema()
    assert s.feature_cols() == ["f0", "f1"]  # insertion order, deterministic
    assert s.target_col() == "target"
    assert s.id_col() == "id"
    assert s.weight_col() is None


def test_no_target_returns_none():
    s = Schema(columns={"id": "id", "f0": "feature"}, dtypes={})
    assert s.target_col() is None


def test_from_to_dict_round_trip():
    s = _toy_schema()
    assert Schema.from_dict(s.to_dict()) == s


def test_unknown_role_rejected():
    with pytest.raises(ValueError, match="unknown column role"):
        Schema(columns={"x": "label"}, dtypes={})


def test_multiple_targets_rejected():
    with pytest.raises(ValueError, match="at most one target"):
        Schema(columns={"a": "target", "b": "target"}, dtypes={})
