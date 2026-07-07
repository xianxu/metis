---
issue: 000014
title: Complete + harden code capture — run-spec, single-run, loud failure
status: active
created: 2026-07-06
---

# Complete + harden capture Implementation Plan

> **For agentic workers:** AGENTS.md §3. TDD throughout. Design source: `workshop/pensive/2026-07-06-reproducible-dirty-run-capture.md` (items 2,3,5). Deps #11 + #13 (both merged).

**Goal:** A dirty run is *actually* reproducibly captured — the exact code closure **and the run-spec `.md`** are snapshotted to git (recoverable via `refs/metis/*`), for **single runs** as well as sweeps, and a run that **can't** durably capture says so **loudly** (in output + record), never silently.

**Architecture:** Three additions on top of #11's multi-root capture (`capture.go`):
1. **Run-spec hook** — the experiment `.md` is a first-party input the Python read-set never sees (the Go runner parses it). Add it explicitly to the closure of its own repo before capture: `git hash-object` the `.md`, include a `CodeRef` for it, and fold it into the side-ref snapshot if dirty.
2. **Single-run capture** — extract the sweep's `captureSweepCode` body into a shared `captureRunCode(root-closures, specPath, id, refPrefix)` and call it from `runResolvedExperiment` (the shared per-run path both single runs and the sweep loop use) — single runs → `refs/metis/runs/<runID>`; sweeps keep the once-per-shape-run `refs/metis/sweeps/<shapeRunID>` (don't double-capture per point).
3. **Loud capture status** — a `CodeManifest.CaptureStatus` (`captured` | `degraded` | `none`) + a stderr line when capture can't run (no git / no closure / a git failure). The run still completes (best-effort stays), but the reproducibility gap is **visible**, not silent.

**Tech Stack:** Go (`cmd/metis` capture/run + `pkg/record` CodeManifest), git plumbing.

## Core concepts

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `captureRunCode` (shared, single+sweep) | `cmd/metis/capture.go` | new | git side-ref snapshot |
| `captureSweepCode` (delegates to shared) | `cmd/metis/capture.go` | modified | — |
| run-spec `.md` in the closure | `cmd/metis/capture.go` | modified | `git hash-object` |
| single-run capture call | `cmd/metis/run.go` (`runResolvedExperiment`) | modified | — |
| `CodeManifest.CaptureStatus` | `pkg/record/*` (+ CUE) | modified | — |

- **`captureRunCode(closureByRepo, specPath, id, refPrefix)`** — hash + snapshot the per-repo closure **plus the run-spec `.md`** (in the spec's repo) to `refs/metis/<refPrefix>/<id>`; return the per-repo manifest + commit + a **status**. Pure-ish over the injected git seam (`gitOut`), so it's testable with a real temp repo.
- **Run-spec inclusion** — the `.md`'s repo root = the spec file's git repo; add `relpath(.md)` to that repo's closure so its bytes are hashed + (if dirty) committed. This is the "second hook" — the trace never sees the spec.
- **CaptureStatus** — `captured` (durable SHA recorded), `degraded` (ran but couldn't fully capture — e.g. no git repo), `none` (no closure). Surfaced in the record + a one-line stderr note on non-`captured`.

## Tasks (TDD)

### Task 1: shared per-run capture + single-run wiring

- [ ] **1.1 RED** — a real-git test: a **single** `metis run` on a dirty experiment records a `CodeManifest` whose D includes the (dirty) code closure + a captured `refs/metis/runs/<runID>` commit; `git cat-file`/`checkout` recovers the dirty bytes. Fails today (single runs capture nothing).
- [ ] **1.2 GREEN** — extract `captureRunCode` (shared); call it from `runResolvedExperiment` for single runs (`refs/metis/runs/<runID>`); `captureSweepCode` delegates to it (`refs/metis/sweeps/<shapeRunID>`, once per shape-run — not per point). Run → PASS. Regression: sweeps still capture.

### Task 2: capture the run-spec `.md` (the second hook)

- [ ] **2.1 RED** — assert the captured manifest includes the experiment `.md`'s `(repo, relpath, blob-hash)`; editing the `.md` dirty and running captures its exact bytes (recoverable). Fails today (spec never in the closure).
- [ ] **2.2 GREEN** — add the spec `.md` to its repo's closure before `captureRunCode`. Run → PASS.

### Task 3: loud capture status

- [ ] **3.1 RED** — a run that can't durably capture (no git repo, or an empty closure) sets `CodeManifest.CaptureStatus != "captured"` AND emits a stderr note; a clean-git run is `captured`. Fails today (silent best-effort, no status).
- [ ] **3.2 GREEN** — thread the status through `captureRunCode` → the record + a `fmt.Fprintf(out, …)` note on non-`captured`. CUE `#CodeManifest` gains `capture_status`. Run → PASS.

### Task 4: integration + atlas + close

- [ ] **4.1** — `go build/vet/test ./...` + `uv run pytest` green; the CUE `#RunRecord`/`#CodeManifest` conformance test covers `capture_status`.
- [ ] **4.2** — atlas: the **two-hook capture** (code via the multi-root trace #11, spec via `git hash-object`) + `CaptureStatus` + `refs/metis/{runs,sweeps}/*`.
- [ ] **4.3** — `sdlc close --issue 14`. Log: this completes the reproducible-dirty-run effort (the pensive) — a dirty `krun` is now reproducibly captured (code + spec), single or sweep, with a loud gap when it can't be.

## Done when (issue) — mapped

- [ ] a dirty single `metis run` records a manifest incl. the `.md` blob + a captured commit; recoverable — Tasks 1,2
- [ ] a dirty sweep captures the spec too — Task 2 (+ regression 1.2)
- [ ] capture failure is loud (status + stderr), not silent — Task 3
- [ ] atlas: two-hook capture + capture-status — Task 4

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.2   impl=0.45
item: smaller-go-module      design=0.2   impl=0.45
item: milestone-review       design=0.0   impl=0.2
item: atlas-docs             design=0.05  impl=0.15
design-buffer: 0.15
total: 1.79
```

Σdesign 0.45 × 1.15 = 0.5175; Σimpl 1.25 × 1.00 = 1.25; total ≈ 1.79. `smaller-go-module` #1 = the shared `captureRunCode` + single-run wiring (`runResolvedExperiment`) + the sweep delegation; `smaller-go-module` #2 = the run-spec `.md` hook + the loud `CaptureStatus` (record + CUE + stderr); `milestone-review` = close boundary; `atlas-docs` = the two-hook capture + capture-status. Single-pass atomic.
