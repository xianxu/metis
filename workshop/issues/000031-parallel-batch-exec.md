---
id: 000031
status: working
deps: []
github_issue:
created: 2026-07-13
updated: 2026-07-13
estimate_hours:
started: 2026-07-13T14:15:42-07:00
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

## Plan

- [ ] (spec at claim) `exec` seam on `Run` (order-preserving return) + sequential default; a **global
  leaf semaphore** (cap `n`, default `NumCPU`, flag/env) acquired at the `execStep` spawn — NOT per-level;
  determinism test (parallel ≡ sequential `Done`); peak-concurrency ≤ n test under nested `driver: cv`;
  wall-clock demo.

## Log

### 2026-07-13
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
