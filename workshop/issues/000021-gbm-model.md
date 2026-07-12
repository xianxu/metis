---
id: 000021
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-11
estimate_hours:
started: 2026-07-11T21:50:20-07:00
---

# GBM model branch — HistGradientBoosting model step-type

## Problem

The model set is only `logreg` + `rf`. Gradient boosting is usually the strongest model on tabular
data like Titanic, and the workbench can't sweep it.

## Spec

metis-v2 M4a (project + pensive). Add **HistGradientBoosting** (sklearn, no extra dep; XGBoost/LightGBM
optional later) to `metis/model.py`'s `make_model` + `parse_model_config`, so it's a `$any`-map model
branch: `model: {$any: {hist_gbm: {learning_rate: …, max_iter: …, max_depth: …}}}`. **Independent of
M1** — a clean quick win + a real exercise of the metis#17 tagged-`$any` model set. Known hyperparams
applied, unknowns ignored (the existing forward-compatible contract).

## Done when

- `model: {$any: {hist_gbm: {...}}}` trains + cross-validates + sweeps through the ledger like logreg/rf.
- Unit test on the model factory; the titanic sweep can include a gbm branch.

## Plan

- [ ] Add `hist_gbm` to `make_model` + `parse_model_config`; unit test; add a gbm branch to a titanic sweep shape.

## Log

### 2026-07-07
- Filed as metis-v2 M4a. Independent additive win (startable in parallel with M1). Design in the project/pensive.
