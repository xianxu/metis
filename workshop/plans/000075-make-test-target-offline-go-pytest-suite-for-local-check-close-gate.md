---
gate: boundary-review
issue: 75
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ec272bcc-efce-4e42-9d85-14471ca6d80d
            repository: github.com/xianxu/metis
            issue_id: "000075"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/5208294fb1210307
            reviewed_head: 180ac8416f3c23900ce2b7a4cb37e1e476d27796
            slot: metis:1
            requested_at: "2026-10-24T02:03:05.167497Z"
          phase: finished
          pid: 45069
          started_at: "2026-10-24T02:03:05.262731Z"
          finished_at: "2026-10-24T02:12:11.40416Z"
          session_id: fb2f68a4-cbe0-413f-869a-dc0c0e7bae4d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/metis
        code_head: 180ac8416f3c23900ce2b7a4cb37e1e476d27796
        tracker_head: 6c1b0cf72742415c7481d9dcab3f90f11a3ee2a0
        branch_point: e8cafbf867e30a740218674b2808b1696404111c
        since: "2026-10-23T17:24:43-07:00"
        until: "2026-10-23T19:02:48-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T17:24:43-07:00"
              end: "2026-10-23T19:02:48-07:00"
              began: "2026-10-23T17:24:43-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/metis-slot1/metis
        sources:
            - harness: claude
              id: 621e2c42-a080-4dc1-9165-83040e197e38
              slot: metis:1
              cwd: /Users/xianxu/workspace/worktree/metis-slot1/metis
              digest: fd0505c29da8f5c530e4b4b84d96dea54a14503b53c4f22af2b2e224cdce0bdb
              started: "2026-10-24T00:13:14.303Z"
              spawned: false
              bytes: 2026902
              windows:
                - start: "2026-10-23T17:24:43-07:00"
                  end: "2026-10-23T19:02:48-07:00"
        queue_waits:
            - attempt_id: c828a6e2d4dbe6fbe35846394e7ff114
              slot: metis:1
              record_digest: 8dd4ac2089851407e87d3fd1811d44081c2307e9f580ee50c99cbf28a78d3c65
              start: "2026-10-24T00:29:21.133819Z"
              end: "2026-10-24T00:55:33.392132Z"
              queue_seconds: 1572.258313
              raw_seconds: 1572.258313
              exact: true
            - attempt_id: 4ac5ab566e10a35b411a14862397bf40
              slot: metis:1
              record_digest: eac58a1eb473112ae6001fc649757f1c1fc75ab27188f151a79e4b244f4ece70
              start: "2026-10-24T00:59:23.806114Z"
              end: "2026-10-24T01:59:49.154935Z"
              queue_seconds: 3625.348821
              raw_seconds: 3625.348821
              exact: true
      "n": 1
      timestamp: "2026-10-23T19:12:31-07:00"
      dispatched: "2026-10-23T19:03:05-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: make test writes its temp go.mod to one fixed path in a shared temp dir, so concurrent runs can collide and nothing deletes it
          detail: 'Makefile:35 sets TEST_MODFILE to $(TMPDIR or /tmp)/metis-test.mod. Two local-check runs on the same host, possibly with different SDLC_DEP_SUBSTRATE values, can overwrite each other''s file mid-run and build against the wrong ariadne, and the file is never removed (ARCH-ORDER, ARCH-FUNERAL). Fix: put it somewhere private to the checkout, e.g. $(shell git rev-parse --absolute-git-dir)/metis-test.mod, or use mktemp -d plus rm inside the recipe.'
          family: shared-temp-artifact-unowned
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: The -modfile path on the go test line is unquoted, unlike the cp and go mod edit lines
          detail: Makefile:46 passes -modfile=$(TEST_MODFILE) without quotes, so a TMPDIR containing a space breaks only this step.
          family: inconsistent-shell-quoting
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: The Spec and Plan don't mention the SDLC_DEP_SUBSTRATE -modfile fallback added in e950317
          detail: The Log explains it, but the plan should get a Revisions entry so the Spec and Plan describe what make test actually does in local-check checkouts.
          family: plan-scope-drift
          round: 1
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 680c10c7-7577-4bd6-8be9-d4629ae78e3c
            repository: github.com/xianxu/metis
            issue_id: "000075"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/ca422ea244c2d5ed
            reviewed_head: aad6f5c5d310ca2dd05dc246bd7a26359d039ee9
            slot: metis:1
            requested_at: "2026-10-24T02:29:37.905438Z"
          phase: finished
          pid: 36522
          started_at: "2026-10-24T02:29:38.015025Z"
          finished_at: "2026-10-24T02:32:32.564623Z"
          session_id: 8a3bdfdd-0458-47aa-a2bc-07a545c08d98
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/metis
        code_head: aad6f5c5d310ca2dd05dc246bd7a26359d039ee9
        tracker_head: 6c1b0cf72742415c7481d9dcab3f90f11a3ee2a0
        branch_point: e8cafbf867e30a740218674b2808b1696404111c
        since: "2026-10-23T17:24:43-07:00"
        until: "2026-10-23T19:15:06-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T17:24:43-07:00"
              end: "2026-10-23T19:15:06-07:00"
              began: "2026-10-23T17:24:43-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/metis-slot1/metis
        sources:
            - harness: claude
              id: 621e2c42-a080-4dc1-9165-83040e197e38
              slot: metis:1
              cwd: /Users/xianxu/workspace/worktree/metis-slot1/metis
              digest: 1b472e04ba7fc29e0193c6a47b15807639bf03c267c6e732dfae879155776992
              started: "2026-10-24T00:13:14.303Z"
              spawned: false
              bytes: 2181417
              windows:
                - start: "2026-10-23T17:24:43-07:00"
                  end: "2026-10-23T19:15:06-07:00"
        queue_waits:
            - attempt_id: c828a6e2d4dbe6fbe35846394e7ff114
              slot: metis:1
              record_digest: 8dd4ac2089851407e87d3fd1811d44081c2307e9f580ee50c99cbf28a78d3c65
              start: "2026-10-24T00:29:21.133819Z"
              end: "2026-10-24T00:55:33.392132Z"
              queue_seconds: 1572.258313
              raw_seconds: 1572.258313
              exact: true
            - attempt_id: 4ac5ab566e10a35b411a14862397bf40
              slot: metis:1
              record_digest: eac58a1eb473112ae6001fc649757f1c1fc75ab27188f151a79e4b244f4ece70
              start: "2026-10-24T00:59:23.806114Z"
              end: "2026-10-24T01:59:49.154935Z"
              queue_seconds: 3625.348821
              raw_seconds: 3625.348821
              exact: true
        cuts:
            - dispatched: "2026-10-23T19:03:05-07:00"
              finalized: "2026-10-23T19:12:31-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-23T19:32:35-07:00"
      dispatched: "2026-10-23T19:29:37-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Makefile:37 uses mktemp -d plus an EXIT trap rm. A run against the pinned substrate left no metis-test.* dir and resolved ariadne through the substrate.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Makefile:44 passes the flag as ${modflag:+"$modflag"}, and modflag is assigned quoted, so a TMPDIR with spaces is now safe.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The issue's Revisions entry (2026-10-23) adds the SDLC_DEP_SUBSTRATE -modfile delta to the Spec and Plan item 1, including the per-run private dir.
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Minor
          title: Done-when 3 (sdlc-local/suite recorded on the pushed head) is ticked, but the Log has no successful local-check record after the fix
          detail: The only local-check run in the Log is the failed one on 8e8b3c96. The later verification was done in a plain clone. Add a Log line citing the head where sdlc-local/suite was recorded.
          family: done-when-claim-unevidenced
          round: 2
      recipe: milestone-review
      isolation: checkout
      reviewed: aad6f5c5d310ca2dd05dc246bd7a26359d039ee9
      blocked: false
