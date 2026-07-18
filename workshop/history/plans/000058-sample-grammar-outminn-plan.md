# `--sample outMinN` Implementation Plan (metis#58)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One `--sample` grammar (`out<M>`, `in<N>`, `out<M>in<N>`; bare integer retired) that subsamples the outer AND inner CV levels as deterministic prefix subsets of the shape-declared partitions — iteration runs escalate into decision runs via cache hits.

**Architecture:** Extend the metis#42 m-of-k principle to the inner level by splitting the one field that currently conflates two roles: `sweepPass.splitK` feeds both the partition (`cvSplitStep`, `partitionRef` — the leaf content-address) and the fold enumeration (`FixedKFolds{K}`). The partition side keeps the declared `inner_k`; a new `runK` bounds the enumeration. `FixedKFolds.Init` already enumerates folds `0..K-1` in order over `ctx.Partition`, so `FixedKFolds{K: runK}` with the unchanged partition ref IS the deterministic prefix subset — leaf addresses identical to a full run's first N folds, cache continuity for free. Outer level already works exactly this way (`analysisRefs` materialized for all k; `CVDriver{K: runFolds}` runs fewer). Parsing is a pure function (`ARCH-PURE`); validation stays in the existing loud-refusal block in `runShapeSweep`.

**Tech Stack:** Go (cmd/metis), table-driven unit tests + the existing in-process e2e harness (`runExperiment` + `foldFakeExec` + `calls` recorder in `shapesweep_test.go`).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `parseSample` | `cmd/metis/sample.go` | new |
| `sampleSpec` | `cmd/metis/sample.go` | new |
| `sweepPass.runK` (field beside `splitK`) | `cmd/metis/sweep.go` | modified |

- **`parseSample(s string) (sampleSpec, error)`** — the grammar. `sampleSpec{Out, In int}` with 0 = unset.
  - **DRY rationale:** the ONE place the grammar exists; main.go calls it, tests table-drive it, error text names the grammar for every malformed form.
  - **Future extensions:** if metis#54 (racing) ever wants a budget grammar (`budget500`), it widens here.
- **`sweepPass.runK`** — folds *enumerated* this pass (`FixedKFolds{K: runK}`), vs `splitK` = folds *partitioned* (`cvSplitStep`, `partitionRef`, leaf addresses). `runK == splitK` everywhere except a sampled inner level. The comment on the two fields must state this split — it IS the design.
  - **Invariant:** `partitionRef(sh, splitFolds)` and `buildFoldExperiment(..., splitK, ...)` NEVER see `runK` — that's what keeps subset runs address-compatible with full runs.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `--sample` flag (Int → String) | `cmd/metis/main.go` | modified | CLI |
| validation + plumb | `cmd/metis/sweep.go` (`runShapeSweep`, `runNestedCV`, `runOuterFold`, `runSweeper`) | modified | sampler wiring |
| banners/progress | `cmd/metis/sweep.go` + `seededTotals` (`progress.go`) | modified | operator display |
| e2e | `cmd/metis/sample_e2e_test.go` | new | in-process harness |

- **Flag:** `fs.Int("sample", …)` → `fs.String("sample", "", …)`; parse in main, put `sampleSpec` on `runOpts` (rename field `sample int` → `sample sampleSpec`; grep every `o.sample` use). `--fast` stays a bool; equivalence to `out1` is applied in the validation block, not the parser.
- **Validation (existing block in `runShapeSweep`, sweep.go:230-248) — same loud-refusal family, messages name the grammar:** `--fast` with any `--sample` → mutually exclusive error; any sampling on a flat (1-config) run → refuse; `1 ≤ out ≤ k`; `1 ≤ in ≤ innerK` (`innerK = CV.InnerFolds()`, which is already `k` when `inner_k` unset). Then `runFolds = out or k`, `runInnerK = in or innerK`.
- **Banners:** when a level is sampled, print `M/k` (e.g. `1/10 outer fold(s) × (7 configs × 2/5 inner folds)`); an UNSAMPLED level keeps today's exact format — `TestNestedCV_InnerKSplit`'s banner assertion must pass UNTOUCHED (regression guard that the default path didn't move). Same for the `--dry-run` banner. `seededTotals(ctx, nested, runFolds, configPts, splitFolds)` gets `runInnerK` as its fold-level count (the board's denominators show what this run will actually do).
- **Ledger/select need NO change** (issue Spec): rows carry fold coordinates; `writeSweepLedger` dedups by point-address; subset runs converge to full coverage. The e2e proves it rather than trusting the prose.

