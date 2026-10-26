package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
	"github.com/xianxu/metis/pkg/ledger"
	"github.com/xianxu/metis/pkg/record"
)

// scoreFakeExec is one nondeterministic step: it reports *score as train.score, reads
// train.py from codeRepo (so the run has a code fingerprint), and fails when fail is set.
type scoreFakeExec struct {
	codeRepo string
	score    *float64
	fail     bool
}

func (f scoreFakeExec) Execute(_ context.Context, step experiment.Step, runDir string) (experiment.StepResult, error) {
	stepDir := filepath.Join(runDir, step.ID)
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		return experiment.StepResult{}, err
	}
	b, _ := json.Marshal(readSet{Roots: map[string][]string{f.codeRepo: {"train.py"}}})
	if err := os.WriteFile(filepath.Join(stepDir, "reads.json"), b, 0o644); err != nil {
		return experiment.StepResult{}, err
	}
	if f.fail {
		return experiment.StepResult{}, fmt.Errorf("injected step failure")
	}
	return experiment.StepResult{Metrics: map[string]float64{"score": *f.score}}, nil
}

const repeatExpMD = `---
type: experiment
id: bench
seed: 3
status: active
steps:
  - id: train
    uses: test/echo
    with: {k: 1, cache: {policy: never}}
---
`

