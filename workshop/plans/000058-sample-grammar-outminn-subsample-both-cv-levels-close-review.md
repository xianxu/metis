# Boundary Review — metis#58 (whole-issue close)

| field | value |
|-------|-------|
| issue | 58 — sample grammar outMinN: subsample both CV levels |
| repo | metis |
| issue file | workshop/issues/000058-sample-grammar-outminn-subsample-both-cv-levels.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0e66914d89694f4ccaf0b24a47f7d25f6014f9fa^..HEAD |
| command | sdlc close --issue 58 |
| reviewer | claude |
| timestamp | 2026-07-18T12:38:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All verification complete. The Spec-mandated atlas note about select-side fairness is missing — that's my one substantive finding. Writing up the review now.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This boundary delivers the #58 grammar cleanly and the core design claim survives adversarial reading: the `splitK`/`runK` split is real and correctly one-directional — `buildFoldExperiment` and `partitionRef` only ever see `splitK`/`splitFolds` (sweep.go:292, 701, 747), while only `FixedKFolds` sees `runK` (sweep.go:180), which is exactly what makes subset runs address-compatible with full runs. The e2e suite pins the Done-when directly (cache-escalation with spawn-count and ledger-convergence assertions), the caller sweep verifiably reached kbench (RUNBOOK-sweep.md:37-38 and titanic-sweep.md:80 use the new grammar; remaining `--sample 3` hits are only in exempt `workshop/history` logs), and the untouched `innerk_e2e_test.go:28` banner assertion proves the unsampled display format didn't move. What keeps this from SHIP is one Spec item that didn't land: the Spec explicitly says "Record this reasoning in the atlas: select-side fairness needs no new guard (residual raggedness exists only after an interrupted run, and a completed re-run heals it)" — the atlas gained the cache-escalation/dedupe mechanics but not that fairness analysis (grep for ragged/fairness/interrupted in `atlas/` finds nothing from this change). One caveat on process: Bash is broken in this review environment (harness EPERM creating its session-env dir, unrelated to the diff), so I could not execute `go test` myself; all verification here is static reading of the committed tests and code paths, which I did line-by-line.

**1. Strengths**

- The `runInnerK := splitFolds` initialization (sweep.go:236-240) is the subtle flat-path bug the plan's review cycle caught, and it's correct: on a flat run `splitFolds = k` (inner_k ignored loudly), so `seededTotals` gets the right board denominator; `seededTotals`'s flat branch (progress.go:136-137) uses only that fold count, confirmed.
- The `< 1` guards kept despite CLI-side rejection (sweep.go:248-250) are genuinely load-bearing — `runOpts` is a direct-construction seam every e2e uses, and both negative-value subtests (`Out: -1` in nestedcv_e2e_test.go, `In: -1` in sample_e2e_test.go) pin that seam explicitly rather than the parser.
- `TestSample_CacheEscalationConverges` (sample_e2e_test.go:110-175) is a model Done-when proof: exact spawn counts (5 then 2 trains, 1 features), outer-refit HIT reasoning documented against fake-winner stability, and per-fold row-count convergence — not a banner-substring smoke test.
- The overflow handling in `parseSample` (sample.go:31-33) — `strconv.Atoi` chosen specifically so `out99999999999999999999` errors loudly instead of silently running the full sweep, with a test case pinning it.
- Test-division discipline: legacy refusals stayed in `TestNestedCV_SampleGuards` (they predate #58), new surface went to `sample_e2e_test.go`, with cross-referencing comments in both — no duplicated coverage, no orphaned coverage.

**2. Critical findings** — none.

**3. Important findings**

- **atlas/experiment.md — Spec-mandated select-fairness note missing.** The Spec (issue lines 43-45) explicitly requires recording in the atlas that select-side fairness needs no new guard because residual ledger raggedness exists only after an interrupted run and a completed re-run heals it. The atlas text (atlas/experiment.md:159-163) covers the dedupe/convergence mechanics but omits this reasoning — which is precisely the sentence a future agent needs when deciding whether `metis select` needs a raggedness guard. Fix: one sentence in the #58 paragraph of atlas/experiment.md, e.g. after "absorbs re-emitted rows": "select needs no raggedness guard — residual raggedness exists only after an interrupted run, and any completed re-run heals it." (Traceability + docs gate; the change is cheap and the close gate's atlas guard is already satisfied structurally, so this is non-blocking.)

**4. Minor findings**

- `seededTotals`'s last parameter is still named `k` (progress.go:131) but now receives `runInnerK` — the plan's Task 3 anticipated this ("only if the param name misleads"); it now mildly misleads. Rename to `runInnerK` or `foldsPerConfig`.
- The `cfKey` struct + inner-row aggregation loop is duplicated between `TestSample_OutInPrefixSubset` (sample_e2e_test.go:76-90) and `TestSample_CacheEscalationConverges` (:147-160) — a small shared helper (`innerFoldsByConfig(led)`) would serve both (ARCH-DRY, test-code tier).
- No test pins the new `--fast` banner rendering as `1/k` (the explicit plan decision at Task 3 Step 3); if it silently regressed to the plain count nothing would fail.
- Issue frontmatter `status: working` — presumably updated by the close flow itself; noting only so it isn't forgotten.

**5. Test coverage notes**

- Covered well: grammar table (16 cases incl. overflow, order, case-sensitivity), both negative-value runOpts seams, inner range vs `inner_k`, out+in subset ledger coordinates, cache escalation + convergence, retired bare-integer form.
- Gap (minor): no e2e for an `in<N>`-only run (`Out` unset → all k outer folds each running the sampled inner prefix). The code path is nearly identical to the covered `out1in2` case (`runFolds` just stays `k`), so risk is low, but it's the one grammar form with zero end-to-end execution.
- I could not run the suite in this environment (Bash harness failure, pre-existing); the assertions were verified statically against the harness helpers (`foldFakeExec`, `countCalls`, `loadLedgerOrFatal` all exist in shapesweep_test.go with matching signatures).

**6. Architecture**

- **ARCH-DRY: pass.** The grammar exists in exactly one place (`parseSample`); `fmtLevel` serves both the live and dry-run banners; validation extends the existing #42 block rather than forking it. Only the test-helper duplication noted above.
- **ARCH-PURE: pass.** `parseSample` is a pure function with a table-driven test requiring zero IO, matching its plan classification. The `runK` plumb threads through the existing thin wiring layer without pulling logic into IO.
- **ARCH-PURPOSE: pass, one shard flagged above.** Shadow-sweep executed: consumers are the flag help (updated), `--fast` help (updated), atlas (updated, two sections), kbench runbooks (verified updated in the peer tree), and `workshop/` (exempt by documented rationale — history logs record what was actually executed). No hand-maintained restatement of the old grammar survives on any current-usage surface. The one under-delivery against the stated purpose is the missing atlas fairness note (Important finding above) — the Spec asked for the *reasoning* to be recorded, not just the mechanism.
- For upcoming work (metis#54 racing): the `splitK`/`runK` seam is exactly the right substrate for an adaptive sampler — racing can vary `runK` per config over an unchanged partition and inherit cache continuity for free. Worth a sentence in #54's eventual plan.

**7. Plan revision recommendations** — none. The plan's Core concepts table matches the code at every row (entities exist at stated paths, `parseSample` is genuinely PURE, the modified entities show the expected diffs), and the implementation's deviations from the plan are already logged as fix-forward deltas in the issue Log.
