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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c0a74e83-49f3-43f3-a8ac-5869f822bc4e
            repository: github.com/xianxu/metis
            issue_id: "000076"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/34f53ec0f9221ad3
            reviewed_head: d357a3e0fcae7b3173c5f28a7f136a7696ac6528
            slot: metis:1
            requested_at: "2026-10-25T15:32:54.975358Z"
          phase: finished
          pid: 57030
          started_at: "2026-10-25T15:32:55.09026Z"
          finished_at: "2026-10-25T15:40:33.650768Z"
          session_id: 0bcf94c3-fb51-4b1a-9aa9-b1677cd5e3f4
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-queue-v1
        repository: github.com/xianxu/metis
        code_head: d357a3e0fcae7b3173c5f28a7f136a7696ac6528
        tracker_head: b0f82b9b5cf96efcff7cfb2c910092de7d0c9832
        branch_point: 27995863bb76020f322317132dc2c1df7783e24b
        since: "2026-10-23T17:48:39-07:00"
        until: "2026-10-25T08:25:19-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T17:48:39-07:00"
              end: "2026-10-25T08:25:19-07:00"
              began: "2026-10-23T17:48:39-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/metis-slot1/metis
        sources:
            - harness: claude
              id: 621e2c42-a080-4dc1-9165-83040e197e38
              slot: metis:1
              cwd: /Users/xianxu/workspace/worktree/metis-slot1/metis
              digest: 9dbc7990ce0955b1d911e749a7b91214d9f2d668d77814fe55bf491d8cbb8b98
              started: "2026-10-24T00:13:14.303Z"
              spawned: false
              bytes: 7032641
              windows:
                - start: "2026-10-24T02:35:19Z"
                  end: "2026-10-24T02:37:38Z"
                - start: "2026-10-24T02:38:16Z"
                  end: "2026-10-24T02:38:23Z"
                - start: "2026-10-25T14:54:53Z"
                  end: "2026-10-25T14:55:04Z"
                - start: "2026-10-25T14:56:09Z"
                  end: "2026-10-25T08:25:19-07:00"
              dispatch_windows:
                - start: "2026-10-23T17:48:39-07:00"
                  end: "2026-10-25T08:25:19-07:00"
        queue_waits:
            - attempt_id: 05a68dc164e62c5fd8b6e237f23727cf
              slot: metis:1
              record_digest: 2d1491332fc987aa0ba1ec7b174a3fbc73d993e593553cc6728e7ce8cb62b856
              start: "2026-10-25T14:56:30.226923Z"
              end: "2026-10-25T15:15:32.149005Z"
              queue_seconds: 1141.922082
              raw_seconds: 1141.922082
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 45.34525
        cuts:
            - dispatched: "2026-10-25T08:18:16-07:00"
              finalized: "2026-10-25T08:22:16-07:00"
              finalizing: false
        branches:
            - root: /Users/xianxu/workspace/worktree/metis-slot1/metis
              path: /Users/xianxu/workspace/metis/.git/worktrees/metis/logs/HEAD
              head: d357a3e0fcae7b3173c5f28a7f136a7696ac6528
              branch: 000076-canonicalize-metis-dir-step-paths-evalsymlinks
              digest: 46612238aef10658869f1229f4bfe45a36d4299875c8dbdcf00d317587f2bf0b
              bytes: 16804
              intervals:
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T02:44:26.641Z"
                  end: "2026-10-24T02:48:39Z"
                  branch: 000072-step-timeout-live-log
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T02:48:39Z"
                  end: "2026-10-24T02:48:51Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T02:48:51Z"
                  end: "2026-10-24T02:53:21Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T02:53:21Z"
                  end: "2026-10-24T02:53:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T02:53:32Z"
                  end: "2026-10-24T02:53:48Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T02:53:48Z"
                  end: "2026-10-24T02:53:49Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T02:53:49Z"
                  end: "2026-10-24T02:55:54Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T02:55:54Z"
                  end: "2026-10-24T03:00:52Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-24T03:00:52Z"
                  end: "2026-10-25T11:16:02Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T11:16:02Z"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T14:40:39Z"
                  branch: 000070-ci-weave-pin
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T14:40:39Z"
                  end: "2026-10-25T14:41:01Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T14:41:01Z"
                  end: "2026-10-25T14:44:45Z"
                  branch: 000075-make-test-target-offline-go-pytest-suite-for-local-check
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T14:44:45Z"
                  end: "2026-10-25T14:45:14Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T14:45:14Z"
                  end: "2026-10-25T14:54:53Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T14:54:53Z"
                  end: "2026-10-25T14:55:04Z"
                  kind: branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T14:55:04Z"
                  end: "2026-10-25T14:55:43Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T14:55:43Z"
                  end: "2026-10-25T14:56:09Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T14:56:09Z"
                  end: "2026-10-25T08:25:19-07:00"
                  kind: branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T02:47:37.483Z"
                  end: "2026-10-24T02:48:39Z"
                  branch: 000072-step-timeout-live-log
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T02:48:39Z"
                  end: "2026-10-24T02:48:51Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T02:48:51Z"
                  end: "2026-10-24T02:53:21Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T02:53:21Z"
                  end: "2026-10-24T02:53:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T02:53:32Z"
                  end: "2026-10-24T02:53:48Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T02:53:48Z"
                  end: "2026-10-24T02:53:49Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T02:53:49Z"
                  end: "2026-10-24T02:55:54Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T02:55:54Z"
                  end: "2026-10-24T03:00:52Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-24T03:00:52Z"
                  end: "2026-10-25T11:16:02Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T11:16:02Z"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T14:40:39Z"
                  branch: 000070-ci-weave-pin
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T14:40:39Z"
                  end: "2026-10-25T14:41:01Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T14:41:01Z"
                  end: "2026-10-25T14:44:45Z"
                  branch: 000075-make-test-target-offline-go-pytest-suite-for-local-check
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T14:44:45Z"
                  end: "2026-10-25T14:45:14Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T14:45:14Z"
                  end: "2026-10-25T14:54:53Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T14:54:53Z"
                  end: "2026-10-25T14:55:04Z"
                  kind: branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T14:55:04Z"
                  end: "2026-10-25T14:55:43Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T14:55:43Z"
                  end: "2026-10-25T14:56:09Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T14:56:09Z"
                  end: "2026-10-25T08:25:19-07:00"
                  kind: branch
                - source_index: 0
                  start: "2026-10-23T17:48:39-07:00"
                  end: "2026-10-24T02:34:58Z"
                  branch: 000075-make-test-target-offline-go-pytest-suite-for-local-check
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:34:58Z"
                  end: "2026-10-24T02:35:19Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-24T02:35:19Z"
                  end: "2026-10-24T02:37:38Z"
                  kind: branch
                - source_index: 0
                  start: "2026-10-24T02:37:38Z"
                  end: "2026-10-24T02:38:16Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:38:16Z"
                  end: "2026-10-24T02:38:23Z"
                  kind: branch
                - source_index: 0
                  start: "2026-10-24T02:38:23Z"
                  end: "2026-10-24T02:38:33Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-24T02:38:33Z"
                  end: "2026-10-24T02:48:39Z"
                  branch: 000072-step-timeout-live-log
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:48:39Z"
                  end: "2026-10-24T02:48:51Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-24T02:48:51Z"
                  end: "2026-10-24T02:53:21Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:53:21Z"
                  end: "2026-10-24T02:53:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-24T02:53:32Z"
                  end: "2026-10-24T02:53:48Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:53:48Z"
                  end: "2026-10-24T02:53:49Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:53:49Z"
                  end: "2026-10-24T02:55:54Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T02:55:54Z"
                  end: "2026-10-24T03:00:52Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-24T03:00:52Z"
                  end: "2026-10-25T11:16:02Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-25T11:16:02Z"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T14:40:39Z"
                  branch: 000070-ci-weave-pin
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-25T14:40:39Z"
                  end: "2026-10-25T14:41:01Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T14:41:01Z"
                  end: "2026-10-25T14:44:45Z"
                  branch: 000075-make-test-target-offline-go-pytest-suite-for-local-check
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-25T14:44:45Z"
                  end: "2026-10-25T14:45:14Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T14:45:14Z"
                  end: "2026-10-25T14:54:53Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-25T14:54:53Z"
                  end: "2026-10-25T14:55:04Z"
                  kind: branch
                - source_index: 0
                  start: "2026-10-25T14:55:04Z"
                  end: "2026-10-25T14:55:43Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source_index: 0
                  start: "2026-10-25T14:55:43Z"
                  end: "2026-10-25T14:56:09Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T14:56:09Z"
                  end: "2026-10-25T08:25:19-07:00"
                  kind: branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T02:47:22.513Z"
                  end: "2026-10-24T02:48:39Z"
                  branch: 000072-step-timeout-live-log
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T02:48:39Z"
                  end: "2026-10-24T02:48:51Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T02:48:51Z"
                  end: "2026-10-24T02:53:21Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T02:53:21Z"
                  end: "2026-10-24T02:53:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T02:53:32Z"
                  end: "2026-10-24T02:53:48Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T02:53:48Z"
                  end: "2026-10-24T02:53:49Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T02:53:49Z"
                  end: "2026-10-24T02:55:54Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T02:55:54Z"
                  end: "2026-10-24T03:00:52Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-24T03:00:52Z"
                  end: "2026-10-25T11:16:02Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T11:16:02Z"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T14:40:39Z"
                  branch: 000070-ci-weave-pin
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T14:40:39Z"
                  end: "2026-10-25T14:41:01Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T14:41:01Z"
                  end: "2026-10-25T14:44:45Z"
                  branch: 000075-make-test-target-offline-go-pytest-suite-for-local-check
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T14:44:45Z"
                  end: "2026-10-25T14:45:14Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T14:45:14Z"
                  end: "2026-10-25T14:54:53Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T14:54:53Z"
                  end: "2026-10-25T14:55:04Z"
                  kind: branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T14:55:04Z"
                  end: "2026-10-25T14:55:43Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T14:55:43Z"
                  end: "2026-10-25T14:56:09Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T14:56:09Z"
                  end: "2026-10-25T08:25:19-07:00"
                  kind: branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T02:46:00.091Z"
                  end: "2026-10-24T02:48:39Z"
                  branch: 000072-step-timeout-live-log
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T02:48:39Z"
                  end: "2026-10-24T02:48:51Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T02:48:51Z"
                  end: "2026-10-24T02:53:21Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T02:53:21Z"
                  end: "2026-10-24T02:53:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T02:53:32Z"
                  end: "2026-10-24T02:53:48Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T02:53:48Z"
                  end: "2026-10-24T02:53:49Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T02:53:49Z"
                  end: "2026-10-24T02:55:54Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T02:55:54Z"
                  end: "2026-10-24T03:00:52Z"
                  branch: 000073-repeated-run-ledger
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-24T03:00:52Z"
                  end: "2026-10-25T11:16:02Z"
                  branch: 000074-manual-grading-step
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T11:16:02Z"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T14:40:39Z"
                  branch: 000070-ci-weave-pin
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T14:40:39Z"
                  end: "2026-10-25T14:41:01Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T14:41:01Z"
                  end: "2026-10-25T14:44:45Z"
                  branch: 000075-make-test-target-offline-go-pytest-suite-for-local-check
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T14:44:45Z"
                  end: "2026-10-25T14:45:14Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T14:45:14Z"
                  end: "2026-10-25T14:54:53Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T14:54:53Z"
                  end: "2026-10-25T14:55:04Z"
                  kind: branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T14:55:04Z"
                  end: "2026-10-25T14:55:43Z"
                  branch: 000071-never-cache-steps
                  kind: foreign-branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T14:55:43Z"
                  end: "2026-10-25T14:56:09Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T14:56:09Z"
                  end: "2026-10-25T08:25:19-07:00"
                  kind: branch
      "n": 2
      timestamp: "2026-10-25T08:40:45-07:00"
      dispatched: "2026-10-25T08:32:54-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: stepEnv now resolves METIS_READ_ROOT via physicalPath (exec.go:54-73); TestExecStep_ConfinementAgreesAcrossPathForms (real uv cv-split) fails with the resolution disabled and passes at head.
          round: 2
        - id: BR-2
          disposition: addressed
          note: blend calls es.Execute, which goes through stepEnv, so its Abs-only expDir/runDir are made physical there; TestExecStep_StepEnvPathsArePhysical (expDir passed as a symlink) fails without the fix.
          round: 2
        - id: BR-3
          disposition: addressed
          note: physicalPath resolves the deepest existing ancestor; the test symlinks runs/ and asserts physical RUN_DIR/STEP_DIR/READ_ROOT, red without the fix.
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Minor
          title: lessons.md gains an unrelated metis#70 lesson inside the metis#76 commit
          detail: workshop/lessons.md:61-62 adds a lesson about a check that hides a tool's stderr (metis#70) in commit d357a3e. Move it to the metis#70 work, or say in the Log why it rides along here.
          family: issue-scope-hygiene
          round: 2
        - kind: defect
          id: BR-5
          severity: Minor
          title: physicalPath's root-fallback, non-ENOENT Lstat and dangling-symlink branches have no direct test
          detail: 'cmd/metis/physpath.go:22-33. Add a small table test over temp dirs: a fully nonexistent path, a path whose ancestor is a regular file (ENOTDIR gives an error), and a dangling symlink (gives an error), so a later edit cannot quietly start falling back.'
          family: helper-edge-branches-tested
          round: 2
      recipe: milestone-review
      isolation: checkout
      reviewed: d357a3e0fcae7b3173c5f28a7f136a7696ac6528
      blocked: false
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

