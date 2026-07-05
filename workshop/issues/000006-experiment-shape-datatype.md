---
id: 000006
status: working
deps: []
github_issue:
created: 2026-07-03
updated: 2026-07-05
estimate_hours: 2
started: 2026-07-05T16:41:40-07:00
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

## Design (settled 2026-07-04)

Settled over the metis-v1 syntax brainstorm (derivation in the pensive §The datatype).
This **refines the Spec above in two ways**, forced by the v0 ground truth (config is an
*untyped bag*): (1) the lift is a **value-level vocabulary**, not a CUE refinement of
typed leaves — v0's CUE types only the DAG *shape* (`id`/`uses`/`needs`; `with` is
`{[string]: _}`), so there are no typed config leaves to lift; (2) the `#Dist`/`loguniform`
leaf is **dropped** — a parameter declares its **domain + metric** (`$log-range`), and the
*sampler* owns traversal (a distribution is not a domain).

### The lift is value-level, on the untyped `with` bag

A `with` leaf is a **literal** (fixed), a **dataflow-ref** (a literal string naming an
upstream step — also fixed), or a **space-descriptor** (a reserved `$`-key map). *Free
variables = exactly the space-descriptor leaves* — opt-in, so dataflow-refs and fixed
values never get swept. Encoding = reserved-`$`-key YAML maps, which drop into v0's
`map[string]any` with **zero YAML-parser change**; the shape-expander recognizes the
`$`-keys. `#Experiment = #ExperimentShape & {all-singleton}` then holds at the
**expand-semantics** level (a literal *is* a size-1 space; `expand(all-singleton)` = the
one v0 experiment) — the CUE schema is untouched. Single-source without CUE-typing config
values.

### The algebra: sum + product + leaf

- **product** — a plain map `{a: …, b: …}`: all fields present, counts **multiply**.
- **sum** — choose one; **one operation, two ergonomic surfaces**:
  - **`$any: [v1, v2, …]`** — sugar for the flat case; alternatives are values (scalars,
    lists, or structs, taken verbatim); discriminator = the value.
  - **`$oneof: {L1: sub, L2: sub}`** — labeled branches each carrying a sub-space;
    discriminator = the label. Counts **ADD** across branches — this is what makes
    conditional params correct (`logreg.C` + `rf.n_estimators`, not ×).
  - (`$any` desugars to `$oneof` with value-labels + empty sub-spaces; kept only for
    ergonomics — flat sets are ~90% of lifts. Prior art: sklearn `ParameterGrid`'s
    list-of-dicts = this exact sum-of-products; Nevergrad's `Choice`/`Dict` compose the
    same way.)
- **leaf domains** — **`$linear-range: [lo, hi, steps?]`** / **`$log-range: [lo, hi, steps?]`**:
  the domain **plus its metric** (intrinsic geometry), *not* a distribution. `steps` is an
  optional positional 3rd element (grid resolution; grid-only; defaults to
  `sweep.range_steps`); other samplers ignore it. (Supersedes Spec's `#Set`/`#Range`/`#Dist`:
  `$any` = Set, `$*-range` = domain-with-metric, no `Dist`.)

### `expand(shape) → [point]`

Pure recursion: product → cartesian; sum → union of `branch × expand(sub)`; a leaf `$any`
→ its values; a `$*-range` → the sampler-materialized set (grid → `linspace`/`logspace(steps)`).
**`$oneof` resolves by *bundling*** (follows the hierarchy — settled): the chosen branch
collapses to a single-key map `{<label>: <resolved-branch-product>}`, e.g.
`model: {rf: {n_estimators: 300, max_depth: 8}}` — *not* hoisted to flat siblings. A leaf
lift resolves to the scalar (`C: {$any: […]}` → `C: 1`). So **every resolved point is
v0-shaped `with`** — nesting is confined to the shape + `expand()`; steps read
`w["model"]` = `{rf: {…}}` and never learn about spaces.

