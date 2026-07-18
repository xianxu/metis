package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/ariadne/pkg/layergraph"
	"github.com/xianxu/metis/internal/repo"
)

// stepPath is the ordered list of directories searched for a step-type executable
// (<layer>/<steptype>). Precedence:
//  1. $METIS_STEP_PATH (OS-list-separated) — explicit override (tests, odd layouts).
//  2. else DISCOVER from the workspace's construct/deps dependency chain: anchor on
//     the experiment file's nearest construct/base.manifest ancestor, walk the layer
//     graph (the SAME topology source weave reads — ariadne/pkg/layergraph, ARCH-DRY),
//     and take each layer's steps/ dir, nearest (leaf) first. No METIS_STEP_PATH, no
//     krun wrapper. NOTE this is leaf-first, which INVERTS the old krun wrapper's
//     base-first order: a workspace step now shadows a base-layer step of the same
//     name (the correct layer-override semantics; harmless today — namespaces are
//     disjoint). METIS_STEP_PATH stays the override.
//  3. else (no construct marker — a bare repo) fall back to <repo.Root(shape dir)>/steps —
//     anchored on the SHAPE's own repo, never cwd (metis#34: the house rule from the #11
//     close-review — anchor on the resolved path; cwd is where the operator happens to
//     stand, not where the work lives).
func stepPath(expPath string) []string {
	if v := os.Getenv("METIS_STEP_PATH"); v != "" {
		return filepath.SplitList(v)
	}
	if abs, err := filepath.Abs(expPath); err == nil {
		if anchor, err := repo.FindUp(filepath.Dir(abs), filepath.Join("construct", "base.manifest")); err == nil {
			order, err := layergraph.Walk(layergraph.OSFS{}, anchor)
			if err != nil {
				// A found anchor with a broken layer graph (e.g. layergraph's loud
				// #155 "present peer missing base.manifest") is an actionable
				// misconfiguration — surface it rather than silently degrade to a
				// misleading "no step-type executable ... on step path [steps]".
				fmt.Fprintln(os.Stderr, "metis: step-layer discovery:", err)
			} else if sp := stepPathFromLayers(order, dirExists); len(sp) > 0 {
				return sp
			}
		}
	}
	if abs, err := filepath.Abs(expPath); err == nil {
		if root, err := repo.Root(filepath.Dir(abs)); err == nil {
			return []string{filepath.Join(root, "steps")}
		}
	}
	return []string{"steps"}
}

// stepPathFromLayers turns layergraph.Walk's base-first ordered layer roots into
// the effective step-path: each layer's steps/ dir that exists, LEAF-FIRST. The
// reversal is the nearest-layer-wins policy — execStep.resolve is first-match-
// wins, so the nearest (most specific) layer must come first. Layers with no
// steps/ dir (e.g. ariadne, the foundation) are dropped. `exists` is injected so
// this stays a pure, disk-free unit (ARCH-PURE).
func stepPathFromLayers(order []string, exists func(string) bool) []string {
	var out []string
	for i := len(order) - 1; i >= 0; i-- { // base-first → leaf-first
		steps := filepath.Join(order[i], "steps")
		if exists(steps) {
			out = append(out, steps)
		}
	}
	return out
}

// dirExists reports whether path is an existing directory (the steps/-dir filter).
func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
