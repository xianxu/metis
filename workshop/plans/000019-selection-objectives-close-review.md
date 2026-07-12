# Boundary Review — metis#19 (whole-issue close)

| field | value |
|-------|-------|
| issue | 19 — selection objectives — 1-SE rule + mean-std (configurable sweeper select rule, not raw cv-max) |
| repo | metis |
| issue file | workshop/issues/000019-selection-objectives.md |
| boundary | whole-issue close |
| milestone | — |
| window | cc1aabbc944a3b4a01cf413915b9cff150659ab7..HEAD |
| command | sdlc close --issue 19 |
| reviewer | claude |
| timestamp | 2026-07-09T10:49:28-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: SHIP
confidence: high
```

The whole-issue close for metis#19 is ready to ship. The diff delivers both halves of the issue cleanly and end-to-end: a required tagged-union `objective.select` (mirroring `driver`), a pure `SelectConfigs` two-level rule (group-by-family → band → ε-binned min-complexity → mean tie-break → cross-family argmax-mean), the fold-output type-widening (`float64`→`FoldOutcome`, `Winner`→`SweepResult`) threaded correctly through all three Sampler levels, measured complexity (`metis.model.complexity` via a single `fold_fit` that feeds both score and complexity), and the ARCH-DRY payoff — one pure rule reused by the in-memory ship path and the new offline `metis ledger select`, keyed identically via the exported `FamilyOf`. I independently verified: `go build`/`go vet`/`go test ./...` (9 pkgs) and `uv run pytest` (46) all green; the load-bearing DRY family-key format (`train.model=rf`) is pinned by test; the guard fires on both surfaces; and all six Done-when items are delivered. Both milestone reviews were FIX-THEN-SHIP and every previously-flagged Important finding (stale authoring doc, minimize-path tests, `mean-std λ<0` validation, plan-table drift, `promote --family`→`ledger select` substitution) is reconciled in this window with tests, docs, atlas, and plan `## Revisions`. Only Minors remain — nothing blocks the boundary.

### 1. Strengths (confirmed-good ground)

- **ARCH-DRY, genuinely enforced, not restated.** `SelectConfigs`/`FamilyOf` (`pkg/sampler/select.go:72,222`) are the single source; the offline path reconstructs the family key by `shape.Expand`+`freeParamsEqual` then calls the *exported* `FamilyOf` (`cmd/metis/select_cmd.go:74-118`). `TestLedgerSelect_PctLoss_PerFamilyAndShip` pins the path-qualified key, closing the exact M1-review divergence risk. This is the correct fix for the DRY finding.
- **ARCH-PURE, exemplary.** All decision logic (`SelectConfigs`, `parsimony`, `withinBand`, `GuardComplexity`, `FamilyOf`) is `math`/`sort`/`strings`-only, unit-tested with hand-built `ConfigStat`s and zero mocks; the IO seams (`runPipelineFold` reading `run.Metrics` at `cmd/metis/sweep.go:253`, train emission, offline file reads) stay thin.
- **The corner regression is pinned, not asserted.** `TestSelect_PctLoss_TieBreaksToMean` traces band→ε-bin→mean tie-break and recovers `depth4/feat6` over both the deep overfitter and the sparse corner — the precise empirical failure the issue exists to fix. `TestSelect_PctLoss_BinnedComplexity` lands cx 15/16 *inside* and 10/16 *outside* the 0.10 bin (the plan-review ε-arithmetic lesson applied).
- **Guard placement is correct.** Post-fold/pre-selection in `sweep.go:157-164`, with raw fold rows persisted *before* the guard so they stay re-selectable — `TestShapeSweep_ParsimonyGuardOnMissingComplexity` explicitly asserts the 4 raw rows survive the error.
- **`AggregateView` generalization is minimal and backward-compatible** (`pkg/ledger/ledger.go:220-239`): means every metric column so `train.complexity` reaches selection, keeps `.se`/`.n` only for the objective, preserves `Fold==nil` passthrough idempotency. A v1 single-metric ledger reduces byte-identically.

### 2. Critical findings

None.

### 3. Important findings

