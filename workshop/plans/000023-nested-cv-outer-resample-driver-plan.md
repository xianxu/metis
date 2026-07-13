# Nested-CV Outer Resample Driver — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `driver: cv` (nested-CV) — an outer resample that wraps the black-box sweeper to produce the **honest procedure estimate** (the workbench telling the truth about what its tune-then-fit procedure actually generalizes to), with outer-assessment structurally sealed from selection and a trace-enforced read-confinement guaranteeing it.

**Architecture:** `driver(sweeper[inner-cv](pipeline))` — isomorphic to mlr3's `resample(AutoTuner(resample(learner)))`. For each outer fold: run the sweeper on a **physically-subset outer-analysis dir** (assessment bytes absent) → a winner; refit the winner on outer-analysis and score on the sealed outer-assessment; aggregate the k outer scores → `mean±SE`, the honest estimate. Result-dependent (folds may pick different winners); produces **no shippable winner**. Two sealing layers: **L1 structural** (outer-analysis subset dirs) makes leakage unrepresentable during selection; **L2 chokepoint** (`METIS_READ_ROOT` asserted in `metis/io.py`) makes it observable/verified and covers parquet. Syscall-level airtightness (rogue non-`metis.io` reads, parquet-via-C) is documented-and-deferred.

**Tech Stack:** Go (`pkg/sampler` pure Sampler algebra; `cmd/metis` IO shell), Python (`metis/` pure model core + `metis/steps/*` + `metis/io.py` + `metis/trace.py`), sklearn, the `Sampler[S,P,O,R]`/`Run` loop (metis#18), `sampler.Aggregate` (metis#18 read-time reduction).

---

## Scope & milestones

Two review boundaries. M1 is independently valuable — the **sealing spine** #20 (leakage-safe features) and kbench#8 (ticket-group survival) also build on. M2 is the nested-CV loop proper, proven by an honest e2e.

- **M1 — structural outer-partition + trace-enforced read-confinement (L1+L2).** Materialize outer-analysis subset dirs; inject + enforce `METIS_READ_ROOT` at the `metis.io` data chokepoint. Tested in isolation (subset dirs correct; an out-of-root data read is caught + named; in-root reads pass). No driver yet.
- **M2 — `CVDriver` nested loop + refit-and-score + honest e2e + cost + reporting + atlas.** The pure `CVDriver` Sampler; extract the sweeper-as-callable (per-outer-fold `Ctx`/partition); refit-the-winner-and-score-on-assessment (reuse `buildFoldExperiment` over the full data with an outer mask — post-selection, no leakage); aggregate → `MeanSE`; delete the `ValidateShape` stub-rejection; ~5× cost surfaced; honest e2e (estimate < inner cv-max, k× cost, no ship, confinement enforced + **fails on an injected leak**); atlas.

---

## Core concepts

### The central design insight (why the split is clean) — and its bound assumption

Only the **selection** phase must be sealed: the sweeper's inner-CV must never see outer-assessment. So the structural subset dir + confinement apply **only to the sweeper runs**. The **scoring** phase (refit the already-selected winner on outer-analysis, evaluate on outer-assessment) happens *after* selection — evaluating a config that was chosen without seeing assessment is an honest held-out eval, so it reuses the existing per-fold machinery: the scoring run is **expressed as a fold** over the full dataset with an *outer* folds array (`fold_fit` trains on `outer_fold != i`, scores on `outer_fold == i`). This means **the only new data materialization is the analysis subset dirs**, and the refit-and-score needs **no new Python primitive**.

**BOUND ASSUMPTION (do not let this claim lie — cf. `lessons.md` "artifacts lie by aspiration"):** the scoring-over-full-data-with-an-outer-mask is honest **iff the pipeline's feature steps are fold-safe** (they fit only on the fold's analysis rows). Today's titanic features (`title`, `family`) are **row-wise stateless** → fitting on full data ≡ fitting on analysis, so it is honest **now**. But a **stateful** feature (target encoding, global imputation — exactly metis#20, the issue this spine serves) fitted on full data before the late `train` mask would silently leak outer-assessment into the transform and re-inflate the "honest" estimate. This is why the scoring MUST be expressed **as a fold** (the outer split is a fold; every pipeline step receives the `_fold` context): when #20 makes features fit per-fold, the outer scoring **inherits** that fold-safety automatically. Do **not** "optimize" the scoring to fit features on full data outside a fold — that would silently break #20-compatibility. The plan states this assumption explicitly (Task 2.3) and the atlas records it; airtight enforcement (fitting the *whole* pipeline on the analysis subset dir) is #20's job, not #23's.

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `CVDriver` | `pkg/sampler/driver.go` | new |
| `OuterFoldPoint` | `pkg/sampler/driver.go` | new |
| `cvDriverState` | `pkg/sampler/driver.go` | new |
| `outerFolds` (assignment) | `metis/split.py` (reuse `cv_folds`) | reused |
| `withinRoot` (confinement predicate) | `metis/io.py` | new |
| `Aggregate` → `MeanSE` | `pkg/sampler/aggregate.go` | reused |
| `Winner` (reconstructable run-key) | `pkg/sampler/winner.go` | reused |

