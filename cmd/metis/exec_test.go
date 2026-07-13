package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/metis/pkg/experiment"
)

// TestExecStep_SemaphoreSerializesRealSubprocess (metis#31 I5) proves the leaf
// semaphore is actually wired into the PRODUCTION execStep — the fake-exec e2e
// tests bypass execStep, so nothing else catches a forgotten acquire or a
// mis-threaded runOpts.leafSem → execStep.sem. A real "sleeper" step-type logs
// start/end around a sleep to a SHARED file (under the common expDir); with a
// cap-1 sem two concurrent Execute calls must serialize (peak concurrency 1),
// while a nil sem overlaps (peak 2 — proving the test can detect concurrency).
func TestExecStep_SemaphoreSerializesRealSubprocess(t *testing.T) {
	mkStep := func(t *testing.T) (stepPath []string, expDir string) {
		root := t.TempDir()
		bin := filepath.Join(root, "steps", "test")
		if err := os.MkdirAll(bin, 0o755); err != nil {
			t.Fatal(err)
		}
		script := "#!/bin/sh\nlog=\"$METIS_EXP_DIR/concurrency.log\"\necho start >> \"$log\"\nsleep 0.1\necho end >> \"$log\"\n"
		if err := os.WriteFile(filepath.Join(bin, "sleeper"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		expDir = filepath.Join(root, "exp")
		if err := os.MkdirAll(expDir, 0o755); err != nil {
			t.Fatal(err)
		}
		return []string{filepath.Join(root, "steps")}, expDir
	}
	// peak concurrency from the start/end log (POSIX O_APPEND writes are atomic).
	peak := func(logPath string) int {
		b, _ := os.ReadFile(logPath)
		cur, mx := 0, 0
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			switch line {
			case "start":
				cur++
				if cur > mx {
					mx = cur
				}
			case "end":
				cur--
			}
		}
		return mx
	}
	runTwo := func(t *testing.T, sem chan struct{}) int {
		sp, expDir := mkStep(t)
		e := execStep{stepPath: sp, expDir: expDir, out: io.Discard, sem: sem}
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, _ = e.Execute(experiment.Step{ID: fmt.Sprintf("s%d", i), Uses: "test/sleeper"},
					filepath.Join(expDir, fmt.Sprintf("run-%d", i)))
			}(i)
		}
		wg.Wait()
		return peak(filepath.Join(expDir, "concurrency.log"))
	}
	if p := runTwo(t, make(chan struct{}, 1)); p != 1 {
		t.Fatalf("cap-1 semaphore: peak concurrency = %d, want 1 (execStep acquire not wired?)", p)
	}
	if p := runTwo(t, nil); p != 2 {
		t.Fatalf("nil-sem control: peak concurrency = %d, want 2 (test blind to overlap — sleep too short?)", p)
	}
}

// TestExecStep_InjectsEnv asserts the subprocess executor sets the full step
// contract env — including the M3 additions METIS_EXP_DIR (the experiment-dir
// anchor for exp-relative inputs) and METIS_SEED (the experiment's seed) — so a
// step-type can resolve committed inputs and be reproducible without threading
// the seed through every `with`. Uses the test/env-dump fake step (no uv).
func TestExecStep_InjectsEnv(t *testing.T) {
	root := repoRoot(t)
	runDir := t.TempDir()
	e := execStep{
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		expDir:   "/anchor/exp",
		seed:     99,
		out:      io.Discard,
	}

	_, err := e.Execute(experiment.Step{ID: "e", Uses: "test/env-dump"}, runDir)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(runDir, "e", "env.txt"))
	if err != nil {
		t.Fatalf("read env.txt: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		"STEP_ID=e",
		"EXP_DIR=/anchor/exp",
		"SEED=99",
		"RUN_DIR=" + runDir,
		"STEP_DIR=" + filepath.Join(runDir, "e"),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("env.txt missing %q; got:\n%s", want, got)
		}
	}
}

// TestExecStep_ReadRootInjectedOnlyWhenSet asserts the metis#23 confinement env:
// METIS_READ_ROOT is injected iff execStep.readRoot is non-empty, so the flat
// driver:single path (readRoot == "") stays unconfined.
func TestExecStep_ReadRootInjectedOnlyWhenSet(t *testing.T) {
	root := repoRoot(t)
	base := execStep{stepPath: []string{filepath.Join(root, "testdata", "steps")}, out: io.Discard}

	// (a) readRoot set → env carries it.
	set := base
	set.readRoot = "/data/analysis_0"
	runDir := t.TempDir()
	if _, err := set.Execute(experiment.Step{ID: "e", Uses: "test/env-dump"}, runDir); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := readEnvDump(t, runDir); !strings.Contains(got, "READ_ROOT=/data/analysis_0") {
		t.Errorf("readRoot set but env missing it; got:\n%s", got)
	}

	// (b) readRoot empty → var absent (env-dump reports <unset>), so single is unconfined.
	runDir2 := t.TempDir()
	if _, err := base.Execute(experiment.Step{ID: "e", Uses: "test/env-dump"}, runDir2); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := readEnvDump(t, runDir2); !strings.Contains(got, "READ_ROOT=<unset>") {
		t.Errorf("readRoot empty but METIS_READ_ROOT was injected; got:\n%s", got)
	}
}

