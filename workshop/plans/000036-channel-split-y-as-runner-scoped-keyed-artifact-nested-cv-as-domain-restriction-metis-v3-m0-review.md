# Boundary Review — metis#36 (milestone M0)

| field | value |
|-------|-------|
| issue | 36 — channel split: y as runner-scoped keyed artifact — nested CV as domain restriction (metis-v3) |
| repo | metis |
| issue file | workshop/issues/000036-channel-split-y-as-runner-scoped-keyed-artifact-nested-cv-as-domain-restriction-metis-v3.md |
| boundary | milestone M0 |
| milestone | M0 |
| window | 2a63ea764433eca238dc7855635aadc571d3a6e6^..HEAD |
| command | sdlc milestone-close --issue 36 --milestone M0 |
| reviewer | claude |
| timestamp | 2026-07-19T19:32:11-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: SHIP
confidence: high
```

The M0 deliverable — regression support in the pure modeling core (`metis/model.py`) — is clean, correct, exemplary-DRY, and well-tested; nothing blocks the boundary. **Scope note up front:** the review window (`2a63ea76^`..HEAD) mechanically spans ~30 prior issues' worth of already-shipped, already-boundary-reviewed work (#30–#67 — board/forkexec/autostop/prioritysem/blend/fingerprints/decision-layer/…). The *only* thing this boundary claims is M0 = "regression support in the modeling core," so I scoped my substantive review to the actual M0 change surface (`metis/model.py` + `tests/test_model.py` + `atlas/experiment.md`/`index.md` + the plan). I read `metis/model.py`, `metis/split.py`, and the regression tests directly from the tree (they match the diff). Caveat: the Bash tool was unavailable this session (harness `EPERM` creating `session-env`), so I could **not** execute the suite — I verified the 12 regression cases by tracing each branch against sklearn's real API and confirming the tests pin real values, not by running `pytest`.

**1. Strengths**
- **ARCH-DRY, textbook.** Regression reuses the classifier measures rather than forking them: `complexity` shares branches `("rf","rf_reg")`, `("logreg","ridge")`, `("hist_gbm","hist_gbm_reg")` (`metis/model.py:296-306`); `predict` shares one path with a single `is_regression`-equivalent guard (`:274`); `fold_fit`/`fold_score`/`cv_score` are unchanged and thread through. `is_regression` is the ONE routing predicate (`:26-29`). No parallel regression code path was created.
- **ARCH-PURE.** The whole surface is deterministic, IO-free, and unit-tested on in-memory arrays; no IO leaked into `model.py`.
- **Tests pin real logic, not mocks.** `test_regression_complexity_mirrors_classifier_measures` asserts the exact leaf/coef formulas + `== 5.0` (`tests/test_model.py:618-629`); `test_rmse_scorer_value_and_perfect` computes RMSE=1 by hand; `test_regression_train_predict_continuous` asserts `dtype.kind=="f"` + corr>0.5 (would fail if the `classes_` guard were missing); `test_decide_offsets_rejected_for_regressor` pins the loud-refusal message. None are vacuous.
- **Correct scoping.** M0 is genuinely the pure core; the engine run and `predict.py` submission path are explicitly deferred to M1 "where the data is," documented in both plan and atlas — a legitimate milestone split, not a deferred purpose (ARCH-PURPOSE passes).
- No classification regression: every M0 addition is additive (MODELS ∪ 3 kinds; new `complexity`/`make_model` branches; the `if not hasattr(classes_)` early-return fires only for regressors).

**2. Critical findings**
None.

**3. Important findings**
- **Traceability of this gate (not a code defect).** `metis#36` M0 close · window covers far more than M0. Because BASE (`2a63ea76^`) sits ~30 issues behind HEAD, a mechanical read of "SHIP at M0 close" would imply the entire #30–#67 body was re-reviewed here — it was not (those closed at their own boundaries, per `workshop/lessons.md`). Treat this SHIP as certifying the **M0 regression core only**. Worth a one-line `## Log` note on the issue recording the actual reviewed surface, so the verdict isn't later read as blanket coverage.

**4. Minor findings**
- `atlas/index.md` complexity one-liner lists `catboost`/`ensemble` but omits the regressor variants (`rf_reg`/`hist_gbm_reg`/`ridge`); `atlas/experiment.md:127-136` documents them fully, so the docs gate is satisfied — this is only a consistency nit in the index summary.
- `fold_fit`/`fold_score`/`cv_score` keep `metric="accuracy"` as the default (`metis/model.py:323,357,371`); a regressor called with the default (`fold_fit(..., "rf_reg", seed)` sans `metric=`) fails deep inside `accuracy_score` ("continuous is not supported") rather than with a metis message. Unreachable in M0 (tests always pass `metric="rmse"`; the engine that pairs kind↔metric is M1), so a note for M1, not a fix now.
- `metis/steps/train.py` ship-refit path calls `tune_offsets_on_holdout` under `decide=offsets` **without** the `is_regression` guard that `fold_fit` has (`metis/model.py:337-340`); a regressor+offsets *ship* refit would `AttributeError` on `predict_proba` instead of the nice "classification-only" message. Not reachable in M0 (no engine wiring; the fold path, which carries the guard, runs first in any sweep) — flag for M1.

**5. Test coverage notes**
- 8 test functions / 12 parametrized cases cover: `is_regression` classification, continuous predict (3 kinds), determinism (3 kinds), hyperparam + seed-passthrough plumbing, RMSE value/perfect, all three complexity formulas, fold_score↔cv_score mean identity under `rmse`, and the offsets-on-regressor refusal. This is the bug class M0 could ship (a regressor mis-routed through the classifier `classes_`/`predict_proba` path) — covered. Could-not-run caveat above stands; the plan's "57 pass, no regression" claim is unverified by me.

**6. Architectural notes for upcoming work**
- ARCH-DRY / ARCH-PURE / ARCH-PURPOSE all **pass** for M0. The kind→branch dispatch is now doing real work across `make_model`/`complexity`/`predict`/`fold_fit`; M1's engine wiring should thread `objective.metric` (`train.fold_score` name vs the `rmse` scorer are distinct — the atlas already calls this out) and set `objective.direction: minimize` for rmse. When M1 pairs regressor↔metric, consider one guard (`is_regression(kind) and metric in classification-metrics → loud`) to collapse the two Minor forward-notes above into a single metis-owned message.

**7. Plan revision recommendations**
- None required — the plan's M0 row (`[x] DONE`, "12 new unit tests") matches the code and the Core-concepts row "regression model kind + RMSE scorer + regression complexity · `metis/model.py` · new (M0)" is delivered at the stated path. (Optional, tied to the Important item: a `## Log`/`## Revisions` line recording that the M0 close review's *effective* window was the model-core surface, not the full 30-issue diff the tooling presented.)
