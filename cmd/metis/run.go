package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/metis/internal/fsx"
	"github.com/xianxu/metis/internal/repo"
	"github.com/xianxu/metis/pkg/cas"
	"github.com/xianxu/metis/pkg/experiment"
)

// syncWriter serializes concurrent Write calls — the metis#31 parallel fan-out's
// progress output. Minimal: it prevents torn lines + the data race on a shared
// writer; it does NOT reorder or buffer per goroutine (clean per-k/n progress is
// metis#30's scope). Established in runExperiment when maxParallel>1.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

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
	ctx      context.Context // metis#72: run cancellation (SIGINT/SIGTERM); nil = Background
	expPath  string
	runID    string
	stepPath []string
	now      func() time.Time
	git      gitProbe
	cache    bool       // enable the metis#2 validating-trace cache (<expDir>/.metis-cache)
	dryRun   bool       // metis#18: list the swept configs without running them
	fast     bool       // metis#32: nested run does ONE outer fold (a ~1/k-cost honest single-point) instead of k
	sample   sampleSpec // metis#58: prefix-subsample the CV levels — Out of the k outer folds, In of the
	//               inner_k per-config inner folds (0 = all). k/inner_k stay the estimand knobs
	//               (train fraction / partition); sampling is the precision/cost knob, and prefix
	//               subsets of the SAME partitions cache-escalate into full runs.
	//               --fast ≡ --sample out1 (kept as shorthand).
	inSweep bool // metis#14: this run is a sweep point — suppress per-point single-run
	//               capture (the sweep captures once per shape-run in captureSweepCode)
	out  io.Writer
	exec experiment.StepExecutor // test seam: an injected fake replaces the subprocess
	//                              execStep (nil → the production execStep). Composes with
	//                              cache: the caching decorator still wraps it.
	activity    activityEmitter // successful step/run facts; nil is a no-op
	runRole     runRole         // role for successful concrete-run activity; zero = ineligible/non-sweep
	readRoot    string          // metis#23: when set, the production execStep confines base-dataset reads to this root
	maxParallel int             // metis#31: >1 ⇒ ParExec batches + a leaf semaphore; sizes leafBudget
	leafBudget  leafBudget      // metis#31/#66: the shared global subprocess budget (nil = serial/cache-only);
	//                           prioritySem (default fold-ordered, #67) | chanSem (--global-fanout)
	priority     int             // metis#66: the outer-fold index threaded to execStep as the prioritySem key
	globalFanout bool            // metis#67: --global-fanout — the escape hatch back to priority-blind chanSem
	autoStop     bool            // metis#66: --auto-stop — sequential-outer + incumbent-referenced loser stop
	stopSignal   <-chan struct{} // metis#66: board Q → graceful finalize (test seam; production reads stdin)
	runControl   *runControl     // one per shape run: global abort + optional 2n admission slots
	runLabel     string          // config/fold/preamble context captured with the first error
	forkserver   bool            // metis#44: warm fork-server leaf executor (cmdRun default true;
	//                           zero-value false keeps direct runOpts callers/tests on legacy exec)
	forkPool *serverPool // metis#44: the per-root warm-server pool, created once per runExperiment
	//                      when forkserver is set; threaded through nested runOpts copies.
	tui bool // metis#38: stdout is a TTY and --no-tui wasn't passed — a SWEEP pins the live board
	//          (a plain experiment ignores it; non-TTY/piped runs stay on the #30 plain lines)
	board           *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
	boardTick       <-chan time.Time  // test seam: nil uses the production 500ms ticker
	beforeBoardTick func()            // test seam: after tick selection, before health observation
	afterBoardTick  func()            // test seam: after the health observation returns
	leafGauge       func() (int, int) // metis#49: (busy, capacity) over leafSem — the board's slots line
	leafPins        []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
	//                             runExperiment (nil = not yet computed; non-nil rides nested runOpts
	//                             copies like forkPool — an all-suppressed result is empty, not nil)
	repeat string // metis#73: --repeat label — this execution is one distinct sample (own dir, record, row)
	// skipPublish is a test seam (metis#73): stop after record.json, as a crash before the
	// ledger row would, so `metis ledger publish` recovery can be exercised.
	skipPublish bool
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

	// metis#38: PARSE FIRST (parsing writes nothing) — the board decision needs the file
	// type, and writer identity is TEMPORAL: everything constructed below (fork-server
	// pool, execs, sink, capture warnings) captures whatever `out` is at ITS construction,
	// so the one compositor must exist before any of them or its writes bypass the board.
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

	// metis#31: establish the parallel invariant in ONE home — maxParallel>1 ⇒ a
	// non-nil SHARED leaf budget AND a serialized writer. Doing it here (not in
	// cmdRun) means no direct-runOpts caller (the tests) can enable maxParallel>1 yet
	// forget the budget or race the fan-out's progress writes on a bare buffer.
	// metis#67: fold-ordered scheduling (prioritySem: freed slots go to the lowest outer-fold
	// index → fold 0 finishes first, backfill preserved) is the DEFAULT; --global-fanout reverts
	// to the priority-blind chanSem. Byte-identical either way (scheduling-only). selectLeafBudget
	// is the pure seam (maxParallel<=1 → nil serial); a direct-injected budget (tests) is kept.
	if o.leafBudget == nil {
		o.leafBudget = selectLeafBudget(o.maxParallel, o.globalFanout)
	}
	if b := o.leafBudget; b != nil && o.leafGauge == nil {
		o.leafGauge = func() (int, int) { return b.gauge() } // metis#38/#66: occupancy IS the budget
	}
	// Exactly ONE writer wrap (metis#38): board mode → the pin-bottom compositor (it
	// serializes internally — no syncWriter stacking); else parallel → syncWriter.
	if o.tui && exp.Type == "experiment-shape" && !o.dryRun {
		o.board = newBoardWriter(out, now, os.Getenv("NO_COLOR") == "") // metis#55 banding; NO_COLOR opts out
		out = o.board
		o.out = out
		defer o.board.close() // idempotent — an error return must not leak a hidden cursor
	} else if o.maxParallel > 1 {
		out = &syncWriter{w: out}
		o.out = out
	}
	// metis#48: default leaf BLAS pins — computed ONCE per top-level run from the ambient
	// env (an exported operator value wins by exclusion in blasPins), announced loudly,
	// and injected at both spawn seams (legacy execStep child env; fork-server process
	// env). Fake-exec runs spawn nothing (no pins, no note); dry-run lists configs (same).
	// Placed AFTER the writer wrap (the note must route through the board) and BEFORE the
	// pool creation (the server spawn env needs the pins).
	if o.exec == nil && !o.dryRun && o.leafPins == nil {
		o.leafPins = blasPins(os.Environ())
		if len(o.leafPins) > 0 {
			fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
				strings.Join(o.leafPins, " "))
		}
	}
	// metis#44: one warm fork-server pool per top-level run, shut down (EOF-drain) when the
	// run ends. Only the production executor uses it (an injected test exec bypasses execStep).
	// Constructed AFTER the writer wrap — its fallback notices must route through the board.
	if o.forkserver && o.exec == nil && o.forkPool == nil {
		o.forkPool = newServerPool(out, o.leafPins)
		defer o.forkPool.shutdown()
	}
	if exp.Type == "experiment-shape" {
		sh, err := experiment.ParseShape(string(raw))
		if err != nil {
			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
		}
		if err := experiment.ValidateShape(sh); err != nil {
			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
		}
		if o.runControl == nil {
			o.runControl = newRunControl(o.maxParallel)
		}
		if o.repeat != "" {
			return experiment.Run{}, fmt.Errorf("%s: --repeat is for a plain experiment; a sweep already samples through its folds and seeds", o.expPath)
		}
		return experiment.Run{}, runShapeSweep(o, sh, now, out)
	}
	return runSingleExperiment(exp, o, now, out)
}

