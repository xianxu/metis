---
reviewer_runs:
    - version: 1
      intent:
        id: 4e777be5-855f-4c4b-b5bf-b9d263e2a8e5
        repository: github.com/xianxu/metis
        issue_id: "000074"
        recipe: estimate-quality
        round: 1
        attempt: 1
        agent: claude
        root: /Users/xianxu/workspace/worktree/metis-slot1/metis
        reviewed_head: 1a28c6045938cf341c0817529ffb87c9bdc54317
        slot: metis:1
        requested_at: "2026-10-24T03:04:23.811707Z"
      phase: finished
      pid: 10353
      started_at: "2026-10-24T03:04:24.084034Z"
      finished_at: "2026-10-24T03:04:49.004178Z"
      session_id: a30ffe7b-3f25-42f0-af36-877b1fc65c1c
      outcome: returned
gate: plan-quality
issue: 74
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 1c5fe7d3-f2d0-46c9-ba71-ac2b37d9db97
            repository: github.com/xianxu/metis
            issue_id: "000074"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/metis-slot1/metis
            reviewed_head: 344e0c3b868300d49fe7eae02d4569c4451093e4
            slot: metis:1
            requested_at: "2026-10-24T03:01:00.029241Z"
          phase: finished
          pid: 79768
          started_at: "2026-10-24T03:01:00.206687Z"
          finished_at: "2026-10-24T03:02:03.089799Z"
          session_id: f467b1f3-037a-46f8-ae3f-1c7bb7453741
          outcome: returned
      "n": 1
      timestamp: "2026-10-23T20:02:03-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: PQ-1
          severity: Important
          title: Grade bookkeeping files would be hashed as step artifacts; re-collection after finalization unstated
          detail: collectArtifacts (cmd/metis/exec.go:229-255) reserves only with.json/metrics.json/reads.json, so grade-request.json, the mutable grade-state.json and submission.json would enter the output hash, contradicting decision 5. execStep.Execute collects the result when the presenter exits (exec.go:161), before grade.json exists. Extend the single reserved list (ARCH-DRY) and state that manualExecutor re-collects after finalization.
          family: step-output-bookkeeping-boundary
          round: 1
        - kind: defect
          id: PQ-2
          severity: Important
          title: Timeout-vs-submission race resolution still loses an accepted grade
          detail: 'If the runner polls (no file), then the submitter does O_EXCL create plus a state read (pending, reports accepted), and only then the runner writes abandoned, the human is told the grade was accepted but it is dropped. Make submission.json the sole linearization point: the runner abandons by creating it exclusively itself, as a tombstone, so the first creator wins. Sequence-test this ordering (ARCH-ORDER).'
          family: single-linearization-point
          round: 1
        - kind: defect
          id: PQ-3
          severity: Important
          title: metis run --resume takes no ownership of the run dir; a second runner is unguarded
          detail: Two concurrent resumes, or a resume while the original process is still alive, would both wait, finalize and publish. State how resume takes over the run-dir claim from metis#73 (verify the owner is dead and take over atomically, or refuse). This is the ARCH-ORDER second-actor event.
          family: run-dir-ownership-on-resume
          round: 1
        - kind: defect
          id: PQ-4
          severity: Minor
          title: wait_seconds measured from requested_at overlaps presenter runtime
          detail: The request is written before the presenter launches, so the wait includes compute time already counted in runtime_seconds. Measure from the presenter's exit (pending entry).
          family: wait-measurement-boundary
          round: 1
        - kind: defect
          id: PQ-5
          severity: Minor
          title: Pending wait's hold on run-control admission and sweep concurrency is unstated
          detail: runResolvedExperiment holds a runControl admission slot for the whole run (cmd/metis/run.go:233). A wait of minutes to days, or many sweep points pending at once, may block other runs. Name the bound (ARCH-CONSTRAINTS).
          family: operating-envelope-unstated
          round: 1
        - kind: defect
          id: PQ-6
          severity: Minor
          title: Tasks 5 and 9 enumerate e2e cases in prose instead of one strategy line
          detail: 'Compress to the adversarial class and guard, for example: sequences of submissions and interruptions against a fake presenter with a controlled clock.'
          family: enumerated-test-prose
          round: 1
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            id: 21d0ef11-4a4b-4446-9a25-828a818065b4
            repository: github.com/xianxu/metis
            issue_id: "000074"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/metis-slot1/metis
            reviewed_head: 1a28c6045938cf341c0817529ffb87c9bdc54317
            slot: metis:1
            requested_at: "2026-10-24T03:03:08.192787Z"
          phase: finished
          pid: 72081
          started_at: "2026-10-24T03:03:08.395311Z"
          finished_at: "2026-10-24T03:03:30.498092Z"
          session_id: 01f7864c-60b8-4d28-918e-def354561c29
          outcome: returned
      "n": 2
      timestamp: "2026-10-23T20:03:30-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Request/state/submission files join the single reserved set; manualExecutor re-collects after finalization so grade.json enters the hash.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: submission.json exclusive create is the sole linearization point; runner abandons via tombstone; first creator wins; sequence-tested.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Lifetime flock lease on run-dir/.lease, taken non-blocking by resume; kernel releases on death.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: wait_seconds measured from presenter exit (pending entry) to winning submission.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Manual steps refused in sweep shapes at ValidateShape; single run holds only its own slot.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Tasks 5 and 9 replaced by one strategy line each.
          round: 2
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: 25bbdaea-538a-4ac4-ab87-0322c1f61a9c
            repository: github.com/xianxu/metis
            issue_id: "000074"
            recipe: plan-quality
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/metis-slot1/metis
            reviewed_head: 1a28c6045938cf341c0817529ffb87c9bdc54317
            slot: metis:1
            requested_at: "2026-10-24T03:04:02.413766Z"
          phase: finished
          pid: 99572
          started_at: "2026-10-24T03:04:02.634867Z"
          finished_at: "2026-10-24T03:04:20.248535Z"
          session_id: d4c72340-606c-410f-a3c1-09e4780ec0d6
          outcome: returned
      "n": 3
      timestamp: "2026-10-23T20:04:20-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Bookkeeping files are reserved (single reservedStepFiles set); re-collect after finalization stated.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: submission.json exclusive create is the sole linearization point; tombstone on timeout/cancel.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: flock lease on run-dir/.lease held by every runner; resume takes it non-blocking.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: wait_seconds measured from presenter exit (pending entry).
          round: 3
        - id: PQ-5
          disposition: addressed
          note: Manual steps refused in sweep runs at ValidateShape; admission hold bounded to the run itself.
          round: 3
        - id: PQ-6
          disposition: addressed
          note: Tasks 5 and 9 replaced by one strategy line each.
          round: 3
      blocked: false
