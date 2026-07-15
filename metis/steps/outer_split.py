"""metis/outer-split step-type — the structural L1 seal for nested-CV (metis#23).

Reads the FULL dataset (UNCONFINED — it must see all rows to split them) and
materializes k `analysis_i/` subset dataset dirs: analysis_i holds the train rows
whose outer fold != i (outer-analysis) PLUS the unchanged test frame, so it is a
SHAPE-IDENTICAL stand-in for the declared base (metis#35 — only train rows
differ; any pipeline that runs flat also runs sealed). The outer-assessment rows
(fold == i) are PHYSICALLY ABSENT from any sweep the driver points at analysis_i. Also writes
`outer_folds.json` (the positional fold assignment) so the outer scoring can refit
the winner on outer-analysis and score on outer-assessment (via the fold mask).

Thin entrypoint (ARCH-PURE): metis.io in → pure cv_folds + a row mask → metis.io out.

with:
  dataset:  experiment-relative path to a serialized Dataset dir  (required)
  k:        number of OUTER folds                                 (required)
  stratify: stratify on the target column if true                (optional)
"""

from __future__ import annotations

import json

import numpy as np

from metis import io
from metis.dataset import Dataset
from metis.split import cv_folds


def main() -> None:
    ctx = io.step_context()
    w = io.read_with(ctx)
    # UNCONFINED full-dataset read: outer-split is the one step that legitimately
    # sees every row (it produces the analysis subsets the confinement then guards).
    ds = io.load_dataset(io.exp_path(ctx, w["dataset"]))
    k = int(w["k"])
    stratify_col = ds.schema.target_col() if w.get("stratify") else None
    folds = cv_folds(ds.train, k=k, seed=ctx.seed, stratify_col=stratify_col)

    with open(io.out_path(ctx, "outer_folds.json"), "w") as f:
        json.dump(folds, f)

    fold_arr = np.asarray(folds)
    for i in range(k):
        analysis = ds.train.iloc[np.flatnonzero(fold_arr != i)]
        # metis#35: carry the test frame — analysis_i is a SHAPE-IDENTICAL stand-in
        # for the declared base (only train rows differ), so a both-frames feature
        # (ticket_size over train+test) sees the same test rows sealed as at ship.
        # Seal-neutral: the outer-assessment rows are fold-i TRAIN rows, which never
        # appear in `test` — carrying test exposes no assessment label regardless of
        # whether the test frame itself carries labels.
        io.save_dataset(Dataset(schema=ds.schema, train=analysis, test=ds.test),
                        io.out_path(ctx, f"analysis_{i}"))

    io.write_metrics(ctx, {"k": float(k), "n": float(len(folds))})


if __name__ == "__main__":
    main()
