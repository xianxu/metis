---
id: 000007
status: working
deps: [metis#6, metis#3, metis#2]
github_issue:
created: 2026-07-03
updated: 2026-07-05
estimate_hours: 1.9
started: 2026-07-05T17:31:17-07:00
---

# Sweep runner + grid sampler (propose_next / should_stop abstraction)

Part of the **metis v1** project (`brain/data/project/metis-v1.md`). Design source:
`brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md`.
Depends on metis#6 (experiment-shape + `expand`).

## Problem

Given an `experiment-shape` (a config-space), run its points — the L2 execution.
Start simple (grid) but leave a clean seam for smarter exploration later (Bayesian,
early-stopping), without rewriting the sweep loop each time.

## Spec

- **Sampler interface (the seam):** `propose_next(state) → point | done` +
  `should_stop(state) → bool`. The **stop-function is the generalization** — it lets
  even grid terminate early (budget/target), and it's exactly what adaptive samplers
  need.
- **Sweep loop (fixed, sampler-agnostic):**
  `while not sampler.should_stop(state): p = sampler.propose_next(state); run(p); state.record(result)`.
  New algorithms = new sampler impls; the loop never changes.
- **Grid sampler (ship this):** enumerate the cross-product of the free params'
  Set/Range values (via metis#6 `expand`); `should_stop` fires on exhaustion, or early
  on a budget (max runs) / target (metric threshold) predicate.
- **Each point runs through the existing single-run path** — an experiment-shape point
  IS an experiment (metis#6), so a sweep is N pinned runs (L0), recorded via the ledger
  (metis#8), reusing cache (metis#2) so shared upstream steps aren't recomputed per point.
- **Determinism:** the sampler is seeded; the same shape + seed + sampler yields the
  same sequence of points (reproducible sweeps).

## Design (settled 2026-07-05)

Settled over the metis-v1 driver/sampler discussion (pensive §Promotion, immutability, the
sampler). **Refines the Spec in two ways:** the sampler is a **stateful ask/tell object** (not
a pure `propose_next(state)`/`should_stop(state)`), and there is **one entrypoint** (`metis run`,
not a separate `metis sweep`) — an experiment is a shape collapsed to one point, so one driver
handles both.

### Single driver — experiment is the degenerate 1-point case

`metis run <file>` parses an experiment *or* an experiment-shape, `expand`s the free leaves
(#6) into points, and drives a loop; an all-singleton experiment yields exactly one point, so
the two share one code path (backward-compatible). `run_one_point` **is** v0's existing
per-point runner (Validate → TopoSort → exec-steps) + cache (#2) + record (#3) — #7 just wraps
`expand`/sampler + loop around it. (A `--dry-run` that expands and lists points without running
is a cheap preview.)

### Stateful ask/tell sampler (seeded)

The seam is the ecosystem-standard **ask/tell** (Optuna `study.ask/tell`, Nevergrad, skopt, Ax)
— impure by design, because adaptive samplers hold a model of the space:

    sampler = make_sampler(shape.sweep, domains, sampler_seed)
    while (point = sampler.ask()) is not DONE:
        result = run_one_point(point)
        sampler.tell(point, result)      # grid: no-op; adaptive: update model

Grid: `ask` = next cross-product point (via #6 `expand`), `tell` = no-op, DONE on exhaustion (or
early on a budget/target predicate — the stop generalization). The sampler is
**seeded-deterministic** (stateful but reproducible from `sampler_seed` + the tell sequence) —
reproducibility comes from the seed, not from purity. This subsumes the Spec's
`propose_next(state)`: history is the sampler's internal state, accumulated via `tell`.

### Run identity — content-address, auto-assigned

A point-run's id = the **point's content-address** (#3), not a timestamp slug: stable, dedup'd
(rerun a point → same id → cache hit, no duplicate), and dirty-safe (use the content identity so
iterating on dirty code doesn't collide). The wall-clock timestamp drops to a provenance field.
Uniform across experiment (1 run) and shape (N runs).

### The experiment-shape-run — a content identity that groups its N point-runs

An invocation of a shape has its own identity: `hash(shape-content, code-content@invocation,
sampler-config, seed)`. Cheap to mint, and worth it — it **groups** the N point-runs it produced
and **stamps each ledger row**, so the accumulating ledger (#8) is filterable by invocation /
code-version ("results at *this* code" vs "all results ever"). For **grid** the point-set is
derivable (`expand`), so the shape-run record is thin (identity + config). For **stochastic**
samplers (post-v1) the trajectory isn't derivable → the record adds the `sampler_seed` + point
sequence.

### Mid-sweep code freeze — (C) detect-and-abort (v1)

A sweep takes a while, during which code can change underneath it. The Go orchestrator is frozen
for free (single binary in memory); the concern is the **Python step code** (subprocess-per-step
re-imports from disk). **v1 = detect-and-abort:** hash the code closure at sweep start (reuse
#2's trace), re-check before each point, abort on drift ("code changed at point k/N — re-run to
sweep the new revision"). Cheap, observable, fail-fast; it enforces the shape-run's
one-code-version identity. **Per-point correctness holds regardless** (each point records its
actual code-content); freezing only protects the shape-run invariant. The hermetic + faster
upgrade arc — **(B) snapshot-at-start via the CAS**, then **(A) resident worker** — is deferred
to **metis#10**.

### Per-point failure, resume, ordering

- **Failure → recorded, sweep continues.** A point whose step errors is written to the ledger as
  a `failed` row (null metrics), and the sweep proceeds — one bad config can't kill a 36-point
  sweep (unlike v0's halt-on-step-failure, which still fails *that* point).
- **Resume is free from the cache.** Interrupt at k/N, re-run the shape → the k done points are
  cache hits (instant), the rest compute. No resume machinery.
- **Sequential in v1.** The ask/tell seam is inherently sequential (adaptive needs results before
  the next ask). Points are independent, so grid could parallelize later; sequential + cache is
  fine for the bench.

### #7 / #8 boundary

#7 drives + samples + records each point to the ledger. **#8 owns the ledger + pick-best +
promote** — the winner/promotion is not #7's job. The `objective` (sweep block, #6) is consumed
by #8 (pick-best) and by adaptive samplers here (what to optimize) — one declaration, both.

## Done when

*(Updated 2026-07-05 to the settled Design — the original bullets described the
`propose_next`/`should_stop` + `metis sweep` interface the Design superseded with ask/tell + the
`metis run` flip.)*

- An ask/tell `Sampler` interface (`Ask()`/`Tell()`) + a seeded grid impl, pure-core unit-tested
  (enumeration + the `MaxPoints`/`TargetReached` stop predicates, no IO).
- `metis run` on a multi-point shape **sweeps** (flips #6's refusal) — driving the loop over the
  existing cached run path; a small shape fixture sweeps end-to-end to N runs under the fake. Run-id
  = the point's `record.PointAddress` (re-runs dedup + resume-from-cache is free); a **shape-run
  manifest** groups the N point-runs (the #8 handoff).
- A stop predicate honored: `--max-points` caps a grid sweep before exhaustion (tested).
- Cache reuse verified: a sweep over a downstream leaf reuses the shared upstream cached outputs
  (metis#2). Per-point failure is recorded + the sweep continues; detect-and-abort halts on mid-sweep
  code drift.

## Plan

Durable impl plan: `workshop/plans/000007-sweep-runner-plan.md` (mechanism recap, scope line vs.
#8/#10, 2 review boundaries). TDD; the pure ask/tell sampler (M1) is reviewed before the driver (M2).

- [x] Design settled 2026-07-05 — single driver, ask/tell sampler, content run-id, shape-run identity, detect-and-abort freeze, failed-row (see `## Design`); impl decomposed into the durable plan (2026-07-05).
- [ ] **M1 — the pure sampler** (`pkg/sweep`). `Sampler` ask/tell interface (`Ask() (Point, bool)` + `Tell(Point, Result)`); `NewGrid(points, stop)` (enumerate in order, Tell no-op, done on exhaustion/stop); `StopPredicate` — `MaxPoints(n)` + `TargetReached(metric, direction, threshold)` + `AnyStop(...)`, pure over tell-history; seeded slot (grid ignores). Unit tests: grid enumerates all N then done, MaxPoints stops at k, TargetReached stops on threshold-cross (both directions), empty-shape edge, Tell-no-op.
- [ ] **M2 — the sweep driver.** Flip `metis run` on a multi-point shape from erroring → **sweeping**: build the grid sampler over `Expand`, loop Ask → run each point via the cached runner (runID = **`record.PointAddress(point.With, repoSHAs, seed)`** — REUSE #3's minter, repoSHAs from the same `gitProbe` read detect-and-abort does; NOT a code-insensitive re-derivation, else re-runs collide + overwrite) → Tell; **per-point failure recorded + sweep continues**; write the **shape-run manifest** (`sweeps/<id>/manifest.json`: shape-run identity + per-point run-id/free-params/status/metrics — the #8 handoff). **Detect-and-abort code-freeze** (git identity via `gitProbe` at start, re-check per point, abort on drift). Stop predicate wired via `--max-points`. e2e: multi-point shape sweeps to N runs; **cache reuse verified** (upstream HIT across points); failure-continues; `--max-points` early-stop; manifest shape; `--dry-run` preview. Atlas.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.4 impl=0.35
item: smaller-go-module      design=0.2 impl=0.4
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.05 impl=0.05
design-buffer: 0.15
total: 1.9
```

Design pre-settled → design near floor. M1 greenfield `pkg/sweep` (ask/tell + grid + stop predicates —
pure); M2 a smaller-go-module *extend* of `cmd/metis` (flip multi-point run → sweep loop + manifest +
detect-and-abort + 2 e2es — the integration breadth is where the hours land). Two `milestone-review`
(2 boundaries). A small atlas note. Impl at 40%-of-v2 (v3.1); +15% thorough-plan buffer.

## Log

### 2026-07-03
- Filed from the metis-v1 design brainstorm. Grid now; the `should_stop` seam is what leaves room for Optuna/Ax/Hyperband later with no loop change. Deps: metis#6 (expand). Interacts with metis#2 (cache reuse makes sweeps cheap) + metis#8 (records each point).

### 2026-07-05
- **Design settled** (driver/sampler discussion). One driver: `metis run` handles experiment (1 point) and shape (N) uniformly — experiment is the degenerate all-singleton case; `run_one_point` = v0's per-point runner + cache(#2) + record(#3). Sampler refined to a **stateful ask/tell** object (seeded-deterministic), superseding the Spec's pure `propose_next(state)` — matches the Optuna/Nevergrad ecosystem seam (grid: ask=next, tell=noop). Run id = the point's **content-address** (auto, dedup'd), not a timestamp. New: the **experiment-shape-run** has a content identity `hash(shape, code-content, sampler-config, seed)` grouping + stamping its N point-runs (ledger filterable by invocation/code-version). Mid-sweep code mutation → **(C) detect-and-abort** for v1; hermetic+perf upgrade arc (B snapshot-via-CAS → A resident worker) filed as **metis#10**. Per-point failure → recorded `failed` ledger row, sweep continues; resume free via cache; sequential in v1. #8 owns pick-best/promote. Full spec in `## Design`.
- **M1 built — the pure sampler `pkg/sweep`** (TDD, all green; build+vet+full-suite clean). `Sampler` ask/tell interface (`Ask() (Point, bool)` + `Tell(Point, Result)`); `Grid` (enumerates `shape.Expand`'s pre-expanded points in order, Tell no-op, done on exhaustion or stop-predicate); `Result{Metrics, Status}`, `TellRecord`. `StopPredicate` (pure over tell-history): `MaxPoints(n)` (budget), `TargetReached(metric, direction, threshold)` (both directions; a missing-metric/failed point never trips it), `AnyStop(...)` (compose). Seeded slot documented (grid ignores). Tests: grid enumerates-N-then-done, empty→immediately-done, MaxPoints stops at k, TargetReached maximize+minimize, missing-metric-ignored, AnyStop. Next: M2 (extract `runResolvedExperiment`; flip multi-point `metis run` → sweep loop + manifest + detect-and-abort).
