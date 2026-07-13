---
id: 000020
status: working
deps: [metis#18]
github_issue:
created: 2026-07-07
updated: 2026-07-12
estimate_hours: 1.05
started: 2026-07-12T19:10:02-07:00
---

# leakage-safe target features — internal cross-fit (features already per-fold via M1a)

## Problem

A target-based feature (e.g. group-survival, kbench#8) computed on all-train **leaks**: a test-fold
passenger's feature encodes labels of group-mates also in the test fold → inflated cv that won't
reproduce. Even *within* a training fold, using a passenger's own group to score that same passenger
leaks their own label (catastrophic for small groups). So such features need per-fold **and** internal
cross-fitting/shrinkage.

## Spec

metis-v2 M3. Under the converged design (pensive), the restructure is **already done by M1a**: `features`
lives in the `pipeline` phase → runs **per-fold structurally** (cross-fold leakage-safe, no marker, no
cv-split). This issue is then the **target-feature's own within-fold cross-fit**: a feature that uses the
label must cross-fit/shrink *internally* — exactly sklearn `TargetEncoder`'s `fit_transform` (internal CV)
or tidymodels `step_lencode_mixed` (shrinkage) — so a passenger's own label doesn't leak into their own
feature. **No engine `fit_scope` marker** (dropped as error-prone); the step owns it. **Deps metis#18.**
(Future: derive "reads the target" from a column-level data-read trace to *enforce* it — pensive.)

## Done when

- A target-based feature computed **per fold with internal cross-fit** — a leakage test shows the naive
  whole-train version inflates cv and the cross-fit version doesn't.
- Existing (non-target) features unchanged; the Titanic thread still green.

## Plan

Single-pass atomic (one review boundary, closes in one `sdlc close`). TDD. Durable plan +
full code: `workshop/plans/000020-fold-aware-features-plan.md` (3 chunks, plan-reviewed → APPROVED).

- [ ] **metis primitive** `metis/encode.py::cross_fit_target_encode` — internal K-fold cross-fit (reuses `metis.split.cv_folds`) + m-estimate shrinkage; `strategy ∈ {kfold(default), loo}` (LOO = seeded-noise + shrinkage, the safe form). Fit rows → OOF encoding; non-fit → full-fit shrunk mean.
- [ ] **metis leak test** `tests/test_encode.py` — no-signal small-group data: naive-incl-self `corr≈0.7` with own label, cross-fit `corr≈0`; real-signal counter-test (`std>0` kills return-prior cheat); LOO within-group invertibility (`1/(n−1)` gap, deterministic); fit-mask/unseen/determinism/ship-path/edge tests.
- [ ] **kbench protocol** `kbench/titanic/features.py` — `apply_features` gains `seed` + a `TARGET_GROUPS` branch (6 stateless groups byte-identical); `target_encode_group` adapter (concat train+test / mark analysis-only / call primitive / split back); register demonstrator `pclass_survival` (NOT wired into the sweep shape); thread `seed=ctx.seed` in `main()`.
- [ ] **kbench tests** — adapter OOF-train/full-fit-holdout; step-level e2e through `_run_features_step`; existing-features-unchanged regression; `TARGET_*` drift guard.
- [ ] **verify + atlas** — full metis + kbench suites green; `metis run … -dry-run` config count unchanged (demonstrator absent from shape); `atlas/experiment.md` note the primitive + protocol.

## Estimate

Two-repo atomic work: a novel-but-well-specced pure primitive (the cross-fit encoder + an 8-test
leak suite) in metis, plus a well-specced protocol extension (adapter + `apply_features` branch +
demonstrator + 4 tests) in kbench, plus atlas + the close-time fresh-eyes review. Design is
pre-resolved by the durable plan (full code in-plan, plan-reviewed) → design trends low (×0.2 spec
discount); impl scaled ×0.40 per v3.1. `greenfield-go-module` is the language-agnostic size anchor
for the primitive (real algorithmic content: cross-fit correctness, shrinkage, two strategies, edge
cases); `smaller-go-module` for the kbench extension.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.2 impl=0.25
item: smaller-go-module      design=0.1 impl=0.2
item: atlas-docs             design=0.05 impl=0.05
item: milestone-review       design=0.0 impl=0.15
design-buffer: 0.15
total: 1.05
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Log

### 2026-07-07
- Filed as metis-v2 M3. The pipeline restructure that unblocks leakage-safe features (kbench#8's ticket
  survival). Same "fold as first-class value" as metis#18. Design in the pensive.
### 2026-07-07 (design converged)
- Scope sharpened: M1a already makes features per-fold (they're in the `pipeline` phase), so cross-fold
  safety is free. This issue is now specifically the target-feature's **internal** cross-fit/shrinkage
  (sklearn `TargetEncoder` / tidymodels `step_lencode_mixed`). The `fit_scope` engine marker was dropped.

### 2026-07-12 (claim → design → durable plan → plan-review APPROVED)
- Claimed; `start-plan`; recon (metis+kbench feature/step architecture) surfaced the key fact: the
  Titanic features live in **kbench** (`kbench/titanic/features.py`), metis is the leakage-agnostic
  substrate. So #20 spans two repos: the reusable primitive in **metis** + a group-protocol extension
  in **kbench**. Operator calls: (1) primitive → `metis/encode.py` [layer-appropriate]; (2) protocol →
  separate `TARGET_GROUPS` registry (stateless groups byte-identical); (3) scope → seam + reusable
  adapter + demonstrator (`pclass_survival`), the real `ticket` feature stays kbench#8; (4) **build the
  `strategy` knob** (kfold default + loo) — operator wants cross-competition reuse, "do it right from
  the start."
- **Cross-fit strategy = K-fold cross-fit + shrinkage** (chosen over LOO): LOO's within-group
  invertibility (`enc_i=(S−y_i)/(n−1)` → survivors/non-survivors separated by a deterministic `1/(n−1)`
  gap) leaks worst for Titanic's small groups + our flexible GBM (#21). Discussed the ads-CTR parallel
  (safe via temporal disjointness + huge per-group n; Titanic has neither → needs cross-fit+shrinkage).
- **Done-when mapping:** the "naive inflates cv" clause is operationalized at the *feature level* —
  `corr(naive-incl-self, own label) ≈ 0.7` vs `corr(cross-fit, own label) ≈ 0` on no-signal small-group
  data (the *cause* of cv inflation, isolated from model/CV noise). Superior, more controllable proof.
- Durable plan authored (`workshop/plans/000020-*-plan.md`, full code in-plan) → fresh-eyes plan review
  **APPROVED (technically sound)**: traced every leak path (cross-fit self-exclusion, NaN-test-labels
  never aggregated, discriminating assertions, edge cases) — all correct. 6 advisories folded in
  (biggest: replaced a flaky LOO corr-inequality test with a deterministic invertibility test). 4 review
  lessons → `workshop/lessons.md`. est **1.05h** (v3.1, Method A; design pre-resolved by plan).

### 2026-07-12 (implemented — Chunks 1-3, TDD, both repos green)
- **metis** (branch `000020-fold-aware-features`, commit `c0c9b21`): `metis/encode.py` primitive +
  `tests/test_encode.py` (9 tests). **Leak proof landed on the nose:** naive-incl-self `corr=0.7286`
  with own label vs cross-fit `corr=-0.0351` on no-signal size-2-group data — exactly the predicted
  ≈0.7 / ≈0. Full metis suite **74 passed**.
- **kbench** (branch `000020-target-group-protocol`, commit `e488def`): `apply_features` `+seed` +
  `TARGET_GROUPS` branch (6 stateless groups byte-identical), `target_encode_group` adapter,
  `pclass_survival` demonstrator, seed threaded in `main()`; +4 tests (adapter OOF/full-fit, step-level
  e2e, stateless-unchanged regression, TARGET_* drift guard). Full kbench suite **40 passed**.
- **Thread green:** `metis run -dry-run titanic-sweep.md` → **42 configs** (= `features(6)×model(7)`,
  matches the shape's declared count), `pclass_survival` **absent** (registered, not wired — kbench#8
  wires `ticket`). Atlas: `experiment.md` gains a "Leakage-safe target features" subsection.
- **No plan deviations.** `metis` binary built via `go build -o bin/metis ./cmd/metis` for the dry-run;
  `-dry-run` flag precedes the positional arg. Close boundary + `Review-Verdict` handled in main session.
