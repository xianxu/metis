---
id: 000007
status: working
deps: [metis#6, metis#3]
github_issue:
created: 2026-07-03
updated: 2026-07-05
estimate_hours:
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

- A `Sampler` interface (`propose_next`/`should_stop`) + a seeded grid impl, pure-core
  unit-tested (the enumeration + stop predicates, no IO).
- A sweep entrypoint (`metis sweep <shape>` or `krun`-level) that drives the loop over
  the existing run path; a small shape fixture sweeps end-to-end to N runs under the fake.
- `should_stop` honored: a budget/target cap stops a grid sweep before exhaustion (tested).
- Cache reuse verified: a sweep over a leaf that doesn't affect `get-data`/`adapt`
  reuses those cached outputs (needs metis#2).

## Plan

- [x] Design settled 2026-07-05 — single driver, ask/tell sampler, content run-id, shape-run identity, detect-and-abort freeze, failed-row (see `## Design`).
- [ ] Stateful `Sampler` (ask/tell) + seeded grid impl (enumeration + budget/target stop); unit test.
- [ ] Single-driver sweep loop over v0's per-point path via `metis run` (experiment = 1-point degenerate); content-address run-id; shape-run identity stamped on ledger rows.
- [ ] Detect-and-abort code-freeze (hash closure at start, re-check per point); failed point → recorded `failed` ledger row + continue.
- [ ] e2e sweep of a fixture shape under the fake → N runs; budget/target early-stop tested; cache-reuse verified (with #2).

## Log

### 2026-07-03
- Filed from the metis-v1 design brainstorm. Grid now; the `should_stop` seam is what leaves room for Optuna/Ax/Hyperband later with no loop change. Deps: metis#6 (expand). Interacts with metis#2 (cache reuse makes sweeps cheap) + metis#8 (records each point).

### 2026-07-05
- **Design settled** (driver/sampler discussion). One driver: `metis run` handles experiment (1 point) and shape (N) uniformly — experiment is the degenerate all-singleton case; `run_one_point` = v0's per-point runner + cache(#2) + record(#3). Sampler refined to a **stateful ask/tell** object (seeded-deterministic), superseding the Spec's pure `propose_next(state)` — matches the Optuna/Nevergrad ecosystem seam (grid: ask=next, tell=noop). Run id = the point's **content-address** (auto, dedup'd), not a timestamp. New: the **experiment-shape-run** has a content identity `hash(shape, code-content, sampler-config, seed)` grouping + stamping its N point-runs (ledger filterable by invocation/code-version). Mid-sweep code mutation → **(C) detect-and-abort** for v1; hermetic+perf upgrade arc (B snapshot-via-CAS → A resident worker) filed as **metis#10**. Per-point failure → recorded `failed` ledger row, sweep continues; resume free via cache; sequential in v1. #8 owns pick-best/promote. Full spec in `## Design`.
