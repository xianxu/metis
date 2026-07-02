package experiment

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestParse_ConformsToCUE is the ARCH-DRY drift guard: the Go structs in this
// package restate the CUE #Experiment shape (construct/vocabulary/experiment.cue,
// the single structural source). This test round-trips valid-baseline.md through
// Parse AND asserts the inherited ariadne `vocabulary validate-instance` binary
// still accepts the same fixture, so the Go structs cannot silently diverge from
// the CUE source without one side failing here.
//
// It SKIPS (not fails) when the toolchain is unavailable in a bare checkout — the
// vocabulary binary must be built at ../ariadne/bin and `cue` must be on PATH — so
// `go test ./...` stays green where the drift guard simply cannot run.
func TestParse_ConformsToCUE(t *testing.T) {
	root := repoRoot(t)
	fixture := filepath.Join(root, "testdata", "experiment", "valid-baseline.md")

	// Half 1 — the Go parser accepts the fixture.
	content, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(string(content)); err != nil {
		t.Fatalf("Parse rejected valid-baseline: %v", err)
	}

	// Half 2 — the CUE validator still accepts the same fixture (drift guard).
	vocab := filepath.Join(root, "..", "ariadne", "bin", "vocabulary")
	if _, err := os.Stat(vocab); err != nil {
		t.Skipf("vocabulary binary not built at %s; skipping CUE drift guard", vocab)
	}
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue not on PATH; skipping CUE drift guard")
	}
	cmd := exec.Command(vocab, "validate-instance", "--type", "experiment", fixture)
	cmd.Dir = root // vocabulary resolves the layer graph from its cwd
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("vocabulary validate-instance rejected valid-baseline (CUE drift?): %v\n%s", err, out)
	}
}

// TestRunConformsToCUE is the ARCH-DRY drift guard for the Run ledger record: the
// Go Run struct restates the CUE #Run (construct/vocabulary/experiment.cue). Unlike
// #Experiment there is no markdown fixture (a Run is emitted as run.json, and #Run
// carries no `type` discriminator for `validate-instance`), so this marshals a
// representative Run to JSON and `cue vet`s it against the closed #Run definition —
// a renamed/removed/extra field would fail, so the struct can't silently drift.
// SKIPS when `cue` is unavailable, mirroring the #Experiment guard.
func TestRunConformsToCUE(t *testing.T) {
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue not on PATH; skipping #Run drift guard")
	}
	root := repoRoot(t)
	cueFile := filepath.Join(root, "construct", "vocabulary", "experiment.cue")

	run := Run{
		ID:         "run-001",
		Experiment: "some-experiment",
		Seed:       42,
		Started:    "2026-07-01T00:00:00Z",
		Finished:   "2026-07-01T00:00:05Z",
		Status:     "ok",
		Metrics:    map[string]float64{"cv_score": 0.9},
		Artifacts:  []string{"split/folds.json"},
	}
	b, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "run.json")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("cue", "vet", "-d", "#Run", tmp, cueFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cue vet rejected a valid Run against #Run (CUE drift?): %v\n%s", err, out)
	}
}
