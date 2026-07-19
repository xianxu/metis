# Boundary Review — metis#65 (whole-issue close)

| field | value |
|-------|-------|
| issue | 65 — arena2 model bench: ensemble kind + catboost + seed passthrough |
| repo | metis |
| issue file | workshop/issues/000065-arena2-model-bench-ensemble-catboost-seed.md |
| boundary | whole-issue close |
| milestone | — |
| window | bf235ee58952aee3bc0eb90d608c69165d6230a1..HEAD |
| command | sdlc close --issue 65 |
| reviewer | claude |
| timestamp | 2026-07-19T01:43:25-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: SHIP
confidence: high
```

This whole-issue close (bf235ee..HEAD) delivers all three of metis#65's committed purposes cleanly: an `ensemble` kind that makes a soft-vote blend scorable *inside* nested CV (composes with decide/metric/seal because `VotingClassifier` exposes the estimator API), a one-site `eff_seed` passthrough that composes into seed-bagging, and a `catboost` mechanism kind with its ARCH-PURE pins enforced *and* tested. I re-verified the two prior-milestone FIX-THEN-SHIP findings were genuinely addressed in the code (not just claimed): `predict()` now casts to `classes_.dtype` (M2 Important #1) and `test_catboost_fold_score_path_and_int_labels` pins the nested-CV path + int labels (M2 Important #2). I also independently confirmed the zero-Go-edit / Python-only purpose claim by reading `FamilyOf` (`pkg/sampler/select.go:267` — derives the family structurally from the `$any`-map branch label, no hardcoded kind list) and confirming no README or CUE vocabulary enumerates model kinds. Nothing blocks SHIP. **Evidence caveat:** I could not execute `pytest`/`go build`/the forkserver smoke in this read-only session — my verdict rests on static analysis of code + tests against the Spec; the "123 green" and smoke runs are implementor-reported. The main agent should confirm the suite is green before recording the close verdict.

**1. Strengths**
- ARCH-DRY complexity recovery: `complexity(ensemble)` recurses via `name.rsplit("-", 1)[0]` back through the single per-kind dispatch (`model.py:290`) — no estimator-type→kind reverse map, and nested ensembles compose for free.
- The `(n,1)→1-D` ravel *and* the float→int label guard are single-sourced at the one `predict()` call site (`model.py:248`), a genuine no-op for the sklearn kinds; catboost's dtype fix keys on `classes_.dtype` rather than trusting `.predict()`'s output — the correct anchor.
- Byte-identical preservation: `eff_seed = int(p.get("seed", seed))` reduces to `int(seed)` when absent (`model.py:172`); `seed` is consumed only for `eff_seed` and never leaks into an estimator constructor (which would be an unknown-kwarg error).
- ARCH-PURE pins are enforced *and* directly tested: `test_catboost_no_side_effect_dir` (no `catboost_info/`), `test_catboost_deterministic_and_seed_override` (thread_count=1 determinism). Lazy import (`model.py:192`) keeps catboost's heavy transitive deps (matplotlib/plotly/graphviz) out of the forkserver preload for the other kinds.
- Real IO-boundary coverage: `test_train_per_fold_ensemble_through_step_path` flows an ensemble through the actual `train` step → `parse_model_config` → `make_model` → `complexity` seam; `test_catboost_fold_score_path_and_int_labels` exercises the exact nested-CV path arena2 consumes.
- Tests pin real behavior, not mocks: member-mean, weights-tilt `(3·p_rf + p_gbm)/4`, complexity=Σ, seed-bagging distinct members, decide=offsets on the blend, catboost-as-ensemble-member with kind recovery.

**2. Critical findings**
None.

**3. Important findings**
None. (Both M2 FIX-THEN-SHIP Important items are confirmed resolved in the shipped code.)

**4. Minor findings**
- Undefended kind-name invariant: `rsplit("-", 1)` assumes no kind name contains `-` (`model.py:289`). Safe today (logreg/rf/hist_gbm/ensemble/catboost all hyphen-free) and documented, but a future hyphenated kind would silently mis-dispatch complexity. An `assert "-" not in kind` in `make_model` would make the invariant loud rather than latent.
- No step-level assertion that `predict.py` writes int-formatted `predictions.csv` for catboost. The pure `predict()` is dtype-tested and both ship branches are structurally int-safe (argmax → `predict()`; offsets → `model.classes_[...]`), so this is coverage-completeness, not a live risk.
- `test_catboost_as_ensemble_member` trains a catboost-in-ensemble but does not assert the no-side-effect-dir property for that composed path (the `allow_writing_files=False` pin rides through sklearn `clone()`; the solo case is asserted). Nice-to-have.
- `max_depth`↔`depth` intentionally un-aliased for catboost (documented in Revisions: cap vs exact oblivious depth mean different things). Reasonable; noting the cross-kind asymmetry for anyone authoring a shape that sweeps depth across kinds.

**5. Test coverage notes**
Coverage is strong and targeted; the class of bug this diff could ship (soft-vote math, complexity dispatch, seed re-key, catboost shape/dtype/determinism/purity) is covered by tests that assert real logic. Untested edges, all non-blocking: `weights` length-mismatch (deferred to sklearn fit-time error), nested-ensemble complexity (works by construction), and the step-level catboost `predictions.csv` dtype (pure function is tested; step delegates to it).

**6. Architectural notes for upcoming work**
ARCH-DRY / ARCH-PURE / ARCH-PURPOSE all PASS. The kind set is now open-ended and consumer-derived end to end (Go `FamilyOf` structural, no CUE enum, no README enumeration), so future kinds remain Python-only — preserve that by keeping kind names hyphen-free and honoring the IO/determinism pins for any new heavy external estimator.

**7. Plan revision recommendations**
None. The plan, Spec, and both Revisions entries (plan-review findings; M2 FIX-THEN-SHIP resolutions incl. the ravel-location reconciliation and the dtype guarantee) match the code as shipped. The one Done-when item I could not personally verify is the green `pytest` + kbench/forkserver smoke — logged as done; recommend the main agent confirm both before recording the close verdict.
