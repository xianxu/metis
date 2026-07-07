---
id: 000024
status: open
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
---

# cache key addressing — input-addressed vs output-hash-chained (interior identity)

## Problem

metis keys each step's cache on `(step-id, uses, with, seed, sorted upstream-OUTPUT-hashes)` — the
interior is **output-hash-chained** (content-addressed). A prior-art survey (Nix, DVC, Nextflow,
Snakemake) surfaced a real fork: the operator leans **input-addressed** — a step's key should be its
**input recipe** (`config + code content-hash + which-rows`), not the *output bytes* of upstream steps.

## Spec

Decision issue (not a feature). Trade-off:
- **Output-hash-chained (today):** "early cutoff" — a code change that yields byte-identical output
  doesn't re-key downstream. But the key is unknowable until upstream runs, and it's fragile to upstream
  output *non-determinism* (a model file with FP/RNG noise re-keys everything below it on a miss).
- **Input-addressed (leaning):** statically **plannable** — a sweep planner computes the cache-hit map +
  cost *before* running. Robust to upstream output non-determinism. Loses early-cutoff — but in ML that
  rarely fires (training outputs aren't byte-reproducible), so little is given up. For a *fold* the two
  are identical (partition output = which-rows); they diverge only for intermediate data steps (features).

Constraint either way: the key stays **two-phase** — `K_pre` (pre-run, from config+seed+upstream) →
**validate** against the runtime-discovered code read-set (metis's validating trace) — AND the reducer's
key must incorporate all folds' **manifested** row-content hashes (not statically knowable from the
shape). So full static plannability is *partial*: structure known up front; fold-content + code-read-set
runtime-established.

Related: metis#25 (root gap — get-data path-hash) is the complementary soundness fix.

## Done when

- A decision (input-addressed vs keep output-chained), recorded with rationale in the atlas.
- If input-addressed: the interior keys on the input recipe; a sweep can print its cache-hit map + cost
  before running; a nondeterministic upstream output no longer spuriously re-keys downstream.

## Plan

- [ ] (spec at claim) evaluate against the current pkg/cache Kpre/Validate; prototype the plannable cache-hit map; decide + document.

## Log

### 2026-07-07
- Surfaced in the metis-v2 design conversation (pensive). The operator's framing: key = input identity
  (recipe + code + rows), not upstream output bytes. Touches existing cache architecture → isolated here.
