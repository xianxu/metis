---
id: 000028
status: open
deps: []
github_issue:
created: 2026-07-11
updated: 2026-07-11
estimate_hours:
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

## Spec

**Detect-and-refuse, don't silently mis-reproduce.**

- **Consistency check (the primitive).** At a chosen root level (`run` | `sweep`), verify **every path
  in the aggregated `D` closure has exactly one blob-hash** across all steps (a run) / all runs (a
  sweep). Consistent → a single restorable tree exists. Conflict → **loud error** naming the
  offending file + the two blobs ("code changed mid-{run,sweep}; not reproducible as a single state").
  This same primitive defines the "well-defined `code_fingerprint`" metis#27 needs.

- **`metis reproduce --at {run|sweep} <id>`.** Run the consistency check; if consistent, restore the
  recorded closure (checkout each repo's side-ref / `Code.Commit` into a scratch worktree — NOT a bare
  `git checkout HEAD`), then re-run. (The theoretically-complete alternative — DAG-ordered per-step
  restore+run to honor a mid-run change — is out of scope; we refuse that case instead.)

- **`metis verify-reproducible <row>` (related capability).** Restore the recorded code, re-run with
  `--cache=false` (force MISS), and assert the fresh `output_hash`/metrics **equal** the recorded
  ones. Stronger than a cache HIT (which only validates code files are unchanged and then *trusts* the
  output): this catches **non-captured nondeterminism** — an untraced input (data/binary, not in `D`),
  a seed leak, wall-clock, thread-ordering. A real reproducibility test.

## Done when

- A consistency check that, given a run/sweep, reports either a single restorable closure or a loud
  conflict naming the divergent file.
- `metis reproduce --at {run|sweep} <id>` restores the recorded closure (side-ref aware, multi-repo)
  and re-runs; refuses on an inconsistent closure.
- `metis verify-reproducible <row>` force-re-runs and asserts output == recorded (catches
  non-captured nondeterminism).

## Plan

- [ ] (spec) the consistency-check primitive first (shared with metis#27), then the two verbs.

## Log

### 2026-07-11
- Filed from a reproducibility architecture walkthrough. Deferred ("work on later"). Supplies the
  "consistent D closure" definition metis#27's `code_fingerprint` is gated on, and fixes the
  `promote` "checkout `<sweep_sha>`" hint (wrong for dirty/multi-repo runs — recover from the
  side-ref).
