package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
)

// runOpts are the inputs to one `metis run`. now/git/out are injected so the e2e
// test gets a deterministic clock, a fake git probe, and can discard progress output.
type runOpts struct {
	expPath  string
	runID    string
	stepPath []string
	now      func() time.Time
	git      gitProbe
	out      io.Writer
}

// runExperiment reads the experiment at o.expPath, runs it through the pure
// pkg/experiment.Runner wired to the real subprocess StepExecutor, writes
// runs/<id>/run.json, and appends a summary to the experiment's `## Runs` log. All
// side effects (read, subprocess, write) live here; the ordering/validation logic
// stays in pkg/experiment. Returns the assembled Run and the run error (if any),
// after the ledger is written — so a failed run is still recorded.
func runExperiment(o runOpts) (experiment.Run, error) {
	now := o.now
	if now == nil {
		now = time.Now
	}
	out := o.out
	if out == nil {
		out = io.Discard
	}

	raw, err := os.ReadFile(o.expPath)
	if err != nil {
		return experiment.Run{}, err
	}
	exp, err := experiment.Parse(string(raw))
	if err != nil {
		return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
	}

	runID := o.runID
	if runID == "" {
		runID = "run-" + now().UTC().Format("20060102T150405Z")
	}
	baseDir := filepath.Dir(o.expPath)
	// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
	// the child's env, and the child's cwd IS the step dir — a relative path would
	// resolve $METIS_STEP_DIR/with.json under itself. Absolute paths are correct
	// from any cwd, so `metis run pipelines/foo.md` (a relative arg) works.
	runDir, err := filepath.Abs(filepath.Join(baseDir, "runs", runID))
	if err != nil {
		return experiment.Run{}, err
	}
	expDir, err := filepath.Abs(baseDir)
	if err != nil {
		return experiment.Run{}, err
	}

	runner := experiment.Runner{
		Exec: execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, out: out},
		Now:  now,
	}
	fmt.Fprintf(out, "metis: run %s of experiment %q\n", runID, exp.ID)
	run, steps, runErr := runner.Run(exp, runID, runDir)

	// Execution-time enforcement: Runner.Run validates the experiment BEFORE any
	// step executes, so a semantically-invalid experiment (dangling needs, bad
	// uses, a cycle) is rejected here — closing the SHAPE-only gap M1 left. Such a
	// rejection never started a run (run.Started is empty), so surface the error
	// without writing a bogus ledger or touching the ## Runs log.
	if run.Started == "" {
		return run, runErr
	}

	// Write the ledger even on a mid-run step failure (status=failed) so every
	// attempt that began is recorded — the record of truth is runs/<id>/run.json.
	if err := writeRunJSON(runDir, run); err != nil {
		return run, err
	}
	// Assemble + persist the provenance record (metis#3): repo provenance, per-step
	// output hashes, and the minted point-address. A config that can't be
	// canonicalized (e.g. a non-finite value) surfaces here as a run error.
	rec, err := assembleRecord(o.git, out, expDir, runDir, run, steps)
	if err != nil {
		return run, err
	}
	if err := writeRecordJSON(runDir, rec); err != nil {
		return run, err
	}
	if err := appendRunLog(o.expPath, rec); err != nil {
		return run, err
	}
	if runErr != nil {
		return run, runErr
	}
	fmt.Fprintf(out, "metis: %s %s\n", run.ID, run.Status)
	return run, nil
}

func writeRunJSON(runDir string, run experiment.Run) error {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "run.json"), append(b, '\n'), 0o644)
}

// appendRunLog appends a one-line knob→score summary (from the provenance record) to
// the experiment's `## Runs` section (creating the heading if absent). The
// human-readable bullet; the machine records are runs/<id>/{run,record}.json.
func appendRunLog(expPath string, rec record.RunRecord) error {
	raw, err := os.ReadFile(expPath)
	if err != nil {
		return err
	}
	body := string(raw)
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	if !strings.Contains(body, "## Runs") {
		body += "\n## Runs\n"
	}
	return os.WriteFile(expPath, []byte(body+"- "+recordSummary(rec)+"\n"), 0o644)
}
