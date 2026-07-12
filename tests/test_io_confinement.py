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


# ── The seal wired into exp_path (Task 1.2) ──────────────────────────────────

from metis.io import StepContext, dataset_dir, exp_path


def _ctx(exp_dir, run_dir):
    return StepContext(step_dir=run_dir, run_dir=run_dir, step_id="s", exp_dir=exp_dir, seed=42)


def test_exp_path_confined_raises_for_out_of_root(tmp_path, monkeypatch):
    # base-dataset read outside METIS_READ_ROOT → caught + named (half a).
    ctx = _ctx(str(tmp_path), str(tmp_path / "runs" / "r"))
    monkeypatch.setenv("METIS_READ_ROOT", str(tmp_path / "analysis_0"))
    with pytest.raises(RuntimeError, match="assessment_0"):
        exp_path(ctx, "assessment_0")  # resolves to <exp>/assessment_0 — outside analysis_0


def test_exp_path_passes_within_root(tmp_path, monkeypatch):
    ctx = _ctx(str(tmp_path), str(tmp_path / "runs" / "r"))
    monkeypatch.setenv("METIS_READ_ROOT", str(tmp_path / "analysis_0"))
    assert exp_path(ctx, "analysis_0/train") == str(tmp_path / "analysis_0" / "train")  # no raise


def test_handoff_read_under_run_dir_not_confined(tmp_path, monkeypatch):
    """C1 regression: a legit upstream handoff (run-dir artifact, a SIBLING of the
    analysis root) must PASS — it goes through dataset_dir's upstream branch, not
    exp_path. No other test covers this; without it the sealed sweep would crash."""
    run_dir = tmp_path / "runs" / "r"
    upstream = run_dir / "features" / "dataset"
    upstream.mkdir(parents=True)
    ctx = _ctx(str(tmp_path), str(run_dir))
    monkeypatch.setenv("METIS_READ_ROOT", str(tmp_path / "analysis_0"))  # unrelated to run_dir
    # dataset_dir resolves the upstream artifact WITHOUT touching exp_path → no confinement.
    assert dataset_dir(ctx, "features") == str(upstream)
