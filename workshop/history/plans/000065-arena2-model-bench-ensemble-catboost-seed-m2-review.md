# Boundary Review — metis#65 (milestone M2)

| field | value |
|-------|-------|
| issue | 65 — arena2 model bench: ensemble kind + catboost + seed passthrough |
| repo | metis |
| issue file | workshop/issues/000065-arena2-model-bench-ensemble-catboost-seed.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 0e2dd659219d6f47f5cbb31fb9e65e71c417e94c..HEAD |
| command | sdlc milestone-close --issue 65 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-19T01:37:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M2 `catboost` kind is a clean, pattern-faithful addition: it slots into the existing `make_model`/`complexity`/`MODELS` dispatch without touching the generic fold/cv machinery, the ARCH-PURE purity+determinism pins are both applied and directly tested, the heavy transitive footprint is correctly contained behind a lazy import, and the zero-Go-edits claim checks out structurally (`FamilyOf` derives the family from the `$any`-map branch label — no hardcoded kind list anywhere in Go or CUE). Nothing blocks the *measured-bench* purpose M2 serves. What holds it back from a clean SHIP is one plausible submission-format drift on the ship path (CatBoost's `.predict()` dtype) plus a matching test gap on the exact fold-scoring path the ravel claims to fix — both cheap, neither affecting the OOF scoring the arena2 sweep needs. Note: I could not reproduce the "123 green" run myself — Bash is blocked in this environment by a harness-level EPERM (`mkdir ~/.claude/session-env`), so tests are implementor-reported; findings below are from static analysis.

**1. Strengths**
- Pattern-faithful dispatch: catboost branch (`model.py:186-212`) + complexity branch (`model.py:276-281`) mirror the existing kinds; the generic `fold_fit`/`cv_score`/`predict` need no per-kind change.
- ARCH-PURE pins are enforced *and tested*: `allow_writing_files=False` verified by `test_catboost_no_side_effect_dir` (`test_model.py:522-527`); `thread_count=1` determinism verified by `test_catboost_deterministic_and_seed_override` (`:489-497`).
- Lazy import (`model.py:192`) correctly keeps catboost's heavy transitive deps (matplotlib/plotly/graphviz) out of the forkserver preload for the other four kinds — the stated rationale, and it holds.
- DRY: the `(n,1)→1-D` ravel is single-sourced at the one `predict()` call site (`model.py:245`), a genuine no-op for sklearn kinds; catboost-as-ensemble-member recovers its kind through the *same* rsplit dispatch (`test_model.py:530-540`), no reverse map.
- Atlas is comprehensively updated (`experiment.md:158-168`) — params, class_weight mapping, ARCH-PURE pins, complexity formula, and the ravel are all documented.

**2. Critical findings**
None.

**3. Important findings**
- **Predictions dtype on the ship path** (`metis/steps/predict.py:62` → `out["prediction"]`). CatBoost's `.predict()` on integer targets commonly returns *float* class labels (the same gotcha that produced the `(n,1)` shape the ravel fixes). `reshape(-1)` corrects the shape but not the dtype, so `predictions.csv` may write `"0.0"` where the sklearn kinds write `"0"` — a submission-format drift. Scoring is unaffected (`accuracy_score` compares `0 == 0.0` as True), so the *measured bench* is fine; only the ship submission is at risk. Fix: cast to the label dtype (e.g. `np.asarray(estimator.predict(X)).reshape(-1).astype(...)` keyed to `classes_`, or map through `classes_`). Confidence medium — I could not run to confirm the dtype.
- **Test gap on the fold-scoring path + dtype.** No catboost test exercises `fold_score`/`fold_fit`/`cv_score` — the nested-CV path the ravel's "fold scoring" comment names, and the path arena2 actually consumes. And `test_catboost_train_predict_shapes_1d`'s `set(...).issubset(...)` assertion (`:485`) is dtype-agnostic, so it cannot catch a float/int label drift. Add one `fold_score(X, y, folds, 0, "catboost", …)` asserting a finite score, plus a `preds.dtype`/int-label assertion — this closes both this gap and the finding above cheaply.

**4. Minor findings**
- `max_depth` has no alias for catboost: `make_model` aliases `iterations↔max_iter` (`:195`) but `depth` has no `max_depth` alias, so a config sweeping `max_depth` across kinds silently gets `depth=6` for catboost (unknown key ignored per the forward-compat contract). Consistent with CatBoost's native naming but asymmetric with the `iterations` alias — consider adding the alias or documenting the asymmetry.
- catboost is a *hard* dep pulling matplotlib/plotly/graphviz/pillow (`uv.lock`) — heavy for a training core. Lazy import contains runtime cost; if install footprint matters, an optional extra is an option. Non-blocking.
- `class_weight` for catboost accepts only `"balanced"`/None (loud otherwise, `model.py:210-211`) whereas rf/hist_gbm pass a dict through to sklearn — documented + loud, fine, noting the cross-kind asymmetry.
- `atlas/index.md:137-138` complexity enumeration lists rf/logreg/hist_gbm but omits both `ensemble` (M1) and `catboost` (M2). Secondary summary line; the primary doc (`experiment.md`) is complete. Append catboost's `tree_count×2^depth` for parallelism.

**5. Test coverage notes**
Unit coverage is good and targeted: 1-D shape, determinism + seed-override, balanced-changes-fit + loud-unknown, complexity monotonicity, no-side-effect-dir, ensemble-member. Real gaps: (a) the fold-scoring path is untested for catboost, and (b) prediction *dtype* is unasserted — the two Important items above. Everything else the M2 Done-when lists is covered.

**6. Architectural notes**
- **ARCH-DRY — PASS.** New branches reuse the existing dispatch; the ravel is single-sourced; member-kind recovery reuses the rsplit dispatch rather than an estimator-type→kind reverse map.
- **ARCH-PURE — PASS.** IO pins (no `catboost_info/` write, no stdout) and determinism (single-thread) are enforced *and* tested; the lazy import keeps the pure core importable without the heavy dep at module load.
- **ARCH-PURPOSE — PASS, with the dtype caveat.** M2 delivers catboost through the full compose surface (predict / predict_proba / complexity / seed override / class_weight / ensemble-member) — no under-delivery of the bench purpose. The single spot where "the easy win" could hide is the ship-submission dtype (Important #1); the OOF-scoring purpose that motivates the issue is fully met.

**7. Plan revision recommendations**
- Spec §M2 (issue lines 73-74) says "ravel at the **make_model boundary**"; the implementation correctly ravels inside `predict()` (and the atlas documents *that* location). Add a `## Revisions` entry reconciling the Spec text to "ravel at the single `predict()` call site" so the Spec stops describing a location the code doesn't use.
- If the Important #1 dtype finding is confirmed on a run, record the predictions-dtype cast as an added M2 scope item (or an explicit follow-up) in `## Revisions`, since it's the one ship-path behavior the current tests don't pin.
