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
  via an injected `gitProbe`), write `record.json`, and render the knob→score `## Runs` line. Scope
  line: #3 owns the record + point-address; the trace/cache-key are #2, side-ref code capture #7/#8.
  See [experiment.md](experiment.md). [metis#3]
- **`pkg/ledger`** (the shape-run ledger) — metis#8, the L1 tracking layer: a pure append-only,
  **point-address-deduped** table (`Row` = free-param tuple / sweep-SHA / point-address / namespaced
  metrics / status) with a **ragged** CSV codec (union columns, blank where absent), objective-driven
  `Best`/`TopN`, and `Filter`. It is an *aggregation view* over #3's per-run records, not a second run
  store. The driver (`cmd/metis/ledger.go`): after a sweep, `rowsFromManifest` (pure) turns #7's
  manifest + the per-point `record.json`s (namespaced per-step metrics — the collision fix) into rows,
  appended to `<shape>.ledger.csv` (idempotent) with the shape body's top-N summary regenerated.
  **`metis ledger show <shape> [--sweep|--sort|--top]`** renders sorted/filtered views. **`metis
  promote <shape> (--best|--point 'k=v') --name X`** reconstructs the winning point as an all-singleton
  experiment (pure `promotedExperiment` — re-expands the shape + matches by free-params, reusing
  `shapePointToExperiment`; id = the name) with a `promoted_from` back-link, committed at its code SHA
  (warns if dirty). Round-trip: the promoted experiment re-runs + reproduces the row. Immutability is by
  per-row snapshot (each row is self-contained, so a shape-space edit can't invalidate old rows). The
  side-ref durability capture is metis#8 M3. [metis#8]
- **`pkg/sweep`** (the sweep sampler) — metis#7, the pure ask/tell seam: `Sampler` (`Ask()`/`Tell()`),
  `Grid` (enumerates `shape.Expand`'s points in order; adaptive samplers slot in with no loop change),
  and `StopPredicate`s (`MaxPoints`, `TargetReached`, `AnyStop`). The **driver** is `cmd/metis`:
  `metis run` on a multi-point shape **sweeps** (via `runSweep`) — loops Ask → run each point through the
  shared `runResolvedExperiment` (cached runner) keyed by its `record.PointAddress` (so re-runs dedup +
  resume-from-cache is free) → Tell; **per-point failure is recorded + the sweep continues**; a
  **shape-run manifest** (`sweeps/<id>/manifest.json`) groups the N point-runs (the **metis#8** ledger
  handoff). Flags: `--max-points` (budget stop), `--dry-run` (list points). **Detect-and-abort**: the
  sweep freezes on the HEAD code sha (via `gitProbe`) and aborts on mid-sweep commit/branch drift (NOT
  the dirty flag — the sweep's own outputs dirty the tree; precise code-dirty detection is metis#10).
  Proven by e2e: N-runs+manifest, cache-reuse across points, failure-continues, max-points, dry-run,
  abort-on-drift, no-false-abort-on-dirty. [metis#7]
- **`pkg/shape`** (the experiment-shape lift) — metis#6, the pure config-space algebra over v0's
  untyped `with` bag. `Expand(steps, rangeSteps) → []Point` collapses a shape's reserved `$`-key
  descriptors (`$any` set / `$oneof` bundled labeled-sum that ADDs / `$linear-range`·`$log-range`
  grid) into concrete v0-shaped points, each carrying its **free-param path** (the swept coordinates →
  #8 ledger key / #3 point-address). `experiment.Shape`/`Sweep` parse `type: experiment-shape` + the
  `sweep:` block; CUE `#ExperimentShape` (with `#Experiment` = the singleton refinement, single-sourced
  via the shared `_pipeline`); the `construct/datatype/experiment-shape.md` prototype. `metis run` on a
  shape expands it — an all-singleton shape runs like a v0 experiment; a multi-point shape points to the
  **sweep driver (metis#7)**. The ledger keyed off the free-param path is **metis#8**. [metis#6]
- **`pkg/cache`** (the validating-trace policy layer) — metis#2, the step cache over `pkg/cas`
  (bytes) + `pkg/record` (key-material). Pure core shipped M1: `Kpre(rec, seed)` (ex-ante key =
  hash of step-id + uses + resolved-with + seed + sorted-upstream), `Validate(D, hasher)` (re-hash
  the read-set → HIT/MISS), `OutputKey(kpre, D)`, the `Entry` index codec. **M2 shipped** the
  read-sensor + blob-hasher: `metis/trace.py` (a `python -m metis.trace <step>` launcher installing a
  `sys.addaudithook` + `sys.modules` snapshot → writes the first-party code closure to
  `runs/<id>/<step>/reads.json`; the step wrappers launch through it), and Go `loadReadSet` /
  `gitBlobHashes` (batched `git hash-object`) / `buildD` turning reads → `D = [(path, git-blob-hash)]`.
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
