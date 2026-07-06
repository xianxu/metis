package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/record"
)

func fixedNow() func() time.Time {
	return func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) }
}

// writeShape writes a shape md and returns its path.
func writeShape(t *testing.T, ws, body string) string {
	t.Helper()
	p := filepath.Join(ws, "sweep.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func readManifest(t *testing.T, ws string) sweepManifest {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(ws, "sweeps", "*", "manifest.json"))
	if len(matches) != 1 {
		t.Fatalf("expected exactly one sweep manifest, found %v", matches)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var man sweepManifest
	if err := json.Unmarshal(b, &man); err != nil {
		t.Fatal(err)
	}
	return man
}

// A multi-point shape SWEEPS: it runs every point and writes a manifest listing them
// with their free-params (the metis#7 driver, superseding #6's refusal).
func TestSweep_RunsAllPointsAndWritesManifest(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: multi
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---
`)
	err := runSweepViaRun(t, expPath, root, runOpts{cache: false})
	if err != nil {
		t.Fatalf("sweep should run, not error: %v", err)
	}
	man := readManifest(t, ws)
	if len(man.Points) != 2 {
		t.Fatalf("2-model sweep should record 2 points, got %d", len(man.Points))
	}
	models := map[any]bool{}
	for _, pr := range man.Points {
		if pr.Status != "ok" {
			t.Errorf("point %s status = %q; want ok", pr.RunID, pr.Status)
		}
		models[pr.FreeParams["train.model"]] = true
	}
	if !models["logreg"] || !models["rf"] {
		t.Errorf("manifest should carry both model free-params; got %v", models)
	}
	// Each point ran in its own content-addressed run dir.
	if entries, _ := filepath.Glob(filepath.Join(ws, "runs", "*")); len(entries) != 2 {
		t.Errorf("expected 2 point run dirs, got %d", len(entries))
	}
	// The LOAD-BEARING identity #8's ledger aggregation rests on: each manifest row's
	// run_id must equal the point_address in that run's record.json. Asserted (not just
	// traced) so a future normalization at either call-site can't silently desync the
	// manifest from the records — a green suite would otherwise hide it.
	for _, pr := range man.Points {
		rb, err := os.ReadFile(filepath.Join(ws, "runs", pr.RunID, "record.json"))
		if err != nil {
			t.Fatalf("record.json for point %s: %v", pr.RunID, err)
		}
		var rec record.RunRecord
		if err := json.Unmarshal(rb, &rec); err != nil {
			t.Fatal(err)
		}
		if string(rec.PointAddress) != pr.RunID {
			t.Errorf("manifest run_id %s != record.json point_address %s — the #8 handoff identity desynced", pr.RunID, rec.PointAddress)
		}
	}
}

// Cache reuse across points: a shared upstream step HITs on the second point (metis#2).
func TestSweep_CacheReuseAcrossPoints(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: reuse
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: echoed, direction: maximize}}
steps:
  - id: prep
    uses: test/echo
    with: {k: 5}
  - id: train
    uses: test/echo
    needs: [prep]
    with: {model: {$any: [logreg, rf]}}
---
`)
	var out strings.Builder
	if err := runSweepCapture(t, expPath, root, runOpts{cache: true}, &out); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	// prep has identical config across both points → the 2nd point's prep is a cache hit.
	if n := strings.Count(out.String(), "step prep (cache hit)"); n < 1 {
		t.Errorf("shared upstream `prep` should HIT on the 2nd point; hits=%d\n%s", n, out.String())
	}
}

// A failing point is recorded and the sweep CONTINUES to the remaining points.
func TestSweep_FailingPointRecordedAndContinues(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	// fail FIRST ($any list order: true then false) so a "record-then-stop" bug would
	// drop the sweep to 1 row — the test only proves "continues past a failure" if the
	// later ok point still runs (the metis#6 lesson: exercise the dimension the bug lives in).
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: hasfail
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {fail: {$any: [true, false]}}
---
`)
	err := runSweepViaRun(t, expPath, root, runOpts{cache: false})
	if err != nil {
		t.Fatalf("a failing point must not abort the sweep: %v", err)
	}
	man := readManifest(t, ws)
	if len(man.Points) != 2 {
		t.Fatalf("both points must be recorded — the sweep must CONTINUE past the (first) failure, got %d", len(man.Points))
	}
	// The first-Ask'd point ($any[0] = fail:true) is failed; the later one is ok.
	if man.Points[0].Status != "failed" {
		t.Errorf("point 0 (fail:true) status = %q; want failed", man.Points[0].Status)
	}
	if man.Points[1].Status != "ok" {
		t.Errorf("point 1 (fail:false) status = %q; want ok — the sweep must run it AFTER the failure", man.Points[1].Status)
	}
	// The run_id == record.point_address identity must hold for a FAILED point too —
	// Runner.Run returns only pre-failure StepRuns, so the record must mint its
	// point-address from the point's FULL intended config (not the partial executed
	// config), else a failed point's address diverges from its run-id and collides
	// across distinct failed configs (breaking #8's aggregation).
	failed := man.Points[0]
	rb, err := os.ReadFile(filepath.Join(ws, "runs", failed.RunID, "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec record.RunRecord
	if err := json.Unmarshal(rb, &rec); err != nil {
		t.Fatal(err)
	}
	if string(rec.PointAddress) != failed.RunID {
		t.Errorf("failed point: record point_address %s != run_id %s (partial-config drift)", rec.PointAddress, failed.RunID)
	}
}

// Two distinct FAILED configs (both failing at their only step) must mint DISTINCT
// point-addresses — a regression guard for the partial-config collision (deriving the
// address from executed steps made both empty → same hash). Uses distinct extra knobs
// alongside fail:true so only the full-config address distinguishes them.
func TestSweep_DistinctFailedConfigsHaveDistinctAddresses(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: failcollide
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {fail: true, model: {$any: [xx, yy]}}
---
`)
	if err := runSweepViaRun(t, expPath, root, runOpts{cache: false}); err != nil {
		t.Fatal(err)
	}
	man := readManifest(t, ws)
	if len(man.Points) != 2 {
		t.Fatalf("both failed points recorded, got %d", len(man.Points))
	}
	// The collision was in record.point_address (minted from PARTIAL executed config —
	// empty for a first-step failure → same hash for both). The manifest run_id is
	// minted from full p.With so it never collided; read the RECORDS to catch it.
	addrs := map[string]bool{}
	for _, pr := range man.Points {
		if pr.Status != "failed" {
			t.Errorf("point %s should be failed, got %q", pr.RunID, pr.Status)
		}
		rb, err := os.ReadFile(filepath.Join(ws, "runs", pr.RunID, "record.json"))
		if err != nil {
			t.Fatal(err)
		}
		var rec record.RunRecord
		if err := json.Unmarshal(rb, &rec); err != nil {
			t.Fatal(err)
		}
		if string(rec.PointAddress) != pr.RunID {
			t.Errorf("failed point: record point_address %s != run_id %s", rec.PointAddress, pr.RunID)
		}
		addrs[string(rec.PointAddress)] = true
	}
	if len(addrs) != 2 {
		t.Errorf("two distinct failed configs must have distinct record point_addresses, got %d unique: %v", len(addrs), addrs)
	}
}

