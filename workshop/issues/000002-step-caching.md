---
id: 000002
status: codecomplete
deps: [metis#9, metis#3]
github_issue:
created: 2026-07-02
updated: 2026-07-05
estimate_hours: 3
started: 2026-07-05T14:32:51-07:00
actual_hours: 1.57
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

## Spec

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

### Revision (2026-07-05): git owns code; the CAS owns only wipeable output bytes

The durability model above is refined by the #8 ledger design (git-native code capture):
- **`D` is a manifest of pointers, not stored content.** Per step, metis persists
  `(path, git-blob-hash, commit)` for each closure file. **git's blob-hash *is* the
  content-hash** (drop the separate `content_hash(D)` hash function — use git's), and git's
  `(commit, path)` is the content location. metis stores no code bytes.
- **Capture:** on a **miss**, use the trace to find the closure files; if any are
  dirty/untracked, **commit just those to a side ref** (`refs/metis/sweeps/*`) — so `main` stays
  clean and the run has a real code SHA. On a **hit**, the code is unchanged → its commit-SHA is
  read back from the cache entry (no new commit, no search).
- **The CAS is a wipeable `content-hash → bytes` map for large *outputs* only.** Code+config are
  **not** CAS bytes — they're git blobs + a pointer-manifest in the durable records. So
  `rm -rf cas/` loses only recomputable output bytes; code + provenance are untouched.
- The output key's code-identity term = a hash of the closure's `(path, git-blob-hash)` pairs
  (git-derived), preserving per-step precision; the cache-hit check = re-hash (git `hash-object`)
  vs the manifest, or `git diff` the closure vs the SHA.

## Done when

- (design-stage — **met**) A design note settles cache-key composition, where cached outputs live +
  are keyed, code-version invalidation, the external-fetch policy, and the interaction with run
  provenance. (settled 2026-07-03, see `## Design`.)
- (implementation) A pure `pkg/cache` computes `K_pre` (from the #3 record's key-material) and
  `Validate`s a stored read-set `D` by re-hashing, deciding HIT/MISS — unit-tested for determinism,
  sensitivity, and each hit/miss path.
- (implementation) A Python read-sensor records each step's `D` (`sys.addaudithook`; first-party
  reads only), and the Go side turns it into `(path, git-blob-hash)` pairs.
- (implementation) The runner **skips** a HIT step (materializing its output from the CAS #9) and on
  a MISS records `D` + stores the output; the v1 leaf policy caches an immutable external fetch.
- (implementation, the payoff) An **e2e proves cheap sweeps**: a re-run HITs every step (no
  subprocess); changing one downstream knob HITs the upstream (`get-data`/`adapt`) and re-runs only
  the changed step + downstream. Atlas updated (validating-trace flow + the sensor's lower-bound
  limit + the #2/#7/#8 scope line).
- Scope honored: the side-ref **commit** of dirty closure files is **#8** (#2 hashes via
  `git hash-object`, needs no commit for cache correctness); the airtight syscall sensor is a later
  swap (unchanged `D`).

## Plan

Durable impl plan: `workshop/plans/000002-step-caching-plan.md` (mechanism recap, scope line vs.
#7/#8, 3 review boundaries). TDD; the risky novel sensor (M2) is de-risked as its own boundary.

- [x] Design note: cache-key composition + invalidation + storage + external-fetch policy + interaction with #3. **(settled 2026-07-03 — see `## Design`)**; impl decomposed into the durable plan (2026-07-05).
- [x] **M1 — pure cache core** (`pkg/cache`, Go). First **extract `record.CanonicalHash`** (no shared helper exists; refactor `PointAddress`/`OutputHash` onto it — ARCH-DRY, plan-judge). `Kpre(rec record.StepRecord, seed int)` = `CanonicalHash{step-id, **uses**, resolved-with, seed, upstream-output-hashes}` — **seed is an explicit arg** (lives on `RunRecord`, not the step); **uses included** (guards the wrong-step-type false-HIT). `Validate(storedD, hasher)` (re-hash D → HIT/MISS, pure over an injected hasher), `OutputKey(kpre, D)`, cache-index codec (`K_pre → {D, output_key}`). Unit tests: K_pre determinism + sensitivity (each of the 5 terms moves it), Validate hit/miss (clean / one-changed / missing), OutputKey, index round-trip.
- [x] **M2 — the read-sensor (Python) + blob-hasher (Go).** `sys.addaudithook` sensor module the step wrappers import → `reads.json` (first-party reads only; site-packages → `uv.lock` digest; drop stdlib/writes/own-output). Go `gitBlobHash(path)` (`git hash-object`) turns reads.json → `D = [(path, blob-hash)]`. Tests: sensor records a known read (uv-gated); D-builder maps reads.json → D (fake-hasher unit + real `git hash-object` skip-guarded); filtering excludes site-packages/stdlib.
- [x] **M3 — runner integration + leaf policy + sweep-reuse e2e.** **First populate `StepRecord.Upstream`** in `buildRecord` (map each step's `needs` → upstream output-hashes — the DAG-wiring K_pre depends on; #3 left the slot empty; plan-judge caught this was unlisted). Runner: per step compute `cache.Kpre(stepRecord, run.Seed)` → look up index → `Validate`; **HIT** materialize outputs from CAS (#9) + skip subprocess; **MISS** run + build D + `cas.Put` outputs + write index + record `Code.D`. Leaf policy (`cache: {leaf: immutable}` fetch-once). e2e: re-run HITs all steps; one-knob-change HITs upstream + re-runs only downstream (the "cheap sweeps" proof). Atlas.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.5 impl=0.4
item: greenfield-go-module   design=0.3 impl=0.4
item: smaller-go-module      design=0.2 impl=0.4
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.05 impl=0.05
design-buffer: 0.15
total: 3.06
```

#2 is the largest v1 issue (cross-language: Go cache core + Python read-sensor + runner integration).
M1 pure Go core (greenfield); M2 the novel sensor + blob-hasher (greenfield, cross-language — its own
boundary to de-risk); M3 the runner integration (smaller-go-module extend + 2 e2es); three
`milestone-review` items (3 boundaries). Design pre-settled → design at/near the floor. Impl at
40%-of-v2 (v3.1); +15% thorough-plan buffer.

## Log

### 2026-07-02
- Filed design-stage from the kbench#1 discussion. Operator: "we shouldn't hurry; caching needs to be uniform and consistent… a well-designed cache greatly improves iteration efficiency by tinkering a small portion and letting the system resolve what to recompute." Cluster with metis#3 + the fork-per-experiment convention.

### 2026-07-03
- **Design settled** (5-round brainstorm with the operator). Cache = a single **validating trace**: ex-ante `K_pre` + recorded read-set `D`, output at `hash(K_pre, content_hash(D))`; validate by **re-hashing `D`, not re-running**; code-version invalidation falls out for free (dropped the git-SHA / static-import-closure idea). Central invariant: *"a step's input-set changes only if a recorded input changes."* Two input classes (keyed immutable artifacts vs. traced source files) + one quarantine (network/env/clock at explicit ingest boundaries). Key subtleties surfaced: downstream must reference upstream by **output-hash**, not cache-key; the key-chain is only as sound as its leaves, so ingest leaves need pin (immutable bet) or etag (version-in-key); folder-scoped git-log was rejected as *unsound* (shared imports leak the boundary). Full spec now in `## Design`. Design-stage done-when met; next is splitting impl milestones.

### 2026-07-05
- 2026-07-05: closed — metis#2 validating-trace cache COMPLETE (M1 pure core + M2 sensor + M3 integration; all boundaries reviewed). go build+vet+test ./... green incl -race. THE CACHE WORKS end-to-end: TestCache_CheapSweeps (test/echo — identical re-run HITs all steps, one-knob change HITs shared upstream + re-runs only downstream) + TestCache_ToyPipelineHitsOnRerun (real uv pipeline — re-run HITs all 3 steps, reproduces cv_score from cache, materializes parquet). K_pre 5-term (fixes the 3 plan-judge false-HIT bugs); D via sensor + git-blob-hash, uv.lock folded in (dep-upgrade invalidation); leaf policy. First pkg/cas consumer. BYPASS --no-verdict: M3 is the FINAL milestone, its boundary review IS this issue-close integration review (redundant second pass avoided, #69). --no-project: brain metis-v1.md ticked by hand (est 3.0/actual 1.57).; review verdict: FIX-THEN-SHIP
- 2026-07-05: closed M2 — M2 read-sensor + blob-hasher: go build+vet+test ./... green (full suite; toy pipeline still runs through the sensor-wired wrappers). metis/trace.py sensor verified capturing the first-party code closure (metis/io.py, model.py, steps/train.py, ...) + used_site_packages via a uv-gated Go test. gitBlobHashes matches real `git hash-object` (real-git test). buildD + loadReadSet unit tests (absent reads.json = empty read-set, the safe direction). atlas pkg/cache entry updated (M2 shipped). BYPASS --no-project: issue-level tracking (milestone progress in the issue Plan/Log); the brain project tracker + final detail land at M3/issue-close.; review verdict: FIX-THEN-SHIP
- 2026-07-05: closed M1 — M1 pure cache core: go build+vet+test ./... green. pkg/record CanonicalHash extracted (PointAddress/OutputHash refactored, existing tests green). pkg/cache: Kpre (5-term sensitivity + upstream-order-invariance + non-finite-error, fixes the 3 plan-judge K_pre bugs) + Validate (hit/miss/vanished/vacuous) + OutputKey (composition + D-order-invariant) + Entry codec round-trip — 6 tests + CanonicalHash test. BYPASS --no-atlas + --no-project: M1 is the pure core; atlas (validating-trace flow) + project-tracker land at M3/final-close per the plan; milestone progress tracked in the issue.; review verdict: FIX-THEN-SHIP
- **Impl decomposed** into `workshop/plans/000002-step-caching-plan.md` (3 review boundaries; scope line #2-vs-#7/#8). `change-code` plan-quality **caught 3 load-bearing K_pre bugs** (all would silently produce an unsound cache): `seed` is on `RunRecord` not `StepRecord` (→ `Kpre(rec, seed)` explicit arg); `StepRecord.Upstream` is an empty #3 slot #2 must populate (M3 wiring); `uses` was dropped from K_pre (→ wrong-step-type false-HIT — folded in). Re-plan verified CLEAN; estimate INFO. Operator chose "build #2 fully now."
- **M1 built — the pure cache core** (TDD, all green; build+vet+full-suite clean). Extracted `record.CanonicalHash(any) (Hash, error)` (ARCH-DRY, plan-judge — refactored `PointAddress`/`OutputHash` onto it; `OutputHash` keeps its total signature via an internal unreachable-panic). New `pkg/cache`: `Kpre(rec, seed)` = `CanonicalHash{step-id, uses, resolved-with, seed, sorted-upstream}` (5-term sensitivity + upstream-order-invariance + non-finite-error tests), `Validate(D, hasher)` (re-hash → HIT/MISS; vanished/changed = MISS; empty-D = vacuous HIT), `OutputKey(kpre, D)` = `hash(K_pre, hash(D))` (D-order invariant), `Entry` index codec + round-trip. Next: M2 (Python read-sensor + git blob-hasher).
- **M2 built — the read-sensor (Python) + blob-hasher (Go)** (TDD, all green incl. the uv-gated sensor test + real-git hash test; full suite green — the toy pipeline still runs through the sensor). `metis/trace.py`: a `python -m metis.trace <step-module>` launcher installing `sys.addaudithook` (records Python `open`s) + a `sys.modules` snapshot at exit (the first-party **code closure**, robust to import caching) → writes `reads.json` (`{project_root, reads, used_site_packages}`) to the step dir; filters to project-root first-party (excludes venv/site-packages/__pycache__/.git/run-dir). Step wrappers (`steps/metis/{cv-split,train,predict}`) now launch through it. Go (`cmd/metis/trace.go`): `loadReadSet` (absent = empty, safe), `gitBlobHashes` (batched `git hash-object` — one call, matches real git; git's blob-hash *is* the content-hash, no commit needed), `buildD` (reads → `D=[(path, blob-hash)]`, pure over an injected hasher). **Honest sensor limit** noted (audit-hook lower-bound; C-extension `fopen` bypasses → but those are class-1 data, not code). Verified: sensor captures metis/io.py, model.py, steps/train.py, … + `used_site_packages`. Next: M3 (runner skip/materialize + leaf policy + cheap-sweeps e2e).
- **M3 built — runner integration + cheap sweeps** (TDD, all green incl. -race; the CACHE WORKS end-to-end). `buildRecord` now populates `StepRecord.Upstream` (needs → sorted upstream output-hashes — the K_pre DAG-wiring). `cmd/metis/caching.go` `cachingExecutor` decorates the step executor: per step compute `cache.Kpre` (config+seed+accumulated upstream hashes), look up `.metis-cache/index/<K_pre>.json`, **HIT** (stored D re-hashes clean via `gitBlobHashes`) → materialize the output manifest (metrics+artifacts) from a `pkg/cas` FSStore (**first CAS consumer!**) + skip the subprocess; **MISS** → run + `cas.Put` outputs + write index. **`uv.lock` folded into D** when site-packages used (dep upgrade → MISS; consumes M2's `used_site_packages`, closes the false-HIT gap). Leaf policy `with:{cache:{leaf:immutable}}` (K_pre-match HIT). `metis run --cache` (default on). **`OutputKey` dropped** (redundant with the Entry — can't compute pre-run). **e2es:** `TestCache_CheapSweeps` (test/echo: identical re-run HITs all; one-knob-change HITs upstream + re-runs only downstream) + `TestCache_ToyPipelineHitsOnRerun` (real uv pipeline: re-run HITs all 3 steps, reproduces cv_score from cache, materializes parquet). Atlas updated. **Deferred to #8:** the #3 record's `Code.D`/`Deps` *provenance* population (entangled with HIT/MISS D-tracking + the git-side-ref durability #8 owns; the cache's functional `Entry.D` is fully populated here).
- **Close-review FIX-THEN-SHIP applied** (0 Critical, 2 Important): (1) **`reads.json` no longer folds into the output-hash** — `collectArtifacts` now excludes it (top-level, alongside with.json/metrics.json). It was leaking into `record.OutputHash` → downstream K_pre, so an upstream *code* edit (read-set shift, byte-identical output) busted all downstream, and its absolute `project_root` defeated cross-machine reuse — both eroded "recompute only what changed." Test asserts `reads.json ∉ artifacts`. (2) **`.metis-cache/.gitignore`** written on first `--cache` use (ignores the whole wipeable cache) so CAS output blobs never get `git add`-ed in a real experiment repo. Also: runner-level immutable-leaf-bypasses-D test (`TestCachingExecutor_ImmutableLeafBypassesDValidation`); `recordMiss`/`isHit` now share `c.projectRoot` (root-consistency); dropped stale `OutputKey` from the pkg/cache doc. Full suite + -race green.
