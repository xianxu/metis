# Boundary Review — metis#19 (milestone M1)

| field | value |
|-------|-------|
| issue | 19 — selection objectives — 1-SE rule + mean-std (configurable sweeper select rule, not raw cv-max) |
| repo | metis |
| issue file | workshop/issues/000019-selection-objectives.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 31af7dc5592da74faeceb346a79d508bf4a370bd^..HEAD |
| command | sdlc milestone-close --issue 19 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-08T23:33:52-07:00 |
| verdict | unknown |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Failed to authenticate. API Error: 401 Invalid authentication credentials

---

## Re-review — 2026-07-09T09:55:13-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 19 — selection objectives — 1-SE rule + mean-std (configurable sweeper select rule, not raw cv-max) |
| repo | metis |
| issue file | workshop/issues/000019-selection-objectives.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 31af7dc5592da74faeceb346a79d508bf4a370bd^..HEAD |
| command | sdlc milestone-close --issue 19 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-09T09:55:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have a complete picture. Build, vet, all package tests, and CUE conformance are green; I've traced the select-rule logic, verified the load-bearing `familyOf` assumption against real `shape.Expand` output, and checked the docs gate.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1 boundary is functionally solid and architecturally clean: `go build ./...`, `go vet ./...`, `go test ./pkg/sampler ./pkg/experiment ./cmd/metis`, and the CUE conformance test all pass. The pure `SelectConfigs` rule is well-separated from IO, exhaustively unit-tested (including the load-bearing corner regression the issue exists to fix), and the type-widening ripple (`float64`→`FoldOutcome`, `Winner`→`SweepResult`) is threaded correctly through all three Sampler levels. No correctness bugs found. What holds it back from a clean SHIP is non-blocking: one docs-gate gap (a stale authoring doc that would now mislead an author into writing a rejected shape), one real test-coverage gap (the entire `minimize` band/parsimony path is unexercised), and Core-concepts plan-table drift that M2 will read. None are Critical.

### 1. Strengths (confirmed-good ground)