content_hash: 098e5e7ecfd8cbd2dc51f0a741d15516f3ceca276bc7458947a93d385c03b15f
---

# Gate ledger — metis#74 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T20:02:03-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `step-output-bookkeeping-boundary` (kind: defect) Grade bookkeeping files would be hashed as step artifacts; re-collection after finalization unstated
  collectArtifacts (cmd/metis/exec.go:229-255) reserves only with.json/metrics.json/reads.json, so grade-request.json, the mutable grade-state.json and submission.json would enter the output hash, contradicting decision 5. execStep.Execute collects the result when the presenter exits (exec.go:161), before grade.json exists. Extend the single reserved list (ARCH-DRY) and state that manualExecutor re-collects after finalization.
- **PQ-2** [Important] `single-linearization-point` (kind: defect) Timeout-vs-submission race resolution still loses an accepted grade
  If the runner polls (no file), then the submitter does O_EXCL create plus a state read (pending, reports accepted), and only then the runner writes abandoned, the human is told the grade was accepted but it is dropped. Make submission.json the sole linearization point: the runner abandons by creating it exclusively itself, as a tombstone, so the first creator wins. Sequence-test this ordering (ARCH-ORDER).
- **PQ-3** [Important] `run-dir-ownership-on-resume` (kind: defect) metis run --resume takes no ownership of the run dir; a second runner is unguarded
  Two concurrent resumes, or a resume while the original process is still alive, would both wait, finalize and publish. State how resume takes over the run-dir claim from metis#73 (verify the owner is dead and take over atomically, or refuse). This is the ARCH-ORDER second-actor event.
- **PQ-4** [Minor] `wait-measurement-boundary` (kind: defect) wait_seconds measured from requested_at overlaps presenter runtime
  The request is written before the presenter launches, so the wait includes compute time already counted in runtime_seconds. Measure from the presenter's exit (pending entry).
- **PQ-5** [Minor] `operating-envelope-unstated` (kind: defect) Pending wait's hold on run-control admission and sweep concurrency is unstated
  runResolvedExperiment holds a runControl admission slot for the whole run (cmd/metis/run.go:233). A wait of minutes to days, or many sweep points pending at once, may block other runs. Name the bound (ARCH-CONSTRAINTS).
- **PQ-6** [Minor] `enumerated-test-prose` (kind: defect) Tasks 5 and 9 enumerate e2e cases in prose instead of one strategy line
  Compress to the adversarial class and guard, for example: sequences of submissions and interruptions against a fake presenter with a controlled clock.

## Round 2 — 2026-10-23T20:03:30-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Request/state/submission files join the single reserved set; manualExecutor re-collects after finalization so grade.json enters the hash.
- PQ-2 — addressed — submission.json exclusive create is the sole linearization point; runner abandons via tombstone; first creator wins; sequence-tested.
- PQ-3 — addressed — Lifetime flock lease on run-dir/.lease, taken non-blocking by resume; kernel releases on death.
- PQ-4 — addressed — wait_seconds measured from presenter exit (pending entry) to winning submission.
- PQ-5 — addressed — Manual steps refused in sweep shapes at ValidateShape; single run holds only its own slot.
- PQ-6 — addressed — Tasks 5 and 9 replaced by one strategy line each.

## Round 3 — 2026-10-23T20:04:20-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Bookkeeping files are reserved (single reservedStepFiles set); re-collect after finalization stated.
- PQ-2 — addressed — submission.json exclusive create is the sole linearization point; tombstone on timeout/cancel.
- PQ-3 — addressed — flock lease on run-dir/.lease held by every runner; resume takes it non-blocking.
- PQ-4 — addressed — wait_seconds measured from presenter exit (pending entry).
- PQ-5 — addressed — Manual steps refused in sweep runs at ValidateShape; admission hold bounded to the run itself.
- PQ-6 — addressed — Tasks 5 and 9 replaced by one strategy line each.

## Open findings

(none — every finding has been disposed)
