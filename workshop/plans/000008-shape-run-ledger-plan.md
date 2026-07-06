---
issue: 000008
title: Shape-run ledger — append-only CSV + pick-best + promote over the sweep manifests
status: active
created: 2026-07-05
---

# Plan: metis#8 — the shape-run ledger (the L1 tracking layer)

Design **settled 2026-07-05** (issue `## Design`; pensive §L1). The last v1 substrate issue — it
turns the sweep's per-point results into a **navigable, promotable** table, so an engineer picks the
winner by sorting, not by scrolling logs. Builds directly on the three merged deps: **#7's shape-run
manifest** (per-point free-params/status/metrics + point-address run-ids), **#6's free-param paths**,
**#3's record** (point-address + commit SHA).

## The mechanism (recap)

- **A row** = `(free-param tuple | sweep-SHA | namespaced metrics | status)` + the derived
  point-address (row identity for dedup). The raw reconstructable recipe + result.
- **Three keys:** the **free-param tuple** (human navigation; ragged/sparse — columns = the union of
  all branches' free-params, blank where inactive); the **sweep-SHA** (the shape-run's code-version,
  the git short-SHA human address); the **point-address** (global content identity, dedup).
- **Append-only, point-address identity:** re-run same code → idempotent (same addresses → same rows);
  re-run at new code → new rows grouped by sweep-SHA. Rows immutable.
- **Namespaced metrics** (`train.cv_score`) fix v0's flat last-write-wins; the sweep `objective`
  (#6) names one unambiguously for pick-best + the top-N body ordering.
- **The lift unification:** the ledger IS the lifted `## Runs` — an experiment is a 1-config ledger
  (empty free-param tuple), a shape is the N-config ledger + promotion.

## Scope line — #8 (full settled design, operator-confirmed)

- **#8 owns (this issue):** the ledger datatype + append-only CSV codec (ragged columns), aggregating
  #7's manifests into rows, the **objective-driven pick-best**, the **`show`** query (sorted/filtered
  views + the top-N body summary), **`promote`** (row → all-singleton experiment committed at its code
  SHA + back-link, round-trip), the immutability discipline, **AND the side-ref dirty-code capture**
  (operator-confirmed in-scope) — so even a dirty-iteration run gets a real committed SHA and every
  row is recoverable. This finally populates the record's `CodeManifest.D`/`commit` slots #3 left for
  #2/#8.
- **Deferred to kbench#4:** authoring the actual titanic sweep + the live submission — #8 delivers the
  ledger/promote *mechanism*; kbench#4 *uses* it.
