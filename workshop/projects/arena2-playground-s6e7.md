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
- [x] **M3 — iterate:** DONE — ran four issue-level rounds under this umbrella: M3-features
  (kbench#13: NaN-through won, interaction ladder flat), M4-decision (kbench#15: loss-vs-decision
  answer), M4-blend (kbench#16: ensemble soft-vote outer-CV), M5-bench (kbench#16: catboost +
  seed-bag). The demand list held: the competition demanded only model-layer additions the
  workbench absorbs Python-only (ensemble/catboost/seed kinds — zero Go edits).
- [x] **M6 — noise floor DECLARED (kbench#17, 2026-07-19):** the model-space was already
  EXHAUSTED; M6 tested the two feature/data levers and both hit the floor. Non-tree class on
  clinical features — decisive NO (by-class minority-recovery gate: recovers ~6-8% of true-minority
  consensus errors, hurts the blend). Feature-v2 through the honest flow (`s6e7-feateng-v2.md`,
  each family at its own best hyperparameters) — FLAT (gbm Δ≈0). **Noise floor confirmed across
  model classes + features + capacity.** The gap is a DATA/source-augmentation problem, not a
  model or feature one. done_when was met at M2; M6 is the honest close of the model+feature space.
- [ ] **M7 — chase the leaderboard (explicitly off-mission, operator-sanctioned 2026-07-19):**
  the generalization thesis is proven; M7 is a deliberate gap-chase whose learnings are LESS
  generalizable. Diagnostic-first (§ M7 plan below): step 1 = two cheap, submission-free
  diagnostics (adversarial validation for train/test shift; generator forensics for
  dupes/quantization/ID-order leaks); step 2 (gated) = source-dataset augmentation.

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

### 2026-07-19 (M4-blend + M5 — the model bench, autonomous overnight run)

- **metis#65 (SHIP, merged PR#47):** shipped the enabling model kinds Python-only (zero Go
  edits — `FamilyOf` derives family structurally): `ensemble` soft-vote (the blend made
  scorable INSIDE nested CV — an honest OOF, vs `metis blend`'s post-hoc leaderboard-only
  combine), `catboost` (the M5 mechanism bet), and seed passthrough (`params.seed` → seed-bagging
  when composed with `ensemble`). 124 pytest, real-binary/forkserver smokes; two milestone
  reviews (M1 SHIP; M2 FIX-THEN-SHIP caught + fixed a catboost predict-dtype ship-path guard).
