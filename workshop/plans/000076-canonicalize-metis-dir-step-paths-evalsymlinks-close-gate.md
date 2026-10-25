---
gate: boundary-review
issue: 76
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5a3d17d4-13a5-44c0-bee1-9d56eb3359e0
            repository: github.com/xianxu/metis
            issue_id: "000076"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/e614388b52144281
            reviewed_head: 22ecea94cb4d54586e2653b0f8b7863d800ca2ac
            slot: metis:1
            requested_at: "2026-10-25T15:18:16.784037Z"
          phase: finished
          pid: 51812
          started_at: "2026-10-25T15:18:16.947059Z"
          finished_at: "2026-10-25T15:22:14.086611Z"
          session_id: d29e7622-e8c3-4be8-ad65-a7a9bd4516e5
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-v1
        repository: github.com/xianxu/metis
        code_head: 22ecea94cb4d54586e2653b0f8b7863d800ca2ac
        tracker_head: b0f82b9b5cf96efcff7cfb2c910092de7d0c9832
        branch_point: 27995863bb76020f322317132dc2c1df7783e24b
        since: "2026-10-23T17:48:39-07:00"
        until: "2026-10-23T19:38:16-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T17:48:39-07:00"
              end: "2026-10-23T19:38:16-07:00"
              began: "2026-10-23T17:48:39-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/metis-slot1/metis
        sources:
            - harness: claude
              id: 621e2c42-a080-4dc1-9165-83040e197e38
              slot: metis:1
              cwd: /Users/xianxu/workspace/worktree/metis-slot1/metis
              digest: 09b39e3a014dd6b4ed323e45c74b61a485eae6b7660b366f9e327eb3248de43a
              started: "2026-10-24T00:13:14.303Z"
              spawned: false
              bytes: 6662995
              windows:
                - start: "2026-10-23T19:35:19-07:00"
                  end: "2026-10-24T02:37:38Z"
              dispatch_windows:
                - start: "2026-10-23T17:48:39-07:00"
                  end: "2026-10-23T19:38:16-07:00"
        branches:
            - root: /Users/xianxu/workspace/worktree/metis-slot1/metis
              path: /Users/xianxu/workspace/metis/.git/worktrees/metis/logs/HEAD
              head: 22ecea94cb4d54586e2653b0f8b7863d800ca2ac
              branch: 000076-canonicalize-metis-dir-step-paths-evalsymlinks
              digest: 97d08e4af911c7d4cde07fb0bbd04a6f1f0e3582e7809bee4d0e6cb87443cf2b
              bytes: 16600
              intervals:
                - source_index: 0
                  start: "2026-10-23T17:48:39-07:00"
                  end: "2026-10-24T02:34:58Z"
                  branch: 000075-make-test-target-offline-go-pytest-suite-for-local-check
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:34:58Z"
                  end: "2026-10-23T19:35:19-07:00"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-23T19:35:19-07:00"
                  end: "2026-10-24T02:37:38Z"
                  kind: branch
                - source_index: 0
                  start: "2026-10-24T02:37:38Z"
                  end: "2026-10-23T19:38:16-07:00"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
      "n": 1
      timestamp: "2026-10-25T08:22:16-07:00"
      dispatched: "2026-10-25T08:18:16-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Physical METIS_EXP_DIR vs logical METIS_READ_ROOT breaks nested-CV read confinement under symlinked dirs
          detail: 'run.go:251 now EvalSymlinks expDir, but sweep.go:743 builds readRoot (analysisAbs) with filepath.Abs only, and metis/io.py within_root compares with os.path.abspath. Under /tmp or macOS TMPDIR every sealed inner-CV base-dataset read raises "read confinement ... outside METIS_READ_ROOT". Reproduced: runExperiment through a symlink with readRoot=Abs(Dir(expPath)/analysis_0) and real metis/cv-split fails at head and passes with base run.go. Fix: canonicalize readRoot with the same helper (and optionally realpath in within_root); add a regression test.'
          family: child-env-paths-canonical-form
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: blend.go still hands Abs-only METIS_EXP_DIR/METIS_RUN_DIR to the submission step
          detail: blend.go:110-114 builds expDir/runsDir with filepath.Abs only and passes them to execStep (line 204), so metis blend under a symlinked dir still gives the child non-physical paths, contradicting the new atlas/experiment.md claim. Extract a shared canonicalize helper (ARCH-DRY/ARCH-PURPOSE) and use it at every place that builds child env paths.
          family: child-env-paths-canonical-form
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: runDir is not physical if expDir/runs is itself a symlink
          detail: runDir = Join(resolved expDir, "runs", runID) does not resolve an existing symlinked runs/ directory; resolving the deepest existing ancestor would cover it.
          family: child-env-paths-canonical-form
          round: 1
      recipe: milestone-review
      isolation: checkout
      blocked: true
---

# Gate ledger — metis#76 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-25T08:22:16-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `child-env-paths-canonical-form` (kind: defect) Physical METIS_EXP_DIR vs logical METIS_READ_ROOT breaks nested-CV read confinement under symlinked dirs
  run.go:251 now EvalSymlinks expDir, but sweep.go:743 builds readRoot (analysisAbs) with filepath.Abs only, and metis/io.py within_root compares with os.path.abspath. Under /tmp or macOS TMPDIR every sealed inner-CV base-dataset read raises "read confinement ... outside METIS_READ_ROOT". Reproduced: runExperiment through a symlink with readRoot=Abs(Dir(expPath)/analysis_0) and real metis/cv-split fails at head and passes with base run.go. Fix: canonicalize readRoot with the same helper (and optionally realpath in within_root); add a regression test.
- **BR-2** [Important] `child-env-paths-canonical-form` (kind: defect) blend.go still hands Abs-only METIS_EXP_DIR/METIS_RUN_DIR to the submission step
  blend.go:110-114 builds expDir/runsDir with filepath.Abs only and passes them to execStep (line 204), so metis blend under a symlinked dir still gives the child non-physical paths, contradicting the new atlas/experiment.md claim. Extract a shared canonicalize helper (ARCH-DRY/ARCH-PURPOSE) and use it at every place that builds child env paths.
- **BR-3** [Minor] `child-env-paths-canonical-form` (kind: defect) runDir is not physical if expDir/runs is itself a symlink
  runDir = Join(resolved expDir, "runs", runID) does not resolve an existing symlinked runs/ directory; resolving the deepest existing ancestor would cover it.

## Open findings

- **BR-1** [Critical] `child-env-paths-canonical-form` (kind: defect) Physical METIS_EXP_DIR vs logical METIS_READ_ROOT breaks nested-CV read confinement under symlinked dirs
- **BR-2** [Important] `child-env-paths-canonical-form` (kind: defect) blend.go still hands Abs-only METIS_EXP_DIR/METIS_RUN_DIR to the submission step
- **BR-3** [Minor] `child-env-paths-canonical-form` (kind: defect) runDir is not physical if expDir/runs is itself a symlink
