---
id: '000030'
status: done
started: 2026-07-15T16:07:52-07:00
created: 2026-07-13
updated: 2026-07-15
estimate_hours: 1.63
actual_hours: 1.51
---

# runner progress reporting — SizeHint + progress callback (k/n + live outer-cv)

## Problem

A sweep runs blind. `titanic-sweep.md` is **495** per-fold runs (`driver: single`, 99 configs × 5
folds) and **2,475** for the honest `driver: cv` (× 5 outer) — with no live signal of how far along it
is or what it's finding. The operator wants **`k / n`** (k = points completed, n = total) **plus the
running estimate** (best-so-far / outer-cv). For grid, n is exact; for adaptive samplers (future: bayes,
racing) n may be a budget or genuinely unknown — so n must be allowed to be `?`, with k + the incumbent
still reported.
