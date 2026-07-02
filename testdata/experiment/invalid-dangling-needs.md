---
type: experiment
id: invalid-dangling-needs
seed: 42
status: active
steps:
  - id: train
    uses: metis/train
    needs: [prep]
---
# invalid — step `train` needs `prep`, which is not a step in this experiment.

Structurally valid (cue vet accepts it) but SEMANTICALLY invalid: the pure Go
`Validate` must reject the dangling `needs` reference.
