---
type: experiment
id: valid-baseline
seed: 42
status: active
steps:
  - id: prep
    uses: metis/cv-split
    with: {k: 5}
  - id: train
    uses: metis/train
    needs: [prep]
    with: {model: logreg}
---
# valid-baseline

A generic 2-step fixture proving the schema accepts a well-formed experiment.
Platform-independent (no competition) — the titanic instance lives in kbench.

## Runs
