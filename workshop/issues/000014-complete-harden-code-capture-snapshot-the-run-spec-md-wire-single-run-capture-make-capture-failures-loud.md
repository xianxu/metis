---
id: 000014
status: working
deps: [metis#11, metis#13]
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 1.79
started: 2026-07-06T16:33:59-07:00
---

# Complete + harden code capture — snapshot the run-spec .md, wire single-run capture, make capture failures loud

Design source: `workshop/pensive/2026-07-06-reproducible-dirty-run-capture.md` (items 2,3,5).
**Deps:** #13 (config immutability — can't snapshot a `.md` the run rewrites) + benefits from #11
(trace multi-root — so a consumer step's code is *in* the closure to capture).

## Problem

metis#8's side-ref capture is supposed to make a dirty run reproducible (snapshot the exact
code+config bytes to `refs/metis/*`, record the `(path, blob-SHA)` manifest + commit in
`record.json`). Today it under-delivers on three fronts:
1. **The run-spec `.md` is never captured.** The capture closure = the Python read-set
   (`sweepClosure` ← each point's `reads.json`); the experiment `.md` is parsed by the *Go* runner,
   read by no Python step, so it never enters the closure. Only its resolved *values* reach the
   point-address. So "this `titanic-sweep.md` produced the result" isn't actually pinned to a blob.
2. **Capture is sweep-only.** `captureSweepCode` runs from `runSweep`; a plain `metis run`
   (`runResolvedExperiment`) captures nothing — a single dirty experiment run is unreproducible.
3. **Failure is silent/best-effort.** No git / no closure / a git hiccup → capture is a no-op that
   only warns. So you can believe a dirty run was durably captured when it wasn't.

(These three are one issue — all "complete + harden the capture the record promises". Cross-repo
*code* capture is separately metis#11; this issue assumes it and adds the spec-hook + single-run
wiring + loudness.)

## Spec

- **Capture the run-spec** — hash the experiment `.md` bytes (`git hash-object -w`) and include the
  `(path, blob-SHA)` in the record's code manifest; if dirty/untracked, fold it into the side-ref
  snapshot alongside the code closure. This is the *second hook* (the trace won't ever see it).
- **Single-run capture** — factor the capture out of `runSweep` into the shared per-run path
  (`runResolvedExperiment`) so both a single `experiment` run and each sweep point capture their
  closure + spec. Ref namespace: `refs/metis/runs/*` for single runs (vs `refs/metis/sweeps/*`),
  or unify under `refs/metis/captures/*` (decide at plan time).
- **Loud failure** — a run that cannot durably capture its code (no git, dirty-but-uncommittable,
  closure empty when it shouldn't be) must **say so** in its output + mark the record (e.g. a
  `capture: none`/`degraded` field), not silently succeed. Reproducibility is a promise; a broken
  promise must be visible.

## Done when

- A dirty single `metis run` (real git) records a code manifest that includes the experiment `.md`'s
  blob-SHA + the captured commit; `git cat-file`/`checkout` recovers the exact dirty spec + code.
- A dirty sweep captures the spec too (not just the Python closure).
- Capture failure is loud: a test asserting a no-git (or degraded) run reports the reproducibility
  gap in its output + record, not a silent success.
- atlas: the two-hook capture (code via trace, spec via explicit hash) + the record's capture-status.

Durable plan: `workshop/plans/000014-complete-harden-capture-plan.md`. Single-pass atomic.

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

Σdesign 0.45 × 1.15 = 0.5175; Σimpl 1.25 × 1.00 = 1.25; total **1.79** (= `estimate_hours`). `smaller-go-module` #1 = shared `captureRunCode` + single-run wiring + sweep delegation; #2 = the run-spec `.md` hook + loud `CaptureStatus` (record + CUE + stderr); `milestone-review` = close; `atlas-docs` = two-hook capture + capture-status.

## Plan

- [x] RED/GREEN: single-run capture — factor capture into `runResolvedExperiment`; a dirty single run captures its closure + records the manifest/commit.
- [x] RED/GREEN: capture the experiment `.md` blob (the second hook) into the manifest + side-ref.
- [x] RED/GREEN: loud failure — degraded/absent capture is surfaced in output + record.
- [x] atlas: two-hook capture + capture-status.

## Log

### 2026-07-06
- Filed from the reproducible-dirty-run design pass (pensive). Deps #13 + #11. Completes metis#8's
  capture: the run-spec hook, single-run wiring, and loud failure — so a dirty iteration loop is
  actually reproducible, not aspirationally so.
- **Implemented via a full-context fork (TDD).** All Plan items done: shared `captureRunCode` + single-run wiring (`runResolvedExperiment`, guarded by `o.inSweep` so the sweep doesn't double-capture per point), the run-spec `.md` hook (`addSpecToClosure`, symlink-resolved + existence-guarded), loud `CodeManifest.CaptureStatus` (captured|degraded|none) + stderr note + CUE `#CodeManifest`. Heart tests green + regression-proofed; sweep regression green. Fork caught+fixed 2 real bugs (Abs-vs-git-toplevel symlink `Rel` trap; `addSpecToClosure` zeroing a repo D on an absent fixture spec). go build+vet+test ./... 9/9 + uv pytest 37 green. Completes the reproducible-dirty-run effort.
