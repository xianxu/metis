---
id: 000018
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
started: 2026-07-07T11:08:31-07:00
---

# experiment-design algebra + nested CV — resample axis ($fold/$cv), per-axis reducers, fold as a first-class value

## Problem

The sweep treats data-splitting (CV) as an internal detail of `train`, and selects by raw cv-max on a
single fold split → selection-overfitting (the metis-v1 overfit gap: ~0.81 cv → 0.78 public). The
workbench can't express **nested CV** (an honest estimate of the sweep procedure) or a **resample
axis** at all. Deeper: `$any` (metis#17) unified config *choice*, but data-splits are a second kind
of axis the algebra doesn't model.

## Spec

The **core** of metis-v2 (`brain/data/project/metis-v2-experiment-algebra.md`, M1). Full design +
open questions: **`workshop/pensive/2026-07-07-experiment-design-algebra.md`** (read first).
Direction:
- **Axis kinds + reducers:** config axes (`$any`, ranges) reduce by **selection** (argmax objective);
  a new **resample axis** (`$fold`/`$cv(k)`, `$repeat`, `$bootstrap`) reduces by **aggregation**
  (mean/std). One config → many fold-runs → one aggregated score.
- **Nested CV** = an outer resample axis wrapping (config-selection over an inner resample axis),
  expressed declaratively (not three hand-written loops). Reports the honest procedure estimate.
- **Fold becomes a first-class threaded value** (visible to `features`/`cv-split`/`train`), not a
  `train` internal — the same restructuring fold-aware features (metis#20) needs.
- **Ledger reduces per-axis:** config→rows, resample→mean/std columns, nested-outer→one estimate.
- Prior art to mirror: **tidymodels** `rsample`/`nested_cv`/`workflow_set`/`collect_metrics`; sklearn
  `Pipeline`+`GridSearchCV(cv=)`.

**DESIGN-FIRST:** brainstorm → durable plan before code (resolve the pensive's 5 open questions).
Everything else in metis-v2 depends on this (esp. the fold-as-first-class value).

## Done when

- A shape can declare a `$fold`/`$cv` resample axis; the engine expands `(config × fold)` runs and the
  ledger aggregates fold-runs → mean/std per config (not one lucky split).
- **Nested CV** expressible: an outer resample around the sweep yields an honest procedure-level
  estimate distinct from the (inflated) inner cv-max.
- `fold` is a first-class value the pipeline threads (unblocks metis#20).
- atlas: the experiment-design algebra (axis kinds + reducers) documented.

## Plan

- [ ] Brainstorm the algebra + resolve the pensive's open design questions (where folds live; ledger per-axis reduction; nested×cache cost; sampler orthogonality) → durable plan.

## Log

### 2026-07-07
- Filed as metis-v2 M1 (the core). Design in the pensive + project (`sources`). The operator's frame:
  resampling & selection are first-class, declarative axes with per-axis reducers — one algebra, not loops.
