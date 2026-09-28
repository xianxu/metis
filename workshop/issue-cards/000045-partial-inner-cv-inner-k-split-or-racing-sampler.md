---
id: '000045'
status: done
started: 2026-07-17T23:24:31-07:00
created: 2026-07-15
updated: 2026-07-17
estimate_hours: 0.86
actual_hours: 1.2
---

# partial inner CV — split inner_k from outer k, and/or an adaptive racing sampler

## Problem

There is no way to run the inner CV partially: every config always runs the FULL inner k
folds inside every outer fold. metis#42's `--sample m` / `--fast` sample the **outer**
folds only — the inner level has no cost knob at all. On the decision grid
(`titanic-sweep.md`: 10 outer × 72 configs × 10 inner = 7,200 leaf folds; still 2,160 with
`--sample 3`) the inner sweep is where nearly all the compute goes, and most of it is spent
finishing full 10-fold CVs on configs that are clearly losing after 3 folds.

Two design facts (from the 2026-07-15 T2 session's Q&A — filed verbatim per operator):

1. **Inner k and outer k are the same knob today.** The outer loop reuses
   `sweeper.resample.cv.k` (`runShapeSweep`: `runFolds = k`), which is why the sweep is
   10×…×10. You can't even declare inner k=5 with outer k=10 right now — splitting them
   would be the cheapest "cheaper inner CV" lever.
2. **The principled version is already designed but unbuilt**: the Sampler ask/tell
   feedback edge exists precisely for adaptive inner sampling (racing /
   successive-halving — kill a config after 3 bad folds instead of running all 10). All
   production samplers are static one-batch; an adaptive `Ask` would be the FIRST real use
   of the feedback loop, and metis#30's `SizeBudget`/`SizeUnknown` SizeHint kinds were
   built anticipating exactly that display case (`k/≤n`, `k/?` in the progress line/board).
