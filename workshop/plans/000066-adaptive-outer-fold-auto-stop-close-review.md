# Boundary Review — metis#66 (whole-issue close)

| field | value |
|-------|-------|
| issue | 66 — adaptive outer-fold scheduling + --auto-stop (incumbent-referenced early stop of losing configs) |
| repo | metis |
| issue file | workshop/issues/000066-adaptive-outer-fold-auto-stop.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7e26a86ae5d763d517bd4b823e0df8245ed1801..HEAD |
| command | sdlc close --issue 66 |
| reviewer | claude |
| timestamp | 2026-07-19T10:00:36-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I now have a complete, hand-verified picture. Let me record the final finding on the stale-comment drift precisely, then write up the review.

The `leafSem`/`sem` references at `main.go:67`, `run.go:106`, `exec_test.go:20` are comment-only drift (the field is now `leafBudget`); `parallel_test.go`'s `peakExec.sem` is an unrelated test-helper's own field. No compile impact.

```verdict
verdict: SHIP
confidence: high
```

This whole-issue close for metis#66 delivers both M1 (fold-ordered `--live` scheduling via a `leafBudget` interface + board `Q` graceful-finalize) and M2 (`--auto-stop` incumbent-referenced loser-stop) cleanly, and — critically — the one substantive defect the milestone reviews caught (the `AggregateView` incumbent-bias, I1) is fixed and verified in this window: `readIncumbent` now derives the incumbent from `familyEstimateFromLedger` → `sampler.FamilySelect` (`autostop.go:44-52`), the *exact* reduce `metis select --best` ships (`select_cmd.go:125,145`), and the regression test genuinely exercises the divergence (rf4/rf8 share family `train.model=rf` but differ in `max_depth` → pooled 0.85, not the per-config max 0.90). I re-derived the predictive-variance bound (`SEpred² = s²·r/k²·(1+r/n)`, Var(S_future)=rσ²(1+r/n)) and it's correct; the pure rule is genuinely IO-free and directly tested; budget reclaim drops stopped families' *inner* sweeps (the real cost); the `stopped` ledger column is a faithful ragged extension; and `FamilyEstimate` correctly counts a prior run's `stopped:auto` rows as honest (Status "ok"). Only documented minors remain. **One caveat: the Bash tool is unavailable this session (harness EPERM on session-env `mkdir`, persists with the sandbox disabled — same as the M1/M2 passes), so I could not execute `go test`/`go vet`; verification is static + hand-traced. The Log claims the suite is green (cmd/metis `-race`, pkg/sampler, pkg/ledger, vet, 124 pytest) — the main agent should confirm before trusting the close.**

### 1. Strengths
- **The I1 fix is genuine, not papered over** (`autostop.go:27-54`): `readIncumbent` threads `ss.sh` in and reuses the canonical select reduce, so "the incumbent" has one definition fleet-wide (ARCH-DRY restored). The docstring now correctly names *why* `AggregateView` was wrong.
- **The regression test exercises the actual divergence** (`autostop_test.go:19-54`): rf's winning config varies across folds (rf4@0.80, rf8@0.90 → pooled 0.85), and the assertion pins 0.85 not 0.90 — it fails against the old reducer, passes against the new. Exactly the test the prior passes demanded.
- **Pure rule, cleanly injected (ARCH-PURE)** (`autostop.go:72,101`): `shouldStop`/`tCrit` are deterministic and unit-tested directly (loser/winner/borderline/both-directions/monotone/n=1/full-k/t-table). `evaluateAutoStop`/`activeConfigs` are the thin seam.
- **`--auto-stop` rejects `--sample`/`--fast`** (`sweep.go:333`): removes the M2-review k-vs-runFolds imprecision *by construction* — under auto-stop `runFolds==k` always, so `shouldStop`'s `r=k−n` model is exact.
- **`prioritySem` backfill invariant is real and provable** (`prioritysem.go`): fast-path guard + slot-transfer-on-release enforce `len(waiters)>0 ⟹ inflight==capacity`; grant order / FIFO-within-priority / backfill peak each pinned by a `-race` unit test.
- **Clean stop-vs-failure separation** (`runcontrol.go`): `requestStop()` soft-latch is distinct from `err`, so the run finalizes over completed folds; `errRunStopped` abandons an in-flight fold before `addManPoints`, keeping the ledger honest. Both the full and Q-stop tails funnel through `persistNestedAndReport` (ARCH-DRY).
- **The reported estimate stays coherent under auto-stop**: dropping losers can never change the per-fold argmax-mean ship family (a loser is by definition below the argmax), so `est` is invariant to loser-dropping — a nice implicit property.

