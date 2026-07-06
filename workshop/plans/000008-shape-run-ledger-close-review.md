# Boundary Review — metis#8 (whole-issue close)

| field | value |
|-------|-------|
| issue | 8 — Shape run-ledger: CSV sidecar keyed by free-param tuple + promotion to an experiment |
| repo | metis |
| issue file | workshop/issues/000008-shape-run-ledger.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23730c010966d0bb8924c82bb7b16b58166e4575..HEAD |
| command | sdlc close --issue 8 |
| reviewer | claude |
| timestamp | 2026-07-05T21:13:24-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have all the evidence I need. Here is my review.

```verdict
verdict: REWORK
confidence: high
```

**Summary.** metis#8 delivers a genuinely strong pure ledger core (`pkg/ledger`) and a well-tested git-side-ref capture (`cmd/metis/capture.go`) — build/vet/`go test ./pkg/ledger ./cmd/metis` are all green, the ragged CSV codec round-trips (including the list-valued-free-param fix the M1 review flagged), append-dedup/`Best`/`TopN`/`Filter` are pure and directly unit-tested (ARCH-PURE holds cleanly), and the dirty-code capture provably commits recoverable bytes to `refs/metis/sweeps/*`. What blocks SHIP is one core deliverable that is silently not wired: **`metis promote` never commits the promoted experiment**. `cmdPromote` builds `promoteOpts` without a `gitCommitter`, no production `gitCommitter` implementation exists anywhere in the tree, and the `if o.commit != nil` block is therefore dead on the CLI path — so the Done-when's "committed at its code SHA → a single self-contained, durable, reproducible commit" is not delivered, even though the code comment and the issue Log both assert it is. The round-trip e2e test can't catch it (it asserts only `run.Status == "ok"`, never that a commit happened or that the row's point-address/metrics reproduce). This is a contract drift on a headline behavior; it needs a fix + re-run, not a note.

### 1. Strengths

- **`pkg/ledger` is a clean pure core, tested without IO** (`pkg/ledger/ledger.go`). Append-only dedup-by-point-address with lazy `seen` rebuild (`:43`), ragged union columns over stdlib `encoding/csv`, objective-driven `Best`/`TopN` that skip failed/metric-missing with stable tie-break. All exercised in `ledger_test.go` with struct literals, zero disk. Textbook ARCH-PURE.
- **The M1-review findings were genuinely fixed** (not papered over): list/map free-params now JSON-encode in `cell`/`parseCell` and round-trip as lists (`TestCSV_ListFreeParamsRoundTrip`); `TopN` clamps negative `n` (`ledger.go:180`); `Filter` returns a fresh `Ledger` that no longer aliases the source's `seen` map (`ledger.go:187`); `freeParamsEqual` compares via canonical JSON so int-vs-float64 CSV drift doesn't break point matching (`ledger.go:79`). Confirmed each in the diff.
- **The side-ref capture is correct and honestly tested with real git** (`capture.go`, `capture_test.go`). `TestCaptureClosure_DirtyFile` proves the captured commit's blob `cat-file`s to the exact dirty bytes and `git checkout <commit>` restores them; `TestCaptureClosure_CleanIsHead` proves a clean closure writes no ref and returns HEAD. The throwaway-`GIT_INDEX_FILE` tree build leaves the real index/worktree untouched — the right plumbing.
- **`promotedExperiment` reconstructs by re-derivation, reusing `shape.Expand` + `shapePointToExperiment`** (`ledger.go:59`) rather than a fragile expand-inversion — ARCH-DRY done right, and unit-tested for the match + no-match + fixed-leaf-preservation cases.
- **`isPointOutcome` / sweep-fatal handling upstream stays intact**; the ledger/capture hooks were inserted after `writeManifest` without disturbing the per-point failure semantics (`sweep.go:115-129`).

### 2. Critical findings

- **`metis promote` silently does not commit — the "committed at its code SHA" Done-when is undelivered** (`cmd/metis/ledger_cmd.go:104-107` + `:172-179`). `cmdPromote` constructs `promoteOpts{…, git: gitCLI{}}` and never sets `commit`, so `o.commit == nil`; the `if o.commit != nil { Add…; Commit… }` block is skipped on every real invocation. Grep confirms **no type anywhere implements `gitCommitter` (Add/Commit)** and **no test sets `commit:`** — the interface is defined, referenced only by a nil field, and never wired. The interface comment even claims "nil → real git," but that fallback was never written; the behavior is the opposite (nil → do nothing).
  - *Failure scenario:* `metis promote titanic.md --best --name winner` writes `winner.md`, prints `metis: promoted … → winner.md`, and exits 0 — but the file is **uncommitted**. The design's "single self-contained, durable, reproducible commit" and Done-when line 152-153 are not met; the implementor's Log (`## Log`, "commit at code SHA (warn-if-dirty)") asserts a behavior the code doesn't perform.
  - *Fix:* implement a real `gitCLI`-backed committer (`Add` = `git -C dir add <path>`, `Commit` = `git -C dir commit -m …`), inject it in `cmdPromote` (or make the `nil` branch shell real git as the comment intends), and add an e2e assertion that after promote `git log`/`git status` shows `winner.md` committed. Handle the "nothing to commit"/no-repo cases (degrade with a warning, matching the dirty-warning style).

### 3. Important findings

