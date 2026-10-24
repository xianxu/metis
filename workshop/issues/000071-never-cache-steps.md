---
id: 000071
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c37f29809c390541ae7ea3e6ae1968bc815da3c0' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

Add a step-level never-cache policy at the existing StepExecutor/cache boundary. It bypasses both lookup and storage for that execution. Preserve ordinary defaults. Ensure descendants cannot reuse results on input identity alone after an uncached nondeterministic producer: either propagate non-cacheability conservatively or bind reuse to the realized upstream artifacts. Document the chosen rule; do not infer determinism from absent reads.json. Update the vocabulary and its consumers through their existing owner.

## Done when

- A never-cache session executes on every repeat even with identical configuration and seed; no cache entry can substitute for it.
- A changed realized session candidate cannot reuse an old downstream hidden score. A sequence test exercises both unchanged and changed outputs through the real cache decorator.
- Ordinary deterministic steps retain existing hit/miss and transitive code-invalidation behavior.
- Human-grade steps can request the same policy without domain-specific logic in Metis; malformed/unknown policy values refuse before execution.

## Plan

- [ ]

## Log

2026-10-23: Filed unclaimed for the operator-approved ariadne#577 re-plan. Seams: cmd/metis/caching.go cachingExecutor.Execute/kpre; pkg/experiment.Step and construct/vocabulary/experiment.cue. No implementation or production experiment changed.
2026-10-23 (metis:1): The slot carried uncommitted local-gates leftovers from slot setup that blocked the fast-forward to main 0350013: merge-check.yml and .gitignore modified (an older seed), .gitattributes and local-gates-guard.yml untracked (identical to main). The hub (ops:0) approved discarding them. By the time I ran the backup, the tree was already clean at origin/main 0350013, so I discarded nothing. The scratchpad backup holds only main's two untracked files; the older modified seed was never captured, and main's versions replace it.
