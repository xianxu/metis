---
id: 000008
status: codecomplete
deps: [metis#6, metis#7, metis#3]
github_issue:
created: 2026-07-03
updated: 2026-07-05
estimate_hours: 3
started: 2026-07-05T18:39:34-07:00
actual_hours: 1.69
---

# Shape run-ledger: CSV sidecar keyed by free-param tuple + promotion to an experiment

Part of the **metis v1** project (`brain/data/project/metis-v1.md`). Design source:
`brain/workshop/pensive/2026-07-03-01-pensive-experiment-shape-workbench-design.md`.
Depends on metis#6 (experiment-shape), metis#7 (sweep runner), metis#3 (point
content-address).

## Problem

A sweep produces thousands of runs. Keeping each as a git-commit-per-run is affordable
but *unnavigable* (`git log` over 10k near-identical commits is unreadable). The runs
must be **durable AND navigable** — a structured, queryable table — without drowning
the ML engineer or churning git per run.

## Spec

- **The shape owns a structured run-ledger — a CSV sidecar**, one row per point.
  Human-navigable (sort/filter/diff by param or metric); it IS the `dvc exp show` /
  MLflow table, embedded + human-keyed.
- **Two keys per row:**
  - **Free-param tuple** (the human key): within a shape the free (non-singleton)
    params fully determine the point → `(model=logreg, k=5, lr=0.01)` is a complete,
    legible name (a float for a `Dist` leaf).
  - **Resolved-point content-address** (the global key, from metis#3): the cache/repro
    identity, unique across shapes.
- **Row contents:** the two keys + the run's metrics + status + a pointer to its CAS
  artifacts. Enough to reproduce (the point snapshot) so the ledger survives shape edits.
- **Body summary:** the shape's freeform body holds a top-N-by-metric summary + a
  pointer to the sidecar (not the full 10k rows — keep the file readable).
- **Batched commits:** commit per-sweep or every ~1K runs — no per-append churn.
- **Immutability discipline:** a shape *with* rows is ~immutable (fork to change the
  space) OR each row snapshots its full resolved point; enforce/warn one way.
- **Promotion:** `promote <shape> <row>` → materialize that point as a standalone
  all-singleton `experiment` file (for hand-iteration / the "best" on `main`), with a
  back-link to the originating shape+row. The durable spine = the sequence of promotions.
- **A `show`/query view** (`metis sweep show <shape>` / `krun show`) that renders the
  sidecar as a sorted/filtered table.

## Design (settled 2026-07-05)

Settled over the driver/ledger/durability discussion (pensive §L1 + §Promotion; builds on
#6/#7/#3). Refines the Spec: the ledger is **append-only**, row identity is the
**point-address** (not the free-param tuple — it repeats across code-versions), there's a
**third key** (the sweep-SHA), columns are **ragged/sparse**, metrics are **namespaced**, and
promotion is a **command** (the `*` inline marker is dropped).

### What a row is — the raw recipe + result

`(free-param tuple | sweep-SHA | namespaced metrics | status)` + the derived point-address. The
row is the **raw reconstructable recipe** (literal free-params, the code reference, the seed) +
the result — a Nix-derivation-shaped record. Hashes are *derived columns* (CAS-output-lookup +
dedup), never the stored identity; humans never type them.

### Three keys (the Spec had two)

- **Free-param tuple** — the human navigation key; **ragged/sparse**: with `$oneof` (#6) logreg
  rows carry `C`, rf rows carry `n_estimators`/`max_depth`, so columns = the **union** of all
  branches' free-params, blank where inactive.
- **Sweep-SHA** — the shape-run identity (#7): the git commit the sweep ran at. **Doubles as the
  human code-version address** — a git short-SHA (`a1b2c3d`) is the eyeballable handle, so a run
  is addressed by `(sweep-SHA, free-param tuple)`; metis invents no id.
- **Point-address** (#3) — the global content identity; **row identity for dedup**.

### Accumulation — append-only, point-address identity

- **Re-run same code → idempotent** (same point-addresses → same rows; nothing added).
- **Re-run at new code → new rows**, each free-param tuple now appearing once per code-version,
  grouped by sweep-SHA.
- Rows are **immutable** (a deterministic point-run's result is fixed; a `failed` point stays a
  `failed` row; fixing it = new code = a new row). "Keep every run durable" = content-addressed
  append. Navigation: best-ever = argmax objective over all rows; "at current code" = filter by
  sweep-SHA; "config X over time" = filter by tuple, group by sweep-SHA.

### The lift unification — one ledger, experiment = 1-config

The ledger is the **lifted `## Runs`**: an `experiment` is a **1-config ledger** (empty
free-param tuple, rows across code-versions — #3's structured `## Runs`); an `experiment-shape`
is the **N-config ledger** + promotion. One datatype; the whole thing lifts consistently —
config→space (#6), run-log→ledger (#8), record→per-step (#3).

### Physical form + the metric-collision fix

An **append-only CSV sidecar** (`show`/query renders sorted/filtered views — sorting is a
*view*, never a storage concern), committed **batched** (per-sweep). The body holds a generated
**top-N-by-objective** summary + a pointer. **Namespaced metrics** (per-step, e.g.
`train.cv_score`) fix v0's flat last-write-wins collision; the sweep's `objective` (#6) names one
unambiguously.

### Pick-best — objective-driven

The `objective: {metric, direction}` (#6 sweep block) drives both the body's top-N ordering and
promotion's selection. Scope: default **whole-ledger champion** (best-ever); `--sweep <SHA>`
restricts to one invocation.

### Promotion — a command, not a marker

```
promote <shape> [--best | --point (model=rf, n_estimators=300)] [--sweep <SHA>] --name titanic-winner
```

→ writes the all-singleton `titanic-winner.md` (the row's raw point, `expand`⁻¹) and, because the
winning point's code is already a committed SHA (§Durability), **commits the experiment.md at that
code** → a **single self-contained, durable, reproducible commit**. The `*` inline marker is
dropped (it would force a *sorted* ledger + manual file-marking, fighting append-only). Since
every sweep *captures* its code (below), **every row is already reproducible** — so promotion
isn't "keep reproducible," it's "graduate a point to a named, editable experiment." Promoting an
older winner (v1 while at v2) = the deliberate `checkout <v1-SHA>` + write + commit ("go back").
Promoting *two* points → two files (the primitive is point→experiment.md).

### Durability — git owns code, the CAS owns only wipeable output bytes

The capture mechanism (refines #2/#3/#9): a sweep **captures its code revision** by, on a CAS
miss, using the trace to find the closure files and — if any are dirty/untracked — **committing
just those to a side ref** (`refs/metis/sweeps/*`), so `main` stays clean and every sweep runs at
a real committed SHA. What metis persists per step is a **manifest of pointers**:
`(path, git-blob-hash, commit)` — *git's blob-hash is the content-hash; git's (commit, path) is
the location*. metis invents no code hash and stores no code bytes.
- **Recovery** = resolve each `(path, hash, commit)` from git (`checkout`/`cat-file`); recover a
  past dirty version = `git checkout <its sweep-SHA>`.
- **Durability by construction** — the manifest lives in the durable records, the blobs in git
  (side-ref, GC-protected). Wiping the CAS loses **zero** code and **zero** provenance.
- **The CAS is a wipeable `content-hash → bytes` map for large *outputs* only** (recompute on
  miss). One-line invariant: *the CAS holds nothing whose loss isn't recomputable; everything
  irreplaceable — code manifest, metrics, git blobs — lives in git.*

### Immutability discipline

A shape (or experiment) *with rows* is ~immutable — fork to change the space (each row already
snapshots its full resolved point + code SHA, so old rows stay reproducible across edits).

## Done when

*(Updated 2026-07-05 to the milestone plan — added the M3 durability criterion, reconciled the
immutability wording to the delivered per-row-snapshot mechanism.)*

- Sweeps (metis#7) append rows to the shape's CSV sidecar (three keys + ragged free-param cols +
  namespaced metrics + status); idempotent (re-run dedups by point-address); the body top-N summary
  regenerates.
- A `metis ledger show` command renders + sorts/filters the ledger (a view).
- `metis promote` materializes a ledger row (Best or a named point) as an all-singleton experiment
  with a back-link, committed at its code SHA; round-trips (the promoted experiment re-runs and
  reproduces the row's point-address + metrics).
- **Immutability by per-row snapshot:** each row snapshots its full resolved point + sweep-SHA, so
  old rows stay reproducible after a shape-space edit — tested (edit the shape's space, assert prior
  rows still reproduce). (The Design's "or per-row snapshot" branch — delivered by the snapshot, not a
  mutation guard.)
- **(M3, durability) Side-ref code capture:** a dirty-closure run captures a real committed SHA to
  `refs/metis/sweeps/*`; `record.CodeManifest.D`/`Commit` are populated; `git cat-file` of a captured
  blob-hash returns the exact bytes; recovery via `git checkout <commit>` restores the code. A clean
  closure captures nothing (Commit == HEAD).
- **Repo-identity note (carried from #3's close-review):** the point-address keys `repo_shas` by the
  local checkout basename, so its "global" dedup is env-invariant only within a single checkout —
  **deliberately deferred** here (harmless within v1 single-checkout/Kaggle use; the ledger dedups
  within an environment). Pinning repo identity to an env-invariant source (remote URL / configured
  identity) is a post-v1 hardening, not in #8's scope.

## Plan

Durable impl plan: `workshop/plans/000008-shape-run-ledger-plan.md` (3 review boundaries; full design
incl. the side-ref durability capture — operator-confirmed in-scope 2026-07-05). TDD.

- [x] Design settled 2026-07-05 — append-only ledger (3 keys, sparse cols, namespaced metrics), objective pick-best, promote command, git-owns-code durability (see `## Design`); impl decomposed into the durable plan (2026-07-05).
- [x] **M1 — pure ledger core** (`pkg/ledger`): `Row` (free-param tuple / sweep-SHA / point-address / namespaced metrics / status); `Ledger` append-only + **dedup by point-address**; **ragged** union columns; CSV codec (round-trip incl. blanks + namespaced metrics); `Best`/`TopN` (objective-driven, skip failed) + `Filter`. Unit tests.
- [x] **M2 — integration: rows + `show` + `promote`.** Aggregate #7's sweep manifest → rows (namespaced per-step metrics from each `record.json`) → append to `<shape>.ledger.csv` (idempotent) + regen body top-N. `metis ledger show <shape> [--sweep|--sort|--top]`. `metis promote` = a **pure** `promotedExperiment(shape, row)` reconstruction (singleton collapse, reusing `shapePointToExperiment`'s overlay — unit-tested without a repo) + write `<name>.md` (back-link) + commit at its code SHA; **round-trip** (re-run reproduces the row's point-address + metrics). **Immutability by per-row snapshot** (tested: edit the space, prior rows still reproduce). Atlas. The ledger is an *aggregation view* over the per-run `record.json`s — NOT a second run store (the "lift unification" is conceptual; no experiment `## Runs` retrofit).
- [x] **M3 — the side-ref dirty-code capture** (git durability). `gitCapture` seam: after a sweep, take the code closure (D from the sensor's reads.json); `git hash-object -w` each dirty/untracked closure file, build a tree + `commit-tree` + `update-ref refs/metis/sweeps/<shape-run-id>` (GC-protected). Populate `record.CodeManifest.D=(path,blob-hash)` + `Commit=<captured-or-HEAD SHA>`. Recovery = `git checkout <commit>`. Point-address stays HEAD-based (don't disturb #7). Tests: dirty-file capture + `cat-file` returns exact bytes, clean → no capture, recovery, CodeManifest populated. Atlas (updates #3/#2 "deferred to #8" → done).

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.4 impl=0.4
item: smaller-go-module      design=0.2 impl=0.4
item: greenfield-go-module   design=0.3 impl=0.4
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.05 impl=0.1
design-buffer: 0.15
total: 2.99
```

The largest v1 issue alongside #2 (3 milestones, full design incl. durability). M1 greenfield
`pkg/ledger` (pure ragged CSV + pick-best); M2 a smaller-go-module *extend* (sweep→rows + `show` +
`promote` + round-trip); M3 the greenfield git-plumbing side-ref capture (novel — hash-object -w /
commit-tree / update-ref + CodeManifest). Three `milestone-review` (3 boundaries). Atlas. Impl at
40%-of-v2 (v3.1); +15% thorough-plan buffer.

## Log

### 2026-07-03
- Filed from the metis-v1 design brainstorm. The L1 tracking layer — the piece that actually solves "don't get overwhelmed." Deps: metis#6, metis#7, metis#3 (global key). The free-param tuple as the human key is the elegant bit (falls out of the schema lift).

### 2026-07-05
- 2026-07-05: closed — Close-review round-2 Important (both fixed): (1) promote warns loudly when the selected row sweep-SHA != HEAD (cross-code-version promotion — reproduces only after git checkout <sweep-SHA>); (2) promoted_from records full origin provenance: <shape> @ <point-addr> (sweep <sha>) (k=v). --point selects an actual ledger row (findRow). go build+vet+test ./... green; VERIFIED in real CLI (back-link: promoted_from: s @ <64-hex> (sweep 538d23d3) (train.model=logreg)). All Critical(0)/Important resolved across the REWORK + 2 close-review rounds. --no-verdict: M2(REWORK)+M3 folded into this integration close. --no-project: brain tracker by hand.; review verdict: FIX-THEN-SHIP
- 2026-07-05: closed — REWORK Criticals resolved + re-reviewed: (1) promote now ACTUALLY commits — gitCLICommitter (shell git add/commit) injected in cmdPromote; message only claims commit when one lands; TestPromote_ActuallyCommits (real git) asserts champ.md committed + promote commit in log. (2) documented arg order works — hoistShapePath pulls the .md positional so flags parse in any order; TestCLI_ArgOrderIndependent. Both slipped because e2es bypassed the CLI parse (lesson captured). Also: fixture objective namespaced (train.cv_score), empty-objective now loud, round-trip asserts point_address==winning row, DRY collapse. go build+vet+test ./... green; VERIFIED in real CLI with documented arg order (ledger show s.md --sort; promote s.md --best --name w → commit lands). --no-verdict: M2 milestone-review (REWORK) + M3 folded into this integration close (fixes cover both). --no-project: brain tracker by hand (est 3.0/actual 1.45). Completes the metis-v1 substrate.; review verdict: FIX-THEN-SHIP
- 2026-07-05: closed M1 — M1 pure ledger core: go build+vet+test ./... green. pkg/ledger — 5 tests: Append append-only + dedup-by-point-address (idempotent re-run, new code-version appends); ragged CSV round-trip over encoding/csv (union columns, $oneof logreg-blanks-n_estimators/rf-blanks-C, namespaced metrics, failed status); Best objective-driven maximize+minimize + skip-failed/metric-missing + empty; TopN ordering; Filter by sweep-SHA. Pure, no IO (aggregation view over #3 records, not a competing store). BYPASS --no-atlas + --no-project: M1 is the pure core; atlas (ledger flow) + project tracker land at M2/M3 per the plan; milestone progress in the issue Plan/Log.; review verdict: FIX-THEN-SHIP
- **Design settled** (ledger/durability discussion). Append-only CSV, **row identity = point-address** (not the free-param tuple — it repeats across code-versions); **three keys** (free-param tuple [human, ragged/sparse via `$oneof`] + sweep-SHA [code-version, = the git short-SHA human address] + point-address [global dedup]); **namespaced metrics** (fixes v0 flat last-write-wins); **objective-driven** pick-best (whole-ledger default). The ledger is the **lifted `## Runs`** — experiment = 1-config ledger, unifying with #3. **Promotion = a command** (`*` marker dropped) → writes an all-singleton experiment committed **at its code SHA** = a self-contained reproducible commit. **Durability refined (updates #2/#3/#9):** a sweep captures its code by committing the traced closure to a side ref on a miss; metis persists a `(path, git-blob-hash, commit)` **pointer-manifest** (git's blob-hash = the content-hash; git = content store) — **the CAS holds only wipeable large-output bytes; code lives in git**, durable across CAS wipes. Full spec in `## Design`.
- **Carried from the #3 close-review (2026-07-05):** the point-address (which the global row-key derives from) currently keys `repo_shas` by the **local checkout basename** (`filepath.Base(git rev-parse --show-toplevel)` in `cmd/metis/record.go`). So the same commit+config+seed in `metis/` vs a `metis-2/` clone mints **different** point-addresses — harmless within v1 single-checkout/Kaggle use, but **#8 must pin repo identity to an environment-invariant source** (remote URL / configured identity) before the global ledger key relies on cross-environment stability. Also: the `dirty` flag sits *outside* the point-address by design (clean-vs-dirty is legibility; the read-set trace `D` is #2's precise encoding) — the coarse point-address is NOT a byte-exact identity.
- **M1 built — the pure ledger core `pkg/ledger`** (TDD, all green; build+vet+full-suite clean). `Row` (free-param tuple / sweep-SHA / point-address / namespaced metrics / status); `Ledger.Append` (append-only + **dedup by point-address** — idempotent re-run, new code-version appends); **ragged** CSV codec over stdlib `encoding/csv` (header = fixed keys + sorted union of `fp.*` + `metric.*`; blank cells for absent keys; round-trips incl. blanks + namespaced metrics + failed-status); `Best`/`TopN` (objective-driven maximize/minimize, skip failed + metric-missing) + `Filter` (by sweep-SHA). Pure, no IO. Tests: dedup, ragged round-trip ($oneof logreg-blanks-n_estimators / rf-blanks-C), Best both directions + skip-failed/missing + empty, TopN ordering, Filter. Next: M2 (rows-from-manifest + `ledger show` + `promote` round-trip).
- **M2 built — integration: rows + `show` + `promote`** (TDD, all green; verified in the real CLI). `cmd/metis/ledger.go` + `ledger_cmd.go`: pure `rowsFromManifest(manifest, records)` (namespaced per-step metrics from each `record.json` — the collision fix; sweep-SHA + point-address from the manifest); `writeSweepLedger` hooks into `runSweep` (append to `<shape>.ledger.csv`, idempotent dedup, regen body top-N between markers). `metis ledger show <shape> [--sweep|--sort|--top|--dir]` (sorted/filtered views). `metis promote <shape> (--best|--point 'k=v') --name X`: pure `promotedExperiment` (re-expands the shape + **matches by free-params** — reuses `Expand`+`shapePointToExperiment`, no fragile inversion; id = the `--name`) + write `<name>.md` (`promoted_from` back-link) + commit at code SHA (warn-if-dirty). e2e: sweep→sidecar+summary+idempotent, `promote --best` **round-trips** (re-runs, reproduces the row), immutability-by-per-row-snapshot (edit the space → prior rows still reproduce). **Also fixed the M1-review findings** (re-reviewed at this boundary): list-valued free-params now round-trip as lists (kbench#4's `features: [[], [title], …]` — was collapsing to a string), `TopN` clamps negative n, `Filter` returns a fresh Ledger (no seen-map aliasing), `freeParamsEqual` is JSON-tolerant to int/float drift. **Real-CLI bug caught + fixed:** promoted id was the shape's, not the `--name` (experiment id must match filename). Atlas: `pkg/ledger` entry. Next: M3 (side-ref dirty-code capture).
- **M3 built — the side-ref dirty-code capture** (TDD, real-git tests green; the "git owns code" durability). `cmd/metis/capture.go`: `captureClosure(root, closure, shapeRunID)` — `git hash-object -w` each closure file → the `(path, blob-hash)` pointer-manifest; if any is dirty/untracked, build a tree (throwaway `GIT_INDEX_FILE` + `read-tree HEAD` + `update-index --cacheinfo` + `write-tree`) → `commit-tree -p HEAD` → `update-ref refs/metis/sweeps/<shape-run-id>` (a real GC-protected commit); a clean closure returns HEAD (no ref). `captureSweepCode` hooks into `runSweep` (per-shape-run granularity): collect the closure (union of the points' `reads.json`), capture once, **backfill each point-record's `CodeManifest.D`+`Commit`**. Recovery = `git checkout <commit>` / `git cat-file blob <hash>` — even a dirty version. Tests (real-git, skip-guarded): dirty-file capture (side-ref commit + `cat-file` returns exact dirty bytes + `checkout` recovers), clean→no-ref, and the e2e `captureSweepCode` backfills the record's CodeManifest. **Closes the #3/#2 "Code.D/Commit deferred to #8"** (record doc + atlas updated); `Deps`/uv.lock-digest re-scoped as a post-v1 provenance follow-up. metis stores no code bytes — git owns code; the CAS holds only wipeable output bytes.
- **Note:** the M2 milestone-close stalled (M2 checkbox was pre-ticked → early refusal); M2+M3 are reviewed together by this final issue-close integration review (the window covers both).
- **REWORK fixes** (both close reviews returned REWORK — 2 Critical, reproduced): (1) **`promote` now actually commits** — added a concrete `gitCLICommitter` (shells `git add`/`commit`), injected in `cmdPromote`; the "committed" message now only prints when a commit lands (was writing `<name>.md`, printing "is committed", and never committing — a false success report + undelivered Done-when). Real-git test `TestPromote_ActuallyCommits` asserts `champ.md` is committed + a promote commit lands. (2) **Documented arg order works** — Go's `flag` stops at the first positional, so `metis promote <shape> --best` (as documented) errored; added `hoistShapePath` (pulls the lone `.md` positional so flags parse in any order) + `TestCLI_ArgOrderIndependent`. Both bugs slipped because the e2es called `runPromote`/`cmdLedger` directly, bypassing the CLI parse (lesson captured). Also: fixed the fixture objective (`cv_score` → `train.cv_score` — rows are namespaced) + made the empty-objective case LOUD (warn, not silent-empty), round-trip now asserts `point_address == winning row` (was status-only), DRY-collapsed `freeParamTuple`→`freeParamTupleMap`. Verified in the real CLI (documented arg order: `ledger show s.md --sort …`, `promote s.md --best --name w` → commit lands).
- **Close-review round-2 Important (both fixed):** (1) **`promote --best` at the wrong code-version** — `Best` is whole-ledger (best-ever), so the champion may be from older code than HEAD, but promote committed at HEAD without comparing. Now `runPromote` warns loudly when the selected row's sweep-SHA ≠ HEAD ("reproduces only after `git checkout <sweep-SHA>`" — the design's deliberate "go back"). (2) **`promoted_from` dropped the code-version** — now records the full origin: `promoted_from: <shape> @ <point-addr> (sweep <sha>) (k=v, …)` (the plan's `@ <point-addr>` + the sweep-SHA), so a promoted experiment is checkable against/recoverable to its row. Also: `--point` now selects an actual ledger **row** (via `findRow`, JSON-tolerant) not just a shape point, so provenance is honest. Test asserts the back-link carries `@ <point-addr>` + `sweep`. Verified in the real CLI.
- **Close-review round-3 Important (both fixed):** (1) **`ledger show` output now testable + headed** — `renderLedger` takes `io.Writer` (was `*os.File`), emits a **header row**, blank-pads absent metric cells; extracted `showLedger(...)` testable core; `TestLedgerShow_RendersSortedTable` asserts the header + objective-sorted rows (maximize: rf/gbm/logreg; minimize flips) — a `--sort`/`--dir` regression can't ship green now. (2) **namespaced-objective documented** — `construct/datatype/experiment-shape.md` now states the objective metric MUST be `<step>.<metric>` (the exact trap the fixture hit). All Critical(0)/Important resolved across REWORK + 3 close-review rounds; the metis-v1 substrate is complete.
