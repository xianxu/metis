"""metis/cv-split step-type — assign CV folds over a dataset.

Thin entrypoint (ARCH-PURE): metis.io in → pure cv_folds → metis.io out. Reads the
dataset from an experiment-relative path (`with.dataset`), writes folds.json (the
per-row fold assignment, consumed downstream via the upstream-artifact convention)
plus metrics.json{k, n}.

with:
  dataset:  experiment-relative path to a serialized Dataset dir  (required)
  k:        number of folds                                       (required)
  stratify: stratify on the target column if true                (optional)
"""

from __future__ import annotations

import json

from metis import io
from metis.split import cv_folds


def main() -> None:
    ctx = io.step_context()
    w = io.read_with(ctx)
    ds = io.load_dataset(io.exp_path(ctx, w["dataset"]))
    stratify_col = ds.schema.target_col() if w.get("stratify") else None
    folds = cv_folds(ds.train, k=int(w["k"]), seed=ctx.seed, stratify_col=stratify_col)
    with open(io.out_path(ctx, "folds.json"), "w") as f:
        json.dump(folds, f)
    io.write_metrics(ctx, {"k": float(int(w["k"])), "n": float(len(folds))})


if __name__ == "__main__":
    main()
