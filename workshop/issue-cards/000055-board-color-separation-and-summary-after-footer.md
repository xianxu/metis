---
id: '000055'
status: done
started: 2026-07-18T09:23:50-07:00
created: 2026-07-18
updated: 2026-07-18
estimate_hours: 0.72
actual_hours: 0.7
---

# board color separation and summary after footer

## Problem

Two operator asks from the first full k10 run on the new stack (2026-07-18, cohort 48b04388):
(1) the scrolling step log and the pinned footer are visually indistinguishable — color should
separate them; (2) the run RESULT (the honest-estimate line + the #50 summary with its
paste-ready `next:` commands) prints BEFORE the footer's final frame, so the most important
output ends up buried above the board — the terminal ends on the status line instead of the
result.
