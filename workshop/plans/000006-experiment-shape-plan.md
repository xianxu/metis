---
issue: 000006
title: experiment-shape datatype — the value-level config-space lift + pure expand()
status: active
created: 2026-07-05
---

# Plan: metis#6 — experiment-shape (the config-space lift)

Design **settled 2026-07-04** (issue `## Design`; pensive §The datatype). The lift is **value-level**
on v0's untyped `with` bag (reserved `$`-key maps), NOT a CUE-typing of config leaves — so the heavy
deliverable is a pure Go `Expand(shape) → []point`, and the CUE work is light (the DAG shape + the
`sweep:` block; `with` stays `{[string]: _}`).

## The algebra (recap)

A `with` leaf is a **literal** (fixed), a **dataflow-ref** (a string naming an upstream step, fixed),
or a **space-descriptor** (a reserved `$`-key map). Free variables = exactly the space-descriptor
leaves. `expand`:
- **product** — a plain map `{a:…, b:…}` → cartesian of its fields' expansions.
- **`$any: [v1,…]`** — a set; each value taken verbatim (scalar/list/struct). Sugar for the flat sum.
- **`$oneof: {L1: sub, L2: sub}`** — labeled sum; counts **ADD**; resolves by **bundling** — the chosen
  branch collapses to `{<label>: <resolved-sub-product>}` (e.g. `model: {rf: {n_estimators: 300}}`),
  NOT hoisted to flat siblings.
- **`$linear-range: [lo,hi,steps?]` / `$log-range: [lo,hi,steps?]`** — a domain + metric; grid
  materializes it (`linspace`/`logspace(steps)`, `steps` defaults to `sweep.range_steps`).

Every resolved **point is v0-shaped `with`** — nesting is confined to the shape + `expand()`. Each
point carries its **free-param path** (the branch choices + swept leaf values) = its identity
coordinate → the #8 ledger key + (with the fixed leaves) the #3 point-address. Ragged by construction
(logreg points carry `C`; rf points carry `n_estimators`/`max_depth`).

## Scope line — #6 vs. #7 vs. #8

- **#6 owns (this issue):** the `experiment-shape` datatype (frontmatter schema + the `sweep:` block
  as *carried data*, not executed), the CUE `#ExperimentShape` (+ `#Experiment` as the singleton
  refinement), and the **pure `Expand(shape) → []Point`** with per-point free-param paths. Plus: the
  parser/validator accepts `type: experiment-shape`, and `Expand` of an all-singleton shape yields
  exactly the one v0 experiment.
- **Deferred to #7 (NOT here):** the **sweep loop** — the `propose(domains, history) → next|stop`
  sampler seam, the ask/tell driver, running each expanded point through the (cached #2) runner. #6
  exposes `Expand` as the pure library #7 drives; grid materialization of ranges lives in #6 (it's
  part of expand), but the *iteration/execution* is #7.
- **Deferred to #8 (NOT here):** consuming the free-param path as a ledger key + the namespaced-metrics
  fix. #6 just emits the path per point.
- **`objective`/`sampler` are validated + carried** by #6 (they're `sweep:` fields), consumed by #7/#8.

## Milestones (2 review boundaries)

### M1 — the pure lift: `pkg/shape` + `Expand`