- **The promote round-trip test under-asserts the Done-when** (`ledger_e2e_test.go:98`, `TestLedger_PromoteBestRoundTrips`). Done-when requires the promoted experiment to "re-run and reproduce the row's point-address + metrics," but the test checks only `run.Status == "ok"` (`:~135`). A reconstruction bug that mints a *different* point (wrong free-param typing, dropped `$oneof` bundling) would still pass. *Fix:* assert the re-run's `record.json` `PointAddress` (and metrics) equal the promoted row's — this is also what would have caught the commit gap had it asserted the commit.
- **`metis ledger show` has zero test coverage.** `cmdLedger` + `renderLedger` (`ledger_cmd.go:20-73`) — the `--sweep`/`--sort`/`--top`/`--dir` view and the table rendering — are exercised by no test; the e2e only reads the sidecar file directly. A user-facing subcommand shipping untested. *Fix:* a table test over `cmdLedger` (widen `renderLedger` to `io.Writer`, below, and assert on a buffer) covering sort + sweep-filter + top.

### 4. Minor findings

- `renderLedger(out *os.File, …)` (`ledger_cmd.go:57`) should take `io.Writer` — the `*os.File` type is the reason `show` can't be unit-tested against a buffer.
- DRY: `freeParamTuple(Row)` (`ledger.go:191`) duplicates `freeParamTupleMap(map)` (`ledger_cmd.go:220`); the former should delegate to the latter.
- DRY + latent gap: `ledgerParseCell` (`ledger_cmd.go:239`) claims to type values "the same way the CSV codec does" but omits the list/map JSON handling `parseCell` (`ledger.go:238`) has — a list-valued `--point` selector can't match a decoded row (and comma-splitting can't express a list anyway). Consolidate to one cell parser.
- `backfillCodeManifest` sets **every** step's `Code.D` to the whole-sweep-closure union (`capture.go:104`), over-attributing files a given step never read — coarser than the per-step `reads.json` already on disk. Fine for recovery, imprecise as per-step provenance; matches the plan's "D = the closure" wording, so note-only.
- `captureClosure` hardcodes mode `100644` in `update-index --cacheinfo` (`capture.go:70`) — an executable closure file would lose its exec bit on `git checkout` recovery. Edge (Python code is 0644); note.
- `sweepSHAOf` returns an arbitrary map entry when `RepoSHAs` has >1 repo (`ledger.go`/`ledger.go:47` in cmd) — non-deterministic beyond v1's single repo; acknowledged in the comment.
- `promoted_from` back-link is written as `promoted_from: <shape> (k=v, …)` (`ledger_cmd.go:196`), not the plan's `<shape> @ <point-addr>`. Arguably better (human free-params), but reconcile the plan text or the code so they agree.

### 5. Test coverage notes

Pure-core coverage is excellent and pins real properties, not the implementation. The gaps that map to shipped-bug risk: (a) **the promote commit path — zero coverage in production and in tests**; (b) the round-trip test asserts liveness, not reproduction (point-address/metrics); (c) `metis ledger show` rendering/sort/filter — untested; (d) capture only fires when `reads.json` exists (cache on, the default) — the e2e ledger sweeps use `cache:false`, so capture is a no-op there and is proven only by the isolated `capture_e2e_test.go` that hand-writes `reads.json` (acceptable, but the sweep→capture wiring is never exercised end-to-end together).

### 6. Architectural notes for upcoming work

- **ARCH-DRY: pass.** `promotedExperiment` reuses `Expand`+`shapePointToExperiment`; the codec reuses stdlib. Only the two `freeParamTuple*` helpers and two cell-parsers are minor local duplication.
- **ARCH-PURE: pass.** `pkg/ledger` and the new pure helpers (`rowsFromManifest`, `namespacedMetrics`, `promotedExperiment`, `freeParamsEqual`) are deterministic and unit-tested without IO; the IO seam (`writeSweepLedger`, `loadLedger`, `captureClosure`) is thin and injected where it matters.
- **ARCH-PURPOSE: flag.** The promote path settles for the easy subset — write the file — and leaves the committed-at-code-SHA guarantee (the point of "the durable spine = the sequence of promotions") as an unwired seam. This is precisely the ARCH-PURPOSE failure mode: a follow-up-shaped gap where the deferred part *is* the purpose, and the Log claims it's done.
- **Ledger `SweepSHA` is HEAD-based, not the captured commit** (`sweepSHAOf` reads `RepoSHAs`). Two dirty iterations at the same HEAD get the same sweep-SHA *and* the same point-address (per #7's HEAD-based identity) → the second dedups away, so the ledger cannot distinguish dirty code-versions even though M3 captures them. This is within the documented v1 scope (precise dirty-identity is metis#10), but downstream navigation ("config X over time, grouped by sweep-SHA") won't separate dirty iterations — worth a Log line so it isn't mistaken for a bug later.

### 7. Plan revision recommendations

- **`workshop/plans/000008-shape-run-ledger-plan.md`** — add a `## Revisions` entry reconciling M2/Done-when to reality: `promote` currently writes the experiment but does **not** commit it (the `gitCommitter` seam is unimplemented/uninjected); the "committed at its code SHA" and "round-trips → reproduces point-address + metrics" criteria are not yet met. Either land the commit + reproduction-assertion (preferred, closes the Critical) or explicitly re-scope the auto-commit as a follow-up and drop the "committed" claim from Done-when so the issue stops asserting what the code doesn't do.
- **`workshop/plans/…-plan.md`** — reconcile the `promoted_from` back-link format (plan says `<shape> @ <point-addr>`; code writes `<shape> (k=v,…)`).
