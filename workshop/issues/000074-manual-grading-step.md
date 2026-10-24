---
id: 000074
status: working
deps: [metis#71, metis#72, metis#73]
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours: 2.93
collaboration-mode: collaboration
card_mirror: '9ba1d8c9a088ebeb7d9cb06c3d0719ebf975540a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:53:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: metis:1
    worktree: /Users/xianxu/workspace/worktree/metis-slot1/metis
    repository: github.com/xianxu/metis
flow: {kind: full, provenance: inferred}
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

Durable plan: `workshop/plans/000074-manual-grading-step-plan.md` (full flow). Design decisions 1 to 8 there were approved by the TL (see Log).

- [ ] M1 — request, submit, wait, evidence
- [ ] M2 — resume and reconciliation

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.05 impl=0.16
item: greenfield-go-module   design=0.1 impl=0.28
item: greenfield-go-module   design=0.1 impl=0.32
item: smaller-go-module      design=0.05 impl=0.2
item: smaller-go-module      design=0.0 impl=0.2
item: smaller-go-module      design=0.05 impl=0.16
item: smaller-go-module      design=0.05 impl=0.16
item: greenfield-go-module   design=0.1 impl=0.28
item: smaller-go-module      design=0.0 impl=0.2
item: atlas-docs             design=0.05 impl=0.06
item: milestone-review       design=0.0 impl=0.14
item: milestone-review       design=0.0 impl=0.14
design-buffer: 0.15
total: 2.93
```

Items, in order:
1. ManualPolicy plus implied uncached.
2. pkg/grade: types, identities, ValidateSubmission.
3. manualExecutor: request, presenter, wait, tombstone, evidence, re-collect.
4. `metis grade submit`.
5. M1 sequence tests.
6. execution.json plus the run-dir lease.
7. resumePlan.
8. `metis run --resume`.
9. Resume tests.
10. Atlas.
11. Two milestone reviews.

Design hours are near zero after TL approval and 2 plan-quality rounds. Impl is 40% of the v2 ranges.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Log

2026-10-23: Filed unclaimed from authenticated ops:0 addition f28bfafd-75e4-4f81-993c-acf79910fa1a for ariadne#577. Coordinate with the never-cache, timeout/live-log and repeat-ledger issues. No new grading scale is imposed here; the consumer declares its coarse scale.
2026-10-23 (metis:1): Concurrent claims in this slot: #71 to #76 are open at once while #75 waits on the operator CI pin (metis#70). Active-time attribution across them is approximate until ariadne#580 lands.
2026-10-23 (metis:1): The TL (ops:0) approved the plan as written. Grader trust stays with couch-bench. Code starts after #71, #72 and #73.
2026-10-23 (metis:1): The TL (ops:0) approved Revisions A, B and C: submission.json as the single linearization point with a tombstone abandon, the run-dir flock lease, and manual steps refused in sweeps.
2026-10-23 (metis:1): change-code passed. Plan-quality was clean after 3 rounds; round 1 raised 3 Important findings, all fixed in plan Revisions (A to C approved by the TL). Estimate 2.93h; estimate-quality INFO, which flags item 3 (manualExecutor) and item 5 (sequence tests) as light and the sweep refusal as unitemized (it folds into item 1). Recorded as accepted. Code waits for #71, #72 and #73.
