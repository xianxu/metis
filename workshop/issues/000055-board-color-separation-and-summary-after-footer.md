---
id: 000055
status: working
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.72
started: 2026-07-18T09:23:50-07:00
---

# board color separation and summary after footer

## Problem

Two operator asks from the first full k10 run on the new stack (2026-07-18, cohort 48b04388):
(1) the scrolling step log and the pinned footer are visually indistinguishable — color should
separate them; (2) the run RESULT (the honest-estimate line + the #50 summary with its
paste-ready `next:` commands) prints BEFORE the footer's final frame, so the most important
output ends up buried above the board — the terminal ends on the status line instead of the
result.

## Spec

- **Color lives in the PAINTER, never the content** (the #38 paint/content split: renderBoard
  keeps returning plain lines — the pyte-replay + byte-clean tests stay untouched). `redraw`
  classifies frame lines and wraps AFTER truncation: a new full-width DIM `─` separator line
  above the frame (the band boundary), the aggregate line BOLD, `✓` green / `▸` yellow on the
  fold-row glyphs, the status line DIM. Scrolling log stays unstyled (pristine copy-paste).
- **Gating:** board mode only (already TTY-gated) AND `NO_COLOR` unset (the standard env
  convention). With NO_COLOR set, output is byte-identical to today. Redirected/`--no-tui`
  runs: zero SGR anywhere (the existing byte-clean invariant).
- **Result after the footer:** boardWriter gains an epilogue buffer (`epilogueWriter()
  io.Writer`); `close()` paints the final frame, restores the cursor, THEN flushes the
  epilogue. `reportEstimate` + `printRunSummary` route through a `summaryWriter(out)` helper:
  the board's epilogue when out is a board, else out unchanged (plain/redirected ordering is
  already correct). Final terminal state: log → separator → footer → estimate + summary +
  `next:` block last.

## Done when

- Board-mode run ends with the estimate + summary AFTER the last footer line (order asserted
  on the raw writer in a board e2e).
- Color on: separator + bold aggregate + green ✓ + dim status present as SGR in the raw
  stream; pyte-replayed EXISTING lines' text unchanged (the separator is one NEW row, expected); NO_COLOR → byte-identical to pre-#55; redirected
  run → zero ESC bytes (existing invariant test extended or cited).
- erase/painted bookkeeping counts the separator line (no ghost line on erase — the #38
  cursor-math invariant).

## Done when

-

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.05 impl=0.30
item: smaller-go-module   design=0.04 impl=0.25
item: atlas-docs          design=0.01 impl=0.05
design-buffer: 0.15
total: 0.72
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Rows: (1) painter color pass + separator + erase bookkeeping; (2) epilogue channel + summary
rerouting + ordering/color tests; (3) atlas board paragraph touch.

## Plan

- [x] painter color: classify-and-wrap in redraw (post-truncation), separator line, painted-count fix, NO_COLOR gate
- [x] epilogue: buffer + close-flush; summaryWriter routing for reportEstimate + printRunSummary
- [x] tests: order-on-raw-writer e2e; SGR presence/absence (NO_COLOR byte-identity; redirected zero-ESC); pyte content unchanged
- [x] atlas + Log; close

## Log

### 2026-07-18

### 2026-07-18 (built)
- Painter-owned banding (redraw classifies post-clamp; renderBoard untouched — pyte content
  tests green as-is): dim separator rule (erase math counts it — pinned), bold aggregate,
  green ✓ / yellow ▸ glyphs, dim status. Color injected at construction; production wiring
  reads NO_COLOR once (empty ≠ set, per no-color.org). paint() carries width for the rule.
- Epilogue channel: reportEstimate + both printRunSummary sites route via summaryWriter →
  board epilogue, flushed after final frame + cursor restore; plain/redirect passthrough
  unchanged. Board e2e now asserts restore → estimate → summary ordering and that output
  ENDS with the next-hints; NO_COLOR SGR-free test; epilogue ordering + ghost-line test.
  Full -race suite green.
