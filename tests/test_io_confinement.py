"""Confinement tests for metis.io — the nested-CV (metis#23) read-root chokepoint.

L2 of the sealing: when METIS_READ_ROOT is set (an outer-fold sweep), every
exp-relative (base-dataset) read must resolve under that root, else a loud error.
Upstream run-dir handoffs are NOT confined (they go through dataset_dir's upstream
branch, not exp_path) — the C1 regression the plan-review flagged.
"""

import os

import pytest

from metis.io import assert_within_read_root, within_root


def test_within_root_true_for_child():
    assert within_root("/data/analysis_0/train.parquet", "/data/analysis_0")


def test_within_root_true_for_root_itself():
    assert within_root("/data/analysis_0", "/data/analysis_0")


def test_within_root_false_for_sibling():
    # the outer-assessment sibling must be OUTSIDE the analysis root
    assert not within_root("/data/assessment_0/train.parquet", "/data/analysis_0")


def test_within_root_false_for_prefix_collision():
    # /data/analysis_00 is NOT under /data/analysis_0 (sep-aware, no string-prefix bug)
    assert not within_root("/data/analysis_00/x", "/data/analysis_0")


def test_assert_raises_and_names_the_file(monkeypatch):
    monkeypatch.setenv("METIS_READ_ROOT", "/data/analysis_0")
    with pytest.raises(RuntimeError, match="assessment_0/train.parquet"):
        assert_within_read_root("/data/assessment_0/train.parquet")


def test_assert_passes_within_root(monkeypatch):
    monkeypatch.setenv("METIS_READ_ROOT", "/data/analysis_0")
    assert_within_read_root("/data/analysis_0/train.parquet")  # no raise


def test_assert_noop_when_root_unset(monkeypatch):
    monkeypatch.delenv("METIS_READ_ROOT", raising=False)
    assert_within_read_root("/anywhere/at/all")  # no root set → no confinement