## Task 1: `parseSample` (TDD)

**Files:** Create `cmd/metis/sample_test.go`, `cmd/metis/sample.go`.

- [ ] **Step 1: failing test** — table-driven:

```go
package main

import "testing"

func TestParseSample(t *testing.T) {
	cases := []struct {
		in        string
		out, inner int
		wantErr   bool
	}{
		{"out1", 1, 0, false},
		{"out3", 3, 0, false},
		{"in2", 0, 2, false},
		{"out1in2", 1, 2, false},
		{"out10in5", 10, 5, false},
		{"", 0, 0, false},      // unset — no sampling
		{"3", 0, 0, true},      // bare integer retired (breaking, by design)
		{"out0", 0, 0, true},   // zero is not a fold count
		{"in0", 0, 0, true},
		{"out", 0, 0, true},    // missing number
		{"in", 0, 0, true},
		{"outin2", 0, 0, true},
		{"in2out1", 0, 0, true}, // fixed order: out before in
		{"out1in2x", 0, 0, true},
		{"OUT1", 0, 0, true},    // lowercase only
	}
	for _, c := range cases {
		got, err := parseSample(c.in)
		if c.wantErr != (err != nil) {
			t.Errorf("parseSample(%q): err=%v, wantErr=%v", c.in, err, c.wantErr)
			continue
		}
		if err == nil && (got.Out != c.out || got.In != c.inner) {
			t.Errorf("parseSample(%q) = %+v, want {Out:%d In:%d}", c.in, got, c.out, c.inner)
		}
	}
}
```

- [ ] **Step 2:** `go test ./cmd/metis -run TestParseSample` → FAIL (undefined).
- [ ] **Step 3: implement** —

```go
// sample.go — the --sample grammar (metis#58): out<M>, in<N>, out<M>in<N>.
// M subsamples the OUTER folds, N the INNER per-config folds; both are
// deterministic prefix subsets of the shape-declared partitions (the shape's
// k/inner_k stay the estimand — the flag only trades precision for cost).
// The bare-integer form (--sample 3) is retired: one grammar, parsed here only.
package main

import (
	"fmt"
	"regexp"
	"strconv"
)

type sampleSpec struct {
	Out int // outer folds to run; 0 = all k
	In  int // inner folds per config; 0 = all inner_k
}

var sampleRe = regexp.MustCompile(`^(?:out([1-9][0-9]*))?(?:in([1-9][0-9]*))?$`)

func parseSample(s string) (sampleSpec, error) {
	if s == "" {
		return sampleSpec{}, nil
	}
	m := sampleRe.FindStringSubmatch(s)
	if m == nil || (m[1] == "" && m[2] == "") {
		return sampleSpec{}, fmt.Errorf(
			"--sample %q: want out<M>, in<N>, or out<M>in<N> (e.g. --sample out1in2; M,N ≥ 1; the bare-integer form is retired — use out<M>)", s)
	}
	// strconv.Atoi, NOT Sscanf: an overflowing count (out99999999999999999999) must be a
	// loud error, not a silently-unset spec that runs the FULL sweep (review issue 5).
	var sp sampleSpec
	var err error
	if m[1] != "" {
		if sp.Out, err = strconv.Atoi(m[1]); err != nil {
			return sampleSpec{}, fmt.Errorf("--sample %q: out count: %v", s, err)
		}
	}
	if m[2] != "" {
		if sp.In, err = strconv.Atoi(m[2]); err != nil {
			return sampleSpec{}, fmt.Errorf("--sample %q: in count: %v", s, err)
		}
	}
	return sp, nil
}
```

  (Add `{"out99999999999999999999", 0, 0, true}` to the Step 1 table — the overflow case.)

