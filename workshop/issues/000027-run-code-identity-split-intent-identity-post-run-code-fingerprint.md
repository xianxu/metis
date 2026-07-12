---
id: 000027
status: codecomplete
deps: []
github_issue:
created: 2026-07-11
updated: 2026-07-11
estimate_hours: 3.47
started: 2026-07-11T20:25:34-07:00
actual_hours: N/A
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
   `code_fingerprint = CanonicalHash` of the run's **run-end `D` closure** — the union of every step's
   read-set paths → their working-tree blob-hashes at capture time (the closure `captureRunCode`
   already produces, `capture.go:141`). "What code actually ran." This **achieves the identity goal**:
   two runs with different code (each stable *within* its run — the normal case) get different
   fingerprints → distinct rows, no collision.
   - **Scope boundary (recon finding):** #27 does **not verify within-run consistency**. Blobs are
     hashed once at capture (post-run), and `backfillCodeManifest` writes one run-wide `D` to every
     step (`capture.go:308-313`) — so a *mid-run* code change (step A read v1, step B read v2) is
     invisible here; the fingerprint hashes the final state. Detecting/refusing that needs per-step
     *step-time* blobs (not recorded today) → **that + the reproduce verbs are metis#28.** #27 assumes
     within-run consistency (true in the common case); #28 verifies it.

3. **Ledger dedup key = (intent, code_fingerprint).**
   Same config+shape, different code → same `intent`, *different* `fingerprint` → **both rows kept**
   (the "all variations we swept" goal). Same everything → dedup. Repo HEAD drops out entirely: intent
   uses the shape blob-hash, code identity uses the actual `D` closure. No coarse proxy, no dirty
   collision.

Note the decoupling this enables: the **run-dir name** (intent, pre-run) no longer has to carry code
identity, so the dir can be minted upfront while the *reproducibility* identity (fingerprint) is
finalized post-run — the row is written after the run anyway.

### Decisions (2026-07-11)

- **`repo_shas` dropped ENTIRELY** — not just from `point_address`, but from `shape_run_id`
  (`sweep.go:401`) and from `record.json` altogether. The code identity lives in each step's
  `code.commit` (the side-ref, HEAD if clean) + its read-set `D`; repo HEAD adds nothing and misleads
  once dirty sweeps are allowed. `shape_run_id` recomposes over `(shape structure, shape_blob_hash,
  seed)`.
- **Single-pass** (no `Mx` milestone split) — the fingerprint and the dedup that consumes it are
  tightly coupled; one review boundary at `sdlc close`. Plain checkboxes in the Plan.