// runSingleExperiment is the one-point path of `metis run` (metis#73): claim the run dir when
// this is a repeat, execute and record through runResolvedExperiment, then publish the run's
// ledger row from its final record, the last effect. The row is published for a failed run
// too (status failed, no metrics), so every execution that began is in the ledger.
func runSingleExperiment(exp experiment.Experiment, o runOpts, now func() time.Time, out io.Writer) (experiment.Run, error) {
	runID := singleRunID(o, exp, now)
	if o.repeat != "" {
		if o.runID != "" {
			return experiment.Run{}, fmt.Errorf("%s: --run and --repeat both name the run dir; pass one", o.expPath)
		}
		if _, err := experiment.ParseRepeatLabel(o.repeat); err != nil {
			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
		}
		// The claim is only single-use if the dir name is the point's content address: an
		// untracked experiment falls back to a timestamped dir, which never collides.
		if _, err := shapeBlobHash(o.expPath); err != nil {
			return experiment.Run{}, fmt.Errorf("%s: --repeat needs the experiment tracked in git (its content address names the repeat's run dir, which makes a label single-use): %v", o.expPath, err)
		}
		runID = repeatRunID(runID, o.repeat)
		runDir, err := runDirFor(o.expPath, runID)
		if err != nil {
			return experiment.Run{}, err
		}
		if err := claimRunDir(o.expPath, runDir, o.repeat); err != nil {
			return experiment.Run{}, err
		}
		if o.cache && len(experiment.UncachedSteps(exp)) == 0 {
			fmt.Fprintf(out, "metis: warning: repeat %s: no step is declared uncached (with.cache.policy: never), so this repeat replays the cache rather than drawing a new sample\n", o.repeat)
		}
	}
	run, runErr := runResolvedExperiment(exp, o, runID, now, out)
	if run.Started == "" && o.repeat != "" {
		// Refused before any step ran (validation): release the claim, so the label is not
		// spent on an execution that never happened. The dir is empty; Remove refuses otherwise.
		if dir, err := runDirFor(o.expPath, runID); err == nil {
			os.Remove(dir)
		}
	}
	if run.Started == "" || o.skipPublish {
		return run, runErr // never started: no record, nothing to publish
	}
	runDir, err := runDirFor(o.expPath, runID)
	if err == nil {
		err = publishRunRow(o.expPath, runDir)
	}
	if err != nil {
		if runErr != nil {
			return run, fmt.Errorf("%w (and the ledger row was not published: %v)", runErr, err)
		}
		return run, fmt.Errorf("run %s recorded, but its ledger row was not published: %w (recover with: metis ledger publish %s %s)", runID, err, o.expPath, runDir)
	}
	return run, runErr
}

