---
id: 000059
status: open
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours:
---

# train-step metric knob: balanced accuracy + class_weight passthrough

## Problem

Arena2's S6E7 scores **balanced accuracy** over a 3-class target skewed 85.9/8.4/5.8
(at-risk/unhealthy/fit), but `metis.model.fold_fit`/`cv_score` hardcode
`sklearn.metrics.accuracy_score` and `make_model` has no `class_weight` passthrough —
so a sweep SELECTS on the wrong objective (a majority-leaning config wins accuracy at
~0.86 while scoring ~0.33 balanced), and the models can't be told to care about the
minority classes. Demand #1 on the arena2 demand list (anticipated at project open,
confirmed by kbench#12 recon 2026-07-18). Gates kbench#12 M2 (the first honest S6E7
submission).

## Spec

- **Metric knob:** `with.metric ∈ {accuracy, balanced_accuracy}` on the `metis/train`
  step-type, default `accuracy` (titanic shapes unchanged, zero re-keying for existing
  cohorts that don't set it). Flows `train.py → model.fold_fit / fold_score / cv_score`
  as a pure parameter; scorer resolved in ONE place (metis.model). NOTE: the shape's
  `objective.metric: train.fold_score` is a ledger NAME, not the scorer — unchanged.
- **class_weight passthrough:** `make_model` accepts `class_weight` in the params dict
  for `rf` and `hist_gbm` (both sklearn estimators support `class_weight="balanced"`);
  swept like any other hyperparam via the `$any`-map bundle. logreg untouched unless free.
- **Loud misuse:** unknown metric string → ValueError naming the closed set (the
  titanic `_SEX` / parse_model_config pattern).
- Pure-core discipline (ARCH-PURE): the scorer choice is data → data; unit-test
  balanced_accuracy on a skewed toy frame (majority-argmax scores 1/n_classes, not 0.86).
- Consumers to touch: `metis/model.py`, `metis/steps/train.py` (docstring `with:` table),
  `atlas/experiment.md` step-type table. kbench's s6e7 shapes adopt the knob in kbench#12 M2
  (not this issue).

## Done when

-

## Plan

- [ ]

## Log

### 2026-07-18
