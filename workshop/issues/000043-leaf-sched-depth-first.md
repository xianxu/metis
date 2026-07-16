---
id: 000043
status: working
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-16
estimate_hours:
started: 2026-07-16T12:57:07-07:00
---

# leaf scheduler: depth-first run priority so cold-cache sweeps reach trains early

## Problem

On a COLD cache, a nested sweep's ~2k run closures all start concurrently and the shared leaf
semaphore serves steps roughly FIFO — so a run's `features` queues behind every other run's
`cv-split`, and execution degenerates into phase WAVES: all cv-splits, then all features, then
all trains (observed on the kbench#9 decision run 2026-07-14: 2,160/2,160 cv-splits done,
1,315 features, 0 trains after ~10 min — looked like a hang; zero records complete until wave 3,
so run-level progress reads as frozen). On a warm cache the waves collapse and trains appear
early, which is why this wasn't seen before. Depth-first (finish admitted runs before admitting
new ones) reaches complete records ~immediately, makes progress meaningful, and bounds
in-flight run state; it also pairs with #38's moving-average runs/sec board (wave 1-2 have zero
"runs/sec" by construction).

## Spec

(at claim) Options: bound the number of ADMITTED run closures (e.g. 2×parallel) with a run-level
semaphore above the leaf semaphore (simplest — preserves leaf fairness within admitted runs);
or a priority leaf queue keyed by run progress. Keep byte-determinism of artifacts/ledger
(ordering already normalized by sortPointRuns).

## Done when

-

## Plan

- [ ]

## Log

### 2026-07-14
