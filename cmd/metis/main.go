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
	case "blend":
		return cmdBlend(args[1:])
	default:
		return fmt.Errorf("unknown subcommand %q (want: run | select | ledger | blend)", args[0])
	}
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	runID := fs.String("run", "", "run id (default: run-<UTC timestamp>; ignored for a multi-point sweep — each point keys off its content-address)")
	cache := fs.Bool("cache", true, "use the metis#2 validating-trace step cache (<expDir>/.metis-cache); --cache=false to disable")
	dryRun := fs.Bool("dry-run", false, "metis#18 sweep: list the swept configs without running them")
	fast := fs.Bool("fast", false, "metis#32: run ONE outer fold instead of the full k (a ~1/k-cost honest single-point per-family holdout) — for iteration; the full nested run (default) gives mean±SE. Shorthand for --sample out1. Only affects a nested (multi-config) run.")
	sampleStr := fs.String("sample", "", "metis#58: run a subset of the declared CV folds — out<M> (M of the k outer folds), in<N> (N of the inner_k per-config inner folds), or out<M>in<N>. Deterministic prefix subsets of the SAME partitions, so subset runs cache-escalate into full runs. k/inner_k stay the estimand; sampling only trades precision for cost (probe with it, don't re-select what ships on it). Nested (multi-config) runs only; errors loudly out of range or with --fast.")
	forkserver := fs.Bool("forkserver", true, "metis#44: run convention-conforming step wrappers through a warm per-root fork-server (pre-imported pandas/sklearn; ~1s spawn tax removed per leaf). --forkserver=false = legacy per-step uv/python spawn (the escape hatch); non-conforming wrappers and failed servers fall back to legacy automatically (loud, once).")
	noTUI := fs.Bool("no-tui", false, "metis#38: force the plain progress lines even on a TTY (the live board is default for a sweep when stdout is a terminal; piped/redirected output always gets plain lines)")
	parallel := fs.Int("parallel", defaultParallel(), "metis#31: max concurrent step subprocesses across ALL sweep levels (driver×sweeper×resample share one global cap); <=1 = serial (exact pre-#31 behavior). Default runtime.NumCPU(), overridable by METIS_MAX_PARALLEL. Leaf BLAS is pinned single-thread by default (metis#48; export a *_NUM_THREADS value yourself to override), so n is the ONE parallelism knob. On a COLD cache the first batch's ≤n points may each recompute the shared upstream (a bounded thundering herd).")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return fmt.Errorf("run: want exactly one <experiment.md>, got %d", len(rest))
	}
	sample, err := parseSample(*sampleStr)
	if err != nil {
		return fmt.Errorf("run: %v", err)
	}
	// cmdRun just passes maxParallel; runExperiment establishes the parallel invariant
	// (leafSem + syncWriter) in one home so no runOpts caller can forget it (#31).
	_, err = runExperiment(runOpts{
		expPath:     rest[0],
		runID:       *runID,
		stepPath:    stepPath(rest[0]),
		cache:       *cache,
		dryRun:      *dryRun,
		fast:        *fast,
		sample:      sample,
		forkserver:  *forkserver,
		tui:         !*noTUI && isCharDevice(os.Stdout), // metis#38: board iff a real terminal
		out:         os.Stdout,
		maxParallel: *parallel,
	})
	return err
}

// isCharDevice is the stdlib isatty: a terminal is a character device; a pipe,
// file, or CI redirect is not (metis#38 — the board must never corrupt captured logs).
func isCharDevice(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// boardWidth is the board's line-clamp width: $COLUMNS when sane, else 80 — read
// once at wiring (no SIGWINCH; a resize garbles at most one frame, the next full
// repaint re-truncates). Stdlib-only by design — no ioctl/x/term (metis#38 plan).
func boardWidth() int {
	if c := os.Getenv("COLUMNS"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n >= 20 {
			return n
		}
	}
	return 80
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
