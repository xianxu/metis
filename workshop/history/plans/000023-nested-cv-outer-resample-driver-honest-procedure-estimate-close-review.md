# Boundary Review — metis#23 (whole-issue close)

| field | value |
|-------|-------|
| issue | 23 — nested-CV outer resample driver — honest procedure estimate |
| repo | metis |
| issue file | workshop/issues/000023-nested-cv-outer-resample-driver-honest-procedure-estimate.md |
| boundary | whole-issue close |
| milestone | — |
| window | ec4ac80ecfa246b35d35c9d88b47014218f9d662..HEAD |
| command | sdlc close --issue 23 |
| reviewer | claude |
| timestamp | 2026-07-12T18:17:50-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I now have a complete picture. All Go and Python tests pass (Go `cmd/metis` 15.4s, `sampler`, `experiment`; Python 65 passed), the real-chain confinement test genuinely runs (not skipped), the honesty-critical partition-reproduction is sound, and both prior boundary-review findings (I1 GuardComplexity, I2 stale restatements) are confirmed fixed in the code.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The whole-issue boundary for metis#23 delivers a genuinely correct nested-CV outer driver: a pure `CVDriver` Sampler over the unchanged `Run` loop, per-outer-fold sealed selection (L1 subset dirs + L2 `METIS_READ_ROOT` chokepoint) followed by an honest refit-and-score whose partition-reproduction I verified end-to-end (outer-split and the score cv-split both call `cv_folds(full_train, k=outerK, seed, stratify)` → identical positional folds → the refit trains on exactly `analysis_i`'s rows and scores the physically-absent assessment fold `i` — a true held-out estimate). Both M1/M2 boundary findings are closed in code (the nested `GuardComplexity` guard at `sweep.go:364` with a real regression test; the stale "rejected at validate" restatements corrected). Nothing blocks the boundary. The one thing keeping this from a clean SHIP is a test-coverage gap: the confinement `readRoot` threading through the *actual* `driver:cv` orchestration is unasserted, so a regression that silently unsealed the inner sweep would leave every test green — mitigated (not eliminated) by the L1 structural backstop.

### 1. Strengths
- **The honesty-critical partition reproduction is correct and subtle** (`sweep.go:369-372`). Both the outer-split preamble and the score-run cv-split reduce to the same `cv_folds(full train, k=outerK, seed, Driver.CV.Stratify)`, so the refit's `fold != i` rows are byte-identical to `analysis_i` and the held fold `i` is exactly the sealed assessment. The c4424d7 "use OUTER k+stratify" insight is load-bearing and correctly applied.
- **The seal is scoped, not blanket.** `exp_path` (`io.py:130`) is the single chokepoint; `dataset_dir`'s upstream branch (`io.py:157-160`) bypasses it, so intra-fold `features→train` handoffs stay unconfined. The C1 regression has an explicit test (`tests/test_io_confinement.py:73`).
- **The parsimony-guard fix (M2 I1) is real, not cosmetic.** `runOuterFold` guards `configStatsOf(pass.configs)` per outer fold (`sweep.go:364`), and `TestNestedCV_ParsimonyGuardOnMissingComplexity` drives `pct-loss` + `noComplexity` fake through the whole nested path and asserts a loud error — matching the flat path's behavior exactly.
- **ARCH-DRY consolidation is clean** (`runSweeper` `sweep.go:107`, `cvSplitStep` `:516`, `configStatsOf` `:226`): flat and sealed/score paths share one select/reduce/guard implementation, parameterized on base-dir/k/stratify/readRoot.
- **Fatal-fold semantics preserved** through the extraction: `firstErr` short-circuit in `runNestedCV` (`sweep.go:294-304`) + `pass.err` in `runPipelineFold` keep a partial resample from being reported as an honest estimate.
- **`CVDriver` is a true PURE entity** — `driver_test.go` unit-tests it with zero IO, including the `k=0` done-immediately edge that the empty-batch panic guard would otherwise hit.

### 2. Critical findings
None.

### 3. Important findings

**I-A — The confinement `readRoot` wiring through the `driver:cv` orchestration is unasserted (test coverage).** `cmd/metis/sweep.go:353` (`pass.readRoot = analysisAbs`) → `sweep.go:423` (`pointOpts.readRoot = p.readRoot`) → `run.go:142` (`execStep{readRoot: o.readRoot}`) is the chain that seals the inner sweep. Only the *last* link is tested, and only in isolation (`exec_test.go` constructs `execStep` directly). Every `driver:cv` e2e uses `foldFakeExec`, which *replaces* `execStep` and never sees `readRoot` (`nestedcv_e2e_test.go:24`). So a regression that dropped `sweep.go:423`, or set `pass.readRoot = ""`, would run every sealed sweep **unconfined** with all tests green — silently removing the L2 seal that is the issue's headline safety property. Non-blocking because L1 (assessment rows physically absent from `analysis_i`) still prevents leakage as a backstop; the loss is the observability/missed-repoint safety net, invisibly.
*Fix sketch:* the clean closure is the deferred real-data `driver:cv` e2e (needs the toy data-step the Log notes metis lacks). Cheaper interim: a focused test asserting `runOuterFold` builds the sealed `sweepPass` with a non-empty `readRoot` equal to `analysis_i`'s abs path (and preamble/score with `""`) — e.g. expose the constructed opts through the exec seam so a spy fake can capture `o.readRoot` per run. At minimum, record this as a stated coverage deferral (see §7) rather than a silent gap.

