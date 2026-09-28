---
id: '000050'
status: done
started: 2026-07-16T08:14:43-07:00
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.54
actual_hours: 0.25
---

# run-end summary — elapsed time, fingerprint, rows, and paste-ready next commands

## Problem

A sweep ends with the estimate line and a generic "ship via `metis select --promote`" hint.
The operator (2026-07-16) then has to scrape the cohort fingerprint out of the scrollback
(the #39 `recording under` line), remember the shape path, and assemble the follow-up
commands by hand. The run KNOWS all of it: wall-clock elapsed, the fingerprint it recorded
under, how many rows landed in which ledger, and what the sensible next commands are.
