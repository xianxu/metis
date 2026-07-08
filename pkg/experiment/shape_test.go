package experiment

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/frontmatter"
)

// validShapeV2 is the canonical M1a shape: three phases with CROSS-PHASE needs
// (features(pipeline) needs adapt(data); predict(ship) needs train(pipeline)), a
// sweeper with an inner CV + argmax-mean select, and driver:single.
const validShapeV2 = `type: experiment-shape
id: titanic-sweep
competition: titanic
seed: 42
status: active
data:
  - id: adapt
    uses: titanic/adapt
    with: {out: ../data/titanic}
pipeline:
  - id: features
    uses: titanic/features
    needs: [adapt]
    with:
      dataset: adapt
      features: {$any: [[], [title]]}
  - id: train
    uses: metis/train
    needs: [features]
    with: {model: {$any: {logreg: {C: {$any: [0.1, 1]}}}}}
ship:
  - id: predict
    uses: metis/predict
    needs: [train]
sweeper:
  sampler: grid
  resample: {cv: {k: 5, stratify: true}}
  objective: {metric: accuracy, direction: maximize, select: argmax-mean}
driver:
  single: {}
`

func mdOf(fm string) string { return "---\n" + fm + "---\n\n# shape\n" }

// T1: the phase-structured shape parses into Data/Pipeline/Ship + Sweeper + Driver,
// the inner resample + select survive, driver:single is a non-nil pointer, and the
// $any descriptor survives untyped into the `with` bag for the expander.
func TestParseShape_v2(t *testing.T) {
	sh, err := ParseShape(mdOf(validShapeV2))
	if err != nil {
		t.Fatal(err)
	}
	if sh.Type != "experiment-shape" || sh.ID != "titanic-sweep" || sh.Seed != 42 {
		t.Errorf("header wrong: %+v", sh)
	}
	if len(sh.Data) != 1 || len(sh.Pipeline) != 2 || len(sh.Ship) != 1 {
		t.Fatalf("phase lengths wrong: data=%d pipeline=%d ship=%d", len(sh.Data), len(sh.Pipeline), len(sh.Ship))
	}
	if sh.Sweeper.Sampler != "grid" || sh.Sweeper.Resample.CV.K != 5 || !sh.Sweeper.Resample.CV.Stratify {
		t.Errorf("sweeper/resample wrong: %+v", sh.Sweeper)
	}
	if sh.Sweeper.Objective.Select != "argmax-mean" {
		t.Errorf("select wrong: %q", sh.Sweeper.Objective.Select)
	}
	if sh.Driver.Single == nil || sh.Driver.CV != nil {
		t.Errorf("driver:single expected: %+v", sh.Driver)
	}
	feat, ok := sh.Pipeline[0].With["features"].(map[string]any)
	if !ok || feat["$any"] == nil {
		t.Errorf("features $any descriptor not preserved: %#v", sh.Pipeline[0].With["features"])
	}
}

// T2: strict parse — an unknown top-level key OR an unknown sweeper sub-key is a loud
// error (KnownFields(true)), matching CUE's closed rejection instead of yaml's silent drop.
func TestParseShape_RejectsUnknownKey(t *testing.T) {
	cases := map[string]string{
		"unknown top-level":      validShapeV2 + "bogus_field: 1\n",
		"unknown sweeper subkey": strings.Replace(validShapeV2, "  sampler: grid\n", "  sampler: grid\n  sweeperr: oops\n", 1),
	}
	for name, fm := range cases {
		if _, err := ParseShape(mdOf(fm)); err == nil {
			t.Errorf("%s: expected an unknown-key error, got nil", name)
		}
	}
}

