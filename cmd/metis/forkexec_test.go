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
	"testing"

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

	pool := newServerPool(io.Discard)
	defer pool.shutdown()

	stepDir := filepath.Join(t.TempDir(), "s1")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resp, ok := pool.execute(wrapperSpec{root: root, module: "toyfork"}, stepDir,
		map[string]string{"METIS_STEP_DIR": stepDir, "METIS_STEP_ID": "s1"})
	if !ok || resp.Exit != 0 {
		t.Fatalf("round trip failed: ok=%v resp=%+v", ok, resp)
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
	resp, ok = pool.execute(wrapperSpec{root: root, module: "toyboom"}, boomDir,
		map[string]string{"METIS_STEP_DIR": boomDir})
	if !ok {
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
	pool := newServerPool(&syncWriter{w: &out})
	defer pool.shutdown()
	bogus := t.TempDir() // no pyproject/venv — uv run will fail (or hang-free error)
	for i := 0; i < 3; i++ {
		if _, ok := pool.execute(wrapperSpec{root: bogus, module: "x"}, t.TempDir(), nil); ok {
			t.Fatal("bogus root must be unavailable")
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
		out: &out, pool: newServerPool(&out)}
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
