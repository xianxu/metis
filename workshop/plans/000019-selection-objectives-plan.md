# Configurable Select Rule + Measured Complexity — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the sweeper's hard-wired `argmax-mean` with a configurable, per-family select rule that prefers the *simpler* near-winner, where "simpler" is a complexity **measured on the fitted model** (not guessed from hyperparameters).

**Architecture:** Two milestones. **M1** builds the pure select machinery — a tagged-union `objective.select` (mirroring `driver`), a pure `SelectConfigs` rule (group-by-family → band → min-complexity → tie-break by mean; cross-family argmax-mean), and the type-widening that lets a per-fold *complexity* number ride alongside the score up through `MeanSE`. Complexity is an *input* to the rule in M1 (unit-tested with hand-built stats; wired to `0` end-to-end). **M2** makes complexity *real*: each model class reports its fitted complexity (`rf` mean leaves, `logreg` coef count), `metis/train` emits it per fold, it flows through cache→ledger, and the acceptance counterfactual is verified over the ledger.

**Tech Stack:** Go 1.x (stdlib `testing`), CUE (`cue vet` drift guard), Python 3 (pytest, scikit-learn).

**ARCH notes:** the rule is a pure function in `pkg/sampler` reused by both selection surfaces (in-memory ship path + offline ledger/promote) — **ARCH-DRY** (one rule, two consumers) + **ARCH-PURE** (rule is IO-free, unit-tested with hand-built stats; the model-introspection + emission is the thin seam). **ARCH-PURPOSE**: the Done-when is *verified over the real ledger*, not asserted — the purpose is a working select lever that recovers the shallower regime, not just a compiling union.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `experiment.Select` | `pkg/experiment/shape.go` | modified (was `Select string`) |
| `sampler.ConfigStat` | `pkg/sampler/select.go` | new |
| `sampler.SweepResult` | `pkg/sampler/select.go` | new |
| `sampler.SelectConfigs` | `pkg/sampler/select.go` | new |
| `sampler.familyOf` | `pkg/sampler/select.go` | new |
| `sampler.FoldOutcome` | `pkg/sampler/aggregate.go` | new |
| `sampler.FoldScore` | `pkg/sampler/aggregate.go` | modified (+`Complexity`) |
| `sampler.MeanSE` | `pkg/sampler/aggregate.go` | modified (+`MeanComplexity`,`HasComplexity`) |
| `sampler.Winner` | `pkg/sampler/winner.go` | modified (+`Family`,`Complexity`) |
| `sampler.GridConfigs` | `pkg/sampler/configs.go` | modified (`Select` type; `Done`→`SweepResult`) |
| `metis.model.complexity` | `metis/model.py` | new |

- **`experiment.Select`** — the tagged-union select rule; exactly one of `ArgmaxMean|OneStdErr|PctLoss|MeanStd` non-nil (mirrors `Driver`'s `Single|CV`).
  - **Relationships:** 1:1 with `Objective` (replaces its `Select string`). Each variant is a pointer to a small param struct (`PctLoss{Tolerance float64}`, `MeanStd{Lambda float64}`, `ArgmaxMean{}`, `OneStdErr{}`).
  - **DRY rationale:** reuses the *exact* `Driver` idiom (optional pointer fields + a Go "exactly one set" count check in `ValidateShape`) — no new pattern; CUE stays optional-fields, not a disjunction.
  - **Future extensions:** a `kSE` multiplier on `OneStdErr`, or a `complexity-penalty` rule — a new pointer field + a new rule branch.

- **`sampler.ConfigStat`** — one config's reduced statistics the rule reasons over: `{FreeParams []shape.FreeParam; Point shape.Point; Family string; Mean, SE, MeanComplexity float64; HasComplexity bool}`.
  - **Relationships:** N `ConfigStat` per sweep; built by `GridConfigs.Done` (from `configResult`) *and* by the offline ledger path (from aggregated rows) — the shared input shape that lets one rule serve both surfaces.
  - **DRY rationale:** without it, the in-memory rule and the offline ledger rule would each re-derive "config → (mean, se, complexity, family)".

- **`sampler.SweepResult`** — the rule's output: `{PerFamily map[string]Winner; Ship Winner}`.
  - **Relationships:** replaces the single `Winner` as the sweeper `Sampler`'s `R` type (and the driver's `O`/`R`). `Ship` = cross-family argmax-mean over `PerFamily` values.
  - **Future extensions:** #22 (ensembling) blends `PerFamily`; #23 (nested-CV) estimates one-per-family — both read this map.

