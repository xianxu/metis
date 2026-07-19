# Boundary Review — metis#66 (milestone M1)

| field | value |
|-------|-------|
| issue | 66 — adaptive outer-fold scheduling + --auto-stop (incumbent-referenced early stop of losing configs) |
| repo | metis |
| issue file | workshop/issues/000066-adaptive-outer-fold-auto-stop.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 12f2ab4e38b51f93135d3312f3a99c45fdf31372^..HEAD |
| command | sdlc milestone-close --issue 66 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-19T09:18:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
The Bash tool is unavailable this session (the harness cannot create its session-env dir — a sandbox/harness restriction, manageable via `/sandbox`), so I could not independently run `go test`/`go vet`. The review below is from close code reading against the Spec + Plan. I verified the aggregation, concurrency invariants, and stop-path honesty by hand.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The M1 machinery is well-built and correct: the `leafBudget` interface cleanly swaps global fan-out (`chanSem`) for fold-ordered dispatch (`prioritySem`), the backfill invariant holds under proof, and the Q graceful-finalize path is carefully guarded so an abandoned fold contributes neither a row nor a score — the partial `out<n>` estimate is honest. No Critical/correctness defects. What holds this back from SHIP is one premature user-facing surface: the `--auto-stop` flag is registered and fully documented in M1 with M2's behavior, but the auto-stop logic is not implemented, so `metis run --auto-stop` silently behaves exactly like `--live` (full run, no loser-stopping) while its help promises incumbent-referenced early stop. Confidence is medium only because I could not execute the suite to confirm the Log's "green" claim.

### 1. Strengths
- **`prioritySem` invariant is real and provable** (`cmd/metis/prioritysem.go:59-87`): the fast-path guard `inflight < capacity && len(waiters) == 0` plus slot-transfer-on-release (inflight held constant while a waiter exists) genuinely enforces `len(waiters)>0 ⟹ inflight==capacity`. Grant order (heap by priority,seq), FIFO-within-priority, and the backfill peak are each pinned by a dedicated unit test with `-race`.
- **ARCH-DRY on the persist tail** (`sweep.go:560-586`): both the full run and the Q-stop funnel through `persistNestedAndReport`, and `completedOuterEstimate` reuses `sampler.Aggregate` rather than re-deriving mean±SE. This is the right consolidation.
- **Clean stop-vs-failure separation** (`runcontrol.go:37-51,120-124`): `requestStop()` sets a soft-latch distinct from `err`, so `firstError()` stays nil and the run finalizes over completed folds instead of aborting. `errRunStopped` is threaded so an in-flight fold is abandoned (no rows via early return before `addManPoints`), keeping the ledger honest.
- **Honest-partial correctness**: because Q only arms in `o.live && o.tui` mode and live mode finishes folds lowest-index-first, the completed set is always a clean prefix → `out<n>` matches `--sample out<n>` semantics; and Mean/SE depend only on scores (not addrs), so the re-aggregation matches the CVDriver reducer.

### 2. Critical findings
None.

### 3. Important findings
- **`--auto-stop` shipped as a lying flag (M2 behavior, M1 no-op)** — `cmd/metis/main.go:53` registers `--auto-stop` with help claiming "after each outer fold (n≥2), a family whose full-k mean is <95%-likely to reach the ledger's incumbent stops its remaining outer folds … marks stopped families `stopped: auto`." But M1 only wires `autoStop: *autoStop` (`main.go:81`) into the budget selection (`run.go:152`, redundant with `live`); there is no sequential-outer scheduling, family filtering, `shouldStop`, incumbent read, or ledger marker (all M2 per the plan). Net effect: `metis run <shape> --auto-stop` does a full-cost run with zero stopping while promising the operator compute savings — the exact purpose the flag names is silently undelivered (ARCH-PURPOSE). **Fix (cheap):** reword the help to state it currently only enables `--live` fold-ordered scheduling (auto-stop lands in M2), OR make the flag error loudly as not-yet-implemented, OR defer registering it until M2 lands. The M1 plan scopes only `--live` + Q; the flag registration is scope creep for this boundary.

