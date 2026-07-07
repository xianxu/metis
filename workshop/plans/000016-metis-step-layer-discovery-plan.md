# metis run: step-layer discovery from the dependency graph — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `metis run <experiment>`, executed in a workspace repo, discover its step-path by walking that repo's `construct/deps` dependency chain — so it works with **no `METIS_STEP_PATH`** and **no `krun` wrapper**.

**Architecture:** Reuse the already-public `github.com/xianxu/ariadne/pkg/layergraph` (the SINGLE source of truth for "what is repo R's layer graph", also consumed by weave) — metis already `require`s + `replace`s ariadne, so it imports with zero new wiring. `stepPath()`'s no-`METIS_STEP_PATH` fallback changes from *"only the current repo's `steps/`"* to *"each layer's `steps/` dir along the dep chain, nearest-first."* `exec.go:resolve` is already first-match-wins, so **nearest-first ordering delivers nearest-layer-wins with no new clash code.** `METIS_STEP_PATH` stays as an explicit override.

**Tech Stack:** Go 1.26; `layergraph.Walk`/`layergraph.OSFS` (ariadne); standard `flag`/`os`/`path/filepath`; shell-script fake steps for hermetic tests (the existing `testdata/steps/test/*` pattern).

---

## Core concepts

### The problem in one line

`layergraph.Walk(OSFS{}, kbenchRoot)` returns the layer roots **base-first**: `[ariadne, metis, kaggle, kbench]`. metis's `execStep.resolve` (`cmd/metis/exec.go:84`) is **first-match-wins**. So the step-path must be assembled **leaf-first** — `[kbench/steps, kaggle/steps, metis/steps]` — for the nearest (most specific) layer to win a name clash. ariadne ships no `steps/` and is dropped.

### The anchor problem

`layergraph.Walk` needs the **workspace repo root** as its starting point, and that root must have `construct/deps` (its upstream edge). metis's existing `repo.Root` keys off **`go.mod`** — but **kbench has no `go.mod`**, so `repo.Root` walks *past* kbench. The correct marker for "a construct layer" is **`construct/base.manifest`** (exactly what `layergraph`'s own ancestor filter uses). So root discovery must walk up to `construct/base.manifest`, not `go.mod`. Anchor on the **experiment file's directory** (an explicit argument to `metis run`), not cwd — more robust than requiring the operator to run from the repo root.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `stepPathFromLayers` | `cmd/metis/steppath.go` | new |
| `repo.FindUp` | `internal/repo/repo.go` | new |
| `repo.Root` | `internal/repo/repo.go` | modified |

- **stepPathFromLayers** — pure: takes `layergraph.Walk`'s base-first ordered roots + an injected `exists func(string) bool`, returns the step-path as each layer's `steps/` dir that exists, **leaf-first** (reversed).
  - **Relationships:** 1:1 with a `metis run` invocation. Sole owner of the base-first→leaf-first reversal + the "keep only layers that ship a `steps/` dir" filter.
  - **DRY rationale:** Isolates the two policy decisions (reverse for nearest-wins; drop no-`steps/` layers like ariadne) into one pure, disk-free, unit-tested function instead of inlining them in the IO seam.
  - **Future extensions:** an `error-on-clash` policy would gate here (detect a `namespace/name` defined in ≥2 layers) — deliberately NOT built (nearest-wins is the chosen policy; see Decisions).

- **repo.FindUp** — pure-ish up-walk: `FindUp(start, marker)` returns the nearest ancestor dir containing the relative `marker` path, else an error.
  - **DRY rationale:** The `repo` package's stated purpose is holding this up-walk **once** ("replacing three near-identical copies", `repo.go` doc). A second marker-walk beside it that didn't reuse it would be the exact duplication `repo` exists to prevent. `Root` becomes `FindUp(start, "go.mod")`; the new construct-anchor is `FindUp(dir, "construct/base.manifest")`.
  - **Future extensions:** any other "walk up to a sentinel file" need routes through here.

