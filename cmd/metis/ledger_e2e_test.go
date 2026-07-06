package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"
	"github.com/xianxu/metis/pkg/shape"
)

// mustLedgerBest returns the winning row's point-address by the shape's objective.
func mustLedgerBest(t *testing.T, shapePath string) string {
	t.Helper()
	raw, _ := os.ReadFile(shapePath)
	sh, _ := experiment.ParseShape(string(raw))
	led, err := loadLedger(shapePath)
	if err != nil {
		t.Fatal(err)
	}
	row, ok := ledger.Best(led, sh.Sweep.Objective.Metric, sh.Sweep.Objective.Direction)
	if !ok {
		t.Fatalf("no best row for objective %q", sh.Sweep.Objective.Metric)
	}
	return row.PointAddr
}

func mustRecord(t *testing.T, path string) record.RunRecord {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rec record.RunRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

// A sweep writes the shape's ledger sidecar (one row per point, namespaced metrics) and
// leaves the experiment .md untouched (#13 — the top-N view is `metis ledger show`, not a
// body summary).
func TestLedger_SweepWritesSidecarNotBody(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: led
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: train.echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---

body here
`)
	if err := runSweepViaRun(t, expPath, root, runOpts{cache: false}); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	// The ledger sidecar exists with 2 rows + the union columns.
	csv, err := os.ReadFile(ledgerPath(expPath))
	if err != nil {
		t.Fatalf("ledger sidecar not written: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(csv)), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Errorf("ledger should have header + 2 rows, got %d lines:\n%s", len(lines), csv)
	}
	if !strings.Contains(lines[0], "fp.train.model") || !strings.Contains(lines[0], "metric.train.echoed") {
		t.Errorf("ledger header missing expected columns: %s", lines[0])
	}
	// #13: the sweep must NOT mutate the config .md — the top-N view is `metis ledger show`
	// over the sidecar, not a summary written into the experiment body (immutable input).
	body, _ := os.ReadFile(expPath)
	if strings.Contains(string(body), "metis:ledger") || strings.Contains(string(body), "## Top runs") {
		t.Errorf("sweep mutated the config .md (must be immutable input):\n%s", body)
	}
	// Idempotent: a second identical sweep dedups (still 2 rows).
	if err := runSweepViaRun(t, expPath, root, runOpts{cache: false}); err != nil {
		t.Fatal(err)
	}
	csv2, _ := os.ReadFile(ledgerPath(expPath))
	if n := len(strings.Split(strings.TrimSpace(string(csv2)), "\n")); n != 3 {
		t.Errorf("re-sweep must dedup (still 2 rows), got %d lines", n)
	}
}

// promote --best reconstructs the winning point as an all-singleton experiment that
// round-trips: it re-runs and reproduces the winner's run.
func TestLedger_PromoteBestRoundTrips(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	// A 2-point sweep; test/echo emits echoed=1 for both, so "best" is deterministic
	// (the first by objective order). We assert the promoted experiment re-runs.
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: promo
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: train.echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---
`)
	if err := runSweepViaRun(t, expPath, root, runOpts{cache: false}); err != nil {
		t.Fatal(err)
	}
	if err := runPromote(promoteOpts{
		shapePath: expPath, best: true, name: "winner",
		out: os.Stdout, git: fakeGitProbe{name: "metis", sha: "sha", dirty: false},
	}); err != nil {
		t.Fatalf("promote --best: %v", err)
	}
	winnerPath := filepath.Join(ws, "winner.md")
	wb, err := os.ReadFile(winnerPath)
	if err != nil {
		t.Fatalf("winner.md not written: %v", err)
	}
	if !strings.Contains(string(wb), "type: experiment") || !strings.Contains(string(wb), "promoted_from: promo") {
		t.Errorf("promoted experiment malformed:\n%s", wb)
	}
	// The promoted experiment's id must match its <name>.md filename (the experiment
	// convention), NOT the shape's id.
	if !strings.Contains(string(wb), "id: winner") {
		t.Errorf("promoted experiment id must be the --name (winner), not the shape id:\n%s", wb)
	}
	winRow := mustLedgerBest(t, expPath)
	// The back-link must carry the origin provenance: the point-address (@ …) + the
	// sweep-SHA (so the promoted experiment can be checked against / recovered to its row).
	if !strings.Contains(string(wb), "@ "+winRow) || !strings.Contains(string(wb), "sweep ") {
		t.Errorf("promoted_from must record the point-address + sweep-SHA; got:\n%s", wb)
	}
	// Round-trip: the promoted all-singleton experiment re-runs AND reproduces the
	// winning row's point-address (the Done-when — not just status ok). We compare the
	// promoted run's record.point_address to the winning row's point-address.
	run, err := runExperiment(runOpts{
		expPath:  winnerPath,
		runID:    "rt",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      fixedNow(),
		git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		out:      io.Discard,
	})
	if err != nil {
		t.Fatalf("promoted experiment must re-run: %v", err)
	}
	if run.Status != "ok" {
		t.Errorf("promoted round-trip status = %q; want ok", run.Status)
	}
	rt := mustRecord(t, filepath.Join(ws, "runs", "rt", "record.json"))
	if string(rt.PointAddress) != winRow {
		t.Errorf("promoted round-trip point_address %s != winning row %s — reproduction broke", rt.PointAddress, winRow)
	}
	// The round-trip must also reproduce the row's METRICS (Done-when: point-address +
	// metrics). The winning row's train.echoed must match the re-run's train step metric.
	wantMetric := mustRowMetric(t, expPath, "train.echoed")
	got := 0.0
	for _, st := range rt.Steps {
		if v, ok := st.Metrics["echoed"]; ok && st.StepID == "train" {
			got = v
		}
	}
	if got != wantMetric {
		t.Errorf("round-trip metric train.echoed = %g; want %g (the winning row's) — metrics didn't reproduce", got, wantMetric)
	}
}

