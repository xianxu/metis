---
id: 000063
status: open
created: 2026-07-18
updated: 2026-07-18
estimate_hours:
github_issue:
---

# per-row sample weights: consume the dormant schema weight role

## Problem

**PARKED BY DESIGN — file-when-demanded; the design is settled, the demand isn't here yet.**
Distribution reweighting currently exists only as the per-class shortcut (`class_weight`
model hyperparam, loss-space) and — once metis#60 lands — the per-class decision rule
(`decide` offsets, decision-space). There is no way to weight individual ROWS (importance
weighting), though the substrate anticipated it: `metis.schema`'s `weight` role has been
dormant since metis#1 — no step emits it, `train` never consumes it. The concrete demand
that will activate this: the source-dataset extension idea (arena2 discussions 2026-07-18/19
— appending the competition's inspiration dataset, whose distribution differs; importance
weights are the principled treatment of "similar but shifted" auxiliary rows).
