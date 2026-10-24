---
id: 000074
status: working
created: 2026-10-23
updated: 2026-10-23
estimate_hours: 2.93
github_issue:
started: 2026-10-23T19:53:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
---

# Manual grading step with attributable result and separate human wait

## Problem

The operator added human grading to couch-bench (ariadne#577): after hidden scoring, a person must sometimes inspect and try the result, especially for UI tasks. Metis needs a truthful pending manual-step outcome, with separately measured human waiting; a cached grade or inferred acceptance is invalid evidence.
