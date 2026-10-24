---
id: 000073
status: working
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
started: 2026-10-23T19:48:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
---

# Repeat-shaped run records and ledger rows without CV folds

## Problem

couch-bench (ariadne#577) repeats an autonomous issue with fixed TL and worker profiles. Current plain experiments write run.json and record.json but no ledger row; sweep rows deduplicate by point address and code fingerprint, and repeated identical points can reuse a deterministic run directory. A repeat must remain a new observed sample, not a CV fold or a fake configuration choice.
