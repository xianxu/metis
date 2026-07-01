---
type: experiment
id: invalid-bad-status
seed: 42
status: running
steps:
  - id: a
    uses: metis/cv-split
---
# invalid fixture — `status: running` is not in the enum, must be REJECTED by cue vet.
