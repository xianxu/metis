"""metis.io — the thin IO layer for the data plane (metis#1 M3).

Two responsibilities, both pure-adjacent glue (ARCH-PURE — no numeric logic here):

  1. Dataset serialization (load_dataset / save_dataset) — schema.json plus
     train/test tables. Datasets are written as parquet (the efficient canonical
     form); committed fixtures may be CSV for git-legibility, so the loader
     accepts either.
  2. The M2 step contract (added in Task 5) — reading with.json + the METIS_*
     env, resolving upstream artifacts, writing metrics.json. This is the SINGLE
     Python encoding of the contract the Go runner (cmd/metis/exec.go) speaks
     (ARCH-DRY); every step-type goes through it.
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass

import pandas as pd

from metis.dataset import Dataset
from metis.schema import Schema

# ── Read confinement (metis#23 nested-CV, L2 chokepoint) ─────────────────────
# When METIS_READ_ROOT is set (an outer-fold sweep runs sealed on its analysis
# subset dir), every EXP-RELATIVE data read must resolve under that root — else a
# loud error. Asserted at exp_path (the base-dataset resolver); upstream run-dir
# handoffs go through dataset_dir's upstream branch (not exp_path) and stay
# unconfined, so a legitimate features→train handoff is never flagged.


def within_root(path: str, root: str) -> bool:
    """True iff `path` resolves under `root` (sep-aware; no string-prefix collision)."""
    ap = os.path.abspath(path)
    ar = os.path.abspath(root)
    return ap == ar or ap.startswith(ar + os.sep)


def assert_within_read_root(path: str) -> None:
    """Refuse a data read outside METIS_READ_ROOT (metis#23 confinement). No-op when
    the var is unset (the flat/single path + the outer scoring run are unconfined)."""
    root = os.environ.get("METIS_READ_ROOT")
    if root and not within_root(path, root):
        raise RuntimeError(
            f"read confinement (metis#23): {path!r} is outside METIS_READ_ROOT {root!r} "
            f"— an outer-fold sweep must not read outside its analysis root (leakage)"
        )


# ── Dataset serialization ────────────────────────────────────────────────────


def load_dataset(path: str) -> Dataset:
    """Load a Dataset from a directory holding schema.json + train/test tables.

    A table is read as parquet if <stem>.parquet exists, else CSV (<stem>.csv);
    test is optional (a Kaggle-style test set typically has no target column).
    """
    schema = Schema.from_dict(_read_json(os.path.join(path, "schema.json")))
    train = _read_table(path, "train")
    if train is None:
        raise FileNotFoundError(f"no train.parquet or train.csv in {path}")
    test = _read_table(path, "test")
    return Dataset(schema=schema, train=train, test=test, provenance={"path": os.path.abspath(path)})


def save_dataset(ds: Dataset, path: str) -> None:
    """Write a Dataset as schema.json + train.parquet (+ test.parquet). Parquet is
    the canonical on-disk form for pipeline-produced datasets."""
    os.makedirs(path, exist_ok=True)
    _write_json(os.path.join(path, "schema.json"), ds.schema.to_dict())
    ds.train.to_parquet(os.path.join(path, "train.parquet"), index=False)
    if ds.test is not None:
        ds.test.to_parquet(os.path.join(path, "test.parquet"), index=False)


def _read_table(path: str, stem: str) -> pd.DataFrame | None:
    parquet = os.path.join(path, stem + ".parquet")
    csv = os.path.join(path, stem + ".csv")
    if os.path.exists(parquet):
        return pd.read_parquet(parquet)
    if os.path.exists(csv):
        return pd.read_csv(csv)
    return None


# ── The M2 step contract (single Python encoding — ARCH-DRY) ─────────────────
# The Go runner (cmd/metis/exec.go) speaks this contract; this is the ONE place
# the Python side decodes it, so every step-type stays consistent. A step runs
# with cwd == its step dir and these env vars set by the runner:
#   METIS_STEP_DIR  absolute step dir (== cwd); where with.json lands + outputs go
#   METIS_RUN_DIR   absolute run dir; upstream steps' outputs live at <run>/<id>/
#   METIS_STEP_ID   this step's id
#   METIS_EXP_DIR   absolute experiment dir; anchor for experiment-relative inputs
#                   (M3 addition — the run dir is ephemeral, the exp dir is stable)
#   METIS_SEED      the experiment's seed, so steps are reproducible without
#                   duplicating the seed into every step's `with` (M3 addition)


@dataclass(frozen=True)
class StepContext:
    step_dir: str
    run_dir: str
    step_id: str
    exp_dir: str
    seed: int


def step_context() -> StepContext:
    """Read the step contract env the runner injected. Raises if a required var is
    missing (a step must be launched by `metis run`, not run bare)."""
    return StepContext(
        step_dir=_require_env("METIS_STEP_DIR"),
        run_dir=_require_env("METIS_RUN_DIR"),
        step_id=_require_env("METIS_STEP_ID"),
        exp_dir=_require_env("METIS_EXP_DIR"),
        seed=int(_require_env("METIS_SEED")),
    )


def read_with(ctx: StepContext) -> dict:
    """The step's `with` config, written by the runner as with.json in the step dir."""
    return _read_json(os.path.join(ctx.step_dir, "with.json"))


def exp_path(ctx: StepContext, rel: str) -> str:
    """Resolve an experiment-relative path (e.g. a committed dataset dir)."""
    return os.path.normpath(os.path.join(ctx.exp_dir, rel))


def upstream_path(ctx: StepContext, step_id: str, filename: str) -> str:
    """Path to an output file an upstream step wrote: <run_dir>/<step_id>/<filename>."""
    return os.path.join(ctx.run_dir, step_id, filename)


def dataset_dir(ctx: StepContext, ref: str) -> str:
    """Resolve a `dataset` reference to a directory load_dataset can read (metis#18).

    `dataset` is polymorphic: it may be an UPSTREAM STEP id whose CAPTURED `dataset/` artifact
    this step reads (the per-fold features→train handoff — the enriched dataset must be a
    run-dir artifact, not an exp-relative shared path, so a features HIT + train MISS can't
    read a different fold's data), OR an EXPERIMENT-RELATIVE path to a committed/v1-shared
    dataset dir. Detect by existence — a captured upstream `<run>/<ref>/dataset/` wins; else
    fall back to the exp-relative path. (Existence, not a name heuristic, so a bare-token
    exp-relative dataset like 'toy' still resolves against the exp dir.)"""
    upstream = upstream_path(ctx, ref, "dataset")
    if os.path.isdir(upstream):
        return upstream
    return exp_path(ctx, ref)


def out_path(ctx: StepContext, filename: str) -> str:
    """Path for an output file this step writes into its own step dir."""
    return os.path.join(ctx.step_dir, filename)


def write_metrics(ctx: StepContext, metrics: dict) -> None:
    """Write the step's metrics.json (flat {name: number}); the runner merges it
    into Run.metrics. This is a reserved contract channel, not an artifact."""
    _write_json(os.path.join(ctx.step_dir, "metrics.json"), metrics)


def _require_env(name: str) -> str:
    val = os.environ.get(name)
    if not val:
        raise RuntimeError(f"{name} not set — a step must be launched by `metis run`")
    return val


# ── JSON helpers ─────────────────────────────────────────────────────────────


def _read_json(path: str) -> dict:
    with open(path) as f:
        return json.load(f)


def _write_json(path: str, obj) -> None:
    with open(path, "w") as f:
        json.dump(obj, f, indent=2, sort_keys=True)
        f.write("\n")
