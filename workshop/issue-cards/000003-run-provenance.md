---
id: '000003'
status: done
started: 2026-07-05T13:37:57-07:00
created: 2026-07-02
updated: 2026-07-05
estimate_hours: 1.9
actual_hours: 0.94
---

# Run provenance: snapshot the resolved pipeline config (+ experiment git sha) so ## Runs is knob→score legible

## Problem

A run records **metrics but not the config that produced them**. `run.json` holds
`{id, experiment, seed, metrics, artifacts}` — no consolidated pipeline/steps
block. The runner does write each step's resolved `with.json` into
`runs/<id>/<step>/with.json`, so config *is* captured per-step in the run dir; but:

- The experiment `.md` frontmatter is **mutable and never snapshotted**. Edit a
  knob (e.g. `model: logreg`→`rf`) and re-run, and `## Runs` just appends a new
  metric line — you **cannot tell which frontmatter produced which score**.
- Reusing a run-id overwrites the per-step `with.json`, losing the prior inputs.

For a bench whose entire value is **knob → score**, this is the central miss.
Today's v0 workaround (a real convention, worth documenting): **treat a run-bearing
experiment file as ~immutable; fork a new file per variation** so each file owns its
own `## Runs` history. Provenance-snapshot is what would later make in-place editing
safe.
