---
id: 000038
status: codecomplete
deps: [metis#30]
github_issue:
created: 2026-07-14
updated: 2026-07-15
estimate_hours: 2.19
started: 2026-07-15T17:24:43-07:00
actual_hours: 1.50
---

# parallel-run TUI — live hierarchical progress board over the #30 event stream

## Problem

With metis#31, a nested sweep fans out across NumCPU leaves — 5 outer folds × 99 configs × 5 inner
folds = 2,475 fold runs executing concurrently — and the terminal shows **nothing** until the pipe
flushes at exit (felt acutely on the metis#35 honest-beat run: minutes of silence, no way to tell
"downloading" from "hung" from "3/5 outer folds done"). metis#30's aggregated single line fixes
blindness for a SERIAL mental model, but under parallelism one line can't render what's actually
happening: several outer folds in flight at once, each with its own inner progress, plus a shared
leaf-semaphore occupancy. The operator (2026-07-14) wants a TUI/curses implementation so parallel
progress is comprehensible.

## Spec

A live terminal progress board rendered from the #30 event stream (`SizeHint` + per-`Tell`
`ProgressEvent`s) — presentation only, no new instrumentation in `pkg/sampler` beyond what #30 adds:

- **Hierarchical board**, one row per active level: overall `outer j/k`, then a row per in-flight
  outer fold (`fold 2 · inner 47/99 · best 0.834`), plus a leaf line (`leaves 8/10 busy`, from the
  #31 semaphore occupancy). Completed folds collapse to their result (`fold 0 ✓ est 0.812`).
- **Running estimates in place**: incumbent per level from the #30 event payload (best-so-far
  inner; outer mean±SE as folds land).
- **Terminal-shaped first** (single-threaded-attention): a full-screen curses board when stdout is
  a TTY; **degrade to #30's aggregated line(s) when piped/non-TTY** (CI, logs, `metis run ... >
  file`) — the TUI must never corrupt captured logs. `--no-tui` forces the plain path.
- Implementation sketch: a Go TUI lib (bubbletea or tcell — pick at design; bubbletea's
  model/update/view fits the event-stream shape) consuming the same injected `progress` callback,
  in `cmd/metis` only (ARCH-PURE: `pkg/sampler` stays pure; the callback remains the seam).
- Explicitly out: historical run browsing, ledger visualization, anything beyond the live run.

## Done when

- A nested `metis run` on a TTY shows a live board: overall outer progress, per-in-flight-fold
  inner progress + incumbent, leaf occupancy; folds collapse to results as they complete.
- Piped / non-TTY / `--no-tui` output is exactly the #30 plain rendering (byte-stable, no escape
  codes) — verified by running with stdout redirected.
- No `pkg/sampler` API change beyond #30's (`SizeHint` + `progress`) — the TUI is a `cmd/metis`
  renderer over the same seam.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.10 impl=0.35
item: smaller-go-module   design=0.10 impl=0.30
item: smaller-go-module   design=0.15 impl=0.40
item: smaller-go-module   design=0.10 impl=0.40
item: atlas-docs          design=0.02 impl=0.13
design-buffer: 0.30
total: 2.19
```

