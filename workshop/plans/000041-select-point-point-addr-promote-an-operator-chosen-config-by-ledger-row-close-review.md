# Boundary Review — metis#41 (whole-issue close)

| field | value |
|-------|-------|
| issue | 41 — select --point <point_addr> — promote an operator-chosen config by ledger row |
| repo | metis |
| issue file | workshop/issues/000041-select-point-point-addr-promote-an-operator-chosen-config-by-ledger-row.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2b1d7039e8f0dc6ef65cb2d9966fd2d90adc56d1..HEAD |
| command | sdlc close --issue 41 |
| reviewer | claude |
| timestamp | 2026-07-14T17:21:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The boundary delivers what metis#41 promised: `select --point` resolves a git-style `point_addr` prefix to exactly one config, prints its board line, and `--promote` ships it through the same `promotedExperiment`/`runResolvedExperiment` path `--best` uses, under a `point-` run-id prefix for operator-chosen provenance. All five Spec behaviors (prefix match, ambiguity, no-match, cohort guard, mutual exclusion) plus the promote reconstruction are tested; `go test ./cmd/metis ./pkg/ledger` and `go vet` are green. What keeps it off SHIP: the new path re-implements the pooled mean±SE statistic that `pkg/ledger` already owns, and in doing so silently drops the failed-fold discipline every other select path enforces — a failed fold's metrics get pooled into the board line and a failed config promotes without a word.

### 1. Strengths

- **The dispatch seam is exactly where the plan said** — `--point` branches after the cohort guard (`select_cmd.go:118-123`), so the multi-cohort refusal and `--fingerprint` pinning apply unchanged; `TestSelectPoint_WrongCohortErrors` pins this with a genuinely multi-cohort fixture.
- **`resolvePointRows` honors the "one config's rows" contract** the plan designed for `--where` reuse (`select_cmd.go:378-421`), including the non-obvious expansion from prefix-hit rows to *all* rows of the resolved config so the board line pools every fold — with the reasoning stated in a comment.
- **The promote test asserts the right invariant** — `TestSelectPoint_PromoteReconstructsRowConfig` checks the shipped run's `record.json` carries the *row's* config (`max_depth: 8`), not just that a run happened; that's the exact bug class this feature could ship (reconstructing the rule-based pick instead of the operator's).
- **Provenance separation is real**: `point-{family}-{hash}` vs `best-…` at `select_cmd.go:524`, documented in `atlas/experiment.md` in the same window.
- **The ambiguity error is operator-grade** — it lists each candidate addr *with its free params*, so the disambiguation is one glance, not another `ledger show`.

### 2. Critical findings

None.

### 3. Important findings

1. **ARCH-DRY — `runPointSelect` re-implements the pooled mean/SE that `pkg/ledger` already single-sources** (`select_cmd.go:485-498`). `meanSE` (`pkg/ledger/ledger.go:299`) is byte-for-byte the same formula (sample-SD with n−1, ÷√n), and its own comment already apologizes for being a second copy of `sampler.Aggregate` — this diff adds a third. `AggregateView` over the resolved rows does the whole job: inner rows (Fold set) reduce to one aggregate row carrying `metric`, `metric+".se"`, `metric+".n"`, while outer rows (Fold nil) pass through untouched — precisely the board line's inputs. Fix sketch: replace the inline loops with `agg := ledger.AggregateView(ledgerOf(rows), metric)` and read the aggregate row + pass-through outer rows. This also fixes finding 2 for free.

2. **Failed fold rows are silently pooled and silently promoted** (`select_cmd.go:475-483`, correctness-adjacent). `rowsFromRecords` writes `Metrics` regardless of `Status` (`cmd/metis/ledger.go:34-35`), so a fold that emitted `train.fold_score` before a later step failed contributes to the `--point` pooled mean with no indication. Every sibling path excludes failed rows — `configStatsFromLedger` skips `Status == "failed"`, `AggregateView` marks the group failed, `Best`/`TopN` skip — so `--point` can promote a config `--best` would refuse to even score, without saying so. An operator override promoting a partially-failed config may be legitimate, but per this repo's loud-error discipline it must be visible. Fix sketch: via the `AggregateView` reuse above, check the aggregate row's `Status` and print a `warning: config has failed fold rows (excluded from the pooled estimate)` line (or refuse without an acknowledging flag). Add a test with one `Status: "failed"` fold row carrying a metric.

### 4. Minor findings

- `sample` in `resolvePointRows` is a `map[string]string` keyed by `fmt.Sprint(i)` (`select_cmd.go:390,401,407`) — an obfuscated `[]string`; use a slice parallel to `configs`.
- The `freeParamMapsEqual` comment claims "the same tolerance freeParamsEqual applies" (`select_cmd.go:438`) — not exact: `freeParamsEqual` is JSON equality (string `"1"` ≠ number `1`), `fmt.Sprint` equates them. Harmless here (row-vs-row from one decoded ledger) but the comment overstates.
- `shape.Expand` + the family scan (`select_cmd.go:464-472`) run even without `--promote`, though `fam` is only used in the run id — wasted work on the inspect path. The swallowed Expand error is acceptable (consistent with `familyEstimateFromLedger`, and `promotedExperiment` re-raises it loudly at promote time).
- ~15 lines of promote glue (empty-ship check, `now` default, `shapeBlobHash`, `pointAddressOf`, `runOpts`, `runResolvedExperiment`) are near-copied from `promoteSelected` (`select_cmd.go:332-359` vs `508-530`); a shared `shipConfig(o, sh, famTag, idPrefix, config)` would let the plan's "no new promote machinery" claim be fully literal.

### 5. Test coverage notes

The six new tests cover the Spec's stated cases well (resolve/inspect, ambiguity, no-match, wrong-cohort, flag conflict, promote-reconstructs-row) and the promote test pins the record contents, not just the run id. Gaps: (a) no test for failed fold rows in the point path — the Important finding 2 bug class; (b) SE on the board line is never asserted (only the 0.86 mean), so the SE formula is effectively untested on this path — both gaps close naturally with the `AggregateView` consolidation. The plan's Core concepts table is accurate: `resolvePointRows` exists at the stated path, is genuinely pure (no mocks needed — tests reach it through the existing file-fixture shell, the established pattern), and the `cmdSelect`/`runSelect` modification is where claimed.

### 6. Architectural notes

- **ARCH-DRY: flag** — findings 3.1 and 4(glue) above; the statistics duplication is the one that bites, since `meanSE`'s comment shows this formula already has a history of copies drifting apart.
- **ARCH-PURE: pass, with a note** — `resolvePointRows` is pure over decoded rows as planned; but `runPointSelect` computes statistics inline in a printing function. The `AggregateView` reuse moves that computation back into the pure layer where it lives.
- **ARCH-PURPOSE: pass** — the purpose ("publish any ledger row") is delivered end-to-end: resolve → inspect → promote → submittable run id. The row-id-only v1 scope is the operator's explicit decision recorded in the Spec, with `--where` designed as the same surface later (the resolve contract supports it); this is legitimate scoping, not under-delivery. The atlas consumer was updated in-window; there is no README.md in metis, so that doc gate is N/A. The kbench RUNBOOK update claimed in the Log lives in the peer repo and is outside this window — not verifiable here, noted only.

### 7. Plan revision recommendations

None — the plan matches the shipped code (tasks ticked, entities where stated). If the `AggregateView` consolidation is applied, an optional one-line `## Revisions` note ("board-line stats via `ledger.AggregateView` rather than inline pooling; failed folds surfaced") would keep the plan's Architecture paragraph exact, but nothing in the plan currently contradicts the code.
