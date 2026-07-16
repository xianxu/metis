# Boundary Review — metis#8 (milestone M2)

| field | value |
|-------|-------|
| issue | 8 — Shape run-ledger: CSV sidecar keyed by free-param tuple + promotion to an experiment |
| repo | metis |
| issue file | workshop/issues/000008-shape-run-ledger.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | bda6e9edfae6ff8484fbd06b64e5f21880201bfd..HEAD |
| command | sdlc milestone-close --issue 8 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-05T21:07:47-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need — all findings are empirically reproduced against a clean worktree at HEAD (`5967703`), and I've cleaned up. Here is the review.

```verdict
verdict: REWORK
confidence: high
```

**Summary.** metis#8 **M2 (ledger integration)** ships a genuinely solid engine — `pkg/ledger`'s M1 findings are all fixed (list free-params round-trip, `TopN` clamps negative n, `Filter` returns a fresh ledger), `rowsFromManifest`'s per-step metric namespacing is correct and pure, and the promote reconstruction actually reproduces a row's point-address (I verified `91ea8a…` in from a sweep round-trips to `91ea8a…` on the promoted re-run). Build/vet/full suite are green at HEAD. But M2's whole point is the **user-facing integration surface** (`ledger show` + `promote`), and three parts of that surface don't work as documented/claimed — all at the exact seams the tests don't exercise. Most seriously: **`metis promote` prints `"the promoted winner is committed against a dirty tree"` and then does not commit anything** — no `gitCommitter` implementation exists, `cmdPromote` never injects one, so the "committed at its code SHA" Done-when + atlas claim is a no-op that reports success. On top of that, both headline commands **fail with their documented argument order** (positional `<shape>` before flags — the form in every usage string, the atlas, the issue, and the plan), and `promote --best` on the repo's own keystone fixture returns "no qualifying row" because its objective metric isn't namespaced. These are blocking for a boundary whose deliverable *is* these commands, and the Log's "verified in the real CLI" is contradicted by the documented invocations. The rework is bounded (the core mechanisms are correct); fix the CLI seams, wire/relocate the commit, fix the fixture, and re-verify in the real CLI.

Note: the working tree is currently dirty with **M3 WIP** (`capture.go`/`capture_test.go` untracked, `sweep.go` calling a not-yet-defined `captureSweepCode`) — that state does not build. I reviewed the committed M2 boundary (`5967703`) via an isolated worktree; the M3 WIP is out of scope here but should not have leaked into an M2-close review tree.

### 1. Strengths

