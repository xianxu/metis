# Repeat-Shaped Runs and Single-Run Ledger Rows Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Depends on:** metis#71 (never-cache, so a repeat's nondeterministic steps really re-execute). Code starts after #71 lands.

**Goal:** `metis run --repeat <label> exp.md` records each execution as a distinct, immutable sample (its own run dir, record and ledger row), plain single runs also get ledger rows, and `metis ledger show` reports per-config means with error and coverage across repeats, never calling a repeat a fold.

**Architecture:** A repeat label is an execution coordinate, not configuration. It joins the run-dir name and the record, and it adds a ragged `repeat` ledger column. Exclusive creation of the run dir is the one-step claim that makes a label single-use. A ledger row is derived only from the finished, atomically written record, and is published last, under a file lock, through an atomic rewrite. Legacy rows (no repeat) keep today's key, skip-duplicate behaviour and aggregation unchanged.

**Tech Stack:** Go, CSV ledger (`pkg/ledger`), CUE vocabulary.

---

## Design decisions (for TL review before code)

1. **Surface: `metis run --repeat <label>`**. The label is a slug (`[a-z0-9-]{1,32}`), validated before any effect. It is allowed only on `type: experiment`; an experiment-shape (sweep) refuses with a clear message. Repeats of sweeps are a non-goal: sweeps already have folds and seeds. couch-bench picks the labels (e.g. `1..N`), and each repeat is one `metis run` invocation, so long sessions can run in parallel through Couch. The seed stays as declared and is recorded. Repeats are different samples because the steps are nondeterministic (#71's `policy: never`), not because metis perturbs the seed. If an experiment has no uncached step, a repeat would only replay the cache, so metis prints a warning line. It does not refuse, because a deterministic replay is a legitimate sanity check.
2. **Execution identity and run dir.** The run ID is `<point_addr>.r-<label>`, and the dir is created with `os.Mkdir` (exclusive) **before** any step runs. That is the claim: a second run with the same label, concurrent or later, refuses with "repeat <label> already executed at runs/…", without touching the dir. No observation-based absence check is needed: mkdir is the lock, held for the life of the dir (tl-worker invariant rule). Plain runs (no `--repeat`) keep today's content-addressed dir, which is overwritten on rerun, unchanged.
3. **Record binds the coordinate.** `RunRecord` gains `repeat` (omitempty), and CUE `#RunRecord` gains `repeat?: string`. The record already binds seed, point address, code fingerprint, experiment and timestamps. run.json and record.json are now written with temp file plus rename (atomic), so an interrupted publication never leaves a truncated record.
4. **Ledger row identity.** The key is `point_addr ␀ code_fingerprint`, extended with `␀ repeat` **only when repeat is non-empty**, so legacy keys are byte-identical. Duplicates:
   - Legacy rows keep skip-duplicate (first wins), unchanged.
   - Repeat rows: an exact duplicate (every field equal) is idempotent. Same key with different content is refused with an error (`Append` returns an error; legacy appends never error).
5. **Single-run rows.** After record.json is final, `type: experiment` runs publish one row to `<exp>.ledger.csv`. That reuses `ledgerPath`, the shape-ledger naming, so `metis ledger show exp.md` works. The row contents: no free params, `Fold` nil, status, namespaced metrics (same helper as sweeps), and `repeat` when set. A failed run publishes `status=failed` with blank metrics, never zeros.
6. **Publication order, interruption and recovery.** The order is: claim dir → execute → run.json → record.json (atomic, code fingerprint backfilled) → row. The row is derived only from record.json. New verb: **`metis ledger publish <run-dir>`** re-derives and appends it idempotently. Outcomes:
   - No record or an unfinished record: refuses "interrupted execution: no final record".
   - A row already present: no-op.
   - A conflicting row: refuses.
   A re-run refused on an existing label prints this verb when that execution's record is final but its row is missing.
7. **Concurrent writers.** Parallel repeats rewrite the same CSV, so every load-append-write takes an exclusive `flock` on `<ledger>.lock`, and the write is temp file plus rename. Today's unlocked full rewrite would lose rows under parallel repeats. Sweeps' `writeSweepLedger` goes through the same helper (ARCH-DRY: one ledger writer).
8. **Unknown columns survive (ARCH-EVOLVE).** `Decode` currently drops unknown columns. `Row` gains `Extra map[string]string`, captured on decode and re-emitted verbatim (sorted, after the known columns), so this or any future column survives a rewrite by this version. Known-column parsing stays strict.
9. **Repeat aggregation.** `AggregateView` gains a second grouping:
   - Repeat rows (`Fold == nil`, `Repeat != ""`) group by `code_fingerprint | point_addr`, meaning the same config, seed and code. Differing fingerprints form separate groups.
   - For **every** metric each group carries the mean, `<m>.se` (sample sd/√n), and `<m>.n` (how many repeats reported it), plus `repeats.total` and `repeats.failed`.
   - Failed repeats count in `total` and `failed` but contribute no values. Missing values shrink only that metric's n; they are never filled.
   - The group's status is `ok` if any member is ok.
   - Fold groups and pass-through rows are unchanged (objective-only se, any failure fails the group), so existing views and select keep their meaning.
   `metis ledger show` renders repeat groups with no fold column. Nested CV is never invoked.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Row.Repeat`, `Row.Extra` | `pkg/ledger/ledger.go` | modified |
| `dedupKey` / `Append` (conflict refusal) | `pkg/ledger/ledger.go` | modified |
| `repeatGroups` (in `AggregateView`) | `pkg/ledger/ledger.go` | new |
| `RepeatLabel` / `ParseRepeatLabel` | `pkg/experiment/repeat.go` | new |
| `repeatRunID` | `cmd/metis/run.go` | new |
| `rowFromRecord` | `cmd/metis/ledger.go` | new |
| `RunRecord.Repeat` | `pkg/record/record.go` | modified |

- **rowFromRecord**: a pure mapping from record to row, shared by the single-run publisher and `ledger publish`. It reuses `namespacedMetrics`. DRY: the one place a row is derived from a record.
- **repeatGroups**: a pure fold over rows, unit-tested on hand-built rows: identical-config repeats, differing fingerprints, a missing metric, a failed member, and a legacy fold group unchanged.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `claimRunDir` | `cmd/metis/run.go` | new | `os.Mkdir` exclusive |
| `writeFileAtomic` | `cmd/metis/atomic.go` | new | temp file + rename (reuses the caching.go `writeEntry` pattern, extracted) |
| `withLedgerLock` / `publishRows` | `cmd/metis/ledger.go` | new | `flock` + load/append/atomic write |
| `metis ledger publish` | `cmd/metis/ledger_cmd.go` | new | CLI |

- **writeFileAtomic**: extracted from `cachingExecutor.writeEntry` (ARCH-DRY) and used for run.json, record.json, the backfilled record, and the ledger.
- **No fake needed**: only the filesystem. Concurrency tests use real goroutines and processes on a temp dir: N parallel `publishRows` lose no rows, and two `claimRunDir` calls on one label leave exactly one winner.

## State design table

Actions available: `metis run --repeat L`, `metis run` (plain), `metis ledger publish <dir>`, `metis ledger show`, and deleting a run dir by hand.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| new label, run succeeds | run dir, final record, ok row | none | none |
| new label, step fails | record status failed; failed row (blank metrics) | rerun with a new label | none (diagnostics kept, #72) |
| label already used, record final, row present | `run --repeat L` refuses; `publish` is a no-op | use a new label | none |
| label used, record final, row missing (crash after record) | `run` refuses and names `metis ledger publish` | `metis ledger publish runs/…` | none |
| label used, no or unfinished record (crash mid-run) | `run` refuses; `publish` refuses "interrupted" | new label; the dir stays as evidence | that execution's sample |
| two processes, same label, concurrently | one runs, the other refuses at mkdir | none | none |
| parallel repeats publishing at once | rows serialized by flock | none | none |
| crash during ledger rewrite | old ledger intact (rename never happened) | `ledger publish` the affected run | none |
| publish of a hand-edited record that conflicts with its row | refuses "conflicting data for execution …" | investigate; the ledger is unchanged | none |
| plain run rerun (legacy) | dir overwritten; row skip-duplicate per (addr, fp) | as today | as today |
| ledger with an unknown column (written by a newer metis) | read, and the column is preserved on rewrite | none | none |
| older metis rewrites a ledger with a `repeat` column | **repeat column dropped** (old decoder) | rerun publish with this version | the repeat label on rewritten rows; flagged under EVOLVE |

Covered: claim, execute, publish, and every interruption point. **Not covered:** an older metis binary rewriting a new ledger. The old decoder cannot be fixed retroactively; couch-bench pins metis, and the risk is stated.

## ARCH notes

- **PURE:** label parsing, row derivation, the key, conflict detection and aggregation are pure. Mkdir, flock and atomic writes are thin glue.
- **ORDER:** the publication sequence is linear, and the row is the last effect. Every crash point maps to a table row with one recovery action. Concurrency: the mkdir claim for one label, the flock for the ledger.
- **CONSTRAINTS:** a batch workload. One row per execution; a publish is O(rows) to load and rewrite the CSV, fine for thousands of rows. Lock wait is bounded by another publisher's rewrite (milliseconds).
- **FUNERAL:** each repeat adds one run dir and one ledger row of roughly 1 KB. The ledger is the research record and grows without automatic removal, by design. Run dirs are removed by hand or by the existing runs/ lifecycle, and the row keeps its meaning without the dir (the record's numbers are in the row). The `.lock` file is one per ledger and empty.
- **SECURE:** the label is validated (slug, so no path traversal into the run dir). Records read by `publish` are parsed strictly; a truncated record refuses rather than publishing partial data.
- **EVOLVE:** the `repeat` column is ragged and additive. Legacy keys and output are byte-identical, unknown columns are now preserved, and record `repeat` is omitempty. Tests: a ledger with an unknown column round-trips through append and rewrite; a legacy CSV round-trips byte-identically.
- **MOCK:** N/A beyond the filesystem; no external service.
- **PURPOSE:** repeats, single-run rows, idempotent and conflicting publication, interrupted publication, and repeat statistics with coverage are all delivered. No couch-bench JSONL ledger.

## Tasks (single review boundary: one close)

- [x] Task 1: `writeFileAtomic`, extracted from caching.go `writeEntry`; run.json and record.json writes switch to it. Test: crash simulation, where an injected failure before rename leaves the old file.
- [x] Task 2: `pkg/ledger`: `Row.Repeat`, `Row.Extra`, the extended `dedupKey`, `Append` returning an error on a repeat conflict, Encode/Decode for `repeat` and unknown columns. Tests: legacy round-trip byte-identical, unknown-column preservation, idempotent exact duplicate, conflict refusal, legacy skip-duplicate unchanged.
- [x] Task 3: `repeatGroups` in `AggregateView`. Tests: per-metric mean/se/n, a missing metric, a failed member, differing fingerprints, fold groups unchanged.
- [x] Task 4: `ParseRepeatLabel`, `RunRecord.Repeat` plus CUE, `--repeat` flag, `repeatRunID`, `claimRunDir`, the no-uncached-step warning, sweep refusal.
- [x] Task 5: `rowFromRecord`, `withLedgerLock`/`publishRows`, single-run publication, `writeSweepLedger` through `publishRows`, and `metis ledger publish`.
- [x] Task 6: e2e tests through `runExperiment` with the `traceFakeExec` seam:
  - Two repeats of an identical config give 2 dirs, 2 records, 2 rows, and an aggregated n=2.
  - Repeat L again refuses.
  - A code edit between repeats gives 2 fingerprint groups.
  - Interrupted publication: record written but publish skipped through a seam; `ledger publish` adds the row; a second publish is a no-op.
  - A plain run gets a row.
  - The legacy sweep ledger's bytes are unchanged (the existing shapesweep tests stay green).
  - Parallel publishers lose no rows.
- [x] Task 7: atlas: the run identity, ledger and `metis ledger publish` sections.

## Revisions

### 2026-10-23, plan-quality round 1 (metis:1)

Reason: three Important findings and five Minor ones. Deltas supersede the matching text above.

- **PQ-2: plain-run rows mirror their run dir.** Today single runs write no row; the earlier "as today" was wrong. A plain row (`Fold` nil, `Level` "", `Repeat` "") describes `runs/<addr>/`, which a rerun overwrites. So a plain row is **replaced** on publish with the same (addr, fp): latest replay wins, and the row always matches the record on disk. A different fp appends a new row (a code change), as for sweeps. Repeat rows stay immutable samples (idempotent or conflict). Sweep fold and outer rows keep first-wins skip-duplicate, unchanged. Test: rerun a plain experiment under never-cache with different metrics; there is one row, and it equals the new record.
- **PQ-1: aggregate rows never collide.** A repeat-group aggregate row carries `Repeat = "*"`, a reserved marker that cannot be a label because labels are slugs, so its key differs from the plain row of the same point. A plain row is a replay, not a sample: it passes through and never joins the repeat family. Test: a plain row plus repeat rows of the same point give both in the view, with the aggregate's n counting the repeats only.
- **PQ-3: one atomic record write defines "final".** The code-fingerprint backfill is refactored to operate on the in-memory `RunRecord` (`backfillCodeManifest(*RunRecord, …)`) before a **single** `writeFileAtomic` of `record.json`. A record that exists therefore parses and is final. When the backfill fails, the record is written with an empty `code_fingerprint` and its existing capture status, still warn-only, and the row is published with fp "" (its own group). A later re-publish derives the identical row, so a crash or retry can never cause a false conflict. `ledger publish` refuses only a missing or unparseable record.
- **Minor: one atomic helper.** `writeFileAtomic` consolidates `pkg/cas` `FSStore.writeAtomic` and `cachingExecutor.writeEntry` into a single helper in `internal/fsx`. No third copy.
- **Minor: the "strict" claim is corrected.** `Decode` keeps today's leniency for legacy columns. The new `repeat` cell must be a valid label, or Decode errors naming the row.
- **Minor: where publication hooks in.** Single-run publication lives in `runExperiment`'s non-shape branch, after `runResolvedExperiment` returns, never inside the runner that sweep folds share.
- **Minor: Append errors and flag precedence.** `Append` returning an error makes `Decode` fail on a ledger holding two conflicting repeat rows. That is intended: a contradictory research record is reported loudly, naming both rows. `--run` with `--repeat` refuses, since the two choose the run dir in conflicting ways.
- **Minor: test strategy lines (replaces the prose enumeration in Tasks 2, 3 and 6).**
  - `Encode`/`Decode`: a round-trip property over generated ledgers mixing legacy, ragged, repeat and unknown columns. Legacy bytes are identical, and unknown columns survive.
  - `Append`: a table over the row kinds (sweep, plain, repeat), crossed with duplicate kinds (identical, conflicting, different fp).
  - `AggregateView`: a property that per-metric n equals the number of reporting members and that no value is invented. Fold groups are byte-compared to the pre-change output.
  - `publishRows`: N real concurrent writers; the final row set equals the union.
  - e2e: claim/refuse, interrupted publish, and replace-on-rerun through `runExperiment`.

### 2026-10-25, implementation and close review round 1 (metis:1)

Reason: the code settled some names and surfaces differently from the text above, and the close review (FIX-THEN-SHIP) found four Important gaps. Deltas:

- **Names:** the flock lives inline in `publishRows` (no separate `withLedgerLock`), and the repeat grouping is a branch of `AggregateView` (no separate `repeatGroups` function). The plain-row replace is `Ledger.Publish`, while `Append` keeps first-wins skip (and refuses repeat conflicts). `Ledger.Has` exposes the one row key for the claim's recovery hint.
- **`metis ledger publish <experiment.md> <run-dir>`:** a run dir does not name its experiment file, and the ledger sits beside the .md. A dir outside that experiment's `runs/` refuses.
- **BR-2, `--repeat` needs a tracked experiment:** without git there is no content address, the run dir falls back to a timestamp, and the mkdir claim would never collide. `--repeat` now refuses there, before any effect.
- **BR-1, labels validated at publish:** `Ledger.Publish` refuses an invalid label, and `readFinalRecord` refuses a repeat record whose dir is not `repeatRunID(point_address, repeat)`. A hand-edited record can no longer make the ledger unreadable.
- **BR-3, records keyed by run id:** `loadLedgerRecords` and `cohortSummaries` key records by `rowRunID`, so each repeat cohort shows its own record.
- **BR-4, no invented uncertainty:** a repeat family omits `<m>.se` when n < 2 (the n=1 se was a made-up zero). Fold groups are unchanged.
- **Minors:** a repeat refused at validation releases its empty claimed dir. A replaced plain row keeps its unknown columns. `SameRow` ignores `Extra`, since a row re-derived from a record cannot know a newer version's columns.
