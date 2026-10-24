---
id: 000076
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
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
