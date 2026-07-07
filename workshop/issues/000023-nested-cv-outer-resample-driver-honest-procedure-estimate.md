---
id: 000023
status: open
deps: [metis#18]
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
---

# nested-CV outer resample driver — honest procedure estimate

## Problem

The flat sweeper (metis#18) selects a winner and reports its inner-CV score — but that score is
*optimistic* (the max over N noisy configs; the selection itself overfits). There's no honest estimate
of *the whole tune-then-fit procedure*. That gap is exactly metis-v1's ~0.81 cv → 0.78 public.

## Spec

metis-v2 **M1b** (pensive `2026-07-07-experiment-design-algebra.md`). The **outer driver** wrapping the
black-box sweeper: `driver: cv` (or `nested`) — `driver(sweeper[inner-cv](pipeline))`, isomorphic to
mlr3 `resample(AutoTuner(resample(learner)))`.

- For each outer fold: hand the sweeper the outer-**analysis** data (sealed from outer-**assessment**);
  the sweeper runs its full inner-CV selection → a winner; **refit** the winner on outer-analysis; score
  on the sealed outer-assessment. Aggregate the k outer scores → the honest procedure estimate.
- **Result-dependent** (unlike the flat cross-product): the refit-and-score depends on *which* config the
  sweeper selected — different outer folds may pick different winners. So it's NOT a static expansion;
  it's `expand → run → select → expand-winner → run → aggregate`.
- **Produces no winner to ship** — it estimates the procedure. The shipped config still comes from the
  sweeper on all data (the flat/ship path). Estimation and selection are different computations.
- **Cost ~5×** (each outer fold changes the data upstream → genuinely independent sweeps; the cache
  can't dedup). The engine must surface this cost so it's opted into knowingly.

Deps **metis#18** (the sweeper substrate + fold-as-artifact + read-time reduction).

## Done when

- `driver: cv`/`nested` expressible; a Titanic run yields an honest procedure-level estimate distinct
  from (and lower than) the inflated inner cv-max.
- The estimate is a mean±SE over outer folds; the ~5× cost is reported before/at run.
- atlas: the driver (outer resample) documented alongside the sweeper.

## Plan

- [ ] (spec at claim, after #18) outer driver over the black-box sweeper; result-dependent refit-on-sealed-fold; honest estimate + cost surfacing; Titanic estimate-vs-inner-cv gap test.

## Log

### 2026-07-07
- Split out of metis#18 (was "M1 + nested CV"). Nested-CV is result-dependent and built ON the sweeper
  substrate — cleanly separable. Design in the pensive (driver/sweeper/pipeline).
