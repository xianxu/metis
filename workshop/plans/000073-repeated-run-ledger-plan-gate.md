---
reviewer_runs:
    - version: 1
      intent:
        id: 59ba244d-5579-4bcc-b049-20b3813faa36
        repository: github.com/xianxu/metis
        issue_id: "000073"
        recipe: estimate-quality
        round: 1
        attempt: 1
        agent: claude
        root: /Users/xianxu/workspace/worktree/metis-slot1/metis
        reviewed_head: 943023af55d5ea8e1cd917200d8b11e2dd1ad219
        slot: metis:1
        requested_at: "2026-10-24T02:59:56.196905Z"
      phase: finished
      pid: 39232
      started_at: "2026-10-24T02:59:56.295367Z"
      finished_at: "2026-10-24T03:00:34.670919Z"
      session_id: c83eb596-38c6-4300-b49e-e88052903fdf
      outcome: returned
gate: plan-quality
issue: 73
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 8eb82a56-98d4-4afd-984e-af96895a4923
            repository: github.com/xianxu/metis
            issue_id: "000073"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/metis-slot1/metis
            reviewed_head: 333f973638e21e61ebd1383c0d0e579e7e7d14fb
            slot: metis:1
            requested_at: "2026-10-24T02:56:26.868391Z"
          phase: finished
          pid: 1802
          started_at: "2026-10-24T02:56:27.037038Z"
          finished_at: "2026-10-24T02:57:55.834389Z"
          session_id: bc552134-8c11-4750-b10d-44c99bca3136
          outcome: returned
      "n": 1
      timestamp: "2026-10-23T19:57:56-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: PQ-1
          severity: Important
          title: 'AggregateView: a plain-run pass-through row and the repeat-group aggregate share (point_addr, fp) and out.Append silently drops one'
          detail: Fold==nil rows pass through via out.Append (pkg/ledger/ledger.go:267-270). Plain and repeat runs of one experiment share point_addr and fp. Define the aggregate row's key and Repeat value, say whether a plain row joins the repeat family, and test the mix.
          family: aggregate-key-collision
          round: 1
        - kind: defect
          id: PQ-2
          severity: Important
          title: Plain-run rerun overwrites runs/addr but first-wins dedup keeps the old row; the "as today" claim is false (single runs write no rows today)
          detail: writeSweepLedger is called only from the sweep paths (cmd/metis/sweep.go:508, :693). With never-cache (#71) a rerun's record differs from its kept row, which contradicts "the row is derived only from record.json". Choose rewrite, refuse or append-sample semantics for plain rows, and test it.
          family: row-derives-from-record
          round: 1
        - kind: defect
          id: PQ-3
          severity: Important
          title: '"Final record" is undefined; record.json is written twice and the fingerprint backfill is best-effort'
          detail: writeRecordJSON (cmd/metis/run.go:294) is followed by a best-effort backfill rewrite (capture.go:391, warn-only at run.go:300-304). Name the field that marks a record final, what publish does when the backfill failed, and how a crash between the two writes avoids a later false conflict.
          family: completion-evidence-undefined
          round: 1
        - kind: defect
          id: PQ-4
          severity: Minor
          title: writeFileAtomic should consolidate pkg/cas/fs.go writeAtomic as well as caching.go writeEntry, not add a third copy
          family: reuse-existing-helper
          round: 1
        - kind: defect
          id: PQ-5
          severity: Minor
          title: '"Known-column parsing stays strict" is false: Decode silently ignores fold, outer_fold and metric values it cannot parse (ledger.go:221-235)'
          family: unbacked-existing-behavior-claim
          round: 1
        - kind: defect
          id: PQ-6
          severity: Minor
          title: Single-run row publication must live outside runResolvedExperiment, which every sweep fold shares, or sweep folds publish stray rows
          family: shared-runner-side-effect
          round: 1
        - kind: defect
          id: PQ-7
          severity: Minor
          title: Append returning an error affects Decode, which builds through Append; say whether a conflicting CSV makes ledger show fail; also state the --run plus --repeat precedence
          family: unstated-api-contract-change
          round: 1
        - kind: defect
          id: PQ-8
          severity: Minor
          title: Tasks 2, 3 and 6 enumerate test cases in prose; compress to one adversarial strategy line per risky function (Decode/Encode fuzz round-trip, concurrent publishRows)
          family: test-prose-enumeration
          round: 1
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            id: de56ef54-55fb-4cb6-a712-8bf26edf4226
            repository: github.com/xianxu/metis
            issue_id: "000073"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/metis-slot1/metis
            reviewed_head: 943023af55d5ea8e1cd917200d8b11e2dd1ad219
            slot: metis:1
            requested_at: "2026-10-24T02:58:38.989274Z"
          phase: finished
          pid: 85010
          started_at: "2026-10-24T02:58:39.147251Z"
          finished_at: "2026-10-24T02:59:06.709576Z"
          session_id: f70666f9-b7e4-41fb-9d57-e1fcee30513c
          outcome: returned
      "n": 2
      timestamp: "2026-10-23T19:59:06-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Aggregate rows carry the reserved Repeat="*", so their key differs from the plain row; plain rows pass through and never join the repeat group; a mixed test is named.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Plain rows are replaced on rerun with the same (addr, fp), and a test is named; the sweep row kinds are distinguishable (Fold always set at cmd/metis/ledger.go:31).
          round: 2
        - id: PQ-3
          disposition: addressed
          note: The backfill runs in memory before a single atomic record.json write; a failed backfill publishes the row with an empty fp, which a re-publish derives identically.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: One internal/fsx helper consolidates pkg/cas writeAtomic and caching.go writeEntry.
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
        - id: PQ-6
          disposition: addressed
          round: 2
        - id: PQ-7
          disposition: addressed
          round: 2
        - id: PQ-8
          disposition: addressed
          round: 2
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: 4d2bc6d5-24ac-42da-9af3-24f666deb532
            repository: github.com/xianxu/metis
            issue_id: "000073"
            recipe: plan-quality
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/metis-slot1/metis
            reviewed_head: 943023af55d5ea8e1cd917200d8b11e2dd1ad219
            slot: metis:1
            requested_at: "2026-10-24T02:59:34.440468Z"
          phase: finished
          pid: 29834
          started_at: "2026-10-24T02:59:34.558223Z"
          finished_at: "2026-10-24T02:59:54.934011Z"
          session_id: f462a39f-878a-4d7d-93f8-ff4e90d54677
          outcome: returned
      "n": 3
      timestamp: "2026-10-23T19:59:55-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Aggregate rows carry the reserved Repeat="*" marker; plain rows pass through and never join the repeat group.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: A plain row is replaced on publish with the same (addr, fp), so it mirrors the overwritten run dir.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: The backfill runs in memory, then one atomic record.json write; a record that exists means final.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: One internal/fsx helper consolidates pkg/cas writeAtomic and caching.go writeEntry.
          round: 3
        - id: PQ-5
          disposition: addressed
          note: 'The strictness claim is corrected: legacy leniency is kept, and the repeat cell is validated.'
          round: 3
        - id: PQ-6
          disposition: addressed
          note: Publication sits in runExperiment's non-shape branch, after runResolvedExperiment returns.
          round: 3
        - id: PQ-7
          disposition: addressed
          note: Decode failing on conflicting repeat rows is intended and stated; --run with --repeat refuses.
          round: 3
        - id: PQ-8
          disposition: addressed
          note: Replaced by one strategy line per risky function.
          round: 3
      blocked: false
content_hash: 2853a5506316fd33945c8b7d2985036ffbe15c3ee60c812b0764e73937dd27f1
---

# Gate ledger — metis#73 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T19:57:56-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `aggregate-key-collision` (kind: defect) AggregateView: a plain-run pass-through row and the repeat-group aggregate share (point_addr, fp) and out.Append silently drops one
  Fold==nil rows pass through via out.Append (pkg/ledger/ledger.go:267-270). Plain and repeat runs of one experiment share point_addr and fp. Define the aggregate row's key and Repeat value, say whether a plain row joins the repeat family, and test the mix.
- **PQ-2** [Important] `row-derives-from-record` (kind: defect) Plain-run rerun overwrites runs/addr but first-wins dedup keeps the old row; the "as today" claim is false (single runs write no rows today)
  writeSweepLedger is called only from the sweep paths (cmd/metis/sweep.go:508, :693). With never-cache (#71) a rerun's record differs from its kept row, which contradicts "the row is derived only from record.json". Choose rewrite, refuse or append-sample semantics for plain rows, and test it.
- **PQ-3** [Important] `completion-evidence-undefined` (kind: defect) "Final record" is undefined; record.json is written twice and the fingerprint backfill is best-effort
  writeRecordJSON (cmd/metis/run.go:294) is followed by a best-effort backfill rewrite (capture.go:391, warn-only at run.go:300-304). Name the field that marks a record final, what publish does when the backfill failed, and how a crash between the two writes avoids a later false conflict.
- **PQ-4** [Minor] `reuse-existing-helper` (kind: defect) writeFileAtomic should consolidate pkg/cas/fs.go writeAtomic as well as caching.go writeEntry, not add a third copy
- **PQ-5** [Minor] `unbacked-existing-behavior-claim` (kind: defect) "Known-column parsing stays strict" is false: Decode silently ignores fold, outer_fold and metric values it cannot parse (ledger.go:221-235)
- **PQ-6** [Minor] `shared-runner-side-effect` (kind: defect) Single-run row publication must live outside runResolvedExperiment, which every sweep fold shares, or sweep folds publish stray rows
- **PQ-7** [Minor] `unstated-api-contract-change` (kind: defect) Append returning an error affects Decode, which builds through Append; say whether a conflicting CSV makes ledger show fail; also state the --run plus --repeat precedence
- **PQ-8** [Minor] `test-prose-enumeration` (kind: defect) Tasks 2, 3 and 6 enumerate test cases in prose; compress to one adversarial strategy line per risky function (Decode/Encode fuzz round-trip, concurrent publishRows)

## Round 2 — 2026-10-23T19:59:06-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Aggregate rows carry the reserved Repeat="*", so their key differs from the plain row; plain rows pass through and never join the repeat group; a mixed test is named.
- PQ-2 — addressed — Plain rows are replaced on rerun with the same (addr, fp), and a test is named; the sweep row kinds are distinguishable (Fold always set at cmd/metis/ledger.go:31).
- PQ-3 — addressed — The backfill runs in memory before a single atomic record.json write; a failed backfill publishes the row with an empty fp, which a re-publish derives identically.
- PQ-4 — addressed — One internal/fsx helper consolidates pkg/cas writeAtomic and caching.go writeEntry.
- PQ-5 — addressed
- PQ-6 — addressed
- PQ-7 — addressed
- PQ-8 — addressed

## Round 3 — 2026-10-23T19:59:55-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Aggregate rows carry the reserved Repeat="*" marker; plain rows pass through and never join the repeat group.
- PQ-2 — addressed — A plain row is replaced on publish with the same (addr, fp), so it mirrors the overwritten run dir.
- PQ-3 — addressed — The backfill runs in memory, then one atomic record.json write; a record that exists means final.
- PQ-4 — addressed — One internal/fsx helper consolidates pkg/cas writeAtomic and caching.go writeEntry.
- PQ-5 — addressed — The strictness claim is corrected: legacy leniency is kept, and the repeat cell is validated.
- PQ-6 — addressed — Publication sits in runExperiment's non-shape branch, after runResolvedExperiment returns.
- PQ-7 — addressed — Decode failing on conflicting repeat rows is intended and stated; --run with --repeat refuses.
- PQ-8 — addressed — Replaced by one strategy line per risky function.

## Open findings

(none — every finding has been disposed)
