---
id: 000074
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '8ab83f8f0d0b8a84a29d8d132bc7151ef8182484' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

2026-10-23: Filed unclaimed from authenticated ops:0 addition f28bfafd-75e4-4f81-993c-acf79910fa1a for ariadne#577. Coordinate with the never-cache, timeout/live-log and repeat-ledger issues. No new grading scale is imposed here; the consumer declares its coarse scale.