---

# Gate ledger — metis#75 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T19:12:31-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `shared-temp-artifact-unowned` (kind: defect) make test writes its temp go.mod to one fixed path in a shared temp dir, so concurrent runs can collide and nothing deletes it
  Makefile:35 sets TEST_MODFILE to $(TMPDIR or /tmp)/metis-test.mod. Two local-check runs on the same host, possibly with different SDLC_DEP_SUBSTRATE values, can overwrite each other's file mid-run and build against the wrong ariadne, and the file is never removed (ARCH-ORDER, ARCH-FUNERAL). Fix: put it somewhere private to the checkout, e.g. $(shell git rev-parse --absolute-git-dir)/metis-test.mod, or use mktemp -d plus rm inside the recipe.
- **BR-2** [Minor] `inconsistent-shell-quoting` (kind: defect) The -modfile path on the go test line is unquoted, unlike the cp and go mod edit lines
  Makefile:46 passes -modfile=$(TEST_MODFILE) without quotes, so a TMPDIR containing a space breaks only this step.
- **BR-3** [Minor] `plan-scope-drift` (kind: defect) The Spec and Plan don't mention the SDLC_DEP_SUBSTRATE -modfile fallback added in e950317
  The Log explains it, but the plan should get a Revisions entry so the Spec and Plan describe what make test actually does in local-check checkouts.

## Round 2 — 2026-10-23T19:32:35-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Makefile:37 uses mktemp -d plus an EXIT trap rm. A run against the pinned substrate left no metis-test.* dir and resolved ariadne through the substrate.
- BR-2 — addressed — Makefile:44 passes the flag as ${modflag:+"$modflag"}, and modflag is assigned quoted, so a TMPDIR with spaces is now safe.
- BR-3 — addressed — The issue's Revisions entry (2026-10-23) adds the SDLC_DEP_SUBSTRATE -modfile delta to the Spec and Plan item 1, including the per-run private dir.

### Raised

- **BR-4** [Minor] `done-when-claim-unevidenced` (kind: defect) Done-when 3 (sdlc-local/suite recorded on the pushed head) is ticked, but the Log has no successful local-check record after the fix
  The only local-check run in the Log is the failed one on 8e8b3c96. The later verification was done in a plain clone. Add a Log line citing the head where sdlc-local/suite was recorded.

## Open findings

- **BR-4** [Minor] `done-when-claim-unevidenced` (kind: defect) Done-when 3 (sdlc-local/suite recorded on the pushed head) is ticked, but the Log has no successful local-check record after the fix
