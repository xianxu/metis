---
issue: 000003
title: Run provenance — the unified per-step record (L0) + point-address + ## Runs legibility
status: active
created: 2026-07-05
---

# Plan: metis#3 — the unified per-step record (L0)

Design is **settled** (issue `## Design`, 2026-07-03 + 2026-07-05 Revision; pensive §Caching).
This plan is the *implementation decomposition* — what code delivers the settled design in v1,
and the scope line between #3 (owns the record + point-address) and #2 (indexes it with the trace).

## Scope — what #3 delivers vs. what #2 owns

The design assigns **one raw record, three consumers**: #3 *owns* it (writes provenance, mints the
point-address, renders knob→score), #2 *indexes* it (validating trace → cache key → skip/recompute),
#8 *derives* its ledger key from the point-address. The three derived views hash *late*:

| view | key-material | owner |
|---|---|---|
| **point-address** | `hash(resolved-with across the DAG, repo-SHAs, seed)` — **no per-step trace** | **#3 (this issue)** |
| cache key | `hash(step-id, resolved-with, seed, upstream-output-hashes, code{D, deps})` | #2 |
| output key | the CAS address of the step's output bytes | #3 computes (`cas.HashOf`), #2 stores |

**The point-address needs no read-set `D`** — it is the coarse repro/identity key (resolved-with +
repo-SHAs + seed). So #3 is self-contained: it delivers the record struct (with *all* fields defined,
including the `code`/`upstream` key-material slots), the **point-address** derivation, populates the
provenance-extras (repo-SHAs+dirty via git, output-hash via `cas.HashOf`, metrics, timestamp), writes
the record durably, and renders `## Runs` as a knob→score table.

**Deferred to #2 (NOT this issue):** the validating trace (recording the read-set `D`), the **cache
key** = `hash(key-material incl. D)`, and skip/recompute. #3 leaves `StepRecord.Code.D` empty and
populates only the coarse code identity (repo commit + dirty). #2 extends the record with `D`.

**Deferred to #7/#8 (NOT this issue):** the side-ref code *capture* (`git commit` the traced closure
to `refs/metis/sweeps/*` on a miss) is a sweep-time op — #3 records the *current* commit + dirty flag;
turning a dirty run into a nameable commit is #7/#8's capture mechanism. #3 records honestly (dirty=true).

`cas.HashOf` is a pure function from the merged #9 — #3 uses it for content hashing (the design's
"references CAS addresses, hashing ≠ storing"); it does **not** use `cas.Store` (no output bytes stored
here — that's #2's materialization).

## Milestones (2 review boundaries)

Datatype-defining work: get the pure core right and reviewed *before* building IO on it. **M1
deliberately reopens `pkg/experiment`** (the per-step-retention change below is pure core, so it is
reviewed at the M1 boundary — not smuggled into M2's "thin IO" — per the plan-judge).

### M1 — the pure record core (`pkg/record`) + per-step retention + CUE type + drift guard

- **Pure-core prerequisite (in `pkg/experiment`):** `Runner.Run` today flat-merges per-step
  metrics/artifacts into the flat `Run` and **discards the per-step `StepResult`s** — so the record's
  per-step data (metrics, output artifacts) is unreachable at the `cmd/metis` assembly site. Fix here,
  in the reviewed pure core: add `StepRun{Step Step; Result StepResult}` and change `Runner.Run` to
  return `(Run, []StepRun, error)` — the flat `Run` stays (ledger/back-compat); the ordered `[]StepRun`
  is the per-step breakdown #3 assembles from. (`pkg/experiment` still imports neither `cas` nor
  `record` — no cycle; `record` imports `experiment`.) Update the runner's existing tests + the one
  `cmd/metis` caller.
