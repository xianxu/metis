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
- **`pkg/cache`** (the validating-trace policy layer) — metis#2, the step cache over `pkg/cas`
  (bytes) + `pkg/record` (key-material). Pure core shipped M1: `Kpre(rec, seed)` (ex-ante key =
  hash of step-id + uses + resolved-with + seed + sorted-upstream), `Validate(D, hasher)` (re-hash
  the read-set → HIT/MISS), `OutputKey(kpre, D)`, the `Entry` index codec. The read-sensor (Python
  audit hooks) + git blob-hasher (M2) and the runner skip/materialize integration + `## the cheap-
  sweeps flow` (M3) are still to come; `record.CanonicalHash` is the shared hashing primitive.
  [metis#2]
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
