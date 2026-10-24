---
id: 000075
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: '5adab9b3f5c5eaca4dc8c7e90d6f398f8faa431e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T17:24:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
---

# make test target: offline Go + pytest suite for local-check

## Problem

metis has no `make test` target and no `.sdlc/local-checks.json`. So `sdlc local-check run --name suite` refuses, and `sdlc merge` cannot get its required `sdlc-local/suite` record. This blocks landing metis#71 and every later metis issue.

## Spec

Add a `test` target to the metis Makefile that runs the full suite: `go test ./...` plus `pytest tests` under uv. The suite runs offline (`UV_OFFLINE=1`, inherited by the Python steps the Go tests spawn). A cold checkout's venv builds from the warm uv cache, and a separate `test-deps` target (`uv sync`, online) warms that cache outside the suite. A missing cache then fails loudly with uv's offline error, never with a network wait. This adds no local-gates configuration (merge-checks, workflows or hooks).

## Done when

- `make test` runs the Go and Python suites and fails if either fails.
- `make test` passes offline in a fresh clone without a `.venv`, given a warm uv cache.
- `sdlc local-check run --name suite` records `sdlc-local/suite` on the pushed head.

## Plan

- [x] Add `test` and `test-deps` targets in `Makefile` (with a comment on the offline contract).
- [x] Verify: in a fresh clone with no `.venv`, with the network denied (default sandbox), `make test` passes; `sdlc local-check run --name suite` records.

## Log

2026-10-23 (metis:1): In a fresh clone with no .venv and the network denied (default sandbox, no allowed domains), `make test` passed: go test ./... ok, then pytest 137 passed with the venv built offline from the uv cache.
2026-10-23 (metis:1): The first recorded suite on 8e8b3c96 failed in Go setup: the local-check checkout (/Volumes/TestScratch/local-checks/sdlc-local-*) has no sibling ../ariadne for go.mod's replace. Fix: when ../ariadne is absent and sdlc exports SDLC_DEP_SUBSTRATE, test with -modfile pointing at a throwaway go.mod copy whose replace targets that path. Rejected a temp go.work: it matches `use` against the cwd, so a symlinked cwd (/tmp vs /private/tmp) breaks it. Verified in a clone without ../ariadne, network denied: Go all ok, pytest 137 passed.

## Revisions

- 2026-10-23 (metis:1), reason: the first recorded suite showed local-check checkouts lack the sibling `../ariadne` that go.mod's replace needs (Done-when #3). Delta to Spec and Plan item 1: when `../ariadne/go.mod` is absent and sdlc exports `SDLC_DEP_SUBSTRATE`, `make test` runs the Go suite with `-modfile` pointing at a private per-run copy of go.mod and go.sum (made with `mktemp -d`, removed on exit), whose `replace` targets the substrate. Slots and clones are unchanged. Close review round 1 (BR-1): the per-run private dir replaces a fixed shared temp path, which concurrent runs could collide on and nothing removed.
