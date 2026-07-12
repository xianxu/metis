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
          # learning_rate FIXED (not swept) so total-leaves complexity is a clean fixed-ν
          # parsimony axis (metis#21); sweep the tree-count × tree-size capacity knobs.
          hist_gbm: {learning_rate: 0.1, max_iter: {$any: [100, 300]}, max_leaf_nodes: {$any: [15, 31]}}
ship:
  - id: predict
    uses: metis/predict
    needs: [train]
sweeper:
  sampler: grid
  resample: {cv: {k: 5, stratify: true}}
  objective: {metric: train.fold_score, direction: maximize, select: {argmax-mean: {}}}
driver:
  single: {}
---

# titanic-baseline-shape

The metis#18 v2 worked example: a three-phase shape (`data │ pipeline │ ship`) with a
black-box sweeper (grid over configs, inner 5-fold CV, argmax-mean select) and
`driver: single`. The pipeline's `features × model` space expands to
`features(3) × [logreg:C(3) + rf:(2×2)=4 + hist_gbm:(2×2)=4] = 3 × 11 = 33` configs, each scored
over 5 folds (hist_gbm sweeps max_iter × max_leaf_nodes at a fixed learning_rate — metis#21).
The partition is materialized by the engine from `sweeper.resample.cv` (no `cv-split` step).
