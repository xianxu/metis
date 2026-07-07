---
id: 000015
status: codecomplete
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 1.3
started: 2026-07-06T17:51:11-07:00
actual_hours: N/A
---

# Trace sensor misses the traced target module's own code + misclassifies exp-relative data as first-party (surfaced by the kbench#5 wrapper flip)

## Problem

Surfaced by the kbench#5 wrapper flip (routing `titanic/features` through `python -m metis.trace`).
Two distinct sensor defects, both making the flip ship a **broken cache key**:

**A. The traced target module's OWN code is not captured.** `metis.trace` runs the target as
`__main__` via `runpy.run_module(target, run_name="__main__")`, which runs the target's code in a
temporary `__main__` module and does NOT leave the target under its qualified name in `sys.modules`.
So `_snapshot_modules` captures the target's **parent packages** (`kbench/__init__.py`,
`kbench/titanic/__init__.py`) but **not the module itself** (`kbench/titanic/features.py`). Empirically
(kbench#5 verify): the `features` step's `reads.json` roots had `kbench/titanic/__init__.py` but NOT
`features.py`. It *happens* to work for metis's own steps (`metis/steps/train.py` IS captured) only
via the audit-hook `open` of a not-yet-`.pyc`-cached `.py` — bytecode-cache-sensitive luck, not
robustness. **Consequence: editing `features.py`'s logic would NOT invalidate the cache** — defeating
the entire point of tracing it.
- **Fix:** in `main()`, resolve the target module's file explicitly
  (`importlib.util.find_spec(target).origin`) and add it to `_reads` before running — so the traced
  module's own bytes are always in D, regardless of bytecode caching or runpy internals.

**B. Exp-relative DATA is misclassified as first-party code.** The multi-root sensor (metis#11)
captures kbench's Dataset files — `competition/titanic/data/titanic/{schema.json,*.parquet}` — into D,
because they sit under the kbench repo root and are NOT under `METIS_RUN_DIR` (kbench's adapt→features
data flow uses an exp-relative COMMITTED dir, not the run-dir upstream-artifact convention). So parquet
bytes enter the code read-set → they'd be committed to `refs/metis/*` side-refs (metis#8/#14 bloat) and
key the cache as if they were code. (This ties to the committed-dir-output cache note from kbench#3's
e2e — the Dataset isn't a CAS/run-dir artifact.)
- **Fix (decide at plan time):** exclude exp-relative data — e.g. skip reads under `METIS_EXP_DIR`'s
  data dir, or exclude by non-code extension (`.parquet`/`.csv`/large binaries), or (cleaner but
  bigger) route kbench's Dataset through the run-dir upstream-artifact convention so `METIS_RUN_DIR`
  exclusion already catches it. D is "first-party **code** + config", not data.

## Spec

Two focused `metis/trace.py` fixes (reads.json shape unchanged → no Go change):
- **Fix A — capture the traced target's own file.** In `main()`, resolve `importlib.util.find_spec(target).origin` and `_classify` it before `run_module`, so the traced module's own `.py` is always in `D` (runpy runs it as `__main__` → the `sys.modules` snapshot misses it; bytecode-cache luck for metis's own steps).
- **Fix B — `D = .py + uv.lock` only.** Enforce the sensor's intended contract (already asserted by `TestSensor_RecordsFirstPartyCodeReads`) in `_classify`: keep a read only if it ends `.py` or its basename is `uv.lock`; drop data (`.parquet`/`schema.json`/`.csv` — class-1, keyed via upstream output-hashes, never in D). metis#11's multi-root broke this by pulling in kbench's exp-relative Dataset.

## Done when

- A module traced via `metis.trace` has its **own** file in `reads.json` (test: trace a module,
  assert its `.py` — not just its package `__init__` — is in D; bytecode-cache-robust).
- Exp-relative data reads (`.parquet`/Dataset dir) do NOT enter D (test).
- **Unblocks kbench#5** (the wrapper flip): after this, routing kbench steps through `metis.trace`
  correctly captures + cache-validates their code without data leak.

Durable plan: `workshop/plans/000015-trace-target-and-data-plan.md`. Single-pass atomic. **Fix B settled:** enforce `D = .py + uv.lock` (the contract `TestSensor_RecordsFirstPartyCodeReads` already asserts) — an allowlist in `_classify`, no Go change (reads.json shape unchanged).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.15  impl=0.3
item: smaller-go-module      design=0.15  impl=0.3
item: milestone-review       design=0.0   impl=0.2
item: atlas-docs             design=0.05  impl=0.1
design-buffer: 0.15
total: 1.3
```

Σdesign 0.35 × 1.15 = 0.4025; Σimpl 0.90 × 1.00 = 0.90; total **1.3** (= `estimate_hours`). `smaller-go-module` ×2 = Fix A (target-module capture) + Fix B (.py/uv.lock allowlist + leaked-data regression); `milestone-review` = close; `atlas-docs` = the two rules.

## Plan

- [x] RED: trace a module, assert its own `.py` is in D (fails today — only parents captured).
- [x] GREEN: capture the target module's file explicitly in `main()`.
- [x] RED/GREEN: exp-relative data (`.parquet`) excluded from D.
- [x] atlas: the target-module capture + the data-exclusion rule.

## Log

### 2026-07-06
- 2026-07-06: closed — metis#15 done via TDD (inline; verified in main: go build+vet+test 9/9 ok, uv pytest 39 passed). Fix A: _capture_target(target) in main() resolves the traced module OWN file (importlib.util.find_spec → _classify) since runpy runs it as __main__ and the sys.modules snapshot misses it (bytecode-cache luck for metis steps). Fix B: enforce D = .py + uv.lock in _classify — data (.parquet/schema.json) dropped (metis#11 multi-root had leaked kbench exp-relative Dataset as first-party). Tests: test_capture_target_records_own_module_file + test_classify_excludes_data_keeps_code; existing 6 trace tests + metis#11 cross-repo/HIT-MISS green (reads.json shape unchanged → no Go change). VERIFIED VIA THE REAL SENSOR: python -m metis.trace metis.steps.predict → metis/steps/predict.py captured + no data leaked. Unblocks kbench#5 (the wrapper flip). --no-actual: the analysis interleaved with the kbench#5 flip verification (cross-repo detour) so the metis window (0.23h) undercounts — interleaved active-time practice.; review verdict: FIX-THEN-SHIP
- Filed from the kbench#5 flip verification: the flip proved the wrappers CAN route through
  `metis.trace` (kbench code roots appear) but (A) the swept module's own code is missing and (B)
  data leaks in — so the flip was reverted (broken cache key) and blocked on this. Both are metis
  sensor fixes on top of metis#11's multi-root.

### 2026-07-06 (implemented)
- **DONE via TDD.** Fix A: `_capture_target(target)` (importlib.util.find_spec → _classify the origin) in main() — the traced module's OWN .py is always in D (runpy runs it as __main__ → snapshot misses it). Fix B: `.py`/`uv.lock` allowlist gate in `_classify` — data (.parquet/schema.json) dropped (metis#11's multi-root had leaked kbench's exp-relative Dataset). Tests: test_capture_target_records_own_module_file, test_classify_excludes_data_keeps_code. **Verified via the REAL sensor:** `python -m metis.trace metis.steps.predict` → `metis/steps/predict.py` captured + no data leaked. Full suite: 39 py passed + go 9/9 ok. Unblocks kbench#5 (the wrapper flip).
- **Close-review round-1 (FIX-THEN-SHIP, 0 Critical / 1 Important — fixed).** Reconciled the  module docstring (the in-code definition of D): it still said "first-party code + config under the project root" but Fix B made D exactly  +  (a non-lock config is now dropped). Updated the docstring to state the allowlist + the target-capture (#15) + that data is class-1. Doc-only.
