---
id: 000027
status: open
deps: [000028]
github_issue:
created: 2026-07-11
updated: 2026-07-11
estimate_hours:
---

# run/code identity split: intent-identity + post-run code fingerprint

## Problem

A run's identity is minted **before** the run (to name the run-dir and to dedup the ledger), but its
true **code identity** (the read-set `D`) is only known **after** the run (D is discovered by tracing
execution). Today metis papers over this tension by folding the **workspace repo HEAD** (`repo_shas`)
into `point_address` = `CanonicalHash(resolved_with, repo_shas, seed)` (`pkg/record/address.go:33`).
That repo-HEAD term is a *pre-run code proxy* — it's what makes two runs of the **same config but
different code** get distinct `point_address`es (why the two Titanic sweep cohorts don't collide).

Consequences:
- **Coarse + commit-forcing:** repo HEAD moves on any repo change; a *dirty* run can't be
  content-identified (two dirty variants share HEAD → same `point_address` → the ledger dedups → one
  **silently overwrites** the other, losing "which variations we swept").
- **Blocks metis#26:** #26 wants to drop the repo-HEAD `sweep_sha`. But dropping it *without a
  replacement for the code-identity role* re-introduces exactly the same-config-different-code
  collision. So #26 depends on this issue.

The deeper fact (surfaced in the walkthrough): **you cannot put precise code identity in the pre-run
key** — it's runtime-discovered. And per-step `D` can even differ *within* a run (each step is a
separate process; a file edited between steps yields two blobs for one path), so a run's code identity
is only well-defined when code is *consistent across its steps* (see metis#28 for the guard).

## Spec

Split the single conflated identity into two:

1. **Intent identity (pre-run, content-addressed, names the dir).**
   `intent = CanonicalHash(resolved_with, shape_blob_hash, seed)` — pure inputs, computable before the
   run. Replaces `repo_shas` in the addressing with the **shape's blob-hash** (ties to metis#26). This
   also **makes single-run and sweep-point dirs symmetric**: BOTH are named by their content-address
   (today a single run is `run-<timestamp>`, a sweep point is its `point_address` — drop the
   asymmetry; `--run` becomes an optional human alias/symlink, not the identity). "What I meant to
   run."

2. **Realized-code fingerprint (post-run, recorded in the row/record).**
   `code_fingerprint = CanonicalHash` of the run's **full, consistent `D` closure** (every step's
   `{repo,path,blob_hash}`), well-defined **only if** the closure is internally consistent (one blob
   per path across all steps — the metis#28 consistency check). "What code actually ran."

3. **Ledger dedup key = (intent, code_fingerprint).**
   Same config+shape, different code → same `intent`, *different* `fingerprint` → **both rows kept**
   (the "all variations we swept" goal). Same everything → dedup. Repo HEAD drops out entirely: intent
   uses the shape blob-hash, code identity uses the actual `D` closure. No coarse proxy, no dirty
   collision.

Note the decoupling this enables: the **run-dir name** (intent, pre-run) no longer has to carry code
identity, so the dir can be minted upfront while the *reproducibility* identity (fingerprint) is
finalized post-run — the row is written after the run anyway.

## Done when

- `point_address`/run identity is derived from `(resolved_with, shape_blob_hash, seed)` — no
  `repo_shas`; single-run + sweep-point dirs are both content-addressed (symmetric).
- Each run records a post-run `code_fingerprint` over its consistent `D` closure; the ledger dedups on
  `(intent, code_fingerprint)` so same-config-different-code runs are both preserved.
- A test: two runs, identical config+shape+seed, a changed `.py` between them → two distinct ledger
  rows (distinct fingerprints), neither overwritten.

## Plan

- [ ] (spec/brainstorm) design the fingerprint + dedup-key change; coordinate with metis#26 (shape
  blob-hash) and metis#28 (consistency guard, which defines "well-defined fingerprint"). Then plan +
  implement.

## Log

### 2026-07-11
- Filed from a reproducibility architecture walkthrough. This is the **keystone** the sweep-key change
  (metis#26) depends on: it replaces the repo-HEAD code proxy in `point_address` with a pre-run
  intent-identity (shape blob-hash) + a post-run code fingerprint, so dropping `sweep_sha` doesn't
  collapse same-config-different-code runs. Folds in the "single-run dir should be content-addressed
  too" symmetry fix. Depends on metis#28 for the "consistent D closure" definition.
