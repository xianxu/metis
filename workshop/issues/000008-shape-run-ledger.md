---
id: 000008
status: open
deps: [metis#6, metis#7, metis#3]
github_issue:
created: 2026-07-03
updated: 2026-07-03
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

## Done when

- Sweeps (metis#7) append rows to the shape's CSV sidecar carrying both keys + metrics
  + a point snapshot; batched commit boundaries; the body summary regenerates.
- A `show`/query command renders + sorts/filters the ledger.
- `promote` materializes a ledger row as an all-singleton experiment with a back-link;
  round-trips (the promoted experiment reproduces the row's run).
- Shape-mutability guard (immutable-with-runs, or per-row point snapshot) is enforced +
  tested.

## Plan

- [ ] Ledger schema (CSV columns: free-param tuple, cas hash, metrics, status, point snapshot); writer appends per run (metis#7 hook); batched commits.
- [ ] Body top-N summary generation + sidecar pointer.
- [ ] `show`/query view over the sidecar (sort/filter/diff).
- [ ] `promote <shape> <row>` → all-singleton experiment + back-link; round-trip test.
- [ ] Shape-mutability guard (immutable-with-runs / per-row snapshot); test.

## Log

### 2026-07-03
- Filed from the metis-v1 design brainstorm. The L1 tracking layer — the piece that actually solves "don't get overwhelmed." Deps: metis#6, metis#7, metis#3 (global key). The free-param tuple as the human key is the elegant bit (falls out of the schema lift).
