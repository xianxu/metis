---
id: 000065
status: working
deps: []
github_issue:
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 1.36
started: 2026-07-19T01:03:44-07:00
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
model knob uses; the ensemble recurses one level).

This is NOT a parallel mechanism to `metis blend` (feedback_minimum_mechanism): the ensemble
KIND *measures* a blend honestly inside the sweep (an OOF estimate); `metis blend` *combines*
heterogeneous cross-cohort PROMOTED artifacts post-hoc (different fingerprints, no shared
fold structure). They share the soft-vote math, which sklearn owns here — no code duplicated.

**M2 — `catboost` kind + seed passthrough.**
- `catboost` — new MODELS branch + dependency (cp312 macOS wheel confirmed). `make_model`
  maps `class_weight: "balanced"` → `auto_class_weights="Balanced"` (CatBoost's spelling of
  the same inverse-frequency reweighting); `predict_proba` + `random_seed` mean the decide
  layer, metric knob, and seed override all compose. `complexity` = `tree_count × 2^depth`
  (oblivious/symmetric trees are full binary at fixed depth — the total-leaves capacity
  proxy, analogous to hist_gbm's summed leaves).
- **seed passthrough** — `make_model` currently pins `random_state = ctx.seed` for every
  kind. Add a single `eff_seed = params.get("seed", seed)` override applied at each estimator
  (random_state / CatBoost random_seed / passed down to ensemble members). Absent `seed` =
  byte-identical to today (no cohort re-key). Present `seed` re-keys the leaf (it rides
  `with.model` → Kpre), so a shape can sweep seed as a dimension — and an `ensemble` of one
  config × several seeds IS seed-bagging (the two features compose).

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

- [ ] **M1** — `ensemble` kind in `metis/model.py` (make_model VotingClassifier-soft, member
  parse via parse_model_config, complexity=sum, MODELS += ensemble) + unit tests; kbench
  ensemble-smoke through the step path; `sdlc milestone-close --milestone M1`.
- [ ] **M2** — `catboost` dep (uv add) + kind (make_model, complexity, class_weight→auto) +
  `params.seed` passthrough in make_model + unit tests (catboost determinism, seed override
  re-keys, balanced maps through); `sdlc milestone-close --milestone M2`.

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
- Opened + claimed. Enables arena2 M4-blend (ensemble outer-CV measurement) + M5 (catboost
  bench). Sibling kbench issue runs the sweeps. Cross-repo: kbench runs against the LOCAL
  metis tree, so metis need not be merged before the kbench smoke/runs (the merge is the
  publish, not the execution dep).
