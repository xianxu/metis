---
id: 000045
status: open
deps: []
github_issue:
created: 2026-07-15
updated: 2026-07-15
estimate_hours:
---

# partial inner CV — split inner_k from outer k, and/or an adaptive racing sampler

## Problem

There is no way to run the inner CV partially: every config always runs the FULL inner k
folds inside every outer fold. metis#42's `--sample m` / `--fast` sample the **outer**
folds only — the inner level has no cost knob at all. On the decision grid
(`titanic-sweep.md`: 10 outer × 72 configs × 10 inner = 7,200 leaf folds; still 2,160 with
`--sample 3`) the inner sweep is where nearly all the compute goes, and most of it is spent
finishing full 10-fold CVs on configs that are clearly losing after 3 folds.

Two design facts (from the 2026-07-15 T2 session's Q&A — filed verbatim per operator):

1. **Inner k and outer k are the same knob today.** The outer loop reuses
   `sweeper.resample.cv.k` (`runShapeSweep`: `runFolds = k`), which is why the sweep is
   10×…×10. You can't even declare inner k=5 with outer k=10 right now — splitting them
   would be the cheapest "cheaper inner CV" lever.
2. **The principled version is already designed but unbuilt**: the Sampler ask/tell
   feedback edge exists precisely for adaptive inner sampling (racing /
   successive-halving — kill a config after 3 bad folds instead of running all 10). All
   production samplers are static one-batch; an adaptive `Ask` would be the FIRST real use
   of the feedback loop, and metis#30's `SizeBudget`/`SizeUnknown` SizeHint kinds were
   built anticipating exactly that display case (`k/≤n`, `k/?` in the progress line/board).

## Spec

Two levers, separable — decide at design whether to ship (a) alone first:

(a) **`inner_k` split (cheap knob):** let the shape declare the inner resample's fold
    count separately from the outer driver's (e.g. `sweeper.resample.cv.k` for inner +
    an outer-k field, or an explicit `outer: {cv: {k: …}}`). Semantics to pin at design:
    the outer k is the ESTIMAND knob (train fraction each outer fold simulates — the
    metis#42 principle); the inner k is a selection-precision/cost knob. Interactions:
    `--sample m` stays outer-only; the seeded progress totals (`seededTotals`) and the
    outer-split preamble (`materializeOuterAnalysis`) must read the right k each.

(b) **Racing / successive-halving inner sampler (the adaptive one):** an inner Sampler
    whose `Ask` uses fold feedback — e.g. run every config 3 folds, drop the clearly-dominated
    ones (band vs the incumbent's mean±SE), promote survivors to the full k. Constraints
    discovered by design, not assumed: the 1-SE/pct-loss select rule consumes per-config
    (mean, SE, n) — uneven n across configs is exactly what `MeanSE.ToldSet` carries, but
    `GuardComplexity`/selection semantics over partial configs need a careful pass; the
    ledger records per-fold rows already (partial configs are naturally representable);
    join-soundness (#32 cohort guard) is unaffected. SizeHint returns `(fullBudget,
    SizeBudget)` — the board renders `k/≤n`.

Explicitly out (both levers): changing the OUTER estimand semantics; any change to the
honest per-family outer estimate (#32's flip stays untouched — this is inner-selection
cost only).

## Done when

- (a) A shape can declare inner k ≠ outer k; the nested run uses each at the right level
  (outer-split dirs + held-out scoring at outer k; per-config CV at inner k); progress
  totals and ledger rows reflect it; the existing same-k shapes run unchanged.
- (b — if built) A racing inner sampler drops dominated configs early: on a fixture where
  one family is strictly dominated, the dominated configs run fewer folds than survivors
  (asserted on the ledger's per-fold rows), the winner matches the full-CV winner, and the
  board/progress line renders the budget kind (`k/≤n`).
- The RUNBOOK documents the new knob(s) with the cost arithmetic.

## Plan

- [ ] (at claim) Decide (a)-first vs (a)+(b); design the shape-schema change with the
  vocabulary model; then spec + change-code.

## Log

### 2026-07-15
- Filed from the operator's post-T2 question ("have we done the feature that allows inner
  cv to run partially?") — answer was no; #42 samples outer folds only. Context: T2 UX
  tranche (#39/#30/#38) merged the same day; the board's `SizeBudget`/`SizeUnknown`
  render paths are ready for an adaptive sampler. Sibling knob already live: `--sample m`
  (outer). The k5→k10 move (kbench#9, attenuation-driven) doubled inner cost — the
  10×72×10 grid is where the pressure comes from.
