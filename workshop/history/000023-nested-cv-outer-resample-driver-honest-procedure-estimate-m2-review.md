# Boundary Review — metis#23 (milestone M2)

| field | value |
|-------|-------|
| issue | 23 — nested-CV outer resample driver — honest procedure estimate |
| repo | metis |
| issue file | workshop/issues/000023-nested-cv-outer-resample-driver-honest-procedure-estimate.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 0f57b15c73391d8736306404af886a0be61869d7..HEAD |
| command | sdlc milestone-close --issue 23 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-12T18:00:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M2 boundary delivers the nested-CV outer driver essentially as the plan specifies: a pure `CVDriver` Sampler over the unchanged `Run` loop, a shared `runSweeper` extraction reused by both `driver:single` and `driver:cv`, per-outer-fold sealed selection + refit-and-score, a `mean±SE` honest estimate that ships nothing, `~outerK×` cost surfaced at dry-run, the `ValidateShape` stub-reject replaced with a `k>=2` guard, and a genuinely non-fakeable real-chain confinement test. Go tests pass (15s) and the `uv` real-subprocess seal test actually runs (not skipped). What blocks a clean SHIP is one silent-wrongness gap — the flat path's `GuardComplexity` invariant is not enforced on the nested path, so a `driver:cv` + parsimony-select + non-reporting-model shape that the flat path *loudly rejects* would silently mis-select per outer fold — plus stale "rejected at validate in M1a" assertions in two model/doc surfaces that now contradict the shipped behavior. All are non-blocking at the gate and cheap to fix.

