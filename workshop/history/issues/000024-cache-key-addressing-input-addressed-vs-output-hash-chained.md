---
id: 000024
status: done
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-17
estimate_hours:
started: 2026-07-17T17:31:17-07:00
actual_hours: 0.15
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

- [x] (spec at claim) evaluate against the current pkg/cache Kpre/Validate; ~~prototype the plannable cache-hit map~~ (deferred by decision — Revisions); decide + document. DONE: decided (input-addressed, #18 M1a-3b) + documented (atlas, incl. trade-off).

## Log

### 2026-07-07
- Surfaced in the metis-v2 design conversation (pensive). The operator's framing: key = input identity
  (recipe + code + rows), not upstream output bytes. Touches existing cache architecture → isolated here.

### 2026-07-07 (folded into metis#18 M1a + soundness fix)
- **Decision: input-addressed, FOLDED INTO M1a** as its own `cache identity` review boundary (M1a-3) —
  it shares the reducer-`Done`-key surface M1a builds anyway, so one coherent cache-identity design beats
  build-on-output-hashing-then-rewrite.
- **Soundness fix (M1a plan review):** the naive swap (delete `Kpre`'s upstream *output-hash* term, use
  upstream `Kpre`) is **unsound** — metis's read-set `D` deliberately EXCLUDES data/upstream artifacts
  (`trace.py`), so the output-hash-chain is the *only* carrier of upstream-**code-edit** propagation
  downstream. Deleting it → an edit to `features.py` re-runs `features` (its own `D`) but NOT `train`
  (whose `Kpre` uses `features`' code-invariant `Kpre`, and whose `D` excludes `features`' output) →
  `train` serves a stale output. **Required pairing:** a **transitive-`D` snapshot stored in each step's
  OWN `Entry`** — a topo-fold `transitiveD[id] = ownD ∪ ⋃_{d∈needs} transitiveD[d]`, validated against
  the current tree. (A re-review round-2 rejected the naive "walk upstreams' live entries at hit-check":
  the topo executor *heals* an edited upstream's `Entry.D` before the downstream is validated → the walk
  re-hashes clean → stale HIT. The downstream must carry its own snapshot; store & validate then key on
  the same bytes — symmetric — and it's eviction-robust + diamond-correct.) Distinguishes a code change
  (MISS) from output nondeterminism (HIT — the win we want). Needs an `Entry`-schema field (`TransitiveD`).
  Plan + real-executor soundness-gate test in `workshop/plans/000018-*-plan.md` (M1a-3, Tasks 11-13).
  Lessons in `workshop/lessons.md`.

## Revisions

### 2026-07-17 — closed as decision-complete (the machinery shipped in #18 M1a-3b)
- 2026-07-17: closed — decision-record close, docs-only: decision+rationale+trade-off in atlas/index.md pkg/cache section; machinery shipped+e2e-proven in #18 M1a-3b (cited); hit-map deferred by decision (Revisions); actual 0.15h labeled judgment (docs window); review verdict: SHIP
- **Done-when clause 1 (the purpose): SATISFIED** — decision = input-addressed, made 2026-07-07,
  implemented + hardened in #18 M1a-3b (`Kpre` on upstream K_pres + the transitive-D soundness
  snapshot; e2e-proven), recorded WITH rationale + trade-off in `atlas/index.md` (pkg/cache
  section; trade-off lines added at this close).
- **Done-when clause 2 (conditional elaboration): partially by-events** — interior keys on the
  input recipe ✓; nondeterministic upstream no longer re-keys ✓; the **pre-run cache-hit-map
  printout stays UNBUILT by decision** (the Spec's own framing: "Decision issue (not a
  feature)"; the next arena's rule is zero new workbench features until a competition demands
  them — a cost-preview command gets filed THEN if the pain is real).
- metis#25 closed the complementary root gap same day (declared content pins) — the
  content-addressed interior is now end-to-end trustworthy, which is what this issue's Problem
  statement asked the pair to deliver.
