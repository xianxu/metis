---
id: '000066'
status: done
started: 2026-07-19T08:32:12-07:00
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 7.6
actual_hours: N/A
---

# adaptive outer-fold scheduling + --auto-stop (incumbent-referenced early stop of losing configs)

## Problem

A full nested-CV run (`--sample out10` on a real competition = ~100 min) commits the whole
budget before showing any honest number — the mean±SE only lands at the very end. Today the
metis#31 parallel executor fans leaves out GLOBALLY (all outer×config×inner leaves scheduled
together, bounded by the semaphore), so no fold "finishes first" and there's no live signal
to act on. Two things the operator wants (arena2 M6 design session, 2026-07-19):

1. **Early partial estimates.** Finish outer fold 0 first, then fold 1, … so a 1-fold →
   2-fold → 3-fold estimate appears live (SE tightening) — the operator can eyeball an
   obvious loser at fold 3 instead of waiting for fold 10.
2. **Auto-stop losers.** After a few folds, if a config is statistically unlikely to beat the
   known incumbent, stop scheduling its remaining folds and reclaim the budget for the rest.

This is the OUTER, incumbent-referenced cousin of metis#54 (racing successive-halving INNER
sampler) — they must not collide (see Non-goals). It also finally delivers the clean per-fold
progress deferred to metis#30.
