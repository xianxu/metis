# Boundary Review — metis#66 (milestone M2)

| field | value |
|-------|-------|
| issue | 66 — adaptive outer-fold scheduling + --auto-stop (incumbent-referenced early stop of losing configs) |
| repo | metis |
| issue file | workshop/issues/000066-adaptive-outer-fold-auto-stop.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | feae0e77733970491329088a3b8f12b36921261b..HEAD |
| command | sdlc milestone-close --issue 66 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-19T09:36:04-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I now have a complete picture. Let me verify my key finding's mechanism once more by confirming `ledger.Best` (what the plan cited) vs the hand-rolled loop, and that the incumbent bias claim holds.

The evidence is conclusive. Here's my review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M2 `--auto-stop` implementation is clean, well-separated (pure `shouldStop`/`tCrit` in `autostop.go`, thin IO seam in `sweep.go`), the predictive-bound derivation is mathematically sound and documented as the Spec required, and the ledger `stopped` column is a faithful ragged-column extension. **What blocks an unqualified SHIP: the incumbent read (`readIncumbent`) uses `ledger.AggregateView` (a per-config reduction) instead of the dedicated per-family reducer `FamilyEstimate` that `metis select` uses — which is exactly the "metis select's best-per-family" incumbent the Spec names.** For any shape with intra-family hyperparameter grids (the primary nested-CV use case), this sets an *optimistically inflated* bar and can auto-stop a family that would beat the honest incumbent — a violation of the feature's headline "never stop a would-be winner" guarantee. The e2e passes only because it uses config-less families (`{rf: {}, logreg: {}}`) where the two reducers coincide, so the gap is invisible to the suite.

### 1. Strengths
- **Pure rule, cleanly injected (ARCH-PURE done right).** `shouldStop`/`tCrit` (`autostop.go:67`,`:96`) are deterministic, IO-free, and unit-tested directly (`autostop_test.go`); `evaluateAutoStop`/`activeConfigs` are the thin seam that consults them. Matches the plan's ARCH-PURE note exactly.
- **The predictive-variance derivation is correct and documented.** `SEpred² = s²·r/k²·(1+r/n)` (`autostop.go:84`) correctly composes the r-future spread with the mean-estimation error (`Var(Σfuture) = σ²·r·(1+r/n)`), and the t_{n-1} one-sided bound with wide small-df values genuinely protects a green estimate. This satisfies the Spec's "documented, not silent" requirement.
- **Budget reclamation is real, not cosmetic.** `activeConfigs` (`sweep.go:110`) drops stopped families' configs from the *sealed inner sweep* — the actual cost — not just the held-out score leaf, honoring the plan's ARCH-PURPOSE note (`plan:38`).
- **Ledger back-compat preserved.** The `stopped` column is ragged/`omitempty` (`ledger.go:130`,`sweep.go:69`); non-auto-stop ledgers stay byte-identical, and the e2e exercises the full Encode→Decode round-trip via `loadLedgerOrFatal`.
- **Sequential-outer gating is race-free by construction** (`outerParallel=false`, `sweep.go:599`); the per-fold decision cleanly gates the next fold's config set.

### 2. Critical findings

**C1 — `readIncumbent` uses `AggregateView` (per-config), not the per-family `FamilyEstimate` the Spec names; incumbent bar is biased optimistic → can stop would-be winners.** `cmd/metis/autostop.go:36`.
- The Spec (issue `:53–54`) defines the incumbent as "the promoted run / `metis select`'s best-per-family." `metis select` computes that via `familyEstimateFromLedger` → `FamilyEstimate` (`select_cmd.go:125,203`), whose docstring (`family.go:13–18`) explicitly warns that `AggregateView` is the *wrong* tool: a family's winning config differs across outer folds, so `AggregateView` (grouping by exact free-params) splits one family into several per-config rows and never pools the per-family outer mean.
- `readIncumbent` iterates `AggregateView(led, metric).Rows` and takes the single best `r.Metrics[metric]`. When a prior run's winning config varied across folds, this is `max` over per-config *subset* means, which is ≥ the honest pooled per-family mean (for maximize; symmetric ≤ for minimize). The bar is inflated above what you'd actually ship.
- **Failure scenario:** prior run, family `rf` — config A won folds {0,1} scoring {0.85, 0.87}, config B won folds {2,3} scoring {0.80, 0.82}. Honest `FamilyEstimate(rf)` = 0.835 (what `metis select` reports and ships). `readIncumbent` returns 0.86. A *new* run's family whose honest estimate is ~0.84 — a genuine would-be winner over the honest incumbent — is auto-stopped because it can't reach 0.86. The headline "losers only" invariant is broken, silently (the printed incumbent looks plausible).
- **Fix:** replace the `AggregateView` loop with the per-family estimate + best-by-direction, reusing the existing helper — e.g. `est := familyEstimateFromLedger(ss.sh, led, metric); pick the best est[fam] by direction`. This requires threading `ss.sh` into `readIncumbent` (the caller `runNestedCV` has it). Note the current code already correctly *avoided* the plan's literal `ledger.Best` (which would wrongly include optimistic inner rows) — this fix completes that instinct by also pooling per-family. Also ARCH-DRY: `FamilyEstimate` exists for precisely this reduction. (The plan's M2.1 at `plan:119–122` authorized `AggregateView→Best`, so it drifted from the Spec too — see §7.)

