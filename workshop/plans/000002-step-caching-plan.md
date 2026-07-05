---
issue: 000002
title: Uniform DAG step caching — the validating trace (K_pre + D) over the CAS + record
status: active
created: 2026-07-05
---

# Plan: metis#2 — content-addressed step caching (the validating trace)

Design **settled** (issue `## Design` 2026-07-03 + 2026-07-05 Revision; pensive §Caching). Built
on the two merged deps: **#9 CAS** (`pkg/cas` — output-byte store) and **#3 record** (`pkg/record`
— per-step key-material + `CodeManifest{D, Deps}` slots left empty *for #2 to populate*). This plan
is the implementation decomposition + the scope line vs. #7/#8.

## The mechanism (recap)

Two keys per step, hashed late:
- **`K_pre`** = `hash(step-id, uses, resolved-with, seed, upstream-output-hashes)` — cheap, ex-ante
  (computable *before* running). **Five terms, and their sources differ** (plan-judge, verified vs. the
  merged #3 structs): `step-id`/`uses`/`resolved-with` are on `StepRecord`; **`seed` is on `RunRecord`,
  NOT the step** (so K_pre takes it as an explicit arg); **`Upstream` is an *empty slot* #3 left for #2
  to populate** (not present data — populating it is M3 work below). `uses` is included deliberately: two
  steps with the same id/with/seed/upstream but different `uses` would otherwise false-HIT and serve the
  wrong step-type's output — a reachable unsoundness the "code-version invalidation falls out free" line
  does NOT cover (a `uses` swap changes no file in `D`).
- **`D`** = the step's recorded **read-set**: `[(repo-relpath, blob-hash)]` — captured *after* a run
  by a read-sensor. Blob-hash = git's blob hash (`git hash-object`) per the Revision (git is the
  content store; drop a separate metis code-hash).

Lifecycle (validate ≠ execute): compute `K_pre` → look up the stored `D` for it → **no D** = cold,
run + record → **D exists** = re-hash its paths → all match = **HIT** (materialize the stored output
from CAS, no execution) / any differ = **MISS** (run, re-record D + output). Code-version
invalidation falls out free (an edited file is a path in D whose hash moved → miss).

## Scope line — #2 vs. #7/#8

- **#2 owns (this issue):** `K_pre` + `D` computation, the validate-vs-execute lifecycle, the cache
  index (`K_pre → {D, output-hash}`), output store/materialize via the CAS, the read-sensor, the
  runner skip-integration, and the v1 leaf policy (fetch-once immutable-leaf marking).