// repeatRunID names a repeat's run dir: the point address plus the label, so repeats of one
// config never share a dir and the dir is derivable from a ledger row (rowRunID).
func repeatRunID(base, label string) string { return base + ".r-" + label }

// runDirFor is the physical run dir of runID (the same resolution the runner uses).
func runDirFor(expPath, runID string) (string, error) {
	expDir, err := physicalPath(filepath.Dir(expPath))
	if err != nil {
		return "", fmt.Errorf("resolve experiment dir: %w", err)
	}
	return filepath.Join(expDir, "runs", runID), nil
}

// claimRunDir makes a repeat's label single-use (metis#73): the exclusive mkdir IS the claim,
// held for the dir's life, so a second run with the label (concurrent or later) refuses
// without touching the dir. When that execution's record is final but its ledger row is
// missing (a crash after the record), the refusal names the recovery verb.
func claimRunDir(expPath, runDir, label string) error {
	if err := os.MkdirAll(filepath.Dir(runDir), 0o755); err != nil {
		return err
	}
	err := os.Mkdir(runDir, 0o755)
	if err == nil {
		return nil
	}
	if !os.IsExist(err) {
		return err
	}
	msg := fmt.Sprintf("repeat %s already executed at %s; use a new label", label, runDir)
	if rec, rerr := readFinalRecord(runDir); rerr == nil {
		if present, lerr := rowPublished(expPath, rowFromRecord(rec)); lerr == nil && !present {
			msg += fmt.Sprintf(" (its ledger row is missing: metis ledger publish %s %s)", expPath, runDir)
		}
	}
	return errors.New(msg)
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
	if o.runControl == nil {
		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
	}
	o.activity = runControlActivityEmitter(o.runControl, o.activity)
	return o.runControl.run(o.runLabel, func() (experiment.Run, error) {
		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
	})
}

