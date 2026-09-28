---
id: '000059'
status: done
started: 2026-07-18T13:14:30-07:00
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.7
actual_hours: 0.30
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
