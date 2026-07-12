package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/xianxu/metis/pkg/experiment"
)

// execStep is the production StepExecutor: it shells out to a step-type executable
// (files + subprocess, never FFI). It is the thin IO impl of the
// pkg/experiment.StepExecutor seam — the pure orchestration is Runner.Run.
//
// Contract per step (what every real step-type honors):
//   - working dir = <runDir>/<stepID>/, created here;
//   - input: with.json (the step's `with` config) written into that dir;
//   - env: METIS_STEP_DIR / METIS_RUN_DIR / METIS_STEP_ID / METIS_EXP_DIR / METIS_SEED;
//   - output: an optional metrics.json (flat {name: number}) + any artifact files
//     the step writes into its dir, which execStep reads back.
type execStep struct {
	stepPath []string  // dirs searched for <layer>/<steptype>
	expDir   string    // absolute experiment dir; anchor for exp-relative step inputs
	seed     int       // the experiment's seed, exposed to every step for reproducibility
	readRoot string    // metis#23: outer-fold analysis root; when set, confines base-dataset reads (empty = unconfined)
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
		"METIS_EXP_DIR="+e.expDir,
		"METIS_SEED="+strconv.Itoa(e.seed),
	)
	if e.readRoot != "" {
		// metis#23 confinement: only inject when sealing an outer-fold sweep, so the
		// flat driver:single path leaves the var unset (unconfined).
		cmd.Env = append(cmd.Env, "METIS_READ_ROOT="+e.readRoot)
	}
	if combined, err := cmd.CombinedOutput(); err != nil {
		// Runner.Run already prefixes `step %q:`; name the executable, not the id
		// again, to avoid a doubled "step first: step first" prefix.
		return experiment.StepResult{}, fmt.Errorf("exec %s: %w\n%s", exe, err, combined)
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

// collectArtifacts lists the artifact files a step wrote in its dir — recursively,
// so a step that writes nested outputs (e.g. sub/*.parquet) has them all recorded —
// as slash paths relative to runDir (i.e. under runs/<id>/). The reserved CONTRACT/
// bookkeeping channels are excluded, but only at the step-dir TOP level: with.json
// (the input config), metrics.json (already parsed into run.Metrics), and reads.json
// (the metis#2 read-sensor's sidecar — bookkeeping, NOT an output; folding it into the
// output-hash would make an upstream code edit bust all downstream even when the real
// output is byte-identical, and its absolute project_root would defeat cross-machine
// cache reuse). A file a step happens to write at a nested path like sub/metrics.json
// is a genuine artifact. Sorted for a deterministic ledger.
func collectArtifacts(stepDir, runDir string) ([]string, error) {
	var arts []string
	err := filepath.WalkDir(stepDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Dir(path) == stepDir {
			if name := d.Name(); name == "with.json" || name == "metrics.json" || name == "reads.json" {
				return nil // reserved contract / bookkeeping channels (top level only)
			}
		}
		rel, err := filepath.Rel(runDir, path)
		if err != nil {
			return err
		}
		arts = append(arts, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(arts)
	return arts, nil
}