- **ARCH-PURE, exemplary.** `pkg/sampler/select.go` is fully pure (`math`/`sort`/`strings` only), unit-tested with hand-built `ConfigStat`s and zero mocks; the IO seam (`runPipelineFold` reading `run.Metrics`) stays thin in `cmd/metis/sweep.go:224`. This is exactly the pure-core/thin-shell split the principle asks for.
- **The load-bearing corner regression is genuinely pinned**, not asserted. `TestSelect_PctLoss_TieBreaksToMean` (select_test.go) traces band → ε-bin → mean tie-break and asserts recovery of `depth4/feat6` over both the deep overfitter and the sparse corner — the precise empirical failure (#19's cv-max→public 0.770 vs 0.782) the rule targets.
- **The plan-review ε-arithmetic lesson was actually applied.** `complexityBinRelTol = 0.10` (select.go:17) with `TestSelect_PctLoss_BinnedComplexity` landing cx 15/16 *inside* and 10/16 *outside* the bin — the off-by-a-boundary the plan review flagged is gone.
- **`familyOf` tagged-vs-untagged disambiguation is careful and its load-bearing assumption verified.** The `stepWith[key].(map[string]any)` assertion (select.go:202) matches real `shape.Expand` output — `bundled := map[string]any{label: r.value}` at pkg/shape/shape.go:176 — so it's not just faithful to a hand-built fixture. `TestFamilyOf` covers tagged sum, bare-string alt, list alt, and empty.
- **`HasComplexity` ("measured 0" vs "not measured") threaded cleanly** through `FoldOutcome`→`FoldScore`→`Aggregate`→`MeanSE` for the M2 guard, without over-building the guard itself. Good forward-design discipline.

### 2. Critical findings

None.

### 3. Important findings

- **Docs gate — stale authoring doc.** `construct/datatype/experiment-shape.md:61-66` still documents `select` as a scalar: *"`argmax-mean` (M1a); `one-std-err` / `pct-loss` are a *different* `Done` … (metis#19)"*. An author following this writes `select: argmax-mean` (bare scalar), which the new union + `KnownFields(true)` decode now **rejects** (can't unmarshal a scalar into `Select`). This is the identical file the M1a-1 lesson already flagged as a hidden doc consumer. Fix: describe the tagged union (`select: {pct-loss: {tolerance}}` etc., exactly-one, mirroring `driver`). atlas/index.md *was* updated — only this datatype doc lags.

- **Test coverage — the entire `minimize` band/parsimony path is unexercised.** Every `SelectConfigs` case in select_test.go uses `"maximize"`; the only minimize test (`TestGridConfigs_MinimizeDirection`) hits the argmax-mean default branch. So `withinBand`'s minimize branch (select.go:130), `parsimony`+minimize tie-break, and `argmaxMeanStd`'s `+lambda*std` minimize penalty (select.go:108) are all untested. The logic reads correct by inspection, but a sign flip in any of them would ship silently. Add one `minimize` pct-loss and one `minimize` mean-std case.

- **Plan Core-concepts table drift (consolidated).** The table claims shapes the code doesn't deliver — the code is *cleaner*, but the table is a stale map M2 will consume:
  - `sampler.FoldOutcome` — table says `pkg/sampler/aggregate.go`; code has it in `folds.go`.
  - `sampler.Winner` — table says `modified (+Family,Complexity)`; code adds only `Family` (complexity lives in `Score.MeanComplexity`, correctly avoiding a redundant field — ARCH-DRY).
  - `sampler.ConfigStat` — table/prose flattens `{FreeParams, Mean, SE, MeanComplexity, HasComplexity, ToldSet}`; code embeds `Score MeanSE` (cleaner reuse).
  - `SelectConfigs` gained a `seed int` param; `familyOf` dropped the planned `sweptTagPaths` param.
  
  (I'm rating this Important, not the rubric's default Critical: every entity *exists* at its stated package and there's no correctness/behavior impact — the risk is only an M2 author trusting a stale map. See §7 for the exact `## Revisions`.)

### 4. Minor findings

- `mean-std.lambda < 0` validation (shape.go:188) is untested — the shape_test.go mutator table covers select-none / two-branch / pct-loss≤0 but omits negative lambda.
- No e2e test derives families from real `shape.Expand` output; the format equivalence is verified against shape.go:176, but a belt-and-suspenders integration assertion (Expand a 2-model shape → 2 families) would harden it against future Expand changes.
- `reportWinner` prints `cx 0.0` across the whole leaderboard in M1 (expected — documented as "real values land in M2").
- `familyOf`'s 2-level `With[step][key]` limitation means a *deeper-nested* tagged sum is silently omitted from the family key — documented in the code comment; relevant when #21/GBM or nested family axes arrive.

### 5. Test coverage notes

Coverage of the new pure surface is strong: all four rules, cross-family argmax, no-tagged-sum implicit family, ε-bin boundaries, `Aggregate` complexity + `HasComplexity`, and validation (none/two/pct-loss≤0) are pinned with real assertions, not mock reasserts. Gaps: the `minimize` direction (Important, above), `mean-std.lambda<0` validation, and a real-Expand family-derivation integration test.

### 6. Architectural notes for upcoming work (M2)

- **Family-key format must not drift across the two surfaces.** `familyOf` emits path-qualified keys (`"train.model=rf"`); the plan's M2 Task 12 says to set the offline ledger's `Family` *bare* from `fp.train.model` (`"rf"`). If followed literally, the two `SelectConfigs` consumers produce **different keys for the same config**, silently breaking the "one rule, two surfaces, identical result" DRY property this design is built on. M2 must call `familyOf` (or exactly match its format) on the ledger path.
- **The guard is what makes parsimony honest — keep it on M2's critical path.** In M1 a `pct-loss` shape degenerates to argmax-mean-within-band (complexity=0, `HasComplexity` threaded but unconsumed). No live M1 shape triggers this (titanic-baseline stays argmax-mean; kbench shapes are a separate repo / M2), so it's not an M1 defect — but M2 Task 13's guard is the piece that turns the wired-but-inert `HasComplexity` into a real gate. Don't let it slip.
- **The second consumer (`metis ledger select`/`promote` family grouping) is the DRY payoff `ConfigStat`/`SelectConfigs` were designed for** and is still deferred to M2 (Task 12). The shared-rule shape is sound; it's just not yet exercised by two callers, so the ARCH-DRY win is unrealized until then. ARCH-PURPOSE passes for the *M1* boundary (M1 delivers its stated milestone purpose; measured complexity + verified acceptance is a genuine M2 boundary, not a disguised deferral of the point).

### 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000019-selection-objectives-plan.md` reconciling the Core-concepts table with what M1 shipped:
- `sampler.FoldOutcome` lives in `pkg/sampler/folds.go`, not `aggregate.go`.
- `sampler.Winner` gained `Family` only; complexity is carried in `Score.MeanComplexity` (no separate `Winner.Complexity` field — deliberate, ARCH-DRY).
- `sampler.ConfigStat` is `{Point shape.Point; Family string; Score MeanSE}` (embeds `MeanSE`), not the flattened field list.
- Signatures as-built: `SelectConfigs(rule, direction string, seed int, stats []ConfigStat)` and `familyOf(p shape.Point)` (no `sweptTagPaths`).
- Amend **Task 12** to require the offline ledger path to produce the *same* family-key format as `familyOf` (path-qualified `train.model=rf`), not the bare `fp.train.model` value the task currently specifies — otherwise the two selection surfaces diverge.
