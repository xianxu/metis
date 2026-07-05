---
id: 000010
status: open
deps: [metis#7, metis#9]
github_issue:
created: 2026-07-05
updated: 2026-07-05
estimate_hours:
---

# Mid-sweep code-freeze upgrade: snapshot-at-start then resident worker (hermetic + faster sweeps)

## Problem

metis#7 ships **(C) detect-and-abort** for mid-sweep code mutation: hash the code closure at
sweep start, re-check before each point, and abort if it changed. That keeps a sweep honestly
at one code revision, but it's disruptive — a long sweep (N points × training) *aborts* if you
edit code underneath it, so you can't iterate while a sweep runs, and a slip costs the whole
sweep. Separately, v0's subprocess-per-step model pays a **Python cold-start per step** — a
36-point Titanic sweep × ~6 steps ≈ 216 interpreter starts, each importing pandas/sklearn
(minutes of pure startup). This ticket documents the upgrade arc that makes sweeps **hermetic
(edit-safe)** and **fast**.

Background: the Go orchestrator is already frozen for free (a single `metis run` loads the
binary into memory). The concern is the **Python step code**, re-imported from disk per step.
Per-point correctness holds regardless (each point-run records its actual code-content); what
freezing protects is the **shape-run's "one code-version" invariant** (see metis#7 `## Design`).

## Spec (the upgrade path — deferred; v1 does (C) in metis#7)

- **(B) snapshot-at-start** — at sweep start, content-address the code closure into the CAS
  (metis#9) once, then materialize each step's imports from that **frozen snapshot** (e.g.
  `PYTHONPATH` → the materialized tree). Hermetic and **dirty-safe** (snapshots working-tree
  content, uncommitted included); mid-sweep edits land in the *next* invocation, not this one.
  Reuses the content-address + materialize machinery of #9/#2 — no new snapshot mechanism.
- **(A) resident worker** — run the whole sweep in one long-lived process: the Go orchestrator
  + a **resident Python worker** that loads code + heavy libs **once** and executes every
  step/point in-process. Freezes the Python code in memory (the "single invocation" freeze)
  **and** eliminates the per-step cold-start (~216 starts → ~1). Wins freeze *and* speed. Cost:
  an architecture change from subprocess-per-step, and it **loses per-step process isolation** —
  a leaking/segfaulting step now contaminates the rest, and global library state (numpy/pandas
  config, RNG) can bleed across steps. For a reproducibility-sensitive bench, clean-slate-per-step
  has real value, so this needs a deliberate isolation story (e.g. a pool of warm workers, or
  per-step state reset).

Likely order: **(B) first** (hermetic + dirty-safe, low architectural risk, reuses the CAS),
then **(A)** if the cold-start tax proves painful — weighed against the isolation loss.

## Done when

- (design-stage) A design note settles: pick (B), (A), or (B→A); the snapshot-materialize
  mechanism (or the resident-worker protocol + isolation story); and how it swaps in behind
  metis#7's detect-and-abort without reshaping the sweep loop or the run/record model.
- (then) implementation milestones.

## Plan

- [ ] Design note: (B) snapshot-via-CAS and/or (A) resident-worker; isolation story; swap-in behind #7's (C).
- [ ] (post-design) implementation milestones.

## Log

### 2026-07-05
- Filed from the metis-v1 #7 design discussion. v1 ships **(C) detect-and-abort** (metis#7); this holds the deferred **hermetic + perf** arc: (B) snapshot-at-start via the CAS (#9), then (A) resident worker (freeze the Python code in memory + kill the ~216-per-sweep Python cold-start tax, at the cost of subprocess isolation). `explicitly_out` of the v1 MVP — a post-v1 hardening/perf upgrade. Deps: metis#7 (the sweep driver it upgrades) + metis#9 (CAS, for the snapshot).
