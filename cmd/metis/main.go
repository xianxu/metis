// Command metis is the Go step-runner control plane: `metis run <experiment.md>`
// reads a CUE-validated experiment, validates its semantics (the checks M1
// deferred), executes its steps in dependency order as subprocesses (files +
// subprocess, never FFI), and records a Run. This is the thin IO layer over
// pkg/experiment (the pure parse / validate / orchestrate core).
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "metis:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: metis run [--run <id>] <experiment.md>")
	}
	switch args[0] {
	case "run":
		return cmdRun(args[1:])
	case "select":
		return cmdSelect(args[1:])
	case "ledger":
		return cmdLedger(args[1:])
	default:
		return fmt.Errorf("unknown subcommand %q (want: run | select | ledger)", args[0])
	}
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	runID := fs.String("run", "", "run id (default: run-<UTC timestamp>; ignored for a multi-point sweep — each point keys off its content-address)")
	cache := fs.Bool("cache", true, "use the metis#2 validating-trace step cache (<expDir>/.metis-cache); --cache=false to disable")
	dryRun := fs.Bool("dry-run", false, "metis#18 sweep: list the swept configs without running them")
	fast := fs.Bool("fast", false, "metis#32: run ONE outer fold instead of the full k (a ~1/k-cost honest single-point per-family holdout) — for iteration; the full nested run (default) gives mean±SE. Shorthand for --sample 1. Only affects a nested (multi-config) run.")
	sampleN := fs.Int("sample", 0, "metis#42: run m of the k outer folds (sparse fold sampling; 0/omitted = all k). k stays the estimand (each fold trains on (k-1)/k of the rows); m only trades precision for cost — use to probe a higher k (e.g. k=10, --sample 3) without the full k× bill. The SE over m<k folds is noisy (m-1 df): probe with it, don't re-select what ships on it. Errors on m>k, on a single-config (flat) run, and combined with --fast.")
	parallel := fs.Int("parallel", defaultParallel(), "metis#31: max concurrent step subprocesses across ALL sweep levels (driver×sweeper×resample share one global cap); <=1 = serial (exact pre-#31 behavior). Default runtime.NumCPU(), overridable by METIS_MAX_PARALLEL. Caveat: each leaf is a Python process that may itself multi-thread (BLAS / sklearn n_jobs) — n=NumCPU can oversubscribe cores; pin OMP_NUM_THREADS=1 or set n below NumCPU. On a COLD cache the first batch's ≤n points may each recompute the shared upstream (a bounded thundering herd).")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return fmt.Errorf("run: want exactly one <experiment.md>, got %d", len(rest))
	}
	// cmdRun just passes maxParallel; runExperiment establishes the parallel invariant
	// (leafSem + syncWriter) in one home so no runOpts caller can forget it (#31).
	_, err := runExperiment(runOpts{
		expPath:     rest[0],
		runID:       *runID,
		stepPath:    stepPath(rest[0]),
		cache:       *cache,
		dryRun:      *dryRun,
		fast:        *fast,
		sample:      *sampleN,
		out:         os.Stdout,
		maxParallel: *parallel,
	})
	return err
}

// defaultParallel is the default subprocess concurrency: METIS_MAX_PARALLEL if set to
// a valid positive int, else runtime.NumCPU() (metis#31).
func defaultParallel() int {
	if v := os.Getenv("METIS_MAX_PARALLEL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			return n
		}
	}
	return runtime.NumCPU()
}
