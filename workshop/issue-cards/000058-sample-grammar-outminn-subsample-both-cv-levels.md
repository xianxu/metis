---
id: '000058'
status: done
started: 2026-07-18T12:06:53-07:00
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 1.2
actual_hours: 0.50
---

# sample grammar outMinN: subsample both CV levels

## Problem

Arena2 (Playground S6E7, 690k train rows ≈ 100× titanic) makes iteration cost real: even the
7-config starter grid is `outer × configs × inner_k` leaf fits on ~620k-row analysis frames.
`--sample m` subsamples only the OUTER level; the inner per-config CV always runs all
`inner_k` (or k) folds. The alternative — editing `inner_k` in the shape — changes the inner
partition itself (a 2-way split shares no fold boundaries with a 5-way), so it re-keys every
leaf and throws iteration spend away. Demand #2 from the arena2 project (operator-proposed
design, 2026-07-18 session): a CLI dial over BOTH levels that keeps the shape's declared
estimand intact and lets iteration runs escalate into decision runs via the cache.
