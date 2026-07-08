---
type: type
name: experiment-shape
description: Use when creating or editing an experiment-shape — an experiment lifted into a config-space that a sweep explores. Triggers on "create a sweep", "author an experiment-shape", "sweep these hyperparameters", editing markdown with `type: experiment-shape`, "/xx-datatype experiment-shape". The metis#18 v2 datatype above `experiment`: three phases (`data│pipeline│ship`) plus a `sweeper` (config-level Sampler) and a `driver` (outer Sampler). `with` leaves in `pipeline` may declare a space ($any [list=untagged / map=tagged] or $*-range) that `expand()` collapses into many config points; the engine runs each config × resample fold.
---

# experiment-shape

An experiment-shape is the **config-space above an `experiment`** (metis#18 v2). Where an
`experiment` is a single flat `steps` DAG, a shape is structured into **three phases** and adds a
**sweeper** and a **driver**:

- **`data`** — steps produced ONCE, above the resample, shared across all folds (get-data, adapt).
- **`pipeline`** — the swept `(algorithm × hyperparameter)` atom, run **per-fold**; its `with` leaves
  may declare a *space* via a reserved `$`-key descriptor. The `data│pipeline` boundary is the one
  structural cut that makes everything downstream leakage-safe with no per-step markers.
- **`ship`** — steps that run ONCE on the promoted winner (predict, submission).

The **`sweeper`** and **`driver`** are the two levels of one first-class construct — the **Sampler
fold node** (`Init/Ask/Tell/Done`): a stateful ask/tell fold that proposes points, consumes each
point's result, and reduces to an answer. `metis run` drives them; the run-ledger (metis#8) records
each per-fold point keyed by its free-param path.

`experiment-shape` and its `expand` (metis `pkg/shape`) are owned by **metis** (competition-independent),
like `experiment`. An *instance* lives in a competition workspace (e.g. `kbench/competition/titanic/`).

Relationship to `experiment` (single-source): `#Experiment` is the flat singleton case (one `steps`
DAG, no sweeper/driver); `#ExperimentShape` is the phase-structured config-space. The shared identity
header (`type/id/competition/seed/status`) is defined once in CUE `_meta` (Go: the embedded `Header`
struct), and the per-phase step-list shape once in `_phase` — so neither is hand-maintained twice.

## Frontmatter shape

Validated structurally against `#ExperimentShape` (closed). Semantic checks — combined-DAG acyclicity +
cross-phase `needs`-resolution + `uses` format, **monotonic phase ordering** (a step may only `needs`
an earlier-or-equal phase — defends the `data│pipeline` cut), a non-empty `pipeline`, and the
sweeper/driver invariants — are enforced by `ValidateShape` at read time.

| Field | Required | Notes |
|---|---|---|
| `type` | yes | `experiment-shape` |
| `id` | yes | Slug, lowercase-hyphenated. Matches the filename without `.md`. |
| `competition` | optional | The competition slug (set on kbench instances). |
| `seed` | yes | Integer seed — the reproducibility anchor. |
| `status` | yes | `draft` \| `active` \| `archived`. |
| `data` | optional | Run-once steps above the resample (`with` leaves are fixed here). |
| `pipeline` | yes | The swept per-fold atom (non-empty); `with` leaves may carry `$`-descriptors. |
| `ship` | optional | Winner-only steps (predict/submission). |
| `sweeper` | yes | The config-level Sampler — see below. |
| `driver` | yes | The outer Sampler — exactly one of `single` \| `cv`. |

### The `sweeper` block (config-level Sampler — mlr3 `AutoTuner`)

Proposes configs over the `pipeline` space, owns the **inner** resample that scores each, and the
objective+select that turns per-config `(mean, SE)` into the winner.

- `sampler` — which sampler proposes configs. `grid` ships (M1a; asks for every point); the ask/tell
  seam (metis#7) lets adaptive samplers (Bayesian, Hyperband) slot in with no loop change.
- `resample` — the inner CV: `{cv: {k, stratify?}}` (`k >= 2`). Each config is scored by k-fold CV; the
  resample Sampler's `Done` reduces the k fold-scores → `(mean, SE)`.
- `objective` — `{metric, direction, select}`:
  - `metric` — the **reduced** score name (e.g. `accuracy`) the pipeline emits per fold. **Not** the v1
    `<step>.<metric>` namespacing — the resample Sampler owns the reduction.
  - `direction` — `maximize | minimize` (required).
  - `select` — the rule turning `(mean, SE)` into a winner. `argmax-mean` (M1a); `one-std-err` /
    `pct-loss` are a *different* `Done` over the same cached fold-scores (metis#19).

### The `driver` block (outer Sampler — the honest evaluator, optional in spirit)

Exactly one mode:

- `single: {}` — the degenerate outer Sampler (M1a): fit the sweeper on all data, ship the winner. No
  honest procedure estimate.
- `cv: {k, stratify?}` — nested-CV: run the sweeper on each outer-train, score the sealed outer-test,
  aggregate → the honest procedure estimate. **metis#23** (parsed but rejected at validate in M1a).

### The `$`-descriptor algebra (in `pipeline` `with` leaves)

A `with` leaf is a **literal** (fixed), a **dataflow-ref** (a string naming an upstream step, fixed),
or a **space-descriptor** (a reserved `$`-key map). Only descriptor leaves are swept:

- **`$any`** — the one choice primitive; **dispatches on its argument shape** (the syntax carries the
  type). Both forms **recurse** into their sub-values and counts **ADD**:
  - **`$any: [v1, v2, …]` (list) → untagged sum.** Each alternative is recursively expanded and the
    value is placed **bare** at the leaf. `features: {$any: [[], [title]]}` → `features: [title]`.
  - **`$any: {label: sub, …}` (map) → tagged sum** (conditional/hierarchical params). Each branch is
    recursively expanded and resolved by **bundling** — `model: {$any: {logreg: {C: …}, rf: {n_estimators: …}}}`
    → a point carries `model: {rf: {n_estimators: 300}}` (the `rf` tag preserved), not flat siblings.
    Use the map when alternatives are structured sub-spaces; use the list for simple values.
- **`$linear-range: [lo, hi, steps?]` / `$log-range: [lo, hi, steps?]`** — a domain + metric; the grid
  sampler materializes it (`linspace`/`logspace`). `steps` defaults to a future range-resolution knob.

A plain map `{a: …, b: …}` is a **product** (counts multiply). So `features(4) × [logreg:C(3) +
rf:(3×2)=6] = 36` configs — the `$any` **map** adds, product multiplies. The engine then runs each
config × `resample.cv.k` folds.

## Body

Freeform: the sweep's hypothesis + notes. The structured run-ledger (per-fold rows keyed by the
free-param tuple + fold, `<shape>.ledger.csv`) is metis#8; the shape's body is **immutable input**
(#13) — browse the aggregated `(mean, SE)` leaderboard via on-demand `metis ledger show`, not a summary
written into the body.

## Distinct from siblings

- `experiment` — a single reproducible recipe (the flat singleton; no sweeper/driver).
- `issue` — a unit of work; `project` — the portfolio view. A shape is a *space of recipes* the sweeper
  searches and the driver honestly evaluates.
