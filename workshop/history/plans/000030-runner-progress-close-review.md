# Boundary Review — metis#30 (whole-issue close)

| field | value |
|-------|-------|
| issue | 30 — runner progress reporting — SizeHint + progress callback (k/n + live outer-cv) |
| repo | metis |
| issue file | workshop/issues/000030-runner-progress.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5bb49eb1d9d0464a22a59cc0ee97b9e53902fa42..HEAD |
| command | sdlc close --issue 30 |
| reviewer | claude |
| timestamp | 2026-07-15T17:21:56-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: SHIP
confidence: medium
```

This boundary delivers issue #30's full purpose — `SizeHint` on the `Sampler` interface, a completion-fired typed `ProgressEvent` in `Run`, and the throttled aggregated line in `cmd/metis` — with the two spec deviations (per-Tell → per-completion timing, flat-format correction) properly recorded in the issue's `## Revisions` rather than papered over. All 20 `Run` call sites are updated (grep-verified: 4 production + 15 legacy tests passing `nil` + the new progress tests), the plan's Core-concepts table matches the code row-for-row, and the docs gate is satisfied (atlas updated in-window; no repo README exists; the kbench peer RUNBOOK line the plan promised is verifiably present at `RUNBOOK-sweep.md:48`). All findings are Minor. Confidence is medium rather than high for one reason only: I could not execute `go test` — the Bash tool fails at the harness level (EPERM creating the session-env directory, even with sandbox disabled), so my verification of the test suite is a by-hand trace of every assertion (all check out) plus the Log's claim of `-race`-green, not an independent run. The main agent should re-run `go test ./pkg/sampler/ ./cmd/metis/ -race -count=1` before closing.

## 1. Strengths

- **The per-Tell → per-completion revision is the right call, made the right way.** The Spec's premise predated #31's batch-scoped exec; firing at point completion (`pkg/sampler/run.go:44-58`) preserves the count contract (one event per point) while making events actually live. The rationale is documented in three places that agree (run.go doc, plan §revision, issue `## Revisions`).
- **Totals seeded at wiring** (`cmd/metis/progress.go:127-141`, wired at `sweep.go:255-256`) solves the denominators-arrive-too-late problem by calling the samplers' own `SizeHint` on their `Init` states — no shape math re-derived in the renderer (ARCH-DRY pass).
- **Sink-owned aggregate counters, never `ev.K`** — and `TestSweepProgress_ConcurrentCounts` (`progress_test.go:130`) pins exactly that bug class (8 concurrent passes each counting their own 1..8; the sink must see 64). This is the mistake a future editor would most plausibly make.
- **The #38 identity seam via `forPass(i)` closure binding** keeps `pkg/sampler` coordinate-free (ARCH-PURE pass), and the `FoldPoint.Partition` lookalike trap is explicitly warned against in both the plan and the atlas entry.
- **Concurrency is carefully reasoned and mostly pinned**: the error latch in `runNestedCV` is mutex-guarded (`sweep.go:360-373`), the progress gate composes with it safely, the documented lock order (Run-mu → sink-mu → syncWriter) is acyclic on inspection, and `TestRunProgress_ParallelMonotoneComplete` pins the monotone-k-under-ParExec guarantee.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

- `cmd/metis/progress.go:149` — `sweepProgress.direction` is stored but never read (vestige of the pre-#32 "best-so-far" display the flat-format correction removed). Drop it, or leave with a `// #38` comment if the TTY board will want it.
- `cmd/metis/progress_test.go:75-77` — the throttle-test comment mis-states which events emit ("event 5 (t=1000) and event 10 (t=2000)"); the actual emits are event 1 (first event always emits — `started` is false) and event 6 (t=1000, ≥1s after t=0). The count assertion (2) is correct; fix the comment so the next reader doesn't distrust the test.
- `cmd/metis/sweep.go:390-394` — the error-gated `driverEvent` wrapper (skip sentinel-zero scores once `firstErr` latches) has no test; a failing nested sweep's progress behavior is unpinned.
- `nestedcv_e2e_test.go:65` / `shapesweep_test.go:330` — `finalProg[:strings.IndexByte(finalProg, '\n')]` panics (rather than fails) if the final line ever lacks a trailing newline; test-only robustness nit.
- The `Sampler` interface gaining `SizeHint` is compile-breaking for any out-of-repo implementer — deliberate and spec-mandated, but since metis is the base layer, worth a line in whatever propagation notes dependent repos read.

## 5. Test coverage notes

Coverage is strong and pins real logic, not mocks: the Seq/Par event contract (order, totals, completed pair), nil-callback no-op (every legacy test now passing `nil` is that regression), the `SizeHint` table across all four production samplers, the pure `progressLine` table including budget (`k/≤n`) and unknown (`k/?`) kinds, the scripted-clock throttle (no sleeps — injected clock, per the repo's controllable-time posture), nil-sink safety, and fixture pins on both the nested and flat paths with the frozen-clock caveat correctly reasoned (only always-emit lines appear; throttle pinned separately). Gaps: the error-gated driver event path (above), and I could not independently execute the suite due to the harness Bash failure — re-run `-race` before close.

## 6. Architectural notes

- **ARCH-DRY: pass.** `SizeHint` is the single source of n; `seededTotals` composes denominators from direct `SizeHint` calls rather than re-deriving grid math. The local `meanSE` in `progress.go:98` is a justified non-duplication: `sampler.Aggregate` takes `[]FoldScore` (keyed, complexity-carrying) — reusing it would mean fabricating fold keys for a display-only reduce, and the comment says exactly why the boundary is drawn there.
- **ARCH-PURE: pass.** `Run` stays pure over its four injected closures; `progressState`/`progressLine` are the pure render core, table-tested with zero IO; `sweepProgress` is the thin shell with injected clock + writer.
- **ARCH-PURPOSE: pass.** The full purpose (interface + loop + renderer + verified real output) shipped; the #38 deferral is a genuinely separable extension whose seam (`forPass`, injected clock already in the sink) was designed, not hand-waved. Shadow-sweep: the only consumer (the sink) derives from `SizeHint`; no hand-maintained restatement of n exists anywhere.
- For #38: the `_ = outer` placeholder in `forPass` and the unused `direction` field are the two dangling threads that issue should either consume or delete.

## 7. Plan revision recommendations

None — the plan already matches the shipped code (the 16→19 call-site correction landed in this window, and the plan-review findings it encodes are all reflected in the implementation). The plan file's internal Step checkboxes remain `- [ ]` while the issue's Plan section is fully checked; if the repo convention is that the issue tracks completion, no action is needed.
