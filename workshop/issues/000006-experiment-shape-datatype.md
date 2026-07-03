---
id: 000006
status: open
deps: []
github_issue:
created: 2026-07-03
updated: 2026-07-03
estimate_hours:
---

# experiment-shape datatype: lift the experiment config schema into a config-space (Space[T])

Part of the **metis v1** project (`brain/data/project/metis-v1.md`). Design source:
`brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md`.

## Problem

v0 has `experiment` — a single reproducible config *instance*. To explore many
configurations (the point of an ML workbench), we need the *type/space* above the
instance: a `experiment-shape` that declares a set of configs and expands into
concrete points. The key insight (design pensive): experiment-shape is the
experiment config schema **lifted** — each leaf value-type `T` becomes a *space over
T* — and `experiment` is the special case where every leaf is a singleton.

## Spec

- **`Space[T]` in CUE:** `Space[#T] = #T | #Set[#T] | #Range[#T] | #Dist[#T]`.
  Because `#T` is inside `Space`, a singleton is a legal space — so the subsumption
  is literal, not analogy.
- **Lift the config schema:** `#ExperimentShape` = the experiment config with every
  leaf `: T` → `: Space[T]`. `#Experiment = #ExperimentShape & {every leaf singleton}`
  — a *refinement*, single-sourced; experiment is NOT a second hand-maintained schema.
  (Open: whether the lift is a clean CUE refinement or wants a small generator — see
  the pensive; resolve toward single-source.)
- **Authoring form:** an `experiment-shape` md datatype prototype
  (`construct/datatype/experiment-shape.md`) — schematized frontmatter (the space) +
  freeform body — mirroring `experiment`. Frontmatter carries the space; the body's
  run-ledger is metis#8.
- **Expansion (Set/Range):** a pure `expand(shape) → [point]` over the free
  (non-singleton) leaves' Sets/Ranges. `Dist` (continuous) leaves are sampled by the
  sampler (metis#7), not enumerated here.
- **Keep experiment recoverable:** a shape with `|space| == 1` expands to exactly one
  point = today's experiment. A conformance test pins `experiment ⊂ experiment-shape`.

## Done when

- `Space[T]` + the lifted `#ExperimentShape` in `construct/vocabulary/`, with
  `#Experiment` derived as the singleton refinement (single-source, conformance-tested).
- `experiment-shape` datatype prototype exists; a `titanic-baseline-shape` fixture
  validates.
- Pure `expand(shape) → [point]` for Set/Range free leaves, unit-tested; the
  all-singleton shape yields exactly one point.
- `metis run`/validator accept an experiment-shape's frontmatter.

## Plan

- [ ] `Space[T]` + lifted schema in CUE; derive `#Experiment` as the singleton refinement; conformance test (experiment ⊂ experiment-shape).
- [ ] `experiment-shape` datatype prototype (mirrors `experiment`); titanic-baseline-shape fixture.
- [ ] Pure `expand(shape) → [point]` (Set/Range; Dist deferred to #7); unit test incl. the singleton→one-point case.
- [ ] Structural validator + atlas entry for the new type.

## Log

### 2026-07-03
- Filed from the metis-v1 design brainstorm. The datatype is the L2 substrate; #7 (sweeper) and #8 (ledger) build on it. No hard dep, but conceptually first in the v1 chain.
