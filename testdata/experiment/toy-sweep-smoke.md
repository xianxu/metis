---
type: experiment-shape
id: toy-sweep-smoke
seed: 42
status: active
data:
  - id: data
    uses: test/echo
    with: {out: ../dataset/toy}
pipeline:
  - id: train
    uses: metis/train
    needs: [data]
    with:
      dataset: ../dataset/toy
      model:
        $any:
          logreg: {C: {$any: [0.5, 1.0, 2.0]}}
sweeper:
  sampler: grid
  resample: {cv: {k: 2, stratify: true}}
  objective: {metric: train.fold_score, direction: maximize, select: {argmax-mean: {}}}
---

# toy-sweep-smoke

A credential-free, disposable real-process nested sweep for cold scheduling smoke checks.
