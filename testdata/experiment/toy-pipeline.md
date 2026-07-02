---
type: experiment
id: toy-pipeline
seed: 42
status: active
steps:
  - id: split
    uses: metis/cv-split
    with: {dataset: ../dataset/toy, k: 3, stratify: true}
  - id: train
    uses: metis/train
    needs: [split]
    with: {dataset: ../dataset/toy, folds: split, model: logreg}
  - id: predict
    uses: metis/predict
    needs: [train]
    with: {dataset: ../dataset/toy, model: train}
---
# toy-pipeline — metis M3 end-to-end walking skeleton

A generic, platform-independent three-step pipeline that exercises the full
Go-runner → uv/Python data-plane thread: cross-validate folds, train a logreg
model (recording its CV score), then predict on the held-out test rows. The
dataset is referenced experiment-relative (`../dataset/toy`, resolved via
`METIS_EXP_DIR`); `folds` and `model` flow between steps via the upstream-artifact
convention (`$METIS_RUN_DIR/<step-id>/`). NOT titanic — the real Kaggle thread is
kbench#1; this is metis proving its own data plane walks.

## Runs
