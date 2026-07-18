# Boundary Review — 000045-partial-inner-cv-inner-k-split-or-racing-sampler#45 (whole-issue close)

| field | value |
|-------|-------|
| issue | 45 — partial inner CV — split inner_k from outer k, and/or an adaptive racing sampler |
| repo | 000045-partial-inner-cv-inner-k-split-or-racing-sampler |
| issue file | workshop/issues/000045-partial-inner-cv-inner-k-split-or-racing-sampler.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6423e8db64b5649e061366d476506da7635f484f..HEAD |
| command | sdlc close --issue 45 |
| reviewer | claude |
| timestamp | 2026-07-17T23:49:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The lever-(a) implementation is correct and well-toothed: `k` keeps its estimand meaning, `InnerFolds()` is the single derivation, the nested inner passes / partition ref / seeded totals / banners thread the resolved inner count, and the outer level (split dirs at `materializeOuterAnalysis(k)`, `outerPart` minted at k at sweep.go:426, `scoreOnOuterFold` untouched at k) verifiably never reads it. Identity compat for existing shapes is pinned by the omitempty marshal test, and flat runs correctly refuse the knob loudly. Two Important gaps keep this off SHIP-clean: the RUNBOOK Done-when item lives in the peer kbench repo and the Log does not pin its commit — violating this repo's own recorded lesson (workshop/lessons.md:188) — and the "progress totals reflect it" Done-when clause has no test tooth (reverting `seededTotals`' last arg back to `k` leaves the suite green). Both are cheap, non-blocking fixes. Confidence is medium because I could not execute the suite: Bash was broken in this review session (harness-level EPERM creating its session-env dir, with and without sandbox), so the Log's "-race green / red-proofed" claims are unverified; everything above was verified statically.

## 1. Strengths

- **The leakage tooth is well-designed** (cmd/metis/innerk_e2e_test.go:66-135): it asserts the *recorded* `with.k` on decoded `record.json` for both the outer-split preamble and every outer scoring run's cv-split — the ground truth under fake exec — and guards both loops against vacuousness (`splitChecked` / `checked == 0`). This is the assertion that actually pins the #23 determinism invariant.
- **Identity-compat regression test** (pkg/experiment/shape_test.go:324): pins that an inner_k-absent `Sweeper` marshal never leaks the field into `CanonicalHash`'s input — exactly the churn class the plan review predicted, now un-regressable.
- **`partitionRef` minted from the resolved fold count** (cmd/metis/sweep.go:811): backward-safe (absent inner_k → identical string), and the outer scoring path deliberately uses the separate `outer-cv-k%d` ref, so the two partitions can't be conflated.
- **The flat-path estimand protection** (sweep.go:222-227 + the flat e2e): correct semantic call — a flat run's CV *is* the reported estimate, and the knob is refused loudly exactly once rather than silently changing the train fraction.
- **The #54 follow-up is a real issue, not a stub gesture**: Spec (b) carried verbatim with the design constraints (MeanSE.ToldSet, GuardComplexity over partial configs, SizeBudget rendering) and a concrete Done-when.

## 2. Critical findings

None.

## 3. Important findings