- **repo.Root** — *modified*: same public contract (nearest `go.mod` ancestor), now a one-line delegate to `FindUp(start, "go.mod")`. No caller change.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `stepPath` | `cmd/metis/steppath.go` | modified (moved from `main.go`) | env + os FS + `layergraph.Walk` |
| `layergraph.Walk` / `layergraph.OSFS` | `ariadne/pkg/layergraph` | referenced (reused) | `construct/deps` transitive walk |

- **stepPath(expPath)** — the IO seam. Order: (1) `METIS_STEP_PATH` set → `filepath.SplitList` it (override, unchanged); (2) else anchor = `repo.FindUp(dir(abs(expPath)), "construct/base.manifest")`, `order = layergraph.Walk(OSFS{}, anchor)`, `sp = stepPathFromLayers(order, dirExists)` — if non-empty, use it; (3) else the current fallback (`repo.Root(cwd)/steps`, then literal `"steps"`) for a bare repo with no construct marker.
  - **Injected into:** built in `cmdRun` (`main.go`) into `runOpts.stepPath`; consumed by `execStep.resolve`.
  - **Note:** signature changes from `stepPath()` to `stepPath(expPath string)` — the one caller (`main.go:56`) already has `rest[0]` in hand.

- **layergraph.Walk / OSFS** — reused verbatim; metis imports `github.com/xianxu/ariadne/pkg/layergraph` (already in go.mod via `replace => ../ariadne`; metis already imports the sibling `ariadne/pkg/frontmatter`). No ariadne change. `Walk` canonicalizes the root via `EvalSymlinks` internally, so a `/tmp`→`/private/tmp` fixture root resolves correctly.

### Test surface

- `steppath_test.go` (new): pure `stepPathFromLayers` ordering/filter; `repo.FindUp` up-walk (hit + no-marker error); and an **invocation-path** test that builds a 3-layer `construct/deps` fixture tree and calls the **real** `stepPath(expPath)` (the exact function `cmdRun` calls) with `METIS_STEP_PATH` unset — asserting the discovered path resolves a step from **each** layer (via the real `execStep.resolve`), plus a nearest-wins clash case. No uv, no git — hermetic shell-script steps.
- The manual end-to-end (Done-when bullet 1): `metis run` in the **real kbench repo**, no `METIS_STEP_PATH`, no `krun` — the invocation-path proof the lessons demand.

---

## Task 1: `repo.FindUp` — generalize the up-walk (ARCH-DRY)

**Files:**
- Modify: `internal/repo/repo.go`
- Test: `internal/repo/repo_test.go`

- [ ] **Step 1: Write the failing test** (append to `internal/repo/repo_test.go`)

```go
func TestFindUp_FindsMarker(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "construct"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "construct", "base.manifest"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FindUp(sub, filepath.Join("construct", "base.manifest"))
	if err != nil {
		t.Fatalf("FindUp: %v", err)
	}
	// EvalSymlinks both sides: macOS TempDir is /var→/private/var symlinked.
	wantR, _ := filepath.EvalSymlinks(root)
	gotR, _ := filepath.EvalSymlinks(got)
	if gotR != wantR {
		t.Fatalf("FindUp = %s; want %s", gotR, wantR)
	}
}

func TestFindUp_NoMarker(t *testing.T) {
	if _, err := FindUp(t.TempDir(), "nonesuch.marker"); err == nil {
		t.Fatal("FindUp: want an error when the marker is never found")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /Users/xianxu/workspace/metis && go test ./internal/repo/ -run TestFindUp -v`
Expected: FAIL — `undefined: FindUp`.

- [ ] **Step 3: Implement** — in `internal/repo/repo.go`, add `FindUp` and make `Root` delegate:

