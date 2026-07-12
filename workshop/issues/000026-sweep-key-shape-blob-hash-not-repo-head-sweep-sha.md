---
id: 000026
status: open
deps: []
github_issue:
created: 2026-07-11
updated: 2026-07-11
estimate_hours:
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

- [ ] (spec) design the column change + point_address interaction; then implement.

## Log

### 2026-07-11
- Filed from an architecture walkthrough of sweep reproducibility. The blob-hash + side-ref machinery
  already exists (`capture.go`); this is a re-keying of the ledger's identity column, gated on the
  point_address/code-fingerprint decision (the "same shape, different code" identity — a sibling
  issue to file once that design converges).
