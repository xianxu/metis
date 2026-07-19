---
type: project
name: arena2-playground-s6e7
goal: "Second arena: run Kaggle Playground Series S6E7 (Predicting Student Health Risk) through the metis workbench end-to-end — prove the workbench GENERALIZES beyond titanic with zero new features until the competition demands them (the v1→v2 pattern's next turn)."
done_when: "A live S6E7 submission produced entirely by the honest flow (metis run → select --best --promote → kaggle submit -C), on a content-pinned dataset, with the nested-CV honest estimate recorded — and a Log entry stating which workbench features the competition actually DEMANDED (possibly none). Leaderboard position is evidence, not the goal (the tight ~0.953 pack makes flips noise; the deliverable is the generalization proof + the demand list)."
status: executing
deadline: 2026-07-31
planned_finish: 2026-07-24
operator: xianxu
explicitly_out: [new workbench features built speculatively]
created: 2026-07-18
updated: 2026-07-18
sources: [brain/data/project/metis-v2-experiment-algebra.md]
---

# arena2 — Playground Series S6E7 through the workbench

Successor to metis-v2 (closed 2026-07-17, done_when met; see `sources`). The arena: Kaggle
**Playground Series S6E7 — "Predicting Student Health Risk"** (closes 2026-07-31; 62MB
train.csv ≈ 100×  titanic's rows; leaderboard packed at ~0.953, metric to confirm from the
overview — likely AUC). Chosen 2026-07-18 (operator) over rogii-wellbore (grouped sequence
data — a step-type lift, candidate third arena) and autonomous-agent-prediction-beta (the
meta-competition mirror — parked, different game).

**The standing rule:** zero new workbench features until S6E7 demands them. Queued demand
candidates, in likelihood order: a new objective metric declaration (AUC — small step-type
addition), metis#33 (GBM regularization defaults — live at real row counts), metis#54
(racing sampler — if the grid cost bites at 100× data), possibly per-competition step reuse
friction (kbench layering). Everything else is out unless the competition says otherwise.

## Milestones

- [x] **M0 — access (operator):** accept S6E7 rules on kaggle.com (the download 403s until
  then). One click; unblocks everything.
- [x] **M1 — bring-up (kbench#12):** `competition/playground-s6e7/` workspace mirroring
  titanic's layout: get-data (kaggle/download + `with.sha256` pins from the first download's
  paste-ready block — the #25 flow), an `adapt` step for the S6E7 schema, a starter shape
  (small grid, `inner_k` from day one), smoke on `--fast`.
- [x] **M2 — first honest submission:** full grid → `select --best --promote` → `kaggle
  submit -C` → record public score vs honest estimate in the Log (the tracking datum).
- [ ] **M3 — iterate:** feature blocks + families as the leaderboard/estimate gap directs;
  file demanded workbench features as they surface (the demand list IS a deliverable).

## Log

### 2026-07-18
- Project opened (operator: "ok, let's do this as 2nd test of metis"). Arena research + the
  candidate ranking recorded in the session; S6E7 chosen. Charter note: this file lives in
  metis/workshop/projects/ (the 2026-07-17 charter — projects in the center-of-gravity repo,
  never brain).
- Demand #2 filed: metis#58 — `--sample outMinN` grammar (subsample BOTH CV levels; breaking:
  bare `--sample 3` retired). Operator-designed in the 2026-07-18 kbench#12 planning session;
  the manual dial that de-risks metis#54 (racing). Demand #1 (balanced-accuracy metric knob)
  files at kbench#12 plan Task 8.
- Demand #2 SHIPPED same-day: metis#58 closed (FIX-THEN-SHIP; actual 0.49h vs 1.21h est).
  `--sample out<M>in<N>` live — iteration runs cache-escalate into decision runs. The
  splitK/runK seam is the natural substrate for metis#54 (racing) if it's ever demanded.
- Demand #1 filed: metis#59 — train-step metric knob (balanced_accuracy) + class_weight
  passthrough. (The project header's "likely AUC" guess was corrected at kbench#12 recon:
  the S6E7 metric is balanced accuracy over an 85.9/8.4/5.8 3-class skew.) Gates kbench#12
  M2; kbench#12 M1 (workspace bring-up) is in flight on its branch.

### 2026-07-18 (M0–M2 complete — done_when MET)

- M0: operator accepted rules. M1: kbench#12 bring-up (SHIP verdict). M2: decision run
  `--sample out3` (14-config balanced-accuracy grid, cohort 79a8dea4) → honest OUTER
  **hist_gbm 0.9504 ± 0.0010** (rf 0.9437 ± 0.0008) → promoted
  `hist_gbm{cw=balanced, iter 400, leaves 31}` → live **public_score 0.94903**.
- **Honesty test: PASSED** — 0.14% gap (~1.4 SE), within the documented family-max
  optimism. The done_when is met: a live submission produced entirely by the honest flow
  on a content-pinned dataset, honest estimate recorded.
- **The demand list (the deliverable):** the competition demanded exactly TWO workbench
  features — metis#59 (balanced-accuracy metric knob + class_weight passthrough; shipped,
  0.30h) and metis#58 (`--sample outMinN` iteration dial; shipped, 0.49h). NOT demanded:
  metis#33 (GBM regularization), metis#54 (racing — the manual dial sufficed at 19m16s
  decision-run cost), per-competition step-reuse friction (the kbench layering absorbed a
  second competition with zero new abstractions; the only base-layer touch was the shared
  e2e `make_ws` extraction). The generalization thesis held.
