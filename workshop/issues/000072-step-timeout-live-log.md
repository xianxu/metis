---
id: 000072
status: working
deps: [metis#71]
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours: 3
collaboration-mode: collaboration
card_mirror: 'f7187b15a233480c8bb666cb23752b4136055335' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:38:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
flow: {kind: full, provenance: inferred}
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

Durable plan: `workshop/plans/000072-step-timeout-live-log-plan.md` (full flow, M1 legacy and cancellation, M2 forkserver). Design decisions 1 to 9 there were approved by the TL (see Log).

- [ ] M1 — policy, outcome, bounded live log, legacy path, cancellation
- [ ] M2 — forkserver path

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.2 impl=0.24
item: smaller-go-module      design=0.1 impl=0.2
item: smaller-go-module      design=0.2 impl=0.2
item: greenfield-go-module   design=0.2 impl=0.32
item: smaller-go-module      design=0.2 impl=0.2
item: smaller-go-module      design=0.2 impl=0.2
item: atlas-docs             design=0.05 impl=0.04
item: milestone-review       design=0.0 impl=0.14
item: milestone-review       design=0.0 impl=0.14
design-buffer: 0.15
total: 3.0
```

Items, in order:
1. Pure core: ExecPolicy, classifyOutcome, boundedLog.
2. ctx threaded through StepExecutor, decorators and fakes.
3. Cancellable acquire (leafBudget, chanSem, prioritySem) with its interleaving test.
4. Legacy lifecycle: pgid, timer, TERM/KILL, join, outcome.json, pending resources, plus fixtures.
5. Python forkserver: streaming capped log and cancel.
6. Go forkexec: cancel, bounded ready wait, late result.
7. Atlas.
8. Two milestone reviews.

The thorough plan gives a +15% design buffer, and impl is 40% of the v2 ranges.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Log

2026-10-23: Filed unclaimed for ariadne#577. Primary seam: cmd/metis/exec.go execStep.Execute, with forkexec.go and runcontrol.go lifecycle integration. No code changed.
2026-10-23 (metis:1): Concurrent claims in this slot: #71, #75, #76 and #72 are open at once while #75 waits on the operator CI pin (metis#70). Active-time attribution across them is approximate until ariadne#580 lands.
2026-10-23 (metis:1): The TL (ops:0) approved the plan with decision 2 as written: the timeout counts from launch, not step entry, with a cancellable leaf-budget acquire. All other decisions stand as written. Code starts after #75 lands (its suite needs make test).

## Revisions

- 2026-10-23 (metis:1), reason: plan-quality round 1. Delta:
  - PQ-1: declared `deps: [metis#71]`. ExecPolicy follows #71's `with.cache` parser and drift guard, and Task 4 threads ctx through `cachingExecutor`, which #71 rewrites. Code starts only once #71 has landed (#71 itself waits on #75), and the branch takes main then.
  - Minors folded into the plan: `reservedStepFiles` is new, extracted from the inline list at exec.go:239. The cancellable acquire covers the `leafBudget` interface, `chanSem` and `prioritySem`. A context cancelled during the forkserver ready wait returns a `cancelled` step error, never `ok=false`, so it never falls back to legacy.
2026-10-23 (metis:1): change-code passed: plan-quality clean after 2 rounds; estimate 3.0h (estimate-logic-v3.1); estimate-quality INFO. Accepted risks from it: the fixture-heavy items may overrun; rebasing onto #71 seam changes is unbudgeted (#71 is already implemented, so its seam is known). Code waits for #71 to land.
