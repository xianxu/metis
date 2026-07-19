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
  model:   id of the upstream train step (reads its model.pkl; if     (required)
           offsets.json sits beside it — a metis#60 tuned decision
           rule — it is validated (classes order) and applied).
Outputs: predictions.csv + probabilities.csv (metis#60: ALWAYS emitted; columns
`proba_<class-label>` in model.classes_ order — the label IS the suffix) +
metrics.json{n_predictions, has_offsets}.
"""

from __future__ import annotations

import json
import os
import pickle

import numpy as np
import pandas as pd

from metis import io
from metis.model import apply_offsets, predict, predict_proba


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
    proba = predict_proba(model, ds.X(frame))

    # metis#60: offsets.json beside the upstream model = a tuned decision rule; validate the
    # class order it was tuned against (that's why classes is persisted), loud on mismatch.
    offsets_path = io.upstream_path(ctx, w["model"], "offsets.json")
    has_offsets = os.path.exists(offsets_path)
    if has_offsets:
        with open(offsets_path) as f:
            payload = json.load(f)
        if [int(c) for c in payload["classes"]] != [int(c) for c in model.classes_]:
            raise ValueError(
                f"offsets.json classes {payload['classes']} != model.classes_ "
                f"{list(model.classes_)} — offsets were tuned against a different class order")
        preds = model.classes_[apply_offsets(proba, np.asarray(payload["offsets"]))]
    else:
        preds = predict(model, ds.X(frame))

    out = pd.DataFrame()
    id_col = ds.schema.id_col()
    if id_col is not None:
        out[id_col] = frame[id_col].to_numpy()
    out["prediction"] = preds
    out.to_csv(io.out_path(ctx, "predictions.csv"), index=False)

    # metis#60: probabilities are ALWAYS emitted (blend/diagnostics material). Column suffix
    # IS the class label from model.classes_ (not a positional index) — cross-run averaging
    # and offsets application key on it.
    pout = pd.DataFrame()
    if id_col is not None:
        pout[id_col] = frame[id_col].to_numpy()
    for j, c in enumerate(model.classes_):
        pout[f"proba_{c}"] = proba[:, j]
    pout.to_csv(io.out_path(ctx, "probabilities.csv"), index=False)
    io.write_metrics(ctx, {"n_predictions": float(len(out)), "has_offsets": float(has_offsets)})


if __name__ == "__main__":
    main()
