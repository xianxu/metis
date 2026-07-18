---
id: 000059
status: working
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.7
started: 2026-07-18T13:14:30-07:00
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.1 impl=0.15
item: smaller-go-module   design=0.05 impl=0.1
item: atlas-docs          design=0.0 impl=0.05
item: milestone-review    design=0.0 impl=0.2
design-buffer: 0.15
total: 0.67
```

(Items: pure core (scorer+threading+class_weight)+unit tests · train-step wiring+step tests ·
atlas both bullets · close review. Buffer 0.15: reviewed plan doc.

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.)

## Done when

- `with.metric ∈ {accuracy, balanced_accuracy}` honored on BOTH train paths (per-fold and
  all-rows), default `accuracy`; unknown metric → eager `ValueError` naming the closed set on
  EVERY path, including the foldless ship refit (which never scores).
- Balanced-accuracy behavior proven on a skewed frame at unit AND step level (majority-argmax:
  high accuracy, ~1/n_classes balanced); `cv_score` threading pinned (== mean of per-fold
  balanced fold_scores).
- `class_weight` reaches rf + hist_gbm constructors via the params bundle; defaults unchanged
  (`None`).
- Absent `metric` key → leaf addresses unchanged (existing titanic cohorts un-re-keyed —
  Kpre hashes the resolved With map).
- `train.py` `with:` table + BOTH atlas/experiment.md bullets (model.py + train step) updated.
- `uv run pytest -q` and `go test ./cmd/metis` green.

## Plan

Durable plan: `workshop/plans/000059-metric-knob-plan.md` (fresh-eyes reviewed, findings folded).

- [x] pure core: resolve_scorer + metric threading + class_weight (TDD)
- [x] train step eager validation + step tests (in-test skewed dataset) + atlas both bullets
- [ ] pr → merge → close

## Log

### 2026-07-18

- Implemented inline (main session): 2 commits on-branch. 93 python + full Go suite green.
  Step-level proof: skewed 10/2 constant-feature dataset — accuracy fold_score 5/6, balanced
  0.5; unknown metric refuses eagerly on the foldless ship path. class_weight verified reaching
  both estimators (default None unchanged).