- **`promotedExperiment` reconstruction is correct and pure** (`cmd/metis/ledger.go`): it re-expands the shape and matches by free-params (reusing `shape.Expand` + `shapePointToExperiment`) rather than inverting the flat map, and `freeParamsEqual` compares via canonical JSON so int-vs-float64 CSV round-trip drift doesn't break the match. I confirmed empirically that the promoted experiment reproduces the row's **exact point-address** — the central reproducibility claim genuinely holds.
- **The M1 review findings were actually fixed and regression-pinned** — `TestCSV_ListFreeParamsRoundTrip` covers the `features: [title, family]` list case that M1 flagged, `TestTopN_NegativeN` the negative-n panic, and `parseCell`/`cell` JSON-encode non-scalars. Good closure of prior feedback.
- **`rowsFromManifest` + `namespacedMetrics` are pure and unit-tested without disk** (`ledger_test.go:TestRowsFromManifest_NamespacedMetrics`) — the `<step>.<metric>` namespacing that fixes v0's flat last-write-wins is done at the right layer, with the record reads left to the `cmd/metis` shell (ARCH-PURE holds cleanly).
- **Immutability-by-snapshot is a real property test** (`TestLedger_PriorRowsReproduceAfterSpaceEdit`): it edits the shape's space and asserts a prior row still reconstructs against its snapshot while the edited space no longer contains the point — pinning the actual design guarantee, not a mock.
- **The sweep→ledger integration e2e asserts the load-bearing identity** (`TestLedger_SweepWritesSidecarAndSummary` + the manifest `run_id == record.point_address` checks carried from #7) — the ragged union columns and idempotent re-sweep dedup are exercised end-to-end.

### 2. Critical findings

- **`promote` never commits, but reports that it did** — `cmd/metis/ledger_cmd.go:104-107` (`cmdPromote` builds `promoteOpts` with `git: gitCLI{}` but leaves `commit` nil) + `:172-179` (commit is skipped when `o.commit == nil`) + **no concrete `gitCommitter` implementation exists anywhere** (`grep` finds only the interface at `:123` and the test-injected field). So the production path writes `<name>.md` and stops. Worse, `:169-170` prints `"metis: warning: repo is dirty — the promoted %s is committed against a dirty tree"` — I confirmed live that after this message `winner.md` is `?? ` (untracked) and no promote commit lands in `git log`.
  - *Failure/contract:* the Done-when (issue:151-153) and atlas (`atlas/index.md:32`, "committed at its code SHA (warns if dirty)") both state promote commits; it doesn't, and the dirty-path message actively asserts a commit that never happens. Undelivered Done-when + false success report + doc drift, all untested.
  - *Fix sketch:* implement a real `gitCommitter` (shell `git -C dir add <file>` / `git commit -m …`) and inject it in `cmdPromote`; **or** consciously descope the commit to M3 (where the SHA becomes real) and, in the same change, drop the "is committed" message, soften the atlas line, and re-scope the M2 Done-when. Either way add a test that asserts the file is (or isn't) committed. The comment `nil → real git` at `:118` is simply false — nil means *no commit*.

- **Both headline commands fail with their documented argument order** — `cmd/metis/ledger_cmd.go:97` (`fs.Parse(args)` in `cmdPromote`) and `:29` (`fs.Parse(args[1:])` in `cmdLedger`) use Go's stdlib `flag`, which stops at the first positional. The usage strings (`:22`, `:102`), the atlas (`atlas/index.md:29-30`), the issue Done-when, and the plan/design all write `<shape>` **first** (`metis promote <shape> --best --name X`, `metis ledger show <shape> --sort M`). I reproduced both: the documented order errors (`usage: …` / `want one <shape.md>, got 3`); only flags-first (`metis promote --best --name X <shape>`) works.
  - *Failure scenario:* a user copy-pasting the atlas example gets an opaque `usage:` error with no hint that reordering fixes it — the primary M2 commands are non-functional as documented.
  - *Fix sketch:* reorder args so flags precede positionals before `Parse` (small helper that pulls the lone `<shape.md>` out of `args`), or adopt a parser that permits interspersed flags; alternatively fix every doc/usage to flags-first (worse UX, but at least honest). Add a `cmdPromote`/`cmdLedger` arg-parse test using the documented order — none exists, which is why this slipped (the e2e calls `runPromote`/pre-ordered args, bypassing the CLI parse).

### 3. Important findings

- **The objective metric must be namespaced, but the keystone fixture isn't — `promote --best` and the body summary silently fail** — `regenLedgerSummary`/`writeSweepLedger` (`cmd/metis/ledger.go`) and `Best` (`ledger_cmd.go:145`) look up `sh.Sweep.Objective.Metric` verbatim against row metrics that are namespaced `<step>.<metric>`. `testdata/experiment/titanic-baseline-shape.md:8` declares `objective: {metric: cv_score}`, but rows carry `train.cv_score`. I reproduced (mirroring with `echoed` vs `train.echoed`): the top-N body summary renders **empty** under "By `echoed`", and `promote --best` returns `no qualifying row (objective "echoed")`. Nothing validates or documents that the objective must be namespaced.
  - *Fix sketch:* correct the fixture to `train.cv_score`; document the `<step>.<metric>` requirement in the experiment-shape datatype doc; and make the empty case *loud* — have `regenLedgerSummary`/`Best` warn "objective metric `train.cv_score` not present in any row" instead of silently emitting an empty summary / "no qualifying row".
- **The round-trip test under-asserts the Done-when** — `TestLedger_PromoteBestRoundTrips` (`ledger_e2e_test.go`) checks only `run.Status == "ok"`, not that the promoted run **reproduces the row's point-address + metrics** (the actual Done-when text). I verified the address does reproduce, so the behavior is right — but a regression that broke reproduction (e.g. a stray field entering the promoted `with`) would pass green. Assert `record.point_address == row.PointAddr` in the round-trip; this is the same "green test blind to the property it claims" pattern already in `lessons.md`.

### 4. Minor findings

- **DRY:** `ledgerParseCell` (`ledger_cmd.go:236`) duplicates `pkg/ledger.parseCell` minus the list branch — export `ledger.ParseCell` and reuse. `freeParamTuple` (`ledger.go`) and `freeParamTupleMap` (`ledger_cmd.go:221`) are the same renderer over a `Row` vs a map; collapse to one. (ARCH-DRY.)
- **`--point` can't express a list free-param** — `parsePointSelector` splits on `,` and `ledgerParseCell` has no list branch, so `--point 'adapt.features=[title,family]'` can't select the titanic keystone's headline coordinate. Scalar selectors only; note the limit.
- **`ledger show --dir` ignores the shape's declared objective direction** (`ledger_cmd.go:27`) — sorting `--sort train.loss` on a `minimize` objective silently maximizes unless the user re-specifies `--dir minimize`. Consider defaulting `--dir` from `sh.Sweep.Objective.Direction`.
- **No `construct/datatype/` ledger doc** — the plan's M2 said "atlas: … + the ledger datatype"; only `atlas/index.md` got a `pkg/ledger` paragraph. Defensible (the ledger is a mechanism, not a user-authored artifact), but reconcile the plan wording.

### 5. Test coverage notes

Tests pin real properties for the *pure* layer (dedup, ragged round-trip, Best/TopN, `rowsFromManifest` namespacing, immutability-by-snapshot, sweep→sidecar idempotence). The gaps are exactly where the shipped bugs live, all at the **CLI/integration seam**: (a) no test drives `cmdPromote`/`cmdLedger` argument parsing → the flag-ordering bug shipped; (b) no test exercises the commit path → the never-commits gap shipped and the false message went unnoticed; (c) the round-trip asserts `status==ok` but not point-address/metric reproduction; (d) no test uses an un-namespaced objective → the fixture-vs-metrics mismatch shipped. A single "real CLI, documented invocation" integration test (parse args → run → assert file written/committed/reproduced) would have caught #2, #3, and the objective mismatch at once.

**Docs gate:** atlas *is* updated for the new surface (`atlas/index.md` documents `ledger show`/`promote`) — but it documents the **broken** positional-first order and the **unimplemented** commit, so the atlas edit encodes the drift rather than the behavior (folded into Criticals #1/#2). No `README.md` exists in the repo → no README finding (consistent with the #3/M1 precedent).

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass (minor).** The load-bearing single-sourcing is good (`upstreamHashes`, `repoSHAsOf` shared across record/sweep/cache; `promotedExperiment` reused by promote *and* the immutability test). Only the cell-parser + tuple-renderer duplication above remains (Minor).
- **ARCH-PURE — pass.** Clean split: `rowsFromManifest`/`namespacedMetrics`/`promotedExperiment`/`freeParamsEqual` are pure and unit-tested without IO; `writeSweepLedger`/`loadLedger`/`loadSweepRecords`/`regenLedgerSummary`/`runPromote` are the thin injected shell. `pkg/ledger` stays a pure aggregation view over #3's records, not a second store — the plan's "no `## Runs` retrofit" scope held.
- **ARCH-PURPOSE — flag.** The shadow-sweep over promote's stated purpose ("materialize a row as an all-singleton experiment **committed at its code SHA**, round-trips") finds the *reconstruction + round-trip* delivered but the **commit half is a no-op** — the diff settles for the easy subset (write the file) and reports the harder, named half as done. The commit is explicitly an M2 deliverable (plan M2: "commit it at the code SHA (warn if dirty)… M3's capture makes even a dirty run's SHA real"), so this is an under-delivered purpose, not a clean deferral — resolve it before the close verdict is recorded (implement it, or descope-and-correct-the-docs).
- Forward: when M3 wires the real captured SHA, the `promote` commit should target that side-ref SHA — decide now whether promote commits against HEAD (M2) or the captured ref (M3) so the "self-contained reproducible commit" story is coherent.

### 7. Plan revision recommendations

- **`workshop/plans/000008-shape-run-ledger-plan.md`** — add a `## Revisions` entry recording the M2 reality: (1) `promote` currently **writes the experiment file but does not commit** (no `gitCommitter` impl wired) — either implement the commit in M2 or move it to M3 and correct the Done-when/atlas; (2) `ledger show`/`promote` require **flags before the positional `<shape>`**, contradicting the documented `<shape>`-first form — record the chosen resolution (arg-reorder vs doc-fix); (3) the sweep **objective metric must be namespaced** (`train.cv_score`), and the committed titanic fixture (`cv_score`) is wrong — note the fixture fix + the namespacing contract. Until then the plan's M2 line (`+ commit at its code SHA`) and the issue Done-when overstate what ships.
- **`workshop/issues/000008-shape-run-ledger.md`** — the 2026-07-05 M2 Log says "commit at code SHA (warn-if-dirty)" and "verified in the real CLI"; both are inaccurate for the documented invocations. Amend once the fixes land so the close evidence matches the code.