### 4. Minor findings
- **Unreachable preamble-stop branch / Q ignored during preamble** (`sweep.go:485-487` vs bridge at `495-504`): the `errors.Is(err, errRunStopped)` handling for `materializeOuterAnalysis` can never fire — the goroutine that calls `requestStop()` starts *after* the preamble returns, so `stopRequested()` is always false while the preamble runs. A Q pressed during a long outer-split is therefore ignored until the split completes. Either move the stop-bridge above the preamble (so a preamble leaf short-circuits → the branch becomes live and Q aborts promptly) or drop the dead branch. The code reads as if Q-during-preamble was intended to work.
- **`--live` on a flat (1-config) run is a silent no-op** while help says "nested (multi-config) runs only" (`main.go:52`) — the sibling `--sample` errors loudly on flat (`sweep.go:280-282`). Harmless (all leaves at priority 0), but the enforcement is inconsistent with its own doc.
- `run.go:152` `if o.live || o.autoStop` — the `|| o.autoStop` is dead (autoStop always sets live true at `main.go:80`); fine as defensive, but it's the only non-redundant use of the `autoStop` field in M1, underscoring finding #3.

### 5. Test coverage notes
- **The determinism gate is weaker than its name** (`live_test.go`): `budgetFakeExec` always calls `acquire(0)`, so `TestLive_ByteIdenticalToDefault` never exercises *different* fold priorities producing identical artifacts — and the fake's output is order-insensitive, so byte-identity is close to trivially satisfied. It does prove the prioritySem is deadlock-free and result-invariant under real nested fan-out with `-race` (valuable), but the load-bearing "byte-identical under real fold-ordered scheduling" claim rests on the order-independent reduce (tested in #18/#31) + the grant-order unit test, not on this test directly. Consider a case that threads genuine per-fold priorities (or drives the production `execStep`) to lock the invariant end-to-end.
- **No n≥2 stop aggregation test**: `TestLive_QFinalizesHonestPartial` only reaches `out1` (SE=0). A stop after ≥2 completed folds would lock the partial mean±SE branch of `completedOuterEstimate` (the non-zero SE path) — currently untested.
- I could not run `go build`/`go test`/`go test -race ./cmd/metis` locally (Bash blocked by the environment). The Log claims full Go + 124 pytest green and `-race` clean; the main agent should confirm this still holds after any fix.

### 6. Architectural notes for upcoming work
- **ARCH-DRY: PASS.** No duplicated logic introduced; the persist tail and the estimate reducer are single-sourced; `chanSem` wraps the existing channel rather than forking a parallel budget.
- **ARCH-PURE: PASS.** `prioritySem` is a pure concurrency primitive unit-tested without IO; the estimate math is `sampler.Aggregate` (pure); the only new IO (`stdinStopSignal`) is a thin injected seam behind `runOpts.stopSignal`, keeping business logic off the terminal.
- **ARCH-PURPOSE: FLAG** — M1's own purpose (fold-ordered live estimate + honest Q-finalize) is delivered; the flag is finding #3 (a documented surface with no deriving implementation).
- For M2: the sequential-outer requirement means `--auto-stop` must force `SeqExec` at the outer level (the plan notes this). The prioritySem's per-fold priority key is already the right seam — under SeqExec fold n fully precedes n+1, so the stop decision is race-free. Watch that the `abandoned`/`stopped` maps don't get conflated: Q-abandon (in-flight, no rows) and auto-stop (`stopped: auto`, partial rows recorded) are different states with different ledger semantics.

### 7. Plan revision recommendations
- The plan (`workshop/plans/000066-…-plan.md`) M1 section does **not** mention registering `--auto-stop` in M1; the code does. Add a `## Revisions` entry recording that `--auto-stop` was wired into the CLI at M1 (implying `--live`) ahead of its M2 logic, and the decision taken for its help text (reworded / gated / deferred per finding #3) — so the plan and the shipped surface stop disagreeing.
- If finding #4 is resolved by dropping the unreachable preamble branch (rather than moving the bridge), note in the plan that Q during the preamble is intentionally a no-op (the operator waits out the one split step).