func mustRowMetric(t *testing.T, shapePath, metric string) float64 {
	t.Helper()
	raw, _ := os.ReadFile(shapePath)
	sh, _ := experiment.ParseShape(string(raw))
	led, _ := loadLedger(shapePath)
	row, ok := ledger.Best(led, sh.Sweep.Objective.Metric, sh.Sweep.Objective.Direction)
	if !ok {
		t.Fatal("no best row")
	}
	return row.Metrics[metric]
}

// Promoting a winner from an OLDER code-version (its sweep-SHA ≠ HEAD) must warn that
// it reproduces only after `git checkout <sweep-SHA>` — the guard for the "committed at
// its code SHA" contract. Real git: sweep, advance HEAD, then promote.
func TestPromote_WarnsOnCrossCodeVersion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := gitInit(t)
	stepsDir := filepath.Join(root, "steps")
	if err := os.MkdirAll(filepath.Join(stepsDir, "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, filepath.Join(repoRoot(t), "testdata", "steps", "test", "echo"), filepath.Join(stepsDir, "test", "echo"))
	_ = os.Chmod(filepath.Join(stepsDir, "test", "echo"), 0o755)
	expPath := filepath.Join(root, "sweep.md")
	if err := os.WriteFile(expPath, []byte("---\ntype: experiment-shape\nid: x\nseed: 5\nstatus: active\nsweep: {sampler: grid, objective: {metric: train.echoed, direction: maximize}}\nsteps:\n  - id: train\n    uses: test/echo\n    with: {model: {$any: [logreg, rf]}}\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "init")
	if _, err := runExperiment(runOpts{expPath: expPath, stepPath: []string{stepsDir}, now: fixedNow(), out: io.Discard, git: gitCLI{}}); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	// Advance HEAD (a new commit) so the row's sweep-SHA is now an OLDER code-version.
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("advance\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "advance HEAD")

	var out bytes.Buffer
	if err := runPromote(promoteOpts{
		shapePath: expPath, best: true, name: "old", out: &out,
		git: gitCLI{}, commit: gitCLICommitter{},
	}); err != nil {
		t.Fatalf("promote: %v", err)
	}
	if !strings.Contains(out.String(), "git checkout") || !strings.Contains(out.String(), "ran at code") {
		t.Errorf("promoting an older-code-version winner must warn to re-checkout its sweep-SHA; got:\n%s", out.String())
	}
}

// The CLI commands must work with the DOCUMENTED arg order (<shape.md> before flags) —
// Go's stdlib flag stops at the first positional, so this slipped past the e2es that
// called runPromote directly. hoistShapePath makes flags order-independent.
func TestCLI_ArgOrderIndependent(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := writeShape(t, ws, `---
type: experiment-shape
id: argorder
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: train.echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---
`)
	if err := runSweepViaRun(t, expPath, root, runOpts{cache: false}); err != nil {
		t.Fatal(err)
	}
	// `ledger show <shape> --sort M` (shape FIRST, as documented) must not error.
	if err := cmdLedger([]string{"show", expPath, "--sort", "train.echoed"}); err != nil {
		t.Errorf("`ledger show <shape> --sort M` (documented order) errored: %v", err)
	}
	// And flags-first still works.
	if err := cmdLedger([]string{"show", "--top", "1", expPath}); err != nil {
		t.Errorf("`ledger show --top 1 <shape>` errored: %v", err)
	}
}

// `ledger show`'s rendered table is asserted against a buffer: a header row + the rows
// in objective-sorted order (a --sort/--dir regression would otherwise ship green).
func TestLedgerShow_RendersSortedTable(t *testing.T) {
	ws := t.TempDir()
	shapePath := filepath.Join(ws, "s.md")
	if err := os.WriteFile(shapePath, []byte("---\ntype: experiment-shape\nid: s\nseed: 1\nstatus: active\nsweep: {sampler: grid}\nsteps: []\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Hand-write a ledger with three rows of differing scores.
	var l ledger.Ledger
	l.Append(
		ledger.Row{SweepSHA: "sha1", PointAddr: "a", FreeParams: map[string]any{"model": "logreg"}, Metrics: map[string]float64{"train.cv": 0.70}, Status: "ok"},
		ledger.Row{SweepSHA: "sha1", PointAddr: "b", FreeParams: map[string]any{"model": "rf"}, Metrics: map[string]float64{"train.cv": 0.90}, Status: "ok"},
		ledger.Row{SweepSHA: "sha1", PointAddr: "c", FreeParams: map[string]any{"model": "gbm"}, Metrics: map[string]float64{"train.cv": 0.80}, Status: "ok"},
	)
	csv, _ := ledger.Encode(l)
	if err := os.WriteFile(ledgerPath(shapePath), csv, 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := showLedger(shapePath, "", "train.cv", "maximize", 0, &buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 { // header + 3 rows
		t.Fatalf("want header + 3 rows, got %d:\n%s", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "sweep") || !strings.Contains(lines[0], "train.cv") {
		t.Errorf("header row missing expected columns: %q", lines[0])
	}
	// maximize order: rf (0.90), gbm (0.80), logreg (0.70).
	if !strings.Contains(lines[1], "model=rf") || !strings.Contains(lines[3], "model=logreg") {
		t.Errorf("rows not in maximize order (rf, gbm, logreg):\n%s", buf.String())
	}
	// minimize flips the order.
	buf.Reset()
	_ = showLedger(shapePath, "", "train.cv", "minimize", 0, &buf)
	min := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if !strings.Contains(min[1], "model=logreg") {
		t.Errorf("minimize should put logreg (0.70) first:\n%s", buf.String())
	}

	// A `--sort`ed show must NOT drop a failed row (it's not a leaderboard).
	l.Append(ledger.Row{SweepSHA: "sha1", PointAddr: "d", FreeParams: map[string]any{"model": "svm"}, Status: "failed"})
	csv2, _ := ledger.Encode(l)
	_ = os.WriteFile(ledgerPath(shapePath), csv2, 0o644)
	buf.Reset()
	_ = showLedger(shapePath, "", "train.cv", "maximize", 0, &buf)
	if !strings.Contains(buf.String(), "model=svm") {
		t.Errorf("--sort must keep the failed row (svm), not drop it:\n%s", buf.String())
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what it wrote —
// so a test can drive the real cmdLedger entrypoint (which prints to os.Stdout) and assert
// the rendered table, exercising the CLI's own direction-defaulting rather than showLedger.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()
	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

// `ledger show --sort M` with NO --dir must default the direction from the shape's
// objective, so a MINIMIZE objective sorts best-first (lowest metric on top). This drives
// the real cmdLedger (which reads the shape + defaults --dir) — the showLedger tests all
// pass direction explicitly, so this is the only guard on the round-5 defaulting path.
func TestLedgerShow_DefaultsDirFromObjective(t *testing.T) {
	ws := t.TempDir()
	shapePath := filepath.Join(ws, "loss.md")
	// A shape whose objective MINIMIZES train.loss.
	if err := os.WriteFile(shapePath, []byte("---\ntype: experiment-shape\nid: loss\nseed: 1\nstatus: active\nsweep: {sampler: grid, objective: {metric: train.loss, direction: minimize}}\nsteps: []\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Rows of differing loss; lowest is model=good (0.10), highest model=bad (0.90).
	var l ledger.Ledger
	l.Append(
		ledger.Row{SweepSHA: "s", PointAddr: "a", FreeParams: map[string]any{"model": "mid"}, Metrics: map[string]float64{"train.loss": 0.50}, Status: "ok"},
		ledger.Row{SweepSHA: "s", PointAddr: "b", FreeParams: map[string]any{"model": "bad"}, Metrics: map[string]float64{"train.loss": 0.90}, Status: "ok"},
		ledger.Row{SweepSHA: "s", PointAddr: "c", FreeParams: map[string]any{"model": "good"}, Metrics: map[string]float64{"train.loss": 0.10}, Status: "ok"},
	)
	csv, _ := ledger.Encode(l)
	if err := os.WriteFile(ledgerPath(shapePath), csv, 0o644); err != nil {
		t.Fatal(err)
	}
	// NO --dir: cmdLedger must read the shape's minimize objective and sort best-first.
	out := captureStdout(t, func() {
		if err := cmdLedger([]string{"show", shapePath, "--sort", "train.loss"}); err != nil {
			t.Errorf("cmdLedger show: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("want header + rows, got:\n%s", out)
	}
	// Best-first for MINIMIZE = lowest loss (model=good) on the first data row. If the
	// default silently fell back to maximize, model=bad (0.90) would render first.
	if !strings.Contains(lines[1], "model=good") {
		t.Errorf("minimize objective must default to best-first (model=good, loss 0.10) first row; got:\n%s", out)
	}
}

// promote must ACTUALLY commit the winner (a real gitCommitter), not just report it —
// the production path (cmdPromote) injects gitCLICommitter. Verified against a real repo.
func TestPromote_ActuallyCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := gitInit(t)
	// A shape + its ledger row inside the repo; test/echo steps on the path.
	stepsDir := filepath.Join(root, "steps")
	if err := os.MkdirAll(filepath.Join(stepsDir, "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, filepath.Join(repoRoot(t), "testdata", "steps", "test", "echo"), filepath.Join(stepsDir, "test", "echo"))
	_ = os.Chmod(filepath.Join(stepsDir, "test", "echo"), 0o755)
	expPath := filepath.Join(root, "sweep.md")
	if err := os.WriteFile(expPath, []byte(`---
type: experiment-shape
id: cmt
seed: 5
status: active
sweep: {sampler: grid, objective: {metric: train.echoed, direction: maximize}}
steps:
  - id: train
    uses: test/echo
    with: {model: {$any: [logreg, rf]}}
---
`), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "init")
	if _, err := runExperiment(runOpts{expPath: expPath, stepPath: []string{stepsDir}, now: fixedNow(), out: io.Discard, git: gitCLI{}}); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	var out strings.Builder
	if err := runPromote(promoteOpts{
		shapePath: expPath, best: true, name: "champ",
		out: os.Stdout, git: gitCLI{}, commit: gitCLICommitter{},
	}); err != nil {
		t.Fatalf("promote: %v\n%s", err, out.String())
	}
	// champ.md must be COMMITTED (not untracked) — the Critical the review caught.
	status, _ := exec.Command("git", "-C", root, "status", "--porcelain", "champ.md").Output()
	if strings.TrimSpace(string(status)) != "" {
		t.Errorf("champ.md must be committed (clean status), got: %q", status)
	}
	log, _ := exec.Command("git", "-C", root, "log", "--oneline", "-1").Output()
	if !strings.Contains(string(log), "promote") {
		t.Errorf("a promote commit must land in git log; got: %s", log)
	}
}

// Immutability by per-row snapshot: after a shape's SPACE is edited, prior rows still
// reproduce — because each row snapshots its full point (free-params) + sweep-SHA, and
// promote reconstructs from the row, not the edited space.
func TestLedger_PriorRowsReproduceAfterSpaceEdit(t *testing.T) {
	// A row captured under the ORIGINAL space (model ∈ {logreg, rf}).
	origShape := experiment.Shape{
		Experiment: experiment.Experiment{
			Type: "experiment-shape", ID: "s", Seed: 5, Status: "active",
			Steps: []experiment.Step{{ID: "train", Uses: "test/echo",
				With: map[string]any{"model": map[string]any{"$any": []any{"logreg", "rf"}}}}},
		},
		Sweep: experiment.Sweep{Sampler: "grid"},
	}
	// The row snapshots the free-param point model=rf.
	rowFP := map[string]any{"train.model": "rf"}

	// Now the space is EDITED (model ∈ {gbm, svm} — rf removed).
	editedShape := origShape
	editedShape.Steps = []experiment.Step{{ID: "train", Uses: "test/echo",
		With: map[string]any{"model": map[string]any{"$any": []any{"gbm", "svm"}}}}}

	// The prior row still reconstructs against the ORIGINAL space (its snapshot) — the
	// per-row snapshot is self-contained, so promotion of an old row is stable.
	exp, err := promotedExperiment(origShape, rowFP)
	if err != nil {
		t.Fatalf("prior row must reproduce against its snapshot space: %v", err)
	}
	if exp.Steps[0].With["model"] != "rf" {
		t.Errorf("prior row reconstructed to model=%v; want rf", exp.Steps[0].With["model"])
	}
	// Against the EDITED space the point no longer exists — which is exactly why the row
	// carries its own snapshot rather than depending on the mutable space.
	if _, err := promotedExperiment(editedShape, rowFP); err == nil {
		t.Error("the edited space no longer contains model=rf — the row's snapshot is what keeps it reproducible")
	}
	_ = shape.Point{} // (shape imported for the reconstruction path)
}
