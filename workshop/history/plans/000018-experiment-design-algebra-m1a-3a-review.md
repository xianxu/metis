# Boundary Review — metis#18 (milestone M1a-3a)

| field | value |
|-------|-------|
| issue | 18 — experiment-design algebra M1a — three-phase shape + Sampler fold node (static samplers, per-fold pipeline, driver:single) |
| repo | metis |
| issue file | workshop/issues/000018-experiment-design-algebra.md |
| boundary | milestone M1a-3a |
| milestone | M1a-3a |
| window | bc0a0f0f82cae3d210d8818f3fae57a92f9c7797..HEAD |
| command | sdlc milestone-close --issue 18 --milestone M1a-3a |
| reviewer | claude |
| timestamp | 2026-07-07T23:03:41-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Let me confirm the final state assessment and write the review.

Verified facts: whole-module `go build/vet/test ./...` green; `pytest tests/test_model.py` green (10 pass); cache `#24`/TransitiveD wiring genuinely deferred (caching.go still output-hash-keyed); the `promote --best` NUL corruption + parse failure CONFIRMED by running; `objective.metric` disconnect confirmed by grep.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core M1a-3a deliverable is solid and verified: `cmd/metis` is rewired off the flat `Shape` onto the nested Sampler loop, whole-module `go build/vet/test ./...` is green, `pkg/sweep` is retired, and the sweeper produces an honest per-config `(mean, SE)` leaderboard with a genuinely cache-fold-distinct interior (tested on the *real* caching layer with only the leaf executor faked). The cache `#24` soundness work is correctly deferred to M1a-3b (foundations landed pure; `caching.go` still output-hash-keyed — honest split). What keeps this from a clean SHIP is not the boundary's stated deliverable but its **collateral**: the diff rewired the shipped `promote`/`ledger show` commands onto the new per-fold model *and deleted all their tests in the same commit*, leaving `promote --best` producing a corrupt, unrunnable experiment; the `objective.metric` field is disconnected from what the engine actually scores by; and `atlas` still points at the deleted `pkg/sweep`.

### 1. Strengths (confirmed-good ground)
- **Pure-core/IO-shell split is clean** (ARCH-PURE): `sampler.Run` stays pure; `shapeSweep` (cmd/metis/sweep.go:60) is the mutable IO accumulator that carries the error channel the pure `Run` lacks (`ss.err` short-circuit). Nesting `Run(GridConfigs) ⊃ Run(FixedKFolds)` reads well.
- **Cache fold-distinctness is tested on the real cache** — `TestShapeSweep_CacheFoldDistinctAndReRunHits` / `_HyperparamChangeRecomputesAffected` run `cache:true` with only the leaf exec faked, so the `_fold` `with`-overlay → fold-distinct `Kpre` (the B2 collision guard) is genuinely exercised, not asserted against a mock.
- **`fold_score` extraction** (metis/model.py) is textbook ARCH-DRY: `cv_score` is re-expressed as `mean(fold_score)`, and `test_fold_score_is_single_fold_and_cv_score_is_their_mean` pins exactly that identity.
- **Failing-fold-is-FATAL** (sweep.go:171) correctly replaces v1's record-and-continue — a partial resample isn't an honest estimate — and is tested (`TestShapeSweep_FailingFoldIsFatal`). The old `isPointOutcome` swallowing classifier was cleanly retired.

### 2. Critical findings
- **`promote --best` produces a corrupt, unrunnable experiment on a per-fold ledger** (confirmed by running). `AggregateView` stamps each aggregate row's `PointAddr` with a synthetic key `r.SweepSHA + "\x00" + json(FreeParams)` (pkg/ledger/ledger.go:210), and `renderPromoted` writes `row.PointAddr` verbatim into the `promoted_from:` frontmatter line (cmd/metis/ledger_cmd.go:302). Result: the committed `.md` contains a **NUL byte**, and re-parsing it fails hard — I reproduced `experiment.Parse` → `yaml: control characters are not allowed`. Compounding it, `shapeConfigToExperiment` (cmd/metis/ledger.go:85) reconstructs `data ++ pipeline ++ ship` but omits the engine-synthesized `cv-split` step and the `folds` key, while train.py:44 reads `w["folds"]` **unconditionally** (before the per-fold/all-rows branch) — so even a parse-clean promoted experiment would crash on run. Both breaks were previously guarded by `TestLedger_PromoteBestRoundTrips`, which this diff deleted. *Fix:* give `AggregateView` rows a NUL-free identity (e.g. a dedicated `agg:` marker or leave `PointAddr` empty and have `renderPromoted` special-case aggregate rows), and either finish the promote reconstruction (thread the partition + folds, or drop the folds read in all-rows mode) **or** make `promote --best` error cleanly on a per-fold ledger until M1a-5 — but don't ship it silently emitting a corrupt file. Restore a promote guard test either way.

