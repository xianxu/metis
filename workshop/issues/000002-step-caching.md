---
id: 000002
status: open
deps: [metis#9, metis#3]
github_issue:
created: 2026-07-02
updated: 2026-07-03
estimate_hours:
---

# Uniform DAG step caching: content-address step inputs, skip unchanged, recompute only what changed

**Stage: DESIGN — design note settled 2026-07-03 (see `## Design`); next is splitting
implementation milestones (don't build until they're split).** Captured from the kbench#1
post-live-run discussion; the design came first. This ticket holds the intent + constraints
+ the settled design. Part of the **metis v1** project (`brain/data/project/metis-v1.md`);
the caching design is also summarized in the pensive
`brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md`
(§Caching) — content-addressed `cas/<hash>` pool, working-tree paths become pointers, the
code-version term realized as a **validating trace** (below), orthogonal to git so
branch-switch never invalidates.

## Problem

`metis run` re-executes the *entire* DAG every time (`cmd/metis/run.go` →
`Runner.Run` → TopoSort → execute-each; no skip/cache logic). So every run
re-downloads from Kaggle (`get-data`, network) and re-trains (compute) even when
nothing about those steps changed. For a **learning bench** whose loop is "tinker
one knob, re-run," that is needlessly slow and re-hits external services each run.

## Spec (intent, not yet a plan)

A **uniform, consistent** cache — the same mechanism whether the step downloads
data, engineers features, splits folds, or trains. The runner resolves **what
actually needs recomputing** from the DAG and skips the rest. Design goals:

- **Content-addressed, not timestamp-based.** A step's cache key = a hash of
  everything determining its output: `uses` (+ the step-type's code version), the
  **resolved** `with`, the **seed**, and the **cache keys of its upstream inputs**
  — so a change propagates downstream but not sideways. Edit adapt's FEATURES →
  adapt + everything downstream recompute; `get-data` is reused.
- **Uniform across layers.** A runner-level concern keyed off the step contract,
  not per-step code — identical for kaggle/kbench/metis steps.
- **Honest invalidation.** A step-type code change must bust the key (hash the
  executable / a declared version) or the cache silently serves stale output — the
  classic caching bug. External-fetch steps (`get-data`) need a policy knob
  (cache the download vs. always re-pull).
- **Cluster with metis#3 (provenance) + fork-per-experiment.** A sibling
  experiment sharing `get-data`/`adapt` config should hit a prior run's cache.
  This is the "tinker a small portion, the system resolves what to compute"
  vision — design the three together.

## Design (settled 2026-07-03)

Derived over a 5-round brainstorm with the operator; the full derivation lives in the
pensive (§Caching). The cache is **one mechanism — a validating trace** — over two input
classes with one quarantine. This refines the Spec above in two places: the "step-code
version" term is realized *by the trace* (no git-SHA / executable hash), and "cache keys of
upstream inputs" becomes upstream **output** content-hashes (see Leaf policy).

### Two keys, not one

- **`K_pre`** — ex-ante, cheap, computable *without running* the step:
  `hash(resolved-with, seed, upstream-output-hashes, step-id)`. The "business-logic" key a
  caller can determine up front. Upstream is referenced by its **output content-hash**, not
  its cache-key (load-bearing — see Leaf policy).
- **`D`** — the step's **recorded read-set**, captured *after* a run: each read file as
  `(repo-relpath, content-hash)`. Filter: keep first-party code+config under the repo root;
  collapse `site-packages` → the single `uv.lock` digest; drop stdlib/system/temp; **reads
  only** (never writes, never the step's own output); upstream artifacts are *not* in `D`
  (they enter via `K_pre`'s output-hashes).
- **Output store**: `hash(K_pre, content_hash(D)) → cas/<hash>`. Both halves are needed —
  `K_pre` carries the *values* (params/seed/upstream), `content_hash(D)` carries the
  *code+config bytes*. Same code + different `with` → shared `D`, different `K_pre`; a code
  edit → shared `K_pre`, different `content_hash(D)`.

### Lifecycle — validate ≠ execute

1. Compute `K_pre` (cheap, in hand).
2. **No stored `D`** for this `K_pre` → *cold* → must run; trace reads → record `D` +
   output. (The only path that executes compute.)
3. **Stored `D` exists** → fetch it, **re-hash each path** (stat+sha of a few small files —
   ms). All match → **HIT**, materialize the stored output, *no execution*. Any differ →
   **MISS** → run, re-record `D` (the read-set itself may have shifted).

A hit *reuses* the `D` a prior miss wrote — nothing is recorded on a hit, because a hit is a
non-run. **Code-version invalidation falls out for free**: an edited `model.py` is a path in
`D` whose hash moved → miss. No git-SHA term and no static import-closure needed.

### The soundness invariant

> A step's input-set can only change if a recorded input changes.

Holds for files because *what* a step reads is decided by code, which is itself in `D` — you
can't make a step read something new without editing a file whose hash is already in `D`.
Every residual hole is a determinant that is *not* a recorded file (below). Sweeps get the
target behaviour for free: shared upstream steps share `K_pre` across grid points, so point 1
records them and points 2..N validate-and-hit; only the varying downstream step re-runs. The
pool accretes across sweeps, sessions, and branches (branch-switch never invalidates — it
just asks for different keys).

### Two input classes + one quarantine

1. **Keyed immutable artifacts** — anything *ingested or produced* (downloads, raw external
   files, upstream outputs). Content-addressed *once* at its boundary, referenced by
   output-hash forever after, never re-hashed. Sound by CAS immutability.
2. **Traced source files** — first-party code + small in-tree config. Recorded in `D`,
   re-hashed to validate.
3. **Quarantined impure ingress** — network / env / clock — pushed to explicit boundaries so
   they never sit in the input-deciding role the invariant reserves for class 2.

### Leaf policy — the key-chain is only as sound as its leaves

Pure interior transforms (adapt, split, train: file → file) are sound with **no** external
assumption. Ingest leaves (get-data, any external read) are not sound for free — they depend
on state (the network response) in neither `D` nor `K_pre`. Two separable questions:

- **Downstream propagation** is sound *iff* (a) the leaf re-observes and (b) downstream keys
  on the leaf's **output content-hash** (not its cache-key — an impure leaf can produce
  different bytes under an unchanged cache-key). Then different bytes → downstream `K_pre`
  moves → re-run.
- **Skipping the fetch** needs **pin** (assume the source immutable — a *scoped, conscious*
  invariant violation confined to one step, justified empirically: Kaggle competition data
  is frozen/versioned) or **etag/version** (fold a cheap pre-fetch identifier into `K_pre` —
  *repairs* the invariant by making "which version" a recorded input). Asymmetry to remember:
  content-hashing the download fixes *propagation* but not the *round-trip* (you had to fetch
  to hash); only a pre-fetch id buys the skip.

**v1 policy:** fetch-once-and-freeze (manual "immutable leaf" marking), etag where the source
offers one. Sufficient for the bench.

### Env / determinism

Env vars are ambient inputs (the filesystem's wall clock): lift any output-determining one
into `with` → `K_pre` (same move as the seed); ignore the rest. Accept-by-convention that
libraries don't branch *output* on env. (Thread-count → float-reduction-order is a separate
*reproducibility* concern, not the cache's job.)

### Observation sensor

Record reads via Python audit hooks (`sys.addaudithook`; `open`/`import` events). Honest
limit: a C-extension `fopen` (some pandas/numpy paths) bypasses the Python layer, so this is
a *lower-bound* sensor. The airtight version is a **sensor swap** (syscall trace —
`fsatrace`/strace/seccomp), not a redesign; `D`'s definition is unchanged. Deferred — route
heavy data through framework helpers so the read stays on the observable path.

### Storage, the record home, and durability (settled 2026-07-03)

- **Storage substrate = metis#9** (content-addressed blob store, `cas/<hash>`), split out
  as the generic primitive #2 sits on. #2 owns *policy* (keying, skip/recompute); #9 owns
  *mechanism* (put/get by hash).
- **The `D`-record lives in #3's unified per-step record** (settled — not a separate cache
  sidecar). #2 is the *validating-trace/index over* that record; #3 *owns* it. One raw
  record, hashed-late into three views (cache key / point-address / output key). See
  metis#3 `## Design`.
- **Durability contract:** the CAS is a **pure, wipeable cache** — never the sole home of
  anything irreplaceable. `output-hash` is a **cache-pointer, not an archive claim**:
  present → reuse; wiped → recompute via the record's recipe, which recurses to durable
  leaves (code from git, data refetched from the immutable source). Every record must be
  **recipe-complete against durable homes** so the DAG reconstructs from an *empty* cache;
  the one violator — a non-refetchable, non-git input — is committed to git or the run is
  flagged non-reproducible (v1: warn). Rooted/durable archival of a promoted closure is a
  post-v1 add-on on #8's promotion hook.
- Prior art for the whole scheme: verifying/constructive traces ("Build Systems à la
  Carte"), gcc `-MMD` depfiles / ccache depend-mode, Nix derivations + GC roots, DVC
  (pointers-in-git + content store), Bazel/Nix sandboxes for the airtight leaf.

## Done when

- (design-stage) A design note settles: cache-key composition, where cached
  outputs live + are keyed, code-version invalidation, the external-fetch policy,
  and the interaction with run provenance. Only then split into build milestones.

## Plan

- [x] Design note: cache-key composition + invalidation + storage + external-fetch policy + interaction with #3. **(settled 2026-07-03 — see `## Design`)**
- [ ] (post-design) split implementation milestones from the `## Design` note.

## Log

### 2026-07-02
- Filed design-stage from the kbench#1 discussion. Operator: "we shouldn't hurry; caching needs to be uniform and consistent… a well-designed cache greatly improves iteration efficiency by tinkering a small portion and letting the system resolve what to recompute." Cluster with metis#3 + the fork-per-experiment convention.

### 2026-07-03
- **Design settled** (5-round brainstorm with the operator). Cache = a single **validating trace**: ex-ante `K_pre` + recorded read-set `D`, output at `hash(K_pre, content_hash(D))`; validate by **re-hashing `D`, not re-running**; code-version invalidation falls out for free (dropped the git-SHA / static-import-closure idea). Central invariant: *"a step's input-set changes only if a recorded input changes."* Two input classes (keyed immutable artifacts vs. traced source files) + one quarantine (network/env/clock at explicit ingest boundaries). Key subtleties surfaced: downstream must reference upstream by **output-hash**, not cache-key; the key-chain is only as sound as its leaves, so ingest leaves need pin (immutable bet) or etag (version-in-key); folder-scoped git-log was rejected as *unsound* (shared imports leak the boundary). Full spec now in `## Design`. Design-stage done-when met; next is splitting impl milestones.
