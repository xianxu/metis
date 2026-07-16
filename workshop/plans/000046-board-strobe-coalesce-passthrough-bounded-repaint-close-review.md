# Boundary Review — metis#46 (whole-issue close)

| field | value |
|-------|-------|
| issue | 46 — board strobes under warm-cache bursts — coalesce passthrough + repaint at a bounded rate |
| repo | metis |
| issue file | workshop/issues/000046-board-strobe-coalesce-passthrough-bounded-repaint.md |
| boundary | whole-issue close |
| milestone | — |
| window | 01d27801b7e58a6febbe5aa8d496fc54155f03be..HEAD |
| command | sdlc close --issue 46 |
| reviewer | claude |
| timestamp | 2026-07-15T23:32:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The double-buffer + flush-budget redesign is sound and delivered as specified: passthrough coalesces into `pending`, `flushLocked` is the single atomic erase→dump→repaint site, quiet writes flush inline (zero `lastFlush` means the first write always draws), the size cap bounds a frozen-budget flood, and close remains idempotent while now also gating cursor-restore on `hidden` (a small pre-existing wart fixed in passing). Lock order `sink.mu → bw.mu` is preserved — `tick()` releases `sp.mu` before `forceFlush()` takes `bw.mu`, so no inversion, and the ticker's deferred stop in `runShapeSweep` runs before `runExperiment`'s deferred `close()`. What keeps this off SHIP: the one changed production line in `progress.go` (`tick()` → `forceFlush`) — the mechanism the spec names as what re-pins the board after a burst — has **no test at all** (no test file references `tick()` or `forceFlush`), and I could not execute the suite to confirm the claimed green state: the Bash tool is broken in this review session at the harness level (`EPERM mkdir ~/.claude/session-env/…` on every invocation, even `true`, sandbox on or off — the main agent should re-run `go test ./cmd/metis/ && go vet ./cmd/metis/` as the executable check).

**1. Strengths**

- `board.go:214-228` — `flushLocked(now time.Time)` taking the instant as a parameter instead of re-reading the clock is exactly right for scripted-clock tests (the Log calls this out), and consolidating hide-cursor, erase, tail-holdback, redraw, and `lastFlush` into the one flush site is clean ARCH-PURE-friendly structure: all budget/coalesce logic in the compositor, renderer and sink untouched.
- `board_test.go:326-356` — `TestBoardWriter_BurstCoalesces` pins the actual contract (≤5 erase cycles per 500ms burst, byte completeness, ordering, final frame) rather than implementation details; with the scripted times it deterministically produces 3 erase cycles, well under the bound.
- `board_test.go:213` / `board.go:240-245` — close's newline-completion now checks the last byte first, so a pending buffer that already ends in `\n` (possible now that multiple complete lines coalesce) isn't double-newlined — a subtle case the old `append(pending, '\n')` would have gotten wrong post-coalescing.
- `progress.go:351-355` — snapshotting `sp.bw` under the lock and flushing after release is the correct way to keep the tick sink-first without holding `sp.mu` across terminal IO.
- `atlas/index.md:101-107` — atlas updated in-range with the semantics *and* the rationale (volume, not correctness); README correctly untouched (no new user-facing surface; `--no-tui` escape hatch unchanged).

**2. Critical findings**

None.

**3. Important findings**

- `progress.go:347-356` + `board.go:203-210` — **missing test coverage for the tick/forceFlush path** (the review-checklist "kind of bug this diff could ship"). Spec item 4 and Done-when both lean on the 500ms tick to re-pin the board and drain a mid-budget pending write, but no test invokes `tick()` or `forceFlush()` (grep over `*_test.go`: zero matches); `TestRunExperiment_BoardMode` runs without the real-time ticker ever plausibly firing. A regression here (e.g. a pending line stranded until the next Write, or a lock-order slip reintroduced in `tick()`) would ship silently. Fix sketch: a unit test that writes within the budget window (scripted clock, no inline flush), asserts the bytes are *not* yet in the terminal, then calls `bw.forceFlush()` (or `sp.tick()` with a wired board) and asserts the pending line + repainted frame appear.

**4. Minor findings**

- `progress.go:352-355` — when the 250ms budget has elapsed at tick time (the common case with a 500ms ticker), `emit()`→`paint()` flushes *and* `forceFlush()` immediately erases/redraws again: two erase cycles per tick. Harmless vs. the 500Hz strobe, but it doubles the tick's contribution to the repaint budget; skipping paint's inline budget-flush on the tick path (store-only, let `forceFlush` draw) would halve it. (This is likely why the live pty run logged 7 cycles in 2.4s rather than ~5.)
- `board.go:182,222` — for a newline-free flood (e.g. a `\r`-only progress stream) the `pendingCap` doesn't actually bound memory: `flushLocked` drains nothing without a newline, so every subsequent Write re-triggers a cap-exceeded erase/redraw while `pending` grows unbounded. Same unbounded-tail behavior as pre-#46 (not a regression), but the cap's stated purpose ("bound memory") isn't met for that input class; consider force-dumping raw pending at cap.
- `board.go:175-186` — `Write` now always returns `nil` error (`flushLocked` discards `b.w.Write`'s error), where the old code propagated it. No production caller checks it (all `fmt.Fprintf`), and `erase`/`redraw` already discarded errors pre-#46, so this is consistency rather than correctness — but worth a one-line comment that errors on the terminal writer are deliberately dropped.
- `board.go:233-252` — ARCH-DRY: `close()` re-implements the flush (erase → dump → redraw) instead of newline-completing `pending` and delegating to `flushLocked(b.now())` before the cursor-restore. Six duplicated lines today; a drift risk if flush semantics evolve again (the `hidden` handling already diverges between the two paths).

**5. Test coverage notes**

The three new tests map one-to-one onto the Done-when bullets (burst-coalesce with order/completeness, quiet-inline, frozen-clock size cap), and the pre-#46 tests were correctly migrated to stepping clocks rather than weakened — the one loosened assertion (`TestRunExperiment_BoardMode` switching from "after the last erase" to "after the final frame") is a legitimate consequence of coalescing, not a dodge. Gaps: the tick/forceFlush path (Important, above), and the live-pty bound (Done-when item 4) rests on the Log's claim (7 erase cycles, 2.4s) — plausible and consistent with my static trace, but not independently reproducible here with the shell down. **The main agent must re-run `go test ./cmd/metis/` + `go vet` before recording the close verdict**, since this review could not execute it.

**6. Architectural notes**

- ARCH-PURE: **pass** — clock injected, all timing logic scripted in tests against `strings.Builder`, no sleeps; render stays pure, budget logic stays in the one compositor.
- ARCH-DRY: **pass with the close/flushLocked note above**; the shared-helper structure (`erase`/`redraw`/`lastNewline`) is otherwise good.
- ARCH-PURPOSE: **pass** — the shadow-sweep holds: every consumer of the terminal (passthrough Write, sink paint, tick, pool notices, close) routes through the same budgeted flush; nothing was deferred as follow-up. The `--no-tui` escape hatch is explicitly preserved per spec.
- Forward note: the compositor now has three flush triggers (budget, cap, tick) plus close; if a fourth arrives (e.g. resize handling), consider extracting a `shouldFlush(now) bool` predicate so the policy stays single-sourced.

**7. Plan revision recommendations**

None — the in-issue Plan (both boxes) matches the code; no Core-concepts table to cross-check. If the tick-path test is added, a one-line `## Log` entry suffices; no `## Revisions` needed.
