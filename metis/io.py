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

import pandas as pd

from metis.dataset import Dataset
from metis.schema import Schema

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


# ── JSON helpers ─────────────────────────────────────────────────────────────


def _read_json(path: str) -> dict:
    with open(path) as f:
        return json.load(f)


def _write_json(path: str, obj) -> None:
    with open(path, "w") as f:
        json.dump(obj, f, indent=2, sort_keys=True)
        f.write("\n")
