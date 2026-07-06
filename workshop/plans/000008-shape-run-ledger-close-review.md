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

---

## Re-review — 2026-07-05T21:20:09-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T21:20:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have verified the code against the Spec/Plan/Done-when and confirmed the prior REWORK Criticals are genuinely fixed. Build, vet, and the full test suite are green. Here is my review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#8's whole-issue close is in good shape: the two blocking Criticals the prior close-review found are genuinely resolved, not papered over. `metis promote` now commits — a real `gitCLICommitter` (`cmd/metis/ledger_cmd.go:135-146`) is injected in `cmdPromote` (`:131`) and `TestPromote_ActuallyCommits` asserts `champ.md` lands as a committed file with a `promote` log entry; the documented `<shape>`-first arg order works via `hoistShapePath` (`:61-76`) with `TestCLI_ArgOrderIndependent` pinning both orders; the namespaced-objective fixture is fixed (`train.cv_score`) and the empty-objective case is now loud (`ledger.go:162-167`, `ledger_cmd.go:185`) instead of silently empty. `pkg/ledger` remains a clean pure core (ARCH-PURE holds), the side-ref capture provably commits recoverable bytes, and the round-trip test now asserts point-address reproduction (`ledger_e2e_test.go` `TestLedger_PromoteBestRoundTrips`). What keeps this from a clean SHIP is one real reproducibility gap that survived the rework: **`promote --best` selects the whole-ledger (best-ever) champion, which can be an older code-version, but always commits at HEAD with no warning and no sweep-SHA recorded** — so in the cross-version case the "committed at its code SHA / reproduces the row's point-address" Done-when silently doesn't hold. That plus an untested `ledger show` renderer are the Important items; none are hard blockers, so the gate passes as FIX-THEN-SHIP.

### 1. Strengths

- **The prior REWORK Criticals are fixed with tests that would catch a regression** — `TestPromote_ActuallyCommits` drives the real `cmdPromote`/`gitCLICommitter` path against a real repo and asserts the *side effect* (committed file + log line), exactly the "don't let the success print outrun the action" lesson (`workshop/lessons.md`). `TestCLI_ArgOrderIndependent` drives the actual `cmdLedger` entrypoint in the documented order. Good closure of feedback.
- **`pkg/ledger` is a textbook pure core** (`pkg/ledger/ledger.go`): append-only dedup-by-point-address with lazy `seen` rebuild (`:44`), ragged union columns over stdlib `encoding/csv`, objective-driven `Best`/`TopN` skipping failed/metric-missing with stable tie-break — all unit-tested with struct literals, zero IO.
- **`promotedExperiment` reconstructs by re-derivation** (`cmd/metis/ledger.go:66`), reusing `shape.Expand` + `shapePointToExperiment` rather than inverting a flat map, and `freeParamsEqual` (`:83`) compares via canonical JSON so int-vs-float64 CSV drift can't break matching. ARCH-DRY done right, and the round-trip now verifies the exact point-address, not just `status==ok`.
- **The side-ref capture is honestly tested with real git** (`capture.go`, `capture_test.go`): the throwaway `GIT_INDEX_FILE` tree build leaves the real index untouched, and `TestCaptureClosure_DirtyFile` proves the captured blob `cat-file`s to the exact dirty bytes and `git checkout` restores them.
- **The record-doc / atlas updates match the delivered behavior** — `pkg/record/record.go:29-43` now says D/Commit are backfilled by #8's capture (Deps re-scoped post-v1), and `atlas/index.md` documents the fixed positional-first commands, not the broken ones the M2 review had flagged.

### 2. Critical findings

None. The two blocking Criticals from the prior close-review (promote never commits; documented arg order errors) are verified fixed in code and pinned by new tests; the full suite is green.

### 3. Important findings

- **`promote --best` can commit at the wrong code-version, breaking the reproducibility Done-when, with no guard** — `cmd/metis/ledger_cmd.go:206-217`. `Best` (`pkg/ledger/ledger.go:147`) scans *all* rows regardless of sweep-SHA, so the whole-ledger (best-ever) champion may be from an older code-version than the current HEAD. `runPromote` reads HEAD only for the dirty check and **discards the sha** (`:208`, `_, _, dirty`), then commits the reconstructed experiment at HEAD. It never compares HEAD to the selected row's `SweepSHA`.
  - *Failure scenario:* sweep at v1 (best row = `logreg`, point-address `X`), commit new code (HEAD → v2), `metis promote titanic.md --best --name winner`. It writes+commits `winner.md` at v2 and prints success — but a re-run of `winner.md` at v2 mints a v2-based point-address `Y ≠ X`, so the promoted winner does **not** reproduce the row it claims to. The design explicitly says promoting an older winner "= the deliberate `checkout <v1-SHA>`," but nothing enforces or warns about it.
  - *Fix:* when `probeRepo`'s sha ≠ the selected row's `SweepSHA`, warn (or refuse without `--force`), and record the `SweepSHA` in the promoted file so it's self-describing (see next finding). At minimum surface the mismatch — a silent wrong-version commit is the worst outcome for a "durable, reproducible" artifact.
- **The `promoted_from` back-link drops the code-version** — `cmd/metis/ledger_cmd.go:246` writes `promoted_from: <shape> (k=v, …)` with no sweep-SHA or point-address. The plan specified `promoted_from: <shape> @ <point-addr>` (`plan.md` M2). Without the SHA/point-addr the promoted experiment can't be checked against its origin row, which is what makes the previous finding silent. *Fix:* include the row's `SweepSHA` (and/or `PointAddr`) in the back-link; reconcile plan text with the chosen format.
- **`metis ledger show` output is untested and unroutable to a buffer** — `renderLedger(out *os.File, …)` (`cmd/metis/ledger_cmd.go:79`) takes a concrete `*os.File`, so the table (sort order, sweep filter, union columns) can't be asserted; `TestCLI_ArgOrderIndependent` only checks "no error." A user-facing subcommand ships with its rendering unexercised. *Fix:* widen to `io.Writer` and add a table test over `cmdLedger` asserting the rendered rows for `--sort`/`--sweep`/`--top`.

### 4. Minor findings

- `gitCLICommitter.Commit` runs `git commit -m` with no pathspec (`ledger_cmd.go:143-145`) — commits the whole index, so any pre-staged unrelated change is swept into the "metis promote" commit, undercutting "self-contained." Prefer `git commit -m … -- <file>` (or commit only the added path).
- Re-promoting the same `--name` with identical content stages nothing → `git commit` errors "nothing to commit" and `runPromote` returns an error *after* rewriting the file (`ledger_cmd.go:211-214`). Degrade gracefully (treat no-op as success).
- `ledgerParseCell` (`ledger_cmd.go:276`) duplicates `pkg/ledger.parseCell` (`ledger.go:247`) **minus** the list/map JSON branch, so `--point` can't select a list- or `$oneof`-valued free-param (and comma-splitting can't express one anyway). Export/reuse one parser; note the scalar-only `--point` limit. (ARCH-DRY.)
- Round-trip asserts point-address but not metrics (`TestLedger_PromoteBestRoundTrips`); Done-when says "point-address + metrics." Cheap to add.
- `ledger show --dir` defaults to `maximize` (`ledger_cmd.go:27`) and ignores the shape's declared objective direction — `--sort train.loss` on a minimize objective silently maximizes. Default `--dir` from `sh.Sweep.Objective.Direction`.
- The `objective.metric` must be namespaced (`train.cv_score`), but `construct/datatype/experiment-shape.md:45` still documents it as plain `{metric, direction}` — a reader will write `cv_score` and hit the (now-loud) warning. Add one line documenting the `<step>.<metric>` requirement.
- `backfillCodeManifest` (`capture.go:148-151`) sets every step's `Code.D` to the whole-sweep closure union — coarser than the per-step `reads.json` on disk. Fine for recovery, imprecise as per-step provenance (matches plan wording; note-only).
- `captureClosure` hardcodes mode `100644` in `update-index --cacheinfo` (`capture.go:61`) — an executable closure file would lose its exec bit on recovery. Edge (Python is 0644).

