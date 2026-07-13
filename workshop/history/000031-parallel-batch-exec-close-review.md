# Boundary Review — metis#31 (whole-issue close)

| field | value |
|-------|-------|
| issue | 31 — parallel batch executor — concurrent Ask-batch execution in Run (determinism-preserving) |
| repo | metis |
| issue file | workshop/issues/000031-parallel-batch-exec.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c72950a3e3b361ec2b17393fe87c0d1912961d3..HEAD |
| command | sdlc close --issue 31 |
| reviewer | claude |
| timestamp | 2026-07-13T15:22:41-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. All 4 production `sampler.Run` sites derive their exec from the single `ExecFor(ss.parallel)` branch point — the shadow-sweep is complete, no consumer left hand-wired. Verdict follows.

```verdict
verdict: SHIP
confidence: high
```

This boundary delivers metis#31's stated purpose cleanly: `Run` gains an injected batch-`exec` seam (ARCH-PURE, mirroring `runPoint`), `SeqExec`/`ParExec`/`ExecFor` are pure order-preserving combinators, and the ONE budgeted resource — the real subprocess spawn — is capped by a single shared leaf semaphore in `execStep`, so nesting fans out freely while live step-subprocesses stay ≤ n, deadlock-free. I independently verified the load-bearing claims rather than trusting the Log: `go test ./... -race` is green across all 9 packages, the determinism (sampler + byte-identical ledger/manifest), peak-≤-cap-under-nesting, real-`execStep` serialization, reader-vs-writer atomic-index, and C1 false-abort tests all pass, and the wall-clock demo reproduced 4.4x. Nothing blocks SHIP; the findings below are all Minor or documented-out-of-scope.

**1. Strengths**
- **ARCH-PURE seam done right** (`pkg/sampler/run.go:20`, `exec.go`): `Run` stays pure over `(smp, runPoint, exec)`; the semaphore + `syncWriter` live entirely in the `cmd/metis` IO shell. The sampler package has zero knowledge of the budget — exactly the pure-core / thin-IO-seam split.
- **Leaf-only budget placement** (`cmd/metis/exec.go:90-96`): acquire/release wrap *only* `cmd.CombinedOutput()`, after the cheap resolve/mkdir/with.json writes. Because a leaf never recurses into `Run`, no orchestration goroutine holds a slot while awaiting children → the nested-pool deadlock is structurally impossible. Verified by `TestNestedCV_PeakConcurrencyWithinCap` (peak ∈ [2,3] at cap 3, run completes).
- **Order-preservation → determinism**: index-addressed writes in `ParExec` (`exec.go:38`) + fixed-order `Tell` (`run.go:34`) + `sortPointRuns` on the append-order side-records (`sweep.go:236`) give byte-identical manifest+ledger. Proven at two altitudes, and `TestRun_ParExecEqualsSeqExec` uses a completion-reversal barrier to genuinely scramble completion order.
- **The concurrency audit was real, not performative**: both true races (torn cache-index write → temp+rename `caching.go:353-367`; git-probe false-abort → `sweep.go:480`) were found and fixed with tests that fail RED against the pre-fix code (reader-vs-writer; flaky probe). The single `ExecFor` branch point (ARCH-DRY) wires all 4 nested `Run` sites from one `ss.parallel`.
- **All shared bookkeeping guarded** (`configs`/`points`/`err` via `sweepPass.mu`; `firstErr` via `errMu`) — `-race` green confirms coverage.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- **ARCH-DRY (`cmd/metis/sweep.go:344-357`):** `runNestedCV` re-implements the set-once mutex error-latch (`errMu`/`firstErr`/`setFirst`/`getFirst`) that `sweepPass.setErr`/`firstError` (`sweep.go:112-126`) already provides — the identical pattern in two structs. A tiny shared `type errOnce struct{ mu sync.Mutex; err error }` with `set`/`get` would consolidate both. ~15 lines; safe to defer.
- **Console leaderboard order (`sweep.go:238`, `reportWinner`):** `ss.configs = pass.configs` is in completion order; `betterFirst` sorts by mean with a *stable* insertion sort, so configs with exactly-tied means print in nondeterministic order on stdout. Cosmetic only — not persisted, and the winner/manifest/ledger are all deterministic. Clean progress is metis#30's scope; fine as-is.
- **`assembleRecord` per-point probe (`cmd/metis/record.go:64`):** `git status --porcelain` runs per point under fan-out and can transiently fail on `.git/index.lock` contention. It degrades gracefully (warns, sets `sha/dirty=""/false`) rather than aborting — and the post-join `captureSweepCode` backfill overwrites `Commit`/`CaptureStatus`/`D`/`CodeFingerprint`, so reproducibility metadata stays correct. Only the coarse per-step `Dirty` bool can be transiently wrong plus some stdout warning noise on a large parallel sweep. `git rev-parse`/`git hash-object` (the other per-point ops) are lock-free, so this is the sole residual concurrent-git surface, and it's non-fatal.

**5. Test coverage notes**
- Strong and well-targeted: the RED-verified reader-vs-writer atomicity test and flaky-probe C1 regression are exactly the tests that would have caught the shipped bugs. `TestExecStep_SemaphoreSerializesRealSubprocess` closes the "fake exec bypasses the real acquire" wiring gap the lessons flagged.
- Acceptable gaps: no single test drives the *real* `execStep` semaphore under *real* nested fan-out — it's split into a wiring test (real leaf, cap 1) + a budget-math test (fake leaf, nesting). Pragmatic given subprocess cost. The `assembleRecord` concurrent-probe degradation is only exercised serially (via `flakyGitProbe`); fine since it's non-fatal and the logic-level graceful-degradation is what matters.

**6. Architectural notes for upcoming work**
- The `Exec[P,O]` seam is the right shape for metis#30 (progress): a future exec that reports per-batch completion slots in without touching `Run`'s purity. Keep the semaphore at the leaf — resist any temptation to bound orchestration goroutines, which would reintroduce the nested-pool deadlock.
- When metis#20 (stateful features) or an adaptive `Tell` lands, the "batch is independent by construction" invariant is what parallelism rests on — an order-sensitive sampler must keep syncing at each `Ask` (it already does; the batch width is the sampler's knob).
- The thundering-herd on a cold cache (≤ n concurrent recomputes of a shared upstream) remains a documented non-goal; if it ever bites, a per-K_pre singleflight at the caching layer is the natural home (not the sampler).

**7. Plan revision recommendations**
The Core concepts table has two stale metadata cells, both already reconciled in the issue's "Deviation from plan" Log — add a `## Revisions` entry to the plan so the table stops contradicting the code:
- Pure entities: `execFor` → **`ExecFor`** (exported, since `cmd/metis` is a different package).
- Integration points: `syncWriter` location `cmd/metis/main.go` → **`cmd/metis/run.go`** (built inside `runExperiment`, not `cmdRun`).

Both entities exist and behave exactly as specified — these are location/casing corrections, not capability gaps, which is why they're Minor rather than blocking.
