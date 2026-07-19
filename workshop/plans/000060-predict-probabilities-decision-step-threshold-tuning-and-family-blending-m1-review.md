# Boundary Review — metis#60 (milestone M1)

| field | value |
|-------|-------|
| issue | 60 — predict probabilities + decision step: threshold tuning and family blending |
| repo | metis |
| issue file | workshop/issues/000060-predict-probabilities-decision-step-threshold-tuning-and-family-blending.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 9f34e4e531c210e40a6dea0f59c1af0abb116dd5^..HEAD |
| command | sdlc milestone-close --issue 60 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-18T23:12:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Static review complete. One material caveat before the verdict: **this session's Bash tool is broken at the harness level** (`EPERM` creating `~/.claude/session-env/...` — it fails before any command runs, in subagents too), so I could not execute `uv run pytest` / `go test` myself. Everything below is from reading the code, tests, issue, plan, and Go cache layer.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** M1 delivers what the issue's Done-when (as revised by the recorded leaf-local pivot) promises: a pure decision core in `metis/model.py` (parse/tune/apply/holdout-tune), eager-loud wiring in the train step, offsets persistence + validated application and always-on `probabilities.csv` in predict, a dedicated legal-and-informative test frame, and atlas docs carrying the two honest costs and the 2-fits price. I verified the re-key claim myself: `Kpre` hashes the full `With` map (`pkg/cache/cache.go:54-62`), so `decide` re-keys with zero Go changes, and the argmax path is byte-identical (predict still calls `predict(model, X)`, not argmax-of-proba). The class-alignment chain (aux `classes_` → tune → main `classes_` → persisted → predict validation) is sound because both models see all classes (the `counts.min() >= k` guard guarantees stratified legality) and sklearn sorts `classes_` identically. Nothing rose to Critical; three cheap Importants below. Confidence is medium solely because the "full python + Go suites green" Done-when item is unverifiable in this session — the environment blocked all command execution, not the code.

### 1. Strengths

- **The no-op-anchored tie-break** in `tune_class_offsets` (`metis/model.py:97-103`) — initializing best at zeros and replacing only on strict improvement — is exactly right, is pinned by the uniform-proba test, and is written up as a lesson. Confirmed-good ground.
- **The legality guard with a metis-voiced error** (`metis/model.py:119-123`) names the constraint and the actual class counts instead of leaking sklearn's opaque stratification error — and the too-small-frame test pins it.
- **`tests/test_model.py:315-317`** (`test_fold_fit_offsets_main_model_is_all_training_rows`) pins the load-bearing design claim — the main model is unchanged under decide=offsets — via prediction equality, not a mock. Best test in the diff.
- **Predict validates `offsets.json` classes against `model.classes_` loudly** (`metis/steps/predict.py:54-57`) — persisting `classes` precisely to enable this check is good artifact hygiene.
- Docs discipline: the atlas honestly documents SE inflation and the aux/main mismatch as accepted costs (`atlas/experiment.md`), and the issue carries a proper `## Revisions` entry reconciling the Spec's superseded step-type design — the artifacts don't lie by aspiration.

### 2. Critical findings

None.

### 3. Important findings

1. **`parse_decide` silently ignores unknown keys inside the offsets bundle** — `metis/model.py:54-61`. `{"offsets": {"holdour": 0.25}}` (a typo) passes validation and silently runs with the default `holdout=0.2`. The docstring claims the "loud misuse pattern," and the top-level is loud (`{"offsets": {}, "extra": 1}` is rejected), but the inner bundle is not. Unlike `parse_model_config`'s params (an open sweep surface, documented forward-compatible), the offsets bundle is a closed config with exactly one legal key today. Fix: reject keys outside `{"holdout"}` with the same loud error shape; add the typo case to `test_parse_decide_table`.
2. **The predict offsets test can pass vacuously; the plan's flip-check was dropped** — `tests/test_steps.py` `test_predict_probabilities_and_offsets_application`. The test recomputes expected predictions *from the persisted offsets* via `apply_offsets`, so if tuning had returned all-zero offsets the assertion would hold while exercising nothing but argmax. Plan Task 2(d) explicitly required "hand-check one row where offsets flip the argmax." Given lessons.md just recorded the vacuously-green-test lesson from this very milestone, close the loop: assert `any(o != 0 for o in payload["offsets"])` or that at least one prediction differs from `argmax(proba)` (deterministic seed makes this stable once verified).
3. **The class-order-mismatch error path is untested** — `metis/steps/predict.py:54-57`. This validation is the honesty guarantee for applying a persisted decision rule; it's the exact kind of bug this diff could ship (a wrong-model/wrong-offsets pairing silently producing garbage labels). Cheap step test: write a mangled `offsets.json` beside a trained model, assert the loud `ValueError`.

