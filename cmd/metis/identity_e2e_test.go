package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"
)

// TestSingleRun_ContentAddressedDir is the metis#27 single-run symmetry: a `metis run` with
// no --run names its dir by the run's point-address (a 64-hex content-address, like a sweep
// point), not run-<timestamp>. The dir name equals the record's point_address (they can't
// desync). Uses the no-uv test/echo step so it runs in a bare checkout.
func TestSingleRun_ContentAddressedDir(t *testing.T) {
	mroot := repoRoot(t) // for testdata/steps/test/echo
	root := gitInit(t)
	expPath := filepath.Join(root, "exp.md")
	if err := os.WriteFile(expPath, []byte(`---
type: experiment
id: ca-e2e
seed: 7
status: active
steps:
  - id: prep
    uses: test/echo
    with: {k: 5}
---

# ca-e2e
`), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "add exp")

	opts := runOpts{
		expPath:  expPath,
		stepPath: []string{filepath.Join(mroot, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		out:      io.Discard,
	}
	if _, err := runExperiment(opts); err != nil {
		t.Fatalf("runExperiment: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(root, "runs"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("want exactly one run dir, got %v (err %v)", entries, err)
	}
	name := entries[0].Name()
	if len(name) != 64 {
		t.Errorf("single-run dir must be content-addressed (64-hex point-address), got %q", name)
	}
	// The dir name equals the record's point_address (no desync).
	rb, _ := os.ReadFile(filepath.Join(root, "runs", name, "record.json"))
	var rec record.RunRecord
	if err := json.Unmarshal(rb, &rec); err != nil {
		t.Fatal(err)
	}
	if string(rec.PointAddress) != name {
		t.Errorf("run dir %q must equal the record point_address %q", name, rec.PointAddress)
	}
}

// TestCodeIdentity_TwoRowsOnCodeChange is the metis#27 acceptance (Done-when): two runs of
// the SAME config (same point-address — the intent) with DIFFERENT code (an edited
// in-closure .py) produce two DISTINCT ledger rows — distinct code_fingerprints, neither
// overwritten. Exercises the real path: captureSweepCode → backfillCodeManifest sets the
// fingerprint over the run-end D closure → rowsFromManifest → the ledger dedups on
// (point_addr, code_fingerprint). No uv/python: the closure is a hand-placed reads.json +
// a real git blob, which is exactly what a traced run leaves behind.
func TestCodeIdentity_TwoRowsOnCodeChange(t *testing.T) {
	root := gitInit(t)
	code := filepath.Join(root, "model.py")
	if err := os.WriteFile(code, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "init")

	const pa = "point-addr-fixed" // both runs share the intent (same config+shape+seed)
	expPath := filepath.Join(root, "sweep.md")
	man := sweepManifest{ShapeRunID: "srun", Points: []pointRun{
		{RunID: pa, Fold: 0, Status: "ok", FreeParams: map[string]any{"model": "rf"}},
	}}
	o := runOpts{expPath: expPath, stepPath: []string{filepath.Join(root, "steps")}}

	// setupPoint (re)writes the point's reads.json (naming the in-closure model.py) + a fresh
	// record.json (no fingerprint yet) — the state a run leaves for capture to backfill.
	setupPoint := func() {
		stepDir := filepath.Join(root, "runs", pa, "train")
		if err := os.MkdirAll(stepDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeJSON(t, filepath.Join(stepDir, "reads.json"), readSet{Roots: map[string][]string{root: {"model.py"}}})
		writeJSON(t, filepath.Join(root, "runs", pa, "record.json"), record.RunRecord{
			RunID: pa, PointAddress: pa,
			Steps: []record.StepRecord{{StepID: "train", Metrics: map[string]float64{"train.fold_score": 0.9}}},
		})
	}
	capture := func() record.RunRecord {
		if _, err := captureSweepCode(o, man); err != nil {
			t.Fatalf("captureSweepCode: %v", err)
		}
		rb, _ := os.ReadFile(filepath.Join(root, "runs", pa, "record.json"))
		var rec record.RunRecord
		if err := json.Unmarshal(rb, &rec); err != nil {
			t.Fatal(err)
		}
		return rec
	}

	// Run 1 (code v1).
	setupPoint()
	rec1 := capture()
	if rec1.CodeFingerprint == "" {
		t.Fatal("code_fingerprint must be set after capture (backfillCodeManifest)")
	}

	// Edit the in-closure .py, then run 2 (code v2) — SAME point-address.
	if err := os.WriteFile(code, []byte("x = 2  # changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	setupPoint()
	rec2 := capture()

	if rec1.CodeFingerprint == rec2.CodeFingerprint {
		t.Fatalf("a code change must change the fingerprint (fp1=%s fp2=%s)", rec1.CodeFingerprint, rec2.CodeFingerprint)
	}

	// Both runs' rows land in the ledger: SAME point-address, DIFFERENT fingerprint → 2 rows.
	var led ledger.Ledger
	led.Append(rowsFromManifest(man, map[string]record.RunRecord{pa: rec1})...)
	led.Append(rowsFromManifest(man, map[string]record.RunRecord{pa: rec2})...)
	if len(led.Rows) != 2 {
		t.Fatalf("same config + changed code must keep BOTH rows (the metis#27 collision fix), got %d", len(led.Rows))
	}
	if led.Rows[0].PointAddr != pa || led.Rows[1].PointAddr != pa {
		t.Errorf("both rows should share the point-address %q, got %q and %q", pa, led.Rows[0].PointAddr, led.Rows[1].PointAddr)
	}
	if led.Rows[0].CodeFingerprint == led.Rows[1].CodeFingerprint {
		t.Error("the two rows must carry DISTINCT code_fingerprints")
	}
}
