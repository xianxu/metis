---
id: 000057
status: codecomplete
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.1
started: 2026-07-18T10:34:46-07:00
actual_hours: 0.15
---

# board scrolling log renders gray

## Problem

Third operator refinement on the #55 banding: the scrolling step log should render GRAY so
the footer (bold aggregate, default status) and the epilogue result carry the eye.

## Spec

Scroll-region chunks (both pending-dump sites: flushLocked + close) wrap in `\x1b[90m`
(bright-black — true gray; dim renders inconsistently) when color is on; step logs emit no
SGR of their own so the chunk wrap can't be cancelled mid-block. Frame + epilogue stay
un-grayed. NO_COLOR: no change (no SGR at all).

## Done when

- Scroll chunks (both dump sites incl. the close-time tail) gray-wrapped when color on;
  frame + epilogue un-grayed; NO_COLOR emits no SGR; atlas banding entry names the gray
  scroll surface.

## Plan

- [x] writeScroll helper (both dump sites); gray-wrap unit test; NoColor extension; suite

## Log

### 2026-07-18
- 2026-07-18: closed — gray-wrap unit test (scroll gray, frame+epilogue un-grayed), NoColor extended, board e2e + full -race green; actual 0.15h labeled judgment; review verdict: FIX-THEN-SHIP

### (built)
- writeScroll wraps both pending-dump sites; sgrGray added; unit test pins gray scroll +
  un-grayed frame/epilogue; NoColor test extended with a scroll write. Full -race green.
