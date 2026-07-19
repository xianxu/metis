---
id: 000063
status: open
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours:
---

# per-row sample weights: consume the dormant schema weight role

## Problem

**PARKED BY DESIGN — file-when-demanded; the design is settled, the demand isn't here yet.**
Distribution reweighting currently exists only as the per-class shortcut (`class_weight`
model hyperparam, loss-space) and — once metis#60 lands — the per-class decision rule
(`decide` offsets, decision-space). There is no way to weight individual ROWS (importance
weighting), though the substrate anticipated it: `metis.schema`'s `weight` role has been
dormant since metis#1 — no step emits it, `train` never consumes it. The concrete demand
that will activate this: the source-dataset extension idea (arena2 discussions 2026-07-18/19
— appending the competition's inspiration dataset, whose distribution differs; importance
weights are the principled treatment of "similar but shifted" auxiliary rows).

## The taxonomy this issue fixes in place (operator-ratified 2026-07-19)

Three orthogonal spaces where "reweight the distribution" can live:
1. **Data space** — physically rewrite samples (over/under-sampling). Rejected: destroys
   row identity, breaks fold accounting and caching.
2. **Loss space** — per-sample weights at fit. Today only the per-class shortcut
   (`class_weight` → sklearn loss multipliers); THIS issue generalizes it to per-row.
3. **Decision space** — post-fit output reweighting (metis#60's cost-sensitive plug-in
   rule; per-class/cost-matrix by nature, never per-row).


## Spec

- A feature-style pipeline step (competition-side) MAY emit a `weight`-role column, computed
  from anything — class membership (reproducing `class_weight=balanced` exactly as
  `n/(K·n_class)`), row provenance (auxiliary-dataset rows at fractional weight), recency,
  difficulty. Fold-scoped under the seal like every fitted parameter (weights computed on
  analysis rows in a per-fold run).
- `metis/train` consumes it: if `schema.weight_col()` is set, pass that column as
  `sample_weight=` to `.fit()` on BOTH paths (per-fold and all-rows refit); both families
  accept it. The weight column is NEVER a model input feature (role already excludes it).
- Why this shape is right for the sweeper: weights are DATASET CONTENT, not an engine
  concept — swept like any knob, cache-addressed for free; the sweeper keeps seeing one
  scalar per-fold score of the declared procedure. `class_weight` (hyperparam) stays as the
  per-class shortcut; `decide` stays decision-space; a shape can sweep any combination and
  the honest estimate arbitrates loss-space vs decision-space empirically.
- Interaction note: `class_weight` AND a weight column together multiply in sklearn —
  refuse loudly or document deliberately (decide at plan time).


## Done when

-

## Plan

- [ ]

## Log

### 2026-07-18