- M3 (iterate) remains open at operator discretion: public 0.94903 vs the ~0.953 pack is a
  ~0.4% gap; candidates are per-class threshold tuning on OOF probabilities and
  missing-indicator features. Project status left `executing` for the operator to decide
  done vs iterate.
- M3 threads opened (operator direction, 2026-07-18 discussion): **metis#60**
  (predict-proba + decision step — demand #3: threshold tuning inside the seal + family
  blending) and **kbench#13** (missingness-native adapt knob + cross-fit interaction-encoding
  rungs — the features half; NaN-native verified on sklearn 1.9.0 for both families).
  Sequencing: kbench#13 is independent; #60 composes after. The arena M0–M3 structure is
  distilled as a reusable template: brain/workshop/pensive/2026-07-18-01-pensive-competition-arena-template.md
  (this project file is its worked example).
- Demand #4 filed: metis#61 — `metis debug` read-only verb family, `feature-importance`
  v0 (`--run <id> -n 10`; rf MDI / hist_gbm permutation-under-declared-metric). Motivated
  by M3's interaction-ladder protocol (extend the strongest combos, guided by what the
  winner actually used). Future members: `debug reliability`, `debug describe` (metis#5).
- kbench#13 M1 CLOSED (FIX-THEN-SHIP; fixes bundled): adapt NaN-preserving; `s6e7/features`
  step live (impute knob fit per-fold on analysis rows; cross-fit K-1 interaction encodings
  incl. missingness cells); M3 grid = 24 configs (`impute(2) × encodings(3) × 4-model
  cw=balanced foil`) on balanced accuracy. Suites: s6e7 18/18 + e2e first-run green, titanic
  54/54 no regression, independent re-verification 4/4. **Ready for the operator decision
  run** (`--sample out3`; fresh cohort by construction — adapt/features re-key). Queued
  after the run: results + honesty protocol → M2 close; kbench guard issue (session-scoped
  fingerprint; measured 24s/test tax); metis#60/#61 as the next demanded builds.

### 2026-07-19 (M3 round complete — kbench#13 closed SHIP)

- Operator decision run (24 configs, cohort 05c28781): **NaN-through beats imputation**
  (+0.14 pts marginal; top-5 all impute=none) — the round's durable win; **interaction
  encodings flat** ([] ≡ pairs ≡ pairs+triple) — the cross-fit ladder is CLOSED, no 4-way;
  rf honest jumped to parity (0.9506 ≈ hist_gbm 0.9507) on d12+NaN routing.
- Dual honest submissions: **hist_gbm public 0.94966** (honest 0.9507±0.0007, 1.5 SE —
  leaderboard gain from NaN-through realized: 0.94903 → 0.94966) · **rf public 0.94906**
  (honest 0.9506±0.0008, 1.9 SE). Honesty held for BOTH families; public ordering matched
  honest ordering. Submission diff: 99.40% agree; the 0.60% concentrated on minority-class
  boundaries — the exact rows metis#60's decision layer targets.
- Housekeeping: kbench#14 filed (session-scoped test guard; measured 24s/test tax).
- **Next (operator-framed M4): the measurement/decision layer** — metis#60 (predict-proba +
  decide step: per-class thresholds FIRST, then blending; the two compose on the same
  boundary rows).
- M5 bench pre-brainstormed (operator + session, 2026-07-19): ranked classifier candidates
  parked in brain/workshop/pensive/2026-07-19-01-pensive-s6e7-classifier-candidates-m5.md —
  CatBoost (the one mechanism bet: per-node ordered target statistics) > seed-bagging >
  ExtraTrees > LightGBM/XGBoost > deep tabular (TabPFN public 0.94756 < our GBM — evidence
  against). Demand gate for M5: the residual gap after M4 (decision grid) + metis#60 M2
  (blend).

### 2026-07-19 (M4 complete — decision layer + blend shipped)

- kbench#15 M4 decision grid run: the loss-vs-decision answer — REPLACE no (representation
  effect real: offsets+None 0.9480 < argmax+balanced 0.9494); ADD-ON-TOP family-dependent
  (gbm ≈no-op, rf +0.10 pts); parsimony picked gbm's pure decision-space winner. Honest
  OUTER: rf 0.9500±0.0004, gbm 0.9496±0.0006 (lower than M3 by REDUCED selection optimism —
  4-config pools vs 12, incumbent cell reproduces exactly).
- metis#64 closed same-day (null-rung ledger round-trip — select healed retroactively).
  metis#60 CLOSED: M1 decision layer + M2 `metis blend` (tilted-log soft vote; provenance
  guard; kaggle submit-compatible). Demands #3–#5 all shipped.
- READY FOR OPERATOR: the blend submission —
  `metis blend competition/playground-s6e7/pipelines/s6e7-sweep.md --runs point-hist_gbm-501f3358,best-rf-d4c852d3`
  (if the provenance guard refuses on mixed fingerprints: --allow-mixed, loud, or re-promote
  both under current code). Then `kaggle submit --run blend-<hash>`. Optional first: solo rf
  M4 winner submit (honest 0.9500±0.0004). M5 gate: the residual gap after blend
  (pensive: brain/workshop/pensive/2026-07-19-01-pensive-s6e7-classifier-candidates-m5.md).