- [ ] **Step 4:** test → PASS. **Step 5: commit** — `git commit -m "#58: parseSample — the outMinN grammar (pure, bare-int retired)"`.

## Task 2: flag + runOpts rewire

**Files:** Modify `cmd/metis/main.go` (flag def ~line 44-45, `runOpts{...}` literal ~line 65), `cmd/metis/run.go` (`runOpts` struct field).

- [ ] **Step 1:** `sampleN := fs.Int(...)` → `sampleStr := fs.String("sample", "", "metis#58: run a subset of the declared CV folds — out<M> (M of the k outer folds), in<N> (N of the inner_k per-config inner folds), or out<M>in<N>. Deterministic prefix subsets of the SAME partitions, so subset runs cache-escalate into full runs. k/inner_k stay the estimand; sampling only trades precision for cost (probe with it, don't re-select what ships on it). Nested (multi-config) runs only; errors loudly out of range or with --fast.")`. Rewrite `--fast` help to say "Shorthand for --sample out1". After `fs.Parse`, `sp, err := parseSample(*sampleStr)` → usage error on err; store `sample: sp`.
- [ ] **Step 2:** `runOpts.sample` type `int` → `sampleSpec`. `go build ./cmd/metis` fixes the non-test consumers (sweep.go validation block) — **but `go build` never compiles `*_test.go`**: also `grep -n 'sample' cmd/metis/*_test.go`. Known test consumers (review issue 1, a DESIGN decision not a compile fix — handled in Task 3 Step 4): `nestedcv_e2e_test.go` `TestNestedCV_SampleRunsMOfKFolds` (~141-183, sets `sample: 2` as int, asserts the old `"2 outer fold(s)"` banner) and `TestNestedCV_SampleGuards` (~189-228, sets `o.sample = 3/-1/1`, asserts the old error text).
- [ ] **Step 3:** `go build ./...` clean — build ONLY: `go vet` and `go test` both compile `*_test.go`, which is still broken (int literals) until Task 3 Step 4; vet + full-package test run at Task 3 Step 5. **NO commit here** — Tasks 2+3 land as ONE green commit at Task 3 Step 6 (commits stay green; the working tree may be red between tasks).

## Task 3: validation + inner plumb (the core)

