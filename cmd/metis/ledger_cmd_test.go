package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"
)

// A minimal parse-able experiment-shape (runPromote/cmdLedger only ParseShape it + read the
// objective; they don't run it), plus a per-fold ledger sidecar written beside it.
const foldShapeForLedger = `---
type: experiment-shape
id: s
seed: 1
status: active
data:
  - id: adapt
    uses: titanic/adapt
    with: {out: ../data/x}
pipeline:
  - id: train
    uses: metis/train
    needs: [adapt]
    with: {dataset: adapt, model: {$any: [a, b]}}
sweeper:
  sampler: grid
  resample: {cv: {k: 2, stratify: false}}
  objective: {metric: train.fold_score, direction: maximize, select: {argmax-mean: {}}}
---
`

func writePerFoldLedger(t *testing.T, dir string) string {
	t.Helper()
	shapePath := filepath.Join(dir, "s.md")
	if err := os.WriteFile(shapePath, []byte(foldShapeForLedger), 0o644); err != nil {
		t.Fatal(err)
	}
	f0, f1 := 0, 1
	var led ledger.Ledger
	led.Append(
		ledger.Row{CodeFingerprint: "cf", PointAddr: "a0", FreeParams: map[string]any{"train.model": "a"}, Fold: &f0, Metrics: map[string]float64{"train.fold_score": 0.80}, Status: "ok"},
		ledger.Row{CodeFingerprint: "cf", PointAddr: "a1", FreeParams: map[string]any{"train.model": "a"}, Fold: &f1, Metrics: map[string]float64{"train.fold_score": 0.90}, Status: "ok"},
		ledger.Row{CodeFingerprint: "cf", PointAddr: "b0", FreeParams: map[string]any{"train.model": "b"}, Fold: &f0, Metrics: map[string]float64{"train.fold_score": 0.70}, Status: "ok"},
		ledger.Row{CodeFingerprint: "cf", PointAddr: "b1", FreeParams: map[string]any{"train.model": "b"}, Fold: &f1, Metrics: map[string]float64{"train.fold_score": 0.72}, Status: "ok"},
	)
	b, err := ledger.Encode(led)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath(shapePath), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return shapePath
}

// `ledger show --sort` on a per-fold ledger renders the AggregateView (per-config mean,SE),
// not the raw fold rows — the honest leaderboard.
func TestShowLedger_AggregatesPerConfig(t *testing.T) {
	dir := t.TempDir()
	shapePath := writePerFoldLedger(t, dir)
	var out strings.Builder
	if err := showLedger(shapePath, "", "train.fold_score", "maximize", 0, &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	// 2 configs (a, b) — not 4 raw fold rows. Config a (mean 0.85) sorts above b (0.71).
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) != 1+2 { // header + 2 config rows
		t.Fatalf("expected a header + 2 per-config rows (aggregated), got %d lines:\n%s", len(lines), s)
	}
	// argmax-mean: config a (0.85) is best-first.
	if !strings.Contains(lines[1], "model=a") {
		t.Errorf("best-first row should be config a (mean 0.85); got: %s", lines[1])
	}
	if !strings.Contains(s, "train.fold_score.se") {
		t.Errorf("the aggregate view should carry the SE column; got:\n%s", s)
	}
}

// hoistShapePath pulls the single <shape.md> positional out regardless of flag position.
func TestHoistShapePath_ArgOrder(t *testing.T) {
	// flags before AND after the path both work (the stdlib-flag-stops-at-positional fix).
	for _, args := range [][]string{
		{"foo.md", "--sort", "train.fold_score"},
		{"--sort", "train.fold_score", "foo.md"},
	} {
		p, flags, err := hoistShapePath(args)
		if err != nil || p != "foo.md" {
			t.Errorf("hoistShapePath(%v) = (%q, %v); want foo.md", args, p, err)
		}
		if len(flags) != 2 {
			t.Errorf("hoistShapePath(%v) flags = %v; want the 2 flag tokens", args, flags)
		}
	}
	// Missing / duplicate positionals error.
	if _, _, err := hoistShapePath([]string{"--sort", "x"}); err == nil {
		t.Error("missing <shape.md> must error")
	}
	if _, _, err := hoistShapePath([]string{"a.md", "b.md"}); err == nil {
		t.Error("two <shape.md> positionals must error")
	}
}

// writeFingerprintFixture builds a multi-cohort workspace: a shape + ledger CSV spanning
// two fingerprint cohorts and a legacy blank row, plus runs/<id>/record.json for the two
// cohort runs (r-old lacks a record — a cleaned run dir the summary must tolerate).
func writeFingerprintFixture(t *testing.T, dir string) string {
	t.Helper()
	shapePath := filepath.Join(dir, "s.md")
	if err := os.WriteFile(shapePath, []byte(foldShapeForLedger), 0o644); err != nil {
		t.Fatal(err)
	}
	var led ledger.Ledger
	led.Append(
		ledger.Row{CodeFingerprint: "aaaa1111ffff", PointAddr: "r-a", Level: "inner",
			FreeParams: map[string]any{"train.model": "a"}, Metrics: map[string]float64{"train.fold_score": 0.8}, Status: "ok"},
		ledger.Row{CodeFingerprint: "bbbb2222ffff", PointAddr: "r-b", Level: "outer",
			FreeParams: map[string]any{"train.model": "b"}, Metrics: map[string]float64{"train.fold_score": 0.7}, Status: "ok"},
		ledger.Row{CodeFingerprint: "", PointAddr: "r-old",
			FreeParams: map[string]any{"train.model": "a"}, Metrics: map[string]float64{"train.fold_score": 0.6}, Status: "ok"},
	)
	b, err := ledger.Encode(led)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath(shapePath), b, 0o644); err != nil {
		t.Fatal(err)
	}
	recA := record.RunRecord{RunID: "r-a", Started: "2026-07-14T10:00:00Z", Finished: "2026-07-14T10:05:00Z",
		CodeFingerprint: "aaaa1111ffff", Dirty: true,
		Steps: []record.StepRecord{{StepID: "train", Code: record.CodeManifest{Commit: "commita11", CaptureStatus: "captured"}}}}
	recB := record.RunRecord{RunID: "r-b", Started: "2026-07-15T09:00:00Z", Finished: "2026-07-15T09:01:00Z",
		CodeFingerprint: "bbbb2222ffff",
		Steps:           []record.StepRecord{{StepID: "train", Code: record.CodeManifest{Commit: "commitb22", CaptureStatus: "captured"}}}}
	for id, rec := range map[string]record.RunRecord{"r-a": recA, "r-b": recB} {
		runDir := filepath.Join(dir, "runs", id)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "record.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return shapePath
}

