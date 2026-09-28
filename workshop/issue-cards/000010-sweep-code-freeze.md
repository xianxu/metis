---
id: 000010
status: open
created: 2026-07-05
updated: 2026-07-05
estimate_hours:
github_issue:
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
