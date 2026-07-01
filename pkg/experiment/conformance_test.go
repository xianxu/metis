package experiment

import (
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