### 2. Critical findings
None. The prior C1/I1 (`AggregateView` incumbent bias) is resolved in this window and independently verified.

### 3. Important findings
None beyond the documented minors below.

### 4. Minor findings
- **Corrupt-ledger conflated with "no prior run"** (`autostop.go:40-43`): any `loadLedger` decode error returns `present=false`, printing "no incumbent (no prior run)" (`sweep.go:613`) and silently degrading to a full sweep. Distinguish a real load error (warn) from a legitimately-absent ledger.
- **All-losers edge over-marks `stopped:auto`** (`sweep.go:125-127` + `markStoppedRows:171-179`): if every family is stopped, `activeConfigs` keeps them all so they run full k, yet their rows are still tagged `stopped:auto` — the marker's documented meaning ("remaining folds were cut") is then inaccurate. Provenance-only; nothing reduces on the marker.
- **Cross-cohort incumbent pooling** (`autostop.go:44`): `FamilyEstimate` pools outer rows across all fingerprint cohorts, whereas `metis select` refuses a multi-cohort ledger without `--fingerprint`. Already documented in the docstring; acceptable (stop cost = a re-run).
- **`--auto-stop` on a flat (1-config) shape is a silent no-op** (never reaches `runNestedCV`) — the `--auto-stop` help doesn't note this, though `--live`'s help does note flat is unaffected. One-line help addition.
- **`incumbentRef.direction` is set but never read** (`autostop.go:23,39`) — `shouldStop` gets direction from `evaluateAutoStop`'s arg. Harmless dead field.
- **Stale `leafSem`/`sem` wording in comments** (`main.go:67`, `run.go:106`, `exec_test.go:20`): the field is now `leafBudget`. Comment drift only, no compile impact.

### 5. Test coverage notes
- `shouldStop`/`tCrit`: excellent, direct, pins real statistical behavior — the "never truncate a would-be winner" invariant the issue is most exposed on.
- `readIncumbent`: now has a direct unit test seeding real multi-fold outer rows with a varying within-family winner — the gap prior passes flagged is closed.
- `stopped` ragged column: dedicated `pkg/ledger` round-trip + no-header byte-identity test *and* e2e round-trip.
- **`TestLive_ByteIdenticalToDefault`** is a valid deadlock-free + result-invariant gate under `-race`, but `budgetFakeExec` always `acquire(0)`, so it doesn't exercise *different* fold priorities producing identical output; grant-order is separately locked by `prioritysem_test`. Non-blocking (composition is sound), but the end-to-end "byte-identical under real fold-ordered scheduling" claim rests on the order-independent reduce + the grant-order test, not this test directly.
- **The `out0` / empty-completion path** (`finalizeStopped(0)`, Q-during-preamble) is safe — `Aggregate([])→MeanSE{}`, no panic — but untested (`TestLive_QFinalizesHonestPartial` reaches `out2`). Narrow window.
- **Suite not executed here** (harness EPERM). Recommend the main agent run `go test ./cmd/metis/ ./pkg/ledger/ ./pkg/sampler/ -race` + `go vet ./...` before recording the verdict.

### 6. Architectural notes for upcoming work
- **ARCH-DRY — PASS.** The biased parallel incumbent reduce is gone; `familyEstimateFromLedger` is the single per-family outer reducer both `metis run --auto-stop` and `metis select` consume. `persistNestedAndReport` single-sources the full + Q-stop tails; the ragged `stopped` column mirrors `fold`/`level`/`outer_fold`.
- **ARCH-PURE — PASS.** The stat rule and `prioritySem` are pure and unit-tested without IO; the sweep methods are a thin coordination seam; the only new IO (`stdinStopSignal`) is a thin injected seam behind `runOpts.stopSignal`.
- **ARCH-PURPOSE — PASS.** Shadow-sweep: "the incumbent" now derives from the source `metis select` ships — no hand-maintained twin remains. The diff delivers the real reclaim, losers-only, sequential gating, marker, live estimate, and Q-finalize. For metis#54 (inner racing): the per-fold priority key is the right seam, and "the incumbent" is now one stable definition to compose against; watch that Q-abandon (no rows) and auto-stop (`stopped:auto`, partial rows) stay distinct ledger states.

### 7. Plan revision recommendations
None required. The plan body's M2.1 still literally reads `ledger.AggregateView → ledger.Best`, but the appended `## Revisions` entries (`plan:170-201`) document both the M1 flag-help correction and the M2.1 incumbent-source correction to `familyEstimateFromLedger`+`FamilySelect` — the sanctioned append-don't-overwrite convention. Plan and code are consistent as of this window.

