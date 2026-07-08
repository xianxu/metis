# Boundary Review — metis#18 (milestone M1a-2)

| field | value |
|-------|-------|
| issue | 18 — experiment-design algebra M1a — three-phase shape + Sampler fold node (static samplers, per-fold pipeline, driver:single) |
| repo | metis |
| issue file | workshop/issues/000018-experiment-design-algebra.md |
| boundary | milestone M1a-2 |
| milestone | M1a-2 |
| window | fbc0acdb8ca40298940a879ca6cbbd926d4620fb..HEAD |
| command | sdlc milestone-close --issue 18 --milestone M1a-2 |
| reviewer | claude |
| timestamp | 2026-07-07T17:47:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have enough to render the verdict. Let me confirm the two edge behaviors I'll cite (negative-K panic, empty-results winner) by reading the exact lines once more — already visible in the diff. Writing up the review now.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The M1a-2 pure Sampler core is clean, correct, and genuinely zero-IO: the generic `Run` loop, the `Sampler[S,P,O,R]` interface, the two static Samplers (`FixedKFolds`, `GridConfigs`), `SingleDriver`, `Aggregate`, and `Winner` all match the Spec and the plan's Core-concepts table, `go build ./pkg/...` + `go vet` + `go test ./pkg/sampler/...` are green, and `cmd/metis` red is the expected/confined M1a-1 carry-over. The load-bearing type-composition proof (`TestRun_NestedComposition`, driver ⊃ sweeper ⊃ resample) exists and passes — the boundary-close criterion is met. What keeps this from a clean SHIP is one **ARCH-DRY** drift: the plan promised `GridConfigs` "moves `sweep.Grid`" / "generalizes `pkg/sweep.Sampler`", but the code *forked* it — `pkg/sweep` (its own `Sampler`/`Grid`/stop-predicates) still compiles live in `./pkg/...` beside the new package, and nothing in the plan schedules its removal. That's a non-blocking, transient-by-design duplication, but it will ossify unless the plan pins the cleanup — hence FIX-THEN-SHIP (the fixes are a plan revision + a scheduled removal, not code rework on the M1a-2 core).

### 1. Strengths
- **`pkg/sampler/run.go:11-24`** — the generic `Run` is exactly the "one loop drives every level" primitive the design promised; all IO stays in the injected `runPoint` closure (ARCH-PURE, textbook).
- **`TestRun_NestedComposition` (run_test.go:63-104)** — verifies the hard claim (the types monomorphize when nested three deep, `R=MeanSE`→sweeper `O`, `R=Winner`→driver `O`). This is the one test that had to exist here, and it's real (asserts winner model, mean, SE=0 for constant scores, seed, and the three fold-keys).
- **`Aggregate` (aggregate.go:26-52)** — correct sample-SD/√n with the n<2 guard, order-independent, sorted told-set; the test pins an *exact* SE value (`sqrt(0.001)/sqrt(5)`), not a re-assertion of the implementation.
- **Deterministic tie-break via strict comparison** (`betterMean`, configs.go:66-71) — keeping the earliest-in-`Expand`-order config on a mean-tie is the right, stable choice, and `TestGridConfigs_DeterministicTieBreak` pins it.
- **`FixedKFolds.Init` stays pure** (folds.go:32-38) — consumes the already-materialized `ctx.Partition` rather than doing IO, exactly as the plan's Task 7 note requires.

### 2. Critical findings
None.

### 3. Important findings

**(a) ARCH-DRY — two ask/tell Sampler abstractions now coexist; the plan said "move", the code forked.** `pkg/sweep/sweep.go:28-73` (`Sampler`, `Grid`, `NewGrid`, stop-predicates) is still live and compiles green in `./pkg/...`; its only importer is the red `cmd/metis/sweep.go`. The plan's Core-concepts row states `GridConfigs` is "new (**moves** `sweep.Grid`)" and the DRY rationale says it "**Generalizes** `pkg/sweep.Sampler`" — but `sampler.GridConfigs` is a parallel reimplementation and `pkg/sweep` was left intact. Deleting it *now* is coupled to the M1a-4 `cmd/metis` rewire (removing it today just trades the field-undefined red for an import-not-found red), so deferral is defensible — **but nothing in the plan schedules the removal** (Task 17 rewires `run.go`/`sweep.go` without mentioning `pkg/sweep`). *Fix:* add an explicit "remove `pkg/sweep` once `cmd/metis` is rewired" step to Task 17 (M1a-4), and note the transient duplication in `## Log`; also update `atlas/index.md:53-54` (which still calls `pkg/sweep` *the* sweep sampler) at that same removal. Non-blocking at this gate.

