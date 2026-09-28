---
id: '000047'
status: done
started: 2026-07-16T00:10:45-07:00
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.28
actual_hours: 0.24
---

# board flashes on repaint — wrap flushes in DEC 2026 synchronized output

## Problem

Operator (2026-07-16, ghostty + iTerm2): the board visibly flashes on each flush. The #46
coalescing bounded the RATE (4Hz), but each flush is still erase-region → dump → redraw —
and a terminal that renders between the erase and the redraw shows a blank board for one
display frame. At 4Hz that reads as flashing.