- **Carried from #6/#7:** register the `experiment-shape` noun in ariadne `vocabulary` + the merge-gate
  grep — still deferred (kbench#4 commits the first shape instance). Re-log for kbench#4.
- **The "lift unification" is CONCEPTUAL framing, not a refactor (plan-judge):** the design's "experiment
  = 1-config ledger" says a `pkg/ledger` Row *can represent* a lone experiment's run (empty free-param
  tuple), and #8's writer handles that case — but **retrofitting the existing experiment `## Runs` /
  `record.json` onto `pkg/ledger` is OUT of scope** (it would touch #3's shipped record path). #8 does
  NOT create a second parallel run representation: the ledger `Row` is an *aggregation view* over the
  per-run `record.json`s (M2 reads them), not a competing store. So no duplication — the record stays
  the L0 truth; the ledger is the L1 table over it.
- **Repo-identity (plan-judge): the point-address's checkout-basename `repo_shas` key is a DELIBERATE
  deferral, not a #8 task** — env-invariant only within a single checkout, which is all the ledger
  dedups over in v1 (single-checkout/Kaggle). Pinning to a remote-URL/configured identity is post-v1.
  Recorded in Done-when so the "global dedup" claim is scoped honestly.

## Milestones (3 review boundaries)

### M1 — the pure ledger core (`pkg/ledger`)

- New pure package `pkg/ledger`:
  - `Row{FreeParams map[string]any; SweepSHA string; PointAddr string; Metrics map[string]float64; Status string}`.
  - `Ledger` = an ordered `[]Row` + the union column-set. `Append(rows...)` — **append-only + dedup by
    point-address** (re-appending a seen point-address is a no-op; a new code-version's rows have new
    point-addresses so they append). Ragged: the free-param + metric column sets are the **union**
    across rows, blank where a row lacks a key.
  - CSV codec (over stdlib **`encoding/csv`** — don't hand-roll): `Encode(led) → []byte` (header =
    the sorted union of `fp.<name>` + `metric.<ns.name>` + the three keys + status; blank cells for
    absent keys) / `Decode([]byte) → Ledger` — round-trips.
  - `Best(led, objective) (Row, bool)` — argmax/argmin the objective metric over rows (skips `failed`
    / metric-missing rows); `TopN(led, objective, n)` for the body summary.
  - `Filter(led, {sweepSHA?, ...})` for `show` views (sorting is a *view*, never storage).
- Unit tests: append dedup (same address idempotent; new address appends), ragged union columns
  (logreg rows blank `n_estimators`, rf rows blank `C`), CSV round-trip (incl. blanks + namespaced
  metrics), Best maximize+minimize + skips-failed, TopN ordering, Filter by sweep-SHA.
- **M1 review boundary.**

### M2 — integration: write rows from a sweep, `show`, `promote`

- **Ledger write:** after a sweep, aggregate its manifest via a **pure `rowsFromManifest(manifest,
  records) []Row`** (plan-judge — the namespaced-metric extraction + sweep-SHA/point-address wiring +
  ragged-column union is unit-testable without disk; the caller does the `record.json` reads). The
  manifest already carries per-point free-params/status/metrics + the point-address run-id; the
  sweep-SHA = the manifest's shape-run repo SHA. `Append` to the shape's CSV sidecar
  (`<shape>.ledger.csv`), regenerating the shape body's top-N summary. Idempotent (re-run → dedup).
  Namespaced metrics: read each point's `record.json` per-step metrics (`train.cv_score`) rather than
  the flat run metrics — the collision fix.
- **`metis ledger show <shape> [--sweep SHA] [--sort metric] [--top N]`** — render the sidecar as a
  sorted/filtered table (a view; the CSV stays append-order).
- **`metis promote <shape> (--best | --point 'model=rf,n_estimators=300') [--sweep SHA] --name X`** —
  select the row (Best by objective, or the named free-param point), then **reconstruct the
  all-singleton experiment via a PURE helper** `promotedExperiment(shape, row) experiment.Experiment`
  (the shape's fixed leaves overlaid with the row's free-param values — the singleton collapse,
  reusing `shapePointToExperiment`'s overlay idiom / #6's `Shape`; unit-tested WITHOUT a repo), leaving
  only write+commit in the IO seam. Write `<name>.md` with a back-link (`promoted_from: <shape> @
  <point-addr>`), and **commit it at the code SHA** (warn if dirty — a promoted winner should be
  commit-nameable; M3's capture makes even a dirty run's SHA real). **Round-trip test:** the promoted
  experiment re-runs (real cached runner) and reproduces the row's point-address + metrics.
- **Immutability by per-row snapshot** (Done-when): each row already carries its full resolved point +
  sweep-SHA, so a shape-space edit can't invalidate old rows — delivered by the snapshot, no mutation
  guard needed. **Tested:** append rows, edit the shape's space, assert the prior rows still reproduce
  (their point + SHA are self-contained).
- atlas: `pkg/ledger` + the ledger datatype + `show`/`promote`.
- **M2 review boundary.**

### M3 — the side-ref dirty-code capture (git durability)

Delivers the design's "git owns code" durability so a dirty-iteration run still has a real committed
SHA and is recoverable — finally filling `record.CodeManifest.D`/`commit`.

- **Capture** (`cmd/metis`, a `gitCapture` seam alongside `gitProbe`): after a run, take the run's
  **code closure** = the first-party read-set `D` (each point's `reads.json` → the paths, via the #2
  sensor already wired). For each closure path that is **dirty or untracked**, `git hash-object -w` the
  working-tree bytes into the object DB, then build a tree (temp `GIT_INDEX_FILE` + `update-index
  --cacheinfo` + `write-tree`), `commit-tree` it with HEAD as parent, and `update-ref
  refs/metis/sweeps/<shape-run-id>` → a real, GC-protected commit. A **clean** closure needs no
  capture (HEAD is already the SHA).
- **Populate the record:** `CodeManifest.D = [(path, git-blob-hash)]` (from the closure) and
  `CodeManifest.Commit = <captured-or-HEAD SHA>`. The `(path, blob-hash, commit)` pointer-manifest is
  the durable, git-resolvable code identity (metis stores no code bytes). **`CodeManifest.Deps` (the
  uv.lock digest) is RE-SCOPED, not populated here** (plan-judge): it's Python-env provenance separable
  from the git code-closure capture — the #2 cache already folds `uv.lock` into its functional `D` for
  invalidation; recording the *digest* into the record's `Deps` is a small provenance follow-up
  (post-v1). The M3 atlas edit says "D+Commit done; Deps re-deferred" — not a blanket "done".
- **Capture granularity — per-shape-run** (confirmed): one `refs/metis/sweeps/<shape-run-id>` per
  invocation (the closure is the same code across the sweep's points), captured once (on the first
  point's closure). The `CodeManifest.Commit` on every point-record of the sweep = that captured SHA.
- **Recovery** = `git checkout <commit>` (or `git cat-file blob <hash>`) restores the exact code,
  even a past dirty version. `promote` commits the winner at that SHA → a self-contained reproducible
  commit even from a dirty tree.
- **Identity note:** the point-address stays HEAD-based (the #7 run identity — do NOT disturb it); the
  *durable code SHA* is the captured commit, recorded in `CodeManifest.Commit`. For a clean run they
  coincide; for a dirty run the capture makes the code recoverable while the run-id stays stable.
- Tests: capture a dirty first-party file → a `refs/metis/sweeps/*` commit exists + `git cat-file` of
  the pointer-manifest's blob-hash returns the exact (dirty) bytes; a clean closure captures nothing
  (Commit == HEAD, no new ref); the record's `CodeManifest.D`/`Commit` are populated; a recovery test
  (`git checkout` the captured SHA restores the file). Real-git, skip-guarded when git absent.
- atlas: the durability/capture flow + the CodeManifest population (updates the #3/#2 "deferred to #8"
  notes to "done").
- **M3 review boundary** (issue close).

## Open decisions (flag for plan-judge / operator)

1. **Durability side-ref capture — IN SCOPE (operator-confirmed 2026-07-05).** M3 delivers it (see
   above): the point-address stays HEAD-based (don't disturb #7's identity), the captured commit is the
   durable code SHA in `CodeManifest.Commit`. The one sub-question the plan-judge should sanity-check:
   is capturing per-**run** (each point) vs. once per-**shape-run** the right granularity? Leaning
   per-shape-run (one `refs/metis/sweeps/<shape-run-id>` for the invocation's closure — the closure is
   the same code across points), captured on the first miss.
2. **Row datatype home — a new `pkg/ledger` vs. extend `pkg/record`.** Chose `pkg/ledger`: the ledger
   is L1 aggregation (many rows, CSV, pick-best), distinct from the L0 per-run record. Reversible.
3. **`show`/`promote` as `metis` subcommands vs. a separate binary.** Chose `metis` subcommands (one
   driver, consistent with `metis run`). `metis ledger show` / `metis promote`.

## Test strategy

Pure core (M1) → table-driven unit tests (append dedup, ragged columns, CSV round-trip, Best/TopN,
Filter). Integration (M2) → a sweep-then-ledger e2e (a multi-point test/echo sweep writes the sidecar;
`show` renders it; `promote --best` writes + commits a round-tripping experiment). Fixtures in
`t.TempDir()`; the fake gitProbe + fixed clock already exist. The round-trip uses the real cached
runner so the promoted experiment genuinely reproduces the row.

## Revisions

### 2026-07-05 — delivered semantics (reconciling the plan with the shipped code)
Appended per AGENTS.md §1 (plan diverged mid-stream during the REWORK + close-review rounds):
- **promote commits at HEAD, not "at the code SHA"** — M2 said "commit at the code SHA (warn if
  dirty)." The code commits the promoted experiment at **HEAD** and, when the selected row's
  sweep-SHA ≠ HEAD (an older-code-version winner), **warns** that it "reproduces only after
  `git checkout <sweep-SHA>`" — the design's deliberate "go back." (You can't commit a file *at* a
  past SHA without checking it out; the warning makes the code-version mismatch loud instead.)
- **Richer `promoted_from` back-link** — delivered as `promoted_from: <shape> @ <point-addr>
  (sweep <sha>) (k=v, …)` (point-address + sweep-SHA + free-param tuple), not the plan's plainer
  `@ <point-addr>` — so a promoted experiment is checkable against AND recoverable to its origin row.
- **The ledger sidecar is written but not auto-committed** — the Design's "committed batched
  (per-sweep)" is not done; `writeSweepLedger` writes `<shape>.ledger.csv` + regenerates the body
  top-N, and committing it is left to the user / `promote` (which commits the promoted experiment,
  not the sidecar). Auto-commit of the sidecar is a post-v1 convenience.
- **`ledger show --sort` defaults `--dir` from the shape's objective direction** (round-5) so a
  minimize objective sorts best-first; and keeps failed/metric-missing rows via `ledger.SortAll`
  (round-4) rather than dropping them through `TopN` (which stays the body-summary leaderboard).
- **`Deps` (uv.lock digest) re-scoped post-v1** (change-code plan-judge) — M3 backfills `CodeManifest.D`
  + `Commit`; the uv.lock digest is separable Python-env provenance.
