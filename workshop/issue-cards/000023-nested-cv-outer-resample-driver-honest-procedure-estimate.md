---
id: '000023'
status: done
started: 2026-07-12T15:10:37-07:00
created: 2026-07-07
updated: 2026-07-12
estimate_hours: 3.1
actual_hours: 2.75
---

# nested-CV outer resample driver — honest procedure estimate

## Problem

The flat sweeper (metis#18) selects a winner and reports its inner-CV score — but that score is
*optimistic* (the max over N noisy configs; the selection itself overfits). There's no honest estimate
of *the whole tune-then-fit procedure*. That gap is exactly metis-v1's ~0.81 cv → 0.78 public.
