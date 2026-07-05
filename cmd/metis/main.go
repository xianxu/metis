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
	"path/filepath"

	"github.com/xianxu/metis/internal/repo"
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
	default:
		return fmt.Errorf("unknown subcommand %q (want: run)", args[0])
	}
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	runID := fs.String("run", "", "run id (default: run-<UTC timestamp>)")
	cache := fs.Bool("cache", true, "use the metis#2 validating-trace step cache (<expDir>/.metis-cache); --cache=false to disable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return fmt.Errorf("run: want exactly one <experiment.md>, got %d", len(rest))
	}
	_, err := runExperiment(runOpts{
		expPath:  rest[0],
		runID:    *runID,
		stepPath: stepPath(),
		cache:    *cache,
		out:      os.Stdout,
	})
	return err
}

// stepPath is the ordered list of directories searched for a step-type executable
// (<layer>/<steptype>): $METIS_STEP_PATH (colon-separated) when set, else
// <repo-root>/steps. Real metis/* step-types land under steps/ in M3.
func stepPath() []string {
	if v := os.Getenv("METIS_STEP_PATH"); v != "" {
		return filepath.SplitList(v)
	}
	if wd, err := os.Getwd(); err == nil {
		if root, err := repo.Root(wd); err == nil {
			return []string{filepath.Join(root, "steps")}
		}
	}
	return []string{"steps"}
}
