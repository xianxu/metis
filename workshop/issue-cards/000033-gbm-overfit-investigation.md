---
id: 000033
status: open
created: 2026-07-13
updated: 2026-07-13
estimate_hours:
github_issue:
---

# GBM overfits hard on Titanic — bug vs regularization defaults vs effective-complexity measure

## Problem

On the real Titanic honest-beat run (kbench#8), the shipped `hist_gbm [title,family]` (iter=100,
max_leaf_nodes=15) scored inner-CV **0.846 → public 0.749** — a ~0.10 gap, far worse than rf/logreg. Even
the *simplest* GBM in the grid overfits this hard. Three questions to resolve, cheapest first.
