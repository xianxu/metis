package main

// metis#44 tests: wrapper-convention parsing (pure), the Go↔forkserver protocol against the
// REAL `uv run … python -m metis.forkserver` (hermetic — the metis venv is synced), and the
// loud legacy fallback for non-conforming wrappers.

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

const conformingWrapper = `#!/usr/bin/env bash
# test wrapper — matches the two-repo convention exactly.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
exec uv run --project "$ROOT" python -m metis.trace toy.mod_ok
`

// writeWrapper lays out <root>/steps/<layer>/<name> (+ optional pyproject.toml) and returns
// the wrapper path — the shape parseWrapper derives the root from.
func writeWrapper(t *testing.T, root, body string, withPyproject bool) string {
	t.Helper()
	dir := filepath.Join(root, "steps", "test")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "toystep")
	if err := os.WriteFile(exe, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if withPyproject {
		if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return exe
}

func TestParseWrapper_ConventionAndFallbacks(t *testing.T) {
	t.Run("conforming wrapper parses root+module", func(t *testing.T) {
		root := t.TempDir()
		exe := writeWrapper(t, root, conformingWrapper, true)
		spec, ok := parseWrapper(exe)
		if !ok {
			t.Fatal("conforming wrapper must be forkable")
		}
		wantRoot, _ := filepath.Abs(root)
		if spec.root != wantRoot || spec.module != "toy.mod_ok" {
			t.Errorf("got %+v; want root=%s module=toy.mod_ok", spec, wantRoot)
		}
	})
	t.Run("non-standard exec line is not forkable", func(t *testing.T) {
		body := strings.Replace(conformingWrapper, "python -m metis.trace toy.mod_ok",
			"python custom_entry.py", 1)
		exe := writeWrapper(t, t.TempDir(), body, true)
		if _, ok := parseWrapper(exe); ok {
			t.Error("a wrapper with a custom entry must fall back to legacy exec")
		}
	})
	t.Run("non-standard ROOT resolution is not forkable", func(t *testing.T) {
		body := strings.Replace(conformingWrapper,
			`ROOT="$(cd "$(dirname "$0")/../.." && pwd)"`, `ROOT="/opt/elsewhere"`, 1)
		exe := writeWrapper(t, t.TempDir(), body, true)
		if _, ok := parseWrapper(exe); ok {
			t.Error("a wrapper resolving ROOT differently must fall back (our root derivation would be wrong)")
		}
	})
	t.Run("root without pyproject.toml is not forkable", func(t *testing.T) {
		exe := writeWrapper(t, t.TempDir(), conformingWrapper, false)
		if _, ok := parseWrapper(exe); ok {
			t.Error("a derived root that isn't a uv project must fall back")
		}
	})
}

// TestForkServerPool_RealServerRoundTrip drives the REAL fork-server in the metis venv end
// to end: env authority, metrics/artifact collection through execStep.collectResult's
// consumers, and a step failure surfacing exit+traceback. PYTHONPATH carries the toy module.
func TestForkServerPool_RealServerRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH")
	}
	root := repoRoot(t)
	mods := t.TempDir()
	if err := os.WriteFile(filepath.Join(mods, "toyfork.py"), []byte(
		"import json, os\n"+
			"json.dump({\"fold_score\": 0.9}, open(\"metrics.json\", \"w\"))\n"+
			"open(\"out.txt\", \"w\").write(os.environ.get(\"METIS_STEP_ID\", \"\"))\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mods, "toyboom.py"),
		[]byte("raise RuntimeError('fork-boom')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", mods)
	t.Setenv("METIS_FORKSERVER_PRELOAD", "") // fast start; preload is a python-side concern

	pool := newServerPool(io.Discard, nil)
	defer pool.shutdown()

	stepDir := filepath.Join(t.TempDir(), "s1")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resp, ok, ferr := pool.execute(wrapperSpec{root: root, module: "toyfork"}, stepDir,
		map[string]string{"METIS_STEP_DIR": stepDir, "METIS_STEP_ID": "s1"})
	if !ok || ferr != nil || resp.Exit != 0 {
		t.Fatalf("round trip failed: ok=%v err=%v resp=%+v", ok, ferr, resp)
	}
	b, err := os.ReadFile(filepath.Join(stepDir, "out.txt"))
	if err != nil || string(b) != "s1" {
		t.Errorf("request env not applied in child: %q, %v", b, err)
	}
	m, err := readMetrics(filepath.Join(stepDir, "metrics.json"))
	if err != nil || m["fold_score"] != 0.9 {
		t.Errorf("metrics not written by forked step: %v, %v", m, err)
	}

	boomDir := filepath.Join(t.TempDir(), "s2")
	if err := os.MkdirAll(boomDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resp, ok, ferr = pool.execute(wrapperSpec{root: root, module: "toyboom"}, boomDir,
		map[string]string{"METIS_STEP_DIR": boomDir})
	if !ok || ferr != nil {
		t.Fatal("a step FAILURE is a real outcome, not server-unavailable")
	}
	if resp.Exit == 0 || !strings.Contains(resp.Output, "fork-boom") {
		t.Errorf("failure must surface exit+traceback, got %+v", resp)
	}
}

// TestForkServerPool_BrokenRootFallsBackOnce: a root whose server can't start (no uv
// project) reports ok=false — and only notices once.
func TestForkServerPool_BrokenRootFallsBack(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH")
	}
	var out strings.Builder
	pool := newServerPool(&syncWriter{w: &out}, nil)
	defer pool.shutdown()
	bogus := t.TempDir() // no pyproject/venv — uv run will fail (or hang-free error)
	for i := 0; i < 3; i++ {
		_, ok, ferr := pool.execute(wrapperSpec{root: bogus, module: "x"}, t.TempDir(), nil)
		if ok || ferr != nil {
			t.Fatal("bogus root must be unavailable pre-dispatch (fallback-safe, no step error)")
		}
	}
	if n := strings.Count(out.String(), "legacy exec"); n != 1 {
		t.Errorf("want exactly ONE loud fallback notice, got %d:\n%s", n, out.String())
	}
}

// TestExecute_NonConformingWrapperUsesLegacyLoudly: routing at the execStep level — a
// non-standard wrapper still RUNS (legacy subprocess) and the pool notices once.
func TestExecute_NonConformingWrapperUsesLegacyLoudly(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "steps", "test")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// a plain shell step — valid executable, not the uv/metis.trace convention
	exe := filepath.Join(dir, "shellstep")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\necho '{\"ok\": 1}' > metrics.json\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	var out strings.Builder
	e := execStep{stepPath: []string{filepath.Join(root, "steps")}, expDir: ws,
		out: &out, pool: newServerPool(&out, nil)}
	defer e.pool.shutdown()
	runDir := filepath.Join(ws, "runs", "r1")
	res, err := e.Execute(experiment.Step{ID: "sh", Uses: "test/shellstep"}, runDir)
	if err != nil {
		t.Fatalf("legacy fallback must still run the step: %v", err)
	}
	if res.Metrics["ok"] != 1 {
		t.Errorf("legacy path result wrong: %+v", res)
	}
	if !strings.Contains(out.String(), "doesn't match") {
		t.Errorf("fallback must be LOUD, got:\n%s", out.String())
	}
	// second execution: no second notice
	if _, err := e.Execute(experiment.Step{ID: "sh2", Uses: "test/shellstep"}, runDir); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out.String(), "doesn't match"); n != 1 {
		t.Errorf("want ONE notice per uses-type, got %d", n)
	}
}

