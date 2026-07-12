# Run/Code Identity Split — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the conflated run identity into a **pre-run intent-identity** (content-addressed by config + shape blob-hash + seed, dropping repo HEAD) and a **post-run code-fingerprint** (hash of the run's D closure), so same-config-different-code runs are both preserved and dirty sweeps are content-addressable — with `repo_shas` dropped entirely.

**Architecture:** `point_address` is recomposed from `(resolved_with, shape_blob_hash, seed)` — a pre-run intent identity computed by git-`hash-object`ing the shape `.md`. A new run-level `code_fingerprint` = `CanonicalHash` of the run-end D closure (the union `captureRunCode` already returns) is recorded post-run in `record.json` + the ledger row. The ledger dedups on `(point_addr, code_fingerprint)`. `repo_shas` — and its derived `sweep_sha` column — are removed everywhere (identity + provenance); code identity lives in each step's `code.commit` side-ref + `D`. Single pass, one review boundary; accept the identity/cache version bump (no migration — sweep ledgers are gitignored + regenerable).

**Tech Stack:** Go 1.x (stdlib `testing`), CUE (`cue vet` drift-guard), git plumbing (`hash-object`).

**ARCH notes:** `point_address`/`code_fingerprint`/the consistency-free closure hash are pure functions in `pkg/record` (**ARCH-PURE**), unit-tested without IO; the git `hash-object` calls are the thin seam. Dropping `repo_shas` is a single-source cleanup — one identity model, no repo-HEAD proxy duplicated across `point_address` + `shape_run_id` + `sweep_sha` (**ARCH-DRY**). **ARCH-PURPOSE:** the Done-when is the *behavioral* acceptance (two runs, changed `.py` between → two rows), not just "compiles."

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `record.PointAddress` | `pkg/record/address.go` | modified (sig: `repoSHAs`→`shapeBlobHash`) |
| `record.CodeFingerprint` | `pkg/record/address.go` | new |
| `record.RunRecord` | `pkg/record/record.go` | modified (−`RepoSHAs`, +`CodeFingerprint`) |
| `ledger.Row` | `pkg/ledger/ledger.go` | modified (−`SweepSHA`, +`CodeFingerprint`) |
| `ledger.Append` (dedup) | `pkg/ledger/ledger.go` | modified (composite key) |
| `#RunRecord` (CUE) | `construct/vocabulary/experiment.cue` | modified (−`repo_shas`, +`code_fingerprint`) |

- **`record.PointAddress`** — the pre-run intent identity. New signature
  `PointAddress(resolvedWith map[string]map[string]any, shapeBlobHash string, seed int) (Hash, error)`;
  hashes `{resolved_with, shape_blob_hash, seed}`.
  - **DRY rationale:** the single intent-hash; replaces the repo-HEAD proxy that was duplicated into `shape_run_id`.
  - **Future extensions:** a multi-file shape would hash a set of blob-hashes instead of one.

- **`record.CodeFingerprint`** — `CodeFingerprint(d []CodeRef) (Hash, error)` = `CanonicalHash` of the sorted run-end D closure. Pure over the closure `captureRunCode` returns.
  - **Relationships:** 1:1 with a run; recorded on `RunRecord` + each ledger row for that run's points.
  - **Future extensions:** metis#28 swaps the input for a per-step-time closure + adds the consistency check; the hash function itself is unchanged.

- **`record.RunRecord`** — drop `RepoSHAs`; add `CodeFingerprint Hash json:"code_fingerprint"`.
- **`ledger.Row`** — drop `SweepSHA`; add `CodeFingerprint string`. Dedup identity becomes `(PointAddr, CodeFingerprint)`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `shapeBlobHash` | `cmd/metis/` (new helper) | new | `git hash-object` |
| single-run dir minting | `cmd/metis/run.go` | modified | run-dir naming |
| post-run fingerprint compute | `cmd/metis/run.go` + `sweep.go` | modified | capture → record write |

