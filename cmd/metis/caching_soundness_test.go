package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/xianxu/metis/pkg/cache"
	"github.com/xianxu/metis/pkg/experiment"
)

// The metis#24 soundness gate — the input-addressed cache is only sound if the transitive-D
// snapshot restores upstream-CODE invalidation. These tests drive the REAL topo executor
// (Runner + cachingExecutor + git-blob-hash), because the property depends on run-time
// ordering (an edited upstream re-runs and HEALS its own entry BEFORE the downstream is
// checked) that a pure Validate/isHit unit test is structurally blind to (workshop/lessons.md).
//
// traceFakeExec is the injected inner StepExecutor (runOpts.exec seam): no subprocess, but it
// writes a real reads.json declaring each step's read-set D (pointing at editable files in a
// temp git repo) so the cache records a genuine transitive-D closure, and a nonce-varied
// artifact so a re-run can produce byte-different output (the non-determinism test).
type traceFakeExec struct {
	codeRepo string            // a git repo holding the step code files the steps "read"
	reads    map[string]string // step-id → code file (relative to codeRepo) it reads → its D
	nonce    *int              // if set, folded into a step's output bytes (output non-determinism)
	calls    *[]string         // MISS trace: the inner exec ran (a HIT skips it)
}

func (f traceFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, step.ID)
	}
	stepDir := filepath.Join(runDir, step.ID)
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		return experiment.StepResult{}, err
	}
	content := step.ID
	if f.nonce != nil {
		content = step.ID + "-" + strconv.Itoa(*f.nonce)
	}
	art := step.ID + "/out.txt"
	if err := os.WriteFile(filepath.Join(runDir, filepath.FromSlash(art)), []byte(content), 0o644); err != nil {
		return experiment.StepResult{}, err
	}
	// Declare this step's read-set D (the code file it reads) via the sensor's reads.json
	// format, so recordMiss folds it into the transitive-D closure and isHit re-hashes it.
	if rel, ok := f.reads[step.ID]; ok {
		rs := readSet{Roots: map[string][]string{f.codeRepo: {rel}}, UsedSitePackages: false}
		b, err := json.Marshal(rs)
		if err != nil {
			return experiment.StepResult{}, err
		}
		if err := os.WriteFile(filepath.Join(stepDir, "reads.json"), b, 0o644); err != nil {
			return experiment.StepResult{}, err
		}
	}
	return experiment.StepResult{Artifacts: []string{art}}, nil
}

