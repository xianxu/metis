"""metis/predict step-type — predict with the trained model, write a submission.

Thin entrypoint (ARCH-PURE): metis.io in → pure predict → metis.io out. Loads the
model from the upstream train step and predicts on the dataset's test rows (falling
back to train rows if there is no test split), writing predictions.csv (id +
prediction) — the submission-shaped output.

with:
  dataset: a serialized Dataset dir — an upstream step-id whose      (required)
           captured `dataset/` artifact this step reads (the ship
           refit's all-rows `features` output) OR an exp-relative
           path (v1). io.dataset_dir resolves both.
  model:   id of the upstream train step (reads its model.pkl)       (required)
Outputs: predictions.csv (artifact) + metrics.json{n_predictions}.
"""

from __future__ import annotations

import pickle

import pandas as pd

from metis import io
from metis.model import predict


def main() -> None:
    ctx = io.step_context()
    w = io.read_with(ctx)
    # `dataset` is polymorphic (metis#18), like train: an upstream step-id whose captured
    # `dataset/` artifact this step reads (the driver:single ship's all-rows `features`
    # output) OR an exp-relative path (v1 plain experiments). io.dataset_dir resolves both.
    ds = io.load_dataset(io.dataset_dir(ctx, w["dataset"]))
    with open(io.upstream_path(ctx, w["model"], "model.pkl"), "rb") as f:
        model = pickle.load(f)

    frame = ds.test if ds.test is not None else ds.train
    preds = predict(model, ds.X(frame))

    out = pd.DataFrame()
    id_col = ds.schema.id_col()
    if id_col is not None:
        out[id_col] = frame[id_col].to_numpy()
    out["prediction"] = preds
    out.to_csv(io.out_path(ctx, "predictions.csv"), index=False)
    io.write_metrics(ctx, {"n_predictions": float(len(out))})


if __name__ == "__main__":
    main()
