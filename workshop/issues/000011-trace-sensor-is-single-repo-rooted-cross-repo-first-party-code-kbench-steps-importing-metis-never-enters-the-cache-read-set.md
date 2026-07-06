---
id: 000011
status: open
deps: []
github_issue:
created: 2026-07-05
updated: 2026-07-05
estimate_hours:
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

## Spec

Make the sensor **multi-root aware** so first-party code is captured from *every* repo on the
step's `METIS_STEP_PATH` (or, more simply, root `D` at the **traced module's own repo**, not the
sensor's). Options to weigh:
- Root the read-set at the traced target module's repo (`git rev-parse --show-toplevel` of the
  target's `__file__`), so `kbench.titanic.features` roots `D` at kbench and captures kbench code.
- Record first-party reads from a **set** of roots (the sensor's repo **and** each root on
  `METIS_STEP_PATH`), keyed per-repo (aligns with the point-address's per-repo `repo_shas`).
- Preserve the existing single-repo behavior when target and sensor share a repo (metis's own steps
  must not regress).

## Done when

- A step run through `metis.trace` from a **consumer** repo captures that repo's first-party code in
  `reads.json` (a test: trace a kbench-style module that imports metis, assert the consumer module
  appears in `D`).
- metis's own steps still capture metis code (no regression).
- Once landed, kbench#3's `titanic/features` wrapper can route through `metis.trace` and its
  feature-code edits correctly invalidate the metis#2 cache (kbench#3 currently defers to direct
  invocation with an honest atlas note — see kbench#3 plan decision #2).

## Plan

- [ ] Reproduce: a failing test tracing a consumer-repo module (imports metis) — assert its
      first-party code is absent from `D` today.
- [ ] Multi-root the sensor (root at the target module's repo and/or the `METIS_STEP_PATH` roots).
- [ ] Regression: metis's own steps still capture metis code.
- [ ] Atlas: the trace sensor's cross-repo behavior + the per-repo read-set.

## Log

### 2026-07-06
- Folded into the **reproducible dirty-run capture** effort (`workshop/pensive/2026-07-06-reproducible-dirty-run-capture.md`, item 1) alongside #13 (config immutability) + #14 (complete/harden capture). This issue is the cross-repo half: without it, a consumer repo's code (e.g. kbench `features.py`) never enters the capture closure, so #14's spec+single-run capture still can't pin a kbench step's bytes.

### 2026-07-05
- Filed from the kbench#3 plan-quality review (the plan proposed routing `titanic/features` through
  `metis.trace` for caching; the judge empirically found the read-set never captures kbench code).
  kbench#3 defers to direct invocation (no cache validation for its swept step) + an atlas note; this
  issue tracks the substrate fix that makes cross-repo swept-step caching real. Not needed for the
  kbench#4 acceptance demo (tiny Titanic data; features recompute per point is negligible, and
  different feature-sets differ by `with.json` so no false intra-sweep hit) — it's a
  correctness+efficiency fix for iterative cross-repo use.
