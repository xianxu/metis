# Parallel-Run TUI Implementation Plan (metis#38)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A nested `metis run` on a TTY shows a live pinned-bottom board — overall progress, one row per in-flight outer fold (inner progress + incumbent best), completed folds collapsed to their held-out score, and a `leaves 8/8 · 42.5 folds/min · ETA 3m10s` line — while step logs keep scrolling above it; non-TTY/`--no-tui` output stays byte-identical to #30's plain lines.

**Architecture:** Presentation-only over #30's seam, exactly as designed: the `sweepProgress` sink grows per-pass state behind the SAME `forPass(i)` hooks (outer-fold identity via closure binding — zero `pkg/sampler` change, the issue's hard constraint), plus a timestamp-ring `movingRate` (injected clock). A pure `renderBoard(boardState) []string` produces the frame; a `boardWriter` (pin-bottom ANSI over the real stdout) interleaves passthrough writes above the board. Mode is chosen once at wiring: TTY → board, else → #30's plain-line emitter (`--no-tui` forces plain).

**Tech Stack:** Go stdlib ONLY — **design decision: hand-rolled ANSI pin-bottom, not bubbletea/tcell.** The issue sketched "a Go TUI lib (pick at design)"; picking neither is the design-time pick, justified: (1) the board is OUTPUT-ONLY — no keyboard, no focus, no alt screens; bubbletea's model/update/view earns its dependency tree with interaction we don't have (Simplicity First: the entire TUI is "N lines that repaint"); (2) metis is deliberately a 2-dep module (yaml + local ariadne) and the charm stack is ~15 transitive modules; (3) the sandbox has no proxy.golang.org route (offline-module lesson, workshop/lessons.md). The pin-bottom pattern is ~120 lines of stdlib. → issue `## Revisions` at change-code.

---

## The board (the deliverable, concretely)

```
metis: nested-CV titanic-sweep (35b4700e) — 3 outer × (12 configs × 3 inner)     ← passthrough scroll region
metis: run d758ec08… of experiment "titanic-sweep"                                  (step logs keep landing here)
⚡ step get-data (cache hit)
outer 1/3 · configs 14/36 · folds 47/108 · est 0.7980                            ← board (repainted in place)
  fold 0 ✓ held-out 0.7980
  fold 1 ▸ configs 8/12 · folds 25/36 · best 0.8340
  fold 2 ▸ configs 6/12 · folds 22/36 · best 0.8100
leaves 8/8 · 42.5 folds/min · ETA 3m10s
```