- **CVDriver** — the outer resample Sampler: `Sampler[cvDriverState, OuterFoldPoint, float64, MeanSE]`. Mirrors `SingleDriver` (the trivial pass-through) but emits k outer folds and aggregates their scores. Replaces the seam `driver.go`'s own doc names ("an adaptive-nesting Sampler that scores the winner on each sealed outer fold").
  - **Relationships:** 1 per shape run with `driver:cv`. Composes over the sweeper exactly as `SingleDriver` does — `runPoint(outerFold) = Run(sweeper-on-analysis_i) → Winner`, then refit-and-score. `O = float64` (one outer score), `R = MeanSE` (the honest estimate). Note the asymmetry vs `SingleDriver` (whose `O = R = SweepResult`): here `runPoint` is `Run(sweeper) → Winner` *composed with* refit-and-score.
  - **DRY rationale:** A new Sampler impl against the **unchanged** `Run` loop (`run.go:13-31`) — the pensive's promise ("#23 is a new Sampler impl, no engine change"). Reuses `sampler.Aggregate` verbatim for the outer `MeanSE`.
  - **Future extensions:** `driver:nested` naming alias; repeated-CV (multiple outer seeds); the `Done` could also emit per-outer-fold selected configs (for a "how stable is selection" report).
- **OuterFoldPoint** — `{Idx int}`, the outer fold index (mirror `FoldPoint`, `folds.go`). One `Run.Ask` batch of k of them.
- **cvDriverState** — accumulator: `{ctx Ctx; scores []FoldScore}`. `Tell` appends each outer `float64` as a `FoldScore`; `Done` = `Aggregate(scores)`.
- **withinRoot** — pure predicate `withinRoot(abs_path, root) -> bool` (`os.path.abspath(path).startswith(abspath(root)+sep)`), plus the loud raiser. Mirrors the existing `METIS_RUN_DIR` exclusion shape in `trace.py:123-125`. Unit-tested with no IO.
  - **DRY rationale:** First occurrence; the same predicate serves #20/kbench#8's within-analysis-root feature checks later.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `outer-split` step | `metis/steps/outer_split.py` + `steps/metis/outer-split` wrapper | new | filesystem (subset dirs) |
| `METIS_READ_ROOT` injection | `cmd/metis/exec.go:59-65` | modified | subprocess env |
| chokepoint assertion | `metis/io.py` (`exp_path` **only** — C1; NOT `load_dataset`/`dataset_dir`) | modified | data-path resolution |
| `read_root` in `StepContext` | `metis/io.py:76-94` | modified | env contract |
| sweeper-as-callable | `cmd/metis/sweep.go` (extract from `:131-140`) | modified | the inlined sweeper Run |
| driver dispatch | `cmd/metis/sweep.go:88-169` (`runShapeSweep`) | modified | outer loop composition |
| `ValidateShape` stub-delete | `pkg/experiment/shape.go:201-203` | modified | the #23 gate |
| cost surfacing | `cmd/metis/sweep.go:105-111, 123` | modified | dry-run + run header |

- **outer-split step** (`outer_split.py`) — reads the base dataset, computes a k-fold outer assignment (`split.py:cv_folds`), and writes k `analysis_i/` subset dataset dirs (train rows where `outer_fold != i`, + `schema.json`), plus an `outer_folds.json` (positional assignment, for the scoring mask). **Only analysis dirs are materialized** (assessment is reached via the mask at score time; see insight above). **M2 note (from M1 review):** when M2 runs this step *above* the driver, it must leave `METIS_READ_ROOT` **unset** for it — the step legitimately reads the full dataset, and `exp_path` would raise if confined.
  - **Injected into:** the `CVDriver`'s per-outer-fold IO (via the driver's `runPoint` closure in `sweep.go`), which points the sweeper's `METIS_EXP_DIR`/base-dataset at `analysis_i/`.
  - **Future extensions:** stratified outer folds (already carried by `CVDriver.Stratify`); grouped outer folds (leave-one-group-out) for kbench#8's ticket groups.
