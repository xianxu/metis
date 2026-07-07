package main

import (
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
		wr, _ := filepath.EvalSymlinks(wantRoot)
		if !strings.HasPrefix(exe, wr) {
			t.Errorf("resolve %q = %s; want under %s", uses, exe, wr)
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
	if !strings.HasPrefix(exe, kb) {
		t.Errorf("shared/thing resolved to %s; want the nearest (kbench) copy under %s, not metis %s", exe, kb, mt)
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
