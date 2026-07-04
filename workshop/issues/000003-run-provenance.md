---
id: 000003
status: open
deps: []
github_issue:
created: 2026-07-02
updated: 2026-07-03
estimate_hours:
---

# Run provenance: snapshot the resolved pipeline config (+ experiment git sha) so ## Runs is knob→score legible

**Stage: DESIGN — design note settled 2026-07-03 (see `## Design`); next is splitting
implementation milestones.** Operator: "don't fix yet, we need to design this well." The skeleton is already useful for discussing direction; this captures
the gap + intent. Part of the **metis v1** project (`brain/data/project/metis-v1.md`);
this issue is **L0** in the v1 layering — the resolved-**point** content-address
`(resolved-with, three repo SHAs, seed)` that is the **repro / run-identity** key
#8's ledger derives from — *not* the cache key (#2 keys itself off the per-step
record's key-material; see `## Design`). Design:
`brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md`.

## Problem

A run records **metrics but not the config that produced them**. `run.json` holds
`{id, experiment, seed, metrics, artifacts}` — no consolidated pipeline/steps
block. The runner does write each step's resolved `with.json` into
`runs/<id>/<step>/with.json`, so config *is* captured per-step in the run dir; but:

- The experiment `.md` frontmatter is **mutable and never snapshotted**. Edit a
  knob (e.g. `model: logreg`→`rf`) and re-run, and `## Runs` just appends a new
  metric line — you **cannot tell which frontmatter produced which score**.
- Reusing a run-id overwrites the per-step `with.json`, losing the prior inputs.

For a bench whose entire value is **knob → score**, this is the central miss.
Today's v0 workaround (a real convention, worth documenting): **treat a run-bearing
experiment file as ~immutable; fork a new file per variation** so each file owns its
own `## Runs` history. Provenance-snapshot is what would later make in-place editing
safe.

## Spec (intent, not yet a plan)

On each run, capture enough to answer "what config produced this score," durably:

- Snapshot the **resolved pipeline** (steps + resolved `with` + seed — the runner
  already parses it) into `run.json` (or a `config.json` in the run dir).
- Stamp the experiment file's **git SHA** (and dirty flag) for exact source
  provenance — git already versions the frontmatter; bind the run to the commit.
- Consider enriching `## Runs` (or a generated index) to show the salient
  config diff beside the metric, so the ledger reads as a knob→score table.
- Design in tandem with metis#2 (caching shares the "resolved-config hash") and
  the fork-per-experiment convention.

## Design (settled 2026-07-03)

Settled together with #2's caching design (see the pensive §Caching + metis#2
`## Design` for the derivation). Core realization: **provenance and the cache key are
the same determinant set, differing only in the operation run on it** — provenance
*inspects / reconstructs* (needs literal values + retrievable bytes), caching
*equality-tests* (needs only a fingerprint). So one raw record serves both — "keep the
raw values, hash last."

### The unified per-step record (the shared spine of #9 / #2 / #3 / #8)

Per step per run, a raw record whose fields split by role:

- **Key-material** (the determinants, hashed → the cache key; precise encoding):
  `step-id`, `resolved-with`, `seed`, `upstream: [output-hash…]` (CAS pointers),
  `code: { D: [(relpath, content-hash)…], deps: uv.lock-digest }`.
- **Provenance-only extras** (NOT hashed into the key — reconstruction aids +
  legibility): `repo-SHAs + dirty-flag`, `fetched: {url, etag, time}` for ingest
  leaves, `output-hash`, `metrics`, `timestamp`.

Provenance = the whole record (raw, inspectable); cache key = `hash(key-material)`. In
shape this is a Nix `.drv` / Bazel action manifest: a raw input list whose hash *is* the
cache identity.

### Three derived views over the one record (all hashed late)

- **cache key** = `hash(key-material)` — #2's skip/recompute currency. Git-SHA
  deliberately *excluded* (it over-invalidates; the per-step content-trace is the precise
  encoding).
- **point-address** = `hash(resolved-with across the DAG, repo-SHAs, seed)` — the L0
  run-identity / repro key **this issue owns**; #8's ledger key derives from it. Git-SHA
  based, human-legible.
- **output key** = the CAS address of what the step produced.

So #3 **owns the record** (writes provenance, renders knob→score, mints the
point-address); #2 **indexes** it (validating trace → skip/recompute); #8 **derives** its
per-row global key from the point-address. One artifact, three consumers. (Supersedes the
earlier framing that the point-address *is* "the cache key" — it is the repro/identity
key; the cache keys itself off the record's key-material.)

### Durability contract (the record, not the CAS, is the source of truth)

The CAS (#9) is a **pure, wipeable cache** — never the sole home of anything
irreplaceable. So `output-hash` is a **cache-pointer, not an archive claim**: present →
reuse; wiped → recompute via the record's recipe, which recurses to durable leaves (code
from git, data refetched from the immutable source). Binding rule:

> **Every record must be recipe-complete against durable homes** (git + external refetch),
> so the whole DAG reconstructs from an *empty* cache.

The one violator — a **non-refetchable, non-git input** (dirty local data, a hand-edited
file) — must be committed to git or the run is flagged non-reproducible (v1: warn; for
Kaggle this never arises).

### Clean-vs-dirty is legibility, not correctness

Because reconstruction is recompute-from-durable-roots, a dirty run reconstructs
identically — clean-vs-dirty only decides whether a run maps to a single nameable commit.
So require a clean repo for **promotable / canonical** runs (a promoted winner should be
commit-nameable); everyday tinker runs may stay dirty.

### `## Runs` legibility + fork-per-experiment (this issue's user-visible core)

The record makes `## Runs` a knob→score table (free-param diff beside the metric);
fork-per-experiment preserves each file's immutable run history *and* yields #2 cache hits
on shared upstream steps (same config → same key-material). Storage: the record is small
metadata → **git** (durable-small, inheriting the brain's replication/encryption); large
bytes stay in the CAS (wipeable) or refetchable externally.

## Done when

- (design-stage) A design note settles what gets snapshotted (config shape, git
  binding), where it lives, how `## Runs` surfaces knob→score, and the
  relationship to the fork-per-experiment convention + caching keys.

## Plan

- [x] Design note: provenance record shape + git binding + ## Runs legibility + relation to #2 and fork-per-experiment. **(settled 2026-07-03 — see `## Design`)**
- [ ] (post-design) split implementation milestones from the `## Design` note.

## Log

### 2026-07-02
- Filed design-stage from the kbench#1 discussion. Operator derived the fork-per-experiment / frontmatter-immutability convention independently and asked to design provenance well before building. Cluster with metis#2 (caching).

### 2026-07-03
- **Design settled** (unified with #2's caching design, multi-round brainstorm). Key move: provenance and the cache key are the *same determinant set*, different operation (reconstruct vs. compare) → **one unified per-step record**, key-material (hashed = cache key) vs. provenance-only extras (repo-SHAs, fetched, metrics). Three derived views: cache key (#2) / point-address (this issue, #8 derives) / output key. Durability contract: CAS is a pure wipeable cache; records must be **recipe-complete against durable homes** (git + refetch) so the DAG reconstructs from an empty cache; clean-vs-dirty demoted to a *legibility* choice (promotable runs → clean). Dropped the stale "point-address is the cache key" self-label. Split the CAS storage primitive out as **metis#9**. Full spec in `## Design`.
