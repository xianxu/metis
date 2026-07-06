package experiment

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/ariadne/pkg/frontmatter"
)

func TestParseShape_ReadsSweepBlock(t *testing.T) {
	md := `---
type: experiment-shape
id: titanic-sweep
competition: titanic
seed: 42
status: active
sweep:
  sampler: grid
  objective: {metric: cv_score, direction: maximize}
  range_steps: 6
steps:
  - id: adapt
    uses: titanic/adapt
    with:
      features: {$any: [[], [title]]}
  - id: train
    uses: metis/train
    needs: [adapt]
    with: {model: logreg}
---

# titanic-sweep
`
	sh, err := ParseShape(md)
	if err != nil {
		t.Fatal(err)
	}
	if sh.Type != "experiment-shape" || sh.ID != "titanic-sweep" || sh.Seed != 42 {
		t.Errorf("header wrong: %+v", sh)
	}
	if sh.Sweep.Sampler != "grid" || sh.Sweep.RangeSteps != 6 {
		t.Errorf("sweep block wrong: %+v", sh.Sweep)
	}
	if sh.Sweep.Objective.Metric != "cv_score" || sh.Sweep.Objective.Direction != "maximize" {
		t.Errorf("objective wrong: %+v", sh.Sweep.Objective)
	}
	if len(sh.Steps) != 2 {
		t.Fatalf("want 2 steps, got %d", len(sh.Steps))
	}
	// The $-descriptor survives into the untyped `with` bag for the expander.
	feat, ok := sh.Steps[0].With["features"].(map[string]any)
	if !ok || feat["$any"] == nil {
		t.Errorf("features $any descriptor not preserved: %#v", sh.Steps[0].With["features"])
	}
}

// TestShapeConformsToCUE is the drift guard for #ExperimentShape: the Go Shape struct
// (+ the titanic-baseline-shape fixture ParseShape accepts) must also validate against
// the CUE #ExperimentShape, so the two can't silently diverge. Skips when cue is absent.
func TestShapeConformsToCUE(t *testing.T) {
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue not on PATH; skipping #ExperimentShape drift guard")
	}
	root := repoRoot(t)
	fixture := filepath.Join(root, "testdata", "experiment", "titanic-baseline-shape.md")
	content, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseShape(string(content)); err != nil {
		t.Fatalf("ParseShape rejected the shape fixture: %v", err)
	}
	// Extract the frontmatter and cue-vet it against #ExperimentShape.
	fm, _, err := frontmatter.Split(string(content))
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "shape.yaml")
	if err := os.WriteFile(tmp, []byte(fm), 0o644); err != nil {
		t.Fatal(err)
	}
	cueFile := filepath.Join(root, "construct", "vocabulary", "experiment.cue")
	if out, err := exec.Command("cue", "vet", "-d", "#ExperimentShape", tmp, cueFile).CombinedOutput(); err != nil {
		t.Fatalf("cue vet rejected the shape fixture against #ExperimentShape (drift?): %v\n%s", err, out)
	}
}

// The single-source _pipeline embed must keep BOTH definitions closed: #Experiment
// must reject a stray `sweep` field, and #ExperimentShape must reject an unknown field.
// A future CUE edit (an accidental `...`, a mis-embed) would regress this silently — so
// assert it, not just the positive cases.
func TestCUE_ClosednessPreservedBySingleSource(t *testing.T) {
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue not on PATH")
	}
	root := repoRoot(t)
	cueFile := filepath.Join(root, "construct", "vocabulary", "experiment.cue")
	dir := t.TempDir()

	// A plain experiment carrying a stray `sweep` must FAIL #Experiment (closedness).
	expWithSweep := "type: experiment\nid: x\nseed: 1\nstatus: active\nsteps: []\n" +
		"sweep: {sampler: grid}\n"
	p1 := filepath.Join(dir, "exp.yaml")
	if err := os.WriteFile(p1, []byte(expWithSweep), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("cue", "vet", "-d", "#Experiment", p1, cueFile).Run(); err == nil {
		t.Error("#Experiment must REJECT a stray `sweep` field (closedness lost)")
	}

	// A shape with an unknown top-level field must FAIL #ExperimentShape.
	shapeStray := "type: experiment-shape\nid: x\nseed: 1\nstatus: active\nsteps: []\n" +
		"sweep: {sampler: grid}\nbogus_field: 1\n"
	p2 := filepath.Join(dir, "shape.yaml")
	if err := os.WriteFile(p2, []byte(shapeStray), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("cue", "vet", "-d", "#ExperimentShape", p2, cueFile).Run(); err == nil {
		t.Error("#ExperimentShape must REJECT an unknown field (closedness lost)")
	}
}

// A shape reuses the experiment DAG semantics (Validate): a dangling `needs` is caught.
func TestShape_ValidateReusesExperimentSemantics(t *testing.T) {
	sh := Shape{
		Experiment: Experiment{
			Type: "experiment-shape", ID: "bad", Seed: 1, Status: "active",
			Steps: []Step{{ID: "train", Uses: "metis/train", Needs: []string{"ghost"}}},
		},
		Sweep: Sweep{Sampler: "grid"},
	}
	if err := ValidateShape(sh); err == nil {
		t.Error("a shape with a dangling `needs` must fail validation (reusing experiment semantics)")
	}
}