- **`shapeBlobHash(dir, specPath) (string, error)`** — pre-run `git hash-object --` of the shape `.md` (reuse `gitBlobHashes`, `trace.go:48-65`; rel-path derivation as in `addSpecToClosure`, `capture.go:246-253`). Feeds `PointAddress` + `shape_run_id`.
  - **Injected into:** `runExperiment`/`runShapeSweep` entry points, replacing `repoSHAsOf`.
- **single-run dir minting** — when `--run` is empty, name the dir by the run's `point_address` (like sweep points), not `run-<timestamp>`. `--run` becomes an optional human alias.
- **post-run fingerprint compute** — after `captureRunCode`, hash its returned `d` → set `RunRecord.CodeFingerprint` before the record write (single run: `run.go:176-180`; sweep: `sweep.go:146-150`).

**Test surface:** pure entities unit-tested in `pkg/record` + `pkg/ledger` without IO; the git-touching helpers via the existing `cmd/metis` e2e harness (real git temp repos). The CUE `#RunRecord` + Go struct move in lockstep, gated by `conformance_test.go` + `record_e2e_test.go`'s `cue vet`.

---

## Tasks (single-pass; plain checkboxes, one review boundary at `sdlc close`)

### Task 1: `PointAddress` recompose (repo_shas → shape_blob_hash)

**Files:** Modify `pkg/record/address.go:29-42`; Test `pkg/record/record_test.go:37-94`.

- [ ] **Step 1 — Migrate the failing tests.** In `record_test.go`: change `mustAddr` (`:37-44`) to the new 3-arg `(resolvedWith, shapeBlobHash string, seed)`; flip `TestPointAddress_Sensitivity`'s `"changed repo-SHA"` case (`:70`) to `"changed shape-blob"` (different `shapeBlobHash` → different address); keep determinism + non-finite-error cases.
- [ ] **Step 2 — Run, verify fail:** `cd /Users/xianxu/workspace/metis && go test ./pkg/record/ -run TestPointAddress -v` → FAIL (old signature).
- [ ] **Step 3 — Implement.** `PointAddress(resolvedWith map[string]map[string]any, shapeBlobHash string, seed int)`; hash `{ResolvedWith, ShapeBlobHash string json:"shape_blob_hash", Seed}`.
- [ ] **Step 4 — Run, pass.** Commit: `#27: PointAddress = hash(resolved_with, shape_blob_hash, seed)`.

### Task 2: `CodeFingerprint` pure function

**Files:** Add to `pkg/record/address.go`; Test `pkg/record/record_test.go`.

- [ ] **Step 1 — Failing test:** `TestCodeFingerprint` — a fixed `[]CodeRef` → a stable hash; order-independent (sort inside); a changed blob → a changed fingerprint; empty closure → a defined value (document: empty = "no first-party code").
- [ ] **Step 2 — Run, fail.**
- [ ] **Step 3 — Implement:** `CodeFingerprint(d []record.CodeRef) (Hash, error)` = sort `d` by `(repo,path)` then `CanonicalHash`.
- [ ] **Step 4 — Run, pass.** Commit: `#27: CodeFingerprint over the run-end D closure`.

### Task 3: `RunRecord` struct + CUE (drop repo_shas, add code_fingerprint) in lockstep

**Files:** `pkg/record/record.go:70-81`; `construct/vocabulary/experiment.cue:131-142`; Tests `pkg/record/conformance_test.go:36-85`, `record_test.go:134-158`.

- [ ] **Step 1 — Migrate tests:** `conformance_test.go` drop `RepoSHAs` (`:51`), add `CodeFingerprint`; `TestRunRecord_JSONRoundTrip` (`:137`) same.
- [ ] **Step 2 — Run, fail:** `go test ./pkg/record/ -run Conform -v`.
- [ ] **Step 3 — Implement:** Go: `RunRecord` drop `RepoSHAs` (`:75`), add `CodeFingerprint Hash json:"code_fingerprint,omitempty"`. CUE `#RunRecord` (`:136`): drop `repo_shas?`, add `code_fingerprint?: string`.
- [ ] **Step 4 — Run, pass** (`go test ./pkg/record/` — incl. the `cue vet`). Commit: `#27: RunRecord/#RunRecord — drop repo_shas, add code_fingerprint`.

### Task 4: `buildRecord`/`assembleRecord` — thread shape_blob_hash + fingerprint, drop repoSHAs

