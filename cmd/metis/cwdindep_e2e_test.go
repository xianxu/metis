package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_CwdIndependentIdentityAndLocation (metis#34): the SAME shape invoked the two
// documented ways — from inside the pipeline dir (`metis run exp.md`) and from the repo
// root (`metis run sub/pipelines/exp.md`) — lands outputs in the SAME physical runs dir
// and records the SAME point_address. Pins the pre-existing invariant (identity is
// content-addressed; anchors are Abs(Dir(expPath))-derived) against future drift.
func TestRun_CwdIndependentIdentityAndLocation(t *testing.T) {
	root := t.TempDir()
	pipe := filepath.Join(root, "sub", "pipelines")
	stepDir := filepath.Join(root, "steps", "test")
	for _, d := range []string{pipe, stepDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script := "#!/bin/sh\necho '{\"ok\": 1}' > metrics.json\n"
	if err := os.WriteFile(filepath.Join(stepDir, "echoer"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	exp := "---\ntype: experiment\nid: cwd-e2e\nseed: 1\nstatus: active\nsteps:\n  - id: e\n    uses: test/echoer\n---\n# cwd e2e\n"
	if err := os.WriteFile(filepath.Join(pipe, "exp.md"), []byte(exp), 0o644); err != nil {
		t.Fatal(err)
	}

	pointAddr := func(runID string) string {
		b, err := os.ReadFile(filepath.Join(pipe, "runs", runID, "record.json"))
		if err != nil {
			t.Fatalf("record.json for %s not under the pipeline dir's runs/: %v", runID, err)
		}
		var rec struct {
			PointAddress string `json:"point_address"`
		}
		if err := json.Unmarshal(b, &rec); err != nil {
			t.Fatal(err)
		}
		return rec.PointAddress
	}

	// invocation 1: from inside the pipeline dir, bare filename
	t.Chdir(pipe)
	if _, err := runExperiment(runOpts{expPath: "exp.md", runID: "r-from-pipe", stepPath: []string{filepath.Join(root, "steps")}, out: discardWriter(t)}); err != nil {
		t.Fatalf("pipeline-dir invocation: %v", err)
	}
	// invocation 2: from the repo root, root-relative path
	t.Chdir(root)
	if _, err := runExperiment(runOpts{expPath: filepath.Join("sub", "pipelines", "exp.md"), runID: "r-from-root", stepPath: []string{filepath.Join(root, "steps")}, out: discardWriter(t)}); err != nil {
		t.Fatalf("repo-root invocation: %v", err)
	}

	a, b := pointAddr("r-from-pipe"), pointAddr("r-from-root")
	if a == "" || a != b {
		t.Errorf("point_address differs by cwd: %q vs %q — identity must be content-addressed", a, b)
	}
}

func discardWriter(t *testing.T) *strings.Builder { t.Helper(); return &strings.Builder{} }
