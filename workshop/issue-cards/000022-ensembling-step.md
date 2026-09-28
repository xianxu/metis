---
id: '000022'
status: punt
created: 2026-07-07
updated: 2026-07-16
---

# ensembling / stacking step-type — blend logreg + rf + gbm

## Problem

The workbench trains one model per run; it can't **combine** models. Top Titanic (and most tabular)
solutions ensemble — blend/stack logreg + rf + gbm — for a real accuracy lift. This is a new workbench
primitive, not a Titanic hack.
