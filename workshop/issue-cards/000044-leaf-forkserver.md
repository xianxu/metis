---
id: '000044'
status: done
started: 2026-07-15T10:33:09-07:00
created: 2026-07-14
updated: 2026-07-15
estimate_hours: 1.08
actual_hours: 2.35
---

# leaf executor: warm fork-server — kill per-step interpreter+import cost

## Problem

Every step execution spawns `uv run → python -m metis.trace <module>`: a fresh interpreter +
`import pandas, sklearn` = **~1.0s measured** (venv python: `import pandas, sklearn, numpy` 0.99s
vs bare interpreter 0.02s) + uv resolver overhead, before any step work runs. A kbench#9-scale
sweep executes ~5,000 leaf steps → ~10-15 min of a ~30-min wall clock is interpreter+import,
repeated identically. Observed: ~4.5s wall per train at 8 slots when the actual rf/gbm fit on
~800 rows is 0.3-1s. Operator question that filed this: "can the nested-CV run be a single
process, or a limited set of processes, each handling many training configurations?"
