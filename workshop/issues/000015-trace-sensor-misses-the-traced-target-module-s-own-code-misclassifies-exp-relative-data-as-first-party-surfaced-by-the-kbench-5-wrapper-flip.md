---
id: 000015
status: working
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours:
started: 2026-07-06T17:51:11-07:00
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

## Done when

- A module traced via `metis.trace` has its **own** file in `reads.json` (test: trace a module,
  assert its `.py` — not just its package `__init__` — is in D; bytecode-cache-robust).
- Exp-relative data reads (`.parquet`/Dataset dir) do NOT enter D (test).
- **Unblocks kbench#5** (the wrapper flip): after this, routing kbench steps through `metis.trace`
  correctly captures + cache-validates their code without data leak.

## Plan

- [ ] RED: trace a module, assert its own `.py` is in D (fails today — only parents captured).
- [ ] GREEN: capture the target module's file explicitly in `main()`.
- [ ] RED/GREEN: exp-relative data (`.parquet`) excluded from D.
- [ ] atlas: the target-module capture + the data-exclusion rule.

## Log

### 2026-07-06
- Filed from the kbench#5 flip verification: the flip proved the wrappers CAN route through
  `metis.trace` (kbench code roots appear) but (A) the swept module's own code is missing and (B)
  data leaks in — so the flip was reverted (broken cache key) and blocked on this. Both are metis
  sensor fixes on top of metis#11's multi-root.
