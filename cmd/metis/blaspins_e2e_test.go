package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_BlasPinsDefaultAndNote (metis#48 e2e): a bare run (no ambient pins) announces
// ONE loud note and the leaf subprocess sees the pins; an operator-exported value passes
// through untouched and drops out of the note. Drives runExperiment (the real wiring:
// blasPins → runOpts.leafPins → execStep), real shell step, no uv.
//
// Sweep-path once-ness is by construction (verified at plan review): runExperiment is
// entered exactly once per invocation (sole caller main.go) and all nested sweep spawns
// are struct copies carrying the computed leafPins — so this single-run count==1 plus
// that structure covers the sweep path.
func TestRun_BlasPinsDefaultAndNote(t *testing.T) {
	// ambient: exactly ONE operator choice set; the other three genuinely absent.
	// t.Setenv registers the restore; Unsetenv then makes absence real (an operator
	// shell following the old RUNBOOK exports all four — CI must not inherit that).
	for _, k := range []string{"OPENBLAS_NUM_THREADS", "VECLIB_MAXIMUM_THREADS", "MKL_NUM_THREADS"} {
		t.Setenv(k, "sentinel")
		os.Unsetenv(k)
	}
	t.Setenv("OMP_NUM_THREADS", "7")

	root := t.TempDir()
	stepDir := filepath.Join(root, "steps", "test")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nenv > env.txt\necho '{\"ok\": 1}' > metrics.json\n"
	if err := os.WriteFile(filepath.Join(stepDir, "envstep"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	expPath := filepath.Join(root, "exp.md")
	// plain-experiment fixture — steps live in the YAML FRONTMATTER (the
	// testdata/experiment/run-echo.md convention)
	exp := "---\ntype: experiment\nid: pins-e2e\nseed: 1\nstatus: active\nsteps:\n  - id: e\n    uses: test/envstep\n---\n# pins e2e\n"
	if err := os.WriteFile(expPath, []byte(exp), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if _, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "r1",
		stepPath: []string{filepath.Join(root, "steps")},
		out:      &out,
	}); err != nil {
		t.Fatalf("run: %v\n%s", err, out.String())
	}

	// (a) ONE note, naming only the three injected pins (OMP dropped — operator won)
	if n := strings.Count(out.String(), "metis: leaf BLAS pinned"); n != 1 {
		t.Fatalf("want exactly 1 pin note, got %d:\n%s", n, out.String())
	}
	if strings.Contains(out.String(), "OMP_NUM_THREADS") {
		t.Errorf("note must not claim the operator-set var:\n%s", out.String())
	}
	// (b) child env: three pins at 1, operator's OMP at 7
	b, err := os.ReadFile(filepath.Join(root, "runs", "r1", "e", "env.txt"))
	if err != nil {
		t.Fatal(err)
	}
	envTxt := string(b)
	for _, want := range []string{"OPENBLAS_NUM_THREADS=1", "VECLIB_MAXIMUM_THREADS=1", "MKL_NUM_THREADS=1", "OMP_NUM_THREADS=7"} {
		if !strings.Contains(envTxt, want) {
			t.Errorf("child env missing %q; got:\n%s", want, envTxt)
		}
	}
}