### 1. Strengths
- **ARCH-DRY extraction is exactly right** (`cmd/metis/sweep.go:107` `runSweeper`): the select/reduce logic lives in one place, consumed by both the flat `SingleDriver` pass and each sealed outer fold. `cvSplitStep` (`sweep.go:499`) is likewise shared across flat/sealed/score paths with the k/stratify parameterized. Clean consolidation, not copy-paste.
- **`CVDriver` is a true PURE entity** (`pkg/sampler/driver.go:57`): unit-tested with zero IO/mocks (`driver_test.go`), mirrors `FixedKFolds` faithfully, derives `done` from told-count (so `k=0` is done-immediately, not an empty-non-done-batch panic) — and that edge is explicitly tested (`TestCVDriver_ZeroKIsDoneImmediately`). Matches ARCH-PURE.
- **The real-chain confinement test is the load-bearing proof** (`exec_test.go:TestExecStep_ConfinesRealUvStep_OutOfRootRead`): it drives `execStep → real uv metis/cv-split → metis.io exp_path` and asserts both halves — out-of-root fires, within-root succeeds (scoped, not a blanket failure). This is the seam the fake exec cannot exercise, and it's proven end-to-end.
- **Per-call `sweepPass` accumulators** (`sweep.go:91`) correctly stop one outer fold's manifest/leaderboard bleeding into another's — the accumulator-scoping hazard the plan (I2) flagged is handled.
- **The OUTER-k score-cv-split fix** (c4424d7, `sweep.go:352-355`) is a subtle, correct partition-reproduction insight (the held fold must equal `analysis_i`'s assessment rows via `cv_folds` determinism) and is well-commented.

### 2. Critical findings
None.

### 3. Important findings

**I1 — `driver:cv` silently bypasses `GuardComplexity`; the flat path raises for the same shape (`cmd/metis/sweep.go:346`).**
The flat path guards before it trusts the winner (`sweep.go:215`: `sampler.GuardComplexity(sh.Sweeper.Objective.Select, ss.configStats())`). `runOuterFold` calls `runSweeper` and uses `sres.Ship` directly with **no** guard. `GridConfigs.Done → SelectConfigs` never raises on missing complexity — the guard is a separate, explicit call. So a shape with `select: {pct-loss}` or `{one-std-err}` and a model class that emits no `complexity` metric behaves divergently:
- `driver:single` → loud error ("select rule … needs a measured complexity …"), sweep aborts.
- `driver:cv` → the parsimony axis is silently dropped (all `Complexity=0`, tie-breaks to mean), a quietly-wrong winner is selected in **each** outer fold, and the "HONEST procedure estimate" is computed over those wrong winners with no signal to the operator.

This is precisely the silent-wrongness `select.go:13-15` documents the guard exists to prevent, now reachable in the headline deliverable. Trigger is narrow (parsimony rule + non-reporting model; `argmax-mean`/`mean-std` never trip it), so Important rather than Critical — but it's the one I'd insist on before crossing.
*Fix:* after the `pass.err` check in `runOuterFold`, guard the sealed sweep's configs, mirroring the flat path — e.g. refactor `configStats()` to take a `[]configScore` (ARCH-DRY) and call `sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs))`, returning the error. (Complexity is only known post-fold, so it must run per fold or on the first.)

**I2 — Stale "rejected at validate in M1a" assertions now contradict shipped behavior (ARCH-PURPOSE).**
M2 deleted the stub-reject, but two hand-maintained restatements still claim it's rejected:
- `construct/datatype/experiment-shape.md:85` — "**metis#23** (parsed but rejected at validate in M1a)."
- `pkg/experiment/shape.go:112` — "…but M1a rejects it at validate time."

Both now lie about `ValidateShape`'s behavior. Plan Task 2.6 explicitly listed the datatype doc. ARCH-PURPOSE at-review lens: these are consumers/restatements of the validation contract that no longer derive from the (changed) source.
*Fix:* update both to reflect `driver:cv` is now accepted (`k>=2`). (`construct/vocabulary/experiment.cue:87` "`cv` is metis#23" is fine — it's an attribution, not a rejection claim.)

### 4. Minor findings
- **atlas/experiment.md not updated** — only `atlas/index.md` got the `driver:cv` narrative. Done-when ("atlas: the driver documented alongside the sweeper") is met via index.md, but plan Task 2.6 named `experiment.md` too; the runtime surface (`runNestedCV`/`reportEstimate`/no-ship) lives only in index.md. Consider mirroring, or drop the experiment.md mention from the plan.
- **nested-CV path writes no `sweepManifest`/ledger and does no `captureSweepCode`** (`sweep.go:271` `runNestedCV` returns after `reportEstimate`). Its per-run `record.json` exist (via `runResolvedExperiment`), but there's no grouped manifest and no durable code side-ref, so a dirty-tree `driver:cv` estimate isn't reproducible the way a flat sweep is. Plausibly intentional for an estimation-only path, but it's undocumented — add a one-line comment/atlas note stating the omission is deliberate (or wire a thin manifest).
- **Double `baseDatasetRef(sh)` call** (`sweep.go:442` and `:454`) — `origOut` re-derives the same value `baseOut` held before the sealed-branch reassignment; capture `origOut := baseOut` before the `else` instead.
- `pkg/experiment/shape.go:112` comment drift is folded into I2 above.

### 5. Test coverage notes
- **No test exercises `driver:cv` + a parsimony select rule** — exactly the combination that would surface I1. Add one: `foldFakeExec{noComplexity: true}` + a `driver:cv` shape with `select: {pct-loss}` asserting the run errors with the guard message (currently it would silently pass with a wrong winner).
- **No test proves `readRoot` threads through the `driver:cv` orchestration to `execStep`.** The fake-exec e2e ignores `readRoot`; the real-chain test proves `execStep` confines in isolation; but nothing asserts `runOuterFold → pass.readRoot → runPipelineFold → pointOpts.readRoot → execStep` carries a non-empty root on the sealed pass (and empty on preamble/score). The wiring is straightforward, but a regression there would be invisible. A focused assertion (spy exec capturing the injected `readRoot` per run) would close it.
- The two e2e tests (`nestedcv_e2e_test.go`) pin real plumbing (k held-out lines, mean±SE line, zero submission artifacts, dry-run cost) against actual output, not mocks reasserting the impl — good. They correctly frame themselves as proving mechanism, not the operator-gated honesty-gap magnitude.

### 6. Architectural notes for upcoming work
- **ARCH-DRY: pass.** `runSweeper`/`cvSplitStep` consolidation is the right shape. When fixing I1, factor the `[]configScore → []ConfigStat` mapping into one helper so both paths share it rather than duplicating the guard-input construction.
- **ARCH-PURE: pass.** Pure Sampler core (`CVDriver`) vs thin IO shell (`runNestedCV`/`runOuterFold`/`materializeOuterAnalysis`) is cleanly separated; the pure layer needs no mocks to test.
- **ARCH-PURPOSE: pass on the mechanism, flag on the restatements (I2).** The core purpose — an expressible `driver:cv` yielding a `mean±SE` honest estimate with structurally sealed selection — is delivered, and the "estimate < cv-max" gap-magnitude is legitimately operator-gated (needs Kaggle creds), a separable extension, not under-delivery. The shadow-sweep only flags the two stale "rejected" restatements (I2) as consumers not yet derived from the source.
- The BOUND ASSUMPTION (score-over-full-data honest only while features are stateless) is correctly deferred to metis#20 and documented in the plan/atlas — carry that link forward when #20 lands so the "honest" claim doesn't quietly start lying.

### 7. Plan revision recommendations
- Add a `## Revisions` entry noting **Task 2.3/2.6 gap: `GuardComplexity` must also run on the nested selection path** (I1) — the plan's tail-fork discussion (Task 2.2 item 3) enumerated writeManifest/ledger/GuardComplexity/reportWinner/shipWinner as "flat-path-only" and forked them all out, but `GuardComplexity` is a *selection-correctness* guard, not a ship-only step, so it belongs on both paths.
- Add a `## Revisions` entry recording the **intentional (or deferred) omission of manifest/ledger/code-capture for `driver:cv`** (Minor above), so the estimation-only provenance posture is a stated decision rather than a silent gap.
- Note that **`atlas/experiment.md`** (named in Task 2.6) was satisfied via `atlas/index.md`; either adjust the task or mirror the surface.