Row 1 = per-pass rows + `movingRate` in the sink (pure, Task 1). Row 2 = `renderBoard` pure frame
(Task 2). Row 3 = `boardWriter` ANSI compositor (Task 3 — the genuinely novel piece: erase-count
bookkeeping, passthrough interleave, idempotent close; priced highest design). Row 4 = wiring
(Task 4 — the runExperiment reorder is a structural touch on the writer plumbing + the two-route
bypass test; the review's Critical lives here). `atlas-docs` = atlas/RUNBOOK/Revisions + the pty
(`script -q`) + redirected evidence runs. Calibration doc [stale] (#127) — provisional.

## Plan

Durable plan: `workshop/plans/000038-parallel-run-tui-plan.md` (fresh-eyes reviewed; 1 Critical +
2 Important folded: runExperiment writer-plumbing reorder — construction-time captures (forkserver
pool, o.out) would bypass a late-constructed compositor; sink.mu → bw.mu global lock order with the
ticker routed through the sink; $COLUMNS/80 width source named). Single-pass close, no milestones.

- [x] Task 1: per-pass rows + `movingRate` (ring, stall-decay) in the sink — pure, TDD
- [x] Task 2: `renderBoard` pure frame (fold rows ✓/▸/queued, overflow, width clamp, no ANSI)
- [x] Task 3: `boardWriter` pin-bottom compositor (erase/passthrough/repaint, idempotent close)
- [x] Task 4: wiring — `--no-tui`, char-device detect, runExperiment reorder, ticker, leaf gauge,
  bypass test (forkserver notice + capture warning route through the compositor)
- [x] Task 5: docs + Revisions (no-lib; pinned-bottom vs full-screen) + pty/redirect evidence + close

## Log

### 2026-07-14
- Filed by operator request during the metis#35 honest-beat run — the run sat minutes with zero
  output (pipe-buffered, no progress channel), indistinguishable from a hang; #31 parallelism makes
  a single aggregated line (the #30 renderer) insufficient for comprehension. Layering: #30 owns
  the instrumentation (SizeHint + progress callback), this issue owns the TUI presentation; deps
  reflect that.
- Operator request (2026-07-14, during the metis#42 k10 probe): the board must carry a
  **throughput line — a moving-average runs/sec (or trains/min) + implied ETA against the known
  total** (`m outer × configs × k inner` is computable up front from the shape). During the probe
  this was reconstructed by hand (`grep -c "✓ step train"` + wall clock ≈ 107/min → ~25 min ETA);
  it also would have caught the BLAS-oversubscription thrash in seconds (throughput ~0 while the
  process looked alive) instead of via system load inspection. Moving average, not instantaneous —
  per-leaf times vary by config (rf500 ≫ logreg).

### 2026-07-15
- 2026-07-15: closed — TDD per task; full module green, -race on touched pkgs, vet clean. Real pty evidence (script -q around a real COLD smoke sweep, comment-edit methodology, reverted): 246 repaints, cursor hide/restore exactly once, leaves gauge live 0/8-8/8, moving rate 664-1890 folds/min, in-flight rows with per-fold incumbents, final frame outer 3/3 + 3 held-out rows + est 0.8103±0.0062. Redirected contrast (same binary): 0 escape bytes, the 6 plain #30 lines. The plan-review Critical (writer-identity-is-temporal) fixed structurally (parse-first reorder) + the o.out bypass route pinned by test. pkg/sampler untouched (diff is cmd/metis only). (Re-run: first review died on an API stream idle timeout — sidecar held zero content.); review verdict: FIX-THEN-SHIP
- Unblocked (metis#30 merged — PR #27 — with this issue's seam designed in: per-pass `forPass(i)`
  hooks carry outer-fold identity via closure binding; totals seeded; clock injected). Claimed +
  start-plan; durable plan authored + fresh-eyes reviewed (1 Critical + 2 Important, all folded —
  see Plan). Lessons persisted (writer-identity-is-temporal; ticker lock-order; width mechanism).
- Design decisions (full rationale in the plan): **hand-rolled ANSI pin-bottom, no TUI lib**
  (output-only board — no interaction to earn bubbletea's tree; metis stays a 2-dep module;
  sandbox lacks a module-proxy route) — the spec's "pick at design" pick, ARCH-Simplicity.
  **Pinned-bottom over full-screen curses**: step logs keep scrolling above the board — hiding
  them would lose the "downloading vs hung" signal this issue was filed for. Throughput = ring of
  last 64 fold completions, `rate = n/(now−oldest)` — a stall DECAYS the rate live (the k10-probe
  BLAS-thrash signature becomes visible in seconds). Leaves = `len/cap(leafSem)` (the #31
  semaphore IS the occupancy — no new accounting, ARCH-DRY). `pkg/sampler` untouched (the spec's
  hard constraint): everything rides #30's hooks (ARCH-PURE: renderer pure, ANSI only in the
  compositor).

## Revisions

### 2026-07-15 (at design, implemented same-day)
1. **No TUI library** (spec sketched "bubbletea or tcell — pick at design"): the board is
   output-only — no keyboard, no focus, no alternate screens — so a model/update/view framework
   earns nothing; metis stays a 2-dep module; the sandbox lacks a module-proxy route. Hand-rolled
   ANSI pin-bottom, ~120 lines of stdlib (`renderBoard` pure + `boardWriter` compositor).
2. **Pinned-bottom board, not the spec's "full-screen curses board"**: full-screen (alt-screen)
   would hide the scrolling step logs — losing exactly the "downloading vs hung" signal this
   issue was filed to provide. The board pins to the bottom; everything else scrolls above it
   through the compositor.

## Log (continued)

### 2026-07-15 (implementation)
- Tasks 1–4 TDD: per-pass rows + `movingRate` (stall-decay pinned) · pure `renderBoard`
  (✓/▸/queued, overflow, width clamp, zero ANSI) · `boardWriter` (pin-bottom; erase-order test;
  held unterminated tails; idempotent close) · wiring (parse-first runExperiment reorder — the
  plan-review Critical: the fork pool + o.out captured the pre-board writer; the o.out bypass
  route is pinned by test; `--no-tui`; char-device detect; 500ms sink-first ticker; leafSem
  gauge). Full module green + `-race` on touched packages; vet clean.
- **Real evidence:** pty via `script -q` (sandbox blocks openpty — loud override, like pushes) on
  the real smoke sweep, COLD cache (comment-edit methodology, reverted): 246 repaints, cursor
  hide/show exactly once, leaves gauge live (0/8→8/8), moving rate 664→1890 folds/min, in-flight
  rows with incumbents (`fold 0 ▸ configs 2/4 · folds 11/12 · best 0.8064`), final frame
  `outer 3/3 … est 0.8103 ± 0.0062` + 3 ✓ rows. Redirected contrast (same binary, no pty):
  0 escape bytes, the 6 plain #30 lines. leaves shows 0/8 on a warm-cache run (no leaf ever
  held) — expected, not a bug.
- Close review (re-run after a transport-failed first attempt — the sidecar held only an API
  stream-timeout, zero review content; announced loudly): **FIX-THEN-SHIP**, no Critical.
  Fixed pre-commit: the fork-server-notice bypass route pinned directly
  (TestServerPool_NoticeRoutesThroughBoard); progressCore extracted (no TrimPrefix coupling);
  snapshotLocked dedupe; plan table corrected via ## Revisions; height-clamp limitation
  documented beside the width note. Important #2 (kbench RUNBOOK unverifiable from this repo):
  confirmed landed — kbench commit c7b2766. Reviewer's Bash was broken (static review); full
  suite + -race re-run green here post-fixes.