- **Migration = accept the identity/cache version bump** (like metis#24): the recomposed
  `point_address` orphans old cache/ledger entries; no migration of old rows (the sweep ledgers are
  gitignored + regenerable). Document the bump.
- **Fingerprint = run-end closure hash (no within-run consistency verification)** — the mid-run
  consistency *detection* (needs per-step step-time blobs) + the `reproduce`/`verify` verbs are
  metis#28. #27 fixes the identity collision; #28 verifies + refuses mid-run drift.

## Done when

- `point_address`/run identity is derived from `(resolved_with, shape_blob_hash, seed)` — no
  `repo_shas`; single-run + sweep-point dirs are both content-addressed (symmetric).
- Each run records a post-run `code_fingerprint` over its consistent `D` closure; the ledger dedups on
  `(intent, code_fingerprint)` so same-config-different-code runs are both preserved.
- A test: two runs, identical config+shape+seed, a changed `.py` between them → two distinct ledger
  rows (distinct fingerprints), neither overwritten.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.3 impl=0.4
item: smaller-go-module      design=0.2 impl=0.3
item: smaller-go-module      design=0.2 impl=0.4
item: smaller-go-module      design=0.2 impl=0.4
item: smaller-go-module      design=0.2 impl=0.4
item: milestone-review       design=0.0 impl=0.2
item: atlas-docs             design=0.05 impl=0.05
design-buffer: 0.15
total: 3.47
```

Design pre-settled (this session's walkthrough + thorough plan) → design near the floor. Keystone =
the pure `PointAddress` recompose + `CodeFingerprint` (`pkg/record`). Four `smaller` refactors:
`RunRecord`+CUE lockstep; `buildRecord`+`shapeBlobHash`+single-run naming; sweep wiring (drop
`repoSHAs`); ledger dedup+columns. One `milestone-review` (single boundary) + atlas. Impl at 40%-of-v2
(v3.1); +15% thorough-plan buffer. (Calibration note: metis#19 ran ~1.7× over a similar-breadth
estimate — the test-migration surface here is broad; watch at close.)

## Plan

Durable plan: `workshop/plans/000027-run-code-identity-split-plan.md` (9 tasks, TDD). Single-pass, one
review boundary at `sdlc close`.

- [x] Implement the identity split per the durable plan: `PointAddress` = `hash(resolved_with,
  shape_blob_hash, seed)` + pure `CodeFingerprint` over the run-end `D` closure; `RunRecord`/CUE drop
  `repo_shas` + add `code_fingerprint`; fingerprint computed in `backfillCodeManifest`; ledger dedups
  on `(point_addr, code_fingerprint)` (`--sweep`→`--fingerprint`); content-addressed single-run dir;
  `repo_shas`/`sweep_sha` dropped everywhere (keep `probeRepo`/`codeID` guard). Acceptance: two
  identical-config sweep runs with an in-closure `.py` edit between → two distinct ledger rows.

## Log

### 2026-07-11
- 2026-07-11: closed — Independently verified: go build ./... + go test ./... (9 pkgs) + go vet ./... all green. Acceptance TestCodeIdentity_TwoRowsOnCodeChange PASS (same point_addr, in-closure model.py edit -> distinct code_fingerprints -> two ledger rows, neither overwritten) + TestSingleRun_ContentAddressedDir PASS (dir = 64-hex point_address). Real-binary drive: record.json has point_address + code_fingerprint (2eb0e91a...), repo_shas ABSENT; CUE #RunRecord has code_fingerprint (repo_shas dropped). repo_shas/sweep_sha/repoSHAsOf/sweepSHAOf gone from non-test Go (only explanatory comments remain). ACTUAL=N/A: measured window (1.06h) EXCLUDES the pre-claim design walkthrough (the 3.47h estimate includes design) AND under-counts the fork-implemented build (active-time-v3 misses fork wall-time, interleaved-sessions caveat) -> recording it would pollute velocity toward under-estimating.; review verdict: FIX-THEN-SHIP
- **`deps: []` is intentional.** The Log/Spec reference to metis#28 is a *conceptual* dependency (the
  "consistent D closure" definition), NOT a blocking code dep: #27 explicitly scopes out consistency
  *verification* and assumes within-run consistency (hashing the run-end closure). #28 depends on #27,
  not the reverse.
- Filed from a reproducibility architecture walkthrough. This is the **keystone** the sweep-key change
  (metis#26) depends on: it replaces the repo-HEAD code proxy in `point_address` with a pre-run
  intent-identity (shape blob-hash) + a post-run code fingerprint, so dropping `sweep_sha` doesn't
  collapse same-config-different-code runs. Folds in the "single-run dir should be content-addressed
  too" symmetry fix. Depends on metis#28 for the "consistent D closure" definition.
- **FIX-THEN-SHIP applied** (close-review: 0 Critical, 2 Important): (1) `atlas/experiment.md` `PointAddress`/`CodeFingerprint` signatures corrected (base-layer API doc was misstating the changed surface); (2) `TestShapeSweep_NestedLoopWinnerAndLedger` now asserts a non-empty `code_fingerprint` in the persisted ledger — guards the load-bearing capture-before-`writeSweepLedger` ordering (a reorder would silently yield empty-fingerprint rows, re-opening the collision, with every other test still green). Minors: ledger package header + promote note de-staled. **Follow-up (minor, deferred):** `shapeBlobHash` duplicates `addSpecToClosure`'s abs→symlink→toplevel→Rel prologue — extract a shared `specRepoRel` helper (ARCH-DRY, cosmetic).
