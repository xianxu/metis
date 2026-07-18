---
id: 000054
status: open
deps: []
github_issue:
created: 2026-07-17
updated: 2026-07-17
estimate_hours:
---

# racing successive-halving inner sampler

## Problem

Every config always runs the FULL inner_k folds inside every outer fold — metis#45's
`inner_k` made the budget declarable, but it is still spent uniformly: most of the decision
grid's compute finishes full CVs on configs that are clearly losing after 3 folds. The
adaptive half of metis#45 (lever (b), split out at its (a)-first close) is unbuilt.

## Spec

(Carried verbatim from metis#45 Spec (b), which designed the seams:) An inner Sampler whose
`Ask` uses fold feedback — run every config ~3 folds, drop the clearly-dominated ones (band
vs the incumbent's mean±SE), promote survivors to the full inner_k. This would be the FIRST
real use of the ask/tell feedback edge; all production samplers are static one-batch.
Constraints discovered by design, not assumed: the 1-SE/pct-loss select rule consumes
per-config (mean, SE, n) — uneven n across configs is exactly what `MeanSE.ToldSet` carries,
but `GuardComplexity`/selection semantics over partial configs need a careful pass; the
ledger records per-fold rows already (partial configs are naturally representable);
join-soundness (#32 cohort guard) unaffected. `SizeHint` returns `(fullBudget, SizeBudget)`
— the #38 board renders `k/≤n` (built anticipating exactly this).

Demand-driven: pick up when the next competition's grid cost makes it worth building
(the metis-v2 close's next-arena rule — zero new workbench features until demanded).

## Done when

- On a fixture where one family is strictly dominated, the dominated configs run FEWER folds
  than survivors (asserted on the ledger's per-fold rows), the winner matches the full-CV
  winner, and the board/progress line renders the budget kind (`k/≤n`).
-

## Plan

- [ ]

## Log

### 2026-07-17
