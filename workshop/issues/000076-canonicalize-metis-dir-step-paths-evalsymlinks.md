---
id: 000076
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: 'adcd72909f1bc4293d4c0b4acd679481c79f9cd7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T17:48:39-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
---

# Canonicalize METIS_*_DIR step paths (EvalSymlinks)

## Problem

`cmd/metis/run.go` (runResolvedExperimentAdmitted) builds the run and experiment dirs with `filepath.Abs` only. A step subprocess therefore receives `METIS_RUN_DIR` and `METIS_STEP_DIR` (and `METIS_EXP_DIR`) as logical paths that may go through a symlink: `/tmp` → `/private/tmp`, or a macOS TMPDIR under `/var` → `/private/var`. couch-bench's ariadne backend refuses non-physical paths, so every step of an experiment under /tmp or TMPDIR fails. Found by couch-bench:0, relayed by the hub (ops:0).

## Spec

Canonicalize the experiment dir at the runner boundary with `filepath.EvalSymlinks` after `filepath.Abs`, and derive the run dir from it. The run dir may not exist yet, so the existing parent is resolved, not the leaf. Every path handed to a child env is then physical. Error on an unresolvable experiment dir; don't fall back silently.

## Done when

- A test runs an experiment through a symlinked directory and asserts that the child sees `METIS_RUN_DIR`, `METIS_STEP_DIR` and `METIS_EXP_DIR` equal to their `EvalSymlinks` form.
- The existing run, cache and record tests stay green.

## Plan

- [x] Resolve expDir with EvalSymlinks in runResolvedExperimentAdmitted; derive runDir from it.
- [x] Add the symlinked-dir test.

## Log


- 2026-10-25: closed — every METIS_* path var a child receives is physical, enforced at the shared execStep.stepEnv via physicalPath (run, blend and the sealed sweep readRoot alike): TestExecStep_StepEnvPathsArePhysical (symlinked exp dir, symlinked runs/, not-yet-created read root) and TestExecStep_ConfinementAgreesAcrossPathForms (real uv cv-split, physical exp dir with logical read root stays within root) both failed before the fix; TestRun_StepEnvPathsArePhysical still passes. Recorded suite passed on d357a3e0fcae7b3173c5f28a7f136a7696ac6528 (go ./... ok, pytest 140 passed).; review verdict: SHIP
2026-10-23 (metis:1): Implemented in runResolvedExperimentAdmitted: Abs, then EvalSymlinks on expDir (error if unresolvable), with runDir joined under it. TestRun_StepEnvPathsArePhysical runs the real execStep (test/env-dump) through a symlinked experiment dir and asserts physical EXP_DIR, RUN_DIR and STEP_DIR. Mutation: with the resolve disabled, all 3 asserts fail. go test ./cmd/metis ./pkg/... is green. The recorded suite waits for #75 (make test) to land.
2026-10-23 (metis:1): Concurrent claims in this slot: #71, #75, #76 and #72 (in planning) are open at once while #75 waits on the operator CI pin (metis#70). Active-time attribution across them is approximate until ariadne#580 lands.

2026-10-25 metis:1: close review round 1 REWORK, family
child-env-paths-canonical-form. BR-1 (Critical): run.go made METIS_EXP_DIR
physical while the sealed sweep's readRoot (sweep.go, filepath.Abs) stayed
logical, so under a symlinked dir every within-root read tripped confinement.
BR-2: blend.go built Abs-only paths. BR-3: a symlinked runs/ was not resolved.
Fixed the class at the one place child paths meet: `execStep.stepEnv` (shared by
the legacy and forkserver executors, reached by run, blend and the sweep) now
resolves all four path vars through the new `physicalPath` (deepest existing
ancestor, so not-yet-created dirs work); run.go uses the same helper. Tests:
TestExecStep_StepEnvPathsArePhysical (symlinked exp dir, symlinked runs/,
not-yet-created read root; all four vars physical) and
TestExecStep_ConfinementAgreesAcrossPathForms (real uv cv-split, physical exp
dir and logical read root: within-root read succeeds). Both failed before the
fix. Several claims are open at once in this slot (ariadne#580 fixes
attribution).