- **chokepoint assertion** — `metis/io.py` is the single place dataset paths resolve. When `METIS_READ_ROOT` is set, every resolved data path is asserted under it; violation → `RuntimeError` naming the offending file (mirroring #28's "loud error naming the file" ergonomics). Covers parquet (assertion is at path-resolution, not the audit hook).
  - **Injected into:** every step run inside a sealed sweeper sub-run. Absent (`METIS_READ_ROOT` unset) for the outer scoring run and the flat `driver:single` path → no behavior change there.
- **sweeper-as-callable** — extract "run `GridConfigs` over (a given base-dataset dir, a fresh inner partition ref, a read-root) → `SweepResult`" out of the inlined closure at `sweep.go:131-140` into a method `runSweeper(baseDir, readRoot) SweepResult`, so `CVDriver` can call it k times. Today it closes over the single shared `ss.partRef`/`ss.man`/`ss.configs`; the extraction parameterizes base-dir + partition + read-root and scopes each outer fold's manifest/ledger.
  - **Test surface:** an e2e fake `StepExecutor` (mirror `soundFoldExec` in `shipe2e_test.go:23-58`) drives the whole nested loop with no subprocess; a `driver_test.go` unit fake drives `CVDriver` alone.

**Deferred (documented, not built):** syscall-level read tracing (catches rogue non-`metis.io` opens and parquet-via-C). The two-layer design confines every read through the sanctioned `metis.io` path and makes leakage unrepresentable structurally; the residual gap is a well-behaved-step assumption, filed as follow-up.

---

## Chunk 1: M1 — structural outer-partition + read-confinement

**Milestone tag `M1`** — closes with its own `sdlc milestone-close` (fresh-eyes review + `Review-Verdict` trailer).

### Task 1.1: `withinRoot` confinement predicate (pure)

**Files:**
- Modify: `metis/io.py` (add `withinRoot` + `_assert_within_read_root`)
- Test: `tests/test_io_confinement.py` (new)

- [ ] **Step 1: Write the failing test**

```python
# tests/test_io_confinement.py
import os
import pytest
from metis.io import within_root, assert_within_read_root

def test_within_root_true_for_child():
    assert within_root("/data/analysis_0/train.parquet", "/data/analysis_0")

def test_within_root_false_for_sibling():
    # the outer-assessment sibling must be OUTSIDE the analysis root
    assert not within_root("/data/assessment_0/train.parquet", "/data/analysis_0")

def test_within_root_false_for_prefix_collision():
    # /data/analysis_00 is NOT under /data/analysis_0 (sep-aware)
    assert not within_root("/data/analysis_00/x", "/data/analysis_0")

def test_assert_raises_and_names_the_file(monkeypatch):
    monkeypatch.setenv("METIS_READ_ROOT", "/data/analysis_0")
    with pytest.raises(RuntimeError, match="assessment_0/train.parquet"):
        assert_within_read_root("/data/assessment_0/train.parquet")

def test_assert_noop_when_root_unset(monkeypatch):
    monkeypatch.delenv("METIS_READ_ROOT", raising=False)
    assert_within_read_root("/anywhere/at/all")  # no root set → no confinement
```

- [ ] **Step 2: Run to verify it fails** — `cd /Users/xianxu/workspace/metis && .venv/bin/python -m pytest tests/test_io_confinement.py -q` → FAIL (import error / undefined).

- [ ] **Step 3: Minimal implementation** in `metis/io.py`:

```python
def within_root(path: str, root: str) -> bool:
    """True iff `path` resolves under `root` (sep-aware, no prefix-collision)."""
    ap = os.path.abspath(path)
    ar = os.path.abspath(root)
    return ap == ar or ap.startswith(ar + os.sep)

def assert_within_read_root(path: str) -> None:
    """When METIS_READ_ROOT is set, refuse a data read outside it — the nested-CV
    (metis#23) confinement chokepoint. No-op when unset (flat/single path)."""
    root = os.environ.get("METIS_READ_ROOT")
    if root and not within_root(path, root):
        raise RuntimeError(
            f"read confinement (metis#23): {path!r} is outside METIS_READ_ROOT {root!r} "
            f"— an outer-fold sweep must not read outside its analysis root (leakage)")
```

- [ ] **Step 4: Run to verify it passes.**
- [ ] **Step 5: Commit** — `#23 M1: within-root confinement predicate + chokepoint raiser`.

### Task 1.2: wire the chokepoint into `metis.io` — at `exp_path`, NOT `load_dataset`

**Critical placement (plan-review C1):** the assertion goes in **`exp_path` (`metis/io.py:102-104`)** — the exp-relative resolver used *directly* by `cv_split.py:25` and as `dataset_dir`'s fallback for the base dataset. It must **NOT** go in `load_dataset` or `dataset_dir`'s upstream branch: those also serve **upstream handoff** reads (`features→train`, `train→predict`) whose artifacts live under `METIS_RUN_DIR` — a **sibling** of `analysis_i` (`run.go:132`), never under it — so confining there would raise on every legitimate handoff and crash the sealed sweep. `exp_path` has exactly two callers (grep to confirm: `cv_split` direct + `dataset_dir` fallback), so guarding it confines every base-dataset read while leaving handoffs unguarded. **This defect is invisible offline** (M1 base-dataset tests + M2's `metis.io`-bypassing fake exec never exercise a confined handoff) → the handoff-passes regression test below is mandatory.

**Files:**
- Modify: `metis/io.py` — call `assert_within_read_root` inside `exp_path` (on the resolved absolute path).
- Test: `tests/test_io_confinement.py` (extend), `tests/test_steps.py` (regression: unconfined path unchanged).

- [ ] **Step 1: Write the failing tests** (both halves of the seal):
  - **base-dataset confined:** set `METIS_READ_ROOT` to a *sibling* of the resolved base dataset; assert `exp_path`/`load_dataset` raises naming the file. Set it to the correct root; assert it loads.
  - **handoff PASSES (the C1 regression):** with `METIS_READ_ROOT=analysis_i` set, a read of an **upstream artifact under `METIS_RUN_DIR`** (a sibling of `analysis_i`, via `dataset_dir`'s upstream branch) must **succeed** (it does not go through `exp_path`). This is the half no other test covers.

- [ ] **Step 2: Run to verify it fails** (`exp_path` currently ignores the root).

- [ ] **Step 3: Implementation** — add `assert_within_read_root(resolved)` inside `exp_path` on the resolved absolute path. Single chokepoint per ARCH-DRY; do NOT scatter it into steps, and do NOT add it to `load_dataset`/`dataset_dir`-upstream.

- [ ] **Step 4: Run — both confinement tests pass; `tests/test_steps.py` stays green** (no `METIS_READ_ROOT` set there → no-op).
- [ ] **Step 5: Commit** — `#23 M1: assert base-dataset reads within METIS_READ_ROOT at the exp_path chokepoint`.

### Task 1.3: inject `METIS_READ_ROOT` into the step env + `StepContext`

**Files:**
- Modify: `cmd/metis/exec.go:59-65` (append `METIS_READ_ROOT` iff non-empty) + `execStep` gains a `readRoot string` field; thread it from a new `runOpts.readRoot` at the `execStep` construction site (`cmd/metis/run.go:141`). (NOT `pkg/experiment/run.go` — that Runner neither builds `execStep` nor sets env, per plan-review M-a.)
- Modify: `metis/io.py:76-94` (`StepContext` gains `read_root: str | None`; decode `METIS_READ_ROOT`).
- Test: `cmd/metis/exec_test.go` (env injected iff set), `tests/test_steps.py` (`step_context().read_root` decodes).

- [ ] **Step 1: Write the failing Go test** — construct `execStep` with a `readRoot`; run a trivial fake step; assert the subprocess saw `METIS_READ_ROOT`. And with empty `readRoot`, assert the var is **absent** (so flat/single path is untouched).

- [ ] **Step 2: Run to verify it fails.**

- [ ] **Step 3: Implementation** — `execStep` gets a `readRoot string` field; in `Execute`, append `"METIS_READ_ROOT="+e.readRoot` **only if non-empty** (empty = unconfined). Python `step_context()` adds `read_root=os.environ.get("METIS_READ_ROOT")`.

- [ ] **Step 4: Run Go + Python tests green.**
- [ ] **Step 5: Commit** — `#23 M1: thread METIS_READ_ROOT through exec.go + StepContext`.

### Task 1.4: the `outer-split` step — materialize analysis subset dirs

**Files:**
- Create: `metis/steps/outer_split.py`, `steps/metis/outer-split` (wrapper: `exec uv run … python -m metis.trace metis.steps.outer_split`, mirror `steps/metis/cv-split`).
- Reuse: `metis/split.py:cv_folds`.
- Test: `tests/test_outer_split.py` (new).

- [ ] **Step 1: Write the failing test** — drive the step (mirror `_run_step` in `tests/test_steps.py`) with `{dataset: toy, k: 3, stratify: true}`; assert it writes `analysis_0/ analysis_1/ analysis_2/` each a valid dataset dir (has `schema.json` + train rows), that `analysis_i` **excludes** the fold-`i` rows, that the three analysis sets' held-out rows are disjoint and cover all rows, and that `outer_folds.json` is a length-n positional array over `{0,1,2}`.

```python
def test_outer_split_materializes_disjoint_analysis_dirs(tmp_path, monkeypatch):
    sd = _run_step(monkeypatch, tmp_path/"runs"/"r", "outer",
                   {"dataset": "toy", "k": 3, "stratify": True}, outer_split.main)
    folds = json.loads((sd/"outer_folds.json").read_text())
    assert len(folds) == 60 and set(folds) == {0,1,2}
    for i in range(3):
        adir = sd/f"analysis_{i}"
        assert (adir/"schema.json").exists()
        n_analysis = _train_rowcount(adir)
        assert n_analysis == 60 - folds.count(i)      # fold-i rows excluded
```

- [ ] **Step 2: Run to verify it fails.**

- [ ] **Step 3: Implementation** — `outer_split.py`: load dataset via `metis.io` (NOTE: `METIS_READ_ROOT` is **unset** for this step — it reads the full dataset legitimately to split it), `cv_folds(df, k, seed, stratify_col)`, write `outer_folds.json`, and for each `i` write `analysis_i/` = the dataset with train filtered to `outer_fold != i` (+ copy `schema.json`). Emit `metrics.json {k, n}`.

- [ ] **Step 4: Run — passes.**
- [ ] **Step 5: Commit** — `#23 M1: outer-split step materializes analysis subset dirs`.

### Task 1.5: integration — a confined step reading outside its analysis root is caught

**Files:**
- Test: `tests/test_outer_split.py` or `cmd/metis/` integration test — the **negative** test that proves the seal.

- [ ] **Step 1: Write the test** — run `outer-split` (→ `analysis_0/`), then run a `train` step with `METIS_READ_ROOT=analysis_0` but pointed (maliciously) at a path outside it (the full dataset dir); assert the run fails with the confinement `RuntimeError` naming the offending file. Then repeat with the dataset = `analysis_0` and assert it succeeds. **This is the load-bearing seal test** — it can't be faked.

- [ ] **Step 2-4: Red → implement (none needed if 1.1-1.3 correct) → green.**
- [ ] **Step 5: Commit** — `#23 M1: seal test — out-of-root data read is caught + named`.

### Task 1.6: M1 atlas + milestone-close

- [ ] Update `atlas/experiment.md` (+ `atlas/index.md`): the read-confinement chokepoint (`METIS_READ_ROOT`, the `metis.io` assertion, L1+L2, the deferred syscall gap) + the `outer-split` step. Note it is the **shared sealing spine** #20/kbench#8 reuse.
- [ ] `sdlc milestone-close --issue 23 --milestone M1 --verified '<evidence>'` — fixes any Critical/Important, logs the verdict.

---

## Chunk 2: M2 — CVDriver nested loop + honest e2e

**Milestone tag `M2`.**

### Task 2.1: `CVDriver` pure Sampler (unit-tested, no IO)

**Files:**
- Modify: `pkg/sampler/driver.go` (add `CVDriver`, `OuterFoldPoint`, `cvDriverState`).
- Test: `pkg/sampler/driver_test.go` (mirror `TestSingleDriver_*`).

- [ ] **Step 1: Write the failing test** — `TestCVDriver_RunsSweeperPerOuterFoldAndAggregates`: drive `Run(ctx, CVDriver{K:3}, fn)` where `fn` returns a fixed `float64` per outer fold; assert (a) the sweeper `fn` is called exactly 3 times with distinct `OuterFoldPoint.Idx ∈ {0,1,2}`, (b) `Done` returns a `MeanSE` equal to `Aggregate` of the three scores. Add `TestCVDriver_AsksOnceThenDone`.

```go
func TestCVDriver_RunsSweeperPerOuterFoldAndAggregates(t *testing.T) {
    var seen []int
    got := sampler.Run(sampler.Ctx{Seed: 1}, sampler.CVDriver{K: 3},
        func(p sampler.OuterFoldPoint) float64 { seen = append(seen, p.Idx); return float64(p.Idx) })
    // 0,1,2 seen once each; mean = 1.0
    ... assert sorted(seen)=={0,1,2}; assert got.Mean == 1.0 ...
}
```

- [ ] **Step 2: Run to verify it fails.**

- [ ] **Step 3: Implementation** — mirror `SingleDriver` (`driver.go`) + `FixedKFolds.Init` (`folds.go:42-48`):

```go
type OuterFoldPoint struct{ Idx int }
type cvDriverState struct { k int; scores []FoldScore }
type CVDriver struct { K int; Stratify bool }

func (d CVDriver) Init(ctx Ctx) cvDriverState { return cvDriverState{k: d.K} }
func (d CVDriver) Ask(s cvDriverState) ([]OuterFoldPoint, bool) {
    if len(s.scores) >= s.k { return nil, true } // done derived from count (mirror FixedKFolds)
    pts := make([]OuterFoldPoint, s.k)
    for i := range pts { pts[i] = OuterFoldPoint{Idx: i} }
    return pts, false
}
func (d CVDriver) Tell(s cvDriverState, p OuterFoldPoint, out float64) cvDriverState {
    // FoldScore is {Addr string, Score, Complexity float64, HasComplexity bool} (aggregate.go:12-17)
    // — NO Fold field. A distinct Addr keeps MeanSE.ToldSet meaningful.
    s.scores = append(s.scores, FoldScore{Addr: fmt.Sprintf("outer#%d", p.Idx), Score: out})
    return s
}
func (d CVDriver) Done(s cvDriverState) MeanSE { return Aggregate(s.scores) }
```
(Verified against `run.go:13-31` + `folds.go:51-56`: deriving `done` from `len(s.scores) >= s.k` — as `FixedKFolds` does, it has NO "asked" flag — gives the correct single-batch-then-done contract and avoids the `k=0` empty-non-done-batch panic at `run.go:24`. `Ctx` is not needed in the state.)

- [ ] **Step 4: Run `go test ./pkg/sampler/` green.**
- [ ] **Step 5: Commit** — `#23 M2: CVDriver pure Sampler (outer folds → aggregate MeanSE)`.

### Task 2.2: extract the sweeper-as-callable — a real re-scope, not a trivial refactor

**Plan-review I2 — this is genuine surgery on three fronts, not "behavior-preserving":**
1. **Accumulator scoping.** The sweep body mutates *shared* `ss` state: `runPipelineFold` appends to `ss.man.Points` (`sweep.go:246`) and the middle closure appends to `ss.configs` (`sweep.go:137`). k outer-fold calls would pile every fold into one manifest and k× the leaderboard, corrupting `configStats`/`GuardComplexity`/`reportWinner`. `runSweeper` must own a **per-call** manifest + config accumulator (return them, or take a fresh sub-`shapeSweep`), not append to the shared `ss`.
2. **Whole-pipeline repointing (the core sealing work).** Sealing needs the **entire** pipeline pointed at `analysis_i`, not just `cv-split`. `buildFoldExperiment` re-runs `sh.Data` (`adapt`, which writes the *full* `../data/titanic`) and `features` reads `dataset: adapt` (full output). So merely parameterizing `baseDatasetRef`/`partitionStep` leaves `features` on full data. The sealed sweep must **skip the data phase** (it ran once, above the driver) and repoint the pipeline's `dataset` refs to `analysis_i` (the outer-split output, already in adapted format). The L2 chokepoint *will* catch a missed repoint (good failure mode) — but plan for the surgery, don't rely on the trap.
3. **Forked tail.** `runShapeSweep`'s tail (`sweep.go:145-168`: writeManifest / writeSweepLedger / GuardComplexity / reportWinner / shipWinner) is flat-path-only and consumes a `SweepResult`; `CVDriver.Done` returns `MeanSE`. Task 2.3 forks the tail (no ship; an estimate report instead) — Task 2.2 only extracts the *per-fold sweeper unit*, leaving the flat tail intact for `driver:single`.

**Files:**
- Modify: `cmd/metis/sweep.go` — extract `func (ss *shapeSweep) runSweeper(baseDir, readRoot string) (sampler.SweepResult, sweepManifest)` (returns its own manifest, does not mutate shared `ss.man`/`ss.configs`); parameterize `buildFoldExperiment`/`partitionStep`/`baseDatasetRef` to (a) skip data steps when `baseDir` is a ready dataset, (b) repoint pipeline `dataset` refs to `baseDir`, (c) stamp a per-call partition ref.
- Test: `shapesweep_test.go` / `shipe2e_test.go` — the `driver:single` path (`runSweeper(fullDataset, "")`) stays green.

- [ ] **Step 1: Baseline** — run `TestShapeSweep_HonestE2E` green first (the `driver:single` behavior `runSweeper` must preserve).

- [ ] **Step 2: Extract** `runSweeper` with its own accumulators; `SingleDriver`'s `runPoint` calls `ss.runSweeper(baseDatasetRef(sh), "")` and the existing flat tail consumes the returned `SweepResult`+manifest exactly as before.

- [ ] **Step 3: Run `go test ./cmd/metis/` — `driver:single` e2e still green** (the extraction preserved the flat path). This is the guard that the accumulator re-scoping didn't regress single.

- [ ] **Step 4: Commit** — `#23 M2: extract runSweeper(baseDir, readRoot) with per-call accumulators`.

### Task 2.3: wire `CVDriver` into `runShapeSweep` — per-outer-fold sealed sweep + refit-and-score

**Files:**
- Modify: `cmd/metis/sweep.go:88-169` — branch on `sh.Driver.CV != nil`; run the `outer-split` step (Task 1.4) → k `analysis_i/` dirs + `outer_folds.json`; compose `sampler.Run(ctx, CVDriver{K,Stratify}, outerRunPoint)`.
- Modify: `pkg/experiment/shape.go:201-203` — delete the stub-rejection.

- [ ] **Step 1: Write the failing e2e** (Task 2.5 below is the full one; here a minimal `go test` that `driver:cv` no longer rejects at validate + runs k outer sweeps). First assert `ValidateShape` accepts `driver:cv` (update `pkg/experiment/shape_test.go:149-151`).

- [ ] **Step 2: Delete the stub-rejection** (`shape.go:201-203`); update `shape_test.go` (the "driver cv is #23" case flips from must-fail to must-pass).

- [ ] **Step 3: Implement `outerRunPoint(p OuterFoldPoint) float64`:**
  1. `analysisDir := analysis_i` (from the outer-split output for `p.Idx`).
  2. `winner := ss.runSweeper(analysisDir, readRoot=analysisDir).Ship` — **sealed + confined selection**. (`Winner.Point.With` carries enough to reconstruct the run — `shipWinner` already does this via `shapeConfigToExperiment`, `ledger.go:76-88`, verified.)
  3. **refit-and-score, expressed AS A FOLD (I3+I4):** build a per-fold experiment from `winner.Point` over the **full** dataset, with the folds coming from the **outer** partition (`outer_folds.json`), held fold = `p.Idx`; run it and read back its `fold_score`. Wiring detail (plan-review I3): `buildFoldExperiment` today synthesizes the *inner* `cv-split` (`k = sh.Sweeper.Resample.CV.K`) and stamps `ss.partRef` — the outer scoring must instead source folds from the outer partition. Reconcile the `outer_folds.json` ↔ `folds.json` naming: either have `outer-split` also emit a `folds.json`-shaped artifact the scoring's `train._load_folds` reads (`train.py:73-76`), or parameterize the folds source. **No `METIS_READ_ROOT`** here — selection is done; this is an honest held-out eval (and the outer-mask fold-expression is what lets it inherit #20's fold-safe features per the BOUND ASSUMPTION above).
  4. return the `fold_score`.
- The driver `Done` → `MeanSE` = the honest estimate. **No `shipWinner` call** (nested-CV ships nothing) — Task 2.2's forked tail emits the estimate report instead (Task 2.6).

- [ ] **Step 4: Focused test — outer-scoring experiment from a winner (plan-review M-c).** A Go test that, given a `Winner.Point.With` + the outer partition + held idx `i`, the builder produces a fold experiment whose pipeline == the winner's config, whose folds source == the outer partition, and whose held fold == `i` (refit-on-`!=i`, score-on-`==i`). Covers the construction directly, not just via the fake e2e.

- [ ] **Step 5: Run the minimal e2e — k outer sweeps run, an estimate is produced.**
- [ ] **Step 6: Commit** — `#23 M2: wire CVDriver — sealed per-fold sweep + refit-and-score; drop stub-reject`.

### Task 2.4: cost surfacing (~5×)

**Files:**
- Modify: `cmd/metis/sweep.go:105-111` (`-dry-run`) + `:123` (run header) — when `driver:cv`, multiply by `K_outer` and show the shape `N configs × K_inner × K_outer (+ K_outer refits)`.

- [ ] **Step 1: Write the failing test** — `metis run -dry-run` on a `driver:cv` shape prints the outer multiplier + a "nested-CV: honest estimate, no shippable winner" note. Assert the line.
- [ ] **Step 2-4: Red → implement → green.**
- [ ] **Step 5: Commit** — `#23 M2: surface the ~5× nested-CV cost at dry-run + run header`.

### Task 2.5: honest e2e — the load-bearing verification

**Files:**
- Test: `cmd/metis/nestedcv_e2e_test.go` (new; mirror `shipe2e_test.go:TestShapeSweep_HonestE2E` + `soundFoldExec`).

- [ ] **Step 1: Write the e2e** — a `driver:cv` shape over the fake exec; assert:
  1. **k outer sweeps ran** (cost = `configs × k_inner × k_outer`, via the fake's call trace).
  2. The **honest outer estimate is a `mean±SE`** computed over k outer folds via a **distinct code path** from the inner cv-max (i.e. the mechanism produces an outer estimate at all, and it aggregates the k outer scores). **Frame honestly (plan-review M-b):** with a config-deterministic fake, the outer scores mechanically equal the inner cv unless artificially rigged, so this asserts the *plumbing* (mean±SE over k folds, no ship) — **NOT** the real "estimate < cv-max" honesty gap, which is a real-data property (operator-gated Titanic, RUNBOOK). Do not over-claim the gap in a fake test.
  3. **No submission is shipped** (assert zero ship runs — the inverse of `TestShapeSweep_ShipsWinner`).
  4. **Confinement holds**: inject a fake step that reads outside `analysis_i` during a sweep and assert the run **fails** with the confinement error (proves L2 is live end-to-end). Then remove it and assert the run succeeds. (Because the fake exec bypasses `metis.io`, this injected leak must go through the *real* `exp_path` chokepoint or an equivalent seam — otherwise the fake also bypasses the confinement; wire the leak so it exercises the actual assertion.)

- [ ] **Step 2-4: Red → implement wiring fixes → green.**
- [ ] **Step 5: Commit** — `#23 M2: honest nested-CV e2e (estimate<cv-max, k× cost, no ship, confinement enforced)`.

### Task 2.6: reporting + atlas + M2 milestone-close

- [ ] **Reporting** — `reportEstimate`: print the honest procedure estimate `mean±SE` **next to** the inner cv-max (the honesty gap made visible), with the "no shippable winner; ship via driver:single" note. Test the output line.
- [ ] **Atlas** — `atlas/experiment.md` + `atlas/index.md`: the `driver:cv` outer resample (result-dependent select-then-assess-on-sealed-fold; produces no winner; ~5× cost; reuses `Aggregate`), alongside the sweeper. Update `construct/datatype/experiment-shape.md` if it enumerates driver modes.
- [ ] `sdlc milestone-close --issue 23 --milestone M2 --verified '<evidence>'`.
- [ ] **Then** `sdlc close --issue 23 …` (final close; real-data honest-estimate-tracks-public run is operator-gated Kaggle — note it).

---

## Test strategy summary

- **Pure** (`withinRoot`, `CVDriver`, `Aggregate` reuse) → colocated unit tests, no IO (`test_io_confinement.py`, `driver_test.go`).
- **Integration** (`outer-split`, chokepoint, env threading) → the step-contract harness (`_run_step`) + a Go exec test; **the seal is proven by a negative test** (out-of-root read caught) — the load-bearing check.
- **E2e** (`nestedcv_e2e_test.go`) → process-level fake `StepExecutor` (`soundFoldExec` twin), asserting the four honest-CV invariants incl. the confinement failing on an injected leak.
- **Operator-gated**: the honesty acceptance ("nested-CV estimate tracks public within noise") needs the real Titanic run (Kaggle creds, RUNBOOK) — documented as the post-merge gated step, not a code test.

## Open decisions folded in (from the design conversation)
- **Sealing = L1 structural + L2 chokepoint**; syscall-level (rogue non-`metis.io` reads, parquet-via-C) documented-and-deferred. (Operator-confirmed.)
- **Refit-and-score reuses full-data + outer-mask** (post-selection, no leakage) → no new Python primitive; only analysis subset dirs are materialized.
- **No shippable winner** — ship stays on `driver:single` (estimation ≠ selection).

---

## Revisions

### 2026-07-12 — post-M1-review (verdict FIX-THEN-SHIP, no Critical/Important)
Reason: fold M1 milestone-review recommendations into the plan so the M2 implementor isn't misled.
- **Chokepoint row corrected** — the integration table said the assertion lives in `load_dataset`/`dataset_dir`/`exp_path`; it lives in **`exp_path` ONLY** (the C1 fix Task 1.2 already stated). Adding it to `load_dataset`/`dataset_dir`-upstream would crash legit run-dir handoff reads. M2: keep confinement at `exp_path`.
- **Step renamed** — the step is **`outer-split`** (`outer_split.py`), not "outer-partition" (the layer/milestone concept keeps the "outer-partition" name; the *step* matches the shipped `outer_split.py`/`steps/metis/outer-split`).
- **M2 orchestration note added** — M2 runs `outer-split` above the driver and must leave `METIS_READ_ROOT` unset for it (it reads the full dataset legitimately).
- **exec.go hardening** (M1 minor): inherited `METIS_READ_ROOT` is now stripped from the subprocess env when `readRoot==""`, so an ambient shell value can never confine the flat `driver:single` path.