- **`sampler.SelectConfigs`** — the pure rule: `func SelectConfigs(rule experiment.Select, direction string, stats []ConfigStat) SweepResult`. Group by `Family`; within each family apply the rule's band (SE / % / none) → the *contention set*; **minimize `MeanComplexity` (ε-binned) → tie-break by better `Mean` → tie-break by first `FreeParams` order**; the cross-family `Ship` = argmax-`Mean` over the per-family winners.
  - **DRY rationale:** the single source of "which config wins"; both `GridConfigs.Done` and the offline `metis ledger select`/`promote` call it.
  - **Future extensions:** a complexity-penalty rule; adaptive samplers (#7) call it at each `tell`.

- **`sampler.familyOf`** — `func familyOf(p shape.Point, sweptPaths map[string]bool) string`. Reads the model-family label from `p.With`'s single-key-map bundling (`{label: sub}`) at each *swept* tagged-sum path; concatenates them into a stable family key (empty = one implicit family).
  - **DRY rationale:** the one place that knows "a tagged-sum branch = a family"; used only by the rule.

- **`metis.model.complexity`** — `def complexity(fitted, kind: str) -> float`: `rf` → `mean(t.tree_.n_leaves for t in fitted.estimators_)`; `logreg` → `int(fitted.coef_.size)` (non-zero under L2 = all). Raises for an unknown kind.
  - **DRY rationale:** the per-model-class introspection, colocated with `make_model`/`train`.
  - **Future extensions:** GBM (#21) → mean leaves/tree; NN → non-zero params. New `kind` → new branch.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `train` per-fold emission | `metis/steps/train.py` | modified | model fit → metrics.json |
| `runPipelineFold` | `cmd/metis/sweep.go` | modified | fold run → `FoldOutcome` |
| ledger complexity + family select | `pkg/ledger/ledger.go`, `cmd/metis/ledger_cmd.go` | modified | offline CSV leaderboard |

- **`train` per-fold emission** — the per-fold branch fits the model, scores it, computes `complexity(fitted, kind)`, and emits `{"fold_score": score, "complexity": cx}`.
  - **Injected into:** nothing pure — it's the thin seam that *produces* the complexity number the pure rule consumes. Tested with a real subprocess-style step test (in-memory arrays) + a `metis.model` unit test.
- **`runPipelineFold`** — returns `sampler.FoldOutcome{Score: run.Metrics["fold_score"], Complexity: run.Metrics["complexity"]}` (bare keys — namespacing to `train.*` happens only at ledger write).
  - **Injected into:** the innermost `sampler.Run` closure (the resample `runPoint`).
- **ledger complexity + family select** — `AggregateView` also reduces the complexity metric (mean); a new `metis ledger select --rule …` applies the pure `SelectConfigs` offline; `promote` gains `--family`.
  - **Injected into:** the offline verification of the acceptance counterfactual (Done-when).

---

## Chunk 1 — M1: Select rule + sampler evolution

**Boundary:** closes with `sdlc milestone-close --issue 19 --milestone M1` (fresh-eyes review over the M1 diff). Delivers the configurable rule end-to-end with complexity wired as `0` (rules that ignore complexity work fully; parsimony rules are unit-tested with hand-built complexity).

### Task 1: `experiment.Select` tagged union (Go struct + validation)

**Files:**
- Modify: `pkg/experiment/shape.go:54-58` (`Objective`), `:63-76` (mirror `Driver`), `:140-142` (validation)
- Test/migrate: `pkg/experiment/shape_test.go` — the struct change also forces: the in-source fixture `validShapeV2:43` (`select: argmax-mean` scalar → `select: {argmax-mean: {}}`), the string assertion `:67-68`, and the `"unsupported select"` mutator `:113` (sets `Select = "one-std-err"` — now a *valid* rule, so the assignment won't compile *and* the negative test is obsolete → replace with a zero-branch/two-branch invalid case).

- [ ] **Step 1 — Write failing tests** in `shape_test.go`: (a) a shape with `select: {pct-loss: {tolerance: 0.02}}` parses + validates; (b) `select: {}` (zero branches) → error "exactly one"; (c) `select: {pct-loss: {tolerance: 0.02}, argmax-mean: {}}` (two branches) → error; (d) `select: {pct-loss: {tolerance: 0}}` → error (tolerance must be > 0). Use `ParseShape` + `ValidateShape` on inline YAML (follow the existing `shape_test.go` idiom).

- [ ] **Step 2 — Run, verify fail:** `cd /Users/xianxu/workspace/metis && go test ./pkg/experiment/ -run TestSelect -v` → FAIL (Select is still a string).

- [ ] **Step 3 — Implement.** Replace `Objective.Select string` with `Select Select` and add the union (mirror `Driver` exactly):

```go
// Objective names the metric, direction, and the select rule (metis#19).
type Objective struct {
	Metric    string `yaml:"metric"`
	Direction string `yaml:"direction"`
	Select    Select `yaml:"select"`
}

// Select is the tagged-union select rule (metis#19) — exactly one branch non-nil,
// mirroring Driver's single|cv (optional pointer fields + a Go count check in
// ValidateShape; the param is bound to its branch as a sub-struct field).
type Select struct {
	ArgmaxMean *ArgmaxMean `yaml:"argmax-mean,omitempty"`
	OneStdErr  *OneStdErr  `yaml:"one-std-err,omitempty"`
	PctLoss    *PctLoss    `yaml:"pct-loss,omitempty"`
	MeanStd    *MeanStd    `yaml:"mean-std,omitempty"`
}

type ArgmaxMean struct{}
type OneStdErr struct{}                          // band = 1×SE, no params
type PctLoss struct{ Tolerance float64 `yaml:"tolerance"` } // %-width band
type MeanStd struct{ Lambda float64 `yaml:"lambda"` }       // mean − λ·std

// Kind returns the single set branch name (for engine dispatch); "" + false if not exactly one.
func (s Select) Kind() (string, bool) { /* count non-nil; return the one name */ }
```

  In `ValidateShape`, replace `shape.go:140-142` with an "exactly one set" check (copy the `Driver` count idiom at `shape.go:143-152`) and per-branch param checks (`PctLoss.Tolerance > 0`; `MeanStd.Lambda >= 0`).

- [ ] **Step 4 — Run, verify pass:** `go test ./pkg/experiment/ -run TestSelect -v` → PASS.

- [ ] **Step 5 — Commit:** `#19 M1: objective.select tagged union (mirrors driver)`.

### Task 2: CUE `#ExperimentShape.select` (drift guard)

**Files:** Modify `construct/vocabulary/experiment.cue:76-80` (the `objective` block). Test via the existing conformance test.

- [ ] **Step 1** — Change the CUE `select: string` to optional-field branches (NOT a disjunction — mirror `driver` at `:82-85`):

```cue
objective: {
	metric:    string
	direction: "maximize" | "minimize"
	select: {                       // exactly-one enforced in Go (like driver)
		"argmax-mean"?: {}
		"one-std-err"?: {}
		"pct-loss"?: {tolerance: float & >0}
		"mean-std"?: {lambda: float & >=0}
	}
}
```

- [ ] **Step 2** — Run the conformance/drift test: `go test ./pkg/experiment/ -run Conform -v` (it `cue vet`s the reshaped `testdata/experiment/titanic-baseline-shape.md`, which Task 3 migrates — expect this to stay red until Task 3, note it).
- [ ] **Step 3 — Commit:** `#19 M1: CUE select union (optional fields, mirrors driver)`.

### Task 3: Migrate shapes to the union

**Files (exact, per recon):**
- Modify `testdata/experiment/titanic-baseline-shape.md:34`
- Modify `kbench` (`/Users/xianxu/workspace/kbench/competition/titanic/pipelines/`): `titanic-sweep.md:51` (+ prose 79-82), `titanic-sweep-smoke.md:44` (+ prose 57-58)

- [ ] **Step 1** — `titanic-baseline-shape.md:34`: `select: argmax-mean` → `select: {argmax-mean: {}}` (keep this fixture on argmax-mean — it exercises the M1a-compatible path).
- [ ] **Step 2** — `titanic-sweep.md:51`: `select: argmax-mean` → `select: {pct-loss: {tolerance: 0.02}}` (the canonical rule); update the prose `argmax-mean` mentions at `:62` and `:79-82` to describe pct-loss + the two-level select.
- [ ] **Step 3** — `titanic-sweep-smoke.md:44`: → `select: {pct-loss: {tolerance: 0.02}}`; update prose 57-58.
- [ ] **Step 4 — Run** `go test ./pkg/experiment/ -run Conform -v` → PASS (fixture now conforms). Commit: `#19 M1: migrate shapes to select union`.

### Task 4: Widen the fold output to carry complexity

**Files:**
- Modify `pkg/sampler/aggregate.go` (`FoldScore:10-13`, `MeanSE:19-23`, `Aggregate:30-53`)
- Create/modify `pkg/sampler/folds.go` (`FoldOutcome`; `Tell:50-53`, `Done:56`, instantiation `:58`)
- Test: `pkg/sampler/aggregate_test.go`, `pkg/sampler/folds_test.go` (`:30,39,55,59` bare-float sites), **`pkg/sampler/run_test.go`** (`:94` `func(FoldPoint) float64` — the internal composition proof; a generic `O`-param change is a *signature* change that breaks whole-package compilation, so this MUST migrate here or `go test ./pkg/sampler/` stays red)

> **Module-red window (T4→T7):** widening the fold `O` type breaks `cmd/metis/sweep.go:124` (`func(f FoldPoint) float64`) and the driver chain; `go build ./...` is intentionally **red from Task 4 through Task 7**, restored green at Task 7 Step 4. Within `pkg/sampler` itself, each task keeps the *package* green by migrating its own test files (below).

- [ ] **Step 1 — Failing test** in `aggregate_test.go`: `TestAggregate_Complexity` — feed `[]FoldScore{{Addr:"a",Score:0.8,Complexity:16},{Addr:"b",Score:0.9,Complexity:14}}` → `MeanSE{Mean:0.85, MeanComplexity:15, HasComplexity:true, …}`. Also assert an all-`Complexity:0` input still gives a valid `MeanSE` with `MeanComplexity:0`.
- [ ] **Step 2 — Run, fail:** `go test ./pkg/sampler/ -run TestAggregate_Complexity -v`.
- [ ] **Step 3 — Implement:**
  - `FoldScore{Addr string; Score float64; Complexity float64}`.
  - `MeanSE{Mean, SE, MeanComplexity float64; HasComplexity bool; ToldSet []string}`.
  - `Aggregate` also means `Complexity` across folds (same n); set `HasComplexity` true when computed.
  - `FoldOutcome{Score, Complexity float64}` in folds.go; `FixedKFolds.Tell(s, p, out FoldOutcome)` → `FoldScore{Addr:p.Addr(), Score:out.Score, Complexity:out.Complexity}`; instantiation `var _ Sampler[foldState, FoldPoint, FoldOutcome, MeanSE] = FixedKFolds{}`.
- [ ] **Step 4 — Fix the internal test call sites so the *package* stays green.** `folds_test.go` (`:30,39,55,59` pass a bare `0.5`/`func(fp) float64`) → `FoldOutcome{Score:0.5}` and `func(fp FoldPoint) FoldOutcome`. `run_test.go:94` (`func(FoldPoint) float64`) → `func(FoldPoint) FoldOutcome`. Run `go test ./pkg/sampler/ -v` → PASS. (`go build ./...` remains red — the sweep.go closure lands in Task 7; that's the flagged window.)
- [ ] **Step 5 — Commit:** `#19 M1: widen fold output to {score, complexity}`.

### Task 5: The pure select rule (`SelectConfigs` + `familyOf`)

**Files:** Create `pkg/sampler/select.go`, `pkg/sampler/select_test.go`.

- [ ] **Step 1 — Failing tests** in `select_test.go` (hand-built `[]ConfigStat`, no IO):
  - `TestSelect_ArgmaxMean` — highest `Mean` wins; `Ship`==that config; `PerFamily` has one entry per family with each family's argmax-mean.
  - `TestSelect_PctLoss_TieBreaksToMean` — **the corner regression**: family `rf` with configs `{depth4,feat6: mean .834, cx 16}`, `{depth4,feat1: mean .827, cx 16}`, `{depth8,feat3: mean .844, cx 40}`, tolerance 0.02. Assert the winner is `{depth4,feat6}` (band floor .827 admits depth4s; both depth4s have cx 16 → tie on complexity → **mean tie-break picks feat6**), NOT the sparse `feat1` and NOT the deep `depth8`.
  - `TestSelect_PctLoss_BinnedComplexity` — with ε=0.10: `feat1` cx 15, `feat6` cx 16 (16 ≤ 15·1.10=16.5 → within ε) → tie → feat6 wins; then `feat1` cx 10 (16 > 10·1.10=11 → outside ε) → feat1 wins. (Documents the ε boundary; the arithmetic must match the pinned constant.)
  - `TestSelect_OneStdErr_BandTooTight` — SE .005 band excludes a config .006 below best → not selected (documents 1-SE tightness).
  - `TestSelect_MeanStd_UsesStd_NotComplexity` — `mean − λ·std` argmax; complexity ignored.
  - `TestSelect_CrossFamily_ArgmaxMean` — two families; `Ship` = argmax-mean over the two per-family winners.
  - `TestSelect_NoTaggedSum_OneImplicitFamily` — empty family key → single group.

- [ ] **Step 2 — Run, fail.**

- [ ] **Step 3 — Implement `select.go`:**

```go
package sampler

import (
	"sort"
	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/shape"
)

// complexityBinRelTol: two complexities within this relative tolerance are "equally
// simple" (the same band idea pct-loss applies to the score) — so the mean tie-break
// can recover a higher-CV config whose realized complexity is ~equal. 0.10 absorbs a
// small integer leaf-count difference (e.g. 15 vs 16 leaves = 6.7% ties). Plan-pinned;
// tuned in the M2 acceptance (Done-when). metis#19.
const complexityBinRelTol = 0.10

type ConfigStat struct {
	FreeParams     []shape.FreeParam
	Point          shape.Point
	Family         string
	Mean, SE       float64
	MeanComplexity float64
	HasComplexity  bool
	ToldSet        []string
}

type SweepResult struct {
	PerFamily map[string]Winner
	Ship      Winner
}

// SelectConfigs applies the rule: group by family; within a family the band sets the
// contention set, then minimize complexity (ε-binned) → tie-break higher mean →
// stable order; cross-family Ship = argmax-mean over the per-family winners. Pure.
func SelectConfigs(rule experiment.Select, direction string, stats []ConfigStat) SweepResult {
	// 1. group by Family
	// 2. per family: contention = band(rule) over the family's stats
	//    - argmax-mean: contention = {family argmax-mean}; winner = it
	//    - mean-std:    re-score mean − λ·std; winner = argmax; (contention = all)
	//    - one-std-err: contention = mean within 1×SE of family best; then parsimony
	//    - pct-loss:    contention = mean within tolerance of family best; then parsimony
	//    parsimony (one-std-err/pct-loss): minComplexity := min over contention;
	//      simplest := {c in contention : c.MeanComplexity <= minComplexity*(1+ε)};
	//      winner := argmax Mean over simplest (tie → first in stable order)
	// 3. Ship = argmax-mean over PerFamily winners (stable tie-break)
}
```

  `familyOf(p shape.Point, sweptTagPaths []string) string` — for each swept tagged-sum path (e.g. `train.model`), read `p.With[step][key]` (a single-key map `{label: sub}`), take the sole key; join `path=label` pairs sorted → the family key. (M1: derive `sweptTagPaths` from the stats' `FreeParams` — a path whose `Value` is a string AND whose `Point.With` at that path is a single-key map. Note the alternative — enrich `shape.Expand` to emit discriminants — as a candidate the reviewer raised; M1 uses the `With` read. `familyOf` reads a **2-level** `With[step][key]` — sufficient for `train.model`; a deeper-nested tagged sum is out of M1 scope, worth a one-line limitation comment.)
  **Band semantics:** `pct-loss.tolerance` is a **relative fraction** of the family-best mean — for `maximize`, floor = `best·(1−tolerance)` (0.02 = 2%); `one-std-err` floor = `best − 1·SE`. (Symmetric for `minimize`.)

- [ ] **Step 4 — Run, verify pass** (`go test ./pkg/sampler/ -run TestSelect -v`). The corner regression (`TestSelect_PctLoss_TieBreaksToMean`) is the load-bearing one.
- [ ] **Step 5 — Commit:** `#19 M1: pure SelectConfigs rule + familyOf (group-by-family, band, ε-binned parsimony)`.

### Task 6: `GridConfigs` consumes the rule; `Done`→`SweepResult`; `Winner`+family

**Files:** Modify `pkg/sampler/winner.go:11-16`, `pkg/sampler/configs.go:14-18,50-67,79`, `pkg/sampler/configs_test.go`, **`pkg/sampler/run_test.go`** (`:84` `GridConfigs{… Select:"argmax-mean"}` → the struct + `:97` `func(SinglePoint) Winner` → `SweepResult` prep — the driver-facing half lands in Task 7).

- [ ] **Step 1 — Failing test** in `configs_test.go`: reshape `TestGridConfigs_*` to expect `Done` returning `SweepResult` (`.Ship` is the old single-winner assertion; add a `.PerFamily` size check). Add `TestGridConfigs_PerFamily` driving `Run` with two families of fake `MeanSE`.
- [ ] **Step 2 — Run, fail.**
- [ ] **Step 3 — Implement:**
  - `Winner` gains `Family string` and `Complexity float64`.
  - `GridConfigs.Select string` → `Select experiment.Select`.
  - `GridConfigs.Done(s configState) SweepResult`: build `[]ConfigStat` from `s.results` (each `configResult{point, meanSE}` → `ConfigStat{Point, FreeParams: point.FreeParams, Family: familyOf(point,…), Mean: meanSE.Mean, SE: meanSE.SE, MeanComplexity: meanSE.MeanComplexity, HasComplexity: meanSE.HasComplexity, ToldSet: meanSE.ToldSet}`) → `SelectConfigs(g.Select, g.Direction, stats)`.
  - Instantiation: `var _ Sampler[configState, shape.Point, MeanSE, SweepResult] = GridConfigs{}`.
- [ ] **Step 4 — Run, pass.** Commit: `#19 M1: GridConfigs.Done → SweepResult via SelectConfigs`.

### Task 7: Thread `SweepResult` through the driver + sweep.go

**Files:** Modify `pkg/sampler/driver.go` (`SingleDriver` O/R type), `pkg/sampler/sampler.go:33` (stale doc comment "R=Winner is the driver's O" → `SweepResult`), **`pkg/sampler/run_test.go`** (`:99` `Run(ctx, SingleDriver{}, driverRun)` + `:101-121` `winner.Point/.Score` → `res.Ship.Point/.Score`), `pkg/sampler/driver_test.go`, `cmd/metis/sweep.go:119-128` (nested Run), `:165-181` (`shipWinner`), `:324-333` (`reportWinner`), `:40-46`/`:51-54` if needed.

- [ ] **Step 1** — `SingleDriver`: `Sampler[driverState, SinglePoint, SweepResult, SweepResult]` (pass-through); update `driver.go` Tell/Done + instantiation + `driver_test.go`.
- [ ] **Step 2** — `sweep.go:119-128`: the outer `sampler.Run(...SingleDriver...)` now yields `SweepResult`; the middle `GridConfigs{... Select: sh.Sweeper.Objective.Select}` (struct, not string). Bind `res := sampler.Run(...)`.
- [ ] **Step 3** — `shipWinner(res.Ship)`; `reportWinner(res)` prints the per-family leaderboard + the ship pick. Keep `runPipelineFold` returning a `FoldOutcome` (Task 8 sets Complexity; here return `FoldOutcome{Score: run.Metrics[foldMetric]}`, Complexity 0).
- [ ] **Step 4 — Run** `go build ./... && go test ./... ` (whole module). Fix any remaining `Winner`/`float64` call sites the compiler flags. Commit: `#19 M1: thread SweepResult through driver + sweep`.

### Task 8: M1 verification + milestone close

- [ ] **Step 1 — Drive it:** `metis run <a titanic-sweep fixture or the smoke shape>` against a tiny fixture (or the smoke shape with the warm cache if available) → confirm it runs green with `select: {pct-loss: {tolerance: 0.02}}` and reports a per-family leaderboard (complexity shows 0 — real values land in M2). Capture the output as evidence.
- [ ] **Step 2 — Whole-module green:** `go test ./... && (cd /Users/xianxu/workspace/metis && go vet ./...)`.
- [ ] **Step 3 — Atlas:** update `atlas/index.md` `pkg/sampler` bullet for `SelectConfigs`/`SweepResult`/`FoldOutcome` + the `objective.select` union.
- [ ] **Step 4 — Close M1:** `sdlc milestone-close --issue 19 --milestone M1` (fresh-eyes review over the M1 diff; fix Critical/Important; log the `Review-Verdict:`).

---

## Chunk 2 — M2: Measured complexity + acceptance

**Boundary:** closes with `sdlc close --issue 19 --milestone M2` (the final fresh-eyes review + the verified acceptance).

### Task 9: `metis.model.complexity` (per-model-class introspection)

**Files:** Modify `metis/model.py` (add near `train:57-61`); Test `tests/test_model.py`.

- [ ] **Step 1 — Failing tests** in `test_model.py`: `test_complexity_rf_mean_leaves` — fit rf on `_separable()`, assert `complexity(m,"rf") == mean(t.tree_.n_leaves for t in m.estimators_)` and that it does NOT scale with `n_estimators` (fit n=10 vs n=50, assert ~equal mean-leaves, |Δ| small). `test_complexity_logreg_is_coef_count` — assert `complexity(m,"logreg") == m.coef_.size`. `test_complexity_unknown_raises`.
- [ ] **Step 2 — Run, fail:** `cd /Users/xianxu/workspace/metis && uv run pytest tests/test_model.py -k complexity -v`.
- [ ] **Step 3 — Implement:**

```python
def complexity(fitted, kind: str) -> float:
    """Realized complexity of a FITTED model (metis#19). rf → mean leaves/tree
    (mean, not total, so it's n_estimators-neutral per Breiman's LLN); logreg →
    coefficient count (L2 zeroes nothing → all non-zero = feature count)."""
    if kind == "rf":
        leaves = [t.tree_.n_leaves for t in fitted.estimators_]
        return float(sum(leaves) / len(leaves))
    if kind == "logreg":
        return float(fitted.coef_.size)
    raise ValueError(f"complexity: unknown model kind {kind!r}")
```

- [ ] **Step 4 — Run, pass.** Commit: `#19 M2: metis.model.complexity (rf mean leaves, logreg coef count)`.

### Task 10: `metis/train` emits `complexity` per fold

**Files:** Modify `metis/steps/train.py:50-56` (per-fold branch); Test `tests/test_steps.py` (or a new `test_train_step.py`).

- [ ] **Step 1 — Failing test:** exercise the per-fold branch (fit over in-memory folds) and assert `metrics.json` has BOTH `fold_score` and `complexity` (> 0). Follow the existing `tests/test_steps.py` step-invocation idiom.
- [ ] **Step 2 — Run, fail.**
- [ ] **Step 3 — Implement:** the per-fold branch must fit once and reuse the estimator for both score and complexity (don't double-fit). Refactor: instead of `fold_score(...)` (which discards the estimator), fit `model = train(Xa[trn], ya[trn], kind, seed, params)`, `score = accuracy_score(ya[val], predict(model, Xa[val]))`, `cx = complexity(model, kind)`, then `io.write_metrics(ctx, {"fold_score": score, "complexity": cx})`. Keep `metis.model.fold_score` for `cv_score`'s reuse, OR add `fold_score_and_model(...) -> (float, estimator)` and have `fold_score` delegate — pick the smaller diff; note in Log.
- [ ] **Step 4 — Run, pass.** Commit: `#19 M2: train emits fold_score + complexity per fold`.

### Task 11: `runPipelineFold` carries real complexity

**Files:** Modify `cmd/metis/sweep.go:26` (add `const foldComplexityMetric = "complexity"`), `:188-224` (`runPipelineFold` return).

- [ ] **Step 1 — Failing test:** an e2e/integration test (follow `cmd/metis` test idiom) that runs a 1-config 2-fold sweep over an in-repo fixture step and asserts the resulting `MeanSE.MeanComplexity > 0` (i.e. complexity threaded fold→config). If no such harness exists cheaply, assert via `metis run` output in Task 13 and mark this step as covered there (note it).
- [ ] **Step 2 — Implement:** `runPipelineFold` returns `sampler.FoldOutcome{Score: run.Metrics[foldMetric], Complexity: run.Metrics[foldComplexityMetric]}`.
- [ ] **Step 3 — Run** `go test ./... `. Commit: `#19 M2: runPipelineFold returns measured complexity`.

### Task 12: Ledger carries complexity + offline family select

**Files:** Modify `pkg/ledger/ledger.go` (`AggregateView:192-240`), `cmd/metis/ledger_cmd.go` (`runPromote:184-257`, add `--family`; a new `metis ledger select`); `cmd/metis/main.go` dispatch.

- [ ] **Step 1 — Failing tests:** `pkg/ledger/ledger_test.go` — `AggregateView` over rows carrying `train.fold_score` AND `train.complexity` emits both the score `(mean, se, n)` and a `train.complexity` mean column. `cmd/metis` — `metis ledger select --shape <sweep> --rule pct-loss` over a small fixture ledger prints the per-family winners + ship pick, applying `sampler.SelectConfigs` (DRY reuse).
- [ ] **Step 2 — Implement:**
  - `AggregateView(l, metric)` → also mean the complexity metric if present (or generalize to aggregate ALL metric columns — smaller, more general: mean every `metric.*` column, keep `.se`/`.n` for the objective metric). Prefer the general form.
  - Build `[]sampler.ConfigStat` from the aggregated rows. **CRITICAL (M1-review finding): the offline `Family` string MUST match `familyOf`'s path-qualified format exactly — `"train.model=rf"`, NOT the bare `"rf"` from `fp.train.model`.** `familyOf` (as-built) emits `<step>.<key>=<label>` joined+sorted; if the offline path sets a bare `"rf"`, the two `SelectConfigs` consumers produce different keys for the same config and the "one rule, two surfaces, identical result" DRY property breaks silently. So reconstruct enough of the `FreeParam`/`With` to call `familyOf` on the ledger row (or replicate its exact `<step>.<key>=<label>` format from the `fp.*` columns). Then call `sampler.SelectConfigs` (which already takes pre-built `[]ConfigStat` — the offline path just supplies `Family` in the matching format).
  - `metis ledger select` command (new): reads shape + ledger, prints the two-level result. `promote --family <name>` promotes that family's robust winner (reuse the same rule).
- [ ] **Step 3 — Run, pass.** Commit: `#19 M2: ledger complexity column + offline family select (reuses SelectConfigs)`.

### Task 13: The guard (parsimony rule + unmodeled family → hard error)

**Files:** the guard runs **post-fold, pre-selection** (a rule's `HasComplexity` is only knowable after folds run — the metric is emitted, not statically declared; there's no pre-sweep registry to check). Place it at the top of `SelectConfigs` (or a check just before it in `cmd/metis/sweep.go`).

- [ ] **Step 1 — Failing test:** a shape with a parsimony rule (`pct-loss`) + a swept family whose model class has no `complexity()` (simulate with a fake kind, or a `HasComplexity:false` stat) → hard error naming the family + the fix; `argmax-mean`/`mean-std` with the same → no error.
- [ ] **Step 2 — Implement:** before selection, if the rule is `one-std-err`/`pct-loss` and ANY swept family's stats have `HasComplexity==false` → error "family %q under a parsimony rule reports no complexity; add metis.model.complexity(kind) or use argmax-mean/mean-std". (Check ALL swept families, not just the winner — per the v2 review.)
- [ ] **Step 3 — Run, pass.** Commit: `#19 M2: guard — parsimony rule requires complexity for every swept family`.

### Task 14: Acceptance counterfactual (verify, don't assert)

**Files:** the RUNBOOK + a Log entry; uses the warm `.metis-cache` under `kbench/competition/titanic/pipelines/`.

- [ ] **Step 1 — Re-fit with complexity:** re-run `metis run titanic-sweep.md` over the warm data cache (get-data HITs; models re-fit cheaply, no creds) so the ledger now carries `train.complexity`. (If the cache invalidates on the train-step code change, that's expected — it re-fits the 42 configs × 5 folds locally; still no creds.)
- [ ] **Step 2 — Run each rule offline:** `metis ledger select --shape titanic-sweep.md --rule argmax-mean|one-std-err|pct-loss|mean-std` → a **per-rule table** of the selected config (family, depth, features, mean, complexity). **Record the actual picks.**
- [ ] **Step 3 — Verify the claim:** confirm `pct-loss` selects a **shallower rf config than argmax-mean's md=8**. Report which config (incl. feature count). **If** the pick is the sparse corner (nfeat=1) rather than the higher-CV config, tune `complexityBinRelTol` (Task 5) or document the outcome + choose a fallback — do NOT assert recovery that didn't happen (ARCH-PURPOSE; the v1 corner lesson).
- [ ] **Step 4 — Fix the RUNBOOK:** `kbench .../RUNBOOK-sweep.md:37` `--sort train.cv_score` → the `metis ledger select`/`--sort train.fold_score` v2 form; sweep the v1-era `cv_score`/`cv-split` refs (`:30` cv-split, `:32,39,49,81` cv_score). Commit (kbench): `#19 M2: RUNBOOK v2 — fold_score + select rule`.
- [ ] **Step 5 — Log** the per-rule table + the verified pick in the issue `## Log`.

### Task 15: Close

- [ ] **Step 1 — Whole-module green** (metis + kbench): `go test ./...`, `go vet ./...`, `uv run pytest`.
- [ ] **Step 2 — Atlas + docs:** `atlas/index.md` (measured complexity + the two selection surfaces); the pensive/spec already record the design.
- [ ] **Step 3 — Actual:** `sdlc actual --issue 19` (measured, not typed).
- [ ] **Step 4 — Close:** `sdlc close --issue 19 --milestone M2 --verified '<per-rule acceptance table + whole-module green + the pct-loss-picks-shallower-than-md=8 evidence>'`.
- [ ] **Step 5 — Brain project row:** tick metis#19 (M2) in `data/project/metis-v2-experiment-algebra.md`.

---

## Non-goals (carried from the spec)
Cross-family complexity comparison (unsound — non-parametric RF; #23 estimates); nested-CV `driver:cv` (#23); adaptive samplers (#7); static/pre-training complexity; the #4 collocated-manifest catalog (de-entangled). `one-std-err` ships + is unit-tested only as the labeled Breiman contrast (documented too-tight here) — don't let it grow.

## Revisions

### 2026-07-09 — reconcile Core-concepts table with what M1 shipped (M1-review finding)
The M1 code is *cleaner* than the drafted table; recording the as-built shapes so an M2 author trusts a current map (no behavior change):
- **`sampler.FoldOutcome`** lives in `pkg/sampler/folds.go` (table said `aggregate.go`). Shape: `{Score, Complexity float64; HasComplexity bool}`.
- **`sampler.Winner`** gained **`Family string` only** — complexity is carried in `Winner.Score.MeanComplexity` (no separate `Winner.Complexity` field; ARCH-DRY).
- **`sampler.ConfigStat`** is `{Point shape.Point; Family string; Score MeanSE}` (embeds `MeanSE`), not the flattened field list the plan drafted. `MeanSE` gained `MeanComplexity float64` + `HasComplexity bool`.
- **`SelectConfigs`** as-built: `SelectConfigs(rule experiment.Select, direction string, seed int, stats []ConfigStat) SweepResult` (has a `seed` param). **`familyOf(p shape.Point) string`** dropped the drafted `sweptTagPaths` param (derives per-point from `FreeParams`+`With`); it emits **path-qualified** keys `"train.model=rf"` (sorted, comma-joined) — see the Task 12 amendment (offline path must match this format).
- **Guard placement** (Task 13): post-fold, pre-selection (not pre-sweep) — `HasComplexity` is only known after folds run.
- `complexityBinRelTol = 0.10` lives in `pkg/sampler/select.go`. If retuned in the M2 acceptance, update `TestSelect_PctLoss_BinnedComplexity` (its arithmetic is pinned to 0.10).