---

## Re-review — 2026-07-19T10:02:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 66 — adaptive outer-fold scheduling + --auto-stop (incumbent-referenced early stop of losing configs) |
| repo | metis |
| issue file | workshop/issues/000066-adaptive-outer-fold-auto-stop.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7e26a86ae5d763d517bd4b823e0df8245ed1801..HEAD |
| command | sdlc close --issue 66 |
| reviewer | claude |
| timestamp | 2026-07-19T10:02:21-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have a complete, independently-verified picture. I traced the incumbent-reducer reuse, the predictive-stop math against every test case, the prioritySem invariant, and the graceful-stop honesty path against the code (not the commit messages or the embedded reviews).

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This whole-issue close for metis#66 delivers both milestones cleanly and — critically — the net window (`c7e26a8..HEAD`) already incorporates the fix for the one substantive finding the prior passes raised (the `AggregateView` incumbent bias). I independently confirmed the fix is real, not papered over: `readIncumbent` computes the incumbent via `familyEstimateFromLedger` → `sampler.FamilySelect` (`autostop.go:44-52`), the exact reduce `metis select` ships (`select_cmd.go:125,145`), passed the same `obj.Metric`/`obj.Direction`. The pure loser-stop rule (`shouldStop`/`tCrit`) is genuinely IO-free and directly unit-tested; I re-derived `SEpred² = s²·r/k²·(1+r/n)` and hand-checked all nine `shouldStop` table cases and the `tCrit` values against a one-sided 0.05 t-table — all correct. The `prioritySem` backfill invariant holds under my trace, budget reclaim is real (stopped families' *inner* sweeps are dropped via `activeConfigs`), the graceful-Q path abandons in-flight folds before any `addManPoints` so the partial `out<n>` stays honest, and the ragged `stopped` column preserves ledger byte-identity. Only documented Minors remain. **Caveat: I could not run `go test`/`go vet` — the harness EPERMs on session-env creation (not sandbox-bypassable), the same block the three prior passes hit — so my verification is static + hand-traced. The Log claims the suite is green (`-race` on cmd/metis, pkg/sampler, pkg/ledger, vet clean, 124 pytest); the main agent should confirm that still holds before trusting the close.**

### 1. Strengths
- **The prior C1/I1 is genuinely fixed and independently verified** (`autostop.go:27-54`). `readIncumbent` threads `ss.sh` in and reuses the select path's per-family pooled reducer — one definition of "the incumbent" fleet-wide (ARCH-DRY restored). The docstring correctly names *why* `AggregateView` was wrong.
- **The regression test exercises the actual divergence** (`autostop_test.go:19-53`): rf's winning config *varies* across folds (rf4@0.80, rf8@0.90 → pooled 0.85), asserting 0.85 not the per-config max 0.90 — it fails against the old reducer, passes against the new. This is the test the earlier passes asked for, and it's here.
- **`prioritySem` invariant is real** (`prioritysem.go:59-87`): the fast-path guard `inflight < capacity && len(waiters)==0` plus slot-transfer-on-release (inflight held constant while a waiter exists) enforces `len(waiters)>0 ⟹ inflight==capacity`. I traced the concurrent-release cascade — never exceeds capacity, deadlock-free. Grant order / FIFO-within-priority / backfill peak each have a dedicated `-race` unit test.
- **Clean stop-vs-failure separation** (`runcontrol.go`): `requestStop()` is a soft-latch distinct from `err`, so `firstError()` stays nil and the run finalizes over completed folds. `errRunStopped` is swallowed in `runPipelineFold` (no `setErr`) and detected in `runOuterFold` *before* `addManPoints` — so an abandoned fold contributes neither a row nor a score. `TestLive_QFinalizesHonestPartial` proves the out2 partial (SE>0) with only folds {0,1} in the ledger.
- **ARCH-DRY tail** (`sweep.go:672-701`): full run and Q-stop funnel through `persistNestedAndReport`; `completedOuterEstimate` reuses `sampler.Aggregate`. `activeConfigs`/`markStoppedRows` handle the never-empty and retroactive-marking edges.

### 2. Critical findings
None.

### 3. Important findings
None. (The one substantive gap the prior passes raised — the `AggregateView` incumbent bias — is resolved in this window and independently confirmed; the `--auto-stop`+`--sample`/`--fast` reject at `sweep.go:333` removes the earlier k-vs-runFolds imprecision by construction.)

