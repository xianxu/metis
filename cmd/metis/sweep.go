package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
	"github.com/xianxu/metis/pkg/shape"
	"github.com/xianxu/metis/pkg/sweep"
)

// sweepManifest groups the N point-runs an experiment-shape invocation produced. Its
// identity (ShapeRunID) filters the accumulating ledger (metis#8) by invocation /
// code-version; each PointRun row is the ledger's raw material (metis#8 builds the
// queryable, promotable ledger over these).
type sweepManifest struct {
	ShapeRunID string     `json:"shape_run_id"`
	Shape      string     `json:"shape"`
	Sampler    string     `json:"sampler"`
	Seed       int        `json:"seed"`
	Points     []pointRun `json:"points"`
}

type pointRun struct {
	RunID      string             `json:"run_id"` // = the point's PointAddress
	FreeParams map[string]any     `json:"free_params"`
	Status     string             `json:"status"` // ok | failed
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

// runSweep drives the metis#7 ask/tell loop over a multi-point shape: build a grid
// sampler over the expanded points (+ the --max-points budget stop), and for each
// point run it through the shared cached runner (runResolvedExperiment) keyed by its
// content-address. Per-point failure is recorded and the sweep CONTINUES; a mid-sweep
// code change aborts (detect-and-abort) to protect the shape-run's one-code identity.
func runSweep(o runOpts, sh experiment.Shape, points []shape.Point, now func() time.Time, out io.Writer) error {
	// Repo identity: the same {repoName: sha} construction buildRecord uses, so a
	// point's pre-computed PointAddress runID matches the address the record mints.
	repoName, sha, _ := probeRepo(o.git, filepath.Dir(o.expPath))
	repoSHAs := repoSHAsOf(repoName, sha)
	// The code-version identity the sweep is frozen at. v1 uses the HEAD commit sha
	// ONLY — NOT the working-tree dirty flag: the sweep writes its own outputs
	// (runs/, the manifest) which dirty the tree, so a whole-repo dirty flag would
	// false-abort on the sweep's own writes. HEAD-sha catches the realistic drift (a
	// mid-sweep commit / branch switch that changes the Python step code the
	// subprocess re-imports). A dirty in-place code edit isn't caught at the
	// shape-run freeze level, but per-point correctness holds regardless — each
	// point's record.json + cache trace captures its ACTUAL code bytes (the design's
	// "freezing only protects the shape-run invariant"). Precise code-dirty detection
	// via the closure trace is the metis#10 hardening.
	codeID := sha

	if o.dryRun {
		fmt.Fprintf(out, "metis: sweep %s — %d points (dry run):\n", sh.ID, len(points))
		for i, p := range points {
			fmt.Fprintf(out, "  [%d] %s\n", i, freeParamStr(p))
		}
		return nil
	}

	var stop sweep.StopPredicate
	if o.maxPoints > 0 {
		stop = sweep.MaxPoints(o.maxPoints)
	}
	sampler := sweep.NewGrid(points, stop)

	shapeRunID, err := shapeRunIdentity(sh, repoSHAs, o.maxPoints)
	if err != nil {
		return err
	}
	man := sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweep.Sampler, Seed: sh.Seed}
	fmt.Fprintf(out, "metis: sweep %s (%s) — up to %d points\n", sh.ID, shapeRunID[:12], len(points))

	n := 0
	for {
		p, ok := sampler.Ask()
		if !ok {
			break
		}
		n++
		// Detect-and-abort: if the HEAD code sha changed underneath the sweep, stop —
		// the shape-run's identity assumes one code version (per-point records stay
		// correct). Compares the HEAD sha only, not the dirty flag (see codeID above).
		if _, s, _ := probeRepo(o.git, filepath.Dir(o.expPath)); s != codeID {
			return fmt.Errorf("code changed at point %d/%d (%s → %s) — re-run to sweep the new revision", n, len(points), codeID, s)
		}

		runID, err := record.PointAddress(p.With, repoSHAs, sh.Seed)
		if err != nil {
			return fmt.Errorf("point %d: %w", n, err)
		}
		exp := shapePointToExperiment(sh, p)
		run, runErr := runResolvedExperiment(exp, o, string(runID), now, out)
		if !isPointOutcome(run, runErr) {
			// Not a per-point outcome (a validation never-started error, or a
			// metis-internal persistence error like a failed writeRecordJSON) — surface
			// it, so a real failure isn't silently recorded as `ok` and left to break
			// #8's per-point aggregation.
			return fmt.Errorf("point %d (%s): %w", n, freeParamStr(p), runErr)
		}
		man.Points = append(man.Points, pointRun{
			RunID:      string(runID),
			FreeParams: freeParamMap(p),
			Status:     run.Status,
			Metrics:    run.Metrics,
		})
		sampler.Tell(p, sweep.Result{Metrics: run.Metrics, Status: run.Status})
	}

	if err := writeManifest(o.expPath, man); err != nil {
		return err
	}
	// Capture the sweep's code closure to a git side ref (metis#8 durability) and
	// backfill each point-record's CodeManifest — so even a dirty-iteration run has a
	// real committed SHA and is recoverable. BEST-EFFORT: the sweep already ran and its
	// per-point records + manifest are valid, so a capture hiccup (no commit identity,
	// read-only object store) must NOT fail the whole run — it only forgoes the durable
	// code SHA. Warn, don't abort.
	if err := captureSweepCode(o, man); err != nil {
		fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
	}
	// Aggregate the sweep into the shape's append-only ledger (metis#8) — idempotent
	// (dedups by point-address) + regenerates the body top-N summary.
	if err := writeSweepLedger(o.expPath, man, sh.Sweep.Objective); err != nil {
		return err
	}
	fmt.Fprintf(out, "metis: sweep %s done — %d points recorded (manifest %s)\n", sh.ID, len(man.Points), shapeRunID[:12])
	return nil
}