**(b) Docs gate — atlas has no entry for the new `pkg/sampler` surface.** A whole new package with novel terminology (the Sampler fold-node algebra: `Init/Ask/Tell/Done`, static-vs-adaptive line) landed with no `atlas/` change in-range. Per the Docs gate this is an Important finding. **Acknowledged/consistent:** `atlas/index.md:73` already forward-references "Full … Sampler-fold-node write-up at (M1a-5)", the plan schedules atlas at Task 21, and M1a-1 closed with `--no-atlas` for the same reason — so this is a documented, plan-consistent deferral, not a miss. Track it: the milestone-close will again need `--no-atlas`, and Task 21 must cover the three static Samplers + `Run` + `Winner`/`Aggregate`.

### 4. Minor findings
- **`run.go:16-23`** — an ill-behaved `Ask` returning `(nil-batch, done=false)` spins the loop forever; the "makes progress" contract is enforced only by the doc comment. No shipped Sampler triggers it, but this is the substrate future *adaptive* Samplers (#19/#23/racing) implement against — a one-line guard (`if len(batch)==0 { break }` or a panic on empty-non-done) converts a silent hang into a diagnosable failure. Harden the primitive before adaptive impls arrive.
- **`folds.go:33`** — `make([]FoldPoint, f.K)` panics for `K<0` and yields an empty→zero `MeanSE` for `K==0`. `K` flows from validated `resample.cv.k` (M1a-4), so it's a precondition, not a live bug — worth either a `k≥1` guard here or an explicit note that validation upstream guarantees it.
- **Dual seed source** — `Ctx.Seed` (sampler.go:20) exists as the run-scoped seed, yet `GridConfigs` carries its *own* `Seed` field (configs.go:14) that flows to `Winner.Seed`; `Init` ignores `ctx.Seed`. Two sources for one fact; consider sourcing `Winner`'s seed from `ctx.Seed` to avoid divergence in M1a-4 wiring.
- **`GridConfigs.Ask` (configs.go:31-35) ≈ `FixedKFolds.Ask` (folds.go:43-47)** — the "emit not-yet-told points once, done when all told" static-Ask is duplicated; a generic `staticAsk[P](points []P, told int) ([]P, bool)` would consolidate it (the `configState`/`foldState` shapes are parallel too). Small, but it's literally the "degenerate static Sampler" the design calls *one* concept.
- **`GridConfigs.Select` (configs.go:15)** stored but never read (M1a is argmax-mean only). Intentional #19 seam — fine, just flagging it's inert; selection-rule validation lives in `ValidateShape`, not here.

### 5. Test coverage notes
Strong where it counts: exact-value SE, order-independence (both `Aggregate` and `FixedKFolds.Tell`), tie-break, minimize/maximize direction, the nested type-composition proof, and `Run`'s Ask/Tell/Done call-counts (`countSampler`). Gaps, all low-severity:
- `GridConfigs.Done` empty-results path (`Winner{Seed: g.Seed}`, configs.go:56-58) is untested.
- `FixedKFolds{K:0}` and the `K<0` panic are untested.
- Minimize-direction tie-break is untested (symmetric to the tested maximize case — low risk).

None of these would have caught a *shipped* bug in the current code; add the empty/degenerate cases when the wiring in M1a-4 starts feeding real values.

### 6. Architectural notes for upcoming work
- **ARCH-DRY:** flag — the `pkg/sweep` coexistence above is the one drift; resolve by scheduled removal at M1a-4, not by keeping both.
- **ARCH-PURE:** pass — genuinely IO-free; all side-effects deferred to the injected `runPoint`; `FixedKFolds.Init` correctly consumes a pre-materialized partition ref rather than materializing. This is the model the rest of M1a should hold to.
- **ARCH-PURPOSE:** pass for this boundary — M1a-2's stated purpose is the *pure core* (static Samplers only; adaptive #19/#23 are explicitly later impls against the same node), and the diff delivers exactly that, including the `Winner` reconstructable run-keys (`FreeParams`+`Seed`+`FoldKeys`+`Score`) that M1a-5/#23 need. The full-issue shadow-sweep (every consumer derives) is an M1a-4/M1a-5 concern, not gameable here.

### 7. Plan revision recommendations
- **`workshop/plans/000018-…-m1a-plan.md`, Task 17 (M1a-4):** add a step "remove `pkg/sweep` (its `Sampler`/`Grid`/stop-predicates are superseded by `pkg/sampler`) once `cmd/metis/sweep.go` is rewired; update `atlas/index.md:53-54`." Add a `## Revisions` entry noting the Core-concepts "moves `sweep.Grid`" was realized as fork-now/remove-at-M1a-4 (transient duplication) rather than an in-place move at M1a-2, and why (the move is coupled to the `cmd/metis` rewire).
- No revision needed for the atlas deferral — `atlas/index.md:73` and Task 21 already record it; just ensure Task 21's scope explicitly enumerates `Run`/`Sampler`/`FixedKFolds`/`GridConfigs`/`SingleDriver`/`Aggregate`/`Winner`.
