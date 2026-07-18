package main

import (
	"io"
	"strings"
	"testing"
)

// sampleShapeMD: foldShapeMD (k=2, 2 configs {a,b} = one family) with inner_k: 3 — the
// two-level shape the metis#58 sample-grammar tests run against.
func sampleShapeMD() string {
	return strings.Replace(foldShapeMD("[a, b]"),
		"resample: {cv: {k: 2, stratify: false}}",
		"resample: {cv: {k: 2, inner_k: 3, stratify: false}}", 1)
}

// TestSample_NewSurfaceRefusals (metis#58): the refusal edges NEW to the grammar — the inner
// range against inner_k, a negative inner count built directly on the runOpts seam, and the
// retired bare-integer CLI form. (Outer range / flat / --fast exclusion live in
// TestNestedCV_SampleGuards — they predate #58.)
func TestSample_NewSurfaceRefusals(t *testing.T) {
	base := func(t *testing.T) runOpts {
		return runOpts{expPath: writeShapeFile(t, t.TempDir(), sampleShapeMD()), now: fixedNow(),
			git:  fakeGitProbe{name: "metis", sha: "sha", dirty: false},
			exec: foldFakeExec{}, out: io.Discard}
	}

	t.Run("in exceeds inner_k", func(t *testing.T) {
		o := base(t)
		o.sample = sampleSpec{In: 4}
		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "inner_k=3") {
			t.Errorf("--sample in4 with inner_k=3 must error naming the inner_k limit, got %v", err)
		}
	})
	t.Run("negative in on the runOpts seam", func(t *testing.T) {
		o := base(t)
		o.sample = sampleSpec{In: -1}
		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
			t.Errorf("sampleSpec{In: -1} must error mentioning sample, got %v", err)
		}
	})
	t.Run("bare integer is retired at the CLI parse", func(t *testing.T) {
		if _, err := parseSample("3"); err == nil || !strings.Contains(err.Error(), "bare-integer form is retired") {
			t.Errorf("parseSample(\"3\") must refuse with the retirement message, got %v", err)
		}
	})
}

// TestSample_OutInPrefixSubset (metis#58): --sample out1in2 on k=2/inner_k=3 runs outer fold 0
// only, and inner folds {0,1} only, of the UNCHANGED 3-way inner partition — asserted on the
// sampled M/k banner and the ledger's fold coordinates.
func TestSample_OutInPrefixSubset(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, sampleShapeMD())

	var out strings.Builder
	_, err := runExperiment(runOpts{
		expPath: expPath, now: fixedNow(),
		git:  fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		exec: foldFakeExec{}, out: &out,
		sample: sampleSpec{Out: 1, In: 2},
	})
	if err != nil {
		t.Fatalf("--sample out1in2 should succeed: %v", err)
	}
	if s := out.String(); !strings.Contains(s, "1/2 outer fold(s) × (2 configs × 2/3 inner folds)") {
		t.Errorf("banner should show both sampled levels as M/k, got:\n%s", s)
	}

	led := loadLedgerOrFatal(t, expPath)
	type cfKey struct {
		model string
		outer int
	}
	innerFolds := map[cfKey]map[int]bool{}
	outerFolds := map[int]bool{}
	for _, r := range led.Rows {
		switch r.Level {
		case "inner":
			k := cfKey{model: r.FreeParams["train.model"].(string), outer: *r.OuterFold}
			if innerFolds[k] == nil {
				innerFolds[k] = map[int]bool{}
			}
			innerFolds[k][*r.Fold] = true
		case "outer":
			outerFolds[*r.OuterFold] = true
		}
	}
	// Outer: fold 0 only (prefix of k=2).
	if len(outerFolds) != 1 || !outerFolds[0] {
		t.Errorf("outer rows should cover exactly sampled fold {0}, got %v", outerFolds)
	}
	// Inner: per config, folds {0,1} only (prefix of the 3-way partition); fold 2 must NOT run.
	if len(innerFolds) != 2 { // 2 configs × 1 outer fold
		t.Fatalf("want 2 (config, outer) groups, got %d", len(innerFolds))
	}
	for k, folds := range innerFolds {
		if len(folds) != 2 || !folds[0] || !folds[1] || folds[2] {
			t.Errorf("(%s, outer %d): inner folds = %v, want {0,1}", k.model, k.outer, folds)
		}
	}
}

