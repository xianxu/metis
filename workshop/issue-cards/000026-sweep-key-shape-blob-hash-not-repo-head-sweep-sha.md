---
id: '000026'
status: done
started: 2026-07-19T16:02:06-07:00
created: 2026-07-11
updated: 2026-07-19
actual_hours: 0.06
---

# sweep key = shape blob-hash, not repo HEAD sweep_sha

## Problem

The ledger's `sweep_sha` column (each row's first identity field) is the **workspace repo's
HEAD commit** (`sweepSHAOf` → first of `repo_shas`, `cmd/metis/ledger.go:56`; `repo_shas` = a single
git probe of the experiment dir, `record.go:60-64`). That's the wrong identifier for "which sweep
this row belongs to":

- **Coarse:** it's the whole-repo commit, so it moves on *any* file change in the repo, not just the
  shape file. Two *different* shapes committed at the same repo SHA share a `sweep_sha`.
- **Forces a commit:** you must commit `titanic-sweep.md` before a run produces a reproducible
  identity — you can't sweep a *dirty* (uncommitted) shape and later reconstruct exactly which shape
  bytes you swept.
- **Confusing once dirty sweeps are allowed:** keeping a repo-HEAD column alongside a content-hash
  key would just mislead (a row whose shape is dirty has a HEAD that doesn't contain that shape).