// --max-points caps a sweep before exhaustion.
func TestSweep_MaxPointsStopsEarly(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: capped
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [a, b, c, d]}}
---
`)
	err := runSweepViaRun(t, expPath, root, runOpts{cache: false, maxPoints: 2})
	if err != nil {
		t.Fatal(err)
	}
	man := readManifest(t, ws)
	if len(man.Points) != 2 {
		t.Errorf("--max-points=2 on a 4-point grid should record 2, got %d", len(man.Points))
	}
}

// --dry-run lists the points and runs nothing (no run dirs, no manifest).
func TestSweep_DryRunListsWithoutRunning(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: dry
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---
`)
	var out strings.Builder
	if err := runSweepCapture(t, expPath, root, runOpts{cache: false, dryRun: true}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "dry run") || !strings.Contains(out.String(), "train.model=logreg") {
		t.Errorf("dry run should list points; got:\n%s", out.String())
	}
	if entries, _ := filepath.Glob(filepath.Join(ws, "runs", "*")); len(entries) != 0 {
		t.Errorf("dry run must not create run dirs, found %d", len(entries))
	}
}

// Detect-and-abort: if the code identity changes mid-sweep, the sweep aborts.
func TestSweep_DetectAndAbortOnCodeChange(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: drift
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [a, b, c]}}
---
`)
	// A gitProbe whose sha flips after the first per-point re-check → simulates a
	// mid-sweep code edit.
	probe := &mutatingGitProbe{shas: []string{"sha1", "sha1", "sha2"}}
	_, err := runExperiment(runOpts{
		expPath:  expPath,
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      fixedNow(),
		git:      probe,
		cache:    false,
		out:      io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "code changed") {
		t.Errorf("a mid-sweep code change should abort with a 'code changed' error; got: %v", err)
	}
}

// Regression: the sweep writes its own outputs (runs/, manifest) which dirty the
// working tree — a constant `dirty=true` probe (what a real repo reports after point 1
// writes) must NOT false-abort the sweep. The freeze compares the HEAD sha only.
func TestSweep_DirtyTreeDoesNotFalseAbort(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: dirtyok
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [a, b, c]}}
---
`)
	_, err := runExperiment(runOpts{
		expPath:  expPath,
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      fixedNow(),
		git:      fakeGitProbe{name: "metis", sha: "constant-sha", dirty: true}, // always dirty
		cache:    false,
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("a persistently-dirty tree (from the sweep's own outputs) must not abort: %v", err)
	}
	if man := readManifest(t, ws); len(man.Points) != 3 {
		t.Errorf("all 3 points should run despite dirty=true, got %d", len(man.Points))
	}
}

