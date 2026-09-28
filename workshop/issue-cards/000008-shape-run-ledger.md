---
id: '000008'
status: done
started: 2026-07-05T18:39:34-07:00
created: 2026-07-03
updated: 2026-07-05
estimate_hours: 3
actual_hours: 2.73
---

# Shape run-ledger: CSV sidecar keyed by free-param tuple + promotion to an experiment

## Problem

A sweep produces thousands of runs. Keeping each as a git-commit-per-run is affordable
but *unnavigable* (`git log` over 10k near-identical commits is unreadable). The runs
must be **durable AND navigable** — a structured, queryable table — without drowning
the ML engineer or churning git per run.
