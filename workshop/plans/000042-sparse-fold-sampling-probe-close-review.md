# Boundary Review — metis#42 (whole-issue close)

| field | value |
|-------|-------|
| issue | 42 — sparse fold sampling — generalize --fast to m-of-k + 10-fold attenuation probe |
| repo | metis |
| issue file | workshop/issues/000042-sparse-fold-sampling-probe.md |
| boundary | whole-issue close |
| milestone | — |
| window | 44a326be3a3726b5ac5250bd4a8f4944831efd91..HEAD |
| command | sdlc close --issue 42 |
| reviewer | claude |
| timestamp | 2026-07-14T18:48:49-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All verification complete. Here is the review.

```verdict
verdict: SHIP
confidence: medium
```

The `--sample m` seam delivers exactly what the Spec committed to, by the mechanism the Spec named: the outer partition is still always materialized k-way (`materializeOuterAnalysis(k, …)`, sweep.go:340), `CVDriver{K: runFolds}` enumerates only the 0..m−1 prefix folds (driver.go:68-73 — in-bounds against the k-length `analysisRefs`), and both the sealed inner CV and the held-out scoring stay at full k (`runOuterFold`/`scoreOnOuterFold` take `k`, not `runFolds`), so the estimand is genuinely preserved while m only trades precision for cost. All three misuse guards error loudly and are tested; the ledger test pins fold indices {0,1} present and fold 2 absent for `--sample 2` of k=3. The Done-when's probe side is also verifiably landed in this repo: metis#36's Log carries the k5-vs-k10 attenuation tables, and atlas/experiment.md documents the flag with the m−1-df caveat. Confidence is **medium** only because I could not execute the test suite — the Bash harness in this review session fails before any command runs (`EPERM` creating `~/.claude/session-env/...`, even with the sandbox bypass), so the green-suite claim rests on reading, not running. The main agent should attach a fresh `go test ./cmd/metis/ ./pkg/sampler/` pass as the `--verified` evidence at close.

**1. Strengths**
- The generalization is the minimal correct one (ARCH-DRY): `--fast` and `--sample` collapse into one `runFolds` resolution switch (sweep.go:206-224) feeding the existing `CVDriver{K: runFolds}`/banner/`reportEstimate` seam — no parallel code path, no touched fold mechanics.
- Guard placement is right: the three guards sit before the `dryRun` branch, so misuse errors even on a dry run, and each error message teaches the correct model ("the outer partition has exactly k folds", "--fast is shorthand for --sample 1").
- `TestNestedCV_SampleRunsMOfKFolds` pins the load-bearing invariant — not just "2 folds ran" but *which* folds of the k-way partition ran, via the ledger's `OuterFold` tags (nestedcv_e2e_test.go:146-157). That is the exact bug class this diff could ship (m-way re-partition instead of m-of-k sampling).
- Doc comments consistently carry the statistical contract (k = estimand, m = precision, m−1 df, "probe with it, never re-select what ships on it") at every surface: flag help, runOpts field, sweep.go switch, atlas. A future reader cannot miss the caveat.

**2. Critical findings** — none.

**3. Important findings** — none in the code. One process note: the suite could not be executed in this review session (harness failure above); re-run it before recording the close verdict.

**4. Minor findings**
- run.go:125-135 — `--sample` on a plain `type: experiment` file is silently ignored (dispatch goes to `runResolvedExperiment`, which never reads `o.sample`). Consistent with `--fast`'s pre-existing behavior, but the diff's own posture is "misuse fails loudly," and the flag help's "errors on a single-config (flat) run" reads as covering this case. Worth a shared guard for both flags someday.
- sweep.go:212-220 — guard ordering means `--sample -1` on a *flat* shape reports the flat-shape message rather than the range message. Errs loudly either way; cosmetic.
- metis#42 Log references `scratchpad/k5_vs_k10.py` with no repo qualifier and it's not in metis (presumably kbench). Qualify the pointer so the "rerunnable" claim is actionable.

**5. Test coverage notes**
- Covered: m-of-k fold subset + ledger tagging + banner/estimate lines; guards for m>k, negative m, flat shape, `--sample`+`--fast`. `foldShapeMD` has exactly one `k: 2`, so the `k: 3` mutation in the new test is unambiguous.
- Gaps (cheap, non-blocking): no `--sample k` boundary case (should be identical to the default full run); no pin on the plain-experiment posture (see Minor 1); nothing asserts `--sample` composes with `--dry-run` (guards-before-dry-run is currently only implicit in code order).
- kbench-side Done-when items (`titanic-sweep-k10.md`, the probe run, RUNBOOK note) are outside this repo's review window; the Log documents them (kbench commit 321036d) and the metis-side residue (#36 Log tables) checks out.

**6. Architectural notes**
- **ARCH-DRY: pass.** One `runFolds` seam serves default/`--fast`/`--sample`; guards and messaging live in a single switch. If a third fold-count knob ever appears, normalize `--fast` to `sample=1` at flag-parse instead of adding a fourth case.
- **ARCH-PURE: pass.** The new logic is a pure integer derivation in the orchestration shell; the pure `CVDriver` is untouched, and the e2e tests drive everything through the injected fake executor — no subprocess, no mocks-reasserting-implementation.
- **ARCH-PURPOSE: pass.** The issue's point was the *probe*, not just the flag — and the probe ran, the pre-committed decision rule was applied, and the findings landed in metis#36's Log (verified in-tree). Shadow-sweep of the flag's documentation surface: flag help, runOpts comment, sweep.go comment, and atlas all state the same k-vs-m model; no stale restatement found.

**7. Plan revision recommendations** — none. The issue's Plan (all three boxes ticked) matches what the code and Logs actually deliver; there is no separate `workshop/plans/` artifact for this issue, appropriate for its size.