The point's **free-param path** (the branch choices + leaf values) is its identity
coordinate → both the human **free-param tuple** (→ #8 ledger key) and, with the fixed
leaves, the **point-address** (→ #3). Ragged by construction (logreg points carry `C`; rf
points carry `n_estimators`/`max_depth`) → sparse ledger columns (→ #8).

### The `sweep:` block — one artifact, no separate "sweeper" concept

Carried in the shape frontmatter alongside `steps:`:
- **`sampler: grid`** — the `propose(domains, history) → next | stop` seam (→ #7); grid
  ships (walks the tree / discretizes ranges), adaptive samplers (Optuna, LHS) slot in with
  no loop change.
- **`objective: {metric, direction}`** — declared **once**, consumed by *both* adaptive
  samplers (what to optimize) and #8's promotion (pick-best). Requires unambiguous metric
  names → the v0 flat-merge collision (step metrics last-write-wins) must become
  per-step/namespaced (a #8/#3 fix).
- **`range_steps: N`** — global default grid resolution for any `$*-range` without its own
  `steps`.

### Worked example — `titanic-sweep`

```yaml
---
type: experiment-shape          # an `experiment` = this with every $any/$oneof/$*-range collapsed to a singleton
id: titanic-sweep
competition: titanic
seed: 42
status: active
sweep:
  sampler: grid
  objective: {metric: cv_score, direction: maximize}
  range_steps: 6
steps:
  - id: get-data
    uses: kaggle/download
    with: {competition: {slug: titanic}}
  - id: adapt
    uses: titanic/adapt
    needs: [get-data]
    with:
      raw: get-data
      out: ../data/titanic
      features: {$any: [ [], [title], [title, family], [title, family, age_bin] ]}
  - id: split
    uses: metis/cv-split
    needs: [adapt]
    with: {dataset: adapt, k: 5, stratify: true}
  - id: train
    uses: metis/train
    needs: [split]
    with:
      dataset: adapt
      folds: split
      model:
        $oneof:
          logreg: { C: {$any: [0.1, 1, 10]} }          # or {$log-range: [0.01, 100, 5]}
          rf:     { n_estimators: {$any: [100, 300, 500]}, max_depth: {$any: [4, 8]} }
  - id: predict
    uses: metis/predict
    needs: [train]
    with: {dataset: adapt, model: train}
  - id: submission
    uses: titanic/submission
    needs: [predict]
    with: {predict: predict}
---
```

Point count = `features(4) × [ logreg:C(3) + rf:n_est(3)×depth(2)=6 ]` = **36** (logreg + rf
*add*, no `logreg × n_estimators` garbage). Two expanded points:

| free-param path (= ledger key) | `adapt.features` | `train.with` (bundled) |
|---|---|---|
| `(features=[title], model=logreg, C=1)` | `[title]` | `{dataset:adapt, folds:split, model:{logreg:{C:1}}}` |
| `(features=[title,family], model=rf, n_estimators=300, max_depth=8)` | `[title,family]` | `{…, model:{rf:{n_estimators:300, max_depth:8}}}` |

### The closed vocabulary

- `$any: [ … ]` — set (sugar for the flat sum).
- `$oneof: { label: subspace, … }` — labeled sum (conditional branches).
- plain map — product.
- `$linear-range` / `$log-range: [lo, hi, steps?]` — domain + metric.
- `sweep: { sampler, objective: {metric, direction}, range_steps }`.
- literal / dataflow-ref — fixed (no `$`-key).

## Done when

- `Space[T]` + the lifted `#ExperimentShape` in `construct/vocabulary/`, with
  `#Experiment` derived as the singleton refinement (single-source, conformance-tested).
- `experiment-shape` datatype prototype exists; a `titanic-baseline-shape` fixture
  validates.
- Pure `expand(shape) → [point]` for Set/Range free leaves, unit-tested; the
  all-singleton shape yields exactly one point.
- `metis run`/validator accept an experiment-shape's frontmatter.

## Plan

Durable impl plan: `workshop/plans/000006-experiment-shape-plan.md` (the algebra recap, scope line
vs. #7/#8, 2 review boundaries). TDD; the pure `Expand` core (M1) is reviewed before the datatype/CUE
integration (M2).

- [x] Design settled 2026-07-04 — value-level `$`-vocab + algebra + `sweep:` block + expand/bundling (see `## Design`); impl decomposed into the durable plan (2026-07-05).
- [ ] **M1 — the pure lift** (`pkg/shape` + `Expand`). `Point{With, FreeParams}`; `Expand(steps, rangeSteps) → []Point` — the pure recursion (product→cartesian; `$any`→set; `$oneof`→bundled labeled sum that ADDs; `$*-range`→grid linspace/logspace). Free-param path per point (only space-descriptor leaves; ragged). Malformed-descriptor errors. Unit tests: the **36-point titanic example** ($oneof adds not multiplies), bundling, range materialization + `range_steps` default, **all-singleton→exactly one v0 point**, ragged free-param paths, malformed errors.
- [ ] **M2 — datatype + CUE + parse integration.** CUE `#ExperimentShape` (`type`, `steps` DAG with untyped `with`, the `sweep:` block; `#Experiment` = singleton refinement) + drift guard. Go `Shape` parse (`type: experiment-shape` + `Sweep`). `experiment-shape` datatype prototype + a `titanic-baseline-shape` fixture that validates + expands. `metis run` on a shape: parse+validate+Expand; all-singleton → run the one point (cached runner); multi-point → clear "sweep driver is metis#7" pointer (the sweep loop is #7). e2e (singleton-shape runs like v0; multi-point expands to the right count). Atlas.

## Log


- 2026-07-05: closed M1 — M1 pure lift: go build+vet+test ./... green. pkg/shape Expand — 7 tests incl the 36-point titanic keystone (proves $oneof ADDs: features(4)×[logreg:C(3)+rf:(3×2)]=36), $oneof bundling ({label:sub}), $any set, product×set, all-singleton→exactly-one-v0-point (byte-identical with), ragged free-param paths, $*-range→grid (linspace/logspace)+range_steps default (materialized value in free-param), malformed-descriptor errors (mixed $/plain, unknown $-key, non-numeric bounds). All pure, no IO. BYPASS --no-atlas + --no-project: M1 is the pure core; atlas (shape datatype + flow) + project tracker land at M2/final-close per the plan; milestone progress in the issue Plan/Log.; review verdict: FIX-THEN-SHIP
### 2026-07-03
- Filed from the metis-v1 design brainstorm. The datatype is the L2 substrate; #7 (sweeper) and #8 (ledger) build on it. No hard dep, but conceptually first in the v1 chain.

### 2026-07-04
- **Design settled** (syntax brainstorm). Lift is **value-level** on v0's untyped `with` bag (reserved `$`-keys), not CUE-typed leaves — reconciles the Spec's `Space[T] in CUE` framing with v0 reality (config is untyped; CUE types only the DAG shape). Algebra: `$any` (set, sugar) / `$oneof` (labeled conditional sum — branches ADD, fixing `logreg.C` vs `rf.n_estimators`) / plain map (product) / `$linear-range`·`$log-range: [lo,hi,steps?]` (domain+metric, **not** a distribution — dropped `Dist`/loguniform, the sampler owns traversal). `expand()` **bundles** `$oneof` to `{label:{…}}` (follows hierarchy, not flat siblings) and yields v0-shaped `with`. A `sweep:` block (sampler / objective{metric,direction} / range_steps) lives in the shape frontmatter — one artifact, no separate sweeper concept. **Cross-cutting decisions captured here pending their own passes:** #7 (the `propose(domains,history)→next|stop` seam + the objective feeding adaptive samplers), #8 (free-param-tuple ledger key, ragged→sparse columns, promotion driven by `objective`, and fixing v0's flat last-write-wins metric merge). Prior art: sklearn `ParameterGrid` list-of-dicts + Nevergrad `Choice`/`Dict` = this exact algebra.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.5 impl=0.4
item: smaller-go-module      design=0.2 impl=0.3
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.05 impl=0.05
design-buffer: 0.15
total: 1.96
```

Design pre-settled → design near the floor. M1 greenfield `pkg/shape` (the pure `Expand` algebra — the
keystone, 36-point example); M2 a smaller-go-module *extend* (CUE `#ExperimentShape` + `Shape` parse +
runner integration + fixture). Two `milestone-review` (2 boundaries). A small atlas note. Impl at
40%-of-v2 (v3.1); +15% thorough-plan buffer.
- **M1 built — the pure lift `pkg/shape`** (TDD, all green; build+vet+full-suite clean). `Point{With, FreeParams}` + `FreeParam{Path, Value}`; `Expand(steps, rangeSteps) → []Point` — the pure recursion: product (map) → cartesian; `$any` → verbatim set; `$oneof` → **bundled** labeled sum that **ADDs** (`{label: resolved-sub}`); `$linear-range`/`$log-range` → grid (linspace/logspace, `range_steps` default). Per-point **free-param path** (only descriptor leaves; ragged; range leaves record the *materialized* value). Malformed descriptors (mixed `$`+plain keys, unknown `$`-key, non-numeric bounds) error. Tests: **the 36-point titanic keystone** (`features(4) × [logreg:C(3) + rf:(3×2)] = 36`, proving `$oneof` adds not multiplies), product×set, all-singleton→1-point (byte-identical v0 with), ragged free-params, range→grid + `range_steps` default, malformed errors. Next: M2 (CUE `#ExperimentShape` + `Shape` parse + runner integration + fixture).
