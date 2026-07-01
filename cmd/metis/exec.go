package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xianxu/metis/pkg/experiment"
)

// execStep is the production StepExecutor: it shells out to a step-type executable
// (files + subprocess, never FFI). It is the thin IO impl of the
// pkg/experiment.StepExecutor seam — the pure orchestration is Runner.Run.
//
// Contract per step (what every real step-type honors):
//   - working dir = <runDir>/<stepID>/, created here;
//   - input: with.json (the step's `with` config) written into that dir;
//   - env: METIS_STEP_DIR / METIS_RUN_DIR / METIS_STEP_ID;
//   - output: an optional metrics.json (flat {name: number}) + any artifact files
//     the step writes into its dir, which execStep reads back.
type execStep struct {
	stepPath []string  // dirs searched for <layer>/<steptype>
	out      io.Writer // plain streaming progress
}

func (e execStep) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	exe, err := e.resolve(step.Uses)
	if err != nil {
		return experiment.StepResult{}, err
	}
	stepDir := filepath.Join(runDir, step.ID)
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		return experiment.StepResult{}, err
	}

	with := step.With
	if with == nil {
		with = map[string]any{}
	}
	withJSON, err := json.MarshalIndent(with, "", "  ")
	if err != nil {
		return experiment.StepResult{}, err
	}
	if err := os.WriteFile(filepath.Join(stepDir, "with.json"), append(withJSON, '\n'), 0o644); err != nil {
		return experiment.StepResult{}, err
	}

	fmt.Fprintf(e.out, "→ step %s (uses %s)\n", step.ID, step.Uses)
	cmd := exec.Command(exe)
	cmd.Dir = stepDir
	cmd.Env = append(os.Environ(),
		"METIS_STEP_DIR="+stepDir,
		"METIS_RUN_DIR="+runDir,
		"METIS_STEP_ID="+step.ID,
	)
	if combined, err := cmd.CombinedOutput(); err != nil {
		return experiment.StepResult{}, fmt.Errorf("step %s (%s) failed: %w\n%s", step.ID, exe, err, combined)
	}

	metrics, err := readMetrics(filepath.Join(stepDir, "metrics.json"))
	if err != nil {
		return experiment.StepResult{}, err
	}
	artifacts, err := collectArtifacts(stepDir, runDir)
	if err != nil {
		return experiment.StepResult{}, err
	}
	fmt.Fprintf(e.out, "✓ step %s\n", step.ID)
	return experiment.StepResult{Metrics: metrics, Artifacts: artifacts}, nil
}

// resolve maps `uses: <layer>/<steptype>` to an executable by searching the step
// path in order; the first existing regular file wins.
func (e execStep) resolve(uses string) (string, error) {
	rel := filepath.FromSlash(uses)
	for _, dir := range e.stepPath {
		cand := filepath.Join(dir, rel)
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand, nil
		}
	}
	return "", fmt.Errorf("no step-type executable for uses %q on step path %v", uses, e.stepPath)
}

// readMetrics reads a step's optional metrics.json (flat {name: number}); absence
// is not an error (a step need not emit metrics).
func readMetrics(path string) (map[string]float64, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]float64
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// collectArtifacts lists the artifact files a step wrote in its dir, as slash
// paths relative to runDir (i.e. under runs/<id>/). The two CONTRACT channels are
// excluded: with.json (the input config) and metrics.json (already parsed into
// run.Metrics — it's the metrics channel, not an artifact).
func collectArtifacts(stepDir, runDir string) ([]string, error) {
	entries, err := os.ReadDir(stepDir)
	if err != nil {
		return nil, err
	}
	var arts []string
	for _, ent := range entries {
		if ent.IsDir() || ent.Name() == "with.json" || ent.Name() == "metrics.json" {
			continue
		}
		rel, err := filepath.Rel(runDir, filepath.Join(stepDir, ent.Name()))
		if err != nil {
			return nil, err
		}
		arts = append(arts, filepath.ToSlash(rel))
	}
	return arts, nil
}