// isPointOutcome classifies a per-point run result: true = a legitimate point outcome
// the sweep records and continues past (a clean run, or a STEP failure — "one bad
// config can't kill the sweep"); false = a sweep-fatal error (a validation
// never-started run, or a metis-internal persistence error on an otherwise-ok run)
// that must be surfaced, not swallowed. Pure — the classification the IO loop rests on,
// unit-tested directly so a revert to the old swallowing guard fails a test.
func isPointOutcome(run experiment.Run, runErr error) bool {
	if runErr == nil {
		return true // a clean run
	}
	return run.Status == "failed" // a step failure is a recorded point outcome; anything else is fatal
}

// shapeRunIdentity mints the invocation identity that groups the sweep's point-runs:
// hash(shape id + steps, repo SHAs, sampler config, seed). Grid's point-set is
// derivable from the shape, so the manifest stays thin.
func shapeRunIdentity(sh experiment.Shape, repoSHAs map[string]string, maxPoints int) (string, error) {
	h, err := record.CanonicalHash(struct {
		Shape     string            `json:"shape"`
		Steps     []experiment.Step `json:"steps"`
		Sweep     experiment.Sweep  `json:"sweep"`
		RepoSHAs  map[string]string `json:"repo_shas"`
		Seed      int               `json:"seed"`
		MaxPoints int               `json:"max_points"`
	}{sh.ID, sh.Steps, sh.Sweep, repoSHAs, sh.Seed, maxPoints})
	return string(h), err
}

func writeManifest(expPath string, man sweepManifest) error {
	dir := filepath.Join(filepath.Dir(expPath), "sweeps", man.ShapeRunID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), append(b, '\n'), 0o644)
}

// probeRepo runs the injected gitProbe, degrading to empty provenance (like
// assembleRecord) when there's no repo — so a sweep outside git still runs.
func probeRepo(git gitProbe, dir string) (name, sha string, dirty bool) {
	if git == nil {
		git = gitCLI{}
	}
	n, s, d, err := git.Probe(dir)
	if err != nil {
		return "", "", false
	}
	return n, s, d
}

// repoSHAsOf builds the {repoName: sha} map buildRecord uses — same construction, so a
// pre-computed PointAddress matches the record's internal one (incl. the no-git case).
func repoSHAsOf(repoName, sha string) map[string]string {
	m := map[string]string{}
	if repoName != "" {
		m[repoName] = sha
	}
	return m
}

// freeParamMap renders a point's free-param path as a {path: value} map (for the
// manifest); freeParamStr renders the same as a compact human string (for logs).
func freeParamMap(p shape.Point) map[string]any {
	m := make(map[string]any, len(p.FreeParams))
	for _, fp := range p.FreeParams {
		m[fp.Path] = fp.Value
	}
	return m
}

func freeParamStr(p shape.Point) string {
	s := ""
	for i, fp := range p.FreeParams {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("%s=%v", fp.Path, fp.Value)
	}
	if s == "" {
		return "(no free params)"
	}
	return s
}
