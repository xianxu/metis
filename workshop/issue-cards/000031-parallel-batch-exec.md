---
id: '000031'
status: done
started: 2026-07-13T14:15:42-07:00
created: 2026-07-13
updated: 2026-07-13
estimate_hours: 2.8
actual_hours: N/A
---

# parallel batch executor — concurrent Ask-batch execution in Run (determinism-preserving)

## Problem

`pkg/sampler/Run` executes an `Ask` batch **sequentially** — `for _, p := range batch { s = Tell(s, p,
runPoint(p)) }`. But a grid sweep of **495** (`driver: single`) / **2,475** (`driver: cv`) per-fold runs
is embarrassingly parallel (every point in a grid's single all-at-once batch is independent), yet runs
one subprocess at a time. This is the dominant wall-clock cost of the honest run.