### 5. Test coverage notes

Pure-core coverage is excellent and pins real properties (dedup, ragged round-trip incl. list free-params, Best/TopN, immutability-by-snapshot). The rework closed the two big gaps (real commit path; documented arg order; point-address reproduction). Remaining gaps mapping to shipped-bug risk: (a) **cross-version `promote --best`** — no test where HEAD ≠ the selected row's SweepSHA, which is exactly the Important finding above; (b) `ledger show` rendering/sort/filter — untested; (c) round-trip asserts point-address but not metrics; (d) the sweep→capture wiring is only proven by the isolated `capture_e2e_test.go` (the ledger e2es use `cache:false`, so capture is a no-op there) — acceptable, but the two never fire together end-to-end.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `freeParamTuple` now delegates to `freeParamTupleMap`; `promotedExperiment` reuses `Expand`+`shapePointToExperiment`; codec reuses stdlib. Only the `ledgerParseCell`/`parseCell` split remains (Minor).
- **ARCH-PURE — pass.** `pkg/ledger` and the pure helpers (`rowsFromManifest`, `namespacedMetrics`, `promotedExperiment`, `freeParamsEqual`) are deterministic and unit-tested without IO; `writeSweepLedger`/`loadLedger`/`captureClosure`/`runPromote` are the thin injected shell.
- **ARCH-PURPOSE — flag.** Shadow-sweep on "the ledger is the single source": `show`, `promote`, and the body top-N all derive from the sidecar — good. The purpose gap is promote's fidelity: "committed at its code SHA, reproduces the row's point-address" is delivered only when HEAD coincides with the selected row's SweepSHA; the whole-ledger `--best` path can silently violate it (Important #1). This is the ARCH-PURPOSE failure mode — the harder half of the named purpose (reproducible-at-the-row's-code-version) is under-delivered for a documented-supported selection.
- **Dirty-iteration identity (carry-forward, not a bug):** two dirty iterations at the same HEAD get the same point-address (dirty flag sits outside the address) → the second dedups away, so the ledger can't separate dirty iterations even though M3 captures them. In-scope for v1 (precise dirty-identity is metis#10); worth a Log line so it isn't mistaken later.

### 7. Plan revision recommendations

- **`workshop/plans/000008-shape-run-ledger-plan.md`** — add a `## Revisions` entry recording the delivered-vs-designed promotion semantics: `promote` commits at **HEAD**, not the selected row's sweep-SHA, and does not warn when they differ; the "committed at its code SHA / reproduces point-address" criterion therefore holds only for the coincident-HEAD case. Record the chosen resolution (warn/refuse on mismatch + carry the SweepSHA in the back-link, vs. explicitly scope promote to "at HEAD").
- **`workshop/plans/000008-shape-run-ledger-plan.md`** — reconcile the `promoted_from` back-link format: plan says `<shape> @ <point-addr>`; code writes `<shape> (k=v, …)` with no SHA/point-addr. Pick one and make code + plan agree.

---

## Re-review — 2026-07-05T21:29:56-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T21:29:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#8's whole-issue close is in genuinely good shape, and the two blocking Criticals the prior close-reviews found are verified-fixed in code, not just in commit messages. `metis promote` now really commits — `gitCLICommitter` (`ledger_cmd.go:137-147`) is injected in `cmdPromote` (`:132`) and `TestPromote_ActuallyCommits` drives the real path against a real repo; the documented `<shape>`-first arg order works via `hoistShapePath` (`:62-77`) with `TestCLI_ArgOrderIndependent` pinning both orders; the round-2 Importants are also closed — `runPromote` now warns loudly when the selected row's sweep-SHA ≠ HEAD (`:201-203`) and `renderPromoted` records full provenance `@ <point-addr> (sweep <sha>) <free-params>` (`:282`). `pkg/ledger` remains a textbook pure core (ARCH-PURE holds), the side-ref capture provably commits recoverable bytes to `refs/metis/sweeps/*`, and `go build`/`go vet`/`go test ./pkg/ledger ./cmd/metis` are all green. No finding rises to Critical, so the gate passes. What keeps it from a clean SHIP are two Important gaps that survived the rework — `metis ledger show`'s rendering/sort/filter is still completely untested, and the namespaced-objective requirement (the exact thing the fixture got wrong) is undocumented in the datatype doc — plus a handful of Minors. All are cheap; none block the boundary.

### 1. Strengths

- **The prior Criticals are fixed with regression-catching tests, not papered over.** `TestPromote_ActuallyCommits` asserts the *side effect* the success message claims (committed `champ.md` + a `promote` log entry) against a real repo — exactly the "don't let the success print outrun the action" lesson (`workshop/lessons.md`). `TestCLI_ArgOrderIndependent` drives the real `cmdLedger` entrypoint in the documented order. Good closure of feedback.
- **`pkg/ledger` is a clean pure core, tested without IO** (`pkg/ledger/ledger.go`): append-only dedup-by-point-address with lazy `seen` rebuild (`:43`), ragged union columns over stdlib `encoding/csv`, objective-driven `Best`/`TopN` skipping failed/metric-missing with stable tie-break — all exercised with struct literals, zero disk. Textbook ARCH-PURE.
- **`promotedExperiment` reconstructs by re-derivation** (`cmd/metis/ledger.go:66`), reusing `shape.Expand` + `shapePointToExperiment` rather than inverting a flat map; `freeParamsEqual` (`:83`) compares via canonical JSON so int-vs-float64 CSV drift can't break matching, and the round-trip e2e now asserts the exact reproduced `PointAddress` (`ledger_e2e_test.go:161`), not just `status==ok`. ARCH-DRY done right.
- **The side-ref capture is honestly tested with real git** (`capture.go`, `capture_test.go`): the throwaway `GIT_INDEX_FILE` tree build leaves the real index/worktree untouched, and `TestCaptureClosure_DirtyFile` proves the captured blob `cat-file`s to the exact dirty bytes and `git checkout` restores them; `TestCaptureClosure_CleanIsHead` proves a clean closure writes no ref.
- **Record-doc + atlas track the delivered behavior** — `pkg/record/record.go:29-43` now says D/Commit are backfilled by #8's capture (Deps re-scoped post-v1), and `atlas/index.md` documents the fixed positional-first commands.