None. (The `promote`-family-grouping gap the M2 review rated Important is reconciled in-window: Spec Done-when item 4 carries the substitution note, plan `## Revisions` records `ledger select` supersedes `promote --family`, and `atlas/experiment.md`/`index.md` document `ledger select`. The purpose — a select lever that recovers the shallower regime, verified over the real ledger — is met.)

### 4. Minor findings

- **Duplicated per-family+ship renderer.** `reportWinner` (`cmd/metis/sweep.go:355-377`) and `printSelectResult` (`cmd/metis/select_cmd.go:186-205`) render the same view with different column widths (`%-24s` vs `%-28s`); a shared `func(io.Writer, SweepResult)` would unify (ARCH-DRY, cosmetic).
- **Duplicated `ConfigStat` construction.** `ss.configStats()` (`cmd/metis/sweep.go:171-177`) rebuilds the `ConfigStat{Point, Family: FamilyOf(point), Score}` that `GridConfigs.Done` (`pkg/sampler/configs.go:52-56`) builds internally, from parallel state — the guard input and the selection input are two constructions of one thing.
- **Offline `resolveSelectRule` `one-std-err`/`mean-std` branches untested** (`cmd/metis/select_cmd.go:127-150`); only `pct-loss`/`argmax-mean` exercised offline. Low risk (identical switch structure), but a wrong default λ/tolerance would ship silently.
- **No single test asserts in-memory ship == offline `ledger select` ship over identical data.** The DRY property is structurally guaranteed (same function) and the key format is pinned, but a direct equivalence test would harden against future drift.
- **Mixed-cohort guard misdiagnosis** — a ledger with an old pre-complexity `sweep_sha` cohort makes `configStatsFromLedger` emit `HasComplexity=false` for old rows, so the guard says "model reports no complexity" instead of "scope with `--sweep`". Documented in `workshop/lessons.md` + plan Revisions; a "multiple cohorts" warning is a fair follow-up (`cmd/metis/select_cmd.go:87-110`).
- **logreg `complexity = coef_.size` is feature-count only for binary** (`metis/model.py:83`); multiclass is n_classes×n_features. Fine for binary Titanic; the "= feature count" comment is binary-specific.
- **Durable plan task checkboxes remain `- [ ]`** (`workshop/plans/000019-...-plan.md` Tasks 1-15) despite completion; the tracked issue `## Plan` (M1/M2) is `[x]`, so this is doc-hygiene lag only.

### 5. Test coverage notes

Coverage of the new pure surface is strong and pins real logic, not mock reasserts: all four rules incl. both `minimize` variants, cross-family argmax, no-tagged-sum implicit family, ε-bin boundaries, the guard in both `pkg/sampler` and `cmd/metis`, `Aggregate` complexity + `HasComplexity`, `AggregateView` all-metric mean, `complexity()` per class + unknown-kind raise, and per-fold `{fold_score, complexity}` emission (with model.pkl correctly absent per-fold). The one claim I could **not** independently reproduce is the real 891-row Titanic acceptance table (`sweep_sha 4b90538`, cx 66.3 vs 14.6, public 0.770 vs 0.782) — it needs the kbench peer repo + warm cache; the *mechanism* that produces it is verified end-to-end by the synthetic-ledger tests, so the claim is credible.

### 6. Architectural notes for upcoming work

- `SweepResult.PerFamily` (keyed by `FamilyOf`'s path-qualified format) is now the load-bearing honest per-family leaderboard #22 (ensembling) blends and #23 (nested-CV) estimates — keep `FamilyOf` exported and its key format stable.
- The `objective-metric → complexity-metric` pairing is convention in two places (`foldComplexityMetric` read bare in `runPipelineFold`; `complexityMetricFor` deriving `<step>.complexity` offline at `select_cmd.go:132`). Coupled correctly today (both key off the same const); #21/#23 adding emitted metrics should keep this single-sourced rather than growing a third derivation.
- The guard is a *separate* pure function both IO shells call, not enforced inside `SelectConfigs` — a future third consumer must remember to call it. Acceptable now; worth a doc note if a third surface appears.

### 7. Plan revision recommendations

None. The plan's two `## Revisions` entries (Core-concepts table reconciliation + `promote --family`→`ledger select` substitution) already match the shipped code; the Spec Done-when item 4 note and atlas both agree. No further drift between plan and code.
