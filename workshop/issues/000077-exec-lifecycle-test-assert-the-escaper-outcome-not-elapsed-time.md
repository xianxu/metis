---
id: 000077
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '14b595cfdfe391c2ed4a6069d717ad4b93feba37' # card fields mirrored from issue-cards; edit via sdlc
---

# exec lifecycle test: assert the escaper outcome, not elapsed time

## Problem

`TestExecStep_EscapedGrandchildIsCleanupUnconfirmed/exit_0` (cmd/metis/exec_lifecycle_test.go, from #72) asserts that the step's recorded runtime is below `joinBound`, which shows the drain wait is not counted as compute. That is an elapsed-time assertion. At 300ms it failed #73's merge-time suite under load: the escaper fixture waits for a python3 grandchild to start, and that startup alone took longer than 300ms. #73 (commit 059c14e) widened the bound to 3s, which the flake rule forbids, because under enough load 3s fails too. It also adds about 6s per run.

## Spec

Replace the time comparison with an outcome assertion, following ariadne#622. Prove that runtime stops at the step's exit, not after the join wait, with an ordering or structural fact instead of a wall-clock bound. For example: compare the recorded runtime against the observed exit-to-join interval (the join wait demonstrably ran to its bound and the runtime ends before it began), or assert from a seam that the runtime clock stopped at the exit event, before the drain started. Restore a small `joinBound` so the test is fast again.

## Done when

- The exit_0 case asserts the outcome (the step exited and was reaped, or the join completed or ran out, and the runtime was taken at the exit), with no comparison of elapsed time against a fixed bound.
- The case's `joinBound` override from 059c14e is removed, and the test runs at its pre-#73 cost.
- A mutation check: measuring runtime after the join wait makes the test fail, whatever the machine load.
- If the pattern is mechanical, a lint rule in the Go lint catalog cites this incident (tl-worker MISTAKE CLASSES).

## Plan

- [ ]

## Log

### 2026-10-25
