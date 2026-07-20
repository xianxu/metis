---
type: project
name: arena3-rogii-wellbore
goal: "Third arena: run the live Featured competition rogii-wellbore-geology-prediction through the metis workbench end-to-end — DRIVING metis#36's channel-split + cluster-unit CV out of the competition's demand (grouped-sequence, whole-well-holdout regression). Prove the workbench generalizes beyond flat-tabular classification (titanic/s6e7) to grouped-sequence REGRESSION data (the arena2 generalization thesis's next turn)."
done_when: "A live rogii submission produced by the honest flow (metis run → select --best --promote → kaggle submit) under WELL-CV (cluster unit = WELLNAME), with the nested-CV honest estimate recorded AND a Log entry on whether the honest estimate tracks the leaderboard. Plus: metis#36's channel-split infra landed (y as a runner-scoped keyed channel; seal deleted; O(k·N)→O(1)) and the transductive-vs-prospective acceptance finding recorded. Leaderboard position is evidence, not the goal — the deliverable is (a) the generalization proof onto grouped-sequence regression, (b) the demanded-feature list (what rogii actually pulled out of #36), and (c) the honest-tracks-public finding."
status: active
deadline: 2026-08-05
operator: xianxu
explicitly_out: [new metis infra built speculatively beyond what rogii demands, the #37 R-scope constructor algebra]
created: 2026-07-19
updated: 2026-07-19
sources: [../pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md, ../issues/000036-channel-split-y-as-runner-scoped-keyed-artifact-nested-cv-as-domain-restriction-metis-v3.md, ../plans/000036-channel-split-y-channel-plan.md]
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

## Regression anchor (risk mitigation)

rogii-first entangles a new ingestion regime with the #36 refactor, so keep a known-good anchor: the
metis#35-era honest-beat on titanic/s6e7 (rf md=4 + ticket_survival → public 0.79186). Per the #36 plan's
C2 correction, the anchor is **prospective** mode (reproduces the seal's internal CV estimate); transductive
is *expected* to diverge (metis#42 quantified it); the shipped **public** score is refactor-invariant.

## Tasks

- [ ] **kbench#18** — rogii workspace (get-data + grouped-sequence adapt + baseline). *drives M1.*
- [ ] **metis#36 M0** — regression support (model kind + RMSE scorer + regression predict/complexity).
- [ ] **metis#36 M1** — rogii hits the wall: naive row-CV demonstrably leaks (truth = leaderboard / out-of-engine well holdout).
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
