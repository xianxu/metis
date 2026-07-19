---
id: 000065
status: codecomplete
deps: []
github_issue:
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 1.36
started: 2026-07-19T01:03:44-07:00
actual_hours: N/A
---

# arena2 model bench: ensemble kind + catboost + seed passthrough

## Problem

Arena2's residual gap to the ~0.953 pack (our best public 0.94966) has, by the M3/M4
ledger, two independent tree families CONVERGED to honest OUTER ~0.9506 — the signature of
a data noise floor at the single-model level. Two operator-directed moves remain untried:

1. **Blend, measured honestly.** `metis blend` (metis#60 M2) is a post-hoc soft-vote over
   PROMOTED runs — leaderboard-only, **no in-sweep OOF**, so it cannot answer "does blending
   help the OUTER CV" without spending a submission slot. The honest way to measure a blend
   is to make it a config the nested-CV sweep scores like any model.
2. **A new mechanism + cheap variance reduction** (the M5 bench pensive): CatBoost — the one
   real mechanism argument (per-node ordered target statistics; the sharper test of the
   "cell signal matters *conditionally*" hypothesis the flat global encoding closed at M3;
   the most-different boosting bias → best blend partner) — and seed-bagging the incumbent,
   which needs only a `params.seed` override to unlock.

These are three additions to the pure model core (`metis/model.py`), each Python-only: the
Go layer derives the family structurally (`FamilyOf` reads the `$any`-map branch label), and
the atlas records "adding a model kind is Python-only (MODELS + make_model + complexity)".

## Spec

**M1 — `ensemble` model kind (the blend, made honestly measurable).** A composite kind whose
`params` carry `members: [<$any-map bundle>, …]` (+ optional `weights`), built as an sklearn
`VotingClassifier(voting="soft")`. Because it exposes `fit`/`predict`/`predict_proba`/
`classes_` like any estimator, it composes UNCHANGED with: the decision layer (metis#60 —
offsets tune on the ensemble's AVERAGED probabilities, the correct place for a blend's
decision tilt), the metric knob (metis#59), the nested-CV seal, and the parallel executor.
`complexity` = SUM of member realized complexities (aggregate capacity; the parsimony axis).
Members are parsed by `parse_model_config` (ARCH-DRY — the same normalizer the top-level
model knob uses; the ensemble recurses one level). **DRY complexity recovery (plan-review
finding #2):** each `VotingClassifier` member is NAMED by its kind (the `parse_model_config`
label, suffixed `-<i>` for uniqueness), so `complexity(ensemble, "ensemble")` recovers each
member's kind from the member NAME and recurses through the existing per-kind dispatch — NO
estimator-type→kind reverse map (which would be a second source of truth for kind identity).

This is NOT a parallel mechanism to `metis blend` (feedback_minimum_mechanism): the ensemble
KIND *measures* a blend honestly inside the sweep (an OOF estimate); `metis blend` *combines*
heterogeneous cross-cohort PROMOTED artifacts post-hoc (different fingerprints, no shared
fold structure). They share the soft-vote math, which sklearn owns here — no code duplicated.

**seed passthrough (part of M1 — see Revisions).** `make_model` currently pins
`random_state = ctx.seed` for every kind. Add a single `eff_seed = params.get("seed", seed)`
override applied at each estimator (random_state / — M2 — CatBoost random_seed / passed down
to ensemble members). Absent `seed` = byte-identical to today (no cohort re-key). Present
`seed` re-keys the leaf (it rides `with.model` → Kpre), so a shape can sweep seed as a
dimension — and an `ensemble` of one config × several seeds IS seed-bagging (the two features
compose; this is why seed passthrough lands WITH ensemble in M1, so seed-bagging is testable
there — plan-review finding #3).

**M2 — `catboost` kind.** New MODELS branch + dependency (cp312 macOS wheel confirmed).
`make_model` maps `class_weight: "balanced"` → `auto_class_weights="Balanced"` (CatBoost's
spelling of the same inverse-frequency reweighting); `predict_proba` + `random_seed` mean the
decide layer, metric knob, and seed override all compose. **Purity/determinism pins
(plan-review finding #1, ARCH-PURE):** `allow_writing_files=False` (CatBoost's default writes
a `catboost_info/` dir — an IO side-effect inside the pure core), `logging_level="Silent"`
(no stdout), and a fixed `thread_count` (determinism under the single-thread-per-leaf
invariant, metis#48). `complexity` = `tree_count × 2^depth` (oblivious/symmetric trees are
full binary at fixed depth — the total-leaves capacity proxy, analogous to hist_gbm's summed
leaves). CatBoost's `.predict()` may return shape `(n,1)` — ravel at the make_model boundary
so the generic `predict()` stays 1-D.

## Done when

- `ensemble` trains + scores through `cv_score`/`fold_fit` (incl. `decide=offsets`) and
  reports a finite `complexity`; unit tests cover the soft-vote average, member parsing,
  decide composition, and complexity = sum-of-members.
- `catboost` trains/predicts deterministically, honors `class_weight: balanced` and a
  `params.seed` override, reports `complexity`; `params.seed` override verified to change the
  fit and re-key (distinct from ctx.seed).
- `pytest` green (metis suite); `go build -o bin/metis ./cmd/metis` clean (no Go edits
  expected — FamilyOf is structural; a conformance check confirms zero-edit).
- A kbench SMOKE run exercises the `ensemble` kind end-to-end through the real step/forkserver
  path (the unit tests cover the pure core; the smoke covers the seal+fork seam) before M1
  milestone-close.

## Plan

- [x] **M1** — `ensemble` kind + seed passthrough in `metis/model.py`: make_model
  VotingClassifier-soft (members named `<kind>-<i>`), member parse via parse_model_config,
  complexity=sum via member-name dispatch, MODELS += ensemble, `params.seed` override
  (`eff_seed`) at each estimator. Unit tests: soft-vote average = member mean, WEIGHTS tilt,
  single-member ≈ bare model, complexity = sum-of-members, decide=offsets composition,
  `params.seed` re-keys + changes fit, **seed-bagging (distinct member seeds → distinct
  fitted members)**. Metis step-path test (train step over a fixture with an ensemble config).
  `sdlc milestone-close --milestone M1`.
- [x] **M2** — `catboost` dep (uv add) + kind (make_model with `allow_writing_files=False` +
  `logging_level="Silent"` + fixed `thread_count`, class_weight→`auto_class_weights`, predict
  ravel; complexity = tree_count×2^depth) + unit tests (determinism, balanced maps through,
  complexity finite, catboost usable as an ensemble member). `sdlc milestone-close --milestone M2`.

## Revisions

### 2026-07-19 — plan-review findings adopted (change-code plan-quality: INFO)
The plan-quality judge passed INFO (safe to start) with three non-blocking refinements, all
folded in before implementation:
1. **CatBoost purity/determinism pins (ARCH-PURE)** — pin `allow_writing_files=False`,
   `logging_level="Silent"`, fixed `thread_count` in make_model (M2 spec updated).
2. **DRY complexity recovery** — recover member kind from the VotingClassifier member NAME
   (derived from the single `parse_model_config` source), NOT an estimator-type→kind reverse
   map (M1 spec updated).
3. **Seed-bagging testability** — seed passthrough MOVED M2→M1 so an ensemble with distinct
   member seeds (seed-bagging, the point of composing seed×ensemble) is testable in M1; M2 is
   now purely the CatBoost external-dep add (isolating that risk — the split's stated rationale).

### 2026-07-19 — M2 milestone-close FIX-THEN-SHIP findings addressed
Boundary review verdict FIX-THEN-SHIP (high confidence, no Critical; two Important + minors),
all folded into the M2 close commit (#174):
1. **Ravel LOCATION reconciled (Spec correction):** the Spec §M2 said "ravel at the make_model
   boundary"; the implementation correctly ravels at the single `predict()` call site (DRY —
   one site, no-op for sklearn kinds). The Spec described a location the code doesn't use.
2. **Predictions dtype (Important #1):** CatBoost's `.predict()` CAN return float labels
   ("0.0") — a ship-submission drift (scoring unaffected). PROBED: on catboost 1.2.10 with our
   int64 target it already returns int64, so it doesn't reproduce here — but `predict()` now
   defensively `.astype(classes_.dtype)` (a no-op for sklearn kinds and int64 catboost) so the
   label dtype is GUARANTEED regardless of catboost version/config.
3. **Fold-scoring test gap (Important #2):** added `test_catboost_fold_score_path_and_int_labels`
   — exercises `fold_score` (the nested-CV path arena2 consumes) for catboost AND asserts INT
   labels (`dtype.kind in "iu"`), pinning both the path and the dtype guarantee.
4. **Minors:** atlas/index.md complexity enumeration now lists catboost + ensemble; the
   `depth` (catboost, exact oblivious depth) vs `max_depth` (rf/gbm, a cap) asymmetry is left
   UN-aliased deliberately — the two mean different things, so aliasing would be semantically
   wrong; the sweep uses `depth` for catboost directly.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: typed-data-prototype  design=0.15 impl=0.25
item: typed-data-prototype  design=0.1 impl=0.2
item: real-api-discovery    design=0.05 impl=0.15
item: milestone-review      design=0.0 impl=0.2
item: milestone-review      design=0.0 impl=0.2
design-buffer: 0.2
total: 1.36
```

(Item legend: M1 ensemble kind · M2 catboost kind · catboost dep/API + seed passthrough ·
two milestone-close reviews.)

(Two patterned model-kind additions + a one-site seed override, over an established
dispatch (`make_model`/`complexity`/`MODELS`); the only genuine discovery is the CatBoost
API surface + dependency. Buffer 0.2: new external dep + the VotingClassifier×decide-layer
composition is the one unpatterned seam. Method A; estimate-source stale + ariadne base
moving — flagged at start-plan.)

## Log

### 2026-07-19
- 2026-07-19: closed — Both milestones shipped (M1 ensemble+seed SHIP, M2 catboost FIX-THEN-SHIP with findings fixed): 124 pytest green; go build -o bin/metis clean (zero Go edits, FamilyOf structural); real-binary+forkserver smokes for ensemble, solo catboost, and catboost-in-ensemble all split→train→predict ok. Enables arena2 M4-blend (ensemble outer-CV) + M5 (catboost/seed-bag). --no-actual: interleaved metis+kbench session (active-time contamination).; review verdict: SHIP
- 2026-07-19: closed M2 — catboost kind: 123 pytest green (6 catboost — 1-D predict ravel, determinism+seed-override, balanced-changes-fit+loud-unknown, complexity=tree_count×2^depth, no-side-effect-dir/ARCH-PURE, catboost-as-ensemble-member); go build -o bin/metis clean; real-binary+forkserver smoke solo AND catboost-in-ensemble split→train→predict ok, predictions well-formed 1-D, NO catboost_info/ leak (purity pin holds); atlas updated (catboost surface). --no-actual: interleaved metis+kbench session.; review verdict: FIX-THEN-SHIP
- 2026-07-19: closed M1 — ensemble kind + seed passthrough: 116 pytest green (soft-vote=member-mean, weights tilt, single-member≈bare, complexity=Σmembers, decide=offsets composition, seed override re-keys+changes-fit, seed-bagging distinct members, ensemble-through-step-path); go build -o bin/metis clean (zero Go edits — FamilyOf structural); real-binary+forkserver smoke on toy ensemble shape split→train→predict ok; atlas updated (ensemble+seed surface). --no-actual: session interleaves metis+kbench issues.; review verdict: SHIP
- Opened + claimed. Enables arena2 M4-blend (ensemble outer-CV measurement) + M5 (catboost
  bench). Sibling kbench issue runs the sweeps. Cross-repo: kbench runs against the LOCAL
  metis tree, so metis need not be merged before the kbench smoke/runs (the merge is the
  publish, not the execution dep).
- **M1 done** — `ensemble` kind + seed passthrough in model.py (3 files: model.py,
  test_model.py, test_steps.py). VotingClassifier(soft), members named `<kind>-<i>`,
  complexity=Σ via member-name dispatch (DRY, no type-map), `params.seed` override. Tests:
  soft-vote=member-mean, weights tilt, single-member≈bare, complexity=sum, decide=offsets
  composition, seed override re-keys+changes-fit, seed-bagging distinct members, ensemble
  through the step path. 116 pytest green; `go build` clean (zero Go edits — FamilyOf derives
  `train.model=ensemble` structurally). Real-binary+forkserver smoke on a toy ensemble shape:
  split→train→predict all ✓. Test-data note: seed-effect tests need NON-separable data
  (rf bootstraps converge to identical hard preds on trivially-separable data) + predict_proba
  comparison — a lesson.
- **M2 done** — `catboost` kind (5 files touched incl. pyproject/uv.lock for the dep).
  make_model catboost branch (lazy import; iterations/depth/learning_rate; class_weight
  balanced→auto_class_weights; ARCH-PURE pins allow_writing_files=False + logging_level Silent
  + thread_count=1); `model.predict` ravels (n,1)→1-D at the one call site; complexity =
  tree_count×2^depth. 123 pytest green (6 catboost: 1-D predict, determinism+seed-override,
  balanced-changes-fit + loud-unknown, complexity formula, no-side-effect-dir, catboost-as-
  ensemble-member). go build clean. Real-binary+forkserver smoke: solo catboost AND
  catboost-in-ensemble both split→train→predict ✓, predictions well-formed 1-D, NO
  catboost_info/ leak (the purity pin holds). CatBoost 1.2.10 (cp312 macOS universal2 wheel;
  pulled matplotlib/plotly/graphviz as transitive deps).
