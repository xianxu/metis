---
gate: boundary-review
issue: 70
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: af406c07-04bc-4f84-a7ad-e15a006f0481
            repository: github.com/xianxu/metis
            issue_id: "000070"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/5135b727569e6e8c
            reviewed_head: c880dd42ca36d313e138407e3266940b529efde7
            slot: metis:1
            requested_at: "2026-10-25T11:21:06.557619Z"
          phase: finished
          pid: 59050
          started_at: "2026-10-25T11:21:06.657592Z"
          finished_at: "2026-10-25T11:21:54.763846Z"
          session_id: 04f6914c-72f1-4681-9d8b-e16d755658ba
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-v1
        repository: github.com/xianxu/metis
        code_head: c880dd42ca36d313e138407e3266940b529efde7
        tracker_head: 84d50b5e9e6213f74b6b8ae18ef116f0871efb36
        branch_point: 4cb2c455fd62430e4ab25fdddeb940236c30f74d
        since: "2026-10-25T04:16:09-07:00"
        until: "2026-10-25T04:20:00-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-25T04:16:09-07:00"
              end: "2026-10-25T04:20:00-07:00"
              began: "2026-10-25T04:16:09-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/metis-slot1/metis
        sources:
            - harness: claude
              id: 621e2c42-a080-4dc1-9165-83040e197e38
              slot: metis:1
              cwd: /Users/xianxu/workspace/worktree/metis-slot1/metis
              digest: 1d881c42371e84ab1784647cacd56e7e52faf3ee612186aa0c86d84eb7c62729
              started: "2026-10-24T00:13:14.303Z"
              spawned: false
              bytes: 5248368
              windows:
                - start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:20:00-07:00"
              dispatch_windows:
                - start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T04:20:00-07:00"
        branches:
            - root: /Users/xianxu/workspace/worktree/metis-slot1/metis
              path: /Users/xianxu/workspace/metis/.git/worktrees/metis/logs/HEAD
              head: c880dd42ca36d313e138407e3266940b529efde7
              branch: 000070-ci-weave-pin
              digest: 4eb7bcf301c5f9ae8f1e278658e02af40a00c5fdcb32b419e74de4cb9d16fa91
              bytes: 11480
              intervals:
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:20:00-07:00"
                  kind: branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:20:00-07:00"
                  kind: branch
                - source_index: 0
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:20:00-07:00"
                  kind: branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:20:00-07:00"
                  kind: branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:20:00-07:00"
                  kind: branch
      "n": 1
      timestamp: "2026-10-25T04:21:56-07:00"
      dispatched: "2026-10-25T04:21:06-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: '"Pin survives compile" is shown structurally, not by a full weave compile run'
          detail: The Log records that weave refresh refused and weave compile stopped at .claude/settings.json, so evidence rests on no manifest naming .sdlc (confirmed). Record a full compile run, or the PR CI run, as the evidence before closing. This is the only instance in the window.
          family: done-when-evidence-partial
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: c880dd42ca36d313e138407e3266940b529efde7
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 33944692-9e9a-41b3-95b8-7fb6c7290239
            repository: github.com/xianxu/metis
            issue_id: "000070"
            recipe: fix-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/de2e5333afd39e97
            reviewed_head: 54810af9368b4cf28cc4168cff626a339c5ab46a
            slot: metis:1
            requested_at: "2026-10-25T11:34:22.989502Z"
          phase: finished
          pid: 47356
          started_at: "2026-10-25T11:34:23.168431Z"
          finished_at: "2026-10-25T11:35:04.923465Z"
          session_id: 327fe450-4bb6-4b84-80ef-584e25ebd7b0
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-v1
        repository: github.com/xianxu/metis
        code_head: 54810af9368b4cf28cc4168cff626a339c5ab46a
        tracker_head: 303fb79c2695ed91822dcb74540fe1524a50928c
        branch_point: 4cb2c455fd62430e4ab25fdddeb940236c30f74d
        since: "2026-10-25T04:16:09-07:00"
        until: "2026-10-25T04:33:32-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-25T04:16:09-07:00"
              end: "2026-10-25T04:33:32-07:00"
              began: "2026-10-25T04:16:09-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/metis-slot1/metis
        sources:
            - harness: claude
              id: 621e2c42-a080-4dc1-9165-83040e197e38
              slot: metis:1
              cwd: /Users/xianxu/workspace/worktree/metis-slot1/metis
              digest: c75a3d8a9754033f8433c1af88ee9e40251d34f1c8a2a3425c562a3bc8ed76bf
              started: "2026-10-24T00:13:14.303Z"
              spawned: false
              bytes: 5954201
              windows:
                - start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:33:32-07:00"
              dispatch_windows:
                - start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T04:33:32-07:00"
        cuts:
            - dispatched: "2026-10-25T04:21:06-07:00"
              finalized: "2026-10-25T04:21:56-07:00"
              finalizing: true
        branches:
            - root: /Users/xianxu/workspace/worktree/metis-slot1/metis
              path: /Users/xianxu/workspace/metis/.git/worktrees/metis/logs/HEAD
              head: 54810af9368b4cf28cc4168cff626a339c5ab46a
              branch: 000070-ci-weave-pin
              digest: 4bde4a007da82e0c73815d89d0850ac0fbb824b099159456414d032366709fdf
              bytes: 12717
              intervals:
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:33:32-07:00"
                  kind: branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:33:32-07:00"
                  kind: branch
                - source_index: 0
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:33:32-07:00"
                  kind: branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:33:32-07:00"
                  kind: branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:33:32-07:00"
                  kind: branch
      "n": 2
      timestamp: "2026-10-25T04:35:07-07:00"
      dispatched: "2026-10-25T04:34:22-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'The Log records only the failing runs 40188584381 and 40188715202; the passing PR #52 merge-check appears only in the subject of commit 54810af, with no run ID or result in the issue.'
          round: 2
      findings:
        - kind: defect
          id: BR-2
          severity: Important
          title: 'Scope changed after close: run a full sdlc close'
          detail: The interdiff adds behaviour no prior finding asked for. construct/deps now clones over https instead of ssh, the new scripts/ci-setup.sh post-compile hook adds layer bin dirs to GITHUB_PATH, and the new scripts/merge-checks.d/_vocab.bash guard makes both experiment checks exit 2 when vocabulary is missing. The Log calls these scope revisions 1 and 2. A narrow fix review does not cover them; run a full sdlc close and record them under the Revisions heading.
          family: scope-changed-after-close
          round: 2
      recipe: fix-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: bc42968f-393f-4a0c-8485-c4cf6853c694
            repository: github.com/xianxu/metis
            issue_id: "000070"
            recipe: small-diff-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/0c71a2a145c281d4
            reviewed_head: 4c7ce7e5c53c45ed0ac5666a1fb81470a1f4b65f
            slot: metis:1
            requested_at: "2026-10-25T11:36:47.060401Z"
          phase: finished
          pid: 28778
          started_at: "2026-10-25T11:36:47.261083Z"
          finished_at: "2026-10-25T11:39:06.250149Z"
          session_id: 34376b90-cfdf-492d-8c22-c00929d3ccb7
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-v1
        repository: github.com/xianxu/metis
        code_head: 4c7ce7e5c53c45ed0ac5666a1fb81470a1f4b65f
        tracker_head: 303fb79c2695ed91822dcb74540fe1524a50928c
        branch_point: 4cb2c455fd62430e4ab25fdddeb940236c30f74d
        since: "2026-10-25T04:16:09-07:00"
        until: "2026-10-25T04:35:51-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-25T04:16:09-07:00"
              end: "2026-10-25T04:35:51-07:00"
              began: "2026-10-25T04:16:09-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/metis-slot1/metis
        sources:
            - harness: claude
              id: 621e2c42-a080-4dc1-9165-83040e197e38
              slot: metis:1
              cwd: /Users/xianxu/workspace/worktree/metis-slot1/metis
              digest: 1dcbe8f73576e2e33f0e9231b1054548cbbab9601e0dd181b9a54544d8f1de8e
              started: "2026-10-24T00:13:14.303Z"
              spawned: false
              bytes: 6055526
              windows:
                - start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:35:51-07:00"
              dispatch_windows:
                - start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T04:35:51-07:00"
        cuts:
            - dispatched: "2026-10-25T04:21:06-07:00"
              finalized: "2026-10-25T04:21:56-07:00"
              finalizing: true
            - dispatched: "2026-10-25T04:34:22-07:00"
              finalized: "2026-10-25T04:35:07-07:00"
              finalizing: false
        branches:
            - root: /Users/xianxu/workspace/worktree/metis-slot1/metis
              path: /Users/xianxu/workspace/metis/.git/worktrees/metis/logs/HEAD
              head: 4c7ce7e5c53c45ed0ac5666a1fb81470a1f4b65f
              branch: 000070-ci-weave-pin
              digest: d8e8b44cda289216c89c87c756d3af99e4cc9890e8cf23e021ba64bfb8d3dc7c
              bytes: 12922
              intervals:
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:35:51-07:00"
                  kind: branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:35:51-07:00"
                  kind: branch
                - source_index: 0
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:35:51-07:00"
                  kind: branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:35:51-07:00"
                  kind: branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:35:51-07:00"
                  kind: branch
      "n": 3
      timestamp: "2026-10-25T04:39:10-07:00"
      dispatched: "2026-10-25T04:36:47-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Log now records passing cold-checkout CI runs 40189012386/40189166668 under the pinned compiler; structurally verified no manifest (metis or ariadne base.manifest) names .sdlc and upstream portable-ci.test.sh:167 asserts the pin is never generated.
          round: 3
        - id: BR-2
          disposition: addressed
          note: 'This is the full close; both scope revisions are recorded under the Revisions heading, and the missing-tool guard is pinned by tests/test_merge_checks.py (verified: reverting both checks fails 2 of 3 tests).'
          round: 3
      findings:
        - kind: defect
          id: BR-3
          severity: Important
          title: Done-when not updated for the two hub-approved scope revisions
          detail: '2nd in family. Rule: a Revisions entry adding behaviour must add the matching Done-when clause in the same commit. Instances in window: (a) construct/deps https clone with local ../ariadne untouched; (b) scripts/ci-setup.sh putting ../ariadne/bin and bin on GITHUB_PATH until ariadne#627; (c) both experiment checks exit 2 naming a missing vocabulary tool (tests/test_merge_checks.py). None appear in the Done-when list at lines 43-54.'
          family: scope-changed-after-close
          round: 3
        - kind: defect
          id: BR-4
          severity: Important
          title: atlas update appears missing for the scripts/ci-setup.sh CI hook and vocabulary guard
          detail: 'atlas/index.md covers the compiler pin only. Add a sentence: ci-setup.sh appends the layer tool dirs to GITHUB_PATH (temporary, removed by ariadne#627), and the experiment merge-checks exit 2 when vocabulary is missing. Only instance in window; the repo has no README.'
          family: atlas-missing-new-surface
          round: 3
        - kind: defect
          id: BR-5
          severity: Minor
          title: Hub report (Done-when clause 5) is ticked in the Plan but has no Log evidence
          detail: '2nd in family. Rule: every Done-when clause needs a Log line naming its evidence (run ID, message, SHA) or saying explicitly that it is pending and who owns it. Instances: clause 5 hub report (no evidence); clause 4 operator label (correctly logged as pending); clause 3 now covered per BR-1.'
          family: done-when-evidence-partial
          round: 3
      recipe: small-diff-review
      isolation: checkout
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 37a641fc-cef9-4328-b6f2-073bd647ec3b
            repository: github.com/xianxu/metis
            issue_id: "000070"
            recipe: small-diff-review
            round: 4
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/f738a43c21729ddf
            reviewed_head: ff82b9576af2eb6b4d9b13eab8307ff752b9dfba
            slot: metis:1
            requested_at: "2026-10-25T11:40:28.550544Z"
          phase: finished
          pid: 57001
          started_at: "2026-10-25T11:40:28.696017Z"
          finished_at: "2026-10-25T11:41:36.600077Z"
          session_id: c304e9bc-752f-4173-98b7-282c759f5c36
          outcome: returned
      actual_inputs:
        version: 4
        method: active-time-v3-branch-v1
        repository: github.com/xianxu/metis
        code_head: ff82b9576af2eb6b4d9b13eab8307ff752b9dfba
        tracker_head: 303fb79c2695ed91822dcb74540fe1524a50928c
        branch_point: 4cb2c455fd62430e4ab25fdddeb940236c30f74d
        since: "2026-10-25T04:16:09-07:00"
        until: "2026-10-25T04:39:36-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-25T04:16:09-07:00"
              end: "2026-10-25T04:39:36-07:00"
              began: "2026-10-25T04:16:09-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/metis-slot1/metis
        sources:
            - harness: claude
              id: 621e2c42-a080-4dc1-9165-83040e197e38
              slot: metis:1
              cwd: /Users/xianxu/workspace/worktree/metis-slot1/metis
              digest: 955b34ae8811e4099990ab76c29b46f1a703f148cb784db2b110fed90941ecd3
              started: "2026-10-24T00:13:14.303Z"
              spawned: false
              bytes: 6147678
              windows:
                - start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:39:36-07:00"
              dispatch_windows:
                - start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T04:39:36-07:00"
        cuts:
            - dispatched: "2026-10-25T04:21:06-07:00"
              finalized: "2026-10-25T04:21:56-07:00"
              finalizing: true
            - dispatched: "2026-10-25T04:34:22-07:00"
              finalized: "2026-10-25T04:35:07-07:00"
              finalizing: false
            - dispatched: "2026-10-25T04:36:47-07:00"
              finalized: "2026-10-25T04:39:10-07:00"
              finalizing: false
        branches:
            - root: /Users/xianxu/workspace/worktree/metis-slot1/metis
              path: /Users/xianxu/workspace/metis/.git/worktrees/metis/logs/HEAD
              head: ff82b9576af2eb6b4d9b13eab8307ff752b9dfba
              branch: 000070-ci-weave-pin
              digest: 24a8dc7c0d92e6dcfa9c9d51168dbc5255253d0e4cac21523eeab847aa8ec88c
              bytes: 13121
              intervals:
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/0869d57a-be29-46b9-99c9-b00decfa590d
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:39:36-07:00"
                  kind: branch
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/551b999c-0156-4396-815b-ab3e30b9f070
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:39:36-07:00"
                  kind: branch
                - source_index: 0
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source_index: 0
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:39:36-07:00"
                  kind: branch
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/e26c5e7a-df9a-4671-86ec-e5ceff7a6ac1
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:39:36-07:00"
                  kind: branch
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T04:16:09-07:00"
                  end: "2026-10-25T11:16:32Z"
                  branch: main-slot1
                  kind: resting-excluded
                - source: claude/f2d4b646-eda5-4c47-8d97-ddd84ac73472
                  start: "2026-10-25T11:16:32Z"
                  end: "2026-10-25T04:39:36-07:00"
                  kind: branch
      "n": 4
      timestamp: "2026-10-25T04:41:40-07:00"
      dispatched: "2026-10-25T04:40:28-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-3
          disposition: addressed
          note: Done-when lines 55-62 now carry Revision 1 (https clone, local ../ariadne never clones) and Revision 2 (ci-setup.sh GITHUB_PATH until ariadne#627; both checks exit 2, pinned by tests/test_merge_checks.py).
          round: 4
        - id: BR-4
          disposition: addressed
          note: atlas/index.md adds a "CI tool PATH hook" entry covering ci-setup.sh, its ariadne#627 removal, _vocab.bash being non-executable, and the exit-2 refusal; this matches scripts/ci-setup.sh and _vocab.bash.
          round: 4
        - id: BR-5
          disposition: addressed
          note: Log clause (5) now names hub receipts 15c3d811, 78b7ef4d, a6af3ffd, 69dc2ddf; clause (4) stays pending and owned by the operator.
          round: 4
      recipe: small-diff-review
      isolation: checkout
      reviewed: ff82b9576af2eb6b4d9b13eab8307ff752b9dfba
      blocked: false
---

# Gate ledger — metis#70 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-25T04:21:56-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `done-when-evidence-partial` (kind: defect) "Pin survives compile" is shown structurally, not by a full weave compile run
  The Log records that weave refresh refused and weave compile stopped at .claude/settings.json, so evidence rests on no manifest naming .sdlc (confirmed). Record a full compile run, or the PR CI run, as the evidence before closing. This is the only instance in the window.

## Round 2 — 2026-10-25T04:35:07-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — The Log records only the failing runs 40188584381 and 40188715202; the passing PR #52 merge-check appears only in the subject of commit 54810af, with no run ID or result in the issue.

### Raised

- **BR-2** [Important] `scope-changed-after-close` (kind: defect) Scope changed after close: run a full sdlc close
  The interdiff adds behaviour no prior finding asked for. construct/deps now clones over https instead of ssh, the new scripts/ci-setup.sh post-compile hook adds layer bin dirs to GITHUB_PATH, and the new scripts/merge-checks.d/_vocab.bash guard makes both experiment checks exit 2 when vocabulary is missing. The Log calls these scope revisions 1 and 2. A narrow fix review does not cover them; run a full sdlc close and record them under the Revisions heading.

## Round 3 — 2026-10-25T04:39:10-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Log now records passing cold-checkout CI runs 40189012386/40189166668 under the pinned compiler; structurally verified no manifest (metis or ariadne base.manifest) names .sdlc and upstream portable-ci.test.sh:167 asserts the pin is never generated.
- BR-2 — addressed — This is the full close; both scope revisions are recorded under the Revisions heading, and the missing-tool guard is pinned by tests/test_merge_checks.py (verified: reverting both checks fails 2 of 3 tests).

### Raised

- **BR-3** [Important] `scope-changed-after-close` (kind: defect) Done-when not updated for the two hub-approved scope revisions
  2nd in family. Rule: a Revisions entry adding behaviour must add the matching Done-when clause in the same commit. Instances in window: (a) construct/deps https clone with local ../ariadne untouched; (b) scripts/ci-setup.sh putting ../ariadne/bin and bin on GITHUB_PATH until ariadne#627; (c) both experiment checks exit 2 naming a missing vocabulary tool (tests/test_merge_checks.py). None appear in the Done-when list at lines 43-54.
- **BR-4** [Important] `atlas-missing-new-surface` (kind: defect) atlas update appears missing for the scripts/ci-setup.sh CI hook and vocabulary guard
  atlas/index.md covers the compiler pin only. Add a sentence: ci-setup.sh appends the layer tool dirs to GITHUB_PATH (temporary, removed by ariadne#627), and the experiment merge-checks exit 2 when vocabulary is missing. Only instance in window; the repo has no README.
- **BR-5** [Minor] `done-when-evidence-partial` (kind: defect) Hub report (Done-when clause 5) is ticked in the Plan but has no Log evidence
  2nd in family. Rule: every Done-when clause needs a Log line naming its evidence (run ID, message, SHA) or saying explicitly that it is pending and who owns it. Instances: clause 5 hub report (no evidence); clause 4 operator label (correctly logged as pending); clause 3 now covered per BR-1.

## Round 4 — 2026-10-25T04:41:40-07:00 (claude) — passed

### Disposed

- BR-3 — addressed — Done-when lines 55-62 now carry Revision 1 (https clone, local ../ariadne never clones) and Revision 2 (ci-setup.sh GITHUB_PATH until ariadne#627; both checks exit 2, pinned by tests/test_merge_checks.py).
- BR-4 — addressed — atlas/index.md adds a "CI tool PATH hook" entry covering ci-setup.sh, its ariadne#627 removal, _vocab.bash being non-executable, and the exit-2 refusal; this matches scripts/ci-setup.sh and _vocab.bash.
- BR-5 — addressed — Log clause (5) now names hub receipts 15c3d811, 78b7ef4d, a6af3ffd, 69dc2ddf; clause (4) stays pending and owned by the operator.

## Open findings

(none — every finding has been disposed)