**Files:** Modify `cmd/metis/sweep.go`, `cmd/metis/progress.go` (only if `seededTotals`'s param name misleads — the call site passes the new value either way).

- [ ] **Step 1:** In `runShapeSweep` (validation block, ~line 229-248): replace the `switch` with:

```go
runFolds := k
runInnerK := splitFolds // NOT innerK: on a FLAT run the per-config CV runs at k (inner_k ignored-loudly), and this value feeds the board denominators — innerK here would display 3 folds for a 2-fold flat run (review issue 3). Identical to innerK on the nested path.
if o.sample.Out != 0 || o.sample.In != 0 {
	if o.fast {
		return fmt.Errorf("run: --sample and --fast are mutually exclusive (--fast is shorthand for --sample out1)")
	}
	if !nested {
		return fmt.Errorf("run: --sample only applies to a nested (multi-config) run — this shape has 1 config, a flat CV with no outer/inner levels to sample")
	}
	if o.sample.Out != 0 && (o.sample.Out < 1 || o.sample.Out > k) {
		return fmt.Errorf("run: --sample out%d out of range — want 1 ≤ M ≤ k=%d (the outer partition has exactly k folds)", o.sample.Out, k)
	}
	if o.sample.In != 0 && (o.sample.In < 1 || o.sample.In > innerK) {
		return fmt.Errorf("run: --sample in%d out of range — want 1 ≤ N ≤ inner_k=%d (the inner partition has exactly inner_k folds)", o.sample.In, innerK)
	}
	if o.sample.Out != 0 {
		runFolds = o.sample.Out
	}
	if o.sample.In != 0 {
		runInnerK = o.sample.In
	}
} else if o.fast {
	runFolds = 1
}
```

  (Preserve the existing comment block about k-as-estimand, extending it to the inner level. The `< 1` guards STAY even though the CLI parser rejects 0: `runOpts` is a direct-construction seam — every e2e builds it without the parser, and a negative value would reach `make([]…, -1)` and panic in `CVDriver.Init`/`FixedKFolds.Init`. The existing "negative m" guard test exists precisely for this seam — review issue 2.)
- [ ] **Step 2: split the pass fields.** `sweepPass` gets `runK int` beside `splitK` with the two-role comment (splitK → partition/addresses; runK → folds enumerated; equal unless inner-sampled). `runSweeper`: `FixedKFolds{K: pass.runK}`. Flat-path pass literal (~line 334): `runK: k`. `runOuterFold` (~line 523): signature gains `runInnerK int`; pass literal gets `runK: runInnerK`. `runNestedCV` signature gains `runInnerK` and threads it to `runOuterFold`; `runShapeSweep`'s call passes it. The held-out scoring run and `buildFoldExperiment`/`partitionRef` are untouched (they only ever see `splitK`/`splitFolds`).
- [ ] **Step 3: banners.** In `runNestedCV` + the `--dry-run` branch: a SAMPLED level prints `M/k`, an unsampled level keeps today's exact format. Helper `fmtLevel(run, total int) string` → `"3"` when `run == total`, else `"1/10"`. **Explicit decision (review issue 4): `--fast` renders as `1/k`** — it IS a sampled run (`--sample out1` equivalent), and no existing test pins the old `--fast` banner; the new sample e2e pins the new format. `seededTotals(...)` call site passes `runInnerK` (board denominators = actual work; correct on flat via the `runInnerK := splitFolds` init).
- [ ] **Step 4: rework the two legacy sample tests** (`nestedcv_e2e_test.go`, review issue 1 — assertion DESIGN, not compile-fixing): `TestNestedCV_SampleRunsMOfKFolds` → `sample: sampleSpec{Out: 2}`, banner assertion becomes the new sampled format (`"2/3 outer fold(s)"`); ledger/fold assertions unchanged (still the real content). `TestNestedCV_SampleGuards` → build specs (`sampleSpec{Out: 3}` on k=2, `{Out: -1}`, `{Out: 1}`+`fast`), assert the NEW error texts (grammar-named); keep the negative-value subtest (it guards the runOpts direct-construction seam, not the parser). Their refusal coverage overlaps Task 4's new e2e — keep flat-refusal/range/fast-exclusion HERE (they predate #58) and give Task 4 only the NEW surface (in-range, bare-string, escalation).
- [ ] **Step 5:** `go build ./... && go test ./cmd/metis` → ALL green, including `TestNestedCV_InnerKSplit`/`TestFlatCV_InnerKIgnoredLoudly` UNTOUCHED (the unsampled-format regression guard). **Step 6: commit** (Tasks 2+3 together) — `git commit -m "#58: --sample outMinN — grammar, validation, splitK/runK inner subsampling"`.

## Task 4: e2e (the Done-when proof)

**Files:** Create `cmd/metis/sample_e2e_test.go` (mirror `innerk_e2e_test.go`'s harness: `foldShapeCVMD` + `strings.Replace` for `inner_k`, `writeShapeFile`, `runExperiment` directly with `runOpts{sample: …, cache: true, exec: foldFakeExec{calls: &calls}}`, `loadLedgerOrFatal`).

- [ ] **Step 1: new-surface refusals test** — the legacy guards test (reworked in Task 3 Step 4) keeps flat/range/fast-exclusion for `Out`; add here only what's NEW: `in4` on inner_k=3 (inner range), `In: -1` (inner negative, the runOpts seam), and the bare-string path: assert `parseSample("3")` errors with the grammar text via the same call main.go makes (the CLI surface of "bare integer retired").
- [ ] **Step 2: subset test** — shape k=2/inner_k=3, 2 configs, `sample: {Out: 1, In: 2}`: banner shows `1/2 outer fold(s) × (2 configs × 2/3 inner folds)`; ledger inner rows per (config, outer fold 0) = folds `{0,1}` exactly; NO outer-fold-1 rows.
- [ ] **Step 3: escalation test (cache continuity + convergence)** — same tmp workspace, `cache: true`, recording `calls`. Run A: `{Out: 1, In: 2}`. Run B: `{Out: 1}` (full inner). Assert: (a) run B's `calls` contains NO train invocation for inner folds 0/1 of either config (cache HIT — a hit never reaches exec; identify train calls the way existing cache tests in `shapesweep_test.go` do), only fold 2's; (b) after B the ledger has EXACTLY 3 inner rows per (config, outer 0) — folds {0,1,2}, each once (dedupe, no double-count).
- [ ] **Step 4:** `go test ./cmd/metis` full package → PASS. **Step 5: commit** — `git commit -m "#58: e2e — refusals, prefix subset, cache-escalation convergence"`.

## Task 5: caller sweep (breaking-change discipline, ARCH-PURPOSE)

- [ ] **Step 1 (metis):** `grep -rn '\-\-sample' cmd/ atlas/ README* docs/ 2>/dev/null` — update every hit to the new grammar: the `main.go` flag/`--fast` help (already done in Task 2 — verify), `atlas/experiment.md` (the `--sample m`/metis#42 paragraphs gain the inner axis + the splitK/runK seam sentence), any `select` hint text. (`run.go` has only the runOpts field comment, renamed in Task 2 — review issue 7.) The metis#42 semantics prose stays (k the estimand, prefix a valid random subset) — extended, not replaced.
- [ ] **Step 2 (kbench, docs-only commit on main):** `competition/titanic/pipelines/RUNBOOK-sweep.md` (`--sample 3` → `--sample out3`; the §1 inner_k paragraph gains one line: iterate with `--sample out1in2`), `competition/titanic/pipelines/titanic-sweep.md` body (`--sample 3` mention), `atlas/titanic-workspace.md` (`--sample 3` mention), `workshop/plans/000012-...-plan.md` + issue #12 (`--sample 3` → `out3` in Task 9/M2/RUNBOOK-s6e7 text). Commit: `docs: metis#58 --sample grammar sweep (out3; +out1in2 iteration line)`.
- [ ] **Step 3: shadow-sweep** — `grep -rn -- '--sample [0-9]' /Users/xianxu/workspace/metis /Users/xianxu/workspace/kbench --include='*.md' --include='*.go' | grep -v '/workshop/'` → zero hits. **Exemption (review issues 6 + re-review 2): `workshop/` is excluded WHOLESALE** — issue Logs record past runs (rewriting falsifies what was executed: metis #33/#36 cohort notes), and #58's own issue/plan/project-log text quotes the retired form *by design* (describing the retirement). Every current-usage surface the sweep defends lives under `cmd/`, `atlas/`, `docs/`, `README*`, and kbench `competition/`.
- [ ] **Step 4: commit** (metis docs part) — `git commit -m "#58: docs — grammar swept to outMinN"`.

## Task 6: close

- [ ] **Step 1:** Issue `## Log`: implementation notes + the splitK/runK seam; tick Plan boxes; atlas updated in Task 5 (the close gate's atlas guard should be satisfied by `atlas/experiment.md`).
- [ ] **Step 2:** `sdlc pr` → `sdlc merge` (single boundary — plain checkboxes, no Mx; the mandatory fresh-eyes review runs inside `sdlc close`).
- [ ] **Step 3:** `sdlc close --issue 58 --verified 'unit+e2e green: parseSample table, refusals, 1/2×2/3 subset ledger, cache-escalation convergence (run B spawns only fold 2, ledger exactly {0,1,2} once each); callers swept, shadow-sweep grep clean'` (actuals measured by the gate).

## Execution notes

- **Before Task 1:** set `estimate_hours: 3` in the issue frontmatter; `sdlc change-code` (branch `000058-sample-grammar-outminn-subsample-both-cv-levels`).
- The kbench docs commit (Task 5 Step 2) is cross-repo: plain git on kbench main (docs-only), referencing `metis#58` in the message.
- After merge: rebuild the metis binary used by kbench e2e/runbooks (`go build -C ../metis -o bin/metis ./cmd/metis`) before running kbench#12 Task 9.
