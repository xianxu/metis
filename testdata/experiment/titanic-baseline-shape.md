---
type: experiment-shape
id: titanic-baseline-shape
competition: titanic
seed: 42
status: active
data:
  - id: adapt
    uses: titanic/adapt
    with:
      out: ../data/titanic
pipeline:
  - id: features
    uses: titanic/features
    needs: [adapt]
    with:
      dataset: adapt
      features: {$any: [[], [title], [title, family]]}
  - id: train
    uses: metis/train
    needs: [features]
    with:
      model:
        $any:
          logreg: {C: {$any: [0.1, 1, 10]}}
          rf: {n_estimators: {$any: [100, 300]}, max_depth: {$any: [4, 8]}}
ship:
  - id: predict
    uses: metis/predict
    needs: [train]
sweeper:
  sampler: grid
  resample: {cv: {k: 5, stratify: true}}
  objective: {metric: accuracy, direction: maximize, select: argmax-mean}
driver:
  single: {}
---

# titanic-baseline-shape

The metis#18 v2 worked example: a three-phase shape (`data │ pipeline │ ship`) with a
black-box sweeper (grid over configs, inner 5-fold CV, argmax-mean select) and
`driver: single`. The pipeline's `features × model` space expands to
`features(3) × [logreg:C(3) + rf:(2×2)=4] = 3 × 7 = 21` configs, each scored over 5 folds.
The partition is materialized by the engine from `sweeper.resample.cv` (no `cv-split` step).
