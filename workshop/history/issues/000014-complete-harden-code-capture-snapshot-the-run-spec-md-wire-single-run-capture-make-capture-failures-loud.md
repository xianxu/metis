---
id: 000014
status: done
deps: [metis#11, metis#13]
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 1.79
started: 2026-07-06T16:33:59-07:00
actual_hours: N/A
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
- 2026-07-06: closed — Close-review round-1 (0 Critical / 2 Important, both fixed). Important-1: the single-run capture WIRING was untested — the direct captureSingleRun tests bypass runResolvedExperiments if !o.inSweep seam. Added TestRunExperiment_SingleRunCapturesViaWiring driving the REAL runExperiment in a git repo (asserts CaptureStatus=captured + refs/metis/runs/<id> == the captured commit); regression-proofed (disabling the run.go call site → fail via the missing ref; restore → pass). The invocation-path-test lesson applied. Important-2: the CUE conformance test built a CodeManifest without CaptureStatus (omitempty dropped it → the closed disjunction never vetted) → set CaptureStatus="captured" (+ Repo) so cue vet exercises it. go build+vet+test ./... 9/9 ok + uv pytest 37 passed. Completes the reproducible-dirty-run effort (pensive): a dirty krun (single or sweep) reproducibly snapshots code + spec to refs/metis/*, loud when it cant. --no-actual: fork impl + close fixes, fork-compressed window.; review verdict: SHIP
- 2026-07-06: closed — metis#14 complete+harden capture (fork impl via TDD; verified in main: go build+vet+test ./... 9/9 ok, uv run pytest 37 passed incl. CUE #CodeManifest capture_status conformance). Three additions on #11 multi-root capture: (1) run-spec .md hook (addSpecToClosure — git-hash-objects the exp .md into its repo closure, symlink-resolved + existence-guarded); (2) single-run capture — shared captureRunCode from runResolvedExperiment (refs/metis/runs/<id>), captureSweepCode delegates (refs/metis/sweeps once per shape-run, not per point, guarded by o.inSweep); (3) loud CodeManifest.CaptureStatus (captured|degraded|none) + stderr note (warnOnUncaptured) + CUE. Heart tests green+regression-proofed (TestCaptureSingleRun_CapturesCodeAndSpec, TestCaptureSingleRun_LoudWhenUncaptured, sweep regression); regression-proofed by disabling addSpecToClosure/warnOnUncaptured (both fail cleanly). Fork caught+fixed 2 real bugs: Abs-vs-git-toplevel symlink Rel trap (EvalSymlinks-before-Rel) + addSpecToClosure zeroing a repo D on an absent fixture spec (existence guard). --no-actual: fork-compressed window (1 commit). Completes the reproducible-dirty-run effort: a dirty krun (single or sweep) reproducibly snapshots code + spec to refs/metis/*, loud when it cant.; review verdict: FIX-THEN-SHIP
- Filed from the reproducible-dirty-run design pass (pensive). Deps #13 + #11. Completes metis#8's
  capture: the run-spec hook, single-run wiring, and loud failure — so a dirty iteration loop is
  actually reproducible, not aspirationally so.
- **Implemented via a full-context fork (TDD).** All Plan items done: shared `captureRunCode` + single-run wiring (`runResolvedExperiment`, guarded by `o.inSweep` so the sweep doesn't double-capture per point), the run-spec `.md` hook (`addSpecToClosure`, symlink-resolved + existence-guarded), loud `CodeManifest.CaptureStatus` (captured|degraded|none) + stderr note + CUE `#CodeManifest`. Heart tests green + regression-proofed; sweep regression green. Fork caught+fixed 2 real bugs (Abs-vs-git-toplevel symlink `Rel` trap; `addSpecToClosure` zeroing a repo D on an absent fixture spec). go build+vet+test ./... 9/9 + uv pytest 37 green. Completes the reproducible-dirty-run effort.
- **Close-review round-1 (FIX-THEN-SHIP, 0 Critical / 2 Important — both fixed).** (1) The single-run capture WIRING was untested (the direct `captureSingleRun` tests bypass `runResolvedExperiment`'s `if !o.inSweep` seam) — added `TestRunExperiment_SingleRunCapturesViaWiring` driving the REAL `runExperiment` in a git repo (asserts CaptureStatus=captured + refs/metis/runs/<id> = the commit), **regression-proofed** (disabling the run.go call site → fail; restore → pass). The invocation-path-test lesson, applied. (2) The CUE conformance test built a `CodeManifest` without `CaptureStatus` (omitempty skipped it) → added `CaptureStatus: "captured"` (+ `Repo`) so `cue vet` actually vets the closed disjunction. go+py suites green.