**Files:** `cmd/metis/record.go:60-64,116-130`; Tests `cmd/metis/record_test.go:30-99`, `record_e2e_test.go`.

- [ ] **Step 1 — Migrate tests:** `record_test.go` — `buildRecord` signature loses `(repoName, sha)`, gains `shapeBlobHash`; flip `TestBuildRecord_MintsStablePointAddress`'s "changes with repo SHA" (`:47-50`) → "changes with shape blob"; drop the `RepoSHAs["metis"]` assert (`:62-64`). `record_e2e_test.go` — drop `RepoSHAs` asserts (`:90-92,190-191`); keep identical-runs-share-address (`:136-137`).
- [ ] **Step 2 — Run, fail.**
- [ ] **Step 3 — Implement:** `assembleRecord` stops probing for `repo_shas`; `buildRecord` takes `shapeBlobHash`, passes it to `PointAddress` (`:119`), no longer sets `RunRecord.RepoSHAs` (`:128`), and sets `CodeFingerprint` (from Task 6's compute, threaded in). Compute `shapeBlobHash` at the entry via the new helper (Task 5).
- [ ] **Step 4 — Run, pass.** Commit: `#27: buildRecord threads shape_blob_hash + fingerprint, drops repo_shas`.

### Task 5: `shapeBlobHash` helper + single-run dir naming

**Files:** new helper in `cmd/metis/` (e.g. `capture.go` near `gitBlobHashes` reuse, or a small `identity.go`); `cmd/metis/run.go:101,104-109`; `main.go:39`.

- [ ] **Step 1 — Failing test:** a `cmd/metis` test that `shapeBlobHash(dir, spec)` equals `git hash-object <spec>` for a temp repo; and that a single `metis run` with no `--run` names the dir by the run's `point_address` (not `run-<ts>`).
- [ ] **Step 2 — Implement:** `shapeBlobHash` (reuse `gitBlobHashes`); in `run.go:101`, when `o.runID==""`, mint via `pointAddressOf(exp, shapeBlobHash)` instead of `defaultRunID`; keep `--run` as an explicit override (update `main.go:39` doc). Remove `defaultRunID`'s timestamp path (or keep only for `--run`-absent-and-no-shape fallback — decide in impl; prefer content-address).
- [ ] **Step 3 — Run, pass.** Commit: `#27: shapeBlobHash helper + content-addressed single-run dir`.

### Task 6: `pointAddressOf`/`shapeRunIdentity` + sweep wiring — drop repoSHAs, compute fingerprint

**Files:** `cmd/metis/sweep.go:73,90,116,146-150,198,229,343-350,398-412,441-447`.

- [ ] **Step 1 — Migrate tests** touching `pointAddressOf`/sweep identity (grep in `shapesweep_test.go`/`shipe2e_test.go`; per recon they don't assert identity directly — expect compile-only fixups).
- [ ] **Step 2 — Implement:**
  - `pointAddressOf(exp, shapeBlobHash)` (drop `repoSHAs` param, `:343-350`); call sites `:198,229` pass `ss.shapeBlobHash`.
  - `shapeRunIdentity(sh, shapeBlobHash)` — replace the `RepoSHAs` term (`:408,410`) with `shape_blob_hash`.
  - Remove `repoSHAsOf` (`:441-447`), `shapeSweep.repoSHAs` (`:73,116`), the `probeRepo` call feeding it (`:90`) → compute `ss.shapeBlobHash` once at sweep entry.
  - Post-run: after `captureSweepCode` (`:146-150`), hash the returned closure `d` → set each point's `RunRecord.CodeFingerprint` (the sweep captures once; the fingerprint is the sweep-run closure, shared by its points — confirm this matches the per-point record write; if per-point closures differ, compute per point).
- [ ] **Step 3 — Run** `go build ./... && go test ./cmd/metis/ -run Sweep`. Commit: `#27: sweep identity drops repo_shas, computes code_fingerprint`.

### Task 7: Ledger — drop sweep_sha, add code_fingerprint, composite dedup

**Files:** `pkg/ledger/ledger.go:34-41,52-66,78-174,192-346`; `cmd/metis/ledger.go:22-52,54-61`; `cmd/metis/ledger_cmd.go:119,226-227,309`; Tests `pkg/ledger/ledger_test.go`, `cmd/metis/ledger_test.go`.

- [ ] **Step 1 — Migrate/author tests:** `ledger_test.go` — `TestAppend_DedupByPointAddress` → dedup on `(PointAddr, CodeFingerprint)` (same point_addr + different fingerprint ⇒ **two** rows; identical ⇒ one); Encode header (`:50`) `sweep_sha`→`code_fingerprint`. `cmd/metis/ledger_test.go` — drop `SweepSHA=="sha1"` (`:35`), assert the fingerprint column.
- [ ] **Step 2 — Implement:**
  - `Row`: drop `SweepSHA`, add `CodeFingerprint string`; `Append` (`:52-66`) `seen` key = `PointAddr+"\x00"+CodeFingerprint`.
  - CSV header/encode/decode (`:89,102,147-149`) `sweep_sha`→`code_fingerprint`; `AggregateView`/`Filter` (`:208,213,335-338`) key on `code_fingerprint` where they used `sweep_sha` (or drop the sweep-scoping — `--sweep` filter becomes `--fingerprint`? decide: keep a `--sweep`-equivalent that filters by `code_fingerprint`).
  - `cmd/metis/ledger.go`: `rowsFromManifest` (`:22-37`) set `CodeFingerprint` from each point's `record.json`; remove `sweepSHAOf` (`:54-61`).
  - `ledger_cmd.go`: the promote HEAD-warning (`:226-227`) + `renderPromoted` back-link (`:309`) reference `SweepSHA` — replace with `code_fingerprint` (and fix the warning text: reproducibility is via the side-ref, per metis#28, not a repo checkout).
- [ ] **Step 3 — Run, pass** (`go test ./pkg/ledger/ ./cmd/metis/`). Commit: `#27: ledger drops sweep_sha, dedups on (point_addr, code_fingerprint)`.

> **Note (`ledger select` — metis#19):** the `--sweep <full-SHA>` scoping used in the #19 acceptance keyed on `sweep_sha`. After this change it becomes `--fingerprint` (or is dropped). Update `select_cmd.go` + the #19 acceptance runbook reference if the flag name changes — the mixed-cohort scoping still works, now keyed by code_fingerprint.

### Task 8: Whole-module green + behavioral acceptance

- [ ] **Step 1 — Whole-module:** `go build ./... && go test ./... && go vet ./...` (fix any remaining `repo_shas`/`SweepSHA` call sites the compiler flags — cross-check against the recon drop-surface).
- [ ] **Step 2 — The acceptance test (Done-when):** an e2e (real temp repo, following `record_e2e_test.go` idiom) — run a config; change a first-party `.py`; run the **same** config+shape+seed again; assert **two distinct ledger rows** (same `point_addr`, different `code_fingerprint`), neither overwritten. This is the load-bearing test.
- [ ] **Step 3 — Drive it in the real binary:** `metis run` a small shape twice with a `.py` edit between; confirm the ledger has two rows + the single-run dir is content-addressed. Capture output as close evidence.
- [ ] **Step 4 — Atlas:** update `atlas/` for the identity model (intent + code_fingerprint; `repo_shas`/`sweep_sha` removed).

### Task 9: Close

- [ ] **Step 1 — Estimate reconciliation** already set (see issue `## Estimate`); measure with `sdlc actual --issue 27`.
- [ ] **Step 2 — Close:** `sdlc close --issue 27 --verified '<acceptance: two-rows-on-code-change + whole-module green + content-addressed run dir>'` (single boundary → the mandatory review runs here).

---

## Non-goals
Mid-run/sweep consistency *detection* + `reproduce`/`verify` verbs (metis#28 — needs per-step step-time blobs). Migrating existing ledgers/records (accept the identity/cache version bump). The sweep-key ledger *column* rename is metis#26 (this issue removes `sweep_sha` and adds `code_fingerprint`; #26's shape-blob-hash *display* keying builds on the `shapeBlobHash` helper landed here).

## Revisions
(none yet)
