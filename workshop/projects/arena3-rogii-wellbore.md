---
type: project
name: arena3-rogii-wellbore
goal: "Third arena: run the live Featured competition rogii-wellbore-geology-prediction through the metis workbench end-to-end — DRIVING metis#36's channel-split + cluster-unit CV out of the competition's demand (grouped-sequence, whole-well-holdout regression). Prove the workbench generalizes beyond flat-tabular classification (titanic/s6e7) to grouped-sequence REGRESSION data (the arena2 generalization thesis's next turn)."
done_when: "A live rogii submission produced by the honest flow (metis run → select --best --promote → kaggle submit) under WELL-CV (cluster unit = WELLNAME), with the nested-CV honest estimate recorded AND a Log entry on whether the honest estimate tracks the leaderboard. Plus: metis#36's channel-split infra landed (y as a runner-scoped keyed channel; seal deleted; O(k·N)→O(1)) and the transductive-vs-prospective acceptance finding recorded. Leaderboard position is evidence, not the goal — the deliverable is (a) the generalization proof onto grouped-sequence regression, (b) the demanded-feature list (what rogii actually pulled out of #36), and (c) the honest-tracks-public finding."
status: executing
deadline: 2026-08-05
operator: xianxu
explicitly_out:
  - new metis infra built speculatively beyond what rogii demands
  - "the #37 R-scope constructor algebra"
created: 2026-07-19
updated: 2026-07-20
sources: [../pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md, ../issues/000036-channel-split-y-as-runner-scoped-keyed-artifact-nested-cv-as-domain-restriction-metis-v3.md, ../plans/000036-channel-split-y-channel-plan.md]
planned_finish: 2026-08-01
---

# arena3 — rogii-wellbore-geology-prediction

The third arena, and the **flagship metis-v3 driver**. Where arena2 (playground-s6e7) proved the
workbench generalizes across *flat-tabular classification* competitions with zero speculative features,
arena3 pushes the generalization onto a genuinely different regime — **grouped-sequence regression** —
and uses that competition's demand to pull out metis#36's channel split (the operator's rogii-first
decision, 2026-07-19: workbench pattern — build the feature when the competition pulls it out of you).

## The competition (demand)

Live **Featured, $50k, ~5,262 teams, deadline 2026-08-05**. Full digest in metis#36's `## Log`; the
load-bearing facts:

- **Grouped SEQUENCE data** — a directory of **773 per-well CSV pairs** (`horizontal_well` + `typewell`),
  each a depth-ordered (`MD`, ~1-ft) sequence. Well = `WELLNAME` (8-char hash). Hidden test = ~200
  **disjoint** held-out wells.
- **Regression** — RMSE on `TVT`; toe-end **extrapolation** (heel given, `TVT_input=NaN` over the eval
  zone). Submission `id = {WELLNAME}_{row_index}`.
- **Naive row-CV LEAKS** — adjacent ~1-ft samples are near-identical → random row-split leaks across a
  well → optimistic score that won't hold on unseen wells. **The well is the CV group.** This is the
  concrete pull for metis#36's cluster-unit CV.

## What rogii demands of the workbench (the pull list)