### 3. Important findings
*(none beyond C1 — the divergence is the one substantive gap)*

### 4. Minor findings
- **Decode error is conflated with "no prior run."** `readIncumbent` (`autostop.go:32–35`) returns `present=false` on any `loadLedger` error, so a *corrupt/undecodable* ledger prints the "no incumbent (no prior run)" message (`sweep.go:604`) and silently degrades to a full sweep. Consider distinguishing a real load error (warn) from a legitimately-absent ledger.
- **All-losers edge marks rows `stopped: auto` that in fact ran full k.** If every family is stopped, `activeConfigs` keeps them all (`sweep.go:125–127`) so they run to full k, yet `markStoppedRows` still tags them `stopped: auto` — the marker's documented meaning ("remaining folds were cut") is then slightly inaccurate. Rare, cosmetic; a reader mining the ledger could misread it.
- **`markStoppedRows` mutates `ss.man.Points` under `stopMu`** (`sweep.go:169–177`) whereas that slice is elsewhere guarded by `manMu`. Harmless (called single-threaded at finalize) but a lock-domain inconsistency worth a comment.
- **`--auto-stop` on a flat (1-config) shape is a silent no-op** — the flag never reaches `runNestedCV`. The `--live` help notes flat is unaffected; `--auto-stop`'s help doesn't. Low priority.

### 5. Test coverage notes
- **The e2e cannot catch C1.** `autostop_e2e_test.go` uses `{rf: {}, logreg: {}}` — one config per family — so `AggregateView` ≡ `FamilyEstimate` and the incumbent is additionally *seeded* as a single pass-through row (0.80), never exercising `readIncumbent`'s reduction over multi-fold outer rows. **Add a test where a prior-run ledger has ≥2 outer folds per family with a *varying winning config* within a family, and assert the incumbent equals `FamilyEstimate`'s best-by-direction** (not the max per-config subset mean). This is the test that would have surfaced C1.
- `shouldStop`/`tCrit` unit coverage is strong (loser/winner/borderline/both directions/monotonicity/n=1/full-k/t-table) — good pinning of the pure rule.
- The `stopped` column round-trip is covered end-to-end by the e2e (write→read `r.Stopped`), so a dedicated `ledger` unit test is optional (nice-to-have, mirroring the existing `level`/`outer_fold` round-trip test at `ledger_test.go:344`).