- **Deferred to #7/#8 (NOT here):** the **side-ref commit** of dirty closure files
  (`refs/metis/sweeps/*`) — that's the *durability/recovery* layer (so a dirty run is
  `git checkout`-able), a sweep-time op #8 owns. **#2 needs no commit for cache *correctness*:** D's
  blob-hashes are computed via `git hash-object` (a hash, no commit) and re-hashed to validate. A
  dirty file still hashes; the cache hits/misses correctly. #8 later adds the commit so the SHA is
  *recoverable*. (So #2 populates `CodeManifest.D` with `(path, blob-hash)`; #8 adds the `commit`.)
- **The read-sensor's honest limit** (design): `sys.addaudithook` sees `open`/`import` at the Python
  layer, but a C-extension `fopen` (some pandas/numpy paths) bypasses it → **lower-bound** D. v1
  accepts this (route heavy data through framework helpers so the read stays observable); the
  airtight version is a **sensor swap** (syscall trace), not a redesign — `D`'s definition is
  unchanged. **This is the one design choice worth an explicit operator nod** (below).

## Milestones (3 review boundaries)

### M1 — the pure cache core (`pkg/cache`, Go)

- **Extract `record.CanonicalHash(any) (Hash, error)` first** (ARCH-DRY, plan-judge): there is no
  shared helper today — `PointAddress`/`OutputHash` each inline `json.Marshal → cas.HashOf`. Create the
  helper in `pkg/record`, refactor those two callers onto it, and use it for K_pre/OutputKey too. (A
  small touch to the merged `pkg/record`; keeps the drift guard green.)
- `Kpre(rec record.StepRecord, seed int) (cache.Hash, error)` — `CanonicalHash` of `{step-id, uses,
  resolved-with, seed, upstream-output-hashes}`. **`seed` is an explicit arg** (it lives on `RunRecord`,
  not `StepRecord`); **`uses` is included** (guards the wrong-step-type false-HIT); `upstream` reads
  `rec.Upstream` (populated by M3 — the unit tests pass StepRecords with it filled).
- `Validate(storedD []record.CodeRef, hash func(path) (record.Hash, error)) (hit bool, err error)`
  — re-hash each path in D via the injected hasher, compare to the stored blob-hash; pure over the
  hasher seam (no filesystem in unit tests). Missing file / changed hash → miss.
- `OutputKey(kpre, storedD) cas.Hash` — `hash(K_pre, hash(D-manifest))`, the CAS address of the
  step's output for this (values × code) pair.
- The **cache index** type + codec: `K_pre → {D, output_key}` persisted as small git-trackable JSON
  (`cache/index/<K_pre>.json`), read/written by a thin IO layer (M3). Pure encode/decode here.
- Unit tests: K_pre determinism + sensitivity iterating **all 5 terms** (step-id, uses, resolved-with,
  seed, upstream — each moves the key; pins every false-HIT vector, esp. the `uses` one), Validate
  hit/miss (clean, one-file-changed, missing-file), OutputKey composition, index round-trip.
  `CanonicalHash` refactor keeps `OutputHash`'s total `Hash` signature (absorb the unreachable error
  internally — don't ripple a new error return out to `hashArtifacts`).
- **M1 review boundary.**

### M2 — the read-set sensor (Python) + `git hash-object` blob-hasher (Go)

- **Read-sensor** (`steps/_lib/metis_trace.py` or similar): a tiny module the step wrappers import
  that installs `sys.addaudithook`, records `open`/`import` file reads under the repo root
  (filtering: keep first-party code+config; collapse `site-packages` → the `uv.lock` digest; drop
  stdlib/system/temp/writes/own-output), and on exit writes `reads.json` (the raw read paths) into
  the step dir. The step wrappers `import` it at the top (one line each).
- **Blob-hasher** (Go, `cmd/metis`): `gitBlobHash(path) record.Hash` shelling `git hash-object`
  (the injected `gitProbe` seam extends here, or a sibling), + `uvLockDigest`. The Go side turns
  `reads.json` (paths) into `D = [(path, blob-hash)]` by hashing each kept path.
- Tests: the sensor records a known read in a fixture run (hermetic Python, uv-gated `t.Skip`); the
  Go D-builder maps reads.json → D with correct blob-hashes (a fake hasher unit test + a real
  `git hash-object` test skipped without git); filtering excludes site-packages/stdlib.
- **M2 review boundary.**

### M3 — runner integration: skip / materialize + leaf policy + sweep-reuse e2e

- **Populate `StepRecord.Upstream` (the #3 slot #2 fills)** — in `cmd/metis/buildRecord`, map each
  step's `needs` → the upstream steps' output-hashes (`assembleRecord` already computes `outputHashes`
  for every step at record.go:70–77 but never threads them per-step). **Sort the upstream hashes**
  before storing so K_pre is invariant to `needs`-declaration order (`[a,b]` vs `[b,a]` → same key —
  plan-judge). This is the DAG-wiring K_pre depends on; without it K_pre is upstream-blind. (Named
  explicitly per the plan-judge — it was hand-waved as "threaded through.")
- Runner: before executing each step, compute `K_pre` (`cache.Kpre(stepRecord, run.Seed)` with
  Upstream now populated), look up the cache index, `Validate`. **HIT** → materialize the step's
  outputs from the CAS into the step dir (so downstream reads them), skip the subprocess, mark the
  step cached. **MISS** → run, read `reads.json` → build D, `cas.Put` each output artifact, write
  the cache index entry (`K_pre → {D, output_key}`), record D into the `#3` record's `Code.D`.
- **Leaf policy:** a step marked `cache: {leaf: immutable}` (in `with` or a step attribute) caches
  its output and is not re-fetched (fetch-once-and-freeze); etag where trivial. External network is
  quarantined to such leaves.
- **e2e (the payoff):** two runs of the same experiment — the second HITs every step (no subprocess;
  assert via a run-count marker). And a *sweep-shaped* pair: change one downstream knob → upstream
  (`get-data`/`adapt`) HIT, only the changed step + downstream MISS. This is kbench#4's "cheap
  sweeps" guarantee, proven early.
- atlas: `pkg/cache` + the validating-trace flow + the sensor's lower-bound limit + the #2/#7/#8
  scope line.
- **M3 review boundary** (issue close).

## Open decisions (flag for plan-judge / operator)

1. **Read-sensor = `sys.addaudithook` (lower-bound), not syscall trace.** The design settles this
   for v1 (route data through framework helpers; sensor-swap is a later upgrade with unchanged D).
   **Worth an explicit operator nod** — it's the one place v1 accepts a known soundness gap (a
   C-extension read that bypasses the hook could serve a stale cache). Mitigation: the leaf policy +
   framework-helper convention keep reads observable for the Titanic pipeline. If the operator wants
   airtight v1, that's the syscall-trace sensor (bigger M2).
2. **Cache index home = `cache/index/<K_pre>.json` (git-trackable), separate from the run record.**
   The per-run record (`runs/<id>/record.json`) is per-run; the cache index is cross-run
   (`K_pre → last-miss D + output`). Small metadata → git, like the record. Alternative: fold into
   the CAS (put D-json, index K_pre → its hash) — more uniform but less legible. Chose the sidecar.
3. **D uses `git hash-object` blob-hashes (Revision), not `cas.HashOf`.** Aligns with git-as-content-
   store (recovery via checkout) and the `#3` `CodeRef.BlobHash` field. Costs a `git hash-object`
   per read file (cheap; a few small files). Self-contained for #2 (no side-ref commit — that's #8).

## Test strategy

Pure core (M1) → direct unit tests over injected hasher/index seams (no filesystem). Sensor (M2) →
hermetic uv-gated Python fixture + Go D-builder unit tests. Integration (M3) → the two-run HIT e2e +
the sweep-reuse e2e (the "cheap sweeps" proof), fixtures copied into `t.TempDir()`. Controllable
time via the existing injected `Clock`.

## Revisions

### 2026-07-05 — M1 built (FIX-THEN-SHIP; atlas + Validate signature)
- **`Validate` returns `bool`, not `(bool, error)`** (M1 review Minor): folding a hasher failure
  (vanished/unreadable D file) into MISS is *sounder* than surfacing it — a MISS only recomputes,
  never serves stale — so the M1 bullet's `(hit bool, err error)` signature is superseded by
  `Validate(storedD, hash) bool`. (M3 note: when the real git-blob hasher is wired, *log* a persistent
  IO error so a disk/permission fault on a D file doesn't silently degrade every run to a cold MISS.)
- **atlas `pkg/cache` stub added at M1** (review Important, §8): the *package/terminology* surface
  exists now even though the validating-trace *flow* is M3, so `atlas/index.md` gets the stub now;
  the full flow entry lands with M3's runner integration.
- **M3 must feed `Kpre` from the freshly-parsed experiment** (review arch-note), never a round-tripped
  `record.json`, so the `With` Go value types stay stable across runs (K_pre determinism).