func repeatSetup(t *testing.T, md string) (expPath, codeRepo string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH (point address and code capture use git)")
	}
	ws, codeRepo := t.TempDir(), t.TempDir()
	mustRun(t, codeRepo, "git", "init", "-q")
	if err := os.WriteFile(filepath.Join(codeRepo, "train.py"), []byte("v = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, codeRepo, "code") // a later edit is then a dirty, captured blob: a new fingerprint
	expPath = filepath.Join(ws, "exp.md")
	if err := os.WriteFile(expPath, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	// A tracked experiment has a content address, so its run dir is named by its point address.
	mustRun(t, ws, "git", "init", "-q")
	gitCommitAll(t, ws, "experiment")
	return expPath, codeRepo
}

func runRepeat(expPath, codeRepo, label string, score float64, mut func(*runOpts)) (*bytes.Buffer, error) {
	var out bytes.Buffer
	o := runOpts{
		expPath: expPath,
		repeat:  label,
		now:     fixedNow(),
		git:     fakeGitProbe{name: "metis", sha: "sha"},
		cache:   true,
		exec:    scoreFakeExec{codeRepo: codeRepo, score: &score},
		out:     &out,
	}
	if mut != nil {
		mut(&o)
	}
	_, err := runExperiment(o)
	return &out, err
}

func mustLedger(t *testing.T, expPath string) ledger.Ledger {
	t.Helper()
	l, err := loadLedger(expPath)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func repeatDirs(t *testing.T, expPath string) []string {
	t.Helper()
	ents, _ := os.ReadDir(filepath.Join(filepath.Dir(expPath), "runs"))
	var dirs []string
	for _, e := range ents {
		if e.IsDir() && strings.Contains(e.Name(), ".r-") {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

// Two repeats of one config are two samples: two dirs, two records carrying their label,
// two ledger rows, and one family (n=2) in the aggregate view.
func TestRepeat_IdenticalConfigRepeatsAreDistinctSamples(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	for label, score := range map[string]float64{"1": 1, "2": 3} {
		if _, err := runRepeat(expPath, codeRepo, label, score, nil); err != nil {
			t.Fatalf("repeat %s: %v", label, err)
		}
	}
	dirs := repeatDirs(t, expPath)
	if len(dirs) != 2 {
		t.Fatalf("want 2 repeat dirs, got %v", dirs)
	}
	for _, d := range dirs {
		rec, err := readFinalRecord(filepath.Join(filepath.Dir(expPath), "runs", d))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(d, ".r-"+rec.Repeat) || rec.CodeFingerprint == "" {
			t.Errorf("record %s: repeat %q fingerprint %q", d, rec.Repeat, rec.CodeFingerprint)
		}
	}
	l := mustLedger(t, expPath)
	if len(l.Rows) != 2 || l.Rows[0].Repeat == l.Rows[1].Repeat || l.Rows[0].PointAddr != l.Rows[1].PointAddr {
		t.Fatalf("want 2 rows of one point with distinct labels, got %+v", l.Rows)
	}
	v := ledger.AggregateView(l, "train.score")
	if len(v.Rows) != 1 || v.Rows[0].Repeat != ledger.AggregateRepeat {
		t.Fatalf("want one family row, got %+v", v.Rows)
	}
	m := v.Rows[0].Metrics
	if m["train.score"] != 2 || m["train.score.n"] != 2 || m[ledger.RepeatsTotal] != 2 {
		t.Fatalf("family metrics = %v", m)
	}
	var shown bytes.Buffer
	if err := showLedger(expPath, "", "train.score", "maximize", 0, &shown); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shown.String(), "repeat") || strings.Contains(shown.String(), "fold") {
		t.Errorf("ledger show must render the repeat family without a fold column:\n%s", shown.String())
	}
}

// A used label refuses at the claim, before any step runs, and leaves that execution's
// record untouched.
func TestRepeat_LabelReuseRefuses(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	if _, err := runRepeat(expPath, codeRepo, "1", 1, nil); err != nil {
		t.Fatal(err)
	}
	recPath := filepath.Join(filepath.Dir(expPath), "runs", repeatDirs(t, expPath)[0], "record.json")
	before, _ := os.ReadFile(recPath)
	_, err := runRepeat(expPath, codeRepo, "1", 99, nil)
	if err == nil || !strings.Contains(err.Error(), "already executed") {
		t.Fatalf("want a reuse refusal, got %v", err)
	}
	if strings.Contains(err.Error(), "ledger publish") {
		t.Errorf("the row is present, so no recovery hint is due: %v", err)
	}
	if after, _ := os.ReadFile(recPath); !bytes.Equal(before, after) {
		t.Fatal("a refused repeat must not touch the existing record")
	}
	if n := len(mustLedger(t, expPath).Rows); n != 1 {
		t.Fatalf("rows = %d, want 1", n)
	}
}

// A code edit between repeats gives two fingerprints, so two families.
func TestRepeat_CodeEditSplitsFamilies(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	if _, err := runRepeat(expPath, codeRepo, "1", 1, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeRepo, "train.py"), []byte("v = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runRepeat(expPath, codeRepo, "2", 3, nil); err != nil {
		t.Fatal(err)
	}
	l := mustLedger(t, expPath)
	if len(l.Rows) != 2 || l.Rows[0].CodeFingerprint == l.Rows[1].CodeFingerprint {
		t.Fatalf("want 2 rows with distinct fingerprints, got %+v", l.Rows)
	}
	if v := ledger.AggregateView(l, "train.score"); len(v.Rows) != 2 {
		t.Fatalf("want 2 families, got %+v", v.Rows)
	}
}

// A crash between the record and the row (the skipPublish seam): the record is final, the
// row is missing; reusing the label refuses and names the recovery verb; `ledger publish`
// adds the row; a second publish is a no-op; a conflicting record refuses.
func TestRepeat_InterruptedPublicationRecovers(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	if _, err := runRepeat(expPath, codeRepo, "1", 1, func(o *runOpts) { o.skipPublish = true }); err != nil {
		t.Fatal(err)
	}
	if n := len(mustLedger(t, expPath).Rows); n != 0 {
		t.Fatalf("rows = %d before publish, want 0", n)
	}
	runDir := filepath.Join(filepath.Dir(expPath), "runs", repeatDirs(t, expPath)[0])
	_, err := runRepeat(expPath, codeRepo, "1", 1, nil)
	if err == nil || !strings.Contains(err.Error(), "metis ledger publish") {
		t.Fatalf("want a refusal naming the recovery verb, got %v", err)
	}
	var out bytes.Buffer
	for i := 0; i < 2; i++ { // the second publish is the idempotent no-op
		if err := cmdLedgerPublish([]string{expPath, runDir}, &out); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}
	l := mustLedger(t, expPath)
	if len(l.Rows) != 1 || l.Rows[0].Repeat != "1" || l.Rows[0].Metrics["train.score"] != 1 {
		t.Fatalf("after publish: %+v", l.Rows)
	}

	// A hand-edited record that disagrees with its published row refuses; the ledger is unchanged.
	rec, _ := readFinalRecord(runDir)
	rec.Steps[0].Metrics = map[string]float64{"score": 42}
	if err := writeRecordJSON(runDir, rec); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(ledgerPath(expPath))
	if err := cmdLedgerPublish([]string{expPath, runDir}, &out); err == nil || !strings.Contains(err.Error(), "conflicting data") {
		t.Fatalf("want a conflict refusal, got %v", err)
	}
	if after, _ := os.ReadFile(ledgerPath(expPath)); !bytes.Equal(before, after) {
		t.Fatal("a refused publish must leave the ledger unchanged")
	}
}

// An execution that never finished (no record) cannot be published, and a dir outside the
// experiment's runs/ is refused.
func TestLedgerPublish_RefusesInterruptedOrForeignDir(t *testing.T) {
	expPath, _ := repeatSetup(t, repeatExpMD)
	runDir := filepath.Join(filepath.Dir(expPath), "runs", "abc.r-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cmdLedgerPublish([]string{expPath, runDir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "interrupted execution") {
		t.Fatalf("want an interrupted-execution refusal, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "record.json"), []byte(`{"run_id": "abc.r-1", "sta`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdLedgerPublish([]string{expPath, runDir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "interrupted execution") {
		t.Fatalf("a truncated record must refuse, got %v", err)
	}
	if err := cmdLedgerPublish([]string{expPath, t.TempDir()}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "not a run of") {
		t.Fatalf("want a foreign-dir refusal, got %v", err)
	}
}

// A plain run gets a row, and a rerun (which overwrites its dir) replaces it: one row,
// equal to the new record. A failed run publishes status failed with no metrics.
func TestPlainRun_RowMirrorsRunDir(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	for _, score := range []float64{1, 2} {
		if _, err := runRepeat(expPath, codeRepo, "", score, nil); err != nil {
			t.Fatal(err)
		}
	}
	l := mustLedger(t, expPath)
	if len(l.Rows) != 1 || l.Rows[0].Repeat != "" || l.Rows[0].Metrics["train.score"] != 2 {
		t.Fatalf("want one plain row with the latest score, got %+v", l.Rows)
	}
	runDir := filepath.Join(filepath.Dir(expPath), "runs", l.Rows[0].PointAddr)
	rec, err := readFinalRecord(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if !ledger.SameRow(l.Rows[0], rowFromRecord(rec)) {
		t.Fatalf("row %+v != record-derived %+v", l.Rows[0], rowFromRecord(rec))
	}

	_, err = runRepeat(expPath, codeRepo, "", 0, func(o *runOpts) { o.exec = scoreFakeExec{codeRepo: codeRepo, fail: true} })
	if err == nil {
		t.Fatal("want the step failure")
	}
	l = mustLedger(t, expPath)
	if len(l.Rows) != 1 || l.Rows[0].Status != "failed" || l.Rows[0].Metrics != nil {
		t.Fatalf("a failed rerun must replace the row with status failed and no metrics: %+v", l.Rows)
	}
}

// Refusals before any effect: a bad label, --run with --repeat, and --repeat on a sweep.
func TestRepeat_RefusalsBeforeAnyEffect(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	if _, err := runRepeat(expPath, codeRepo, "../x", 1, nil); err == nil || !strings.Contains(err.Error(), "invalid repeat label") {
		t.Fatalf("bad label: %v", err)
	}
	if _, err := runRepeat(expPath, codeRepo, "1", 1, func(o *runOpts) { o.runID = "mine" }); err == nil || !strings.Contains(err.Error(), "--run and --repeat") {
		t.Fatalf("--run with --repeat: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(expPath), "runs")); !os.IsNotExist(err) {
		t.Fatalf("a refused run must create nothing (runs/ stat: %v)", err)
	}
	shapePath := filepath.Join(filepath.Dir(expPath), "shape.md")
	if err := os.WriteFile(shapePath, []byte(foldShape("[logreg, rf]", "")), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runRepeat(shapePath, codeRepo, "1", 1, nil); err == nil || !strings.Contains(err.Error(), "--repeat is for a plain experiment") {
		t.Fatalf("sweep: %v", err)
	}
}

// A repeat of an experiment with no uncached step only replays the cache: metis says so,
// and still runs (a deterministic replay is a legitimate check).
func TestRepeat_WarnsWhenNothingIsUncached(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, strings.Replace(repeatExpMD, ", cache: {policy: never}", "", 1))
	out, err := runRepeat(expPath, codeRepo, "1", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "replays the cache") {
		t.Fatalf("want the replay warning:\n%s", out.String())
	}
}

// Parallel repeats publishing into one ledger lose no rows (the flock serializes the
// load-append-rewrite), and two claims of one label leave exactly one winner.
func TestPublishRows_ParallelWritersAndClaims(t *testing.T) {
	dir := t.TempDir()
	expPath := filepath.Join(dir, "exp.md")
	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			row := ledger.Row{CodeFingerprint: "fp", PointAddr: "addr", Repeat: fmt.Sprintf("r%d", i), Status: "ok",
				Metrics: map[string]float64{"train.score": float64(i)}}
			errs <- publishRows(ledgerPath(expPath), func(l *ledger.Ledger) error { return l.Publish(row) })
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := len(mustLedger(t, expPath).Rows); got != n {
		t.Fatalf("rows = %d, want %d (a lost update)", got, n)
	}

	runDir := filepath.Join(dir, "runs", "addr.r-x")
	wins := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { wins <- claimRunDir(expPath, runDir, "x") }()
	}
	a, b := <-wins, <-wins
	if (a == nil) == (b == nil) {
		t.Fatalf("want exactly one claim winner, got %v and %v", a, b)
	}
}

// rowFromRecord is the one record→row mapping: point address as the identity, the label,
// and no metrics for a failed run.
func TestRowFromRecord(t *testing.T) {
	ok := record.RunRecord{RunID: "pa.r-2", PointAddress: "pa", CodeFingerprint: "fp", Repeat: "2", Status: "ok",
		Steps: []record.StepRecord{{StepID: "train", Metrics: map[string]float64{"score": 0.5}}}}
	r := rowFromRecord(ok)
	if r.PointAddr != "pa" || r.Repeat != "2" || r.Metrics["train.score"] != 0.5 || rowRunID(r) != "pa.r-2" {
		t.Fatalf("row = %+v", r)
	}
	failed := ok
	failed.Status = "failed"
	if r := rowFromRecord(failed); r.Metrics != nil || r.Status != "failed" {
		t.Fatalf("failed row = %+v", r)
	}
}

// BR-1: a record whose repeat label is not a valid label (or does not match its dir) is
// refused by publish, so the ledger stays loadable.
func TestLedgerPublish_RefusesInvalidOrMisplacedRepeatLabel(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	if _, err := runRepeat(expPath, codeRepo, "1", 1, func(o *runOpts) { o.skipPublish = true }); err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(filepath.Dir(expPath), "runs", repeatDirs(t, expPath)[0])
	rec, _ := readFinalRecord(runDir)
	for _, label := range []string{"*", "2"} { // invalid; valid but names another dir
		rec.Repeat = label
		if err := writeRecordJSON(runDir, rec); err != nil {
			t.Fatal(err)
		}
		if err := cmdLedgerPublish([]string{expPath, runDir}, &bytes.Buffer{}); err == nil {
			t.Fatalf("label %q: want a refusal", label)
		}
	}
	if _, err := loadLedger(expPath); err != nil {
		t.Fatalf("the ledger must stay loadable: %v", err)
	}
}

// BR-2: an untracked experiment has no content address, so its run dir would be timestamped
// and a label could be reused; --repeat refuses before any effect.
func TestRepeat_RefusesUntrackedExperiment(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	untracked := filepath.Join(t.TempDir(), "exp.md")
	if err := os.WriteFile(untracked, []byte(repeatExpMD), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = expPath
	if _, err := runRepeat(untracked, codeRepo, "1", 1, nil); err == nil || !strings.Contains(err.Error(), "tracked in git") {
		t.Fatalf("want an untracked refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(untracked), "runs")); !os.IsNotExist(err) {
		t.Fatalf("a refused repeat must create nothing (runs/ stat: %v)", err)
	}
}

// BR-3: repeats at one point under two fingerprints each show their own record's commit in
// the fingerprint view (records are keyed by run id, not by point).
func TestRepeat_FingerprintCohortsUseOwnRecords(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, repeatExpMD)
	if _, err := runRepeat(expPath, codeRepo, "1", 1, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeRepo, "train.py"), []byte("v = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runRepeat(expPath, codeRepo, "2", 3, nil); err != nil {
		t.Fatal(err)
	}
	led := mustLedger(t, expPath)
	recs := loadLedgerRecords(expPath, led)
	if len(recs) != 2 {
		t.Fatalf("want one record per repeat, got %d", len(recs))
	}
	cs := cohortSummaries(led, recs)
	if len(cs) != 2 {
		t.Fatalf("want 2 cohorts, got %+v", cs)
	}
	for _, c := range cs {
		var own record.RunRecord
		for _, r := range led.Rows {
			if r.CodeFingerprint == c.Fingerprint {
				own = recs[rowRunID(r)]
			}
		}
		if c.Records != 1 || c.Commit != recCommit(own) || c.Commit == "" {
			t.Errorf("cohort %s: records %d commit %q, want its own record's commit %q", short(c.Fingerprint), c.Records, c.Commit, recCommit(own))
		}
	}
}

// A repeat refused at validation (before any step ran) releases its claim: the label is
// not spent on an execution that never happened.
func TestRepeat_ValidationFailureReleasesClaim(t *testing.T) {
	bad := strings.Replace(repeatExpMD, "    uses: test/echo\n", "    uses: test/echo\n    needs: [missing]\n", 1)
	expPath, codeRepo := repeatSetup(t, bad)
	if _, err := runRepeat(expPath, codeRepo, "1", 1, nil); err == nil {
		t.Fatal("want the validation error")
	}
	if dirs := repeatDirs(t, expPath); len(dirs) != 0 {
		t.Fatalf("the claim must be released, found %v", dirs)
	}
}

// A config that cannot be content-addressed (a non-finite value) has no point address, so a
// repeat refuses rather than claiming a timestamped dir that would never collide.
func TestRepeat_RefusesUnaddressableConfig(t *testing.T) {
	expPath, codeRepo := repeatSetup(t, strings.Replace(repeatExpMD, "with: {k: 1,", "with: {k: .nan,", 1))
	if _, err := runRepeat(expPath, codeRepo, "1", 1, nil); err == nil || !strings.Contains(err.Error(), "needs a point address") {
		t.Fatalf("want a point-address refusal, got %v", err)
	}
	if dirs := repeatDirs(t, expPath); len(dirs) != 0 {
		t.Fatalf("nothing may be claimed, found %v", dirs)
	}
}
