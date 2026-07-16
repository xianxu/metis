# Boundary Review — metis#19 (milestone M2)

| field | value |
|-------|-------|
| issue | 19 — selection objectives — 1-SE rule + mean-std (configurable sweeper select rule, not raw cv-max) |
| repo | metis |
| issue file | workshop/issues/000019-selection-objectives.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | ba9255d0e7d1df3d857fd59db1e28ec85ffe266f..HEAD |
| command | sdlc milestone-close --issue 19 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-09T10:41:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have completed a thorough review. Build, vet, and all affected Go + Python tests are green; I traced the logic against the Spec/Plan and ran the mechanism. Here is my assessment.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** M2 delivers the measured-complexity half of #19 cleanly: `metis.model.complexity` per class (rf mean-leaves, logreg coef-count) via a single `fold_fit` that feeds both score and complexity; `train` emits it per fold; `runPipelineFold` threads `FoldOutcome{Score,Complexity,HasComplexity}`; `AggregateView` generalizes to mean every metric column; a pure `GuardComplexity` gates parsimony rules on both surfaces; and a new offline `metis ledger select` reuses the same pure `SelectConfigs`+exported `FamilyOf` so the two surfaces key families identically. The ARCH-DRY "one rule, two consumers" property is genuinely enforced and tested. Nothing blocks SHIP. The one thing worth fixing before crossing is a **traceability gap**: the Spec (§A) and Plan (Task 12) both promised `promote` would gain family grouping / a `--family` selector, and that was silently dropped in favor of the new `ledger select` command — with no `## Revisions` note recording the substitution. Everything else is minor.

### 1. Strengths (confirmed-good ground)
- **ARCH-DRY, done right (the load-bearing property).** `SelectConfigs`/`FamilyOf` (`pkg/sampler/select.go:72,222`) are the single source; the offline path reconstructs the family key by `shape.Expand` + `freeParamsEqual` match, then calls the *exported* `FamilyOf` (`cmd/metis/select_cmd.go:74-118`). `TestLedgerSelect_PctLoss_PerFamilyAndShip` pins the path-qualified `train.model=rf` key to prevent the exact M1-review divergence. This is the right fix for the DRY finding, not a restatement.
- **`fold_fit` refactor** (`metis/model.py:87`): one fit feeds both score and complexity (no double-fit); `fold_score` delegates. Clean ARCH-PURE seam.
- **`AggregateView` generalization** (`pkg/ledger/ledger.go:230-239`): means every metric column but keeps `.se`/`.n` only for the objective — minimal and general, and backward-compatible (v1 single-metric ledger reduces identically; pass-through idempotency preserved).
- **Guard placement** (`cmd/metis/sweep.go:157-164`): post-fold/pre-selection, with raw fold rows persisted *before* the guard so they stay re-selectable — and `TestShapeSweep_ParsimonyGuardOnMissingComplexity` explicitly asserts the 4 raw rows survive the error. Correct design.
- **ARCH-PURE:** all decision logic (`SelectConfigs`, `parsimony`, `GuardComplexity`, `FamilyOf`) is IO-free and unit-tested with hand-built stats; the IO seam (train emission, file reads) is thin.

### 2. Critical findings
None.

### 3. Important findings
- **`promote` family-grouping declared but not delivered — no deferral note (ARCH-PURPOSE / traceability).** Spec §A ("`promote` gains a family selector") and Done-when item 4 ("`pkg/ledger`/`promote` (offline) group by family") plus Plan Task 12 Step 2 ("`promote --family <name>` promotes that family's robust winner") all commit to it. The diff adds no `--family` flag, and `promote --best` still selects via raw `ledger.Best` argmax (`cmd/metis/ledger_cmd.go:208-213`), *not* the family-grouped select rule. The offline family-grouped surface was instead delivered as the new `metis ledger select`, which does fulfill the acceptance counterfactual — so the *purpose* is met and a workaround exists (`promote --point 'train.model=rf,…'`). But a Spec+Plan deliverable was dropped silently. **Fix:** either add `promote --family` reusing `SelectConfigs`, or add a `## Revisions` entry to the plan recording that `ledger select` supersedes `promote --family` (and note it in the issue Log). Non-blocking.

