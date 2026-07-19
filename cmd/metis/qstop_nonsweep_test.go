package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestQFinalize_NonSweepIgnoresStopSignal pins the metis#67 broadening of the q-finalize gate.
// main.go now spawns the stdin `q` signal on ANY TTY run (gate is `o.tui` alone, was `o.live &&
// o.tui`), including a plain NON-sweep experiment. But a plain run has no nested outer-fold loop
// to finalize — sweep.go consumes `stopSignal` only on the nested path (`sweep.go:576`). So a
// plain run handed an ALREADY-FIRED stopSignal must complete normally, never hang or error on it.
func TestQFinalize_NonSweepIgnoresStopSignal(t *testing.T) {
	root := t.TempDir()
	stepDir := filepath.Join(root, "steps", "test")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stepDir, "echoer"),
		[]byte("#!/bin/sh\necho '{\"ok\": 1}' > metrics.json\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	pipe := filepath.Join(root, "pipelines")
	if err := os.MkdirAll(pipe, 0o755); err != nil {
		t.Fatal(err)
	}
	exp := "---\ntype: experiment\nid: q-noop\nseed: 1\nstatus: active\nsteps:\n  - id: e\n    uses: test/echoer\n---\n# q noop\n"
	expPath := filepath.Join(pipe, "exp.md")
	if err := os.WriteFile(expPath, []byte(exp), 0o644); err != nil {
		t.Fatal(err)
	}

	fired := make(chan struct{})
	close(fired) // already fired — the non-sweep path must ignore it, not hang or error
	if _, err := runExperiment(runOpts{
		expPath:    expPath,
		runID:      "r",
		stepPath:   []string{filepath.Join(root, "steps")},
		out:        discardWriter(t),
		stopSignal: fired,
	}); err != nil {
		t.Fatalf("a plain (non-sweep) run must ignore a fired stopSignal and complete cleanly, got: %v", err)
	}
}
