"""Contract test for the outer-split step (metis#23 M1) — the structural L1 seal.

outer-split reads the FULL dataset (unconfined) and materializes k analysis_i/
subset dirs (train rows where outer_fold != i) + an outer_folds.json assignment.
Each analysis_i is a valid dataset dir the sealed sweeper runs against, so the
outer-assessment rows are physically absent from selection.
"""

import json
from pathlib import Path

import pytest

from metis import io
from metis.steps import cv_split, outer_split

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


def test_outer_split_materializes_disjoint_analysis_dirs(tmp_path, monkeypatch):
    sd = _run_step(monkeypatch, tmp_path / "runs" / "r", "outer",
                   {"dataset": "toy", "k": 3, "stratify": True}, outer_split.main)

    folds = json.loads((sd / "outer_folds.json").read_text())
    assert len(folds) == 60 and set(folds) == {0, 1, 2}

    for i in range(3):
        adir = sd / f"analysis_{i}"
        assert (adir / "schema.json").exists()
        ds_i = io.load_dataset(str(adir))
        # analysis_i excludes exactly the fold-i (outer-assessment) rows
        assert len(ds_i.train) == 60 - folds.count(i)

    # the three held-out sets partition all rows (disjoint + covering)
    assert sum(folds.count(i) for i in range(3)) == 60

    metrics = json.loads((sd / "metrics.json").read_text())
    assert metrics["k"] == 3 and metrics["n"] == 60


def test_outer_split_analysis_rows_carry_the_right_data(tmp_path, monkeypatch):
    """analysis_i's train must be exactly the non-fold-i rows of the source (positional)."""
    sd = _run_step(monkeypatch, tmp_path / "runs" / "r2", "outer",
                   {"dataset": "toy", "k": 3, "stratify": False}, outer_split.main)
    folds = json.loads((sd / "outer_folds.json").read_text())
    full = io.load_dataset(str(TOY_PARENT / "toy"))
    kept0 = [j for j, f in enumerate(folds) if f != 0]
    a0 = io.load_dataset(str(sd / "analysis_0"))
    assert len(a0.train) == len(kept0)
    # the kept rows match the source's non-fold-0 rows, in order
    assert a0.train.reset_index(drop=True).equals(full.train.iloc[kept0].reset_index(drop=True))


# ── The seal, end-to-end through a real step + the real exp_path chokepoint (Task 1.5) ──


def test_seal_out_of_root_base_dataset_read_is_caught(tmp_path, monkeypatch):
    """A confined step whose base dataset resolves OUTSIDE METIS_READ_ROOT fails
    loudly (the load-bearing seal). Uses cv_split (a real base-dataset reader)."""
    # READ_ROOT is a sibling of the toy dataset → reading 'toy' escapes it.
    monkeypatch.setenv("METIS_READ_ROOT", str(TOY_PARENT / "analysis_0"))
    with pytest.raises(RuntimeError, match="read confinement"):
        _run_step(monkeypatch, tmp_path / "runs" / "r", "split",
                  {"dataset": "toy", "k": 3, "stratify": True}, cv_split.main)


def test_seal_in_root_base_dataset_read_passes(tmp_path, monkeypatch):
    """The same confined step, with READ_ROOT covering the base dataset, succeeds —
    proving the seal confines out-of-root reads without breaking in-root ones."""
    monkeypatch.setenv("METIS_READ_ROOT", str(TOY_PARENT / "toy"))
    sd = _run_step(monkeypatch, tmp_path / "runs" / "r2", "split",
                   {"dataset": "toy", "k": 3, "stratify": True}, cv_split.main)
    folds = json.loads((sd / "folds.json").read_text())
    assert len(folds) == 60  # ran to completion inside the root
