---
id: 000046
status: done
deps: []
github_issue:
created: 2026-07-15
updated: 2026-07-15
estimate_hours: 0.61
started: 2026-07-15T23:19:12-07:00
actual_hours: 0.13
---

# board strobes under warm-cache bursts — coalesce passthrough + repaint at a bounded rate

## Problem

Operator smoke test (ghostty inside cmux, warm cache, default `--parallel`=NumCPU): the #38
board rendered as "unorganized lines" — step lines fused at odd columns, output truncated
mid-word, no final board frame visible. Root cause: `boardWriter.Write` runs a full
erase-board → write-line → repaint-board cycle for EVERY passthrough write. A warm-cache
smoke emits hundreds of lines in ~2s → hundreds of 5-row erase/redraw cycles per second.
Idealized emulators apply each cycle atomically (pty + pyte replays of the exact operator
invocation render clean at 3 geometries); real terminals — and especially mux layers
(cmux/tmux re-interpret the escape stream) — paint asynchronously mid-sequence and drop/tear
under that flood. The strobe is a design bug regardless of terminal: nobody can read a board
repainting 500×/s.

## Spec

Make the compositor **double-buffered with a bounded flush rate** (~250ms):

1. `boardWriter.Write` COALESCES: append to the pending buffer; flush inline only if the
   flush budget elapsed (quiet path — a cold run's sparse lines still appear immediately) or
   the buffer exceeds a size cap (64KB — bound memory under flood).
2. `flush` = ONE atomic erase → dump all complete pending lines → repaint stored frame.
   Under flood the terminal sees ~4 large atomic updates/sec with the board stably pinned —
   no strobe, no per-line erase cycles.
3. `paint(lines)` stores the frame and flushes under the same budget (the sink's 100ms event
   throttle no longer drives the terminal directly).
4. The 500ms tick and `close()` force-flush (leftover pending lines + the final frame are
   never lost; close stays idempotent, cursor restored).
5. Clock injected into `boardWriter` (scripted in tests — controllable-time posture).

Escape hatch unchanged: `--no-tui` (and any redirect) keeps plain lines — the right mode for
hostile mux layers if any residue remains.

## Done when

- Under a burst (many writes, scripted clock inside the budget window) the underlying writer
  sees ONE erase+repaint per budget window, not per line; all passthrough bytes come out in
  order; nothing is lost.
- Quiet writes (≥budget apart) flush inline (a cold run's step lines appear immediately).
- `close()` flushes pending + final frame (existing pin-bottom/close tests keep passing,
  updated for the injected clock).
- A live warm-cache smoke on a real pty shows bounded repaint counts (erase sequences ≈
  run-seconds × 4, not ≈ line count).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.10 impl=0.35
item: atlas-docs          design=0.02 impl=0.10
design-buffer: 0.30
total: 0.61
```

One focused module touch (boardWriter coalescing + clock injection + test updates) + the
live-pty verification and atlas/Log sweep. Calibration doc [stale] (#127) — provisional.

## Plan

- [x] TDD in `cmd/metis/board.go`/`board_test.go`: inject clock; pending-coalesce + budgeted
  `flushLocked` (erase → dump → redraw); paint stores + budget-flush; tick/close force.
  Update the wiring (`newBoardWriter(w, now)`) + existing tests.
- [x] Live pty verification (warm smoke): erase count bounded; final frame intact.

## Log

### 2026-07-15
- 2026-07-15: closed — TDD red-green (burst-coalesce, quiet-inline, size-cap); pre-#46 tests updated to injected clocks; full package green + vet clean. Live warm pty of the operator exact invocation (190x50): 2.4s wall, 7 erase cycles total vs ~150+ pre-fix (4Hz budget bound holds), pyte render 0 fused rows, final board frame intact.; review verdict: FIX-THEN-SHIP
- Filed from the operator's smoke test (ghostty in cmux): fused rows mid-run, truncated tail,
  no final frame. My pty+pyte replays of the same invocation render CLEAN — the corruption
  lives in real/mux terminal timing under the per-line repaint flood, so the fix targets
  sequence VOLUME (coalescing), not sequence correctness. Design: double-buffer + 250ms flush
  budget; quiet-path inline flush keeps cold runs feeling live. (§7 autonomous bugfix;
  simple work — plan in-issue, no separate plan file. ARCH-PURE: the budget/coalesce logic
  stays in the one compositor; renderer + sink untouched.)
- Implemented TDD (3 new tests red→green: burst-coalesce ≤5 erases/500ms with order+completeness,
  quiet-inline-flush, frozen-clock size-cap; pre-#46 tests updated to injected stepping clocks —
  two assertions had encoded the per-write-repaint semantics). flushLocked takes `now` as an arg
  (a re-read inside would break scripted clocks). tick() force-flushes after storing the frame —
  re-pins the board post-burst + keeps ETA/rate moving. Full package green + vet clean.
- **Live warm pty verification (the operator's exact invocation, 190×50):** wall 2.4s, **7 erase
  cycles total (pre-#46: ~150+)** — within the 4Hz budget bound; pyte render: 0 fused rows; final
  frame intact (outer 3/3, 3 ✓ rows, leaves line).
- Close review: **FIX-THEN-SHIP**, no Critical. Fixed pre-commit: the tick/forceFlush drain path
  pinned (TestBoardWriter_ForceFlushDrainsPending — frozen-budget pending write stays coalesced,
  forceFlush drains + re-pins, sp.tick() renders + force-paints). Reviewer's shell was down
  (static review); full suite + vet re-run green here post-fix.