// isPointOutcome is the sweep's continue-vs-abort classifier. Pinned directly so a
// revert to the old `run.Started=="" && runErr!=nil` swallowing (which would treat an
// ok-run-with-persistence-error as a recordable outcome) fails a test.
func TestIsPointOutcome_Classification(t *testing.T) {
	errBoom := fmt.Errorf("boom")
	cases := []struct {
		name string
		run  experiment.Run
		err  error
		want bool // true = record + continue; false = surface (sweep-fatal)
	}{
		{"clean run", experiment.Run{Status: "ok", Started: "t"}, nil, true},
		{"step failure", experiment.Run{Status: "failed", Started: "t"}, errBoom, true},
		{"ok run with persistence error", experiment.Run{Status: "ok", Started: "t"}, errBoom, false},
		{"never-started validation error", experiment.Run{}, errBoom, false},
	}
	for _, c := range cases {
		if got := isPointOutcome(c.run, c.err); got != c.want {
			t.Errorf("%s: isPointOutcome = %v, want %v", c.name, got, c.want)
		}
	}
}

// mutatingGitProbe returns a different sha per call, simulating code drift mid-sweep.
type mutatingGitProbe struct {
	shas []string
	n    int
}

func (m *mutatingGitProbe) Probe(string) (string, string, bool, error) {
	s := m.shas[len(m.shas)-1]
	if m.n < len(m.shas) {
		s = m.shas[m.n]
	}
	m.n++
	return "metis", s, false, nil
}

// runSweepViaRun runs a shape sweep through runExperiment (discarding output).
func runSweepViaRun(t *testing.T, expPath, root string, o runOpts) error {
	t.Helper()
	return runSweepCapture(t, expPath, root, o, io.Discard)
}

func runSweepCapture(t *testing.T, expPath, root string, o runOpts, out io.Writer) error {
	t.Helper()
	o.expPath = expPath
	o.stepPath = []string{filepath.Join(root, "testdata", "steps")}
	o.now = fixedNow()
	if o.git == nil {
		o.git = fakeGitProbe{name: "metis", sha: "sha", dirty: false}
	}
	o.out = out
	_, err := runExperiment(o)
	return err
}
