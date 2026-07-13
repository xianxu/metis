# Boundary Review — metis#20 (whole-issue close)

| field | value |
|-------|-------|
| issue | 20 — leakage-safe target features — internal cross-fit (features already per-fold via M1a) |
| repo | metis |
| issue file | workshop/issues/000020-fold-aware-features.md |
| boundary | whole-issue close |
| milestone | — |
| window | c175e0dc26078766cf914582a980d8856da783ef..HEAD |
| command | sdlc close --issue 20 |
| reviewer | claude |
| timestamp | 2026-07-12T23:22:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The metis boundary of #20 is a clean, correct, pure primitive with a genuinely load-bearing test suite. `cross_fit_target_encode` does exactly what it claims: fit rows get an out-of-fold encoding whose group aggregate excludes the row's own label (I verified the `pool = ~out` exclusion and the O(1/N) prior residual is the only self-dependence — matching the documented sklearn-TargetEncoder behavior), non-fit rows get the full-fit shrunk mean, unseen groups fall to the prior, and both strategies plus every edge case I probed (all-non-fit, single-fit, continuous/3-class y, groups isolated in one fold, mixed-type keys, LOO mixed sizes) return finite values without crashing. Full metis suite is green (74 passed) and the leak proof lands with wide margins. Nothing blocks the boundary; the findings are trivial doc/robustness cleanups, and one cross-repo traceability note worth recording.

**1. Strengths**
- The leak proof is the right operationalization: `test_kfold_no_self_leak_on_random_data` (encode.py test) proves the *cause* of cv-inflation (`corr(naive,label)≈0.73` vs `corr(cross-fit,label)≈−0.04`) at the feature level with fixed seeds and comfortable margins, and the real-signal counter-test (`enc.std()>0.1` + per-group `enc≈true rate`) correctly kills the "return prior" cheat. Non-flaky by construction.
- `test_loo_within_group_structure_is_label_invertible` asserts the deterministic `1/(n−1)` gap directly rather than a fragile marginal-correlation inequality — exactly the lesson the plan review distilled. Good discipline.
- ARCH-DRY is honored: `cv_folds` is reused for the internal folds (no second fold generator), and `_shrunk`/`_group_stats` factor the repeated per-subset math cleanly (encode.py:31-38).
- Defensive stratification guard (`len(classes)>1 and counts.min()>=k`, encode.py:84) correctly avoids `StratifiedKFold` blowing up on continuous or rare-class targets — falls back to plain KFold.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- `metis/encode.py:57` — the function docstring says fit rows get "a cross-fit encoding (**own label never used**)", the exact unqualified phrasing `workshop/lessons.md` (this issue's own lesson) says to avoid. The module docstring (line 13) and atlas (line 190) correctly qualify it "never enters via the group aggregate" and note the O(1/N) prior residual. Fix: qualify line 57 to match, so the function docstring stops overclaiming.
- `metis/encode.py:41` — no length-consistency check on `groups`/`y`/`fit_mask`; a mismatched call would fail with an opaque numpy index/broadcast error deep inside. Acceptable for a single-caller internal primitive, but a one-line `len` assert would give the kbench adapter a clear error.
- `metis/encode.py:79-81` — `n_folds < 2` silently degrades to all-prior (the `k < 2` guard short-circuits before `cv_folds`, which *raises* for `k<2`). A caller mistakenly passing `n_folds=1` gets no cross-fit and no signal, silently. The prior-fallback is correct for a genuinely tiny `fit_idx`; consider distinguishing "too few rows" (fall back) from "misconfigured n_folds" (raise).

**5. Test coverage notes**
Load-bearing paths (leak prevention, fit-mask/unseen/full-fit, determinism, LOO invertibility, unknown-strategy) are well covered and pin real logic, not mocks. Untested branches (all behave correctly under my manual probe, so this is future-hardening, not a shipped bug): the `k<2` too-few-fit-rows → prior branch; the `strat=None` fallback on continuous/>2-class y; and LOO with mixed group sizes (only a single size-4 group is tested). A one-line assertion that a 1-fit-row (or all-non-fit) input returns prior without crashing would close the highest-value gap.

**6. Architectural notes for upcoming work**
- ARCH-PURE: **pass** — the primitive is deterministic, IO-free (the only randomness is seeded `default_rng`), and its tests import only numpy/pandas/pytest. The Core-concepts "PURE" claim holds.
- ARCH-DRY: **pass** — reuses `cv_folds`; helpers dedupe the group-stat/shrink math.
- ARCH-PURPOSE: **pass for the metis window** — the metis issue's purpose is the reusable substrate primitive + a rigorous leak proof, both delivered; the consumer wiring (`target_encode_group`, `pclass_survival`, sweep-integration) is legitimately a separate repo/branch (kbench) and the high-value `ticket` group is the tracked kbench#8, not a dodged deferral. The `strategy` dispatch is the deliberate widening axis, built now. When kbench#8 lands, run the shadow-sweep there to confirm every target group derives from this one primitive (no re-implemented concat/mask/split glue).

**7. Plan revision recommendations**
None required — the plan's Core-concepts table matches the metis code exactly (`metis/encode.py::cross_fit_target_encode` new + PURE; `tests/test_encode.py`), and the atlas/lessons entries are accurate.

**Traceability note (not a code defect):** metis#20's Done-when spans two repos — the leak-proof clause is delivered and verified *in this window*, but "existing features unchanged / the Titanic thread still green" is a kbench-side assertion (Log records "kbench 40 passed, 42 configs") that lives outside this metis review window and I could not independently verify here. This split is intentional and documented in the plan's "Scope boundary," so the metis boundary is complete on its own; just confirm the kbench half is reviewed in the kbench tracker before treating the full cross-repo issue as landed.