### 4. Minor findings (all documented in code/comments; note for future)
- **Corrupt-ledger conflated with "no prior run"** (`autostop.go:40-43`): any `loadLedger` error → `present=false` → prints "no incumbent (no prior run)" (`sweep.go:613`) and silently degrades to a full sweep. Distinguish a decode error (warn) from a legitimately-absent ledger.
- **All-losers edge over-marks `stopped: auto`** (`sweep.go:125-127` + `markStoppedRows`): if every family is stopped, `activeConfigs` keeps them all so they run full k, yet their rows are still tagged `stopped: auto` — the marker's "remaining folds were cut" meaning is then inaccurate. Cosmetic/provenance only.
- **`--auto-stop` on a flat (1-config) shape is a silent no-op** (never reaches `runNestedCV`), whereas the sibling `--sample` errors loudly on flat (`sweep.go:373-375`). The help text doesn't note it; inconsistent enforcement.
- **Cross-cohort incumbent pooling** (`autostop.go:44`): `familyEstimateFromLedger` pools across all fingerprint cohorts, whereas `metis select` refuses a multi-cohort ledger without `--fingerprint`. A stale cohort could set the bar. Documented in the docstring (`autostop.go:35-37`); stop-cost = a re-run, acceptable.
- **`markStoppedRows` mutates `ss.man.Points` under `stopMu`** while that slice is elsewhere guarded by `manMu` (`sweep.go:171-179`). Harmless — called single-threaded at finalize after `sampler.Run` joined — and the comment says so; a lock-domain inconsistency worth the existing note.

### 5. Test coverage notes
- `shouldStop`/`tCrit`: excellent, pure, direct — loser/winner/borderline/both-directions/monotone/n=1/full-k/t-table. Pins the safety property.
- `readIncumbent` now has a direct multi-fold-varying-winner unit test (the I1 gap closed); the `stopped` ragged column has its own `pkg/ledger` round-trip + no-header byte-identity test (the I2 gap closed).
- **Weaker-than-its-name determinism test** (`live_test.go`): `budgetFakeExec` always calls `acquire(0)`, so `TestLive_ByteIdenticalToDefault` never runs genuinely *different* fold priorities through the two budgets — and the fake's output is order-insensitive, so byte-identity is near-trivially satisfied. It does prove the prioritySem is deadlock-free + result-invariant under real nested fan-out with `-race` (valuable). The load-bearing "byte-identical under real fold-ordered scheduling" claim rests architecturally on the order-independent reduce (#18/#31) + `sortPointRuns` + the grant-order unit test — sound, but not locked *end-to-end* by this test. Consider a case threading genuine per-fold priorities (or driving the production `execStep`). Non-blocking.
- The production `execStep.priority` threading (`runOpts.priority → execStep.priority → budget.acquire`) is verified by reading, not integration-tested (fakes bypass the real execStep). Simple field-threading; low risk.
- **Not executed this pass** (harness EPERM). Recommend the main agent run `go test ./cmd/metis/ ./pkg/ledger/ ./pkg/sampler/ -race` + `go vet ./...` before recording the verdict, to confirm the Log's green claim.

### 6. Architectural notes for upcoming work
- **ARCH-DRY — PASS.** The biased parallel incumbent reduce is gone; `familyEstimateFromLedger` is the single per-family outer reducer both `metis run --auto-stop` and `metis select` consume. Downstream metis#54 (inner racing) can build on one stable "incumbent" definition — stabilized here, good.
- **ARCH-PURE — PASS.** The stat rule and `prioritySem` are pure and unit-tested without IO; `stdinStopSignal` is the only new IO and it's a thin injected seam behind `runOpts.stopSignal`.
- **ARCH-PURPOSE — PASS.** Shadow-sweep of the single-source change: the incumbent is *enforced* (derived from the select reducer), not a hand-maintained restatement; both M1 (live estimate + honest Q) and M2 (losers-only stop with real inner-sweep reclaim) fulfill the issue's stated purpose, not a cheap subset. For metis#54 composition, the per-fold priority key is the right seam; watch that Q-abandon (no rows) and auto-stop (`stopped: auto`, partial rows) stay distinct ledger states — they currently are.

### 7. Plan revision recommendations
None required. The plan body's M2.1 still literally reads "`ledger.AggregateView → ledger.Best`" (`plan:119-122`), but the appended `## Revisions` entry (`plan:180-194`, in the fix commit) documents the correction to `familyEstimateFromLedger`+`FamilySelect` per the append-don't-overwrite convention. Plan and code are consistent as of this window.
