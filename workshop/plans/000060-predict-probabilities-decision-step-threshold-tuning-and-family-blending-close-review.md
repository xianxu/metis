# Boundary Review — metis#60 (whole-issue close)

| field | value |
|-------|-------|
| issue | 60 — predict probabilities + decision step: threshold tuning and family blending |
| repo | metis |
| issue file | workshop/issues/000060-predict-probabilities-decision-step-threshold-tuning-and-family-blending.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9f34e4e531c210e40a6dea0f59c1af0abb116dd5^..HEAD |
| command | sdlc close --issue 60 |
| reviewer | claude |
| timestamp | 2026-07-19T00:41:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have what I need. Assembling the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** The whole-issue close delivers both Done-when items: M1 (the leaf-local decision layer — already boundary-reviewed FIX-THEN-SHIP, and I confirmed all three of that review's Important findings were actually folded: `parse_decide` now rejects unknown inner keys with the typo'd-key test, the anti-vacuity `any(o != 0)` assert landed, and the class-mismatch loud test exists) and M2 (`metis blend`, a weights-only tilted-log soft vote over promoted runs, materialized as a run dir whose submission step executes through the real `execStep.Execute` seam — I verified `execStep.Execute`, `stepPath`, and `hoistShapePath` all exist as the plan named them, and that promoted runs' `record.json` genuinely carries `experiment`/`code_fingerprint` via `assembleRecord`/`backfillCodeManifest`, so the provenance guard is real in production). The metis#64 commits (`freeParamsEqual` null≡absent, `renderFreeParamValue`) fall inside this window but belong to a separately-closed issue with its own review in `workshop/history/`; I sanity-checked them anyway and found them sound. Nothing is Critical; three cheap Importants below, led by a docs-gate gap: the atlas never mentions the new `blend` verb. One caveat: **this session's Bash tool is broken at the harness level** (`EPERM` creating `~/.claude/session-env/...`, before any command runs, sandbox override included — the same failure the M1 review recorded), so I could not execute either test suite; everything here is static reading, and "suites green" remains the implementor's Log claim.

### 1. Strengths

- **The tilted-log combination rule is correctly derived and correctly pinned.** `blendCombine` (`cmd/metis/blend.go:299`) matches `apply_offsets` semantics exactly at the single-member/weight-1 corner, and `TestBlendCombine_SingleMemberIdentityPinsClip` pins the cross-language clip constant (1e-12) by constructing a case that fails under any other clip — a genuinely clever invariant test for a constant that can't be shared across Go/Python.
- **The symbol-level plan paid off verbatim.** The header comment in `blend.go:17-19` records *why* `runResolvedExperiment` is wrong for this path (DAG-validates `needs`, clobbers record.json, fires capture) — that's a constraint the code can't show, exactly the right kind of comment.
- **Offsets align by class label, never position** (`readOffsets`, `blend.go:264`; `realignColumns`, `blend.go:445`), with the label-collision surface documented. The `classLabel` int-rendering mirrors predict.py's f-string suffixes.
- **Guard posture inherited as the plan-review lesson demanded:** missing `probabilities.csv` refuses naming the remedy ("re-promote"), mixed provenance refuses without `--allow-mixed`, and `--allow-mixed` still warns loudly — all three paths tested.
- **The blendID hashes (member, NORMALIZED weight) pairs** so equal-weight blends collide correctly across `--weights 1,1` vs default, and different weights don't — pinned by test.

### 2. Critical findings

None.

### 3. Important findings

1. **Atlas update missing for the `metis blend` verb** — docs gate. `atlas/experiment.md` documents the run/select/ledger command model (line ~47 and the step catalog), and the M1 diff added the decision-layer bullets, but the word "blend" appears nowhere as a verb: no `metis blend` CLI surface, no `runs/blend-<hash>/` dir flavor, no blend-flavored record.json, no leaderboard-only honesty caveat. A new subcommand + a new run-dir species is exactly the "new architectural surface/flow" AGENTS.md §8 gates on. Fix: a short bullet in experiment.md's command-model section (verb, tilted-log rule, provenance guard, the no-OOF caveat, `kaggle submit --run blend-...` compatibility). (No README.md exists in this repo, so the README half of the gate is vacuous.)
2. **`realignColumns` is never exercised by any test** — `cmd/metis/blend.go:445`. Every test member shares column order, so the by-name permutation path (an index-map over probabilities that, if mis-indexed, silently permutes class probabilities into garbage predictions) ships untested. This is precisely the "kind of bug this diff could ship" class. Cheap fix: a unit test feeding a reversed-column member and asserting the blend result is order-invariant, plus the column-set-mismatch loud path. (The runBlend-level id-order mismatch refusal is similarly untested; fold it in.)
3. **`normalizeWeights` lets NaN and ±Inf through** — `cmd/metis/blend.go:344`. `strconv.ParseFloat` accepts `"NaN"`/`"Inf"`; `NaN <= 0` is false, so the positivity guard passes, the sum goes NaN, every score is NaN, `score > bestScore` is always false, and every row silently gets the first class — garbage submission material with zero warning, in a verb whose whole posture is loud honesty. Inconsistent error handling within the same function (positivity is loud, non-finiteness is silent). Fix: `if !(v > 0) || math.IsInf(v, 0) { return loud error }`.

### 4. Minor findings

- `shipSteps` (`blend.go:196`): predict takes the *first* matching ship step (`!haveP` guard) but submission takes the *last* (no `!haveS` guard) — make the tie-break consistent.
- `--runs` entries aren't `TrimSpace`d but `--weights` entries are (`cmdBlend`) — `--runs "m1, m2"` fails with a confusing not-found on `" m2"`.
- Provenance reader swallows `record.json` read/unmarshal errors silently (`blend.go:139-146`): if *all* members lack records (hand-rolled dirs), the mixed-provenance guard vacuously passes; and a mismatch against a missing record prints `{ }` vs `{...}`. Practical exposure is low (promoted runs always have records), but a one-line warning on read failure would match the posture.
- `equalStrings` re-implements `slices.Equal` (ARCH-DRY, stdlib since Go 1.21).
- `readProbabilities` error text can render as "unreadable or empty: <nil>" on a header-only file.
- `readOffsets`' classes-don't-cover-column loud path is untested (only the happy alignment runs in e2e).
- Housekeeping: `workshop/plans/000060-decide-step-plan.md` Task 1–3 checkboxes are still `- [ ]` on disk despite both M1 commits + close (the issue's `## Plan` M1/M2 rows *are* ticked, so the issue-side gate is satisfied); the M1 review flagged this too.

### 5. Test coverage notes

Blend coverage is strong where it exists: the pure combine (boundary flip with/without tilt, single-member clip pin, shape mismatches), weight normalization, id sensitivity, and a real e2e through a 2-line `#!/bin/sh` toy submission step that exercises the full `execStep` env contract, the literal `submission/submission.csv` landing path, runref-semantics slug resolution, the provenance guard both ways, and the missing-probabilities refusal. The gaps are Importants 2–3 (realign path, non-finite weights) plus the readOffsets coverage minor. **I could not run either suite** — the session's Bash is harness-broken (EPERM before any command executes), so "blend 8/8, full Go, 104 python green" is the implementor's claim in the Log, not something I re-verified. The gate operator should weigh that; it's why confidence is medium rather than high.

### 6. Architectural notes

- **ARCH-DRY: pass.** Blend reuses `execStep.Execute` + `stepPath` (the plan's named symbols, confirmed real), embeds `record.RunRecord` rather than inventing a schema, and derives step ids from the shape instead of hardcoding "predict"/"train". The unavoidable cross-language duplications (clip constant, class-label rendering) are each pinned by a test or comment naming the twin. Residual nits: `equalStrings`, and the decide-frame duplication across the two Python test files (carried from M1, tolerable).
- **ARCH-PURE: pass.** `blendCombine`/`normalizeWeights`/`blendID`/`realignColumns`/`readOffsets`-alignment are pure and unit-tested without IO; `runBlend` is the thin shell; the test seam is an injected `stepPath`, not a mock of the executor. Python side unchanged from M1's pass.
- **ARCH-PURPOSE: pass.** The issue's purpose — threshold tuning inside the seal plus family blending on probability outputs — is fulfilled, not subset-shipped: the M2 verb consumes exactly the M1 artifacts (`probabilities.csv` by label, `offsets.json` by persisted classes), the honesty caveat prints on every run, and the Spec's superseded step-type design is reconciled in the issue's `## Revisions`, not silently dropped. Shadow-sweep on the new artifacts: both consumers (predict, blend) derive from the persisted source with validation; the one hand-maintained restatement that *didn't* get updated is the atlas (Important 1).
- Forward note: `runs/blend-*/` is a new run-dir species that ledger/select readers never scan today (by design); if any future reader globs `runs/*`, the blend dirs' record.json (status "ok", no PointAddr) will be its first surprise — worth a line in the atlas bullet when Important 1 is fixed.

### 7. Plan revision recommendations

None required — the plan's M2 section matches the delivered code closely (the named reuse symbols, the guard list, and the test list all landed). Two optional touch-ups if the plan file is opened anyway: tick the Task 1–3 checkboxes (housekeeping minor above), and note under M2 that the atlas command-model bullet for the verb ships at issue close (Important 1) so the plan doesn't imply docs were complete at the M2 commit.
