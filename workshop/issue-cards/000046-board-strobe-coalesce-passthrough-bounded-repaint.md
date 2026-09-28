---
id: '000046'
status: done
started: 2026-07-15T23:19:12-07:00
created: 2026-07-15
updated: 2026-07-15
estimate_hours: 0.61
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
