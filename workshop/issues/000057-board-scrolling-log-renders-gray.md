---
id: 000057
status: working
deps: []
github_issue:
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.1
started: 2026-07-18T10:34:46-07:00
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

-

## Plan

- [x] writeScroll helper (both dump sites); gray-wrap unit test; NoColor extension; suite

## Log

### 2026-07-18

### (built)
- writeScroll wraps both pending-dump sites; sgrGray added; unit test pins gray scroll +
  un-grayed frame/epilogue; NoColor test extended with a scroll write. Full -race green.
