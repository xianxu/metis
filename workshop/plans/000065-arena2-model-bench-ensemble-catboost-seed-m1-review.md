# Boundary Review — metis#65 (milestone M1)

| field | value |
|-------|-------|
| issue | 65 — arena2 model bench: ensemble kind + catboost + seed passthrough |
| repo | metis |
| issue file | workshop/issues/000065-arena2-model-bench-ensemble-catboost-seed.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 61c0adcb203b549a360a3ce436db8b92c7be34ab^..HEAD |
| command | sdlc milestone-close --issue 65 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-19T01:27:40-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: SHIP
confidence: high
```

M1 delivers exactly its stated purpose: an `ensemble` kind that makes a soft-vote blend scorable *inside* nested CV (an honest OOF estimate, not `metis blend`'s post-hoc leaderboard combine), plus a one-site `eff_seed` passthrough that composes with it into seed-bagging. The implementation is a clean extension of the existing `make_model`/`complexity`/`MODELS` dispatch — no new IO, no hand-rolled soft-vote math (sklearn's `VotingClassifier` owns it), and DRY throughout (members parsed by the same `parse_model_config`, member kind recovered from the estimator NAME so complexity recurses through the existing per-kind dispatch with no reverse type→kind map). Tests pin the real behaviors (member-mean, weights-tilt, complexity=Σ, seed-bagging distinct members, decide=offsets composition, step-path). I verified the Go zero-edit claim by reading `FamilyOf` (`pkg/sampler/select.go:267`) — it derives the family structurally from the `$any`-map branch label, so `train.model=ensemble` needs no Go change. Nothing blocks SHIP. **Caveat on evidence:** Bash was unavailable this session (harness EPERM on `.claude/session-env`), so I could not execute `pytest` or the forkserver smoke myself — my verdict rests on static analysis of code + tests; the main agent should confirm the suite is green before crossing the gate.

**1. Strengths**
- ARCH-DRY complexity recovery done right: `complexity(ensemble)` recurses via `name.rsplit("-", 1)[0]` back through the *single* per-kind dispatch (`metis/model.py:250`), avoiding a second source of truth for kind identity. Nested ensembles even compose for free (`ensemble-0` → recurse).
- Member parsing reuses `parse_model_config` (`model.py:200`), so members accept both bare strings (`"rf"`) and `$any`-map bundles — same normalizer as the top-level knob.
- Byte-identical preservation for existing kinds: `eff_seed = int(p.get("seed", seed))` reduces to `int(seed)` when absent (`model.py:172`); the `"seed"` key is consumed for `eff_seed` only and never leaks to estimator constructors.
- Seed-effect test uses genuinely non-separable data (`_noisy`, `test_model.py`) and compares `predict_proba` — the correct sensitivity, and a lesson captured in the issue Log.
- Loud validation on empty/malformed `members` (`model.py:197-198`), tested across `{}`, `[]`, `"rf"`, and weights-only (`test_make_model_ensemble_requires_members`).
- Step/IO-boundary coverage via `test_train_per_fold_ensemble_through_step_path` — the ensemble flows through the real `train` step → `parse_model_config` → `make_model` → `complexity` seam, not just the unit path.
- Atlas updated in-window (`atlas/experiment.md`) with an accurate description of both features; `atlas/index.md` already links `experiment.md`. No README model-kind enumeration exists to fall out of sync (kinds are shape-authoring config, not a CLI flag).

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- Latent naming coupling: kind recovery via `rsplit("-", 1)` assumes no kind name contains `-` (documented at `model.py:249`). Currently safe (`logreg`/`rf`/`hist_gbm`/`ensemble`; M2's `catboost` is also hyphen-free), but it's an undefended invariant — a future hyphenated kind would silently mis-dispatch complexity. Consider an assert or a non-collidable separator if the kind set ever grows toward hyphens.
- `test_ensemble_single_member_matches_bare_model` unpacks the *fitted* member out of `m.estimators_` and compares `m.predict_proba` to that same object's proba — so it pins "soft-vote-of-one is a no-op" (mean of one element), not equivalence to a separately-trained bare `train(X,y,"rf",…)` as the name suggests. Assertion is valid; name slightly overpromises.
- Ensemble-level `params["seed"]` propagation (ensemble seed → members lacking their own seed) is exercised only indirectly. A one-liner asserting a member's `random_state` equals the ensemble `eff_seed` would pin it directly.

**5. Test coverage notes**
Core behaviors all covered and the tests assert real logic, not mocks. Gaps are edge-only and non-blocking: (a) `weights` length-mismatch → deferred to sklearn's fit-time error (acceptable; sklearn owns it, but untested); (b) nested ensemble (ensemble as a member) complexity/proba — works by construction, untested; (c) direct ensemble-level seed propagation (above). None is the class of bug this diff is likely to ship.

**6. Architectural notes for upcoming work (M2 catboost)**
- ARCH-DRY / ARCH-PURE / ARCH-PURPOSE all pass for M1. When M2 lands `catboost`, keep the hyphen-free kind name so the `rsplit` recovery stays unambiguous, and honor the ARCH-PURE pins already spec'd (`allow_writing_files=False`, `logging_level="Silent"`, fixed `thread_count`) so the "pure core, no IO side-effects" invariant holds — CatBoost's default `catboost_info/` write would otherwise breach it. Add a test that `catboost` is usable as an ensemble member (the compose seam), since the ensemble recursion will name it `catboost-<i>`.

**7. Plan revision recommendations**
None — the plan (M1 checkbox, Spec, and the 2026-07-19 Revisions folding seed→M1 + DRY complexity recovery) matches the code as shipped. The one Done-when item I could not personally verify is the "kbench SMOKE run through the real step/forkserver path"; it is logged as done — recommend the main agent confirm that smoke and a green `pytest` before recording the close verdict, since I could not run either.
