# metis atlas — index

metis is the **platform-independent ML workbench** — the base layer of the
`kaggle-ml-base-layer` stack (`kbench → kaggle → metis → ariadne`). It owns the
reproducible unit of ML work (the **experiment**) and, as they land, the step-runner
and the Dataset/Split/step-type data plane. "Platform-independent" test: *would this be
identical on a non-Kaggle platform?* — if yes, it lives here.

- [experiment datatype](experiment.md) — the reproducible pipeline noun: the CUE schema
  (`#Experiment`/`#Step`/`#Status`/`#Run`), the `xx-datatype` authoring prototype, the
  `vocabulary validate-instance` structural validator, the enforcement merge-check (M1), the
  Go step-runner (M2), and the Python data plane — Dataset/Schema/Split + `cv-split`/`train`/
  `predict` step-types run hermetically via uv (M3).
- **`pkg/record`** (the L0 provenance record) — the unified per-step record (metis#3), the
  reproducibility atom the v1 cache/ledger chain keys off. Pure leaf over `pkg/cas`: `RunRecord`/
  `StepRecord` (emitted as `runs/<id>/record.json`, CUE-drift-guarded), `PointAddress` (the L0
  run-identity: config+repo-SHAs+seed content-address), `OutputHash` (multi-file output reduction).
  `Runner.Run` returns per-step `[]StepRun` so `cmd/metis` can assemble the record (git provenance
  via an injected `gitProbe`) and write `record.json` (the experiment `.md` is immutable input, #13 —
  no `## Runs` write-back). Scope
  line: #3 owns the record + point-address; the trace/cache-key are #2, side-ref code capture #7/#8.
  See [experiment.md](experiment.md). [metis#3]
- **`pkg/ledger`** (the shape-run ledger) — metis#8, the L1 tracking layer: a pure append-only,
  **point-address-deduped** table (`Row` = free-param tuple / sweep-SHA / point-address / namespaced
  metrics / status) with a **ragged** CSV codec (union columns, blank where absent), objective-driven
  `Best`/`TopN`, and `Filter`. It is an *aggregation view* over #3's per-run records, not a second run
  store. The driver (`cmd/metis/ledger.go`): after a sweep, `rowsFromManifest` (pure) turns #7's
  manifest + the per-point `record.json`s (namespaced per-step metrics — the collision fix) into rows,
  appended to `<shape>.ledger.csv` (idempotent); the shape `.md` is immutable input (#13) — the
  human top-N view is on-demand `metis ledger show`, not a summary written into the body.
  **`metis ledger show <shape> [--sweep|--sort|--top]`** renders sorted/filtered views. **`metis
  promote <shape> (--best|--point 'k=v') --name X`** reconstructs the winning point as an all-singleton
  experiment (pure `promotedExperiment` — re-expands the shape + matches by free-params, reusing
  `shapePointToExperiment`; id = the name) with a `promoted_from` back-link, committed at its code SHA
  (warns if dirty). Round-trip: the promoted experiment re-runs + reproduces the row. Immutability is by
  per-row snapshot (each row is self-contained, so a shape-space edit can't invalidate old rows).
  **The side-ref dirty-code capture** (`cmd/metis/capture.go`): the shared `captureRunCode` collects a
  run's code closure (`git hash-object -w`s each file) and, if any is dirty/untracked, commits it to a
  side ref (parented on HEAD, GC-protected) — a real code SHA even for a dirty run — then backfills the
  record's `CodeManifest.D` (the `(path, blob-hash)` pointer-manifest) + `Commit`. **Two capture hooks
  (metis#14):** (a) *code a step runs* → the multi-root read-set trace (metis#11 — spans every repo,
  metis + a consumer; **metis#15**: captures the traced module's OWN file explicitly since runpy runs
  it as `__main__`, and keeps only `.py` + `uv.lock` — data like `.parquet`/`schema.json` is class-1,
  never in `D`); (b) *the run-spec `.md` itself* → `git hash-object`'d explicitly (the trace never
  sees it — the Go runner parses it). It runs for **single runs** (`refs/metis/runs/<run-id>`, from
  `runResolvedExperiment`) and **sweeps** (`refs/metis/sweeps/<shape-run-id>`, once per shape-run — the
  `runOpts.inSweep` guard suppresses redundant per-point capture). **Loud** (metis#14): a
  `CodeManifest.CaptureStatus` (`captured`|`degraded`|`none`) + a stderr note whenever a run couldn't
  durably capture — reproducibility gaps are visible, never silent. Recovery = `git checkout <commit>`
  / `git cat-file blob <hash>`. metis stores no code bytes (git owns code); the CAS holds only wipeable
  output bytes. (`CodeManifest.Deps`/uv.lock-digest is a post-v1 follow-up; per-repo record `Commit`
  is single-valued — the D is fully repo-qualified.) [metis#8/#11/#14]
- **`pkg/sampler`** (the Sampler fold node) — metis#18, the pure ask/tell resample/sweep construct
  superseding metis#7's `pkg/sweep`: `Sampler[S,P,O,R]` (`Init`/`Ask`/`Tell`/`Done`) + a generic `Run`
  loop, instantiated nested (driver ⊃ sweeper ⊃ resample). Static Samplers: `GridConfigs` (the sweeper —
  every `shape.Expand` config at once), `FixedKFolds` (the inner resample — k folds over the materialized
  partition), `SingleDriver` (the degenerate outer driver:single). `Aggregate` → `MeanSE` (the honest
  per-config `(mean,SE)`); `Winner` (reconstructable run-keys). The **driver** is `cmd/metis`: `metis run`
  on an experiment-shape drives the nested loop (`runShapeSweep`: `Run(GridConfigs) ⊃ Run(FixedKFolds)`),
  running each `(config, fold)` through the shared `runResolvedExperiment` (cached runner) keyed by its
  content-address. Each fold builds a per-fold experiment (`data ++ engine-synthesized cv-split ++
  pipeline`, config + `_fold` overlaid so `Kpre` is fold-distinct); a **failing fold is FATAL** (a partial
  resample is not an honest estimate — unlike a v1 flat point). A **shape-run manifest**
  (`sweeps/<id>/manifest.json`) groups the runs; the ledger sidecar keeps the **raw per-fold rows** and
  `ledger.AggregateView` reduces them read-time → the per-config `(mean,SE)` leaderboard (metis#8 handoff).
  `--dry-run` lists configs. **Detect-and-abort** on mid-sweep HEAD-sha drift (not the dirty flag — the
  sweep's own outputs dirty the tree). Proven by `shapesweep_test.go` (fake exec on the real cache):
  nested loop → winner + N×k ledger + per-config `(mean,SE)`, fold-distinct cache, warm-HIT, incremental
  recompute, dry-run, fatal-fold, abort-on-drift. (Full `pkg/sampler` surface + the input-addressed cache
  land with metis#24/M1a-5.) [metis#18]
- **`pkg/shape`** (the experiment-shape lift) — metis#6, the pure config-space algebra over v0's
  untyped `with` bag. `Expand(steps, rangeSteps) → []Point` collapses a shape's reserved `$`-key
  descriptors (`$any` — one choice primitive dispatching on shape: list=untagged set / map=tagged bundled labeled-sum, both recursive + ADD / `$linear-range`·`$log-range`
  grid) into concrete v0-shaped points, each carrying its **free-param path** (the swept coordinates →
  #8 ledger key / #3 point-address). `experiment.Shape` (metis#18 v2) parses `type: experiment-shape`
  into three phases (`data│pipeline│ship`) + a `sweeper` (config-level Sampler) + a `driver` (outer
  Sampler); CUE `#ExperimentShape` is closed, with `#Experiment` = the flat singleton (no sweeper/driver)
  — the shared identity header single-sourced via `_meta`, the per-phase step-list via `_phase`; the
  `construct/datatype/experiment-shape.md` prototype. (Full three-phase / Sampler-fold-node write-up at
  metis#18 M1a-5.) `metis run` on a
  shape expands it — an all-singleton shape runs like a v0 experiment; a multi-point shape points to the
  **sweep driver (metis#7)**. The ledger keyed off the free-param path is **metis#8**. [metis#6]
- **`pkg/cache`** (the validating-trace policy layer) — metis#2, the step cache over `pkg/cas`
  (bytes) + `pkg/record` (key-material). Pure core shipped M1: `Kpre(rec, seed)` (ex-ante key =
  hash of step-id + uses + resolved-with + seed + sorted-upstream), `Validate(D, hasher)` (re-hash
  the read-set → HIT/MISS), `OutputKey(kpre, D)`, the `Entry` index codec. **M2 shipped** the
  read-sensor + blob-hasher: `metis/trace.py` (a `python -m metis.trace <step>` launcher installing a
  `sys.addaudithook` + `sys.modules` snapshot → writes the first-party code closure to
  `runs/<id>/<step>/reads.json`; the step wrappers launch through it), and Go `loadReadSet` /
  `gitBlobHashes` (batched `git hash-object`) / `buildD` turning reads → `D`. **Multi-root (metis#11):**
  the sensor discovers each read's git repo root (`_repo_root` walk-up for a `.git` marker — dir OR
  file, minus stdlib/site-packages/.venv), so a **consumer repo's** code (kbench importing metis)
  enters `D` alongside metis's. `reads.json` v2 = `{roots: {<repo-root>: [rel-paths]}, used_site_packages}`;
  `D = [(repo, path, git-blob-hash)]` is **repo-qualified**, hashed + re-validated per-repo (store side
  `recordMiss` and validate side `isHit` group by repo identically — a false-HIT/MISS pair otherwise).
  `loadReadSet` rejects a legacy v1 `reads.json` LOUD rather than let it parse to an empty `D` → a
  vacuous K_pre-only false HIT.
  Honest limit: the audit hook is a *lower-bound* (a C-extension `fopen` bypasses it), but those are
  class-1 data reads (keyed via upstream output-hashes), not first-party code. **M3 shipped** the
  runner integration: `cachingExecutor` (cmd/metis) decorates the step executor — per step it computes
  `K_pre` (from config + seed + upstream output-hashes accumulated in topo order), looks up
  `.metis-cache/index/<K_pre>.json`, and on a HIT (stored `D` re-hashes clean via `git hash-object`;
  `uv.lock` folded into `D` so a dep upgrade invalidates) **materializes** the output manifest
  (metrics + artifacts) from the CAS and **skips the subprocess**; a MISS runs, stores the output +
  writes the index entry. `metis run --cache` (default on). The **leaf policy**
  (`with: {cache: {leaf: immutable}}`) HITs on the K_pre match alone (pinned external fetch). Proven
  by two e2es: identical re-run HITs every step; a one-knob change HITs the shared upstream + re-runs
  only downstream ("cheap sweeps"). (The #3 record's `Code.D` provenance population is deferred to #8
  with the git-side-ref durability.) `record.CanonicalHash` is the shared hashing primitive. [metis#2]
- **`pkg/cas`** (content-addressed blob store) — the storage floor of the metis-v1 cache
  chain (**CAS ‹ #3 record ‹ #2 cache**). Mechanism only: `Store` (`Put(data)→Hash` /
  `Get` integrity-verified / `Has`), sha256 keys, self-deduplicating, sharded FS pool
  `cas/<h[:2]>/<h>` with atomic temp+rename writes and injected-clock LRU eviction (`maxBytes`).
  A **pure wipeable cache** in v1 — `rm -rf cas/` is always safe (a wiped blob recomputes; git
  owns code/config, external data refetches; the CAS holds only large *output* bytes). Swappable
  interface: `MemStore` is the in-memory fake for #2's tests, S3 slots in later (out of v1). No
  cache-keying/provenance/durable-retention here (→ #2/#3/#8). [metis#9]
- [workflow/](workflow) — inherited ariadne workflow docs (symlink into the substrate).

**Roadmap (metis#1 — all milestones shipped):** M1 (experiment datatype), M2 (Go step-runner:
`cmd/metis run` + pure `pkg/experiment` `Parse`/`Validate`/`TopoSort`, semantics enforced on
read, steps run as subprocesses over a `steps/<layer>/<steptype>` + files contract), and M3
(Python data plane: pure `metis/` core + thin `metis/io` contract + `cv-split`/`train`/`predict`
entrypoints + uv env; `metis run` walks a toy pipeline to a real CV score). The end-to-end
Kaggle proof is kbench's Titanic walking skeleton (kbench#1), which builds on this base.