```go
// FindUp walks up from start to the nearest ancestor directory containing the
// relative marker path (e.g. "go.mod" or "construct/base.manifest"), returning
// that directory. It errors if none is found before the filesystem root. One
// up-walk shared by every "nearest ancestor holding X" lookup (ARCH-DRY).
func FindUp(start, marker string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found above %s", marker, start)
		}
		dir = parent
	}
}

// Root walks up from start to the nearest ancestor directory containing a go.mod,
// returning that directory. It errors if none is found before the filesystem root.
func Root(start string) (string, error) {
	return FindUp(start, "go.mod")
}
```

Delete the old inlined loop body of `Root`. Keep the existing imports (`fmt`, `os`, `path/filepath`).

- [ ] **Step 4: Run to verify pass** — `go test ./internal/repo/ -v` → all PASS (existing `TestRoot_*` still green via the delegate).

- [ ] **Step 5: Commit**

```bash
git add internal/repo/repo.go internal/repo/repo_test.go
git commit -m "#16: repo.FindUp — generalize the ancestor-marker up-walk"
```

---

## Task 2: `stepPathFromLayers` — pure base-first→leaf-first assembly

**Files:**
- Create: `cmd/metis/steppath.go` (move `stepPath` here from `main.go` in Task 3)
- Test: `cmd/metis/steppath_test.go`

- [ ] **Step 1: Write the failing test** (`cmd/metis/steppath_test.go`)

```go
package main

import (
	"path/filepath"
	"reflect"
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
```

- [ ] **Step 2: Run to verify it fails** — `go test ./cmd/metis/ -run TestStepPathFromLayers -v` → FAIL (`undefined: stepPathFromLayers`).

- [ ] **Step 3: Implement** — in `cmd/metis/steppath.go`:

```go
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
```

- [ ] **Step 4: Run to verify pass** — `go test ./cmd/metis/ -run TestStepPathFromLayers -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/metis/steppath.go cmd/metis/steppath_test.go
git commit -m "#16: stepPathFromLayers — pure leaf-first step-path assembly"
```

---

## Task 3: `stepPath(expPath)` — wire discovery through `layergraph.Walk`

