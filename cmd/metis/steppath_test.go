package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// stepPathFromLayers takes layergraph.Walk's base-first roots and must emit each
// layer's steps/ dir LEAF-FIRST (nearest-wins), dropping layers with no steps/.
func TestStepPathFromLayers_LeafFirstDropsNoSteps(t *testing.T) {
	order := []string{"/w/ariadne", "/w/metis", "/w/kaggle", "/w/kbench"} // base-first
	has := map[string]bool{
		"/w/metis/steps":  true,
		"/w/kaggle/steps": true,
		"/w/kbench/steps": true,
		// ariadne: no steps/
	}
	got := stepPathFromLayers(order, func(p string) bool { return has[p] })
	want := []string{
		filepath.FromSlash("/w/kbench/steps"),
		filepath.FromSlash("/w/kaggle/steps"),
		filepath.FromSlash("/w/metis/steps"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stepPathFromLayers = %v; want %v (leaf-first, ariadne dropped)", got, want)
	}
}

// mkLayer writes construct/base.manifest, an optional construct/deps edge, and a
// steps/<ns>/<name> executable for a fixture layer.
func mkLayer(t *testing.T, ws, name, depRel, ns, step string) string {
	t.Helper()
	root := filepath.Join(ws, name)
	must(t, os.MkdirAll(filepath.Join(root, "construct"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "construct", "base.manifest"), []byte("# "+name), 0o644))
	if depRel != "" {
		must(t, os.WriteFile(filepath.Join(root, "construct", "deps"), []byte("substrate "+depRel+"\n"), 0o644))
	}
	if step != "" {
		sd := filepath.Join(root, "steps", ns)
		must(t, os.MkdirAll(sd, 0o755))
		must(t, os.WriteFile(filepath.Join(sd, step), []byte("#!/bin/sh\ntrue\n"), 0o755))
	}
	return root
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// TestStepPath_DiscoversLayersFromDepChain is the metis#16 Done-when in a
// hermetic fixture: metis run in a workspace with a dep chain resolves a step
// from EACH layer with NO METIS_STEP_PATH — via the real stepPath + resolve.
func TestStepPath_DiscoversLayersFromDepChain(t *testing.T) {
	t.Setenv("METIS_STEP_PATH", "") // force discovery, not the override
	ws := t.TempDir()
	mkLayer(t, ws, "ariadne", "", "", "") // foundation: manifest, no deps, no steps
	metis := mkLayer(t, ws, "metis", "../ariadne", "metis", "cv-split")
	kaggle := mkLayer(t, ws, "kaggle", "../metis", "kaggle", "download")
	kbench := mkLayer(t, ws, "kbench", "../kaggle", "titanic", "adapt")

	exp := filepath.Join(kbench, "pipelines", "p.md")
	must(t, os.MkdirAll(filepath.Dir(exp), 0o755))
	must(t, os.WriteFile(exp, []byte("# exp"), 0o644))

	sp := stepPath(exp)

	// Each layer's step resolves via the discovered path (the real resolver).
	e := execStep{stepPath: sp, out: io.Discard}
	for uses, wantRoot := range map[string]string{
		"metis/cv-split":  metis,
		"kaggle/download": kaggle,
		"titanic/adapt":   kbench,
	} {
		exe, err := e.resolve(uses)
		if err != nil {
			t.Fatalf("resolve %q via discovered path %v: %v", uses, sp, err)
		}
		// EvalSymlinks: sp entries are physical-canonicalized by layergraph.Walk.
		// Separator-terminated /steps/ prefix so no layer root can prefix another.
		wr, _ := filepath.EvalSymlinks(wantRoot)
		want := filepath.Join(wr, "steps") + string(os.PathSeparator)
		if !strings.HasPrefix(exe, want) {
			t.Errorf("resolve %q = %s; want under %s", uses, exe, want)
		}
	}
}

// TestStepPath_NearestLayerWins: a step-type defined in BOTH the workspace and a
// base layer resolves to the workspace (nearest) copy.
func TestStepPath_NearestLayerWins(t *testing.T) {
	t.Setenv("METIS_STEP_PATH", "")
	ws := t.TempDir()
	mkLayer(t, ws, "ariadne", "", "", "")
	metis := mkLayer(t, ws, "metis", "../ariadne", "shared", "thing")
	mkLayer(t, ws, "kaggle", "../metis", "", "")
	kbench := mkLayer(t, ws, "kbench", "../kaggle", "shared", "thing") // same shared/thing

	exp := filepath.Join(kbench, "p.md")
	must(t, os.WriteFile(exp, []byte("# exp"), 0o644))

	exe, err := execStep{stepPath: stepPath(exp), out: io.Discard}.resolve("shared/thing")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	kb, _ := filepath.EvalSymlinks(kbench)
	mt, _ := filepath.EvalSymlinks(metis)
	if !strings.HasPrefix(exe, filepath.Join(kb, "steps")+string(os.PathSeparator)) {
		t.Errorf("shared/thing resolved to %s; want the nearest (kbench) copy under %s, not metis %s", exe, kb, mt)
	}
}

// TestStepPath_BrokenGraphDegradesLoudly: when the anchor is found but the layer
// graph is broken (a substrate target present on disk but missing base.manifest —
// layergraph's loud #155 case), stepPath must NOT return a half-discovered path;
// it surfaces the error to stderr and degrades to the fallback.
func TestStepPath_BrokenGraphDegradesLoudly(t *testing.T) {
	t.Setenv("METIS_STEP_PATH", "")
	ws := t.TempDir()
	// kbench: a real layer with steps, depending on ../kaggle …
	kbench := mkLayer(t, ws, "kbench", "../kaggle", "titanic", "adapt")
	// … but kaggle is PRESENT on disk without a construct/base.manifest → not a
	// compilable layer → layergraph.Walk returns the actionable #155 error.
	must(t, os.MkdirAll(filepath.Join(ws, "kaggle", "construct"), 0o755))

	exp := filepath.Join(kbench, "p.md")
	must(t, os.WriteFile(exp, []byte("# exp"), 0o644))

	// Capture stderr to assert the loud note fires.
	r, w, _ := os.Pipe()
	orig := os.Stderr
	os.Stderr = w
	sp := stepPath(exp)
	_ = w.Close()
	os.Stderr = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	kbSteps := filepath.Join(kbench, "steps")
	for _, p := range sp {
		if p == kbSteps {
			t.Errorf("stepPath returned %s from a broken-graph discovery; want degrade to fallback (sp=%v)", kbSteps, sp)
		}
	}
	if !strings.Contains(buf.String(), "step-layer discovery") {
		t.Errorf("expected a loud stderr note on a broken layer graph; got %q", buf.String())
	}
}

// TestStepPath_EnvOverride: a set METIS_STEP_PATH is honored verbatim, bypassing
// dependency-graph discovery (the explicit-override branch).
func TestStepPath_EnvOverride(t *testing.T) {
	t.Setenv("METIS_STEP_PATH", filepath.FromSlash("/a/steps")+string(os.PathListSeparator)+filepath.FromSlash("/b/steps"))
	got := stepPath("/anywhere/exp.md")
	want := []string{filepath.FromSlash("/a/steps"), filepath.FromSlash("/b/steps")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stepPath override = %v; want %v (verbatim SplitList)", got, want)
	}
}

// TestStepPath_BareRepoFallbackAnchorsOnShapeDir (metis#34): with no construct
// workspace, the fallback anchors on the SHAPE's repo — never on cwd. Two bare
// go.mod repos; cwd in B; a shape in A/sub must resolve A/steps.
func TestStepPath_BareRepoFallbackAnchorsOnShapeDir(t *testing.T) {
	a := t.TempDir()
	for _, d := range []string{"steps", "sub"} {
		if err := os.MkdirAll(filepath.Join(a, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(a, "go.mod"), []byte("module a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	shape := filepath.Join(a, "sub", "exp.md")
	if err := os.WriteFile(shape, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := t.TempDir()
	if err := os.WriteFile(filepath.Join(b, "go.mod"), []byte("module b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(b)

	sp := stepPath(shape)
	want, _ := filepath.EvalSymlinks(filepath.Join(a, "steps"))
	if len(sp) != 1 {
		t.Fatalf("stepPath = %v, want exactly the shape repo's steps", sp)
	}
	got, _ := filepath.EvalSymlinks(sp[0])
	if got != want {
		t.Errorf("stepPath anchored at %s, want %s (cwd must not win)", sp[0], want)
	}
}