### 2. Critical findings

None. The two blocking Criticals from the prior close-reviews (promote never commits; documented arg order errors) are verified fixed in code and pinned by new tests; the full suite is green.

### 3. Important findings

- **`metis ledger show` output is untested and unroutable to a buffer** — `renderLedger(out *os.File, …)` (`cmd/metis/ledger_cmd.go:80`) takes a concrete `*os.File`, so the table (sort order, sweep-filter, union columns) can't be asserted against a buffer; `TestCLI_ArgOrderIndependent` only checks "no error." A user-facing subcommand ships with its entire rendering path unexercised — and it prints **no header row** (`:107` joins bare values), so column identity is unverifiable too. *Failure scenario:* a `--sort`/`--dir` regression (e.g. minimize sorted as maximize) or a dropped column ships green because nothing asserts the rendered rows. *Fix:* widen `renderLedger` to `io.Writer` and add a table test over `cmdLedger` asserting the rendered rows for `--sort`/`--sweep`/`--top` (and consider emitting a header line).
- **The namespaced-objective requirement is undocumented where a user writes it** — rows carry `<step>.<metric>` (e.g. `train.cv_score`) and both `promote --best` and the body top-N look up `sh.Sweep.Objective.Metric` verbatim, but `construct/datatype/experiment-shape.md:45` still documents `objective` as plain `{metric, direction}`. This is the exact class of bug the committed fixture shipped (`cv_score` → had to be fixed to `train.cv_score`). *Failure scenario:* a reader authors `objective: {metric: cv_score}`, the sweep succeeds, and the top-N summary silently emits the (now-loud, but only on stderr) warning while `promote --best` returns "no qualifying row." *Fix:* add one line to the datatype doc stating the objective metric must be namespaced `<step>.<metric>`.

### 4. Minor findings

- `gitCLICommitter.Commit` runs `git commit -m` with **no pathspec** (`ledger_cmd.go:144-145`) — commits the whole staged index, so any pre-staged unrelated change is swept into the "metis promote" commit, undercutting the "single self-contained" promise. Prefer `git commit -m … -- <file>` (only `git add`'d file is normally staged, so this bites only with pre-staged state — still trivial to harden).
- `hoistShapePath` (`ledger_cmd.go:62`) grabs **any** `.md`-suffixed non-flag token, including a flag *value* — `metis promote shape.md --name winner.md` errors with "want exactly one <shape.md>, got multiple" instead of writing `winner.md`. Documented usage uses bare names, so it works, but the token scan is order-blind to flag values.
- **ARCH-DRY:** `ledgerParseCell` (`ledger_cmd.go:312`) duplicates `pkg/ledger.parseCell` (`ledger.go:247`) **minus** the list/map JSON branch, so a `--point` selector can't match a list-/`$oneof`-valued free-param (and comma-splitting can't express one anyway). Export/reuse one parser; note the scalar-only `--point` limit.
- `ledger show --dir` defaults to `maximize` (`ledger_cmd.go:28`) and ignores the shape's declared objective direction — `--sort train.loss` on a minimize objective silently maximizes. Default `--dir` from `sh.Sweep.Objective.Direction`.
- Re-promoting the same `--name` with byte-identical content stages nothing → `git commit` errors "nothing to commit" **after** `winner.md` is rewritten (`ledger_cmd.go:224-228`). Degrade gracefully (treat no-op as success).
- Round-trip asserts point-address but **not metrics** (`ledger_e2e_test.go:160`); Done-when (issue:154) says "point-address + metrics." Cheap to add.
- `backfillCodeManifest` sets **every** step's `Code.D` to the whole-sweep closure union (`capture.go:148-151`) — coarser than the per-step `reads.json` on disk; imprecise as per-step provenance (matches plan wording, note-only).
- `captureClosure` hardcodes mode `100644` in `update-index --cacheinfo` (`capture.go:61`) — an executable closure file loses its exec bit on `git checkout` recovery. Edge (Python is 0644).
- `sweepSHAOf` returns an arbitrary map entry when `RepoSHAs` has >1 repo (`ledger.go:54`) — non-deterministic beyond v1's single repo; acknowledged in the comment.
- The ledger sidecar is **written but never committed** by the sweep (`writeSweepLedger`), while the Design says the CSV is "committed batched (per-sweep)." The M2 Done-when doesn't require it (the sidecar is a reconstructable view over the durable records), so this is a Design-vs-delivered wording gap — reconcile the Design text or wire a batched commit.

### 5. Test coverage notes