// TestSample_CacheEscalationConverges (metis#58, the Done-when proof): an in2 iteration run
// escalates into a full-inner run via the cache — run B (--sample out1, full inner 3) HITs the
// 2 folds run A measured (a HIT never reaches exec: no train spawns for folds 0/1) and the
// ledger CONVERGES to exactly inner_k rows per (config, outer fold), each fold once (the
// point-address dedupe absorbs run B's re-emitted fold-0/1 rows).
func TestSample_CacheEscalationConverges(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, sampleShapeMD())
	base := func(calls *[]string, sp sampleSpec) runOpts {
		return runOpts{expPath: expPath, now: fixedNow(),
			git:  fakeGitProbe{name: "metis", sha: "sha", dirty: false},
			exec: foldFakeExec{calls: calls}, out: io.Discard,
			cache: true, sample: sp}
	}

	// Run A — the iteration run: out1in2. Sealed inner = 2 configs × folds {0,1} = 4 train
	// runs + 1 outer-scoring (family winner b refit as a full-data fold) = 5.
	var callsA []string
	if _, err := runExperiment(base(&callsA, sampleSpec{Out: 1, In: 2})); err != nil {
		t.Fatalf("run A (out1in2): %v", err)
	}
	if n := countCalls(callsA, "train"); n != 5 {
		t.Errorf("run A train spawns: want 5 (4 sealed inner + 1 outer-scoring), got %d: %v", n, callsA)
	}

	// Run B — the escalation: out1 (full inner 3). Folds {0,1} HIT (same 3-way partition →
	// identical leaf addresses); only fold 2 spawns (2 configs = 2 train). The outer-scoring
	// run HITs too: the fake's winner (b, base 0.90 > a's 0.80 on any fold subset) is stable,
	// so the refit run's address is unchanged from run A.
	var callsB []string
	if _, err := runExperiment(base(&callsB, sampleSpec{Out: 1})); err != nil {
		t.Fatalf("run B (out1): %v", err)
	}
	if n := countCalls(callsB, "train"); n != 2 {
		t.Errorf("run B train spawns: want exactly 2 (fold 2 per config; folds 0/1 + outer-scoring HIT), got %d: %v", n, callsB)
	}
	if n := countCalls(callsB, "features"); n != 1 {
		t.Errorf("run B features spawns: want exactly 1 (fold 2; fold-distinct, config-invariant), got %d: %v", n, callsB)
	}

	// Convergence: after B the ledger holds exactly inner_k=3 rows per (config, outer 0),
	// each fold ONCE — run B re-emits fold-0/1 rows and the point-address dedupe absorbs them.
	led := loadLedgerOrFatal(t, expPath)
	type cfKey struct {
		model string
		outer int
	}
	perFold := map[cfKey]map[int]int{}
	for _, r := range led.Rows {
		if r.Level != "inner" {
			continue
		}
		k := cfKey{model: r.FreeParams["train.model"].(string), outer: *r.OuterFold}
		if perFold[k] == nil {
			perFold[k] = map[int]int{}
		}
		perFold[k][*r.Fold]++
	}
	if len(perFold) != 2 {
		t.Fatalf("want 2 (config, outer) groups after escalation, got %d", len(perFold))
	}
	for k, folds := range perFold {
		if len(folds) != 3 {
			t.Errorf("(%s, outer %d): want folds {0,1,2}, got %v", k.model, k.outer, folds)
		}
		for f, n := range folds {
			if n != 1 {
				t.Errorf("(%s, outer %d) fold %d: recorded %d× — dedupe must keep exactly one row", k.model, k.outer, f, n)
			}
		}
	}
}

// TestSample_FastBannerRendersSampled pins the explicit #58 decision that --fast — being
// --sample out1 — renders as a SAMPLED level (`1/k`), not the plain unsampled count. Without
// this pin a silent regression to "1 outer fold(s)" would fail nothing (close-review minor).
func TestSample_FastBannerRendersSampled(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, sampleShapeMD())

	var out strings.Builder
	_, err := runExperiment(runOpts{
		expPath: expPath, now: fixedNow(),
		git:  fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		exec: foldFakeExec{}, out: &out,
		fast: true,
	})
	if err != nil {
		t.Fatalf("--fast run should succeed: %v", err)
	}
	if s := out.String(); !strings.Contains(s, "1/2 outer fold(s)") {
		t.Errorf("--fast banner should render the outer level as sampled 1/k, got:\n%s", s)
	}
}
