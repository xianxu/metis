---
type: experiment
id: invalid-cycle
seed: 42
status: active
steps:
  - id: a
    uses: metis/step-a
    needs: [b]
  - id: b
    uses: metis/step-b
    needs: [a]
---
# invalid — steps `a` and `b` need each other (a cycle).

Structurally valid (cue vet accepts it) but SEMANTICALLY invalid: the pure Go
`Validate`/`TopoSort` must reject the cycle. This is exactly the SHAPE-vs-SEMANTICS
split M1 deferred to M2.
