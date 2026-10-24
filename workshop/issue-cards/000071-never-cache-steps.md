---
id: 000071
status: working
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
started: 2026-10-23T17:13:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
---

# Step-level never-cache policy with safe downstream invalidation

## Problem

ariadne#577 is being re-planned as a Metis consumer (couch-bench). An agent session and a human grading step are deliberately nondeterministic; a cache hit would erase a repeat or reuse a person's old judgment. The current cache has only a global switch, and downstream keys use upstream input identities, so simply bypassing the session cache can still reuse a stale score.
