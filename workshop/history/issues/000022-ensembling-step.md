---
id: 000022
status: punt
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-16
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

## Revisions

### 2026-07-16 — PUNTED (deferred at triage, operator call)
- **Reason:** the 2026-07-14 reframe made this issue's Titanic payoff the **gated rule+residual**
  combiner (Deotte's 0.84688 architecture) — but on 2026-07-16 the operator **ruled out
  hard-coded-rule submissions on principle** (the workbench's learned intelligence is the point,
  not the leaderboard number). That parks the entire 0.80+ rule-expression tier this issue was
  reframed to serve. Plain stacking/blending (the original spec) was already downgraded by the
  research digest (stacking kernels score below the no-ML WCG rule).
- **Delta:** status open → punt. kbench#11 (the diagnostic that would have priced this build)
  punted with it. Reopen trigger: a future competition where model combination is a live lever
  on its own merits (no rule tier involved).
- Context: metis-v2 rescope (project file, brain repo); done_when was already MET without this.
