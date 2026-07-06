---
issue: 000011
title: Trace sensor multi-root — capture first-party code from every repo (not just metis)
status: active
created: 2026-07-06
---

# Trace multi-root Implementation Plan

> **For agentic workers:** AGENTS.md §3. TDD throughout.

**Goal:** The read-set `D` (`metis.trace`) captures first-party code from **every repo the step touches** — so a **consumer repo's** code (`kbench/titanic/features.py`) enters `D`, and editing it correctly busts the metis#2 cache / enters the metis#8 capture closure. Today `D` is rooted at the metis repo alone, so consumer code is dropped (`_classify`: "another repo → not first-party").

**Architecture:** Replace the single `_PROJECT_ROOT` with **per-read git-repo-root discovery**: a read is first-party iff it lives under a git working tree that is not `site-packages`/`.venv`/stdlib/run-dir/`__pycache__`. The sensor walks up from each read's path to its containing repo root (`.git` marker, cached per-dir) and groups reads **by repo root**. `reads.json` gains a `roots: {<repo-root>: [rel-paths]}` map (replacing the single `project_root` + flat `reads`). The Go side (`trace.go` `readSet`, `caching.go` `buildD`/`gitBlobHashes`, `capture.go` closure) consumes the per-repo grouping — `git -C <root> hash-object` each path in **its** repo, and `D` entries become repo-qualified so the same relative path in two repos can't collide (and recovery knows which repo).

**Tech Stack:** Python (`sys.addaudithook` sensor), Go (`cmd/metis` cache/capture), git plumbing.

## Core concepts

### Pure/IO entities

| Name | Lives in | Status |
|------|----------|--------|
| `_repo_root(path)` (walk-up + cache) | `metis/trace.py` | new |
| `_classify` (multi-root) | `metis/trace.py` | modified |
| `_write_reads` (emit `roots` map) | `metis/trace.py` | modified |
| `readSet` (`Roots map[string][]string`) | `cmd/metis/trace.go` | modified |
| `buildD` / read-set assembly (per-repo) | `cmd/metis/caching.go` | modified |
| `sweepClosure`/capture closure (per-repo) | `cmd/metis/capture.go` | modified |

- **`_repo_root(path) -> str|None`** — walk up from a file to the nearest ancestor containing a `.git` **marker (a dir OR a file** — linked worktrees/submodules use a `.git` *file*); `None` if none (stdlib/temp) or if under `site-packages`/`.venv`. Cached per-directory (avoid repeated walks). A read is first-party iff `_repo_root` returns a root.
- **`_classify` multi-root** — site-packages/.venv → `used_site_packages`; else find `_repo_root`; if found and not run-dir/`__pycache__`, record `(root, relpath)` into a `roots` dict. Drop stdlib/temp (no repo root).
- **`readSet.Roots`** — `map[repoRoot][]relPath`. `D` = the union over roots of `(repo-qualified path, git-blob-hash)`; each root's paths hashed via `git -C <root> hash-object`. The point-address's per-repo `repo_shas` already keys by repo — `D` now aligns.

### The `reads.json` format change (v2)

Old: `{"project_root": "<metis>", "reads": ["metis/io.py", …], "used_site_packages": bool}`.
New: `{"roots": {"<abs-repo-root>": ["titanic/features.py", …], "<metis-root>": ["metis/io.py", …]}, "used_site_packages": bool}`. `reads.json` is ephemeral (gitignored, per-run) so no persisted back-compat is needed — change both sides together; update all fixtures/tests.

## Tasks (TDD)

### Task 1: sensor multi-root (Python)

- [ ] **1.1 RED** — a `metis/trace` test (or `tests/`): simulate reads from **two** repo roots (metis + a temp git repo standing in for a consumer) and assert both appear in the emitted `roots` map, grouped by root; a site-packages read sets `used_site_packages`; a stdlib/temp read is dropped. Fails today (single-root).
- [ ] **1.2 GREEN** — `_repo_root` walk-up + cache; `_classify` groups by root; `_write_reads` emits `roots`. Run → PASS.
- [ ] **1.3** — regression: a metis-only step still captures metis code (its root is metis) — no regression.

### Task 2: Go consumer per-repo (trace.go + caching.go)

