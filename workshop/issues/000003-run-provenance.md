---
id: 000003
status: working
deps: []
github_issue:
created: 2026-07-02
updated: 2026-07-05
estimate_hours: 1.9
started: 2026-07-05T13:37:57-07:00
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

## Spec

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

### Revision (2026-07-05): the `code` field is a git-pointer manifest

Refined by the #8 ledger/durability discussion:
- The record's **`code`** key-material is a **manifest of `(path, git-blob-hash, commit)`
  pointers** (not `content-hash` bytes): **git's blob-hash is the content-hash**, git's
  `(commit, path)` is the location. metis stores no code bytes — git does.
- **Capture:** a sweep commits its traced code closure to a side ref (`refs/metis/sweeps/*`) on a
  miss if dirty, so every run has a real code SHA. This makes the git-SHA point-address **always
  valid** (every sweep is committed) — collapsing the clean-vs-dirty nuance: recovery of a past
  (even "dirty") version = `git checkout <its sweep-SHA>`.
- **Human run-address = (sweep short-SHA, free-param tuple)** — a git short-SHA (`a1b2c3d`) is the
  eyeballable handle; metis invents no id.
- **Durability by construction:** the pointer-manifest lives in the durable records, the blobs in
  git (side-ref, GC-protected). The CAS holds only wipeable large-output bytes; wiping it loses no
  code or provenance.

## Done when

- (design-stage — **met**) A design note settles what gets snapshotted (config shape, git
  binding), where it lives, how `## Runs` surfaces knob→score, and the relationship to the
  fork-per-experiment convention + caching keys. (settled 2026-07-03, see `## Design`.)
- (implementation) A pure `pkg/record` defines the unified `StepRecord`/`RunRecord` (all
  key-material + provenance-extra fields) with a `#RunRecord` CUE type + drift guard, and mints
  the **point-address** (`hash(resolved-with across the DAG, repo-SHAs, seed)`) — deterministic
  and map-order-independent, unit-tested for stability + sensitivity.
- (implementation) A run assembles + writes `runs/<id>/record.json` (git-trackable) with repo-SHAs,
  per-step resolved-with, output-hashes (`cas.HashOf`), metrics, and the minted point-address; two
  identical runs mint the **same** point-address (e2e).
- (implementation) `## Runs` renders a **knob→score** line from the record (config beside metric),
  superseding the freeform summary.
- Scope line honored: the traced read-set `D` + cache key + skip/recompute are **#2**; side-ref code
  *capture* is **#7/#8** — #3 records the current commit + dirty flag honestly. Atlas updated.

## Plan

Durable impl plan: `workshop/plans/000003-run-provenance-plan.md` (scope line #3-vs-#2-vs-#7/#8;
2 review boundaries). TDD; the pure core (M1) is reviewed before IO builds on it (M2).

