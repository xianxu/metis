package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/metis/internal/repo"
	"github.com/xianxu/metis/pkg/cas"
	"github.com/xianxu/metis/pkg/experiment"
)

// cacheProjectRoot resolves the metis code root (the module dir above steps/) that D
// paths are relative to and `git hash-object` runs in — the same root metis.trace
// records in reads.json. Falls back to the experiment dir if step paths don't resolve.
func cacheProjectRoot(stepPath []string, fallback string) string {
	for _, p := range stepPath {
		if root, err := repo.Root(p); err == nil {
			return root
		}
	}
	return fallback
}

// ensureCacheGitignore writes .metis-cache/.gitignore so the local, wipeable cache
// (content-addressed output blobs) is never committed to the experiment's repo — the
// cache is safe to `rm -rf` and rebuild. Idempotent. (Sharing the git-trackable index
// across clones is a future enhancement; v1 ignores the whole cache dir.)
func ensureCacheGitignore(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	gi := filepath.Join(cacheDir, ".gitignore")
	if _, err := os.Stat(gi); err == nil {
		return nil
	}
	body := "# metis#2 step cache — a local, wipeable content-addressed cache (rm -rf is safe).\n" +
		"# Never commit its output blobs.\n*\n"
	return os.WriteFile(gi, []byte(body), 0o644)
}

// runOpts are the inputs to one `metis run`. now/git/out are injected so the e2e
// test gets a deterministic clock, a fake git probe, and can discard progress output.
type runOpts struct {
	expPath  string
	runID    string
	stepPath []string
	now      func() time.Time
	git      gitProbe
	cache    bool // enable the metis#2 validating-trace cache (<expDir>/.metis-cache)
	dryRun   bool // metis#18: list the swept configs without running them
	inSweep  bool // metis#14: this run is a sweep point — suppress per-point single-run
	//               capture (the sweep captures once per shape-run in captureSweepCode)
	out  io.Writer
	exec experiment.StepExecutor // test seam: an injected fake replaces the subprocess
	//                              execStep (nil → the production execStep). Composes with
	//                              cache: the caching decorator still wraps it.
	readRoot string // metis#23: when set, the production execStep confines base-dataset reads to this root
}

// runExperiment reads the experiment at o.expPath and dispatches: a `type:
// experiment-shape` is the metis#18 nested-Sampler SWEEP (the sweeper grids over configs,
// the inner resample folds each — runShapeSweep); a plain `type: experiment` is the
// one-point path (runResolvedExperiment). The `.md` is immutable input (#13) — never
// written back; all side effects live in the shell below, the ordering/validation logic
// stays in pkg/experiment. Returns the assembled Run (empty for a sweep — the manifest +
// per-fold records + ledger are its output) and the run error.
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
	// Peek the type with the tolerant experiment parser (it ignores the shape-only
	// data/pipeline/ship/sweeper keys); a shape then re-parses through the STRICT
	// ParseShape (unknown-key-loud) for the sweep path.
	exp, err := experiment.Parse(string(raw))
	if err != nil {
		return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
	}
	if exp.Type == "experiment-shape" {
		sh, err := experiment.ParseShape(string(raw))
		if err != nil {
			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
		}
		if err := experiment.ValidateShape(sh); err != nil {
			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
		}
		return experiment.Run{}, runShapeSweep(o, sh, now, out)
	}
	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
}

// singleRunID names a single run's dir. metis#27: content-address it by the run's
// point-address (symmetric with a sweep point's dir), so the dir name IS the run identity.
// An explicit --run overrides; the timestamp form survives only as the no-git fallback
// (when the shape blob-hash — hence the point-address — can't be computed).
func singleRunID(o runOpts, exp experiment.Experiment, now func() time.Time) string {
	if o.runID != "" {
		return o.runID
	}
	sbh, err := shapeBlobHash(o.expPath)
	if err == nil {
		if addr, err := pointAddressOf(exp, sbh); err == nil {
			return addr
		}
	}
	return "run-" + now().UTC().Format("20060102T150405Z")
}

// runResolvedExperiment runs one already-resolved experiment (a single point) under
// runID, through the cached runner, and writes its run.json + provenance record (the
// experiment `.md` is immutable input — not written back, #13). The shared per-point runner
// both the 1-point path and the sweep loop (metis#7) call — so the run/cache/record wiring
// lives in ONE place (ARCH-DRY).
func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
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

	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out}
	if o.exec != nil {
		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
	}
	if o.cache {
		cacheDir := filepath.Join(expDir, ".metis-cache")
		if err := ensureCacheGitignore(cacheDir); err != nil {
			return experiment.Run{}, err
		}
		store := cas.NewFSStore(filepath.Join(cacheDir, "cas"), 0, cas.Clock(now))
		exec = newCachingExecutor(exec, store, cacheDir, exp.Seed, out)
	}
	runner := experiment.Runner{Exec: exec, Now: now}
	fmt.Fprintf(out, "metis: run %s of experiment %q\n", runID, exp.ID)
	run, steps, runErr := runner.Run(exp, runID, runDir)

	// Execution-time enforcement: Runner.Run validates the experiment BEFORE any
	// step executes, so a semantically-invalid experiment (dangling needs, bad
	// uses, a cycle) is rejected here — closing the SHAPE-only gap M1 left. Such a
	// rejection never started a run (run.Started is empty), so surface the error
	// without writing a bogus record.
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
	// The shape's blob-hash content-addresses the intent (metis#27); computed the SAME way
	// singleRunID/pointAddressOf did, so the record's point_address matches the run dir.
	// A no-git spec yields "" (a degraded, non-content-addressed run — warned via capture status).
	sbh, _ := shapeBlobHash(o.expPath)
	rec, err := assembleRecord(o.git, out, expDir, runDir, exp, run, steps, sbh)
	if err != nil {
		return run, err
	}
	if err := writeRecordJSON(runDir, rec); err != nil {
		return run, err
	}
	// Capture this run's code closure + run-spec to a git side-ref (metis#14), backfilling
	// the record with the durable SHA + capture status — so a dirty single run is
	// reproducible (git checkout the SHA). The sweep loop sets inSweep to capture ONCE
	// per shape-run instead (captureSweepCode), avoiding redundant per-point capture.
	// Best-effort (like the sweep path): a backfill hiccup warns, never aborts a finished run.
	if !o.inSweep {
		if err := captureSingleRun(o, runID); err != nil {
			fmt.Fprintf(out, "metis: warning: code-capture backfill failed for run %s: %v\n", runID, err)
		}
	}
	// The experiment .md is IMMUTABLE input (#13): a run writes its output to
	// runs/<id>/{run,record}.json (+ the .ledger.csv sidecar for sweeps), NEVER to the
	// config file — so a committed config is a stable content-hash. The human "recent
	// runs / top-N" view is on-demand via `metis ledger show` over the sidecar.
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