// Drives the real `metis ledger fingerprints <shape.md>` CLI path (run() → cmdLedger),
// documented arg order (lessons.md: never call the handler directly), and asserts the
// per-cohort table: short fingerprints, the legacy group, level counts, timestamps,
// commit + dirty, capture status.
func TestLedgerFingerprints_CLI(t *testing.T) {
	shapePath := writeFingerprintFixture(t, t.TempDir())

	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	err := run([]string{"ledger", "fingerprints", shapePath})
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("run(ledger fingerprints): %v", err)
	}
	out := buf.String()
	for _, want := range []string{"aaaa1111", "bbbb2222", "(legacy)", "inner:1", "outer:1", "flat:1",
		"2026-07-14T10:00:00Z", "2026-07-15T09:01:00Z", "commit commita1, dirty, captured", "commit commitb2"} {
		if !strings.Contains(out, want) {
			t.Errorf("fingerprints output missing %q:\n%s", want, out)
		}
	}
	// Deterministic newest-last: bbbb (finished 07-15) renders after aaaa (07-14), legacy first.
	if !(strings.Index(out, "(legacy)") < strings.Index(out, "aaaa1111") &&
		strings.Index(out, "aaaa1111") < strings.Index(out, "bbbb2222")) {
		t.Errorf("cohorts must order newest-last (legacy first):\n%s", out)
	}
	// Unknown args must not be silently swallowed.
	if err := run([]string{"ledger", "fingerprints", shapePath, "--bogus"}); err == nil {
		t.Error("unexpected flag must error, not be ignored")
	}
}

// metis#39: `ledger show --fingerprint` shares select's git-style prefix resolution
// (one resolver — the flags must not diverge in matching semantics again).
func TestLedgerShow_FingerprintPrefix(t *testing.T) {
	shapePath := writeFingerprintFixture(t, t.TempDir())
	var out strings.Builder
	if err := showLedger(shapePath, "aaaa", "", "maximize", 0, &out); err != nil {
		t.Fatalf("prefix filter: %v", err)
	}
	if !strings.Contains(out.String(), "aaaa1111") || strings.Contains(out.String(), "bbbb2222") {
		t.Errorf("prefix must pin the aaaa cohort only:\n%s", out.String())
	}
	// A no-match prefix now errors with the cohort listing (was: silent "(no rows)").
	if err := showLedger(shapePath, "cccc", "", "maximize", 0, &strings.Builder{}); err == nil ||
		!strings.Contains(err.Error(), "nothing in the ledger matches") {
		t.Errorf("no-match prefix must error with the cohort listing, got: %v", err)
	}
}

// TestRenderLedger_PointColumnRoundTrips (metis#51): every row carries a short point
// handle, and the rendered handle resolves back to exactly its source row through the
// REAL --point prefix path (resolvePointRows) — the discovery surface for #41's flow.
func TestRenderLedger_PointColumnRoundTrips(t *testing.T) {
	rows := []ledger.Row{
		{PointAddr: "aaaa1111bbbb2222cccc", CodeFingerprint: "f1f1f1f1f1", Status: "ok",
			FreeParams: map[string]any{"train.model": "a"}, Metrics: map[string]float64{"cv_score": 0.8}},
		{PointAddr: "dddd3333eeee4444ffff", CodeFingerprint: "f1f1f1f1f1", Status: "ok",
			FreeParams: map[string]any{"train.model": "b"}, Metrics: map[string]float64{"cv_score": 0.7}},
	}
	var out strings.Builder
	renderLedger(&out, rows)
	got := out.String()
	if !strings.Contains(strings.SplitN(got, "\n", 2)[0], "point") {
		t.Fatalf("header missing point column:\n%s", got)
	}
	for _, r := range rows {
		handle := r.PointAddr[:8]
		if !strings.Contains(got, handle) {
			t.Errorf("row handle %s not rendered:\n%s", handle, got)
			continue
		}
		resolved, err := resolvePointRows(ledger.Ledger{Rows: rows}, handle)
		if err != nil {
			t.Errorf("rendered handle %s must resolve via the --point path: %v", handle, err)
			continue
		}
		if len(resolved) == 0 || resolved[0].PointAddr != r.PointAddr {
			t.Errorf("handle %s resolved to %+v, want its source row %s", handle, resolved, r.PointAddr)
		}
	}
}
