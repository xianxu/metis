---
id: 000071
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '8cd175d5700698d1d275bb0b4064a8603806adc7' # card fields mirrored from issue-cards; edit via sdlc
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