Pure-core coverage is excellent and pins real properties (dedup, ragged round-trip incl. list free-params, `Best`/`TopN`, immutability-by-snapshot). The rework closed the two big gaps (real commit path; documented arg order; point-address reproduction). Remaining gaps that map to shipped-bug risk: (a) **`metis ledger show` rendering/sort/filter — zero assertion coverage** (Important #1); (b) round-trip asserts point-address but not metrics; (c) cross-version `promote --best` warning has no test where HEAD ≠ the selected row's `SweepSHA` (the code path is now present but unexercised); (d) the sweep→capture wiring fires only in the isolated `capture_e2e_test.go` (the ledger e2es use `cache:false`, so capture is a no-op there) — the two never run end-to-end together.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `freeParamTuple` delegates to `freeParamTupleMap`; `promotedExperiment` reuses `Expand`+`shapePointToExperiment`; the codec reuses stdlib. Only the `ledgerParseCell`/`parseCell` split remains (Minor).
- **ARCH-PURE — pass.** `pkg/ledger` and the pure helpers (`rowsFromManifest`, `namespacedMetrics`, `promotedExperiment`, `freeParamsEqual`) are deterministic and unit-tested without IO; `writeSweepLedger`/`loadLedger`/`captureClosure`/`runPromote` are the thin injected shell.
- **ARCH-PURPOSE — pass (with a scoped residual).** Shadow-sweep on "the ledger is the single source": `show`, `promote`, and the body top-N all derive from the sidecar — good; the prior blocking gap (promote wrote-but-didn't-commit) is closed and the cross-version case is now surfaced. The one residual is dirty-run fidelity: the ledger's `SweepSHA`/back-link is **HEAD-based** (`sweepSHAOf` reads `RepoSHAs`, not the captured commit), so for a dirty run `git checkout <back-link sweep-SHA>` restores HEAD's code, not the captured recoverable version — reproduction of a dirty row needs the `refs/metis/sweeps/*` commit that M3 captured into `CodeManifest.Commit`. This is in the documented v1 scope (precise dirty-identity = metis#10); worth a `## Log` line so it isn't later mistaken for a bug. Same axis: two dirty iterations at one HEAD share a point-address, so the second dedups away.

### 7. Plan revision recommendations

- **`workshop/plans/000008-shape-run-ledger-plan.md`** — the two prior close-reviews recommended `## Revisions` entries reconciling (a) delivered promote semantics (commits at **HEAD**, warns when it differs from the row's sweep-SHA) and (b) the `promoted_from` format (plan says `<shape> @ <point-addr>`; code writes `<shape> @ <point-addr> (sweep <sha>) (k=v,…)`). The code now matches the *richer* format and the warning is in place — so update the plan text to the delivered format (`@ <point-addr> (sweep <sha>) <free-params>`) so the plan stops claiming the plainer form.
- **Design vs. delivered — batched commit:** add a `## Revisions` note that the sweep writes but does not commit the CSV sidecar (the Design's "committed batched (per-sweep)" is not wired; the sidecar is a reconstructable view, so nothing irreplaceable is at risk), or scope a follow-up if auto-commit is still wanted.
- No Core-concepts-table cross-check applies: the plan uses per-milestone prose rather than a greppable name/kind/status table; all named entities (`pkg/ledger`, `cmd/metis/ledger.go`/`ledger_cmd.go`/`capture.go`, `CodeManifest.D`/`Commit`) exist at their stated paths and are verified above.

---

## Re-review — 2026-07-05T21:39:07-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T21:39:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have enough to render the review. Build/vet/full-suite are green, and I've traced the git plumbing, the sweep→capture→ledger wiring, the promote path, and the read-set closure paths against the Spec/Plan/Done-when.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#8 lands a genuinely strong pure ledger core (`pkg/ledger`) and an honestly-tested git-side-ref capture, and the three prior close-review Criticals (promote never committed; documented arg order errored; silently-empty objective) are verified-fixed in code and pinned by regression tests — `go build`/`go vet`/`go test ./...` all green. `pkg/ledger` is a textbook pure core (append-only dedup, ragged CSV codec, objective `Best`/`TopN`/`Filter`), all unit-tested without IO (ARCH-PURE holds); `promotedExperiment` reconstructs by re-derivation reusing `shape.Expand`+`shapePointToExperiment` (ARCH-DRY), and the round-trip e2e asserts the reproduced point-address. Nothing I found rises to Critical, so the gate passes. What keeps it from a clean SHIP: (1) `metis ledger show --sort` silently drops failed and metric-missing rows (it routes through `TopN`, which filters), with no test over that path — a user auditing a sweep would not see failed points; (2) capture is *fatal* to an already-completed sweep, inconsistent with the best-effort git-absent path; and a documented-but-unlogged dirty-run reproducibility residual. All are cheap; none block the boundary.

### 1. Strengths

- **`pkg/ledger` is a clean pure core, tested without IO** (`pkg/ledger/ledger.go`): append-only dedup-by-point-address with lazy `seen` rebuild (`:43`), ragged union columns over stdlib `encoding/csv`, objective-driven `Best`/`TopN` skipping failed/metric-missing with a stable strict tie-break. Exercised with struct literals, zero disk.
- **The prior Criticals are fixed with side-effect-asserting tests, not papered over.** `TestPromote_ActuallyCommits` drives the real `cmdPromote`/`gitCLICommitter` path against a real repo and asserts `champ.md` is *committed* + a promote log entry (`ledger_e2e_test.go`); `TestCLI_ArgOrderIndependent` drives the real `cmdLedger` entrypoint in the documented `<shape>`-first order via `hoistShapePath` (`ledger_cmd.go:68`). Good closure of feedback and the `lessons.md` "don't let the success print outrun the action" rule.
- **The side-ref capture is honestly tested with real git** (`capture.go`, `capture_test.go`): the throwaway `GIT_INDEX_FILE` tree build (`capture.go:54-64`) leaves the real index/worktree untouched, and `TestCaptureClosure_DirtyFile` proves the captured blob `cat-file`s to the exact dirty bytes and `git checkout` restores them; `TestCaptureClosure_CleanIsHead` proves a clean closure writes no ref.
- **`promotedExperiment` reconstructs by re-derivation** (`cmd/metis/ledger.go:66`), and `freeParamsEqual` (`:83`) compares via canonical JSON so int-vs-float64 CSV drift can't break matching — the immutability-by-snapshot property is a real test (`TestLedger_PriorRowsReproduceAfterSpaceEdit`), not a mock.
- **Docs gate satisfied:** `atlas/index.md` and `construct/datatype/experiment-shape.md` (the namespaced-objective requirement) are both updated in-range; no README exists in the repo, so no README surface to update.

### 2. Critical findings

None. The three blocking Criticals from the prior close-reviews are verified fixed in code and pinned by new tests; the full suite is green.

### 3. Important findings

- **`metis ledger show --sort <metric>` silently drops failed and metric-missing rows** — `cmd/metis/ledger_cmd.go:50-58` routes a sorted show through `ledger.TopN` (`pkg/ledger/ledger.go:165-176`), which skips `status=="failed"` rows and rows lacking the metric. So the un-sorted `show` renders all rows (failed included, append order), but the moment you add `--sort` the table quietly omits failures. `TestLedgerShow_RendersSortedTable` uses three `ok` rows, so this path is uncovered.
  - *Failure scenario:* `metis ledger show titanic.md --sort train.cv_score` on a 21-point sweep where 4 points failed renders 17 rows with no indication the other 4 exist — a user auditing the sweep concludes everything ran. Data is still in the CSV, but the primary query view hides it.
  - *Fix:* either append the non-comparable rows after the ranked ones (blank metric cell), or document that `--sort` restricts to comparable rows; add a test with a failed row present under `--sort`.
- **Inconsistent error handling: capture is fatal to an already-finished sweep, but the git-absent path is best-effort** — `cmd/metis/sweep.go:122-124` returns any `captureSweepCode` error, aborting `runSweep` *after* the manifest and all per-point records are written and valid — and before `writeSweepLedger` runs, so the ledger aggregation is also lost. `captureSweepCode` itself carefully degrades to a no-op when git is missing / not a repo / no closure (`capture.go:87-95`), but a `hash-object`/`commit-tree`/`update-ref` hiccup (e.g. a repo with no commit identity, a read-only object store) hard-fails the whole run.
  - *Failure scenario:* a sweep completes all 21 points, `commit-tree` fails because the environment has no `user.email` set → `metis run` exits non-zero and the shape's ledger is never written, even though every point succeeded and is recorded.
  - *Fix:* make a git-present capture failure best-effort too (warn + continue to `writeSweepLedger`), consistent with the git-absent branch and the "the sweep already ran, capture just doesn't apply" comment — or if a fatal capture failure is intentional, run capture *before* declaring the points recorded and say so.

### 4. Minor findings

- **Dirty-run back-link points at a non-recoverable SHA (documented v1 residual, but not logged).** The ledger row's `SweepSHA` = `rec.RepoSHAs` HEAD sha (`ledger.go:54`), and `renderPromoted` writes `promoted_from: … (sweep <headSHA>)` (`ledger_cmd.go:288`) — *not* the captured `refs/metis/sweeps/*` commit in `CodeManifest.Commit`. For a dirty run, `git checkout <back-link sweep-SHA>` restores clean HEAD, not the captured dirty code; and two dirty iterations at one HEAD share a point-address so the second dedups away. In scope for v1 (precise dirty-identity = metis#10) but absent from the issue Done-when/Log as a residual — add a `## Log` line so it isn't later mistaken for a bug. (ARCH-PURPOSE partial — see §6.)
- **`gitCLICommitter.Commit` runs `git commit -m` with no pathspec** (`ledger_cmd.go:150-152`) — commits the whole staged index, so any pre-staged unrelated change is swept into the "metis promote" commit, undercutting "single self-contained." Prefer `git commit -m … -- <file>`.
- **Re-promoting the same `--name` with byte-identical content errors** — `git add` stages nothing, `git commit` fails "nothing to commit" *after* `winner.md` was rewritten (`ledger_cmd.go:230-235`), so `runPromote` returns an error post-write. Treat the no-op as success.
- **ARCH-DRY:** `ledgerParseCell` (`ledger_cmd.go:318`) duplicates `pkg/ledger.parseCell` (`ledger.go:247`) minus the list/map JSON branch, so a `--point` selector can't match a list-/`$oneof`-valued free-param (and comma-splitting can't express one). Export/reuse one parser; note the scalar-only `--point` limit.
- **`ledger show --dir` defaults to `maximize`** (`ledger_cmd.go:29`) and ignores the shape's declared objective direction — `--sort train.loss` on a minimize objective silently maximizes. Default `--dir` from `sh.Sweep.Objective.Direction`.
- **Round-trip asserts point-address but not metrics** (`TestLedger_PromoteBestRoundTrips`); Done-when (issue:154) says "point-address + metrics." Largely implied by address reproduction for a deterministic run, but cheap to add.
- **`hoistShapePath` grabs any `.md`-suffixed non-flag token, including a flag *value*** (`ledger_cmd.go:70`) — `--name winner.md` or `--sweep x.md` would be misread as the positional. Documented usage uses bare names, so it works, but the scan is order-blind to flag values.
- **Cross-version promote warning is untested** (`ledger_cmd.go:207-209`) — no test exercises HEAD ≠ selected-row `SweepSHA`; a regression that drops the warning ships green.
- **`captureClosure` hardcodes mode `100644`** in `update-index --cacheinfo` (`capture.go:61`) — an executable closure file loses its exec bit on `git checkout` recovery. Edge (Python is 0644).
- **`sweepSHAOf` returns an arbitrary map entry** when `RepoSHAs` has >1 repo (`ledger.go:54`) — non-deterministic beyond v1's single repo; acknowledged in the comment.

### 5. Test coverage notes

Pure-core coverage is excellent and pins real properties (dedup, ragged round-trip incl. list free-params, `Best`/`TopN`, immutability-by-snapshot). The rework closed the big gaps (real commit path; documented arg order; point-address reproduction). Remaining gaps that map to shipped-bug risk: (a) **`ledger show --sort` with a failed/metric-missing row present** — the Important above; (b) the cross-version promote warning path; (c) round-trip metrics reproduction; (d) the sweep→capture wiring fires only in the isolated `capture_e2e_test.go` (the ledger e2es use `cache:false`, so capture is a no-op there) — the two never run end-to-end together.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `freeParamTuple` delegates to `freeParamTupleMap`; `promotedExperiment` reuses `Expand`+`shapePointToExperiment`; the codec reuses stdlib. Only the `ledgerParseCell`/`parseCell` split remains (Minor).
- **ARCH-PURE — pass.** `pkg/ledger` and the pure helpers (`rowsFromManifest`, `namespacedMetrics`, `promotedExperiment`, `freeParamsEqual`) are deterministic and unit-tested without IO; `writeSweepLedger`/`loadLedger`/`captureClosure`/`runPromote` are the thin injected shell.
- **ARCH-PURPOSE — pass, with one scoped residual.** Shadow-sweep on "the ledger is the single navigable/promotable source": `show`, `promote`, and the body top-N all derive from the sidecar — good; capture populates `CodeManifest.D`/`Commit`. The residual is dirty-run fidelity (Minor #1): M3 captures the recoverable commit into `CodeManifest.Commit`, but the L1 navigation layer (ledger `SweepSHA` + promote back-link) points at HEAD, so the "every row is recoverable" purpose is under-wired for dirty rows — the recorded sweep-SHA can't reach the captured code. Documented as metis#10; worth a Log line so the deferral is explicit rather than looking like a bug.

### 7. Plan revision recommendations

The plan (`workshop/plans/000008-shape-run-ledger-plan.md`, added in this range) has no `## Revisions` section and its M2 text still describes the pre-delivery design; reconcile with what shipped:
- Add a `## Revisions` entry recording the delivered `promoted_from` format (`<shape> @ <point-addr> (sweep <sha>) (k=v,…)`), superseding the plan's plainer `<shape> @ <point-addr>`, and that `promote` commits at **HEAD** and *warns* when HEAD ≠ the selected row's sweep-SHA (rather than committing at the row's code SHA).
- Add a `## Revisions` note that the sweep **writes but does not commit** the CSV sidecar (the Design's "committed batched (per-sweep)" is not wired; the sidecar is a reconstructable view, so nothing irreplaceable is at risk) — or scope a follow-up if auto-commit is still wanted.
- Add a `## Revisions` note on the dirty-run residual (Minor #1 / §6): the ledger `SweepSHA` and back-link are HEAD-based, so dirty-run recovery requires `CodeManifest.Commit`, not the recorded sweep-SHA — the L1 layer doesn't yet carry the captured commit (metis#10).

---

## Re-review — 2026-07-05T21:47:02-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T21:47:02-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have verified everything I need: build/vet/full-suite green, the two prior REWORK Criticals and the round-4 Importants are genuinely fixed in code with regression tests, and I've traced the promote/round-trip, capture, and ledger paths against the Spec/Plan/Done-when. Here is my fresh review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#8 delivers a genuinely strong pure ledger core (`pkg/ledger`) and an honestly-tested git-side-ref capture, and I independently confirmed the fixes the prior close-reviews claimed are real, not just asserted in commit messages: `metis promote` now actually commits (`gitCLICommitter` injected in `cmdPromote:135`, pinned by `TestPromote_ActuallyCommits`), the documented `<shape>`-first arg order works (`hoistShapePath` + `TestCLI_ArgOrderIndependent`), `ledger show --sort` keeps failed/metric-missing rows via `SortAll` (tested), and capture is now best-effort (`sweep.go:124` warns, doesn't abort). `go build`/`go vet`/`go test ./...` all green. ARCH-PURE and ARCH-DRY hold cleanly. Nothing I found rises to Critical, so the gate passes. What keeps it from a clean SHIP is one real UX-correctness gap on the headline query command — `ledger show --sort` defaults `--dir maximize` regardless of the shape's declared objective, so a `minimize` objective (e.g. loss) sorts *worst-first* by default — plus a promote-fidelity test gap (the cross-version warning path and the round-trip's metric reproduction are both unasserted). Both are cheap and non-blocking.

### 1. Strengths

- **`pkg/ledger` is a textbook pure core, unit-tested without IO** (`pkg/ledger/ledger.go`): append-only dedup-by-point-address with lazy `seen` rebuild (`:44`), ragged union columns over stdlib `encoding/csv`, objective-driven `Best`/`TopN`/`SortAll` with stable strict tie-break. Exercised with struct literals, zero disk. ARCH-PURE textbook.
- **The prior Criticals are fixed with side-effect-asserting tests, not papered over.** `TestPromote_ActuallyCommits` drives the real `cmdPromote`/`gitCLICommitter` path against a real repo and asserts `champ.md` is *committed* + a `promote` log entry — exactly the "don't let the success print outrun the action" lesson. `TestCLI_ArgOrderIndependent` drives the real `cmdLedger` entrypoint in the documented order.
- **The side-ref capture is honestly tested with real git** (`capture.go`, `capture_test.go`): the throwaway `GIT_INDEX_FILE` tree build (`capture.go:54-64`) leaves the real index/worktree untouched, and `TestCaptureClosure_DirtyFile` proves the captured blob `cat-file`s to the exact dirty bytes and `git checkout` restores them; `TestCaptureClosure_CleanIsHead` proves a clean closure writes no ref.
- **`promotedExperiment` reconstructs by re-derivation** (`cmd/metis/ledger.go:66`), reusing `shape.Expand`+`shapePointToExperiment` rather than inverting a flat map, and `freeParamsEqual` (`:83`) compares via canonical JSON so int-vs-float64 CSV drift can't break matching; the round-trip e2e asserts the exact reproduced `PointAddress`. I traced the type-drift chain (yaml-int → `cell` → `parseCell` → JSON-normalized) and it holds for the realistic scalar/list free-param cases.
- **Docs gate satisfied:** `atlas/index.md` gets a full `pkg/ledger` paragraph documenting the fixed positional-first commands; `construct/datatype/experiment-shape.md:46-49` now documents the namespaced-objective requirement (the exact trap the fixture hit). No `README.md` exists in the repo → no README surface to update.

### 2. Critical findings

None. The three blocking Criticals from the prior close-reviews (promote never commits; documented arg order errors; silently-empty objective) are verified fixed in code and pinned by new tests; the full suite is green.

### 3. Important findings

- **`ledger show --sort` ignores the shape's objective direction and defaults to `maximize`** — `cmd/metis/ledger_cmd.go:29` (`direction := fs.String("dir", "maximize", …)`), consumed at `:51`. `cmdLedger` reads only the ledger CSV; it never parses the shape's `sweep.objective.direction`, so `metis ledger show titanic.md --sort train.loss` on a `minimize` objective silently sorts **descending** (worst-first) unless the user remembers `--dir minimize`. The whole purpose of `show` is "pick the winner by sorting," and for every minimize objective the default inverts that.
  - *Failure scenario:* an engineer sweeps a loss-minimizing shape, runs `ledger show shape.md --sort train.loss` to eyeball the best config, and the top row is the *worst* run — a silent-wrong-order on the headline command.
  - *Fix:* parse the shape once in `cmdLedger`/`showLedger` and default `--dir` from `sh.Sweep.Objective.Direction` when `--dir` wasn't explicitly passed (keep the flag as an override).
- **Promote-fidelity guarantees are under-tested — a regression would ship green.** Two Done-when-adjacent behaviors have no assertion: (a) the cross-version warning (`ledger_cmd.go:204-206`, warn when the selected row's `SweepSHA` ≠ HEAD) is exercised by no test — a change that drops the warning passes green, and the warning is the *only* thing protecting the "committed at its code SHA / reproduces the row" contract for an older-winner promote; (b) `TestLedger_PromoteBestRoundTrips` asserts the reproduced `PointAddress` but not the metrics, though Done-when (issue:154) says "point-address **+ metrics**." *Fix:* add a test where HEAD ≠ the selected row's `SweepSHA` asserting the warning fires, and assert the round-trip run's metrics equal the promoted row's.

### 4. Minor findings

- `gitCLICommitter.Commit` runs `git commit -m` with **no pathspec** (`ledger_cmd.go:147-149`) — commits the whole staged index, so any pre-staged unrelated change is swept into the "metis promote" commit, undercutting "single self-contained." Prefer `git commit -m … -- <file>`.
- Re-promoting the same `--name` with byte-identical content errors — `git add` stages nothing, `git commit` fails "nothing to commit" *after* `<name>.md` was rewritten (`ledger_cmd.go:227-232`), so `runPromote` returns an error post-write. Treat the no-op as success.
- **ARCH-DRY:** `ledgerParseCell` (`ledger_cmd.go:315`) duplicates `pkg/ledger.parseCell` (`ledger.go:267`) minus the list/map JSON branch, so a `--point` selector can't match a list-/`$oneof`-valued free-param (and comma-splitting can't express one). Export/reuse one parser; note the scalar-only `--point` limit.
- `ledger show --sort <metric>` with a metric matching **no** row (typo / un-namespaced) silently renders unsorted output with no warning (`SortAll` funnels all rows to `rest`) — pairs with the objective-namespacing trap; a one-line warning would help.
- `hoistShapePath` grabs **any** `.md`-suffixed non-flag token, including a flag *value* (`ledger_cmd.go:66-73`) — `--name winner.md` would be misread as the positional. Documented usage uses bare names, so it works; the scan is order-blind to flag values.
- `backfillCodeManifest` sets **every** step's `Code.D` to the whole-sweep closure union (`capture.go:148-151`) — coarser than the per-step `reads.json` on disk; fine for recovery, imprecise as per-step provenance (matches plan wording, note-only).
- `captureClosure` hardcodes mode `100644` in `update-index --cacheinfo` (`capture.go:61`) — an executable closure file loses its exec bit on `git checkout` recovery. Edge (Python is 0644).
- `sweepSHAOf` returns an arbitrary map entry when `RepoSHAs` has >1 repo (`cmd/metis/ledger.go:54`) — non-deterministic beyond v1's single repo; acknowledged in the comment.
- The `.ledger.csv` sidecar is **written but never committed** by the sweep (`writeSweepLedger`), while the Design says the CSV is "committed batched (per-sweep)." The sidecar is a reconstructable idempotent view, so nothing irreplaceable is at risk — a Design-vs-delivered wording gap; reconcile the plan text or wire a batched commit.

### 5. Test coverage notes

Pure-core coverage is excellent and pins real properties (dedup, ragged round-trip incl. list free-params, `Best`/`TopN`/`SortAll`, immutability-by-snapshot). The rework closed the big gaps (real commit path; documented arg order; point-address reproduction; `--sort` keeps failed rows). Remaining gaps that map to shipped-bug risk: (a) the cross-version `promote --best` warning path — code present, unexercised (Important); (b) round-trip metrics reproduction — unasserted despite the Done-when; (c) `ledger show --dir` default-vs-objective — untested; (d) capture-best-effort (`sweep.go:124`) — the new warn-and-continue branch has no failure-injection test; (e) the sweep→capture wiring fires only in the isolated `capture_e2e_test.go` (the ledger e2es use `cache:false`, so capture is a no-op there) — the two never run end-to-end together.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `freeParamTuple` delegates to `freeParamTupleMap`; `promotedExperiment` reuses `Expand`+`shapePointToExperiment`; codec reuses stdlib. Only the `ledgerParseCell`/`parseCell` split remains (Minor).
- **ARCH-PURE — pass.** `pkg/ledger` and the pure helpers (`rowsFromManifest`, `namespacedMetrics`, `promotedExperiment`, `freeParamsEqual`) are deterministic and unit-tested without IO; `writeSweepLedger`/`loadLedger`/`captureClosure`/`runPromote` are the thin injected shell.
- **ARCH-PURPOSE — pass, with two scoped residuals.** Shadow-sweep on "the ledger is the single navigable/promotable source": `show`, `promote`, and the body top-N all derive from the sidecar — good; capture populates `CodeManifest.D`/`Commit`. Residual 1 (dirty-run fidelity, documented metis#10): the ledger's `SweepSHA` + promote back-link are **HEAD-based** (`sweepSHAOf` reads `RepoSHAs`, not the captured `refs/metis/sweeps/*` commit in `CodeManifest.Commit`), so `git checkout <back-link sweep-SHA>` restores clean HEAD, not the captured dirty code — worth a `## Log` line so it isn't later mistaken for a bug. Residual 2 (forward, for kbench#4): capture's `root` is the *code* repo (`cacheProjectRoot`, above `steps/`) while promote commits into the *experiment* repo (`filepath.Dir(shapePath)`) and the ledger SweepSHA is the experiment repo's HEAD — coincident under v1's single-repo assumption, but the "committed at its code SHA" story needs an explicit decision once code and experiment live in different repos.

### 7. Plan revision recommendations

The plan (`workshop/plans/000008-shape-run-ledger-plan.md`) has no `## Revisions` section and its M2 text still describes the pre-delivery design; the prior close-reviews recommended these and they remain unmade:
- Add a `## Revisions` entry recording the delivered promote semantics: commits at **HEAD** and *warns* when HEAD ≠ the selected row's sweep-SHA (superseding the plan's "commit it at the code SHA"), and the `promoted_from` back-link format is the richer `<shape> @ <point-addr> (sweep <sha>) (k=v,…)` (superseding the plan's plainer `@ <point-addr>`).
- Add a `## Revisions` note that the sweep **writes but does not commit** the CSV sidecar (the Design's "committed batched (per-sweep)" is unwired; the sidecar is a reconstructable view) — or scope a follow-up if auto-commit is still wanted.
- Add a `## Revisions` note on the dirty-run residual (§6): the ledger `SweepSHA`/back-link are HEAD-based, so dirty-run recovery requires `CodeManifest.Commit`, not the recorded sweep-SHA (metis#10).

No Core-concepts-table cross-check applies — the plan uses per-milestone prose; all named entities (`pkg/ledger`, `cmd/metis/ledger.go`/`ledger_cmd.go`/`capture.go`, `CodeManifest.D`/`Commit`) exist at their stated paths and are verified above.

---

## Re-review — 2026-07-05T21:56:32-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T21:56:32-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Build/vet/`go test ./pkg/ledger ./cmd/metis` are green at HEAD (`d5a9141`), I've traced the promote/capture/ledger paths and the SHA-format consistency of the cross-version guard against the actual source, and confirmed the docs-gate state (no README exists; atlas + datatype doc updated). Here is my fresh review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#8's whole-issue close is genuinely done: I independently verified every Done-when item is delivered and pinned by a test that would catch its regression, and the three blocking Criticals from earlier rounds (promote never committed; documented arg order errored; silently-empty objective) are fixed in code, not just in commit messages. `metis promote` really commits (`gitCLICommitter` injected at `ledger_cmd.go:132`, `TestPromote_ActuallyCommits` asserts the committed file + log entry against a real repo); `hoistShapePath` makes the documented `<shape>`-first order work; `ledger show --sort` keeps failed/metric-missing rows via `SortAll` and defaults its direction from the shape's objective; the round-trip e2e asserts the reproduced point-address **and** metrics; the side-ref capture provably commits recoverable bytes and is now best-effort (a git hiccup warns, doesn't abort a finished sweep). `pkg/ledger` is a textbook pure core — ARCH-PURE and ARCH-DRY both hold. Nothing I found rises to Critical, so the gate passes. What keeps it from a clean SHIP: the plan still asserts a promote semantics the code doesn't deliver (commits at HEAD + warns, not "at its code SHA") and has no `## Revisions` section — a boundary-docs discipline the plan violates; plus round-5's default-`--dir`-from-objective fix ships with no direct test. Both are cheap and non-blocking.

### 1. Strengths

- **`pkg/ledger` is a clean pure core, unit-tested without IO** (`pkg/ledger/ledger.go`): append-only dedup-by-point-address with lazy `seen` rebuild (`:43`), ragged union columns over stdlib `encoding/csv`, objective-driven `Best`/`TopN`/`SortAll` with a stable strict tie-break. Exercised with struct literals, zero disk.
- **The prior Criticals are fixed with side-effect-asserting tests, not papered over.** `TestPromote_ActuallyCommits` asserts the *committed file* the success message claims — exactly the `lessons.md` "don't let the success print outrun the action" rule; `TestCLI_ArgOrderIndependent` drives the real `cmdLedger` entrypoint in the documented order. I re-ran the suite: green.
- **The cross-version guard is format-consistent and tested.** Both `row.SweepSHA` (`sweepSHAOf` → `rec.RepoSHAs`) and `probeRepo`'s HEAD sha derive from the same `git.Probe` source (`record.go:110/117`, `sweep.go:178`), so the `row.SweepSHA != headSHA` comparison at `ledger_cmd.go:214` can't false-fire on a short-vs-full mismatch; `TestPromote_WarnsOnCrossCodeVersion` drives it with real git.
- **The side-ref capture is honestly tested with real git** (`capture.go`, `capture_test.go`): the throwaway `GIT_INDEX_FILE` tree build leaves the real index/worktree untouched, and `TestCaptureClosure_DirtyFile` proves the captured blob `cat-file`s to the exact dirty bytes and `git checkout` restores them; a clean closure writes no ref.
- **`promotedExperiment` reconstructs by re-derivation** (`cmd/metis/ledger.go:66`), reusing `shape.Expand`+`shapePointToExperiment`, and `freeParamsEqual` compares via canonical JSON so int-vs-float64 CSV drift can't break matching; `TestLedger_PriorRowsReproduceAfterSpaceEdit` pins the immutability-by-snapshot property against a real space edit.

### 2. Critical findings

None. The three blocking Criticals from the prior close-reviews are verified fixed in code and pinned by new tests; the full suite is green.

### 3. Important findings

- **The plan drifted from the delivered promote semantics and has no `## Revisions` section** — `workshop/plans/000008-shape-run-ledger-plan.md:94` still reads "**commit it at the code SHA** (warn if dirty)," but the code commits the promoted experiment at **HEAD** and only *warns* when the selected row's sweep-SHA ≠ HEAD (`ledger_cmd.go:214-217`) — the design's deliberate "go back." The `promoted_from` back-link is also the richer `@ <point-addr> (sweep <sha>) (k=v,…)` (`ledger_cmd.go:296`), not the plan's plainer `@ <point-addr>`, and the sidecar is written-but-never-committed vs the Design's "committed batched (per-sweep)." AGENTS.md §1 requires a `## Revisions` append when a plan diverges mid-stream; all five prior close-rounds recommended it and it's still absent (`grep -c "## Revisions"` → 0). *Fix:* append a `## Revisions` entry reconciling (a) commit-at-HEAD-plus-warn vs "at the code SHA," (b) the richer back-link format, (c) sidecar written-not-committed. This is doc-only — no code change — but it stops the plan asserting behavior the code doesn't perform.
- **Round-5's default-`--dir`-from-objective fix ships untested** — `cmdLedger` reads `sh.Sweep.Objective.Direction` to default the sort direction (`ledger_cmd.go:33-42`), the fix for "`--sort train.loss` on a minimize objective sorts worst-first." But every test passes direction explicitly: `TestLedgerShow_RendersSortedTable` calls `showLedger(..., "maximize"/"minimize", ...)` directly, and `TestCLI_ArgOrderIndependent` exercises `cmdLedger` only under a *maximize* objective checking no-error. So the defaulting-from-a-minimize-objective path — the exact thing round-5 fixed — has zero assertion coverage. *Failure scenario:* a future refactor reads the wrong field (or drops the default), a loss-minimizing shape's `ledger show --sort train.loss` renders worst-first, and it ships green. *Fix:* a `cmdLedger` test with a `minimize` objective asserting best-first default order (no `--dir`).

### 4. Minor findings

- `gitCLICommitter.Commit` runs `git commit -m` with **no pathspec** (`ledger_cmd.go:144-146`) — commits the whole staged index, so a pre-staged unrelated change is swept into the "metis promote" commit, undercutting "self-contained." Prefer `git commit -m … -- <file>`.
- Re-promoting the same `--name` with byte-identical content → `git add` stages nothing, `git commit` errors "nothing to commit" *after* `<name>.md` was rewritten (`runPromote`), so it returns an error post-write. Treat the no-op as success.
- **ARCH-DRY / `--point` limits:** `ledgerParseCell` (`ledger_cmd.go`) duplicates `pkg/ledger.parseCell` minus the list/map JSON branch, and `findRow` requires the *full* free-param tuple (a partial subset silently fails with a generic "no ledger row matches" error). So `--point` selects scalar-only, full-tuple points; a point with a list/`$oneof` free-param is unreachable except via `--best`. Export/reuse one parser and note the limit.
- `hoistShapePath` grabs any `.md`-suffixed non-flag token, including a flag *value* (`--name winner.md` → "want exactly one <shape.md>, got multiple"). Documented usage uses bare names, so it works; the scan is order-blind to flag values.
- `captureClosure` hardcodes mode `100644` in `update-index --cacheinfo` (`capture.go:70`) — an executable closure file loses its exec bit on `git checkout` recovery. Edge (Python is 0644).
- `sweepSHAOf` returns an arbitrary map entry when `RepoSHAs` has >1 repo (`ledger.go:54`) — non-deterministic beyond v1's single repo; acknowledged in the comment.
- **Dirty-run residual is undocumented in the issue Log.** The ledger row `SweepSHA` and the `promoted_from` back-link are HEAD-based (`sweepSHAOf` reads `RepoSHAs`, not the captured `refs/metis/sweeps/*` commit in `CodeManifest.Commit`), so recovering a *dirty* row needs `CodeManifest.Commit`, not the recorded sweep-SHA; two dirty iterations at one HEAD also share a point-address and the second dedups away. In-scope for v1 (precise dirty-identity = metis#10), but add a `## Log` line so it isn't later mistaken for a bug — all prior rounds recommended this and it's still unwritten.

### 5. Test coverage notes

Pure-core coverage is excellent and pins real properties (dedup, ragged round-trip incl. list free-params, `Best`/`TopN`/`SortAll`, immutability-by-snapshot). The rework closed the big gaps: real commit path, documented arg order, point-address **and** metrics reproduction, `--sort` keeps failed rows, and the cross-version warning (`TestPromote_WarnsOnCrossCodeVersion`) now has a real-git test. Remaining gaps that map to shipped-bug risk: (a) `cmdLedger` default-`--dir`-from-objective (Important #2); (b) the sweep→capture wiring fires only in the isolated `capture_e2e_test.go` — the ledger e2es use `cache:false`, so `sweep.go`'s `captureSweepCode` hook is a no-op there; the two never run end-to-end together (acceptable, but the wiring line itself is unexercised).

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `freeParamTuple` delegates to `freeParamTupleMap`; `promotedExperiment` reuses `Expand`+`shapePointToExperiment`; the codec reuses stdlib. Only the `ledgerParseCell`/`parseCell` split remains (Minor).
- **ARCH-PURE — pass.** `pkg/ledger` and the pure helpers (`rowsFromManifest`, `namespacedMetrics`, `promotedExperiment`, `freeParamsEqual`) are deterministic and unit-tested without IO; `writeSweepLedger`/`loadLedger`/`captureClosure`/`runPromote` are the thin injected shell. The ledger stays a pure aggregation *view* over #3's records, not a competing store — the plan's "no `## Runs` retrofit" scope held.
- **ARCH-PURPOSE — pass, with one scoped residual.** Shadow-sweep on "the ledger is the single navigable/promotable source": `show`, `promote`, and the body top-N all derive from the sidecar; capture populates `CodeManifest.D`/`Commit`. The residual is dirty-run fidelity (Minor #7): the L1 navigation layer (ledger `SweepSHA` + promote back-link) is HEAD-based while the recoverable commit lives in `CodeManifest.Commit`, so "every row is recoverable" is under-wired for dirty rows — documented as metis#10. Forward (kbench#4): capture's `root` is the *code* repo (`cacheProjectRoot`) while promote commits into the *experiment* repo (`filepath.Dir(shapePath)`) and the ledger SweepSHA is the experiment repo's HEAD — coincident under v1's single-repo assumption, but the "committed at its code SHA" story needs an explicit decision once code and experiment live in different repos.

### 7. Plan revision recommendations

`workshop/plans/000008-shape-run-ledger-plan.md` has no `## Revisions` section and its M2 text still describes the pre-delivery design. Add:
- A `## Revisions` entry recording the delivered promote semantics: `promote` commits at **HEAD** and *warns* when HEAD ≠ the selected row's sweep-SHA (superseding "commit it at the code SHA"), and the `promoted_from` back-link is the richer `<shape> @ <point-addr> (sweep <sha>) (k=v,…)` (superseding `@ <point-addr>`).
- A `## Revisions` note that the sweep **writes but does not commit** the `.ledger.csv` sidecar (the Design's "committed batched (per-sweep)" is unwired; the sidecar is a reconstructable idempotent view, so nothing irreplaceable is at risk) — or scope a follow-up if auto-commit is still wanted.
- A `## Revisions` note on the dirty-run residual: the ledger `SweepSHA`/back-link are HEAD-based, so dirty-run recovery requires `CodeManifest.Commit`, not the recorded sweep-SHA (metis#10).
- No Core-concepts-table cross-check applies — the plan uses per-milestone prose; all named entities (`pkg/ledger`, `cmd/metis/ledger.go`/`ledger_cmd.go`/`capture.go`, `CodeManifest.D`/`Commit`) exist at their stated paths and are verified above.
