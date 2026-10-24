---
id: 000075
status: codecomplete
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
started: 2026-10-23T17:24:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
actual_hours: 0.37
tracker:
    version: 1
    completion:
        token: close-8f04ca6ff1ce
        repository: github.com/xianxu/metis
        reviewed_head: aad6f5c5d310ca2dd05dc246bd7a26359d039ee9
        evidence_commit: 51bd2c350204b61f2052933ee14d6a99f4350266
---

# make test target: offline Go + pytest suite for local-check

## Problem

metis has no `make test` target and no `.sdlc/local-checks.json`. So `sdlc local-check run --name suite` refuses, and `sdlc merge` cannot get its required `sdlc-local/suite` record. This blocks landing metis#71 and every later metis issue.
