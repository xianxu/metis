---
id: 000047
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.28
started: 2026-07-16T00:10:45-07:00
---

# board flashes on repaint — wrap flushes in DEC 2026 synchronized output

## Problem

Operator (2026-07-16, ghostty + iTerm2): the board visibly flashes on each flush. The #46
coalescing bounded the RATE (4Hz), but each flush is still erase-region → dump → redraw —
and a terminal that renders between the erase and the redraw shows a blank board for one
display frame. At 4Hz that reads as flashing.

## Spec

Wrap every flush (and close) in DEC private mode 2026 "synchronized output":
`\x1b[?2026h` before the erase, `\x1b[?2026l` after the redraw. Supporting terminals
(ghostty, iTerm2 ≥3.5, kitty, wezterm, alacritty ≥0.13) apply the whole update atomically —
zero flash; terminals without it ignore unknown private modes (safe no-op — degradation is
today's behavior, not corruption).

## Done when

- Every flushed update (Write-inline, paint, forceFlush, close) is bracketed by BSU/ESU in
  the byte stream (unit-asserted; balanced pairs, nothing outside close unbracketed).
- Live pty run shows the bracketing; existing board tests keep passing.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.02 impl=0.25
design-buffer: 0.30
total: 0.28
```

One seam (flushLocked/close bracketing) + bracket-balance test + live pty check.

## Plan

- [x] TDD: bracket assertions in board_test.go → emit in flushLocked + close.

## Log

### 2026-07-16
- Filed from the operator's UX pass (issue 1 of 3: flashes / startup delay / 3h ETA). BSU/ESU
  is the standard flicker cure for erase+redraw compositors; private-mode no-op elsewhere.
- Implemented TDD (bracket-balance + every-erase-inside-bracket assertions red→green); flushLocked
  + close both bracket. Side-fix caught by -race: the #46 steppingClock test helper raced
  (runOpts.now is called from concurrent ParExec goroutines for record timestamps) — mutexed; the
  race had also produced a pathological 169s test run, now 2.6s. Live pty: BSU=23/ESU=23 balanced,
  every erase inside a bracket. Full suite green.
