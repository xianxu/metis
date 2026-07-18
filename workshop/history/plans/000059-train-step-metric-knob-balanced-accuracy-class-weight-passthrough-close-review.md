# Boundary Review — metis#59 (whole-issue close)

| field | value |
|-------|-------|
| issue | 59 — train-step metric knob: balanced accuracy + class_weight passthrough |
| repo | metis |
| issue file | workshop/issues/000059-train-step-metric-knob-balanced-accuracy-class-weight-passthrough.md |
| boundary | whole-issue close |
| milestone | — |
| window | 662402a4decc4cef1a467313a1849803c79a500b^..HEAD |
| command | sdlc close --issue 59 |
| reviewer | claude |
| timestamp | 2026-07-18T13:26:42-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Static review is complete; the harness shell is broken for this session (EPERM creating its session-env dir, with and without sandbox), so I could not execute the suites myself — reflected in the confidence below.

```verdict
verdict: SHIP
confidence: medium
```

This boundary delivers exactly what metis#59's Spec and Done-when commit to: a closed-set `with.metric` knob (`accuracy` default | `balanced_accuracy`) resolved in one place (`resolve_scorer`, metis/model.py:24), threaded as a pure keyword parameter through `fold_fit`/`fold_score`/`cv_score`, validated eagerly at the train-step entry so the foldless ship refit fails loudly on garbage, plus `class_weight` passthrough to the rf and hist_gbm constructors. I traced every consumer of the retyped signatures (grep across the tree: only `train.py` and the tests call them — all keyword-default-compatible), confirmed the plan's Core concepts table matches the code row-for-row, and confirmed both atlas bullets and the `with:` docstring table were updated in-window (no README.md exists at repo root, so the docs gate reduces to atlas, which is satisfied). The one caveat on confidence: the Bash tool is failing at the harness level in this review session, so I could not independently execute `uv run pytest -q` or the Go suite — the Log claims 93 python + full Go green, and my static read of every test finds them deterministic and correctly constructed, but the close gate should have a real test run on record, not just the implementor's word and my desk-check.

**1. Strengths**

- **Single resolution site done right** (metis/model.py:21-29): `_SCORERS` + `resolve_scorer` mirrors the established `parse_model_config` loud-misuse pattern, error message names the closed set via `sorted(_SCORERS)`. ARCH-DRY pass — no second name→scorer mapping anywhere in the tree.
- **Eager validation is genuinely load-bearing, and tested as such** (metis/steps/train.py:59-60, tests/test_steps.py:174-181): the foldless ship refit never calls a scorer, so without the entry-point `resolve_scorer(metric)` an unknown metric would be silently accepted on exactly that path. The step test pins this specific failure mode, not a generic error path.
- **The skewed-frame tests prove the semantic, not just the plumbing** (tests/test_model.py:185-201, tests/test_steps.py:153-171): constant feature + 10/2 skew + *stratified* k=2 (the stratification comment correctly explains why unstratified would make the assertion flaky) yields exact expected values (accuracy 5/6, balanced 0.5) at both unit and step level, and the cv_score→fold_score threading is pinned against the hand-computed mean.
- **Cache-identity semantics stated where the operator will look** (metis/steps/train.py:31-32, atlas/experiment.md): setting the key re-keys the leaf, absent key leaves existing cohorts untouched — the exact question a titanic-cohort owner would ask.
- **`class_weight` default preserved and asserted** (tests/test_model.py:204-208): `p.get("class_weight")` → `None` default, pinned so existing sweeps can't drift. sklearn pin `>=1.5` comfortably covers `HistGradientBoostingClassifier.class_weight` (added 1.2).

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**

- metis/model.py:141 and :154 — `fold_score`/`cv_score` docstring first lines still say "Validation **accuracy**" / "Mean validation **accuracy**"; now metric-parameterized. One-word staleness each.
- metis/model.py:4 — the module-docstring edit left an overlong unwrapped line ("…balanced_accuracy). All deterministic given a seed and IO-free…").
- metis/model.py:26 — a non-hashable `with.metric` (e.g. a JSON object) raises `TypeError` from `_SCORERS.get`, not the intended loud `ValueError`. Still fails loudly, just off-pattern; `_SCORERS.get(metric) if isinstance(metric, str) else None` would close it. Not worth blocking on.
- workshop/plans/000059-metric-knob-plan.md — all step checkboxes remain `- [ ]` although the work shipped and the issue's Plan section is ticked. Tick or note before archiving to history.

**5. Test coverage notes**

Coverage maps cleanly onto the Done-when: scorer semantics (unit), unknown-metric rejection (unit + step, on the path that matters), metric threading per-fold and through `cv_score` (unit + step), `class_weight` reach + default (unit). The one uncovered combination is the *all-rows-with-folds* path (`cv_score` line at train.py:82) with a non-default metric at step level — it's covered compositionally (step reads the knob per-fold + unit pins cv_score threading), so I'd call it acceptable rather than a gap. The in-test skewed-dataset construction (tests/test_steps.py:135-150) correctly avoids minting a phantom fixture, per the plan-review lesson. Main gap: I could not execute either suite due to the session's broken shell — ensure a green run is on record at the gate.

**6. Architectural notes**

- **ARCH-DRY: pass.** One scorer map, `fold_score` delegates to `fold_fit`, no copy-paste. The skewed frame appears in both test files but at different layers with different materialization (in-memory arrays vs. captured Dataset artifact) — justified, not duplication.
- **ARCH-PURE: pass.** The metric is data→data through the pure core; `train.py` stays a thin io→pure→io shell; validation sits at the IO seam. Unit tests run with zero IO.
- **ARCH-PURPOSE: pass.** Shadow-sweep of consumers: both train-step paths thread the knob; the ledger-name `objective.metric` is correctly distinguished (not a shadow restatement); the kbench s6e7 shape adoption is deferred to kbench#12 M2 — but that deferral is *declared in this issue's own Spec* as out of scope and lives in a peer repo, so it's a separable extension, not the deferred point of this issue. For future work: with two metrics live, downstream select/report surfaces show bare `fold_score` numbers with no record of *which* metric produced them beyond the With map — fine while the With is the source of truth, but worth remembering if a cross-metric comparison surface ever appears.

**7. Plan revision recommendations** — none; the plan matches the code as shipped (only the checkbox-tick housekeeping noted above).