- New pure package `pkg/record` (imports `pkg/experiment` for Step/StepRun, `pkg/cas` for `HashOf`):
  - `StepRecord{StepID, Uses, With(resolved), Upstream []cas.Hash, Code CodeManifest, OutputHash cas.Hash, Metrics}`
  - `CodeManifest{Commit string, Dirty bool, D []CodeRef(path,blobHash), Deps string}` — `D` (read-set) + `Deps` are defined slots #2 populates; #3 fills only `Commit`/`Dirty`.
  - `RunRecord{RunID, Experiment, Seed, PointAddress cas.Hash, RepoSHAs map[string]string, Dirty bool, Steps []StepRecord, Started, Finished, Status}`
  - `PointAddress(resolvedWith map[stepID]map[string]any, repoSHAs map[string]string, seed) cas.Hash` — **pure**: canonical JSON (Go `json.Marshal` sorts map keys → deterministic) of `{resolved_with, repo_shas, seed}`, then `cas.HashOf`. Stable across map-iteration order (unit-tested); v1 uses `json.Marshal` as the canonical form (a stricter RFC-8785 canonicalizer can slot in later — noted).
  - `OutputHash(files []FileHash) cas.Hash` where `FileHash{Path string; Hash cas.Hash}` — **the multi-file reduction** (Finding 2): a step's output is a *set* of artifacts, so reduce to one address by hashing a **canonical sorted-by-path manifest of `(relpath, content-hash)`**. Stable across walk order; changes iff a path or its bytes change; empty set → a defined empty-manifest hash. Pure → unit-tested in M1; M2 wires `cas.HashOf(os.ReadFile(artifact))` into it. The point-address does **not** include output-hashes (it is config+repo+seed); `OutputHash` feeds the record's provenance-extra + #2's upstream key-material later.
  - (No `CacheKeyMaterial` here — it has no consumer/test until #2; #2 reuses this package's canonical-JSON helper for the cache key. Avoids speculative untested code — Simplicity-First.)
- CUE `#StepRecord`/`#RunRecord` in `construct/vocabulary/experiment.cue` (structural single source) + a drift guard (marshal a Go `RunRecord` → JSON → `cue vet -d '#RunRecord'`, `t.Skip` if `cue` absent) — the established metis#1 pattern (`pkg/experiment/conformance_test.go:TestRunConformsToCUE`).
- Unit tests: point-address determinism (map-order-independent) + sensitivity (changed resolved-with/repo-SHA/seed moves it; unchanged doesn't); `OutputHash` order-independence + path/byte sensitivity + empty set; `Runner.Run` returns per-step `[]StepRun` in topo order; JSON round-trip; CUE drift guard.
- **M1 review boundary** (`sdlc milestone-close`).

### M2 — populate during a run + write durably + `## Runs` legibility

- `cmd/metis` (thin IO): during `runExperiment`, assemble the `RunRecord` from `exp.Steps` + the new
  `[]StepRun`:
  - repo-SHAs + dirty via `git rev-parse HEAD` / `git status --porcelain`, behind an injected
    `gitProbe` seam (fake in tests; no real git in unit tests).
  - per step: `With`/`Uses` (from `exp.Steps`), `Metrics` (from `StepRun.Result`), `OutputHash` =
    `record.OutputHash([]FileHash)` built by `cas.HashOf`-ing each artifact file in `StepRun.Result.Artifacts`.
  - mint the point-address; write `runs/<id>/record.json` (**git-trackable small metadata** — durable-small → git, per the design; NOT the CAS).
- `## Runs` legibility: render the record as a **knob→score** line — config beside the metric —
  superseding today's freeform `runSummary`, **reusing its metric-formatting** (sorted keys, `k=%g`)
  rather than re-implementing the loop (ARCH-DRY, Finding 4). (Full free-param *table* is #8's ledger.)
- e2e: a run writes a valid `record.json` (conforms to `#RunRecord`); the point-address is stable
  across two identical runs; `## Runs` shows the knob→score line. Copy fixtures into `t.TempDir()`.
- atlas: add `pkg/record` + the record datatype to `atlas/index.md`; note the #3/#2 scope line.
- **M2 review boundary** (`sdlc close --milestone M2` / issue close).

## Open decisions (flag for plan-judge / operator; not blocking)

1. **`pkg/record` vs. extend `pkg/experiment`** (Go *and* CUE, Finding 5). Chose a new Go package: the
   record is a distinct concern (identity/provenance) from orchestration, and #2/#8 import it without
   pulling the runner. On the **CUE** side, `#StepRecord`/`#RunRecord` co-locate in `experiment.cue`
   for v1 (they're the same experiment-run vocabulary, and the drift-guard tooling is already wired
   there) rather than a new `record.cue` — reversible if the record noun grows independently.
2. **Record home = `runs/<id>/record.json`, git-tracked.** The design says durable-small → git. v1 writes
   it into the run dir (git-trackable); the *committing* cadence (per-run vs per-sweep batch) is #7/#8's
   (batched commits). #3 just writes the file; whether it's auto-committed is deferred.
3. **`## Runs` knob→score line** shows *all* resolved config for a lone experiment (no free params yet —
   the shape/free-params arrive in #6). For a bare experiment the "knobs" are just its config; the
   free-param *diff* becomes meaningful once #6 lifts the shape. #3 renders what's available.

## Test strategy

Pure core (M1) → direct unit tests (point-address determinism/sensitivity, JSON round-trip, CUE drift
guard). IO (M2) → a `gitProbe` fake for repo-SHAs (no real git in unit tests) + a hermetic e2e that runs
a toy experiment and asserts `record.json` conforms + point-address stability + `## Runs` line. Controllable
time via the existing injected `Clock`.

## Revisions

### 2026-07-05 — M1 built (SHIP)
- **`pkg/record` is a clean leaf over `pkg/cas` only** — it does NOT import `pkg/experiment` (the
  M1 bullet + Open-decision 1 said it would). The `StepRun`→`StepRecord` mapping moved to the
  `cmd/metis` M2 assembly site, so the record package carries zero orchestration/IO coupling (the
  M1 reviewer called this better than the plan). M2's `pkg/record` types take plain values, not
  `experiment.StepRun`.
- **M2 must handle NaN/Inf config** (M1 review Minor): `PointAddress`/`OutputHash` canonicalize via
  `json.Marshal`, which *rejects* `float64(NaN/Inf)` — and `.nan`/`.inf` are valid YAML that reach
  `resolvedWith`. Root-cause fix in M2: the derivations **return an error** (not panic) on an
  un-marshalable value, and `runExperiment` surfaces it as a run error. (Panicking on user-reachable
  input isn't senior-dev — the M1 panic was a placeholder for "programming error," which this isn't.)
- **M2 atlas pass** also updates `atlas/experiment.md` for the new `Runner.Run` `[]StepRun` return.
