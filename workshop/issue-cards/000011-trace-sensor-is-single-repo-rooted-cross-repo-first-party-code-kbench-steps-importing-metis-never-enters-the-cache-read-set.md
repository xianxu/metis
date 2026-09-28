---
id: '000011'
status: done
started: 2026-07-06T15:42:13-07:00
created: 2026-07-05
updated: 2026-07-06
estimate_hours: 2.35
actual_hours: N/A
---

# Trace sensor is single-repo-rooted — cross-repo first-party code (kbench steps importing metis) never enters the cache read-set

## Problem

`metis/trace.py` computes `_PROJECT_ROOT` from its own `__file__` (= the **metis** repo) and
`_classify` drops any read whose path isn't under that root (`if not
ap.startswith(_PROJECT_ROOT + os.sep): return  # another repo → not first-party`). So when a
**consumer repo's** step is run through the sensor — e.g. `python -m metis.trace
kbench.titanic.features` — the read-set `D` captures **only metis's own modules**; the consumer's
first-party code (`kbench/titanic/features.py`, its `group_*` fns) is captured **0 times**.

Empirically confirmed (kbench#3 plan-quality probe, 2026-07-05): the emitted `reads.json` shows
`"project_root": "/Users/xianxu/workspace/metis"` and `reads` = `[metis/__init__.py, dataset.py,
io.py, schema.py, trace.py]` — no kbench paths.

**Consequences (all silent):**
- Editing a consumer step's own logic does **not** change `D` → the metis#2 validating trace still
  HITs the stale cache → a sweep returns outputs computed by *old* consumer code. A correctness
  landmine in exactly the cross-repo topology (kbench step imports metis) the whole project uses.
- Inverted invalidation: a change to *metis* busts the consumer step (metis modules are in `D`),
  but a change to the step's own code does not.

This is why the two existing kbench wrappers (`adapt`, `submission`) deliberately run the module
**directly** (bypassing `metis.trace`) — so no consumer step is currently cache-validated at all.
The substrate's cross-repo caching has never actually been exercised (metis#2 was built + tested
single-repo).