// TestForkServerPerf_LooseBound (metis#44 acceptance): N leaves that each import pandas —
// legacy pays uv+interpreter+import per spawn; the fork-server pays preload once and forks.
// Loose bound (faster, not a ratio) to stay robust on loaded CI boxes; the real-sweep
// wall-clock comparison lives in the issue Log.
func TestForkServerPerf_LooseBound(t *testing.T) {
	if testing.Short() {
		t.Skip("perf bound — skipped in -short")
	}
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH")
	}
	root := repoRoot(t)
	mods := t.TempDir()
	if err := os.WriteFile(filepath.Join(mods, "toyheavy.py"), []byte(
		"import pandas\nimport json\njson.dump({\"ok\": 1}, open(\"metrics.json\", \"w\"))\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", mods)
	const n = 4

	start := time.Now()
	for i := 0; i < n; i++ {
		dir := t.TempDir()
		cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.trace", "toyheavy")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "METIS_STEP_DIR="+dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("legacy spawn: %v\n%s", err, out)
		}
	}
	legacy := time.Since(start)

	start = time.Now()
	pool := newServerPool(io.Discard, nil)
	defer pool.shutdown()
	forkOne := func() {
		dir := t.TempDir()
		resp, ok, ferr := pool.execute(wrapperSpec{root: root, module: "toyheavy"}, dir,
			map[string]string{"METIS_STEP_DIR": dir})
		if !ok || ferr != nil || resp.Exit != 0 {
			t.Fatalf("fork run failed: ok=%v err=%v resp=%+v", ok, ferr, resp)
		}
	}
	for i := 0; i < n; i++ {
		forkOne()
	}
	forked := time.Since(start)

	// The Done-when bound: >=2x on the WARM MARGINAL (the per-leaf cost a sweep actually
	// pays — the server is already up after the first leaf). Total-time stays a loose >1x
	// (server start + preload amortize over ~5k leaves in a real sweep, not over n=4).
	start = time.Now()
	for i := 0; i < n; i++ {
		forkOne()
	}
	warmMarginal := time.Since(start)
	t.Logf("legacy %v vs forkserver %v cold / %v warm-marginal (n=%d)", legacy, forked, warmMarginal, n)
	if forked >= legacy {
		t.Errorf("forkserver cold (%v) not faster than legacy (%v) at n=%d", forked, legacy, n)
	}
	if 2*warmMarginal >= legacy {
		t.Errorf("forkserver warm marginal (%v) not >=2x faster than legacy (%v) at n=%d", warmMarginal, legacy, n)
	}
}

