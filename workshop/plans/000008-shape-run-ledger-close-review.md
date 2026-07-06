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
