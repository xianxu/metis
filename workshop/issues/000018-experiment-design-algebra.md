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

# experiment-design algebra M1a — three-phase shape + black-box sweeper (inner-CV, read-time reduction, fold-as-artifact)

## Problem

The sweep treats data-splitting (CV) as an internal detail of `train` and selects by raw cv-max on a
single split → selection-overfitting (the metis-v1 gap: ~0.81 cv → 0.78 public). Resampling and
selection aren't first-class; the workbench can't produce an honest per-config mean/std, and the
structure that nested-CV (#23) and leakage-safe features (#20) need doesn't exist.

## Spec

metis-v2 **M1a** — the substrate everything else depends on. Full design:
**`workshop/pensive/2026-07-07-experiment-design-algebra.md`** (read first). The converged model
(supersedes the earlier `fold: {$cv: cv-split}` resample-axis framing):

- **Three-phase shape:** `data` (get-data/adapt — run ONCE, above the resample) │ `pipeline` (the swept
  algorithm×hyperparameter atom — always per-fold) │ `ship` (predict/submission — winner only). The
  `data│pipeline` boundary is the ONE structural cut → cross-fold leakage-safety with no per-step markers.
- **Black-box sweeper:** `training data → winner`. Owns its **inner resampling** (`resample: {cv:k}`),
  objective, and select rule; hands the driver the winner's **reconstructable run-keys** (config + keys
  that pin the run + provenance), not just abstract hyperparameters. Reuses the metis#7 Ask/Tell Sampler
  seam (grid = the degenerate sampler; adaptive later). = mlr3 `AutoTuner`.
- **Read-time reduction → `(mean, SE)`:** the ledger keeps raw per-fold rows; a pure `Aggregate` groups
  by config and reduces over folds → mean/SE (so #19's select rule is a free re-reduction — no re-run).
- **Fold-as-artifact + fan-in reducer:** the sweeper materializes k partition artifacts (content-hash =
  which-rows); downstream re-keys via the existing upstream chain; the reducer is a gather node keyed on
  the sorted set of manifested fold-content hashes. Invariant/per-fold boundary emergent from the DAG.
- **`driver: single`** for M1a (fit sweeper on all → ship winner). The outer `driver:cv` (nested-CV) is
  **metis#23**.

Keeps metis's two-phase key (`K_pre` → validate); folds AND code read-set are runtime-manifested.
Prior art: mlr3 (`AutoTuner` = our sweeper), tidymodels (three-phase), sklearn (Pipeline per-fold).
**DESIGN-FIRST:** durable plan (superpowers-writing-plans → `workshop/plans/`) before code.

## Done when

- A shape declares `data│pipeline│ship` + a `sweeper` with `resample: {cv:k}`; the engine runs
  `(config × fold)`, the ledger keeps raw fold rows and `Aggregate` reduces → per-config `(mean, SE)`
  (not one lucky split); the sweeper selects a winner and ships it via `driver: single`.
- Fold enters the cache key as a partition artifact; get-data/adapt cache once, features/train per fold
  (emergent from the DAG); the fan-in reducer's CV score is content-addressed + order-independent.
- Titanic runs through the new shape and reproduces (or beats) v1's promoted winner honestly.
- atlas: the driver/sweeper/pipeline algebra + three-phase shape documented.

## Plan

- [ ] (spec → durable plan) three-phase shape parse + validate; black-box sweeper owning inner-CV; read-time `Aggregate` → (mean,SE); fold-as-artifact scatter + fan-in reducer; driver:single ship path; Titanic e2e.

## Log

### 2026-07-07
- Filed as metis-v2 M1 (the core). Design in the pensive + project (`sources`). The operator's frame:
  resampling & selection are first-class, declarative axes with per-axis reducers — one algebra, not loops.

### 2026-07-07 (design converged)
- Reframed from "M1 + nested CV" to **M1a** (the sweeper substrate); nested-CV split to **metis#23**.
  Converged model: driver/sweeper/pipeline + three-phase shape + no fit_scope marker + input-addressed
  cache leaning (metis#24). Superseded the `fold: {$cv: cv-split}` axis framing after a 3-front prior-art
  survey (mlr3 is the structural twin). Design + reshaped titanic-sweep.md in the pensive.
