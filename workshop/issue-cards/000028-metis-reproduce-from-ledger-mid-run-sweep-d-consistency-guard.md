---
id: 000028
status: open
created: 2026-07-11
updated: 2026-07-11
estimate_hours:
github_issue:
---

# metis reproduce from ledger + mid-run/sweep D-consistency guard

## Problem

There is no automatic way to reconstruct a recorded run's exact code state and re-run it. Today the
side-refs (`refs/metis/{runs,sweeps}/<id>`) *durably store* the code closure as full-tree overlay
snapshots (`cmd/metis/capture.go:54-78`), and each step's `record.json` carries `Code.Commit` +
`Code.D` `{repo,path,blob_hash}` — but nothing reads them back. Reconstruction is a manual `git
checkout` (and `promote`'s hint says "checkout `<sweep_sha>`", which is **wrong for a dirty run** —
the bytes live in the side-ref, and a closure can span multiple repos, so recovery is a per-repo
checkout of each step's recorded `Code.Commit`).

Two subtleties make "just restore one state and run" **incorrect in general**:
1. **Per-step D can differ within a run.** Each step is a separate process tracing its own reads, so a
   `.py` edited between step A and step B yields two blobs for one path. A run's code state is only a
   single well-defined tree when code is **consistent across its steps**. There is no guarantee code
   is constant during a run (or across a sweep's runs).
2. **Three levels — step │ run │ sweep.** A "code changed" event can occur mid-run or mid-sweep. A
   step is always internally consistent (one process); a run is consistent iff no file changed between
   its steps; a sweep iff no file changed across its runs.
