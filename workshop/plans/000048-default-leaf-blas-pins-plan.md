# Default Leaf BLAS Pins Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `metis run` pins leaf BLAS to single-thread by default (OMP/OPENBLAS/VECLIB/MKL=1) at both leaf-spawn seams, unless the operator already exported a value — making the safe thing the default thing (metis#48).

**Architecture:** One pure function (`blasPins`) computes the injectable pins from the ambient env (operator-set names excluded — escape hatch by construction); `runExperiment` computes it ONCE per top-level run, announces it loudly, and threads it (like `forkPool`) to the two spawn seams: legacy `execStep` appends the pins to the child env, and `startForkServer` appends them to the server-process env (forked children inherit). Env stays outside cache identity by existing design, so injection perturbs nothing.

**Tech Stack:** Go (cmd/metis), shell-script fake steps (no uv) for seam tests, one skip-guarded real-uv fork-server test.

---

## Core concepts

### Cache-identity non-interaction (the spec's design question — answered)

Run identity has three legs, and env is in none of them:
- `Kpre` (pkg/cache/cache.go:54) hashes `{step_id, uses, with, seed, upstream}` — no env term.
- HIT-validation re-hashes the read-set `D` / `TransitiveD` (file blob hashes) — no env term.
- The code fingerprint (metis#14/#39) is git state (commit + blob hashes) — no env term.

Injecting pins therefore cannot perturb cache keys or fingerprints — exactly as the RUNBOOK's manual `OMP_NUM_THREADS=1 metis run` never did. This is *by design*, not accident: BLAS thread count changes wall-clock, not outputs (same trained model, modulo float nondeterminism that already exists across machines). Documented in `blaspins.go`'s doc comment (the code home of the fact) — no cache change in this issue.

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `blasPins` | `cmd/metis/blaspins.go` | new |

- **`blasPins(environ []string) []string`** — returns the `K=1` pin entries for the four BLAS thread vars NOT already set in `environ`. Deterministic order (alphabetical). Always non-nil (so a computed-empty result is distinguishable from not-yet-computed `nil` in `runOpts`).
  - **Relationships:** consumed by `runExperiment` (computed once per run); the result rides `runOpts.leafPins` into both spawn seams.
  - **DRY rationale:** ONE definition of "the four pins + ambient-wins rule"; both seams and the loud note derive from its output. (ARCH-DRY — the RUNBOOK's hand-maintained env line stops being load-bearing.)
  - **Future extensions:** a fifth env var (e.g. `NUMEXPR_NUM_THREADS`) is one line in `blasPinDefaults`.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runOpts.leafPins` + once-per-run note | `cmd/metis/run.go` | modified | ambient env (`os.Environ()`) + progress writer |
| `execStep.pins` | `cmd/metis/exec.go` | modified | legacy subprocess env |
| `serverPool`/`startForkServer` pins | `cmd/metis/forkexec.go` | modified | fork-server process env |

- **`runOpts.leafPins`** — computed in `runExperiment` beside the fork-pool creation (AFTER the writer wrap, so the note routes through the #38 board — the same temporal-writer invariant #38 pinned). Guarded on `o.exec == nil && !o.dryRun && o.leafPins == nil`: an injected fake exec spawns nothing (no note noise in fake e2es); nested `runOpts` copies inherit the computed value exactly like `forkPool`.
  - **Injected into:** `execStep{pins:}` and `newServerPool(out, pins)`.
- **`execStep.pins`** — appended to the child env after the ambient base (which strips `METIS_READ_ROOT`), before the per-step `METIS_*` vars. No collision by construction: `blasPins` already excluded ambient-set names, and pins are disjoint from `METIS_*`.
- **`startForkServer` env** — `cmd.Env = append(os.Environ(), pins...)`; forked step children inherit the server env (per-request env carries only `METIS_*`, unchanged). The stale comment at forkexec.go:111–112 ("the operator's BLAS pins apply to every child") is REPLACED — it documented the operator doing a default's job.
- **ARCH-PURPOSE note:** the purpose is "bare `metis run` is safe on BOTH executor paths"; covering only the legacy seam and deferring the fork-server (today's default path!) would be under-delivery. Both seams land in this issue, each with a test at its seam.
- **`metis select --promote` is deliberately UNPINNED (explicit decision, plan review finding 1):** select_cmd.go:361-362 and :533-534 build a fresh `runOpts` and call `runResolvedExperiment` directly, never entering `runExperiment` — so promoted-ship runs get no pins. That is the RIGHT behavior, not a gap: promote is a SERIAL single all-data fit (one leaf at a time — no oversubscription), where multi-threaded BLAS is desirable (faster fit). Document with a one-line comment at each select_cmd runOpts construction ("no leafPins: serial single fit — multi-threaded BLAS wanted, no oversubscription (#48)"). The completeness check in Task 4 covers BOTH constructor sites AND direct `runResolvedExperiment` callers, so future paths make this decision consciously.

---

## Chunk 1: all tasks (single-chunk plan)

### Task 1: `blasPins` pure core

**Files:**
- Create: `cmd/metis/blaspins.go`
- Test: `cmd/metis/blaspins_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package main

import (
	"reflect"
	"testing"
)

// TestBlasPins_BareEnv: no ambient thread vars → all four pins injected, sorted.
func TestBlasPins_BareEnv(t *testing.T) {
	got := blasPins([]string{"PATH=/usr/bin", "HOME=/h"})
	want := []string{
		"MKL_NUM_THREADS=1",
		"OMP_NUM_THREADS=1",
		"OPENBLAS_NUM_THREADS=1",
		"VECLIB_MAXIMUM_THREADS=1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("blasPins = %v, want %v", got, want)
	}
}

// TestBlasPins_OperatorValueWins: an ambient-set var is NOT pinned — an explicit
// operator choice always wins (the issue's escape hatch by construction).
func TestBlasPins_OperatorValueWins(t *testing.T) {
	got := blasPins([]string{"OMP_NUM_THREADS=8", "PATH=/usr/bin"})
	for _, kv := range got {
		if kv == "OMP_NUM_THREADS=1" {
			t.Fatalf("ambient OMP_NUM_THREADS=8 must suppress the pin; got %v", got)
		}
	}
	if len(got) != 3 {
		t.Errorf("want 3 remaining pins, got %v", got)
	}
}

// TestBlasPins_AllSetIsEmptyNonNil: fully pinned ambient env → empty but NON-nil
// (runExperiment uses nil as "not yet computed"; empty must not recompute).
func TestBlasPins_AllSetIsEmptyNonNil(t *testing.T) {
	got := blasPins([]string{
		"OMP_NUM_THREADS=4", "OPENBLAS_NUM_THREADS=4",
		"VECLIB_MAXIMUM_THREADS=4", "MKL_NUM_THREADS=4",
	})
	if got == nil || len(got) != 0 {
		t.Errorf("want empty non-nil, got %#v (nil=%v)", got, got == nil)
	}
}

// TestBlasPins_PrefixNotName: OMP_NUM_THREADS_X=9 is a DIFFERENT var — must not
// suppress the OMP_NUM_THREADS pin (name match is exact, up to '=').
func TestBlasPins_PrefixNotName(t *testing.T) {
	got := blasPins([]string{"OMP_NUM_THREADS_X=9"})
	found := false
	for _, kv := range got {
		if kv == "OMP_NUM_THREADS=1" {
			found = true
		}
	}
	if !found {
		t.Errorf("prefix-named var must not suppress the real pin; got %v", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/metis/ -run TestBlasPins -v`
Expected: FAIL — `undefined: blasPins`

- [ ] **Step 3: Minimal implementation**

```go
package main

import "strings"

// blasPinDefaults are the single-thread pins metis injects into LEAF subprocesses by
// default (metis#48): the parallelism budget belongs to the ORCHESTRATOR (the metis#31
// leaf semaphore), not to each leaf's BLAS — NumCPU leaves × multi-threaded BLAS
// oversubscribes ~NumCPU× (observed: load-avg 83 on 12 cores, throughput ≈ 0).
//
// Cache identity: env is deliberately OUTSIDE run identity — Kpre hashes
// {step_id, uses, with, seed, upstream} (pkg/cache), HIT-validation re-hashes the
// read-set D (file blob hashes), and the code fingerprint is git state. Injecting
// pins perturbs neither cache keys nor fingerprints — exactly as the RUNBOOK's
// manual `OMP_NUM_THREADS=1 metis run` never did.
var blasPinDefaults = []string{
	"MKL_NUM_THREADS=1",
	"OMP_NUM_THREADS=1",
	"OPENBLAS_NUM_THREADS=1",
	"VECLIB_MAXIMUM_THREADS=1",
}

// blasPins returns the defaults NOT already set in environ — an explicit operator
// value always wins (escape hatch by construction: `export OMP_NUM_THREADS=8`
// passes through untouched). Pure. Always non-nil: an all-suppressed result is
// empty, distinguishable from runOpts' nil "not yet computed" sentinel.
func blasPins(environ []string) []string {
	pins := make([]string, 0, len(blasPinDefaults))
	for _, def := range blasPinDefaults {
		name := def[:strings.IndexByte(def, '=')]
		if !envHasName(environ, name) {
			pins = append(pins, def)
		}
	}
	return pins
}

// envHasName reports whether environ sets exactly `name` (match up to '=').
func envHasName(environ []string, name string) bool {
	for _, kv := range environ {
		if strings.HasPrefix(kv, name) && len(kv) > len(name) && kv[len(name)] == '=' {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./cmd/metis/ -run TestBlasPins -v`
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add cmd/metis/blaspins.go cmd/metis/blaspins_test.go
git commit -m "#48: blasPins pure core — default pins minus operator-set names"
```

### Task 2: legacy spawn seam (`execStep.pins`)

**Files:**
- Modify: `cmd/metis/exec.go` (struct field ~line 40; env build ~line 122–131)
- Modify: `testdata/steps/test/env-dump` (dumps ONLY the six `METIS_*` lines today — must also dump the four `*_NUM_THREADS` vars, else the seam test can NEVER pass and the red-green sequence breaks with a wrong-reason failure)
- Test: `cmd/metis/exec_test.go` (follow `TestExecStep_InjectsEnv`)

- [ ] **Step 0: Extend the env-dump fixture** — append the four thread vars to its dump (e.g. `env | grep -E '^(METIS_|OMP_NUM|OPENBLAS_NUM|VECLIB_MAX|MKL_NUM)' > env.txt` or equivalent per its current script style). Safe for existing consumers: exec_test.go:106-116, :133, :142 all assert via `strings.Contains`, so added lines can't break them.

- [ ] **Step 1: Write the failing test**

```go
// TestExecStep_InjectsBlasPins (metis#48): the pins field reaches the child env at the
// legacy spawn seam. The ambient-wins RULE lives in blasPins (unit-tested); this pins
// the PLUMBING — whatever pins the wiring computed are in the subprocess env.
func TestExecStep_InjectsBlasPins(t *testing.T) {
	root := repoRoot(t)
	runDir := t.TempDir()
	e := execStep{
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		out:      io.Discard,
		pins:     []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"},
	}
	if _, err := e.Execute(experiment.Step{ID: "e", Uses: "test/env-dump"}, runDir); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(runDir, "e", "env.txt"))
	if err != nil {
		t.Fatalf("read env.txt: %v", err)
	}
	for _, want := range []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("child env missing %q; got:\n%s", want, b)
		}
	}
}
```

(Check `testdata/steps/test/env-dump` dumps the full env — if it greps METIS_* only, extend the fixture to dump both.)

- [ ] **Step 2: Run to verify failure** — `go test ./cmd/metis/ -run TestExecStep_InjectsBlasPins -v` → FAIL (`unknown field pins`)

- [ ] **Step 3: Implement** — add to `execStep`:

```go
	pins []string // metis#48: leaf BLAS pins (computed once per run by runExperiment;
	//               ambient-set names already excluded) — appended to the child env
```

and in `Execute`, after the `METIS_READ_ROOT` strip loop, before the per-step env append:

```go
	base = append(base, e.pins...) // metis#48: default leaf BLAS pins (operator values already won in blasPins)
```

- [ ] **Step 4: Run to verify pass** — plus the whole file: `go test ./cmd/metis/ -run TestExecStep -v`

- [ ] **Step 5: Commit** — `#48: legacy exec seam — pins reach the child env`

### Task 3: fork-server spawn seam

**Files:**
- Modify: `cmd/metis/forkexec.go` (`newServerPool` ~line 278, `serverPool` struct ~line 269, `startForkServer` ~line 110–120, the stale comment at 111–112, the pool→start call site)
- Test: `cmd/metis/forkexec_test.go` (skip-guarded real-uv test, `toyenv` module pattern from `TestForkServerPerf_LooseBound`)
- Modify (call sites): `newServerPool(...)` callers in forkexec_test.go (~lines 148, 178) pass `nil` pins

- [ ] **Step 1: Write the failing test**

```go
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
			"json.dump({\"omp\": os.environ.get(\"OMP_NUM_THREADS\", \"\")}, open(\"env.json\", \"w\"))\n"+
			"json.dump({\"ok\": 1}, open(\"metrics.json\", \"w\"))\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", mods)
	t.Setenv("METIS_FORKSERVER_PRELOAD", "") // fast start — skip the numpy/pandas preload (sibling tests' pattern, forkexec_test.go:104)
	// exactness: ambient OMP_NUM_THREADS would DUPLICATE the appended pin (CPython os.environ
	// is last-wins so the assertion would still green, but keep the env unambiguous):
	t.Setenv("OMP_NUM_THREADS", "sentinel") // registers restore...
	os.Unsetenv("OMP_NUM_THREADS")          // ...then genuinely absent for the spawn
	var out strings.Builder
	pool := newServerPool(&syncWriter{w: &out}, []string{"OMP_NUM_THREADS=1"})
	defer pool.shutdown()
	stepDir := t.TempDir()
	resp, ok, err := pool.execute(wrapperSpec{root: root, module: "toyenv"}, stepDir, nil)
	if err != nil || !ok || resp.Exit != 0 {
		t.Fatalf("fork exec: ok=%v err=%v resp=%+v\n%s", ok, err, resp, out.String())
	}
	b, err := os.ReadFile(filepath.Join(stepDir, "env.json"))
	if err != nil {
		t.Fatalf("read env.json: %v", err)
	}
	if !strings.Contains(string(b), `"omp": "1"`) {
		t.Errorf("forked child must inherit the server's pin; env.json = %s", b)
	}
}
```

(Adapt `wrapperSpec`/root plumbing to how `TestForkServerPerf_LooseBound` drives a real module — same harness, one new module that reports its env.)

- [ ] **Step 2: Run to verify failure** — `go test ./cmd/metis/ -run TestForkServer_ChildInheritsBlasPins -v` → FAIL (`newServerPool` arity)

- [ ] **Step 3: Implement**
  - `serverPool` gains `pins []string`; `newServerPool(out io.Writer, pins []string) *serverPool` stores it; the pool's `startForkServer(root)` call site passes `p.pins`.
  - `startForkServer(root string, pins []string)`: after `cmd.Dir = root`:

```go
	// metis#48: the server env = ambient + default single-thread BLAS pins (names the
	// operator exported are already excluded by blasPins — an explicit choice wins).
	// Forked step children inherit this env; per-step METIS_* travel in requests.
	cmd.Env = append(os.Environ(), pins...)
```

  - REPLACE the stale doc comment ("The server inherits the ambient env (the operator's BLAS pins apply to every child)") with the new contract above.
  - Fix in-package callers: tests pass `nil` (or explicit pins where asserted).

- [ ] **Step 4: Run to verify pass** — `go test ./cmd/metis/ -run 'TestForkServer' -v` (uv present locally)

- [ ] **Step 5: Commit** — `#48: fork-server seam — pins on the server env, children inherit`

### Task 4: once-per-run wiring + loud note (runExperiment e2e)

**Files:**
- Modify: `cmd/metis/run.go` (`runOpts` field ~line 84; wiring in `runExperiment` immediately BEFORE the forkPool block ~line 151; `execStep{...}` construction line ~205)
- Test: `cmd/metis/run_test.go` (or a new `blaspins_e2e_test.go`) — full-chain, real shell step, no uv

- [ ] **Step 1: Write the failing test**

```go
// TestRun_BlasPinsDefaultAndNote (metis#48 e2e): a bare run (no ambient pins) announces
// ONE loud note and the leaf subprocess sees all four pins; an operator-exported value
// passes through untouched and drops out of the note. Drives runExperiment (the real
// wiring: blasPins → runOpts.leafPins → execStep), real shell step, no uv.
func TestRun_BlasPinsDefaultAndNote(t *testing.T) {
	// ambient: exactly ONE operator choice set; the other three genuinely absent.
	// t.Setenv registers the restore; Unsetenv then makes absence real (an operator
	// shell following the old RUNBOOK exports all four — CI must not inherit that).
	for _, k := range []string{"OPENBLAS_NUM_THREADS", "VECLIB_MAXIMUM_THREADS", "MKL_NUM_THREADS"} {
		t.Setenv(k, "sentinel")
		os.Unsetenv(k)
	}
	t.Setenv("OMP_NUM_THREADS", "7")
	root := t.TempDir()
	stepDir := filepath.Join(root, "steps", "test")
	if err := os.MkdirAll(stepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nenv > env.txt\necho '{\"ok\": 1}' > metrics.json\n"
	if err := os.WriteFile(filepath.Join(stepDir, "envstep"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	expPath := filepath.Join(root, "exp.md")
	// plain-experiment fixture — steps live in the YAML FRONTMATTER (the
	// testdata/experiment/run-echo.md convention; there is no ```steps fence parser)
	exp := "---\ntype: experiment\nid: pins-e2e\nseed: 1\nstatus: active\nsteps:\n  - id: e\n    uses: test/envstep\n---\n# pins e2e\n"
	if err := os.WriteFile(expPath, []byte(exp), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	_, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "r1",
		stepPath: []string{filepath.Join(root, "steps")},
		out:      &out,
	})
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out.String())
	}
	// (a) ONE note, naming only the three injected pins (OMP dropped — operator won)
	if n := strings.Count(out.String(), "metis: leaf BLAS pinned"); n != 1 {
		t.Fatalf("want exactly 1 pin note, got %d:\n%s", n, out.String())
	}
	if strings.Contains(out.String(), "OMP_NUM_THREADS") {
		t.Errorf("note must not claim the operator-set var:\n%s", out.String())
	}
	// (b) child env: three pins at 1, operator's OMP at 7
	b, err := os.ReadFile(filepath.Join(root, "runs", "r1", "e", "env.txt"))
	if err != nil {
		t.Fatal(err)
	}
	envTxt := string(b)
	for _, want := range []string{"OPENBLAS_NUM_THREADS=1", "VECLIB_MAXIMUM_THREADS=1", "MKL_NUM_THREADS=1", "OMP_NUM_THREADS=7"} {
		if !strings.Contains(envTxt, want) {
			t.Errorf("child env missing %q", want)
		}
	}
}
```

(The test must drive `runExperiment`, not `execStep`, so the nil-guard/threading/note wiring is all on the hook. Verify the runs-dir location against `run_test.go` conventions. **Sweep-path once-ness is by construction, verified at plan review:** `runExperiment` is entered exactly once per invocation — sole caller main.go:58 — and all nested sweep/nested-CV spawns are `ss.o` struct copies carrying the computed `leafPins` (sweep.go:463, :550-553, :608-610), so the single-run count==1 assertion here plus that structure covers the sweep; a 2-point real-step sweep asserting count==1 is optional belt-and-braces. Note also: this new note line will appear in OTHER real-subprocess e2e outputs — existing assertions are Contains-based and survive; the full-suite run in Step 4 catches surprises.)

- [ ] **Step 2: Run to verify failure** — FAIL (`unknown field leafPins` / no note)

- [ ] **Step 3: Implement**
  - `runOpts` gains:

```go
	leafPins []string // metis#48: leaf BLAS pins, computed ONCE per top-level run in
	//                   runExperiment (nil = not yet; non-nil rides nested copies like forkPool)
```

  - In `runExperiment`, immediately before the forkPool block (so the pool spawn gets pins; after the writer wrap, so the note hits the board):

```go
	// metis#48: default leaf BLAS pins — computed ONCE from the ambient env (an exported
	// operator value wins by exclusion), announced loudly, injected at both spawn seams.
	// Fake-exec runs spawn nothing (no pins, no note); dry-run lists configs (same).
	if o.exec == nil && !o.dryRun && o.leafPins == nil {
		o.leafPins = blasPins(os.Environ())
		if len(o.leafPins) > 0 {
			fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
				strings.Join(o.leafPins, " "))
		}
	}
```

  - `newServerPool(out)` call → `newServerPool(out, o.leafPins)`.
  - `execStep{...}` at line ~205 gains `pins: o.leafPins`.
  - Completeness check — a constructor-grep alone is NOT a coverage proof (plan review finding 1): grep BOTH `execStep{\|newServerPool(` AND direct callers of `runResolvedExperiment`/`runShapeSweep` in non-test code. Known result: run.go:205 (threaded here) + select_cmd.go:361-362/:533-534 (deliberately unpinned — serial single fit; add the one-line decision comment per Core concepts). Any OTHER hit = an unpinned spawn path to resolve consciously (ARCH-PURPOSE).
  - `newServerPool` arity change touches in-package callers at forkexec_test.go:106, :148, :178, :233, :282 and board_test.go:307 (compiler-caught; tests pass `nil` unless asserting pins).

- [ ] **Step 4: Run to verify pass** — `go test ./cmd/metis/ -run TestRun_BlasPins -v`, then the full suite `go test ./... -race`

- [ ] **Step 5: Commit** — `#48: once-per-run pin wiring + loud note`

### Task 5: docs — RUNBOOK + atlas

**Files:**
- Modify: `kbench/competition/titanic/pipelines/RUNBOOK-sweep.md` (§1, lines ~37–45): the pinned invocation drops the four `*_NUM_THREADS=1` prefixes; reword to "metis pins leaf BLAS single-thread by default (metis#48) — export a value yourself to override; keep `--parallel` as the one knob."
- Modify: `metis/atlas/` — the executor/runner page (find via `grep -rn "fork-server\|execStep" atlas/`): one paragraph on default pins + the env-outside-cache-identity fact.

- [ ] **Step 1: RUNBOOK edit** (kbench repo — separate commit there; cite metis#48). Side-note from review: the current RUNBOOK invocation omits `MKL_NUM_THREADS=1` — the default fixes that inconsistency for free; say so in the commit body.
- [ ] **Step 2: `--parallel` flag help text** — main.go:48 says "pin OMP_NUM_THREADS=1 or set n below NumCPU": now contradicts the default; reword ("leaf BLAS is pinned single-thread by default (#48); n is the one knob").
- [ ] **Step 3: atlas edit** + `git add atlas/ && git commit -m "#48: atlas — default leaf BLAS pins"` (metis)
- [ ] **Step 4: grep-sweep for stale pin advice — Go sources INCLUDED, not just `*.md`** (operator guidance lives in `--help` strings and comments): `grep -rn "NUM_THREADS" /Users/xianxu/workspace/metis /Users/xianxu/workspace/kbench --include="*.md" --include="*.go" --include="*.py"` and reconcile every hit that tells the operator to hand-pin (ARCH-PURPOSE shadow-sweep: the RUNBOOK line was the hand-maintained consumer; docs must now DESCRIBE the default, not restate the mechanism).

### Task 6: real bare-sweep smoke (close evidence)

- [ ] **Step 1:** From kbench: `metis run competition/titanic/pipelines/titanic-sweep.md --sample 1` **bare** (no env pins). Capture: the one-line note, the board's rate line after warm-up (sane folds/min, no thrash ETA), load-avg sanity (`uptime`).
- [ ] **Step 2:** Record the evidence in the issue `## Log` — this is the Done-when's "rate line evidence in the close."

---

**Verification gate (before close):** full `go test ./... -race` green in metis; kbench e2e suite green (`uv` present); smoke evidence logged. Close via `sdlc close --issue 48 --verified '<evidence>'` (measured actuals; atlas touched in Task 5 so the atlas gate is genuinely satisfied, no `--no-atlas`).
