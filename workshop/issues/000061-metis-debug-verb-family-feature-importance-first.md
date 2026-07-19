---
id: 000061
status: open
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours:
---

# metis debug verb family: feature-importance first

## Problem

The workbench measures and selects but has no microscope: after a sweep there is no way to
ask a materialized run "WHICH features did you actually use?" — needed right now to guide
arena2 M3's interaction-ladder climb (extend the strongest combos, not blind 4-way search),
and generally whenever a rung's value needs explaining rather than just ranking. Operator
design (2026-07-18 session): a `metis debug <sub>` READ-ONLY verb family over materialized
runs/ledgers — inspection, not pipeline steps — starting with feature-importance.


## Spec

- **`metis debug feature-importance --run <id> [-n 10]`** (v0): resolve the run dir
  (`runs/<id>/`), load `model.pkl` + the captured dataset's schema (feature_cols order =
  model input order — the existing determinism seam), print top-N (name, importance),
  aligned columns.
- **Per-family mechanics (the design fact that shapes this):** `rf` → native
  `feature_importances_` (MDI, free). `hist_gbm` → has NO importances attribute; use
  `sklearn.inspection.permutation_importance` with the run's declared metric via
  metis#59's `resolve_scorer` (importance FOR the optimized objective — better than MDI),
  seeded, on a bounded row sample (deterministic + fast); print which method was used.
- **Selector abstraction:** `--point <addr>` composes later via the select/promote
  reconstruction path (a point's leaf runs don't persist model.pkl — only materialized
  ship runs do); v0 is `--run`. The selector language should stay shared with `metis
  select` (one grammar, ARCH-DRY).
- **The family (future members, filed-when-demanded):** `debug reliability` (calibration
  curve buckets — the 2026-07-18 calibration discussion; pairs with metis#60's
  probabilities), `debug describe` (per-feature distributions + class balance — subsumes or
  complements the metis#5 STEP idea; reference, don't hijack). Family contract: read-only,
  no cache writes, no ledger writes.


## Done when

-

## Plan

- [ ]

## Log

### 2026-07-18
