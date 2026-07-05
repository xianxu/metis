---
id: 000008
status: open
deps: [metis#6, metis#7, metis#3]
github_issue:
created: 2026-07-03
updated: 2026-07-05
estimate_hours:
---

# Shape run-ledger: CSV sidecar keyed by free-param tuple + promotion to an experiment

Part of the **metis v1** project (`brain/data/project/metis-v1.md`). Design source:
`brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md`.
Depends on metis#6 (experiment-shape), metis#7 (sweep runner), metis#3 (point
content-address).

## Problem

A sweep produces thousands of runs. Keeping each as a git-commit-per-run is affordable
but *unnavigable* (`git log` over 10k near-identical commits is unreadable). The runs
must be **durable AND navigable** — a structured, queryable table — without drowning
the ML engineer or churning git per run.

## Spec

- **The shape owns a structured run-ledger — a CSV sidecar**, one row per point.
  Human-navigable (sort/filter/diff by param or metric); it IS the `dvc exp show` /
  MLflow table, embedded + human-keyed.
- **Two keys per row:**
  - **Free-param tuple** (the human key): within a shape the free (non-singleton)
    params fully determine the point → `(model=logreg, k=5, lr=0.01)` is a complete,
    legible name (a float for a `Dist` leaf).
  - **Resolved-point content-address** (the global key, from metis#3): the cache/repro
    identity, unique across shapes.
- **Row contents:** the two keys + the run's metrics + status + a pointer to its CAS
  artifacts. Enough to reproduce (the point snapshot) so the ledger survives shape edits.
- **Body summary:** the shape's freeform body holds a top-N-by-metric summary + a
  pointer to the sidecar (not the full 10k rows — keep the file readable).
- **Batched commits:** commit per-sweep or every ~1K runs — no per-append churn.
- **Immutability discipline:** a shape *with* rows is ~immutable (fork to change the
  space) OR each row snapshots its full resolved point; enforce/warn one way.
- **Promotion:** `promote <shape> <row>` → materialize that point as a standalone
  all-singleton `experiment` file (for hand-iteration / the "best" on `main`), with a
  back-link to the originating shape+row. The durable spine = the sequence of promotions.
- **A `show`/query view** (`metis sweep show <shape>` / `krun show`) that renders the
  sidecar as a sorted/filtered table.

## Design (settled 2026-07-05)

Settled over the driver/ledger/durability discussion (pensive §L1 + §Promotion; builds on
#6/#7/#3). Refines the Spec: the ledger is **append-only**, row identity is the
**point-address** (not the free-param tuple — it repeats across code-versions), there's a
**third key** (the sweep-SHA), columns are **ragged/sparse**, metrics are **namespaced**, and
promotion is a **command** (the `*` inline marker is dropped).

### What a row is — the raw recipe + result

`(free-param tuple | sweep-SHA | namespaced metrics | status)` + the derived point-address. The
row is the **raw reconstructable recipe** (literal free-params, the code reference, the seed) +
the result — a Nix-derivation-shaped record. Hashes are *derived columns* (CAS-output-lookup +
dedup), never the stored identity; humans never type them.

### Three keys (the Spec had two)

- **Free-param tuple** — the human navigation key; **ragged/sparse**: with `$oneof` (#6) logreg
  rows carry `C`, rf rows carry `n_estimators`/`max_depth`, so columns = the **union** of all
  branches' free-params, blank where inactive.
- **Sweep-SHA** — the shape-run identity (#7): the git commit the sweep ran at. **Doubles as the
  human code-version address** — a git short-SHA (`a1b2c3d`) is the eyeballable handle, so a run
  is addressed by `(sweep-SHA, free-param tuple)`; metis invents no id.
- **Point-address** (#3) — the global content identity; **row identity for dedup**.

### Accumulation — append-only, point-address identity

- **Re-run same code → idempotent** (same point-addresses → same rows; nothing added).
- **Re-run at new code → new rows**, each free-param tuple now appearing once per code-version,
  grouped by sweep-SHA.
- Rows are **immutable** (a deterministic point-run's result is fixed; a `failed` point stays a
  `failed` row; fixing it = new code = a new row). "Keep every run durable" = content-addressed
  append. Navigation: best-ever = argmax objective over all rows; "at current code" = filter by
  sweep-SHA; "config X over time" = filter by tuple, group by sweep-SHA.

### The lift unification — one ledger, experiment = 1-config

The ledger is the **lifted `## Runs`**: an `experiment` is a **1-config ledger** (empty
free-param tuple, rows across code-versions — #3's structured `## Runs`); an `experiment-shape`
is the **N-config ledger** + promotion. One datatype; the whole thing lifts consistently —
config→space (#6), run-log→ledger (#8), record→per-step (#3).

### Physical form + the metric-collision fix

An **append-only CSV sidecar** (`show`/query renders sorted/filtered views — sorting is a
*view*, never a storage concern), committed **batched** (per-sweep). The body holds a generated
**top-N-by-objective** summary + a pointer. **Namespaced metrics** (per-step, e.g.
`train.cv_score`) fix v0's flat last-write-wins collision; the sweep's `objective` (#6) names one
unambiguously.

### Pick-best — objective-driven

The `objective: {metric, direction}` (#6 sweep block) drives both the body's top-N ordering and
promotion's selection. Scope: default **whole-ledger champion** (best-ever); `--sweep <SHA>`
restricts to one invocation.

### Promotion — a command, not a marker

```
promote <shape> [--best | --point (model=rf, n_estimators=300)] [--sweep <SHA>] --name titanic-winner
```

→ writes the all-singleton `titanic-winner.md` (the row's raw point, `expand`⁻¹) and, because the
winning point's code is already a committed SHA (§Durability), **commits the experiment.md at that
code** → a **single self-contained, durable, reproducible commit**. The `*` inline marker is
dropped (it would force a *sorted* ledger + manual file-marking, fighting append-only). Since
every sweep *captures* its code (below), **every row is already reproducible** — so promotion
isn't "keep reproducible," it's "graduate a point to a named, editable experiment." Promoting an
older winner (v1 while at v2) = the deliberate `checkout <v1-SHA>` + write + commit ("go back").
Promoting *two* points → two files (the primitive is point→experiment.md).

### Durability — git owns code, the CAS owns only wipeable output bytes

The capture mechanism (refines #2/#3/#9): a sweep **captures its code revision** by, on a CAS
miss, using the trace to find the closure files and — if any are dirty/untracked — **committing
just those to a side ref** (`refs/metis/sweeps/*`), so `main` stays clean and every sweep runs at
a real committed SHA. What metis persists per step is a **manifest of pointers**:
`(path, git-blob-hash, commit)` — *git's blob-hash is the content-hash; git's (commit, path) is
the location*. metis invents no code hash and stores no code bytes.
- **Recovery** = resolve each `(path, hash, commit)` from git (`checkout`/`cat-file`); recover a
  past dirty version = `git checkout <its sweep-SHA>`.
- **Durability by construction** — the manifest lives in the durable records, the blobs in git
  (side-ref, GC-protected). Wiping the CAS loses **zero** code and **zero** provenance.
- **The CAS is a wipeable `content-hash → bytes` map for large *outputs* only** (recompute on
  miss). One-line invariant: *the CAS holds nothing whose loss isn't recomputable; everything
  irreplaceable — code manifest, metrics, git blobs — lives in git.*

### Immutability discipline

A shape (or experiment) *with rows* is ~immutable — fork to change the space (each row already
snapshots its full resolved point + code SHA, so old rows stay reproducible across edits).

## Done when

- Sweeps (metis#7) append rows to the shape's CSV sidecar carrying both keys + metrics
  + a point snapshot; batched commit boundaries; the body summary regenerates.
- A `show`/query command renders + sorts/filters the ledger.
- `promote` materializes a ledger row as an all-singleton experiment with a back-link;
  round-trips (the promoted experiment reproduces the row's run).
- Shape-mutability guard (immutable-with-runs, or per-row point snapshot) is enforced +
  tested.

## Plan

- [x] Design settled 2026-07-05 — append-only ledger (3 keys, sparse cols, namespaced metrics), objective pick-best, promote command, git-owns-code durability (see `## Design`).
- [ ] Append-only CSV writer (3 keys + sparse free-param cols + namespaced metrics + status); batched per-sweep commit; the sweep-captures-code side-ref hook.
- [ ] Body top-N-by-objective summary + sidecar pointer; `show`/query sorted/filtered views.
- [ ] `promote <shape> [--best|--point ..] --name X` → all-singleton experiment committed at its code SHA (self-contained); round-trip test.
- [ ] Immutability guard (shape-with-rows ~immutable / per-row snapshot); test.

## Log

### 2026-07-03
- Filed from the metis-v1 design brainstorm. The L1 tracking layer — the piece that actually solves "don't get overwhelmed." Deps: metis#6, metis#7, metis#3 (global key). The free-param tuple as the human key is the elegant bit (falls out of the schema lift).

### 2026-07-05
- **Design settled** (ledger/durability discussion). Append-only CSV, **row identity = point-address** (not the free-param tuple — it repeats across code-versions); **three keys** (free-param tuple [human, ragged/sparse via `$oneof`] + sweep-SHA [code-version, = the git short-SHA human address] + point-address [global dedup]); **namespaced metrics** (fixes v0 flat last-write-wins); **objective-driven** pick-best (whole-ledger default). The ledger is the **lifted `## Runs`** — experiment = 1-config ledger, unifying with #3. **Promotion = a command** (`*` marker dropped) → writes an all-singleton experiment committed **at its code SHA** = a self-contained reproducible commit. **Durability refined (updates #2/#3/#9):** a sweep captures its code by committing the traced closure to a side ref on a miss; metis persists a `(path, git-blob-hash, commit)` **pointer-manifest** (git's blob-hash = the content-hash; git = content store) — **the CAS holds only wipeable large-output bytes; code lives in git**, durable across CAS wipes. Full spec in `## Design`.