### 4. Minor findings
- **Guard error misdiagnoses a mixed-cohort ledger.** On a ledger holding an old pre-complexity `sweep_sha` cohort + a new one, `configStatsFromLedger` emits `HasComplexity=false` stats for the old cohort, so `GuardComplexity` errors with "families … report none — have each model class emit a complexity metric" even though the model *does* emit it. This is exactly the footgun captured in `workshop/lessons.md`; consider defaulting offline `ledger select` to the latest sweep, or warning when >1 `sweep_sha` is present, so the error doesn't point at the wrong cause. (`cmd/metis/select_cmd.go:87-110`, `pkg/sampler/select.go:20`)
- **Duplicated per-family+ship renderer.** `reportWinner` (`cmd/metis/sweep.go:355-377`) and `printSelectResult` (`cmd/metis/select_cmd.go:186-205`) render the same view with slightly different column widths (`%-24s` vs `%-28s`). Mild ARCH-DRY; a shared `func(io.Writer, SweepResult, …)` renderer would unify them.
- **Duplicated `ConfigStat` construction.** `ss.configStats()` (`cmd/metis/sweep.go:171-177`) rebuilds the same `ConfigStat{Point, Family: FamilyOf(point), Score: meanSE}` that `GridConfigs.Done` (`pkg/sampler/configs.go:52-56`) builds internally, from parallel state (`ss.configs` vs `configState.results`). Works, but the guard input and the selection input are two constructions of one thing.
- **`atlas/experiment.md` still documents only `metis ledger show`** (`atlas/experiment.md:6,101`), not the new `ledger select` sibling. `atlas/index.md` does cover it, so the gate is satisfied at the index level — but the feature-sketch doc reads as stale.
- **logreg `complexity = coef_.size` is feature-count only for binary** (`metis/model.py:83`); multiclass would be n_classes×n_features. Fine for the binary Titanic domain, but the "= feature count" comment is binary-specific.
- **Offline `resolveSelectRule` branches for `one-std-err`/`mean-std` are untested** (`cmd/metis/select_cmd.go:120-150`); only `pct-loss` and `argmax-mean` are exercised through the offline command.

### 5. Test coverage notes
- Mechanism coverage is strong and pins real logic, not mocks: `complexity()` per class + unknown-kind raise (`tests/test_model.py`), per-fold `{fold_score, complexity}` emission (`tests/test_steps.py`), `AggregateView` all-metric mean with `.se`/`.n` only on the objective (`pkg/ledger/ledger_test.go`), the guard in both `pkg/sampler` and `cmd/metis`, and the offline `ledger select` proving pct-loss recovers md=4 over md=8 while argmax picks md=8.
- **The real 891-row Titanic acceptance (Log's per-rule table, `sweep_sha 4b90538`) I could not independently reproduce in-review** — it needs the kbench peer repo + warm `.metis-cache`. I take the specific real-data numbers (cx 66.3 vs 14.6, public 0.770 vs 0.782) on the documented run; the *mechanism* that produces them is verified end-to-end by the synthetic-ledger tests above, so the claim is credible, not asserted-in-a-vacuum.

### 6. Architectural notes for upcoming work
- The `objective-metric → complexity-metric` pairing is convention in two places (`foldComplexityMetric` read bare in `runPipelineFold`; `complexityMetricFor` derives `<step>.complexity` offline, `cmd/metis/select_cmd.go:132`). It's coupled correctly today (both reference the same `foldComplexityMetric` const), but #21/#23 adding more emitted metrics should keep this pairing single-sourced rather than growing a third derivation.
- `SweepResult.PerFamily` is now the honest per-family leaderboard the Spec earmarked for #22 (ensembling) and #23 (nested-CV). The map key format (`FamilyOf`) is now load-bearing for those consumers — keep it exported and stable.

### 7. Plan revision recommendations
- **`workshop/plans/000019-selection-objectives-plan.md` `## Revisions`** — add an entry: *"Task 12 `promote --family` superseded by the new `metis ledger select` command; `promote` retains raw `--best`/`--point` (no family grouping). The offline family-grouped acceptance runs over `ledger select`, not `promote`."* This stops the Plan (and Spec Done-when item 4's "promote group by family") from claiming a surface the code doesn't ship. If instead you choose to add `promote --family`, that's the alternative resolution.