- [ ] **2.1 RED** — a Go test: a `reads.json` with two roots yields a `D` covering **both** repos' files (repo-qualified `CodeRef`s), each hashed in its own repo. Fails today (`Reads []string` + single-root `gitBlobHashes`).
- [ ] **2.2 GREEN** — `readSet.Roots map[string][]string`; `buildD`/read-set assembly loops roots, `gitBlobHashes(root, paths)` per root, folds into a repo-qualified `D`. Run → PASS.
- [ ] **2.3 store/validate symmetry (the cache correctness pair)** — the cache Entry **persists `D`** so a later run re-hashes it to decide HIT/MISS (`pkg/cache` `Validate`/`isHit`). The persisted `D` schema + the store side (`storeAndRecord`) + the validate side must all use the **same repo-qualified `D`** — asymmetry (store per-repo, validate single-root) is a false-HIT/false-MISS bug. Widen the Entry's `D` to carry the repo per ref; store and validate identically.
- [ ] **2.4** — `capture.go` closure (`sweepClosure`) iterates all roots (so #14's capture, once built, snapshots consumer code too). Keep it working for the single-repo case.

### Task 3: integration + wire kbench + atlas

- [ ] **3.1** — an integration test: trace a module that imports across two repos (a fixture consumer that imports metis) → `D` contains **both** the consumer file and the metis modules. This is the exact kbench topology the issue exists for.
- [ ] **3.1b — the HIT→MISS behavioral test (the whole point):** cache a step whose closure spans two repos; re-run identically → **HIT**; then edit the **consumer** repo's file → the persisted `D` re-hash moves → **MISS** (was a false HIT before this issue). This is the correctness guarantee metis#11 exists to deliver — assert it end-to-end, not just that the path appears in `D`.
- [ ] **3.1c — the empty-D-false-HIT guard (the severe lockstep risk):** the v2 `reads.json` (`roots` map) MUST NOT parse to an empty `D` on the Go side — a silent format mismatch (old struct reads `reads`, new file has `roots`) → empty `D` → a **vacuous K_pre-only HIT** (worse than the bug we're fixing). RED test: a two-repo closure whose `reads.json` is parsed yields a **non-empty** `D` covering the consumer file; and land 2.2/2.3/2.4 **atomically** (schema + store + validate together) so the store/validate sides never disagree on the format. Prefer a loud parse error over a silent empty `D`.
- [ ] **3.2** — `metis` own steps regression (`go test ./... && uv run pytest`) green.
- [ ] **3.3** — **wire kbench**: `kbench/steps/titanic/{adapt,features,submission}` can now route through `python -m metis.trace kbench.titanic.<mod>` (kbench#3 deferred `features` to direct invocation with an atlas note — flip it once this lands). *(Cross-repo follow-up — do in kbench after metis#11 merges; note here, don't block metis#11 on it.)*
- [ ] **3.4** — atlas: the sensor's multi-root read-set + the per-repo `D` (aligns with `repo_shas`); update the `reads.json` format doc.

### Task 4: close

- [ ] **4.1** `sdlc close --issue 11`. Log: the kbench wrapper flip (3.3) is a tracked kbench follow-up (unblocks kbench#3's decision #2).

## Done when (issue) — mapped

- [ ] a consumer-repo module traced through `metis.trace` has its first-party code in `D` (test) — Tasks 1,3
- [ ] metis's own steps still capture metis code (no regression) — Tasks 1.3, 3.2
- [ ] atlas: cross-repo read-set + per-repo D — Task 3.4

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: typed-data-prototype   design=0.2   impl=0.4
item: smaller-go-module      design=0.2   impl=0.4
item: smaller-go-module      design=0.2   impl=0.45
item: milestone-review       design=0.0   impl=0.2
item: atlas-docs             design=0.05  impl=0.15
design-buffer: 0.15
total: 2.35
```

Σdesign 0.65 × 1.15 = 0.7475; Σimpl 1.60 × 1.00 = 1.60; total ≈ 2.35. `typed-data-prototype` = the Python sensor multi-root (`_repo_root` walk-up + grouped `roots`) + the `reads.json` v2 schema; `smaller-go-module` #1 = the Go per-repo consumer (`readSet.Roots` + per-root `gitBlobHashes` + repo-qualified `buildD`); `smaller-go-module` #2 = the **cache store/validate symmetry** (persisted per-repo `D` schema in `pkg/cache` Entry + `Validate`/`isHit` re-hash) **+ the HIT→MISS behavioral test** + capture multi-root (the wider Go surface the plan-judge flagged); `milestone-review` = close boundary; `atlas-docs` = the multi-root read-set + `reads.json` v2. Single-pass atomic. The kbench wrapper flip (Task 3.3) is a separate cross-repo follow-up, not in this estimate.