## Round 2 — 2026-10-25T08:40:45-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — stepEnv now resolves METIS_READ_ROOT via physicalPath (exec.go:54-73); TestExecStep_ConfinementAgreesAcrossPathForms (real uv cv-split) fails with the resolution disabled and passes at head.
- BR-2 — addressed — blend calls es.Execute, which goes through stepEnv, so its Abs-only expDir/runDir are made physical there; TestExecStep_StepEnvPathsArePhysical (expDir passed as a symlink) fails without the fix.
- BR-3 — addressed — physicalPath resolves the deepest existing ancestor; the test symlinks runs/ and asserts physical RUN_DIR/STEP_DIR/READ_ROOT, red without the fix.

### Raised

- **BR-4** [Minor] `issue-scope-hygiene` (kind: defect) lessons.md gains an unrelated metis#70 lesson inside the metis#76 commit
  workshop/lessons.md:61-62 adds a lesson about a check that hides a tool's stderr (metis#70) in commit d357a3e. Move it to the metis#70 work, or say in the Log why it rides along here.
- **BR-5** [Minor] `helper-edge-branches-tested` (kind: defect) physicalPath's root-fallback, non-ENOENT Lstat and dangling-symlink branches have no direct test
  cmd/metis/physpath.go:22-33. Add a small table test over temp dirs: a fully nonexistent path, a path whose ancestor is a regular file (ENOTDIR gives an error), and a dangling symlink (gives an error), so a later edit cannot quietly start falling back.

## Open findings

- **BR-4** [Minor] `issue-scope-hygiene` (kind: defect) lessons.md gains an unrelated metis#70 lesson inside the metis#76 commit
- **BR-5** [Minor] `helper-edge-branches-tested` (kind: defect) physicalPath's root-fallback, non-ENOENT Lstat and dangling-symlink branches have no direct test
