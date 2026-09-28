---
id: '000015'
status: done
started: 2026-07-06T17:51:11-07:00
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 1.3
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
