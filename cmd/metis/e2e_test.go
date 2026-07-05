package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

// TestToyPipeline_EndToEnd is the metis#1 M3 Done-when: `metis run` a real toy
// experiment end-to-end through the uv/Python data plane and confirm it produces
// a real CV score, a submission file, and an ok ledger. It drives the REAL
// steps/metis/* wrappers (not a fake), so it validates the whole Go-runner →
// subprocess → Python thread and the upstream-artifact data-flow.
//
// Skips when uv is absent (the hermetic Python env can't run); the venv must be
// synced (`uv sync`) so the per-step `uv run` is fast and offline.
func TestToyPipeline_EndToEnd(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH; skipping the Python data-plane e2e")
	}
	root := repoRoot(t)

	// Recreate the committed experiment/ + dataset/ sibling layout in a temp
	// workspace so the run's artifacts and the ## Runs append never touch testdata/.
	ws := t.TempDir()
	expDir := filepath.Join(ws, "experiment")
	if err := os.MkdirAll(expDir, 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, filepath.Join(root, "testdata", "experiment", "toy-pipeline.md"),
		filepath.Join(expDir, "toy-pipeline.md"))
	copyDir(t, filepath.Join(root, "testdata", "dataset", "toy"),
		filepath.Join(ws, "dataset", "toy"))

	run, err := runExperiment(runOpts{
		expPath:  filepath.Join(expDir, "toy-pipeline.md"),
		runID:    "run-e2e",
		stepPath: []string{filepath.Join(root, "steps")}, // the real metis/* step-types
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "toysha", dirty: false}, // workspace is a bare TempDir, not a git repo
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("runExperiment: %v", err)
	}
	if run.Status != "ok" {
		t.Fatalf("run status = %q; want ok", run.Status)
	}
	if cv := run.Metrics["cv_score"]; cv <= 0.5 {
		t.Errorf("cv_score = %v; want a real score > 0.5", cv)
	}

	// predictions.csv written by the predict step (the submission-shaped output).
	preds := filepath.Join(expDir, "runs", "run-e2e", "predict", "predictions.csv")
	if _, err := os.Stat(preds); err != nil {
		t.Fatalf("predictions.csv not written: %v", err)
	}

	// run.json is the record of truth: status ok, carries the CV score.
	rb, err := os.ReadFile(filepath.Join(expDir, "runs", "run-e2e", "run.json"))
	if err != nil {
		t.Fatalf("read run.json: %v", err)
	}
	var got experiment.Run
	if err := json.Unmarshal(rb, &got); err != nil {
		t.Fatalf("parse run.json: %v", err)
	}
	if got.Status != "ok" || got.Metrics["cv_score"] <= 0.5 {
		t.Errorf("run.json wrong: %+v", got)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue // toy dataset is flat
		}
		copyFile(t, filepath.Join(src, ent.Name()), filepath.Join(dst, ent.Name()))
	}
}