Board height = `2 + min(outerTotal, 12) + (1 if overflow)`: the aggregate line, ≤12 fold rows, an
`… +N more` overflow line when outerTotal > 12, the leaves line. No separator row (the plan-review
reconciliation — the sketch's earlier rule line is dropped; fewer painted lines, same information).

- Row 1 = #30's aggregate line (same `progressLine` core, sans prefix). One row per outer fold: `▸` in-flight (per-pass configs/folds counters + per-pass incumbent `best` by the objective direction — display-only, NOT the 1-SE rule), `✓` done (its held-out score from `driverEvent`'s `Point.Idx`), pending folds show `· fold 2 — queued`. Flat (1-config) runs: header + leaves line only (no fold rows — one pass, the aggregate line already says it).
- **Throughput (the operator's k10-probe requirement, #38 Log):** moving average, not instantaneous — a ring of the last 64 fold-completion timestamps (injected clock); `rate = n / (now − oldest)` — `now` in the denominator means a stall DECAYS the rate toward 0 live (the BLAS-thrash signature: "throughput ≈ 0 while the process looks alive" becomes visible in seconds). `ETA = remaining folds / rate`, against the seeded total (#30's `progressTotals`). Rendered `— folds/min · ETA ?` until n≥2.
- **Leaves:** `len(leafSem)/cap(leafSem)` — a gauge closure injected from `runOpts` (the #31 semaphore IS the occupancy; no new accounting).
- **Width source (plan-review finding — load-bearing, not cosmetic):** a wrapped board line breaks the cursor-up erase count (the repaint scheme assumes one physical row per rendered line). Stdlib-only pick: `$COLUMNS` env when set and parseable, else **80** — read ONCE at boardWriter construction (no SIGWINCH handling; a mid-run resize may transiently garble one frame, and the next full repaint's `\x1b[J` clear-below + re-truncated lines self-limit the damage — documented limitation, not handled). No ioctl/`unsafe`, no `x/term`. Long lines truncate with `…` at width−1.

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `passRow` + per-pass state in `sweepProgress` | `cmd/metis/progress.go` | modified |
| `movingRate` | `cmd/metis/progress.go` | new |
| `boardState` + `renderBoard` | `cmd/metis/board.go` | new |
| `progressLine` | `cmd/metis/progress.go` | unchanged (reused as the board's row-1 core) |

- **`passRow`** — one outer fold's live state: `{configK, foldK int; best float64; hasBest bool; done bool; heldOut float64}`. Folded by the EXISTING `forPass(i)` hooks (they gain per-pass writes beside the aggregate ones — the `_ = outer` placeholder becomes load-bearing, exactly as #30's seam note promised) and by `driverEvent` (its `Point.Idx` names the completed fold — the payload already carries it). `best` needs the objective direction: **reintroduce the `direction` param** #30's close dropped as vestigial (the close review predicted this: "#38 reintroduces it when the board renders per-fold incumbents").
  - **DRY rationale:** per-pass rows reuse the same event stream + hooks; no second instrumentation path (the issue's "no new instrumentation" constraint, honored structurally).
- **`movingRate`** — `{times []time.Time (ring, cap 64)}`; `add(t)`, `rate(now) (perMin float64, ok bool)` = `n/(now−oldest)`; `eta(now, remaining) (time.Duration, ok)`. Pure over passed-in times — table-tested with scripted instants, no sleeps (controllable-time posture).
- **`boardState` / `renderBoard(st boardState, width int) []string`** — the pure frame renderer: aggregate line (reusing `progressLine`'s core), fold rows (✓/▸/queued, overflow-capped), leaves+throughput+ETA line. Pure, golden-lite table tests (contains/not-contains + line count + width clamp). NO ANSI in the renderer — it returns plain lines; escape codes live only in `boardWriter` (the paint/content split keeps the renderer testable byte-for-byte).

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `boardWriter` | `cmd/metis/board.go` | new | the real stdout (ANSI repaint) |
| mode selection + `--no-tui` | `cmd/metis/main.go`, `run.go`, `sweep.go` | modified | TTY detection, flag |
| repaint ticker | `cmd/metis/sweep.go` | new | wall-clock ticker (thin shell) |
| leaf gauge | `cmd/metis/run.go` | modified | `leafSem` chan len/cap |

- **`boardWriter`** — the pin-bottom compositor, one mutex, **paint-only: it stores the last rendered frame and NEVER calls back into the sink** (the lock-order rule below). API: `Write(p)` (passthrough) = erase board region (cursor-up N + `\x1b[J` clear-down), write `p` (newline-completed if unterminated), repaint the STORED frame — exempt from any throttle (the board must be restored after every passthrough; the frame may be ≤500ms stale, refreshed by the next tick); `paint(lines []string)` = store + erase + redraw (called by the sink, ≥100ms event-throttle inside the sink); `close()` = final repaint, newline, `\x1b[?25h` cursor restore — **idempotent, installed as a `defer` at the construction site** so error returns never leak a hidden cursor (plan-review finding). ANSI: `\x1b[NA`, `\x1b[J`, `\x1b[?25l/h`.
- **Writer plumbing (plan-review CRITICAL — construction order, not call graph):** writer identity is *temporal*: `runExperiment` currently wraps `out` in `syncWriter` and constructs the fork-server pool (which CAPTURES that writer for its fallback notices) *before* parsing — so a boardWriter created later in `runShapeSweep` would leak two bypass routes (the pool's `noticeOnce`, and `captureSweepCode`'s `warnOnUncaptured(o.out, …)`). **Fix: reorder `runExperiment`** — parse the file FIRST (parsing writes nothing), decide the mode (`o.tui && isShape`), then wrap exactly one writer: board mode → `boardWriter` (it serializes internally — no syncWriter stacking), else parallel → `syncWriter`; assign it to BOTH the local `out` and `o.out`; only THEN `newServerPool(out)`. Every construction-time capture (pool, execs, sink, capture warnings) then holds the compositor. Task 4 carries an explicit bypass test for the two named routes.
- **Mode selection** — in `cmdRun`: `--no-tui` flag (default false); `runOpts.tui = !noTUI && stdout is a char device` (`os.Stdout.Stat(): Mode()&os.ModeCharDevice != 0` — stdlib isatty, no x/term dep); board engages iff `o.tui` **&& the parsed file is a shape** (decided in `runExperiment`, above). Piped/redirected/CI → plain #30 lines automatically (Done-when 2's byte-stable requirement — pinned by the existing no-escape-codes tests + a new explicit assertion). The single-experiment (non-sweep) path is untouched (nil sink, as today).
- **Repaint ticker + lock order (plan-review finding):** ONE global lock order, `sink.mu → bw.mu`, everywhere. The 500ms ticker goroutine calls `sp.tick()` — a SINK method that locks `sp.mu`, renders the frame (rate/ETA recomputed with a fresh `now`, which is the ticker's whole purpose — stall decay stays live), and hands lines to `bw.paint()` — never a boardWriter-first path (that would either invert locks via a state callback or repaint a frame whose ETA can't move). Started in board mode only; stopped via `defer` at sweep end. Wall-clock in the thin shell (tests call `sp.tick()`/`bw.paint()` directly, never tick).
- **Leaf gauge** — `runOpts` gains `leafGauge func() (busy, capacity int)` set where `leafSem` is made (`len(sem), cap(sem)`); threaded to the sink. Nil-safe (serial runs: no gauge → the leaves segment is omitted).

**Test surface:** pure — `movingRate` (scripted instants: warm rate, stall decay, ring wrap, n<2), `renderBoard` (nested mid-run / all-done / flat / overflow >12 folds / width clamp / queued rows), per-pass fold via hooks (two passes' events → two distinct rows; `driverEvent(Idx)` collapses the right row). IO-seam — `boardWriter` against a `bytes.Buffer` "terminal": passthrough bytes precede the repainted board, erase sequences present between frames, `close()` restores the cursor; deterministic (no ticker in tests — `repaint()` called directly). Degradation — non-TTY: existing #30 fixture pins (byte-stable plain lines) + explicit `--no-tui` + not-a-chardevice assertions (no `\x1b` anywhere). Real-TTY evidence at close: `script -q` allocates a pty around a real smoke sweep — the captured file shows repaint sequences + the final board (macOS ships `script`).

---

## Tasks

Single-pass close (one cohesive renderer feature; plain checkboxes, §3).

### Task 1: `movingRate` + per-pass rows in the sink (pure)

**Files:** modify `cmd/metis/progress.go`; test `cmd/metis/progress_test.go`.

- [ ] **Step 1: failing tests** — (a) `TestMovingRate`: scripted instants — 5 completions 1s apart, rate measured at now=t₀+4s → **75/min** (`n/(now−oldest)` = 5/4s; the plan-review corrected arithmetic); the same 5 measured at now=t₀+64s (a 60s stall) → rate < 5/min (now-in-denominator decay pinned explicitly); ring wraps at 64 (65th evicts the oldest); n<2 → `ok=false`. (b) `TestSweepProgress_PerPassRows`: two `forPass` hooks fed interleaved config/fold events with distinct `MeanSE` outs → the sink's `boardState()` snapshot has two `passRow`s with per-pass counters + each pass's own best (direction=maximize: higher mean wins; also assert minimize flips it); `driverEvent{Point: OuterFoldPoint{Idx: 1}, Out: 0.83}` marks row 1 done with heldOut=0.83, row 0 still in-flight.
- [ ] **Step 2: verify FAIL.**
- [ ] **Step 3: implement** — `movingRate` (ring; `rate(now)` = `float64(n) / now.Sub(oldest).Minutes()`); `sweepProgress` gains `direction string` back (constructor param — the #30 close-review note anticipated this), `rows []passRow` sized `outerTotal` (flat: nil), `rateRing movingRate` fed in the fold hook (every fold completion, any pass), and `boardState() boardState` (mutex'd snapshot: aggregate `progressState` copy + rows copy + rate/eta inputs). `forPass(i)`'s closures write `rows[i]` when `i >= 0` (the `_ = outer` placeholder becomes real); `driverEvent` uses `ev.Point.Idx`.
- [ ] **Step 4: PASS** — `go test ./cmd/metis/ -race -run 'TestMovingRate|TestSweepProgress'`.
- [ ] **Step 5: commit** — `#38: per-pass rows + moving-average rate in the progress sink (pure)`.

### Task 2: `renderBoard` (pure frame)

**Files:** create `cmd/metis/board.go` (renderer half) + `cmd/metis/board_test.go`.

- [ ] **Step 1: failing tests** — `TestRenderBoard` table: nested mid-run (3 folds: one ✓ with `held-out 0.7980`, one ▸ with `configs 8/12 · folds 25/36 · best 0.8340`, one queued) + leaves line `leaves 8/8 · 42.5 folds/min · ETA`; all-done (every row ✓, ETA omitted); flat (no fold rows — exactly 2 lines); overflow (14 folds → 12 rows + `… +2 more`); width clamp (narrow width → lines ≤ width, `…`-truncated); no-gauge (leaves segment absent); rate-unavailable (`— folds/min`). Assert NO `\x1b` in any rendered line (paint/content split).
- [ ] **Step 2: FAIL.** **Step 3: implement** `boardState` + `renderBoard` (reuse `progressLine`'s fraction/est helpers — extract `frac` if needed rather than duplicating, ARCH-DRY). **Step 4: PASS.** **Step 5: commit** — `#38: pure board frame renderer`.

### Task 3: `boardWriter` (pin-bottom ANSI compositor)

**Files:** `cmd/metis/board.go` (writer half) + `cmd/metis/board_test.go`.

- [ ] **Step 1: failing tests** — against a `bytes.Buffer`: (a) first `repaint()` paints the frame + hides the cursor; (b) a passthrough `Write("⚡ step x\n")` erases (cursor-up + `\x1b[J` present between frames), writes the passthrough line, repaints — the passthrough text appears BEFORE the last frame in the byte stream; (c) un-terminated passthrough writes get newline-completed before the board repaints (a leaf's partial line must not fuse into the board); (d) `close()` leaves the final frame + `\x1b[?25h`; (e) repaint throttle: two `repaint()`s 10ms apart (scripted clock) → one frame, forced `close()` still paints.
- [ ] **Step 2: FAIL.** **Step 3: implement** (one mutex; track painted-line-count for the erase; injected clock for the throttle). **Step 4: PASS.** **Step 5: commit** — `#38: pin-bottom boardWriter compositor`.

### Task 4: wiring — mode selection, `--no-tui`, ticker, leaf gauge

**Files:** modify `cmd/metis/main.go` (flag), `run.go` (isTTY + gauge + mode plumb through runOpts), `sweep.go` (board construction, ticker, finish); tests in `progress_test.go`/`shapesweep_test.go`.

- [ ] **Step 1: failing tests** — (a) fixture sweep with `runOpts{tui: false}` (the default in every existing test): output byte-free of `\x1b` and contains the #30 plain lines — the EXISTING pins already assert this; add the explicit `--no-tui`-equivalent assertion naming the mode field. (b) fixture sweep with `tui: true` + a `bytes.Buffer`: board frames present (`\x1b[?25l`, fold rows), NO `metis: progress` plain lines (the board replaces them), final frame shows `outer 2/2`. (CLI flag parse: extend the existing cmdRun flag test if one exists, else a small `run([]string{"run", "--no-tui", ...})` parse check.)
- [ ] **Step 2: FAIL.** **Step 3: implement** — `cmdRun`: `noTUI := fs.Bool("no-tui", false, "force the plain progress lines even on a TTY (metis#38)")`; `runOpts.tui = !*noTUI && isCharDevice(os.Stdout)` + `leafGauge`. **`runExperiment` reorder (the CRITICAL fix):** parse first → decide board mode (`o.tui && isShape`) → wrap ONE writer (boardWriter | syncWriter) into both `out` and `o.out` → `defer bw.close()` (idempotent — covers error returns) → only then `newServerPool(out)`. Sink in board mode: `maybeEmit` renders + `bw.paint(lines)` under the fixed `sink.mu → bw.mu` order instead of `Fprintln`; plain #30 lines suppressed. Start/stop the 500ms `sp.tick()` ticker around the sweep. **Bypass test (the review's two named routes):** in a board-mode fixture, force (i) a fork-server fallback notice and (ii) a degraded-capture warning — assert both land ABOVE the board (through the compositor: their text precedes the final frame, and no bare write follows the last erase sequence).
- [ ] **Step 4: PASS + full suite** — `go test ./cmd/metis/ -race -count=1`.
- [ ] **Step 5: commit** — `#38: TTY board wiring — --no-tui, char-device detection, ticker, leaf gauge`.

### Task 5: docs + real-pty evidence + close

- [ ] atlas/index.md (the #30 seam bullet grows the #38 half: board, boardWriter, mode rules) + kbench RUNBOOK (the "#38 pending" note → how the board reads, `--no-tui` for logs).
- [ ] Issue `## Revisions`: BOTH deliberate spec deviations — (1) the no-lib decision (rationale above); (2) **pinned-bottom board, not the spec's "full-screen curses board"** — full-screen (alt-screen) would hide the scrolling step logs, and losing that stream trades away the "downloading vs hung" signal the issue was filed to provide; pin-bottom keeps both.
- [ ] **Real evidence:** `script -q "$TMPDIR/board.txt" $TMPDIR/metis run --parallel 8 titanic-sweep-smoke.md` (pty-allocated real sweep) → the capture shows cursor-hide, repaint sequences, fold rows with live counters, the leaves/throughput line, and the final board; then the SAME sweep `> file` (no pty) → byte-clean #30 lines. Both land in `--verified`.
- [ ] `sdlc close --issue 38 --verified '<evidence>'`.

## Verification (Done-when → checks)

| Done-when | Check |
|---|---|
| TTY board: overall + per-in-flight-fold + incumbent + leaf occupancy; folds collapse to results | Task 2 renderer tests + Task 4 board-mode fixture + Task 5 pty capture |
| non-TTY/`--no-tui` = exactly #30's plain rendering, no escape codes | existing #30 pins + Task 4(a) + Task 5's redirected run |
| no `pkg/sampler` API change beyond #30's | structural: the diff touches `cmd/metis` only (review-checkable); per-fold identity via `forPass` closure binding |
| (Log) moving-average runs/sec + ETA vs computable total | Task 1 `movingRate` tests (incl. stall decay — the BLAS-thrash signature) + the board line in Task 2/5 |

## Revisions

*2026-07-15 — boundary review (metis#38 close, FIX-THEN-SHIP):* Core-concepts table correction:
`boardState` landed in `cmd/metis/progress.go` (beside the sink state it snapshots), not
`board.go` as the table said. Review fixes applied in-code before the close commit: (1) the
fork-server-notice bypass route now has a direct test (`TestServerPool_NoticeRoutesThroughBoard` —
pool built on the compositor, notice lands between erase and repaint); (2) `progressCore` extracted
(the board's row 1 shared the plain line's core instead of TrimPrefix string-coupling); (3)
`snapshotLocked` dedupes the two rows-copy sites. Deferred as noted (minor): overflow prefers
low-indexed rows; short-terminal height clamp (comment added beside the width note); `$COLUMNS`
usually unset → 80 (accepted trade).
