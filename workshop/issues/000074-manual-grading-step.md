---
id: 000074
status: working
deps: [metis#71, metis#72, metis#73]
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0862d9a7b7271903efe32c9acf202d8582de7b24' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:53:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
---

# Manual grading step with attributable result and separate human wait

## Problem

The operator added human grading to couch-bench (ariadne#577): after hidden scoring, a person must sometimes inspect and try the result, especially for UI tasks. Metis needs a truthful pending manual-step outcome, with separately measured human waiting; a cached grade or inferred acceptance is invalid evidence.

## Spec

Provide the minimal generic manual-step support needed by couch/grade: a never-cached request bound to the exact execution and result, a visible notification/wait, an explicitly supplied coarse numeric grade, and an artifact retaining grader identity, submission time and the reviewed result identity. Keep the flat numeric metrics.json contract: numeric grade and wait seconds are metrics; identity and timestamp are typed evidence artifacts. Record human wait separately and exclude it from measured automated run time while retaining total elapsed time. Explicit no-look tasks may skip with a reason; a skipped or missing grade is not a passing grade. Couch notification and result presentation belong to couch-bench, not the Metis core.

## Done when

- A requested grade remains pending until a matching human submission; wrong-run, wrong-result, duplicate conflicting or malformed submissions refuse.
- Grade, who and when are retained and hash-bound by the existing record; never-cache prevents reuse across repeats.
- Waiting time is separately visible and excluded from automated runtime; total elapsed duration remains available. Timeout/cancellation never synthesizes a grade.
- An explicitly skippable task records skipped status/reason and omits grade rather than writing zero or pass.
- Stateful fixtures cover request, response, pending interruption, resume/reconciliation, skip and stale submission without sending real operator notifications.

## Plan

Durable plan: `workshop/plans/000074-manual-grading-step-plan.md` (full flow). Design decisions 1 to 8 there wait for TL approval.

- [ ] M1 — request, submit, wait, evidence
- [ ] M2 — resume and reconciliation

## Log

2026-10-23: Filed unclaimed from authenticated ops:0 addition f28bfafd-75e4-4f81-993c-acf79910fa1a for ariadne#577. Coordinate with the never-cache, timeout/live-log and repeat-ledger issues. No new grading scale is imposed here; the consumer declares its coarse scale.
2026-10-23 (metis:1): Concurrent claims in this slot: #71 to #76 are open at once while #75 waits on the operator CI pin (metis#70). Active-time attribution across them is approximate until ariadne#580 lands.
