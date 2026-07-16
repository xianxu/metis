---
id: 000011
status: done
deps: []
github_issue:
created: 2026-07-05
updated: 2026-07-06
estimate_hours: 2.35
started: 2026-07-06T15:42:13-07:00
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

Durable plan: `workshop/plans/000011-trace-multi-root-plan.md`. Single-pass atomic.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: typed-data-prototype   design=0.2   impl=0.4
item: smaller-go-module      design=0.2   impl=0.4
item: smaller-go-module      design=0.2   impl=0.45
item: milestone-review       design=0.0   impl=0.2
item: atlas-docs             design=0.05  impl=0.15
design-buffer: 0.15
total: 2.35
```

Σdesign 0.45 × 1.15 = 0.5175; Σimpl 1.15 × 1.00 = 1.15; total **2.35** (= `estimate_hours`). Widened for the Go surface the plan-judge flagged: sensor multi-root + reads.json v2 (`typed-data-prototype`); Go per-repo consumer (`smaller-go-module`); the cache store/validate symmetry (persisted per-repo `D` + `Validate`/`isHit`) + HIT→MISS test + capture multi-root (`smaller-go-module`); close review; atlas. kbench wrapper flip is a tracked follow-up, out of estimate.

## Plan

- [x] Reproduce + fix: two-repo tests (Python `_classify` groups by repo root; Go `buildD`/`isHit` repo-qualified).
- [x] Multi-root the sensor (`_repo_root` walk-up, per-read repo discovery, `reads.json` v2 `roots` map) + Go per-repo consumer + store/validate symmetry.
- [x] Regression: metis's own steps unbroken (single-repo case); stdlib/site-packages excluded (the multi-root walk-would-mis-root-stdlib bug, fixed via `_STDLIB_PREFIXES`).
- [x] Atlas: multi-root read-set + `reads.json` v2 + repo-qualified `D`.

## Log

### 2026-07-06 (implemented — fork)
- 2026-07-06: closed — Close-review round-1 (0 Critical / 2 Important, both fixed). Important-1: added the REAL cross-repo invocation-path test TestSensor_MultiRepo_CapturesConsumerCode (a second git repo importing metis, run through the actual python -m metis.trace sensor, asserts both repos in reads.json roots + the consumer module captured) — the Done-when had only direct-_classify/hand-built-D coverage (the invocation-path-test lesson). Important-2: git -C "" hash-object does NOT fail (no-op → resolves against cwd → returns a hash), so the legacy-entry soundness comment was wrong + HIT/MISS was cwd-dependent; added an explicit Repo=="" guard in hashDByRepo (error → cwd-independent MISS, symmetric with loadReadSet loud v1 rejection) + TestHashDByRepo_RejectsLegacyEmptyRepo; fixed the comment. go build+vet+test ./... 9/9 ok + uv run pytest 37 passed. --no-actual: fork impl (fork-compressed window, active-time-v3 misses forks).; review verdict: SHIP
- 2026-07-06: closed — metis#11 trace multi-root done (fork impl via TDD; verified in main: go build+vet+test ./... 9/9 ok, uv run pytest 37 passed). metis/trace.py: single _PROJECT_ROOT → per-read git-repo-root discovery (_repo_root walk-up for a .git dir OR file, cached; _STDLIB_PREFIXES exclusion); reads.json v2 = {roots: {<repo>: [rel]}, used_site_packages}. Go: readSet.Roots map; buildD repo-qualified (record.CodeRef gained Repo + CUE #CodeRef); cache store/validate symmetry via one shared hashDByRepo (per-repo git -C hash-object); loadReadSet rejects legacy v1 LOUD (no silent empty-D false HIT). The 3 heart-tests green + regression-proofed: two-repo→D, HIT→MISS-on-consumer-edit (the whole point), empty-D-guard. NEW BUG the fork caught+fixed: the walk mis-rooted the entire uv stdlib as first-party because HOME is a git repo (~/.local/.../python → HOME .git) → excluded stdlib prefixes. Deviation: per-repo record Commit is single-valued (the cache D is fully multi-root; per-repo commits are a metis#14 refinement, noted). --no-actual: fork impl (window = the 2 fork commits, ~0.42h fork-compressed = 0.18x est) — active-time-v3 cannot faithfully measure fork work; excluded per the fork active-time practice. kbench wrapper flip (route titanic/features through metis.trace) is a tracked kbench follow-up.; review verdict: FIX-THEN-SHIP
- **DONE via TDD.** Sensor multi-root (`metis/trace.py`: `_repo_root` walk-up for a `.git` marker [dir OR file], `_STDLIB_PREFIXES` exclusion, `reads.json` v2 `{roots: {repo: [paths]}}`); Go per-repo consumer (`readSet.Roots`, repo-qualified `buildD`, `record.CodeRef.Repo` + CUE, `cache.Validate` ref-hasher); store/validate symmetry (`recordMiss` + `isHit` both group by repo via `hashDByRepo`); capture (`sweepClosure` per-root, `captureSweepCode` loops roots + per-repo side refs); `loadReadSet` rejects legacy v1 LOUD (the empty-D false-HIT guard). Removed the now-dead `cachingExecutor.projectRoot`. **Three heart-tests green + regression-proofed:** two-repo→D (`test_classify_groups_reads_by_repo_root`, `TestBuildD_MapsReadsToCodeRefs`), **HIT→MISS on consumer edit** (`TestCachingExecutor_MultiRepoDMissesOnConsumerEdit` — breaking per-repo grouping fails it), empty-D guard (`TestLoadReadSet_RejectsLegacyV1`). 9 Go pkgs + 37 Python tests green. **Bug caught:** the multi-root walk mis-rooted the whole uv stdlib under a git-tracked HOME → excluded Python-install prefixes.
- **kbench wrapper flip is a follow-up (Task 3.3, NOT done here — metis#11 is metis-only):** `kbench/steps/titanic/{adapt,features,submission}` can now route through `python -m metis.trace kbench.titanic.<mod>`; kbench#3 deferred `features` to direct invocation with an atlas note — flip it in a kbench change after this merges.

### 2026-07-06 (close-review round-1: FIX-THEN-SHIP, 0 Critical / 2 Important — both fixed)
- **Important (invocation-path test gap — the lesson I added this session):** the Done-when's cross-repo test was proven only via direct `_classify`/hand-built-D, not the REAL `python -m metis.trace` path. Added `TestSensor_MultiRepo_CapturesConsumerCode` — a second git repo with a package importing metis, run through the actual sensor (`uv run python -m metis.trace consumerpkg.mod`), asserting BOTH repos land as `reads.json` roots + the consumer module is captured. **Important (wrong soundness comment):** the legacy-entry comment claimed `git -C "" hash-object` fails→MISS, but `-C ""` is a no-op that resolves against cwd (returns a hash) — made HIT/MISS cwd-dependent. Added an explicit `Repo==""` guard in `hashDByRepo` (error→cwd-independent MISS, symmetric with loadReadSet's loud v1 rejection) + `TestHashDByRepo_RejectsLegacyEmptyRepo`; fixed the comment. go test ./... 9/9 + uv pytest 37 green.

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
