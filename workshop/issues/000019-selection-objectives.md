---
id: 000019
status: open
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
---

# selection objectives — 1-SE rule + mean-std (configurable promote objective, not raw cv-max)

## Problem

`promote` selects by **raw cv-max**, which is biased toward overfitters (the max over N noisy configs
inflates + favors fragile high-variance fits). There's no way to prefer a *robust* or *simpler*
config. Nested CV (metis#18) *estimates* the consequence but doesn't change *which* config is picked —
the **selection objective** is the actual lever.

## Spec

metis-v2 M2 (`brain/data/project/metis-v2-experiment-algebra.md`; pensive: the "estimation vs
selection are separate knobs" section). Make the objective `promote` argmaxes configurable:
- **`mean − λ·std`** — penalize configs whose cv swings across folds (fragility ≈ overfit).
- **1-standard-error rule** (Breiman; tidymodels `select_by_one_std_err`) — among configs within one
  SE of the best, pick the **simplest** (fewest features / most regularization). "Prefer the
  less-overfitting near-winner."
- In algebra terms (metis#18): the objective is the **reducer of the config-selection axis**.
- Needs the ledger to carry per-config **std** (from a resample axis / repeated CV), so composes with
  metis#18; a `mean−std` variant is shippable on the existing ragged ledger sooner.

## Done when

- `promote` supports a selectable objective (cv-max | mean−std | 1-SE); default documented.
- A robust/simpler config that loses on cv-max but wins on 1-SE is demonstrably promoted.

## Plan

- [ ] (spec at claim) objective as a pluggable reducer over the ledger; 1-SE + mean−std; test that a simpler within-1-SE config is chosen.

## Log

### 2026-07-07
- Filed as metis-v2 M2. The **selection** knob (separate from estimation/metis#18). Design in the pensive.
