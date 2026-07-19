---
id: 000036
status: working
deps: [metis#35]
github_issue:
created: 2026-07-14
updated: 2026-07-19
estimate_hours:
started: 2026-07-19T16:22:10-07:00
---

# channel split: y as runner-scoped keyed artifact — nested CV as domain restriction (metis-v3)

## Problem

metis#35's root cause is architectural: the nested-CV seal (metis#23) substitutes a derived
artifact (`analysis_i`) and deletes its producers — sound only when that artifact is the sole road
from raw data to the pipeline, an invariant nothing enforces (the `raw: get-data` bypass proved
it). Row-cloning also costs O(k·N) storage, forces `test=None` shape mismatches, and hides
*features* when only *labels* need hiding — under transductive (Kaggle) semantics, hiding held
rows' features actively mismatches the deployment (both-frames features like ticket_size see
train+test at ship time). The protection we actually need is: **no step's fitted parameter may
depend on a held row's label.**

**Research notes (read first):**
`workshop/pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md` — the full
model (two-channel data, fit∘apply scope signatures, aggregate classes), the verified literature
map (two deep-research passes + a 27-agent adversarial verify), and the framing: the fit boundary
is a **declassification point**; cross-fitting is the declassification policy for the y channel.

## Spec

Replace the row-cloning seal with a runner-owned label channel:

- **y is a separately-keyed artifact** (id → label), split from X at `adapt` (the demultiplexer —
  the first schema'd node). A fold context = a **domain restriction of y alone**: outer fold i
  hands the pipeline `y|dom(y)\Oᵢ`; inner folds restrict further ((y|A)|B = y|A∩B — nesting
  composes). X is never restricted (transductive; see estimand knob below).
- **fit_mask is derived, not passed**: `fit_mask ≡ k ∈ dom(y)`. Steps stop receiving a mask they
  are trusted to honor; a label absent from the artifact cannot be used.
- **The scorer is the declassification boundary**: split `metis/train` into fit/predict (label-
  restricted) + a terminal `score` step — the only step handed held labels (`y|B`), and its output
  is a scalar. No dataset-shaped artifact downstream of held labels.
- **Estimand declared in the shape header**: `semantics: transductive` (default — Kaggle) vs
  `prospective` (mask labels AND drop rows — the current behavior, kept reachable). Decides
  whether label-free constructors are hoisted or per-fold.
- **Static one-road rule**: y has exactly one producer; no pipeline step's `with` may reference a
  data-phase step other than the base (kills the #35 bug class at parse time). The runner (split/
  stratification) is the one sanctioned full-y reader.
- **Deletions**: `analysis_i` cloning, `METIS_READ_ROOT` + `readRoot` plumbing (run.go/exec.go/
  sweep.go), `buildFoldExperiment`'s sealed branch (`baseRef`/`dropNeeds` surgery). Nested CV
  becomes the same experiment as flat, run under a mask pair. O(k·N) → O(1) storage.
- **Fold-level caching shape** (from the SystemDS finding, CIDR'20/LIMA): folds are a partition,
  so a fold complement-θ = merge of the other parts' per-fold partials — monoid suffices at the
  fold level; subtraction (abelian-group aggregates) is only needed for per-row S(k) (LOO/
  cross-fit). Don't build subtraction machinery for the fold axis.
- **Acceptance experiment (answers an open research question)**: run ticket_size hoisted vs
  fold-scoped; compare which honest family estimate tracks the public leaderboard. Moscovich &
  Rosset 2022 is inductive-only; the transductive case is verifiably open in the literature.

## Done when

- Nested CV runs the real kbench sweep with NO analysis_i materialization and NO METIS_READ_ROOT;
  results match the metis#35-era honest-beat baseline (the stage-A run is the regression anchor).
- A leakage e2e proves structurally that a target-encoding feature cannot see held labels: a
  step attempting to read `y` beyond its restricted domain fails loudly (label absent, not
  convention violated).
- `score` is the only step receiving held labels; `train` no longer both fits and scores.
- Shape header declares transductive/prospective; prospective mode reproduces row-hiding.
- The ticket_size hoisted-vs-fold-scoped experiment ran; finding recorded (pensive/project).
- Sealed-branch code, readRoot plumbing, and outer-split cloning are deleted; e2e green.

## Plan

- [ ] Brainstorm-first with operator (design via superpowers-writing-plans; durable plan in
  workshop/plans/). Key open questions from the pensive: y-artifact on-disk shape; how the
  runner injects restricted y (per-step artifact vs env-pointed file); score-step metric contract;
  where prospective mode's row-drop lives.

## Log

### 2026-07-14
- Filed as stage B of the three-stage plan agreed with operator (A = metis#35 one-road fix on the
  current seal, closes metis-v2; B = this, the structural redesign; C = metis#37 constructor
  algebra). Design substance + verified literature in the pensive (in-repo; see Problem). Depends on
  #35 landing first — stage A's honest-beat run is this issue's regression baseline.
- Hypothesis sharpened by the 2026-07-14 point-rf submission (public 0.78229 for the ticket config
  the honest estimate ranked BELOW the no-ticket pick that scored 0.77751): three same-direction
  public samples now suggest nested measurement under-ranks co-occurrence features (fragmentation:
  labeled-partner coverage 38.6%→~30% under the seal + m=10 shrinkage). The ticket_size experiment
  in this issue's acceptance should quantify exactly this bias, not just hoisted-vs-fold-scoped.
- **ATTENUATION QUANTIFIED (metis#42 k10 probe, 2026-07-14 evening).** Same grid at k=10
  (`titanic-sweep-k10.md`, seed 42) run `--sample 3` vs the k=5 b7aee3de cohort; seal-time
  labeled-partner coverage ~64% → ~81% of ship-time. Pre-committed rule: SUPPORTED if ticket
  increments rise for families that exploit them while controls stay flat. **Result: SUPPORTED.**
  Inner increment of `+ticket_survival` over the all6 base, mean across configs (k5 → k10):
  **rf 0.0020 → 0.0078 (+0.0058, ~4×) · hist_gbm 0.0059 → 0.0098 (+0.0039)** · logreg
  0.0019 → 0.0024 (flat — logreg can't exploit the interaction anyway). **Internal control:**
  `+ticket_size` is label-FREE (coverage doesn't depend on labeled partners) and its increment
  stays flat (rf +0.0009, gbm +0.0004) — the shift is specific to the label-dependent channel,
  exactly what the fragmentation mechanism predicts. **Selection flips:** the sealed inner
  selection picks a ticket_survival config in 2/3 rf outer folds and 1/3 gbm folds at k10, vs
  1/5 and 0/5 at k5. Honest outer means at k10 (3 folds, wide SE): gbm 0.8282±0.0250,
  rf 0.8283±0.0140, logreg 0.7760±0.0179 — flat vs k5 within noise, as expected (outer mixes
  winners). Implication for this issue's design: the transductive estimand knob is not
  cosmetic — at 90% train we STILL under-measure vs the shipped model's full-coverage + 61%
  test-side split-group deployment; the channel split should make the estimand declarable
  rather than fold-count-implied. Full comparison script + numbers: metis#42 Log.
- **Design input (2026-07-14 bootstrap brainstorm): the resampling UNIT should be declarable.**
  Rows within a ticket/family group share fate — the exchangeable unit is the GROUP, not the row.
  Every resampling surface (outer/inner folds, any future bootstrap replicate, and the estimand
  the seal simulates) is a statement about which unit is drawn; today all of them silently assume
  row-exchangeability. The channel split should let a shape declare a cluster key (e.g.
  `cluster: Ticket`) once, with folds/seals/replicates restricting y by CLUSTER — Recio's
  grouped-CV finding, metis#42's attenuation, and the transductive knob are all projections of
  this one declaration. (Also from the same brainstorm: rf's bagging resamples rows with FROZEN
  upstream encodings — replicate-scoped feature recompute is expressible once y-restriction is
  runner-owned; a "full-pipeline bootstrap" falls out of the same mechanism as folds.)