**Files:**
- Modify: `cmd/metis/main.go` (remove `stepPath`; call `stepPath(rest[0])`; drop now-unused `repo` import if it becomes unused — it stays used by `stepPath`'s fallback which moves to `steppath.go`, so the import moves too)
- Modify: `cmd/metis/steppath.go` (add `stepPath` + `dirExists`)
- Modify: `go.mod`/`go.sum` if needed (likely no change — ariadne already required)

- [ ] **Step 1: Write the failing invocation-path test** (append to `cmd/metis/steppath_test.go`)

Build a 3-layer `construct/deps` fixture and drive the **real** `stepPath` (the function `cmdRun` calls), then resolve through the **real** `execStep.resolve`. The final import set for `cmd/metis/steppath_test.go` is exactly `io, os, path/filepath, reflect, strings, testing` — no `experiment` import (these tests only call `execStep.resolve(string)`; Go hard-errors on an unused import):

```go
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

func must(t *testing.T, err error) { t.Helper(); if err != nil { t.Fatal(err) } }

// TestStepPath_DiscoversLayersFromDepChain is the metis#16 Done-when in a
// hermetic fixture: metis run in a workspace with a dep chain resolves a step
// from EACH layer with NO METIS_STEP_PATH — via the real stepPath + resolve.
func TestStepPath_DiscoversLayersFromDepChain(t *testing.T) {
	t.Setenv("METIS_STEP_PATH", "") // force discovery, not the override
	ws := t.TempDir()
	mkLayer(t, ws, "ariadne", "", "", "")                 // foundation: manifest, no deps, no steps
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
```

(Add `strings` to the test imports.)

- [ ] **Step 2: Run to verify it fails** — `go test ./cmd/metis/ -run TestStepPath_ -v` → FAIL (`stepPath` still takes no args / still returns only the current repo).

- [ ] **Step 3: Implement** — move `stepPath` from `main.go` into `steppath.go`, rewrite it, add `dirExists`:

```go
// stepPath is the ordered list of directories searched for a step-type executable
// (<layer>/<steptype>). Precedence:
//  1. $METIS_STEP_PATH (colon-separated) — explicit override (tests, odd layouts).
//  2. else DISCOVER from the workspace's construct/deps dependency chain: anchor on
//     the experiment file's nearest construct/base.manifest ancestor, walk the layer
//     graph (the SAME source weave reads — ariadne/pkg/layergraph), and take each
//     layer's steps/ dir, nearest (leaf) first. No METIS_STEP_PATH, no krun wrapper.
//  3. else (no construct marker — a bare repo) fall back to <repo.Root(cwd)>/steps.
func stepPath(expPath string) []string {
	if v := os.Getenv("METIS_STEP_PATH"); v != "" {
		return filepath.SplitList(v)
	}
	if abs, err := filepath.Abs(expPath); err == nil {
		if anchor, err := repo.FindUp(filepath.Dir(abs), filepath.Join("construct", "base.manifest")); err == nil {
			// A Walk error (e.g. layergraph's loud #155 "present peer missing
			// base.manifest") is deliberately swallowed here — we degrade to the
			// bare-repo fallback rather than abort. The operator still sees an
			// actionable "no step-type executable ... on step path [...]" downstream.
			if order, err := layergraph.Walk(layergraph.OSFS{}, anchor); err == nil {
				if sp := stepPathFromLayers(order, dirExists); len(sp) > 0 {
					return sp
				}
			}
		}
	}
	if wd, err := os.Getwd(); err == nil {
		if root, err := repo.Root(wd); err == nil {
			return []string{filepath.Join(root, "steps")}
		}
	}
	return []string{"steps"}
}

// dirExists reports whether path is an existing directory (the steps/-dir filter).
func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
```

Imports for `steppath.go`: `os`, `path/filepath`, `github.com/xianxu/metis/internal/repo`, `github.com/xianxu/ariadne/pkg/layergraph`.

In `main.go`: delete the old `stepPath()` func and change the call site `stepPath: stepPath(),` → `stepPath: stepPath(rest[0]),`. After the move, `main.go` uses only `flag`/`fmt`/`os` — remove **both** `path/filepath` **and** `github.com/xianxu/metis/internal/repo` from its import block (both are now unused there; `go build`/`go vet` in Step 4 confirms).

- [ ] **Step 4: Run to verify pass**

Run: `go test ./cmd/metis/ -run TestStepPath_ -v` → PASS (both discovery + nearest-wins).
Run: `go build ./... && go vet ./...` → clean (confirms `main.go` imports are correct after the move).

- [ ] **Step 5: Full metis suite (no regression)** — `go test ./...` → all PASS. The existing e2e (`TestToyPipeline_EndToEnd`) can't regress: it passes an explicit `stepPath` (`e2e_test.go:44`), so the discovery function never runs in that test.

- [ ] **Step 6: Commit**

```bash
git add cmd/metis/steppath.go cmd/metis/main.go cmd/metis/steppath_test.go go.mod go.sum
git commit -m "#16: metis run discovers step-path from the construct/deps layer graph

No METIS_STEP_PATH, no krun: metis run walks the workspace's dependency
chain (reusing ariadne/pkg/layergraph, the same source weave reads) and
assembles the step-path nearest-first. METIS_STEP_PATH stays an override."
```

---

## Task 4: Manual end-to-end in real kbench + atlas + kbench follow-up

**Files:**
- Modify: `atlas/` (step-layer discovery entry) + `atlas/index.md` link if new file
- Create (kbench, separate issue): a follow-up to collapse `bin/krun`

- [ ] **Step 1: Build metis** — `go build -C /Users/xianxu/workspace/metis -o bin/metis ./cmd/metis`.

- [ ] **Step 2: Real-kbench invocation-path proof (Done-when bullet 1)** — run the **real** titanic-baseline thread in the **real** kbench tree, **hermetically** (fake-kaggle, no network), with `METIS_STEP_PATH` **unset** and **not** via `krun` — a completed run proves all three layers' steps (`titanic/adapt`, `kaggle/download`, `metis/cv-split`, …) resolved *and* ran through the dep-graph discovery. This mirrors `e2e/thread_test.py:52-69`'s env but calls `metis run` directly. **Flags precede the positional** (Go's `flag` stops at the first non-flag; `--dry-run` is sweep-only and does NOT apply to a plain experiment — so a real hermetic run, not a dry one, is the proof). Needs `go` + `uv` on PATH.

```bash
cd /Users/xianxu/workspace/kbench
go build -C ../metis   -o bin/metis            ./cmd/metis
go build -C ../kaggle  -o "$TMPDIR/fake-kaggle" ./cmd/fake-kaggle
env -u METIS_STEP_PATH \
  KAGGLE_CLI="$TMPDIR/fake-kaggle" KAGGLE_FAKE=1 KAGGLE_FAKE_STATE="$TMPDIR/kstate" \
  KAGGLE_FAKE_DATA_DIR=competition/titanic/testdata/raw KAGGLE_FAKE_SCORE_AFTER=1 \
  KAGGLE_SUBMIT_MAX_ATTEMPTS=5 KAGGLE_SUBMIT_DELAY=0 \
  bin/metis run --run run-16verify competition/titanic/pipelines/titanic-baseline.md
```

Expected: exit 0, all seven steps run (no "no step-type executable for uses …" error). Restore the tracked experiment file afterward (metis appends a `Run` line — `git checkout competition/titanic/pipelines/titanic-baseline.md`) and wipe `runs/`/`data/`/`.metis-cache`. Record the exact command + the key output lines in the issue `## Log`. (The automated Task 3 test already proves the mechanism hermetically; this confirms it against the real `construct/deps` + `steps/` tree.)

- [ ] **Step 3: Atlas** — add/adjust an atlas entry: *step-layer discovery = the `construct/deps` dependency layer-walk (`ariadne/pkg/layergraph`, the same topology weave uses); `METIS_STEP_PATH` is an override; nearest layer wins a clash.* Cite the weave/layergraph parallel. Keep `atlas/index.md` linking it.

- [ ] **Step 4: File the kbench follow-up** — `sdlc issue new` in kbench: "collapse `bin/krun` → `metis run` now that metis discovers the step-path." Scope: repoint `e2e/thread_test.py:_krun`, the runbooks/READMEs/pipeline prose (the `krun` callers Part C enumerated), then delete `bin/krun`. **Out of scope for metis#16** (this issue is metis-only; krun collapse is the kbench consequence).

- [ ] **Step 5: Close** — `sdlc close --issue 16 --verified '<the real-kbench command + output + the go test ./... pass>'`. Compute actuals via `sdlc actual` / let close suggest; if fork/interleave-contaminated, `--no-actual` with the reason.

---

## Decisions

- **Nearest-layer-wins (not error-on-clash).** A workspace overriding a base step is the *point* of layering (the leaf is most specific), and it costs zero code (reverse + existing first-match-wins `resolve`). Erroring would fight the resolver and forbid a legitimate override. Today all namespaces are disjoint (`metis`/`kaggle`/`titanic`), so this is a *policy for the future*, not a behavior change now. `stepPathFromLayers` is the single place an error-on-clash policy would later gate.
- **Anchor on `construct/base.manifest`, not `go.mod`.** kbench has no `go.mod`; `base.manifest` is the canonical construct-layer marker (what `layergraph`'s ancestor filter uses). Anchor on the **experiment file's** dir, not cwd — robust to running from anywhere.
- **Reuse `layergraph`, don't re-parse `construct/deps`.** It's public, importable, already a metis dependency, and explicitly designed as the shared topology source (ARCH-DRY). No ariadne change; no second dep list.
- **metis#16 is metis-only.** `krun`'s collapse is a kbench follow-up (Task 4 Step 4) — keeps this issue atomic and single-boundary (plain checkboxes, one `sdlc close`, no `Mx` split).
