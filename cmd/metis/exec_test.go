package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
)

// TestExecStep_InjectsEnv asserts the subprocess executor sets the full step
// contract env — including the M3 additions METIS_EXP_DIR (the experiment-dir
// anchor for exp-relative inputs) and METIS_SEED (the experiment's seed) — so a
// step-type can resolve committed inputs and be reproducible without threading
// the seed through every `with`. Uses the test/env-dump fake step (no uv).
func TestExecStep_InjectsEnv(t *testing.T) {
	root := repoRoot(t)
	runDir := t.TempDir()
	e := execStep{
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		expDir:   "/anchor/exp",
		seed:     99,
		out:      io.Discard,
	}

	_, err := e.Execute(experiment.Step{ID: "e", Uses: "test/env-dump"}, runDir)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(runDir, "e", "env.txt"))
	if err != nil {
		t.Fatalf("read env.txt: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		"STEP_ID=e",
		"EXP_DIR=/anchor/exp",
		"SEED=99",
		"RUN_DIR=" + runDir,
		"STEP_DIR=" + filepath.Join(runDir, "e"),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("env.txt missing %q; got:\n%s", want, got)
		}
	}
}
