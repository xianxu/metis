---
id: 000007
status: open
deps: [metis#6]
github_issue:
created: 2026-07-03
updated: 2026-07-03
estimate_hours:
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

## Done when

- A `Sampler` interface (`propose_next`/`should_stop`) + a seeded grid impl, pure-core
  unit-tested (the enumeration + stop predicates, no IO).
- A sweep entrypoint (`metis sweep <shape>` or `krun`-level) that drives the loop over
  the existing run path; a small shape fixture sweeps end-to-end to N runs under the fake.
- `should_stop` honored: a budget/target cap stops a grid sweep before exhaustion (tested).
- Cache reuse verified: a sweep over a leaf that doesn't affect `get-data`/`adapt`
  reuses those cached outputs (needs metis#2).

## Plan

- [ ] `Sampler` interface + seeded grid impl (pure enumeration + stop predicates); unit test.
- [ ] The sampler-agnostic sweep loop over the existing single-run path.
- [ ] `metis sweep` (or krun) entrypoint; e2e sweep of a fixture shape under the fake.
- [ ] Budget/target `should_stop` predicates; test early-stop. (Cache-reuse test lands with metis#2.)

## Log

### 2026-07-03
- Filed from the metis-v1 design brainstorm. Grid now; the `should_stop` seam is what leaves room for Optuna/Ax/Hyperband later with no loop change. Deps: metis#6 (expand). Interacts with metis#2 (cache reuse makes sweeps cheap) + metis#8 (records each point).
