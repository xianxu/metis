"""metis/train step-type — score ONE CV fold (M1a per-fold), or fit-on-all (ship).

Thin entrypoint (ARCH-PURE): metis.io in → pure fold_score/cv_score+train → metis.io
out. Two modes, selected by the engine-injected fold-context:

- **per-fold (metis#18 M1a):** when `with._fold.idx` is present, the resample Sampler
  is driving the fold axis — score ONE assessment fold via `fold_score` and emit
  `fold_score`. The `_fold` term enters the step's Kpre so folds are cache-distinct;
  Done reduces the k fold-scores → (mean, SE), and the ledger keeps the raw rows so
  metis#19's 1-SE select is a free re-reduction.
- **all-rows (v1 / the M1a-5 ship refit):** when there's no `_fold`, compute the
  whole-CV `cv_score` mean AND fit a model on ALL training rows, persisted as
  model.pkl for the downstream predict/submission (ship) steps.

with:
  dataset: a serialized Dataset dir — either an experiment-relative    (required)
           path (v1 shared) OR an upstream step-id whose captured
           `dataset/` artifact this step reads (metis#18 per-fold handoff).
  folds:   id of the upstream cv-split step (reads its folds.json) —   (per-fold)
           REQUIRED per-fold (defines the split); OPTIONAL all-rows
           (a v1 plain experiment → a reference cv_score; the ship omits it).
  model:   a kind string ("logreg" | "rf") OR the $any-map bundle      (required)
           ({"rf": {"n_estimators": 200, "max_depth": 4}}). Parsed by
           metis.model.parse_model_config → (kind, params).
  _fold:   {partition, idx} — engine-injected fold-context; present in  (per-fold)
           the per-fold run, absent for the all-rows ship refit.
Outputs: metrics.json{fold_score} (per-fold) OR model.pkl + metrics.json{cv_score}.
"""

from __future__ import annotations

import json
import pickle

from metis import io
from metis.model import cv_score, fold_score, parse_model_config, train


def main() -> None:
    ctx = io.step_context()
    w = io.read_with(ctx)
    # `dataset` is polymorphic (metis#18): an exp-relative path (v1 shared dataset) OR an
    # upstream step-id whose captured `dataset/` artifact this step reads (the per-fold
    # features→train handoff). io.dataset_dir resolves both.
    ds = io.load_dataset(io.dataset_dir(ctx, w["dataset"]))
    X, y = ds.X(ds.train), ds.y(ds.train)
    # `model` is a kind string ("logreg") OR the $any-map bundle ({"rf": {n_estimators…}}).
    kind, params = parse_model_config(w["model"])

    fold = w.get("_fold")
    if isinstance(fold, dict) and "idx" in fold:
        # per-fold: the engine drives the fold axis; score the one assessment fold. `folds`
        # is REQUIRED here — the fold assignment is what defines the analysis/assessment split.
        score = fold_score(X, y, _load_folds(ctx, w), int(fold["idx"]), kind, ctx.seed, params)
        io.write_metrics(ctx, {"fold_score": score})
        return

    # all-rows (v1 plain / the M1a-5 ship refit): fit a model on ALL rows for predict. The
    # ship refit needs NO CV — the honest (mean, SE) is the sweep's job. `folds` is OPTIONAL:
    # a v1 plain experiment supplies cv-split and gets a reference cv_score; the ship omits it.
    model = train(X, y, kind, ctx.seed, params)
    with open(io.out_path(ctx, "model.pkl"), "wb") as f:
        pickle.dump(model, f)
    metrics: dict[str, float] = {}
    if "folds" in w:
        metrics["cv_score"] = cv_score(X, y, _load_folds(ctx, w), kind, ctx.seed, params)
    io.write_metrics(ctx, metrics)


def _load_folds(ctx: io.StepContext, w: dict) -> list:
    """Read the upstream cv-split step's folds.json (the per-row fold assignment)."""
    with open(io.upstream_path(ctx, w["folds"], "folds.json")) as f:
        return json.load(f)


if __name__ == "__main__":
    main()