### 4. Minor findings

- `tests/test_model.py:249` comment says "80/12/8-style priors" but `_posterior_matrix`'s actual priors are 67.5/18.5/14 (270/74/56 of 400); the plan specified 80/12/8. Behavior fine (optima ≈1.29/1.57, well inside ±4), comment misleading.
- `int(c)` normalization in `train.py:100` / `predict.py:54` would silently collide float-coded class labels (0.4 vs 0.6 → both 0), weakening the validation; defensible under the numeric-int target contract but worth a one-line comment or guard.
- `tune_class_offsets` is a Python loop over 41^(K−1) sklearn scorer calls, not the plan's "vectorized grid search" — fine at K≤3, steep at K≥4; docstring already prices it, no action.
- Test frame construction duplicated between `_decide_frame` (test_model.py) and `_save_decide_dataset` (test_steps.py) — same rng/params rebuilt by hand (ARCH-DRY at test level, tolerable).
- Housekeeping: the plan file's Task 1/2 checkboxes are still `- [ ]` despite both commits landing, and the issue's `## Log` has an empty `### 2026-07-18` header — tick/fill at milestone-close.

### 5. Test coverage notes

Coverage is genuinely strong: determinism, no-op anchoring, clip path, parse table (including bool-adjacent range checks — though `holdout: true` itself isn't in the bad list), legality guard, main-model invariance, eager foldless refusal, offsets-persistence conditionality, and the argmax compat anchor. Gaps: the two Importants above (mismatch path, nonzero-offsets flip), and `fold_fit`/step-level flows are binary-only — `tune_class_offsets` covers K=3 but no K=3 frame flows through `fold_fit`→persist→predict, where the class-indexing chain has the most surface. **I could not run either suite** (session environment failure, not a code issue); the gate operator should treat "suites green" as the implementor's claim until re-run.

### 6. Architectural notes

- **ARCH-DRY: pass.** The inner split reuses `cv_folds`, scoring reuses `resolve_scorer` (the ONE name→scorer site), `fold_score` delegates to `fold_fit`, and the 3-tuple change updated both unpack sites in the same motion as planned. The log-argmax recomputation inside `tune_class_offsets` rather than calling `apply_offsets` per combo is a deliberate hoist of the log, not duplication.
- **ARCH-PURE: pass.** The decision core is deterministic, RNG-explicit, IO-free, and tested directly on arrays with zero mocks; the steps remain thin io→pure→io shells.
- **ARCH-PURPOSE: pass for M1.** The leaf-local pivot narrows the Spec, but the issue's `## Revisions` entry records the supersession with rationale and accepted losses — this is a reconciled re-scope, not a purpose dodge. Blend (M2) is a declared separable milestone. Shadow-sweep on the new artifact contract: `probabilities.csv` columns key on the actual class label (M2's blend material), `offsets.json` carries its own classes for validation — consumers derive, nothing hand-maintained.
- For M2: `apply_offsets` returning **indices** while blend will consume **label-keyed** `proba_<label>` columns is the seam where an off-by-mapping bug will live; the docstring pin helps, but consider a small label-space helper when blend lands.

### 7. Plan revision recommendations

None required — the plan matches the delivered code, and the issue's Revisions entry already reconciles the Spec. Optional nit for the plan's `## Revisions` if you touch it: note that Task 1(a)'s test frame shipped with 67.5/18.5/14 priors rather than the specified 80/12/8, and that Task 2(d)'s "flip one row" check is pending (Important #2).
