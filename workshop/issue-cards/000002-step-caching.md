---
id: '000002'
status: done
started: 2026-07-05T14:32:51-07:00
created: 2026-07-02
updated: 2026-07-05
estimate_hours: 3
actual_hours: 2.74
---

# Uniform DAG step caching: content-address step inputs, skip unchanged, recompute only what changed

## Problem

`metis run` re-executes the *entire* DAG every time (`cmd/metis/run.go` →
`Runner.Run` → TopoSort → execute-each; no skip/cache logic). So every run
re-downloads from Kaggle (`get-data`, network) and re-trains (compute) even when
nothing about those steps changed. For a **learning bench** whose loop is "tinker
one knob, re-run," that is needlessly slow and re-hits external services each run.
