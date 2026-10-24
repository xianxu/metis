---
id: 000075
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# make test target: offline Go + pytest suite for local-check

## Problem

metis has no `make test` target and no `.sdlc/local-checks.json`. So `sdlc local-check run --name suite` refuses, and `sdlc merge` cannot get its required `sdlc-local/suite` record. This blocks landing metis#71 and every later metis issue.
