---
id: 000072
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ce904a72581940e603167613ec708d9255cb030a' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

Extend the existing execution owner with explicit step timeouts, persistent bounded live logs, and cancellation/join semantics. Cover legacy subprocesses and forkserver execution explicitly; a lost fork response must never trigger duplicate fallback. Preserve ordinary default behavior and use the existing progress/log surface rather than adding a second runner. Distinguish child completion from external-resource reconciliation: preserve artifacts and an actionable failure when a Couch-driving child stops before its actors are retired.

## Done when

- A long-running fixture exposes new log bytes before it exits; bounded capture cannot exhaust memory or disk silently.
- Timeout and cancellation terminate and join owned child processes under a bounded cleanup policy, retain diagnostics, and never publish a successful result or retry an unconfirmed execution.
- Deadline expiry before launch does not start the child; explicit timeout validation precedes effects.
- Native readiness-controlled fixtures prove legacy and applicable forkserver outcomes, including a late result and unconfirmed cleanup; ordinary successful steps still work.
- The human-wait distinction is compatible with the separately filed grading issue; time waiting for a grade is labelled separately, never disguised as compute runtime.

## Plan

Durable plan: `workshop/plans/000072-step-timeout-live-log-plan.md` (full flow, M1 legacy and cancellation, M2 forkserver). Design decisions 1 to 9 there wait for TL approval.

- [ ] M1 — policy, outcome, bounded live log, legacy path, cancellation
- [ ] M2 — forkserver path

## Log

2026-10-23: Filed unclaimed for ariadne#577. Primary seam: cmd/metis/exec.go execStep.Execute, with forkexec.go and runcontrol.go lifecycle integration. No code changed.
2026-10-23 (metis:1): Concurrent claims in this slot: #71, #75, #76 and #72 are open at once while #75 waits on the operator CI pin (metis#70). Active-time attribution across them is approximate until ariadne#580 lands.