### 3. Important findings
- **Test-coverage deletion is the root enabler of the Critical.** This diff removed `ledger_e2e_test.go` (467 lines), `shape_e2e_test.go`, and `sweep_e2e_test.go`; `shapesweep_test.go` replaces the *sweep* coverage but not the still-live `promote` (round-trip, cross-code-version warn at ledger_cmd.go:221, actually-commits), `ledger show` (`showLedger`/`renderLedger`/dir-defaulting at ledger_cmd.go:41-48), or CLI arg-order (`hoistShapePath`). All of that code still ships — now untested. Port the still-relevant cases (retargeted to the phase model), not just the sweep ones.
- **`AggregateView` is a new PURE reducer with untested branches** (pkg/ledger/ledger.go:192). Only the happy path is exercised (via the cmd/metis integration test). Untested: the failed-fold → `row.Status="failed"` marking (ledger.go:214,224), the `Fold==nil` passthrough/idempotency (ledger.go:202), and the `.n` count. Per ARCH-PURE + the checklist, add a colocated `pkg/ledger` unit test.
- **`objective.metric` is disconnected from what the engine scores by.** Winner selection reduces the hardcoded `foldMetric="fold_score"` (sweep.go:26,182) and never reads `Objective.Metric`; only `ledger show --sort`/`promote --best` consume `Objective.Metric` (ledger_cmd.go:203-204). The committed fixture `testdata/experiment/titanic-baseline-shape.md:34` declares `objective: {metric: accuracy}`, which matches neither `fold_score` nor the ledger's namespaced `train.fold_score` — so `promote --best`/`show --sort accuracy` over the real shape find **no qualifying metric** (the error string at ledger_cmd.go:206 even tells you it should be `train.fold_score`). Fix the fixture to `train.fold_score`, and have the engine derive the reduced metric from `Objective.Metric` (or have `ValidateShape` assert the objective names the emitted fold metric) so the sweep's reported winner and `promote --best` can't diverge.
- **`atlas/index.md` is now wrong, and fixing it was in-scope.** Plan Task 17 Step 6 explicitly said to fix `atlas/index.md:53-54` when retiring `pkg/sweep`. It wasn't: line 53-54 still describe the **deleted** `pkg/sweep` as "the sweep sampler"; lines 56-63 describe `runSweep` (now `runShapeSweep`), the removed `--max-points` flag, and "per-point failure is recorded + the sweep continues" — the **reverse** of the new fatal-fold behavior. Full `pkg/sampler` documentation is a legitimate `--no-atlas`/M1a-5 deferral, but a dangling reference to a deleted package and a documented-but-reversed behavior actively mislead and are a direct consequence of *this* diff's deletions.

### 4. Minor findings
- `writeSweepLedger` (cmd/metis/ledger.go:122) still takes `objective experiment.Objective` but no longer uses it (`warnIfObjectiveMissing` was removed); the doc comment still claims "objective sets the show/promote sort direction" — false. Drop the dead param + fix the comment. Separately, the removed `warnIfObjectiveMissing` was a real "objective metric matches no row" diagnostic — its loss means a mis-namespaced objective (see finding #4) is now silent at sweep time.
- `SingleDriver` (pkg/sampler/driver.go) is built + tested but **not in the `runShapeSweep` call path** — the outer driver level is inlined as a bare `sampler.Run(GridConfigs)` (sweep.go:112), so the "driver ⊃ sweeper ⊃ resample" algebra is only 2 levels in the actual loop. Fine for winner-only M1a-3a, but the seam `#23` is meant to *swap into* isn't in the path (see §6).
- Stale `M1a-4` references in comments after the plan folded M1a-4 into M1a-3a (sweep.go:261 `partitionRef`, buildFoldExperiment doc).
- `io.dataset_dir` (metis/io.py:112) and train.py's per-fold branch (train.py:51-56) are new but have no colocated Python test; `dataset_dir` only ever hits the exp-path fallback until `features` writes a captured `dataset/` (the documented Phase B), so the upstream-artifact branch is currently unreachable in practice.

### 5. Test coverage notes
Well-covered: `fold_score` pure core; the nested-loop winner + N×k raw ledger + per-config `(mean,SE)`; cache fold-distinctness/warm-HIT/incremental-recompute on the real caching layer; dry-run; fatal-fold; detect-and-abort. Gaps that let bugs through: the entire `promote`/`ledger show` CLI surface (§3), `AggregateView`'s failed/passthrough branches (§3), and the objective-metric consistency (§3, would be caught by a promote-round-trip test over the real fixture).

### 6. Architectural notes for upcoming work
- **The driver seam isn't in the call path.** Because `runShapeSweep` inlines `driver:single`, M1a-5 (ship) and `#23` (nested-CV `driver:cv`) will have to *insert* the outer `Run(SingleDriver/CVDriver, …)` wrap around the sweeper, not swap one Sampler for another. Wiring `SingleDriver` into the loop now (even as a pass-through) would make the three-level algebra real and turn `#23` into the intended one-line swap.
- **Promote/ship reconstruction (`shapeConfigToExperiment`) is the shared M1a-5 seam** and currently under-delivers (no partition/folds, no `{mode:all}`). Track that `train.py`'s unconditional `w["folds"]` read (train.py:44) constrains what a ship/promote experiment must carry — the all-rows path needs either a folds source or a folds-free branch.

### 7. Plan revision recommendations
- Add a `## Revisions` entry to `workshop/plans/000018-experiment-design-algebra-m1a-plan.md` recording that **`promote`/`ledger show` are rewired onto the per-fold model but left incomplete in M1a-3a** (aggregate-row identity + reconstruction gap) and that their e2e coverage was dropped — so M1a-5's Task 18/19 explicitly owns restoring a working, tested `promote`, and the interim state either errors cleanly or is guarded.
- Note in the plan (or the issue `## Log`) that **Task 17 Step 6's atlas fix was not completed** — `atlas/index.md:53-63` still describes the retired `pkg/sweep`, `--max-points`, and the pre-reversal failure-continues behavior — so M1a-5's atlas task (Task 21) inherits a *correction*, not just an *addition*.
- Record the **`objective.metric` vs hardcoded `foldMetric` disconnect** as an open seam (the fixture says `accuracy`; the engine scores `fold_score`) so it's resolved before the honest Titanic e2e (Task 20) rather than surfacing as a silent no-op sort.
