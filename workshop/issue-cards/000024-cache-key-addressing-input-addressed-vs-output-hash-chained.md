---
id: '000024'
status: done
started: 2026-07-17T17:31:17-07:00
created: 2026-07-07
updated: 2026-07-17
actual_hours: 0.15
---

# cache key addressing — input-addressed vs output-hash-chained (interior identity)

## Problem

metis keys each step's cache on `(step-id, uses, with, seed, sorted upstream-OUTPUT-hashes)` — the
interior is **output-hash-chained** (content-addressed). A prior-art survey (Nix, DVC, Nextflow,
Snakemake) surfaced a real fork: the operator leans **input-addressed** — a step's key should be its
**input recipe** (`config + code content-hash + which-rows`), not the *output bytes* of upstream steps.
