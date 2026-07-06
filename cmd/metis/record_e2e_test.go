package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/record"
)

// fakeGitProbe returns canned provenance so record assembly is exercised without a
// real git repo (the e2e workspace is a bare t.TempDir()). A non-nil err simulates
// running outside a git repo.
type fakeGitProbe struct {
	name, sha string
	dirty     bool
	err       error
}

func (f fakeGitProbe) Probe(string) (string, string, bool, error) {
	return f.name, f.sha, f.dirty, f.err
}

// TestRunExperiment_WritesProvenanceRecord is the metis#3 M2 e2e: a `metis run`
// through the no-uv test/echo fake steps writes runs/<id>/record.json that conforms
// to #RunRecord, carries the minted point-address + repo provenance + per-step
// output hashes, and appends a knob→score line to ## Runs. It also confirms two
// identical runs mint the SAME point-address (the repro-identity guarantee). No uv,
// so it runs in a bare checkout.
func TestRunExperiment_WritesProvenanceRecord(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := filepath.Join(ws, "exp.md")
	expMD := `---
type: experiment
id: rec-e2e
seed: 7
status: active
steps:
  - id: prep
    uses: test/echo
    with: {k: 5}
  - id: train
    uses: test/echo
    needs: [prep]
    with: {model: logreg}
---

# rec-e2e
`
	if err := os.WriteFile(expPath, []byte(expMD), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := runOpts{
		expPath:  expPath,
		runID:    "run-rec",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "deadbeef", dirty: false},
		out:      io.Discard,
	}
	run, err := runExperiment(opts)
	if err != nil {
		t.Fatalf("runExperiment: %v", err)
	}
	if run.Status != "ok" {
		t.Fatalf("run status = %q; want ok", run.Status)
	}

	// record.json written, parseable, with a well-formed point-address + provenance.
	recPath := filepath.Join(ws, "runs", "run-rec", "record.json")
	rb, err := os.ReadFile(recPath)
	if err != nil {
		t.Fatalf("record.json not written: %v", err)
	}
	var rec record.RunRecord
	if err := json.Unmarshal(rb, &rec); err != nil {
		t.Fatalf("parse record.json: %v", err)
	}
	if len(rec.PointAddress) != 64 {
		t.Errorf("point-address = %q; want a 64-hex hash", rec.PointAddress)
	}
	if rec.RepoSHAs["metis"] != "deadbeef" || rec.Dirty {
		t.Errorf("repo provenance wrong: shas=%v dirty=%v", rec.RepoSHAs, rec.Dirty)
	}
	if len(rec.Steps) != 2 || rec.Steps[0].StepID != "prep" || rec.Steps[1].StepID != "train" {
		t.Fatalf("step records wrong: %+v", rec.Steps)
	}
	if rec.Steps[0].Code.Commit != "deadbeef" {
		t.Errorf("step code commit = %q; want deadbeef", rec.Steps[0].Code.Commit)
	}
	if rec.Steps[0].OutputHash == "" {
		t.Error("prep OutputHash empty — the echoed.json artifact should have been hashed")
	}

	// The real record.json conforms to the CUE #RunRecord (drift guard on the actual
	// output shape, not just a hand-built fixture). Skips when cue is unavailable.
	if _, err := exec.LookPath("cue"); err == nil {
		cueFile := filepath.Join(root, "construct", "vocabulary", "experiment.cue")
		if out, err := exec.Command("cue", "vet", "-d", "#RunRecord", recPath, cueFile).CombinedOutput(); err != nil {
			t.Fatalf("real record.json failed #RunRecord conformance: %v\n%s", err, out)
		}
	}

	// #13: the config .md is immutable input — the run's knob→score provenance lives in
	// record.json (validated above via #RunRecord conformance), NOT appended to the config body.
	body, err := os.ReadFile(expPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "## Runs") {
		t.Errorf("run mutated the config .md (must be immutable input):\n%s", string(body))
	}

	// A second identical run mints the SAME point-address (config+repo+seed
	// unchanged) — the repro-identity guarantee, even with a fresh run id.
	opts.runID = "run-rec-2"
	if _, err := runExperiment(opts); err != nil {
		t.Fatalf("second run: %v", err)
	}
	rb2, err := os.ReadFile(filepath.Join(ws, "runs", "run-rec-2", "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec2 record.RunRecord
	if err := json.Unmarshal(rb2, &rec2); err != nil {
		t.Fatal(err)
	}
	if rec2.PointAddress != rec.PointAddress {
		t.Errorf("identical runs must share a point-address: %q != %q", rec2.PointAddress, rec.PointAddress)
	}
}

// A run outside a git repo must NOT fail — provenance capture degrades gracefully:
// the run completes, warns, and writes a record with no repo-SHAs (the design's
// "v1: warn", not a hard requirement).
func TestRunExperiment_DegradesWithoutGitProvenance(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := filepath.Join(ws, "exp.md")
	expMD := `---
type: experiment
id: nogit
seed: 1
status: active
steps:
  - id: only
    uses: test/echo
    with: {a: 1}
---
`
	if err := os.WriteFile(expPath, []byte(expMD), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	run, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-nogit",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{err: errors.New("not a git repository")},
		out:      &out,
	})
	if err != nil {
		t.Fatalf("run must not fail without git provenance: %v", err)
	}
	if run.Status != "ok" {
		t.Fatalf("run status = %q; want ok", run.Status)
	}
	if !strings.Contains(out.String(), "warning") {
		t.Errorf("expected a provenance warning on stderr/out; got:\n%s", out.String())
	}

	rb, err := os.ReadFile(filepath.Join(ws, "runs", "run-nogit", "record.json"))
	if err != nil {
		t.Fatalf("record.json still expected without git: %v", err)
	}
	var rec record.RunRecord
	if err := json.Unmarshal(rb, &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.RepoSHAs) != 0 {
		t.Errorf("no-git record should carry no repo-SHAs, got %v", rec.RepoSHAs)
	}
	if len(rec.PointAddress) != 64 {
		t.Errorf("point-address should still mint (from config+seed): %q", rec.PointAddress)
	}
}
