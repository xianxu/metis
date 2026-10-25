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

## Open findings

- **BR-1** [Minor] `done-when-evidence-partial` (kind: defect) "Pin survives compile" is shown structurally, not by a full weave compile run
- **BR-2** [Important] `scope-changed-after-close` (kind: defect) Scope changed after close: run a full sdlc close