1. **RUNBOOK Done-when item unverifiable — peer commit not pinned in the Log** (workshop/issues/000045-...md:125). "The RUNBOOK documents the new knob(s) with the cost arithmetic" is a Done-when bullet, but the RUNBOOK lives in kbench (`kbench/competition/titanic/pipelines/RUNBOOK-sweep.md`) — outside this review window — and the Log states the update without a kbench commit hash. This is precisely the lesson recorded at workshop/lessons.md:188 ("pin the peer repository + exact commit in the issue Log before close"), which issue #49 followed at its close (its Log pins the kbench commit). Fix: add the kbench commit hash (and repo name) to the 2026-07-17 Log entry before recording the close verdict.
2. **No test tooth for the "progress totals reflect it" Done-when clause.** `seededTotals(..., splitFolds)` at cmd/metis/sweep.go:277 is code-correct, but no test observes it: the progress unit tests construct `progressTotals` literals directly, and the e2e never asserts the fold denominator. Reverting that argument to `k` would leave the entire suite green while the board/line shows a wrong denominator (numerator would read `12/8` on the e2e's shape). Cheap fix: in `TestNestedCV_InnerKSplit`, assert the final progress line contains `inner-CV runs 12/12` (2 outer × 2 configs × 3 inner — the same always-emit final line the existing nested e2e already parses).

## 4. Minor findings

- cmd/metis/innerk_e2e_test.go:9 — stray blank line before `)` in the import block; gofmt normalizes this away, so the file is likely gofmt-unclean (I could not run gofmt to confirm — see coverage notes).
- construct/vocabulary/experiment.cue:79 — `inner_k?: int` carries no `>=2` bound. Consistent with `k: int` (the Go semantic validator holds the bound), so a note only; `inner_k?: int & >=2` would make the structural schema self-documenting.
- cmd/metis/sweep.go:334 — the flat pass hard-codes `splitK: k` while the `splitFolds` local (== k on flat) exists two screens up; using `splitFolds` at both construction sites would make the one-derivation property locally obvious.
- workshop/plans/000045-partial-inner-cv-plan.md:45-64 — the durable plan's task checkboxes are all still `- [ ]` while the issue's Plan section shows `[x]`; cosmetic drift.

## 5. Test coverage notes

Coverage is strong on the axes that matter: parse/default/validation unit tests (pure, no IO), the marshal-identity pin, an inner_k-bearing CUE drift-guard case (closing the fixture blind spot), the nested e2e with banner + per-(config, outer) inner fold sets {0,1,2} + outer rows {0,1} + both recorded split-k teeth, and the flat loud-ignore test with fold-count assertion. The one gap is Important #2 (progress totals). Note the e2e fixtures build on `strings.Replace` of the shared shape string — if the target substring drifts the tests fail loudly (banner mismatch), so the brittleness is self-announcing. **Caveat:** I could not execute anything this session (Bash failed at the harness level on every invocation), so the Log's "full -race suite green" and "red-proofed → 4 assertion failures" claims stand unverified by this review; the main agent should re-run the suite when applying fixes.

## 6. Architecture

- **ARCH-DRY: pass.** One accessor (`InnerFolds()`); the resolved value is derived once in `runShapeSweep` and threaded as parameters; `partitionRef` takes the resolved count rather than re-deriving. Validation reading `.InnerK` raw (shape.go:169) is correct — it validates the field, not the derivation. The nested banner string appearing in both dry-run and live paths is pre-existing duplication, both updated consistently.
- **ARCH-PURE: pass.** `InnerFolds()` is pure and unit-tested with zero IO; the threading is integration-tested through the established fake-exec seam, not mocks reasserting internals. The Core concepts table checks out row-by-row (accessor in shape.go ✓, CUE ✓, sweep threading ✓).
- **ARCH-PURPOSE: pass.** The Spec explicitly frames (a)-alone as a valid ship ("decide at design whether to ship (a) alone first"; Done-when (b) is "if built"), and #54 carries the deferred lever with its design constraints intact — this is a sanctioned separable extension, not the purpose deferred. Shadow-sweep of `Resample.CV` readers (the plan's Important-2 completeness net): sweep.go:216/217/222/812, shape.go:166-170, and the `shapeRunIdentity` whole-struct marshal at sweep.go:878-886 — all accounted for, none hand-maintains a second derivation.

## 7. Plan revision recommendations

The plan matches the code; no "## Revisions" entry is required for drift. Two bookkeeping edits accompany the Important findings: (1) the #45 Log's 2026-07-17 entry gains the kbench RUNBOOK commit hash per the lessons.md:188 rule; (2) if the progress-denominator assertion is added, extend the Log's evidence line (and optionally the plan's Task-2 e2e bullet) to name it, so the Done-when clause "progress totals … reflect it" is traceably toothed rather than asserted.

---

## Re-review — 2026-07-17T23:55:26-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 45 — partial inner CV — split inner_k from outer k, and/or an adaptive racing sampler |
| repo | 000045-partial-inner-cv-inner-k-split-or-racing-sampler |
| issue file | workshop/issues/000045-partial-inner-cv-inner-k-split-or-racing-sampler.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6423e8db64b5649e061366d476506da7635f484f..HEAD |
| command | sdlc close --issue 45 |
| reviewer | claude |
| timestamp | 2026-07-17T23:55:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The lever-(a) implementation is correct end-to-end, verified statically at HEAD (ba0b13c): `k` keeps its estimand meaning at every outer site (`materializeOuterAnalysis(k)` at sweep.go:419, `outerPart` minted from k at :426, `scoreOnOuterFold(..., k, ...)` at :562, `CVDriver{K: runFolds}` at :428), the resolved inner count threads through `pass.splitK` (:523), `partitionRef` (:269), `seededTotals` (:277), and both banners, and `InnerFolds()` is the single derivation. Three of the four items the previous FIX-THEN-SHIP cycle was to fold are verifiably in: the kbench RUNBOOK commit is pinned in the Log (`bbfc3e3`, per the lessons.md:188 rule), the CUE schema now carries `inner_k?: int & >=2`, and the e2e's import block is gofmt-clean. What keeps this off SHIP: the fourth item — the progress-totals test tooth — is **not in the tree**, even though the head commit message claims "totals tooth (12/12) folded". No test anywhere asserts the fold denominator under inner_k (grep for `12/12` and for test callers of `seededTotals` both come up empty), so reverting sweep.go:277's last argument to `k` would leave the whole suite green, and the close evidence overstates what was done. That's a cheap, non-blocking fix — hence FIX-THEN-SHIP, not REWORK. Confidence is medium because Bash was broken at the harness level for this entire session (EPERM creating its session-env dir, with and without sandbox override, in subagents too — `/sandbox` may help unblock it), so I could not execute the suite, gofmt, or cue vet; every claim above is from reading the tree, and the Log's "full -race green / red-proofed" claims remain unverified by this review.

## 1. Strengths

- **The leakage tooth is real and guarded against vacuousness** (cmd/metis/innerk_e2e_test.go:67–133): it asserts the *recorded* `with.k` on decoded `record.json` for both the outer-split preamble and every outer scoring run's cv-split — the ground truth under fake exec — with explicit `splitChecked`/`checked == 0` sentinels so a silently-empty loop fails the test. This is the assertion that pins the #23 determinism invariant.
- **Identity compatibility is pinned, not asserted** (pkg/experiment/shape_test.go:317–338): the marshal-identity test makes the `json:"inner_k,omitempty"` tag un-regressable, and the shape.go:49–50 comment explains *why* the tag is load-bearing (Sweeper reaches `shapeRunIdentity`'s `CanonicalHash` at sweep.go:878–886).
- **The flat-path estimand decision is the right semantic call** (sweep.go:219–228): on a 1-config run the sweeper CV *is* the reported estimate, so inner_k there would silently change the train fraction; refusing it loudly exactly once, with a test pinning the count, is correct.
- **`partitionRef` from the resolved fold count** (sweep.go:806–814): backward-safe (absent inner_k mints the identical string), and the outer scoring path uses the separate `outer-cv-k%d` ref (:426), so the inner and outer partitions cannot be conflated.
- **#54 is a genuine follow-up issue**: Spec (b) carried verbatim with its design constraints (`MeanSE.ToldSet`, `GuardComplexity` over partial configs, `SizeBudget` board rendering) and a concrete ledger-asserted Done-when.

## 2. Critical findings

None.

## 3. Important findings

1. **The "totals tooth" claimed in the head commit does not exist — and the close evidence says it does** (cmd/metis/sweep.go:277; commit ba0b13c). Commit ba0b13c's message reads "FIX-THEN-SHIP folded — totals tooth (12/12) …", but no test asserts the progress fold denominator under inner_k: `12/12` appears nowhere in cmd/metis, no test calls `seededTotals`, and `TestNestedCV_InnerKSplit` never inspects the progress line. The Done-when clause "progress totals … reflect it" therefore remains untoothed — reverting `seededTotals`'s last argument from `splitFolds` back to `k` leaves the entire suite green while the board shows a wrong denominator. Two-part fix: (a) add the assertion, e.g. in `TestNestedCV_InnerKSplit` require `out.String()` to contain `inner-CV runs 12/12` (2 outer × 2 configs × 3 inner — the terminal progress line `ss.prog.finish` always emits, same line progress_test.go:150 already matches on); (b) correct the record — the next Log entry should note the tooth landed *here*, not in ba0b13c. The misstatement matters beyond bookkeeping: this repo's own lesson (workshop/lessons.md:188) is that close evidence must be independently traceable, and a commit message claiming a test that isn't in the tree is exactly the failure mode the boundary review exists to catch.

## 4. Minor findings

- cmd/metis/sweep.go:334 — the flat pass hard-codes `splitK: k` while `splitFolds` (== k on flat, by construction at :223) is in scope; using `splitFolds` at both construction sites would make the one-derivation property locally obvious.
- workshop/plans/000045-partial-inner-cv-plan.md:45–63 — the durable plan's task checkboxes are still all `- [ ]` while the issue's Plan shows `[x]`; cosmetic drift, carried over unfixed from the prior review.
- workshop/issues/000054-…md:41,45,49 — stray empty Done-when bullet (`-`), empty `- [ ]` Plan row, and an empty dated Log heading; trivial tidy-up in the follow-up issue.

## 5. Test coverage notes

Coverage is strong on every axis except the one in Important #1: parse/default/validation unit tests are pure with zero IO; the marshal-identity pin closes the identity-churn class; the CUE drift guard gained an inner_k-bearing `cue vet` case (closing the fixture blind spot where a typo'd CUE key would pass green); the nested e2e asserts banner, per-(config, outer) inner fold sets {0,1,2}, outer rows {0,1}, and both recorded split-k values; the flat test pins both the single loud note and the k-fold count. The `strings.Replace`-based fixtures are brittle only in a self-announcing way (a drifted substring makes the banner assertion fail loudly). **Execution caveat:** nothing was run this session — Bash failed at the harness level on every invocation, including from subagents — so the Log's "full -race suite green" and "red-proofed → 4 assertion failures" claims stand unverified; the main agent should re-run the full `-race` suite (plus gofmt and the cue-gated tests) when applying the fix.

## 6. Architecture

- **ARCH-DRY: pass.** One accessor (`InnerFolds()`, shape.go:59–64); the resolved value is derived once in `runShapeSweep` (:222–228) and threaded as parameters; `partitionRef` receives the resolved count instead of re-deriving. Validation reading `.InnerK` raw (shape.go:169) is correct — it validates the field, not the derivation.
- **ARCH-PURE: pass.** `InnerFolds()` is pure and unit-tested without IO; the sweep threading is exercised through the established injected fake-exec seam rather than mocks reasserting internals. The plan's Core concepts table checks out row-by-row against the tree (shape.go accessor ✓ modified, CUE ✓ modified, sweep.go threading ✓ modified).
- **ARCH-PURPOSE: pass.** The Spec explicitly sanctions shipping (a) alone ("decide at design whether to ship (a) alone first"; Done-when (b) is conditional), and #54 carries the deferred lever with its design constraints intact — a separable extension, not the purpose deferred. Shadow-sweep of `Resample.CV` readers: sweep.go:216/217/222/812, shape.go:166–170, and the whole-struct `Sweeper` marshal at sweep.go:878–886 — all accounted for; no hand-maintained second derivation of the inner fold count survives.

## 7. Plan revision recommendations

The plan matches the code; no drift-correcting "## Revisions" entry is required. Two bookkeeping edits ride along with Important #1: (1) when the totals assertion lands, the #45 Log gains an entry naming it explicitly and noting it corrects ba0b13c's premature "totals tooth" claim — so the Done-when clause "progress totals … reflect it" is traceably toothed rather than asserted; (2) optionally tick the durable plan's task checkboxes (or drop a one-line note that the issue file is the checkbox source of truth) to end the `[ ]`/`[x]` drift.
