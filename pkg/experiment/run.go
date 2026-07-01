package experiment

import (
	"fmt"
	"time"
)

// StepResult is one step's outcome as reported by a StepExecutor: the metrics it
// emitted and the artifact paths it wrote (repo-relative, under runs/<id>/).
type StepResult struct {
	Metrics   map[string]float64
	Artifacts []string
}

// StepExecutor is the injected seam between the pure orchestration (Runner.Run)
// and the actual step execution. The production impl lives in cmd/metis and shells
// out to a subprocess (files + subprocess, never FFI); tests inject a fake, so
// Runner.Run is exercised with NO subprocess. This is the ARCH-PURE line: the
// wiring below is pure/thin; all step IO sits behind this interface.
type StepExecutor interface {
	// Execute runs one step against the given run directory and returns its result.
	Execute(step Step, runDir string) (StepResult, error)
}

// Clock returns the current time. Injected so Run's timestamps are deterministic
// in tests — no direct wall-clock calls in the core (controllable time as
// architecture, not a test-only bolt-on).
type Clock func() time.Time

// Runner orchestrates one experiment execution: Validate → TopoSort → execute each
// step in dependency order via the injected StepExecutor → assemble the Run ledger
// record. The orchestration is pure/thin; every side effect is behind StepExecutor.
type Runner struct {
	Exec StepExecutor
	Now  Clock // optional; defaults to time.Now
}

// Run validates exp (execution-time enforcement of the semantic checks M1
// deferred), orders its steps, executes each through the StepExecutor, and
// assembles the Run record — merging each step's metrics and collecting its
// artifacts in execution order. runDir is where step artifacts land (runs/<id>/).
// It stops at the first step error, returning a "failed" Run and the error.
func (r Runner) Run(exp Experiment, runID, runDir string) (Run, error) {
	now := r.Now
	if now == nil {
		now = time.Now
	}
	stamp := func() string { return now().UTC().Format(time.RFC3339) }

	if err := Validate(exp); err != nil {
		return Run{}, fmt.Errorf("invalid experiment %q: %w", exp.ID, err)
	}
	order, err := TopoSort(exp)
	if err != nil {
		// Unreachable once Validate passes (it delegates acyclicity to TopoSort),
		// but keep the guard honest rather than dropping the error.
		return Run{}, err
	}

	run := Run{
		ID:         runID,
		Experiment: exp.ID,
		Seed:       exp.Seed,
		Started:    stamp(),
		Status:     "ok",
		Metrics:    map[string]float64{},
	}
	for _, step := range order {
		res, err := r.Exec.Execute(step, runDir)
		if err != nil {
			run.Status = "failed"
			run.Finished = stamp()
			return run, fmt.Errorf("step %q: %w", step.ID, err)
		}
		for k, v := range res.Metrics {
			run.Metrics[k] = v // flat merge; move-1 sequential runner, last-write-wins
		}
		run.Artifacts = append(run.Artifacts, res.Artifacts...)
	}
	run.Finished = stamp()
	if len(run.Metrics) == 0 {
		run.Metrics = nil // keep the ledger JSON clean when no step emitted metrics
	}
	return run, nil
}
