---
id: 000061
status: open
created: 2026-07-18
updated: 2026-07-18
estimate_hours:
github_issue:
---

# metis debug verb family: feature-importance first

## Problem

The workbench measures and selects but has no microscope: after a sweep there is no way to
ask a materialized run "WHICH features did you actually use?" — needed right now to guide
arena2 M3's interaction-ladder climb (extend the strongest combos, not blind 4-way search),
and generally whenever a rung's value needs explaining rather than just ranking. Operator
design (2026-07-18 session): a `metis debug <sub>` READ-ONLY verb family over materialized
runs/ledgers — inspection, not pipeline steps — starting with feature-importance.