- [x] Design note: provenance record shape + git binding + ## Runs legibility + relation to #2 and fork-per-experiment. **(settled 2026-07-03 — see `## Design`)**; impl decomposed into the durable plan (2026-07-05).
- [x] **M1 — pure record core** (+ pure-core runner change, reviewed at this boundary). `pkg/experiment`: add `StepRun{Step, Result}`; `Runner.Run` → `(Run, []StepRun, error)` so per-step metrics/artifacts are retained (today they're flat-merged + discarded — the record can't reach them otherwise). `pkg/record`: `StepRecord`/`RunRecord`/`CodeManifest`/`FileHash` (all key-material + provenance slots; `D`/`Deps` are #2-populated slots), pure `PointAddress(resolvedWith, repoSHAs, seed)` (canonical `json.Marshal` → `cas.HashOf`), pure `OutputHash([]FileHash)` (**multi-file reduction**: hash of the sorted `(relpath, content-hash)` manifest). `#StepRecord`/`#RunRecord` CUE + drift guard. Unit tests: point-address determinism/sensitivity, `OutputHash` order-independence/sensitivity/empty, per-step retention, JSON round-trip. (No `CacheKeyMaterial` — deferred to #2 where it's testable.)
- [x] **M2 — populate + persist + legibility.** `cmd/metis/record.go`: `assembleRecord`/`buildRecord` assemble the record during `runExperiment` from `exp.Steps` + `[]StepRun` (repo-SHAs+dirty via an injected `gitProbe` seam — real `gitCLI` + fake; per-step metrics + `OutputHash` from `cas.HashOf`-ing each artifact file), mint the point-address, write `runs/<id>/record.json` (git-trackable). `## Runs` → knob→score line via `recordSummary`, reusing an extracted `formatMetrics` (ARCH-DRY). **Graceful degradation:** a run outside a git repo warns + records no repo-SHAs (design's "v1: warn"), doesn't fail. e2e (`record.json` conforms to `#RunRecord`; point-address stable across two identical runs; `## Runs` knob→score line; no-git degraded path) — no `uv` needed (test/echo). Atlas: `pkg/record` + record datatype + `Runner.Run` `[]StepRun` + the #3/#2 scope line. **Verified in the real CLI** (real git SHA + dirty; knob→score line).

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.5 impl=0.4
item: smaller-go-module      design=0.2 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.05 impl=0.05
design-buffer: 0.15
total: 1.91
```

Design pre-settled → greenfield `pkg/record` (M1: point-address + `OutputHash` multi-file reducer +
the pure-core `Runner.Run` per-step-retention change + CUE + drift guard) at the design floor, impl
bumped to 0.4 for the extra pure-core work the plan-judge surfaced; the runner integration (M2) is a
smaller-go-module *extend* of `cmd/metis`; **two** `milestone-review` items — one per review boundary
(M1 milestone-close + M2 final close, each auto-dispatches a fresh-eyes review; #9 showed reviews are
real work, and the estimate-quality judge flagged the second boundary as un-costed). A small atlas note.
Impl at 40%-of-v2 (v3.1). +15% thorough-plan buffer.

## Log

### 2026-07-02
- Filed design-stage from the kbench#1 discussion. Operator derived the fork-per-experiment / frontmatter-immutability convention independently and asked to design provenance well before building. Cluster with metis#2 (caching).

### 2026-07-03
- **Design settled** (unified with #2's caching design, multi-round brainstorm). Key move: provenance and the cache key are the *same determinant set*, different operation (reconstruct vs. compare) → **one unified per-step record**, key-material (hashed = cache key) vs. provenance-only extras (repo-SHAs, fetched, metrics). Three derived views: cache key (#2) / point-address (this issue, #8 derives) / output key. Durability contract: CAS is a pure wipeable cache; records must be **recipe-complete against durable homes** (git + refetch) so the DAG reconstructs from an empty cache; clean-vs-dirty demoted to a *legibility* choice (promotable runs → clean). Dropped the stale "point-address is the cache key" self-label. Split the CAS storage primitive out as **metis#9**. Full spec in `## Design`.

### 2026-07-05
- 2026-07-05: closed M2 — M2 runner integration: go build+vet+test ./... green incl -race. Hermetic e2e (test/echo, no uv): record.json conforms to #RunRecord, point-address stable across 2 identical runs, ## Runs knob->score line, no-git degraded path. Verified in the REAL CLI: record.json carried the real HEAD sha + dirty flag + per-step output-hashes; ## Runs rendered knobs+metrics. Root-caused M1-review Minor (PointAddress returns error on non-finite config); graceful git degradation (no-repo run warns, not fails). --no-project: brain/data/project/metis-v1.md uses issue-link convention, not the close <a id> anchors; ticked metis#3 + est/actual by hand (committed in brain).
- 2026-07-05: closed M1 — M1 pure core: go build+vet+test ./... green. pkg/experiment Runner.Run -> (Run, []StepRun, error) with per-step-retention test; pkg/record 6 unit tests (PointAddress determinism-across-25-calls + sensitivity; OutputHash order-independence/sensitivity/nil==empty; JSON round-trip) + TestRunRecordConformsToCUE drift guard (cue present, passes). BYPASS --no-atlas + --no-project: M1 is the pure core — the atlas + project-tracker updates are deliberately M2/final-close work (record datatype + #3/#2 scope line land with the runner integration); milestone progress is tracked in the issue Plan/Log.; review verdict: SHIP
- **Impl decomposed** into `workshop/plans/000003-run-provenance-plan.md` (2 review boundaries, scope line #3-vs-#2-vs-#7/#8). `change-code` plan-quality: **CLEAN** (after revising the plan for the 2 gaps below); estimate-quality: INFO (added the 2nd milestone-review item).
- **M1 built — the pure record core** (TDD, all green; build+vet+full-suite clean). `pkg/experiment`: `StepRun{Step, Result}` + `Runner.Run` → `(Run, []StepRun, error)` for **per-step retention** (plan-judge Finding 1: the flat `Run` merge discarded per-step data the record needs; callers/tests updated). `pkg/record` (new pure pkg): `StepRecord`/`RunRecord`/`CodeManifest`/`FileHash`/`CodeRef` (`Hash` re-exports `cas.Hash`), pure `PointAddress` (canonical `json.Marshal`→`cas.HashOf`; deterministic-across-calls + sensitivity tests), pure `OutputHash([]FileHash)` (**multi-file reducer** — plan-judge Finding 2 — sorted `(path,hash)` manifest; order-independence + sensitivity + nil==empty tests). CUE `#StepRecord`/`#RunRecord` in `experiment.cue` + `TestRunRecordConformsToCUE` drift guard (passes). **M1 boundary review: SHIP.**
- **M2 built — runner integration + `## Runs` legibility** (TDD, all green incl. `-race`; verified in the real CLI). `cmd/metis/record.go`: `gitProbe` seam (real `gitCLI` shelling `git -C`; fake for tests), `assembleRecord` (probe git → hash each step's artifacts via `cas.HashOf` → `record.OutputHash`) + pure `buildRecord` (mints point-address, fills coarse code identity, leaves Upstream/D/Deps for #2), `writeRecordJSON` (`runs/<id>/record.json`), `recordSummary`/`formatKnobs`/`formatMetrics` (knob→score `## Runs`, reuses one metric-formatter — ARCH-DRY). Wired into `runExperiment` (returns `[]StepRun` now). **Root-caused the M1-review Minor:** `PointAddress` returns an error (not panic) on non-finite config; **graceful git degradation** (run outside a repo warns + omits repo-SHAs, doesn't fail — design's "v1: warn"). Hermetic e2e (no `uv`): record conforms to `#RunRecord`, point-address stable across identical runs, knob→score line, no-git path. Atlas updated (`pkg/record` + `Runner.Run` + #3/#2 scope line). Real-CLI check: record.json carries a real git SHA + dirty flag + per-step output-hashes; `## Runs` = `knobs: prep.k=5 … — metrics: echoed=1`. **Noticed a pre-existing latent bug** (relative step-path fallback breaks exec in a `go.mod`-less repo) — out of scope for #3, to be filed.