### 4. Minor findings
- **`reportEstimate` (`sweep.go:391`) prints the honest mean±SE alone, not next to the inner cv-max** — plan Task 2.6 wanted the honesty gap made visible side-by-side. Low value as-is (there's no single inner cv-max across k independent sealed sweeps), but it means the Done-when's "distinct from (and lower than) the inflated inner cv-max" is only observable via the operator-gated real run, never in the tool output.
- **`nested` driver alias not expressible.** Spec/Done-when say "`driver: cv`/`nested`"; only `cv` parses (`Driver` has no `nested` yaml tag, and `KnownFields(true)` would reject it loudly). Documented as a future naming alias in the plan — a cosmetic synonym for the same mode, legitimately deferred, but Done-when literally lists it.
- **ARCH-DRY (minor):** `CVDriver.Init/Ask` (`driver.go:68-84`) structurally duplicate `FixedKFolds.Init/Ask` (`folds.go:42-56`); the point/outcome types differ so a generic unification isn't worth forcing. Noted, not actionable now.
- **`outer-split` via `metis.trace`/uv is not exercised through the real chain** — `test_outer_split.py` calls `outer_split.main()` directly; only `cv-split` is proven through the real wrapper (`exec_test.go`). The wrapper is byte-parallel to its siblings and always-unconfined, so risk is low.

### 5. Test coverage notes
Coverage is strong on the mechanism: within-root predicate edges (child/root/sibling/prefix-collision), env on/off, both halves of the `exp_path` chokepoint, the handoff bypass, subset disjoint+covering + positional row correctness against the source, `CVDriver` incl. `k=0`, the parsimony guard on the nested path, no-ship, dry-run cost, and a non-fakeable real-uv confinement test. The single meaningful gap is I-A (the seal's orchestration wiring). Everything else that could ship a bug of this diff's class is pinned.

### 6. Architectural notes for upcoming work
- **ARCH-DRY — PASS** (with the CVDriver/FixedKFolds boilerplate as an accepted minor). The shared `runSweeper`/`cvSplitStep`/`configStatsOf`/`Aggregate` reuse is the right shape; keep new guards free over `[]configScore`.
- **ARCH-PURE — PASS.** Pure Sampler core (`CVDriver`, `within_root`) vs thin IO shell (`runNestedCV`/`runOuterFold`/`materializeOuterAnalysis`, `assert_within_read_root`); `outer_split.py` is a clean io→pure→io entrypoint.
- **ARCH-PURPOSE — PASS.** Shadow-sweep of the driver-mode source: the shape parser, `runShapeSweep` dispatch, `experiment-shape.md`, and `experiment.cue` all now derive from the accept-with-`k>=2` reality; no stale restatement remains (the M2 I2 consumers are corrected). The operator-gated honesty-gap magnitude and the `nested` alias are separable extensions, not the deferred point — the honest procedure estimate itself ships.
- **For metis#20 (stateful features):** the BOUND ASSUMPTION (score-over-full-data honest only while features are stateless, because scoring is expressed *as a fold*) is correctly deferred and documented in the plan/atlas. When #20 lands, re-verify the "honest" claim doesn't start lying and carry the link forward.
- **Keep the confinement at `exp_path` only** — resist adding asserts to `load_dataset`/`dataset_dir`-upstream (that reintroduces the C1 handoff crash).

### 7. Plan revision recommendations
The plan already folds in both boundary reviews and otherwise matches the code. One entry worth adding to `workshop/plans/000023-nested-cv-outer-resample-driver-plan.md` `## Revisions`:
- **Confinement-wiring e2e remains deferred (coverage gap, I-A).** Task 2.5 item 4 called for the honest e2e to prove "confinement holds … inject a fake step that reads outside `analysis_i` … the injected leak must go through the *real* `exp_path` chokepoint." As shipped, that sub-assertion is *not* delivered: the fake exec bypasses `metis.io`, so no test drives a leak through the real chokepoint *within the driver:cv orchestration* — the real-chain proof is isolated to `execStep`. State this as a known deferral (blocked on the missing toy data-step for a real-data `driver:cv` e2e), so the L2 seal's end-to-end verification is a recorded decision, not an implied-but-absent guarantee.