// TestExecStep_ConfinesRealUvStep_OutOfRootRead is the metis#23 confinement seal proven
// end-to-end through the REAL chain: Go execStep injects METIS_READ_ROOT → a real uv
// metis/cv-split subprocess → metis.io's exp_path assertion. M1 proved exp_path catches an
// out-of-root read (via _run_step, which sets the env directly) and TestExecStep_ReadRoot…
// proved execStep injects the var (via the non-uv env-dump step); THIS test closes the seam
// between them — a real DATA step, launched by execStep, reading outside its analysis root is
// caught. Combined with runOuterFold setting readRoot=analysis_i on the sealed sweep, a leaked
// outer-assessment read in a driver:cv sweep cannot pass silently.
func TestExecStep_ConfinesRealUvStep_OutOfRootRead(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH; skipping the real-step confinement-chain test")
	}
	root := repoRoot(t)
	ws := t.TempDir()
	expDir := filepath.Join(ws, "experiment")
	if err := os.MkdirAll(filepath.Join(expDir, "analysis_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The dataset lives at ws/dataset/toy; a step resolves `../dataset/toy` against
	// METIS_EXP_DIR=ws/experiment → ws/dataset/toy.
	copyDir(t, filepath.Join(root, "testdata", "dataset", "toy"), filepath.Join(ws, "dataset", "toy"))
	split := experiment.Step{ID: "split", Uses: "metis/cv-split",
		With: map[string]any{"dataset": "../dataset/toy", "k": 3, "stratify": true}}
	stepPath := []string{filepath.Join(root, "steps")}

	// (a) readRoot = ws/experiment/analysis_0 EXCLUDES ws/dataset/toy → the sealed sweep's
	// dataset read is outside its analysis root → confinement fires through the real chain.
	confined := execStep{stepPath: stepPath, expDir: expDir, seed: 42,
		readRoot: filepath.Join(expDir, "analysis_0"), out: io.Discard}
	_, err := confined.Execute(split, filepath.Join(ws, "run-a"))
	if err == nil {
		t.Fatal("confinement did NOT fire: a real sealed-sweep read outside its analysis root must be caught")
	}
	if !strings.Contains(err.Error(), "confinement") {
		t.Errorf("expected the metis#23 confinement error, got: %v", err)
	}

	// (b) readRoot = ws/dataset CONTAINS ws/dataset/toy → the same read is within-root and
	// succeeds — the confinement is SCOPED to the root, not a blanket failure.
	within := execStep{stepPath: stepPath, expDir: expDir, seed: 42,
		readRoot: filepath.Join(ws, "dataset"), out: io.Discard}
	if _, err := within.Execute(split, filepath.Join(ws, "run-b")); err != nil {
		t.Fatalf("within-root read must succeed (confinement over-fired): %v", err)
	}
}

func readEnvDump(t *testing.T, runDir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(runDir, "e", "env.txt"))
	if err != nil {
		t.Fatalf("read env.txt: %v", err)
	}
	return string(b)
}

// TestCollectArtifacts_RecursiveExcludesReserved covers the M2-deferred fix: the
// collector now recurses into subdirectories, and excludes with.json/metrics.json
// only at the step-dir TOP level (a nested sub/metrics.json is a real artifact).
func TestCollectArtifacts_RecursiveExcludesReserved(t *testing.T) {
	runDir := t.TempDir()
	stepDir := filepath.Join(runDir, "s")
	if err := os.MkdirAll(filepath.Join(stepDir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writes := map[string]string{
		"with.json":        "{}", // reserved (top level) — excluded
		"metrics.json":     "{}", // reserved (top level) — excluded
		"reads.json":       "{}", // metis#2 sensor sidecar (top level) — excluded
		"top.txt":          "x",  // artifact
		"sub/nested.csv":   "y",  // artifact (recursion)
		"sub/metrics.json": "{}", // artifact (nested — NOT reserved)
	}
	for rel, body := range writes {
		if err := os.WriteFile(filepath.Join(stepDir, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	arts, err := collectArtifacts(stepDir, runDir)
	if err != nil {
		t.Fatalf("collectArtifacts: %v", err)
	}
	want := []string{"s/sub/metrics.json", "s/sub/nested.csv", "s/top.txt"}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("artifacts = %v; want %v", arts, want)
	}
}
