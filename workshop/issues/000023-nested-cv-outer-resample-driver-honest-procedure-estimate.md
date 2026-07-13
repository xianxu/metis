---
id: 000023
status: working
deps: [metis#18]
github_issue:
created: 2026-07-07
updated: 2026-07-12
estimate_hours: 3.1
started: 2026-07-12T15:10:37-07:00
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

**Design converged + reviewed (this session).** Full design in `workshop/plans/000023-nested-cv-outer-resample-driver-plan.md`
(2 recon passes + a fresh-eyes plan review: 1 Critical + 4 Important, all fixed). Key decisions:
- **`CVDriver`** — a pure `Sampler[cvDriverState, OuterFoldPoint, float64, MeanSE]` over the **unchanged**
  `Run` loop (zero engine change); reuses `sampler.Aggregate` for the outer mean±SE.
- **Sealing = L1 structural + L2 chokepoint** (operator-chosen). L1: `outer-split` materializes physical
  `analysis_i/` subset dirs (assessment bytes absent from selection). L2: `METIS_READ_ROOT` injected via
  `exec.go` and asserted at **`metis/io.py:exp_path`** (confines base-dataset reads, leaves run-dir
  handoffs alone, covers parquet). Syscall-level airtightness (rogue non-`metis.io` reads, parquet-via-C)
  **documented-and-deferred**.
- **The clean split + its bound assumption:** only *selection* is sealed; *scoring* the chosen winner is
  an honest held-out eval expressed **as a fold** (so it inherits **#20**'s fold-safe features when they
  land). Honest today (stateless features); assumption stated, not left to lie ("artifacts lie by aspiration").
- **The sealing spine (M1) is shared** — #20 (leakage-safe features) + kbench#8 (ticket-group survival) inherit it.

## Done when

- `driver: cv`/`nested` expressible; a Titanic run yields an honest procedure-level estimate distinct
  from (and lower than) the inflated inner cv-max.
- The estimate is a mean±SE over outer folds; the ~5× cost is reported before/at run.
- atlas: the driver (outer resample) documented alongside the sweeper.

## Plan

Two review boundaries (full task breakdown in the durable plan). Each `Mx` closes with its own `sdlc milestone-close`.

- [ ] **M1 — structural outer-partition + trace-enforced read-confinement** (the sealing spine; #20/kbench#8 inherit it). `within_root` predicate + `exp_path` chokepoint assertion; `METIS_READ_ROOT` threaded through `exec.go`/`StepContext`; the `outer-split` step materializing `analysis_i/` subset dirs. Load-bearing tests: an out-of-root data read is caught + named; a legit run-dir handoff read passes (the C1 regression).
- [ ] **M2 — `CVDriver` nested loop + honest e2e** (built on M1). Pure `CVDriver` Sampler; extract the sweeper-as-callable (per-fold accumulators + whole-pipeline repoint to `analysis_i` + forked tail); refit-and-score as a fold over the outer partition; delete the `ValidateShape` stub-reject; ~5× cost surfaced; honest e2e (mean±SE over outer folds, no ship, confinement fails on an injected leak); reporting + atlas.

Honest-estimate-tracks-public acceptance is **operator-gated** (real Titanic, Kaggle) — the offline e2e proves plumbing + seal, not the gap magnitude.

## Estimate

Meatiest metis-v2 milestone: structural data-plane changes (subset-dir materialization) + a new confinement
mechanism + the outer driver + a real extraction re-scope. Most design is banked (2 recons + durable plan +
review, this session); impl ahead across M1+M2 + 2 boundary reviews.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module    design=0.4 impl=0.2
item: smaller-go-module       design=0.3 impl=0.2
item: smaller-go-module       design=0.15 impl=0.16
item: cross-cutting-refactor  design=0.3 impl=0.2
item: smaller-go-module       design=0.2 impl=0.2
item: atlas-docs              design=0.1 impl=0.15
item: milestone-review        design=0.0 impl=0.3
design-buffer: 0.15
total: 3.1
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*
(items: outer-split step · read-confinement · CVDriver Sampler · sweeper-extraction surgery · refit-and-score+e2e · cost/reporting/atlas · 2 boundary reviews. Go-slug primitives used as language-agnostic size anchors; work spans Go + Python.)

## Log

### 2026-07-07
- Split out of metis#18 (was "M1 + nested CV"). Nested-CV is result-dependent and built ON the sweeper
  substrate — cleanly separable. Design in the pensive (driver/sweeper/pipeline).

### 2026-07-12 (claimed; design converged + reviewed)
- 2026-07-12: closed M1 — M1 sealing spine: Python 65 pass incl. both seal halves (out-of-root base-dataset read caught+named via exp_path chokepoint; C1 regression — legit run-dir handoff read passes) + outer-split materializes k analysis subset dirs + within_root unit tests; Go 9/9 ok, METIS_READ_ROOT injected iff-non-empty so driver:single is untouched. Atlas updated (confinement chokepoint + outer-split, shared spine for #20/kbench#8).; review verdict: FIX-THEN-SHIP
- 2 recon passes (nested-CV architecture + read-trace machinery) → design → durable plan
  (`workshop/plans/000023-*.md`) → fresh-eyes plan review (1 Critical + 4 Important + minors, all fixed).
  Operator confirmed: (a) structural separation, (b) trace-enforced read-confinement (L1+L2), syscall-level
  deferred. **Critical the review caught:** the confinement chokepoint must be `metis/io.py:exp_path` (not
  `load_dataset`, which also serves run-dir handoff reads → would crash the sealed sweep) — invisible to
  every offline test, so a "handoff read passes" regression test is mandatory. est 3.1h.
