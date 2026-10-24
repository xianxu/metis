---
id: 000072
status: working
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
started: 2026-10-23T19:38:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
---

# Bound step execution with timeout and live logs

## Problem

ariadne#577 needs a long Couch session as a Metis step. The legacy execStep path uses exec.Command and CombinedOutput: no step deadline, no live child log, and output accumulates in memory. A timeout cannot be mistaken for retirement of independently managed Couch actors.