// T3: ValidateShape v2 — the valid shape passes, and each structural violation is caught.
// Crucially, cross-phase needs must RESOLVE (the combined-DAG check), while a dangling or
// cyclic need, duplicate ids across phases, and the shape-only invariants must all fail.
func TestValidateShape_v2(t *testing.T) {
	sh, err := ParseShape(mdOf(validShapeV2))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateShape(sh); err != nil {
		t.Fatalf("valid v2 shape rejected (cross-phase needs must resolve): %v", err)
	}

	// Each mutator should make ValidateShape fail.
	bad := map[string]func(s *Shape){
		"dangling cross-phase need":  func(s *Shape) { s.Pipeline[0].Needs = []string{"ghost"} },
		"duplicate id across phases": func(s *Shape) { s.Ship[0].ID = "train" },
		"empty pipeline":             func(s *Shape) { s.Pipeline = nil },
		"missing sampler":            func(s *Shape) { s.Sweeper.Sampler = "" },
		"resample k<2":               func(s *Shape) { s.Sweeper.Resample.CV.K = 1 },
		"bad direction":              func(s *Shape) { s.Sweeper.Objective.Direction = "sideways" },
		"unsupported select":         func(s *Shape) { s.Sweeper.Objective.Select = "one-std-err" },
		"driver none":                func(s *Shape) { s.Driver = Driver{} },
		"driver both":                func(s *Shape) { s.Driver.CV = &CVDriver{K: 5} },
		"driver cv (is #23)":         func(s *Shape) { s.Driver = Driver{CV: &CVDriver{K: 5}} },
	}
	for name, mut := range bad {
		s, err := ParseShape(mdOf(validShapeV2))
		if err != nil {
			t.Fatal(err)
		}
		mut(&s)
		if err := ValidateShape(s); err == nil {
			t.Errorf("%s: expected ValidateShape to fail, got nil", name)
		}
	}
}

// A backward cross-phase edge (a step depending on a LATER-phase step) must be rejected —
// it would violate the data│pipeline leakage cut. Distinct from acyclicity: the edge here
// is acyclic (an isolated ship step nothing else depends on) but phase-backward.
func TestValidateShape_RejectsBackwardPhaseEdge(t *testing.T) {
	sh, err := ParseShape(mdOf(validShapeV2))
	if err != nil {
		t.Fatal(err)
	}
	sh.Ship = append(sh.Ship, Step{ID: "extra", Uses: "titanic/submission"})
	sh.Data[0].Needs = []string{"extra"} // data(0) → ship(2): acyclic, but phase-backward
	err = ValidateShape(sh)
	if err == nil {
		t.Fatal("expected a backward-phase-edge error, got nil")
	}
	if !strings.Contains(err.Error(), "phase") && !strings.Contains(err.Error(), "leakage") {
		t.Errorf("error should name the phase/leakage violation, got: %v", err)
	}
}

// The empty-pipeline guard must be exercised in ISOLATION — the T3 mutator nils Pipeline
// on the full fixture, so `predict needs [train]` goes dangling and Validate fails FIRST
// (masking the guard). Here Pipeline+Ship are dropped so the DAG is valid and only the
// pipeline-required guard can fire; reverting the guard must fail this test.
func TestValidateShape_EmptyPipelineGuard(t *testing.T) {
	sh, err := ParseShape(mdOf(validShapeV2))
	if err != nil {
		t.Fatal(err)
	}
	sh.Pipeline = nil
	sh.Ship = nil // predict needed train (now gone) — drop ship so there's no dangling need
	err = ValidateShape(sh)
	if err == nil {
		t.Fatal("expected an empty-pipeline error, got nil")
	}
	if !strings.Contains(err.Error(), "pipeline") {
		t.Errorf("expected pipeline-required error, got: %v", err)
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
	sh, err := ParseShape(string(content))
	if err != nil {
		t.Fatalf("ParseShape rejected the shape fixture: %v", err)
	}
	if err := ValidateShape(sh); err != nil {
		t.Fatalf("ValidateShape rejected the shape fixture: %v", err)
	}
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

// The closed definitions must reject stray fields: #Experiment must reject a `sweeper`,
// and #ExperimentShape must reject an unknown field. A future CUE edit (an accidental
// `...`, a mis-embed) would regress this silently — so assert it.
func TestCUE_ClosednessPreservedBySingleSource(t *testing.T) {
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue not on PATH")
	}
	root := repoRoot(t)
	cueFile := filepath.Join(root, "construct", "vocabulary", "experiment.cue")
	dir := t.TempDir()

	// A plain experiment carrying a stray `sweeper` must FAIL #Experiment (closedness).
	expStray := "type: experiment\nid: x\nseed: 1\nstatus: active\nsteps: []\nsweeper: {sampler: grid}\n"
	p1 := filepath.Join(dir, "exp.yaml")
	if err := os.WriteFile(p1, []byte(expStray), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("cue", "vet", "-d", "#Experiment", p1, cueFile).Run(); err == nil {
		t.Error("#Experiment must REJECT a stray `sweeper` field (closedness lost)")
	}

	// A v2 shape with an unknown top-level field must FAIL #ExperimentShape.
	p2 := filepath.Join(dir, "shape.yaml")
	if err := os.WriteFile(p2, []byte(validShapeV2+"bogus_field: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("cue", "vet", "-d", "#ExperimentShape", p2, cueFile).Run(); err == nil {
		t.Error("#ExperimentShape must REJECT an unknown field (closedness lost)")
	}
}
