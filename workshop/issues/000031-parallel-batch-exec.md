---
id: 000031
status: codecomplete
deps: []
github_issue:
created: 2026-07-13
updated: 2026-07-13
estimate_hours: 2.8
started: 2026-07-13T14:15:42-07:00
actual_hours: N/A
---

# parallel batch executor — concurrent Ask-batch execution in Run (determinism-preserving)

## Problem

`pkg/sampler/Run` executes an `Ask` batch **sequentially** — `for _, p := range batch { s = Tell(s, p,
runPoint(p)) }`. But a grid sweep of **495** (`driver: single`) / **2,475** (`driver: cv`) per-fold runs
is embarrassingly parallel (every point in a grid's single all-at-once batch is independent), yet runs
one subprocess at a time. This is the dominant wall-clock cost of the honest run.

## Spec

The parallel unit is **one `Ask` batch**: every point in a batch was proposed without any other's result
(that is what makes it a batch), so a batch is independent **by construction** → safe to run concurrently,
sync at the batch boundary (the next `Ask`). Width = the sampler's own knob: grid → whole point-set (fully
parallel); sequential bayes → 1 (serial, correctly); q-batch bayes / Hyperband bracket → q. **No per-sampler
runner** (sibling metis#30) — one seam:

1. **Inject a batch `exec` into `Run`** (the way `runPoint` is injected, ARCH-PURE) —
   `exec(points []P, runPoint func(P) O) []O` runs the batch and returns outputs **in batch order**;
   `Run` then `Tell`s them in that fixed order. Default `exec` = sequential map (today's behavior,
   backward-compatible; pure tests pass the sequential exec).
2. **One GLOBAL concurrency cap `n`, enforced at the leaf.** The runner owns a single limit on the total
   number of concurrent step-subprocess executions across ALL nesting levels — **default `n =
   runtime.NumCPU()`**, overridable by flag + env (`METIS_MAX_PARALLEL` or similar). It must be ONE global
   budget, not a per-`exec` / per-level pool: `Run` nests (`driver ⊃ sweeper ⊃ resample`), so a per-level
   pool composes multiplicatively (5 outer folds × a 99-wide inner sweep = up to 495 subprocesses, not n).
   The clean enforcement — which also avoids the classic **nested-pool deadlock** (an orchestration
   goroutine holding a slot while it waits for child work that also needs slots):
   - **Orchestration ≠ execution.** The driver/sweeper `exec` fan-out is just goroutines — cheap,
     unbounded; they *structure* concurrency but consume NO budget.
   - **The global semaphore (capacity `n`) is acquired only at the LEAF** — the single point that spawns
     a real subprocess (`execStep`, the uv/python config-fold run): acquire immediately before spawn,
     release immediately after. Orchestration goroutines never hold a slot while awaiting children → no
     deadlock, and at most `n` real subprocesses run at once no matter how driver×sweeper fans out.
   - **Caveat (write it into the RUNBOOK/flag help):** each leaf is a Python process that itself may
     multi-thread (BLAS / sklearn `n_jobs`), so `n = NumCPU` processes can oversubscribe the cores —
     the default may want per-process threads pinned (`n_jobs=1` / `OMP_NUM_THREADS=1`) or `n` set below
     NumCPU. A tuning knob, not a correctness issue.
3. **Determinism contract (write it down + test it):** results reduce **order-independently** — config
   identity + fold assignment are content-addressed and `Aggregate` is order-independent (metis#18 M1a),
   so parallel and sequential runs produce an identical `Done(S)`. Parallelism is confined to WITHIN a
   batch; a future order-sensitive adaptive `Tell` still syncs at each `Ask`. `METIS_SEED` determinism is
   unaffected (seeds are per-config, not per-wall-clock).

## Done when

- `Run` takes an injected `exec`; the default sequential exec reproduces current behavior (all existing
  sampler tests green with the new signature).
- A parallel exec and the sequential exec produce a **byte-identical** `Done(S)` for a grid sampler over
  a multi-config × multi-fold set (determinism test — the load-bearing one).
- `metis run` executes a grid sweep concurrently under a **single global cap `n`** (default
  `runtime.NumCPU()`, flag/env-overridable); a wall-clock drop vs. sequential is demonstrated on a
  real/fixture sweep.
- **The global cap holds across nesting:** a `driver: cv` run never has more than `n` step-subprocesses
  in flight at once, even though driver×sweeper fans out to hundreds of orchestration goroutines
  (test via an instrumented leaf exec that records peak concurrency ≤ n). No deadlock under nesting.

## Estimate

Derived against estimate-logic-v3.1 (impl = ship-wall-clock, AI-paired). The design decisions
(leaf-semaphore placement, deadlock-avoidance, determinism contract) are resolved — but that design
was done IN this issue's window (durable plan + recon + fresh-eyes review, all post-claim), so those
design hours are counted (they're part of what `sdlc actual` measures), not free. The impl weight is
the fiddly concurrency test surface (reader-vs-writer atomicity, real-`execStep` serialization,
peak-≤-n under nesting, parallel≡serial determinism).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.3 impl=0.4
item: smaller-go-module      design=0.4 impl=0.6
item: cross-cutting-refactor design=0.2 impl=0.3
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.1 impl=0.15
design-buffer: 0.15
total: 2.80
```

- `smaller-go-module` ×1 (design=0.3 impl=0.4) — `pkg/sampler` exec seam: `SeqExec`/`ParExec`/`execFor` + the `Run` change + barrier-based concurrency tests.
- `smaller-go-module` ×1 (design=0.4 impl=0.6) — `cmd/metis` wiring: leaf semaphore in `execStep`, `--parallel`/env, `syncWriter`, atomic `writeEntry`, the C1 git-probe fix, sort-side-records + the reader-vs-writer / real-`execStep` / peak-≤-n / determinism tests (the bulk).
- `cross-cutting-refactor` (design=0.2 impl=0.3) — thread the `exec` param through the 4 `Run` call sites + every test call site (signature ripple across `pkg/sampler` + `cmd/metis`).
- `milestone-review` (impl=0.2) — the single close-boundary review (single-pass, no `Mx`).
- `atlas-docs` (design=0.1 impl=0.15) — RUNBOOK `--parallel` note (BLAS/thundering-herd caveats) + atlas run/sweep-flow update.

## Plan

Durable plan: `workshop/plans/000031-parallel-batch-exec-plan.md` (fresh-eyes reviewed; findings folded in).
Single-pass atomic work — one close boundary (no `Mx` milestones).

- [x] `exec` seam on `Run` (order-preserving) + `SeqExec`/`ParExec`/`ExecFor`; backward-compat (existing sampler tests green).
- [x] Global leaf semaphore in `execStep` + `--parallel`/`METIS_MAX_PARALLEL` (default `NumCPU`) + `syncWriter`; real-`execStep` serialization test.
- [x] Atomic cache-index write (reader-vs-writer test) + the C1 git-probe false-abort fix.
- [x] Thread `ExecFor` into the 4 sweep `Run` sites; guard `configs`/`points`/`err`/`firstErr`; sort side-records; determinism + peak-≤-n tests.
- [x] Atlas docs + wall-clock demo. *(`sdlc close` pending — the boundary review is run in the main session, not the impl fork.)*

## Log

### 2026-07-13
- 2026-07-13: closed — Build clean + full `go test ./... -race` green across all 9 packages (independently re-run in the main session, not just the impl fork). 7 load-bearing tests present & passing: reader-vs-writer atomicity (I1), real-execStep serialization (I5), sampler+cmd determinism parallel≡serial (M3), peak-≤-n under nested driver:cv no-deadlock, C1 no-false-abort regression, ParExec order-preservation — the atomicity/C1/determinism trio verified RED-first by the fork. --parallel flag wired (default NumCPU, METIS_MAX_PARALLEL override) with the BLAS/thundering-herd caveats in help. Hermetic wall-clock demo: serial 432ms → parallel(8) 95ms = 4.5× (10ms sleeping leaf through the real orchestration). Fresh-eyes plan review (1 Critical + 5 Important + 3 Minor) + both change-code judges (plan-quality + estimate-quality INFO) preceded impl. ACTUAL = N/A (--no-actual): sdlc actual attributes 1.74h across #30/#31/#32/#33 by mention-fallback (interleaved-session, no clean per-issue commit boundary) AND the impl ran in a fork the commit-based measure captures imperfectly — recording N/A rather than polluting velocity calibration with a contaminated number (per the interleaved-sessions lesson).; review verdict: SHIP
- Filed from the kbench#8 sweep-scale discussion (operator, metis-v2). Parallelism = the `Ask` **batch
  width**, a property the sampler already declares — so it's one injected `exec` on the shared `Run`
  loop, not a grid-specific runner. Sibling: metis#30 (progress). The content-addressed, order-independent
  reduce (M1a) is what makes concurrent execution provably safe — the determinism test is the deliverable's
  spine. Bigger perf win than #30 but more care; #30 is the cheaper near-term one.
- **Refinement (operator):** the cap is ONE **global** limit `n` (default `NumCPU`), enforced at the
  **leaf** subprocess spawn (a shared semaphore), NOT per-`exec`/per-level — else nesting multiplies it
  (outer × inner) and it fork-bombs. Orchestration goroutines stay unbounded+cheap and hold no slot while
  awaiting children (deadlock-free); only real `execStep` spawns draw from the budget. Peak-concurrency ≤ n
  under nested `driver: cv` is the guard test. BLAS/`n_jobs` per-process threading can oversubscribe → a
  documented tuning caveat.
- **BUILT (fork, full SDLC: claim → start-plan → durable plan → fresh-eyes review → change-code judges
  [plan+estimate INFO] → TDD).** 5 commits (T1+T2 seam/strategies · T3 semaphore+flag+syncWriter · T4
  atomic index · T5 sweep wiring+guards+C1 · T6 atlas+demo). **Full `go test ./... -race` green** (all 9
  packages). Load-bearing tests: `TestRun_ParExecEqualsSeqExec` (sampler-level determinism) ·
  `TestSweep_ParallelEqualsSerial` (byte-identical ledger+manifest, cmd-level) ·
  `TestNestedCV_PeakConcurrencyWithinCap` (peak ≤ cap under 3×3×2 nesting, no deadlock) ·
  `TestExecStep_SemaphoreSerializesRealSubprocess` (I5 — the REAL execStep acquire, not a fake) ·
  `TestWriteEntry_ReaderNeverSeesTornIndex` (I1 — reader-vs-writer, verified RED against the old
  non-atomic write) · `TestSweep_ProbeFailureDoesNotFalseAbort` (C1 — verified RED against `s != codeID`).
- **Wall-clock demo** (`TestSweep_ParallelWallClockDemo`, hermetic — 10ms sleeping leaf through the real
  orchestration, 3 configs × 2 folds × 4 steps = 24 leaf calls): **serial 432ms → parallel(8) 95ms =
  4.5× speedup.** (The real 99×5 Titanic sweep demo is operator-run — needs kbench's data + would write
  the peer repo; deferred to the operator.)
- **Deviation from plan:** `execFor` exported as `sampler.ExecFor` (cmd/metis is a different package).
  The `out` sync-writer + `leafSem` are built inside `runExperiment` (plan-quality judge INFO #1), not
  `cmdRun`. Task 6 docs pinned to the metis atlas (INFO #2); the kbench RUNBOOK `--parallel` note is a
  deferred peer write. All other steps as planned.
