---
id: 000076
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# Canonicalize METIS_*_DIR step paths (EvalSymlinks)

## Problem

`cmd/metis/run.go` (runResolvedExperimentAdmitted) builds the run and experiment dirs with `filepath.Abs` only. A step subprocess therefore receives `METIS_RUN_DIR` and `METIS_STEP_DIR` (and `METIS_EXP_DIR`) as logical paths that may go through a symlink: `/tmp` → `/private/tmp`, or a macOS TMPDIR under `/var` → `/private/var`. couch-bench's ariadne backend refuses non-physical paths, so every step of an experiment under /tmp or TMPDIR fails. Found by couch-bench:0, relayed by the hub (ops:0).