// soundnessSetup builds a 2-step chain up→down: a temp workspace (holding the experiment +
// .metis-cache) and a temp git repo holding up.py + down.py (each step reads its own file).
func soundnessSetup(t *testing.T) (ws, codeRepo string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH (transitive-D re-hash uses git hash-object)")
	}
	ws = t.TempDir()
	codeRepo = t.TempDir()
	mustRun(t, codeRepo, "git", "init", "-q")
	if err := os.WriteFile(filepath.Join(codeRepo, "up.py"), []byte("u = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeRepo, "down.py"), []byte("d = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	md := `---
type: experiment
id: sound
seed: 5
status: active
steps:
  - id: up
    uses: test/echo
    with: {k: 1}
  - id: down
    uses: test/echo
    needs: [up]
    with: {k: 2}
---
`
	if err := os.WriteFile(filepath.Join(ws, "exp.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	return ws, codeRepo
}

func runSound(t *testing.T, ws string, fake traceFakeExec, runID string) {
	t.Helper()
	_, err := runExperiment(runOpts{
		expPath: filepath.Join(ws, "exp.md"),
		runID:   runID,
		now:     fixedNow(),
		git:     fakeGitProbe{name: "metis", sha: "sha", dirty: false},
		cache:   true,
		exec:    fake,
		out:     nil,
	})
	if err != nil {
		t.Fatalf("run %s: %v", runID, err)
	}
}

// THE gate (metis#24): a warm cache, then an edit to an UPSTREAM step's code (up.py) → the
// downstream step (down) MISSes, even though down's OWN read-set (down.py) is unchanged and
// its input-addressed K_pre (keyed on up's code-invariant K_pre) is unchanged. Only the
// transitive-D closure stored in down's OWN entry catches the edit. Driven through the real
// topo executor: up re-runs and OVERWRITES up's entry before down is checked — a scheme that
// re-hashed up's LIVE entry would see it already healed and false-HIT (the inert bug).
func TestCachingExecutor_UpstreamCodeEditMissesDownstream(t *testing.T) {
	ws, codeRepo := soundnessSetup(t)
	fake := traceFakeExec{codeRepo: codeRepo, reads: map[string]string{"up": "up.py", "down": "down.py"}}

	var cold []string
	fake.calls = &cold
	runSound(t, ws, fake, "r1")
	if !contains(cold, "up") || !contains(cold, "down") {
		t.Fatalf("cold run should run both steps, got %v", cold)
	}

	// Warm re-run (no edits): both HIT — 0 inner execs.
	var warm []string
	fake.calls = &warm
	runSound(t, ws, fake, "r2")
	if len(warm) != 0 {
		t.Fatalf("a warm re-run should HIT everything (0 inner execs), got %v", warm)
	}

	// Edit the UPSTREAM step's code — up.py is in up's D and (transitively) in down's closure.
	if err := os.WriteFile(filepath.Join(codeRepo, "up.py"), []byte("u = 2  # edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var after []string
	fake.calls = &after
	runSound(t, ws, fake, "r3")
	if !contains(after, "up") {
		t.Errorf("up (its own code edited) must MISS + re-run, got %v", after)
	}
	if !contains(after, "down") {
		t.Errorf("down MUST MISS on an UPSTREAM code edit — the transitive-D closure carries up.py "+
			"(the input-addressed K_pre alone would false-HIT); got %v", after)
	}
}

// Non-determinism robustness (metis#24): an upstream whose OUTPUT changes byte-for-byte with
// its CODE unchanged must NOT re-key the downstream — the input-addressed key drops the
// upstream output-hash term, so down HITs and serves its cached output. Under the OLD
// output-hash cache this MISSed (spurious invalidation). Evicts up's entry (identified by its
// single-ref closure) so up genuinely re-runs with a bumped nonce → different output bytes.
func TestCachingExecutor_UpstreamOutputChangeDoesNotRekeyDownstream(t *testing.T) {
	ws, codeRepo := soundnessSetup(t)
	nonce := 0
	fake := traceFakeExec{codeRepo: codeRepo, reads: map[string]string{"up": "up.py", "down": "down.py"}, nonce: &nonce}

	runSound(t, ws, fake, "r1") // cold: up→"up-0", down→"down-0" (both stored)

	// Evict ONLY up's index entry (its closure is the single-ref one — {up.py}; down's is
	// {up.py, down.py}). up then re-runs; down's entry survives so its key/closure are tested.
	evictSingleRefEntry(t, ws)
	nonce = 1 // up's re-run now emits "up-1" — byte-different output, same code

	var calls []string
	fake.calls = &calls
	runSound(t, ws, fake, "r2")

	if !contains(calls, "up") {
		t.Errorf("up (index-evicted) must re-run, got %v", calls)
	}
	if contains(calls, "down") {
		t.Errorf("down MUST HIT — its input-addressed key is invariant to up's OUTPUT change "+
			"(same up code → same up K_pre → same down key); got re-run %v", calls)
	}
	// down served its CACHED output (nonce-0 bytes), proving a real HIT, not a recompute.
	got, err := os.ReadFile(filepath.Join(ws, "runs", "r2", "down", "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "down-0" {
		t.Errorf("down should materialize its cached (nonce-0) output %q, got %q", "down-0", string(got))
	}
}

// HIT-feeds-downstream (metis#24 — the repopulation seam): a step that HITs must still feed its
// stored closure to a downstream step re-stored in the SAME run. This is invisible to the
// all-MISS gate above (there up MISSed, so its closure came from the MISS path); it surfaces
// only one edit later. Edit DOWN's own code first (up HITS, down re-stores its closure from the
// repopulated c.transitiveD[up]); THEN edit up's code — down must MISS because its re-stored
// closure still carries up.py. Reverting the HIT-repopulation line makes THIS test fail (down
// would re-store an up-less closure → false-HIT on the up edit) while the all-MISS gate stays green.
func TestCachingExecutor_HitFeedsDownstreamClosure(t *testing.T) {
	ws, codeRepo := soundnessSetup(t)
	fake := traceFakeExec{codeRepo: codeRepo, reads: map[string]string{"up": "up.py", "down": "down.py"}}

	runSound(t, ws, fake, "r1") // cold: down's closure = {up.py, down.py}

	// Edit DOWN's own code → up HITS (unchanged), down MISSes + re-stores its closure, folding
	// in up's closure ONLY via the HIT-repopulation of c.transitiveD[up].
	if err := os.WriteFile(filepath.Join(codeRepo, "down.py"), []byte("d = 2  # edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var mid []string
	fake.calls = &mid
	runSound(t, ws, fake, "r2")
	if contains(mid, "up") {
		t.Fatalf("up (unchanged) must HIT while down's own code changed, got %v", mid)
	}
	if !contains(mid, "down") {
		t.Fatalf("down (its own code edited) must MISS + re-store, got %v", mid)
	}

	// NOW edit UP's code. down's re-stored closure must still carry up.py → down MISSes. If the
	// HIT-repopulation were dropped, down would have re-stored a {down.py}-only closure and now
	// false-HIT.
	if err := os.WriteFile(filepath.Join(codeRepo, "up.py"), []byte("u = 2  # edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var after []string
	fake.calls = &after
	runSound(t, ws, fake, "r3")
	if !contains(after, "down") {
		t.Errorf("down MUST MISS on the up edit — its re-stored closure carries up.py via the HIT "+
			"repopulation of the upstream closure; got %v (a dropped repopulation false-HITs here)", after)
	}
}

// evictSingleRefEntry deletes the cache index entry whose transitive-D closure has exactly one
// ref — in the up→down chain that uniquely identifies `up` ({up.py}) vs `down` ({up.py, down.py}).
func evictSingleRefEntry(t *testing.T, ws string) {
	t.Helper()
	indexDir := filepath.Join(ws, ".metis-cache", "index")
	entries, err := filepath.Glob(filepath.Join(indexDir, "*.json"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected 2 index entries, found %v (err %v)", entries, err)
	}
	evicted := 0
	for _, p := range entries {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		e, err := cache.DecodeEntry(b)
		if err != nil {
			t.Fatal(err)
		}
		if len(e.TransitiveD) == 1 {
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			evicted++
		}
	}
	if evicted != 1 {
		t.Fatalf("expected to evict exactly up's single-ref entry, evicted %d", evicted)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