- **M4-blend (kbench#16 M1, cohort 5b3f38ee, out3):** ensemble-BLEND {rf-bal, gbm-bal}
  **0.9507±0.0010** > rf 0.9500±0.0004 > gbm 0.9496±0.0006. Highest POINT estimate (+0.0007
  over rf, at the minority-class boundaries M4's diff localized; offsets-on-blend won all 3
  folds) but WITHIN 1 SE at out3 → not distinguishable from rf; reliability tiebreak ships rf.
  A small, noise-level yes — matches the pre-registered "modest ~0.001-0.002".
- **M5 bench (kbench#16 M2, cohort 56303278, out3):** seed-bag **0.9510±0.0009** ≈ gbm
  0.9508±0.0008 > catboost **0.9501±0.0007** > rf 0.9493±0.0007. **CatBoost does NOT beat the
  incumbent** — a genuinely different boosting mechanism finds no new signal → strong
  confirmation of the data noise floor. Seed-bag adds only within-noise +0.0002.
- **The synthesis (why M6 is a feature/data thread, not a model thread):** across M3→M5, EVERY
  model-space lever — capacity, class-weight, decision offsets, a diverse-family blend, a new
  boosting mechanism (catboost), variance reduction (seed-bag) — converges to honest OUTER
  **~0.950-0.951**. Two independent families agreeing to the 4th decimal, and a third mechanism
  landing in the same band, IS the data noise floor (synthetic-data label noise). The ~0.003
  gap to the ~0.953 pack is therefore almost certainly NOT reachable by model changes.

### M6 — the gap-closing hypotheses (ranked; operator go/no-go)

The model space is exhausted. If the gap is worth chasing, in likelihood order:
1. **Source-dataset augmentation (the classic Playground move).** S6E7 is synthetic, generated
   from a real "student health" dataset; top Playground finishers routinely concat the ORIGINAL
   dataset (found via the competition's "generated from" note) into training — it adds real
   signal the synthetic generator smoothed out. Legal but tests PROVENANCE, not the workbench —
   deprioritized as off-mission all arena, but it's the single most likely gap-closer. Needs a
   separate shape (different estimand) + importance weights (metis#63, the dormant `weight` role).
2. **Confirm the blend edge at out10 (cheap, high-info).** M4-blend 0.9507 is within 1 SE at
   out3; a full k=10 run (cache-escalates from 5b3f38ee's 3 folds) tightens SE ~1.8× — decides
   whether the blend edge is real or noise before investing in a wider blend.
3. **Stacking over soft-vote.** A logistic meta-learner on the base models' OOF probabilities
   (vs the current unweighted average) can weight members by reliability. Needs a new metis
   `stack` mechanism (OOF-proba meta-fit inside the seal) — a real workbench feature, only worth
   it if #2 shows blending has headroom.
4. **Feature engineering round 2.** The M3 flat-encoding ladder closed, but untried: aggregate/
   ratio features over the 13 raw columns, or a missingness-count feature. Lower prior (the
   noise-floor evidence argues features are saturated), but cheap to probe.
5. **NOT worth it:** deeper single-model tuning (converged), deep tabular (TabPFN public 0.94756
   < our GBM — evidence against), more capacity (saturated at M2).

Recommendation: if the operator wants the leaderboard, **#1 (source augmentation)** is the move,
accepting it's a provenance experiment, not a workbench one. If the goal stays "prove the
workbench generalizes" (the project's actual done_when, met at M2), arena2 is **DONE** — the
model bench is the honest, complete answer: the workbench measured the noise floor correctly.

### M6 update — the cascade diagnostic + out10 confirmation (2026-07-19, operator design session)

The ranked list above was refined by two diagnostics run this session (kbench#17 is the plan of
record; scripts in `kbench/competition/playground-s6e7/analysis/`):

- **#2 RESOLVED — the blend edge is NOISE.** The out10 confirmation (`s6e7-blend-out10.md`,
  `--sample out10in5`, cohort 460e219c): ensemble **0.9496±0.0006** > rf 0.9492±0.0005 > gbm
  0.9483±0.0006. The blend's point edge over rf shrank to +0.0004 (within 1 SE) → `select` ships
  rf. Everything dropped ~0.001 vs out3 because folds {0,1,2} were a favorable subsample
  regressing to the honest mean (an `--sample out<M>` optimism datum). The blend does not beat
  solo rf even at full k=10.
- **The cascade diagnostic KILLED routing/blending/specialists over trees** (`cascade_diag.py`,
  5-fold OOF on all 690k rows): 87% of gbm's errors are CONSENSUS errors (rf+catboost also wrong
  — correlated axis-aligned bias); the recoverable 12.8% is UNROUTABLE (oracle 0.661 needs
  per-row omniscience; best real router −0.0010) AND METRIC-ORTHOGONAL (the uncertain rows are
  majority-heavy, balanced accuracy is minority-driven). So #3 (stacking) is bounded by the same
  oracle ceiling — a smarter combiner over these trees won't move the metric.
- **The one open lever, sharpened (kbench#17):** the diagnostic only tested TREES, so a DIFFERENT
  model CLASS (non-tree: logreg/MLP) whose bias sits off the trees' shared failure mode is
  untested — and it's synergistic with #4 (clinical features: a linear model NEEDS them; trees
  get univariate non-linearity free). The decisive gate is a **minority-class consensus-error
  recovery** diagnostic (does a non-tree on engineered features recover the at-risk/unhealthy
  rows all three trees miss?) BEFORE any build. Features grounded in the health literature
  (U-shaped sleep, BMI J-curve + fat-but-fit `bmi×activity`, step dose-response, RHR threshold).
- **Tooling shipped this session (metis#66, merged PR#48):** fold-ordered scheduling (live per-fold
  mean±SE + board Q graceful-finalize; shipped as `--live`, then graduated to the DEFAULT in
  metis#67 — flag removed) + `--auto-stop` (incumbent-referenced loser-stop) — future arena runs
  get live partial estimates and can auto-drop losing configs.

**Standing status (post-M6):** the model+feature space is CLOSED and honest — M6 confirmed the
noise floor across model classes (kbench#17 non-tree gate = NO) AND engineered features
(`s6e7-feateng-v2.md` = flat) AND capacity. done_when was met at M2. The only lever that adds
INFORMATION is source-dataset augmentation → M7.

### M7 — the leaderboard-gap chase (opened 2026-07-19, operator-sanctioned off-mission)

Framing: the workbench-generalization thesis is proven, so M7's learnings are deliberately LESS
generalizable — it's a gap-chase, run diagnostic-first (the operator's discipline). Ranked from a
fresh-context brainstorm (codex, file-grounded, 2026-07-19; both the model+feature exhaustion and
the "gap is an INFORMATION problem" diagnosis constrain it).

**Step 1 — two cheap, submission-free diagnostics FIRST** (`analysis/m7_shift_forensics.py`):
- **Adversarial validation** — train a classifier `train-rows vs test-rows` on the 13 features +
  missingness flags; the CV AUC measures covariate shift. `<0.55` → no shift (abandon test-weighting);
  `0.55-0.65` → mild density-ratio importance weights (clip 0.5-2.0); `>0.65` → real shift, weighting
  becomes high-value. Adds test-distribution leverage without new labels.
- **Generator forensics** — exact + near-duplicate counts (within train, within test, across
  train↔test); label purity of dup groups; per-feature quantization/decimal patterns by class;
  ID/row-order class-prior drift; missingness-pattern label rates. Exploits generator artifacts if
  the synthetic process leaked structure (non-model information). Usually negative, cheap, sometimes
  decisive on synthetic tabular.

**Step 2 — source-dataset augmentation (gated on step 1 + the source hunt)** — the highest-prior
gap-closer and the ONE information-adding lever. Sequence:
0. **Confirm the source** — the S6E7 competition OVERVIEW almost certainly names it (Playground
   pattern: "generated from a deep learning model trained on the [X] dataset"). NOT yet fetched
   (the README/data don't state it) — step 0 is reading the overview to get the named dataset +
   locating it (Kaggle/UCI). If unnamed → adversarial-search or abandon.
1. **Schema-align** the original to the 13-feature adapted schema (column/unit match; classes to
   {at-risk, fit, unhealthy}).
2. **Distribution diagnostic** — adversarial validation synthetic-train vs original + per-class
   feature-distribution compare (shift magnitude → whether naive concat helps or needs weighting).
3. **Orthogonality diagnostic** — does an original-ONLY model recover the M6 consensus-error rows
   (true fit/unhealthy)? If its errors are orthogonal to the trees' shared mode → real added signal.
4. **Weighted-concat OOF gate** — synthetic + original at source-weights {0.1, 0.25, 0.5, 1.0},
   honest nested-CV, per-class recall. **Proceed only if +0.0015 on the same folds, esp. minorities.**
5. **Ship** (iff positive) — wire into the honest flow (needs the dormant metis#63 `weight` role for
   importance weights), full sweep → promote → submit. If the source is heavily shifted, use it as
   TEACHER/calibration (source-model probs as meta-features, or source class-centroid/KNN features)
   rather than raw concat.

Lower-priority / likely-noise (codex-flagged, gated behind step 1-2): pseudo-labeling the test set,
LB threshold-probing, submission ensembling, rule-mining, TTA, more thresholds. Run only if a
diagnostic surfaces signal; do not burn submissions speculatively.
