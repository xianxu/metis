---
id: '000032'
status: done
started: 2026-07-13T15:52:26-07:00
created: 2026-07-13
updated: 2026-07-14
estimate_hours: 4.5
actual_hours: N/A
---

# outer-CV model-family selection — close the loop (nested-CV selects, not just reports)

## Problem

**metis#23's nested CV (`driver: cv`) is a passive reporter — it produces an honest estimate but ships
NO winner, so the outer CV isn't used in any automated fashion.** That wastes exactly the signal we
need. Demonstrated empirically on the real Titanic honest-beat run (kbench#8):

- The sweeper's cross-family pick is inner-CV **argmax-mean** (metis#19's parsimony is intra-family only).
- Inner CV shipped `hist_gbm [title,family]` (mean **0.846**, cx 1500) → **public 0.749** — a ~0.10 overfit
  gap. The rf robust winner `[title…embarked, ticket_survival]` md=4 (mean 0.839, cx 14.3) would generalize
  far better (that family ~0.78 public historically).
- The outer CV *holds the signal that GBM overfits* (its honest estimate drops toward ~0.75 while rf's
  holds) — and we throw it away. The cross-family choice is made on the optimistic inner CV instead.
