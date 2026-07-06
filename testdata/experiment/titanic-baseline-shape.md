---
type: experiment-shape
id: titanic-baseline-shape
competition: titanic
seed: 42
status: active
sweep:
  sampler: grid
  objective: {metric: cv_score, direction: maximize}
  range_steps: 6
steps:
  - id: adapt
    uses: titanic/adapt
    with:
      features: {$any: [[], [title], [title, family]]}
  - id: split
    uses: metis/cv-split
    needs: [adapt]
    with: {dataset: adapt, k: 5}
  - id: train
    uses: metis/train
    needs: [split]
    with:
      dataset: adapt
      folds: split
      model:
        $oneof:
          logreg: {C: {$any: [0.1, 1, 10]}}
          rf: {n_estimators: {$any: [100, 300]}, max_depth: {$any: [4, 8]}}
---

# titanic-baseline-shape

The metis#6 worked example: a features × model × hyperparams shape. Expands to
`features(3) × [logreg:C(3) + rf:(2×2)=4] = 3 × 7 = 21` points.
