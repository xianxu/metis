---
id: 000042
status: codecomplete
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-14
estimate_hours: 0.95
started: 2026-07-14T18:11:47-07:00
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

## Spec

**Seam (metis):** `metis run --sample m` — run m of the k outer folds (1 ≤ m ≤ k) of the
always-materialized k-way partition. `--fast` becomes an alias for `--sample 1` (kept for
compat + docs). Mechanically: `runFolds = m` in the existing `runFolds ≤ k` path
(`cmd/metis/sweep.go`; `CVDriver{K: runFolds}` + banner + `reportEstimate` already take
runFolds). Guard: `--sample` > k errors loudly; `--sample` with a single-config (flat) shape
errors (the flat path has no outer folds to sample; --fast has the same non-applicability
today — keep the behaviors consistent).

**Probe (kbench):** copy `titanic-sweep.md` → `titanic-sweep-k10.md` (copy-working-variant),
same seed 42, `resample: {cv: {k: 10, stratify: true}}`, id `titanic-sweep-k10`. Run
`metis run … --sample 3` (3 outer × 99 configs × 10 inner ≈ 1.2× the 5×5 cost). Analysis
reads the ledger's outer rows directly (no select/promote — the probe measures, it does not
ship): compare per-family + per-config honest estimates (esp. ticket vs no-ticket configs)
against the b7aee3de 5-fold cohort. **Decision rule (pre-committed):** the attenuation
hypothesis is SUPPORTED if ticket-config outer estimates rise toward their public performance
under k=10 (relative to their own k=5 estimates) while no-ticket configs stay flat within
noise; findings land in metis#36's Log either way. SE caveat recorded up front: m=3 gives an
SE estimate with 2 df — the probe reads systematic SHIFT, not tight intervals; it must not be
used to re-select what ships.

## Done when

- `metis run --sample m` runs exactly m outer folds of the k-way partition; `--sample > k`
  and flat-shape misuse fail loudly; `--fast` ≡ `--sample 1`. Unit/e2e coverage for the new
  flag (incl. the ledger carrying m outer rows tagged by fold idx).
- `titanic-sweep-k10.md` exists in kbench; the `--sample 3` probe ran; the k10 cohort's outer
  rows are in the ledger.
- The k5-vs-k10 comparison (ticket vs no-ticket families/configs) is written into metis#36's
  Log with the pre-committed decision rule applied; RUNBOOK gets a §-note that `--sample`
  exists and what it's for (probe, not selection).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.10 impl=0.40
item: atlas-docs          design=0.05 impl=0.35
design-buffer: 0.30
total: 0.95
```

`smaller-go-module` = the `--sample` seam (flag → runFolds + guards + nested-e2e extension —
mirrors the existing `--fast` path). `atlas-docs` = the k10 pipeline copy, probe-run babysit,
k5-vs-k10 ledger reduction, and the #36-Log + RUNBOOK write-up.

## Plan

- [x] metis: `--sample m` flag on `run` (run.go) → `runFolds` (sweep.go); guards (m>k, flat
      shape); `--fast` aliased; banner/estimate lines show m-of-k. TDD: extend the nested-CV
      e2e + a unit test on the guard paths.
- [x] kbench: `titanic-sweep-k10.md` (k:10 copy, new id, same seed); run the `--sample 3`
      probe (background, `--parallel`).
- [x] analysis: reduce k10 outer rows vs b7aee3de k5 cohort (ticket vs no-ticket); apply the
      decision rule; write findings to metis#36 Log + RUNBOOK note; log here.

## Log

### 2026-07-14
- 2026-07-14: closed — TDD: --sample e2e (m-of-k fold coverage, ledger rows, 4 loud-guard cases) red->green; full cmd/metis suite green (18s). Probe verified end-to-end on real data: 3x99x10 run completed exit 0, ledger cohort written, k5-vs-k10 analysis executed with pre-committed decision rule (attenuation SUPPORTED — findings in #36 Log). Atlas experiment.md updated for --sample. RUNBOOK updated (incl. flag-order side-quest fix).; review verdict: SHIP

- Filed from the direction brainstorm (brain session): Titanic LB research digest
  (kbench pensive 2026-07-14-01) says the 0.78→0.83 headroom is WCG group-survival rules, not
  ensembling; the measurement question (does our seal under-rank group features?) is worth one
  cheap probe BEFORE building in that direction. This issue is the probe + the tiny seam it
  needs. Sibling issues: metis#36 (owns the hypothesis + the structural fix), metis#22
  (ensembling — evidence downgraded its Titanic payoff; still a platform primitive).
- Estimate 0.95h via the `## Estimate` block (v3.1 primitives). Calibration source flagged
  [stale] by start-plan — noted, derivation itemized anyway.
- Seam BUILT (commit 2aa12e5, TDD red→green): `--sample` on run → guarded runFolds resolution in
  runShapeSweep; full cmd/metis suite green. kbench side committed (321036d): titanic-sweep-k10.md
  (one-delta copy, k:10) + the LB research pensive + noise-floor lesson; probe launched
  `--sample 3` (3 × 99 × 10 inner). Side-quest in RUNBOOK §1: the documented
  `metis run <path> --fast` was flag-order-broken (Go flag parsing stops at the first
  positional) — reordered + noted, `--sample` row added.
- **PROBE RAN + DECISION RULE APPLIED: attenuation SUPPORTED.** Run: 3 outer × 99 × 10 inner,
  ~28 min wall after a relaunch (first launch thrashed at load-avg 83 — the `--parallel` help's
  own BLAS-oversubscription caveat; relaunched `OMP_NUM_THREADS=1 --parallel 8` on 12 cores,
  load ~21, ~107 trains/min). Numbers (full tables in #36's Log; script
  `scratchpad/k5_vs_k10.py`, rerunnable against the two ledger csvs): `+ticket_survival` inner
  increment over all6, k5→k10: rf 0.0020→0.0078, gbm 0.0059→0.0098, logreg flat; label-free
  `+ticket_size` control flat everywhere; sealed selection flips toward ticket configs (rf 2/3
  outer folds at k10 vs 1/5 at k5). k10 honest outer means flat within (wide, m=3) noise.
  Ledger sidecar `titanic-sweep-k10.ledger.csv` is gitignored like its k5 sibling — `git add -f`
  if the cohort should survive.
- Operator process feedback mid-run (filed to metis#38): the babysitting stats I read by hand
  (trains done, trains/min) belong in the parallel-TUI progress board — moving-average runs/sec.
- Seam scoped: outer k reuses `Sweeper.Resample.CV.K` (sweep.go:202); `--fast` → `runFolds=1`
  (sweep.go:206-208) over an always-materialized k-way partition (sweep.go:322-324, "always
  split into k dirs; --fast just runs fewer") — `--sample m` is the same mechanism with m
  free. Seeded stratified partition ⇒ first-m folds are a valid random m-subset (no fold-
  picker needed).
