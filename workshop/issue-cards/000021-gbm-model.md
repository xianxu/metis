---
id: '000021'
status: done
started: 2026-07-11T21:50:20-07:00
created: 2026-07-07
updated: 2026-07-12
estimate_hours: 0.6
actual_hours: 0.62
---

# GBM model branch — HistGradientBoosting model step-type

## Problem

The model set is only `logreg` + `rf`. Gradient boosting is usually the strongest model on tabular
data like Titanic, and the workbench can't sweep it.
