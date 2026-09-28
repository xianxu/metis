---
id: '000019'
status: done
started: 2026-07-08T12:01:41-07:00
created: 2026-07-07
updated: 2026-07-11
estimate_hours: 3.7
actual_hours: 6.20
---

# selection objectives — 1-SE rule + mean-std (configurable sweeper select rule, not raw cv-max)

## Problem

The sweeper selects by **raw cv-max** (`argmax-mean`), biased toward overfitters (the max over N noisy
configs inflates + favors fragile high-variance fits). There's no way to prefer a *robust* or *simpler*
config. Nested CV (metis#23) *estimates* the consequence but doesn't change *which* config is picked —
the **select rule** is the actual lever.
