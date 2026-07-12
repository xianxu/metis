# Boundary Review — metis#21 (whole-issue close)

| field | value |
|-------|-------|
| issue | 21 — GBM model branch — HistGradientBoosting model step-type |
| repo | metis |
| issue file | workshop/issues/000021-gbm-model.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9b1b28b871b74d6b5ada4415d9e5dea879b4f319..HEAD |
| command | sdlc close --issue 21 |
| reviewer | claude |
| timestamp | 2026-07-11T22:27:52-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All green: **Python 50 passed**, **Go sampler + cmd/metis ok** — the "zero Go edits" claim holds (structural `FamilyOf` derivation, pre-existing `configs_test.go "gbm"` family test covers it). I have everything needed for the verdict.

```verdict
verdict: SHIP
confidence: high
```

This is a tight, single-purpose extension that adds `hist_gbm` (sklearn `HistGradientBoostingClassifier`) at the three single-source touch points in `metis/model.py` (`MODELS`, `make_model`, `complexity`), with a literature-grounded complexity measure (total leaves summed across boosted trees — the deliberate inverse of rf's n_estimators-neutral mean). Every Done-when item is delivered and independently verified: the sweep expands correctly (dry-run → **33 configs, 12 hist_gbm**, `learning_rate` fixed/not swept as designed), the model factory + complexity are unit-tested, the per-fold step seam emits complexity for hist_gbm, and the "zero Go edits" claim is confirmed (Go tests pass; `FamilyOf` derives the family structurally from the `$any`-map tag). Nothing blocks SHIP; the findings below are all Minor.

**1. Strengths**
- **Single-source discipline held (ARCH-DRY / ARCH-PURPOSE).** The change touches only the three Python enumeration points; the Go layer derives the family structurally. I verified this isn't just a claim: `pkg/sampler/configs_test.go:19` already exercises a `"gbm"` family, and `go test ./pkg/sampler/ ./cmd/metis/` passes with no diff. Shadow-sweep clean — no hand-maintained restatement of the model set left behind.
- **Complexity measure is correct and defended.** `complexity(fitted, "hist_gbm")` reads *realized* leaves from the fitted object (not configured `max_iter × max_leaf_nodes`), matching the existing rf/logreg pattern of measuring realized capacity. The sum-not-mean rationale is sound and the docstring (`metis/model.py:83-89`) documents both the reasoning and the ν-shrinkage caveat.
- **The caveat is contained structurally, not just documented.** The baseline shape fixes `learning_rate: 0.1` and sweeps only `max_iter × max_leaf_nodes` (`testdata/experiment/titanic-baseline-shape.md:27`), placing the sweep in the fixed-ν stratum where total-leaves is a clean monotone DoF proxy. Dry-run confirms `learning_rate` is absent from the swept free-params. This is the right ARCH-PURPOSE move: deliver the axis where the measure is valid, defer the ν-weighted measure with a documented trigger.
- **Test pins real behavior, not the implementation.** `test_complexity_hist_gbm_total_leaves` (`tests/test_model.py:107`) asserts both the exact sum *and* max_iter-monotonicity (`m40 > m10`) — the latter is the load-bearing property (it's what makes the parsimony rule prefer fewer rounds) and can't be faked by a mock.

**2. Critical findings** — none.

**3. Important findings** — none. (No README.md exists in the repo → no README gate finding. `atlas/experiment.md` + `atlas/index.md` both updated for the new surface.)

**4. Minor findings**
- `metis/model.py:99` — `complexity` reads the private `fitted._predictors` attribute. I confirmed sklearn 1.9.0 exposes **no public** predictors accessor (`_predictors` is the only one), so this is unavoidable for per-tree realized leaf counts — but it's fragile across sklearn upgrades. The inline comment acknowledges the shape; consider also noting the sklearn version this was verified against, so a future upgrade regression is diagnosable.
- `tests/test_model.py:108` — the `test_complexity_hist_gbm_total_leaves` docstring cites `(metis#19)` while this is metis#21 work (every other reference — atlas, `test_steps.py:114` — uses #21). Trivial citation drift; `#19` is the complexity-measure lineage so it's not wrong, just inconsistent.
- Multiclass path not exercised: the flatten `for stage in _predictors for t in stage` correctly handles K>1 predictors-per-stage, but every test dataset is binary (1 tree/stage), so the K>1 branch is unverified. Low priority — the target domain (Titanic) is binary and the logic is trivially correct — but a passing note for whenever a multiclass shape appears.

**5. Test coverage notes**
- Coverage is proportionate and hits the bug classes that matter: shape/determinism (parametrized), hyperparam-reaches-estimator, exact-sum + monotonicity, and the end-to-end per-fold step emitting `complexity > 0`. 50/50 Python pass, 23/23 in the two touched files.
- One gap versus rf's coverage: there's no hist_gbm analogue of `test_hyperparams_change_the_fit` (proving distinct hyperparams → distinct CV scores). Low value here — `test_make_model_applies_hyperparams` already proves the params reach the estimator and the mechanism is shared with rf — so not worth adding unless you want symmetry.

**6. Architectural notes for upcoming work**
- The Spec flags XGBoost/LightGBM as optional later additions. At 3 kinds the triple `if kind == …` dispatch across `MODELS`/`make_model`/`complexity` is correctly Simplicity-First (a registry now would be over-engineering). But note the threshold: once a 4th/5th kind lands, a dict-dispatch registry keyed by kind (constructor + complexity fn) would collapse the three parallel switch chains into one source of truth. Reassess then, not now.
- The deferred ν-weighted complexity measure is real future work, not a loose end: it becomes necessary the moment a shape sweeps `learning_rate`. The docstring's trigger ("a ν-sweeping shape would need a ν-weighted measure") is the right measure-before-rebuild posture — the follow-up issue should reference it when a real sweep exposes a misranking.

**7. Plan revision recommendations** — none. All five Plan checkboxes are delivered and the code matches the Spec exactly (3-kind model set, total-leaves complexity, fixed-ν shape, 33-config sweep). The plan still describes the code faithfully; no `## Revisions` entry needed.
