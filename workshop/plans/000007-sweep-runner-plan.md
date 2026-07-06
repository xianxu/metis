---
issue: 000007
title: Sweep runner + grid sampler — the ask/tell driver over expand + the cached runner
status: active
created: 2026-07-05
---

# Plan: metis#7 — the sweep runner (ask/tell + the loop)

Design **settled 2026-07-05** (issue `## Design`; pensive §Promotion/sampler). Builds directly on
the three merged deps: **#6 `shape.Expand`** (shape → points), **#2 cache** (cheap sweeps), **#3
record** (per-point provenance). #7 is the *driver*: an ask/tell sampler + the loop that runs each
expanded point through the existing cached runner.

## The mechanism (recap)

- **Single driver.** `metis run <file>` already resolves a shape (#6); today a multi-point shape is
  *refused* with a "sweep driver is #7" pointer. #7 **flips that**: a multi-point shape now sweeps.
  An all-singleton shape / plain experiment stays the 1-point case (one code path).
- **Ask/tell sampler** (ecosystem-standard seam): `sampler.Ask() → (Point, ok)`; run the point;
  `sampler.Tell(point, result)`. Grid: Ask = next `Expand` point, Tell = no-op, done on exhaustion
  (or early via a stop predicate). Seeded-deterministic (reproducible from `sampler_seed` + tell
  order). Adaptive samplers (#10+/post-v1) slot in with no loop change.
- **run_one_point IS the existing cached runner** — #7 wraps expand/sampler + loop around it; each
  point runs through `runExperiment`'s core (Validate → TopoSort → exec, + cache #2 + record #3).
- **Run identity = the point's content-address** (not a timestamp) — stable, dedup'd (re-run a point
  → same id → cache hit), dirty-safe. So **resume is free**: interrupt at k/N, re-run → k cache hits,
  rest compute.
- **Per-point failure → recorded, sweep continues** (unlike v0's halt) — one bad config can't kill a
  36-point sweep.

## Scope line — #7 vs. #8 vs. #10

- **#7 owns (this issue):** the ask/tell `Sampler` + grid impl + stop predicates; the sweep loop
  (flip `metis run` multi-point → sweep); per-point execution via the cached runner; run-id =
  point-address; per-point-failure-continues; the **shape-run manifest** (identity + the list of
  point run-ids + each point's free-params/status/metrics) grouping the N runs; v1 **detect-and-abort
  code-freeze**.
- **Deferred to #8 (NOT here):** the **queryable ledger** (CSV keyed by the free-param tuple),
  **pick-best**, and **promote**. #7 emits the per-point results + the shape-run manifest; #8 builds
  the navigable ledger + promotion over them. (#7 writes a concrete manifest.json, not a throwaway
  ledger format — #8 aggregates the manifest + each point's record.json.)
- **Deferred to #10 (NOT here):** the hermetic + faster code-freeze upgrade — (B) snapshot-at-start
  via the CAS, then (A) resident worker. #7 ships only the cheap **detect-and-abort** (hash code
  identity at sweep start via the `gitProbe`, re-check before each point, abort on drift).
- **objective** (#6's sweep block) is consumed here by adaptive samplers (what to optimize); #8's
  pick-best consumes the same declaration. v1 grid ignores objective for *sampling* (it enumerates),
  but the stop predicate can use a target on the objective metric.
- **#6 follow-up carried here:** register the `experiment-shape` noun in ariadne `vocabulary` + extend
  the merge-gate grep, since #7 commits shape *instances* (kbench#4). (Tracked; do at M2 if cheap,
  else log for kbench#4.)

## Milestones (2 review boundaries)

### M1 — the pure sampler (`pkg/sweep`): ask/tell + grid + stop predicates

- New pure package `pkg/sweep`:
  - `Sampler` interface: `Ask() (shape.Point, bool)` (bool=has-next) + `Tell(shape.Point, Result)`.
    `Result{Metrics map[string]float64; Status string}` (a thin per-point outcome).
  - `NewGrid(points []shape.Point, stop StopPredicate) Sampler` — enumerates the pre-expanded points
    in order; Tell is a no-op (grid holds no model); done on exhaustion OR when `stop` fires.
  - `StopPredicate` — `func(history []TellRecord) bool`; v1 predicates: `MaxPoints(n)` (budget) and
    `TargetReached(metric, direction, threshold)` (stop once a point hits the target). Composable
    (`AnyStop(...)`). Pure over the tell history.
  - Deterministic + seeded: grid needs no randomness, but the interface carries a `sampler_seed` slot
    so an adaptive impl is reproducible (documented; grid ignores it).
- Unit tests: grid enumerates all N points in order then reports done; `MaxPoints(k)` stops at k;
  `TargetReached` stops when a told result crosses the threshold (both directions); the empty-shape
  edge; Tell-is-no-op for grid. Pure, no IO.
- **M1 review boundary.**

### M2 — the sweep driver: flip multi-point `metis run` → sweep

- **First extract the per-resolved-point runner seam** (ARCH-DRY, plan-judge): today `runExperiment`
  is monolithic (reads file → `resolveExperiment` [which *refuses* multi-point] → runs+records inline),
  so there's no unit a sweep loop can call per point. Split out `runResolvedExperiment(exp, o, runID,
  now, out)` — the runID/dir setup + cache wiring + `Runner.Run` + `writeRunJSON`/`assembleRecord`/
  `writeRecordJSON`/`appendRunLog` body — and have BOTH the 1-point path and the new sweep loop call it
  (else the loop copy-pastes the run+cache+record wiring). Build `repoSHAs` for the pre-computed
  `PointAddress` runID from the **same** `git.Probe(expDir)` construction `buildRecord` uses (`{repoName:
  sha}` only when `repoName != ""`), so the runID can't drift from the record's internal address on the
  no-git path.
- `cmd/metis`: when `resolveExperiment` sees a multi-point shape, **sweep** instead of erroring:
  - build the grid sampler over `shape.Expand(...)` + the stop predicate (from `sweep` block: a
    `max_points`/`target` if present, else exhaust);
  - loop Ask → run the point through the cached runner (each point = a resolved experiment, run with
    **runID = the point's content-address**, so re-runs dedup + resume-from-cache is free) → Tell;
  - **per-point failure recorded + continue** (a failed point is a `failed` manifest row; the sweep
    proceeds — don't propagate the error);
  - write a **shape-run manifest** `sweeps/<shape-run-id>/manifest.json`: the shape-run identity
    (`hash(shape-content, code-identity, sampler-config, seed)`), and per point `{run_id (=point-addr),
    free_params, status, metrics}` — the grouping #8's ledger aggregates.
- **Detect-and-abort code-freeze (v1):** snapshot the code identity (git HEAD + dirty-hash via the
  injected `gitProbe`) at sweep start; re-check before each point; abort with "code changed at point
  k/N — re-run to sweep the new revision" on drift. Protects the shape-run's one-code-version identity;
  per-point correctness holds regardless (each point's record.json carries its actual code).
- Run-id = the point's L0 content-address — **`record.PointAddress(point.With, repoSHAs, seed)`**
  (REUSE #3's minter, ARCH-DRY — do NOT re-derive a parallel `CanonicalHash(With+free-params)`, which
  would omit `repoSHAs`+`seed` and be code-INSENSITIVE → re-running a point after a commit would collide
  on the same `runs/<id>/` dir and overwrite its `record.json`, breaking "each point records its actual
  code-content"). The `repoSHAs` come from the **same `gitProbe` read the detect-and-abort already
  does** — one read, shared. This is exactly the identity #8's ledger derives from, so the run dir, the
  record, and the ledger key all agree.
- e2e (the payoff): a multi-point shape (test/echo, no uv) **sweeps to N runs**; **cache reuse
  verified** (an upstream step shared across points HITs after point 1 — reuses #2); a **failing
  point is recorded + the sweep continues**; a **stop predicate** (`max_points`) halts early; the
  **manifest** lists the N points with free-params. A `--dry-run` lists points without running (cheap
  preview).
- atlas: `pkg/sweep` + the sweep-driver flow + the #7/#8/#10 scope line.
- **M2 review boundary** (issue close).

## Open decisions (flag for plan-judge / operator)

1. **Run-id = point content-address vs. a free-param slug.** The design says content-address (stable,
   dedup, dirty-safe). A free-param slug (`features=title,model=rf`) is human-legible but collides
   across code-versions. Chose content-address for the run *dir*; the free-param tuple is the human
   key #8's ledger uses. (Both are in the manifest.)
2. **Where the stop predicate's config lives — DECIDED: a CLI flag.** v1 wires the budget stop via
   `metis run --max-points N` (keeps the shape declaratively pure — a sweep's *extent* is an
   invocation choice, not part of the shape's identity, so it must NOT enter the point-address / shape
   content). `TargetReached` is built + unit-tested in M1 (it reads `sweep.objective`'s metric+
   direction, already in `Sweep`) and wired opportunistically via `--target <value>`; `--max-points`
   is the keystone budget stop the M2 e2e pins. This is fixed now so M1's `TargetReached` test and
   M2's `max_points` e2e aren't rewritten.
3. **Detect-and-abort granularity** — git HEAD+dirty-hash (coarse, cheap, reuses `gitProbe`) vs. the
   #2 per-step trace closure (precise). Chose coarse for v1 (the design's "hash the code closure" is
   satisfied by the repo dirty-state for the freeze *invariant*; per-point correctness already uses
   the precise #2 trace). #10 upgrades this.

## Test strategy

Pure sampler (M1) → table-driven unit tests (enumeration, both stop predicates, edges). Driver (M2) →
the multi-point-sweep e2e over test/echo (no uv): N runs, cache-reuse (upstream HIT across points),
failure-continues, stop-predicate, manifest shape, detect-and-abort (mutate the fake gitProbe
mid-sweep → abort). Fixtures in `t.TempDir()`; controllable time + fake gitProbe already exist.