func runResolvedExperimentAdmitted(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
	baseDir := filepath.Dir(o.expPath)
	// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
	// the child's env, and the child's cwd IS the step dir — a relative path would
	// resolve $METIS_STEP_DIR/with.json under itself. Absolute paths are correct
	// from any cwd, so `metis run pipelines/foo.md` (a relative arg) works. Physical, too
	// (metis#76, physicalPath); stepEnv re-asserts that for every path a child receives.
	expDir, err := physicalPath(baseDir)
	if err != nil {
		return experiment.Run{}, fmt.Errorf("resolve experiment dir: %w", err)
	}
	runDir := filepath.Join(expDir, "runs", runID)

	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, budget: o.leafBudget, priority: o.priority, pool: o.forkPool, pins: o.leafPins}
	if o.exec != nil {
		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
	}
	if o.cache {
		cacheDir := filepath.Join(expDir, ".metis-cache")
		if err := ensureCacheGitignore(cacheDir); err != nil {
			return experiment.Run{}, err
		}
		store := cas.NewFSStore(filepath.Join(cacheDir, "cas"), 0, cas.Clock(now))
		exec = newCachingExecutor(exec, store, cacheDir, exp, out)
	}
	exec = activityExecutor{inner: exec, now: now, emit: o.activity}
	runner := experiment.Runner{Exec: exec, Now: now}
	fmt.Fprintf(out, "metis: run %s of experiment %q\n", runID, exp.ID)
	run, steps, runErr := runner.Run(orBackground(o.ctx), exp, runID, runDir)

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
	rec.Repeat = o.repeat // metis#73: the execution coordinate (empty for a plain run or a sweep point)
	// Capture this run's code closure + run-spec to a git side-ref (metis#14), backfilling
	// the record with the durable SHA + capture status — so a dirty single run is
	// reproducible (git checkout the SHA). The sweep loop sets inSweep to capture ONCE
	// per shape-run instead (captureSweepCode), avoiding redundant per-point capture.
	// Best-effort (like the sweep path): a backfill hiccup warns, never aborts a finished run.
	// metis#73: the backfill is in memory, before the ONE atomic record write, so a
	// record.json that exists is final (a failed backfill leaves an empty fingerprint).
	if !o.inSweep {
		if err := captureSingleRun(o, runID, &rec); err != nil {
			fmt.Fprintf(out, "metis: warning: code-capture backfill failed for run %s: %v\n", runID, err)
		}
	}
	if err := writeRecordJSON(runDir, rec); err != nil {
		return run, err
	}
	// The experiment .md is IMMUTABLE input (#13): a run writes its output to
	// runs/<id>/{run,record}.json (+ the .ledger.csv sidecar for sweeps), NEVER to the
	// config file — so a committed config is a stable content-hash. The human "recent
	// runs / top-N" view is on-demand via `metis ledger show` over the sidecar.
	if runErr != nil {
		return run, runErr
	}
	o.activity.emit(activityEvent{Kind: activityRunSuccess, At: now(), RunID: runID, Role: o.runRole})
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
	return fsx.WriteFileAtomic(filepath.Join(runDir, "run.json"), append(b, '\n'), 0o644) // metis#73: never torn
}

// orBackground is the nil-safe run context: option structs built by tests leave ctx unset.
func orBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
