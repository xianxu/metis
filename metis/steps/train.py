"""metis/train step-type — cross-validate then fit a model on all training rows.

Thin entrypoint (ARCH-PURE): metis.io in → pure cv_score + train → metis.io out.
Reads the dataset (experiment-relative) and the fold assignment from the upstream
cv-split step, records the CV score, and persists the model fit on all rows.

with:
  dataset: experiment-relative path to a serialized Dataset dir   (required)
  folds:   id of the upstream cv-split step (reads its folds.json) (required)
  model:   a kind string ("logreg" | "rf"), OR metis#6's $oneof    (required)
           bundle carrying the swept hyperparams
           ({"rf": {"n_estimators": 200, "max_depth": 4}}). Parsed by
           metis.model.parse_model_config → (kind, params).
Outputs: model.pkl (artifact) + metrics.json{cv_score}.
"""

from __future__ import annotations

import json
import pickle

from metis import io
from metis.model import cv_score, parse_model_config, train


def main() -> None:
    ctx = io.step_context()
    w = io.read_with(ctx)
    ds = io.load_dataset(io.exp_path(ctx, w["dataset"]))
    with open(io.upstream_path(ctx, w["folds"], "folds.json")) as f:
        folds = json.load(f)

    X, y = ds.X(ds.train), ds.y(ds.train)
    # `model` is a kind string ("logreg") OR metis#6's $oneof bundle ({"rf": {n_estimators…}}).
    kind, params = parse_model_config(w["model"])
    score = cv_score(X, y, folds, kind, ctx.seed, params)
    model = train(X, y, kind, ctx.seed, params)  # final model: fit on ALL training rows

    with open(io.out_path(ctx, "model.pkl"), "wb") as f:
        pickle.dump(model, f)
    io.write_metrics(ctx, {"cv_score": score})


if __name__ == "__main__":
    main()
