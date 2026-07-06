# Boundary Review — metis#13 (whole-issue close)

| field | value |
|-------|-------|
| issue | 13 — Config immutability — run output (## Runs / ledger top-N) must leave the experiment .md |
| repo | metis |
| issue file | workshop/issues/000013-config-immutability-run-output-runs-ledger-top-n-must-leave-the-experiment-md.md |
| boundary | whole-issue close |
| milestone | — |
| window | 34b11db85dd7dc9b809ed4c67c1f538c1726e4b1..HEAD |
| command | sdlc close --issue 13 |
| reviewer | claude |
| timestamp | 2026-07-06T15:30:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core change is correct and complete: the run path no longer mutates the experiment `.md`. I shadow-swept every `os.WriteFile` in `cmd/metis/` — the only run-time writers now target `runs/<id>/{run,record}.json`, `sweeps/.../manifest.json`, and the `<stem>.ledger.csv` sidecar; **no path writes `o.expPath`/`shapePath`** during a run. `appendRunLog` and `regenLedgerSummary`'s body-write are gone, the objective-missing warning is faithfully preserved in `warnIfObjectiveMissing`, `go build`/`vet`/`test ./cmd/metis/` are green, and the tests were flipped to assert byte-identity of the config. What blocks a clean SHIP is not code behavior but drift: the `experiment` **datatype contract** and one **atlas bullet** still tell readers the runner appends to `## Runs`/regenerates the body top-N — the exact behavior this issue removed — and the plan-flagged dead code (`recordSummary` chain) was left in. All non-blocking; fix the docs + dead code and ship.

**1. Strengths**
- Clean, minimal removal — both mutation sites deleted, the shared per-point runner (`runResolvedExperiment`) keeps a single write site (ARCH-DRY preserved), and `run.go` correctly drops the now-unused `strings`/`record` imports (`run.go:6-14`).
- The objective-missing check was *distilled*, not dropped: `warnIfObjectiveMissing` (`ledger.go:157`) keeps the one genuinely-useful diagnostic from the old summary with equivalent logic — a real judgment call done right.
- Tests pin real logic: `run_test.go:79` and `:177` assert full `string(updated) == string(b)` byte-identity against a fixture that already carries a `## Runs` heading — so they actually prove "no append below the heading," not a weaker absence check.
- atlas/experiment.md is thoroughly rewritten to state the immutable-input invariant at the surface, flow, and ledger levels.

**2. Critical findings**
None — behavior is correct.

**3. Important findings**
- **`construct/datatype/experiment.md:39-41, :50, :59` — datatype contract now contradicts the shipped invariant.** Still reads "The body carries a `## Runs` section — the execution ledger, **appended by the runner**", "Leave a `## Runs` heading for the runner to append to", and "Runs accumulate in `## Runs`". This is the authoring contract agents/users follow when creating experiments, and it lives in `construct/` (base layer — propagates to dependent repos). Leaving it asserting the opposite of #13 is a documented-contract drift and a Docs-gate miss. Fix: rewrite the "Runs convention"/authoring/rules to state the `.md` is immutable input; run history is `runs/<id>/record.json` + `.ledger.csv`, browsed via `metis ledger show`. (Same class: `construct/datatype/experiment-shape.md:70` still says "a shape's body carries a top-N summary + a pointer to the ledger" — no longer true.)
- **`atlas/index.md:29` — the atlas update it claims to make is incomplete.** The same commit fixed the `pkg/record` bullet (`:19-20`) but the `pkg/ledger` bullet still ends "...appended to `<shape>.ledger.csv` (idempotent) **with the shape body's top-N summary regenerated**." That's precisely the removed behavior, stated as current, inside the file that was the explicit deliverable. Fix: drop "with the shape body's top-N summary regenerated" → "the human view is on-demand `metis ledger show`".
- **Dead code: `recordSummary` + `formatKnobs` + `formatMetrics` (`record.go:160-210`) are now orphaned** — referenced only by `TestRecordSummary_RendersKnobToScore` (`record_test.go:103`). The plan's Task 1.2 explicitly said "delete `appendRunLog` … **(and `recordSummary` if now unused)**"; it is now unused. Go won't flag package-level dead functions, so this passes build silently. This leaves a test asserting the shape of an implementation no production path calls (the "tests reasserting implementation" smell) and a second, now-purposeless free-param/metric renderer alongside `freeParamTuple` (ARCH-DRY). Fix: delete the three functions + the test. If they're intentionally retained for the metis#8 unification, that intent isn't documented anywhere — prefer removal now and reintroduction in #8 (YAGNI).

**4. Minor findings**
- Stale doc-comments in `run.go` describing the removed append: `:77` ("appends a summary to the experiment's `## Runs` log"), `:128-129` ("writes its run.json + provenance record + ## Runs line"), `:162` ("without … touching the ## Runs log").
- Stale test doc-comments still narrating the old append: `run_test.go:19-21` and `:127`, `e2e_test.go:30`, `record_e2e_test.go:33`.
- `testdata/experiment/run-fail.md:16` fixture prose: "…and appends a `## Runs` line, then returns an error" — now false.
- `workshop/lessons.md:33`: "The step-runner appends `## Runs` and writes `runs/`" — the `runs/` half still justifies the TempDir rule; drop the `## Runs` clause.
- `TestLedger_SweepWritesSidecarAndSummary` (`ledger_e2e_test.go:50`) — the name's "AndSummary" is now a misnomer; the test asserts the summary is *absent*. Rename (e.g. `_SweepWritesSidecarNotBody`).

**5. Test coverage notes**
- The single-run tests assert full byte-identity (strong). The **sweep** e2e (`ledger_e2e_test.go:84-87`) only asserts the body lacks `metis:ledger`/`## Top runs` markers, not byte-identity — weaker than the plan's "shape `.md` byte-identical" Done-when. It catches this regression (the markers were the only thing written), but for parity with the single-run tests and durability against a future body-writer, capture the pre-sweep body (the `writeShape` heredoc) and assert equality after the sweep. Non-blocking.
- Removing `recordSummary` removes its test — no coverage lost, since nothing consumes it.

**6. Architectural notes**
- **ARCH-DRY — flag:** the orphaned `recordSummary` chain (Important #3) and the doc drift (one source now says "immutable," another says "runner appends") are two-sources-of-truth violations. Consolidate to the code invariant.
- **ARCH-PURE — pass:** the removal keeps the pure/IO split intact — the deleted writes were in the IO shell (`cmd/metis`), the pure `pkg/experiment`/`pkg/ledger` cores are untouched, and `warnIfObjectiveMissing` is a correctly thin IO wrapper over pure `ledger.TopN`.
- **ARCH-PURPOSE — pass (code), flag (docs):** the shadow-sweep confirms the purpose is *enforced* — no run path mutates the config, so a committed config is now a stable content-hash, unblocking #14. The remaining hand-maintained restatements of the old model (the datatype docs above) are exactly what ARCH-PURPOSE's shadow-sweep flags; they're documentation rather than a derived consumer, so this is a docs fix, not a code gap.

**7. Plan revision recommendations**
- The plan/Core-concepts table (`workshop/plans/000013-config-immutability-plan.md:24-27`) still lists `regenLedgerSummary` as "deleted (or reduced to no-op-on-.md)" and does not name the `construct/datatype/*` docs as consumers to update. If the docs are fixed as part of this boundary, no revision is strictly needed; if they are deferred, add a `## Revisions` entry recording that `construct/datatype/experiment.md` + `experiment-shape.md` doc updates and the `recordSummary` deletion were carried out of scope, so the plan stops implying a pure removal that left stale contracts and dead code behind.