// TestForkServerPool_MidFlightDeathErrorsTheStep (close-review I1): SIGKILL the server with
// a request in flight — the forked child may still be running, so pool.execute must return a
// STEP ERROR (dispatched-and-lost), never ok=false (which would trigger a legacy re-run into
// the same stepDir).
func TestForkServerPool_MidFlightDeathErrorsTheStep(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH")
	}
	root := repoRoot(t)
	mods := t.TempDir()
	if err := os.WriteFile(filepath.Join(mods, "toyslow.py"),
		[]byte("import time\ntime.sleep(5)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", mods)
	t.Setenv("METIS_FORKSERVER_PRELOAD", "")

	pool := newServerPool(io.Discard, nil)
	defer pool.shutdown()
	type result struct {
		ok  bool
		err error
	}
	done := make(chan result, 1)
	go func() {
		_, ok, ferr := pool.execute(wrapperSpec{root: root, module: "toyslow"}, t.TempDir(),
			map[string]string{})
		done <- result{ok, ferr}
	}()
	// Wait until the server exists and the request had time to dispatch, then kill it.
	var srv *forkServer
	for i := 0; i < 200; i++ {
		pool.mu.Lock()
		srv = pool.servers[root]
		pool.mu.Unlock()
		if srv != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if srv == nil {
		t.Fatal("server never started")
	}
	<-srv.ready
	time.Sleep(300 * time.Millisecond) // let the request dispatch + fork
	// Group-kill: `uv run` spawns python as a child — killing only the uv parent would
	// leave the real server (and its forked step child) alive and the request would
	// complete normally after the sleep.
	if err := syscall.Kill(-srv.cmd.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	res := <-done
	if res.err == nil {
		t.Fatalf("mid-flight death must be a STEP ERROR (dispatched-and-lost), got ok=%v err=nil", res.ok)
	}
}

// TestForkServer_ChildInheritsBlasPins (metis#48): the pool's pins land on the SERVER
// process env at spawn, and a forked step child inherits them (per-request env carries
// only METIS_*). Real uv + real fork-server — the seam the operator's sweeps ride.
func TestForkServer_ChildInheritsBlasPins(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH")
	}
	root := repoRoot(t)
	mods := t.TempDir()
	if err := os.WriteFile(filepath.Join(mods, "toyenv.py"), []byte(
		"import json, os\n"+
			"json.dump({\"omp\": os.environ.get(\"OMP_NUM_THREADS\", \"\")}, open(\"envcap.json\", \"w\"))\n"+
			"json.dump({\"ok\": 1}, open(\"metrics.json\", \"w\"))\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", mods)
	t.Setenv("METIS_FORKSERVER_PRELOAD", "") // fast start; preload is a python-side concern
	// exactness: an ambient OMP_NUM_THREADS would DUPLICATE the appended pin (CPython's
	// os.environ is last-wins so the assertion would still green, but keep it unambiguous):
	t.Setenv("OMP_NUM_THREADS", "sentinel") // registers restore...
	os.Unsetenv("OMP_NUM_THREADS")          // ...then genuinely absent for the spawn

	pool := newServerPool(io.Discard, []string{"OMP_NUM_THREADS=1"})
	defer pool.shutdown()

	stepDir := filepath.Join(t.TempDir(), "s1")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resp, ok, ferr := pool.execute(wrapperSpec{root: root, module: "toyenv"}, stepDir,
		map[string]string{"METIS_STEP_DIR": stepDir})
	if !ok || ferr != nil || resp.Exit != 0 {
		t.Fatalf("fork exec failed: ok=%v err=%v resp=%+v", ok, ferr, resp)
	}
	b, err := os.ReadFile(filepath.Join(stepDir, "envcap.json"))
	if err != nil {
		t.Fatalf("read envcap.json: %v", err)
	}
	if !strings.Contains(string(b), `"omp": "1"`) {
		t.Errorf("forked child must inherit the server's pin; envcap.json = %s", b)
	}
}