1. **metis-core regression support** — metis is classification-only; rogii is RMSE regression (metis#36 M0).
2. **cluster-unit CV** — hold out whole wells, not rows (metis#36 M3). The headline demand.
3. **grouped-sequence ingestion** — a directory of 773 CSV pairs → schema'd `Dataset` with the well as the
   row-group unit; a grouped-sequence `adapt` step-type (kbench — beyond the flat-`train.csv` `adapt`).
4. **toe-end masking** — within a held well, mask the toe (mimic `TVT_input=NaN`) — rogii-specific adapt
   detail for now (demand-gated; generalize only on a second competition's demand).

## Fleet (cross-repo scope)

- **metis#36** — the channel split (y as a runner-scoped keyed channel; nested CV as domain restriction;
  cluster unit; estimand knob; delete the seal). Plan: `workshop/plans/000036-channel-split-y-channel-plan.md`.
  Milestones **M0** regression support · **M1** rogii hits the wall · **M2** channel core + prospective
  anchor · **M3** cluster-unit CV · **M4** delete the seal · **M5** acceptance.
- **kbench#18** — the rogii workspace (get-data over 773 well pairs + grouped-sequence `adapt` + baseline
  + submission). ONE issue (operator decision 2026-07-19), not per-step-type. Deps metis#36 (M0 regression).
  **CLOSED** (plumbing baseline + leak + live persistence submission 15.883).
- **kbench#19 / #20 / #21** — the **"real baseline" milestone** (see Milestones): geo-aware spatial-block
  CV (#19) + GR-typewell log-correlation features (#20) + neural sequence model & live submission (#21,
  deps #19+#20). All kbench-local; the metis generalization is deferred to the workbench-drive milestone.

## Regression anchor (risk mitigation)

rogii-first entangles a new ingestion regime with the #36 refactor, so keep a known-good anchor: the
metis#35-era honest-beat on titanic/s6e7 (rf md=4 + ticket_survival → public 0.79186). Per the #36 plan's
C2 correction, the anchor is **prospective** mode (reproduces the seal's internal CV estimate); transductive
is *expected* to diverge (metis#42 quantified it); the shipped **public** score is refactor-invariant.

## Milestones

The arena is now phased (operator decision 2026-07-20 — "we don't have a good baseline yet"):

1. **M-plumbing (DONE)** — kbench#18 + metis#36 M0/M1: the workbench generalizes onto grouped-sequence
   regression; the leak is quantified; the notebook-submission infra is proven (persistence 15.883 live).
   *We learned the row model is the wrong shape and the real signal is GR↔typewell correlation.*
2. **M-real-baseline (CURRENT)** — "real modeling with geospatial awareness + geological knowledge":
   **kbench#19** (geo-aware spatial-block CV) + **kbench#20** (GR-typewell log-correlation features,
   DTW+markers) + **kbench#21** (neural sequence model, geo-CV validated, live submission). Goal: a
   genuinely competitive baseline on the board (beat persistence 15.88), all kbench-local. **Go neural.**
3. **M-workbench-drive (DEFERRED, next)** — generalize what the baseline proved back into metis: a
   `ResampleUnit = spatial-block(buffer)` split unit, a `torch`/GPU model-kind, and the queued metis#36
   M2→M5 channel-split (cluster-unit CV). Demand-gated: build in the workspace first, promote once it works.

## Tasks

- [x] **kbench#18** — rogii workspace (grouped-sequence adapt + baseline + typewell join + leak). CLOSED 2026-07-19: submission.csv (held-out 74.4→42.1 w/ typewell); leak row 8.0 vs well 74.7. Live persistence 15.883 (M-plumbing).
- [ ] **kbench#19** — geo-aware spatial-block CV (buffered; wells ~470 ft apart leak). *M-real-baseline; the honest-validation foundation.*
- [ ] **kbench#20** — GR-typewell log-correlation features (DTW/cross-correlation implied-TVT + markers). *M-real-baseline; the competition signal.*
- [ ] **kbench#21** — neural sequence model + live submission (deps #19,#20; go-neural). *M-real-baseline; beat persistence 15.88.*
- [x] **metis#36 M0** — regression support (model kind + RMSE scorer + regression predict/complexity). DONE (+M1 predict-step regression branch, commit 58a51e9).
- [x] **metis#36 M1** — rogii hits the wall: naive row-CV demonstrably leaks. DONE via kbench#18's out-of-engine well-holdout (`leak_demo.py`): row-CV 8.0 vs well-CV 74.7 = 9.35×.
- [ ] **metis#36 M2** — channel split core + prospective anchor (reproduce titanic/s6e7 seal number).
- [ ] **metis#36 M3** — cluster-unit CV (`cluster: WELLNAME`); rogii's row-CV-leak closes under well-CV.
- [ ] **metis#36 M4** — delete the seal (analysis_i cloning, sealed branch); O(k·N)→O(1) confirmed.
- [ ] **metis#36 M5** — acceptance: rogii honest estimate vs leaderboard; transductive-vs-prospective finding.

## Log

### 2026-07-19
- Project opened. Operator picked rogii as arena3 + chose rogii-first (accept the full lift) over decoupling
  (metis#26 closed subsumed-by-#27 earlier this session; metis-v2 project archived brain→metis). Rules
  accepted (download unblocked). metis#36 plan v2 written + fresh-eyes-reviewed (3 critical + 4 important
  folded in). Next: file the kbench rogii-workspace issue, then metis#36 M0 (regression support).

### 2026-07-19 — kbench#18 M1a baseline BUILT (honest flow → submission.csv); leaderboard post blocked (kernels-only)
- **metis#36 M0** (regression support) confirmed DONE earlier; **metis#36 M1** advanced: `predict.py` regression
  branch fixed on the #36 branch (`58a51e9`) — predict step no longer crashes on a regressor's missing
  `predict_proba`. (The row-CV-leak demonstration + cluster-unit CV remain M1/M3.)
- **kbench#18 M1a DONE:** grouped-sequence `adapt` (dir of 773 well-pairs → Dataset; well-id from filename;
  toe mask; train/test disjoint) + `rogii/submission` + `rogii-baseline.md`. `metis run` → valid
  `submission.csv` (14,151 rows == sample_submission). **Baseline held-out RMSE ≈ 74.4** (offline, genuine —
  the 3 test wells excluded from training but their TVT is in train/).
- **Generalization proof (partial):** the workbench GENERALIZED onto grouped-sequence regression — the honest
  flow ran end-to-end on a genuinely new regime (directory ingestion, regression, toe extrapolation). The
  arena2 thesis's next turn holds at the plumbing level.
- **Finding for the pull-list:** the naive row-model (RMSE 74) is far worse than trivial persistence (RMSE 10),
  which is worse than the leaders (RMSE 4.86). The sequence-continuity structure the row-model ignores is the
  real signal — this is the concrete demand for sequence-aware features (M1b typewell join) and the row-CV
  leak (M1b) → cluster-unit CV (metis#36 M3).
- **BLOCKER:** `is_kernels_submissions_only=True` — a live leaderboard number needs a Kaggle *notebook*
  submission (CSV API → 403). done_when's "live submission" is pending an operator decision on the notebook path;
  the offline held-out estimate is the honest stand-in. kbench#18 M1b (typewell + leak) is next regardless.

### 2026-07-19 — kbench#18 M1b DONE (typewell join + the leak quantified) → kbench#18 CLOSED
- **Typewell join (Done-when #2):** per-well `tw_{tvt_min,tvt_max,gr_mean,gr_std}` features (the type
  curve's TVT bracket = the per-well anchor). **Held-out RMSE 74.4 → 42.1** (−43%, zero model change).
- **The leak (Done-when #3):** `leak_demo.py` (out-of-engine) — row-CV **8.0** vs well-CV **74.7** =
  **9.35×** optimism. This is the concrete, quantified demand for **metis#36 M3** (cluster-unit CV =
  `ResampleUnit.cluster`; row-CV = the degenerate unit=id case). arena3's "rogii hits the wall" (M1) is
  now measured, not asserted.
- **Generalization proof:** the honest ladder geometry 74.4 → +typewell 42.1 → persistence ~10 →
  leaders ~4.86 — the workbench generalized onto grouped-sequence regression AND the domain join helped.
- **Next (metis#36):** M2 channel core + prospective anchor → M3 cluster-unit CV (rogii's leak closes
  under well-CV) → M4 delete the seal → M5 acceptance.
- **LIVE SUBMISSION LANDED — the leaderboard is real now (2026-07-20).** Operator identity-verified;
  the notebook-submission infra is proven end-to-end. Three empirical findings, each correcting an earlier
  inference (submission mechanics must be *proven*, not reasoned):
  1. **Hidden-rerun code competition** (NOT scored-on-download, as I'd argued ~85%). The fixed-CSV
     *passthrough* kernel scored COMPLETE but the scorer REJECTED it — "incorrect format / wrong number of
     rows": the scored test ≠ the 3 downloadable wells. A **self-contained notebook that READS the mounted
     test wells and predicts** (kernel `xianxu/rogii-persistence-baseline`, no dataset/model/internet) was
     ACCEPTED → the rerun mounts the hidden test at scoring; the notebook must generate for whatever it's given.
  2. **Metric is RMSE** (deck-authoritative; the Kaggle "Mean Squared Error" tag is loose): persistence
     public **15.883** lines up with offline persistence ~10 (easier download wells) + leaders ~4.86; under
     MSE those wouldn't line up.
  3. **Persistence (public 15.883) BEATS the geometry+typewell ML baseline (offline held-out ~42).** Carrying
     the last heel TVT across the toe > cross-well row-regression — the task is sequence-continuity + GR-typewell
     correlation, not row-wise geometry. The real signal (leaders ~4.86) is the GR-log correlation (demand-gated).
  - **Infra reusable:** self-contained read-test → predict → `/kaggle/working/submission.csv`, kernels-only,
    internet-off. Swap the predictor to submit any model (the ML port would score ~worse than persistence here).

### 2026-07-20 — M-plumbing wrapped; M-real-baseline framed + filed (fresh-context handoff)
- Domain nailed down (geosteering: GR log correlation to the typewell recovers TVT; target is a human
  interpretation → mimic the convention). Code Requirements confirmed: notebooks-only, ≤9h, internet OFF,
  **pre-trained models allowed** (→ train offline, upload weights as a dataset, infer in-notebook — the
  sweeper stays offline; no metis-into-notebook port). Field geometry: 773 wells, ~34×24 mi, median
  nearest-neighbor **~470 ft** (neighbors leak → need buffered spatial-block CV).
- **Arena phased into 3 milestones** (see Milestones). **M-real-baseline** filed as kbench#19 (geo-CV) +
  #20 (DTW/correlation features) + #21 (neural seq model + live submission). Operator: **go neural this
  time**; keep torch kbench-local; **workbench-drive (metis generalization) DEFERRED to the next milestone**
  — "we don't have a good baseline yet."
- **Handoff:** continuation written for a fresh session to start M-real-baseline (kbench#19 first — the
  honest-validation foundation). NOTE: kbench#18 is codecomplete but UNMERGED (8 commits ahead of main on
  branch `000018-…`); #19/#20/#21 issue files ride that branch. First step in the fresh session: publish
  #18 → main (needs a push), then branch #19 off clean main.

### 2026-07-20 — transition evidence

- issues-cover-prd: done_when's 3 deliverables are covered by the issue fleet: (a) generalization proof onto grouped-sequence regression → metis#36 M0 (regression core ✓) + kbench#18 (grouped-sequence workspace+baseline ✓); (b) demanded-feature list → metis#36 M2-M5 (channel-split/cluster-CV/seal-delete) + kbench#19 (geo-CV)/#20 (GR-typewell correlation features); (c) honest-tracks-public finding → kbench#21 (neural model + live submission, geo-CV estimate vs public score) + metis#36 M5 acceptance.

### 2026-07-20 — planned_finish

- planned_finish set manually: 2026-08-01
