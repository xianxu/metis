package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/shape"
)

// The titanic-baseline-shape fixture expands to exactly 21 points (as its body
// documents: features(3) × [logreg:C(3) + rf:(2×2)=4] = 3 × 7 = 21) — asserting the
// "validates AND expands" claim on the real committed fixture.
func TestShapeFixture_ExpandsTo21Points(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "experiment", "titanic-baseline-shape.md"))
	if err != nil {
		t.Fatal(err)
	}
	sh, err := experiment.ParseShape(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	points, err := shape.Expand(sh.Steps, sh.Sweep.RangeSteps)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 21 {
		t.Errorf("titanic-baseline-shape should expand to 21 points, got %d", len(points))
	}
}

// A singleton experiment-shape (every knob pinned) runs exactly like a v0 experiment
// through `metis run` — the #Experiment = #ExperimentShape & all-singleton collapse,
// made runnable. Uses the no-uv test/echo steps.
func TestRunShape_SingletonRunsLikeExperiment(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := filepath.Join(ws, "shape.md")
	md := `---
type: experiment-shape
id: singleton
seed: 5
status: active
sweep:
  sampler: grid
  objective: {metric: echoed, direction: maximize}
steps:
  - id: prep
    uses: test/echo
    with: {k: 5}
  - id: train
    uses: test/echo
    needs: [prep]
    with: {model: logreg}
---
`
	if err := os.WriteFile(expPath, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	run, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "s1",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("singleton shape should run like an experiment: %v", err)
	}
	if run.Status != "ok" {
		t.Errorf("run status = %q; want ok", run.Status)
	}
	// The resolved record shows the pinned singleton config (knob→score).
	body, _ := os.ReadFile(expPath)
	if !strings.Contains(string(body), "prep.k=5") {
		t.Errorf("## Runs should show the resolved singleton config; got:\n%s", body)
	}
}

// (The old TestRunShape_MultiPointPointsToSweeper — asserting a multi-point shape was
// refused — is superseded by metis#7's sweep driver in caching_sweep_test.go, which
// runs the sweep instead.)
