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
	"strings"

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
	stepPath []string      // dirs searched for <layer>/<steptype>
	expDir   string        // absolute experiment dir; anchor for exp-relative step inputs
	seed     int           // the experiment's seed, exposed to every step for reproducibility
	readRoot string        // metis#23: outer-fold analysis root; when set, confines base-dataset reads (empty = unconfined)
	out      io.Writer     // plain streaming progress
	sem      chan struct{} // metis#31: the GLOBAL leaf budget — acquired around the subprocess
	//                        spawn ONLY (a cache HIT never reaches here). One shared channel across
	//                        all nesting levels ⇒ ≤ cap(sem) concurrent step subprocesses no matter
	//                        how driver×sweeper×resample fans out. nil = unbounded (the serial path).
	pool *serverPool // metis#44: when non-nil, convention-conforming wrappers route through the
	//                  warm fork-server (one per project root) instead of a fresh uv/python spawn;
	//                  non-conforming wrappers + broken servers fall back to the legacy path below.
	pins []string // metis#48: default leaf BLAS pins (computed once per run by runExperiment;
	//              ambient-set names already excluded there) — appended to the legacy child env.
	//              The fork-server path carries them on the SERVER env instead (children inherit).
}

// stepEnv builds the per-step METIS_* contract vars — the ONE definition both executors
// share: the legacy subprocess appends them to the ambient env; a fork-server request
// carries them verbatim (the child scrubs METIS_* first, so absence — e.g. no READ_ROOT on
// an unconfined run — is authoritative).
func (e execStep) stepEnv(step experiment.Step, stepDir, runDir string) map[string]string {
	m := map[string]string{
		"METIS_STEP_DIR": stepDir,
		"METIS_RUN_DIR":  runDir,
		"METIS_STEP_ID":  step.ID,
		"METIS_EXP_DIR":  e.expDir,
		"METIS_SEED":     strconv.Itoa(e.seed),
	}
	if e.readRoot != "" {
		// confinement: only inject when sealing an outer-fold sweep, so the flat
		// driver:single path leaves the var unset (unconfined).
		m["METIS_READ_ROOT"] = e.readRoot
	}
	return m
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
	env := e.stepEnv(step, stepDir, runDir)

	// metis#44: route a convention-conforming wrapper through the warm fork-server. A leaf
	// slot is held around the in-flight fork exactly as around a legacy subprocess. ok=false
	// (non-conforming wrapper, or a server that failed/died — noticed loudly, once) falls
	// through to the legacy spawn below; a RESPONSE, even exit != 0, is a real step outcome.
	if e.pool != nil {
		if spec, forkable := parseWrapper(exe); forkable {
			if e.sem != nil {
				e.sem <- struct{}{}
			}
			resp, ok, ferr := e.pool.execute(spec, stepDir, env)
			if e.sem != nil {
				<-e.sem
			}
			if ferr != nil {
				// I1: dispatched-and-lost — the forked child may still be running in this
				// stepDir; a legacy re-run would double-execute. Error the step instead.
				return experiment.StepResult{}, fmt.Errorf("exec %s (forkserver): %v", exe, ferr)
			}
			if ok {
				if resp.Exit != 0 {
					return experiment.StepResult{}, fmt.Errorf("exec %s (forkserver): exit status %d\n%s", exe, resp.Exit, resp.Output)
				}
				return e.collectResult(step, stepDir, runDir)
			}
		} else {
			e.pool.noticeOnce("uses:"+step.Uses,
				fmt.Sprintf("step %q wrapper doesn't match the uv/metis.trace convention — legacy exec (no warm-start)", step.Uses))
		}
	}

	cmd := exec.Command(exe)
	cmd.Dir = stepDir
	// metis#23: strip any inherited METIS_READ_ROOT so an ambient shell value can never
	// confine the flat (driver:single) path — we set it ourselves below only when sealing.
	base := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "METIS_READ_ROOT=") {
			base = append(base, kv)
		}
	}
	// metis#48: default leaf BLAS pins (operator-exported values already won in blasPins,
	// so no duplicate names reach the child).
	base = append(base, e.pins...)
	for _, k := range sortedKeys(env) {
		base = append(base, k+"="+env[k])
	}
	cmd.Env = base
	// metis#31: acquire the global leaf budget around the ONLY real subprocess spawn
	// (resolve/mkdir/with.json above are cheap, non-subprocess — they draw no budget).
	// Release immediately after the process exits, before the cheap metrics/artifact
	// reads, so a slot is held only while a subprocess is actually running. An
	// orchestration goroutine never reaches here holding another slot ⇒ deadlock-free.
	if e.sem != nil {
		e.sem <- struct{}{}
	}
	combined, cmdErr := cmd.CombinedOutput()
	if e.sem != nil {
		<-e.sem
	}
	if cmdErr != nil {
		// Runner.Run already prefixes `step %q:`; name the executable, not the id
		// again, to avoid a doubled "step first: step first" prefix.
		return experiment.StepResult{}, fmt.Errorf("exec %s: %w\n%s", exe, cmdErr, combined)
	}

	return e.collectResult(step, stepDir, runDir)
}

// collectResult reads back a completed step's outputs (metrics.json + artifacts) — shared by
// the legacy-subprocess and fork-server paths, which differ only in HOW the step ran.
func (e execStep) collectResult(step experiment.Step, stepDir, runDir string) (experiment.StepResult, error) {
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

// sortedKeys gives a deterministic env append order (map iteration is randomized).
func sortedKeys(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
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
