---
id: '000020'
status: done
started: 2026-07-12T19:10:02-07:00
created: 2026-07-07
updated: 2026-07-12
estimate_hours: 1.05
actual_hours: 1.62
---

# leakage-safe target features — internal cross-fit (features already per-fold via M1a)

## Problem

A target-based feature (e.g. group-survival, kbench#8) computed on all-train **leaks**: a test-fold
passenger's feature encodes labels of group-mates also in the test fold → inflated cv that won't
reproduce. Even *within* a training fold, using a passenger's own group to score that same passenger
leaks their own label (catastrophic for small groups). So such features need per-fold **and** internal
cross-fitting/shrinkage.
