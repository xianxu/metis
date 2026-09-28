---
id: '000007'
status: done
started: 2026-07-05T17:31:17-07:00
created: 2026-07-03
updated: 2026-07-05
estimate_hours: 1.9
actual_hours: 1.59
---

# Sweep runner + grid sampler (propose_next / should_stop abstraction)

## Problem

Given an `experiment-shape` (a config-space), run its points — the L2 execution.
Start simple (grid) but leave a clean seam for smarter exploration later (Bayesian,
early-stopping), without rewriting the sweep loop each time.
