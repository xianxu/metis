---
id: 000019
status: open
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
---

# selection objectives — 1-SE rule + mean-std (configurable sweeper select rule, not raw cv-max)

## Problem

The sweeper selects by **raw cv-max** (`argmax-mean`), biased toward overfitters (the max over N noisy
configs inflates + favors fragile high-variance fits). There's no way to prefer a *robust* or *simpler*
config. Nested CV (metis#23) *estimates* the consequence but doesn't change *which* config is picked —
the **select rule** is the actual lever.

## Spec

metis-v2 M2. The **select rule** lives INSIDE the black-box sweeper (converged design; pensive
"estimation vs selection") and consumes the per-config **`(mean, SE)`** the read-time reducer (M1a)
produces. Make it configurable (was `argmax-mean`):
- **`mean − λ·std`** — penalize configs whose cv swings across folds (fragility ≈ overfit).
- **1-standard-error rule** (Breiman; tidymodels `select_by_one_std_err`) — among configs within one
  SE of the best, pick the **simplest** (fewest features / most regularization). "Prefer the
  less-overfitting near-winner."
- The **parsimony ordering** (which config is "simpler") comes free from the tagged `$any` tree (fewer
  features / more regularization / shallower branch); cf. tidymodels `select_by_one_std_err`/`_pct_loss`.
- Composes with metis#18 (needs the `(mean, SE)` from read-time reduction) and is honestly *estimated*
  by nested-CV (metis#23). **Uncontested across all surveyed frameworks — our differentiator.**

## Done when

- The sweeper supports a selectable rule (argmax-mean | mean−std | one-std-err); default documented.
- A robust/simpler config that loses on argmax-mean but wins on 1-SE is demonstrably selected.

## Plan

- [ ] (spec at claim) select rule as a pluggable reducer over the sweeper's per-config `(mean, SE)`; 1-SE + mean−std; test that a simpler within-1-SE config is chosen.

## Log

### 2026-07-07
- Filed as metis-v2 M2. The **selection** knob (separate from estimation/metis#23). Design in the pensive.
### 2026-07-07 (design converged)
- Home clarified: the select rule is INTERNAL to the black-box sweeper and consumes `(mean, SE)` from
  M1a's read-time reduction (not a "ledger objective"). Prior-art survey: 1-SE is uncontested across all
  six frameworks — our sharpest differentiator; parsimony ordering falls out of the tagged `$any` tree.