- New pure package `pkg/shape`:
  - `Point{With map[string]map[string]any; FreeParams []FreeParam}` — a resolved experiment's per-step
    `with` (v0-shaped) + its free-param path. `FreeParam{Path string; Value any}` (e.g.
    `adapt.features=[title]`, `train.model=rf`, `train.model.rf.n_estimators=300`).
  - `Expand(steps []experiment.Step, rangeSteps int) ([]Point, error)` — the pure recursion over each
    step's `With`. Walks a `with` value: a `$any`/`$oneof`/`$*-range` map is a space-descriptor (expand
    it); a plain map is a product (cartesian of fields); anything else is a literal. `$oneof` bundles;
    ranges grid-materialize (`linspace`/`logspace`). Deterministic order (sorted keys / declared branch
    order) so point enumeration is stable.
  - Free-param path extraction: only space-descriptor leaves contribute (a `$any` leaf → its
    `step.key=value`; a `$oneof` → `step.key=<label>` + the chosen sub-product's swept leaves). Fixed
    leaves + dataflow-refs never appear (they're not free).
  - Reject malformed descriptors (a map with a `$`-key mixed with plain keys; an unknown `$`-key; a
    `$*-range` whose bounds aren't numbers) with a clear error — a shape author's mistake, surfaced not
    silently mis-expanded.
- Unit tests: the **36-point titanic example** (`features(4) × [logreg:C(3) + rf:(3×2)] = 36`, proving
  `$oneof` ADDs not multiplies); `$any` flat set; nested product; `$oneof` **bundling** (`model:
  {rf:{…}}`, not flat); `$linear-range`/`$log-range` grid materialization (+ `range_steps` default);
  **all-singleton → exactly one point** = the v0 experiment; free-param paths correct + ragged;
  **a range-leaf free-param records the MATERIALIZED value** (`train.C=0.1`, not the `$log-range`
  descriptor — so #8's ledger key stays a concrete coordinate; plan-judge finding 3);
  malformed-descriptor errors. All pure, no IO.
- **M1 review boundary.**

### M2 — the datatype + CUE + parse integration

- CUE `#ExperimentShape` in `construct/vocabulary/experiment.cue`: the **shared field set** (`id`/
  `seed`/`status`/`steps: [...#Step]`, `with` stays `{[string]: _}`) + `type: "experiment-shape"` +
  the `sweep:` block (`sampler: string`, `objective: {metric: string, direction:
  "maximize"|"minimize"}`, `range_steps?: int`). **Single-source the two schemas** (plan-judge finding
  1, ARCH-DRY): make `#Experiment: #ExperimentShape & {type: "experiment", sweep?: _|*_|_}` — i.e.
  `#Experiment` is `#ExperimentShape` refined to `type: "experiment"` with no `sweep`, so the DAG/field
  set is defined ONCE and `#Experiment` derives from it (not two hand-maintained top-level schemas).
  Where CUE can't express "all-singleton", the Go all-singleton→one-point test pins that behavior.
- Go: extend `pkg/experiment` parsing to accept `type: experiment-shape` (the `Sweep` struct +
  shape-typed frontmatter), or a sibling `Shape` type — decide in M1 (likely `Shape` wrapping `[]Step`
  + `Sweep`). Drift guard: a `#ExperimentShape` fixture round-trips (marshal → `cue vet -d`).
- The `experiment-shape` **datatype prototype** (the `xx-datatype`-style frontmatter skeleton) + a
  `titanic-baseline-shape` fixture (the worked example, or a smaller one) that validates and `Expand`s.
- `metis run` on a shape: parse + validate + `Expand`; if all-singleton, run the one point through the
  existing (cached) runner; if multi-point, a clear "this is a sweep — the sweep driver is metis#7"
  message (the sweep loop is #7's, not smuggled here). e2e: the singleton-shape path runs like a v0
  experiment; a multi-point shape validates + expands to the right count.
- atlas: `pkg/shape` + the experiment-shape datatype + the #6/#7/#8 scope line.
- **M2 review boundary** (issue close).

## Open decisions (flag for plan-judge / operator)

1. **`Point.With` as `map[stepID]map[string]any` vs. `[]experiment.Step`.** Leaning the former (the
   resolved per-step config is what a point *is*; steps' id/uses/needs are shape-invariant). A point
   feeds the runner by overlaying its `With` onto the shape's steps. Reversible.
2. **Range materialization lives in `Expand` (grid), not the sampler.** The design says the sampler owns
   *traversal*, but grid materialization (turn `$log-range` into a fixed set) is deterministic and part
   of expand for the grid sampler. An adaptive sampler (#7) proposes points *within* the domain without
   pre-materializing — so `Expand` is the grid-specific expansion; #7's non-grid samplers call a
   different path. Note this seam so #7 isn't surprised. (v1 ships grid only, so `Expand` suffices.)
3. **`metis run` on a multi-point shape** — error-with-pointer-to-#7 vs. run-all-inline. Chose
   error-with-pointer for #6 (the sweep loop is #7's deliverable); #7 flips it to run-all. Keeps the
   scope line clean.

## Test strategy

Pure core (M1) → direct table-driven unit tests over the algebra (the 36-point example is the
keystone). Datatype/CUE (M2) → a `cue vet -d` drift guard + a fixture that validates & expands +
a singleton-shape e2e through the runner. No new external service; controllable time via the existing
injected `Clock`.
