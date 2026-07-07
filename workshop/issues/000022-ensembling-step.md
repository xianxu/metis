---
id: 000022
status: open
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
---

# ensembling / stacking step-type — blend logreg + rf + gbm

## Problem

The workbench trains one model per run; it can't **combine** models. Top Titanic (and most tabular)
solutions ensemble — blend/stack logreg + rf + gbm — for a real accuracy lift. This is a new workbench
primitive, not a Titanic hack.

## Spec

metis-v2 M4b (project + pensive). A new **ensembling/stacking step-type** — takes several upstream
models' out-of-fold predictions and fits a meta-learner (stacking) or averages (blending). Design
questions at claim: does it consume N upstream `train` steps, or sweep-winners from the ledger? How do
out-of-fold predictions flow (ties to the fold-aware seam, metis#18/#20)? Keep it platform-agnostic
(a metis step-type, not titanic-specific).

## Done when

- A stack/blend of ≥2 models trains end-to-end and scores; a hermetic test.
- Expressible as a step in an experiment (composes with the sweep/ledger).

## Plan

- [ ] (spec at claim) decide the ensembling contract (upstream models vs ledger winners; OOF prediction flow); build + test.

## Log

### 2026-07-07
- Filed as metis-v2 M4b. New workbench primitive (combine models). Likely wants metis#18's fold-aware OOF
  predictions. Design in the project/pensive.
