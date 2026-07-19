# Boundary Review — metis#67 (whole-issue close)

| field | value |
|-------|-------|
| issue | 67 — Default fold-ordered scheduling (graduate --live); --global-fanout escape hatch |
| repo | metis |
| issue file | workshop/issues/000067-default-fold-ordered-scheduling.md |
| boundary | whole-issue close |
| milestone | — |
| window | 18d26a8c8f21d98a38a7670e214cd84912c542fe..HEAD |
| command | sdlc close --issue 67 |
| reviewer | claude |
| timestamp | 2026-07-19T12:06:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The change delivers exactly what metis#67 specs: `prioritySem` (fold-ordered) becomes the default leaf budget, `--global-fanout` is the loud escape hatch back to `chanSem`, `--live` and the `o.live` field are removed, and `q`-finalize is regated on `o.tui` alone. The default flip is byte-identical by construction (both sems produce the same artifacts; the reduce is order-independent), so the behavioral risk is genuinely low. The one real thing to address before committing is Go formatting — the new `globalFanout` identifier is longer than its neighbors and the surrounding alignment padding wasn't reflowed, so `run.go` and `main.go` look non-gofmt-clean. Nothing blocks the boundary. Note: I could **not** run `go test -race`, `go vet`, or `gofmt` — Bash is unavailable in this environment (harness `EPERM` on session-env) — so the verification below is static; I relied on the Log's test claims where I couldn't execute.

**1. Strengths**
- `selectLeafBudget` (`prioritysem.go:66`) is a clean ARCH-PURE extraction: it collapses the old `run.go` `if o.live || o.autoStop {…} else {…}` branch into one pure, IO-free decision, unit-tested by concrete type. This is precisely the fix the plan-quality judge demanded (the default is unobservable from artifacts, so the pure seam is the only writeable assertion). Good ARCH-DRY + ARCH-PURE win.
- Behavior preservation is faithful: old `if o.maxParallel>1 && o.leafBudget==nil` → new `if o.leafBudget==nil` with the `maxParallel<=1 → nil` moved *into* the selector is exactly equivalent (`run.go:153`). Injected-budget test paths still win.
- The broadened `q`-gate is correctly proven safe: `stopSignal` is consumed only on the nested path (`sweep.go:576`, gated on `stopSignal!=nil && runControl!=nil`, never on `o.live`), and `TestQFinalize_NonSweepIgnoresStopSignal` pins that a plain run ignores an already-fired signal.
- Thorough shadow-sweep: `o.live` is fully gone from code (verified by grep — remaining `live` hits are all `live board`/`live estimate`/history); atlas `experiment.md` + `index.md` reconciled; `--auto-stop` help updated to drop "implies --live".
- The `--auto-stop --global-fanout → chanSem` co-existence case is genuinely reasoned (auto-stop forces sequential-outer in `sweep.go:607` independent of budget type; within a fold all leaves share one priority, so the two sems are identical). Correct, not hand-waved.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- **gofmt (primary, actionable):** `run.go` struct fields (~`88–96`) and `main.go`'s `cmdRun` runOpts literal (~`68–82`) appear not gofmt-clean — `globalFanout`/`globalFanout:` is now the longest name/key in its block but the shorter siblings kept their old padding (e.g. `priority    int` type-col 13 vs `globalFanout bool` type-col 14; `maxParallel: *parallel` vs `globalFanout: *globalFanout` values misaligned). Fix: `gofmt -w cmd/metis/run.go cmd/metis/main.go` (confirm first with `gofmt -l`). *(Flagged as PLAUSIBLE — reasoned from the file bytes; I couldn't run gofmt here.)*
- `exec.go:40` — stale parenthetical "`0 on the flat/preamble path (chanSem ignores it anyway)`": the default flat/preamble path now runs on `prioritySem` (all leaves at priority 0, FIFO — still correct), so the aside references the non-default sem.
- `live_test.go` — `TestLive_ByteIdenticalToDefault` name + comments still say `--live`/"default"; it's now a `prioritySem`-vs-`chanSem` determinism check. Harmless historical naming; optional rename.
- `workshop/projects/arena2-playground-s6e7.md:224` — a live project note still advertises `--live` fold-ordered scheduling as shipped tooling. Low signal and arguably out of #67's scope (historical session log in a peer project), but the flag it names no longer exists.

**5. Test coverage notes**
- `TestSelectLeafBudget` pins the default at the pure seam — the right altitude, since byte-identical artifacts make the choice unobservable end-to-end.
- Benign gap: nothing asserts `run.go` actually threads `o.globalFanout` (not a literal) into `selectLeafBudget`. A mis-wire would be invisible anyway — both sems are byte-identical, so `--global-fanout` mis-selection is functionally harmless. Not worth a test.
- The determinism, `q`-stop (`live_stop_test.go`, now correctly relying on `stopSignal!=nil` alone), and auto-stop (`live: true` was vestigial under `maxParallel=1`) edits are all correct by inspection. I could not execute the suite — recommend the main agent run `go test ./cmd/metis -race && go vet ./cmd/metis && gofmt -l cmd/metis/*.go` before committing.

**6. Architectural notes**
- **ARCH-DRY: PASS** — one source of truth for the budget decision (`selectLeafBudget`); no direct `newChanSem`/`newPrioritySem` remains in production business logic (only the pure seam + tests).
- **ARCH-PURE: PASS** — the decision is pure and unit-tested without mocks; `runExperiment` keeps only the thin `o.leafBudget = selectLeafBudget(...)` glue.
- **ARCH-PURPOSE: PASS** — the purpose (fold-ordering as the *enforced* default) is delivered in `run.go`, not merely documented; every consumer (flag parse, selector, `q`-gate, help text, atlas, tests) derives from the change. The only remaining hand-restatement is the arena2 historical note (Minor above), which is a log, not a live consumer.

**7. Plan revision recommendations**
None — the plan matches the code; all four Plan checkboxes are delivered as described.