### 6. Architectural notes for upcoming work
- **ARCH-DRY:** flag on C1 — `readIncumbent` reimplements a subtly-different (and biased) version of `FamilyEstimate`. Consolidate onto `familyEstimateFromLedger`. Otherwise DRY is clean (`persistNestedAndReport` single-sources full + stopped tails; `markStoppedRows` single-sources the marker).
- **ARCH-PURE:** pass — pure rule + thin seam, exactly as the plan promised.
- **ARCH-PURPOSE:** partial — the feature reclaims real budget and stops losers, but the incumbent it references is *not* the one the Spec committed to ("metis select's best-per-family"). Fulfilling the stated purpose means the incumbent must be the honest shippable family estimate, so C1 is a purpose gap, not just a style nit. Downstream (metis#54 inner-racing composition) will consume this incumbent notion — stabilize it on `FamilyEstimate` now.

### 7. Plan revision recommendations
Add a `## Revisions` entry to `workshop/plans/000066-adaptive-outer-fold-auto-stop-plan.md`:
- **M2.1 incumbent source correction.** M2.1 (`plan:119–122`) specifies `ledger.AggregateView → ledger.Best`, which contradicts (a) the issue Spec's "metis select's best-per-family" (`issue:53–54`), (b) this plan's own ARCH-DRY note to "reuse `FamilyOf`/`FamilySelect`" (`plan:37`), and (c) metis's honesty principle (`family.go:13–18`, why `FamilyEstimate` exists). Revise M2.1 to read: incumbent = best-by-direction over `familyEstimateFromLedger` (the per-family honest outer estimate, the same reduction `metis select` ships), and record that the implementation was updated to match once C1 is fixed.

---

## Re-review — 2026-07-19T09:39:45-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 66 — adaptive outer-fold scheduling + --auto-stop (incumbent-referenced early stop of losing configs) |
| repo | metis |
| issue file | workshop/issues/000066-adaptive-outer-fold-auto-stop.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | feae0e77733970491329088a3b8f12b36921261b..HEAD |
| command | sdlc milestone-close --issue 66 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-19T09:39:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have enough to complete the review. Note: the shell tool is unavailable in this environment (harness-level EPERM on session setup), so I could not independently run the test suite — my review is static, cross-checked against the code and the codebase's own documented invariants.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** M2 delivers the incumbent-referenced loser-stop: a well-documented pure predictive rule (`shouldStop`/`tCrit`), sequential-outer scheduling that cleanly gates each fold's config set, real budget reclaim by dropping stopped families' inner sweeps (`activeConfigs`), a ragged `stopped: auto` ledger marker, and a loser-stopped/winner-full e2e. The pure rule is genuinely pure and directly unit-tested (ARCH-PURE ✓), and the loud-announcement / never-empty / retroactive-marking edges are handled. The one thing that blocks a clean SHIP is that **`readIncumbent` computes the incumbent with the wrong reducer** — it uses `ledger.AggregateView` (groups by exact free-params) where the codebase's own `family.go` explicitly documents that AggregateView is *wrong* for a per-family outer reduce and that `FamilyEstimate` is the correct one. This drifts from the Spec's stated incumbent ("`metis select`'s best-per-family") and biases toward over-stopping. It's reversible (re-run) and invisible to the single-config-per-family e2e, so it's Important rather than blocking — but it's the #1 fix.

### 1. Strengths
- **`shouldStop`/`tCrit` are properly pure and directly tested** (`autostop.go:67`, `autostop_test.go`): the loser-stop / winner-never-truncated / borderline-spared / both-directions / monotone-in-incumbent / t-table cases pin real statistical behavior, not the implementation. This is exactly the ARCH-PURE shape the plan promised.
- **Real budget reclaim, not a cosmetic stop** (`sweep.go:110` `activeConfigs`, `runOuterFold:741`): a stopped family's *inner sweep* is dropped on later folds — the actual cost — matching the plan's ARCH-PURPOSE note that skipping only the score leaf would reclaim ~nothing.
- **Sequential-outer gating is correct and race-free** (`sweep.go:597-599`, `evaluateAutoStop:820`): `outerParallel=false` while inner sweeper/resample stay parallel; the decision after fold i cleanly gates fold i+1's `activeConfigs`. `stopMu` guards the shared maps.
- **Ragged ledger column follows the established pattern faithfully** (`ledger.go:106-135,163-166,219`): `hasStopped` mirrors `hasFold`/`hasLevel`/`hasOuterFold` so a v1 ledger stays byte-identical (ARCH-DRY), and `AggregateView` passes it through as provenance (not treated as failed).
- **Never-empty and retroactive-marking edges handled** (`activeConfigs:125`, `markStoppedRows:169`): an all-losers fold keeps every family; a family stopped after fold i has folds 0..i marked at finalize.

### 2. Critical findings
None (no crash/always-wrong path).

### 3. Important findings

**I1 — `readIncumbent` uses the reducer `family.go` documents as wrong for this exact purpose; incumbent drifts from `metis select`'s best-per-family (ARCH-DRY + ARCH-PURPOSE + contract drift).** `cmd/metis/autostop.go:36`
`readIncumbent` does `ledger.AggregateView(led, metric)` over the outer rows and takes the best group mean. But `AggregateView` groups by *exact free-params*, and `family.go:13-18` spells out why that's wrong here: "a family's winning config DIFFERS across outer folds … so those rows share the family but carry DISTINCT free-params — AggregateView would put them in separate groups and never compute the per-family mean." The Spec (issue §M2) defines the incumbent as "the promoted run / `metis select`'s best-per-family," and `metis select` computes it via `familyEstimateFromLedger`→`FamilyEstimate` (`select_cmd.go:125`, `family.go`). So whenever a prior run's family won with different configs across folds (the common case under noise), `readIncumbent` splits its outer rows into per-config subgroups and takes the **max** subgroup mean — an optimistic number ≥ the honest pooled family estimate (for minimize, symmetrically a lower/harder bar). The incumbent becomes *harder to beat than the operator's actual shippable number*, so auto-stop over-stops — and can truncate a family that, by the honest per-family estimate, would be competitive with the real incumbent. That is the "never truncate a would-be winner" invariant the issue says it's most exposed on, undermined upstream of `shouldStop`.
*Failure scenario:* prior run: family rf wins fold0 with `md=4` (0.81), fold1 with `md=8` (0.79); honest rf incumbent = 0.80. `readIncumbent` returns 0.81 (the `md=4` single-fold group). A new run's genuinely-0.80 family is now judged against 0.81 and gets stopped.
*Fix sketch:* thread `ss.sh` into `readIncumbent` and reuse the select path: `familyEstimateFromLedger(sh, led, metric)` then take the best family mean by direction (or `sampler.FamilySelect`). This both matches the contract and removes the parallel reduce. The e2e can't catch this (single-config families) — add a multi-config-per-family case (see T2).

**I2 — no pkg/ledger unit test for the `stopped` ragged column** (round-trip + byte-identity), unlike every other ragged column. `pkg/ledger/ledger_test.go`
`TestEncodeDecode_LevelOuterFoldRoundTrip` (`ledger_test.go:346`) pins the level/outer_fold columns and the "present only when ≥1 row carries it" property. The new `stopped` column has no equivalent. The e2e always has a stopped row, so the byte-identity guard (`hasStopped`, `ledger.go:119`) — "a ledger with no stopped rows stays column-free" — is untested; a regression that always emitted the column would pass the whole suite. Cheap fix: extend/duplicate that test to assert `Stopped` round-trips and that a no-stopped-row ledger has no `stopped` header.

### 4. Minor findings
- **`evaluateAutoStop(k, …)` uses the estimand `k`, not `runFolds`, under `--sample out<M>`** (`sweep.go:820`, `autostop.go:69`): `shouldStop` models `k−n` remaining folds when only `runFolds−n` will actually run. Conservative (less stopping), so harmless — but imprecise. If `--auto-stop`+`--sample` isn't meant to compose, consider rejecting the combo loudly; otherwise pass `runFolds`.
- **Incumbent read has no cohort guard** (`autostop.go:36`): it takes the best outer estimate across *all* fingerprint cohorts, whereas `metis select` refuses a multi-cohort ledger without `--fingerprint` (`select_cmd.go:98`). The incumbent bar could come from a stale code version.
- **Atlas wording "best per-family OUTER aggregate mean"** (`atlas/experiment.md`, `autostop.go:26`) is inaccurate for the current code (it's per-config-group); it becomes accurate once I1 is fixed.

### 5. Test coverage notes
- `shouldStop`/`tCrit`: excellent, pure, direct.
- **`readIncumbent` and `betterMeanSE` have zero direct unit tests.** The e2e's seed row uses `Fold=nil` (a pass-through), so it never exercises `AggregateView`'s *grouping* path — precisely where I1's bug lives. Add a `readIncumbent` unit test that seeds real multi-fold outer rows, including a family whose winning config varies across folds, and asserts the incumbent equals the honest pooled per-family estimate (this test should fail today and pass after the I1 fix).
- The e2e (`autostop_e2e_test.go`) is a solid Done-when gate for the happy path (loser stopped at 2/4, winner full k, marker set, announcement printed) — I verified its numeric expectations against the rule by hand and they hold.
- I could **not** run `go test`/`go vet` (harness shell unavailable — EPERM on session setup). The Log claims the suite is green; that remains unverified in this pass.

### 6. Architectural notes for upcoming work
- **ARCH-DRY — FLAG (I1):** `readIncumbent` is a second per-family outer reduce competing with the canonical `FamilyEstimate`; consolidate onto the select path so "the incumbent" has one definition fleet-wide.
- **ARCH-PURE — PASS:** the stat rule is pure and injected; the sweep methods are a thin coordination seam on the accumulator. Good.
- **ARCH-PURPOSE — mostly PASS, tied to I1:** the diff delivers the real reclaim, losers-only, sequential gating, marker, and e2e — but "measure against `metis select`'s best-per-family" is only partially served while the incumbent is read with the wrong reducer.

### 7. Plan revision recommendations
- If I1 is fixed as recommended, no plan-code contradiction remains. If the team elects to keep the per-config-group incumbent, add a `## Revisions` entry to `workshop/plans/000066-…-plan.md` under M2.1 documenting the deliberate deviation from "`metis select`'s best-per-family" and the over-stop bias it introduces (so the plan stops claiming the select-parity semantics). Same note belongs in `atlas/experiment.md` to correct the "best per-family" wording.
