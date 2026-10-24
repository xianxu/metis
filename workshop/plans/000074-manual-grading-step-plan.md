# Manual Grading Step Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Depends on:** metis#71 (never-cache), metis#72 (`outcome.json`, ctx cancellation, step timeout), metis#73 (claimed run dirs, atomic writes, ledger coverage). Code starts after all three land.

**Goal:** A step declared `with: {manual: {scale: [0,1,2,3], skippable: true}}` runs its presenter (for example couch-bench's `couch/grade`), then stays pending until a matching human submission arrives through `metis grade submit`. It records grade, grader and time as a hash-bound evidence artifact. Human wait is measured separately from compute time. A timeout or cancellation never synthesizes a grade. A run interrupted while pending can be resumed.

**Architecture:** metis, not the child, owns the request, the wait, the validation and the evidence. Before launching the step's executable, the runner writes `grade-request.json`. It binds the run, repeat, step and the **result identity**, which metis computes from the upstream outputs under review. The executable only presents and notifies (consumer logic, outside the core). After it exits 0, the runner enters a pending phase: it waits for `submission.json`, which `metis grade submit` creates exclusively after validating it against the request. A run-level `execution.json` written at claim time lets `metis run --resume` continue a run that was interrupted while pending, without re-executing completed steps.

**Tech Stack:** Go, CUE vocabulary.

---

## Design decisions (for TL review before code)

1. **Policy surface: `with.manual`**, a closed sub-map like #71's `cache` and #72's `exec`, parsed by a pure `ParseManualPolicy` that Validate calls before any effect:
   - `scale` (required): a non-empty list of distinct numbers. The consumer declares its coarse scale; metis imposes none.
   - `skippable` (bool, default false).
   - `wait_timeout` (a duration; default none, meaning wait until a submission or cancellation).
   A manual step is **implicitly uncached**: `UncachedSteps` treats `with.manual` like `policy: never`, in one place. So a grade can never be reused across repeats, and every descendant is uncached too.
2. **Request binding.** `grade-request.json` = `{request_id, run_id, repeat, step_id, result_identity, scale, skippable, requested_at}`.
   - `result_identity` is the canonical hash of the manual step's needs and their realized output hashes (`record.OutputHash` per need), computed by metis from the step dirs. The child cannot spoof it.
   - `request_id` is the canonical hash of `{run_id, repeat, step_id, result_identity, requested_at}`.
   - The env gains `METIS_GRADE_REQUEST=<path>`, so the presenter can show the reviewer exactly what to submit.
3. **Submission: `metis grade submit <run-dir> --step S --request R (--grade G | --skip --reason TEXT) --grader NAME`.**
   - Validation is pure, against the request: the request exists and is pending; `R` matches its `request_id`, so a wrong run or wrong result refuses; `G` is in the scale; `--skip` only if skippable, with a non-empty reason; the fields are well-formed.
   - Write: `submission.json` is created with `O_CREATE|O_EXCL`. That is the single-use claim:
     - an exact duplicate (same request, grade or skip, and grader) is idempotent;
     - anything else refuses as "conflicting submission";
     - a request that is no longer pending (abandoned, timed out or finalized) refuses as stale.
   - Recorded: `grader` (declared), `submitted_by` (OS user, not authenticated, recorded as is) and `submitted_at`.
4. **The wait.** After the presenter exits 0, the runner prints a pending line on `out`, the existing progress surface (`⏸ step S awaiting grade — metis grade submit …`). It then polls the step dir for `submission.json` once a second, cancellable by ctx (#72) and bounded by `wait_timeout`. If the presenter exits non-zero, the step fails and no wait starts. The request state lives in `grade-request.json`'s companion `grade-state.json`, with states `pending → finalized | abandoned`, written atomically.
5. **Evidence and metrics.**
   - On a valid submission the runner writes the artifact `grade.json` = `{request_id, result_identity, grader, submitted_by, submitted_at, grade | skipped+reason}`. It is an artifact, so it enters the step's output hash, which the existing record hash-binds.
   - Metrics: `grade` (omitted when skipped, never zero), and `wait_seconds` (from requested_at to the accepted submission).
   - `outcome.json` (#72) gains `wait_seconds`. `runtime_seconds` stays the presenter's compute time, which naturally excludes the wait because the child has already exited. Total elapsed = ended − started.
   - A skip records `skipped` and its reason in `grade.json` and the outcome, with no grade metric. #73's per-metric n then shows the missing grade as missing coverage.
6. **Timeout and cancel while pending.** The outcome is `timed-out` or `cancelled` (#72's classifier), `grade-state` becomes `abandoned`, no `grade.json` is written, and no metric is emitted. A later submission refuses as stale.
7. **Resume after interruption: `metis run --resume <run-dir>`.**
   - At claim time (#73), the runner writes `execution.json` = `{experiment_path, shape_blob_hash, point_address, repeat, seed, started}`, atomically.
   - Resume refuses unless:
     - the experiment's current blob hash equals the recorded one;
     - no step outcome is failed;
     - exactly one step has `grade-state: pending`.
   - Resume then runs the DAG in the **same** run dir:
     - steps with an ok `outcome.json` are restored from disk (`collectResult`), not re-executed;
     - the pending manual step re-enters the wait, without re-running the presenter and without a new request;
     - remaining steps run normally, uncached on resume, because their cache keys would need upstream K_pre state that a restore lacks. The cache is an optimization; skipping it is sound.
   - A submission made while metis was down is picked up immediately.
   - A resume also refuses if the result identity recomputed from the restored upstream outputs differs from the request's. That is reconciliation: an edited or corrupted upstream dir must never be graded.
8. **Presentation and notification stay outside the core.** metis prints the pending line and writes the request. couch-bench's presenter step sends the Couch notification. Tests use a fake presenter, a shell step that copies the request and exits, so no real notifications are sent.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ManualPolicy` / `ParseManualPolicy` | `pkg/experiment/manualpolicy.go` | new |
| `UncachedSteps` (manual implies uncached) | `pkg/experiment/cachepolicy.go` | modified |
| `GradeRequest`, `Submission`, `GradeState` | `pkg/grade/grade.go` | new |
| `ValidateSubmission` | `pkg/grade/grade.go` | new |
| `ResultIdentity`, `RequestID` | `pkg/grade/grade.go` | new |
| `resumePlan` | `cmd/metis/resume.go` | new |

- **ValidateSubmission(req, state, sub, existing\*)** → accept | idempotent | refuse(reason). A pure transition from (request state, submission) to a decision. The table: pending + valid gives accept; pending + an identical existing submission gives idempotent; a different existing submission gives conflict; abandoned or finalized gives stale; a request mismatch gives wrong-run/result; a grade off the scale gives malformed; a skip that is not allowed gives refuse.
- **resumePlan(steps, outcomes, gradeStates)** → per step `restore | wait | run`, or a refusal. Pure and table-tested.
- **pkg/grade** is a new package because both the runner and the `grade submit` CLI need the same types and validation (ARCH-DRY), and it stays free of exec and IO.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `manualExecutor` | `cmd/metis/manual.go` | new | decorates `execStep`: request write, presenter run, wait loop, evidence |
| `metis grade submit` | `cmd/metis/grade_cmd.go` | new | CLI plus `O_EXCL` create |
| `metis run --resume` | `cmd/metis/run.go`, `resume.go` | new | reads execution.json, outcomes and grade state |
| `execution.json` writer | `cmd/metis/run.go` | new | atomic write (#73's `writeFileAtomic`) |

- **manualExecutor** sits inside the cache decorator, which never sees manual steps because they are uncached, and inside the activity decorator. Its wait loop takes an injected clock and poll interval (test seam), so tests are deterministic. Readiness: the test submits only after observing the request file and pending state.

## State design table

Actions available: `metis grade submit` (grade or skip), Ctrl-C/SIGTERM, wait timeout, `metis run --resume`, and reading the request, state and evidence files.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| presenter exits ≠ 0 | step failed; no request pending (state abandoned) | fix the presenter; rerun with a new repeat | none |
| pending, valid grade submitted | grade.json, grade + wait_seconds metrics, state finalized, outcome ok | none | none |
| pending, valid skip | grade.json skipped + reason, wait_seconds only, outcome ok (skipped) | none | the grade, by choice |
| submission for a wrong run or result | refused; still pending | submit against the right request | none |
| grade off the scale / skip not allowed / malformed | refused; still pending | resubmit correctly | none |
| identical duplicate submission | idempotent success | none | none |
| conflicting second submission | refused "conflicting" | none (the first stands) | none |
| wait_timeout or Ctrl-C while pending | outcome timed-out/cancelled, state abandoned, no grade | rerun as a new repeat | the human's time; no grade is invented |
| submission after abandonment | refused as stale | none | none |
| metis dies while pending | state still pending; no run.json or record | `metis run --resume <dir>` (a submission made meanwhile is picked up) | none |
| resume with an edited experiment file | refused "experiment changed since execution" | restore the file or rerun | the pending execution, if not restorable |
| resume with an upstream output changed on disk | refused "result identity mismatch" | investigate; rerun | that execution |
| resume while a step outcome is failed, or nothing is pending | refused "nothing to resume" | rerun | none |
| two `grade submit` calls racing | O_EXCL: one accepted, the other is identical (idempotent) or conflicting (refused) | none | none |

Covered: every request, submission and interruption path. **Not covered:** authenticating the grader (the declared name plus the OS user is recorded; couch-bench owns trust), and resuming a run interrupted outside a manual wait (a plain step dying mid-run stays a dead execution, #73).

## ARCH notes

- **PURE:** policy parsing, identities, submission validation and resume planning are pure (`pkg/grade`, `resumePlan`). The wait loop, file writes and CLI are glue.
- **ORDER:** grade state is an explicit machine (`pending → finalized | abandoned`); submission uses an `O_EXCL` claim. The event most likely to be mishandled is a submission landing at the same moment as a timeout. The rule is that the runner's transition to `abandoned` is written atomically *before* it reports timed-out, and `grade submit` re-reads the state after its exclusive create and deletes its own submission if it finds `abandoned`, refusing as stale. That is sequence-tested with a controlled clock.
- **CONSTRAINTS:** the wait is human-scale, minutes to days. Polling once a second is negligible. No hot path.
- **FUNERAL:** request, state, submission and evidence files live in the step dir and go with the run dir; `execution.json` is one per run dir. All are bounded and tiny.
- **SECURE:** the submission is untrusted input, parsed strictly and validated against the metis-written request. The result identity is computed by metis, never read from the child. Reason and grader strings are capped (256 bytes) and stored verbatim.
- **EVOLVE:** all new files are additive JSON, and readers keep unknown fields when rewriting `grade-state.json`. `outcome.json` gains `wait_seconds` additively.
- **MOCK:** a fake presenter step (shell) and in-process submissions. No real notifications.
- **PURPOSE:** pending until a matching submission, the refusal classes, hash-bound evidence, a separate wait, skip semantics, never-cache, and interruption/resume are all delivered.

## Milestones

- [ ] **M1 — request, submit, wait, evidence.**
  - Task 1: `ParseManualPolicy` + Validate + CUE, and `UncachedSteps` treats manual as uncached. Tests: a table of policies; a manual step and its descendants are uncached.
  - Task 2: `pkg/grade`: types, `ResultIdentity`, `RequestID`, `ValidateSubmission`. Tests: every row of the decision table.
  - Task 3: `manualExecutor`: request before launch, `METIS_GRADE_REQUEST`, presenter, wait loop (injected clock and poll), evidence, metrics, outcome `wait_seconds`, abandon on timeout or cancel.
  - Task 4: `metis grade submit` with `O_EXCL` and the post-create stale re-check.
  - Task 5: e2e through `runExperiment` with a fake presenter:
    - a graded run, with `wait_seconds` separate from `runtime_seconds`;
    - a skip;
    - wrong-request, off-scale, duplicate and conflicting submissions;
    - timeout and cancel while pending (no grade, stale afterwards);
    - a timeout racing a submission;
    - two repeats each getting their own request (no reuse).
  - Atlas.
- [ ] **M2 — resume and reconciliation.**
  - Task 6: `execution.json` at claim.
  - Task 7: `resumePlan` (pure, table-tested).
  - Task 8: `metis run --resume`: identity checks, restore from disk, re-enter the wait, run the remaining steps uncached, final record and row (#73's publish).
  - Task 9: e2e:
    - interrupt while pending (cancel the runner but keep the state pending; a seam simulates the process dying);
    - submit while down, then resume finalizes;
    - resume with an edited experiment refuses;
    - an upstream byte changed refuses as a mismatch;
    - nothing pending refuses.
  - Atlas.
