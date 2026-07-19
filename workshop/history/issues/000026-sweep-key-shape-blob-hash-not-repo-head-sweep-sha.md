---
id: 000026
status: done
deps: [000027]
github_issue:
created: 2026-07-11
updated: 2026-07-19
estimate_hours:
started: 2026-07-19T16:02:06-07:00
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

## Spec

Make the ledger's sweep identifier the **shape file's git blob-hash** (content hash of
`titanic-sweep.md`), not the repo HEAD. The machinery already exists: `addSpecToClosure`
git-`hash-object`s the `.md` into the captured closure (`capture.go:93,232-267`), and the
dirty/untracked side-ref commit (`refs/metis/{runs,sweeps}/<id>`) already stores its bytes so they're
recoverable (`capture.go:54-78` — a full-tree overlay snapshot). So:

- **Ledger first column = shape blob-hash.** "All variations of this shape we swept" = the distinct
  blob-hashes for that path in the ledger; each is retrievable via `git cat-file blob <hash>` (from
  the object DB / side-ref).
- **Enables dirty-shape sweeps:** hash-object gives the working-tree blob; the side-ref stores it →
  reconstructable without committing the shape first.
- **Drop the repo-HEAD `sweep_sha`.** The sweep-level identifier only needs to pin the *input* (the
  shape); code identity is pinned **per-step** in each row's read-set `D` (orthogonal, and finer than
  a repo commit). Repo HEAD as a coarse code+input proxy is redundant once the shape is
  content-addressed and code is per-step `D` — and misleading for dirty sweeps. (Operator: not
  convinced repo HEAD should be kept at all.)

**Open (see the point_address / dedup tension):** `point_address` currently includes `repo_shas`
(`pkg/record/address.go:33`) — it's the *pre-run* code proxy that today distinguishes "same config,
different code" runs (why the two cohorts don't collide). Dropping repo HEAD needs a replacement for
that role (a post-run code fingerprint over the run's `D` closure, or accepting a coarser dedup) —
tracked separately; this issue is the *sweep-file* identity, that one is the *run/code* identity.

## Done when

- The ledger identifies a sweep by the shape's blob-hash; the shape's bytes are retrievable from a row
  (incl. a dirty/uncommitted shape, via the side-ref).
- The repo-HEAD `sweep_sha` column is removed (or demoted to non-identity provenance) without losing
  the ability to distinguish runs (coordinate with the point_address/code-fingerprint decision).

## Plan

- [x] (spec) design the column change + point_address interaction — **RESOLVED: subsumed by #27**
  (the sibling "run/code identity" issue this was gated on). #27 landed both halves; no new code needed.
  See the 2026-07-19 Log for the verified trace.

## Log

### 2026-07-11
- Filed from an architecture walkthrough of sweep reproducibility. The blob-hash + side-ref machinery
  already exists (`capture.go`); this is a re-keying of the ledger's identity column, gated on the
  point_address/code-fingerprint decision (the "same shape, different code" identity — a sibling
  issue to file once that design converges).

### 2026-07-19 — CLOSED as subsumed by #27 (verified)
- 2026-07-19: closed — Subsumed by #27 (verified by code trace, no new code needed). Both Done-when met: (1) shape blob-hash IS the intent identity via PointAddress=hash(resolved_with,shape_blob_hash,seed); shape bytes recoverable from a row: point_addr->runs/<id>/record.json->Steps[].Code.D carries the shape .md (path,blob-hash) via addSpecToClosure->git cat-file blob <hash>, dirty-safe (hash-object -w, pinned by refs/metis/sweeps/<id>). (2) repo-HEAD sweep_sha column removed; runs still distinguished by point_addr+code_fingerprint; no identity term keys off repo-HEAD (only Code.Commit provenance remains). Continuation risk cleared: nothing keys off repo-HEAD sweep_sha. Reconciled stale sweep_sha/--sweep vocabulary in workshop/lessons.md. Residual "all shape variations swept" query demand-gated/deferred per operator (natural home = #28 metis reproduce).; review verdict: SHIP
Re-surveyed the current code before planning (the `deps:[27]` gate resolved — #27 is in history). Every
goal here is already MET by #27 (commits `8c7483a` "PointAddress = hash(resolved_with, shape_blob_hash,
seed)" + `cfafcfa` "CodeFingerprint over the run-end D closure"). Traced against the two Done-when items:

1. **"Ledger identifies a sweep by the shape's blob-hash; bytes retrievable from a row (incl. dirty)"** —
   MET. `PointAddress` (`pkg/record/address.go:36`) folds the shape's `git hash-object` (working-tree,
   dirty-safe — `shapeBlobHash`, `capture.go:309`) into the `point_addr` identity term. Recovery chain
   from a row: `point_addr` (= RunID) → `runs/<id>/record.json` → `Steps[].Code.D` carries the shape
   `.md`'s `(path, blob-hash)` (`addSpecToClosure` puts the spec in the D-closure) → `git cat-file blob
   <hash>` recovers the exact bytes, dirty or not (`hash-object -w` into the object DB, pinned by
   `refs/metis/sweeps/<id>`).
2. **"repo-HEAD `sweep_sha` column removed without losing run distinction"** — MET. The `sweep_sha`
   column no longer exists; the ledger keys on `code_fingerprint` + `point_addr`. `repo_shas` was dropped
   from `PointAddress`, `RunRecord`, and the fingerprint. Runs stay distinguished: `point_addr` folds
   shape-blob+config+seed; `code_fingerprint` folds the per-file D-closure. No identity term keys off
   repo-HEAD (the only repo-HEAD use left is `Code.Commit`, which is durable-side-ref provenance, not
   identity) — the continuation's "confirm nothing keys off repo-HEAD sweep_sha" risk is CLEARED.

**Residual NOT built (demand-gated):** the one thing #27 doesn't surface is the literal *"all variations
of this shape we swept = distinct blob-hashes for that path"* query — the shape blob-hash is folded into
the opaque `point_addr` + buried in `record.json`'s `Code.D`, not a first-class ledger column/view. Not
demanded by any arena; deferred (the natural home for a user-facing recovery command is #28's `metis
reproduce`). Operator decision 2026-07-19: close subsumed, skip the speculative legibility build.

**Side-quest done at close:** reconciled the stale `sweep_sha`/`--sweep` vocabulary in
`workshop/lessons.md` (the "re-run appends a cohort" lesson) to the post-#27 `code_fingerprint`/
`--fingerprint` reality.
