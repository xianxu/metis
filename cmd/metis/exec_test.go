package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
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

// TestCollectArtifacts_RecursiveExcludesReserved covers the M2-deferred fix: the
// collector now recurses into subdirectories, and excludes with.json/metrics.json
// only at the step-dir TOP level (a nested sub/metrics.json is a real artifact).
func TestCollectArtifacts_RecursiveExcludesReserved(t *testing.T) {
	runDir := t.TempDir()
	stepDir := filepath.Join(runDir, "s")
	if err := os.MkdirAll(filepath.Join(stepDir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writes := map[string]string{
		"with.json":        "{}", // reserved (top level) — excluded
		"metrics.json":     "{}", // reserved (top level) — excluded
		"reads.json":       "{}", // metis#2 sensor sidecar (top level) — excluded
		"top.txt":          "x",  // artifact
		"sub/nested.csv":   "y",  // artifact (recursion)
		"sub/metrics.json": "{}", // artifact (nested — NOT reserved)
	}
	for rel, body := range writes {
		if err := os.WriteFile(filepath.Join(stepDir, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	arts, err := collectArtifacts(stepDir, runDir)
	if err != nil {
		t.Fatalf("collectArtifacts: %v", err)
	}
	want := []string{"s/sub/metrics.json", "s/sub/nested.csv", "s/top.txt"}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("artifacts = %v; want %v", arts, want)
	}
}
