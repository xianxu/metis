---
type: type
name: experiment-shape
description: Use when creating or editing an experiment-shape — an experiment lifted into a config-space that a sweep explores. Triggers on "create a sweep", "author an experiment-shape", "sweep these hyperparameters", editing markdown with `type: experiment-shape`, "/xx-datatype experiment-shape". The metis#6 datatype above `experiment`: same pipeline, but `with` leaves may declare a space ($any/$oneof/$*-range) that `expand()` collapses into many concrete experiment points. The sweep driver (metis#7) runs them; the ledger (metis#8) records them.
---

# experiment-shape

An experiment-shape is the **config-space above an `experiment`** — the type of which a
single experiment is the all-singleton point. It has the same pipeline (`steps` DAG) as an
experiment, but any `with` leaf may declare a *space* of values via a reserved `$`-key
descriptor; a pure `expand(shape) → [point]` (metis `pkg/shape`) collapses the shape into
concrete v0-shaped experiment points. The **sweep driver** (`metis run` + the sampler, metis#7)
runs each point through the cached runner; the **run-ledger** (metis#8) records them keyed by
each point's free-param path.

`experiment-shape` and its `expand` are owned by **metis** (platform-independent), exactly like
`experiment`. An *instance* lives in a competition workspace (e.g. `kbench/competition/titanic/`).

Relationship to `experiment` (single-source): `#Experiment = #ExperimentShape` collapsed to
singletons — the shared pipeline field set is defined once (CUE `_pipeline`), and `#Experiment`
narrows it to `type: experiment` with no sweep. A shape with every descriptor collapsed to one
value expands to exactly one experiment, and `metis run` on such a shape runs it like a v0
experiment.

## Frontmatter shape

Validated structurally against `#ExperimentShape`. Semantic checks (DAG acyclicity, `needs`
resolution, `uses` format, sweep block) are enforced by `ValidateShape` at read time.

| Field | Required | Notes |
|---|---|---|
| `type` | yes | `experiment-shape` |
| `id` | yes | Slug, lowercase-hyphenated. Matches the filename without `.md`. |
| `competition` | optional | The competition slug (set on kbench instances). |
| `seed` | yes | Integer seed — the reproducibility anchor. |
| `status` | yes | `draft` \| `active` \| `archived`. |
| `steps` | yes | The pipeline DAG (same as `experiment`; `with` leaves may carry `$`-descriptors). |
| `sweep` | yes | The sweep block — see below. |

### The `sweep` block

- `sampler` — which sampler drives the space. `grid` ships (v1); the `propose/should-stop` seam
  (metis#7) lets adaptive samplers (Optuna, LHS) slot in with no loop change.
- `objective` — `{metric, direction}` (`direction: maximize | minimize`). Declared once; consumed
  by adaptive samplers (what to optimize) and metis#8's promotion (pick-best).
- `range_steps` — optional; default grid resolution for a `$*-range` that omits its own `steps`.

### The `$`-descriptor algebra (in `with` leaves)

A `with` leaf is a **literal** (fixed), a **dataflow-ref** (a string naming an upstream step,
fixed), or a **space-descriptor** (a reserved `$`-key map). Only descriptor leaves are swept:

- **`$any: [v1, v2, …]`** — a set; each value taken verbatim. `features: {$any: [[], [title]]}`.
- **`$oneof: {label: sub, …}`** — a labeled sum; counts **ADD** (conditional params); resolves by
  **bundling** — `model: {$oneof: {logreg: {C: …}, rf: {n_estimators: …}}}` → a point carries
  `model: {rf: {n_estimators: 300}}`, not flat siblings.
- **`$linear-range: [lo, hi, steps?]` / `$log-range: [lo, hi, steps?]`** — a domain + metric; the
  grid sampler materializes it (`linspace`/`logspace`). `steps` defaults to `sweep.range_steps`.

A plain map `{a: …, b: …}` is a **product** (counts multiply). So
`features(4) × [logreg:C(3) + rf:(3×2)=6] = 36` points — `$oneof` adds, product multiplies.

## Body

Freeform: the sweep's hypothesis + notes. The structured run-ledger (per-point rows keyed by the
free-param tuple) is metis#8; a shape's body carries a top-N summary + a pointer to the ledger.

## Distinct from siblings

- `experiment` — a single reproducible recipe (the all-singleton point of a shape).
- `issue` — a unit of work; `project` — the portfolio view. A shape is a *space of recipes* to sweep.
