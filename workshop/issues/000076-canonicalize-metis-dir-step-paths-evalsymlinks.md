---
id: 000076
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: '415a776e997ede5dc65cca602ee0f244e347a9be' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Resolve expDir with EvalSymlinks in runResolvedExperimentAdmitted; derive runDir from it.
- [ ] Add the symlinked-dir test.

## Log

