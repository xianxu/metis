---
id: 000031
status: open
deps: []
github_issue:
created: 2026-07-13
updated: 2026-07-13
estimate_hours:
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
2. **A bounded worker-pool exec** in `cmd/metis` — concurrency capped (flag / `GOMAXPROCS`-derived
   default) so 2,475 subprocesses don't fork-bomb. Nesting composes: the driver's outer-fold batch and
   each inner sweep's batch each go through `exec`, so total parallel surface = outer × inner (bound the
   TOTAL, not per-level, to avoid k² processes).
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
- `metis run` executes a grid sweep concurrently with a **bounded** pool (configurable); a wall-clock
  drop vs. sequential is demonstrated on a real/fixture sweep. Concurrency never exceeds the bound
  (incl. nested driver × sweeper).

## Plan

- [ ] (spec at claim) `exec` seam on `Run` (order-preserving return) + sequential default; bounded
  worker-pool exec in `cmd/metis` (total-bound across nesting); determinism test (parallel ≡ sequential
  `Done`); wall-clock demo.

## Log

### 2026-07-13
- Filed from the kbench#8 sweep-scale discussion (operator, metis-v2). Parallelism = the `Ask` **batch
  width**, a property the sampler already declares — so it's one injected `exec` on the shared `Run`
  loop, not a grid-specific runner. Sibling: metis#30 (progress). The content-addressed, order-independent
  reduce (M1a) is what makes concurrent execution provably safe — the determinism test is the deliverable's
  spine. Bigger perf win than #30 but more care; #30 is the cheaper near-term one.
