---
id: 000054
status: open
created: 2026-07-17
updated: 2026-07-17
estimate_hours:
github_issue:
---

# racing successive-halving inner sampler

## Problem

Every config always runs the FULL inner_k folds inside every outer fold — metis#45's
`inner_k` made the budget declarable, but it is still spent uniformly: most of the decision
grid's compute finishes full CVs on configs that are clearly losing after 3 folds. The
adaptive half of metis#45 (lever (b), split out at its (a)-first close) is unbuilt.
