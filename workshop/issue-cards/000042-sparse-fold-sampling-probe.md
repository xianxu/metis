---
id: '000042'
status: done
started: 2026-07-14T18:11:47-07:00
created: 2026-07-14
updated: 2026-07-14
estimate_hours: 0.95
actual_hours: 0.27
---

# sparse fold sampling — generalize --fast to m-of-k + 10-fold attenuation probe

## Problem

Two entangled facts from the 2026-07-14 honest-beat + the Titanic LB research digest
(`kbench/workshop/pensive/2026-07-14-01-pensive-titanic-lb-research-digest.md`):

1. **The seal attenuates group features** (metis#36 hypothesis): under 5×5 nested CV a
   ticket-group feature is measured at ~0.8×0.8 ≈ 64% of its ship-time partner coverage
   (38.6% → ~30% labeled-partner coverage), plus m=10 shrinkage — while the shipped model
   (fit on all 891) gets it at full strength. Empirically: nested CV ranks ticket configs
   BELOW no-ticket, yet they hold the top two public-LB spots.
2. **Fold count k is the estimand knob, folds-evaluated m is the precision knob.** k sets the
   train fraction the measurement simulates (k=10 → 90% train → ~81% coverage); each evaluated
   fold is an unbiased sample of that estimand no matter how many run. `--fast` already runs
   1-of-k over a stable k-way partition — but the general m-of-k is not expressible, so raising
   k to reduce attenuation bias forces the full 4× cost (10 outer × 10 inner vs 5×5).

Operator direction (2026-07-14 brainstorm): "there got to be some sort of sparse cross-cv —
only 10 fold, but only run 3 random of the 10." Since the partition is seeded+stratified, the
first m folds ARE a random m-subset — the existing `--fast` mechanism generalizes directly.
