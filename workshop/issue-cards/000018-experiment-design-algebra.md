---
id: '000018'
status: done
started: 2026-07-07T11:08:31-07:00
created: 2026-07-07
updated: 2026-07-08
estimate_hours: 7.0
actual_hours: N/A
---

# experiment-design algebra M1a — three-phase shape + Sampler fold node (static samplers, per-fold pipeline, driver:single)

## Problem

The sweep treats data-splitting (CV) as an internal detail of `train` and selects by raw cv-max on a
single split → selection-overfitting (the metis-v1 gap: ~0.81 cv → 0.78 public). Resampling and
selection aren't first-class; the workbench can't produce an honest per-config mean/std, and the
structure that nested-CV (#23) and leakage-safe features (#20) need doesn't exist.
