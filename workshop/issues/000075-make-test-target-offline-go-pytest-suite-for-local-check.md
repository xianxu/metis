---
id: 000075
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: '9ceeeadc6e3de9d26c7b267cd84040de2c49057c' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Add `test` and `test-deps` targets in `Makefile` (with a comment on the offline contract).
- [ ] Verify: in a fresh clone with no `.venv`, with the network denied (default sandbox), `make test` passes; `sdlc local-check run --name suite` records.

## Log

