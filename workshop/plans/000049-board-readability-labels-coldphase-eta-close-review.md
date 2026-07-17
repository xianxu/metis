# Boundary Review — 000049-board-readability-labels-coldphase-eta#49 (whole-issue close)

| field | value |
|-------|-------|
| issue | 49 — board readability — label semantics, cold-phase "no progress" confusion, jumpy leaves, wild early ETA |
| repo | 000049-board-readability-labels-coldphase-eta |
| issue file | workshop/issues/000049-board-readability-labels-coldphase-eta.md |
| boundary | whole-issue close |
| milestone | — |
| window | 72f2d895fc676a474304fb4cde8f4606e9c5f48c^..HEAD |
| command | sdlc close --issue 49 |
| reviewer | codex |
| timestamp | 2026-07-17T10:12:11-07:00 |
| verdict | REWORK |

## Review

Reading additional input from stdin...
OpenAI Codex v0.144.5
--------
workdir: /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
model: gpt-5.5
provider: openai
approval: never
sandbox: read-only
reasoning effort: medium
reasoning summaries: none
session id: 019f710f-0279-72b0-9b3a-c0a63df5b6dc
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **000049-board-readability-labels-coldphase-eta** repository.

- repository: 000049-board-readability-labels-coldphase-eta   (root: /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta)
- issue:      000049-board-readability-labels-coldphase-eta#49   (file: workshop/issues/000049-board-readability-labels-coldphase-eta.md)
- window:     Base: 72f2d895fc676a474304fb4cde8f4606e9c5f48c^   Head: HEAD

Review the **000049-board-readability-labels-coldphase-eta** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
other repository or apply another repo's conventions.

You have no prior session context — that is the anti-collusion property. Verify
behavior against the issue's documented Spec/Plan and the code itself; do NOT
take the implementor's word in commit messages or docs at face value. Tools are
read-only: report findings precisely; the main agent (which has session context)
applies the fixes, commits, and re-runs.

Read the diff against the issue's Spec + Plan, then work the checklist below.
Categorize every finding by severity — not everything is Critical; a nitpick
marked Critical is noise.

  Critical (must fix before crossing the boundary)
    - correctness bugs; crashes / panics on unexpected input
    - behavior drift from stated contracts (for ports of existing code where
      byte-faithfulness was promised, diff against the source)
    - silent error swallowing where the source raised
  Important (fix before the boundary if cheap)
    - API design of newly-introduced internal packages (downstream work will
      consume them; is the surface stable?)
    - missing test coverage that would catch the kind of bug shipped
    - inconsistent error handling across the diff
  Minor (note for future)
    - style nits, naming, comment density; performance only if hot-path

## Review checklist

Code quality
  - Clean separation of concerns; edge cases handled (empty / nil / unexpected).
  - Proper error handling — no silent swallowing where the source raised.
  - No duplicated logic / copy-paste that should be a shared helper.

Testing
  - Tests pin real logic, not mocks reasserting the implementation.
  - The kind of bug this diff could ship is covered.
  - PURE entities tested without IO; INTEGRATION via injected fakes (see below).

Requirements traceability
  - Every Plan checklist item this boundary claims is actually delivered.
  - Implementation matches the Spec; no undeclared scope creep.
  - Breaking changes documented.

Production readiness
  - Migration / backward-compatibility considered where state or formats change.
  - Docs / atlas updated for new surface (see the Docs update gate).

## Core concepts cross-check (if the plan has a Core concepts table)

The plan should list entities in a greppable table — name, kind
(PURE/INTEGRATION), file location, status (new/modified/deleted). For each row:
  - Verify the entity exists at the stated path (grep the diff or filesystem).
  - PURE: tests run without IO (no exec, net, mutable fs). If tests need mocks
    to run, it isn't really PURE — flag Critical and recommend promoting it to
    INTEGRATION.
  - INTEGRATION: injected into pure callers, not invoked directly from business
    logic.
  - "modified" / "deleted": the diff shows the expected change/removal at the
    stated location.
Any contradiction between table and code = Critical finding, plus a plan-revision
recommendation (a "## Revisions" entry so the plan stops claiming what the code
doesn't deliver).

## Docs update gate (atlas + README, per AGENTS.md §8)

The boundary should update user-facing docs for any new surface introduced:

  - **atlas/** — new architectural surface, flow, or terminology. Scan the diff
    for new entity types, subcommands, conventions, file-tree locations. Any
    present without corresponding atlas/ changes in the same range = Important
    finding ("atlas update appears missing for <surface>").
  - **README.md** — new user-facing surface a reader runs or types: subcommands,
    flags, keybindings, config keys, install/usage steps. If the diff adds or
    changes such surface and README.md is not updated in the same range =
    Important finding ("README update appears missing for <surface>"). This is the
    class of gap that used to surface only at the merge-time `specs` judge (#142);
    catch it here, at the earliest gate, before the close verdict is recorded.

## Architecture (the at-review backstop — these matter most long-term)

Work through each of ARCH-DRY, ARCH-PURE, ARCH-PURPOSE explicitly, applying its at-review lens. The
full principle definitions are delivered in the ARCHITECTURE PRINCIPLES block
right after this prompt — for EACH marker, state pass or flag, and cite the
marker (e.g. ARCH-DRY) in any finding. Architecture is where review has the
least training signal and the longest-delayed payoff, so be deliberate here, not
holistic.

## Verdict + output

Begin your response with this fenced verdict block — the machine-read handoff:

```verdict
verdict: <SHIP | FIX-THEN-SHIP | REWORK>
confidence: <high | medium | low>
```

  SHIP           ready; ship it
  FIX-THEN-SHIP  ship after addressing the findings (non-blocking at the gate)
  REWORK         blocking; needs rework before shipping — fix + re-run

The fenced ```` ```verdict ```` block above is the **authoritative machine-read
handoff** — emit it as the first thing in your response. (A prose
`VERDICT: <TOKEN>` first line still satisfies the legacy contract as a fallback,
but the block is what the binary trusts.)

After the verdict block: a 1-paragraph summary — what worked, what blocks SHIP if
it isn't — followed by:
  1. Strengths: 2-5 specific things done well (file:line where useful). Affirm
     validated approaches so the operator knows what's confirmed-good ground.
     Empty acceptable for trivial boundaries.
  2. Critical findings (file:line + fix sketch); empty if none.
  3. Important findings (same format).
  4. Minor findings (terse one-liners).
  5. Test coverage notes.
  6. Architectural notes for upcoming work.
  7. Plan revision recommendations: specific "## Revisions" entries the plan
     needs (empty if the plan still matches the code).


ARCHITECTURE PRINCIPLES — work through each of the 3 entries below explicitly, applying its `at-review` lens; cite the marker (e.g. ARCH-DRY) in any finding.

# Architecture principles (ARCH-*)

Injected architectural taste — the structural decisions whose payoff (or cost)
shows up many turns, often months, down the road. Agents are strong at local
tactics and weak here, so these are checked **at-plan** (when the design is being
made — highest leverage) and **at-review** (backstop, on the diff). Cite the
marker (e.g. `ARCH-DRY`) in plans, `## Log` entries, and review findings.

This file is the single source; it is embedded into the planning, plan-quality,
and code-review prompts. The human narrative lives in AGENTS.md "Core Design
Principles"; this is its machine-delivered companion.

## ARCH-DRY — Don't Repeat Yourself

- **principle:** Reuse before adding. One source of truth per fact/behavior; no
  duplicated logic, copy-pasted blocks, or parallel functions that should be one
  shared helper.
- **at-plan:** Flag a plan that re-implements something the codebase already has,
  or that will obviously duplicate logic across the new files instead of
  extracting a shared helper. Name the existing thing it should reuse.
- **at-review:** Flag duplicated logic / copy-pasted blocks / near-identical
  functions in the diff; point at the consolidation (file:line + the shared
  helper they should become).

## ARCH-PURE — Pure core, thin IO shell

- **principle:** The majority of code is pure functions (deterministic, no side
  effects); a thin "glue" layer at the boundary touches IO/UI/network/clock. Pure
  functions are unit-tested directly; the glue is kept small and injected.
- **at-plan:** Flag a design that buries business logic inside IO/handlers, or
  that will only be testable with heavy mocks (a sign logic isn't separated from
  IO). The plan should name what's pure vs the thin IO seam.
- **at-review:** Flag business logic mixed with IO in the diff; logic that should
  be a pure function injected into a thin caller. If a test needs mocks to run a
  "pure" entity, it isn't pure — recommend extracting the IO to the boundary.

## ARCH-PURPOSE — Serve the issue's actual purpose

- **principle:** Deliver the issue's stated purpose, not the easy subset of it. A
  single-source / "compiled to consumers" change is not done until **every
  consumer derives** from the source — the source is *enforced*, not just
  documentation a surface happens to restate; a hand-maintained restatement of the
  model is a deferred consumer, not a finished one. "Follow-up" is for separable
  extensions, never for the thing that is the point. This is the *opposite axis*
  from Simplicity-First/YAGNI: not "build for an imagined future," but "don't
  **under**-deliver the purpose you already committed to."
- **at-plan:** Flag a plan whose scope is a strict subset of the issue's stated
  goal / Done-when where the part deferred as "follow-up" *is* the purpose (e.g.
  wires one consumer + enforcement but leaves the consumers that motivated the
  issue as documentation that doesn't derive). Ask: does the plan fulfill the
  purpose, or just the cheap win? Name the deferred purpose.
- **at-review:** Does the diff *fulfill* the purpose or settle for the easy win?
  For a single-source change, run the **shadow-sweep** — enumerate the consumers,
  confirm each derives from the source, flag any remaining hand-maintained
  restatement of the model. A "follow-up" that is actually the deferred point of
  the issue is a finding, not a deferral.


OUTPUT CONTRACT (machine-read — do not deviate). LEAD your response with the
fenced ```verdict block shown above — that is the authoritative handoff the binary
reads (its `verdict:` value is one of the listed tokens). Everything after the block
is advisory: a non-blocking verdict WITH findings still PASSES the gate. A bare
`VERDICT: <TOKEN>` line is accepted only as a FALLBACK when the block is absent.

Diff:
diff --git a/atlas/experiment.md b/atlas/experiment.md
index 7eb9adb..5f047e0 100644
--- a/atlas/experiment.md
+++ b/atlas/experiment.md
@@ -166,7 +166,14 @@ wrapped by **thin step-executables** honoring the contract above. Hermetic via *
   `Aggregate` → **mean±SE**, the honest procedure estimate (`reportEstimate`). **metis#32:** the run now
   **records** per-`(outer-fold, config)` inner rows + per-`(outer-fold, family)` outer rows to the ledger
   (`Level`-keyed) — the signal `metis select` reduces to pick the family. `metis run` **measures only,
-  never ships** (shipping moved to `metis select --promote`). Honesty of the score-over-full-data
+  never ships** (shipping moved to `metis select --promote`). **metis#52:** `select --cohort` lists the
+  fingerprint cohorts (delegates to the #39 core), and every pick line carries its
+  `· point <addr>` override handle (a representative ledger-row addr; round-trips through
+  `--point`). **metis#50:** a sweep ends with the
+  run-end summary — elapsed wall-clock, rows→ledger, the cohort fingerprint, and the paste-ready
+  `metis select … --fingerprint <fp>` follow-ups (completing #39's visibility loop: the operator
+  never scrapes scrollback to assemble the next command; degraded capture degrades to `cohort ?`
+  with un-pinned hints). Honesty of the score-over-full-data
   refit holds while features are stateless; stateful features (metis#20) inherit fold-safety via the
   fold-expressed score run.
 - **Honest family selection (metis#32) — three commands, `run` measures / `select` chooses / `kaggle
@@ -196,9 +203,17 @@ wrapped by **thin step-executables** honoring the contract above. Hermetic via *
   (non-nil sem + a `syncWriter` over `out`) in one home. Determinism of persisted artifacts: the fan-out's
   completion-order `pass.points` are `sortPointRuns`-sorted before the manifest/ledger write; the
   `sweepPass` mutex guards the shared `configs`/`points`/`err` bookkeeping (the honest reduce stays pure
-  in the sampler). Caveats (flag help): each leaf is a Python process that may itself multi-thread
-  (BLAS/`n_jobs`) so `n=NumCPU` can oversubscribe; a COLD cache thundering-herds the shared upstream;
-  clean per-`k/n` progress is deferred to metis#30.
+  in the sampler). Caveats (flag help): a COLD cache thundering-herds the shared upstream; clean
+  per-`k/n` progress is deferred to metis#30.
+- **Default leaf BLAS pins (metis#48) — `cmd/metis/blaspins.go`:** the parallelism budget belongs
+  to the ORCHESTRATOR (the #31 semaphore), so `runExperiment` computes the four single-thread pins
+  (`OMP/OPENBLAS/VECLIB/MKL_NUM_THREADS=1`) ONCE per top-level run — minus any name the operator
+  exported (an explicit value always wins) — announces one loud note, and injects them at BOTH
+  spawn seams: the legacy `execStep` child env and the fork-server process env (children inherit).
+  `metis select --promote` is deliberately unpinned (serial single all-data fit — multi-threaded
+  BLAS wanted). Env is outside run identity by design (`Kpre` = {step_id, uses, with, seed,
+  upstream}; HIT-validation re-hashes read-set D; fingerprint is git state), so pins perturb no
+  cache key or fingerprint.
 - **Warm fork-server leaf executor (metis#44) — `metis/forkserver.py` + `cmd/metis/forkexec.go`:**
   kills the per-leaf `uv run → fresh python → import pandas/sklearn` tax (~1s measured/spawn, ~5k
   spawns/sweep). One warm server per **project root** (metis's and kbench's venvs differ), started
diff --git a/atlas/index.md b/atlas/index.md
index a48ef7e..18d8b0a 100644
--- a/atlas/index.md
+++ b/atlas/index.md
@@ -44,7 +44,7 @@ identical on a non-Kaggle platform?* — if yes, it lives here.
   command (record IO on inspect/error paths only); each `metis run` prints
   `recording under code_fingerprint <hash> (commit <sha>, clean|dirty)` at capture time.
   **metis#32 retired `metis ledger select` + `metis promote`** — selection
-  moved to **`metis select <shape> [--best | --best-per-model-class] [--promote]`** (see the run/select
+  moved to **`metis select <shape> [--best | --best-per-model-class | --point ADDR | --cohort] [--promote]`** (see the run/select
   command model below): it reads the nested-CV ledger, picks the FAMILY on the honest OUTER estimate + the
   CONFIG on the inner CV, and `--promote` reconstructs the winner (the pure `promotedExperiment` —
   re-expands the shape + matches by free-params, reusing `shapeConfigToExperiment`: `data ++
@@ -139,7 +139,13 @@ identical on a non-Kaggle platform?* — if yes, it lives here.
   rebuild the exact run DIRECTLY, not by re-expanding the grid. The **driver** is `cmd/metis`: `metis run` on an experiment-shape
   drives the real three-level loop (`runShapeSweep`: `Run(SingleDriver) ⊃ Run(GridConfigs) ⊃
   Run(FixedKFolds)`), running each `(config, fold)` through the shared `runResolvedExperiment` (cached
-  runner) keyed by its content-address. The sweeper (`GridConfigs ⊃ FixedKFolds`) is extracted as
+  runner) keyed by its content-address. Parallel sampler fan-out remains order-preserving, while every
+  concrete run crosses one sweep-scoped `2n` admission controller before side effects; the independent
+  leaf semaphore remains capped at `n`. The controller (`cmd/metis/runcontrol.go`, installed at
+  `runResolvedExperiment`) also owns the experiment-wide first failure, so queued runs produce no
+  observable state after cancellation; the scheduling and cancellation regressions in
+  `cmd/metis/parallel_test.go` pin both budgets and the failure boundary. The sweeper
+  (`GridConfigs ⊃ FixedKFolds`) is extracted as
   `runSweeper`/`sweepPass` (per-call accumulators) so `driver:cv` can run it once **per outer fold**,
   each pass repointed at that fold's sealed `analysis_i` + confined (`METIS_READ_ROOT`) — the flat
   `driver:single` path is `runSweeper` with `baseRef=nil`, unconfined. Each fold builds a per-fold experiment (`data ++ engine-synthesized
diff --git a/cmd/metis/blaspins.go b/cmd/metis/blaspins.go
new file mode 100644
index 0000000..7f440e7
--- /dev/null
+++ b/cmd/metis/blaspins.go
@@ -0,0 +1,45 @@
+package main
+
+import "strings"
+
+// blasPinDefaults are the single-thread pins metis injects into LEAF subprocesses by
+// default (metis#48): the parallelism budget belongs to the ORCHESTRATOR (the metis#31
+// leaf semaphore), not to each leaf's BLAS — NumCPU leaves × multi-threaded BLAS
+// oversubscribes ~NumCPU× (observed: load-avg 83 on 12 cores, throughput ≈ 0).
+//
+// Cache identity: env is deliberately OUTSIDE run identity — Kpre hashes
+// {step_id, uses, with, seed, upstream} (pkg/cache), HIT-validation re-hashes the
+// read-set D (file blob hashes), and the code fingerprint is git state. Injecting
+// pins perturbs neither cache keys nor fingerprints — exactly as the RUNBOOK's
+// manual `OMP_NUM_THREADS=1 metis run` never did.
+var blasPinDefaults = []string{
+	"MKL_NUM_THREADS=1",
+	"OMP_NUM_THREADS=1",
+	"OPENBLAS_NUM_THREADS=1",
+	"VECLIB_MAXIMUM_THREADS=1",
+}
+
+// blasPins returns the defaults NOT already set in environ — an explicit operator
+// value always wins (escape hatch by construction: `export OMP_NUM_THREADS=8`
+// passes through untouched). Pure. Always non-nil: an all-suppressed result is
+// empty, distinguishable from runOpts' nil "not yet computed" sentinel.
+func blasPins(environ []string) []string {
+	pins := make([]string, 0, len(blasPinDefaults))
+	for _, def := range blasPinDefaults {
+		name := def[:strings.IndexByte(def, '=')]
+		if !envHasName(environ, name) {
+			pins = append(pins, def)
+		}
+	}
+	return pins
+}
+
+// envHasName reports whether environ sets exactly `name` (match up to '=').
+func envHasName(environ []string, name string) bool {
+	for _, kv := range environ {
+		if strings.HasPrefix(kv, name) && len(kv) > len(name) && kv[len(name)] == '=' {
+			return true
+		}
+	}
+	return false
+}
diff --git a/cmd/metis/blaspins_e2e_test.go b/cmd/metis/blaspins_e2e_test.go
new file mode 100644
index 0000000..d9652f9
--- /dev/null
+++ b/cmd/metis/blaspins_e2e_test.go
@@ -0,0 +1,74 @@
+package main
+
+import (
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+// TestRun_BlasPinsDefaultAndNote (metis#48 e2e): a bare run (no ambient pins) announces
+// ONE loud note and the leaf subprocess sees the pins; an operator-exported value passes
+// through untouched and drops out of the note. Drives runExperiment (the real wiring:
+// blasPins → runOpts.leafPins → execStep), real shell step, no uv.
+//
+// Sweep-path once-ness is by construction (verified at plan review): runExperiment is
+// entered exactly once per invocation (sole caller main.go) and all nested sweep spawns
+// are struct copies carrying the computed leafPins — so this single-run count==1 plus
+// that structure covers the sweep path.
+func TestRun_BlasPinsDefaultAndNote(t *testing.T) {
+	// ambient: exactly ONE operator choice set; the other three genuinely absent.
+	// t.Setenv registers the restore; Unsetenv then makes absence real (an operator
+	// shell following the old RUNBOOK exports all four — CI must not inherit that).
+	for _, k := range []string{"OPENBLAS_NUM_THREADS", "VECLIB_MAXIMUM_THREADS", "MKL_NUM_THREADS"} {
+		t.Setenv(k, "sentinel")
+		os.Unsetenv(k)
+	}
+	t.Setenv("OMP_NUM_THREADS", "7")
+
+	root := t.TempDir()
+	stepDir := filepath.Join(root, "steps", "test")
+	if err := os.MkdirAll(stepDir, 0o755); err != nil {
+		t.Fatal(err)
+	}
+	script := "#!/bin/sh\nenv > env.txt\necho '{\"ok\": 1}' > metrics.json\n"
+	if err := os.WriteFile(filepath.Join(stepDir, "envstep"), []byte(script), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	expPath := filepath.Join(root, "exp.md")
+	// plain-experiment fixture — steps live in the YAML FRONTMATTER (the
+	// testdata/experiment/run-echo.md convention)
+	exp := "---\ntype: experiment\nid: pins-e2e\nseed: 1\nstatus: active\nsteps:\n  - id: e\n    uses: test/envstep\n---\n# pins e2e\n"
+	if err := os.WriteFile(expPath, []byte(exp), 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	var out strings.Builder
+	if _, err := runExperiment(runOpts{
+		expPath:  expPath,
+		runID:    "r1",
+		stepPath: []string{filepath.Join(root, "steps")},
+		out:      &out,
+	}); err != nil {
+		t.Fatalf("run: %v\n%s", err, out.String())
+	}
+
+	// (a) ONE note, naming only the three injected pins (OMP dropped — operator won)
+	if n := strings.Count(out.String(), "metis: leaf BLAS pinned"); n != 1 {
+		t.Fatalf("want exactly 1 pin note, got %d:\n%s", n, out.String())
+	}
+	if strings.Contains(out.String(), "OMP_NUM_THREADS") {
+		t.Errorf("note must not claim the operator-set var:\n%s", out.String())
+	}
+	// (b) child env: three pins at 1, operator's OMP at 7
+	b, err := os.ReadFile(filepath.Join(root, "runs", "r1", "e", "env.txt"))
+	if err != nil {
+		t.Fatal(err)
+	}
+	envTxt := string(b)
+	for _, want := range []string{"OPENBLAS_NUM_THREADS=1", "VECLIB_MAXIMUM_THREADS=1", "MKL_NUM_THREADS=1", "OMP_NUM_THREADS=7"} {
+		if !strings.Contains(envTxt, want) {
+			t.Errorf("child env missing %q; got:\n%s", want, envTxt)
+		}
+	}
+}
diff --git a/cmd/metis/blaspins_test.go b/cmd/metis/blaspins_test.go
new file mode 100644
index 0000000..e42d051
--- /dev/null
+++ b/cmd/metis/blaspins_test.go
@@ -0,0 +1,61 @@
+package main
+
+import (
+	"reflect"
+	"testing"
+)
+
+// TestBlasPins_BareEnv: no ambient thread vars → all four pins injected, sorted.
+func TestBlasPins_BareEnv(t *testing.T) {
+	got := blasPins([]string{"PATH=/usr/bin", "HOME=/h"})
+	want := []string{
+		"MKL_NUM_THREADS=1",
+		"OMP_NUM_THREADS=1",
+		"OPENBLAS_NUM_THREADS=1",
+		"VECLIB_MAXIMUM_THREADS=1",
+	}
+	if !reflect.DeepEqual(got, want) {
+		t.Errorf("blasPins = %v, want %v", got, want)
+	}
+}
+
+// TestBlasPins_OperatorValueWins: an ambient-set var is NOT pinned — an explicit
+// operator choice always wins (the issue's escape hatch by construction).
+func TestBlasPins_OperatorValueWins(t *testing.T) {
+	got := blasPins([]string{"OMP_NUM_THREADS=8", "PATH=/usr/bin"})
+	for _, kv := range got {
+		if kv == "OMP_NUM_THREADS=1" {
+			t.Fatalf("ambient OMP_NUM_THREADS=8 must suppress the pin; got %v", got)
+		}
+	}
+	if len(got) != 3 {
+		t.Errorf("want 3 remaining pins, got %v", got)
+	}
+}
+
+// TestBlasPins_AllSetIsEmptyNonNil: fully pinned ambient env → empty but NON-nil
+// (runExperiment uses nil as "not yet computed"; empty must not recompute).
+func TestBlasPins_AllSetIsEmptyNonNil(t *testing.T) {
+	got := blasPins([]string{
+		"OMP_NUM_THREADS=4", "OPENBLAS_NUM_THREADS=4",
+		"VECLIB_MAXIMUM_THREADS=4", "MKL_NUM_THREADS=4",
+	})
+	if got == nil || len(got) != 0 {
+		t.Errorf("want empty non-nil, got %#v (nil=%v)", got, got == nil)
+	}
+}
+
+// TestBlasPins_PrefixNotName: OMP_NUM_THREADS_X=9 is a DIFFERENT var — must not
+// suppress the OMP_NUM_THREADS pin (name match is exact, up to '=').
+func TestBlasPins_PrefixNotName(t *testing.T) {
+	got := blasPins([]string{"OMP_NUM_THREADS_X=9"})
+	found := false
+	for _, kv := range got {
+		if kv == "OMP_NUM_THREADS=1" {
+			found = true
+		}
+	}
+	if !found {
+		t.Errorf("prefix-named var must not suppress the real pin; got %v", got)
+	}
+}
diff --git a/cmd/metis/board.go b/cmd/metis/board.go
index a8a8d51..0407870 100644
--- a/cmd/metis/board.go
+++ b/cmd/metis/board.go
@@ -209,6 +209,19 @@ func (b *boardWriter) forceFlush() {
	b.flushLocked(b.now())
 }

+// discardFrame atomically erases the live board and forgets it. Pending complete
+// ordinary lines still flush, but neither this update nor the later deferred close
+// can redraw stale progress after a sweep failure.
+func (b *boardWriter) discardFrame() {
+	b.mu.Lock()
+	defer b.mu.Unlock()
+	if b.closed {
+		return
+	}
+	b.frame = nil
+	b.flushLocked(b.now())
+}
+
 // flushLocked is the ONE atomic terminal update: erase the painted board, dump the
 // complete pending lines, redraw the stored frame — bracketed in DEC 2026 synchronized
 // output (metis#47: supporting terminals apply it atomically, killing the erase→redraw
diff --git a/cmd/metis/board_test.go b/cmd/metis/board_test.go
index 284b76b..67ba9e9 100644
--- a/cmd/metis/board_test.go
+++ b/cmd/metis/board_test.go
@@ -221,6 +221,33 @@ func TestBoardWriter_CloseFlushesPending(t *testing.T) {
	}
 }

+func TestBoardWriter_DiscardFrameErasesWithoutRedraw(t *testing.T) {
+	var term strings.Builder
+	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond))
+	bw.paint([]string{"folds 2/8", "31.2 folds/min · ETA 12s"})
+	offset := term.Len()
+
+	bw.discardFrame()
+	bw.close()
+	suffix := term.String()[offset:]
+	if !strings.Contains(suffix, "\x1b[2A\x1b[J") {
+		t.Fatalf("discard must erase the painted two-line frame: %q", suffix)
+	}
+	for _, stale := range []string{"folds 2/8", "folds/min", "ETA"} {
+		if strings.Contains(suffix, stale) {
+			t.Errorf("discard/close redrew stale token %q: %q", stale, suffix)
+		}
+	}
+	if !strings.HasSuffix(suffix, "\x1b[?25h") {
+		t.Errorf("close after discard must restore the cursor: %q", suffix)
+	}
+	n := term.Len()
+	bw.close()
+	if term.Len() != n {
+		t.Fatal("close after discard must remain idempotent")
+	}
+}
+
 // Board mode end-to-end over the fixture sweep: frames paint (cursor hide, fold rows),
 // the #30 plain lines are REPLACED (not duplicated), the final frame carries the
 // completed counts, and a capture warning — the plan-review bypass route (o.out) —
@@ -283,6 +310,85 @@ func TestRunExperiment_BoardMode(t *testing.T) {
	}
 }

+func TestRunExperiment_BoardFailureRejectsPostPublicationTickAndDiscardsFrame(t *testing.T) {
+	ws := t.TempDir()
+	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
+	control := newRunControl(2)
+	exec := newFailureBarrierExec()
+	out := &concurrentBuffer{}
+	boardTick := make(chan time.Time)
+	tickSelected := make(chan struct{}, 2)
+	tickFinished := make(chan struct{}, 2)
+	publishedOffset := make(chan int, 1)
+	postFailureTickSend := make(chan error, 1)
+	control.beforeFailureUnlock = func() {
+		publishedOffset <- out.len()
+		postFailureTickSend <- sendBoardTickWithin(boardTick, at(2000), "post-publication tick receive")
+		close(exec.failurePublished)
+	}
+	result := make(chan error, 1)
+	go func() {
+		_, err := runExperiment(runOpts{
+			expPath: expPath, now: fixedNow(),
+			git: fakeGitProbe{name: "metis", sha: "sha"}, exec: exec, out: out,
+			maxParallel: 2, runControl: control, tui: true, boardTick: boardTick,
+			beforeBoardTick: func() { tickSelected <- struct{}{} },
+			afterBoardTick:  func() { tickFinished <- struct{}{} },
+		})
+		result <- err
+	}()
+	for i := 0; i < 4; i++ {
+		awaitRunControl(t, exec.innerEntered, "four board-mode inner run directories")
+	}
+
+	if err := sendBoardTickWithin(boardTick, at(1000), "pre-failure tick receive"); err != nil {
+		t.Fatal(err)
+	}
+	awaitRunControl(t, tickSelected, "pre-failure board tick selection")
+	awaitRunControl(t, tickFinished, "pre-failure board tick completion")
+	preFailure := out.snapshot()
+	for _, want := range []string{"outer 0/2", "fold 0 — queued", "folds/min"} {
+		if !strings.Contains(preFailure, want) {
+			t.Fatalf("pre-failure board missing %q:\n%s", want, preFailure)
+		}
+	}
+
+	close(exec.releaseFailure)
+	offset := awaitRunControl(t, publishedOffset, "board failure publication offset")
+	if err := awaitRunControl(t, postFailureTickSend, "post-publication tick send result"); err != nil {
+		awaitRunControl(t, result, "board-mode failure cleanup after tick-send timeout")
+		t.Fatal(err)
+	}
+	awaitRunControl(t, tickSelected, "post-publication board tick selection")
+	awaitRunControl(t, tickFinished, "rejected post-publication board tick")
+	err := awaitRunControl(t, result, "board-mode failure cleanup")
+	if err == nil || !strings.Contains(err.Error(), "injected train failure") {
+		t.Fatalf("board-mode error = %v, want injected train failure", err)
+	}
+	suffix := out.snapshot()[offset:]
+	for _, forbidden := range []string{
+		"outer 0/2", "fold 0 — queued", "configs ", "folds ", "folds/min", "ETA", "score ", "estimate", "mean ",
+	} {
+		if strings.Contains(suffix, forbidden) {
+			t.Errorf("post-publication board output contains stale token %q:\n%q", forbidden, suffix)
+		}
+	}
+	if !strings.Contains(suffix, "\x1b[J") || !strings.HasSuffix(suffix, "\x1b[?25h") {
+		t.Errorf("failure cleanup must erase the board and restore the cursor: %q", suffix)
+	}
+}
+
+func sendBoardTickWithin(ch chan<- time.Time, tick time.Time, what string) error {
+	timer := time.NewTimer(runControlTestTimeout)
+	defer timer.Stop()
+	select {
+	case ch <- tick:
+		return nil
+	case <-timer.C:
+		return fmt.Errorf("timed out waiting for %s", what)
+	}
+}
+
 // --no-tui and non-TTY stdout both force tui=false through the real CLI parse; a
 // dry run never boards. (isCharDevice on a test's non-terminal stdout is false, so
 // the flag path is what we can pin here; the char-device branch is covered by the
@@ -304,7 +410,7 @@ func TestServerPool_NoticeRoutesThroughBoard(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond))
	bw.paint([]string{"BOARD"})
-	pool := newServerPool(bw) // what runExperiment does post-reorder: pool captures the compositor
+	pool := newServerPool(bw, nil) // what runExperiment does post-reorder: pool captures the compositor
	pool.noticeOnce("k", "server died; falling back to legacy exec")
	s := term.String()
	notice := strings.Index(s, "metis: forkserver: server died")
diff --git a/cmd/metis/capture.go b/cmd/metis/capture.go
index a38fecd..d462aca 100644
--- a/cmd/metis/capture.go
+++ b/cmd/metis/capture.go
@@ -144,7 +144,7 @@ func captureRunCode(closureByRepo map[string][]string, primaryRoot, specPath, re
 // captureSweepCode captures the sweep's code closure + spec ONCE (per-shape-run: the
 // closure is the same across points) to `refs/metis/sweeps/<shapeRunID>` and backfills
 // every point-record's CodeManifest with the D + commit + capture status.
-func captureSweepCode(o runOpts, man sweepManifest) error {
+func captureSweepCode(o runOpts, man sweepManifest) (record.Hash, error) {
	closureByRepo := sweepClosure(o.expPath, man)
	primary := cacheProjectRoot(o.stepPath, filepath.Dir(o.expPath))
	commit, d, status := captureRunCode(closureByRepo, primary, o.expPath, "refs/metis/sweeps/"+man.ShapeRunID)
@@ -157,14 +157,14 @@ func captureSweepCode(o runOpts, man sweepManifest) error {
	for _, p := range man.Points {
		pfp, pdirty, err := backfillCodeManifest(o.expPath, p.RunID, d, commit, status)
		if err != nil {
-			return err
+			return fp, err
		}
		if fp == "" && pfp != "" {
			fp, dirty = pfp, pdirty
		}
	}
	printFingerprintLine(o.out, fp, commit, dirty)
-	return nil
+	return fp, nil // metis#50: the run-end summary reuses the minted cohort (one mint site)
 }

 // captureSingleRun captures ONE run's code closure + spec to `refs/metis/runs/<runID>`
diff --git a/cmd/metis/capture_e2e_test.go b/cmd/metis/capture_e2e_test.go
index 38961bd..1e4286b 100644
--- a/cmd/metis/capture_e2e_test.go
+++ b/cmd/metis/capture_e2e_test.go
@@ -104,7 +104,7 @@ func TestCaptureSweepCode_BackfillsCodeManifest(t *testing.T) {
	man := sweepManifest{ShapeRunID: "srun-e2e", Points: []pointRun{{RunID: "pt-0", Status: "failed"}, {RunID: runID, Status: "ok"}}}
	var out bytes.Buffer
	o := runOpts{expPath: expPath, stepPath: []string{filepath.Join(root, "steps")}, out: &out}
-	if err := captureSweepCode(o, man); err != nil {
+	if _, err := captureSweepCode(o, man); err != nil {
		t.Fatalf("captureSweepCode: %v", err)
	}

diff --git a/cmd/metis/exec.go b/cmd/metis/exec.go
index 28be377..0dd6d1b 100644
--- a/cmd/metis/exec.go
+++ b/cmd/metis/exec.go
@@ -38,6 +38,9 @@ type execStep struct {
	pool *serverPool // metis#44: when non-nil, convention-conforming wrappers route through the
	//                  warm fork-server (one per project root) instead of a fresh uv/python spawn;
	//                  non-conforming wrappers + broken servers fall back to the legacy path below.
+	pins []string // metis#48: default leaf BLAS pins (computed once per run by runExperiment;
+	//              ambient-set names already excluded there) — appended to the legacy child env.
+	//              The fork-server path carries them on the SERVER env instead (children inherit).
 }

 // stepEnv builds the per-step METIS_* contract vars — the ONE definition both executors
@@ -125,6 +128,9 @@ func (e execStep) Execute(step experiment.Step, runDir string) (experiment.StepR
			base = append(base, kv)
		}
	}
+	// metis#48: default leaf BLAS pins (operator-exported values already won in blasPins,
+	// so no duplicate names reach the child).
+	base = append(base, e.pins...)
	for _, k := range sortedKeys(env) {
		base = append(base, k+"="+env[k])
	}
diff --git a/cmd/metis/exec_test.go b/cmd/metis/exec_test.go
index 71ff5a4..7d7adb9 100644
--- a/cmd/metis/exec_test.go
+++ b/cmd/metis/exec_test.go
@@ -231,3 +231,28 @@ func TestCollectArtifacts_RecursiveExcludesReserved(t *testing.T) {
		t.Errorf("artifacts = %v; want %v", arts, want)
	}
 }
+
+// TestExecStep_InjectsBlasPins (metis#48): the pins field reaches the child env at the
+// legacy spawn seam. The ambient-wins RULE lives in blasPins (unit-tested); this pins
+// the PLUMBING — whatever pins the wiring computed are in the subprocess env.
+func TestExecStep_InjectsBlasPins(t *testing.T) {
+	root := repoRoot(t)
+	runDir := t.TempDir()
+	e := execStep{
+		stepPath: []string{filepath.Join(root, "testdata", "steps")},
+		out:      io.Discard,
+		pins:     []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"},
+	}
+	if _, err := e.Execute(experiment.Step{ID: "e", Uses: "test/env-dump"}, runDir); err != nil {
+		t.Fatalf("Execute: %v", err)
+	}
+	b, err := os.ReadFile(filepath.Join(runDir, "e", "env.txt"))
+	if err != nil {
+		t.Fatalf("read env.txt: %v", err)
+	}
+	for _, want := range []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"} {
+		if !strings.Contains(string(b), want) {
+			t.Errorf("child env missing %q; got:\n%s", want, b)
+		}
+	}
+}
diff --git a/cmd/metis/forkexec.go b/cmd/metis/forkexec.go
index 39d1be0..b82d818 100644
--- a/cmd/metis/forkexec.go
+++ b/cmd/metis/forkexec.go
@@ -108,11 +108,14 @@ type forkReq struct {
 }

 // startForkServer launches `uv run --project <root> python -m metis.forkserver` and wires
-// the reader goroutines. The server inherits the ambient env (the operator's BLAS pins
-// apply to every child); per-step METIS_* vars travel in requests, never here.
-func startForkServer(root string) (*forkServer, error) {
+// the reader goroutines. The server env = ambient + the default single-thread BLAS pins
+// (metis#48; names the operator exported are already excluded by blasPins — an explicit
+// choice wins). Forked step children inherit this env; per-step METIS_* vars travel in
+// requests, never here.
+func startForkServer(root string, pins []string) (*forkServer, error) {
	cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.forkserver")
	cmd.Dir = root
+	cmd.Env = append(os.Environ(), pins...)
	// Own process GROUP: `uv run` spawns python as a child (no exec), and the server forks
	// step children — group-kill is the only way to reap the whole tree on a hung shutdown
	// (and a test's mid-flight kill). Normal shutdown stays graceful (stdin EOF → drain);
@@ -273,14 +276,16 @@ type serverPool struct {
	warned  map[string]bool
	out     io.Writer
	outMu   sync.Mutex
+	pins    []string // metis#48: default leaf BLAS pins, applied to every server's spawn env
 }

-func newServerPool(out io.Writer) *serverPool {
+func newServerPool(out io.Writer, pins []string) *serverPool {
	return &serverPool{
		servers: map[string]*forkServer{},
		broken:  map[string]bool{},
		warned:  map[string]bool{},
		out:     out,
+		pins:    pins,
	}
 }

@@ -310,7 +315,7 @@ func (p *serverPool) execute(spec wrapperSpec, cwd string, env map[string]string
	s := p.servers[spec.root]
	if s == nil {
		var err error
-		s, err = startForkServer(spec.root)
+		s, err = startForkServer(spec.root, p.pins)
		if err != nil {
			p.broken[spec.root] = true
			p.mu.Unlock()
diff --git a/cmd/metis/forkexec_test.go b/cmd/metis/forkexec_test.go
index ccd6861..66a55c3 100644
--- a/cmd/metis/forkexec_test.go
+++ b/cmd/metis/forkexec_test.go
@@ -103,7 +103,7 @@ func TestForkServerPool_RealServerRoundTrip(t *testing.T) {
	t.Setenv("PYTHONPATH", mods)
	t.Setenv("METIS_FORKSERVER_PRELOAD", "") // fast start; preload is a python-side concern

-	pool := newServerPool(io.Discard)
+	pool := newServerPool(io.Discard, nil)
	defer pool.shutdown()

	stepDir := filepath.Join(t.TempDir(), "s1")
@@ -145,7 +145,7 @@ func TestForkServerPool_BrokenRootFallsBack(t *testing.T) {
		t.Skip("uv not on PATH")
	}
	var out strings.Builder
-	pool := newServerPool(&syncWriter{w: &out})
+	pool := newServerPool(&syncWriter{w: &out}, nil)
	defer pool.shutdown()
	bogus := t.TempDir() // no pyproject/venv — uv run will fail (or hang-free error)
	for i := 0; i < 3; i++ {
@@ -175,7 +175,7 @@ func TestExecute_NonConformingWrapperUsesLegacyLoudly(t *testing.T) {
	ws := t.TempDir()
	var out strings.Builder
	e := execStep{stepPath: []string{filepath.Join(root, "steps")}, expDir: ws,
-		out: &out, pool: newServerPool(&out)}
+		out: &out, pool: newServerPool(&out, nil)}
	defer e.pool.shutdown()
	runDir := filepath.Join(ws, "runs", "r1")
	res, err := e.Execute(experiment.Step{ID: "sh", Uses: "test/shellstep"}, runDir)
@@ -230,7 +230,7 @@ func TestForkServerPerf_LooseBound(t *testing.T) {
	legacy := time.Since(start)

	start = time.Now()
-	pool := newServerPool(io.Discard)
+	pool := newServerPool(io.Discard, nil)
	defer pool.shutdown()
	forkOne := func() {
		dir := t.TempDir()
@@ -279,7 +279,7 @@ func TestForkServerPool_MidFlightDeathErrorsTheStep(t *testing.T) {
	t.Setenv("PYTHONPATH", mods)
	t.Setenv("METIS_FORKSERVER_PRELOAD", "")

-	pool := newServerPool(io.Discard)
+	pool := newServerPool(io.Discard, nil)
	defer pool.shutdown()
	type result struct {
		ok  bool
@@ -318,3 +318,46 @@ func TestForkServerPool_MidFlightDeathErrorsTheStep(t *testing.T) {
		t.Fatalf("mid-flight death must be a STEP ERROR (dispatched-and-lost), got ok=%v err=nil", res.ok)
	}
 }
+
+// TestForkServer_ChildInheritsBlasPins (metis#48): the pool's pins land on the SERVER
+// process env at spawn, and a forked step child inherits them (per-request env carries
+// only METIS_*). Real uv + real fork-server — the seam the operator's sweeps ride.
+func TestForkServer_ChildInheritsBlasPins(t *testing.T) {
+	if _, err := exec.LookPath("uv"); err != nil {
+		t.Skip("uv not on PATH")
+	}
+	root := repoRoot(t)
+	mods := t.TempDir()
+	if err := os.WriteFile(filepath.Join(mods, "toyenv.py"), []byte(
+		"import json, os\n"+
+			"json.dump({\"omp\": os.environ.get(\"OMP_NUM_THREADS\", \"\")}, open(\"envcap.json\", \"w\"))\n"+
+			"json.dump({\"ok\": 1}, open(\"metrics.json\", \"w\"))\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	t.Setenv("PYTHONPATH", mods)
+	t.Setenv("METIS_FORKSERVER_PRELOAD", "") // fast start; preload is a python-side concern
+	// exactness: an ambient OMP_NUM_THREADS would DUPLICATE the appended pin (CPython's
+	// os.environ is last-wins so the assertion would still green, but keep it unambiguous):
+	t.Setenv("OMP_NUM_THREADS", "sentinel") // registers restore...
+	os.Unsetenv("OMP_NUM_THREADS")          // ...then genuinely absent for the spawn
+
+	pool := newServerPool(io.Discard, []string{"OMP_NUM_THREADS=1"})
+	defer pool.shutdown()
+
+	stepDir := filepath.Join(t.TempDir(), "s1")
+	if err := os.MkdirAll(stepDir, 0o755); err != nil {
+		t.Fatal(err)
+	}
+	resp, ok, ferr := pool.execute(wrapperSpec{root: root, module: "toyenv"}, stepDir,
+		map[string]string{"METIS_STEP_DIR": stepDir})
+	if !ok || ferr != nil || resp.Exit != 0 {
+		t.Fatalf("fork exec failed: ok=%v err=%v resp=%+v", ok, ferr, resp)
+	}
+	b, err := os.ReadFile(filepath.Join(stepDir, "envcap.json"))
+	if err != nil {
+		t.Fatalf("read envcap.json: %v", err)
+	}
+	if !strings.Contains(string(b), `"omp": "1"`) {
+		t.Errorf("forked child must inherit the server's pin; envcap.json = %s", b)
+	}
+}
diff --git a/cmd/metis/identity_e2e_test.go b/cmd/metis/identity_e2e_test.go
index bf7312c..e82c1c6 100644
--- a/cmd/metis/identity_e2e_test.go
+++ b/cmd/metis/identity_e2e_test.go
@@ -102,7 +102,7 @@ func TestCodeIdentity_TwoRowsOnCodeChange(t *testing.T) {
		})
	}
	capture := func() record.RunRecord {
-		if err := captureSweepCode(o, man); err != nil {
+		if _, err := captureSweepCode(o, man); err != nil {
			t.Fatalf("captureSweepCode: %v", err)
		}
		rb, _ := os.ReadFile(filepath.Join(root, "runs", pa, "record.json"))
diff --git a/cmd/metis/main.go b/cmd/metis/main.go
index 65e8331..227ddba 100644
--- a/cmd/metis/main.go
+++ b/cmd/metis/main.go
@@ -45,7 +45,7 @@ func cmdRun(args []string) error {
	sampleN := fs.Int("sample", 0, "metis#42: run m of the k outer folds (sparse fold sampling; 0/omitted = all k). k stays the estimand (each fold trains on (k-1)/k of the rows); m only trades precision for cost — use to probe a higher k (e.g. k=10, --sample 3) without the full k× bill. The SE over m<k folds is noisy (m-1 df): probe with it, don't re-select what ships on it. Errors on m>k, on a single-config (flat) run, and combined with --fast.")
	forkserver := fs.Bool("forkserver", true, "metis#44: run convention-conforming step wrappers through a warm per-root fork-server (pre-imported pandas/sklearn; ~1s spawn tax removed per leaf). --forkserver=false = legacy per-step uv/python spawn (the escape hatch); non-conforming wrappers and failed servers fall back to legacy automatically (loud, once).")
	noTUI := fs.Bool("no-tui", false, "metis#38: force the plain progress lines even on a TTY (the live board is default for a sweep when stdout is a terminal; piped/redirected output always gets plain lines)")
-	parallel := fs.Int("parallel", defaultParallel(), "metis#31: max concurrent step subprocesses across ALL sweep levels (driver×sweeper×resample share one global cap); <=1 = serial (exact pre-#31 behavior). Default runtime.NumCPU(), overridable by METIS_MAX_PARALLEL. Caveat: each leaf is a Python process that may itself multi-thread (BLAS / sklearn n_jobs) — n=NumCPU can oversubscribe cores; pin OMP_NUM_THREADS=1 or set n below NumCPU. On a COLD cache the first batch's ≤n points may each recompute the shared upstream (a bounded thundering herd).")
+	parallel := fs.Int("parallel", defaultParallel(), "metis#31: max concurrent step subprocesses across ALL sweep levels (driver×sweeper×resample share one global cap); <=1 = serial (exact pre-#31 behavior). Default runtime.NumCPU(), overridable by METIS_MAX_PARALLEL. Leaf BLAS is pinned single-thread by default (metis#48; export a *_NUM_THREADS value yourself to override), so n is the ONE parallelism knob. On a COLD cache the first batch's ≤n points may each recompute the shared upstream (a bounded thundering herd).")
	if err := fs.Parse(args); err != nil {
		return err
	}
diff --git a/cmd/metis/nestedcv_e2e_test.go b/cmd/metis/nestedcv_e2e_test.go
index 1539587..1800d60 100644
--- a/cmd/metis/nestedcv_e2e_test.go
+++ b/cmd/metis/nestedcv_e2e_test.go
@@ -69,6 +69,17 @@ func TestNestedCV_ProducesHonestEstimateNoShip(t *testing.T) {
	if !strings.Contains(finalProg, "outer 2/2") || !strings.Contains(finalProg, "est 0.") {
		t.Errorf("the final progress line must carry the completed outer count + a numeric est; got: %q", finalProg)
	}
+	// metis#50: the run ends with a paste-ready summary — elapsed, rows → ledger, cohort,
+	// and the select commands with the fingerprint pre-filled.
+	if !strings.Contains(s, "metis: done in ") {
+		t.Errorf("missing the run-end summary:\n%s", s)
+	}
+	sum := s[strings.Index(s, "metis: done in "):]
+	for _, want := range []string{"rows →", ".ledger.csv", "cohort ", "metis select ", "--fingerprint ", "--best --promote", "metis ledger fingerprints "} {
+		if !strings.Contains(sum, want) {
+			t.Errorf("summary missing %q:\n%s", want, sum)
+		}
+	}
	// One held-out score per (outer fold × family): outerK = sweeper.cv.k = 2, and a,b are one
	// family → 2 held-out lines.
	if n := strings.Count(s, "→ held-out "); n != 2 {
diff --git a/cmd/metis/parallel_test.go b/cmd/metis/parallel_test.go
index 568ff2a..afb86cc 100644
--- a/cmd/metis/parallel_test.go
+++ b/cmd/metis/parallel_test.go
@@ -2,15 +2,21 @@ package main

 import (
	"bytes"
+	"encoding/json"
+	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
+	"reflect"
+	"strings"
	"sync"
+	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
+	"github.com/xianxu/metis/pkg/record"
 )

 // TestSweep_ParallelEqualsSerial (metis#31 M3, cmd-level): the SAME sweep run
@@ -22,7 +28,7 @@ import (
 // write the fan-out touches (configs/points/err).
 func TestSweep_ParallelEqualsSerial(t *testing.T) {
	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds = 6 per-fold rows
-	run := func(maxPar int) (ledger, manifest []byte) {
+	run := func(maxPar int) (ledger, manifest []byte, runs map[string]experiment.Run, records map[string]record.RunRecord) {
		ws := t.TempDir()
		expPath := writeShapeFile(t, ws, body)
		if _, err := runExperiment(runOpts{
@@ -48,16 +54,212 @@ func TestSweep_ParallelEqualsSerial(t *testing.T) {
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
-		return lb, mb
+		return lb, mb, loadPersistedRuns(t, ws), loadPersistedRecords(t, ws)
	}
-	sL, sM := run(1)
-	pL, pM := run(8)
+	sL, sM, sRuns, sRecords := run(1)
+	pL, pM, pRuns, pRecords := run(8)
	if !bytes.Equal(sL, pL) {
		t.Errorf("ledger bytes differ serial vs parallel:\n--serial--\n%s\n--parallel--\n%s", sL, pL)
	}
	if !bytes.Equal(sM, pM) {
		t.Errorf("manifest bytes differ serial vs parallel:\n--serial--\n%s\n--parallel--\n%s", sM, pM)
	}
+	if !reflect.DeepEqual(sRuns, pRuns) {
+		t.Errorf("run.json values differ serial vs parallel:\nserial=%#v\nparallel=%#v", sRuns, pRuns)
+	}
+	if !reflect.DeepEqual(sRecords, pRecords) {
+		t.Errorf("record.json values differ serial vs parallel:\nserial=%#v\nparallel=%#v", sRecords, pRecords)
+	}
+}
+
+func loadPersistedRuns(t *testing.T, root string) map[string]experiment.Run {
+	t.Helper()
+	paths, err := filepath.Glob(filepath.Join(root, "runs", "*", "run.json"))
+	if err != nil {
+		t.Fatal(err)
+	}
+	got, err := decodePersistedRuns(paths)
+	if err != nil {
+		t.Fatal(err)
+	}
+	return got
+}
+
+func decodePersistedRuns(paths []string) (map[string]experiment.Run, error) {
+	got := make(map[string]experiment.Run, len(paths))
+	for _, path := range paths {
+		directoryID := filepath.Base(filepath.Dir(path))
+		if _, exists := got[directoryID]; exists {
+			return nil, fmt.Errorf("duplicate run directory identity %q", directoryID)
+		}
+		b, err := os.ReadFile(path)
+		if err != nil {
+			return nil, err
+		}
+		var run experiment.Run
+		if err := json.Unmarshal(b, &run); err != nil {
+			return nil, fmt.Errorf("parse %s: %w", path, err)
+		}
+		if run.ID != directoryID {
+			return nil, fmt.Errorf("run payload id %q does not match directory identity %q", run.ID, directoryID)
+		}
+		run.Started = ""
+		run.Finished = ""
+		got[directoryID] = run
+	}
+	return got, nil
+}
+
+func loadPersistedRecords(t *testing.T, root string) map[string]record.RunRecord {
+	t.Helper()
+	paths, err := filepath.Glob(filepath.Join(root, "runs", "*", "record.json"))
+	if err != nil {
+		t.Fatal(err)
+	}
+	got, err := decodePersistedRecords(paths)
+	if err != nil {
+		t.Fatal(err)
+	}
+	return got
+}
+
+func decodePersistedRecords(paths []string) (map[string]record.RunRecord, error) {
+	got := make(map[string]record.RunRecord, len(paths))
+	for _, path := range paths {
+		directoryID := filepath.Base(filepath.Dir(path))
+		if _, exists := got[directoryID]; exists {
+			return nil, fmt.Errorf("duplicate record directory identity %q", directoryID)
+		}
+		b, err := os.ReadFile(path)
+		if err != nil {
+			return nil, err
+		}
+		var rec record.RunRecord
+		if err := json.Unmarshal(b, &rec); err != nil {
+			return nil, fmt.Errorf("parse %s: %w", path, err)
+		}
+		if rec.RunID != directoryID {
+			return nil, fmt.Errorf("record payload run_id %q does not match directory identity %q", rec.RunID, directoryID)
+		}
+		rec.Started = ""
+		rec.Finished = ""
+		got[directoryID] = rec
+	}
+	return got, nil
+}
+
+func TestSemanticArtifactLoadersRejectMismatchedAndDuplicateDirectoryIdentity(t *testing.T) {
+	writeArtifact := func(t *testing.T, path string, value any) {
+		t.Helper()
+		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
+			t.Fatal(err)
+		}
+		b, err := json.Marshal(value)
+		if err != nil {
+			t.Fatal(err)
+		}
+		if err := os.WriteFile(path, b, 0o644); err != nil {
+			t.Fatal(err)
+		}
+	}
+
+	t.Run("run.json", func(t *testing.T) {
+		path := filepath.Join(t.TempDir(), "runs", "dir-id", "run.json")
+		writeArtifact(t, path, experiment.Run{ID: "payload-id"})
+		if _, err := decodePersistedRuns([]string{path}); err == nil || !strings.Contains(err.Error(), "dir-id") {
+			t.Fatalf("mismatched run identity error = %v, want directory identity diagnostic", err)
+		}
+		writeArtifact(t, path, experiment.Run{ID: "dir-id"})
+		if _, err := decodePersistedRuns([]string{path, path}); err == nil || !strings.Contains(err.Error(), "duplicate") {
+			t.Fatalf("duplicate run directory error = %v, want duplicate diagnostic", err)
+		}
+	})
+
+	t.Run("record.json", func(t *testing.T) {
+		path := filepath.Join(t.TempDir(), "runs", "dir-id", "record.json")
+		writeArtifact(t, path, record.RunRecord{RunID: "payload-id"})
+		if _, err := decodePersistedRecords([]string{path}); err == nil || !strings.Contains(err.Error(), "dir-id") {
+			t.Fatalf("mismatched record identity error = %v, want directory identity diagnostic", err)
+		}
+		writeArtifact(t, path, record.RunRecord{RunID: "dir-id"})
+		if _, err := decodePersistedRecords([]string{path, path}); err == nil || !strings.Contains(err.Error(), "duplicate") {
+			t.Fatalf("duplicate record directory error = %v, want duplicate diagnostic", err)
+		}
+	})
+}
+
+type scheduleTrace struct {
+	mu     sync.Mutex
+	events []string
+}
+
+func (t *scheduleTrace) add(event string) {
+	t.mu.Lock()
+	defer t.mu.Unlock()
+	t.events = append(t.events, event)
+}
+
+func (t *scheduleTrace) snapshot() []string {
+	t.mu.Lock()
+	defer t.mu.Unlock()
+	return append([]string(nil), t.events...)
+}
+
+type scheduleTraceExec struct {
+	in    foldFakeExec
+	trace *scheduleTrace
+}
+
+func (e scheduleTraceExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+	result, err := e.in.Execute(step, runDir)
+	if err == nil && step.ID == "train" {
+		e.trace.add("train-complete:" + runDir)
+	}
+	return result, err
+}
+
+func TestSweep_ColdAdmissionCompletesTrainBeforeFifthAcquire(t *testing.T) {
+	ws := t.TempDir()
+	body := strings.Replace(foldShapeMD("[a]"), "k: 2", "k: 6", 1)
+	expPath := writeShapeFile(t, ws, body)
+	control := newRunControl(2)
+	trace := &scheduleTrace{}
+	control.afterAcquire = func(label string) { trace.add("acquire:" + label) }
+
+	result := make(chan error, 1)
+	go func() {
+		_, err := runExperiment(runOpts{
+			expPath: expPath, now: fixedNow(),
+			git: fakeGitProbe{name: "metis", sha: "sha"}, cache: false,
+			exec: scheduleTraceExec{in: foldFakeExec{}, trace: trace}, out: io.Discard,
+			maxParallel: 2, runControl: control,
+		})
+		result <- err
+	}()
+	if err := awaitRunControl(t, result, "flat k=6 cold sweep"); err != nil {
+		t.Fatal(err)
+	}
+
+	events := trace.snapshot()
+	firstTrain, fifthAcquire := -1, -1
+	acquires := 0
+	for i, event := range events {
+		switch {
+		case strings.HasPrefix(event, "train-complete:") && firstTrain < 0:
+			firstTrain = i
+		case strings.HasPrefix(event, "acquire:"):
+			acquires++
+			if acquires == 5 {
+				fifthAcquire = i
+			}
+		}
+	}
+	if firstTrain < 0 || fifthAcquire < 0 {
+		t.Fatalf("trace missing first train completion or fifth acquire: %v", events)
+	}
+	if firstTrain >= fifthAcquire {
+		t.Fatalf("cold wave acquired five runs before completing a train: %v", events)
+	}
 }

 // peakExec wraps foldFakeExec, acquiring the SHARED leaf semaphore around each step
@@ -97,29 +299,56 @@ func (p peakExec) Execute(step experiment.Step, runDir string) (experiment.StepR
 // the run must complete (no deadlock). The fake leaf acquires the SAME sem the
 // production execStep would.
 func TestNestedCV_PeakConcurrencyWithinCap(t *testing.T) {
-	const cap = 3
+	const cap = 2 // runControl admits 2n=4 concrete runs; the nested fan-out has >4 children
	ws := t.TempDir()
	// 3 configs → nested (outer folds = sweeper.cv.k = 2) × 2 inner folds → deep nesting, ~many leaf calls.
	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
	sem := make(chan struct{}, cap)
	var mu sync.Mutex
	var cur, peak int
+	var activeRuns, peakRuns, acquiredRuns, releasedRuns int
+	control := newRunControl(cap)
+	control.afterAcquire = func(string) {
+		mu.Lock()
+		defer mu.Unlock()
+		activeRuns++
+		acquiredRuns++
+		if activeRuns > peakRuns {
+			peakRuns = activeRuns
+		}
+	}
+	control.beforeRelease = func(string) {
+		mu.Lock()
+		defer mu.Unlock()
+		activeRuns--
+		releasedRuns++
+	}
	pe := peakExec{sem: sem, mu: &mu, cur: &cur, peak: &peak, in: foldFakeExec{}}
-	_, err := runExperiment(runOpts{
-		expPath:     expPath,
-		now:         fixedNow(),
-		git:         fakeGitProbe{name: "metis", sha: "sha"},
-		cache:       false, // every step runs → maximum fan-out against the cap
-		exec:        pe,
-		out:         io.Discard,
-		maxParallel: cap,
-		leafSem:     sem, // runExperiment reuses my sem (maxParallel>1 & non-nil)
-	})
+	result := make(chan error, 1)
+	go func() {
+		_, err := runExperiment(runOpts{
+			expPath:     expPath,
+			now:         fixedNow(),
+			git:         fakeGitProbe{name: "metis", sha: "sha"},
+			cache:       false, // every step runs → maximum fan-out against the cap
+			exec:        pe,
+			out:         io.Discard,
+			maxParallel: cap,
+			leafSem:     sem, // runExperiment reuses my sem (maxParallel>1 & non-nil)
+			runControl:  control,
+		})
+		result <- err
+	}()
+	err := awaitRunControl(t, result, "nested run with more children than admission capacity")
	if err != nil {
		t.Fatalf("driver:cv run must complete (no deadlock), got: %v", err)
	}
	mu.Lock()
	got := peak
+	gotActiveRuns := activeRuns
+	gotPeakRuns := peakRuns
+	gotAcquiredRuns := acquiredRuns
+	gotReleasedRuns := releasedRuns
	mu.Unlock()
	if got > cap {
		t.Fatalf("peak concurrency %d exceeded the global cap %d — the leaf budget leaked across nesting", got, cap)
@@ -127,6 +356,21 @@ func TestNestedCV_PeakConcurrencyWithinCap(t *testing.T) {
	if got < 2 {
		t.Fatalf("peak concurrency %d — the fan-out never overlapped, so the test can't prove the cap actually holds", got)
	}
+	if gotPeakRuns > 2*cap {
+		t.Fatalf("peak admitted runs %d exceeded controller cap %d", gotPeakRuns, 2*cap)
+	}
+	if gotPeakRuns == 0 {
+		t.Fatal("injected controller observed no admitted runs")
+	}
+	if gotAcquiredRuns <= 2*cap {
+		t.Fatalf("controller observed %d concrete run attempts, want more than its %d-slot capacity", gotAcquiredRuns, 2*cap)
+	}
+	if gotActiveRuns != 0 {
+		t.Fatalf("active admitted runs after completion = %d, want 0", gotActiveRuns)
+	}
+	if gotAcquiredRuns != gotReleasedRuns {
+		t.Fatalf("admission hooks acquired=%d released=%d, want equal", gotAcquiredRuns, gotReleasedRuns)
+	}
 }

 // sleepExec is foldFakeExec with a fixed per-step delay, so a sweep has real
@@ -137,6 +381,152 @@ type sleepExec struct {
	d  time.Duration
 }

+type concurrentBuffer struct {
+	mu sync.Mutex
+	b  bytes.Buffer
+}
+
+func (b *concurrentBuffer) Write(p []byte) (int, error) {
+	b.mu.Lock()
+	defer b.mu.Unlock()
+	return b.b.Write(p)
+}
+
+func (b *concurrentBuffer) snapshot() string {
+	b.mu.Lock()
+	defer b.mu.Unlock()
+	return b.b.String()
+}
+
+func (b *concurrentBuffer) len() int {
+	b.mu.Lock()
+	defer b.mu.Unlock()
+	return b.b.Len()
+}
+
+// failureBarrierExec holds the first four admitted inner folds at their train
+// step. Exactly one returns the concrete injected failure; its admitted siblings
+// wait for controller publication and then return the cancellation sentinel.
+type failureBarrierExec struct {
+	in               foldFakeExec
+	mu               sync.Mutex
+	innerDirs        map[string]struct{}
+	innerEntered     chan string
+	fourEntered      chan struct{}
+	releaseFailure   chan struct{}
+	failurePublished chan struct{}
+	winner           atomic.Bool
+}
+
+func newFailureBarrierExec() *failureBarrierExec {
+	return &failureBarrierExec{
+		innerDirs:        make(map[string]struct{}),
+		innerEntered:     make(chan string, 8),
+		fourEntered:      make(chan struct{}),
+		releaseFailure:   make(chan struct{}),
+		failurePublished: make(chan struct{}),
+	}
+}
+
+func (f *failureBarrierExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+	if step.ID == partitionStepID {
+		f.mu.Lock()
+		if _, seen := f.innerDirs[runDir]; !seen {
+			f.innerDirs[runDir] = struct{}{}
+			f.innerEntered <- runDir
+			if len(f.innerDirs) == 4 {
+				close(f.fourEntered)
+			}
+		}
+		f.mu.Unlock()
+		if err := waitFailureBarrier(f.fourEntered, "four inner runs to enter"); err != nil {
+			return experiment.StepResult{}, err
+		}
+	}
+	if step.ID == "train" {
+		if f.winner.CompareAndSwap(false, true) {
+			if err := waitFailureBarrier(f.releaseFailure, "failing train release"); err != nil {
+				return experiment.StepResult{}, err
+			}
+			return experiment.StepResult{}, errors.New("injected train failure")
+		}
+		if err := waitFailureBarrier(f.failurePublished, "controller failure publication"); err != nil {
+			return experiment.StepResult{}, err
+		}
+		return experiment.StepResult{}, errRunAborted
+	}
+	return f.in.Execute(step, runDir)
+}
+
+func waitFailureBarrier(ch <-chan struct{}, what string) error {
+	// Fire before the enclosing test's 2s await so a broken orchestration returns
+	// the specific barrier error instead of only the outer generic timeout.
+	timer := time.NewTimer(runControlTestTimeout / 2)
+	defer timer.Stop()
+	select {
+	case <-ch:
+		return nil
+	case <-timer.C:
+		return fmt.Errorf("failure barrier timed out waiting for %s", what)
+	}
+}
+
+func (f *failureBarrierExec) dirCount() int {
+	f.mu.Lock()
+	defer f.mu.Unlock()
+	return len(f.innerDirs)
+}
+
+func TestNestedCV_FirstFailureStopsAllObservableWork(t *testing.T) {
+	ws := t.TempDir()
+	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
+	control := newRunControl(2)
+	exec := newFailureBarrierExec()
+	out := &concurrentBuffer{}
+	publishedOffset := make(chan int, 1)
+	control.beforeFailureUnlock = func() {
+		publishedOffset <- out.len()
+		close(exec.failurePublished)
+	}
+	result := make(chan error, 1)
+	go func() {
+		_, err := runExperiment(runOpts{
+			expPath: expPath, now: fixedNow(),
+			git: fakeGitProbe{name: "metis", sha: "sha"}, exec: exec, out: out,
+			maxParallel: 2, runControl: control,
+		})
+		result <- err
+	}()
+
+	for i := 0; i < 4; i++ {
+		awaitRunControl(t, exec.innerEntered, "four admitted inner run directories")
+	}
+	close(exec.releaseFailure)
+	offset := awaitRunControl(t, publishedOffset, "first failure publication")
+	err := awaitRunControl(t, result, "nested failure to return without parent/child admission deadlock")
+	if err == nil || !strings.Contains(err.Error(), "config ") || !strings.Contains(err.Error(), "injected train failure") {
+		t.Fatalf("error = %v, want contextual concrete config/fold failure", err)
+	}
+	if errors.Is(err, errRunAborted) || strings.Contains(err.Error(), errRunAborted.Error()) {
+		t.Fatalf("top-level error exposed cancellation sentinel instead of cause: %v", err)
+	}
+	if got := exec.dirCount(); got != 4 {
+		t.Fatalf("inner run dirs = %d, want exactly four admitted dirs and no fifth start", got)
+	}
+	suffix := out.snapshot()[offset:]
+	for _, forbidden := range []string{"metis: progress", "folds/min", "ETA", "score ", "estimate", "mean "} {
+		if strings.Contains(suffix, forbidden) {
+			t.Errorf("post-failure output contains %q:\n%s", forbidden, suffix)
+		}
+	}
+	if matches, _ := filepath.Glob(filepath.Join(ws, "sweeps", "*", "manifest.json")); len(matches) != 0 {
+		t.Fatalf("failure persisted %d manifest(s), want none", len(matches))
+	}
+	if _, statErr := os.Stat(filepath.Join(ws, "shape.ledger.csv")); !errors.Is(statErr, os.ErrNotExist) {
+		t.Fatalf("failure persisted a ledger: %v", statErr)
+	}
+}
+
 func (s sleepExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	res, err := s.in.Execute(step, runDir)
	time.Sleep(s.d)
diff --git a/cmd/metis/progress.go b/cmd/metis/progress.go
index 19d5c7d..71ee241 100644
--- a/cmd/metis/progress.go
+++ b/cmd/metis/progress.go
@@ -205,8 +205,8 @@ type boardState struct {

 // sweepProgress is the mutex'd sink shared by every pass of one shape-run. Events
 // arrive concurrently (ParExec goroutines across sibling outer folds, each holding
-// its own Run's event mutex); lock order is strictly Run-mu → sink-mu → the
-// syncWriter under `out` — acyclic. Emit policy: fold/config events are throttled
+// its own Run's event mutex); health-gated paths use the strict order runControl.mu
+// → sink.mu → boardWriter.mu (never the reverse). Emit policy: fold/config events are throttled
 // to one line per second (injected clock — tests script it, never sleep); a
 // driver-level (outer fold) completion ALWAYS emits; finish() emits the terminal
 // line. A nil *sweepProgress is a no-op everywhere (the non-sweep path is silent).
@@ -355,6 +355,17 @@ func (sp *sweepProgress) tick() {
	bw.forceFlush() // metis#46: the tick is what re-pins the board after a burst window
 }

+// abort removes the stored live frame after a sweep failure. Lock order remains
+// progress -> board; the controller is never called while either lock is held.
+func (sp *sweepProgress) abort() {
+	if sp == nil || sp.bw == nil {
+		return
+	}
+	sp.mu.Lock()
+	defer sp.mu.Unlock()
+	sp.bw.discardFrame()
+}
+
 // maybeEmit writes the line if forced (driver/finish) or the throttle elapsed —
 // 1s for plain lines (a log is a record), 100ms for board repaints (a board is a
 // display; the 500ms ticker guarantees freshness anyway). Caller holds sp.mu.
diff --git a/cmd/metis/run.go b/cmd/metis/run.go
index 8323bc4..c71ede6 100644
--- a/cmd/metis/run.go
+++ b/cmd/metis/run.go
@@ -6,6 +6,7 @@ import (
	"io"
	"os"
	"path/filepath"
+	"strings"
	"sync"
	"time"

@@ -81,14 +82,22 @@ type runOpts struct {
	readRoot    string        // metis#23: when set, the production execStep confines base-dataset reads to this root
	maxParallel int           // metis#31: >1 ⇒ ParExec batches + a leaf semaphore; sizes leafSem
	leafSem     chan struct{} // metis#31: the shared global subprocess budget (nil = serial/cache-only)
+	runControl  *runControl   // one per shape run: global abort + optional 2n admission slots
+	runLabel    string        // config/fold/preamble context captured with the first error
	forkserver  bool          // metis#44: warm fork-server leaf executor (cmdRun default true;
	//                           zero-value false keeps direct runOpts callers/tests on legacy exec)
	forkPool *serverPool // metis#44: the per-root warm-server pool, created once per runExperiment
	//                      when forkserver is set; threaded through nested runOpts copies.
	tui bool // metis#38: stdout is a TTY and --no-tui wasn't passed — a SWEEP pins the live board
	//          (a plain experiment ignores it; non-TTY/piped runs stay on the #30 plain lines)
-	board     *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
-	leafGauge func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
+	board           *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
+	boardTick       <-chan time.Time  // test seam: nil uses the production 500ms ticker
+	beforeBoardTick func()            // test seam: after tick selection, before health observation
+	afterBoardTick  func()            // test seam: after the health observation returns
+	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
+	leafPins        []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
+	//                             runExperiment (nil = not yet computed; non-nil rides nested runOpts
+	//                             copies like forkPool — an all-suppressed result is empty, not nil)
 }

 // runExperiment reads the experiment at o.expPath and dispatches: a `type:
@@ -145,11 +154,24 @@ func runExperiment(o runOpts) (experiment.Run, error) {
		out = &syncWriter{w: out}
		o.out = out
	}
+	// metis#48: default leaf BLAS pins — computed ONCE per top-level run from the ambient
+	// env (an exported operator value wins by exclusion in blasPins), announced loudly,
+	// and injected at both spawn seams (legacy execStep child env; fork-server process
+	// env). Fake-exec runs spawn nothing (no pins, no note); dry-run lists configs (same).
+	// Placed AFTER the writer wrap (the note must route through the board) and BEFORE the
+	// pool creation (the server spawn env needs the pins).
+	if o.exec == nil && !o.dryRun && o.leafPins == nil {
+		o.leafPins = blasPins(os.Environ())
+		if len(o.leafPins) > 0 {
+			fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
+				strings.Join(o.leafPins, " "))
+		}
+	}
	// metis#44: one warm fork-server pool per top-level run, shut down (EOF-drain) when the
	// run ends. Only the production executor uses it (an injected test exec bypasses execStep).
	// Constructed AFTER the writer wrap — its fallback notices must route through the board.
	if o.forkserver && o.exec == nil && o.forkPool == nil {
-		o.forkPool = newServerPool(out)
+		o.forkPool = newServerPool(out, o.leafPins)
		defer o.forkPool.shutdown()
	}
	if exp.Type == "experiment-shape" {
@@ -160,6 +182,9 @@ func runExperiment(o runOpts) (experiment.Run, error) {
		if err := experiment.ValidateShape(sh); err != nil {
			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
		}
+		if o.runControl == nil {
+			o.runControl = newRunControl(o.maxParallel)
+		}
		return experiment.Run{}, runShapeSweep(o, sh, now, out)
	}
	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
@@ -188,6 +213,15 @@ func singleRunID(o runOpts, exp experiment.Experiment, now func() time.Time) str
 // both the 1-point path and the sweep loop (metis#7) call — so the run/cache/record wiring
 // lives in ONE place (ARCH-DRY).
 func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
+	if o.runControl == nil {
+		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
+	}
+	return o.runControl.run(o.runLabel, func() (experiment.Run, error) {
+		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
+	})
+}
+
+func runResolvedExperimentAdmitted(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
	baseDir := filepath.Dir(o.expPath)
	// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
	// the child's env, and the child's cwd IS the step dir — a relative path would
@@ -202,7 +236,7 @@ func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, n
		return experiment.Run{}, err
	}

-	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool}
+	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool, pins: o.leafPins}
	if o.exec != nil {
		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
	}
diff --git a/cmd/metis/runcontrol.go b/cmd/metis/runcontrol.go
new file mode 100644
index 0000000..f1201fd
--- /dev/null
+++ b/cmd/metis/runcontrol.go
@@ -0,0 +1,104 @@
+package main
+
+import (
+	"errors"
+	"fmt"
+	"sync"
+
+	"github.com/xianxu/metis/pkg/experiment"
+)
+
+var errRunAborted = errors.New("run aborted after earlier sweep failure")
+
+// runControl bounds admitted concrete runs independently of leaf subprocess
+// parallelism and latches the first whole-run failure. Observation callbacks
+// must not call back into the controller or block production work.
+type runControl struct {
+	slots chan struct{}
+
+	mu  sync.Mutex
+	err error
+
+	beforeFailureLock   func()
+	beforeFailureUnlock func()
+	afterAcquire        func(label string)
+	beforeRelease       func(label string)
+}
+
+func newRunControl(maxParallel int) *runControl {
+	control := &runControl{}
+	if maxParallel > 1 {
+		control.slots = make(chan struct{}, 2*maxParallel)
+	}
+	return control
+}
+
+func (c *runControl) firstError() error {
+	c.mu.Lock()
+	defer c.mu.Unlock()
+	return c.err
+}
+
+// whileHealthy linearizes an observable transition against first-failure
+// publication. The callback runs while c.mu is held and therefore must not call
+// back into runControl. Downstream locks are acquired only inside fn, preserving
+// the global control -> progress/pass/manifest order.
+func (c *runControl) whileHealthy(fn func()) bool {
+	c.mu.Lock()
+	defer c.mu.Unlock()
+	if c.err != nil {
+		return false
+	}
+	fn()
+	return true
+}
+
+func (c *runControl) fail(label string, err error) error {
+	if err == nil {
+		return nil
+	}
+	contextual := err
+	if label != "" {
+		contextual = fmt.Errorf("%s: %w", label, err)
+	}
+	if c.beforeFailureLock != nil {
+		c.beforeFailureLock()
+	}
+
+	c.mu.Lock()
+	if c.err == nil {
+		c.err = contextual
+		if c.beforeFailureUnlock != nil {
+			c.beforeFailureUnlock()
+		}
+	}
+	authoritative := c.err
+	c.mu.Unlock()
+	return authoritative
+}
+
+func (c *runControl) run(label string, fn func() (experiment.Run, error)) (experiment.Run, error) {
+	if c.slots != nil {
+		c.slots <- struct{}{}
+		defer func() { <-c.slots }()
+		if c.afterAcquire != nil {
+			c.afterAcquire(label)
+		}
+		if c.beforeRelease != nil {
+			defer func() { c.beforeRelease(label) }()
+		}
+	}
+
+	if c.firstError() != nil {
+		return experiment.Run{}, errRunAborted
+	}
+
+	run, err := fn()
+	if err != nil {
+		return experiment.Run{}, c.fail(label, err)
+	}
+	if c.firstError() != nil {
+		return experiment.Run{}, errRunAborted
+	}
+	return run, nil
+}
diff --git a/cmd/metis/runcontrol_test.go b/cmd/metis/runcontrol_test.go
new file mode 100644
index 0000000..04c9e8a
--- /dev/null
+++ b/cmd/metis/runcontrol_test.go
@@ -0,0 +1,508 @@
+package main
+
+import (
+	"bytes"
+	"errors"
+	"os"
+	"path/filepath"
+	"reflect"
+	"runtime"
+	"strings"
+	"sync/atomic"
+	"testing"
+	"time"
+
+	"github.com/xianxu/metis/pkg/experiment"
+)
+
+func TestRunResolvedExperiment_AbortedBeforeSideEffects(t *testing.T) {
+	ws := t.TempDir()
+	control := newRunControl(2)
+	control.fail("earlier fold", errors.New("failed"))
+	var out bytes.Buffer
+	exp := experiment.Experiment{Header: experiment.Header{Type: "experiment", ID: "queued"}}
+
+	_, err := runResolvedExperiment(exp, runOpts{
+		expPath:    filepath.Join(ws, "shape.md"),
+		runControl: control,
+		runLabel:   "queued fold",
+		cache:      true,
+	}, "queued", fixedNow(), &out)
+	if !errors.Is(err, errRunAborted) {
+		t.Fatalf("error = %v, want errRunAborted", err)
+	}
+	if out.Len() != 0 {
+		t.Fatalf("aborted run wrote output: %q", out.String())
+	}
+	if _, statErr := os.Stat(filepath.Join(ws, "runs", "queued")); !errors.Is(statErr, os.ErrNotExist) {
+		t.Fatalf("queued run created state: %v", statErr)
+	}
+	if _, statErr := os.Stat(filepath.Join(ws, ".metis-cache")); !errors.Is(statErr, os.ErrNotExist) {
+		t.Fatalf("queued run initialized cache state: %v", statErr)
+	}
+}
+
+const runControlTestTimeout = 2 * time.Second
+
+type runControlResult struct {
+	run experiment.Run
+	err error
+}
+
+func isZeroRun(run experiment.Run) bool {
+	return reflect.DeepEqual(run, experiment.Run{})
+}
+
+func awaitRunControl[T any](t *testing.T, ch <-chan T, what string) T {
+	t.Helper()
+	select {
+	case got := <-ch:
+		return got
+	case <-time.After(runControlTestTimeout):
+		t.Fatalf("timed out waiting for %s", what)
+		var zero T
+		return zero
+	}
+}
+
+func TestRunControlBoundsAdmissionAtTwiceParallelism(t *testing.T) {
+	control := newRunControl(3)
+	entered := make(chan struct{}, 12)
+	release := make(chan struct{})
+	results := make(chan runControlResult, 12)
+	var active atomic.Int32
+	var peak atomic.Int32
+	var acquired atomic.Int32
+	var released atomic.Int32
+	control.afterAcquire = func(string) { acquired.Add(1) }
+	control.beforeRelease = func(string) { released.Add(1) }
+
+	for range 12 {
+		go func() {
+			run, err := control.run("point", func() (experiment.Run, error) {
+				current := active.Add(1)
+				for old := peak.Load(); current > old && !peak.CompareAndSwap(old, current); old = peak.Load() {
+				}
+				entered <- struct{}{}
+				<-release
+				active.Add(-1)
+				return experiment.Run{ID: "ok"}, nil
+			})
+			results <- runControlResult{run: run, err: err}
+		}()
+	}
+
+	for i := 0; i < 6; i++ {
+		awaitRunControl(t, entered, "six admitted callbacks")
+	}
+	if got := len(control.slots); got != 6 {
+		t.Fatalf("admitted slots = %d, want 6", got)
+	}
+	select {
+	case <-entered:
+		t.Fatal("more than six callbacks entered before an admission slot was released")
+	default:
+	}
+	close(release)
+
+	for i := 0; i < 12; i++ {
+		got := awaitRunControl(t, results, "all bounded runs to drain")
+		if got.err != nil || got.run.ID != "ok" {
+			t.Fatalf("run result = (%+v, %v), want successful run", got.run, got.err)
+		}
+	}
+	if got := peak.Load(); got != 6 {
+		t.Fatalf("peak callbacks = %d, want exactly 6", got)
+	}
+	if got := acquired.Load(); got != 12 {
+		t.Fatalf("acquire hook calls = %d, want 12 attempted runs", got)
+	}
+	if got := released.Load(); got != 12 {
+		t.Fatalf("release hook calls = %d, want 12 attempted runs", got)
+	}
+}
+
+func TestRunControlHookPanicsStillReleaseAdmission(t *testing.T) {
+	panicValue := errors.New("observation hook panic")
+	for _, tc := range []struct {
+		name string
+		set  func(*runControl)
+	}{
+		{
+			name: "after acquire",
+			set: func(control *runControl) {
+				control.afterAcquire = func(string) { panic(panicValue) }
+			},
+		},
+		{
+			name: "before release",
+			set: func(control *runControl) {
+				control.beforeRelease = func(string) { panic(panicValue) }
+			},
+		},
+	} {
+		t.Run(tc.name, func(t *testing.T) {
+			control := &runControl{slots: make(chan struct{}, 1)}
+			tc.set(control)
+			got := recoverRunControlPanic(func() {
+				_, _ = control.run("observed", func() (experiment.Run, error) {
+					return experiment.Run{ID: "ok"}, nil
+				})
+			})
+			if got != panicValue {
+				t.Fatalf("recovered panic = %v, want exact hook panic %v", got, panicValue)
+			}
+			if got := len(control.slots); got != 0 {
+				t.Fatalf("admission slots after recovered hook panic = %d, want 0", got)
+			}
+		})
+	}
+}
+
+func recoverRunControlPanic(fn func()) (recovered any) {
+	defer func() { recovered = recover() }()
+	fn()
+	return nil
+}
+
+func TestRunControlPublishesFailureBeforeAdmissionRelease(t *testing.T) {
+	control := &runControl{slots: make(chan struct{}, 1)}
+	published := make(chan struct{})
+	letTokenRelease := make(chan struct{})
+	control.beforeFailureUnlock = func() {
+		close(published)
+		<-letTokenRelease
+	}
+
+	firstResult := make(chan runControlResult, 1)
+	go func() {
+		run, err := control.run("first", func() (experiment.Run, error) {
+			return experiment.Run{ID: "must-be-discarded"}, errors.New("boom")
+		})
+		firstResult <- runControlResult{run: run, err: err}
+	}()
+	awaitRunControl(t, published, "failure publication hook")
+
+	var secondCalled atomic.Bool
+	secondResult := make(chan runControlResult, 1)
+	go func() {
+		run, err := control.run("second", func() (experiment.Run, error) {
+			secondCalled.Store(true)
+			return experiment.Run{ID: "must-not-run"}, nil
+		})
+		secondResult <- runControlResult{run: run, err: err}
+	}()
+
+	if got := len(control.slots); got != 1 {
+		t.Fatalf("slots while failure publisher holds the mutex = %d, want 1", got)
+	}
+	close(letTokenRelease)
+
+	first := awaitRunControl(t, firstResult, "first failed run")
+	second := awaitRunControl(t, secondResult, "second aborted run")
+	if !isZeroRun(first.run) {
+		t.Fatalf("failed run = %+v, want zero Run", first.run)
+	}
+	if first.err == nil || first.err.Error() != "first: boom" {
+		t.Fatalf("first error = %v, want contextual first failure", first.err)
+	}
+	if !isZeroRun(second.run) || !errors.Is(second.err, errRunAborted) {
+		t.Fatalf("second result = (%+v, %v), want zero Run and errRunAborted", second.run, second.err)
+	}
+	if secondCalled.Load() {
+		t.Fatal("second callback executed after failure publication")
+	}
+	if got := control.firstError(); got == nil || got.Error() != "first: boom" {
+		t.Fatalf("stored first error = %v, want first: boom", got)
+	}
+}
+
+func TestRunControlAcquiresAdmissionBeforeCheckingFailure(t *testing.T) {
+	control := &runControl{slots: make(chan struct{}, 1)}
+	control.slots <- struct{}{}
+	prior := errors.New("prior failure")
+	var callbackCalled atomic.Bool
+	result := make(chan runControlResult, 1)
+
+	control.mu.Lock()
+	go func() {
+		run, err := control.run("later", func() (experiment.Run, error) {
+			callbackCalled.Store(true)
+			return experiment.Run{ID: "must-not-run"}, nil
+		})
+		result <- runControlResult{run: run, err: err}
+	}()
+
+	// Make one admission slot available while firstError remains blocked on mu.
+	// A correctly ordered run refills the slot before attempting the error check.
+	<-control.slots
+	timer := time.NewTimer(runControlTestTimeout)
+	defer timer.Stop()
+	for len(control.slots) != 1 {
+		select {
+		case <-timer.C:
+			control.err = prior
+			control.mu.Unlock()
+			t.Fatal("run did not acquire admission before attempting the failure check")
+		default:
+			runtime.Gosched()
+		}
+	}
+	control.err = prior
+	control.mu.Unlock()
+
+	got := awaitRunControl(t, result, "admitted run to observe prior failure")
+	if !isZeroRun(got.run) || !errors.Is(got.err, errRunAborted) {
+		t.Fatalf("run result = (%+v, %v), want zero Run and errRunAborted", got.run, got.err)
+	}
+	if callbackCalled.Load() {
+		t.Fatal("callback executed despite failure installed before the post-admission check")
+	}
+	if got := len(control.slots); got != 0 {
+		t.Fatalf("slots after aborted run = %d, want released", got)
+	}
+}
+
+func TestRunControlSerialStillLatchesFailure(t *testing.T) {
+	control := newRunControl(1)
+	if control.slots != nil {
+		t.Fatal("serial controller unexpectedly allocated admission slots")
+	}
+	if got := control.fail("ignored", nil); got != nil || control.firstError() != nil {
+		t.Fatalf("nil failure = %v with stored error %v, want neither", got, control.firstError())
+	}
+
+	failed, err := control.run("serial", func() (experiment.Run, error) {
+		return experiment.Run{ID: "must-be-discarded"}, errors.New("broken")
+	})
+	if !isZeroRun(failed) || err == nil || err.Error() != "serial: broken" {
+		t.Fatalf("failed result = (%+v, %v), want zero Run and contextual error", failed, err)
+	}
+
+	called := false
+	aborted, err := control.run("later", func() (experiment.Run, error) {
+		called = true
+		return experiment.Run{ID: "must-not-run"}, nil
+	})
+	if !isZeroRun(aborted) || !errors.Is(err, errRunAborted) {
+		t.Fatalf("later result = (%+v, %v), want zero Run and errRunAborted", aborted, err)
+	}
+	if called {
+		t.Fatal("later serial callback executed after failure")
+	}
+	if got := control.firstError(); got == nil || got.Error() != "serial: broken" {
+		t.Fatalf("stored first error = %v, want serial: broken", got)
+	}
+}
+
+func TestRunControlFailureWithoutLabelPreservesError(t *testing.T) {
+	control := newRunControl(1)
+	cause := errors.New("unlabeled failure")
+
+	got := control.fail("", cause)
+	if got != cause {
+		t.Fatalf("unlabeled failure = %v (%p), want original error %v (%p)", got, got, cause, cause)
+	}
+	if stored := control.firstError(); stored != cause {
+		t.Fatalf("stored unlabeled failure = %v (%p), want original error %v (%p)", stored, stored, cause, cause)
+	}
+}
+
+func TestRunControlConcurrentFailuresKeepOneContextualCause(t *testing.T) {
+	control := newRunControl(2)
+	entered := make(chan struct{}, 2)
+	release := make(chan struct{})
+	results := make(chan runControlResult, 2)
+
+	for _, tc := range []struct {
+		label string
+		err   string
+	}{{label: "left", err: "left failed"}, {label: "right", err: "right failed"}} {
+		tc := tc
+		go func() {
+			run, err := control.run(tc.label, func() (experiment.Run, error) {
+				entered <- struct{}{}
+				<-release
+				return experiment.Run{ID: tc.label}, errors.New(tc.err)
+			})
+			results <- runControlResult{run: run, err: err}
+		}()
+	}
+	awaitRunControl(t, entered, "left failure callback")
+	awaitRunControl(t, entered, "right failure callback")
+	close(release)
+
+	first := awaitRunControl(t, results, "first concurrent failure")
+	second := awaitRunControl(t, results, "second concurrent failure")
+	if !isZeroRun(first.run) || !isZeroRun(second.run) {
+		t.Fatalf("failed runs = (%+v, %+v), want zero Runs", first.run, second.run)
+	}
+	if first.err == nil || second.err == nil || first.err.Error() != second.err.Error() {
+		t.Fatalf("concurrent errors = (%v, %v), want one authoritative error", first.err, second.err)
+	}
+	if got := first.err.Error(); got != "left: left failed" && got != "right: right failed" {
+		t.Fatalf("authoritative error = %q, want one contextual cause", got)
+	}
+	if got := control.firstError(); got == nil || got.Error() != first.err.Error() {
+		t.Fatalf("stored first error = %v, want %v", got, first.err)
+	}
+}
+
+func TestRunControlWinnerHookRunsOnceInsideFailureMutex(t *testing.T) {
+	control := newRunControl(2)
+	callbacksEntered := make(chan struct{}, 2)
+	releaseLeft := make(chan struct{})
+	releaseRight := make(chan struct{})
+	hookEntered := make(chan struct{}, 1)
+	releaseWinner := make(chan struct{})
+	results := make(chan runControlResult, 2)
+	var hookCalls atomic.Int32
+	control.beforeFailureUnlock = func() {
+		hookCalls.Add(1)
+		hookEntered <- struct{}{}
+		<-releaseWinner
+	}
+
+	for _, failure := range []struct {
+		label   string
+		release <-chan struct{}
+	}{{label: "left", release: releaseLeft}, {label: "right", release: releaseRight}} {
+		failure := failure
+		go func() {
+			run, err := control.run(failure.label, func() (experiment.Run, error) {
+				callbacksEntered <- struct{}{}
+				<-failure.release
+				return experiment.Run{}, errors.New("failed")
+			})
+			results <- runControlResult{run: run, err: err}
+		}()
+	}
+	awaitRunControl(t, callbacksEntered, "first failing callback")
+	awaitRunControl(t, callbacksEntered, "second failing callback")
+	close(releaseLeft)
+	awaitRunControl(t, hookEntered, "winner failure hook")
+
+	hookHeldMutex := !control.mu.TryLock()
+	if !hookHeldMutex {
+		control.mu.Unlock()
+	}
+	lookupStarted := make(chan struct{})
+	lookupResult := make(chan error, 1)
+	go func() {
+		close(lookupStarted)
+		lookupResult <- control.firstError()
+	}()
+	awaitRunControl(t, lookupStarted, "firstError lookup to start")
+	runtime.Gosched()
+	lookupReturnedEarly := false
+	var stored error
+	select {
+	case stored = <-lookupResult:
+		lookupReturnedEarly = true
+	default:
+	}
+
+	close(releaseRight)
+	close(releaseWinner)
+	first := awaitRunControl(t, results, "first concurrent failure result")
+	second := awaitRunControl(t, results, "second concurrent failure result")
+	if !lookupReturnedEarly {
+		stored = awaitRunControl(t, lookupResult, "blocked firstError lookup")
+	}
+
+	if !hookHeldMutex {
+		t.Fatal("winner hook ran outside the failure mutex")
+	}
+	if lookupReturnedEarly {
+		t.Fatal("firstError returned while winner hook was blocked")
+	}
+	if got := hookCalls.Load(); got != 1 {
+		t.Fatalf("winner hook calls = %d, want exactly 1", got)
+	}
+	if first.err == nil || second.err == nil || first.err.Error() != second.err.Error() {
+		t.Fatalf("concurrent failures = (%v, %v), want one authoritative error", first.err, second.err)
+	}
+	if stored == nil || stored.Error() != first.err.Error() {
+		t.Fatalf("stored first error = %v, want %v", stored, first.err)
+	}
+}
+
+func TestRunControlDiscardsLateSuccessAfterSiblingFailure(t *testing.T) {
+	control := newRunControl(2)
+	successEntered := make(chan struct{})
+	failurePublished := make(chan struct{})
+	control.beforeFailureUnlock = func() { close(failurePublished) }
+
+	successResult := make(chan runControlResult, 1)
+	go func() {
+		run, err := control.run("slow success", func() (experiment.Run, error) {
+			close(successEntered)
+			<-failurePublished
+			return experiment.Run{ID: "late"}, nil
+		})
+		successResult <- runControlResult{run: run, err: err}
+	}()
+	awaitRunControl(t, successEntered, "successful callback to enter")
+
+	failureResult := make(chan runControlResult, 1)
+	go func() {
+		run, err := control.run("sibling", func() (experiment.Run, error) {
+			return experiment.Run{}, errors.New("failed")
+		})
+		failureResult <- runControlResult{run: run, err: err}
+	}()
+
+	failure := awaitRunControl(t, failureResult, "sibling failure")
+	success := awaitRunControl(t, successResult, "late success")
+	if failure.err == nil || failure.err.Error() != "sibling: failed" {
+		t.Fatalf("failure error = %v, want sibling: failed", failure.err)
+	}
+	if !isZeroRun(success.run) || !errors.Is(success.err, errRunAborted) {
+		t.Fatalf("late success = (%+v, %v), want discarded Run and errRunAborted", success.run, success.err)
+	}
+	if strings.Contains(success.err.Error(), "sibling") {
+		t.Fatalf("late success exposed sibling cause instead of abort sentinel: %v", success.err)
+	}
+}
+
+func TestRunControlObservationLinearizesBeforeFailure(t *testing.T) {
+	control := newRunControl(2)
+	observationEntered := make(chan struct{})
+	releaseObservation := make(chan struct{})
+	failureReachedLock := make(chan struct{})
+	failureReturned := make(chan error, 1)
+
+	control.beforeFailureLock = func() { close(failureReachedLock) }
+	observationReturned := make(chan bool, 1)
+	go func() {
+		observationReturned <- control.whileHealthy(func() {
+			close(observationEntered)
+			<-releaseObservation
+		})
+	}()
+	awaitRunControl(t, observationEntered, "observation callback to hold the controller")
+
+	go func() { failureReturned <- control.fail("fold", errors.New("boom")) }()
+	awaitRunControl(t, failureReachedLock, "failure to reach the controller mutex")
+	select {
+	case err := <-failureReturned:
+		t.Fatalf("failure returned while an earlier observation held the controller: %v", err)
+	default:
+	}
+
+	close(releaseObservation)
+	if ok := awaitRunControl(t, observationReturned, "observation to finish"); !ok {
+		t.Fatal("observation admitted before failure was unexpectedly rejected")
+	}
+	if err := awaitRunControl(t, failureReturned, "failure to publish"); err == nil || err.Error() != "fold: boom" {
+		t.Fatalf("failure = %v, want fold: boom", err)
+	}
+
+	called := false
+	if ok := control.whileHealthy(func() { called = true }); ok {
+		t.Fatal("observation after failure publication was admitted")
+	}
+	if called {
+		t.Fatal("rejected observation callback ran")
+	}
+}
diff --git a/cmd/metis/select_cmd.go b/cmd/metis/select_cmd.go
index 8bb3f84..50688db 100644
--- a/cmd/metis/select_cmd.go
+++ b/cmd/metis/select_cmd.go
@@ -32,6 +32,7 @@ func cmdSelect(args []string) error {
	promote := fs.Bool("promote", false, "materialize the selected config(s): reconstruct from the ledger + run on ALL data → runs/best-{family}-{hash}/submission.csv; prints the run id(s)")
	fingerprint := fs.String("fingerprint", "", "restrict to one code-fingerprint (metis#27)")
	point := fs.String("point", "", "metis#41: publish an OPERATOR-CHOSEN config by ledger row — a point_addr (git-style prefix ok); ships as point-{family}-{hash}. Mutually exclusive with --best/--best-per-model-class")
+	cohort := fs.Bool("cohort", false, "metis#52: list the ledger's code-fingerprint cohorts and exit (the `metis ledger fingerprints` table, on select's surface)")
	shapePath, flags, err := hoistShapePath(args)
	if err != nil {
		return fmt.Errorf("select: %w (usage: metis select <shape.md> [--best | --best-per-model-class | --point ADDR] [--promote] [--fingerprint HASH])", err)
@@ -39,6 +40,11 @@ func cmdSelect(args []string) error {
	if err := fs.Parse(flags); err != nil {
		return err
	}
+	if *cohort {
+		// metis#52: a listing door where the operator's hands already are — pure
+		// delegation to the #39 core (one implementation, two CLI surfaces).
+		return showFingerprints(shapePath, os.Stdout)
+	}
	if *point == "" && !*best && !*perClass {
		*best = true // default view = the single ship recommendation
	}
@@ -146,6 +152,12 @@ func runSelect(o selectOpts) error {
	}

	sort.Slice(picks, func(i, j int) bool { return picks[i].family < picks[j].family })
+	// metis#52: attach each pick's --point handle (the first cohort-filtered ledger row of
+	// that config — any fold row is a valid handle by #41's resolver). Good practice made
+	// mechanical: a concrete "best" config is always shown WITH its override handle.
+	for i := range picks {
+		picks[i].handle = pointHandleFor(led, picks[i].winner.Point)
+	}
	printSelect(o.out, sh, est, picks, o.perClass)

	if o.promote {
@@ -161,6 +173,7 @@ type familyPick struct {
	est    sampler.MeanSE
	hasEst bool
	caveat string
+	handle string // metis#52: a representative ledger-row point_addr — the --point override handle
 }

 // perFamilyConfigWinners runs the shape's metis#19 select rule over the INNER rows (Level != "outer":
@@ -233,13 +246,28 @@ func printSelect(out io.Writer, sh experiment.Shape, est map[string]sampler.Mean
	}
	fmt.Fprintf(out, "  %s:\n", head)
	for _, p := range picks {
-		fmt.Fprintf(out, "    %-24s %s\n", famLabel(p.family), freeParamStrFromParams(p.winner.Point.FreeParams))
+		h := ""
+		if p.handle != "" {
+			h = " · point " + short(p.handle) // metis#52: the --point override handle
+		}
+		fmt.Fprintf(out, "    %-24s %s%s\n", famLabel(p.family), freeParamStrFromParams(p.winner.Point.FreeParams), h)
		if p.caveat != "" {
			fmt.Fprintf(out, "      caveat: %s\n", p.caveat)
		}
	}
 }

+// pointHandleFor finds a representative ledger-row point_addr for a config (first match
+// in append order) — "" when the config has no rows (then no handle is shown; never lie).
+func pointHandleFor(led ledger.Ledger, p shape.Point) string {
+	for _, r := range led.Rows {
+		if freeParamsEqual(p, r.FreeParams) {
+			return r.PointAddr
+		}
+	}
+	return ""
+}
+
 func famLabel(f string) string {
	if f == "" {
		return "(one family)"
@@ -330,6 +358,8 @@ func promoteSelected(o selectOpts, sh experiment.Shape, picks []familyPick) erro
			return fmt.Errorf("select --promote %s: %w", famLabel(p.family), err)
		}
		runID := "best-" + familyTag(p.family) + "-" + short(addr)
+		// no leafPins: a promoted ship is a SERIAL single all-data fit — multi-threaded
+		// BLAS is wanted here, and one leaf can't oversubscribe (#48's conscious exclusion)
		ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
		if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
			return fmt.Errorf("select --promote %s (%s): %w", famLabel(p.family), runID, err)
@@ -502,6 +532,8 @@ func runPointSelect(o selectOpts, sh experiment.Shape, led ledger.Ledger, metric
		return fmt.Errorf("select --point --promote: %w", err)
	}
	runID := "point-" + familyTag(fam) + "-" + short(addr)
+	// no leafPins: a promoted ship is a SERIAL single all-data fit — multi-threaded
+	// BLAS is wanted here, and one leaf can't oversubscribe (#48's conscious exclusion)
	ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
	if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
		return fmt.Errorf("select --point --promote (%s): %w", runID, err)
diff --git a/cmd/metis/select_cmd_test.go b/cmd/metis/select_cmd_test.go
index 227f6ef..204827e 100644
--- a/cmd/metis/select_cmd_test.go
+++ b/cmd/metis/select_cmd_test.go
@@ -1,6 +1,8 @@
 package main

 import (
+	"bytes"
+	"io"
	"os"
	"path/filepath"
	"strings"
@@ -528,3 +530,58 @@ func TestSelect_CohortGuardNamesInspectCommand(t *testing.T) {
		t.Errorf("guard must inline the per-cohort summary, got: %v", err)
	}
 }
+
+// metis#52: `metis select <shape> --cohort` lists the fingerprint cohorts — the same
+// table as `metis ledger fingerprints` (one core, a second door on select's surface).
+func TestSelect_CohortFlag(t *testing.T) {
+	shapePath := writeFingerprintFixture(t, t.TempDir())
+	r, w, _ := os.Pipe()
+	orig := os.Stdout
+	os.Stdout = w
+	err := run([]string{"select", shapePath, "--cohort"})
+	_ = w.Close()
+	os.Stdout = orig
+	var buf bytes.Buffer
+	_, _ = io.Copy(&buf, r)
+	if err != nil {
+		t.Fatalf("select --cohort: %v", err)
+	}
+	out := buf.String()
+	for _, want := range []string{"code-fingerprint cohort(s)", "aaaa1111", "bbbb2222", "(legacy)"} {
+		if !strings.Contains(out, want) {
+			t.Errorf("--cohort output missing %q:\n%s", want, out)
+		}
+	}
+}
+
+// metis#52: every concrete config shown as best carries its point handle — and the
+// printed handle round-trips through `select --point`.
+func TestSelect_PickCarriesPointHandle(t *testing.T) {
+	dir := t.TempDir()
+	shapePath := writeSelectLedger(t, dir, taggedShapeForSelect, true)
+	var out strings.Builder
+	if err := runSelect(selectOpts{shapePath: shapePath, best: true, out: &out}); err != nil {
+		t.Fatalf("select: %v", err)
+	}
+	s := out.String()
+	// The logreg winner's handle = the first ledger row of that config (i-lr1-0).
+	if !strings.Contains(s, "point i-lr1-0") {
+		t.Errorf("the ship recommendation must carry its --point handle; got:\n%s", s)
+	}
+	// Round-trip: the printed handle IS a valid --point selector on the same ledger.
+	var out2 strings.Builder
+	if err := runSelect(selectOpts{shapePath: shapePath, point: "i-lr1-0", out: &out2}); err != nil {
+		t.Fatalf("printed handle must round-trip through --point: %v", err)
+	}
+	if !strings.Contains(out2.String(), "train.model=logreg") {
+		t.Errorf("--point round-trip should land on the same config:\n%s", out2.String())
+	}
+	// --best-per-model-class carries handles too.
+	var out3 strings.Builder
+	if err := runSelect(selectOpts{shapePath: shapePath, perClass: true, out: &out3}); err != nil {
+		t.Fatalf("per-class: %v", err)
+	}
+	if got := strings.Count(out3.String(), "point i-"); got < 2 {
+		t.Errorf("per-class winners must each carry a handle, got %d:\n%s", got, out3.String())
+	}
+}
diff --git a/cmd/metis/shapesweep_test.go b/cmd/metis/shapesweep_test.go
index ab5b42c..471378e 100644
--- a/cmd/metis/shapesweep_test.go
+++ b/cmd/metis/shapesweep_test.go
@@ -332,6 +332,10 @@ func TestShapeSweep_OneConfigDegeneratesToSingleLevelCV(t *testing.T) {
		if !strings.Contains(final, "folds 2/2") || !strings.Contains(final, "score 0.") {
			t.Errorf("the flat final progress line must carry folds k/k + score; got: %q", final)
		}
+		// metis#50: the flat path ends with the same summary block.
+		if !strings.Contains(s, "metis: done in ") || !strings.Contains(s, "metis select ") {
+			t.Errorf("flat run missing the run-end summary:\n%s", s)
+		}
	}
	if strings.Contains(out.String(), "nested-CV") {
		t.Errorf("a 1-config shape must NOT run nested-CV; got:\n%s", out.String())
@@ -521,3 +525,20 @@ func loadLedgerOrFatal(t *testing.T, expPath string) ledger.Ledger {
	}
	return led
 }
+
+// metis#50: a degraded capture (no fingerprint) degrades the summary honestly —
+// `cohort ?` and NO lying `--fingerprint` pin (a single-cohort ledger needs none).
+func TestPrintRunSummary_DegradedCohort(t *testing.T) {
+	var out strings.Builder
+	printRunSummary(&out, "s.md", 90*time.Second, 42, "")
+	s := out.String()
+	if !strings.Contains(s, "(cohort ?)") {
+		t.Errorf("degraded capture must render cohort ?: %s", s)
+	}
+	if strings.Contains(s, "--fingerprint") {
+		t.Errorf("degraded capture must not emit a lying pin: %s", s)
+	}
+	if !strings.Contains(s, "done in 1m30s") || !strings.Contains(s, "42 rows") || !strings.Contains(s, "metis select s.md") {
+		t.Errorf("summary basics: %s", s)
+	}
+}
diff --git a/cmd/metis/sweep.go b/cmd/metis/sweep.go
index 88023f6..12a8504 100644
--- a/cmd/metis/sweep.go
+++ b/cmd/metis/sweep.go
@@ -72,9 +72,7 @@ type configScore struct {
 }

 // shapeSweep is the mutable accumulator the pure nested-Sampler loop folds through the IO
-// shell: it drives each (config, fold) point-run through the shared cached runner, records
-// the manifest + per-config estimates, and captures the first fatal error (the pure Run
-// has no error channel, so a fatal fold sets ss.err and short-circuits the rest).
+// shell. Its runControl is the one failure authority across every nested pass.
 type shapeSweep struct {
	o             runOpts
	sh            experiment.Shape
@@ -88,14 +86,29 @@ type shapeSweep struct {
	parallel      bool           // metis#31: >1 max-parallel ⇒ the sweeper/resample/driver batches run via ParExec
	manMu         sync.Mutex     // metis#32: guards man.Points — concurrent outer folds (ParExec) each record rows
	prog          *sweepProgress // metis#30: the live-progress sink (nil = silent)
+	start         time.Time      // metis#50: sweep wall-clock start (injected clock)
 }

 // addManPoints appends a batch of manifest rows under the manifest lock (metis#32: the
 // nested run's outer folds run concurrently under ParExec, each recording its inner+outer rows).
-func (ss *shapeSweep) addManPoints(pts []pointRun) {
-	ss.manMu.Lock()
-	defer ss.manMu.Unlock()
-	ss.man.Points = append(ss.man.Points, pts...)
+func (ss *shapeSweep) fail(label string, err error) error {
+	return ss.o.runControl.fail(label, err)
+}
+
+func (ss *shapeSweep) firstError() error {
+	return ss.o.runControl.firstError()
+}
+
+func (ss *shapeSweep) whileHealthy(fn func()) bool {
+	return ss.o.runControl.whileHealthy(fn)
+}
+
+func (ss *shapeSweep) addManPoints(pts []pointRun) bool {
+	return ss.whileHealthy(func() {
+		ss.manMu.Lock()
+		defer ss.manMu.Unlock()
+		ss.man.Points = append(ss.man.Points, pts...)
+	})
 }

 // sweepPass accumulates ONE black-box sweeper run (the sweeper ⊃ resample loop): its per-config
@@ -115,44 +128,37 @@ type sweepPass struct {
	hooks    passHooks            // metis#30: this pass's progress hooks, closure-bound to its outer fold
	// metis#31: under ParExec the sweeper fans out over configs and each config's
	// resample fans out over folds — all appending to this ONE pass. `mu` guards the
-	// orchestration bookkeeping (configs/points/err); the honest reduce stays in the
+	// orchestration bookkeeping (configs/points); the honest reduce stays in the
	// sampler's pure Tell/Done, not here.
	mu      sync.Mutex
	configs []configScore
	points  []pointRun
-	err     error
 }

-// setErr records the FIRST fatal error of the pass (set-once, mutex-guarded — the
-// fan-out writes it concurrently). A no-op once an error is already latched.
-func (p *sweepPass) setErr(err error) {
-	p.mu.Lock()
-	defer p.mu.Unlock()
-	if p.err == nil {
-		p.err = err
-	}
+func (p *sweepPass) setErr(label string, err error) error {
+	return p.ss.fail(label, err)
 }

-// firstError returns the latched error (mutex-guarded read — a concurrent read+write
-// is a race even when the write is set-once).
 func (p *sweepPass) firstError() error {
-	p.mu.Lock()
-	defer p.mu.Unlock()
-	return p.err
+	return p.ss.firstError()
 }

 // addConfigScore / addPoint append the per-config estimate / per-fold row under the
 // pass lock (concurrent under ParExec).
-func (p *sweepPass) addConfigScore(cs configScore) {
-	p.mu.Lock()
-	defer p.mu.Unlock()
-	p.configs = append(p.configs, cs)
+func (p *sweepPass) addConfigScore(cs configScore) bool {
+	return p.ss.whileHealthy(func() {
+		p.mu.Lock()
+		defer p.mu.Unlock()
+		p.configs = append(p.configs, cs)
+	})
 }

-func (p *sweepPass) addPoint(pr pointRun) {
-	p.mu.Lock()
-	defer p.mu.Unlock()
-	p.points = append(p.points, pr)
+func (p *sweepPass) addPoint(pr pointRun) bool {
+	return p.ss.whileHealthy(func() {
+		p.mu.Lock()
+		defer p.mu.Unlock()
+		p.points = append(p.points, pr)
+	})
 }

 // runSweeper runs the black-box sweeper (GridConfigs ⊃ FixedKFolds) over configPts, folding each
@@ -165,11 +171,15 @@ func (ss *shapeSweep) runSweeper(ctx sampler.Ctx, configPts []shape.Point, pass
		func(c shape.Point) sampler.MeanSE {
			ms := sampler.Run(ctx, sampler.FixedKFolds{K: pass.splitK},
				func(f sampler.FoldPoint) sampler.FoldOutcome { return pass.runPipelineFold(c, f) },
-				sampler.ExecFor[sampler.FoldPoint, sampler.FoldOutcome](ss.parallel), pass.hooks.fold)
+				sampler.ExecFor[sampler.FoldPoint, sampler.FoldOutcome](ss.parallel), func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
+					ss.whileHealthy(func() { pass.hooks.fold(ev) })
+				})
			pass.addConfigScore(configScore{point: c, meanSE: ms})
			return ms
		},
-		sampler.ExecFor[shape.Point, sampler.MeanSE](ss.parallel), pass.hooks.config)
+		sampler.ExecFor[shape.Point, sampler.MeanSE](ss.parallel), func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
+			ss.whileHealthy(func() { pass.hooks.config(ev) })
+		})
 }

 // runShapeSweep drives the metis#18 nested Sampler loop: the sweeper (GridConfigs over the
@@ -179,7 +189,8 @@ func (ss *shapeSweep) runSweeper(ctx sampler.Ctx, configPts []shape.Point, pass
 // the winner by the objective; driver:single ships it (M1a-5). Produces per-config (mean,SE)
 // + the manifest + the raw per-fold ledger. Per-fold failure is fatal to the sweep (surfaced,
 // not swallowed — a partial resample is not an honest estimate).
-func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) error {
+func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) (result error) {
+	sweepStart := now() // metis#50: the run-end summary reports wall-clock elapsed
	// probeRepo's HEAD sha still drives the mid-sweep code-freeze guard (codeID) — NOT the
	// identity (metis#27 dropped repo_shas). The shape's blob-hash content-addresses the intent.
	_, sha, _ := probeRepo(o.git, filepath.Dir(o.expPath))
@@ -243,7 +254,7 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
		return err
	}
	ss := &shapeSweep{
-		o: o, sh: sh, now: now, out: out, shapeBlobHash: sbh, codeID: sha,
+		o: o, sh: sh, now: now, out: out, shapeBlobHash: sbh, codeID: sha, start: sweepStart,
		partRef:  partitionRef(sh),
		man:      sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweeper.Sampler, Seed: sh.Seed},
		parallel: o.maxParallel > 1, // metis#31: fan out the sweeper/resample/driver batches
@@ -260,19 +271,41 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
		ss.prog.bw = o.board
		ss.prog.width = boardWidth()
		ss.prog.gauge = o.leafGauge
-		ticker := time.NewTicker(500 * time.Millisecond)
+		tickC := o.boardTick
+		var ticker *time.Ticker
+		if tickC == nil {
+			ticker = time.NewTicker(500 * time.Millisecond)
+			tickC = ticker.C
+		}
		tickDone := make(chan struct{})
+		tickStopped := make(chan struct{})
		go func() {
+			defer close(tickStopped)
			for {
				select {
-				case <-ticker.C:
-					ss.prog.tick()
+				case <-tickC:
+					if o.beforeBoardTick != nil {
+						o.beforeBoardTick()
+					}
+					ss.whileHealthy(func() { ss.prog.tick() })
+					if o.afterBoardTick != nil {
+						o.afterBoardTick()
+					}
				case <-tickDone:
					return
				}
			}
		}()
-		defer func() { ticker.Stop(); close(tickDone) }()
+		defer func() {
+			if ticker != nil {
+				ticker.Stop()
+			}
+			close(tickDone)
+			<-tickStopped
+			if result != nil {
+				ss.prog.abort()
+			}
+		}()
	}

	// metis#32: >1 config → nested CV (records inner + per-family outer rows; the honest measure).
@@ -291,7 +324,10 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
	res := sampler.Run(ctx, sampler.SingleDriver{}, func(sampler.SinglePoint) sampler.SweepResult {
		return ss.runSweeper(ctx, configPts, pass)
	}, sampler.ExecFor[sampler.SinglePoint, sampler.SweepResult](ss.parallel), nil)
-	ss.prog.finish() // metis#30: the terminal progress line, before the report
+	if err := ss.firstError(); err != nil {
+		return err
+	}
+	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the report
	// metis#31: sort the fan-out's completion-order bookkeeping to a stable content key
	// BEFORE persisting, so manifest.json + the ledger are byte-deterministic across
	// serial/parallel runs (the winner/estimate are already deterministic; this makes
@@ -299,23 +335,22 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
	sortPointRuns(pass.points)
	ss.man.Points = pass.points
	ss.configs = pass.configs
-	if err := pass.firstError(); err != nil {
-		return err
-	}
-
	if err := writeManifest(o.expPath, ss.man); err != nil {
-		return err
+		return ss.fail("write sweep manifest", err)
	}
	// Capture the sweep's code closure to a git side ref (metis#8/#14) — BEST-EFFORT: the
	// records + manifest are already valid, so a capture hiccup warns, never aborts.
-	if err := captureSweepCode(o, ss.man); err != nil {
-		fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
+	cohort, err := captureSweepCode(o, ss.man)
+	if err != nil {
+		ss.whileHealthy(func() {
+			fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
+		})
	}
	// Persist the raw per-fold rows to the shape's append-only ledger sidecar (metis#8/#18):
	// AggregateView reduces them read-time to per-config (mean, SE) — so metis#19's 1-SE
	// select re-reduces the same rows without a re-run.
	if err := writeSweepLedger(o.expPath, ss.man); err != nil {
-		return err
+		return ss.fail("write sweep ledger", err)
	}
	// Guard (metis#19): a parsimony rule (one-std-err/pct-loss) needs a measured complexity
	// for every swept family — else the parsimony axis is silently dropped and the winner is
@@ -323,10 +358,13 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
	// the ship/report is gated. Checked here (post-fold) because HasComplexity is only known
	// after the folds run.
	if err := sampler.GuardComplexity(sh.Sweeper.Objective.Select, configStatsOf(ss.configs)); err != nil {
-		return err
+		return ss.fail("sweep complexity guard", err)
	}
-	ss.reportWinner(res)
-	return nil
+	ss.whileHealthy(func() {
+		ss.reportWinner(res)
+		printRunSummary(out, o.expPath, now().Sub(sweepStart), len(ss.man.Points), cohort)
+	})
+	return ss.firstError()
 }

 // configStatsOf builds the per-config stats (with each config's family) from a completed
@@ -368,54 +406,38 @@ func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, r
	// fewer of them (runFolds ≤ k). Deterministic run id → the analysis_i refs are locatable.
	analysisRefs, err := ss.materializeOuterAnalysis(k, stratify)
	if err != nil {
-		return err
+		if first := ss.firstError(); first != nil {
+			return first
+		}
+		return ss.fail("nested-CV preamble", err)
	}
	outerPart := sampler.PartitionRef(fmt.Sprintf("outer-cv-k%d-strat%t-seed%d", k, stratify, ss.sh.Seed))

-	// metis#31: CVDriver.Ask emits all outer folds as one batch, so ParExec runs these
-	// closures concurrently — firstErr's read+write must be mutex-guarded (a set-once
-	// write racing a read is still a data race). Each outer fold has its OWN sweepPass
-	// (created in runOuterFold), so the inner fan-out is guarded there; this guards only
-	// the outer-fold error latch. The per-fold ledger-row recording is guarded by ss.manMu.
-	var errMu sync.Mutex
-	var firstErr error
-	setFirst := func(err error) {
-		errMu.Lock()
-		defer errMu.Unlock()
-		if firstErr == nil {
-			firstErr = err
-		}
-	}
-	getFirst := func() error {
-		errMu.Lock()
-		defer errMu.Unlock()
-		return firstErr
-	}
	est := sampler.Run(ctx, sampler.CVDriver{K: runFolds, Stratify: stratify},
		func(p sampler.OuterFoldPoint) float64 {
-			if getFirst() != nil {
+			if ss.firstError() != nil {
				return 0
			}
			score, ferr := ss.runOuterFold(ctx, configPts, k, stratify, analysisRefs[p.Idx], outerPart, p.Idx)
			if ferr != nil {
-				setFirst(ferr)
+				if ss.firstError() == nil {
+					ss.fail(fmt.Sprintf("outer fold %d", p.Idx), ferr)
+				}
				return 0
			}
			return score
		},
		sampler.ExecFor[sampler.OuterFoldPoint, float64](ss.parallel),
-		// metis#30: outer-fold completions always emit. Error-gated: once firstErr
+		// metis#30: outer-fold completions always emit. Error-gated: once runControl
		// latches, remaining closures return sentinel zeros — don't fold those into
		// the displayed est (the run is aborting; a fake 0 would tank the line).
		func(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
-			if getFirst() == nil {
-				ss.prog.driverEvent(ev)
-			}
+			ss.whileHealthy(func() { ss.prog.driverEvent(ev) })
		})
-	if err := getFirst(); err != nil {
+	if err := ss.firstError(); err != nil {
		return err
	}
-	ss.prog.finish() // metis#30: the terminal progress line, before the estimate report
+	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the estimate report

	// metis#32: the nested run now RECORDS (unlike metis#23's estimation-only path) — persist the
	// inner + per-family outer rows accumulated in ss.man.Points so `metis select` can reduce them
@@ -423,16 +445,22 @@ func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, r
	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
	sortPointRuns(ss.man.Points)
	if err := writeManifest(ss.o.expPath, ss.man); err != nil {
-		return err
+		return ss.fail("write nested sweep manifest", err)
	}
-	if err := captureSweepCode(ss.o, ss.man); err != nil {
-		fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", err)
+	cohort, cerr := captureSweepCode(ss.o, ss.man)
+	if cerr != nil {
+		ss.whileHealthy(func() {
+			fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
+		})
	}
	if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
-		return err
+		return ss.fail("write nested sweep ledger", err)
	}
-	ss.reportEstimate(est, runFolds)
-	return nil
+	ss.whileHealthy(func() {
+		ss.reportEstimate(est, runFolds)
+		printRunSummary(ss.out, ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
+	})
+	return ss.firstError()
 }

 // materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
@@ -451,13 +479,14 @@ func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]str
	exp.Type = "experiment"
	preID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
-		return nil, fmt.Errorf("nested-CV preamble: %w", err)
+		return nil, ss.fail("nested-CV preamble address", err)
	}
	preOpts := ss.o
	preOpts.inSweep = true // one preamble run; skip the per-run capture noise
	preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
+	preOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
-		return nil, fmt.Errorf("nested-CV preamble (%s): %w", preID, err)
+		return nil, err
	}
	refs := make([]string, outerK)
	for i := 0; i < outerK; i++ {
@@ -473,7 +502,7 @@ func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]str
 func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k int, stratify bool, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
	if err != nil {
-		return 0, err
+		return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
	}
	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
@@ -481,22 +510,26 @@ func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k i
		hooks: ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
	sres := ss.runSweeper(ctx, configPts, pass)
	if err := pass.firstError(); err != nil {
-		return 0, fmt.Errorf("outer fold %d sealed sweep: %w", i, err)
+		return 0, err
	}
	// Guard (metis#19/#23 I1): the parsimony select rule needs a measured complexity for every
	// swept family — same guard the flat path runs before trusting its winner. Without it, a
	// parsimony-select + non-reporting-model shape would SILENTLY mis-select in each outer fold.
	if err := sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs)); err != nil {
-		return 0, fmt.Errorf("outer fold %d: %w", i, err)
+		return 0, ss.fail(fmt.Sprintf("outer fold %d complexity guard", i), err)
	}

	// metis#32: record the sealed sweep's INNER rows (Level=inner, tagged with this outer fold).
	of := i
	rows := make([]pointRun, 0, len(pass.points)+len(sres.PerFamily))
-	for _, pr := range pass.points {
-		pr.Level = "inner"
-		pr.OuterFold = &of
-		rows = append(rows, pr)
+	if !ss.whileHealthy(func() {
+		for _, pr := range pass.points {
+			pr.Level = "inner"
+			pr.OuterFold = &of
+			rows = append(rows, pr)
+		}
+	}) {
+		return 0, errRunAborted
	}

	// (b) score EACH family's inner-winner on the held outer-assessment — post-selection, so
@@ -510,44 +543,51 @@ func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k i
	var shipScore float64
	for _, fam := range sortedFamilies(sres.PerFamily) {
		w := sres.PerFamily[fam]
-		score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart)
+		score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart, fam)
		if ferr != nil {
-			return 0, fmt.Errorf("outer fold %d family %s score: %w", i, fam, ferr)
+			return 0, ferr
		}
-		rows = append(rows, pointRun{
-			RunID:      scoreID,
-			FreeParams: freeParamMap(w.Point),
-			Fold:       of, // the outer fold this held-out score is on
-			Level:      "outer",
-			OuterFold:  &of,
-			Status:     status,
-			// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
-		})
-		if fam == shipFamily {
-			shipScore = score
+		if !ss.whileHealthy(func() {
+			rows = append(rows, pointRun{
+				RunID:      scoreID,
+				FreeParams: freeParamMap(w.Point),
+				Fold:       of, // the outer fold this held-out score is on
+				Level:      "outer",
+				OuterFold:  &of,
+				Status:     status,
+				// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
+			})
+			if fam == shipFamily {
+				shipScore = score
+			}
+			fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
+				i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
+		}) {
+			return 0, errRunAborted
		}
-		fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
-			i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
	}
-	ss.addManPoints(rows)
+	if !ss.addManPoints(rows) {
+		return 0, errRunAborted
+	}
	return shipScore, nil
 }

 // scoreOnOuterFold refit-and-scores one config's winner on the held outer-assessment fold i (a
 // full-data fold run at outer-k; post-selection, so unconfined). Returns the held-out fold_score,
 // the run id (→ its record.json carries the namespaced metric the ledger reads), and its status.
-func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef) (float64, string, string, error) {
+func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef, fam string) (float64, string, string, error) {
	scoreExp := ss.buildFoldExperiment(point, sampler.FoldPoint{Idx: i}, nil, k, stratify, outerPart)
	scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
	if err != nil {
-		return 0, "", "", err
+		return 0, "", "", ss.fail(fmt.Sprintf("outer fold %d family %s score address", i, fam), err)
	}
	scoreOpts := ss.o
	scoreOpts.inSweep = true
	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
+	scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
	if err != nil {
-		return 0, "", "", fmt.Errorf("%s: %w", scoreID, err)
+		return 0, "", "", err
	}
	return run.Metrics[foldMetric], scoreID, run.Status, nil
 }
@@ -574,8 +614,9 @@ func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
 // runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
 // synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
 // the shared cached runner, record the manifest row, and return the fold_score the inner
-// resample Sampler folds. A fatal outcome sets ss.err and returns 0 (the pure Run keeps
-// going; runShapeSweep checks ss.err before using the winner).
+// resample Sampler folds. A fatal outcome publishes through the experiment-wide
+// runControl and returns 0; every sampler callback/sink rejects placeholders after
+// publication, and the top level returns the stored concrete cause.
 func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.FoldOutcome {
	ss := p.ss
	if p.firstError() != nil {
@@ -589,34 +630,39 @@ func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.
	// .git/index.lock so a transient probe failure is expected; treating "" as a change
	// would false-abort the whole honest run.
	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
-		p.setErr(fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
+		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx),
+			fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
		return sampler.FoldOutcome{}
	}

	exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.stratify, p.partRef)
	runID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
-		p.setErr(fmt.Errorf("config %s fold %d: %w", freeParamStr(c), f.Idx, err))
+		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx), err)
		return sampler.FoldOutcome{}
	}
	pointOpts := ss.o
	pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
+	pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
	if runErr != nil {
-		p.setErr(fmt.Errorf("config %s fold %d (%s): %w", freeParamStr(c), f.Idx, runID, runErr))
+		// runControl already published a concrete admitted-run failure. A queued or
+		// late sibling returns errRunAborted; neither path may republish the sentinel.
		return sampler.FoldOutcome{}
	}
-	p.addPoint(pointRun{
+	if !p.addPoint(pointRun{
		RunID:      runID,
		FreeParams: freeParamMap(c),
		Fold:       f.Idx,
		Status:     run.Status,
		Metrics:    run.Metrics,
-	})
+	}) {
+		return sampler.FoldOutcome{}
+	}
	// metis#19 M2: read the train step's realized-complexity metric. Present → the parsimony
	// rules consume it; absent (HasComplexity false) → the guard rejects a parsimony rule.
	cx, hasCx := run.Metrics[foldComplexityMetric]
@@ -886,3 +932,24 @@ func freeParamStrFromParams(fps []shape.FreeParam) string {
	}
	return s
 }
+
+// printRunSummary is metis#50's run-end handoff: elapsed wall-clock, what landed where,
+// and the paste-ready follow-up commands with the cohort fingerprint pre-filled — the
+// operator should never scrape the scrollback to assemble a `metis select`. A degraded
+// capture (no fingerprint) degrades honestly: `cohort ?` and un-pinned hints (a
+// single-cohort ledger needs no pin).
+func printRunSummary(out io.Writer, expPath string, elapsed time.Duration, rows int, cohort record.Hash) {
+	base := filepath.Base(ledgerPath(expPath))
+	if cohort == "" {
+		fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort ?)\n", fmtETA(elapsed), rows, base)
+		fmt.Fprintf(out, "  next: metis select %s\n", expPath)
+		fmt.Fprintf(out, "        metis select %s --best --promote\n", expPath)
+		fmt.Fprintf(out, "        metis ledger fingerprints %s\n", expPath)
+		return
+	}
+	fp := short(string(cohort))
+	fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort %s)\n", fmtETA(elapsed), rows, base, fp)
+	fmt.Fprintf(out, "  next: metis select %s --fingerprint %s               # the honest pick\n", expPath, fp)
+	fmt.Fprintf(out, "        metis select %s --fingerprint %s --best --promote   # materialize it\n", expPath, fp)
+	fmt.Fprintf(out, "        metis ledger fingerprints %s                   # cohorts\n", expPath)
+}
diff --git a/testdata/experiment/toy-sweep-smoke.md b/testdata/experiment/toy-sweep-smoke.md
new file mode 100644
index 0000000..bc2e52f
--- /dev/null
+++ b/testdata/experiment/toy-sweep-smoke.md
@@ -0,0 +1,27 @@
+---
+type: experiment-shape
+id: toy-sweep-smoke
+seed: 42
+status: active
+data:
+  - id: data
+    uses: test/echo
+    with: {out: ../dataset/toy}
+pipeline:
+  - id: train
+    uses: metis/train
+    needs: [data]
+    with:
+      dataset: ../dataset/toy
+      model:
+        $any:
+          logreg: {C: {$any: [0.5, 1.0, 2.0]}}
+sweeper:
+  sampler: grid
+  resample: {cv: {k: 2, stratify: true}}
+  objective: {metric: train.fold_score, direction: maximize, select: {argmax-mean: {}}}
+---
+
+# toy-sweep-smoke
+
+A credential-free, disposable real-process nested sweep for cold scheduling smoke checks.
diff --git a/testdata/steps/test/env-dump b/testdata/steps/test/env-dump
index 6f4b8c5..716fa69 100755
--- a/testdata/steps/test/env-dump
+++ b/testdata/steps/test/env-dump
@@ -11,4 +11,9 @@ d="${METIS_STEP_DIR:-$PWD}"
   echo "EXP_DIR=$METIS_EXP_DIR"
   echo "SEED=$METIS_SEED"
   echo "READ_ROOT=${METIS_READ_ROOT:-<unset>}"
+  # metis#48: the four leaf BLAS pins (full names — asserted verbatim by the seam test)
+  echo "OMP_NUM_THREADS=${OMP_NUM_THREADS:-<unset>}"
+  echo "OPENBLAS_NUM_THREADS=${OPENBLAS_NUM_THREADS:-<unset>}"
+  echo "VECLIB_MAXIMUM_THREADS=${VECLIB_MAXIMUM_THREADS:-<unset>}"
+  echo "MKL_NUM_THREADS=${MKL_NUM_THREADS:-<unset>}"
 } > "$d/env.txt"
diff --git a/workshop/lessons.md b/workshop/lessons.md
index baa18a9..a137592 100644
--- a/workshop/lessons.md
+++ b/workshop/lessons.md
@@ -158,12 +158,12 @@ finished in ~4 min — the run looked alive while making no progress. The `--par
 documents exactly this caveat; it still shipped as the default behavior on a real sweep. Relaunch
 with `OMP_NUM_THREADS=1 …=1 --parallel 8` → load ~21, ~107 trains/min, done in ~28 min.

-**Rule:** for a real (subprocess-leaf) sweep, ALWAYS pin the leaf's thread env
-(`OMP_NUM_THREADS=1 OPENBLAS_NUM_THREADS=1 VECLIB_MAXIMUM_THREADS=1 MKL_NUM_THREADS=1`) and cap
-`--parallel` below core count. Diagnostic signature of the thrash: starts ≫ completions with the
-process alive (throughput ≈ 0) — which is also why the #38 progress board needs a moving-average
-runs/sec line, not just liveness. Deeper fix candidate: metis could set single-thread BLAS env for
-its leaf subprocesses BY DEFAULT (the parallelism budget belongs to the orchestrator, not the leaf).
+**Rule:** for a real (subprocess-leaf) sweep, the leaf's thread env must be pinned and
+`--parallel` capped below core count. **RESOLVED BY DEFAULT since metis#48** — bare `metis run`
+now injects the four pins at both spawn seams (export a `*_NUM_THREADS` value to override); the
+rule survives for non-metis contexts and as the WHY behind the default. Diagnostic signature of
+the thrash: starts ≫ completions with the process alive (throughput ≈ 0) — which is also why the
+#38 progress board needs a moving-average runs/sec line, not just liveness.

 ## Plan-sketch folds: set-cardinality, not incremental counts (metis#39 plan review)
 - **A "keep the latest, count the others" fold specified incrementally silently overcounts under non-monotone input — specify it as set-cardinality (`len(set)-1`) with the latest tracked separately.** The #39 plan's ExtraCommits sketch counted displacement transitions per ROW; interleaved-timestamp records (two concurrent sweeps, same fingerprint) would have inflated it row-for-row, and the plan's own happy-path fixture (2 records, monotone) structurally couldn't catch it — add an out-of-order fixture whenever a fold's correctness depends on input order. metis-specific ground truth: **ledger rows are NOT time-ordered** (`sortPointRuns` orders by content key; append order is sweep-completion order).
@@ -179,3 +179,9 @@ its leaf subprocesses BY DEFAULT (the parallelism budget belongs to the orchestr
 - **Writer identity is temporal, not call-graph.** When a plan claims "all output routes through one wrapper," audit every construction-time capture of the underlying writer (pools, closures, structs built earlier in the call chain) — a component that grabbed the writer BEFORE the wrap exists is an invisible bypass. Grep the writer variable at every `new*(out)` site, not just Fprintf sites. (The forkserver pool + captureSweepCode's o.out both captured pre-board writers.)
 - **"Ticker calls repaint()" is a deadlock-or-staleness smell.** In state-owner + painter designs, fix ONE global lock order (state.mu → painter.mu) and route timers through the state owner; a painter-first timer either inverts locks via a state callback or repaints a frame that can't refresh time-derived values (ETA/rate decay).
 - **A stdlib-only TUI plan must name its terminal-size mechanism explicitly.** Width detection is the one capability ANSI pin-bottom genuinely needs beyond stdlib's comfortable surface, and a wrong width isn't cosmetic — a wrapped line breaks the cursor-up erase-count bookkeeping the whole repaint scheme rests on.
+
+## Plan-review lessons (metis#48 plan)
+- **A constructor-grep is not a coverage proof — also grep direct callers of the layer BELOW the wiring point.** Wiring computed in an entry function (`runExperiment`) silently misses call paths that enter beneath it (select_cmd.go builds fresh `runOpts` and calls `runResolvedExperiment` directly). When a plan claims "every production construction is threaded," check the constructor sites AND downstream-function callers; every bypass found is either threaded or documented as a conscious exclusion.
+- **Doc-consistency sweeps must include Go sources, not just `*.md`** — operator guidance lives in flag `--help` strings and load-bearing comments (main.go's `--parallel` help told operators to hand-pin BLAS).
+- **A plan's inline test code must be written against the VERIFIED fixture/format; promote any fixture gap to an explicit numbered step.** A parenthetical "check the fixture" hedge next to code that contradicts it breaks the TDD red-green sequence and invites wrong-reason debugging (env-dump dumps METIS_* only; experiment steps live in YAML frontmatter, not a fenced block).
+- **A cross-repo deliverable is invisible to the closing repo's review window — pin the peer repository + exact commit in the issue Log before close.** A checked plan row and prose saying “RUNBOOK updated” are not independently traceable when the actual diff lives in kbench. Record the peer commit as soon as it lands so the boundary reviewer can verify the requirement without trusting the implementor's assertion.
diff --git a/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md b/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md
new file mode 100644
index 0000000..321f7f5
--- /dev/null
+++ b/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md
@@ -0,0 +1,202 @@
+# Board Readability: Typed Activity, Cold-Phase Truth, and Stable ETA — Implementation Plan
+
+> **For the implementer:** Execute this plan with `superpowers-executing-plans`; use TDD for every
+> behavior change. The issue has one delivery boundary, so do not add milestone tags or run
+> `sdlc milestone-close`; cross the mandatory fresh-context review once with `sdlc close`.
+
+**Goal:** Make the sweep board explain cold work truthfully, use unambiguous flat/nested vocabulary,
+and withhold rate/ETA until time-based evidence is stable without weakening stall visibility.
+
+**Architecture:** A typed activity emitter connects two concrete facts to `sweepProgress`: successful
+final-executor steps (outside cache) and successfully executed, durably persisted concrete runs.
+`sweepProgress` is the sole synchronized reducer; pure bounded windows derive smoothed occupancy,
+event-time rate readiness, decay, and last-run age. `progressCore` remains the shared semantic source
+for plain and TUI output, while `renderBoard` only formats snapshots. Activity publication is gated by
+`runControl` before taking the progress mutex, preserving the established controller → progress →
+writer lock order and preventing post-failure repaint (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).
+
+**Tech stack:** Go, standard library, existing metis run/sweep/progress/board abstractions, Go tests,
+and the kbench Markdown RUNBOOK.
+
+## Core concepts
+
+### PURE entities and transforms
+
+| Concept | Responsibility | Invariants |
+|---|---|---|
+| `activityEvent` | Immutable successful activity fact with kind, typed run role/identity, and injected-clock timestamp. | Failed work creates no event; timestamps describe completion, not callback delivery. |
+| `runRole` | Distinguish nested inner-CV, flat CV, preamble, outer score, and ineligible/no-role runs. | Only inner-CV and flat CV are rate/counter eligible. |
+| `occupancyWindow` | Retain the last four 500ms occupancy samples and return their rounded mean. | Event count cannot affect the result; capacity is four. |
+| `movingRate` | Retain the latest 64 eligible completion times in event-time order and derive readiness/rate from `now`. | Ready only at n≥16 and span≥15s; rate is `(n-1)/(now-oldest)`; reversed delivery is deterministic. |
+| `activitySnapshot` | Read-only facts consumed by formatting: steps, max step time, eligible runs, max run time, smoothed slots, and optional rate. | Last times are maxima; startup ends on the first eligible run. |
+
+### INTEGRATION boundaries
+
+| Boundary | Responsibility | Failure semantics |
+|---|---|---|
+| `activityExecutor` | Decorate the final cache-aware executor and emit one step event after a successful real execution or cache hit. | Inner error is returned unchanged and emits nothing. |
+| `runResolvedExperiment` activity publication | Emit the typed run event only after execution and required `runs/<id>/{run,record}.json` persistence succeed. | Execution failure, `run.json`/`record.json` failure, or provenance assembly failure emits no successful-run event; best-effort capture is not a success gate. |
+| `runControl`-gated emitter | Linearize all step and run activity against fatal failure before calling `sweepProgress`. | Rejected after failure; never acquire controller state while holding progress state. |
+| `sweepProgress` | Synchronize activity/tick reduction and publish immutable board snapshots. | Short callbacks; non-sweep callers receive a no-op emitter. |
+| `renderBoard` / `progressCore` | Apply shared vocabulary and factual startup/mature wording to snapshots. | No diagnosis such as “not hung”; width, cadence, failure flush, and terminal cleanup remain intact. |
+| kbench RUNBOOK | Document the shipped board contract using the exact operator-facing nouns. | Full peer commit SHA is recorded in issue #49 before close. |
+
+## Chunk 1: Typed activity at concrete success seams
+
+### Task 1: Define activity facts and the final-executor decorator
+
+**Files:**
+- Modify: `cmd/metis/run.go`
+- Create or modify: `cmd/metis/activity_test.go`
+- Modify: `cmd/metis/caching_test.go`
+
+1. Write failing table tests proving the decorator emits exactly one timestamped step-success event
+   after a successful inner executor, emits none on error, and preserves the exact result/error.
+2. Add a cache wiring regression: one cold execution and its warm cache hit each produce one step
+   event. Assert the decorator is outside the cache wrapper, not merely that the cold path works.
+3. Introduce the smallest typed `activityEvent`/kind/role vocabulary and a no-op-capable emitter.
+   Implement `activityExecutor` around the final executor built in `runResolvedExperimentAdmitted`.
+   It must publish through the same `runControl`-gated emitter used by successful-run events, so a
+   late successful step callback after sibling failure is rejected before progress repaint.
+4. Run the focused tests and keep identities/timestamps injected; do not parse step output or inspect
+   cache implementation details.
+
+### Task 2: Publish successful concrete-run events at the persistence boundary
+
+**Files:**
+- Modify: `cmd/metis/run.go`
+- Modify: `cmd/metis/run_test.go`
+
+1. Extend the successful-run test with an activity callback that observes both required
+   `runs/<id>/run.json` and `runs/<id>/record.json` artifacts already persisted when the event arrives.
+2. Pin negative paths: a failed execution that successfully writes its failure record emits no run
+   event, and a forced required-persistence failure (for example, a directory at the record path)
+   emits no run event.
+3. Add the run role to `runOpts` and publish only after `runErr == nil` plus required persistence.
+   Preserve best-effort capture behavior: capture failure must not retroactively make a successful
+   run ineligible unless the current contract already treats that artifact as required.
+4. Route successful-run publication through the shared `runControl`-gated emitter before the progress
+   callback. Add barrier regressions showing a sibling fatal failure prevents both a later step event
+   and a later run event from repainting without introducing controller↔progress lock inversion.
+
+### Task 3: Assign roles at every sweep call site
+
+**Files:**
+- Modify: `cmd/metis/sweep.go`
+- Modify: `cmd/metis/run.go`
+- Modify: `cmd/metis/run_test.go`
+
+1. Write a call-site trace test that distinguishes flat CV, nested inner-CV, nested preamble, and
+   outer-score runs and proves only the first two are eligibility candidates.
+2. Add `sweepPass.runRole`; set it at flat pass construction and nested pass construction, then copy
+   it into `pointOpts.runRole` immediately before `runPipelineFold` calls `runResolvedExperiment`.
+   Assign preamble and outer-score roles at their direct launch sites. Keep plain `metis run` and
+   `metis select --promote` explicitly no-op/ineligible; document the bypass rather than silently
+   relying on a zero value.
+3. Assert emitted run-event roles from the concrete call paths, not just enum eligibility. Run focused
+   tests plus `go test ./cmd/metis -run 'Activity|Cache|RunResolved|RunControl' -race`.
+
+## Chunk 2: Deterministic telemetry reduction and board semantics
+
+### Task 4: Replace callback-count rate sampling with event-time reduction
+
+**Files:**
+- Modify: `cmd/metis/progress.go`
+- Modify: `cmd/metis/progress_test.go`
+
+1. Write failing pure tests for eligible/ineligible roles, max last-step/run timestamps, and reversed
+   callback delivery. Feed 65 shuffled completions and prove the latest 64 by event time survive.
+2. Pin readiness boundaries: 15 events are unready; 16 spanning under 15s are unready; 16 spanning
+   exactly 15s are ready. Assert `(n-1)/(now-oldest)`, including a `now` later than the newest event.
+3. Add a mature trace followed by five 1s ticks: last-run age advances five seconds, numeric rate is
+   non-increasing, and ETA is non-decreasing. Then add completions and prove gradual 64-event-window
+   recovery rather than a one-interval snap.
+4. Refactor `movingRate` into a sorted, bounded event-time window. Reduce typed eligible run events
+   under the existing progress mutex and remove fold-callback-time rate mutation.
+5. Move the aggregate displayed `inner-CV runs` / `CV runs` counter to typed eligible run-completion
+   events. Add a reversed-delivery regression proving typed events alone advance the aggregate
+   counter/rate/ETA, while sampler fold callbacks retain only score and per-row duties and cannot
+   double-count or lag the board counter.
+
+### Task 5: Make occupancy tick-driven and event-density independent
+
+**Files:**
+- Modify: `cmd/metis/progress.go`
+- Modify: `cmd/metis/progress_test.go`
+
+1. Add a pure four-sample test: occupancies `[1,2,3,4]` at capacity 12 render as rounded mean 3, and
+   a fifth sample evicts the first.
+2. Drive equal timestamped occupancy samples through traces with sparse versus dense activity events;
+   assert identical snapshots.
+3. Sample `leafGauge` only from the existing 500ms tick, retain four values, and expose the rounded
+   mean plus capacity. Do not update the window from activity callbacks or repaint flushes.
+
+### Task 6: Render shared vocabulary, factual startup, and confidence states
+
+**Files:**
+- Modify: `cmd/metis/progress.go`
+- Modify: `cmd/metis/board.go`
+- Modify: `cmd/metis/progress_test.go`
+- Modify: `cmd/metis/board_test.go`
+
+1. Replace existing expected strings with the exact shared nouns: `outer folds`, `configs scored`,
+   nested `inner-CV runs`, flat `CV runs`, and row prefix `outer fold N`.
+2. Add nested and flat startup golden tests for: no occupancy/activity, occupied but silent, successful
+   steps with last-step age, and the first eligible run removing startup. Positive text must derive
+   only from typed successful events; never print “not hung” or infer a phase.
+3. Add pre-confidence tests showing `— inner-CV runs/min` or `— CV runs/min` and no ETA. Add mature
+   tests showing `~ETA`, the matching rate noun, remaining eligible-run scope, and tick-driven
+   `last … run Ns ago`.
+4. Update `progressCore` once so plain and TUI output share counter semantics; keep board-only temporal
+   observations in the board snapshot/renderer. Preserve width clamping with narrow-width tests.
+
+## Chunk 3: End-to-end wiring, terminal proof, and peer documentation
+
+### Task 7: Wire tick/activity flow through flat and nested sweeps
+
+**Files:**
+- Modify: `cmd/metis/sweep.go`
+- Modify: `cmd/metis/progress.go`
+- Modify: `cmd/metis/progress_test.go`
+- Modify: `cmd/metis/board_test.go`
+
+1. Add end-to-end scripted flat and nested tests that execute successful steps/runs, advance the fake
+   clock across ticks, and assert startup → confidence → mature/stall transitions.
+2. Extend the TUI fatal-failure test so activity is visible before failure, then prove the final error
+   frame is stable, no post-failure activity repaints, the ticker joins, and terminal cleanup remains
+   correct.
+3. Connect the activity emitter after board-writer replacement so callbacks use the compositor's
+   temporal writer identity. Keep the 500ms ticker and existing health gates; do not introduce a
+   second clock loop.
+4. Run `go test ./cmd/metis -run 'Progress|Board|Sweep|Activity|RunControl' -race` and fix the cause of
+   any flake, race, or lock-order timeout before proceeding.
+
+### Task 8: Update the operator contract in kbench
+
+**Files:**
+- Modify in peer repo: `competition/titanic/pipelines/RUNBOOK-sweep.md`
+- Modify: `workshop/issues/000049-board-readability-labels-coldphase-eta.md`
+
+1. In `/Users/xianxu/workspace/kbench`, update the board example/description to the exact flat/nested
+   vocabulary, factual startup line, `~slots` smoothing, 16-completion/15-second confidence gate,
+   last-run age, and mature `~ETA`. Preserve the documented plain-output distinction unless behavior
+   actually changed.
+2. Search the RUNBOOK for stale `fold`, `leaves`, and `folds/min` board terminology; inspect each hit
+   rather than globally replacing legitimate domain language.
+3. Run Markdown/diff checks available in kbench, commit that documentation-only peer change, and add
+   its full 40-character SHA to issue #49's Log.
+
+### Task 9: Full verification and close-boundary preparation
+
+**Files:**
+- Modify if architecture changed: `atlas/` and `atlas/index.md`
+- Modify: `workshop/issues/000049-board-readability-labels-coldphase-eta.md`
+
+1. Run `gofmt` on changed Go files, `go test ./cmd/metis -race`, `go test ./... -race`, and
+   `git diff --check` in metis. Run the relevant kbench checks and `git diff --check` there.
+2. Grep Go, tests, atlas, and operator docs for obsolete board strings; classify remaining domain
+   uses. Confirm flat/nested output, width, repaint cadence, failure cleanup, and no-op non-sweep paths.
+3. Update atlas only if the implementation adds a durable architectural seam; otherwise record why
+   `--no-atlas` is appropriate in close evidence. Check all issue/plan boxes and log exact commands.
+4. Run `sdlc close --issue 49 --verified '<evidence>'` once. Let the binary dispatch the mandatory
+   fresh-context boundary review; fix every Critical/Important finding and rerun the gate as directed.


codex
I’ll review this as a boundary review against the issue and actual repository state, not just the pasted diff. I’m going to read the issue, plan, relevant changed files, and current diff metadata first.
exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000049-board-readability-labels-coldphase-eta.md" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
---
id: 000049
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-17
estimate_hours: 2.63
started: 2026-07-16T12:57:08-07:00
---

# board readability — label semantics, cold-phase "no progress" confusion, jumpy leaves, wild early ETA

## Problem

Operator's first real-sweep board session (titanic-sweep.md, BLAS-pinned, 2026-07-16) surfaced
four readability issues — the board is mechanically correct but hard to READ in exactly the
early phase where the operator most wants reassurance:

1. **Label semantics unclear:** `outer 0/10 · configs 0/720 · folds 0/7200` — operator asked
   "is folds about inner folds?" `folds` = leaf-level (config × inner-fold) RUNS aggregated
   across the whole sweep (10 outer × 72 configs × 10 inner); `configs` = per-outer-pass
   config completions aggregated (72 × 10). Neither is legible from the labels.
2. **Cold-phase "lack of progress":** early in a cold run every fold row shows `queued`, all
   counters sit at 0 for many minutes. Root cause is the metis#43 phase-wave scheduling (all
   cv-splits/features across the grid complete before ANY train chain finishes → zero fold
   completions for ~10 min while heavy upstream work happens). The board renders that
   truthfully but unhelpfully — nothing distinguishes "working through upstream steps" from
   "hung". (#43 fixes the schedule; this issue covers what the board shows MEANWHILE.)
3. **`leaves 2/12` jumps around:** the gauge samples instantaneous `len(leafSem)` at flush
   time — honest, but at 4Hz it reads as noise, and low occupancy during the upstream herd
   phase looks like a problem when it isn't.
4. **ETA changes wildly:** the 64-completion moving window over sparse, phase-heterogeneous
   early completions (fast cache-hit folds vs slow cold trains) swings the rate — the ETA
   flapped across hours on the operator's run. An estimate that volatile is worse than none.

## Spec

### Claim-time directions (retained)

1. Rename/annotate segments (e.g. `inner-folds 421/7200` or a one-line legend on the first
   frame); consider showing per-outer-fold denominators in the fold rows as the source of
   truth for "what is 7200".
2. A PHASE indicator when zero folds have completed but leaves are active (e.g.
   `warming: upstream steps running · 0 folds complete yet`) — distinguish wave-phase from
   hang using signals the sink already has (leaf occupancy > 0, passthrough step lines
   flowing).
3. Smooth the leaves gauge (peak-or-mean over the flush window, or EWMA) — the sink already
   samples at each event; keep a tiny window instead of the instant.
4. ETA damping: suppress until n≥K completions AND the window spans ≥T seconds; consider
   EWMA on the rate; show a range or `~` marker while confidence is low. (The stall-decay
   property — rate → 0 on thrash — must SURVIVE damping; that alarm is the line's purpose.)

### Approved design — typed, time-driven board telemetry

#### Counter vocabulary

Rename the shared `progressCore` labels once (plain output and TUI derive from it; ARCH-DRY), while
letting the already-known `nested` state select truthful mode-specific vocabulary:

- `outer folds k/n` — completed held-out outer-fold evaluations.
- `configs scored k/n` — configs whose full inner resample completed, aggregated across outer folds.
- Nested: `inner-CV runs k/n` — completed `(outer fold × config × inner fold)` pipeline runs. Flat:
  `CV runs k/n` — completed folds in the one single-level CV. These replace the ambiguous `folds`;
  per-outer rows use `outer fold N` and the nested `configs scored` / `inner-CV runs` vocabulary
  with per-pass denominators. Startup, rate, and ETA wording uses the same mode-specific noun.

#### Typed activity seam

- Add a thin `activityExecutor` decorator around the **final** step executor, outside the cache
  decorator. It emits a typed successful-step completion after either a real execution or cache hit;
  no terminal text is parsed, and failed steps continue through the existing fatal error path without
  being counted as reassuring activity (ARCH-PURE).
- The same injected activity sink has a distinct successful-run event emitted by
  `runResolvedExperiment` only when execution has `runErr == nil` **and** required run-record /
  provenance persistence succeeds. A failed execution remains uncounted even when its failure record
  is written successfully. Its typed run role distinguishes nested inner-CV, flat CV, preamble, and
  outer-score runs; only the first two feed the corresponding run counter/rate/ETA. This records
  actual completion time at the concrete-run seam, rather than the later input-ordered `sampler.Run`
  delivery of a completed batch (ARCH-PURPOSE).
- Events carry an injected-clock time and typed identity. `sweepProgress` reduces successful-step
  events to a count/last-step time and eligible run events to the run count/rate window. Concurrent
  callback delivery may differ from event-time order, so each last-time is a max and the latest 64
  eligible run timestamps remain sorted by event time before readiness/rate math. Reversed-delivery
  traces pin both invariants. Callbacks are short and mutex-protected; non-sweep callers use a no-op
  sink.
- Before the first eligible run completes, the board reports observations, never an unprovable
  diagnosis. Nested example: `starting · ~8/12 subprocess slots · 37 steps completed · last step 1s
  ago · no inner-CV run complete`; flat output substitutes `CV run`. With occupancy but no successful
  step yet, it says only that slots are occupied. It never claims “not hung” or infers an upstream
  phase from a start line.
- The startup line disappears on the first eligible inner-CV/CV run. This remains correct both before
  and after #43: #43 shortens startup; #49 truthfully describes it.

#### Time-driven smoothing and ETA confidence

- Sample leaf occupancy on the existing 500ms board tick, not per event. Display the rolling mean of
  the last four samples (two seconds) rounded to a whole slot and prefixed `~`. Equal timestamped
  occupancy traces must render identically regardless of event density.
- Keep the latest 64 eligible run-completion timestamps, but withhold rate and ETA until at least 16
  completions span at least 15 seconds. Compute rate as `(n−1) / (now−oldest)`, removing the current
  early upward bias that counts `n` completions across `n−1` observed intervals.
- Continue using the current time in the denominator on every 500ms tick. With no new completions, the
  numeric rate is non-increasing and ETA is non-decreasing, although display rounding need not change
  within an arbitrary number of seconds after a long mature window. Preserve the fast, truthful stall
  signal separately: after startup, show `last inner-CV run 8s ago` (or `last CV run …` flat), updated
  on every tick. A scripted mature trace followed by five seconds of silence must advance that age by
  five seconds while rate/ETA move monotonically; it makes no diagnosis about why work is quiet
  (ARCH-PURPOSE). After a stall, the 64-completion window recovers gradually rather than snapping to
  one interarrival.
- Render the mature estimate as `~ETA …`; nested mode labels its rate `inner-CV runs/min` and scopes
  ETA to remaining inner-CV runs, while flat mode uses `CV runs/min` and remaining CV runs. Neither
  includes held-out scoring, capture, or final ledger work. Before confidence, show the corresponding
  mode-specific `— … runs/min` and no ETA rather than a volatile fiction.

#### Documentation boundary

Update the kbench Titanic sweep RUNBOOK's board description/example after metis#49 lands. Record the
exact peer commit in this issue's Log before close so the cross-repo requirement is reviewable.

## Done when

- Operator can read the first 2 minutes of a cold real sweep without wondering if it hung:
  factual startup activity present until the first inner-CV run completes; labels self-explanatory.
- Leaves + rate/ETA move smoothly (unit tests over scripted event traces pin the damping); after a
  mature trace, five seconds without a completion visibly advances the mode-specific last-run age by
  five seconds while the numeric rate/ETA move monotonically (regression test).
- Occupied-but-silent subprocesses are never described as proven progress; typed successful step
  completions/cache hits are the only positive activity signal.
- A failed run that successfully persists its failure record advances neither the eligible run
  counter nor rate/ETA (regression test).
- Flat and nested output, width clamping, repaint cadence, and terminal cleanup remain correct.
- RUNBOOK board description/example updated and its peer commit pinned in the Log.

## Plan

Durable plan: `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md`
(single pass, no Mx — one close boundary).

- [x] Add typed step/run activity at the concrete executor and persistence seams, including cache,
  failure, role, ordering, and cancellation tests.
- [x] Reduce time-driven occupancy and eligible-run telemetry with deterministic readiness, decay,
  recovery, and out-of-order-event tests.
- [x] Render truthful flat/nested vocabulary, startup observations, last-run age, and mature rate/ETA;
  preserve repaint, failure, width, and terminal behavior.
- [x] Update and commit the kbench Titanic RUNBOOK, pin its full commit here, then run focused,
  race, full-suite, formatting, and stale-vocabulary verification.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.05 impl=0.35
item: smaller-go-module   design=0.06 impl=0.35
item: tui-screen          design=0.15 impl=0.55
item: cross-cutting-refactor design=0.05 impl=0.25
item: smaller-go-module   design=0.06 impl=0.35
item: atlas-docs          design=0.02 impl=0.10
item: milestone-review    design=0.03 impl=0.20
design-buffer: 0.15
total: 2.63
```

Rows: (1) typed activity entities/decorator; (2) event-time reducer and rate/occupancy math;
(3) board state/rendering and scripted traces; (4) shared vocabulary migration; (5) concrete-run,
controller, and sweep wiring; (6) peer RUNBOOK plus atlas/stale-term sweep; (7) one SDLC milestone-review row.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

## Log

### 2026-07-16
- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
  #43 symptom: depth-first scheduling would give early train completions → a stable rate
  much sooner.

### 2026-07-16 — paired #43/#49 design approved
- Operator chose typed step-completion telemetry over presentation-only occupancy wording because
  occupancy cannot distinguish useful work from a hung subprocess. Co-designed after mapping the
  current sink/rate/compositor flow; #43 merges first, then #49 builds against its schedule.

### 2026-07-17 — planning checkpoint
- Reconciled the brain project after #43 merged, ran `sdlc start-plan`, mapped the final-executor,
  concrete-run persistence, run-control, sweep-progress, board, and kbench documentation seams, and
  authored the durable single-boundary TDD plan. Estimate uses v3.1 Method A; the approved spec and
  existing #38/#43 patterns make this familiar extension work rather than a novel TUI subsystem.
- Fresh-eyes plan review: Chunk 1 found Important gaps in shared run-control activity gating and
  concrete flat/nested role propagation; Chunk 2 found an Important gap in aggregate counter ownership.
  Patched the plan and both reviewers re-checked clean. Chunk 3 fresh-eyes review returned clean.

### 2026-07-17 — implementation verification
- Implemented typed step/run activity, run-role propagation, run-control-gated publication, event-time
  eligible-run rate reduction, tick-smoothed `~slots`, factual cold-start board wording, shared
  flat/nested progress vocabulary, and last-run-age/mature `~ETA` rendering. Updated atlas with the
  durable activity seam and board contract.
- Updated the kbench Titanic operator RUNBOOK and committed it as
  `68edc6ca312dbe5306ce87315abec2d67e678005` (`docs: update metis board progress contract`).
- Verification run: `go test ./cmd/metis -count=1`; `go test ./cmd/metis -race -count=1`;
  `go test ./... -race -count=1`; metis `git diff --check`; kbench `git diff --check`; stale board
  terminology grep over metis `cmd/`, `atlas/`, issue/plan, and the kbench RUNBOOK. Remaining grep hits
  are historical problem/spec wording or negative test fixtures, not live operator output.

## Revisions

### 2026-07-16 — fresh-eyes spec review
- Split flat `CV runs` from nested `inner-CV runs`, made out-of-order callback reduction explicit,
  and replaced an unprovable seconds-level rounded-rate promise with a tick-driven last-run-age
  freshness signal plus a measurable scripted-trace criterion. Separated typed step activity from
  actual concrete-run completion so batch-order sampler callbacks cannot falsify the rate window;
  required execution success independently of successful failure-record persistence.

### 2026-07-17 — durable implementation plan
- Replaced the two placeholder plan rows with concrete typed-activity, pure-reducer, rendering,
  cross-repo documentation, and verification steps; added the calibrated estimate and explicit
  single-close-boundary decision.

exec
/bin/zsh -lc 'git diff --name-only 72f2d895fc676a474304fb4cde8f4606e9c5f48c''^..HEAD' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
atlas/experiment.md
atlas/index.md
cmd/metis/blaspins.go
cmd/metis/blaspins_e2e_test.go
cmd/metis/blaspins_test.go
cmd/metis/board.go
cmd/metis/board_test.go
cmd/metis/capture.go
cmd/metis/capture_e2e_test.go
cmd/metis/exec.go
cmd/metis/exec_test.go
cmd/metis/forkexec.go
cmd/metis/forkexec_test.go
cmd/metis/identity_e2e_test.go
cmd/metis/main.go
cmd/metis/nestedcv_e2e_test.go
cmd/metis/parallel_test.go
cmd/metis/progress.go
cmd/metis/run.go
cmd/metis/runcontrol.go
cmd/metis/runcontrol_test.go
cmd/metis/select_cmd.go
cmd/metis/select_cmd_test.go
cmd/metis/shapesweep_test.go
cmd/metis/sweep.go
testdata/experiment/toy-sweep-smoke.md
testdata/steps/test/env-dump
workshop/history/issues/000022-ensembling-step.md
workshop/history/issues/000043-leaf-sched-depth-first.md
workshop/history/issues/000048-default-leaf-blas-pins.md
workshop/history/issues/000050-run-end-summary-elapsed-fingerprint-next-commands.md
workshop/history/issues/000052-select-cohort-flag-point-handles.md
workshop/history/plans/000043-leaf-sched-depth-first-close-review.md
workshop/history/plans/000043-leaf-sched-depth-first-plan.md
workshop/history/plans/000048-default-leaf-blas-pins-close-review.md
workshop/history/plans/000048-default-leaf-blas-pins-plan.md
workshop/history/plans/000050-run-end-summary-elapsed-fingerprint-next-commands-close-review.md
workshop/history/plans/000052-select-cohort-flag-point-handles-close-review.md
workshop/issues/000043-leaf-sched-depth-first.md
workshop/issues/000048-default-leaf-blas-pins.md
workshop/issues/000049-board-readability-labels-coldphase-eta.md
workshop/issues/000051-ledger-show-point-addr-column.md
workshop/issues/000053-promote-fingerprint-consistency-guard.md
workshop/lessons.md
workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md

exec
/bin/zsh -lc 'git diff --stat 72f2d895fc676a474304fb4cde8f4606e9c5f48c''^..HEAD' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
 atlas/experiment.md                                |   23 +-
 atlas/index.md                                     |   10 +-
 cmd/metis/blaspins.go                              |   45 +
 cmd/metis/blaspins_e2e_test.go                     |   74 +
 cmd/metis/blaspins_test.go                         |   61 +
 cmd/metis/board.go                                 |   13 +
 cmd/metis/board_test.go                            |  108 +-
 cmd/metis/capture.go                               |    6 +-
 cmd/metis/capture_e2e_test.go                      |    2 +-
 cmd/metis/exec.go                                  |    6 +
 cmd/metis/exec_test.go                             |   25 +
 cmd/metis/forkexec.go                              |   15 +-
 cmd/metis/forkexec_test.go                         |   53 +-
 cmd/metis/identity_e2e_test.go                     |    2 +-
 cmd/metis/main.go                                  |    2 +-
 cmd/metis/nestedcv_e2e_test.go                     |   11 +
 cmd/metis/parallel_test.go                         |  420 ++-
 cmd/metis/progress.go                              |   15 +-
 cmd/metis/run.go                                   |   42 +-
 cmd/metis/runcontrol.go                            |  104 +
 cmd/metis/runcontrol_test.go                       |  508 +++
 cmd/metis/select_cmd.go                            |   34 +-
 cmd/metis/select_cmd_test.go                       |   57 +
 cmd/metis/shapesweep_test.go                       |   21 +
 cmd/metis/sweep.go                                 |  305 +-
 testdata/experiment/toy-sweep-smoke.md             |   27 +
 testdata/steps/test/env-dump                       |    5 +
 .../{ => history}/issues/000022-ensembling-step.md |   18 +-
 .../issues/000043-leaf-sched-depth-first.md        |  230 ++
 .../issues/000048-default-leaf-blas-pins.md        |  125 +
 ...nd-summary-elapsed-fingerprint-next-commands.md |   88 +
 .../000052-select-cohort-flag-point-handles.md     |   78 +
 .../000043-leaf-sched-depth-first-close-review.md  | 3641 ++++++++++++++++++++
 .../plans/000043-leaf-sched-depth-first-plan.md    | 1017 ++++++
 .../000048-default-leaf-blas-pins-close-review.md  | 2300 +++++++++++++
 .../plans/000048-default-leaf-blas-pins-plan.md    |  446 +++
 ...apsed-fingerprint-next-commands-close-review.md |   64 +
 ...elect-cohort-flag-point-handles-close-review.md |   66 +
 workshop/issues/000043-leaf-sched-depth-first.md   |   43 -
 workshop/issues/000048-default-leaf-blas-pins.md   |   57 -
 ...00049-board-readability-labels-coldphase-eta.md |  193 +-
 .../issues/000051-ledger-show-point-addr-column.md |   42 +
 ...000053-promote-fingerprint-consistency-guard.md |   60 +
 workshop/lessons.md                                |   18 +-
 ...-board-readability-labels-coldphase-eta-plan.md |  202 ++
 45 files changed, 10404 insertions(+), 278 deletions(-)

codex
The issue spec is explicit about typed activity telemetry, renamed board vocabulary, startup wording, `~slots`, and confidence-gated ETA. The changed-file list is already suspicious: much of the diff is #48/#50/#52 and run-control work, so I’m checking whether the promised #49 surfaces actually exist in code rather than assuming the plan checkboxes are accurate.
exec
/bin/zsh -lc "sed -n '1,460p' cmd/metis/progress.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

// metis#30: the sweep progress sink — folds pkg/sampler's per-completion
// ProgressEvents (typed per level) into ONE throttled aggregated line, so a
// 2,000-fold sweep reports live without a per-fold firehose (single-threaded-
// attention budget). Plain appended lines, no escape codes — non-TTY-safe by
// construction; the TTY board is metis#38, which extends this sink behind the
// same per-pass hooks (outer-fold identity rides the forPass closure binding,
// NEVER an event payload field — pkg/sampler stays coordinate-free).

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/metis/pkg/sampler"
	"github.com/xianxu/metis/pkg/shape"
)

// progressTotals seeds the sink with each level's SizeHint AT WIRING TIME —
// stream-learned totals arrive only with a level's first completion (for the
// driver level: the first COMPLETED outer fold, near the end of a parallel run),
// which starves the display of denominators. cmd/metis constructs the samplers,
// so it reads SizeHint directly; SizeHint stays the single source (ARCH-DRY).
type progressTotals struct {
	nested     bool
	outer      int
	outerKind  sampler.SizeKind
	configs    int // aggregate across outer folds (outer × per-pass configs)
	configKind sampler.SizeKind
	folds      int // aggregate leaf count (outer × configs × inner k)
	foldKind   sampler.SizeKind
}

// progressState is the pure render input: sink-owned aggregate counters per level
// (NEVER ev.K — each concurrent Run instance counts its own 1..k), the completed
// outer scores (→ est mean±SE), and the flat path's running fold scores.
type progressState struct {
	nested               bool
	outerK, outerTotal   int
	outerKind            sampler.SizeKind
	configK, configTotal int
	configKind           sampler.SizeKind
	foldK, foldTotal     int
	foldKind             sampler.SizeKind
	stepK                int
	lastStepAt           time.Time
	lastRunAt            time.Time
	outerScores          []float64 // nested: completed outer-fold held-out scores
	flatScores           []float64 // flat: the one config's completed fold scores
}

// progressLine renders the aggregated line. Nested:
// `outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`
// (est — until an outer fold lands; ± only at n≥2). Flat (since metis#32: iff 1
// config): `CV runs 3/5 · score 0.8400` (the running fold mean — nothing to be
// "best" of). Kinds render k/n (exact), k/≤n (budget), k/? (unknown). Pure.
func progressLine(st progressState) string {
	return "metis: progress " + progressCore(st)
}

// progressCore is the un-prefixed aggregate segment — shared by the plain line and
// the board's first row (extracted so the board never string-strips the prefix;
// a TrimPrefix coupling would silently no-op if the prefix changed — close review).
func progressCore(st progressState) string {
	frac := func(k, total int, kind sampler.SizeKind) string {
		switch kind {
		case sampler.SizeExact:
			return fmt.Sprintf("%d/%d", k, total)
		case sampler.SizeBudget:
			return fmt.Sprintf("%d/≤%d", k, total)
		default:
			return fmt.Sprintf("%d/?", k)
		}
	}
	var parts []string
	if st.nested {
		parts = append(parts, "outer folds "+frac(st.outerK, st.outerTotal, st.outerKind))
		parts = append(parts, "configs scored "+frac(st.configK, st.configTotal, st.configKind))
		parts = append(parts, "inner-CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
		mean, se, n := meanSE(st.outerScores)
		switch {
		case n == 0:
			parts = append(parts, "est —")
		case n == 1:
			parts = append(parts, fmt.Sprintf("est %.4f", mean))
		default:
			parts = append(parts, fmt.Sprintf("est %.4f ± %.4f", mean, se))
		}
	} else {
		parts = append(parts, "CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
		if mean, _, n := meanSE(st.flatScores); n > 0 {
			parts = append(parts, fmt.Sprintf("score %.4f", mean))
		}
	}
	return strings.Join(parts, " · ")
}

// meanSE is the display-only mean ± standard-error reduce over completed scores.
// Computed locally: the honest estimate stays pkg/sampler's Aggregate/MeanSE —
// this is presentation, not selection (do not export sampler surface for it).
func meanSE(xs []float64) (mean, se float64, n int) {
	n = len(xs)
	if n == 0 {
		return 0, 0, 0
	}
	for _, x := range xs {
		mean += x
	}
	mean /= float64(n)
	if n < 2 {
		return mean, 0, n
	}
	var ss float64
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	se = math.Sqrt(ss/float64(n-1)) / math.Sqrt(float64(n))
	return mean, se, n
}

// seededTotals reads each level's SizeHint on its initial state — the SAME source
// the Run loops stamp on events (ARCH-DRY; no shape math re-derived here) — and
// composes the aggregate denominators: configs = outer × per-pass configs,
// folds = outer × configs × inner k (each sealed pass sweeps the full grid).
// Flat (1 config): folds = the single pass's inner k.
func seededTotals(ctx sampler.Ctx, nested bool, runFolds int, configPts []shape.Point, k int) progressTotals {
	grid := sampler.GridConfigs{Points: configPts}
	nConfigs, kindConfigs := grid.SizeHint(grid.Init(ctx))
	foldsSmp := sampler.FixedKFolds{K: k}
	nFolds, kindFolds := foldsSmp.SizeHint(foldsSmp.Init(ctx))
	if !nested {
		return progressTotals{folds: nFolds, foldKind: kindFolds}
	}
	cv := sampler.CVDriver{K: runFolds}
	nOuter, kindOuter := cv.SizeHint(cv.Init(ctx))
	return progressTotals{
		nested: true,
		outer:  nOuter, outerKind: kindOuter,
		configs: nOuter * nConfigs, configKind: kindConfigs,
		folds: nOuter * nConfigs * nFolds, foldKind: kindFolds,
	}
}

// movingRate retains the latest eligible run completions by event time. rate(now)
// = (n-1)/(now-oldest) after the confidence gate; `now` in the denominator means
// a STALL decays live while last-run age remains the sharp freshness signal.
type movingRate struct {
	times []time.Time
}

func (m *movingRate) add(t time.Time) {
	i := sort.Search(len(m.times), func(i int) bool { return !m.times[i].Before(t) })
	m.times = append(m.times, time.Time{})
	copy(m.times[i+1:], m.times[i:])
	m.times[i] = t
	if len(m.times) > 64 {
		m.times = m.times[1:]
	}
}

// rate returns eligible runs/minute over the kept event-time window.
func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
	if len(m.times) < 16 {
		return 0, false
	}
	oldest := m.times[0]
	newest := m.times[len(m.times)-1]
	if newest.Sub(oldest) < 15*time.Second {
		return 0, false
	}
	mins := now.Sub(oldest).Minutes()
	if mins <= 0 {
		return 0, false
	}
	return float64(len(m.times)-1) / mins, true
}

// eta = remaining / rate; ok=false when the rate is unavailable or zero.
func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
	r, ok := m.rate(now)
	if !ok || r <= 0 || remaining <= 0 {
		return 0, false
	}
	return time.Duration(float64(remaining) / r * float64(time.Minute)), true
}

type occupancySample struct {
	busy, capacity int
}

type occupancyWindow struct {
	samples []occupancySample
}

func (w *occupancyWindow) add(busy, capacity int) {
	if capacity <= 0 {
		return
	}
	w.samples = append(w.samples, occupancySample{busy: busy, capacity: capacity})
	if len(w.samples) > 4 {
		w.samples = w.samples[1:]
	}
}

func (w occupancyWindow) mean() (busy, capacity int, ok bool) {
	if len(w.samples) == 0 {
		return 0, 0, false
	}
	var sum int
	for _, s := range w.samples {
		sum += s.busy
		capacity = s.capacity
	}
	return int(math.Round(float64(sum) / float64(len(w.samples)))), capacity, true
}

// passRow is one outer fold's live board row (metis#38): in-flight counters + the
// pass's incumbent best (display-only — NOT the 1-SE select rule), collapsing to its
// held-out score when the driver reports the fold done.
type passRow struct {
	configK, foldK int
	best           float64
	hasBest        bool
	done           bool
	heldOut        float64
}

// boardState is the pure render input for metis#38's board: the #30 aggregate state
// plus the per-pass rows and the throughput ring (a mutex'd snapshot — renderers never
// touch the live sink).
type boardState struct {
	st   progressState
	rows []passRow
	rate movingRate
}

// sweepProgress is the mutex'd sink shared by every pass of one shape-run. Events
// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
// its own Run's event mutex); health-gated paths use the strict order runControl.mu
// → sink.mu → boardWriter.mu (never the reverse). Emit policy: fold/config events are throttled
// to one line per second (injected clock — tests script it, never sleep); a
// driver-level (outer fold) completion ALWAYS emits; finish() emits the terminal
// line. A nil *sweepProgress is a no-op everywhere (the non-sweep path is silent).
type sweepProgress struct {
	mu        sync.Mutex
	out       io.Writer
	now       func() time.Time
	direction string // the objective direction — orients each pass's display-best (#38)
	st        progressState
	rows      []passRow  // metis#38: one row per outer fold (nil on the flat path)
	rate      movingRate // metis#38: fold-completion throughput window
	lastEmit  time.Time
	started   bool
	// metis#38 board mode (all nil/zero in plain mode): emits paint the rendered frame
	// to bw instead of printing plain lines. Lock order: sink.mu → bw.mu, ALWAYS — the
	// ticker enters via tick() (a sink method), never a boardWriter-first path.
	bw        *boardWriter
	width     int               // terminal width ($COLUMNS | 80), read once at wiring
	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
	occupancy occupancyWindow
}

func newSweepProgress(out io.Writer, now func() time.Time, direction string, totals progressTotals) *sweepProgress {
	var rows []passRow
	if totals.nested && totals.outer > 0 {
		rows = make([]passRow, totals.outer)
	}
	return &sweepProgress{
		out: out, now: now, direction: direction, rows: rows,
		st: progressState{
			nested:     totals.nested,
			outerTotal: totals.outer, outerKind: totals.outerKind,
			configTotal: totals.configs, configKind: totals.configKind,
			foldTotal: totals.folds, foldKind: totals.foldKind,
		},
	}
}

// boardState snapshots the sink for a renderer (rows copied — the caller may hold
// the snapshot without racing the live fold-in).
func (sp *sweepProgress) boardState() boardState {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.snapshotLocked()
}

// snapshotLocked builds the render snapshot; caller holds sp.mu (shared by
// boardState() and the board-mode emit — one copy site, close-review DRY note).
func (sp *sweepProgress) snapshotLocked() boardState {
	rows := make([]passRow, len(sp.rows))
	copy(rows, sp.rows)
	return boardState{st: sp.st, rows: rows, rate: sp.rate}
}

// passHooks are one pass's typed event targets, closure-bound to its outer-fold
// index (-1 = the flat path's single pass) — the metis#38 identity seam.
type passHooks struct {
	config func(sampler.ProgressEvent[shape.Point, sampler.MeanSE])
	fold   func(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome])
}

// forPass hands out a pass's hooks. #30's sink aggregates across passes (the
// single-line mental model); the per-pass binding exists so #38 can add per-fold
// board rows behind the same API without touching pkg/sampler.
func (sp *sweepProgress) forPass(outer int) passHooks {
	if sp == nil {
		return passHooks{
			config: func(sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {},
			fold:   func(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {},
		}
	}
	return passHooks{
		config: func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
			sp.mu.Lock()
			defer sp.mu.Unlock()
			sp.st.configK++
			if outer >= 0 && outer < len(sp.rows) { // #38: this pass's row
				r := &sp.rows[outer]
				r.configK++
				if !r.hasBest || better(sp.direction, ev.Out.Mean, r.best) {
					r.best, r.hasBest = ev.Out.Mean, true
				}
			}
			sp.maybeEmit(false)
		},
		fold: func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
			sp.mu.Lock()
			defer sp.mu.Unlock()
			if !sp.st.nested {
				sp.st.flatScores = append(sp.st.flatScores, ev.Out.Score)
			}
			if outer >= 0 && outer < len(sp.rows) {
				sp.rows[outer].foldK++
			}
			sp.maybeEmit(false)
		},
	}
}

func (sp *sweepProgress) activity(ev activityEvent) {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	switch ev.Kind {
	case activityStepSuccess:
		sp.st.stepK++
		at := ev.At
		if at.IsZero() {
			at = sp.now()
		}
		if at.After(sp.st.lastStepAt) {
			sp.st.lastStepAt = at
		}
	case activityRunSuccess:
		if ev.Role != runRoleNestedInnerCV && ev.Role != runRoleFlatCV {
			return
		}
		sp.st.foldK++
		at := ev.At
		if at.IsZero() {
			at = sp.now()
		}
		if at.After(sp.st.lastRunAt) {
			sp.st.lastRunAt = at
		}
		sp.rate.add(at)
	default:
		return
	}
	sp.maybeEmit(false)
}

// better orients a display-best comparison by the objective direction.
func better(direction string, candidate, incumbent float64) bool {
	if direction == "minimize" {
		return candidate < incumbent
	}
	return candidate > incumbent
}

// driverEvent folds a completed OUTER fold in — always emits (the coarse level is
// the one the operator watches; its completions are rare and load-bearing).
func (sp *sweepProgress) driverEvent(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.st.outerK++
	sp.st.outerScores = append(sp.st.outerScores, ev.Out)
	if i := ev.Point.Idx; i >= 0 && i < len(sp.rows) { // #38: collapse this fold's row
		sp.rows[i].done = true
		sp.rows[i].heldOut = ev.Out
	}
	sp.maybeEmit(true)
}

// finish emits the terminal state line (always).
func (sp *sweepProgress) finish() {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.emit()
}

// tick is the board ticker's entry point (metis#38): repaint with a fresh `now` so
// the rate decay + ETA move even between events. Sink-first (sp.mu → bw.mu).
func (sp *sweepProgress) tick() {
	if sp == nil || sp.bw == nil {
		return
	}
	sp.mu.Lock()
	if sp.gauge != nil {
		busy, capacity := sp.gauge()
		sp.occupancy.add(busy, capacity)
	}
	sp.emit() // stores the fresh frame (budget may skip the draw)
	bw := sp.bw
	sp.mu.Unlock()
	bw.forceFlush() // metis#46: the tick is what re-pins the board after a burst window
}

// abort removes the stored live frame after a sweep failure. Lock order remains
// progress -> board; the controller is never called while either lock is held.
func (sp *sweepProgress) abort() {
	if sp == nil || sp.bw == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.bw.discardFrame()
}

// maybeEmit writes the line if forced (driver/finish) or the throttle elapsed —
// 1s for plain lines (a log is a record), 100ms for board repaints (a board is a
// display; the 500ms ticker guarantees freshness anyway). Caller holds sp.mu.
func (sp *sweepProgress) maybeEmit(force bool) {
	now := sp.now()
	throttle := time.Second
	if sp.bw != nil {
		throttle = 100 * time.Millisecond
	}
	if !force && sp.started && now.Sub(sp.lastEmit) < throttle {
		return
	}
	sp.started = true
	sp.lastEmit = now
	sp.emit()
}

// emit renders the current state: board mode paints the frame (under the fixed

exec
/bin/zsh -lc 'rg -n "activityEvent|activityExecutor|runRole|occupancyWindow|movingRate|inner-CV runs|CV runs|configs scored|outer folds|~slots|last .* run|no inner-CV|no CV run|runs/min|ETA|leaves|folds/min|folds " cmd/metis' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
cmd/metis/progress.go:3:// metis#30: the sweep progress sink — folds pkg/sampler's per-completion
cmd/metis/progress.go:33:	configs    int // aggregate across outer folds (outer × per-pass configs)
cmd/metis/progress.go:35:	folds      int // aggregate leaf count (outer × configs × inner k)
cmd/metis/progress.go:58:// `outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`
cmd/metis/progress.go:60:// config): `CV runs 3/5 · score 0.8400` (the running fold mean — nothing to be
cmd/metis/progress.go:82:		parts = append(parts, "outer folds "+frac(st.outerK, st.outerTotal, st.outerKind))
cmd/metis/progress.go:83:		parts = append(parts, "configs scored "+frac(st.configK, st.configTotal, st.configKind))
cmd/metis/progress.go:84:		parts = append(parts, "inner-CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
cmd/metis/progress.go:95:		parts = append(parts, "CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
cmd/metis/progress.go:129:// folds = outer × configs × inner k (each sealed pass sweeps the full grid).
cmd/metis/progress.go:130:// Flat (1 config): folds = the single pass's inner k.
cmd/metis/progress.go:149:// movingRate retains the latest eligible run completions by event time. rate(now)
cmd/metis/progress.go:152:type movingRate struct {
cmd/metis/progress.go:156:func (m *movingRate) add(t time.Time) {
cmd/metis/progress.go:166:// rate returns eligible runs/minute over the kept event-time window.
cmd/metis/progress.go:167:func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
cmd/metis/progress.go:184:func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
cmd/metis/progress.go:196:type occupancyWindow struct {
cmd/metis/progress.go:200:func (w *occupancyWindow) add(busy, capacity int) {
cmd/metis/progress.go:210:func (w occupancyWindow) mean() (busy, capacity int, ok bool) {
cmd/metis/progress.go:239:	rate movingRate
cmd/metis/progress.go:243:// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
cmd/metis/progress.go:256:	rate      movingRate // metis#38: fold-completion throughput window
cmd/metis/progress.go:264:	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
cmd/metis/progress.go:265:	occupancy occupancyWindow
cmd/metis/progress.go:345:func (sp *sweepProgress) activity(ev activityEvent) {
cmd/metis/progress.go:362:		if ev.Role != runRoleNestedInnerCV && ev.Role != runRoleFlatCV {
cmd/metis/progress.go:388:// driverEvent folds a completed OUTER fold in — always emits (the coarse level is
cmd/metis/progress.go:416:// the rate decay + ETA move even between events. Sink-first (sp.mu → bw.mu).
cmd/metis/blaspins.go:7:// leaf semaphore), not to each leaf's BLAS — NumCPU leaves × multi-threaded BLAS
cmd/metis/main.go:45:	sampleN := fs.Int("sample", 0, "metis#42: run m of the k outer folds (sparse fold sampling; 0/omitted = all k). k stays the estimand (each fold trains on (k-1)/k of the rows); m only trades precision for cost — use to probe a higher k (e.g. k=10, --sample 3) without the full k× bill. The SE over m<k folds is noisy (m-1 df): probe with it, don't re-select what ships on it. Errors on m>k, on a single-config (flat) run, and combined with --fast.")
cmd/metis/fingerprints_test.go:17:// folds record metadata (first/last timestamps, latest commit/dirty/status, distinct extra
cmd/metis/shipe2e_test.go:38:	// Declare this step's read-set D (the code file it reads) so recordMiss folds it into the
cmd/metis/shipe2e_test.go:59:// inner folds → an honest per-config (mean, SE) leaderboard, argmax-mean winner, driver:single
cmd/metis/shipe2e_test.go:61:// re-runs the downstream folds through the sweep while the config/fold-invariant data + partition
cmd/metis/capture_e2e_test.go:19:// call site leaves the direct-helper tests green but must fail THIS one (the Done-when is
cmd/metis/forkexec_test.go:200:// TestForkServerPerf_LooseBound (metis#44 acceptance): N leaves that each import pandas —
cmd/metis/forkexec_test.go:250:	// (server start + preload amortize over ~5k leaves in a real sweep, not over n=4).
cmd/metis/caching.go:55:// (from the sensor's reads.json), folds the transitive-D snapshot, stores the output, and
cmd/metis/caching.go:112:			// downstream step in the same run still folds an upstream HIT's closure into
cmd/metis/caching.go:186:	// side (recordMiss folds + stores the SAME closure) — store and HIT-check can't disagree.
cmd/metis/select_cmd.go:240:			fmt.Fprintf(out, "    %-24s mean %.4f  SE %.4f  (n=%d outer folds)\n", famLabel(f), est[f].Mean, est[f].SE, len(est[f].ToldSet))
cmd/metis/family.go:10:// reduce the objective metric over the outer folds via the shared sampler.Aggregate →
cmd/metis/family.go:14:// outer folds (outer-fold 0 → rf md=4, outer-fold 1 → rf md=8), so those rows share the family
cmd/metis/family.go:16:// them in separate groups and never compute the per-family mean over the outer folds. Only
cmd/metis/family.go:18:// AggregateView, which pools a config's inner folds across outer folds), not this.
cmd/metis/caching_test.go:157:	var stepEvents []activityEvent
cmd/metis/caching_test.go:168:			activity: func(ev activityEvent) {
cmd/metis/caching_test.go:448:	// guard at the executor level — two folds of one step get distinct entries).
cmd/metis/run_test.go:195:	var runEvents []activityEvent
cmd/metis/run_test.go:204:		activity: func(ev activityEvent) {
cmd/metis/run_test.go:228:	if runEvents[0].Role != runRoleNone {
cmd/metis/run_test.go:246:	var runEvents []activityEvent
cmd/metis/run_test.go:254:		activity: func(ev activityEvent) {
cmd/metis/run_test.go:290:	var runEvents []activityEvent
cmd/metis/run_test.go:298:		activity: func(ev activityEvent) {
cmd/metis/run_test.go:343:	// A rejected run writes no runs/ dir and leaves the config .md untouched (#13 — the
cmd/metis/sweep.go:28:// foldMetric is the per-fold score the resample folds over — the metric the train step
cmd/metis/sweep.go:35:// fold_score (metis#19): the fitted model's realized complexity (rf mean leaves / logreg
cmd/metis/sweep.go:74:// shapeSweep is the mutable accumulator the pure nested-Sampler loop folds through the IO
cmd/metis/sweep.go:87:	manMu         sync.Mutex     // metis#32: guards man.Points — concurrent outer folds (ParExec) each record rows
cmd/metis/sweep.go:93:// nested run's outer folds run concurrently under ParExec, each recording its inner+outer rows).
cmd/metis/sweep.go:128:	runRole  runRole              // concrete-run role for every pipeline fold in this pass
cmd/metis/sweep.go:131:	// resample fans out over folds — all appending to this ONE pass. `mu` guards the
cmd/metis/sweep.go:213:	// a plain k-fold of the one config on ALL data, not the nested subset-sweep). The outer folds
cmd/metis/sweep.go:230:			return fmt.Errorf("run: --sample only applies to a nested (multi-config) run — this shape has 1 config, a flat CV with no outer folds to sample")
cmd/metis/sweep.go:245:			fmt.Fprintf(out, "metis: single-level CV %s — %d config × %d folds (dry run):\n", sh.ID, len(configPts), k)
cmd/metis/sweep.go:270:	// and a 500ms ticker keeps the rate decay + ETA live between events (sink-first:
cmd/metis/sweep.go:324:	pass := &sweepPass{ss: ss, splitK: k, stratify: stratify, partRef: ss.partRef, runRole: runRoleFlatCV,
cmd/metis/sweep.go:361:	// after the folds run.
cmd/metis/sweep.go:447:	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
cmd/metis/sweep.go:490:	preOpts.runRole = runRoleNestedPreamble
cmd/metis/sweep.go:510:	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
cmd/metis/sweep.go:513:		runRole: runRoleNestedInnerCV,
cmd/metis/sweep.go:592:	scoreOpts.runRole = runRoleOuterScore
cmd/metis/sweep.go:611:// reportEstimate prints the honest procedure estimate — mean±SE over the outer folds — and the
cmd/metis/sweep.go:653:	pointOpts.runRole = p.runRole
cmd/metis/sweep.go:818:	fmt.Fprintf(ss.out, "metis: sweep %s done — %d configs scored (manifest %s)\n", ss.sh.ID, len(ss.configs), ss.man.ShapeRunID[:12])
cmd/metis/sweep.go:950:		fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort ?)\n", fmtETA(elapsed), rows, base)
cmd/metis/sweep.go:957:	fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort %s)\n", fmtETA(elapsed), rows, base, fp)
cmd/metis/trace.go:18:// site-packages (→ the uv.lock digest folds into D per-root, below).
cmd/metis/forkexec.go:335:			p.noticeOnce("root:"+spec.root, fmt.Sprintf("server for %s died mid-flight (%v) — erroring the step; later leaves use legacy", spec.root, err))
cmd/metis/progress_test.go:31:			[]string{"outer folds 0/3", "configs scored 84/216", "inner-CV runs 421/1080", "est —"}, []string{"±"}},
cmd/metis/progress_test.go:33:			[]string{"outer folds 1/3", "est 0.8200"}, []string{"±"}},
cmd/metis/progress_test.go:35:			[]string{"outer folds 2/3", "est 0.8200 ± 0.0200"}, nil},
cmd/metis/progress_test.go:39:			[]string{"CV runs 3/5", "score 0.8400"}, []string{"configs", "outer", "folds 3/5"}},
cmd/metis/progress_test.go:44:			[]string{"outer folds 1/?", "configs scored 3/?"}, nil},
cmd/metis/progress_test.go:48:			[]string{"outer folds 1/≤8"}, nil},
cmd/metis/progress_test.go:95:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 200)})
cmd/metis/progress_test.go:111:	for _, w := range []string{"outer folds 1/2", "inner-CV runs 10/20", "est 0.8300"} {
cmd/metis/progress_test.go:144:				prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
cmd/metis/progress_test.go:150:	if !strings.Contains(out.String(), "inner-CV runs 64/64") {
cmd/metis/progress_test.go:160:// movingRate: keep the latest 64 eligible completion times by event time. It is
cmd/metis/progress_test.go:163:	var r movingRate
cmd/metis/progress_test.go:173:	var short movingRate
cmd/metis/progress_test.go:180:	var ready movingRate
cmd/metis/progress_test.go:192:	var r2 movingRate
cmd/metis/progress_test.go:200:	// ETA: remaining/rate.
cmd/metis/progress_test.go:207:	var w occupancyWindow
cmd/metis/progress_test.go:241:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
cmd/metis/progress_test.go:256:// Per-pass rows: each forPass(i) hook folds into ITS row (closure-bound identity);
cmd/metis/progress_test.go:326:	prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedPreamble, RunID: "pre", At: at(0)})
cmd/metis/progress_test.go:332:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, RunID: "inner", At: at(i * 1000)})
cmd/metis/board.go:19:// (for rate/ETA), and the leaf-semaphore gauge (capacity 0 = no gauge → segment
cmd/metis/board.go:33:// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
cmd/metis/board.go:64:			lines = append(lines, fmt.Sprintf("  outer fold %d ▸ configs scored %d/%d · inner-CV runs %d/%d%s",
cmd/metis/board.go:72:	// Leaves / throughput / ETA.
cmd/metis/board.go:74:	runNoun := "CV runs"
cmd/metis/board.go:77:		runNoun = "inner-CV runs"
cmd/metis/board.go:81:		segs = append(segs, fmt.Sprintf("~slots %d/%d", env.busy, env.capacity))
cmd/metis/board.go:100:		segs = append(segs, fmt.Sprintf("last %s %s ago", runSingular, fmtAge(env.now.Sub(bs.st.lastRunAt))))
cmd/metis/board.go:109:			segs = append(segs, "~ETA "+fmtETA(eta))
cmd/metis/board.go:124:	return fmtETA(d)
cmd/metis/board.go:127:// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
cmd/metis/board.go:128:func fmtETA(d time.Duration) string {
cmd/metis/board.go:234:// (this is what re-pins the board after a burst window and keeps ETA/rate moving).
cmd/metis/activity.go:16:type runRole string
cmd/metis/activity.go:19:	runRoleNone           runRole = ""
cmd/metis/activity.go:20:	runRoleNestedInnerCV  runRole = "nested-inner-cv"
cmd/metis/activity.go:21:	runRoleFlatCV         runRole = "flat-cv"
cmd/metis/activity.go:22:	runRoleNestedPreamble runRole = "nested-preamble"
cmd/metis/activity.go:23:	runRoleOuterScore     runRole = "outer-score"
cmd/metis/activity.go:26:type activityEvent struct {
cmd/metis/activity.go:31:	Role   runRole
cmd/metis/activity.go:34:type activityEmitter func(activityEvent)
cmd/metis/activity.go:36:func (e activityEmitter) emit(ev activityEvent) {
cmd/metis/activity.go:49:	return func(ev activityEvent) {
cmd/metis/activity.go:59:	return func(ev activityEvent) {
cmd/metis/activity.go:64:type activityExecutor struct {
cmd/metis/activity.go:70:func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
cmd/metis/activity.go:79:	e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
cmd/metis/run.go:73:	sample   int  // metis#42: nested run does m of the k outer folds (0 = all k). k stays the
cmd/metis/run.go:83:	runRole     runRole         // role for successful concrete-run activity; zero = ineligible/non-sweep
cmd/metis/run.go:99:	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
cmd/metis/run.go:107:// the inner resample folds each — runShapeSweep); a plain `type: experiment` is the
cmd/metis/run.go:254:	exec = activityExecutor{inner: exec, now: now, emit: o.activity}
cmd/metis/run.go:304:	o.activity.emit(activityEvent{Kind: activityRunSuccess, At: now(), RunID: runID, Role: o.runRole})
cmd/metis/runcontrol_test.go:512:	var events []activityEvent
cmd/metis/runcontrol_test.go:513:	emit := runControlActivityEmitter(control, func(ev activityEvent) {
cmd/metis/runcontrol_test.go:517:	emit(activityEvent{Kind: activityStepSuccess, StepID: "prep"})
cmd/metis/runcontrol_test.go:523:	emit(activityEvent{Kind: activityStepSuccess, StepID: "late-step"})
cmd/metis/runcontrol_test.go:524:	emit(activityEvent{Kind: activityRunSuccess, RunID: "late-run"})
cmd/metis/shapesweep_test.go:61:// nudge, so distinct models have distinct MEANS (winner selection) and distinct folds give
cmd/metis/shapesweep_test.go:75:// fixed value per model, fold-independent (like a tree's realized leaves), so a config's
cmd/metis/shapesweep_test.go:85:// isolation (shipWinner no-ops on an empty ship). 2 configs × 2 folds = 4 per-fold runs.
cmd/metis/shapesweep_test.go:176:// family) × 2 inner × 2 outer folds → 8 inner + 2 outer rows.
cmd/metis/shapesweep_test.go:207:		t.Errorf("2 configs × 2 inner × 2 outer folds → 8 inner rows, got %d", nInner)
cmd/metis/shapesweep_test.go:210:		t.Errorf("1 family × 2 outer folds → 2 outer rows, got %d", nOuter)
cmd/metis/shapesweep_test.go:213:	// The honest procedure estimate (mean±SE over outer folds) is reported — NOT a single winner
cmd/metis/shapesweep_test.go:332:		if !strings.Contains(final, "CV runs 2/2") || !strings.Contains(final, "score 0.") {
cmd/metis/shapesweep_test.go:333:			t.Errorf("the flat final progress line must carry folds k/k + score; got: %q", final)
cmd/metis/shapesweep_test.go:356:	t.Run("flat CV runs are eligible flat roles", func(t *testing.T) {
cmd/metis/shapesweep_test.go:359:		counts := map[runRole]int{}
cmd/metis/shapesweep_test.go:363:			activity: func(ev activityEvent) {
cmd/metis/shapesweep_test.go:372:		if counts[runRoleFlatCV] != 2 || len(counts) != 1 {
cmd/metis/shapesweep_test.go:373:			t.Fatalf("flat roles = %v; want exactly 2 flat-CV runs", counts)
cmd/metis/shapesweep_test.go:380:		counts := map[runRole]int{}
cmd/metis/shapesweep_test.go:384:			activity: func(ev activityEvent) {
cmd/metis/shapesweep_test.go:393:		if counts[runRoleNestedPreamble] != 1 {
cmd/metis/shapesweep_test.go:396:		if counts[runRoleNestedInnerCV] != 8 {
cmd/metis/shapesweep_test.go:397:			t.Fatalf("nested roles = %v; want 8 inner-CV runs", counts)
cmd/metis/shapesweep_test.go:399:		if counts[runRoleOuterScore] != 2 {
cmd/metis/family_test.go:11:// the outer folds — the reduction AggregateView cannot do, because a family's winner differs
cmd/metis/family_test.go:12:// across outer folds (rf md=4 in fold 0, rf md=8 in fold 1) → distinct free-params, same family.
cmd/metis/family_test.go:23:	// rf winner differs across outer folds (md4 then md8) — same family, must pool.
cmd/metis/family_test.go:44:		t.Errorf("rf family mean=%v want 0.79 (pooled two outer folds, inner row ignored)", rf.Mean)
cmd/metis/family_test.go:47:		t.Errorf("rf family SE should be non-zero over 2 outer folds")
cmd/metis/exec.go:60:		// driver:single path leaves the var unset (unconfined).
cmd/metis/identity_e2e_test.go:75:// a real git blob, which is exactly what a traced run leaves behind.
cmd/metis/identity_e2e_test.go:92:	// record.json (no fingerprint yet) — the state a run leaves for capture to backfill.
cmd/metis/board_test.go:14:// queued), overflow cap, leaves+throughput line. NO ANSI — escape codes live only in
cmd/metis/board_test.go:41:		"outer folds 1/3", "configs scored 14/36", "inner-CV runs 47/108", "est 0.7980",
cmd/metis/board_test.go:43:		"outer fold 1 ▸ configs scored 8/12 · inner-CV runs 25/36 · best 0.8340",
cmd/metis/board_test.go:45:		"~slots 8/8", "42.5 inner-CV runs/min", "~ETA",
cmd/metis/board_test.go:51:	if len(lines) != 5 { // aggregate + 3 fold rows + leaves
cmd/metis/board_test.go:59:	// All-done: every row ✓, no ETA segment (nothing remaining).
cmd/metis/board_test.go:65:	if strings.Contains(frame, "▸") || strings.Contains(frame, "ETA") {
cmd/metis/board_test.go:66:		t.Errorf("all-done: no in-flight rows, no ETA:\n%s", frame)
cmd/metis/board_test.go:72:		t.Errorf("flat board = aggregate + leaves, got %d lines: %v", len(got), got)
cmd/metis/board_test.go:75:	// Overflow: 14 folds → 12 rows + "… +2 more" + slots/rate + aggregate = 15 lines.
cmd/metis/board_test.go:101:	if !strings.Contains(frame, "inner-CV runs/min") {
cmd/metis/board_test.go:105:	// Rate unavailable (fresh ring): "— inner-CV runs/min".
cmd/metis/board_test.go:107:	st.rate = movingRate{}
cmd/metis/board_test.go:109:	if !strings.Contains(frame, "— inner-CV runs/min") {
cmd/metis/board_test.go:122:		"starting", "~slots 8/12", "37 steps completed", "last step 1s ago", "no inner-CV run complete",
cmd/metis/board_test.go:137:	if strings.Contains(frame, "starting") || strings.Contains(frame, "no inner-CV run complete") {
cmd/metis/board_test.go:150:		"CV runs 0/20", "starting", "~slots 3/8", "4 steps completed", "last step 2s ago", "no CV run complete",
cmd/metis/board_test.go:163:	for _, want := range []string{"CV runs 1/20", "last CV run 1s ago", "— CV runs/min"} {
cmd/metis/board_test.go:168:	if strings.Contains(frame, "starting") || strings.Contains(frame, "ETA") {
cmd/metis/board_test.go:169:		t.Fatalf("flat pre-confidence frame must be post-startup without ETA:\n%s", frame)
cmd/metis/board_test.go:175:	var rate movingRate
cmd/metis/board_test.go:187:	for _, want := range []string{"45.0 inner-CV runs/min", "last inner-CV run 5s ago", "~ETA"} {
cmd/metis/board_test.go:194:// fmtETA is compact and human: seconds under a minute, m+s under an hour.
cmd/metis/board_test.go:195:func TestFmtETA(t *testing.T) {
cmd/metis/board_test.go:203:		if got := fmtETA(d); got != want {
cmd/metis/board_test.go:204:			t.Errorf("fmtETA(%v) = %q, want %q", d, got, want)
cmd/metis/board_test.go:308:	bw.paint([]string{"folds 2/8", "31.2 folds/min · ETA 12s"})
cmd/metis/board_test.go:317:	for _, stale := range []string{"folds 2/8", "folds/min", "ETA"} {
cmd/metis/board_test.go:361:	if !strings.Contains(s, "outer folds 2/2") || !strings.Contains(s, "outer fold 0 ✓") || !strings.Contains(s, "outer fold 1 ✓") {
cmd/metis/board_test.go:372:	if finalFrame := strings.LastIndex(s, "outer folds 2/2"); warnIdx > finalFrame {
cmd/metis/board_test.go:431:	for _, want := range []string{"outer folds 0/2", "outer fold 0 — queued", "no inner-CV run"} {
cmd/metis/board_test.go:451:		"outer folds 0/2", "outer fold 0 — queued", "configs ", "inner-CV runs ", "inner-CV runs/min", "ETA", "score ", "estimate", "mean ",
cmd/metis/board_test.go:610:	if !strings.Contains(term2.String(), "outer folds 0/1") {
cmd/metis/parallel_test.go:30:	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds = 6 per-fold rows
cmd/metis/parallel_test.go:304:	// 3 configs → nested (outer folds = sweeper.cv.k = 2) × 2 inner folds → deep nesting, ~many leaf calls.
cmd/metis/parallel_test.go:407:// failureBarrierExec holds the first four admitted inner folds at their train
cmd/metis/parallel_test.go:517:	for _, forbidden := range []string{"metis: progress", "folds/min", "ETA", "score ", "estimate", "mean "} {
cmd/metis/record_test.go:65:	// Upstream/D/Deps are metis#2-populated slots — #3 leaves them empty.
cmd/metis/activity_test.go:30:	var events []activityEvent
cmd/metis/activity_test.go:32:	got, err := activityExecutor{
cmd/metis/activity_test.go:35:		emit:  func(ev activityEvent) { events = append(events, ev) },
cmd/metis/activity_test.go:58:	var events []activityEvent
cmd/metis/activity_test.go:60:	got, err := activityExecutor{
cmd/metis/activity_test.go:63:		emit:  func(ev activityEvent) { events = append(events, ev) },
cmd/metis/nestedcv_e2e_test.go:56:	// The honest estimate is reported as a mean±SE over the outer folds.
cmd/metis/nestedcv_e2e_test.go:69:	if !strings.Contains(finalProg, "outer folds 2/2") || !strings.Contains(finalProg, "est 0.") {
cmd/metis/nestedcv_e2e_test.go:86:		t.Errorf("expected 2 outer-fold held-out scores (2 outer folds × 1 family), got %d:\n%s", n, s)
cmd/metis/nestedcv_e2e_test.go:132:	// 2 configs → nested; outer folds = sweeper.resample.cv.k = 2.
cmd/metis/nestedcv_e2e_test.go:141:// TestNestedCV_SampleRunsMOfKFolds (metis#42): `--sample m` runs exactly m of the k outer folds of
cmd/metis/nestedcv_e2e_test.go:144:// Asserts: m held-out scores, m outer ledger rows (folds 0..m-1 of the k-partition), and the
cmd/metis/nestedcv_e2e_test.go:165:	// 2 sampled outer folds × 1 family (a,b share the scalar `model` knob) = 2 held-out lines.
cmd/metis/nestedcv_e2e_test.go:182:		t.Errorf("outer rows should cover exactly sampled folds {0,1} of k=3, got %v", outerFolds)
cmd/metis/nestedcv_e2e_test.go:187:// k folds), --sample on a single-config shape (the flat path has no outer folds to sample), and
cmd/metis/nestedcv_e2e_test.go:215:		o := base(newShape(t, "[a]")) // 1 config → flat CV, no outer folds
cmd/metis/caching_soundness_test.go:49:	// format, so recordMiss folds it into the transitive-D closure and isHit re-hashes it.

exec
/bin/zsh -lc "sed -n '1,320p' cmd/metis/run.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/metis/internal/repo"
	"github.com/xianxu/metis/pkg/cas"
	"github.com/xianxu/metis/pkg/experiment"
)

// syncWriter serializes concurrent Write calls — the metis#31 parallel fan-out's
// progress output. Minimal: it prevents torn lines + the data race on a shared
// writer; it does NOT reorder or buffer per goroutine (clean per-k/n progress is
// metis#30's scope). Established in runExperiment when maxParallel>1.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

// cacheProjectRoot resolves the metis code root (the module dir above steps/) that D
// paths are relative to and `git hash-object` runs in — the same root metis.trace
// records in reads.json. Falls back to the experiment dir if step paths don't resolve.
func cacheProjectRoot(stepPath []string, fallback string) string {
	for _, p := range stepPath {
		if root, err := repo.Root(p); err == nil {
			return root
		}
	}
	return fallback
}

// ensureCacheGitignore writes .metis-cache/.gitignore so the local, wipeable cache
// (content-addressed output blobs) is never committed to the experiment's repo — the
// cache is safe to `rm -rf` and rebuild. Idempotent. (Sharing the git-trackable index
// across clones is a future enhancement; v1 ignores the whole cache dir.)
func ensureCacheGitignore(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	gi := filepath.Join(cacheDir, ".gitignore")
	if _, err := os.Stat(gi); err == nil {
		return nil
	}
	body := "# metis#2 step cache — a local, wipeable content-addressed cache (rm -rf is safe).\n" +
		"# Never commit its output blobs.\n*\n"
	return os.WriteFile(gi, []byte(body), 0o644)
}

// runOpts are the inputs to one `metis run`. now/git/out are injected so the e2e
// test gets a deterministic clock, a fake git probe, and can discard progress output.
type runOpts struct {
	expPath  string
	runID    string
	stepPath []string
	now      func() time.Time
	git      gitProbe
	cache    bool // enable the metis#2 validating-trace cache (<expDir>/.metis-cache)
	dryRun   bool // metis#18: list the swept configs without running them
	fast     bool // metis#32: nested run does ONE outer fold (a ~1/k-cost honest single-point) instead of k
	sample   int  // metis#42: nested run does m of the k outer folds (0 = all k). k stays the
	//               estimand knob (train fraction); m is the precision/cost knob — each fold is an
	//               unbiased sample of the k-fold estimand. --fast ≡ --sample 1 (kept as shorthand).
	inSweep bool // metis#14: this run is a sweep point — suppress per-point single-run
	//               capture (the sweep captures once per shape-run in captureSweepCode)
	out  io.Writer
	exec experiment.StepExecutor // test seam: an injected fake replaces the subprocess
	//                              execStep (nil → the production execStep). Composes with
	//                              cache: the caching decorator still wraps it.
	activity    activityEmitter // successful step/run facts; nil is a no-op
	runRole     runRole         // role for successful concrete-run activity; zero = ineligible/non-sweep
	readRoot    string          // metis#23: when set, the production execStep confines base-dataset reads to this root
	maxParallel int             // metis#31: >1 ⇒ ParExec batches + a leaf semaphore; sizes leafSem
	leafSem     chan struct{}   // metis#31: the shared global subprocess budget (nil = serial/cache-only)
	runControl  *runControl     // one per shape run: global abort + optional 2n admission slots
	runLabel    string          // config/fold/preamble context captured with the first error
	forkserver  bool            // metis#44: warm fork-server leaf executor (cmdRun default true;
	//                           zero-value false keeps direct runOpts callers/tests on legacy exec)
	forkPool *serverPool // metis#44: the per-root warm-server pool, created once per runExperiment
	//                      when forkserver is set; threaded through nested runOpts copies.
	tui bool // metis#38: stdout is a TTY and --no-tui wasn't passed — a SWEEP pins the live board
	//          (a plain experiment ignores it; non-TTY/piped runs stay on the #30 plain lines)
	board           *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
	boardTick       <-chan time.Time  // test seam: nil uses the production 500ms ticker
	beforeBoardTick func()            // test seam: after tick selection, before health observation
	afterBoardTick  func()            // test seam: after the health observation returns
	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
	leafPins        []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
	//                             runExperiment (nil = not yet computed; non-nil rides nested runOpts
	//                             copies like forkPool — an all-suppressed result is empty, not nil)
}

// runExperiment reads the experiment at o.expPath and dispatches: a `type:
// experiment-shape` is the metis#18 nested-Sampler SWEEP (the sweeper grids over configs,
// the inner resample folds each — runShapeSweep); a plain `type: experiment` is the
// one-point path (runResolvedExperiment). The `.md` is immutable input (#13) — never
// written back; all side effects live in the shell below, the ordering/validation logic
// stays in pkg/experiment. Returns the assembled Run (empty for a sweep — the manifest +
// per-fold records + ledger are its output) and the run error.
func runExperiment(o runOpts) (experiment.Run, error) {
	now := o.now
	if now == nil {
		now = time.Now
	}
	out := o.out
	if out == nil {
		out = io.Discard
	}

	// metis#38: PARSE FIRST (parsing writes nothing) — the board decision needs the file
	// type, and writer identity is TEMPORAL: everything constructed below (fork-server
	// pool, execs, sink, capture warnings) captures whatever `out` is at ITS construction,
	// so the one compositor must exist before any of them or its writes bypass the board.
	raw, err := os.ReadFile(o.expPath)
	if err != nil {
		return experiment.Run{}, err
	}
	// Peek the type with the tolerant experiment parser (it ignores the shape-only
	// data/pipeline/ship/sweeper keys); a shape then re-parses through the STRICT
	// ParseShape (unknown-key-loud) for the sweep path.
	exp, err := experiment.Parse(string(raw))
	if err != nil {
		return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
	}

	// metis#31: establish the parallel invariant in ONE home — maxParallel>1 ⇒ a
	// non-nil SHARED leaf semaphore AND a serialized writer. Doing it here (not in
	// cmdRun) means no direct-runOpts caller (the tests) can enable maxParallel>1 yet
	// forget the sem or race the fan-out's progress writes on a bare buffer.
	if o.maxParallel > 1 && o.leafSem == nil {
		o.leafSem = make(chan struct{}, o.maxParallel)
	}
	if sem := o.leafSem; sem != nil && o.leafGauge == nil {
		o.leafGauge = func() (int, int) { return len(sem), cap(sem) } // metis#38: occupancy IS the semaphore
	}
	// Exactly ONE writer wrap (metis#38): board mode → the pin-bottom compositor (it
	// serializes internally — no syncWriter stacking); else parallel → syncWriter.
	if o.tui && exp.Type == "experiment-shape" && !o.dryRun {
		o.board = newBoardWriter(out, now)
		out = o.board
		o.out = out
		defer o.board.close() // idempotent — an error return must not leak a hidden cursor
	} else if o.maxParallel > 1 {
		out = &syncWriter{w: out}
		o.out = out
	}
	// metis#48: default leaf BLAS pins — computed ONCE per top-level run from the ambient
	// env (an exported operator value wins by exclusion in blasPins), announced loudly,
	// and injected at both spawn seams (legacy execStep child env; fork-server process
	// env). Fake-exec runs spawn nothing (no pins, no note); dry-run lists configs (same).
	// Placed AFTER the writer wrap (the note must route through the board) and BEFORE the
	// pool creation (the server spawn env needs the pins).
	if o.exec == nil && !o.dryRun && o.leafPins == nil {
		o.leafPins = blasPins(os.Environ())
		if len(o.leafPins) > 0 {
			fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
				strings.Join(o.leafPins, " "))
		}
	}
	// metis#44: one warm fork-server pool per top-level run, shut down (EOF-drain) when the
	// run ends. Only the production executor uses it (an injected test exec bypasses execStep).
	// Constructed AFTER the writer wrap — its fallback notices must route through the board.
	if o.forkserver && o.exec == nil && o.forkPool == nil {
		o.forkPool = newServerPool(out, o.leafPins)
		defer o.forkPool.shutdown()
	}
	if exp.Type == "experiment-shape" {
		sh, err := experiment.ParseShape(string(raw))
		if err != nil {
			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
		}
		if err := experiment.ValidateShape(sh); err != nil {
			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
		}
		if o.runControl == nil {
			o.runControl = newRunControl(o.maxParallel)
		}
		return experiment.Run{}, runShapeSweep(o, sh, now, out)
	}
	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
}

// singleRunID names a single run's dir. metis#27: content-address it by the run's
// point-address (symmetric with a sweep point's dir), so the dir name IS the run identity.
// An explicit --run overrides; the timestamp form survives only as the no-git fallback
// (when the shape blob-hash — hence the point-address — can't be computed).
func singleRunID(o runOpts, exp experiment.Experiment, now func() time.Time) string {
	if o.runID != "" {
		return o.runID
	}
	sbh, err := shapeBlobHash(o.expPath)
	if err == nil {
		if addr, err := pointAddressOf(exp, sbh); err == nil {
			return addr
		}
	}
	return "run-" + now().UTC().Format("20060102T150405Z")
}

// runResolvedExperiment runs one already-resolved experiment (a single point) under
// runID, through the cached runner, and writes its run.json + provenance record (the
// experiment `.md` is immutable input — not written back, #13). The shared per-point runner
// both the 1-point path and the sweep loop (metis#7) call — so the run/cache/record wiring
// lives in ONE place (ARCH-DRY).
func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
	if o.runControl == nil {
		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
	}
	o.activity = runControlActivityEmitter(o.runControl, o.activity)
	return o.runControl.run(o.runLabel, func() (experiment.Run, error) {
		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
	})
}

func runResolvedExperimentAdmitted(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
	baseDir := filepath.Dir(o.expPath)
	// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
	// the child's env, and the child's cwd IS the step dir — a relative path would
	// resolve $METIS_STEP_DIR/with.json under itself. Absolute paths are correct
	// from any cwd, so `metis run pipelines/foo.md` (a relative arg) works.
	runDir, err := filepath.Abs(filepath.Join(baseDir, "runs", runID))
	if err != nil {
		return experiment.Run{}, err
	}
	expDir, err := filepath.Abs(baseDir)
	if err != nil {
		return experiment.Run{}, err
	}

	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool, pins: o.leafPins}
	if o.exec != nil {
		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
	}
	if o.cache {
		cacheDir := filepath.Join(expDir, ".metis-cache")
		if err := ensureCacheGitignore(cacheDir); err != nil {
			return experiment.Run{}, err
		}
		store := cas.NewFSStore(filepath.Join(cacheDir, "cas"), 0, cas.Clock(now))
		exec = newCachingExecutor(exec, store, cacheDir, exp.Seed, out)
	}
	exec = activityExecutor{inner: exec, now: now, emit: o.activity}
	runner := experiment.Runner{Exec: exec, Now: now}
	fmt.Fprintf(out, "metis: run %s of experiment %q\n", runID, exp.ID)
	run, steps, runErr := runner.Run(exp, runID, runDir)

	// Execution-time enforcement: Runner.Run validates the experiment BEFORE any
	// step executes, so a semantically-invalid experiment (dangling needs, bad
	// uses, a cycle) is rejected here — closing the SHAPE-only gap M1 left. Such a
	// rejection never started a run (run.Started is empty), so surface the error
	// without writing a bogus record.
	if run.Started == "" {
		return run, runErr
	}

	// Write the ledger even on a mid-run step failure (status=failed) so every
	// attempt that began is recorded — the record of truth is runs/<id>/run.json.
	if err := writeRunJSON(runDir, run); err != nil {
		return run, err
	}
	// Assemble + persist the provenance record (metis#3): repo provenance, per-step
	// output hashes, and the minted point-address. A config that can't be
	// canonicalized (e.g. a non-finite value) surfaces here as a run error.
	// The shape's blob-hash content-addresses the intent (metis#27); computed the SAME way
	// singleRunID/pointAddressOf did, so the record's point_address matches the run dir.
	// A no-git spec yields "" (a degraded, non-content-addressed run — warned via capture status).
	sbh, _ := shapeBlobHash(o.expPath)
	rec, err := assembleRecord(o.git, out, expDir, runDir, exp, run, steps, sbh)
	if err != nil {
		return run, err
	}
	if err := writeRecordJSON(runDir, rec); err != nil {
		return run, err
	}
	// Capture this run's code closure + run-spec to a git side-ref (metis#14), backfilling
	// the record with the durable SHA + capture status — so a dirty single run is
	// reproducible (git checkout the SHA). The sweep loop sets inSweep to capture ONCE
	// per shape-run instead (captureSweepCode), avoiding redundant per-point capture.
	// Best-effort (like the sweep path): a backfill hiccup warns, never aborts a finished run.
	if !o.inSweep {
		if err := captureSingleRun(o, runID); err != nil {
			fmt.Fprintf(out, "metis: warning: code-capture backfill failed for run %s: %v\n", runID, err)
		}
	}
	// The experiment .md is IMMUTABLE input (#13): a run writes its output to
	// runs/<id>/{run,record}.json (+ the .ledger.csv sidecar for sweeps), NEVER to the
	// config file — so a committed config is a stable content-hash. The human "recent
	// runs / top-N" view is on-demand via `metis ledger show` over the sidecar.
	if runErr != nil {
		return run, runErr
	}
	o.activity.emit(activityEvent{Kind: activityRunSuccess, At: now(), RunID: runID, Role: o.runRole})
	fmt.Fprintf(out, "metis: %s %s\n", run.ID, run.Status)
	return run, nil
}

func writeRunJSON(runDir string, run experiment.Run) error {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "run.json"), append(b, '\n'), 0o644)
}

codex
The current tree does include the #49 typed-activity code that was not visible in the pasted excerpt, so I’m switching from “missing implementation” to targeted correctness checks. I’m going to inspect the activity wiring, board rendering, and persistence/error seams for contract drift.
exec
/bin/zsh -lc "sed -n '1,180p' cmd/metis/activity.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

import (
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

type activityKind string

const (
	activityStepSuccess activityKind = "step-success"
	activityRunSuccess  activityKind = "run-success"
)

type runRole string

const (
	runRoleNone           runRole = ""
	runRoleNestedInnerCV  runRole = "nested-inner-cv"
	runRoleFlatCV         runRole = "flat-cv"
	runRoleNestedPreamble runRole = "nested-preamble"
	runRoleOuterScore     runRole = "outer-score"
)

type activityEvent struct {
	Kind   activityKind
	At     time.Time
	StepID string
	RunID  string
	Role   runRole
}

type activityEmitter func(activityEvent)

func (e activityEmitter) emit(ev activityEvent) {
	if e != nil {
		e(ev)
	}
}

func teeActivityEmitter(a, b activityEmitter) activityEmitter {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(ev activityEvent) {
		a.emit(ev)
		b.emit(ev)
	}
}

func runControlActivityEmitter(control *runControl, emit activityEmitter) activityEmitter {
	if control == nil {
		return emit
	}
	return func(ev activityEvent) {
		control.whileHealthy(func() { emit.emit(ev) })
	}
}

type activityExecutor struct {
	inner experiment.StepExecutor
	now   func() time.Time
	emit  activityEmitter
}

func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	res, err := e.inner.Execute(step, runDir)
	if err != nil {
		return res, err
	}
	now := e.now
	if now == nil {
		now = time.Now
	}
	e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
	return res, nil
}

exec
/bin/zsh -lc "sed -n '1,170p' cmd/metis/board.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

// metis#38: the live progress board — the PURE frame renderer (this half) and the
// pin-bottom ANSI compositor (boardWriter, below). Presentation only, over the #30
// sink's boardState snapshot: no pkg/sampler change, no TUI library (the board is
// output-only — a hand-rolled repaint of N lines; see the plan's no-lib rationale).
// The paint/content split is deliberate: renderBoard returns plain lines (byte-
// testable, zero escape codes); ANSI lives ONLY in boardWriter.

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// boardEnv is the render-time environment: terminal width, the current instant
// (for rate/ETA), and the leaf-semaphore gauge (capacity 0 = no gauge → segment
// omitted, e.g. a serial run).
type boardEnv struct {
	width          int
	now            time.Time
	busy, capacity int
}

// maxFoldRows caps the per-fold section; beyond it the remainder collapses to
// an "… +N more" line (a 20-fold sweep must not paint a 22-line board).
const maxFoldRows = 12

// renderBoard renders the frame: the #30 aggregate line, one row per outer fold
// (✓ done → held-out score · ▸ in-flight → per-pass counters + incumbent · queued),
// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
// break the compositor's erase-count bookkeeping).
func renderBoard(bs boardState, env boardEnv) []string {
	var lines []string
	// Row 1: the aggregate — the same core the plain line prints (one source, no
	// prefix stripping).
	lines = append(lines, progressCore(bs.st))

	// Per-fold rows (nested only; flat runs have no rows).
	shown := len(bs.rows)
	if shown > maxFoldRows {
		shown = maxFoldRows
	}
	// Per-row denominators derive from the seeded aggregate totals (per-pass share).
	perConfigs, perFolds := 0, 0
	if n := len(bs.rows); n > 0 {
		perConfigs = bs.st.configTotal / n
		perFolds = bs.st.foldTotal / n
	}
	for i := 0; i < shown; i++ {
		r := bs.rows[i]
		switch {
		case r.done:
			lines = append(lines, fmt.Sprintf("  outer fold %d ✓ held-out %.4f", i, r.heldOut))
		case r.configK == 0 && r.foldK == 0:
			lines = append(lines, fmt.Sprintf("  outer fold %d — queued", i))
		default:
			b := ""
			if r.hasBest {
				b = fmt.Sprintf(" · best %.4f", r.best)
			}
			lines = append(lines, fmt.Sprintf("  outer fold %d ▸ configs scored %d/%d · inner-CV runs %d/%d%s",
				i, r.configK, perConfigs, r.foldK, perFolds, b))
		}
	}
	if hidden := len(bs.rows) - shown; hidden > 0 {
		lines = append(lines, fmt.Sprintf("  … +%d more", hidden))
	}

	// Leaves / throughput / ETA.
	var segs []string
	runNoun := "CV runs"
	runSingular := "CV run"
	if bs.st.nested {
		runNoun = "inner-CV runs"
		runSingular = "inner-CV run"
	}
	if env.capacity > 0 {
		segs = append(segs, fmt.Sprintf("~slots %d/%d", env.busy, env.capacity))
	}
	if bs.st.foldK == 0 {
		start := []string{"starting"}
		start = append(start, segs...)
		if bs.st.stepK > 0 {
			start = append(start, fmt.Sprintf("%d steps completed", bs.st.stepK))
			if !bs.st.lastStepAt.IsZero() {
				start = append(start, "last step "+fmtAge(env.now.Sub(bs.st.lastStepAt))+" ago")
			}
		}
		start = append(start, fmt.Sprintf("no %s complete", runSingular))
		lines = append(lines, strings.Join(start, " · "))
		for i, l := range lines {
			lines[i] = clampLine(l, env.width)
		}
		return lines
	}
	if !bs.st.lastRunAt.IsZero() {
		segs = append(segs, fmt.Sprintf("last %s %s ago", runSingular, fmtAge(env.now.Sub(bs.st.lastRunAt))))
	}
	if perMin, ok := bs.rate.rate(env.now); ok {
		segs = append(segs, fmt.Sprintf("%.1f %s/min", perMin, runNoun))
	} else {
		segs = append(segs, fmt.Sprintf("— %s/min", runNoun))
	}
	if remaining := bs.st.foldTotal - bs.st.foldK; remaining > 0 {
		if eta, ok := bs.rate.eta(env.now, remaining); ok {
			segs = append(segs, "~ETA "+fmtETA(eta))
		}
	}
	lines = append(lines, strings.Join(segs, " · "))

	for i, l := range lines {
		lines[i] = clampLine(l, env.width)
	}
	return lines
}

func fmtAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return fmtETA(d)
}

// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
func fmtETA(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// (Height analog of the width limitation: a terminal SHORTER than the board clamps
// cursor-up at the screen top and desyncs the erase count — the board caps at ~15
// lines; terminals that small are out of scope, same accepted trade as resize.)

// clampLine truncates to width runes with a trailing … (a wrapped physical line
// would desync the compositor's cursor-up erase count — width is load-bearing).
func clampLine(s string, width int) string {
	if width <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width-1]) + "…"
}

// ── boardWriter: the pin-bottom ANSI compositor ──────────────────────────────

// boardWriter owns the terminal: the board is pinned to the bottom while every
// other write (step logs, warnings) scrolls ABOVE it. Paint-only — it stores the
// last rendered frame and NEVER calls back into the sink (the one global lock
// order is sink.mu → bw.mu; a callback here would invert it). All output must
// route through this writer once it exists — a bypassing write corrupts the board
// (see the plan's writer-plumbing note: writer identity is temporal).
//
// metis#46: DOUBLE-BUFFERED with a bounded flush rate. The original design ran a
// full erase→write→repaint cycle per passthrough write; a warm-cache sweep emits
// hundreds of lines/second and real terminals — especially mux layers (the
// operator's ghostty-in-cmux) — paint asynchronously mid-sequence and tear under
// that flood. Now passthrough COALESCES into `pending` and the terminal sees one
// atomic erase→dump→repaint per flushBudget (~4Hz) — quiet writes (a cold run's

exec
/bin/zsh -lc "sed -n '1,220p' cmd/metis/activity_test.go && sed -n '140,330p' cmd/metis/run_test.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

type activityFakeExec struct {
	result experiment.StepResult
	err    error
	calls  int
}

func (f *activityFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	f.calls++
	return f.result, f.err
}

func TestActivityExecutorEmitsOneStepSuccessAfterSuccessfulInnerExecution(t *testing.T) {
	at := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	wantResult := experiment.StepResult{
		Metrics:   map[string]float64{"score": 0.91},
		Artifacts: []string{"train/model.bin"},
	}
	inner := &activityFakeExec{result: wantResult}
	var events []activityEvent

	got, err := activityExecutor{
		inner: inner,
		now:   func() time.Time { return at },
		emit:  func(ev activityEvent) { events = append(events, ev) },
	}.Execute(experiment.Step{ID: "train"}, "/tmp/run")

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("Execute result = %+v; want %+v", got, wantResult)
	}
	if inner.calls != 1 {
		t.Fatalf("inner calls = %d; want 1", inner.calls)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d; want 1 (%+v)", len(events), events)
	}
	if events[0].Kind != activityStepSuccess || events[0].StepID != "train" || !events[0].At.Equal(at) {
		t.Fatalf("event = %+v; want one step-success event for train at %s", events[0], at.Format(time.RFC3339))
	}
}

func TestActivityExecutorEmitsNothingOnInnerErrorAndPreservesFailure(t *testing.T) {
	wantErr := errors.New("boom")
	inner := &activityFakeExec{err: wantErr}
	var events []activityEvent

	got, err := activityExecutor{
		inner: inner,
		now:   func() time.Time { return time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC) },
		emit:  func(ev activityEvent) { events = append(events, ev) },
	}.Execute(experiment.Step{ID: "train"}, "/tmp/run")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute error = %v; want %v", err, wantErr)
	}
	if !reflect.DeepEqual(got, experiment.StepResult{}) {
		t.Fatalf("Execute result = %+v; want zero result from inner failure", got)
	}
	if inner.calls != 1 {
		t.Fatalf("inner calls = %d; want 1", inner.calls)
	}
	if len(events) != 0 {
		t.Fatalf("events = %+v; want none on error", events)
	}
}
	}

	run, err := runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-001",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
	})
	if err == nil {
		t.Fatal("runExperiment: want an error from the failing step, got nil")
	}
	if run.Status != "failed" {
		t.Errorf("returned run status = %q; want failed", run.Status)
	}

	// runs/run-001/run.json written with status=failed.
	rb, err := os.ReadFile(filepath.Join(dir, "runs", "run-001", "run.json"))
	if err != nil {
		t.Fatalf("read run.json (failed run should still be recorded): %v", err)
	}
	var got experiment.Run
	if err := json.Unmarshal(rb, &got); err != nil {
		t.Fatalf("parse run.json: %v", err)
	}
	if got.ID != "run-001" || got.Experiment != "run-fail" || got.Status != "failed" {
		t.Errorf("run.json wrong: %+v", got)
	}

	// #13: a failed run is recorded in run.json/record.json (asserted above) — the config .md
	// stays byte-for-byte immutable input (the fixture already carries a `## Runs` heading, so
	// this asserts equality, not absence).
	updated, err := os.ReadFile(expPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != string(b) {
		t.Errorf("failed run mutated the config .md (must be immutable input):\n%s", updated)
	}
}

func TestRunExperimentActivityRunSuccessPublishesAfterRequiredArtifacts(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-echo.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	var runEvents []activityEvent
	var publishErrs []error
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-activity",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind != activityRunSuccess {
				return
			}
			runEvents = append(runEvents, ev)
			for _, name := range []string{"run.json", "record.json"} {
				if _, statErr := os.Stat(filepath.Join(dir, "runs", ev.RunID, name)); statErr != nil {
					publishErrs = append(publishErrs, statErr)
				}
			}
		},
	})
	if err != nil {
		t.Fatalf("runExperiment: %v", err)
	}
	if len(publishErrs) > 0 {
		t.Fatalf("run-success event arrived before required artifacts existed: %v", publishErrs)
	}
	if len(runEvents) != 1 {
		t.Fatalf("run-success events = %d; want 1 (%+v)", len(runEvents), runEvents)
	}
	if runEvents[0].RunID != "run-activity" {
		t.Fatalf("run event = %+v; want runID run-activity", runEvents[0])
	}
	if runEvents[0].Role != runRoleNone {
		t.Fatalf("plain run event role = %q; want no role/ineligible", runEvents[0].Role)
	}
}

func TestRunExperimentActivityRunSuccessNotPublishedForFailedExecution(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-fail.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-fail.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	var runEvents []activityEvent
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-failed-activity",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind == activityRunSuccess {
				runEvents = append(runEvents, ev)
			}
		},
	})
	if err == nil {
		t.Fatal("runExperiment: want failing step error")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "run.json")); statErr != nil {
		t.Fatalf("failed run should still write run.json: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "record.json")); statErr != nil {
		t.Fatalf("failed run should still write record.json: %v", statErr)
	}
	if len(runEvents) != 0 {
		t.Fatalf("failed execution emitted run-success events: %+v", runEvents)
	}
}

func TestRunExperimentActivityRunSuccessNotPublishedWhenRecordPersistenceFails(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-echo.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "runs", "run-record-fails", "record.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	var runEvents []activityEvent
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-record-fails",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind == activityRunSuccess {
				runEvents = append(runEvents, ev)
			}
		},
	})
	if err == nil {
		t.Fatal("runExperiment: want record persistence error")
	}
	if len(runEvents) != 0 {
		t.Fatalf("record persistence failure emitted run-success events: %+v", runEvents)
	}
}

// TestRunExperiment_RejectsInvalidAtRunTime is the execution-time enforcement
// test: a semantically-invalid experiment (a cycle — shape-valid, so CUE accepts
// it) is rejected by `metis run` BEFORE any step runs, closing the SHAPE-only gap
// M1 deferred. No ledger and no `## Runs` line are written for a rejected run.
func TestRunExperiment_RejectsInvalidAtRunTime(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "invalid-cycle.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "invalid-cycle.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = runExperiment(runOpts{
		expPath:  expPath,

exec
/bin/zsh -lc "sed -n '440,700p' cmd/metis/sweep.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
		return err
	}
	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the estimate report

	// metis#32: the nested run now RECORDS (unlike metis#23's estimation-only path) — persist the
	// inner + per-family outer rows accumulated in ss.man.Points so `metis select` can reduce them
	// (family from the outer rows, config from the inner rows). Sort to a stable content key first
	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
	sortPointRuns(ss.man.Points)
	if err := writeManifest(ss.o.expPath, ss.man); err != nil {
		return ss.fail("write nested sweep manifest", err)
	}
	cohort, cerr := captureSweepCode(ss.o, ss.man)
	if cerr != nil {
		ss.whileHealthy(func() {
			fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
		})
	}
	if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
		return ss.fail("write nested sweep ledger", err)
	}
	ss.whileHealthy(func() {
		ss.reportEstimate(est, runFolds)
		printRunSummary(ss.out, ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
	})
	return ss.firstError()
}

// materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
// and returns the k analysis_i refs (experiment-relative, so a sealed sweep reading one routes
// through exp_path → confined). Unconfined (outer-split reads the full dataset to split it).
func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]string, error) {
	baseOut, baseID := baseDatasetRef(ss.sh)
	var needs []string
	if baseID != "" {
		needs = []string{baseID}
	}
	osStep := experiment.Step{ID: outerSplitStepID, Uses: "metis/outer-split", Needs: needs,
		With: map[string]any{"dataset": baseOut, "k": outerK, "stratify": stratify}}
	steps := append(append([]experiment.Step{}, ss.sh.Data...), osStep)
	exp := experiment.Experiment{Header: ss.sh.Header, Steps: steps}
	exp.Type = "experiment"
	preID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
		return nil, ss.fail("nested-CV preamble address", err)
	}
	preOpts := ss.o
	preOpts.inSweep = true // one preamble run; skip the per-run capture noise
	preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
	preOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
	preOpts.runRole = runRoleNestedPreamble
	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
		return nil, err
	}
	refs := make([]string, outerK)
	for i := 0; i < outerK; i++ {
		refs[i] = filepath.ToSlash(filepath.Join("runs", preID, outerSplitStepID, fmt.Sprintf("analysis_%d", i)))
	}
	return refs, nil
}

// runOuterFold runs one outer fold: (a) the SEALED sweeper on analysis_i → a winner (confined via
// the exp_path chokepoint — readRoot = analysis_i abs), then (b) the refit-and-score of that winner
// on the held outer-assessment (a full-data fold run at outer-k, held=i; unconfined). Returns the
// honest outer-fold score.
func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k int, stratify bool, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
	if err != nil {
		return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
	}
	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
		stratify: stratify, partRef: ss.partRef,
		runRole: runRoleNestedInnerCV,
		hooks:   ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
	sres := ss.runSweeper(ctx, configPts, pass)
	if err := pass.firstError(); err != nil {
		return 0, err
	}
	// Guard (metis#19/#23 I1): the parsimony select rule needs a measured complexity for every
	// swept family — same guard the flat path runs before trusting its winner. Without it, a
	// parsimony-select + non-reporting-model shape would SILENTLY mis-select in each outer fold.
	if err := sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs)); err != nil {
		return 0, ss.fail(fmt.Sprintf("outer fold %d complexity guard", i), err)
	}

	// metis#32: record the sealed sweep's INNER rows (Level=inner, tagged with this outer fold).
	of := i
	rows := make([]pointRun, 0, len(pass.points)+len(sres.PerFamily))
	if !ss.whileHealthy(func() {
		for _, pr := range pass.points {
			pr.Level = "inner"
			pr.OuterFold = &of
			rows = append(rows, pr)
		}
	}) {
		return 0, errRunAborted
	}

	// (b) score EACH family's inner-winner on the held outer-assessment — post-selection, so
	// unconfined and leakage-free (each winner was selected SEALED within analysis_i; scoring on
	// the held-out fold never influenced that selection). One OUTER row per family → the honest
	// per-family measure `metis select` reduces (metis#32). The metis#23 estimate the CVDriver
	// aggregates stays the SHIP-family's outer score (the argmax-mean procedure's honest number).
	// The cv-split uses the OUTER k + stratify so cv_folds's determinism reproduces the exact
	// partition outer-split materialized (else the held fold ≠ analysis_i's assessment rows).
	shipFamily := sres.Ship.Family
	var shipScore float64
	for _, fam := range sortedFamilies(sres.PerFamily) {
		w := sres.PerFamily[fam]
		score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart, fam)
		if ferr != nil {
			return 0, ferr
		}
		if !ss.whileHealthy(func() {
			rows = append(rows, pointRun{
				RunID:      scoreID,
				FreeParams: freeParamMap(w.Point),
				Fold:       of, // the outer fold this held-out score is on
				Level:      "outer",
				OuterFold:  &of,
				Status:     status,
				// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
			})
			if fam == shipFamily {
				shipScore = score
			}
			fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
				i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
		}) {
			return 0, errRunAborted
		}
	}
	if !ss.addManPoints(rows) {
		return 0, errRunAborted
	}
	return shipScore, nil
}

// scoreOnOuterFold refit-and-scores one config's winner on the held outer-assessment fold i (a
// full-data fold run at outer-k; post-selection, so unconfined). Returns the held-out fold_score,
// the run id (→ its record.json carries the namespaced metric the ledger reads), and its status.
func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef, fam string) (float64, string, string, error) {
	scoreExp := ss.buildFoldExperiment(point, sampler.FoldPoint{Idx: i}, nil, k, stratify, outerPart)
	scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
	if err != nil {
		return 0, "", "", ss.fail(fmt.Sprintf("outer fold %d family %s score address", i, fam), err)
	}
	scoreOpts := ss.o
	scoreOpts.inSweep = true
	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
	scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
	scoreOpts.runRole = runRoleOuterScore
	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
	if err != nil {
		return 0, "", "", err
	}
	return run.Metrics[foldMetric], scoreID, run.Status, nil
}

// sortedFamilies returns the family keys of a per-family winner map in deterministic order
// (the recording + the returned ship-score must not depend on Go's random map iteration).
func sortedFamilies(perFamily map[string]sampler.Winner) []string {
	fams := make([]string, 0, len(perFamily))
	for fam := range perFamily {
		fams = append(fams, fam)
	}
	sort.Strings(fams)
	return fams
}

// reportEstimate prints the honest procedure estimate — mean±SE over the outer folds — and the
// standing reminder that driver:cv produces NO shippable winner (estimation ≠ selection).
func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
	fmt.Fprintf(ss.out, "metis: nested-CV estimate — mean %.4f (SE %.4f) over %d outer fold(s) — the HONEST procedure estimate (argmax-mean family)\n",
		est.Mean, est.SE, outerK)
	fmt.Fprintf(ss.out, "  (per-family honest estimates recorded to the ledger; choose + ship via `metis select --best --promote`)\n")
}

// runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
// synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
// the shared cached runner, record the manifest row, and return the fold_score the inner
// resample Sampler folds. A fatal outcome publishes through the experiment-wide
// runControl and returns 0; every sampler callback/sink rejects placeholders after
// publication, and the top level returns the stored concrete cause.
func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.FoldOutcome {
	ss := p.ss
	if p.firstError() != nil {
		return sampler.FoldOutcome{}
	}
	// Detect-and-abort: a mid-sweep HEAD-sha change breaks the shape-run's one-code
	// identity (per-fold records stay correct). Compares the HEAD sha only, not the dirty
	// flag — the sweep's own writes (runs/, manifest) dirty the tree (see codeID freeze).
	// metis#31: only a DEFINITE sha change aborts — `s != ""`. probeRepo swallows any
	// probe error to "", and under parallel fan-out concurrent `git status` contends on
	// .git/index.lock so a transient probe failure is expected; treating "" as a change
	// would false-abort the whole honest run.
	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx),
			fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
		return sampler.FoldOutcome{}
	}

	exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.stratify, p.partRef)
	runID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx), err)
		return sampler.FoldOutcome{}
	}
	pointOpts := ss.o
	pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
	pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
	pointOpts.runRole = p.runRole
	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
	if runErr != nil {
		// runControl already published a concrete admitted-run failure. A queued or
		// late sibling returns errRunAborted; neither path may republish the sentinel.
		return sampler.FoldOutcome{}
	}
	if !p.addPoint(pointRun{
		RunID:      runID,
		FreeParams: freeParamMap(c),
		Fold:       f.Idx,
		Status:     run.Status,
		Metrics:    run.Metrics,
	}) {
		return sampler.FoldOutcome{}
	}
	// metis#19 M2: read the train step's realized-complexity metric. Present → the parsimony
	// rules consume it; absent (HasComplexity false) → the guard rejects a parsimony rule.
	cx, hasCx := run.Metrics[foldComplexityMetric]
	return sampler.FoldOutcome{Score: run.Metrics[foldMetric], Complexity: cx, HasComplexity: hasCx}
}

// buildFoldExperiment reconstructs the runnable per-fold experiment for one (config, fold):
// the data steps (as declared — cache-shared, config+fold-invariant) + the engine-synthesized
// cv-split partition step + the pipeline steps with the config's resolved `with` overlaid AND
// the fold-context injected. The fold-context ({_fold:{partition,idx}, folds:<cv-split>}) enters
// each pipeline step's `with` so its Kpre is fold-distinct (the B2 collision guard) and the step
// can read the fold assignment. Ship is NOT included (winner-only, M1a-5).
// baseRef nil = the flat driver:single path (data phase + cv-split over the declared base).
// baseRef non-nil = a sealed nested outer fold (metis#23): the data phase is DROPPED (analysis_i
// is already the adapted base) and cv-split + every pipeline step that read the declared base are
// repointed to baseRef (analysis_i), so their reads route through exp_path → confined to the
// outer-analysis root and the sweeper's inner-CV structurally cannot see outer-assessment.
func (ss *shapeSweep) buildFoldExperiment(c shape.Point, f sampler.FoldPoint, baseRef any, splitK int, stratify bool, partRef sampler.PartitionRef) experiment.Experiment {
	sh := ss.sh
	steps := make([]experiment.Step, 0, len(sh.Data)+1+len(sh.Pipeline))
	baseOut, baseID := baseDatasetRef(sh)
	origOut := baseOut // the declared base, captured before the sealed branch reassigns baseOut
	var partNeeds []string
	if baseRef == nil {
		steps = append(steps, sh.Data...)
		if baseID != "" {
			partNeeds = []string{baseID}
		}
	} else {

codex
The core #49 code is present in the working tree, but it is not in the committed diff range shown by `git diff --name-only`. That means I need to distinguish tracked HEAD content from uncommitted workspace content, since the boundary window is Base..HEAD.
exec
/bin/zsh -lc 'git status --short' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
 M atlas/index.md
 M cmd/metis/board.go
 M cmd/metis/board_test.go
 M cmd/metis/caching_test.go
 M cmd/metis/nestedcv_e2e_test.go
 M cmd/metis/progress.go
 M cmd/metis/progress_test.go
 M cmd/metis/run.go
 M cmd/metis/run_test.go
 M cmd/metis/runcontrol_test.go
 M cmd/metis/shapesweep_test.go
 M cmd/metis/sweep.go
 M workshop/issues/000049-board-readability-labels-coldphase-eta.md
?? cmd/metis/activity.go
?? cmd/metis/activity_test.go

exec
/bin/zsh -lc 'git ls-files cmd/metis/activity.go cmd/metis/activity_test.go' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead

exec
/bin/zsh -lc 'git log --oneline --decorate -12' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
exec
/bin/zsh -lc 'git diff --name-only HEAD -- cmd/metis/activity.go cmd/metis/activity_test.go cmd/metis/progress.go cmd/metis/board.go cmd/metis/run.go cmd/metis/sweep.go workshop/issues/000049-board-readability-labels-coldphase-eta.md' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
73cf699 (HEAD -> 000049-board-readability-labels-coldphase-eta, main) #49: fix estimate primitive slugs
b2fff03 #49: plan board readability telemetry
b37415c (origin/main, origin/HEAD) archive completed issues to history
8a0e410 Merge pull request #34 from xianxu/000043-leaf-sched-depth-first
745e9f5 (origin/000043-leaf-sched-depth-first) #43: close bounded whole-run admission
53f0488 #43: enter verified close boundary
56e560c #43: pin cold smoke peer dependency
1ec6976 #43: record issue validation evidence
ae372a2 #43: document bounded sweep admission
667c3b0 #43: record scheduling acceptance checkpoint
7b717c3 #43: harden scheduling proof hooks
b2e218a #43: pin bounded depth-first sweep scheduling

 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
cmd/metis/board.go
cmd/metis/progress.go
cmd/metis/run.go
cmd/metis/sweep.go
workshop/issues/000049-board-readability-labels-coldphase-eta.md

codex
There is a boundary-level problem: the #49 implementation files are uncommitted or modified in the working tree, while the requested review window is `Base..HEAD`. I’m checking HEAD’s actual contents now to avoid reviewing changes that would not cross the boundary.
exec
/bin/zsh -lc 'git show HEAD:cmd/metis/progress.go | rg -n "inner-CV runs|CV runs|folds|min|movingRate|occupancyWindow|activityEvent|configs scored|outer folds"' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
3:// metis#30: the sweep progress sink — folds pkg/sampler's per-completion
26:// which starves the display of denominators. cmd/metis constructs the samplers,
32:	configs    int // aggregate across outer folds (outer × per-pass configs)
34:	folds      int // aggregate leaf count (outer × configs × inner k)
54:// `outer 1/3 · configs 84/216 · folds 421/1080 · est 0.8283 ± 0.0140`
56:// config): `folds 3/5 · score 0.8400` (the running fold mean — nothing to be
80:		parts = append(parts, "folds "+frac(st.foldK, st.foldTotal, st.foldKind))
91:		parts = append(parts, "folds "+frac(st.foldK, st.foldTotal, st.foldKind))
124:// composes the aggregate denominators: configs = outer × per-pass configs,
125:// folds = outer × configs × inner k (each sealed pass sweeps the full grid).
126:// Flat (1 config): folds = the single pass's inner k.
130:	foldsSmp := sampler.FixedKFolds{K: k}
131:	nFolds, kindFolds := foldsSmp.SizeHint(foldsSmp.Init(ctx))
133:		return progressTotals{folds: nFolds, foldKind: kindFolds}
141:		folds: nOuter * nConfigs * nFolds, foldKind: kindFolds,
145:// movingRate is metis#38's throughput window: a ring of the last 64 fold-completion
146:// instants. rate(now) = n / (now − oldest) — `now` in the denominator means a STALL
150:type movingRate struct {
155:func (m *movingRate) add(t time.Time) {
160:// rate returns completions/minute over the kept window; ok=false until 2 completions.
161:func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
170:	mins := now.Sub(oldest).Minutes()
171:	if mins <= 0 {
174:	return float64(kept) / mins, true
178:func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
203:	rate movingRate
207:// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
211:// driver-level (outer fold) completion ALWAYS emits; finish() emits the terminal
220:	rate      movingRate // metis#38: fold-completion throughput window
227:	width int               // terminal width ($COLUMNS | 80), read once at wiring
242:			foldTotal: totals.folds, foldKind: totals.foldKind,
312:	if direction == "minimize" {
318:// driverEvent folds a completed OUTER fold in — always emits (the coarse level is
335:// finish emits the terminal state line (always).

exec
/bin/zsh -lc 'git show HEAD:cmd/metis/board.go | rg -n "inner-CV runs|CV runs|folds|min|ETA|leaves|slots|starting|no .*run|last step"' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
18:// boardEnv is the render-time environment: terminal width, the current instant
19:// (for rate/ETA), and the leaf-semaphore gauge (capacity 0 = no gauge → segment
33:// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
46:	// Per-row denominators derive from the seeded aggregate totals (per-pass share).
64:			lines = append(lines, fmt.Sprintf("  fold %d ▸ configs %d/%d · folds %d/%d%s",
72:	// Leaves / throughput / ETA.
75:		segs = append(segs, fmt.Sprintf("leaves %d/%d", env.busy, env.capacity))
78:		segs = append(segs, fmt.Sprintf("%.1f folds/min", perMin))
80:		segs = append(segs, "— folds/min")
84:			segs = append(segs, "ETA "+fmtETA(eta))
95:// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
96:func fmtETA(d time.Duration) string {
107:// (Height analog of the width limitation: a terminal SHORTER than the board clamps
109:// lines; terminals that small are out of scope, same accepted trade as resize.)
126:// boardWriter owns the terminal: the board is pinned to the bottom while every
135:// hundreds of lines/second and real terminals — especially mux layers (the
137:// that flood. Now passthrough COALESCES into `pending` and the terminal sees one
160:// flushBudget bounds the erase/repaint rate: under a flood the terminal gets one
202:// (this is what re-pins the board after a burst window and keeps ETA/rate moving).
225:// flushLocked is the ONE atomic terminal update: erase the painted board, dump the
227:// output (metis#47: supporting terminals apply it atomically, killing the erase→redraw
237:	// Hold back an unterminated tail: a partial line fused into the board's first

exec
/bin/zsh -lc 'git show HEAD:cmd/metis/run.go | rg -n "activity|runRole|activityExecutor|runControlActivityEmitter"' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 exited 1 in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead

exec
/bin/zsh -lc "git show HEAD:workshop/issues/000049-board-readability-labels-coldphase-eta.md | sed -n '120,230p'" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead

## Done when

- Operator can read the first 2 minutes of a cold real sweep without wondering if it hung:
  factual startup activity present until the first inner-CV run completes; labels self-explanatory.
- Leaves + rate/ETA move smoothly (unit tests over scripted event traces pin the damping); after a
  mature trace, five seconds without a completion visibly advances the mode-specific last-run age by
  five seconds while the numeric rate/ETA move monotonically (regression test).
- Occupied-but-silent subprocesses are never described as proven progress; typed successful step
  completions/cache hits are the only positive activity signal.
- A failed run that successfully persists its failure record advances neither the eligible run
  counter nor rate/ETA (regression test).
- Flat and nested output, width clamping, repaint cadence, and terminal cleanup remain correct.
- RUNBOOK board description/example updated and its peer commit pinned in the Log.

## Plan

Durable plan: `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md`
(single pass, no Mx — one close boundary).

- [ ] Add typed step/run activity at the concrete executor and persistence seams, including cache,
  failure, role, ordering, and cancellation tests.
- [ ] Reduce time-driven occupancy and eligible-run telemetry with deterministic readiness, decay,
  recovery, and out-of-order-event tests.
- [ ] Render truthful flat/nested vocabulary, startup observations, last-run age, and mature rate/ETA;
  preserve repaint, failure, width, and terminal behavior.
- [ ] Update and commit the kbench Titanic RUNBOOK, pin its full commit here, then run focused,
  race, full-suite, formatting, and stale-vocabulary verification.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.05 impl=0.35
item: smaller-go-module   design=0.06 impl=0.35
item: tui-screen          design=0.15 impl=0.55
item: cross-cutting-refactor design=0.05 impl=0.25
item: smaller-go-module   design=0.06 impl=0.35
item: atlas-docs          design=0.02 impl=0.10
item: milestone-review    design=0.03 impl=0.20
design-buffer: 0.15
total: 2.63
```

Rows: (1) typed activity entities/decorator; (2) event-time reducer and rate/occupancy math;
(3) board state/rendering and scripted traces; (4) shared vocabulary migration; (5) concrete-run,
controller, and sweep wiring; (6) peer RUNBOOK plus atlas/stale-term sweep; (7) one SDLC milestone-review row.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

## Log

### 2026-07-16
- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
  #43 symptom: depth-first scheduling would give early train completions → a stable rate
  much sooner.

### 2026-07-16 — paired #43/#49 design approved
- Operator chose typed step-completion telemetry over presentation-only occupancy wording because
  occupancy cannot distinguish useful work from a hung subprocess. Co-designed after mapping the
  current sink/rate/compositor flow; #43 merges first, then #49 builds against its schedule.

### 2026-07-17 — planning checkpoint
- Reconciled the brain project after #43 merged, ran `sdlc start-plan`, mapped the final-executor,
  concrete-run persistence, run-control, sweep-progress, board, and kbench documentation seams, and
  authored the durable single-boundary TDD plan. Estimate uses v3.1 Method A; the approved spec and
  existing #38/#43 patterns make this familiar extension work rather than a novel TUI subsystem.
- Fresh-eyes plan review: Chunk 1 found Important gaps in shared run-control activity gating and
  concrete flat/nested role propagation; Chunk 2 found an Important gap in aggregate counter ownership.
  Patched the plan and both reviewers re-checked clean. Chunk 3 fresh-eyes review returned clean.

## Revisions

### 2026-07-16 — fresh-eyes spec review
- Split flat `CV runs` from nested `inner-CV runs`, made out-of-order callback reduction explicit,
  and replaced an unprovable seconds-level rounded-rate promise with a tick-driven last-run-age
  freshness signal plus a measurable scripted-trace criterion. Separated typed step activity from
  actual concrete-run completion so batch-order sampler callbacks cannot falsify the rate window;
  required execution success independently of successful failure-record persistence.

### 2026-07-17 — durable implementation plan
- Replaced the two placeholder plan rows with concrete typed-activity, pure-reducer, rendering,
  cross-repo documentation, and verification steps; added the calibrated estimate and explicit
  single-close-boundary decision.

codex
The committed HEAD still has the old `folds`/`leaves`/early-ETA board, while the working tree has uncommitted #49 edits. I’m going to do one more pass for any obvious issues in the uncommitted implementation, but the main verdict is already blocked because `HEAD` is not the code being claimed.
exec
/bin/zsh -lc "nl -ba workshop/issues/000049-board-readability-labels-coldphase-eta.md | sed -n '130,220p'" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
   130	- A failed run that successfully persists its failure record advances neither the eligible run
   131	  counter nor rate/ETA (regression test).
   132	- Flat and nested output, width clamping, repaint cadence, and terminal cleanup remain correct.
   133	- RUNBOOK board description/example updated and its peer commit pinned in the Log.
   134
   135	## Plan
   136
   137	Durable plan: `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md`
   138	(single pass, no Mx — one close boundary).
   139
   140	- [x] Add typed step/run activity at the concrete executor and persistence seams, including cache,
   141	  failure, role, ordering, and cancellation tests.
   142	- [x] Reduce time-driven occupancy and eligible-run telemetry with deterministic readiness, decay,
   143	  recovery, and out-of-order-event tests.
   144	- [x] Render truthful flat/nested vocabulary, startup observations, last-run age, and mature rate/ETA;
   145	  preserve repaint, failure, width, and terminal behavior.
   146	- [x] Update and commit the kbench Titanic RUNBOOK, pin its full commit here, then run focused,
   147	  race, full-suite, formatting, and stale-vocabulary verification.
   148
   149	## Estimate
   150
   151	```estimate
   152	model: estimate-logic-v3.1
   153	familiarity: 1.0
   154	item: smaller-go-module   design=0.05 impl=0.35
   155	item: smaller-go-module   design=0.06 impl=0.35
   156	item: tui-screen          design=0.15 impl=0.55
   157	item: cross-cutting-refactor design=0.05 impl=0.25
   158	item: smaller-go-module   design=0.06 impl=0.35
   159	item: atlas-docs          design=0.02 impl=0.10
   160	item: milestone-review    design=0.03 impl=0.20
   161	design-buffer: 0.15
   162	total: 2.63
   163	```
   164
   165	Rows: (1) typed activity entities/decorator; (2) event-time reducer and rate/occupancy math;
   166	(3) board state/rendering and scripted traces; (4) shared vocabulary migration; (5) concrete-run,
   167	controller, and sweep wiring; (6) peer RUNBOOK plus atlas/stale-term sweep; (7) one SDLC milestone-review row.
   168
   169	*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
   170	`baseline-v3.1.md`. Method A only.*
   171
   172	## Log
   173
   174	### 2026-07-16
   175	- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
   176	  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
   177	  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
   178	  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
   179	  #43 symptom: depth-first scheduling would give early train completions → a stable rate
   180	  much sooner.
   181
   182	### 2026-07-16 — paired #43/#49 design approved
   183	- Operator chose typed step-completion telemetry over presentation-only occupancy wording because
   184	  occupancy cannot distinguish useful work from a hung subprocess. Co-designed after mapping the
   185	  current sink/rate/compositor flow; #43 merges first, then #49 builds against its schedule.
   186
   187	### 2026-07-17 — planning checkpoint
   188	- Reconciled the brain project after #43 merged, ran `sdlc start-plan`, mapped the final-executor,
   189	  concrete-run persistence, run-control, sweep-progress, board, and kbench documentation seams, and
   190	  authored the durable single-boundary TDD plan. Estimate uses v3.1 Method A; the approved spec and
   191	  existing #38/#43 patterns make this familiar extension work rather than a novel TUI subsystem.
   192	- Fresh-eyes plan review: Chunk 1 found Important gaps in shared run-control activity gating and
   193	  concrete flat/nested role propagation; Chunk 2 found an Important gap in aggregate counter ownership.
   194	  Patched the plan and both reviewers re-checked clean. Chunk 3 fresh-eyes review returned clean.
   195
   196	### 2026-07-17 — implementation verification
   197	- Implemented typed step/run activity, run-role propagation, run-control-gated publication, event-time
   198	  eligible-run rate reduction, tick-smoothed `~slots`, factual cold-start board wording, shared
   199	  flat/nested progress vocabulary, and last-run-age/mature `~ETA` rendering. Updated atlas with the
   200	  durable activity seam and board contract.
   201	- Updated the kbench Titanic operator RUNBOOK and committed it as
   202	  `68edc6ca312dbe5306ce87315abec2d67e678005` (`docs: update metis board progress contract`).
   203	- Verification run: `go test ./cmd/metis -count=1`; `go test ./cmd/metis -race -count=1`;
   204	  `go test ./... -race -count=1`; metis `git diff --check`; kbench `git diff --check`; stale board
   205	  terminology grep over metis `cmd/`, `atlas/`, issue/plan, and the kbench RUNBOOK. Remaining grep hits
   206	  are historical problem/spec wording or negative test fixtures, not live operator output.
   207
   208	## Revisions
   209
   210	### 2026-07-16 — fresh-eyes spec review
   211	- Split flat `CV runs` from nested `inner-CV runs`, made out-of-order callback reduction explicit,
   212	  and replaced an unprovable seconds-level rounded-rate promise with a tick-driven last-run-age
   213	  freshness signal plus a measurable scripted-trace criterion. Separated typed step activity from
   214	  actual concrete-run completion so batch-order sampler callbacks cannot falsify the rate window;
   215	  required execution success independently of successful failure-record persistence.
   216
   217	### 2026-07-17 — durable implementation plan
   218	- Replaced the two placeholder plan rows with concrete typed-activity, pure-reducer, rendering,
   219	  cross-repo documentation, and verification steps; added the calibrated estimate and explicit
   220	  single-close-boundary decision.

exec
/bin/zsh -lc "nl -ba cmd/metis/activity.go | sed -n '1,120p'" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
     1	package main
     2
     3	import (
     4		"time"
     5
     6		"github.com/xianxu/metis/pkg/experiment"
     7	)
     8
     9	type activityKind string
    10
    11	const (
    12		activityStepSuccess activityKind = "step-success"
    13		activityRunSuccess  activityKind = "run-success"
    14	)
    15
    16	type runRole string
    17
    18	const (
    19		runRoleNone           runRole = ""
    20		runRoleNestedInnerCV  runRole = "nested-inner-cv"
    21		runRoleFlatCV         runRole = "flat-cv"
    22		runRoleNestedPreamble runRole = "nested-preamble"
    23		runRoleOuterScore     runRole = "outer-score"
    24	)
    25
    26	type activityEvent struct {
    27		Kind   activityKind
    28		At     time.Time
    29		StepID string
    30		RunID  string
    31		Role   runRole
    32	}
    33
    34	type activityEmitter func(activityEvent)
    35
    36	func (e activityEmitter) emit(ev activityEvent) {
    37		if e != nil {
    38			e(ev)
    39		}
    40	}
    41
    42	func teeActivityEmitter(a, b activityEmitter) activityEmitter {
    43		if a == nil {
    44			return b
    45		}
    46		if b == nil {
    47			return a
    48		}
    49		return func(ev activityEvent) {
    50			a.emit(ev)
    51			b.emit(ev)
    52		}
    53	}
    54
    55	func runControlActivityEmitter(control *runControl, emit activityEmitter) activityEmitter {
    56		if control == nil {
    57			return emit
    58		}
    59		return func(ev activityEvent) {
    60			control.whileHealthy(func() { emit.emit(ev) })
    61		}
    62	}
    63
    64	type activityExecutor struct {
    65		inner experiment.StepExecutor
    66		now   func() time.Time
    67		emit  activityEmitter
    68	}
    69
    70	func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
    71		res, err := e.inner.Execute(step, runDir)
    72		if err != nil {
    73			return res, err
    74		}
    75		now := e.now
    76		if now == nil {
    77			now = time.Now
    78		}
    79		e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
    80		return res, nil
    81	}

exec
/bin/zsh -lc 'git diff --check' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead

exec
/bin/zsh -lc 'git diff -- cmd/metis/activity.go cmd/metis/activity_test.go cmd/metis/progress.go cmd/metis/board.go cmd/metis/run.go cmd/metis/sweep.go | git apply --check --cached --allow-empty' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead

codex
```verdict
verdict: REWORK
confidence: high
```

The boundary cannot ship as `HEAD`: the issue tracker claims #49 is implemented and verified, but the implementation is still dirty/untracked in the working tree, outside the requested `Base..HEAD` window. `HEAD` still has the old board vocabulary and early ETA behavior, so the issue’s Done-when is not delivered by the reviewed revision.

1. Strengths:
- The uncommitted implementation direction looks aligned with the spec: typed `activityEvent`, run roles, pure `movingRate`, `occupancyWindow`, and board wording are present in the working tree.
- The working-tree tests appear to target real seams: cache activity, persistence-bound run success, run-control gating, and board rendering.

2. Critical findings:
- `workshop/issues/000049-board-readability-labels-coldphase-eta.md:140`: tracker marks all #49 plan rows complete, but `git status --short` shows the implementation is not in `HEAD` (`cmd/metis/activity.go` and `cmd/metis/activity_test.go` are untracked; core files are modified). The reviewed `HEAD` has no activity seam, and `HEAD:cmd/metis/board.go:64`, `:75`, `:78`, `:84` still render `fold`, `leaves`, `folds/min`, and un-gated `ETA`. Fix: commit the actual #49 implementation and rerun the boundary review on the new `HEAD`.
- `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:20`: Core concepts tables do not include the required greppable kind/file/status columns. This weakens the boundary traceability check. Fix: add a `## Revisions` entry or update the table with entity name, PURE/INTEGRATION kind, file location, and status.

3. Important findings:
- Docs gate is not reviewable for the committed boundary because the claimed implementation/log evidence is dirty relative to `HEAD`. After committing, verify `atlas/` and the pinned kbench RUNBOOK commit are in the same close evidence.

4. Minor findings:
- None beyond the boundary mismatch.

5. Test coverage notes:
- I did not run tests. The main blocker is that the implementation is outside `HEAD`, so test results against the dirty tree would not validate the requested boundary.

6. Architectural notes:
- ARCH-DRY: blocked for committed `HEAD`; old sampler callback counters still drive fold/rate display.
- ARCH-PURE: blocked for committed `HEAD`; typed pure reducers are not shipped.
- ARCH-PURPOSE: fail. The committed boundary under-delivers the issue’s purpose; the working tree may satisfy it, but it is not the reviewed artifact.

7. Plan revision recommendations:
- Add `## Revisions`: “Close review found #49 implementation was dirty/untracked outside `HEAD`; commit the activity/progress/board/run/sweep changes before re-running boundary review.”
- Add `## Revisions`: “Core concepts table updated with kind, file location, and status for every PURE/INTEGRATION entity.”
tokens used
86,560
```verdict
verdict: REWORK
confidence: high
```

The boundary cannot ship as `HEAD`: the issue tracker claims #49 is implemented and verified, but the implementation is still dirty/untracked in the working tree, outside the requested `Base..HEAD` window. `HEAD` still has the old board vocabulary and early ETA behavior, so the issue’s Done-when is not delivered by the reviewed revision.

1. Strengths:
- The uncommitted implementation direction looks aligned with the spec: typed `activityEvent`, run roles, pure `movingRate`, `occupancyWindow`, and board wording are present in the working tree.
- The working-tree tests appear to target real seams: cache activity, persistence-bound run success, run-control gating, and board rendering.

2. Critical findings:
- `workshop/issues/000049-board-readability-labels-coldphase-eta.md:140`: tracker marks all #49 plan rows complete, but `git status --short` shows the implementation is not in `HEAD` (`cmd/metis/activity.go` and `cmd/metis/activity_test.go` are untracked; core files are modified). The reviewed `HEAD` has no activity seam, and `HEAD:cmd/metis/board.go:64`, `:75`, `:78`, `:84` still render `fold`, `leaves`, `folds/min`, and un-gated `ETA`. Fix: commit the actual #49 implementation and rerun the boundary review on the new `HEAD`.
- `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:20`: Core concepts tables do not include the required greppable kind/file/status columns. This weakens the boundary traceability check. Fix: add a `## Revisions` entry or update the table with entity name, PURE/INTEGRATION kind, file location, and status.

3. Important findings:
- Docs gate is not reviewable for the committed boundary because the claimed implementation/log evidence is dirty relative to `HEAD`. After committing, verify `atlas/` and the pinned kbench RUNBOOK commit are in the same close evidence.

4. Minor findings:
- None beyond the boundary mismatch.

5. Test coverage notes:
- I did not run tests. The main blocker is that the implementation is outside `HEAD`, so test results against the dirty tree would not validate the requested boundary.

6. Architectural notes:
- ARCH-DRY: blocked for committed `HEAD`; old sampler callback counters still drive fold/rate display.
- ARCH-PURE: blocked for committed `HEAD`; typed pure reducers are not shipped.
- ARCH-PURPOSE: fail. The committed boundary under-delivers the issue’s purpose; the working tree may satisfy it, but it is not the reviewed artifact.

7. Plan revision recommendations:
- Add `## Revisions`: “Close review found #49 implementation was dirty/untracked outside `HEAD`; commit the activity/progress/board/run/sweep changes before re-running boundary review.”
- Add `## Revisions`: “Core concepts table updated with kind, file location, and status for every PURE/INTEGRATION entity.”

---

## Re-review — 2026-07-17T10:16:30-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 49 — board readability — label semantics, cold-phase "no progress" confusion, jumpy leaves, wild early ETA |
| repo | 000049-board-readability-labels-coldphase-eta |
| issue file | workshop/issues/000049-board-readability-labels-coldphase-eta.md |
| boundary | whole-issue close |
| milestone | — |
| window | 73cf6993cb9cbb87a11696f7adb378b071b39c81..HEAD |
| command | sdlc close --issue 49 |
| reviewer | codex |
| timestamp | 2026-07-17T10:16:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reading additional input from stdin...
OpenAI Codex v0.144.5
--------
workdir: /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
model: gpt-5.5
provider: openai
approval: never
sandbox: read-only
reasoning effort: medium
reasoning summaries: none
session id: 019f7111-f76e-7412-bc6b-97b91971a465
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **000049-board-readability-labels-coldphase-eta** repository.

- repository: 000049-board-readability-labels-coldphase-eta   (root: /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta)
- issue:      000049-board-readability-labels-coldphase-eta#49   (file: workshop/issues/000049-board-readability-labels-coldphase-eta.md)
- window:     Base: 73cf6993cb9cbb87a11696f7adb378b071b39c81   Head: HEAD

Review the **000049-board-readability-labels-coldphase-eta** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
other repository or apply another repo's conventions.

You have no prior session context — that is the anti-collusion property. Verify
behavior against the issue's documented Spec/Plan and the code itself; do NOT
take the implementor's word in commit messages or docs at face value. Tools are
read-only: report findings precisely; the main agent (which has session context)
applies the fixes, commits, and re-runs.

Read the diff against the issue's Spec + Plan, then work the checklist below.
Categorize every finding by severity — not everything is Critical; a nitpick
marked Critical is noise.

  Critical (must fix before crossing the boundary)
    - correctness bugs; crashes / panics on unexpected input
    - behavior drift from stated contracts (for ports of existing code where
      byte-faithfulness was promised, diff against the source)
    - silent error swallowing where the source raised
  Important (fix before the boundary if cheap)
    - API design of newly-introduced internal packages (downstream work will
      consume them; is the surface stable?)
    - missing test coverage that would catch the kind of bug shipped
    - inconsistent error handling across the diff
  Minor (note for future)
    - style nits, naming, comment density; performance only if hot-path

## Review checklist

Code quality
  - Clean separation of concerns; edge cases handled (empty / nil / unexpected).
  - Proper error handling — no silent swallowing where the source raised.
  - No duplicated logic / copy-paste that should be a shared helper.

Testing
  - Tests pin real logic, not mocks reasserting the implementation.
  - The kind of bug this diff could ship is covered.
  - PURE entities tested without IO; INTEGRATION via injected fakes (see below).

Requirements traceability
  - Every Plan checklist item this boundary claims is actually delivered.
  - Implementation matches the Spec; no undeclared scope creep.
  - Breaking changes documented.

Production readiness
  - Migration / backward-compatibility considered where state or formats change.
  - Docs / atlas updated for new surface (see the Docs update gate).

## Core concepts cross-check (if the plan has a Core concepts table)

The plan should list entities in a greppable table — name, kind
(PURE/INTEGRATION), file location, status (new/modified/deleted). For each row:
  - Verify the entity exists at the stated path (grep the diff or filesystem).
  - PURE: tests run without IO (no exec, net, mutable fs). If tests need mocks
    to run, it isn't really PURE — flag Critical and recommend promoting it to
    INTEGRATION.
  - INTEGRATION: injected into pure callers, not invoked directly from business
    logic.
  - "modified" / "deleted": the diff shows the expected change/removal at the
    stated location.
Any contradiction between table and code = Critical finding, plus a plan-revision
recommendation (a "## Revisions" entry so the plan stops claiming what the code
doesn't deliver).

## Docs update gate (atlas + README, per AGENTS.md §8)

The boundary should update user-facing docs for any new surface introduced:

  - **atlas/** — new architectural surface, flow, or terminology. Scan the diff
    for new entity types, subcommands, conventions, file-tree locations. Any
    present without corresponding atlas/ changes in the same range = Important
    finding ("atlas update appears missing for <surface>").
  - **README.md** — new user-facing surface a reader runs or types: subcommands,
    flags, keybindings, config keys, install/usage steps. If the diff adds or
    changes such surface and README.md is not updated in the same range =
    Important finding ("README update appears missing for <surface>"). This is the
    class of gap that used to surface only at the merge-time `specs` judge (#142);
    catch it here, at the earliest gate, before the close verdict is recorded.

## Architecture (the at-review backstop — these matter most long-term)

Work through each of ARCH-DRY, ARCH-PURE, ARCH-PURPOSE explicitly, applying its at-review lens. The
full principle definitions are delivered in the ARCHITECTURE PRINCIPLES block
right after this prompt — for EACH marker, state pass or flag, and cite the
marker (e.g. ARCH-DRY) in any finding. Architecture is where review has the
least training signal and the longest-delayed payoff, so be deliberate here, not
holistic.

## Verdict + output

Begin your response with this fenced verdict block — the machine-read handoff:

```verdict
verdict: <SHIP | FIX-THEN-SHIP | REWORK>
confidence: <high | medium | low>
```

  SHIP           ready; ship it
  FIX-THEN-SHIP  ship after addressing the findings (non-blocking at the gate)
  REWORK         blocking; needs rework before shipping — fix + re-run

The fenced ```` ```verdict ```` block above is the **authoritative machine-read
handoff** — emit it as the first thing in your response. (A prose
`VERDICT: <TOKEN>` first line still satisfies the legacy contract as a fallback,
but the block is what the binary trusts.)

After the verdict block: a 1-paragraph summary — what worked, what blocks SHIP if
it isn't — followed by:
  1. Strengths: 2-5 specific things done well (file:line where useful). Affirm
     validated approaches so the operator knows what's confirmed-good ground.
     Empty acceptable for trivial boundaries.
  2. Critical findings (file:line + fix sketch); empty if none.
  3. Important findings (same format).
  4. Minor findings (terse one-liners).
  5. Test coverage notes.
  6. Architectural notes for upcoming work.
  7. Plan revision recommendations: specific "## Revisions" entries the plan
     needs (empty if the plan still matches the code).


ARCHITECTURE PRINCIPLES — work through each of the 3 entries below explicitly, applying its `at-review` lens; cite the marker (e.g. ARCH-DRY) in any finding.

# Architecture principles (ARCH-*)

Injected architectural taste — the structural decisions whose payoff (or cost)
shows up many turns, often months, down the road. Agents are strong at local
tactics and weak here, so these are checked **at-plan** (when the design is being
made — highest leverage) and **at-review** (backstop, on the diff). Cite the
marker (e.g. `ARCH-DRY`) in plans, `## Log` entries, and review findings.

This file is the single source; it is embedded into the planning, plan-quality,
and code-review prompts. The human narrative lives in AGENTS.md "Core Design
Principles"; this is its machine-delivered companion.

## ARCH-DRY — Don't Repeat Yourself

- **principle:** Reuse before adding. One source of truth per fact/behavior; no
  duplicated logic, copy-pasted blocks, or parallel functions that should be one
  shared helper.
- **at-plan:** Flag a plan that re-implements something the codebase already has,
  or that will obviously duplicate logic across the new files instead of
  extracting a shared helper. Name the existing thing it should reuse.
- **at-review:** Flag duplicated logic / copy-pasted blocks / near-identical
  functions in the diff; point at the consolidation (file:line + the shared
  helper they should become).

## ARCH-PURE — Pure core, thin IO shell

- **principle:** The majority of code is pure functions (deterministic, no side
  effects); a thin "glue" layer at the boundary touches IO/UI/network/clock. Pure
  functions are unit-tested directly; the glue is kept small and injected.
- **at-plan:** Flag a design that buries business logic inside IO/handlers, or
  that will only be testable with heavy mocks (a sign logic isn't separated from
  IO). The plan should name what's pure vs the thin IO seam.
- **at-review:** Flag business logic mixed with IO in the diff; logic that should
  be a pure function injected into a thin caller. If a test needs mocks to run a
  "pure" entity, it isn't pure — recommend extracting the IO to the boundary.

## ARCH-PURPOSE — Serve the issue's actual purpose

- **principle:** Deliver the issue's stated purpose, not the easy subset of it. A
  single-source / "compiled to consumers" change is not done until **every
  consumer derives** from the source — the source is *enforced*, not just
  documentation a surface happens to restate; a hand-maintained restatement of the
  model is a deferred consumer, not a finished one. "Follow-up" is for separable
  extensions, never for the thing that is the point. This is the *opposite axis*
  from Simplicity-First/YAGNI: not "build for an imagined future," but "don't
  **under**-deliver the purpose you already committed to."
- **at-plan:** Flag a plan whose scope is a strict subset of the issue's stated
  goal / Done-when where the part deferred as "follow-up" *is* the purpose (e.g.
  wires one consumer + enforcement but leaves the consumers that motivated the
  issue as documentation that doesn't derive). Ask: does the plan fulfill the
  purpose, or just the cheap win? Name the deferred purpose.
- **at-review:** Does the diff *fulfill* the purpose or settle for the easy win?
  For a single-source change, run the **shadow-sweep** — enumerate the consumers,
  confirm each derives from the source, flag any remaining hand-maintained
  restatement of the model. A "follow-up" that is actually the deferred point of
  the issue is a finding, not a deferral.


OUTPUT CONTRACT (machine-read — do not deviate). LEAD your response with the
fenced ```verdict block shown above — that is the authoritative handoff the binary
reads (its `verdict:` value is one of the listed tokens). Everything after the block
is advisory: a non-blocking verdict WITH findings still PASSES the gate. A bare
`VERDICT: <TOKEN>` line is accepted only as a FALLBACK when the block is absent.

Diff:
diff --git a/atlas/index.md b/atlas/index.md
index 18d8b0a..a33080c 100644
--- a/atlas/index.md
+++ b/atlas/index.md
@@ -86,17 +86,28 @@ identical on a non-Kaggle platform?* — if yes, it lives here.
   `progress func(ProgressEvent[P,O])` fired **at point completion** (not Tell — under #31's batch exec
   every Tell lands at batch end), mutex-serialized so k arrives monotone even under ParExec; nil = the
   unwrapped loop. `cmd/metis/progress.go` is the sink: ONE throttled aggregated line
-  (`metis: progress outer 1/3 · configs 84/216 · folds 421/1080 · est 0.8283 ± 0.0140`; flat 1-config →
-  `folds 3/5 · score …`), totals **seeded at wiring** from direct SizeHint calls, 1s throttle on the
-  injected clock, always-emit on outer completions + finish, plain lines (no escape codes — non-TTY-safe).
-  **Outer-fold identity rides the per-pass closure binding** (`prog.forPass(i)` → `passHooks`), never an
-  event payload field — metis#38's TTY board extends the sink behind those hooks with zero pkg/sampler
-  change. (`FoldPoint.Partition` looks like a discriminator but is byte-identical across outer folds.)
-  **metis#38 (the live board, `cmd/metis/board.go`):** on a TTY a sweep pins a live board to the bottom
-  while step logs scroll above — aggregate line, one row per outer fold (`✓ held-out` | `▸ configs a/b ·
-  folds x/y · best` | queued, ≤12 rows + overflow), and `leaves b/c · R folds/min · ETA` (leaves =
-  `len/cap(leafSem)`; rate = a 64-completion ring with `now` in the denominator so a stall DECAYS it live
-  — the BLAS-thrash signature). Hand-rolled ANSI pin-bottom, NO TUI lib (output-only board; 2-dep module).
+  (`metis: progress outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`;
+  flat 1-config → `CV runs 3/5 · score …`), totals **seeded at wiring** from direct SizeHint calls, 1s
+  throttle on the injected clock, always-emit on outer completions + finish, plain lines (no escape codes —
+  non-TTY-safe). **metis#49:** sampler fold callbacks now own per-row score/config display only; aggregate
+  run counts/rate come from typed concrete-run activity events emitted after successful run execution and
+  required `run.json`/`record.json` persistence. Step activity is emitted by an executor decorator outside
+  the cache, so warm cache hits still publish successful work facts. Run roles (`flat CV`, `nested inner-CV`,
+  `nested preamble`, `outer score`) keep rate/ETA eligibility at the concrete call sites, and `runControl`
+  gates both step and run activity before repaint after a fatal sibling failure. **Outer-fold identity rides
+  the per-pass closure binding** (`prog.forPass(i)` → `passHooks`), never an event payload field — metis#38's
+  TTY board extends the sink behind those hooks with zero pkg/sampler change. (`FoldPoint.Partition` looks
+  like a discriminator but is byte-identical across outer folds.)
+  **metis#38/#49 (the live board, `cmd/metis/board.go`):** on a TTY a sweep pins a live board to the bottom
+  while step logs scroll above — aggregate line, one row per outer fold (`outer fold N ✓ held-out` |
+  `outer fold N ▸ configs scored a/b · inner-CV runs x/y · best` | queued, ≤12 rows + overflow), and a
+  status line with tick-smoothed `~slots b/c`, last-run age, mode-specific rate (`inner-CV runs/min` or
+  `CV runs/min`), and mature `~ETA`. Rate/ETA are withheld until 16 eligible completions span at least
+  15 seconds; before the first eligible run, the board prints factual startup observations such as step
+  count/last-step age and never diagnoses "not hung" or a phase. The event-time rate keeps the latest 64
+  eligible completions sorted by completion time and uses `now-oldest`, so a stall decays live while
+  last-run age remains the sharp freshness signal. Hand-rolled ANSI pin-bottom, NO TUI lib (output-only
+  board; 2-dep module).
   Split: `renderBoard` is pure (plain lines, zero ANSI) / `boardWriter` is the paint-only compositor
   (erase-count bookkeeping; holds unterminated passthrough tails; idempotent deferred close).
   **metis#46: the compositor is DOUBLE-BUFFERED with a 250ms flush budget** — passthrough coalesces
diff --git a/cmd/metis/activity.go b/cmd/metis/activity.go
new file mode 100644
index 0000000..437b43b
--- /dev/null
+++ b/cmd/metis/activity.go
@@ -0,0 +1,81 @@
+package main
+
+import (
+	"time"
+
+	"github.com/xianxu/metis/pkg/experiment"
+)
+
+type activityKind string
+
+const (
+	activityStepSuccess activityKind = "step-success"
+	activityRunSuccess  activityKind = "run-success"
+)
+
+type runRole string
+
+const (
+	runRoleNone           runRole = ""
+	runRoleNestedInnerCV  runRole = "nested-inner-cv"
+	runRoleFlatCV         runRole = "flat-cv"
+	runRoleNestedPreamble runRole = "nested-preamble"
+	runRoleOuterScore     runRole = "outer-score"
+)
+
+type activityEvent struct {
+	Kind   activityKind
+	At     time.Time
+	StepID string
+	RunID  string
+	Role   runRole
+}
+
+type activityEmitter func(activityEvent)
+
+func (e activityEmitter) emit(ev activityEvent) {
+	if e != nil {
+		e(ev)
+	}
+}
+
+func teeActivityEmitter(a, b activityEmitter) activityEmitter {
+	if a == nil {
+		return b
+	}
+	if b == nil {
+		return a
+	}
+	return func(ev activityEvent) {
+		a.emit(ev)
+		b.emit(ev)
+	}
+}
+
+func runControlActivityEmitter(control *runControl, emit activityEmitter) activityEmitter {
+	if control == nil {
+		return emit
+	}
+	return func(ev activityEvent) {
+		control.whileHealthy(func() { emit.emit(ev) })
+	}
+}
+
+type activityExecutor struct {
+	inner experiment.StepExecutor
+	now   func() time.Time
+	emit  activityEmitter
+}
+
+func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+	res, err := e.inner.Execute(step, runDir)
+	if err != nil {
+		return res, err
+	}
+	now := e.now
+	if now == nil {
+		now = time.Now
+	}
+	e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
+	return res, nil
+}
diff --git a/cmd/metis/activity_test.go b/cmd/metis/activity_test.go
new file mode 100644
index 0000000..98bd5ad
--- /dev/null
+++ b/cmd/metis/activity_test.go
@@ -0,0 +1,78 @@
+package main
+
+import (
+	"errors"
+	"reflect"
+	"testing"
+	"time"
+
+	"github.com/xianxu/metis/pkg/experiment"
+)
+
+type activityFakeExec struct {
+	result experiment.StepResult
+	err    error
+	calls  int
+}
+
+func (f *activityFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+	f.calls++
+	return f.result, f.err
+}
+
+func TestActivityExecutorEmitsOneStepSuccessAfterSuccessfulInnerExecution(t *testing.T) {
+	at := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
+	wantResult := experiment.StepResult{
+		Metrics:   map[string]float64{"score": 0.91},
+		Artifacts: []string{"train/model.bin"},
+	}
+	inner := &activityFakeExec{result: wantResult}
+	var events []activityEvent
+
+	got, err := activityExecutor{
+		inner: inner,
+		now:   func() time.Time { return at },
+		emit:  func(ev activityEvent) { events = append(events, ev) },
+	}.Execute(experiment.Step{ID: "train"}, "/tmp/run")
+
+	if err != nil {
+		t.Fatalf("Execute returned error: %v", err)
+	}
+	if !reflect.DeepEqual(got, wantResult) {
+		t.Fatalf("Execute result = %+v; want %+v", got, wantResult)
+	}
+	if inner.calls != 1 {
+		t.Fatalf("inner calls = %d; want 1", inner.calls)
+	}
+	if len(events) != 1 {
+		t.Fatalf("events = %d; want 1 (%+v)", len(events), events)
+	}
+	if events[0].Kind != activityStepSuccess || events[0].StepID != "train" || !events[0].At.Equal(at) {
+		t.Fatalf("event = %+v; want one step-success event for train at %s", events[0], at.Format(time.RFC3339))
+	}
+}
+
+func TestActivityExecutorEmitsNothingOnInnerErrorAndPreservesFailure(t *testing.T) {
+	wantErr := errors.New("boom")
+	inner := &activityFakeExec{err: wantErr}
+	var events []activityEvent
+
+	got, err := activityExecutor{
+		inner: inner,
+		now:   func() time.Time { return time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC) },
+		emit:  func(ev activityEvent) { events = append(events, ev) },
+	}.Execute(experiment.Step{ID: "train"}, "/tmp/run")
+
+	if !errors.Is(err, wantErr) {
+		t.Fatalf("Execute error = %v; want %v", err, wantErr)
+	}
+	if !reflect.DeepEqual(got, experiment.StepResult{}) {
+		t.Fatalf("Execute result = %+v; want zero result from inner failure", got)
+	}
+	if inner.calls != 1 {
+		t.Fatalf("inner calls = %d; want 1", inner.calls)
+	}
+	if len(events) != 0 {
+		t.Fatalf("events = %+v; want none on error", events)
+	}
+}
diff --git a/cmd/metis/board.go b/cmd/metis/board.go
index 0407870..6e8d98e 100644
--- a/cmd/metis/board.go
+++ b/cmd/metis/board.go
@@ -53,15 +53,15 @@ func renderBoard(bs boardState, env boardEnv) []string {
		r := bs.rows[i]
		switch {
		case r.done:
-			lines = append(lines, fmt.Sprintf("  fold %d ✓ held-out %.4f", i, r.heldOut))
+			lines = append(lines, fmt.Sprintf("  outer fold %d ✓ held-out %.4f", i, r.heldOut))
		case r.configK == 0 && r.foldK == 0:
-			lines = append(lines, fmt.Sprintf("  fold %d — queued", i))
+			lines = append(lines, fmt.Sprintf("  outer fold %d — queued", i))
		default:
			b := ""
			if r.hasBest {
				b = fmt.Sprintf(" · best %.4f", r.best)
			}
-			lines = append(lines, fmt.Sprintf("  fold %d ▸ configs %d/%d · folds %d/%d%s",
+			lines = append(lines, fmt.Sprintf("  outer fold %d ▸ configs scored %d/%d · inner-CV runs %d/%d%s",
				i, r.configK, perConfigs, r.foldK, perFolds, b))
		}
	}
@@ -71,17 +71,42 @@ func renderBoard(bs boardState, env boardEnv) []string {

	// Leaves / throughput / ETA.
	var segs []string
+	runNoun := "CV runs"
+	runSingular := "CV run"
+	if bs.st.nested {
+		runNoun = "inner-CV runs"
+		runSingular = "inner-CV run"
+	}
	if env.capacity > 0 {
-		segs = append(segs, fmt.Sprintf("leaves %d/%d", env.busy, env.capacity))
+		segs = append(segs, fmt.Sprintf("~slots %d/%d", env.busy, env.capacity))
+	}
+	if bs.st.foldK == 0 {
+		start := []string{"starting"}
+		start = append(start, segs...)
+		if bs.st.stepK > 0 {
+			start = append(start, fmt.Sprintf("%d steps completed", bs.st.stepK))
+			if !bs.st.lastStepAt.IsZero() {
+				start = append(start, "last step "+fmtAge(env.now.Sub(bs.st.lastStepAt))+" ago")
+			}
+		}
+		start = append(start, fmt.Sprintf("no %s complete", runSingular))
+		lines = append(lines, strings.Join(start, " · "))
+		for i, l := range lines {
+			lines[i] = clampLine(l, env.width)
+		}
+		return lines
+	}
+	if !bs.st.lastRunAt.IsZero() {
+		segs = append(segs, fmt.Sprintf("last %s %s ago", runSingular, fmtAge(env.now.Sub(bs.st.lastRunAt))))
	}
	if perMin, ok := bs.rate.rate(env.now); ok {
-		segs = append(segs, fmt.Sprintf("%.1f folds/min", perMin))
+		segs = append(segs, fmt.Sprintf("%.1f %s/min", perMin, runNoun))
	} else {
-		segs = append(segs, "— folds/min")
+		segs = append(segs, fmt.Sprintf("— %s/min", runNoun))
	}
	if remaining := bs.st.foldTotal - bs.st.foldK; remaining > 0 {
		if eta, ok := bs.rate.eta(env.now, remaining); ok {
-			segs = append(segs, "ETA "+fmtETA(eta))
+			segs = append(segs, "~ETA "+fmtETA(eta))
		}
	}
	lines = append(lines, strings.Join(segs, " · "))
@@ -92,6 +117,13 @@ func renderBoard(bs boardState, env boardEnv) []string {
	return lines
 }

+func fmtAge(d time.Duration) string {
+	if d < 0 {
+		d = 0
+	}
+	return fmtETA(d)
+}
+
 // fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
 func fmtETA(d time.Duration) string {
	d = d.Round(time.Second)
diff --git a/cmd/metis/board_test.go b/cmd/metis/board_test.go
index 67ba9e9..29861d3 100644
--- a/cmd/metis/board_test.go
+++ b/cmd/metis/board_test.go
@@ -25,8 +25,9 @@ func TestRenderBoard(t *testing.T) {
			},
			rows: rows,
		}
-		st.rate.add(at(0))
-		st.rate.add(at(2824)) // 2 completions in 2.824s ≈ 42.5/min at now=2824
+		for i := 0; i < 16; i++ {
+			st.rate.add(at(i * 1000))
+		}
		return st
	}
	rows := []passRow{
@@ -34,14 +35,14 @@ func TestRenderBoard(t *testing.T) {
		{configK: 8, foldK: 25, best: 0.834, hasBest: true},
		{}, // queued: no events yet
	}
-	lines := renderBoard(mkState(rows), boardEnv{width: 100, now: at(2824), busy: 8, capacity: 8})
+	lines := renderBoard(mkState(rows), boardEnv{width: 100, now: at(21176), busy: 8, capacity: 8})
	frame := strings.Join(lines, "\n")
	for _, want := range []string{
-		"outer 1/3", "configs 14/36", "folds 47/108", "est 0.7980",
-		"fold 0 ✓ held-out 0.7980",
-		"fold 1 ▸ configs 8/12 · folds 25/36 · best 0.8340",
-		"fold 2 — queued",
-		"leaves 8/8", "42.5 folds/min", "ETA",
+		"outer folds 1/3", "configs scored 14/36", "inner-CV runs 47/108", "est 0.7980",
+		"outer fold 0 ✓ held-out 0.7980",
+		"outer fold 1 ▸ configs scored 8/12 · inner-CV runs 25/36 · best 0.8340",
+		"outer fold 2 — queued",
+		"~slots 8/8", "42.5 inner-CV runs/min", "~ETA",
	} {
		if !strings.Contains(frame, want) {
			t.Errorf("frame missing %q:\n%s", want, frame)
@@ -60,22 +61,22 @@ func TestRenderBoard(t *testing.T) {
	st := mkState(allDone)
	st.st.outerK, st.st.foldK, st.st.configK = 3, 108, 36
	st.st.outerScores = []float64{0.79, 0.81, 0.82}
-	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(2824), busy: 0, capacity: 8}), "\n")
+	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(21176), busy: 0, capacity: 8}), "\n")
	if strings.Contains(frame, "▸") || strings.Contains(frame, "ETA") {
		t.Errorf("all-done: no in-flight rows, no ETA:\n%s", frame)
	}

-	// Flat (no rows): exactly 2 lines — the aggregate + leaves.
+	// Flat (no rows): exactly 2 lines — the aggregate + slots/rate.
	flat := boardState{st: progressState{foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact, flatScores: []float64{0.8}}}
	if got := renderBoard(flat, boardEnv{width: 100, now: at(0), busy: 2, capacity: 8}); len(got) != 2 {
		t.Errorf("flat board = aggregate + leaves, got %d lines: %v", len(got), got)
	}

-	// Overflow: 14 folds → 12 rows + "… +2 more" + leaves + aggregate = 15 lines.
+	// Overflow: 14 folds → 12 rows + "… +2 more" + slots/rate + aggregate = 15 lines.
	many := make([]passRow, 14)
	st = mkState(many)
	st.st.outerTotal = 14
-	if got := renderBoard(st, boardEnv{width: 100, now: at(2824), busy: 8, capacity: 8}); len(got) != 15 {
+	if got := renderBoard(st, boardEnv{width: 100, now: at(21176), busy: 8, capacity: 8}); len(got) != 15 {
		t.Errorf("overflow: want 15 lines (1+12+1+1), got %d", len(got))
	} else if !strings.Contains(strings.Join(got, "\n"), "+2 more") {
		t.Errorf("overflow marker missing:\n%s", strings.Join(got, "\n"))
@@ -92,24 +93,104 @@ func TestRenderBoard(t *testing.T) {
		t.Error("clamped lines should truncate with …")
	}

-	// No gauge (capacity 0): leaves segment absent, throughput still present.
-	frame = strings.Join(renderBoard(mkState(rows), boardEnv{width: 100, now: at(2824)}), "\n")
-	if strings.Contains(frame, "leaves") {
-		t.Errorf("no gauge → no leaves segment:\n%s", frame)
+	// No gauge (capacity 0): slots segment absent, throughput still present.
+	frame = strings.Join(renderBoard(mkState(rows), boardEnv{width: 100, now: at(21176)}), "\n")
+	if strings.Contains(frame, "slots") {
+		t.Errorf("no gauge → no slots segment:\n%s", frame)
	}
-	if !strings.Contains(frame, "folds/min") {
+	if !strings.Contains(frame, "inner-CV runs/min") {
		t.Errorf("throughput must survive a missing gauge:\n%s", frame)
	}

-	// Rate unavailable (fresh ring): "— folds/min".
+	// Rate unavailable (fresh ring): "— inner-CV runs/min".
	st = mkState(rows)
	st.rate = movingRate{}
	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(0), busy: 1, capacity: 8}), "\n")
-	if !strings.Contains(frame, "— folds/min") {
+	if !strings.Contains(frame, "— inner-CV runs/min") {
		t.Errorf("unavailable rate renders —:\n%s", frame)
	}
 }

+func TestRenderBoardStartupObservationsAreFactual(t *testing.T) {
+	now := at(5000)
+	bs := boardState{st: progressState{
+		nested: true, foldTotal: 10, foldKind: sampler.SizeExact,
+		stepK: 37, lastStepAt: at(4000),
+	}}
+	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 8, capacity: 12}), "\n")
+	for _, want := range []string{
+		"starting", "~slots 8/12", "37 steps completed", "last step 1s ago", "no inner-CV run complete",
+	} {
+		if !strings.Contains(frame, want) {
+			t.Fatalf("startup frame missing %q:\n%s", want, frame)
+		}
+	}
+	for _, forbidden := range []string{"not hung", "warming"} {
+		if strings.Contains(frame, forbidden) {
+			t.Fatalf("startup frame made diagnosis %q:\n%s", forbidden, frame)
+		}
+	}
+
+	bs.st.foldK = 1
+	bs.st.lastRunAt = now
+	frame = strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 8, capacity: 12}), "\n")
+	if strings.Contains(frame, "starting") || strings.Contains(frame, "no inner-CV run complete") {
+		t.Fatalf("startup line must disappear after first eligible run:\n%s", frame)
+	}
+}
+
+func TestRenderBoardFlatStartupAndConfidenceLabels(t *testing.T) {
+	now := at(30000)
+	bs := boardState{st: progressState{
+		foldTotal: 20, foldKind: sampler.SizeExact,
+		stepK: 4, lastStepAt: at(28000),
+	}}
+	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 3, capacity: 8}), "\n")
+	for _, want := range []string{
+		"CV runs 0/20", "starting", "~slots 3/8", "4 steps completed", "last step 2s ago", "no CV run complete",
+	} {
+		if !strings.Contains(frame, want) {
+			t.Fatalf("flat startup frame missing %q:\n%s", want, frame)
+		}
+	}
+	if strings.Contains(frame, "inner-CV") || strings.Contains(frame, "warming") || strings.Contains(frame, "not hung") {
+		t.Fatalf("flat startup frame contains nested or diagnostic wording:\n%s", frame)
+	}
+
+	bs.st.foldK = 1
+	bs.st.lastRunAt = at(29000)
+	frame = strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 3, capacity: 8}), "\n")
+	for _, want := range []string{"CV runs 1/20", "last CV run 1s ago", "— CV runs/min"} {
+		if !strings.Contains(frame, want) {
+			t.Fatalf("flat pre-confidence frame missing %q:\n%s", want, frame)
+		}
+	}
+	if strings.Contains(frame, "starting") || strings.Contains(frame, "ETA") {
+		t.Fatalf("flat pre-confidence frame must be post-startup without ETA:\n%s", frame)
+	}
+}
+
+func TestRenderBoardMatureShowsLastRunAge(t *testing.T) {
+	now := at(20000)
+	var rate movingRate
+	for i := 0; i < 16; i++ {
+		rate.add(at(i * 1000))
+	}
+	bs := boardState{
+		st: progressState{
+			nested: true, foldK: 16, foldTotal: 32, foldKind: sampler.SizeExact,
+			lastRunAt: at(15000),
+		},
+		rate: rate,
+	}
+	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 4, capacity: 8}), "\n")
+	for _, want := range []string{"45.0 inner-CV runs/min", "last inner-CV run 5s ago", "~ETA"} {
+		if !strings.Contains(frame, want) {
+			t.Fatalf("mature frame missing %q:\n%s", want, frame)
+		}
+	}
+}
+
 // fmtETA is compact and human: seconds under a minute, m+s under an hour.
 func TestFmtETA(t *testing.T) {
	cases := map[time.Duration]string{
@@ -277,7 +358,7 @@ func TestRunExperiment_BoardMode(t *testing.T) {
	if strings.Contains(s, "metis: progress") {
		t.Errorf("the board REPLACES the plain progress lines")
	}
-	if !strings.Contains(s, "outer 2/2") || !strings.Contains(s, "fold 0 ✓") || !strings.Contains(s, "fold 1 ✓") {
+	if !strings.Contains(s, "outer folds 2/2") || !strings.Contains(s, "outer fold 0 ✓") || !strings.Contains(s, "outer fold 1 ✓") {
		t.Errorf("the final frame must show completed folds:\n%s", s)
	}
	// The bypass route: the fake-exec fixture has no traced closure → captureSweepCode
@@ -288,7 +369,7 @@ func TestRunExperiment_BoardMode(t *testing.T) {
	if warnIdx < 0 {
		t.Fatalf("expected the uncaptured-code note in a fake-exec fixture:\n%s", s)
	}
-	if finalFrame := strings.LastIndex(s, "outer 2/2"); warnIdx > finalFrame {
+	if finalFrame := strings.LastIndex(s, "outer folds 2/2"); warnIdx > finalFrame {
		t.Errorf("the capture warning bypassed the compositor (after the final frame)")
	}

@@ -347,7 +428,7 @@ func TestRunExperiment_BoardFailureRejectsPostPublicationTickAndDiscardsFrame(t
	awaitRunControl(t, tickSelected, "pre-failure board tick selection")
	awaitRunControl(t, tickFinished, "pre-failure board tick completion")
	preFailure := out.snapshot()
-	for _, want := range []string{"outer 0/2", "fold 0 — queued", "folds/min"} {
+	for _, want := range []string{"outer folds 0/2", "outer fold 0 — queued", "no inner-CV run"} {
		if !strings.Contains(preFailure, want) {
			t.Fatalf("pre-failure board missing %q:\n%s", want, preFailure)
		}
@@ -367,7 +448,7 @@ func TestRunExperiment_BoardFailureRejectsPostPublicationTickAndDiscardsFrame(t
	}
	suffix := out.snapshot()[offset:]
	for _, forbidden := range []string{
-		"outer 0/2", "fold 0 — queued", "configs ", "folds ", "folds/min", "ETA", "score ", "estimate", "mean ",
+		"outer folds 0/2", "outer fold 0 — queued", "configs ", "inner-CV runs ", "inner-CV runs/min", "ETA", "score ", "estimate", "mean ",
	} {
		if strings.Contains(suffix, forbidden) {
			t.Errorf("post-publication board output contains stale token %q:\n%q", forbidden, suffix)
@@ -526,7 +607,7 @@ func TestBoardWriter_ForceFlushDrainsPending(t *testing.T) {
	bw2 := newBoardWriter(&term2, func() time.Time { return at(0) })
	prog.bw, prog.width = bw2, 100
	prog.tick()
-	if !strings.Contains(term2.String(), "outer 0/1") {
+	if !strings.Contains(term2.String(), "outer folds 0/1") {
		t.Errorf("tick must render + force-paint the current frame: %q", term2.String())
	}
 }
diff --git a/cmd/metis/caching_test.go b/cmd/metis/caching_test.go
index 5b07810..66bba37 100644
--- a/cmd/metis/caching_test.go
+++ b/cmd/metis/caching_test.go
@@ -132,6 +132,69 @@ steps:
	}
 }

+func TestCacheActivityEmitsOnColdExecutionAndWarmHit(t *testing.T) {
+	if _, err := exec.LookPath("git"); err != nil {
+		t.Skip("git not on PATH (cache re-hash uses git hash-object)")
+	}
+	root := repoRoot(t)
+	ws := t.TempDir()
+	expPath := filepath.Join(ws, "exp.md")
+	md := `---
+type: experiment
+id: cache-activity
+seed: 5
+status: active
+steps:
+  - id: prep
+    uses: test/echo
+    with: {k: 5}
+---
+`
+	if err := os.WriteFile(expPath, []byte(md), 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	var stepEvents []activityEvent
+	run := func(runID string) string {
+		var out strings.Builder
+		_, err := runExperiment(runOpts{
+			expPath:  expPath,
+			runID:    runID,
+			stepPath: []string{filepath.Join(root, "testdata", "steps")},
+			now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
+			git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
+			cache:    true,
+			out:      &out,
+			activity: func(ev activityEvent) {
+				if ev.Kind == activityStepSuccess {
+					stepEvents = append(stepEvents, ev)
+				}
+			},
+		})
+		if err != nil {
+			t.Fatalf("run %s: %v", runID, err)
+		}
+		return out.String()
+	}
+
+	out1 := run("activity-cold")
+	if strings.Contains(out1, "cache hit") {
+		t.Fatalf("cold run should not have cache hits; got:\n%s", out1)
+	}
+	out2 := run("activity-warm")
+	if !hitFor(out2, "prep") {
+		t.Fatalf("warm run should hit prep; got:\n%s", out2)
+	}
+	if len(stepEvents) != 2 {
+		t.Fatalf("step events = %d; want one cold execution and one warm cache-hit event: %+v", len(stepEvents), stepEvents)
+	}
+	for i, ev := range stepEvents {
+		if ev.Kind != activityStepSuccess || ev.StepID != "prep" {
+			t.Fatalf("event %d = %+v; want prep step success", i, ev)
+		}
+	}
+}
+
 func hitFor(out, step string) bool {
	return strings.Contains(out, "step "+step+" (cache hit)")
 }
diff --git a/cmd/metis/nestedcv_e2e_test.go b/cmd/metis/nestedcv_e2e_test.go
index 1800d60..f6b9133 100644
--- a/cmd/metis/nestedcv_e2e_test.go
+++ b/cmd/metis/nestedcv_e2e_test.go
@@ -66,7 +66,7 @@ func TestNestedCV_ProducesHonestEstimateNoShip(t *testing.T) {
	}
	finalProg := s[strings.LastIndex(s, "metis: progress"):]
	finalProg = finalProg[:strings.IndexByte(finalProg, '\n')]
-	if !strings.Contains(finalProg, "outer 2/2") || !strings.Contains(finalProg, "est 0.") {
+	if !strings.Contains(finalProg, "outer folds 2/2") || !strings.Contains(finalProg, "est 0.") {
		t.Errorf("the final progress line must carry the completed outer count + a numeric est; got: %q", finalProg)
	}
	// metis#50: the run ends with a paste-ready summary — elapsed, rows → ledger, cohort,
diff --git a/cmd/metis/progress.go b/cmd/metis/progress.go
index 71ee241..c3463c0 100644
--- a/cmd/metis/progress.go
+++ b/cmd/metis/progress.go
@@ -12,6 +12,7 @@ import (
	"fmt"
	"io"
	"math"
+	"sort"
	"strings"
	"sync"
	"time"
@@ -46,14 +47,17 @@ type progressState struct {
	configKind           sampler.SizeKind
	foldK, foldTotal     int
	foldKind             sampler.SizeKind
+	stepK                int
+	lastStepAt           time.Time
+	lastRunAt            time.Time
	outerScores          []float64 // nested: completed outer-fold held-out scores
	flatScores           []float64 // flat: the one config's completed fold scores
 }

 // progressLine renders the aggregated line. Nested:
-// `outer 1/3 · configs 84/216 · folds 421/1080 · est 0.8283 ± 0.0140`
+// `outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`
 // (est — until an outer fold lands; ± only at n≥2). Flat (since metis#32: iff 1
-// config): `folds 3/5 · score 0.8400` (the running fold mean — nothing to be
+// config): `CV runs 3/5 · score 0.8400` (the running fold mean — nothing to be
 // "best" of). Kinds render k/n (exact), k/≤n (budget), k/? (unknown). Pure.
 func progressLine(st progressState) string {
	return "metis: progress " + progressCore(st)
@@ -75,9 +79,9 @@ func progressCore(st progressState) string {
	}
	var parts []string
	if st.nested {
-		parts = append(parts, "outer "+frac(st.outerK, st.outerTotal, st.outerKind))
-		parts = append(parts, "configs "+frac(st.configK, st.configTotal, st.configKind))
-		parts = append(parts, "folds "+frac(st.foldK, st.foldTotal, st.foldKind))
+		parts = append(parts, "outer folds "+frac(st.outerK, st.outerTotal, st.outerKind))
+		parts = append(parts, "configs scored "+frac(st.configK, st.configTotal, st.configKind))
+		parts = append(parts, "inner-CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
		mean, se, n := meanSE(st.outerScores)
		switch {
		case n == 0:
@@ -88,7 +92,7 @@ func progressCore(st progressState) string {
			parts = append(parts, fmt.Sprintf("est %.4f ± %.4f", mean, se))
		}
	} else {
-		parts = append(parts, "folds "+frac(st.foldK, st.foldTotal, st.foldKind))
+		parts = append(parts, "CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
		if mean, _, n := meanSE(st.flatScores); n > 0 {
			parts = append(parts, fmt.Sprintf("score %.4f", mean))
		}
@@ -142,36 +146,38 @@ func seededTotals(ctx sampler.Ctx, nested bool, runFolds int, configPts []shape.
	}
 }

-// movingRate is metis#38's throughput window: a ring of the last 64 fold-completion
-// instants. rate(now) = n / (now − oldest) — `now` in the denominator means a STALL
-// decays the rate live (the k10-probe BLAS-thrash signature: throughput → 0 while the
-// process looks alive). Moving-average by construction (the operator's requirement:
-// per-leaf times vary by config, rf500 ≫ logreg). Pure over passed-in instants.
+// movingRate retains the latest eligible run completions by event time. rate(now)
+// = (n-1)/(now-oldest) after the confidence gate; `now` in the denominator means
+// a STALL decays live while last-run age remains the sharp freshness signal.
 type movingRate struct {
-	times [64]time.Time
-	n     int // total adds (ring index = n % len)
+	times []time.Time
 }

 func (m *movingRate) add(t time.Time) {
-	m.times[m.n%len(m.times)] = t
-	m.n++
+	i := sort.Search(len(m.times), func(i int) bool { return !m.times[i].Before(t) })
+	m.times = append(m.times, time.Time{})
+	copy(m.times[i+1:], m.times[i:])
+	m.times[i] = t
+	if len(m.times) > 64 {
+		m.times = m.times[1:]
+	}
 }

-// rate returns completions/minute over the kept window; ok=false until 2 completions.
+// rate returns eligible runs/minute over the kept event-time window.
 func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
-	if m.n < 2 {
+	if len(m.times) < 16 {
		return 0, false
	}
-	kept := m.n
-	if kept > len(m.times) {
-		kept = len(m.times)
+	oldest := m.times[0]
+	newest := m.times[len(m.times)-1]
+	if newest.Sub(oldest) < 15*time.Second {
+		return 0, false
	}
-	oldest := m.times[(m.n-kept)%len(m.times)]
	mins := now.Sub(oldest).Minutes()
	if mins <= 0 {
		return 0, false
	}
-	return float64(kept) / mins, true
+	return float64(len(m.times)-1) / mins, true
 }

 // eta = remaining / rate; ok=false when the rate is unavailable or zero.
@@ -183,6 +189,36 @@ func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
	return time.Duration(float64(remaining) / r * float64(time.Minute)), true
 }

+type occupancySample struct {
+	busy, capacity int
+}
+
+type occupancyWindow struct {
+	samples []occupancySample
+}
+
+func (w *occupancyWindow) add(busy, capacity int) {
+	if capacity <= 0 {
+		return
+	}
+	w.samples = append(w.samples, occupancySample{busy: busy, capacity: capacity})
+	if len(w.samples) > 4 {
+		w.samples = w.samples[1:]
+	}
+}
+
+func (w occupancyWindow) mean() (busy, capacity int, ok bool) {
+	if len(w.samples) == 0 {
+		return 0, 0, false
+	}
+	var sum int
+	for _, s := range w.samples {
+		sum += s.busy
+		capacity = s.capacity
+	}
+	return int(math.Round(float64(sum) / float64(len(w.samples)))), capacity, true
+}
+
 // passRow is one outer fold's live board row (metis#38): in-flight counters + the
 // pass's incumbent best (display-only — NOT the 1-SE select rule), collapsing to its
 // held-out score when the driver reports the fold done.
@@ -223,9 +259,10 @@ type sweepProgress struct {
	// metis#38 board mode (all nil/zero in plain mode): emits paint the rendered frame
	// to bw instead of printing plain lines. Lock order: sink.mu → bw.mu, ALWAYS — the
	// ticker enters via tick() (a sink method), never a boardWriter-first path.
-	bw    *boardWriter
-	width int               // terminal width ($COLUMNS | 80), read once at wiring
-	gauge func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
+	bw        *boardWriter
+	width     int               // terminal width ($COLUMNS | 80), read once at wiring
+	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
+	occupancy occupancyWindow
 }

 func newSweepProgress(out io.Writer, now func() time.Time, direction string, totals progressTotals) *sweepProgress {
@@ -294,8 +331,6 @@ func (sp *sweepProgress) forPass(outer int) passHooks {
		fold: func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
			sp.mu.Lock()
			defer sp.mu.Unlock()
-			sp.st.foldK++
-			sp.rate.add(sp.now()) // #38: throughput window feeds off every leaf completion
			if !sp.st.nested {
				sp.st.flatScores = append(sp.st.flatScores, ev.Out.Score)
			}
@@ -307,6 +342,41 @@ func (sp *sweepProgress) forPass(outer int) passHooks {
	}
 }

+func (sp *sweepProgress) activity(ev activityEvent) {
+	if sp == nil {
+		return
+	}
+	sp.mu.Lock()
+	defer sp.mu.Unlock()
+	switch ev.Kind {
+	case activityStepSuccess:
+		sp.st.stepK++
+		at := ev.At
+		if at.IsZero() {
+			at = sp.now()
+		}
+		if at.After(sp.st.lastStepAt) {
+			sp.st.lastStepAt = at
+		}
+	case activityRunSuccess:
+		if ev.Role != runRoleNestedInnerCV && ev.Role != runRoleFlatCV {
+			return
+		}
+		sp.st.foldK++
+		at := ev.At
+		if at.IsZero() {
+			at = sp.now()
+		}
+		if at.After(sp.st.lastRunAt) {
+			sp.st.lastRunAt = at
+		}
+		sp.rate.add(at)
+	default:
+		return
+	}
+	sp.maybeEmit(false)
+}
+
 // better orients a display-best comparison by the objective direction.
 func better(direction string, candidate, incumbent float64) bool {
	if direction == "minimize" {
@@ -349,6 +419,10 @@ func (sp *sweepProgress) tick() {
		return
	}
	sp.mu.Lock()
+	if sp.gauge != nil {
+		busy, capacity := sp.gauge()
+		sp.occupancy.add(busy, capacity)
+	}
	sp.emit() // stores the fresh frame (budget may skip the draw)
	bw := sp.bw
	sp.mu.Unlock()
@@ -388,10 +462,7 @@ func (sp *sweepProgress) maybeEmit(force bool) {
 // plain mode prints the #30 aggregated line. Caller holds sp.mu.
 func (sp *sweepProgress) emit() {
	if sp.bw != nil {
-		busy, capacity := 0, 0
-		if sp.gauge != nil {
-			busy, capacity = sp.gauge()
-		}
+		busy, capacity, _ := sp.occupancy.mean()
		sp.bw.paint(renderBoard(sp.snapshotLocked(),
			boardEnv{width: sp.width, now: sp.now(), busy: busy, capacity: capacity}))
		return
diff --git a/cmd/metis/progress_test.go b/cmd/metis/progress_test.go
index e1e1eaa..3c807d2 100644
--- a/cmd/metis/progress_test.go
+++ b/cmd/metis/progress_test.go
@@ -28,24 +28,24 @@ func TestProgressLine(t *testing.T) {
		not  []string
	}{
		{"nested pre-outer", nested(0, 84, 421, nil),
-			[]string{"outer 0/3", "configs 84/216", "folds 421/1080", "est —"}, []string{"±"}},
+			[]string{"outer folds 0/3", "configs scored 84/216", "inner-CV runs 421/1080", "est —"}, []string{"±"}},
		{"nested one outer", nested(1, 100, 500, []float64{0.82}),
-			[]string{"outer 1/3", "est 0.8200"}, []string{"±"}},
+			[]string{"outer folds 1/3", "est 0.8200"}, []string{"±"}},
		{"nested two outer", nested(2, 200, 900, []float64{0.80, 0.84}),
-			[]string{"outer 2/3", "est 0.8200 ± 0.0200"}, nil},
+			[]string{"outer folds 2/3", "est 0.8200 ± 0.0200"}, nil},
		{"flat one config", progressState{
			foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact,
			flatScores: []float64{0.80, 0.84, 0.88}},
-			[]string{"folds 3/5", "score 0.8400"}, []string{"configs", "outer"}},
+			[]string{"CV runs 3/5", "score 0.8400"}, []string{"configs", "outer", "folds 3/5"}},
		{"unknown kind", progressState{
			nested: true,
			outerK: 1, outerTotal: 0, outerKind: sampler.SizeUnknown,
			configK: 3, configTotal: 0, configKind: sampler.SizeUnknown},
-			[]string{"outer 1/?", "configs 3/?"}, nil},
+			[]string{"outer folds 1/?", "configs scored 3/?"}, nil},
		{"budget kind", progressState{
			nested: true,
			outerK: 1, outerTotal: 8, outerKind: sampler.SizeBudget},
-			[]string{"outer 1/≤8"}, nil},
+			[]string{"outer folds 1/≤8"}, nil},
	}
	for _, c := range cases {
		got := progressLine(c.st)
@@ -91,10 +91,8 @@ func TestSweepProgress_ThrottleAndAlwaysEmit(t *testing.T) {
	prog := newSweepProgress(&out, scriptedClock(times...), "maximize",
		progressTotals{nested: true, outer: 2, outerKind: sampler.SizeExact,
			configs: 4, configKind: sampler.SizeExact, folds: 20, foldKind: sampler.SizeExact})
-	hooks := prog.forPass(0)
	for i := 1; i <= 10; i++ {
-		hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{K: i, Total: 5, Kind: sampler.SizeExact,
-			Out: sampler.FoldOutcome{Score: 0.8}})
+		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 200)})
	}
	throttled := strings.Count(out.String(), "metis: progress")
	if throttled != 2 { // event 1 (first) + event 6 (throttle boundary)
@@ -110,7 +108,7 @@ func TestSweepProgress_ThrottleAndAlwaysEmit(t *testing.T) {
		t.Fatalf("finish must emit the final line, got %d lines", got)
	}
	final := out.String()[strings.LastIndex(out.String(), "metis: progress"):]
-	for _, w := range []string{"outer 1/2", "folds 10/20", "est 0.8300"} {
+	for _, w := range []string{"outer folds 1/2", "inner-CV runs 10/20", "est 0.8300"} {
		if !strings.Contains(final, w) {
			t.Errorf("final line missing %q: %q", w, final)
		}
@@ -139,18 +137,17 @@ func TestSweepProgress_ConcurrentCounts(t *testing.T) {
		progressTotals{nested: true, folds: 64, foldKind: sampler.SizeExact})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
-		hooks := prog.forPass(g)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 8; i++ {
-				hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{K: i + 1, Out: sampler.FoldOutcome{Score: 0.5}})
+				prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
			}
		}()
	}
	wg.Wait()
	prog.finish()
-	if !strings.Contains(out.String(), "folds 64/64") {
+	if !strings.Contains(out.String(), "inner-CV runs 64/64") {
		t.Errorf("sink-owned counter must see all 64 events (never ev.K):\n%s", out.String())
	}
 }
@@ -160,39 +157,99 @@ type writerFunc func([]byte) (int, error)

 func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

-// movingRate: rate = n/(now-oldest) over a ring of the last 64 completions — `now` in
-// the denominator makes a stall DECAY the rate live (the BLAS-thrash signature).
+// movingRate: keep the latest 64 eligible completion times by event time. It is
+// ready only after enough event-time evidence, and rate = (n-1)/(now-oldest).
 func TestMovingRate(t *testing.T) {
	var r movingRate
	if _, ok := r.rate(at(0)); ok {
		t.Error("n=0 must be not-ok")
	}
-	r.add(at(0))
-	if _, ok := r.rate(at(1000)); ok {
-		t.Error("n=1 must be not-ok (no interval yet)")
+	for i := 0; i < 15; i++ {
+		r.add(at(i * 1000))
+	}
+	if _, ok := r.rate(at(15000)); ok {
+		t.Error("15 completions must be below confidence")
+	}
+	var short movingRate
+	for i := 0; i < 16; i++ {
+		short.add(at(i * 900))
	}
-	for i := 1; i <= 4; i++ {
-		r.add(at(i * 1000)) // 5 completions total, 1s apart (t=0..4s)
+	if _, ok := short.rate(at(15000)); ok {
+		t.Error("16 completions spanning under 15s must be below confidence")
	}
-	if got, ok := r.rate(at(4000)); !ok || got < 74.9 || got > 75.1 {
-		t.Errorf("5 in 4s at now=4s → 75/min, got %v ok=%v", got, ok)
+	var ready movingRate
+	for i := 0; i < 16; i++ {
+		ready.add(at(i * 1000))
	}
-	// A 60s stall decays the same 5 completions: 5/64s ≈ 4.7/min.
-	if got, ok := r.rate(at(64000)); !ok || got > 5 {
-		t.Errorf("stall must decay the rate (now in denominator), got %v ok=%v", got, ok)
+	if got, ok := ready.rate(at(20000)); !ok || got < 44.9 || got > 45.1 {
+		t.Errorf("16+ completions spanning ≥15s at now=20s → 45/min, got %v ok=%v", got, ok)
	}
-	// Ring wraps at 64: 65 completions 1s apart keeps the newest 64.
+	if got, ok := ready.rate(at(25000)); !ok || got >= 45 {
+		t.Errorf("stall must decay the rate using now in the denominator, got %v ok=%v", got, ok)
+	}
+
+	// Reversed delivery: 65 completions 1s apart keeps the newest 64 by event time.
	var r2 movingRate
-	for i := 0; i < 65; i++ {
+	for i := 64; i >= 0; i-- {
		r2.add(at(i * 1000))
	}
-	// oldest kept = t=1s; at now=64s: 64/(63s) ≈ 60.95/min (not 65/64s ≈ 60.9... distinguish by n)
-	if got, _ := r2.rate(at(64000)); got < 60.5 || got > 61.5 {
-		t.Errorf("ring wrap: want ~60.95/min over the kept 64, got %v", got)
+	// newest kept window = t=1s..64s; at now=65s: 63 completions over 64s ≈ 59.06/min.
+	if got, ok := r2.rate(at(65000)); !ok || got < 58.9 || got > 59.2 {
+		t.Errorf("reversed delivery latest-64 window: want ~59.06/min, got %v ok=%v", got, ok)
	}
	// ETA: remaining/rate.
-	if eta, ok := r.eta(at(4000), 75); !ok || eta != time.Minute {
-		t.Errorf("eta 75 remaining at 75/min → 1m, got %v ok=%v", eta, ok)
+	if eta, ok := ready.eta(at(20000), 45); !ok || eta != time.Minute {
+		t.Errorf("eta 45 remaining at 45/min → 1m, got %v ok=%v", eta, ok)
+	}
+}
+
+func TestOccupancyWindowRoundedMeanOfLastFourSamples(t *testing.T) {
+	var w occupancyWindow
+	for _, busy := range []int{1, 2, 3, 4} {
+		w.add(busy, 12)
+	}
+	busy, capacity, ok := w.mean()
+	if !ok || busy != 3 || capacity != 12 {
+		t.Fatalf("[1,2,3,4]/12 mean = (%d,%d,%v); want (3,12,true)", busy, capacity, ok)
+	}
+	w.add(9, 12)
+	busy, capacity, ok = w.mean()
+	if !ok || busy != 5 || capacity != 12 {
+		t.Fatalf("[2,3,4,9]/12 rounded mean = (%d,%d,%v); want (5,12,true)", busy, capacity, ok)
+	}
+}
+
+func TestSweepProgressOccupancySamplesOnlyOnTick(t *testing.T) {
+	var out strings.Builder
+	busy := 0
+	prog := newSweepProgress(&out, func() time.Time { return at(20000) }, "maximize",
+		progressTotals{nested: true, folds: 32, foldKind: sampler.SizeExact})
+	prog.bw = newBoardWriter(&out, func() time.Time { return at(20000) })
+	prog.gauge = func() (int, int) { return busy, 12 }
+
+	for _, v := range []int{1, 2, 3, 4} {
+		busy = v
+		prog.tick()
+	}
+	got, cap, ok := prog.occupancy.mean()
+	if !ok || got != 3 || cap != 12 {
+		t.Fatalf("tick samples [1,2,3,4] mean = (%d,%d,%v); want (3,12,true)", got, cap, ok)
+	}
+
+	for i := 0; i < 10; i++ {
+		busy = 12
+		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
+	}
+	got, cap, ok = prog.occupancy.mean()
+	if !ok || got != 3 || cap != 12 {
+		t.Fatalf("activity burst changed occupancy mean to (%d,%d,%v); want unchanged (3,12,true)", got, cap, ok)
+	}
+
+	busy = 5
+	prog.tick()
+	got, cap, ok = prog.occupancy.mean()
+	if !ok || got != 4 || cap != 12 {
+		t.Fatalf("fifth tick samples [2,3,4,5] mean = (%d,%d,%v); want (4,12,true)", got, cap, ok)
	}
 }

@@ -247,8 +304,38 @@ func TestSweepProgress_PerPassRows(t *testing.T) {
	if st.rows[0].done {
		t.Errorf("row 0 must stay in-flight: %+v", st.rows[0])
	}
-	// The fold events fed the rate ring (3 adds at the frozen clock — n≥2 → ok).
-	if _, ok := st.rate.rate(at(1000)); !ok {
-		t.Error("fold completions must feed the rate ring")
+}
+
+func TestSweepProgressActivityRunEventsOwnAggregateRunCounterAndRate(t *testing.T) {
+	var out strings.Builder
+	prog := newSweepProgress(&out, func() time.Time { return at(20000) }, "maximize",
+		progressTotals{nested: true, outer: 1, outerKind: sampler.SizeExact,
+			configs: 2, configKind: sampler.SizeExact, folds: 16, foldKind: sampler.SizeExact})
+	hooks := prog.forPass(0)
+	hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{
+		Out: sampler.FoldOutcome{Score: 0.7},
+	})
+	st := prog.boardState()
+	if st.st.foldK != 0 {
+		t.Fatalf("sampler fold callback advanced aggregate run counter to %d; want typed events only", st.st.foldK)
+	}
+	if st.rows[0].foldK != 1 {
+		t.Fatalf("sampler fold callback should retain per-row duties; row = %+v", st.rows[0])
+	}
+
+	prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedPreamble, RunID: "pre", At: at(0)})
+	if got := prog.boardState().st.foldK; got != 0 {
+		t.Fatalf("ineligible preamble advanced aggregate run counter to %d", got)
+	}
+
+	for i := 15; i >= 0; i-- {
+		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, RunID: "inner", At: at(i * 1000)})
+	}
+	st = prog.boardState()
+	if st.st.foldK != 16 {
+		t.Fatalf("typed eligible run events advanced foldK to %d; want 16", st.st.foldK)
+	}
+	if got, ok := st.rate.rate(at(20000)); !ok || got < 44.9 || got > 45.1 {
+		t.Fatalf("typed event-time rate = %v ok=%v; want 45/min", got, ok)
	}
 }
diff --git a/cmd/metis/run.go b/cmd/metis/run.go
index c71ede6..71acdbb 100644
--- a/cmd/metis/run.go
+++ b/cmd/metis/run.go
@@ -79,12 +79,14 @@ type runOpts struct {
	exec experiment.StepExecutor // test seam: an injected fake replaces the subprocess
	//                              execStep (nil → the production execStep). Composes with
	//                              cache: the caching decorator still wraps it.
-	readRoot    string        // metis#23: when set, the production execStep confines base-dataset reads to this root
-	maxParallel int           // metis#31: >1 ⇒ ParExec batches + a leaf semaphore; sizes leafSem
-	leafSem     chan struct{} // metis#31: the shared global subprocess budget (nil = serial/cache-only)
-	runControl  *runControl   // one per shape run: global abort + optional 2n admission slots
-	runLabel    string        // config/fold/preamble context captured with the first error
-	forkserver  bool          // metis#44: warm fork-server leaf executor (cmdRun default true;
+	activity    activityEmitter // successful step/run facts; nil is a no-op
+	runRole     runRole         // role for successful concrete-run activity; zero = ineligible/non-sweep
+	readRoot    string          // metis#23: when set, the production execStep confines base-dataset reads to this root
+	maxParallel int             // metis#31: >1 ⇒ ParExec batches + a leaf semaphore; sizes leafSem
+	leafSem     chan struct{}   // metis#31: the shared global subprocess budget (nil = serial/cache-only)
+	runControl  *runControl     // one per shape run: global abort + optional 2n admission slots
+	runLabel    string          // config/fold/preamble context captured with the first error
+	forkserver  bool            // metis#44: warm fork-server leaf executor (cmdRun default true;
	//                           zero-value false keeps direct runOpts callers/tests on legacy exec)
	forkPool *serverPool // metis#44: the per-root warm-server pool, created once per runExperiment
	//                      when forkserver is set; threaded through nested runOpts copies.
@@ -216,6 +218,7 @@ func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, n
	if o.runControl == nil {
		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
	}
+	o.activity = runControlActivityEmitter(o.runControl, o.activity)
	return o.runControl.run(o.runLabel, func() (experiment.Run, error) {
		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
	})
@@ -248,6 +251,7 @@ func runResolvedExperimentAdmitted(exp experiment.Experiment, o runOpts, runID s
		store := cas.NewFSStore(filepath.Join(cacheDir, "cas"), 0, cas.Clock(now))
		exec = newCachingExecutor(exec, store, cacheDir, exp.Seed, out)
	}
+	exec = activityExecutor{inner: exec, now: now, emit: o.activity}
	runner := experiment.Runner{Exec: exec, Now: now}
	fmt.Fprintf(out, "metis: run %s of experiment %q\n", runID, exp.ID)
	run, steps, runErr := runner.Run(exp, runID, runDir)
@@ -297,6 +301,7 @@ func runResolvedExperimentAdmitted(exp experiment.Experiment, o runOpts, runID s
	if runErr != nil {
		return run, runErr
	}
+	o.activity.emit(activityEvent{Kind: activityRunSuccess, At: now(), RunID: runID, Role: o.runRole})
	fmt.Fprintf(out, "metis: %s %s\n", run.ID, run.Status)
	return run, nil
 }
diff --git a/cmd/metis/run_test.go b/cmd/metis/run_test.go
index 3c89ca8..1451027 100644
--- a/cmd/metis/run_test.go
+++ b/cmd/metis/run_test.go
@@ -179,6 +179,136 @@ func TestRunExperiment_FailedStepStillWritesLedger(t *testing.T) {
	}
 }

+func TestRunExperimentActivityRunSuccessPublishesAfterRequiredArtifacts(t *testing.T) {
+	root := repoRoot(t)
+	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
+	b, err := os.ReadFile(src)
+	if err != nil {
+		t.Fatal(err)
+	}
+	dir := t.TempDir()
+	expPath := filepath.Join(dir, "run-echo.md")
+	if err := os.WriteFile(expPath, b, 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	var runEvents []activityEvent
+	var publishErrs []error
+	_, err = runExperiment(runOpts{
+		expPath:  expPath,
+		runID:    "run-activity",
+		stepPath: []string{filepath.Join(root, "testdata", "steps")},
+		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
+		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
+		out:      io.Discard,
+		activity: func(ev activityEvent) {
+			if ev.Kind != activityRunSuccess {
+				return
+			}
+			runEvents = append(runEvents, ev)
+			for _, name := range []string{"run.json", "record.json"} {
+				if _, statErr := os.Stat(filepath.Join(dir, "runs", ev.RunID, name)); statErr != nil {
+					publishErrs = append(publishErrs, statErr)
+				}
+			}
+		},
+	})
+	if err != nil {
+		t.Fatalf("runExperiment: %v", err)
+	}
+	if len(publishErrs) > 0 {
+		t.Fatalf("run-success event arrived before required artifacts existed: %v", publishErrs)
+	}
+	if len(runEvents) != 1 {
+		t.Fatalf("run-success events = %d; want 1 (%+v)", len(runEvents), runEvents)
+	}
+	if runEvents[0].RunID != "run-activity" {
+		t.Fatalf("run event = %+v; want runID run-activity", runEvents[0])
+	}
+	if runEvents[0].Role != runRoleNone {
+		t.Fatalf("plain run event role = %q; want no role/ineligible", runEvents[0].Role)
+	}
+}
+
+func TestRunExperimentActivityRunSuccessNotPublishedForFailedExecution(t *testing.T) {
+	root := repoRoot(t)
+	src := filepath.Join(root, "testdata", "experiment", "run-fail.md")
+	b, err := os.ReadFile(src)
+	if err != nil {
+		t.Fatal(err)
+	}
+	dir := t.TempDir()
+	expPath := filepath.Join(dir, "run-fail.md")
+	if err := os.WriteFile(expPath, b, 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	var runEvents []activityEvent
+	_, err = runExperiment(runOpts{
+		expPath:  expPath,
+		runID:    "run-failed-activity",
+		stepPath: []string{filepath.Join(root, "testdata", "steps")},
+		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
+		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
+		out:      io.Discard,
+		activity: func(ev activityEvent) {
+			if ev.Kind == activityRunSuccess {
+				runEvents = append(runEvents, ev)
+			}
+		},
+	})
+	if err == nil {
+		t.Fatal("runExperiment: want failing step error")
+	}
+	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "run.json")); statErr != nil {
+		t.Fatalf("failed run should still write run.json: %v", statErr)
+	}
+	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "record.json")); statErr != nil {
+		t.Fatalf("failed run should still write record.json: %v", statErr)
+	}
+	if len(runEvents) != 0 {
+		t.Fatalf("failed execution emitted run-success events: %+v", runEvents)
+	}
+}
+
+func TestRunExperimentActivityRunSuccessNotPublishedWhenRecordPersistenceFails(t *testing.T) {
+	root := repoRoot(t)
+	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
+	b, err := os.ReadFile(src)
+	if err != nil {
+		t.Fatal(err)
+	}
+	dir := t.TempDir()
+	expPath := filepath.Join(dir, "run-echo.md")
+	if err := os.WriteFile(expPath, b, 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.MkdirAll(filepath.Join(dir, "runs", "run-record-fails", "record.json"), 0o755); err != nil {
+		t.Fatal(err)
+	}
+
+	var runEvents []activityEvent
+	_, err = runExperiment(runOpts{
+		expPath:  expPath,
+		runID:    "run-record-fails",
+		stepPath: []string{filepath.Join(root, "testdata", "steps")},
+		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
+		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
+		out:      io.Discard,
+		activity: func(ev activityEvent) {
+			if ev.Kind == activityRunSuccess {
+				runEvents = append(runEvents, ev)
+			}
+		},
+	})
+	if err == nil {
+		t.Fatal("runExperiment: want record persistence error")
+	}
+	if len(runEvents) != 0 {
+		t.Fatalf("record persistence failure emitted run-success events: %+v", runEvents)
+	}
+}
+
 // TestRunExperiment_RejectsInvalidAtRunTime is the execution-time enforcement
 // test: a semantically-invalid experiment (a cycle — shape-valid, so CUE accepts
 // it) is rejected by `metis run` BEFORE any step runs, closing the SHAPE-only gap
diff --git a/cmd/metis/runcontrol_test.go b/cmd/metis/runcontrol_test.go
index 04c9e8a..ad64ac2 100644
--- a/cmd/metis/runcontrol_test.go
+++ b/cmd/metis/runcontrol_test.go
@@ -506,3 +506,23 @@ func TestRunControlObservationLinearizesBeforeFailure(t *testing.T) {
		t.Fatal("rejected observation callback ran")
	}
 }
+
+func TestRunControlActivityEmitterDropsLateStepAndRunEventsAfterFailure(t *testing.T) {
+	control := newRunControl(2)
+	var events []activityEvent
+	emit := runControlActivityEmitter(control, func(ev activityEvent) {
+		events = append(events, ev)
+	})
+
+	emit(activityEvent{Kind: activityStepSuccess, StepID: "prep"})
+	if len(events) != 1 {
+		t.Fatalf("healthy activity events = %d; want 1", len(events))
+	}
+
+	control.fail("first", errors.New("boom"))
+	emit(activityEvent{Kind: activityStepSuccess, StepID: "late-step"})
+	emit(activityEvent{Kind: activityRunSuccess, RunID: "late-run"})
+	if len(events) != 1 {
+		t.Fatalf("late activity after failure was published: %+v", events)
+	}
+}
diff --git a/cmd/metis/shapesweep_test.go b/cmd/metis/shapesweep_test.go
index 471378e..fcbe404 100644
--- a/cmd/metis/shapesweep_test.go
+++ b/cmd/metis/shapesweep_test.go
@@ -329,7 +329,7 @@ func TestShapeSweep_OneConfigDegeneratesToSingleLevelCV(t *testing.T) {
		}
		final := s[strings.LastIndex(s, "metis: progress"):]
		final = final[:strings.IndexByte(final, '\n')]
-		if !strings.Contains(final, "folds 2/2") || !strings.Contains(final, "score 0.") {
+		if !strings.Contains(final, "CV runs 2/2") || !strings.Contains(final, "score 0.") {
			t.Errorf("the flat final progress line must carry folds k/k + score; got: %q", final)
		}
		// metis#50: the flat path ends with the same summary block.
@@ -352,6 +352,59 @@ func TestShapeSweep_OneConfigDegeneratesToSingleLevelCV(t *testing.T) {
	}
 }

+func TestShapeSweepActivityRunRolesFromFlatAndNestedCallPaths(t *testing.T) {
+	t.Run("flat CV runs are eligible flat roles", func(t *testing.T) {
+		ws := t.TempDir()
+		expPath := writeShapeFile(t, ws, foldShapeShipMD("[a]"))
+		counts := map[runRole]int{}
+		_, err := runExperiment(runOpts{
+			expPath: expPath, now: fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
+			exec: foldFakeExec{}, out: io.Discard,
+			activity: func(ev activityEvent) {
+				if ev.Kind == activityRunSuccess {
+					counts[ev.Role]++
+				}
+			},
+		})
+		if err != nil {
+			t.Fatalf("flat sweep: %v", err)
+		}
+		if counts[runRoleFlatCV] != 2 || len(counts) != 1 {
+			t.Fatalf("flat roles = %v; want exactly 2 flat-CV runs", counts)
+		}
+	})
+
+	t.Run("nested emits preamble inner and outer score roles", func(t *testing.T) {
+		ws := t.TempDir()
+		expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
+		counts := map[runRole]int{}
+		_, err := runExperiment(runOpts{
+			expPath: expPath, now: fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
+			exec: foldFakeExec{}, out: io.Discard,
+			activity: func(ev activityEvent) {
+				if ev.Kind == activityRunSuccess {
+					counts[ev.Role]++
+				}
+			},
+		})
+		if err != nil {
+			t.Fatalf("nested sweep: %v", err)
+		}
+		if counts[runRoleNestedPreamble] != 1 {
+			t.Fatalf("nested roles = %v; want 1 preamble run", counts)
+		}
+		if counts[runRoleNestedInnerCV] != 8 {
+			t.Fatalf("nested roles = %v; want 8 inner-CV runs", counts)
+		}
+		if counts[runRoleOuterScore] != 2 {
+			t.Fatalf("nested roles = %v; want 2 outer-score runs", counts)
+		}
+		if len(counts) != 3 {
+			t.Fatalf("nested roles = %v; want no ineligible/unexpected run roles", counts)
+		}
+	})
+}
+
 // Fold-distinctness + cache under NESTED (metis#32): each (outer-fold, config, inner-fold) of
 // `train` gets a DISTINCT cache entry (the _fold overlay makes Kpre fold-distinct), the
 // config/fold-invariant data steps HIT, and a warm re-run HITs everything (0 inner execs).
diff --git a/cmd/metis/sweep.go b/cmd/metis/sweep.go
index 12a8504..61b2027 100644
--- a/cmd/metis/sweep.go
+++ b/cmd/metis/sweep.go
@@ -125,6 +125,7 @@ type sweepPass struct {
	splitK   int                  // the cv-split / FixedKFolds fold count for this pass
	stratify bool                 // the cv-split stratify flag for this pass
	partRef  sampler.PartitionRef // this pass's partition identity (fed into each point's address)
+	runRole  runRole              // concrete-run role for every pipeline fold in this pass
	hooks    passHooks            // metis#30: this pass's progress hooks, closure-bound to its outer fold
	// metis#31: under ParExec the sweeper fans out over configs and each config's
	// resample fans out over folds — all appending to this ONE pass. `mu` guards the
@@ -264,6 +265,7 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
	// levels report (stream-learned totals would arrive only with each level's first
	// completion — for the driver level that's the first COMPLETED outer fold, too late).
	ss.prog = newSweepProgress(out, now, sh.Sweeper.Objective.Direction, seededTotals(ctx, nested, runFolds, configPts, k))
+	ss.o.activity = teeActivityEmitter(ss.o.activity, ss.prog.activity)
	// metis#38: board mode — the sink paints the pinned board instead of plain lines,
	// and a 500ms ticker keeps the rate decay + ETA live between events (sink-first:
	// tick() locks sp.mu then hands the frame to bw — the one global lock order).
@@ -319,7 +321,7 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
	// deleted shape `driver:`) runs the sweeper once on all data → the sweeper's inner k-fold CV
	// scores the one config → (mean, SE, complexity) recorded to the ledger. metis#32: it MEASURES
	// ONLY — no `shipWinner` (shipping is `metis select --promote`).
-	pass := &sweepPass{ss: ss, splitK: k, stratify: stratify, partRef: ss.partRef,
+	pass := &sweepPass{ss: ss, splitK: k, stratify: stratify, partRef: ss.partRef, runRole: runRoleFlatCV,
		hooks: ss.prog.forPass(-1)} // metis#30: the flat path's single pass
	res := sampler.Run(ctx, sampler.SingleDriver{}, func(sampler.SinglePoint) sampler.SweepResult {
		return ss.runSweeper(ctx, configPts, pass)
@@ -485,6 +487,7 @@ func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]str
	preOpts.inSweep = true // one preamble run; skip the per-run capture noise
	preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
	preOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
+	preOpts.runRole = runRoleNestedPreamble
	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
		return nil, err
	}
@@ -507,7 +510,8 @@ func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k i
	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
		stratify: stratify, partRef: ss.partRef,
-		hooks: ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
+		runRole: runRoleNestedInnerCV,
+		hooks:   ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
	sres := ss.runSweeper(ctx, configPts, pass)
	if err := pass.firstError(); err != nil {
		return 0, err
@@ -585,6 +589,7 @@ func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify boo
	scoreOpts.inSweep = true
	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
	scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
+	scoreOpts.runRole = runRoleOuterScore
	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
	if err != nil {
		return 0, "", "", err
@@ -645,6 +650,7 @@ func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.
	pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
	pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
+	pointOpts.runRole = p.runRole
	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
diff --git a/workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md b/workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md
new file mode 100644
index 0000000..f4818ca
--- /dev/null
+++ b/workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md
@@ -0,0 +1,5866 @@
+# Boundary Review — 000049-board-readability-labels-coldphase-eta#49 (whole-issue close)
+
+| field | value |
+|-------|-------|
+| issue | 49 — board readability — label semantics, cold-phase "no progress" confusion, jumpy leaves, wild early ETA |
+| repo | 000049-board-readability-labels-coldphase-eta |
+| issue file | workshop/issues/000049-board-readability-labels-coldphase-eta.md |
+| boundary | whole-issue close |
+| milestone | — |
+| window | 72f2d895fc676a474304fb4cde8f4606e9c5f48c^..HEAD |
+| command | sdlc close --issue 49 |
+| reviewer | codex |
+| timestamp | 2026-07-17T10:12:11-07:00 |
+| verdict | REWORK |
+
+## Review
+
+Reading additional input from stdin...
+OpenAI Codex v0.144.5
+--------
+workdir: /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+model: gpt-5.5
+provider: openai
+approval: never
+sandbox: read-only
+reasoning effort: medium
+reasoning summaries: none
+session id: 019f710f-0279-72b0-9b3a-c0a63df5b6dc
+--------
+user
+# Code review — the one SDLC boundary review
+
+You are conducting a fresh-context code review at a development boundary —
+whole-issue close — in the **000049-board-readability-labels-coldphase-eta** repository.
+
+- repository: 000049-board-readability-labels-coldphase-eta   (root: /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta)
+- issue:      000049-board-readability-labels-coldphase-eta#49   (file: workshop/issues/000049-board-readability-labels-coldphase-eta.md)
+- window:     Base: 72f2d895fc676a474304fb4cde8f4606e9c5f48c^   Head: HEAD
+
+Review the **000049-board-readability-labels-coldphase-eta** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
+other repository or apply another repo's conventions.
+
+You have no prior session context — that is the anti-collusion property. Verify
+behavior against the issue's documented Spec/Plan and the code itself; do NOT
+take the implementor's word in commit messages or docs at face value. Tools are
+read-only: report findings precisely; the main agent (which has session context)
+applies the fixes, commits, and re-runs.
+
+Read the diff against the issue's Spec + Plan, then work the checklist below.
+Categorize every finding by severity — not everything is Critical; a nitpick
+marked Critical is noise.
+
+  Critical (must fix before crossing the boundary)
+    - correctness bugs; crashes / panics on unexpected input
+    - behavior drift from stated contracts (for ports of existing code where
+      byte-faithfulness was promised, diff against the source)
+    - silent error swallowing where the source raised
+  Important (fix before the boundary if cheap)
+    - API design of newly-introduced internal packages (downstream work will
+      consume them; is the surface stable?)
+    - missing test coverage that would catch the kind of bug shipped
+    - inconsistent error handling across the diff
+  Minor (note for future)
+    - style nits, naming, comment density; performance only if hot-path
+
+## Review checklist
+
+Code quality
+  - Clean separation of concerns; edge cases handled (empty / nil / unexpected).
+  - Proper error handling — no silent swallowing where the source raised.
+  - No duplicated logic / copy-paste that should be a shared helper.
+
+Testing
+  - Tests pin real logic, not mocks reasserting the implementation.
+  - The kind of bug this diff could ship is covered.
+  - PURE entities tested without IO; INTEGRATION via injected fakes (see below).
+
+Requirements traceability
+  - Every Plan checklist item this boundary claims is actually delivered.
+  - Implementation matches the Spec; no undeclared scope creep.
+  - Breaking changes documented.
+
+Production readiness
+  - Migration / backward-compatibility considered where state or formats change.
+  - Docs / atlas updated for new surface (see the Docs update gate).
+
+## Core concepts cross-check (if the plan has a Core concepts table)
+
+The plan should list entities in a greppable table — name, kind
+(PURE/INTEGRATION), file location, status (new/modified/deleted). For each row:
+  - Verify the entity exists at the stated path (grep the diff or filesystem).
+  - PURE: tests run without IO (no exec, net, mutable fs). If tests need mocks
+    to run, it isn't really PURE — flag Critical and recommend promoting it to
+    INTEGRATION.
+  - INTEGRATION: injected into pure callers, not invoked directly from business
+    logic.
+  - "modified" / "deleted": the diff shows the expected change/removal at the
+    stated location.
+Any contradiction between table and code = Critical finding, plus a plan-revision
+recommendation (a "## Revisions" entry so the plan stops claiming what the code
+doesn't deliver).
+
+## Docs update gate (atlas + README, per AGENTS.md §8)
+
+The boundary should update user-facing docs for any new surface introduced:
+
+  - **atlas/** — new architectural surface, flow, or terminology. Scan the diff
+    for new entity types, subcommands, conventions, file-tree locations. Any
+    present without corresponding atlas/ changes in the same range = Important
+    finding ("atlas update appears missing for <surface>").
+  - **README.md** — new user-facing surface a reader runs or types: subcommands,
+    flags, keybindings, config keys, install/usage steps. If the diff adds or
+    changes such surface and README.md is not updated in the same range =
+    Important finding ("README update appears missing for <surface>"). This is the
+    class of gap that used to surface only at the merge-time `specs` judge (#142);
+    catch it here, at the earliest gate, before the close verdict is recorded.
+
+## Architecture (the at-review backstop — these matter most long-term)
+
+Work through each of ARCH-DRY, ARCH-PURE, ARCH-PURPOSE explicitly, applying its at-review lens. The
+full principle definitions are delivered in the ARCHITECTURE PRINCIPLES block
+right after this prompt — for EACH marker, state pass or flag, and cite the
+marker (e.g. ARCH-DRY) in any finding. Architecture is where review has the
+least training signal and the longest-delayed payoff, so be deliberate here, not
+holistic.
+
+## Verdict + output
+
+Begin your response with this fenced verdict block — the machine-read handoff:
+
+```verdict
+verdict: <SHIP | FIX-THEN-SHIP | REWORK>
+confidence: <high | medium | low>
+```
+
+  SHIP           ready; ship it
+  FIX-THEN-SHIP  ship after addressing the findings (non-blocking at the gate)
+  REWORK         blocking; needs rework before shipping — fix + re-run
+
+The fenced ```` ```verdict ```` block above is the **authoritative machine-read
+handoff** — emit it as the first thing in your response. (A prose
+`VERDICT: <TOKEN>` first line still satisfies the legacy contract as a fallback,
+but the block is what the binary trusts.)
+
+After the verdict block: a 1-paragraph summary — what worked, what blocks SHIP if
+it isn't — followed by:
+  1. Strengths: 2-5 specific things done well (file:line where useful). Affirm
+     validated approaches so the operator knows what's confirmed-good ground.
+     Empty acceptable for trivial boundaries.
+  2. Critical findings (file:line + fix sketch); empty if none.
+  3. Important findings (same format).
+  4. Minor findings (terse one-liners).
+  5. Test coverage notes.
+  6. Architectural notes for upcoming work.
+  7. Plan revision recommendations: specific "## Revisions" entries the plan
+     needs (empty if the plan still matches the code).
+
+
+ARCHITECTURE PRINCIPLES — work through each of the 3 entries below explicitly, applying its `at-review` lens; cite the marker (e.g. ARCH-DRY) in any finding.
+
+# Architecture principles (ARCH-*)
+
+Injected architectural taste — the structural decisions whose payoff (or cost)
+shows up many turns, often months, down the road. Agents are strong at local
+tactics and weak here, so these are checked **at-plan** (when the design is being
+made — highest leverage) and **at-review** (backstop, on the diff). Cite the
+marker (e.g. `ARCH-DRY`) in plans, `## Log` entries, and review findings.
+
+This file is the single source; it is embedded into the planning, plan-quality,
+and code-review prompts. The human narrative lives in AGENTS.md "Core Design
+Principles"; this is its machine-delivered companion.
+
+## ARCH-DRY — Don't Repeat Yourself
+
+- **principle:** Reuse before adding. One source of truth per fact/behavior; no
+  duplicated logic, copy-pasted blocks, or parallel functions that should be one
+  shared helper.
+- **at-plan:** Flag a plan that re-implements something the codebase already has,
+  or that will obviously duplicate logic across the new files instead of
+  extracting a shared helper. Name the existing thing it should reuse.
+- **at-review:** Flag duplicated logic / copy-pasted blocks / near-identical
+  functions in the diff; point at the consolidation (file:line + the shared
+  helper they should become).
+
+## ARCH-PURE — Pure core, thin IO shell
+
+- **principle:** The majority of code is pure functions (deterministic, no side
+  effects); a thin "glue" layer at the boundary touches IO/UI/network/clock. Pure
+  functions are unit-tested directly; the glue is kept small and injected.
+- **at-plan:** Flag a design that buries business logic inside IO/handlers, or
+  that will only be testable with heavy mocks (a sign logic isn't separated from
+  IO). The plan should name what's pure vs the thin IO seam.
+- **at-review:** Flag business logic mixed with IO in the diff; logic that should
+  be a pure function injected into a thin caller. If a test needs mocks to run a
+  "pure" entity, it isn't pure — recommend extracting the IO to the boundary.
+
+## ARCH-PURPOSE — Serve the issue's actual purpose
+
+- **principle:** Deliver the issue's stated purpose, not the easy subset of it. A
+  single-source / "compiled to consumers" change is not done until **every
+  consumer derives** from the source — the source is *enforced*, not just
+  documentation a surface happens to restate; a hand-maintained restatement of the
+  model is a deferred consumer, not a finished one. "Follow-up" is for separable
+  extensions, never for the thing that is the point. This is the *opposite axis*
+  from Simplicity-First/YAGNI: not "build for an imagined future," but "don't
+  **under**-deliver the purpose you already committed to."
+- **at-plan:** Flag a plan whose scope is a strict subset of the issue's stated
+  goal / Done-when where the part deferred as "follow-up" *is* the purpose (e.g.
+  wires one consumer + enforcement but leaves the consumers that motivated the
+  issue as documentation that doesn't derive). Ask: does the plan fulfill the
+  purpose, or just the cheap win? Name the deferred purpose.
+- **at-review:** Does the diff *fulfill* the purpose or settle for the easy win?
+  For a single-source change, run the **shadow-sweep** — enumerate the consumers,
+  confirm each derives from the source, flag any remaining hand-maintained
+  restatement of the model. A "follow-up" that is actually the deferred point of
+  the issue is a finding, not a deferral.
+
+
+OUTPUT CONTRACT (machine-read — do not deviate). LEAD your response with the
+fenced ```verdict block shown above — that is the authoritative handoff the binary
+reads (its `verdict:` value is one of the listed tokens). Everything after the block
+is advisory: a non-blocking verdict WITH findings still PASSES the gate. A bare
+`VERDICT: <TOKEN>` line is accepted only as a FALLBACK when the block is absent.
+
+Diff:
+diff --git a/atlas/experiment.md b/atlas/experiment.md
+index 7eb9adb..5f047e0 100644
+--- a/atlas/experiment.md
++++ b/atlas/experiment.md
+@@ -166,7 +166,14 @@ wrapped by **thin step-executables** honoring the contract above. Hermetic via *
+   `Aggregate` → **mean±SE**, the honest procedure estimate (`reportEstimate`). **metis#32:** the run now
+   **records** per-`(outer-fold, config)` inner rows + per-`(outer-fold, family)` outer rows to the ledger
+   (`Level`-keyed) — the signal `metis select` reduces to pick the family. `metis run` **measures only,
+-  never ships** (shipping moved to `metis select --promote`). Honesty of the score-over-full-data
++  never ships** (shipping moved to `metis select --promote`). **metis#52:** `select --cohort` lists the
++  fingerprint cohorts (delegates to the #39 core), and every pick line carries its
++  `· point <addr>` override handle (a representative ledger-row addr; round-trips through
++  `--point`). **metis#50:** a sweep ends with the
++  run-end summary — elapsed wall-clock, rows→ledger, the cohort fingerprint, and the paste-ready
++  `metis select … --fingerprint <fp>` follow-ups (completing #39's visibility loop: the operator
++  never scrapes scrollback to assemble the next command; degraded capture degrades to `cohort ?`
++  with un-pinned hints). Honesty of the score-over-full-data
+   refit holds while features are stateless; stateful features (metis#20) inherit fold-safety via the
+   fold-expressed score run.
+ - **Honest family selection (metis#32) — three commands, `run` measures / `select` chooses / `kaggle
+@@ -196,9 +203,17 @@ wrapped by **thin step-executables** honoring the contract above. Hermetic via *
+   (non-nil sem + a `syncWriter` over `out`) in one home. Determinism of persisted artifacts: the fan-out's
+   completion-order `pass.points` are `sortPointRuns`-sorted before the manifest/ledger write; the
+   `sweepPass` mutex guards the shared `configs`/`points`/`err` bookkeeping (the honest reduce stays pure
+-  in the sampler). Caveats (flag help): each leaf is a Python process that may itself multi-thread
+-  (BLAS/`n_jobs`) so `n=NumCPU` can oversubscribe; a COLD cache thundering-herds the shared upstream;
+-  clean per-`k/n` progress is deferred to metis#30.
++  in the sampler). Caveats (flag help): a COLD cache thundering-herds the shared upstream; clean
++  per-`k/n` progress is deferred to metis#30.
++- **Default leaf BLAS pins (metis#48) — `cmd/metis/blaspins.go`:** the parallelism budget belongs
++  to the ORCHESTRATOR (the #31 semaphore), so `runExperiment` computes the four single-thread pins
++  (`OMP/OPENBLAS/VECLIB/MKL_NUM_THREADS=1`) ONCE per top-level run — minus any name the operator
++  exported (an explicit value always wins) — announces one loud note, and injects them at BOTH
++  spawn seams: the legacy `execStep` child env and the fork-server process env (children inherit).
++  `metis select --promote` is deliberately unpinned (serial single all-data fit — multi-threaded
++  BLAS wanted). Env is outside run identity by design (`Kpre` = {step_id, uses, with, seed,
++  upstream}; HIT-validation re-hashes read-set D; fingerprint is git state), so pins perturb no
++  cache key or fingerprint.
+ - **Warm fork-server leaf executor (metis#44) — `metis/forkserver.py` + `cmd/metis/forkexec.go`:**
+   kills the per-leaf `uv run → fresh python → import pandas/sklearn` tax (~1s measured/spawn, ~5k
+   spawns/sweep). One warm server per **project root** (metis's and kbench's venvs differ), started
+diff --git a/atlas/index.md b/atlas/index.md
+index a48ef7e..18d8b0a 100644
+--- a/atlas/index.md
++++ b/atlas/index.md
+@@ -44,7 +44,7 @@ identical on a non-Kaggle platform?* — if yes, it lives here.
+   command (record IO on inspect/error paths only); each `metis run` prints
+   `recording under code_fingerprint <hash> (commit <sha>, clean|dirty)` at capture time.
+   **metis#32 retired `metis ledger select` + `metis promote`** — selection
+-  moved to **`metis select <shape> [--best | --best-per-model-class] [--promote]`** (see the run/select
++  moved to **`metis select <shape> [--best | --best-per-model-class | --point ADDR | --cohort] [--promote]`** (see the run/select
+   command model below): it reads the nested-CV ledger, picks the FAMILY on the honest OUTER estimate + the
+   CONFIG on the inner CV, and `--promote` reconstructs the winner (the pure `promotedExperiment` —
+   re-expands the shape + matches by free-params, reusing `shapeConfigToExperiment`: `data ++
+@@ -139,7 +139,13 @@ identical on a non-Kaggle platform?* — if yes, it lives here.
+   rebuild the exact run DIRECTLY, not by re-expanding the grid. The **driver** is `cmd/metis`: `metis run` on an experiment-shape
+   drives the real three-level loop (`runShapeSweep`: `Run(SingleDriver) ⊃ Run(GridConfigs) ⊃
+   Run(FixedKFolds)`), running each `(config, fold)` through the shared `runResolvedExperiment` (cached
+-  runner) keyed by its content-address. The sweeper (`GridConfigs ⊃ FixedKFolds`) is extracted as
++  runner) keyed by its content-address. Parallel sampler fan-out remains order-preserving, while every
++  concrete run crosses one sweep-scoped `2n` admission controller before side effects; the independent
++  leaf semaphore remains capped at `n`. The controller (`cmd/metis/runcontrol.go`, installed at
++  `runResolvedExperiment`) also owns the experiment-wide first failure, so queued runs produce no
++  observable state after cancellation; the scheduling and cancellation regressions in
++  `cmd/metis/parallel_test.go` pin both budgets and the failure boundary. The sweeper
++  (`GridConfigs ⊃ FixedKFolds`) is extracted as
+   `runSweeper`/`sweepPass` (per-call accumulators) so `driver:cv` can run it once **per outer fold**,
+   each pass repointed at that fold's sealed `analysis_i` + confined (`METIS_READ_ROOT`) — the flat
+   `driver:single` path is `runSweeper` with `baseRef=nil`, unconfined. Each fold builds a per-fold experiment (`data ++ engine-synthesized
+diff --git a/cmd/metis/blaspins.go b/cmd/metis/blaspins.go
+new file mode 100644
+index 0000000..7f440e7
+--- /dev/null
++++ b/cmd/metis/blaspins.go
+@@ -0,0 +1,45 @@
++package main
++
++import "strings"
++
++// blasPinDefaults are the single-thread pins metis injects into LEAF subprocesses by
++// default (metis#48): the parallelism budget belongs to the ORCHESTRATOR (the metis#31
++// leaf semaphore), not to each leaf's BLAS — NumCPU leaves × multi-threaded BLAS
++// oversubscribes ~NumCPU× (observed: load-avg 83 on 12 cores, throughput ≈ 0).
++//
++// Cache identity: env is deliberately OUTSIDE run identity — Kpre hashes
++// {step_id, uses, with, seed, upstream} (pkg/cache), HIT-validation re-hashes the
++// read-set D (file blob hashes), and the code fingerprint is git state. Injecting
++// pins perturbs neither cache keys nor fingerprints — exactly as the RUNBOOK's
++// manual `OMP_NUM_THREADS=1 metis run` never did.
++var blasPinDefaults = []string{
++	"MKL_NUM_THREADS=1",
++	"OMP_NUM_THREADS=1",
++	"OPENBLAS_NUM_THREADS=1",
++	"VECLIB_MAXIMUM_THREADS=1",
++}
++
++// blasPins returns the defaults NOT already set in environ — an explicit operator
++// value always wins (escape hatch by construction: `export OMP_NUM_THREADS=8`
++// passes through untouched). Pure. Always non-nil: an all-suppressed result is
++// empty, distinguishable from runOpts' nil "not yet computed" sentinel.
++func blasPins(environ []string) []string {
++	pins := make([]string, 0, len(blasPinDefaults))
++	for _, def := range blasPinDefaults {
++		name := def[:strings.IndexByte(def, '=')]
++		if !envHasName(environ, name) {
++			pins = append(pins, def)
++		}
++	}
++	return pins
++}
++
++// envHasName reports whether environ sets exactly `name` (match up to '=').
++func envHasName(environ []string, name string) bool {
++	for _, kv := range environ {
++		if strings.HasPrefix(kv, name) && len(kv) > len(name) && kv[len(name)] == '=' {
++			return true
++		}
++	}
++	return false
++}
+diff --git a/cmd/metis/blaspins_e2e_test.go b/cmd/metis/blaspins_e2e_test.go
+new file mode 100644
+index 0000000..d9652f9
+--- /dev/null
++++ b/cmd/metis/blaspins_e2e_test.go
+@@ -0,0 +1,74 @@
++package main
++
++import (
++	"os"
++	"path/filepath"
++	"strings"
++	"testing"
++)
++
++// TestRun_BlasPinsDefaultAndNote (metis#48 e2e): a bare run (no ambient pins) announces
++// ONE loud note and the leaf subprocess sees the pins; an operator-exported value passes
++// through untouched and drops out of the note. Drives runExperiment (the real wiring:
++// blasPins → runOpts.leafPins → execStep), real shell step, no uv.
++//
++// Sweep-path once-ness is by construction (verified at plan review): runExperiment is
++// entered exactly once per invocation (sole caller main.go) and all nested sweep spawns
++// are struct copies carrying the computed leafPins — so this single-run count==1 plus
++// that structure covers the sweep path.
++func TestRun_BlasPinsDefaultAndNote(t *testing.T) {
++	// ambient: exactly ONE operator choice set; the other three genuinely absent.
++	// t.Setenv registers the restore; Unsetenv then makes absence real (an operator
++	// shell following the old RUNBOOK exports all four — CI must not inherit that).
++	for _, k := range []string{"OPENBLAS_NUM_THREADS", "VECLIB_MAXIMUM_THREADS", "MKL_NUM_THREADS"} {
++		t.Setenv(k, "sentinel")
++		os.Unsetenv(k)
++	}
++	t.Setenv("OMP_NUM_THREADS", "7")
++
++	root := t.TempDir()
++	stepDir := filepath.Join(root, "steps", "test")
++	if err := os.MkdirAll(stepDir, 0o755); err != nil {
++		t.Fatal(err)
++	}
++	script := "#!/bin/sh\nenv > env.txt\necho '{\"ok\": 1}' > metrics.json\n"
++	if err := os.WriteFile(filepath.Join(stepDir, "envstep"), []byte(script), 0o755); err != nil {
++		t.Fatal(err)
++	}
++	expPath := filepath.Join(root, "exp.md")
++	// plain-experiment fixture — steps live in the YAML FRONTMATTER (the
++	// testdata/experiment/run-echo.md convention)
++	exp := "---\ntype: experiment\nid: pins-e2e\nseed: 1\nstatus: active\nsteps:\n  - id: e\n    uses: test/envstep\n---\n# pins e2e\n"
++	if err := os.WriteFile(expPath, []byte(exp), 0o644); err != nil {
++		t.Fatal(err)
++	}
++
++	var out strings.Builder
++	if _, err := runExperiment(runOpts{
++		expPath:  expPath,
++		runID:    "r1",
++		stepPath: []string{filepath.Join(root, "steps")},
++		out:      &out,
++	}); err != nil {
++		t.Fatalf("run: %v\n%s", err, out.String())
++	}
++
++	// (a) ONE note, naming only the three injected pins (OMP dropped — operator won)
++	if n := strings.Count(out.String(), "metis: leaf BLAS pinned"); n != 1 {
++		t.Fatalf("want exactly 1 pin note, got %d:\n%s", n, out.String())
++	}
++	if strings.Contains(out.String(), "OMP_NUM_THREADS") {
++		t.Errorf("note must not claim the operator-set var:\n%s", out.String())
++	}
++	// (b) child env: three pins at 1, operator's OMP at 7
++	b, err := os.ReadFile(filepath.Join(root, "runs", "r1", "e", "env.txt"))
++	if err != nil {
++		t.Fatal(err)
++	}
++	envTxt := string(b)
++	for _, want := range []string{"OPENBLAS_NUM_THREADS=1", "VECLIB_MAXIMUM_THREADS=1", "MKL_NUM_THREADS=1", "OMP_NUM_THREADS=7"} {
++		if !strings.Contains(envTxt, want) {
++			t.Errorf("child env missing %q; got:\n%s", want, envTxt)
++		}
++	}
++}
+diff --git a/cmd/metis/blaspins_test.go b/cmd/metis/blaspins_test.go
+new file mode 100644
+index 0000000..e42d051
+--- /dev/null
++++ b/cmd/metis/blaspins_test.go
+@@ -0,0 +1,61 @@
++package main
++
++import (
++	"reflect"
++	"testing"
++)
++
++// TestBlasPins_BareEnv: no ambient thread vars → all four pins injected, sorted.
++func TestBlasPins_BareEnv(t *testing.T) {
++	got := blasPins([]string{"PATH=/usr/bin", "HOME=/h"})
++	want := []string{
++		"MKL_NUM_THREADS=1",
++		"OMP_NUM_THREADS=1",
++		"OPENBLAS_NUM_THREADS=1",
++		"VECLIB_MAXIMUM_THREADS=1",
++	}
++	if !reflect.DeepEqual(got, want) {
++		t.Errorf("blasPins = %v, want %v", got, want)
++	}
++}
++
++// TestBlasPins_OperatorValueWins: an ambient-set var is NOT pinned — an explicit
++// operator choice always wins (the issue's escape hatch by construction).
++func TestBlasPins_OperatorValueWins(t *testing.T) {
++	got := blasPins([]string{"OMP_NUM_THREADS=8", "PATH=/usr/bin"})
++	for _, kv := range got {
++		if kv == "OMP_NUM_THREADS=1" {
++			t.Fatalf("ambient OMP_NUM_THREADS=8 must suppress the pin; got %v", got)
++		}
++	}
++	if len(got) != 3 {
++		t.Errorf("want 3 remaining pins, got %v", got)
++	}
++}
++
++// TestBlasPins_AllSetIsEmptyNonNil: fully pinned ambient env → empty but NON-nil
++// (runExperiment uses nil as "not yet computed"; empty must not recompute).
++func TestBlasPins_AllSetIsEmptyNonNil(t *testing.T) {
++	got := blasPins([]string{
++		"OMP_NUM_THREADS=4", "OPENBLAS_NUM_THREADS=4",
++		"VECLIB_MAXIMUM_THREADS=4", "MKL_NUM_THREADS=4",
++	})
++	if got == nil || len(got) != 0 {
++		t.Errorf("want empty non-nil, got %#v (nil=%v)", got, got == nil)
++	}
++}
++
++// TestBlasPins_PrefixNotName: OMP_NUM_THREADS_X=9 is a DIFFERENT var — must not
++// suppress the OMP_NUM_THREADS pin (name match is exact, up to '=').
++func TestBlasPins_PrefixNotName(t *testing.T) {
++	got := blasPins([]string{"OMP_NUM_THREADS_X=9"})
++	found := false
++	for _, kv := range got {
++		if kv == "OMP_NUM_THREADS=1" {
++			found = true
++		}
++	}
++	if !found {
++		t.Errorf("prefix-named var must not suppress the real pin; got %v", got)
++	}
++}
+diff --git a/cmd/metis/board.go b/cmd/metis/board.go
+index a8a8d51..0407870 100644
+--- a/cmd/metis/board.go
++++ b/cmd/metis/board.go
+@@ -209,6 +209,19 @@ func (b *boardWriter) forceFlush() {
+	b.flushLocked(b.now())
+ }
+
++// discardFrame atomically erases the live board and forgets it. Pending complete
++// ordinary lines still flush, but neither this update nor the later deferred close
++// can redraw stale progress after a sweep failure.
++func (b *boardWriter) discardFrame() {
++	b.mu.Lock()
++	defer b.mu.Unlock()
++	if b.closed {
++		return
++	}
++	b.frame = nil
++	b.flushLocked(b.now())
++}
++
+ // flushLocked is the ONE atomic terminal update: erase the painted board, dump the
+ // complete pending lines, redraw the stored frame — bracketed in DEC 2026 synchronized
+ // output (metis#47: supporting terminals apply it atomically, killing the erase→redraw
+diff --git a/cmd/metis/board_test.go b/cmd/metis/board_test.go
+index 284b76b..67ba9e9 100644
+--- a/cmd/metis/board_test.go
++++ b/cmd/metis/board_test.go
+@@ -221,6 +221,33 @@ func TestBoardWriter_CloseFlushesPending(t *testing.T) {
+	}
+ }
+
++func TestBoardWriter_DiscardFrameErasesWithoutRedraw(t *testing.T) {
++	var term strings.Builder
++	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond))
++	bw.paint([]string{"folds 2/8", "31.2 folds/min · ETA 12s"})
++	offset := term.Len()
++
++	bw.discardFrame()
++	bw.close()
++	suffix := term.String()[offset:]
++	if !strings.Contains(suffix, "\x1b[2A\x1b[J") {
++		t.Fatalf("discard must erase the painted two-line frame: %q", suffix)
++	}
++	for _, stale := range []string{"folds 2/8", "folds/min", "ETA"} {
++		if strings.Contains(suffix, stale) {
++			t.Errorf("discard/close redrew stale token %q: %q", stale, suffix)
++		}
++	}
++	if !strings.HasSuffix(suffix, "\x1b[?25h") {
++		t.Errorf("close after discard must restore the cursor: %q", suffix)
++	}
++	n := term.Len()
++	bw.close()
++	if term.Len() != n {
++		t.Fatal("close after discard must remain idempotent")
++	}
++}
++
+ // Board mode end-to-end over the fixture sweep: frames paint (cursor hide, fold rows),
+ // the #30 plain lines are REPLACED (not duplicated), the final frame carries the
+ // completed counts, and a capture warning — the plan-review bypass route (o.out) —
+@@ -283,6 +310,85 @@ func TestRunExperiment_BoardMode(t *testing.T) {
+	}
+ }
+
++func TestRunExperiment_BoardFailureRejectsPostPublicationTickAndDiscardsFrame(t *testing.T) {
++	ws := t.TempDir()
++	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
++	control := newRunControl(2)
++	exec := newFailureBarrierExec()
++	out := &concurrentBuffer{}
++	boardTick := make(chan time.Time)
++	tickSelected := make(chan struct{}, 2)
++	tickFinished := make(chan struct{}, 2)
++	publishedOffset := make(chan int, 1)
++	postFailureTickSend := make(chan error, 1)
++	control.beforeFailureUnlock = func() {
++		publishedOffset <- out.len()
++		postFailureTickSend <- sendBoardTickWithin(boardTick, at(2000), "post-publication tick receive")
++		close(exec.failurePublished)
++	}
++	result := make(chan error, 1)
++	go func() {
++		_, err := runExperiment(runOpts{
++			expPath: expPath, now: fixedNow(),
++			git: fakeGitProbe{name: "metis", sha: "sha"}, exec: exec, out: out,
++			maxParallel: 2, runControl: control, tui: true, boardTick: boardTick,
++			beforeBoardTick: func() { tickSelected <- struct{}{} },
++			afterBoardTick:  func() { tickFinished <- struct{}{} },
++		})
++		result <- err
++	}()
++	for i := 0; i < 4; i++ {
++		awaitRunControl(t, exec.innerEntered, "four board-mode inner run directories")
++	}
++
++	if err := sendBoardTickWithin(boardTick, at(1000), "pre-failure tick receive"); err != nil {
++		t.Fatal(err)
++	}
++	awaitRunControl(t, tickSelected, "pre-failure board tick selection")
++	awaitRunControl(t, tickFinished, "pre-failure board tick completion")
++	preFailure := out.snapshot()
++	for _, want := range []string{"outer 0/2", "fold 0 — queued", "folds/min"} {
++		if !strings.Contains(preFailure, want) {
++			t.Fatalf("pre-failure board missing %q:\n%s", want, preFailure)
++		}
++	}
++
++	close(exec.releaseFailure)
++	offset := awaitRunControl(t, publishedOffset, "board failure publication offset")
++	if err := awaitRunControl(t, postFailureTickSend, "post-publication tick send result"); err != nil {
++		awaitRunControl(t, result, "board-mode failure cleanup after tick-send timeout")
++		t.Fatal(err)
++	}
++	awaitRunControl(t, tickSelected, "post-publication board tick selection")
++	awaitRunControl(t, tickFinished, "rejected post-publication board tick")
++	err := awaitRunControl(t, result, "board-mode failure cleanup")
++	if err == nil || !strings.Contains(err.Error(), "injected train failure") {
++		t.Fatalf("board-mode error = %v, want injected train failure", err)
++	}
++	suffix := out.snapshot()[offset:]
++	for _, forbidden := range []string{
++		"outer 0/2", "fold 0 — queued", "configs ", "folds ", "folds/min", "ETA", "score ", "estimate", "mean ",
++	} {
++		if strings.Contains(suffix, forbidden) {
++			t.Errorf("post-publication board output contains stale token %q:\n%q", forbidden, suffix)
++		}
++	}
++	if !strings.Contains(suffix, "\x1b[J") || !strings.HasSuffix(suffix, "\x1b[?25h") {
++		t.Errorf("failure cleanup must erase the board and restore the cursor: %q", suffix)
++	}
++}
++
++func sendBoardTickWithin(ch chan<- time.Time, tick time.Time, what string) error {
++	timer := time.NewTimer(runControlTestTimeout)
++	defer timer.Stop()
++	select {
++	case ch <- tick:
++		return nil
++	case <-timer.C:
++		return fmt.Errorf("timed out waiting for %s", what)
++	}
++}
++
+ // --no-tui and non-TTY stdout both force tui=false through the real CLI parse; a
+ // dry run never boards. (isCharDevice on a test's non-terminal stdout is false, so
+ // the flag path is what we can pin here; the char-device branch is covered by the
+@@ -304,7 +410,7 @@ func TestServerPool_NoticeRoutesThroughBoard(t *testing.T) {
+	var term strings.Builder
+	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond))
+	bw.paint([]string{"BOARD"})
+-	pool := newServerPool(bw) // what runExperiment does post-reorder: pool captures the compositor
++	pool := newServerPool(bw, nil) // what runExperiment does post-reorder: pool captures the compositor
+	pool.noticeOnce("k", "server died; falling back to legacy exec")
+	s := term.String()
+	notice := strings.Index(s, "metis: forkserver: server died")
+diff --git a/cmd/metis/capture.go b/cmd/metis/capture.go
+index a38fecd..d462aca 100644
+--- a/cmd/metis/capture.go
++++ b/cmd/metis/capture.go
+@@ -144,7 +144,7 @@ func captureRunCode(closureByRepo map[string][]string, primaryRoot, specPath, re
+ // captureSweepCode captures the sweep's code closure + spec ONCE (per-shape-run: the
+ // closure is the same across points) to `refs/metis/sweeps/<shapeRunID>` and backfills
+ // every point-record's CodeManifest with the D + commit + capture status.
+-func captureSweepCode(o runOpts, man sweepManifest) error {
++func captureSweepCode(o runOpts, man sweepManifest) (record.Hash, error) {
+	closureByRepo := sweepClosure(o.expPath, man)
+	primary := cacheProjectRoot(o.stepPath, filepath.Dir(o.expPath))
+	commit, d, status := captureRunCode(closureByRepo, primary, o.expPath, "refs/metis/sweeps/"+man.ShapeRunID)
+@@ -157,14 +157,14 @@ func captureSweepCode(o runOpts, man sweepManifest) error {
+	for _, p := range man.Points {
+		pfp, pdirty, err := backfillCodeManifest(o.expPath, p.RunID, d, commit, status)
+		if err != nil {
+-			return err
++			return fp, err
+		}
+		if fp == "" && pfp != "" {
+			fp, dirty = pfp, pdirty
+		}
+	}
+	printFingerprintLine(o.out, fp, commit, dirty)
+-	return nil
++	return fp, nil // metis#50: the run-end summary reuses the minted cohort (one mint site)
+ }
+
+ // captureSingleRun captures ONE run's code closure + spec to `refs/metis/runs/<runID>`
+diff --git a/cmd/metis/capture_e2e_test.go b/cmd/metis/capture_e2e_test.go
+index 38961bd..1e4286b 100644
+--- a/cmd/metis/capture_e2e_test.go
++++ b/cmd/metis/capture_e2e_test.go
+@@ -104,7 +104,7 @@ func TestCaptureSweepCode_BackfillsCodeManifest(t *testing.T) {
+	man := sweepManifest{ShapeRunID: "srun-e2e", Points: []pointRun{{RunID: "pt-0", Status: "failed"}, {RunID: runID, Status: "ok"}}}
+	var out bytes.Buffer
+	o := runOpts{expPath: expPath, stepPath: []string{filepath.Join(root, "steps")}, out: &out}
+-	if err := captureSweepCode(o, man); err != nil {
++	if _, err := captureSweepCode(o, man); err != nil {
+		t.Fatalf("captureSweepCode: %v", err)
+	}
+
+diff --git a/cmd/metis/exec.go b/cmd/metis/exec.go
+index 28be377..0dd6d1b 100644
+--- a/cmd/metis/exec.go
++++ b/cmd/metis/exec.go
+@@ -38,6 +38,9 @@ type execStep struct {
+	pool *serverPool // metis#44: when non-nil, convention-conforming wrappers route through the
+	//                  warm fork-server (one per project root) instead of a fresh uv/python spawn;
+	//                  non-conforming wrappers + broken servers fall back to the legacy path below.
++	pins []string // metis#48: default leaf BLAS pins (computed once per run by runExperiment;
++	//              ambient-set names already excluded there) — appended to the legacy child env.
++	//              The fork-server path carries them on the SERVER env instead (children inherit).
+ }
+
+ // stepEnv builds the per-step METIS_* contract vars — the ONE definition both executors
+@@ -125,6 +128,9 @@ func (e execStep) Execute(step experiment.Step, runDir string) (experiment.StepR
+			base = append(base, kv)
+		}
+	}
++	// metis#48: default leaf BLAS pins (operator-exported values already won in blasPins,
++	// so no duplicate names reach the child).
++	base = append(base, e.pins...)
+	for _, k := range sortedKeys(env) {
+		base = append(base, k+"="+env[k])
+	}
+diff --git a/cmd/metis/exec_test.go b/cmd/metis/exec_test.go
+index 71ff5a4..7d7adb9 100644
+--- a/cmd/metis/exec_test.go
++++ b/cmd/metis/exec_test.go
+@@ -231,3 +231,28 @@ func TestCollectArtifacts_RecursiveExcludesReserved(t *testing.T) {
+		t.Errorf("artifacts = %v; want %v", arts, want)
+	}
+ }
++
++// TestExecStep_InjectsBlasPins (metis#48): the pins field reaches the child env at the
++// legacy spawn seam. The ambient-wins RULE lives in blasPins (unit-tested); this pins
++// the PLUMBING — whatever pins the wiring computed are in the subprocess env.
++func TestExecStep_InjectsBlasPins(t *testing.T) {
++	root := repoRoot(t)
++	runDir := t.TempDir()
++	e := execStep{
++		stepPath: []string{filepath.Join(root, "testdata", "steps")},
++		out:      io.Discard,
++		pins:     []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"},
++	}
++	if _, err := e.Execute(experiment.Step{ID: "e", Uses: "test/env-dump"}, runDir); err != nil {
++		t.Fatalf("Execute: %v", err)
++	}
++	b, err := os.ReadFile(filepath.Join(runDir, "e", "env.txt"))
++	if err != nil {
++		t.Fatalf("read env.txt: %v", err)
++	}
++	for _, want := range []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"} {
++		if !strings.Contains(string(b), want) {
++			t.Errorf("child env missing %q; got:\n%s", want, b)
++		}
++	}
++}
+diff --git a/cmd/metis/forkexec.go b/cmd/metis/forkexec.go
+index 39d1be0..b82d818 100644
+--- a/cmd/metis/forkexec.go
++++ b/cmd/metis/forkexec.go
+@@ -108,11 +108,14 @@ type forkReq struct {
+ }
+
+ // startForkServer launches `uv run --project <root> python -m metis.forkserver` and wires
+-// the reader goroutines. The server inherits the ambient env (the operator's BLAS pins
+-// apply to every child); per-step METIS_* vars travel in requests, never here.
+-func startForkServer(root string) (*forkServer, error) {
++// the reader goroutines. The server env = ambient + the default single-thread BLAS pins
++// (metis#48; names the operator exported are already excluded by blasPins — an explicit
++// choice wins). Forked step children inherit this env; per-step METIS_* vars travel in
++// requests, never here.
++func startForkServer(root string, pins []string) (*forkServer, error) {
+	cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.forkserver")
+	cmd.Dir = root
++	cmd.Env = append(os.Environ(), pins...)
+	// Own process GROUP: `uv run` spawns python as a child (no exec), and the server forks
+	// step children — group-kill is the only way to reap the whole tree on a hung shutdown
+	// (and a test's mid-flight kill). Normal shutdown stays graceful (stdin EOF → drain);
+@@ -273,14 +276,16 @@ type serverPool struct {
+	warned  map[string]bool
+	out     io.Writer
+	outMu   sync.Mutex
++	pins    []string // metis#48: default leaf BLAS pins, applied to every server's spawn env
+ }
+
+-func newServerPool(out io.Writer) *serverPool {
++func newServerPool(out io.Writer, pins []string) *serverPool {
+	return &serverPool{
+		servers: map[string]*forkServer{},
+		broken:  map[string]bool{},
+		warned:  map[string]bool{},
+		out:     out,
++		pins:    pins,
+	}
+ }
+
+@@ -310,7 +315,7 @@ func (p *serverPool) execute(spec wrapperSpec, cwd string, env map[string]string
+	s := p.servers[spec.root]
+	if s == nil {
+		var err error
+-		s, err = startForkServer(spec.root)
++		s, err = startForkServer(spec.root, p.pins)
+		if err != nil {
+			p.broken[spec.root] = true
+			p.mu.Unlock()
+diff --git a/cmd/metis/forkexec_test.go b/cmd/metis/forkexec_test.go
+index ccd6861..66a55c3 100644
+--- a/cmd/metis/forkexec_test.go
++++ b/cmd/metis/forkexec_test.go
+@@ -103,7 +103,7 @@ func TestForkServerPool_RealServerRoundTrip(t *testing.T) {
+	t.Setenv("PYTHONPATH", mods)
+	t.Setenv("METIS_FORKSERVER_PRELOAD", "") // fast start; preload is a python-side concern
+
+-	pool := newServerPool(io.Discard)
++	pool := newServerPool(io.Discard, nil)
+	defer pool.shutdown()
+
+	stepDir := filepath.Join(t.TempDir(), "s1")
+@@ -145,7 +145,7 @@ func TestForkServerPool_BrokenRootFallsBack(t *testing.T) {
+		t.Skip("uv not on PATH")
+	}
+	var out strings.Builder
+-	pool := newServerPool(&syncWriter{w: &out})
++	pool := newServerPool(&syncWriter{w: &out}, nil)
+	defer pool.shutdown()
+	bogus := t.TempDir() // no pyproject/venv — uv run will fail (or hang-free error)
+	for i := 0; i < 3; i++ {
+@@ -175,7 +175,7 @@ func TestExecute_NonConformingWrapperUsesLegacyLoudly(t *testing.T) {
+	ws := t.TempDir()
+	var out strings.Builder
+	e := execStep{stepPath: []string{filepath.Join(root, "steps")}, expDir: ws,
+-		out: &out, pool: newServerPool(&out)}
++		out: &out, pool: newServerPool(&out, nil)}
+	defer e.pool.shutdown()
+	runDir := filepath.Join(ws, "runs", "r1")
+	res, err := e.Execute(experiment.Step{ID: "sh", Uses: "test/shellstep"}, runDir)
+@@ -230,7 +230,7 @@ func TestForkServerPerf_LooseBound(t *testing.T) {
+	legacy := time.Since(start)
+
+	start = time.Now()
+-	pool := newServerPool(io.Discard)
++	pool := newServerPool(io.Discard, nil)
+	defer pool.shutdown()
+	forkOne := func() {
+		dir := t.TempDir()
+@@ -279,7 +279,7 @@ func TestForkServerPool_MidFlightDeathErrorsTheStep(t *testing.T) {
+	t.Setenv("PYTHONPATH", mods)
+	t.Setenv("METIS_FORKSERVER_PRELOAD", "")
+
+-	pool := newServerPool(io.Discard)
++	pool := newServerPool(io.Discard, nil)
+	defer pool.shutdown()
+	type result struct {
+		ok  bool
+@@ -318,3 +318,46 @@ func TestForkServerPool_MidFlightDeathErrorsTheStep(t *testing.T) {
+		t.Fatalf("mid-flight death must be a STEP ERROR (dispatched-and-lost), got ok=%v err=nil", res.ok)
+	}
+ }
++
++// TestForkServer_ChildInheritsBlasPins (metis#48): the pool's pins land on the SERVER
++// process env at spawn, and a forked step child inherits them (per-request env carries
++// only METIS_*). Real uv + real fork-server — the seam the operator's sweeps ride.
++func TestForkServer_ChildInheritsBlasPins(t *testing.T) {
++	if _, err := exec.LookPath("uv"); err != nil {
++		t.Skip("uv not on PATH")
++	}
++	root := repoRoot(t)
++	mods := t.TempDir()
++	if err := os.WriteFile(filepath.Join(mods, "toyenv.py"), []byte(
++		"import json, os\n"+
++			"json.dump({\"omp\": os.environ.get(\"OMP_NUM_THREADS\", \"\")}, open(\"envcap.json\", \"w\"))\n"+
++			"json.dump({\"ok\": 1}, open(\"metrics.json\", \"w\"))\n"), 0o644); err != nil {
++		t.Fatal(err)
++	}
++	t.Setenv("PYTHONPATH", mods)
++	t.Setenv("METIS_FORKSERVER_PRELOAD", "") // fast start; preload is a python-side concern
++	// exactness: an ambient OMP_NUM_THREADS would DUPLICATE the appended pin (CPython's
++	// os.environ is last-wins so the assertion would still green, but keep it unambiguous):
++	t.Setenv("OMP_NUM_THREADS", "sentinel") // registers restore...
++	os.Unsetenv("OMP_NUM_THREADS")          // ...then genuinely absent for the spawn
++
++	pool := newServerPool(io.Discard, []string{"OMP_NUM_THREADS=1"})
++	defer pool.shutdown()
++
++	stepDir := filepath.Join(t.TempDir(), "s1")
++	if err := os.MkdirAll(stepDir, 0o755); err != nil {
++		t.Fatal(err)
++	}
++	resp, ok, ferr := pool.execute(wrapperSpec{root: root, module: "toyenv"}, stepDir,
++		map[string]string{"METIS_STEP_DIR": stepDir})
++	if !ok || ferr != nil || resp.Exit != 0 {
++		t.Fatalf("fork exec failed: ok=%v err=%v resp=%+v", ok, ferr, resp)
++	}
++	b, err := os.ReadFile(filepath.Join(stepDir, "envcap.json"))
++	if err != nil {
++		t.Fatalf("read envcap.json: %v", err)
++	}
++	if !strings.Contains(string(b), `"omp": "1"`) {
++		t.Errorf("forked child must inherit the server's pin; envcap.json = %s", b)
++	}
++}
+diff --git a/cmd/metis/identity_e2e_test.go b/cmd/metis/identity_e2e_test.go
+index bf7312c..e82c1c6 100644
+--- a/cmd/metis/identity_e2e_test.go
++++ b/cmd/metis/identity_e2e_test.go
+@@ -102,7 +102,7 @@ func TestCodeIdentity_TwoRowsOnCodeChange(t *testing.T) {
+		})
+	}
+	capture := func() record.RunRecord {
+-		if err := captureSweepCode(o, man); err != nil {
++		if _, err := captureSweepCode(o, man); err != nil {
+			t.Fatalf("captureSweepCode: %v", err)
+		}
+		rb, _ := os.ReadFile(filepath.Join(root, "runs", pa, "record.json"))
+diff --git a/cmd/metis/main.go b/cmd/metis/main.go
+index 65e8331..227ddba 100644
+--- a/cmd/metis/main.go
++++ b/cmd/metis/main.go
+@@ -45,7 +45,7 @@ func cmdRun(args []string) error {
+	sampleN := fs.Int("sample", 0, "metis#42: run m of the k outer folds (sparse fold sampling; 0/omitted = all k). k stays the estimand (each fold trains on (k-1)/k of the rows); m only trades precision for cost — use to probe a higher k (e.g. k=10, --sample 3) without the full k× bill. The SE over m<k folds is noisy (m-1 df): probe with it, don't re-select what ships on it. Errors on m>k, on a single-config (flat) run, and combined with --fast.")
+	forkserver := fs.Bool("forkserver", true, "metis#44: run convention-conforming step wrappers through a warm per-root fork-server (pre-imported pandas/sklearn; ~1s spawn tax removed per leaf). --forkserver=false = legacy per-step uv/python spawn (the escape hatch); non-conforming wrappers and failed servers fall back to legacy automatically (loud, once).")
+	noTUI := fs.Bool("no-tui", false, "metis#38: force the plain progress lines even on a TTY (the live board is default for a sweep when stdout is a terminal; piped/redirected output always gets plain lines)")
+-	parallel := fs.Int("parallel", defaultParallel(), "metis#31: max concurrent step subprocesses across ALL sweep levels (driver×sweeper×resample share one global cap); <=1 = serial (exact pre-#31 behavior). Default runtime.NumCPU(), overridable by METIS_MAX_PARALLEL. Caveat: each leaf is a Python process that may itself multi-thread (BLAS / sklearn n_jobs) — n=NumCPU can oversubscribe cores; pin OMP_NUM_THREADS=1 or set n below NumCPU. On a COLD cache the first batch's ≤n points may each recompute the shared upstream (a bounded thundering herd).")
++	parallel := fs.Int("parallel", defaultParallel(), "metis#31: max concurrent step subprocesses across ALL sweep levels (driver×sweeper×resample share one global cap); <=1 = serial (exact pre-#31 behavior). Default runtime.NumCPU(), overridable by METIS_MAX_PARALLEL. Leaf BLAS is pinned single-thread by default (metis#48; export a *_NUM_THREADS value yourself to override), so n is the ONE parallelism knob. On a COLD cache the first batch's ≤n points may each recompute the shared upstream (a bounded thundering herd).")
+	if err := fs.Parse(args); err != nil {
+		return err
+	}
+diff --git a/cmd/metis/nestedcv_e2e_test.go b/cmd/metis/nestedcv_e2e_test.go
+index 1539587..1800d60 100644
+--- a/cmd/metis/nestedcv_e2e_test.go
++++ b/cmd/metis/nestedcv_e2e_test.go
+@@ -69,6 +69,17 @@ func TestNestedCV_ProducesHonestEstimateNoShip(t *testing.T) {
+	if !strings.Contains(finalProg, "outer 2/2") || !strings.Contains(finalProg, "est 0.") {
+		t.Errorf("the final progress line must carry the completed outer count + a numeric est; got: %q", finalProg)
+	}
++	// metis#50: the run ends with a paste-ready summary — elapsed, rows → ledger, cohort,
++	// and the select commands with the fingerprint pre-filled.
++	if !strings.Contains(s, "metis: done in ") {
++		t.Errorf("missing the run-end summary:\n%s", s)
++	}
++	sum := s[strings.Index(s, "metis: done in "):]
++	for _, want := range []string{"rows →", ".ledger.csv", "cohort ", "metis select ", "--fingerprint ", "--best --promote", "metis ledger fingerprints "} {
++		if !strings.Contains(sum, want) {
++			t.Errorf("summary missing %q:\n%s", want, sum)
++		}
++	}
+	// One held-out score per (outer fold × family): outerK = sweeper.cv.k = 2, and a,b are one
+	// family → 2 held-out lines.
+	if n := strings.Count(s, "→ held-out "); n != 2 {
+diff --git a/cmd/metis/parallel_test.go b/cmd/metis/parallel_test.go
+index 568ff2a..afb86cc 100644
+--- a/cmd/metis/parallel_test.go
++++ b/cmd/metis/parallel_test.go
+@@ -2,15 +2,21 @@ package main
+
+ import (
+	"bytes"
++	"encoding/json"
++	"errors"
+	"fmt"
+	"io"
+	"os"
+	"path/filepath"
++	"reflect"
++	"strings"
+	"sync"
++	"sync/atomic"
+	"testing"
+	"time"
+
+	"github.com/xianxu/metis/pkg/experiment"
++	"github.com/xianxu/metis/pkg/record"
+ )
+
+ // TestSweep_ParallelEqualsSerial (metis#31 M3, cmd-level): the SAME sweep run
+@@ -22,7 +28,7 @@ import (
+ // write the fan-out touches (configs/points/err).
+ func TestSweep_ParallelEqualsSerial(t *testing.T) {
+	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds = 6 per-fold rows
+-	run := func(maxPar int) (ledger, manifest []byte) {
++	run := func(maxPar int) (ledger, manifest []byte, runs map[string]experiment.Run, records map[string]record.RunRecord) {
+		ws := t.TempDir()
+		expPath := writeShapeFile(t, ws, body)
+		if _, err := runExperiment(runOpts{
+@@ -48,16 +54,212 @@ func TestSweep_ParallelEqualsSerial(t *testing.T) {
+		if err != nil {
+			t.Fatalf("read manifest: %v", err)
+		}
+-		return lb, mb
++		return lb, mb, loadPersistedRuns(t, ws), loadPersistedRecords(t, ws)
+	}
+-	sL, sM := run(1)
+-	pL, pM := run(8)
++	sL, sM, sRuns, sRecords := run(1)
++	pL, pM, pRuns, pRecords := run(8)
+	if !bytes.Equal(sL, pL) {
+		t.Errorf("ledger bytes differ serial vs parallel:\n--serial--\n%s\n--parallel--\n%s", sL, pL)
+	}
+	if !bytes.Equal(sM, pM) {
+		t.Errorf("manifest bytes differ serial vs parallel:\n--serial--\n%s\n--parallel--\n%s", sM, pM)
+	}
++	if !reflect.DeepEqual(sRuns, pRuns) {
++		t.Errorf("run.json values differ serial vs parallel:\nserial=%#v\nparallel=%#v", sRuns, pRuns)
++	}
++	if !reflect.DeepEqual(sRecords, pRecords) {
++		t.Errorf("record.json values differ serial vs parallel:\nserial=%#v\nparallel=%#v", sRecords, pRecords)
++	}
++}
++
++func loadPersistedRuns(t *testing.T, root string) map[string]experiment.Run {
++	t.Helper()
++	paths, err := filepath.Glob(filepath.Join(root, "runs", "*", "run.json"))
++	if err != nil {
++		t.Fatal(err)
++	}
++	got, err := decodePersistedRuns(paths)
++	if err != nil {
++		t.Fatal(err)
++	}
++	return got
++}
++
++func decodePersistedRuns(paths []string) (map[string]experiment.Run, error) {
++	got := make(map[string]experiment.Run, len(paths))
++	for _, path := range paths {
++		directoryID := filepath.Base(filepath.Dir(path))
++		if _, exists := got[directoryID]; exists {
++			return nil, fmt.Errorf("duplicate run directory identity %q", directoryID)
++		}
++		b, err := os.ReadFile(path)
++		if err != nil {
++			return nil, err
++		}
++		var run experiment.Run
++		if err := json.Unmarshal(b, &run); err != nil {
++			return nil, fmt.Errorf("parse %s: %w", path, err)
++		}
++		if run.ID != directoryID {
++			return nil, fmt.Errorf("run payload id %q does not match directory identity %q", run.ID, directoryID)
++		}
++		run.Started = ""
++		run.Finished = ""
++		got[directoryID] = run
++	}
++	return got, nil
++}
++
++func loadPersistedRecords(t *testing.T, root string) map[string]record.RunRecord {
++	t.Helper()
++	paths, err := filepath.Glob(filepath.Join(root, "runs", "*", "record.json"))
++	if err != nil {
++		t.Fatal(err)
++	}
++	got, err := decodePersistedRecords(paths)
++	if err != nil {
++		t.Fatal(err)
++	}
++	return got
++}
++
++func decodePersistedRecords(paths []string) (map[string]record.RunRecord, error) {
++	got := make(map[string]record.RunRecord, len(paths))
++	for _, path := range paths {
++		directoryID := filepath.Base(filepath.Dir(path))
++		if _, exists := got[directoryID]; exists {
++			return nil, fmt.Errorf("duplicate record directory identity %q", directoryID)
++		}
++		b, err := os.ReadFile(path)
++		if err != nil {
++			return nil, err
++		}
++		var rec record.RunRecord
++		if err := json.Unmarshal(b, &rec); err != nil {
++			return nil, fmt.Errorf("parse %s: %w", path, err)
++		}
++		if rec.RunID != directoryID {
++			return nil, fmt.Errorf("record payload run_id %q does not match directory identity %q", rec.RunID, directoryID)
++		}
++		rec.Started = ""
++		rec.Finished = ""
++		got[directoryID] = rec
++	}
++	return got, nil
++}
++
++func TestSemanticArtifactLoadersRejectMismatchedAndDuplicateDirectoryIdentity(t *testing.T) {
++	writeArtifact := func(t *testing.T, path string, value any) {
++		t.Helper()
++		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
++			t.Fatal(err)
++		}
++		b, err := json.Marshal(value)
++		if err != nil {
++			t.Fatal(err)
++		}
++		if err := os.WriteFile(path, b, 0o644); err != nil {
++			t.Fatal(err)
++		}
++	}
++
++	t.Run("run.json", func(t *testing.T) {
++		path := filepath.Join(t.TempDir(), "runs", "dir-id", "run.json")
++		writeArtifact(t, path, experiment.Run{ID: "payload-id"})
++		if _, err := decodePersistedRuns([]string{path}); err == nil || !strings.Contains(err.Error(), "dir-id") {
++			t.Fatalf("mismatched run identity error = %v, want directory identity diagnostic", err)
++		}
++		writeArtifact(t, path, experiment.Run{ID: "dir-id"})
++		if _, err := decodePersistedRuns([]string{path, path}); err == nil || !strings.Contains(err.Error(), "duplicate") {
++			t.Fatalf("duplicate run directory error = %v, want duplicate diagnostic", err)
++		}
++	})
++
++	t.Run("record.json", func(t *testing.T) {
++		path := filepath.Join(t.TempDir(), "runs", "dir-id", "record.json")
++		writeArtifact(t, path, record.RunRecord{RunID: "payload-id"})
++		if _, err := decodePersistedRecords([]string{path}); err == nil || !strings.Contains(err.Error(), "dir-id") {
++			t.Fatalf("mismatched record identity error = %v, want directory identity diagnostic", err)
++		}
++		writeArtifact(t, path, record.RunRecord{RunID: "dir-id"})
++		if _, err := decodePersistedRecords([]string{path, path}); err == nil || !strings.Contains(err.Error(), "duplicate") {
++			t.Fatalf("duplicate record directory error = %v, want duplicate diagnostic", err)
++		}
++	})
++}
++
++type scheduleTrace struct {
++	mu     sync.Mutex
++	events []string
++}
++
++func (t *scheduleTrace) add(event string) {
++	t.mu.Lock()
++	defer t.mu.Unlock()
++	t.events = append(t.events, event)
++}
++
++func (t *scheduleTrace) snapshot() []string {
++	t.mu.Lock()
++	defer t.mu.Unlock()
++	return append([]string(nil), t.events...)
++}
++
++type scheduleTraceExec struct {
++	in    foldFakeExec
++	trace *scheduleTrace
++}
++
++func (e scheduleTraceExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
++	result, err := e.in.Execute(step, runDir)
++	if err == nil && step.ID == "train" {
++		e.trace.add("train-complete:" + runDir)
++	}
++	return result, err
++}
++
++func TestSweep_ColdAdmissionCompletesTrainBeforeFifthAcquire(t *testing.T) {
++	ws := t.TempDir()
++	body := strings.Replace(foldShapeMD("[a]"), "k: 2", "k: 6", 1)
++	expPath := writeShapeFile(t, ws, body)
++	control := newRunControl(2)
++	trace := &scheduleTrace{}
++	control.afterAcquire = func(label string) { trace.add("acquire:" + label) }
++
++	result := make(chan error, 1)
++	go func() {
++		_, err := runExperiment(runOpts{
++			expPath: expPath, now: fixedNow(),
++			git: fakeGitProbe{name: "metis", sha: "sha"}, cache: false,
++			exec: scheduleTraceExec{in: foldFakeExec{}, trace: trace}, out: io.Discard,
++			maxParallel: 2, runControl: control,
++		})
++		result <- err
++	}()
++	if err := awaitRunControl(t, result, "flat k=6 cold sweep"); err != nil {
++		t.Fatal(err)
++	}
++
++	events := trace.snapshot()
++	firstTrain, fifthAcquire := -1, -1
++	acquires := 0
++	for i, event := range events {
++		switch {
++		case strings.HasPrefix(event, "train-complete:") && firstTrain < 0:
++			firstTrain = i
++		case strings.HasPrefix(event, "acquire:"):
++			acquires++
++			if acquires == 5 {
++				fifthAcquire = i
++			}
++		}
++	}
++	if firstTrain < 0 || fifthAcquire < 0 {
++		t.Fatalf("trace missing first train completion or fifth acquire: %v", events)
++	}
++	if firstTrain >= fifthAcquire {
++		t.Fatalf("cold wave acquired five runs before completing a train: %v", events)
++	}
+ }
+
+ // peakExec wraps foldFakeExec, acquiring the SHARED leaf semaphore around each step
+@@ -97,29 +299,56 @@ func (p peakExec) Execute(step experiment.Step, runDir string) (experiment.StepR
+ // the run must complete (no deadlock). The fake leaf acquires the SAME sem the
+ // production execStep would.
+ func TestNestedCV_PeakConcurrencyWithinCap(t *testing.T) {
+-	const cap = 3
++	const cap = 2 // runControl admits 2n=4 concrete runs; the nested fan-out has >4 children
+	ws := t.TempDir()
+	// 3 configs → nested (outer folds = sweeper.cv.k = 2) × 2 inner folds → deep nesting, ~many leaf calls.
+	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
+	sem := make(chan struct{}, cap)
+	var mu sync.Mutex
+	var cur, peak int
++	var activeRuns, peakRuns, acquiredRuns, releasedRuns int
++	control := newRunControl(cap)
++	control.afterAcquire = func(string) {
++		mu.Lock()
++		defer mu.Unlock()
++		activeRuns++
++		acquiredRuns++
++		if activeRuns > peakRuns {
++			peakRuns = activeRuns
++		}
++	}
++	control.beforeRelease = func(string) {
++		mu.Lock()
++		defer mu.Unlock()
++		activeRuns--
++		releasedRuns++
++	}
+	pe := peakExec{sem: sem, mu: &mu, cur: &cur, peak: &peak, in: foldFakeExec{}}
+-	_, err := runExperiment(runOpts{
+-		expPath:     expPath,
+-		now:         fixedNow(),
+-		git:         fakeGitProbe{name: "metis", sha: "sha"},
+-		cache:       false, // every step runs → maximum fan-out against the cap
+-		exec:        pe,
+-		out:         io.Discard,
+-		maxParallel: cap,
+-		leafSem:     sem, // runExperiment reuses my sem (maxParallel>1 & non-nil)
+-	})
++	result := make(chan error, 1)
++	go func() {
++		_, err := runExperiment(runOpts{
++			expPath:     expPath,
++			now:         fixedNow(),
++			git:         fakeGitProbe{name: "metis", sha: "sha"},
++			cache:       false, // every step runs → maximum fan-out against the cap
++			exec:        pe,
++			out:         io.Discard,
++			maxParallel: cap,
++			leafSem:     sem, // runExperiment reuses my sem (maxParallel>1 & non-nil)
++			runControl:  control,
++		})
++		result <- err
++	}()
++	err := awaitRunControl(t, result, "nested run with more children than admission capacity")
+	if err != nil {
+		t.Fatalf("driver:cv run must complete (no deadlock), got: %v", err)
+	}
+	mu.Lock()
+	got := peak
++	gotActiveRuns := activeRuns
++	gotPeakRuns := peakRuns
++	gotAcquiredRuns := acquiredRuns
++	gotReleasedRuns := releasedRuns
+	mu.Unlock()
+	if got > cap {
+		t.Fatalf("peak concurrency %d exceeded the global cap %d — the leaf budget leaked across nesting", got, cap)
+@@ -127,6 +356,21 @@ func TestNestedCV_PeakConcurrencyWithinCap(t *testing.T) {
+	if got < 2 {
+		t.Fatalf("peak concurrency %d — the fan-out never overlapped, so the test can't prove the cap actually holds", got)
+	}
++	if gotPeakRuns > 2*cap {
++		t.Fatalf("peak admitted runs %d exceeded controller cap %d", gotPeakRuns, 2*cap)
++	}
++	if gotPeakRuns == 0 {
++		t.Fatal("injected controller observed no admitted runs")
++	}
++	if gotAcquiredRuns <= 2*cap {
++		t.Fatalf("controller observed %d concrete run attempts, want more than its %d-slot capacity", gotAcquiredRuns, 2*cap)
++	}
++	if gotActiveRuns != 0 {
++		t.Fatalf("active admitted runs after completion = %d, want 0", gotActiveRuns)
++	}
++	if gotAcquiredRuns != gotReleasedRuns {
++		t.Fatalf("admission hooks acquired=%d released=%d, want equal", gotAcquiredRuns, gotReleasedRuns)
++	}
+ }
+
+ // sleepExec is foldFakeExec with a fixed per-step delay, so a sweep has real
+@@ -137,6 +381,152 @@ type sleepExec struct {
+	d  time.Duration
+ }
+
++type concurrentBuffer struct {
++	mu sync.Mutex
++	b  bytes.Buffer
++}
++
++func (b *concurrentBuffer) Write(p []byte) (int, error) {
++	b.mu.Lock()
++	defer b.mu.Unlock()
++	return b.b.Write(p)
++}
++
++func (b *concurrentBuffer) snapshot() string {
++	b.mu.Lock()
++	defer b.mu.Unlock()
++	return b.b.String()
++}
++
++func (b *concurrentBuffer) len() int {
++	b.mu.Lock()
++	defer b.mu.Unlock()
++	return b.b.Len()
++}
++
++// failureBarrierExec holds the first four admitted inner folds at their train
++// step. Exactly one returns the concrete injected failure; its admitted siblings
++// wait for controller publication and then return the cancellation sentinel.
++type failureBarrierExec struct {
++	in               foldFakeExec
++	mu               sync.Mutex
++	innerDirs        map[string]struct{}
++	innerEntered     chan string
++	fourEntered      chan struct{}
++	releaseFailure   chan struct{}
++	failurePublished chan struct{}
++	winner           atomic.Bool
++}
++
++func newFailureBarrierExec() *failureBarrierExec {
++	return &failureBarrierExec{
++		innerDirs:        make(map[string]struct{}),
++		innerEntered:     make(chan string, 8),
++		fourEntered:      make(chan struct{}),
++		releaseFailure:   make(chan struct{}),
++		failurePublished: make(chan struct{}),
++	}
++}
++
++func (f *failureBarrierExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
++	if step.ID == partitionStepID {
++		f.mu.Lock()
++		if _, seen := f.innerDirs[runDir]; !seen {
++			f.innerDirs[runDir] = struct{}{}
++			f.innerEntered <- runDir
++			if len(f.innerDirs) == 4 {
++				close(f.fourEntered)
++			}
++		}
++		f.mu.Unlock()
++		if err := waitFailureBarrier(f.fourEntered, "four inner runs to enter"); err != nil {
++			return experiment.StepResult{}, err
++		}
++	}
++	if step.ID == "train" {
++		if f.winner.CompareAndSwap(false, true) {
++			if err := waitFailureBarrier(f.releaseFailure, "failing train release"); err != nil {
++				return experiment.StepResult{}, err
++			}
++			return experiment.StepResult{}, errors.New("injected train failure")
++		}
++		if err := waitFailureBarrier(f.failurePublished, "controller failure publication"); err != nil {
++			return experiment.StepResult{}, err
++		}
++		return experiment.StepResult{}, errRunAborted
++	}
++	return f.in.Execute(step, runDir)
++}
++
++func waitFailureBarrier(ch <-chan struct{}, what string) error {
++	// Fire before the enclosing test's 2s await so a broken orchestration returns
++	// the specific barrier error instead of only the outer generic timeout.
++	timer := time.NewTimer(runControlTestTimeout / 2)
++	defer timer.Stop()
++	select {
++	case <-ch:
++		return nil
++	case <-timer.C:
++		return fmt.Errorf("failure barrier timed out waiting for %s", what)
++	}
++}
++
++func (f *failureBarrierExec) dirCount() int {
++	f.mu.Lock()
++	defer f.mu.Unlock()
++	return len(f.innerDirs)
++}
++
++func TestNestedCV_FirstFailureStopsAllObservableWork(t *testing.T) {
++	ws := t.TempDir()
++	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
++	control := newRunControl(2)
++	exec := newFailureBarrierExec()
++	out := &concurrentBuffer{}
++	publishedOffset := make(chan int, 1)
++	control.beforeFailureUnlock = func() {
++		publishedOffset <- out.len()
++		close(exec.failurePublished)
++	}
++	result := make(chan error, 1)
++	go func() {
++		_, err := runExperiment(runOpts{
++			expPath: expPath, now: fixedNow(),
++			git: fakeGitProbe{name: "metis", sha: "sha"}, exec: exec, out: out,
++			maxParallel: 2, runControl: control,
++		})
++		result <- err
++	}()
++
++	for i := 0; i < 4; i++ {
++		awaitRunControl(t, exec.innerEntered, "four admitted inner run directories")
++	}
++	close(exec.releaseFailure)
++	offset := awaitRunControl(t, publishedOffset, "first failure publication")
++	err := awaitRunControl(t, result, "nested failure to return without parent/child admission deadlock")
++	if err == nil || !strings.Contains(err.Error(), "config ") || !strings.Contains(err.Error(), "injected train failure") {
++		t.Fatalf("error = %v, want contextual concrete config/fold failure", err)
++	}
++	if errors.Is(err, errRunAborted) || strings.Contains(err.Error(), errRunAborted.Error()) {
++		t.Fatalf("top-level error exposed cancellation sentinel instead of cause: %v", err)
++	}
++	if got := exec.dirCount(); got != 4 {
++		t.Fatalf("inner run dirs = %d, want exactly four admitted dirs and no fifth start", got)
++	}
++	suffix := out.snapshot()[offset:]
++	for _, forbidden := range []string{"metis: progress", "folds/min", "ETA", "score ", "estimate", "mean "} {
++		if strings.Contains(suffix, forbidden) {
++			t.Errorf("post-failure output contains %q:\n%s", forbidden, suffix)
++		}
++	}
++	if matches, _ := filepath.Glob(filepath.Join(ws, "sweeps", "*", "manifest.json")); len(matches) != 0 {
++		t.Fatalf("failure persisted %d manifest(s), want none", len(matches))
++	}
++	if _, statErr := os.Stat(filepath.Join(ws, "shape.ledger.csv")); !errors.Is(statErr, os.ErrNotExist) {
++		t.Fatalf("failure persisted a ledger: %v", statErr)
++	}
++}
++
+ func (s sleepExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+	res, err := s.in.Execute(step, runDir)
+	time.Sleep(s.d)
+diff --git a/cmd/metis/progress.go b/cmd/metis/progress.go
+index 19d5c7d..71ee241 100644
+--- a/cmd/metis/progress.go
++++ b/cmd/metis/progress.go
+@@ -205,8 +205,8 @@ type boardState struct {
+
+ // sweepProgress is the mutex'd sink shared by every pass of one shape-run. Events
+ // arrive concurrently (ParExec goroutines across sibling outer folds, each holding
+-// its own Run's event mutex); lock order is strictly Run-mu → sink-mu → the
+-// syncWriter under `out` — acyclic. Emit policy: fold/config events are throttled
++// its own Run's event mutex); health-gated paths use the strict order runControl.mu
++// → sink.mu → boardWriter.mu (never the reverse). Emit policy: fold/config events are throttled
+ // to one line per second (injected clock — tests script it, never sleep); a
+ // driver-level (outer fold) completion ALWAYS emits; finish() emits the terminal
+ // line. A nil *sweepProgress is a no-op everywhere (the non-sweep path is silent).
+@@ -355,6 +355,17 @@ func (sp *sweepProgress) tick() {
+	bw.forceFlush() // metis#46: the tick is what re-pins the board after a burst window
+ }
+
++// abort removes the stored live frame after a sweep failure. Lock order remains
++// progress -> board; the controller is never called while either lock is held.
++func (sp *sweepProgress) abort() {
++	if sp == nil || sp.bw == nil {
++		return
++	}
++	sp.mu.Lock()
++	defer sp.mu.Unlock()
++	sp.bw.discardFrame()
++}
++
+ // maybeEmit writes the line if forced (driver/finish) or the throttle elapsed —
+ // 1s for plain lines (a log is a record), 100ms for board repaints (a board is a
+ // display; the 500ms ticker guarantees freshness anyway). Caller holds sp.mu.
+diff --git a/cmd/metis/run.go b/cmd/metis/run.go
+index 8323bc4..c71ede6 100644
+--- a/cmd/metis/run.go
++++ b/cmd/metis/run.go
+@@ -6,6 +6,7 @@ import (
+	"io"
+	"os"
+	"path/filepath"
++	"strings"
+	"sync"
+	"time"
+
+@@ -81,14 +82,22 @@ type runOpts struct {
+	readRoot    string        // metis#23: when set, the production execStep confines base-dataset reads to this root
+	maxParallel int           // metis#31: >1 ⇒ ParExec batches + a leaf semaphore; sizes leafSem
+	leafSem     chan struct{} // metis#31: the shared global subprocess budget (nil = serial/cache-only)
++	runControl  *runControl   // one per shape run: global abort + optional 2n admission slots
++	runLabel    string        // config/fold/preamble context captured with the first error
+	forkserver  bool          // metis#44: warm fork-server leaf executor (cmdRun default true;
+	//                           zero-value false keeps direct runOpts callers/tests on legacy exec)
+	forkPool *serverPool // metis#44: the per-root warm-server pool, created once per runExperiment
+	//                      when forkserver is set; threaded through nested runOpts copies.
+	tui bool // metis#38: stdout is a TTY and --no-tui wasn't passed — a SWEEP pins the live board
+	//          (a plain experiment ignores it; non-TTY/piped runs stay on the #30 plain lines)
+-	board     *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
+-	leafGauge func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
++	board           *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
++	boardTick       <-chan time.Time  // test seam: nil uses the production 500ms ticker
++	beforeBoardTick func()            // test seam: after tick selection, before health observation
++	afterBoardTick  func()            // test seam: after the health observation returns
++	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
++	leafPins        []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
++	//                             runExperiment (nil = not yet computed; non-nil rides nested runOpts
++	//                             copies like forkPool — an all-suppressed result is empty, not nil)
+ }
+
+ // runExperiment reads the experiment at o.expPath and dispatches: a `type:
+@@ -145,11 +154,24 @@ func runExperiment(o runOpts) (experiment.Run, error) {
+		out = &syncWriter{w: out}
+		o.out = out
+	}
++	// metis#48: default leaf BLAS pins — computed ONCE per top-level run from the ambient
++	// env (an exported operator value wins by exclusion in blasPins), announced loudly,
++	// and injected at both spawn seams (legacy execStep child env; fork-server process
++	// env). Fake-exec runs spawn nothing (no pins, no note); dry-run lists configs (same).
++	// Placed AFTER the writer wrap (the note must route through the board) and BEFORE the
++	// pool creation (the server spawn env needs the pins).
++	if o.exec == nil && !o.dryRun && o.leafPins == nil {
++		o.leafPins = blasPins(os.Environ())
++		if len(o.leafPins) > 0 {
++			fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
++				strings.Join(o.leafPins, " "))
++		}
++	}
+	// metis#44: one warm fork-server pool per top-level run, shut down (EOF-drain) when the
+	// run ends. Only the production executor uses it (an injected test exec bypasses execStep).
+	// Constructed AFTER the writer wrap — its fallback notices must route through the board.
+	if o.forkserver && o.exec == nil && o.forkPool == nil {
+-		o.forkPool = newServerPool(out)
++		o.forkPool = newServerPool(out, o.leafPins)
+		defer o.forkPool.shutdown()
+	}
+	if exp.Type == "experiment-shape" {
+@@ -160,6 +182,9 @@ func runExperiment(o runOpts) (experiment.Run, error) {
+		if err := experiment.ValidateShape(sh); err != nil {
+			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
+		}
++		if o.runControl == nil {
++			o.runControl = newRunControl(o.maxParallel)
++		}
+		return experiment.Run{}, runShapeSweep(o, sh, now, out)
+	}
+	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
+@@ -188,6 +213,15 @@ func singleRunID(o runOpts, exp experiment.Experiment, now func() time.Time) str
+ // both the 1-point path and the sweep loop (metis#7) call — so the run/cache/record wiring
+ // lives in ONE place (ARCH-DRY).
+ func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
++	if o.runControl == nil {
++		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
++	}
++	return o.runControl.run(o.runLabel, func() (experiment.Run, error) {
++		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
++	})
++}
++
++func runResolvedExperimentAdmitted(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
+	baseDir := filepath.Dir(o.expPath)
+	// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
+	// the child's env, and the child's cwd IS the step dir — a relative path would
+@@ -202,7 +236,7 @@ func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, n
+		return experiment.Run{}, err
+	}
+
+-	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool}
++	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool, pins: o.leafPins}
+	if o.exec != nil {
+		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
+	}
+diff --git a/cmd/metis/runcontrol.go b/cmd/metis/runcontrol.go
+new file mode 100644
+index 0000000..f1201fd
+--- /dev/null
++++ b/cmd/metis/runcontrol.go
+@@ -0,0 +1,104 @@
++package main
++
++import (
++	"errors"
++	"fmt"
++	"sync"
++
++	"github.com/xianxu/metis/pkg/experiment"
++)
++
++var errRunAborted = errors.New("run aborted after earlier sweep failure")
++
++// runControl bounds admitted concrete runs independently of leaf subprocess
++// parallelism and latches the first whole-run failure. Observation callbacks
++// must not call back into the controller or block production work.
++type runControl struct {
++	slots chan struct{}
++
++	mu  sync.Mutex
++	err error
++
++	beforeFailureLock   func()
++	beforeFailureUnlock func()
++	afterAcquire        func(label string)
++	beforeRelease       func(label string)
++}
++
++func newRunControl(maxParallel int) *runControl {
++	control := &runControl{}
++	if maxParallel > 1 {
++		control.slots = make(chan struct{}, 2*maxParallel)
++	}
++	return control
++}
++
++func (c *runControl) firstError() error {
++	c.mu.Lock()
++	defer c.mu.Unlock()
++	return c.err
++}
++
++// whileHealthy linearizes an observable transition against first-failure
++// publication. The callback runs while c.mu is held and therefore must not call
++// back into runControl. Downstream locks are acquired only inside fn, preserving
++// the global control -> progress/pass/manifest order.
++func (c *runControl) whileHealthy(fn func()) bool {
++	c.mu.Lock()
++	defer c.mu.Unlock()
++	if c.err != nil {
++		return false
++	}
++	fn()
++	return true
++}
++
++func (c *runControl) fail(label string, err error) error {
++	if err == nil {
++		return nil
++	}
++	contextual := err
++	if label != "" {
++		contextual = fmt.Errorf("%s: %w", label, err)
++	}
++	if c.beforeFailureLock != nil {
++		c.beforeFailureLock()
++	}
++
++	c.mu.Lock()
++	if c.err == nil {
++		c.err = contextual
++		if c.beforeFailureUnlock != nil {
++			c.beforeFailureUnlock()
++		}
++	}
++	authoritative := c.err
++	c.mu.Unlock()
++	return authoritative
++}
++
++func (c *runControl) run(label string, fn func() (experiment.Run, error)) (experiment.Run, error) {
++	if c.slots != nil {
++		c.slots <- struct{}{}
++		defer func() { <-c.slots }()
++		if c.afterAcquire != nil {
++			c.afterAcquire(label)
++		}
++		if c.beforeRelease != nil {
++			defer func() { c.beforeRelease(label) }()
++		}
++	}
++
++	if c.firstError() != nil {
++		return experiment.Run{}, errRunAborted
++	}
++
++	run, err := fn()
++	if err != nil {
++		return experiment.Run{}, c.fail(label, err)
++	}
++	if c.firstError() != nil {
++		return experiment.Run{}, errRunAborted
++	}
++	return run, nil
++}
+diff --git a/cmd/metis/runcontrol_test.go b/cmd/metis/runcontrol_test.go
+new file mode 100644
+index 0000000..04c9e8a
+--- /dev/null
++++ b/cmd/metis/runcontrol_test.go
+@@ -0,0 +1,508 @@
++package main
++
++import (
++	"bytes"
++	"errors"
++	"os"
++	"path/filepath"
++	"reflect"
++	"runtime"
++	"strings"
++	"sync/atomic"
++	"testing"
++	"time"
++
++	"github.com/xianxu/metis/pkg/experiment"
++)
++
++func TestRunResolvedExperiment_AbortedBeforeSideEffects(t *testing.T) {
++	ws := t.TempDir()
++	control := newRunControl(2)
++	control.fail("earlier fold", errors.New("failed"))
++	var out bytes.Buffer
++	exp := experiment.Experiment{Header: experiment.Header{Type: "experiment", ID: "queued"}}
++
++	_, err := runResolvedExperiment(exp, runOpts{
++		expPath:    filepath.Join(ws, "shape.md"),
++		runControl: control,
++		runLabel:   "queued fold",
++		cache:      true,
++	}, "queued", fixedNow(), &out)
++	if !errors.Is(err, errRunAborted) {
++		t.Fatalf("error = %v, want errRunAborted", err)
++	}
++	if out.Len() != 0 {
++		t.Fatalf("aborted run wrote output: %q", out.String())
++	}
++	if _, statErr := os.Stat(filepath.Join(ws, "runs", "queued")); !errors.Is(statErr, os.ErrNotExist) {
++		t.Fatalf("queued run created state: %v", statErr)
++	}
++	if _, statErr := os.Stat(filepath.Join(ws, ".metis-cache")); !errors.Is(statErr, os.ErrNotExist) {
++		t.Fatalf("queued run initialized cache state: %v", statErr)
++	}
++}
++
++const runControlTestTimeout = 2 * time.Second
++
++type runControlResult struct {
++	run experiment.Run
++	err error
++}
++
++func isZeroRun(run experiment.Run) bool {
++	return reflect.DeepEqual(run, experiment.Run{})
++}
++
++func awaitRunControl[T any](t *testing.T, ch <-chan T, what string) T {
++	t.Helper()
++	select {
++	case got := <-ch:
++		return got
++	case <-time.After(runControlTestTimeout):
++		t.Fatalf("timed out waiting for %s", what)
++		var zero T
++		return zero
++	}
++}
++
++func TestRunControlBoundsAdmissionAtTwiceParallelism(t *testing.T) {
++	control := newRunControl(3)
++	entered := make(chan struct{}, 12)
++	release := make(chan struct{})
++	results := make(chan runControlResult, 12)
++	var active atomic.Int32
++	var peak atomic.Int32
++	var acquired atomic.Int32
++	var released atomic.Int32
++	control.afterAcquire = func(string) { acquired.Add(1) }
++	control.beforeRelease = func(string) { released.Add(1) }
++
++	for range 12 {
++		go func() {
++			run, err := control.run("point", func() (experiment.Run, error) {
++				current := active.Add(1)
++				for old := peak.Load(); current > old && !peak.CompareAndSwap(old, current); old = peak.Load() {
++				}
++				entered <- struct{}{}
++				<-release
++				active.Add(-1)
++				return experiment.Run{ID: "ok"}, nil
++			})
++			results <- runControlResult{run: run, err: err}
++		}()
++	}
++
++	for i := 0; i < 6; i++ {
++		awaitRunControl(t, entered, "six admitted callbacks")
++	}
++	if got := len(control.slots); got != 6 {
++		t.Fatalf("admitted slots = %d, want 6", got)
++	}
++	select {
++	case <-entered:
++		t.Fatal("more than six callbacks entered before an admission slot was released")
++	default:
++	}
++	close(release)
++
++	for i := 0; i < 12; i++ {
++		got := awaitRunControl(t, results, "all bounded runs to drain")
++		if got.err != nil || got.run.ID != "ok" {
++			t.Fatalf("run result = (%+v, %v), want successful run", got.run, got.err)
++		}
++	}
++	if got := peak.Load(); got != 6 {
++		t.Fatalf("peak callbacks = %d, want exactly 6", got)
++	}
++	if got := acquired.Load(); got != 12 {
++		t.Fatalf("acquire hook calls = %d, want 12 attempted runs", got)
++	}
++	if got := released.Load(); got != 12 {
++		t.Fatalf("release hook calls = %d, want 12 attempted runs", got)
++	}
++}
++
++func TestRunControlHookPanicsStillReleaseAdmission(t *testing.T) {
++	panicValue := errors.New("observation hook panic")
++	for _, tc := range []struct {
++		name string
++		set  func(*runControl)
++	}{
++		{
++			name: "after acquire",
++			set: func(control *runControl) {
++				control.afterAcquire = func(string) { panic(panicValue) }
++			},
++		},
++		{
++			name: "before release",
++			set: func(control *runControl) {
++				control.beforeRelease = func(string) { panic(panicValue) }
++			},
++		},
++	} {
++		t.Run(tc.name, func(t *testing.T) {
++			control := &runControl{slots: make(chan struct{}, 1)}
++			tc.set(control)
++			got := recoverRunControlPanic(func() {
++				_, _ = control.run("observed", func() (experiment.Run, error) {
++					return experiment.Run{ID: "ok"}, nil
++				})
++			})
++			if got != panicValue {
++				t.Fatalf("recovered panic = %v, want exact hook panic %v", got, panicValue)
++			}
++			if got := len(control.slots); got != 0 {
++				t.Fatalf("admission slots after recovered hook panic = %d, want 0", got)
++			}
++		})
++	}
++}
++
++func recoverRunControlPanic(fn func()) (recovered any) {
++	defer func() { recovered = recover() }()
++	fn()
++	return nil
++}
++
++func TestRunControlPublishesFailureBeforeAdmissionRelease(t *testing.T) {
++	control := &runControl{slots: make(chan struct{}, 1)}
++	published := make(chan struct{})
++	letTokenRelease := make(chan struct{})
++	control.beforeFailureUnlock = func() {
++		close(published)
++		<-letTokenRelease
++	}
++
++	firstResult := make(chan runControlResult, 1)
++	go func() {
++		run, err := control.run("first", func() (experiment.Run, error) {
++			return experiment.Run{ID: "must-be-discarded"}, errors.New("boom")
++		})
++		firstResult <- runControlResult{run: run, err: err}
++	}()
++	awaitRunControl(t, published, "failure publication hook")
++
++	var secondCalled atomic.Bool
++	secondResult := make(chan runControlResult, 1)
++	go func() {
++		run, err := control.run("second", func() (experiment.Run, error) {
++			secondCalled.Store(true)
++			return experiment.Run{ID: "must-not-run"}, nil
++		})
++		secondResult <- runControlResult{run: run, err: err}
++	}()
++
++	if got := len(control.slots); got != 1 {
++		t.Fatalf("slots while failure publisher holds the mutex = %d, want 1", got)
++	}
++	close(letTokenRelease)
++
++	first := awaitRunControl(t, firstResult, "first failed run")
++	second := awaitRunControl(t, secondResult, "second aborted run")
++	if !isZeroRun(first.run) {
++		t.Fatalf("failed run = %+v, want zero Run", first.run)
++	}
++	if first.err == nil || first.err.Error() != "first: boom" {
++		t.Fatalf("first error = %v, want contextual first failure", first.err)
++	}
++	if !isZeroRun(second.run) || !errors.Is(second.err, errRunAborted) {
++		t.Fatalf("second result = (%+v, %v), want zero Run and errRunAborted", second.run, second.err)
++	}
++	if secondCalled.Load() {
++		t.Fatal("second callback executed after failure publication")
++	}
++	if got := control.firstError(); got == nil || got.Error() != "first: boom" {
++		t.Fatalf("stored first error = %v, want first: boom", got)
++	}
++}
++
++func TestRunControlAcquiresAdmissionBeforeCheckingFailure(t *testing.T) {
++	control := &runControl{slots: make(chan struct{}, 1)}
++	control.slots <- struct{}{}
++	prior := errors.New("prior failure")
++	var callbackCalled atomic.Bool
++	result := make(chan runControlResult, 1)
++
++	control.mu.Lock()
++	go func() {
++		run, err := control.run("later", func() (experiment.Run, error) {
++			callbackCalled.Store(true)
++			return experiment.Run{ID: "must-not-run"}, nil
++		})
++		result <- runControlResult{run: run, err: err}
++	}()
++
++	// Make one admission slot available while firstError remains blocked on mu.
++	// A correctly ordered run refills the slot before attempting the error check.
++	<-control.slots
++	timer := time.NewTimer(runControlTestTimeout)
++	defer timer.Stop()
++	for len(control.slots) != 1 {
++		select {
++		case <-timer.C:
++			control.err = prior
++			control.mu.Unlock()
++			t.Fatal("run did not acquire admission before attempting the failure check")
++		default:
++			runtime.Gosched()
++		}
++	}
++	control.err = prior
++	control.mu.Unlock()
++
++	got := awaitRunControl(t, result, "admitted run to observe prior failure")
++	if !isZeroRun(got.run) || !errors.Is(got.err, errRunAborted) {
++		t.Fatalf("run result = (%+v, %v), want zero Run and errRunAborted", got.run, got.err)
++	}
++	if callbackCalled.Load() {
++		t.Fatal("callback executed despite failure installed before the post-admission check")
++	}
++	if got := len(control.slots); got != 0 {
++		t.Fatalf("slots after aborted run = %d, want released", got)
++	}
++}
++
++func TestRunControlSerialStillLatchesFailure(t *testing.T) {
++	control := newRunControl(1)
++	if control.slots != nil {
++		t.Fatal("serial controller unexpectedly allocated admission slots")
++	}
++	if got := control.fail("ignored", nil); got != nil || control.firstError() != nil {
++		t.Fatalf("nil failure = %v with stored error %v, want neither", got, control.firstError())
++	}
++
++	failed, err := control.run("serial", func() (experiment.Run, error) {
++		return experiment.Run{ID: "must-be-discarded"}, errors.New("broken")
++	})
++	if !isZeroRun(failed) || err == nil || err.Error() != "serial: broken" {
++		t.Fatalf("failed result = (%+v, %v), want zero Run and contextual error", failed, err)
++	}
++
++	called := false
++	aborted, err := control.run("later", func() (experiment.Run, error) {
++		called = true
++		return experiment.Run{ID: "must-not-run"}, nil
++	})
++	if !isZeroRun(aborted) || !errors.Is(err, errRunAborted) {
++		t.Fatalf("later result = (%+v, %v), want zero Run and errRunAborted", aborted, err)
++	}
++	if called {
++		t.Fatal("later serial callback executed after failure")
++	}
++	if got := control.firstError(); got == nil || got.Error() != "serial: broken" {
++		t.Fatalf("stored first error = %v, want serial: broken", got)
++	}
++}
++
++func TestRunControlFailureWithoutLabelPreservesError(t *testing.T) {
++	control := newRunControl(1)
++	cause := errors.New("unlabeled failure")
++
++	got := control.fail("", cause)
++	if got != cause {
++		t.Fatalf("unlabeled failure = %v (%p), want original error %v (%p)", got, got, cause, cause)
++	}
++	if stored := control.firstError(); stored != cause {
++		t.Fatalf("stored unlabeled failure = %v (%p), want original error %v (%p)", stored, stored, cause, cause)
++	}
++}
++
++func TestRunControlConcurrentFailuresKeepOneContextualCause(t *testing.T) {
++	control := newRunControl(2)
++	entered := make(chan struct{}, 2)
++	release := make(chan struct{})
++	results := make(chan runControlResult, 2)
++
++	for _, tc := range []struct {
++		label string
++		err   string
++	}{{label: "left", err: "left failed"}, {label: "right", err: "right failed"}} {
++		tc := tc
++		go func() {
++			run, err := control.run(tc.label, func() (experiment.Run, error) {
++				entered <- struct{}{}
++				<-release
++				return experiment.Run{ID: tc.label}, errors.New(tc.err)
++			})
++			results <- runControlResult{run: run, err: err}
++		}()
++	}
++	awaitRunControl(t, entered, "left failure callback")
++	awaitRunControl(t, entered, "right failure callback")
++	close(release)
++
++	first := awaitRunControl(t, results, "first concurrent failure")
++	second := awaitRunControl(t, results, "second concurrent failure")
++	if !isZeroRun(first.run) || !isZeroRun(second.run) {
++		t.Fatalf("failed runs = (%+v, %+v), want zero Runs", first.run, second.run)
++	}
++	if first.err == nil || second.err == nil || first.err.Error() != second.err.Error() {
++		t.Fatalf("concurrent errors = (%v, %v), want one authoritative error", first.err, second.err)
++	}
++	if got := first.err.Error(); got != "left: left failed" && got != "right: right failed" {
++		t.Fatalf("authoritative error = %q, want one contextual cause", got)
++	}
++	if got := control.firstError(); got == nil || got.Error() != first.err.Error() {
++		t.Fatalf("stored first error = %v, want %v", got, first.err)
++	}
++}
++
++func TestRunControlWinnerHookRunsOnceInsideFailureMutex(t *testing.T) {
++	control := newRunControl(2)
++	callbacksEntered := make(chan struct{}, 2)
++	releaseLeft := make(chan struct{})
++	releaseRight := make(chan struct{})
++	hookEntered := make(chan struct{}, 1)
++	releaseWinner := make(chan struct{})
++	results := make(chan runControlResult, 2)
++	var hookCalls atomic.Int32
++	control.beforeFailureUnlock = func() {
++		hookCalls.Add(1)
++		hookEntered <- struct{}{}
++		<-releaseWinner
++	}
++
++	for _, failure := range []struct {
++		label   string
++		release <-chan struct{}
++	}{{label: "left", release: releaseLeft}, {label: "right", release: releaseRight}} {
++		failure := failure
++		go func() {
++			run, err := control.run(failure.label, func() (experiment.Run, error) {
++				callbacksEntered <- struct{}{}
++				<-failure.release
++				return experiment.Run{}, errors.New("failed")
++			})
++			results <- runControlResult{run: run, err: err}
++		}()
++	}
++	awaitRunControl(t, callbacksEntered, "first failing callback")
++	awaitRunControl(t, callbacksEntered, "second failing callback")
++	close(releaseLeft)
++	awaitRunControl(t, hookEntered, "winner failure hook")
++
++	hookHeldMutex := !control.mu.TryLock()
++	if !hookHeldMutex {
++		control.mu.Unlock()
++	}
++	lookupStarted := make(chan struct{})
++	lookupResult := make(chan error, 1)
++	go func() {
++		close(lookupStarted)
++		lookupResult <- control.firstError()
++	}()
++	awaitRunControl(t, lookupStarted, "firstError lookup to start")
++	runtime.Gosched()
++	lookupReturnedEarly := false
++	var stored error
++	select {
++	case stored = <-lookupResult:
++		lookupReturnedEarly = true
++	default:
++	}
++
++	close(releaseRight)
++	close(releaseWinner)
++	first := awaitRunControl(t, results, "first concurrent failure result")
++	second := awaitRunControl(t, results, "second concurrent failure result")
++	if !lookupReturnedEarly {
++		stored = awaitRunControl(t, lookupResult, "blocked firstError lookup")
++	}
++
++	if !hookHeldMutex {
++		t.Fatal("winner hook ran outside the failure mutex")
++	}
++	if lookupReturnedEarly {
++		t.Fatal("firstError returned while winner hook was blocked")
++	}
++	if got := hookCalls.Load(); got != 1 {
++		t.Fatalf("winner hook calls = %d, want exactly 1", got)
++	}
++	if first.err == nil || second.err == nil || first.err.Error() != second.err.Error() {
++		t.Fatalf("concurrent failures = (%v, %v), want one authoritative error", first.err, second.err)
++	}
++	if stored == nil || stored.Error() != first.err.Error() {
++		t.Fatalf("stored first error = %v, want %v", stored, first.err)
++	}
++}
++
++func TestRunControlDiscardsLateSuccessAfterSiblingFailure(t *testing.T) {
++	control := newRunControl(2)
++	successEntered := make(chan struct{})
++	failurePublished := make(chan struct{})
++	control.beforeFailureUnlock = func() { close(failurePublished) }
++
++	successResult := make(chan runControlResult, 1)
++	go func() {
++		run, err := control.run("slow success", func() (experiment.Run, error) {
++			close(successEntered)
++			<-failurePublished
++			return experiment.Run{ID: "late"}, nil
++		})
++		successResult <- runControlResult{run: run, err: err}
++	}()
++	awaitRunControl(t, successEntered, "successful callback to enter")
++
++	failureResult := make(chan runControlResult, 1)
++	go func() {
++		run, err := control.run("sibling", func() (experiment.Run, error) {
++			return experiment.Run{}, errors.New("failed")
++		})
++		failureResult <- runControlResult{run: run, err: err}
++	}()
++
++	failure := awaitRunControl(t, failureResult, "sibling failure")
++	success := awaitRunControl(t, successResult, "late success")
++	if failure.err == nil || failure.err.Error() != "sibling: failed" {
++		t.Fatalf("failure error = %v, want sibling: failed", failure.err)
++	}
++	if !isZeroRun(success.run) || !errors.Is(success.err, errRunAborted) {
++		t.Fatalf("late success = (%+v, %v), want discarded Run and errRunAborted", success.run, success.err)
++	}
++	if strings.Contains(success.err.Error(), "sibling") {
++		t.Fatalf("late success exposed sibling cause instead of abort sentinel: %v", success.err)
++	}
++}
++
++func TestRunControlObservationLinearizesBeforeFailure(t *testing.T) {
++	control := newRunControl(2)
++	observationEntered := make(chan struct{})
++	releaseObservation := make(chan struct{})
++	failureReachedLock := make(chan struct{})
++	failureReturned := make(chan error, 1)
++
++	control.beforeFailureLock = func() { close(failureReachedLock) }
++	observationReturned := make(chan bool, 1)
++	go func() {
++		observationReturned <- control.whileHealthy(func() {
++			close(observationEntered)
++			<-releaseObservation
++		})
++	}()
++	awaitRunControl(t, observationEntered, "observation callback to hold the controller")
++
++	go func() { failureReturned <- control.fail("fold", errors.New("boom")) }()
++	awaitRunControl(t, failureReachedLock, "failure to reach the controller mutex")
++	select {
++	case err := <-failureReturned:
++		t.Fatalf("failure returned while an earlier observation held the controller: %v", err)
++	default:
++	}
++
++	close(releaseObservation)
++	if ok := awaitRunControl(t, observationReturned, "observation to finish"); !ok {
++		t.Fatal("observation admitted before failure was unexpectedly rejected")
++	}
++	if err := awaitRunControl(t, failureReturned, "failure to publish"); err == nil || err.Error() != "fold: boom" {
++		t.Fatalf("failure = %v, want fold: boom", err)
++	}
++
++	called := false
++	if ok := control.whileHealthy(func() { called = true }); ok {
++		t.Fatal("observation after failure publication was admitted")
++	}
++	if called {
++		t.Fatal("rejected observation callback ran")
++	}
++}
+diff --git a/cmd/metis/select_cmd.go b/cmd/metis/select_cmd.go
+index 8bb3f84..50688db 100644
+--- a/cmd/metis/select_cmd.go
++++ b/cmd/metis/select_cmd.go
+@@ -32,6 +32,7 @@ func cmdSelect(args []string) error {
+	promote := fs.Bool("promote", false, "materialize the selected config(s): reconstruct from the ledger + run on ALL data → runs/best-{family}-{hash}/submission.csv; prints the run id(s)")
+	fingerprint := fs.String("fingerprint", "", "restrict to one code-fingerprint (metis#27)")
+	point := fs.String("point", "", "metis#41: publish an OPERATOR-CHOSEN config by ledger row — a point_addr (git-style prefix ok); ships as point-{family}-{hash}. Mutually exclusive with --best/--best-per-model-class")
++	cohort := fs.Bool("cohort", false, "metis#52: list the ledger's code-fingerprint cohorts and exit (the `metis ledger fingerprints` table, on select's surface)")
+	shapePath, flags, err := hoistShapePath(args)
+	if err != nil {
+		return fmt.Errorf("select: %w (usage: metis select <shape.md> [--best | --best-per-model-class | --point ADDR] [--promote] [--fingerprint HASH])", err)
+@@ -39,6 +40,11 @@ func cmdSelect(args []string) error {
+	if err := fs.Parse(flags); err != nil {
+		return err
+	}
++	if *cohort {
++		// metis#52: a listing door where the operator's hands already are — pure
++		// delegation to the #39 core (one implementation, two CLI surfaces).
++		return showFingerprints(shapePath, os.Stdout)
++	}
+	if *point == "" && !*best && !*perClass {
+		*best = true // default view = the single ship recommendation
+	}
+@@ -146,6 +152,12 @@ func runSelect(o selectOpts) error {
+	}
+
+	sort.Slice(picks, func(i, j int) bool { return picks[i].family < picks[j].family })
++	// metis#52: attach each pick's --point handle (the first cohort-filtered ledger row of
++	// that config — any fold row is a valid handle by #41's resolver). Good practice made
++	// mechanical: a concrete "best" config is always shown WITH its override handle.
++	for i := range picks {
++		picks[i].handle = pointHandleFor(led, picks[i].winner.Point)
++	}
+	printSelect(o.out, sh, est, picks, o.perClass)
+
+	if o.promote {
+@@ -161,6 +173,7 @@ type familyPick struct {
+	est    sampler.MeanSE
+	hasEst bool
+	caveat string
++	handle string // metis#52: a representative ledger-row point_addr — the --point override handle
+ }
+
+ // perFamilyConfigWinners runs the shape's metis#19 select rule over the INNER rows (Level != "outer":
+@@ -233,13 +246,28 @@ func printSelect(out io.Writer, sh experiment.Shape, est map[string]sampler.Mean
+	}
+	fmt.Fprintf(out, "  %s:\n", head)
+	for _, p := range picks {
+-		fmt.Fprintf(out, "    %-24s %s\n", famLabel(p.family), freeParamStrFromParams(p.winner.Point.FreeParams))
++		h := ""
++		if p.handle != "" {
++			h = " · point " + short(p.handle) // metis#52: the --point override handle
++		}
++		fmt.Fprintf(out, "    %-24s %s%s\n", famLabel(p.family), freeParamStrFromParams(p.winner.Point.FreeParams), h)
+		if p.caveat != "" {
+			fmt.Fprintf(out, "      caveat: %s\n", p.caveat)
+		}
+	}
+ }
+
++// pointHandleFor finds a representative ledger-row point_addr for a config (first match
++// in append order) — "" when the config has no rows (then no handle is shown; never lie).
++func pointHandleFor(led ledger.Ledger, p shape.Point) string {
++	for _, r := range led.Rows {
++		if freeParamsEqual(p, r.FreeParams) {
++			return r.PointAddr
++		}
++	}
++	return ""
++}
++
+ func famLabel(f string) string {
+	if f == "" {
+		return "(one family)"
+@@ -330,6 +358,8 @@ func promoteSelected(o selectOpts, sh experiment.Shape, picks []familyPick) erro
+			return fmt.Errorf("select --promote %s: %w", famLabel(p.family), err)
+		}
+		runID := "best-" + familyTag(p.family) + "-" + short(addr)
++		// no leafPins: a promoted ship is a SERIAL single all-data fit — multi-threaded
++		// BLAS is wanted here, and one leaf can't oversubscribe (#48's conscious exclusion)
+		ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
+		if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
+			return fmt.Errorf("select --promote %s (%s): %w", famLabel(p.family), runID, err)
+@@ -502,6 +532,8 @@ func runPointSelect(o selectOpts, sh experiment.Shape, led ledger.Ledger, metric
+		return fmt.Errorf("select --point --promote: %w", err)
+	}
+	runID := "point-" + familyTag(fam) + "-" + short(addr)
++	// no leafPins: a promoted ship is a SERIAL single all-data fit — multi-threaded
++	// BLAS is wanted here, and one leaf can't oversubscribe (#48's conscious exclusion)
+	ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
+	if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
+		return fmt.Errorf("select --point --promote (%s): %w", runID, err)
+diff --git a/cmd/metis/select_cmd_test.go b/cmd/metis/select_cmd_test.go
+index 227f6ef..204827e 100644
+--- a/cmd/metis/select_cmd_test.go
++++ b/cmd/metis/select_cmd_test.go
+@@ -1,6 +1,8 @@
+ package main
+
+ import (
++	"bytes"
++	"io"
+	"os"
+	"path/filepath"
+	"strings"
+@@ -528,3 +530,58 @@ func TestSelect_CohortGuardNamesInspectCommand(t *testing.T) {
+		t.Errorf("guard must inline the per-cohort summary, got: %v", err)
+	}
+ }
++
++// metis#52: `metis select <shape> --cohort` lists the fingerprint cohorts — the same
++// table as `metis ledger fingerprints` (one core, a second door on select's surface).
++func TestSelect_CohortFlag(t *testing.T) {
++	shapePath := writeFingerprintFixture(t, t.TempDir())
++	r, w, _ := os.Pipe()
++	orig := os.Stdout
++	os.Stdout = w
++	err := run([]string{"select", shapePath, "--cohort"})
++	_ = w.Close()
++	os.Stdout = orig
++	var buf bytes.Buffer
++	_, _ = io.Copy(&buf, r)
++	if err != nil {
++		t.Fatalf("select --cohort: %v", err)
++	}
++	out := buf.String()
++	for _, want := range []string{"code-fingerprint cohort(s)", "aaaa1111", "bbbb2222", "(legacy)"} {
++		if !strings.Contains(out, want) {
++			t.Errorf("--cohort output missing %q:\n%s", want, out)
++		}
++	}
++}
++
++// metis#52: every concrete config shown as best carries its point handle — and the
++// printed handle round-trips through `select --point`.
++func TestSelect_PickCarriesPointHandle(t *testing.T) {
++	dir := t.TempDir()
++	shapePath := writeSelectLedger(t, dir, taggedShapeForSelect, true)
++	var out strings.Builder
++	if err := runSelect(selectOpts{shapePath: shapePath, best: true, out: &out}); err != nil {
++		t.Fatalf("select: %v", err)
++	}
++	s := out.String()
++	// The logreg winner's handle = the first ledger row of that config (i-lr1-0).
++	if !strings.Contains(s, "point i-lr1-0") {
++		t.Errorf("the ship recommendation must carry its --point handle; got:\n%s", s)
++	}
++	// Round-trip: the printed handle IS a valid --point selector on the same ledger.
++	var out2 strings.Builder
++	if err := runSelect(selectOpts{shapePath: shapePath, point: "i-lr1-0", out: &out2}); err != nil {
++		t.Fatalf("printed handle must round-trip through --point: %v", err)
++	}
++	if !strings.Contains(out2.String(), "train.model=logreg") {
++		t.Errorf("--point round-trip should land on the same config:\n%s", out2.String())
++	}
++	// --best-per-model-class carries handles too.
++	var out3 strings.Builder
++	if err := runSelect(selectOpts{shapePath: shapePath, perClass: true, out: &out3}); err != nil {
++		t.Fatalf("per-class: %v", err)
++	}
++	if got := strings.Count(out3.String(), "point i-"); got < 2 {
++		t.Errorf("per-class winners must each carry a handle, got %d:\n%s", got, out3.String())
++	}
++}
+diff --git a/cmd/metis/shapesweep_test.go b/cmd/metis/shapesweep_test.go
+index ab5b42c..471378e 100644
+--- a/cmd/metis/shapesweep_test.go
++++ b/cmd/metis/shapesweep_test.go
+@@ -332,6 +332,10 @@ func TestShapeSweep_OneConfigDegeneratesToSingleLevelCV(t *testing.T) {
+		if !strings.Contains(final, "folds 2/2") || !strings.Contains(final, "score 0.") {
+			t.Errorf("the flat final progress line must carry folds k/k + score; got: %q", final)
+		}
++		// metis#50: the flat path ends with the same summary block.
++		if !strings.Contains(s, "metis: done in ") || !strings.Contains(s, "metis select ") {
++			t.Errorf("flat run missing the run-end summary:\n%s", s)
++		}
+	}
+	if strings.Contains(out.String(), "nested-CV") {
+		t.Errorf("a 1-config shape must NOT run nested-CV; got:\n%s", out.String())
+@@ -521,3 +525,20 @@ func loadLedgerOrFatal(t *testing.T, expPath string) ledger.Ledger {
+	}
+	return led
+ }
++
++// metis#50: a degraded capture (no fingerprint) degrades the summary honestly —
++// `cohort ?` and NO lying `--fingerprint` pin (a single-cohort ledger needs none).
++func TestPrintRunSummary_DegradedCohort(t *testing.T) {
++	var out strings.Builder
++	printRunSummary(&out, "s.md", 90*time.Second, 42, "")
++	s := out.String()
++	if !strings.Contains(s, "(cohort ?)") {
++		t.Errorf("degraded capture must render cohort ?: %s", s)
++	}
++	if strings.Contains(s, "--fingerprint") {
++		t.Errorf("degraded capture must not emit a lying pin: %s", s)
++	}
++	if !strings.Contains(s, "done in 1m30s") || !strings.Contains(s, "42 rows") || !strings.Contains(s, "metis select s.md") {
++		t.Errorf("summary basics: %s", s)
++	}
++}
+diff --git a/cmd/metis/sweep.go b/cmd/metis/sweep.go
+index 88023f6..12a8504 100644
+--- a/cmd/metis/sweep.go
++++ b/cmd/metis/sweep.go
+@@ -72,9 +72,7 @@ type configScore struct {
+ }
+
+ // shapeSweep is the mutable accumulator the pure nested-Sampler loop folds through the IO
+-// shell: it drives each (config, fold) point-run through the shared cached runner, records
+-// the manifest + per-config estimates, and captures the first fatal error (the pure Run
+-// has no error channel, so a fatal fold sets ss.err and short-circuits the rest).
++// shell. Its runControl is the one failure authority across every nested pass.
+ type shapeSweep struct {
+	o             runOpts
+	sh            experiment.Shape
+@@ -88,14 +86,29 @@ type shapeSweep struct {
+	parallel      bool           // metis#31: >1 max-parallel ⇒ the sweeper/resample/driver batches run via ParExec
+	manMu         sync.Mutex     // metis#32: guards man.Points — concurrent outer folds (ParExec) each record rows
+	prog          *sweepProgress // metis#30: the live-progress sink (nil = silent)
++	start         time.Time      // metis#50: sweep wall-clock start (injected clock)
+ }
+
+ // addManPoints appends a batch of manifest rows under the manifest lock (metis#32: the
+ // nested run's outer folds run concurrently under ParExec, each recording its inner+outer rows).
+-func (ss *shapeSweep) addManPoints(pts []pointRun) {
+-	ss.manMu.Lock()
+-	defer ss.manMu.Unlock()
+-	ss.man.Points = append(ss.man.Points, pts...)
++func (ss *shapeSweep) fail(label string, err error) error {
++	return ss.o.runControl.fail(label, err)
++}
++
++func (ss *shapeSweep) firstError() error {
++	return ss.o.runControl.firstError()
++}
++
++func (ss *shapeSweep) whileHealthy(fn func()) bool {
++	return ss.o.runControl.whileHealthy(fn)
++}
++
++func (ss *shapeSweep) addManPoints(pts []pointRun) bool {
++	return ss.whileHealthy(func() {
++		ss.manMu.Lock()
++		defer ss.manMu.Unlock()
++		ss.man.Points = append(ss.man.Points, pts...)
++	})
+ }
+
+ // sweepPass accumulates ONE black-box sweeper run (the sweeper ⊃ resample loop): its per-config
+@@ -115,44 +128,37 @@ type sweepPass struct {
+	hooks    passHooks            // metis#30: this pass's progress hooks, closure-bound to its outer fold
+	// metis#31: under ParExec the sweeper fans out over configs and each config's
+	// resample fans out over folds — all appending to this ONE pass. `mu` guards the
+-	// orchestration bookkeeping (configs/points/err); the honest reduce stays in the
++	// orchestration bookkeeping (configs/points); the honest reduce stays in the
+	// sampler's pure Tell/Done, not here.
+	mu      sync.Mutex
+	configs []configScore
+	points  []pointRun
+-	err     error
+ }
+
+-// setErr records the FIRST fatal error of the pass (set-once, mutex-guarded — the
+-// fan-out writes it concurrently). A no-op once an error is already latched.
+-func (p *sweepPass) setErr(err error) {
+-	p.mu.Lock()
+-	defer p.mu.Unlock()
+-	if p.err == nil {
+-		p.err = err
+-	}
++func (p *sweepPass) setErr(label string, err error) error {
++	return p.ss.fail(label, err)
+ }
+
+-// firstError returns the latched error (mutex-guarded read — a concurrent read+write
+-// is a race even when the write is set-once).
+ func (p *sweepPass) firstError() error {
+-	p.mu.Lock()
+-	defer p.mu.Unlock()
+-	return p.err
++	return p.ss.firstError()
+ }
+
+ // addConfigScore / addPoint append the per-config estimate / per-fold row under the
+ // pass lock (concurrent under ParExec).
+-func (p *sweepPass) addConfigScore(cs configScore) {
+-	p.mu.Lock()
+-	defer p.mu.Unlock()
+-	p.configs = append(p.configs, cs)
++func (p *sweepPass) addConfigScore(cs configScore) bool {
++	return p.ss.whileHealthy(func() {
++		p.mu.Lock()
++		defer p.mu.Unlock()
++		p.configs = append(p.configs, cs)
++	})
+ }
+
+-func (p *sweepPass) addPoint(pr pointRun) {
+-	p.mu.Lock()
+-	defer p.mu.Unlock()
+-	p.points = append(p.points, pr)
++func (p *sweepPass) addPoint(pr pointRun) bool {
++	return p.ss.whileHealthy(func() {
++		p.mu.Lock()
++		defer p.mu.Unlock()
++		p.points = append(p.points, pr)
++	})
+ }
+
+ // runSweeper runs the black-box sweeper (GridConfigs ⊃ FixedKFolds) over configPts, folding each
+@@ -165,11 +171,15 @@ func (ss *shapeSweep) runSweeper(ctx sampler.Ctx, configPts []shape.Point, pass
+		func(c shape.Point) sampler.MeanSE {
+			ms := sampler.Run(ctx, sampler.FixedKFolds{K: pass.splitK},
+				func(f sampler.FoldPoint) sampler.FoldOutcome { return pass.runPipelineFold(c, f) },
+-				sampler.ExecFor[sampler.FoldPoint, sampler.FoldOutcome](ss.parallel), pass.hooks.fold)
++				sampler.ExecFor[sampler.FoldPoint, sampler.FoldOutcome](ss.parallel), func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
++					ss.whileHealthy(func() { pass.hooks.fold(ev) })
++				})
+			pass.addConfigScore(configScore{point: c, meanSE: ms})
+			return ms
+		},
+-		sampler.ExecFor[shape.Point, sampler.MeanSE](ss.parallel), pass.hooks.config)
++		sampler.ExecFor[shape.Point, sampler.MeanSE](ss.parallel), func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
++			ss.whileHealthy(func() { pass.hooks.config(ev) })
++		})
+ }
+
+ // runShapeSweep drives the metis#18 nested Sampler loop: the sweeper (GridConfigs over the
+@@ -179,7 +189,8 @@ func (ss *shapeSweep) runSweeper(ctx sampler.Ctx, configPts []shape.Point, pass
+ // the winner by the objective; driver:single ships it (M1a-5). Produces per-config (mean,SE)
+ // + the manifest + the raw per-fold ledger. Per-fold failure is fatal to the sweep (surfaced,
+ // not swallowed — a partial resample is not an honest estimate).
+-func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) error {
++func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) (result error) {
++	sweepStart := now() // metis#50: the run-end summary reports wall-clock elapsed
+	// probeRepo's HEAD sha still drives the mid-sweep code-freeze guard (codeID) — NOT the
+	// identity (metis#27 dropped repo_shas). The shape's blob-hash content-addresses the intent.
+	_, sha, _ := probeRepo(o.git, filepath.Dir(o.expPath))
+@@ -243,7 +254,7 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
+		return err
+	}
+	ss := &shapeSweep{
+-		o: o, sh: sh, now: now, out: out, shapeBlobHash: sbh, codeID: sha,
++		o: o, sh: sh, now: now, out: out, shapeBlobHash: sbh, codeID: sha, start: sweepStart,
+		partRef:  partitionRef(sh),
+		man:      sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweeper.Sampler, Seed: sh.Seed},
+		parallel: o.maxParallel > 1, // metis#31: fan out the sweeper/resample/driver batches
+@@ -260,19 +271,41 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
+		ss.prog.bw = o.board
+		ss.prog.width = boardWidth()
+		ss.prog.gauge = o.leafGauge
+-		ticker := time.NewTicker(500 * time.Millisecond)
++		tickC := o.boardTick
++		var ticker *time.Ticker
++		if tickC == nil {
++			ticker = time.NewTicker(500 * time.Millisecond)
++			tickC = ticker.C
++		}
+		tickDone := make(chan struct{})
++		tickStopped := make(chan struct{})
+		go func() {
++			defer close(tickStopped)
+			for {
+				select {
+-				case <-ticker.C:
+-					ss.prog.tick()
++				case <-tickC:
++					if o.beforeBoardTick != nil {
++						o.beforeBoardTick()
++					}
++					ss.whileHealthy(func() { ss.prog.tick() })
++					if o.afterBoardTick != nil {
++						o.afterBoardTick()
++					}
+				case <-tickDone:
+					return
+				}
+			}
+		}()
+-		defer func() { ticker.Stop(); close(tickDone) }()
++		defer func() {
++			if ticker != nil {
++				ticker.Stop()
++			}
++			close(tickDone)
++			<-tickStopped
++			if result != nil {
++				ss.prog.abort()
++			}
++		}()
+	}
+
+	// metis#32: >1 config → nested CV (records inner + per-family outer rows; the honest measure).
+@@ -291,7 +324,10 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
+	res := sampler.Run(ctx, sampler.SingleDriver{}, func(sampler.SinglePoint) sampler.SweepResult {
+		return ss.runSweeper(ctx, configPts, pass)
+	}, sampler.ExecFor[sampler.SinglePoint, sampler.SweepResult](ss.parallel), nil)
+-	ss.prog.finish() // metis#30: the terminal progress line, before the report
++	if err := ss.firstError(); err != nil {
++		return err
++	}
++	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the report
+	// metis#31: sort the fan-out's completion-order bookkeeping to a stable content key
+	// BEFORE persisting, so manifest.json + the ledger are byte-deterministic across
+	// serial/parallel runs (the winner/estimate are already deterministic; this makes
+@@ -299,23 +335,22 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
+	sortPointRuns(pass.points)
+	ss.man.Points = pass.points
+	ss.configs = pass.configs
+-	if err := pass.firstError(); err != nil {
+-		return err
+-	}
+-
+	if err := writeManifest(o.expPath, ss.man); err != nil {
+-		return err
++		return ss.fail("write sweep manifest", err)
+	}
+	// Capture the sweep's code closure to a git side ref (metis#8/#14) — BEST-EFFORT: the
+	// records + manifest are already valid, so a capture hiccup warns, never aborts.
+-	if err := captureSweepCode(o, ss.man); err != nil {
+-		fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
++	cohort, err := captureSweepCode(o, ss.man)
++	if err != nil {
++		ss.whileHealthy(func() {
++			fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
++		})
+	}
+	// Persist the raw per-fold rows to the shape's append-only ledger sidecar (metis#8/#18):
+	// AggregateView reduces them read-time to per-config (mean, SE) — so metis#19's 1-SE
+	// select re-reduces the same rows without a re-run.
+	if err := writeSweepLedger(o.expPath, ss.man); err != nil {
+-		return err
++		return ss.fail("write sweep ledger", err)
+	}
+	// Guard (metis#19): a parsimony rule (one-std-err/pct-loss) needs a measured complexity
+	// for every swept family — else the parsimony axis is silently dropped and the winner is
+@@ -323,10 +358,13 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
+	// the ship/report is gated. Checked here (post-fold) because HasComplexity is only known
+	// after the folds run.
+	if err := sampler.GuardComplexity(sh.Sweeper.Objective.Select, configStatsOf(ss.configs)); err != nil {
+-		return err
++		return ss.fail("sweep complexity guard", err)
+	}
+-	ss.reportWinner(res)
+-	return nil
++	ss.whileHealthy(func() {
++		ss.reportWinner(res)
++		printRunSummary(out, o.expPath, now().Sub(sweepStart), len(ss.man.Points), cohort)
++	})
++	return ss.firstError()
+ }
+
+ // configStatsOf builds the per-config stats (with each config's family) from a completed
+@@ -368,54 +406,38 @@ func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, r
+	// fewer of them (runFolds ≤ k). Deterministic run id → the analysis_i refs are locatable.
+	analysisRefs, err := ss.materializeOuterAnalysis(k, stratify)
+	if err != nil {
+-		return err
++		if first := ss.firstError(); first != nil {
++			return first
++		}
++		return ss.fail("nested-CV preamble", err)
+	}
+	outerPart := sampler.PartitionRef(fmt.Sprintf("outer-cv-k%d-strat%t-seed%d", k, stratify, ss.sh.Seed))
+
+-	// metis#31: CVDriver.Ask emits all outer folds as one batch, so ParExec runs these
+-	// closures concurrently — firstErr's read+write must be mutex-guarded (a set-once
+-	// write racing a read is still a data race). Each outer fold has its OWN sweepPass
+-	// (created in runOuterFold), so the inner fan-out is guarded there; this guards only
+-	// the outer-fold error latch. The per-fold ledger-row recording is guarded by ss.manMu.
+-	var errMu sync.Mutex
+-	var firstErr error
+-	setFirst := func(err error) {
+-		errMu.Lock()
+-		defer errMu.Unlock()
+-		if firstErr == nil {
+-			firstErr = err
+-		}
+-	}
+-	getFirst := func() error {
+-		errMu.Lock()
+-		defer errMu.Unlock()
+-		return firstErr
+-	}
+	est := sampler.Run(ctx, sampler.CVDriver{K: runFolds, Stratify: stratify},
+		func(p sampler.OuterFoldPoint) float64 {
+-			if getFirst() != nil {
++			if ss.firstError() != nil {
+				return 0
+			}
+			score, ferr := ss.runOuterFold(ctx, configPts, k, stratify, analysisRefs[p.Idx], outerPart, p.Idx)
+			if ferr != nil {
+-				setFirst(ferr)
++				if ss.firstError() == nil {
++					ss.fail(fmt.Sprintf("outer fold %d", p.Idx), ferr)
++				}
+				return 0
+			}
+			return score
+		},
+		sampler.ExecFor[sampler.OuterFoldPoint, float64](ss.parallel),
+-		// metis#30: outer-fold completions always emit. Error-gated: once firstErr
++		// metis#30: outer-fold completions always emit. Error-gated: once runControl
+		// latches, remaining closures return sentinel zeros — don't fold those into
+		// the displayed est (the run is aborting; a fake 0 would tank the line).
+		func(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
+-			if getFirst() == nil {
+-				ss.prog.driverEvent(ev)
+-			}
++			ss.whileHealthy(func() { ss.prog.driverEvent(ev) })
+		})
+-	if err := getFirst(); err != nil {
++	if err := ss.firstError(); err != nil {
+		return err
+	}
+-	ss.prog.finish() // metis#30: the terminal progress line, before the estimate report
++	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the estimate report
+
+	// metis#32: the nested run now RECORDS (unlike metis#23's estimation-only path) — persist the
+	// inner + per-family outer rows accumulated in ss.man.Points so `metis select` can reduce them
+@@ -423,16 +445,22 @@ func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, r
+	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
+	sortPointRuns(ss.man.Points)
+	if err := writeManifest(ss.o.expPath, ss.man); err != nil {
+-		return err
++		return ss.fail("write nested sweep manifest", err)
+	}
+-	if err := captureSweepCode(ss.o, ss.man); err != nil {
+-		fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", err)
++	cohort, cerr := captureSweepCode(ss.o, ss.man)
++	if cerr != nil {
++		ss.whileHealthy(func() {
++			fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
++		})
+	}
+	if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
+-		return err
++		return ss.fail("write nested sweep ledger", err)
+	}
+-	ss.reportEstimate(est, runFolds)
+-	return nil
++	ss.whileHealthy(func() {
++		ss.reportEstimate(est, runFolds)
++		printRunSummary(ss.out, ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
++	})
++	return ss.firstError()
+ }
+
+ // materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
+@@ -451,13 +479,14 @@ func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]str
+	exp.Type = "experiment"
+	preID, err := pointAddressOf(exp, ss.shapeBlobHash)
+	if err != nil {
+-		return nil, fmt.Errorf("nested-CV preamble: %w", err)
++		return nil, ss.fail("nested-CV preamble address", err)
+	}
+	preOpts := ss.o
+	preOpts.inSweep = true // one preamble run; skip the per-run capture noise
+	preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
++	preOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
+	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
+-		return nil, fmt.Errorf("nested-CV preamble (%s): %w", preID, err)
++		return nil, err
+	}
+	refs := make([]string, outerK)
+	for i := 0; i < outerK; i++ {
+@@ -473,7 +502,7 @@ func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]str
+ func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k int, stratify bool, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
+	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
+	if err != nil {
+-		return 0, err
++		return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
+	}
+	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
+	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
+@@ -481,22 +510,26 @@ func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k i
+		hooks: ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
+	sres := ss.runSweeper(ctx, configPts, pass)
+	if err := pass.firstError(); err != nil {
+-		return 0, fmt.Errorf("outer fold %d sealed sweep: %w", i, err)
++		return 0, err
+	}
+	// Guard (metis#19/#23 I1): the parsimony select rule needs a measured complexity for every
+	// swept family — same guard the flat path runs before trusting its winner. Without it, a
+	// parsimony-select + non-reporting-model shape would SILENTLY mis-select in each outer fold.
+	if err := sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs)); err != nil {
+-		return 0, fmt.Errorf("outer fold %d: %w", i, err)
++		return 0, ss.fail(fmt.Sprintf("outer fold %d complexity guard", i), err)
+	}
+
+	// metis#32: record the sealed sweep's INNER rows (Level=inner, tagged with this outer fold).
+	of := i
+	rows := make([]pointRun, 0, len(pass.points)+len(sres.PerFamily))
+-	for _, pr := range pass.points {
+-		pr.Level = "inner"
+-		pr.OuterFold = &of
+-		rows = append(rows, pr)
++	if !ss.whileHealthy(func() {
++		for _, pr := range pass.points {
++			pr.Level = "inner"
++			pr.OuterFold = &of
++			rows = append(rows, pr)
++		}
++	}) {
++		return 0, errRunAborted
+	}
+
+	// (b) score EACH family's inner-winner on the held outer-assessment — post-selection, so
+@@ -510,44 +543,51 @@ func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k i
+	var shipScore float64
+	for _, fam := range sortedFamilies(sres.PerFamily) {
+		w := sres.PerFamily[fam]
+-		score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart)
++		score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart, fam)
+		if ferr != nil {
+-			return 0, fmt.Errorf("outer fold %d family %s score: %w", i, fam, ferr)
++			return 0, ferr
+		}
+-		rows = append(rows, pointRun{
+-			RunID:      scoreID,
+-			FreeParams: freeParamMap(w.Point),
+-			Fold:       of, // the outer fold this held-out score is on
+-			Level:      "outer",
+-			OuterFold:  &of,
+-			Status:     status,
+-			// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
+-		})
+-		if fam == shipFamily {
+-			shipScore = score
++		if !ss.whileHealthy(func() {
++			rows = append(rows, pointRun{
++				RunID:      scoreID,
++				FreeParams: freeParamMap(w.Point),
++				Fold:       of, // the outer fold this held-out score is on
++				Level:      "outer",
++				OuterFold:  &of,
++				Status:     status,
++				// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
++			})
++			if fam == shipFamily {
++				shipScore = score
++			}
++			fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
++				i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
++		}) {
++			return 0, errRunAborted
+		}
+-		fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
+-			i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
+	}
+-	ss.addManPoints(rows)
++	if !ss.addManPoints(rows) {
++		return 0, errRunAborted
++	}
+	return shipScore, nil
+ }
+
+ // scoreOnOuterFold refit-and-scores one config's winner on the held outer-assessment fold i (a
+ // full-data fold run at outer-k; post-selection, so unconfined). Returns the held-out fold_score,
+ // the run id (→ its record.json carries the namespaced metric the ledger reads), and its status.
+-func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef) (float64, string, string, error) {
++func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef, fam string) (float64, string, string, error) {
+	scoreExp := ss.buildFoldExperiment(point, sampler.FoldPoint{Idx: i}, nil, k, stratify, outerPart)
+	scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
+	if err != nil {
+-		return 0, "", "", err
++		return 0, "", "", ss.fail(fmt.Sprintf("outer fold %d family %s score address", i, fam), err)
+	}
+	scoreOpts := ss.o
+	scoreOpts.inSweep = true
+	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
++	scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
+	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
+	if err != nil {
+-		return 0, "", "", fmt.Errorf("%s: %w", scoreID, err)
++		return 0, "", "", err
+	}
+	return run.Metrics[foldMetric], scoreID, run.Status, nil
+ }
+@@ -574,8 +614,9 @@ func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
+ // runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
+ // synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
+ // the shared cached runner, record the manifest row, and return the fold_score the inner
+-// resample Sampler folds. A fatal outcome sets ss.err and returns 0 (the pure Run keeps
+-// going; runShapeSweep checks ss.err before using the winner).
++// resample Sampler folds. A fatal outcome publishes through the experiment-wide
++// runControl and returns 0; every sampler callback/sink rejects placeholders after
++// publication, and the top level returns the stored concrete cause.
+ func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.FoldOutcome {
+	ss := p.ss
+	if p.firstError() != nil {
+@@ -589,34 +630,39 @@ func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.
+	// .git/index.lock so a transient probe failure is expected; treating "" as a change
+	// would false-abort the whole honest run.
+	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
+-		p.setErr(fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
++		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx),
++			fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
+		return sampler.FoldOutcome{}
+	}
+
+	exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.stratify, p.partRef)
+	runID, err := pointAddressOf(exp, ss.shapeBlobHash)
+	if err != nil {
+-		p.setErr(fmt.Errorf("config %s fold %d: %w", freeParamStr(c), f.Idx, err))
++		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx), err)
+		return sampler.FoldOutcome{}
+	}
+	pointOpts := ss.o
+	pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
+	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
++	pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
+	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
+	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
+	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
+	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
+	if runErr != nil {
+-		p.setErr(fmt.Errorf("config %s fold %d (%s): %w", freeParamStr(c), f.Idx, runID, runErr))
++		// runControl already published a concrete admitted-run failure. A queued or
++		// late sibling returns errRunAborted; neither path may republish the sentinel.
+		return sampler.FoldOutcome{}
+	}
+-	p.addPoint(pointRun{
++	if !p.addPoint(pointRun{
+		RunID:      runID,
+		FreeParams: freeParamMap(c),
+		Fold:       f.Idx,
+		Status:     run.Status,
+		Metrics:    run.Metrics,
+-	})
++	}) {
++		return sampler.FoldOutcome{}
++	}
+	// metis#19 M2: read the train step's realized-complexity metric. Present → the parsimony
+	// rules consume it; absent (HasComplexity false) → the guard rejects a parsimony rule.
+	cx, hasCx := run.Metrics[foldComplexityMetric]
+@@ -886,3 +932,24 @@ func freeParamStrFromParams(fps []shape.FreeParam) string {
+	}
+	return s
+ }
++
++// printRunSummary is metis#50's run-end handoff: elapsed wall-clock, what landed where,
++// and the paste-ready follow-up commands with the cohort fingerprint pre-filled — the
++// operator should never scrape the scrollback to assemble a `metis select`. A degraded
++// capture (no fingerprint) degrades honestly: `cohort ?` and un-pinned hints (a
++// single-cohort ledger needs no pin).
++func printRunSummary(out io.Writer, expPath string, elapsed time.Duration, rows int, cohort record.Hash) {
++	base := filepath.Base(ledgerPath(expPath))
++	if cohort == "" {
++		fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort ?)\n", fmtETA(elapsed), rows, base)
++		fmt.Fprintf(out, "  next: metis select %s\n", expPath)
++		fmt.Fprintf(out, "        metis select %s --best --promote\n", expPath)
++		fmt.Fprintf(out, "        metis ledger fingerprints %s\n", expPath)
++		return
++	}
++	fp := short(string(cohort))
++	fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort %s)\n", fmtETA(elapsed), rows, base, fp)
++	fmt.Fprintf(out, "  next: metis select %s --fingerprint %s               # the honest pick\n", expPath, fp)
++	fmt.Fprintf(out, "        metis select %s --fingerprint %s --best --promote   # materialize it\n", expPath, fp)
++	fmt.Fprintf(out, "        metis ledger fingerprints %s                   # cohorts\n", expPath)
++}
+diff --git a/testdata/experiment/toy-sweep-smoke.md b/testdata/experiment/toy-sweep-smoke.md
+new file mode 100644
+index 0000000..bc2e52f
+--- /dev/null
++++ b/testdata/experiment/toy-sweep-smoke.md
+@@ -0,0 +1,27 @@
++---
++type: experiment-shape
++id: toy-sweep-smoke
++seed: 42
++status: active
++data:
++  - id: data
++    uses: test/echo
++    with: {out: ../dataset/toy}
++pipeline:
++  - id: train
++    uses: metis/train
++    needs: [data]
++    with:
++      dataset: ../dataset/toy
++      model:
++        $any:
++          logreg: {C: {$any: [0.5, 1.0, 2.0]}}
++sweeper:
++  sampler: grid
++  resample: {cv: {k: 2, stratify: true}}
++  objective: {metric: train.fold_score, direction: maximize, select: {argmax-mean: {}}}
++---
++
++# toy-sweep-smoke
++
++A credential-free, disposable real-process nested sweep for cold scheduling smoke checks.
+diff --git a/testdata/steps/test/env-dump b/testdata/steps/test/env-dump
+index 6f4b8c5..716fa69 100755
+--- a/testdata/steps/test/env-dump
++++ b/testdata/steps/test/env-dump
+@@ -11,4 +11,9 @@ d="${METIS_STEP_DIR:-$PWD}"
+   echo "EXP_DIR=$METIS_EXP_DIR"
+   echo "SEED=$METIS_SEED"
+   echo "READ_ROOT=${METIS_READ_ROOT:-<unset>}"
++  # metis#48: the four leaf BLAS pins (full names — asserted verbatim by the seam test)
++  echo "OMP_NUM_THREADS=${OMP_NUM_THREADS:-<unset>}"
++  echo "OPENBLAS_NUM_THREADS=${OPENBLAS_NUM_THREADS:-<unset>}"
++  echo "VECLIB_MAXIMUM_THREADS=${VECLIB_MAXIMUM_THREADS:-<unset>}"
++  echo "MKL_NUM_THREADS=${MKL_NUM_THREADS:-<unset>}"
+ } > "$d/env.txt"
+diff --git a/workshop/lessons.md b/workshop/lessons.md
+index baa18a9..a137592 100644
+--- a/workshop/lessons.md
++++ b/workshop/lessons.md
+@@ -158,12 +158,12 @@ finished in ~4 min — the run looked alive while making no progress. The `--par
+ documents exactly this caveat; it still shipped as the default behavior on a real sweep. Relaunch
+ with `OMP_NUM_THREADS=1 …=1 --parallel 8` → load ~21, ~107 trains/min, done in ~28 min.
+
+-**Rule:** for a real (subprocess-leaf) sweep, ALWAYS pin the leaf's thread env
+-(`OMP_NUM_THREADS=1 OPENBLAS_NUM_THREADS=1 VECLIB_MAXIMUM_THREADS=1 MKL_NUM_THREADS=1`) and cap
+-`--parallel` below core count. Diagnostic signature of the thrash: starts ≫ completions with the
+-process alive (throughput ≈ 0) — which is also why the #38 progress board needs a moving-average
+-runs/sec line, not just liveness. Deeper fix candidate: metis could set single-thread BLAS env for
+-its leaf subprocesses BY DEFAULT (the parallelism budget belongs to the orchestrator, not the leaf).
++**Rule:** for a real (subprocess-leaf) sweep, the leaf's thread env must be pinned and
++`--parallel` capped below core count. **RESOLVED BY DEFAULT since metis#48** — bare `metis run`
++now injects the four pins at both spawn seams (export a `*_NUM_THREADS` value to override); the
++rule survives for non-metis contexts and as the WHY behind the default. Diagnostic signature of
++the thrash: starts ≫ completions with the process alive (throughput ≈ 0) — which is also why the
++#38 progress board needs a moving-average runs/sec line, not just liveness.
+
+ ## Plan-sketch folds: set-cardinality, not incremental counts (metis#39 plan review)
+ - **A "keep the latest, count the others" fold specified incrementally silently overcounts under non-monotone input — specify it as set-cardinality (`len(set)-1`) with the latest tracked separately.** The #39 plan's ExtraCommits sketch counted displacement transitions per ROW; interleaved-timestamp records (two concurrent sweeps, same fingerprint) would have inflated it row-for-row, and the plan's own happy-path fixture (2 records, monotone) structurally couldn't catch it — add an out-of-order fixture whenever a fold's correctness depends on input order. metis-specific ground truth: **ledger rows are NOT time-ordered** (`sortPointRuns` orders by content key; append order is sweep-completion order).
+@@ -179,3 +179,9 @@ its leaf subprocesses BY DEFAULT (the parallelism budget belongs to the orchestr
+ - **Writer identity is temporal, not call-graph.** When a plan claims "all output routes through one wrapper," audit every construction-time capture of the underlying writer (pools, closures, structs built earlier in the call chain) — a component that grabbed the writer BEFORE the wrap exists is an invisible bypass. Grep the writer variable at every `new*(out)` site, not just Fprintf sites. (The forkserver pool + captureSweepCode's o.out both captured pre-board writers.)
+ - **"Ticker calls repaint()" is a deadlock-or-staleness smell.** In state-owner + painter designs, fix ONE global lock order (state.mu → painter.mu) and route timers through the state owner; a painter-first timer either inverts locks via a state callback or repaints a frame that can't refresh time-derived values (ETA/rate decay).
+ - **A stdlib-only TUI plan must name its terminal-size mechanism explicitly.** Width detection is the one capability ANSI pin-bottom genuinely needs beyond stdlib's comfortable surface, and a wrong width isn't cosmetic — a wrapped line breaks the cursor-up erase-count bookkeeping the whole repaint scheme rests on.
++
++## Plan-review lessons (metis#48 plan)
++- **A constructor-grep is not a coverage proof — also grep direct callers of the layer BELOW the wiring point.** Wiring computed in an entry function (`runExperiment`) silently misses call paths that enter beneath it (select_cmd.go builds fresh `runOpts` and calls `runResolvedExperiment` directly). When a plan claims "every production construction is threaded," check the constructor sites AND downstream-function callers; every bypass found is either threaded or documented as a conscious exclusion.
++- **Doc-consistency sweeps must include Go sources, not just `*.md`** — operator guidance lives in flag `--help` strings and load-bearing comments (main.go's `--parallel` help told operators to hand-pin BLAS).
++- **A plan's inline test code must be written against the VERIFIED fixture/format; promote any fixture gap to an explicit numbered step.** A parenthetical "check the fixture" hedge next to code that contradicts it breaks the TDD red-green sequence and invites wrong-reason debugging (env-dump dumps METIS_* only; experiment steps live in YAML frontmatter, not a fenced block).
++- **A cross-repo deliverable is invisible to the closing repo's review window — pin the peer repository + exact commit in the issue Log before close.** A checked plan row and prose saying “RUNBOOK updated” are not independently traceable when the actual diff lives in kbench. Record the peer commit as soon as it lands so the boundary reviewer can verify the requirement without trusting the implementor's assertion.
+diff --git a/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md b/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md
+new file mode 100644
+index 0000000..321f7f5
+--- /dev/null
++++ b/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md
+@@ -0,0 +1,202 @@
++# Board Readability: Typed Activity, Cold-Phase Truth, and Stable ETA — Implementation Plan
++
++> **For the implementer:** Execute this plan with `superpowers-executing-plans`; use TDD for every
++> behavior change. The issue has one delivery boundary, so do not add milestone tags or run
++> `sdlc milestone-close`; cross the mandatory fresh-context review once with `sdlc close`.
++
++**Goal:** Make the sweep board explain cold work truthfully, use unambiguous flat/nested vocabulary,
++and withhold rate/ETA until time-based evidence is stable without weakening stall visibility.
++
++**Architecture:** A typed activity emitter connects two concrete facts to `sweepProgress`: successful
++final-executor steps (outside cache) and successfully executed, durably persisted concrete runs.
++`sweepProgress` is the sole synchronized reducer; pure bounded windows derive smoothed occupancy,
++event-time rate readiness, decay, and last-run age. `progressCore` remains the shared semantic source
++for plain and TUI output, while `renderBoard` only formats snapshots. Activity publication is gated by
++`runControl` before taking the progress mutex, preserving the established controller → progress →
++writer lock order and preventing post-failure repaint (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).
++
++**Tech stack:** Go, standard library, existing metis run/sweep/progress/board abstractions, Go tests,
++and the kbench Markdown RUNBOOK.
++
++## Core concepts
++
++### PURE entities and transforms
++
++| Concept | Responsibility | Invariants |
++|---|---|---|
++| `activityEvent` | Immutable successful activity fact with kind, typed run role/identity, and injected-clock timestamp. | Failed work creates no event; timestamps describe completion, not callback delivery. |
++| `runRole` | Distinguish nested inner-CV, flat CV, preamble, outer score, and ineligible/no-role runs. | Only inner-CV and flat CV are rate/counter eligible. |
++| `occupancyWindow` | Retain the last four 500ms occupancy samples and return their rounded mean. | Event count cannot affect the result; capacity is four. |
++| `movingRate` | Retain the latest 64 eligible completion times in event-time order and derive readiness/rate from `now`. | Ready only at n≥16 and span≥15s; rate is `(n-1)/(now-oldest)`; reversed delivery is deterministic. |
++| `activitySnapshot` | Read-only facts consumed by formatting: steps, max step time, eligible runs, max run time, smoothed slots, and optional rate. | Last times are maxima; startup ends on the first eligible run. |
++
++### INTEGRATION boundaries
++
++| Boundary | Responsibility | Failure semantics |
++|---|---|---|
++| `activityExecutor` | Decorate the final cache-aware executor and emit one step event after a successful real execution or cache hit. | Inner error is returned unchanged and emits nothing. |
++| `runResolvedExperiment` activity publication | Emit the typed run event only after execution and required `runs/<id>/{run,record}.json` persistence succeed. | Execution failure, `run.json`/`record.json` failure, or provenance assembly failure emits no successful-run event; best-effort capture is not a success gate. |
++| `runControl`-gated emitter | Linearize all step and run activity against fatal failure before calling `sweepProgress`. | Rejected after failure; never acquire controller state while holding progress state. |
++| `sweepProgress` | Synchronize activity/tick reduction and publish immutable board snapshots. | Short callbacks; non-sweep callers receive a no-op emitter. |
++| `renderBoard` / `progressCore` | Apply shared vocabulary and factual startup/mature wording to snapshots. | No diagnosis such as “not hung”; width, cadence, failure flush, and terminal cleanup remain intact. |
++| kbench RUNBOOK | Document the shipped board contract using the exact operator-facing nouns. | Full peer commit SHA is recorded in issue #49 before close. |
++
++## Chunk 1: Typed activity at concrete success seams
++
++### Task 1: Define activity facts and the final-executor decorator
++
++**Files:**
++- Modify: `cmd/metis/run.go`
++- Create or modify: `cmd/metis/activity_test.go`
++- Modify: `cmd/metis/caching_test.go`
++
++1. Write failing table tests proving the decorator emits exactly one timestamped step-success event
++   after a successful inner executor, emits none on error, and preserves the exact result/error.
++2. Add a cache wiring regression: one cold execution and its warm cache hit each produce one step
++   event. Assert the decorator is outside the cache wrapper, not merely that the cold path works.
++3. Introduce the smallest typed `activityEvent`/kind/role vocabulary and a no-op-capable emitter.
++   Implement `activityExecutor` around the final executor built in `runResolvedExperimentAdmitted`.
++   It must publish through the same `runControl`-gated emitter used by successful-run events, so a
++   late successful step callback after sibling failure is rejected before progress repaint.
++4. Run the focused tests and keep identities/timestamps injected; do not parse step output or inspect
++   cache implementation details.
++
++### Task 2: Publish successful concrete-run events at the persistence boundary
++
++**Files:**
++- Modify: `cmd/metis/run.go`
++- Modify: `cmd/metis/run_test.go`
++
++1. Extend the successful-run test with an activity callback that observes both required
++   `runs/<id>/run.json` and `runs/<id>/record.json` artifacts already persisted when the event arrives.
++2. Pin negative paths: a failed execution that successfully writes its failure record emits no run
++   event, and a forced required-persistence failure (for example, a directory at the record path)
++   emits no run event.
++3. Add the run role to `runOpts` and publish only after `runErr == nil` plus required persistence.
++   Preserve best-effort capture behavior: capture failure must not retroactively make a successful
++   run ineligible unless the current contract already treats that artifact as required.
++4. Route successful-run publication through the shared `runControl`-gated emitter before the progress
++   callback. Add barrier regressions showing a sibling fatal failure prevents both a later step event
++   and a later run event from repainting without introducing controller↔progress lock inversion.
++
++### Task 3: Assign roles at every sweep call site
++
++**Files:**
++- Modify: `cmd/metis/sweep.go`
++- Modify: `cmd/metis/run.go`
++- Modify: `cmd/metis/run_test.go`
++
++1. Write a call-site trace test that distinguishes flat CV, nested inner-CV, nested preamble, and
++   outer-score runs and proves only the first two are eligibility candidates.
++2. Add `sweepPass.runRole`; set it at flat pass construction and nested pass construction, then copy
++   it into `pointOpts.runRole` immediately before `runPipelineFold` calls `runResolvedExperiment`.
++   Assign preamble and outer-score roles at their direct launch sites. Keep plain `metis run` and
++   `metis select --promote` explicitly no-op/ineligible; document the bypass rather than silently
++   relying on a zero value.
++3. Assert emitted run-event roles from the concrete call paths, not just enum eligibility. Run focused
++   tests plus `go test ./cmd/metis -run 'Activity|Cache|RunResolved|RunControl' -race`.
++
++## Chunk 2: Deterministic telemetry reduction and board semantics
++
++### Task 4: Replace callback-count rate sampling with event-time reduction
++
++**Files:**
++- Modify: `cmd/metis/progress.go`
++- Modify: `cmd/metis/progress_test.go`
++
++1. Write failing pure tests for eligible/ineligible roles, max last-step/run timestamps, and reversed
++   callback delivery. Feed 65 shuffled completions and prove the latest 64 by event time survive.
++2. Pin readiness boundaries: 15 events are unready; 16 spanning under 15s are unready; 16 spanning
++   exactly 15s are ready. Assert `(n-1)/(now-oldest)`, including a `now` later than the newest event.
++3. Add a mature trace followed by five 1s ticks: last-run age advances five seconds, numeric rate is
++   non-increasing, and ETA is non-decreasing. Then add completions and prove gradual 64-event-window
++   recovery rather than a one-interval snap.
++4. Refactor `movingRate` into a sorted, bounded event-time window. Reduce typed eligible run events
++   under the existing progress mutex and remove fold-callback-time rate mutation.
++5. Move the aggregate displayed `inner-CV runs` / `CV runs` counter to typed eligible run-completion
++   events. Add a reversed-delivery regression proving typed events alone advance the aggregate
++   counter/rate/ETA, while sampler fold callbacks retain only score and per-row duties and cannot
++   double-count or lag the board counter.
++
++### Task 5: Make occupancy tick-driven and event-density independent
++
++**Files:**
++- Modify: `cmd/metis/progress.go`
++- Modify: `cmd/metis/progress_test.go`
++
++1. Add a pure four-sample test: occupancies `[1,2,3,4]` at capacity 12 render as rounded mean 3, and
++   a fifth sample evicts the first.
++2. Drive equal timestamped occupancy samples through traces with sparse versus dense activity events;
++   assert identical snapshots.
++3. Sample `leafGauge` only from the existing 500ms tick, retain four values, and expose the rounded
++   mean plus capacity. Do not update the window from activity callbacks or repaint flushes.
++
++### Task 6: Render shared vocabulary, factual startup, and confidence states
++
++**Files:**
++- Modify: `cmd/metis/progress.go`
++- Modify: `cmd/metis/board.go`
++- Modify: `cmd/metis/progress_test.go`
++- Modify: `cmd/metis/board_test.go`
++
++1. Replace existing expected strings with the exact shared nouns: `outer folds`, `configs scored`,
++   nested `inner-CV runs`, flat `CV runs`, and row prefix `outer fold N`.
++2. Add nested and flat startup golden tests for: no occupancy/activity, occupied but silent, successful
++   steps with last-step age, and the first eligible run removing startup. Positive text must derive
++   only from typed successful events; never print “not hung” or infer a phase.
++3. Add pre-confidence tests showing `— inner-CV runs/min` or `— CV runs/min` and no ETA. Add mature
++   tests showing `~ETA`, the matching rate noun, remaining eligible-run scope, and tick-driven
++   `last … run Ns ago`.
++4. Update `progressCore` once so plain and TUI output share counter semantics; keep board-only temporal
++   observations in the board snapshot/renderer. Preserve width clamping with narrow-width tests.
++
++## Chunk 3: End-to-end wiring, terminal proof, and peer documentation
++
++### Task 7: Wire tick/activity flow through flat and nested sweeps
++
++**Files:**
++- Modify: `cmd/metis/sweep.go`
++- Modify: `cmd/metis/progress.go`
++- Modify: `cmd/metis/progress_test.go`
++- Modify: `cmd/metis/board_test.go`
++
++1. Add end-to-end scripted flat and nested tests that execute successful steps/runs, advance the fake
++   clock across ticks, and assert startup → confidence → mature/stall transitions.
++2. Extend the TUI fatal-failure test so activity is visible before failure, then prove the final error
++   frame is stable, no post-failure activity repaints, the ticker joins, and terminal cleanup remains
++   correct.
++3. Connect the activity emitter after board-writer replacement so callbacks use the compositor's
++   temporal writer identity. Keep the 500ms ticker and existing health gates; do not introduce a
++   second clock loop.
++4. Run `go test ./cmd/metis -run 'Progress|Board|Sweep|Activity|RunControl' -race` and fix the cause of
++   any flake, race, or lock-order timeout before proceeding.
++
++### Task 8: Update the operator contract in kbench
++
++**Files:**
++- Modify in peer repo: `competition/titanic/pipelines/RUNBOOK-sweep.md`
++- Modify: `workshop/issues/000049-board-readability-labels-coldphase-eta.md`
++
++1. In `/Users/xianxu/workspace/kbench`, update the board example/description to the exact flat/nested
++   vocabulary, factual startup line, `~slots` smoothing, 16-completion/15-second confidence gate,
++   last-run age, and mature `~ETA`. Preserve the documented plain-output distinction unless behavior
++   actually changed.
++2. Search the RUNBOOK for stale `fold`, `leaves`, and `folds/min` board terminology; inspect each hit
++   rather than globally replacing legitimate domain language.
++3. Run Markdown/diff checks available in kbench, commit that documentation-only peer change, and add
++   its full 40-character SHA to issue #49's Log.
++
++### Task 9: Full verification and close-boundary preparation
++
++**Files:**
++- Modify if architecture changed: `atlas/` and `atlas/index.md`
++- Modify: `workshop/issues/000049-board-readability-labels-coldphase-eta.md`
++
++1. Run `gofmt` on changed Go files, `go test ./cmd/metis -race`, `go test ./... -race`, and
++   `git diff --check` in metis. Run the relevant kbench checks and `git diff --check` there.
++2. Grep Go, tests, atlas, and operator docs for obsolete board strings; classify remaining domain
++   uses. Confirm flat/nested output, width, repaint cadence, failure cleanup, and no-op non-sweep paths.
++3. Update atlas only if the implementation adds a durable architectural seam; otherwise record why
++   `--no-atlas` is appropriate in close evidence. Check all issue/plan boxes and log exact commands.
++4. Run `sdlc close --issue 49 --verified '<evidence>'` once. Let the binary dispatch the mandatory
++   fresh-context boundary review; fix every Critical/Important finding and rerun the gate as directed.
+
+
+codex
+I’ll review this as a boundary review against the issue and actual repository state, not just the pasted diff. I’m going to read the issue, plan, relevant changed files, and current diff metadata first.
+exec
+/bin/zsh -lc "sed -n '1,260p' workshop/issues/000049-board-readability-labels-coldphase-eta.md" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+---
+id: 000049
+status: working
+deps: []
+github_issue:
+created: 2026-07-16
+updated: 2026-07-17
+estimate_hours: 2.63
+started: 2026-07-16T12:57:08-07:00
+---
+
+# board readability — label semantics, cold-phase "no progress" confusion, jumpy leaves, wild early ETA
+
+## Problem
+
+Operator's first real-sweep board session (titanic-sweep.md, BLAS-pinned, 2026-07-16) surfaced
+four readability issues — the board is mechanically correct but hard to READ in exactly the
+early phase where the operator most wants reassurance:
+
+1. **Label semantics unclear:** `outer 0/10 · configs 0/720 · folds 0/7200` — operator asked
+   "is folds about inner folds?" `folds` = leaf-level (config × inner-fold) RUNS aggregated
+   across the whole sweep (10 outer × 72 configs × 10 inner); `configs` = per-outer-pass
+   config completions aggregated (72 × 10). Neither is legible from the labels.
+2. **Cold-phase "lack of progress":** early in a cold run every fold row shows `queued`, all
+   counters sit at 0 for many minutes. Root cause is the metis#43 phase-wave scheduling (all
+   cv-splits/features across the grid complete before ANY train chain finishes → zero fold
+   completions for ~10 min while heavy upstream work happens). The board renders that
+   truthfully but unhelpfully — nothing distinguishes "working through upstream steps" from
+   "hung". (#43 fixes the schedule; this issue covers what the board shows MEANWHILE.)
+3. **`leaves 2/12` jumps around:** the gauge samples instantaneous `len(leafSem)` at flush
+   time — honest, but at 4Hz it reads as noise, and low occupancy during the upstream herd
+   phase looks like a problem when it isn't.
+4. **ETA changes wildly:** the 64-completion moving window over sparse, phase-heterogeneous
+   early completions (fast cache-hit folds vs slow cold trains) swings the rate — the ETA
+   flapped across hours on the operator's run. An estimate that volatile is worse than none.
+
+## Spec
+
+### Claim-time directions (retained)
+
+1. Rename/annotate segments (e.g. `inner-folds 421/7200` or a one-line legend on the first
+   frame); consider showing per-outer-fold denominators in the fold rows as the source of
+   truth for "what is 7200".
+2. A PHASE indicator when zero folds have completed but leaves are active (e.g.
+   `warming: upstream steps running · 0 folds complete yet`) — distinguish wave-phase from
+   hang using signals the sink already has (leaf occupancy > 0, passthrough step lines
+   flowing).
+3. Smooth the leaves gauge (peak-or-mean over the flush window, or EWMA) — the sink already
+   samples at each event; keep a tiny window instead of the instant.
+4. ETA damping: suppress until n≥K completions AND the window spans ≥T seconds; consider
+   EWMA on the rate; show a range or `~` marker while confidence is low. (The stall-decay
+   property — rate → 0 on thrash — must SURVIVE damping; that alarm is the line's purpose.)
+
+### Approved design — typed, time-driven board telemetry
+
+#### Counter vocabulary
+
+Rename the shared `progressCore` labels once (plain output and TUI derive from it; ARCH-DRY), while
+letting the already-known `nested` state select truthful mode-specific vocabulary:
+
+- `outer folds k/n` — completed held-out outer-fold evaluations.
+- `configs scored k/n` — configs whose full inner resample completed, aggregated across outer folds.
+- Nested: `inner-CV runs k/n` — completed `(outer fold × config × inner fold)` pipeline runs. Flat:
+  `CV runs k/n` — completed folds in the one single-level CV. These replace the ambiguous `folds`;
+  per-outer rows use `outer fold N` and the nested `configs scored` / `inner-CV runs` vocabulary
+  with per-pass denominators. Startup, rate, and ETA wording uses the same mode-specific noun.
+
+#### Typed activity seam
+
+- Add a thin `activityExecutor` decorator around the **final** step executor, outside the cache
+  decorator. It emits a typed successful-step completion after either a real execution or cache hit;
+  no terminal text is parsed, and failed steps continue through the existing fatal error path without
+  being counted as reassuring activity (ARCH-PURE).
+- The same injected activity sink has a distinct successful-run event emitted by
+  `runResolvedExperiment` only when execution has `runErr == nil` **and** required run-record /
+  provenance persistence succeeds. A failed execution remains uncounted even when its failure record
+  is written successfully. Its typed run role distinguishes nested inner-CV, flat CV, preamble, and
+  outer-score runs; only the first two feed the corresponding run counter/rate/ETA. This records
+  actual completion time at the concrete-run seam, rather than the later input-ordered `sampler.Run`
+  delivery of a completed batch (ARCH-PURPOSE).
+- Events carry an injected-clock time and typed identity. `sweepProgress` reduces successful-step
+  events to a count/last-step time and eligible run events to the run count/rate window. Concurrent
+  callback delivery may differ from event-time order, so each last-time is a max and the latest 64
+  eligible run timestamps remain sorted by event time before readiness/rate math. Reversed-delivery
+  traces pin both invariants. Callbacks are short and mutex-protected; non-sweep callers use a no-op
+  sink.
+- Before the first eligible run completes, the board reports observations, never an unprovable
+  diagnosis. Nested example: `starting · ~8/12 subprocess slots · 37 steps completed · last step 1s
+  ago · no inner-CV run complete`; flat output substitutes `CV run`. With occupancy but no successful
+  step yet, it says only that slots are occupied. It never claims “not hung” or infers an upstream
+  phase from a start line.
+- The startup line disappears on the first eligible inner-CV/CV run. This remains correct both before
+  and after #43: #43 shortens startup; #49 truthfully describes it.
+
+#### Time-driven smoothing and ETA confidence
+
+- Sample leaf occupancy on the existing 500ms board tick, not per event. Display the rolling mean of
+  the last four samples (two seconds) rounded to a whole slot and prefixed `~`. Equal timestamped
+  occupancy traces must render identically regardless of event density.
+- Keep the latest 64 eligible run-completion timestamps, but withhold rate and ETA until at least 16
+  completions span at least 15 seconds. Compute rate as `(n−1) / (now−oldest)`, removing the current
+  early upward bias that counts `n` completions across `n−1` observed intervals.
+- Continue using the current time in the denominator on every 500ms tick. With no new completions, the
+  numeric rate is non-increasing and ETA is non-decreasing, although display rounding need not change
+  within an arbitrary number of seconds after a long mature window. Preserve the fast, truthful stall
+  signal separately: after startup, show `last inner-CV run 8s ago` (or `last CV run …` flat), updated
+  on every tick. A scripted mature trace followed by five seconds of silence must advance that age by
+  five seconds while rate/ETA move monotonically; it makes no diagnosis about why work is quiet
+  (ARCH-PURPOSE). After a stall, the 64-completion window recovers gradually rather than snapping to
+  one interarrival.
+- Render the mature estimate as `~ETA …`; nested mode labels its rate `inner-CV runs/min` and scopes
+  ETA to remaining inner-CV runs, while flat mode uses `CV runs/min` and remaining CV runs. Neither
+  includes held-out scoring, capture, or final ledger work. Before confidence, show the corresponding
+  mode-specific `— … runs/min` and no ETA rather than a volatile fiction.
+
+#### Documentation boundary
+
+Update the kbench Titanic sweep RUNBOOK's board description/example after metis#49 lands. Record the
+exact peer commit in this issue's Log before close so the cross-repo requirement is reviewable.
+
+## Done when
+
+- Operator can read the first 2 minutes of a cold real sweep without wondering if it hung:
+  factual startup activity present until the first inner-CV run completes; labels self-explanatory.
+- Leaves + rate/ETA move smoothly (unit tests over scripted event traces pin the damping); after a
+  mature trace, five seconds without a completion visibly advances the mode-specific last-run age by
+  five seconds while the numeric rate/ETA move monotonically (regression test).
+- Occupied-but-silent subprocesses are never described as proven progress; typed successful step
+  completions/cache hits are the only positive activity signal.
+- A failed run that successfully persists its failure record advances neither the eligible run
+  counter nor rate/ETA (regression test).
+- Flat and nested output, width clamping, repaint cadence, and terminal cleanup remain correct.
+- RUNBOOK board description/example updated and its peer commit pinned in the Log.
+
+## Plan
+
+Durable plan: `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md`
+(single pass, no Mx — one close boundary).
+
+- [x] Add typed step/run activity at the concrete executor and persistence seams, including cache,
+  failure, role, ordering, and cancellation tests.
+- [x] Reduce time-driven occupancy and eligible-run telemetry with deterministic readiness, decay,
+  recovery, and out-of-order-event tests.
+- [x] Render truthful flat/nested vocabulary, startup observations, last-run age, and mature rate/ETA;
+  preserve repaint, failure, width, and terminal behavior.
+- [x] Update and commit the kbench Titanic RUNBOOK, pin its full commit here, then run focused,
+  race, full-suite, formatting, and stale-vocabulary verification.
+
+## Estimate
+
+```estimate
+model: estimate-logic-v3.1
+familiarity: 1.0
+item: smaller-go-module   design=0.05 impl=0.35
+item: smaller-go-module   design=0.06 impl=0.35
+item: tui-screen          design=0.15 impl=0.55
+item: cross-cutting-refactor design=0.05 impl=0.25
+item: smaller-go-module   design=0.06 impl=0.35
+item: atlas-docs          design=0.02 impl=0.10
+item: milestone-review    design=0.03 impl=0.20
+design-buffer: 0.15
+total: 2.63
+```
+
+Rows: (1) typed activity entities/decorator; (2) event-time reducer and rate/occupancy math;
+(3) board state/rendering and scripted traces; (4) shared vocabulary migration; (5) concrete-run,
+controller, and sweep wiring; (6) peer RUNBOOK plus atlas/stale-term sweep; (7) one SDLC milestone-review row.
+
+*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
+`baseline-v3.1.md`. Method A only.*
+
+## Log
+
+### 2026-07-16
+- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
+  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
+  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
+  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
+  #43 symptom: depth-first scheduling would give early train completions → a stable rate
+  much sooner.
+
+### 2026-07-16 — paired #43/#49 design approved
+- Operator chose typed step-completion telemetry over presentation-only occupancy wording because
+  occupancy cannot distinguish useful work from a hung subprocess. Co-designed after mapping the
+  current sink/rate/compositor flow; #43 merges first, then #49 builds against its schedule.
+
+### 2026-07-17 — planning checkpoint
+- Reconciled the brain project after #43 merged, ran `sdlc start-plan`, mapped the final-executor,
+  concrete-run persistence, run-control, sweep-progress, board, and kbench documentation seams, and
+  authored the durable single-boundary TDD plan. Estimate uses v3.1 Method A; the approved spec and
+  existing #38/#43 patterns make this familiar extension work rather than a novel TUI subsystem.
+- Fresh-eyes plan review: Chunk 1 found Important gaps in shared run-control activity gating and
+  concrete flat/nested role propagation; Chunk 2 found an Important gap in aggregate counter ownership.
+  Patched the plan and both reviewers re-checked clean. Chunk 3 fresh-eyes review returned clean.
+
+### 2026-07-17 — implementation verification
+- Implemented typed step/run activity, run-role propagation, run-control-gated publication, event-time
+  eligible-run rate reduction, tick-smoothed `~slots`, factual cold-start board wording, shared
+  flat/nested progress vocabulary, and last-run-age/mature `~ETA` rendering. Updated atlas with the
+  durable activity seam and board contract.
+- Updated the kbench Titanic operator RUNBOOK and committed it as
+  `68edc6ca312dbe5306ce87315abec2d67e678005` (`docs: update metis board progress contract`).
+- Verification run: `go test ./cmd/metis -count=1`; `go test ./cmd/metis -race -count=1`;
+  `go test ./... -race -count=1`; metis `git diff --check`; kbench `git diff --check`; stale board
+  terminology grep over metis `cmd/`, `atlas/`, issue/plan, and the kbench RUNBOOK. Remaining grep hits
+  are historical problem/spec wording or negative test fixtures, not live operator output.
+
+## Revisions
+
+### 2026-07-16 — fresh-eyes spec review
+- Split flat `CV runs` from nested `inner-CV runs`, made out-of-order callback reduction explicit,
+  and replaced an unprovable seconds-level rounded-rate promise with a tick-driven last-run-age
+  freshness signal plus a measurable scripted-trace criterion. Separated typed step activity from
+  actual concrete-run completion so batch-order sampler callbacks cannot falsify the rate window;
+  required execution success independently of successful failure-record persistence.
+
+### 2026-07-17 — durable implementation plan
+- Replaced the two placeholder plan rows with concrete typed-activity, pure-reducer, rendering,
+  cross-repo documentation, and verification steps; added the calibrated estimate and explicit
+  single-close-boundary decision.
+
+exec
+/bin/zsh -lc 'git diff --name-only 72f2d895fc676a474304fb4cde8f4606e9c5f48c''^..HEAD' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+atlas/experiment.md
+atlas/index.md
+cmd/metis/blaspins.go
+cmd/metis/blaspins_e2e_test.go
+cmd/metis/blaspins_test.go
+cmd/metis/board.go
+cmd/metis/board_test.go
+cmd/metis/capture.go
+cmd/metis/capture_e2e_test.go
+cmd/metis/exec.go
+cmd/metis/exec_test.go
+cmd/metis/forkexec.go
+cmd/metis/forkexec_test.go
+cmd/metis/identity_e2e_test.go
+cmd/metis/main.go
+cmd/metis/nestedcv_e2e_test.go
+cmd/metis/parallel_test.go
+cmd/metis/progress.go
+cmd/metis/run.go
+cmd/metis/runcontrol.go
+cmd/metis/runcontrol_test.go
+cmd/metis/select_cmd.go
+cmd/metis/select_cmd_test.go
+cmd/metis/shapesweep_test.go
+cmd/metis/sweep.go
+testdata/experiment/toy-sweep-smoke.md
+testdata/steps/test/env-dump
+workshop/history/issues/000022-ensembling-step.md
+workshop/history/issues/000043-leaf-sched-depth-first.md
+workshop/history/issues/000048-default-leaf-blas-pins.md
+workshop/history/issues/000050-run-end-summary-elapsed-fingerprint-next-commands.md
+workshop/history/issues/000052-select-cohort-flag-point-handles.md
+workshop/history/plans/000043-leaf-sched-depth-first-close-review.md
+workshop/history/plans/000043-leaf-sched-depth-first-plan.md
+workshop/history/plans/000048-default-leaf-blas-pins-close-review.md
+workshop/history/plans/000048-default-leaf-blas-pins-plan.md
+workshop/history/plans/000050-run-end-summary-elapsed-fingerprint-next-commands-close-review.md
+workshop/history/plans/000052-select-cohort-flag-point-handles-close-review.md
+workshop/issues/000043-leaf-sched-depth-first.md
+workshop/issues/000048-default-leaf-blas-pins.md
+workshop/issues/000049-board-readability-labels-coldphase-eta.md
+workshop/issues/000051-ledger-show-point-addr-column.md
+workshop/issues/000053-promote-fingerprint-consistency-guard.md
+workshop/lessons.md
+workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md
+
+exec
+/bin/zsh -lc 'git diff --stat 72f2d895fc676a474304fb4cde8f4606e9c5f48c''^..HEAD' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+ atlas/experiment.md                                |   23 +-
+ atlas/index.md                                     |   10 +-
+ cmd/metis/blaspins.go                              |   45 +
+ cmd/metis/blaspins_e2e_test.go                     |   74 +
+ cmd/metis/blaspins_test.go                         |   61 +
+ cmd/metis/board.go                                 |   13 +
+ cmd/metis/board_test.go                            |  108 +-
+ cmd/metis/capture.go                               |    6 +-
+ cmd/metis/capture_e2e_test.go                      |    2 +-
+ cmd/metis/exec.go                                  |    6 +
+ cmd/metis/exec_test.go                             |   25 +
+ cmd/metis/forkexec.go                              |   15 +-
+ cmd/metis/forkexec_test.go                         |   53 +-
+ cmd/metis/identity_e2e_test.go                     |    2 +-
+ cmd/metis/main.go                                  |    2 +-
+ cmd/metis/nestedcv_e2e_test.go                     |   11 +
+ cmd/metis/parallel_test.go                         |  420 ++-
+ cmd/metis/progress.go                              |   15 +-
+ cmd/metis/run.go                                   |   42 +-
+ cmd/metis/runcontrol.go                            |  104 +
+ cmd/metis/runcontrol_test.go                       |  508 +++
+ cmd/metis/select_cmd.go                            |   34 +-
+ cmd/metis/select_cmd_test.go                       |   57 +
+ cmd/metis/shapesweep_test.go                       |   21 +
+ cmd/metis/sweep.go                                 |  305 +-
+ testdata/experiment/toy-sweep-smoke.md             |   27 +
+ testdata/steps/test/env-dump                       |    5 +
+ .../{ => history}/issues/000022-ensembling-step.md |   18 +-
+ .../issues/000043-leaf-sched-depth-first.md        |  230 ++
+ .../issues/000048-default-leaf-blas-pins.md        |  125 +
+ ...nd-summary-elapsed-fingerprint-next-commands.md |   88 +
+ .../000052-select-cohort-flag-point-handles.md     |   78 +
+ .../000043-leaf-sched-depth-first-close-review.md  | 3641 ++++++++++++++++++++
+ .../plans/000043-leaf-sched-depth-first-plan.md    | 1017 ++++++
+ .../000048-default-leaf-blas-pins-close-review.md  | 2300 +++++++++++++
+ .../plans/000048-default-leaf-blas-pins-plan.md    |  446 +++
+ ...apsed-fingerprint-next-commands-close-review.md |   64 +
+ ...elect-cohort-flag-point-handles-close-review.md |   66 +
+ workshop/issues/000043-leaf-sched-depth-first.md   |   43 -
+ workshop/issues/000048-default-leaf-blas-pins.md   |   57 -
+ ...00049-board-readability-labels-coldphase-eta.md |  193 +-
+ .../issues/000051-ledger-show-point-addr-column.md |   42 +
+ ...000053-promote-fingerprint-consistency-guard.md |   60 +
+ workshop/lessons.md                                |   18 +-
+ ...-board-readability-labels-coldphase-eta-plan.md |  202 ++
+ 45 files changed, 10404 insertions(+), 278 deletions(-)
+
+codex
+The issue spec is explicit about typed activity telemetry, renamed board vocabulary, startup wording, `~slots`, and confidence-gated ETA. The changed-file list is already suspicious: much of the diff is #48/#50/#52 and run-control work, so I’m checking whether the promised #49 surfaces actually exist in code rather than assuming the plan checkboxes are accurate.
+exec
+/bin/zsh -lc "sed -n '1,460p' cmd/metis/progress.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+package main
+
+// metis#30: the sweep progress sink — folds pkg/sampler's per-completion
+// ProgressEvents (typed per level) into ONE throttled aggregated line, so a
+// 2,000-fold sweep reports live without a per-fold firehose (single-threaded-
+// attention budget). Plain appended lines, no escape codes — non-TTY-safe by
+// construction; the TTY board is metis#38, which extends this sink behind the
+// same per-pass hooks (outer-fold identity rides the forPass closure binding,
+// NEVER an event payload field — pkg/sampler stays coordinate-free).
+
+import (
+	"fmt"
+	"io"
+	"math"
+	"sort"
+	"strings"
+	"sync"
+	"time"
+
+	"github.com/xianxu/metis/pkg/sampler"
+	"github.com/xianxu/metis/pkg/shape"
+)
+
+// progressTotals seeds the sink with each level's SizeHint AT WIRING TIME —
+// stream-learned totals arrive only with a level's first completion (for the
+// driver level: the first COMPLETED outer fold, near the end of a parallel run),
+// which starves the display of denominators. cmd/metis constructs the samplers,
+// so it reads SizeHint directly; SizeHint stays the single source (ARCH-DRY).
+type progressTotals struct {
+	nested     bool
+	outer      int
+	outerKind  sampler.SizeKind
+	configs    int // aggregate across outer folds (outer × per-pass configs)
+	configKind sampler.SizeKind
+	folds      int // aggregate leaf count (outer × configs × inner k)
+	foldKind   sampler.SizeKind
+}
+
+// progressState is the pure render input: sink-owned aggregate counters per level
+// (NEVER ev.K — each concurrent Run instance counts its own 1..k), the completed
+// outer scores (→ est mean±SE), and the flat path's running fold scores.
+type progressState struct {
+	nested               bool
+	outerK, outerTotal   int
+	outerKind            sampler.SizeKind
+	configK, configTotal int
+	configKind           sampler.SizeKind
+	foldK, foldTotal     int
+	foldKind             sampler.SizeKind
+	stepK                int
+	lastStepAt           time.Time
+	lastRunAt            time.Time
+	outerScores          []float64 // nested: completed outer-fold held-out scores
+	flatScores           []float64 // flat: the one config's completed fold scores
+}
+
+// progressLine renders the aggregated line. Nested:
+// `outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`
+// (est — until an outer fold lands; ± only at n≥2). Flat (since metis#32: iff 1
+// config): `CV runs 3/5 · score 0.8400` (the running fold mean — nothing to be
+// "best" of). Kinds render k/n (exact), k/≤n (budget), k/? (unknown). Pure.
+func progressLine(st progressState) string {
+	return "metis: progress " + progressCore(st)
+}
+
+// progressCore is the un-prefixed aggregate segment — shared by the plain line and
+// the board's first row (extracted so the board never string-strips the prefix;
+// a TrimPrefix coupling would silently no-op if the prefix changed — close review).
+func progressCore(st progressState) string {
+	frac := func(k, total int, kind sampler.SizeKind) string {
+		switch kind {
+		case sampler.SizeExact:
+			return fmt.Sprintf("%d/%d", k, total)
+		case sampler.SizeBudget:
+			return fmt.Sprintf("%d/≤%d", k, total)
+		default:
+			return fmt.Sprintf("%d/?", k)
+		}
+	}
+	var parts []string
+	if st.nested {
+		parts = append(parts, "outer folds "+frac(st.outerK, st.outerTotal, st.outerKind))
+		parts = append(parts, "configs scored "+frac(st.configK, st.configTotal, st.configKind))
+		parts = append(parts, "inner-CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
+		mean, se, n := meanSE(st.outerScores)
+		switch {
+		case n == 0:
+			parts = append(parts, "est —")
+		case n == 1:
+			parts = append(parts, fmt.Sprintf("est %.4f", mean))
+		default:
+			parts = append(parts, fmt.Sprintf("est %.4f ± %.4f", mean, se))
+		}
+	} else {
+		parts = append(parts, "CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
+		if mean, _, n := meanSE(st.flatScores); n > 0 {
+			parts = append(parts, fmt.Sprintf("score %.4f", mean))
+		}
+	}
+	return strings.Join(parts, " · ")
+}
+
+// meanSE is the display-only mean ± standard-error reduce over completed scores.
+// Computed locally: the honest estimate stays pkg/sampler's Aggregate/MeanSE —
+// this is presentation, not selection (do not export sampler surface for it).
+func meanSE(xs []float64) (mean, se float64, n int) {
+	n = len(xs)
+	if n == 0 {
+		return 0, 0, 0
+	}
+	for _, x := range xs {
+		mean += x
+	}
+	mean /= float64(n)
+	if n < 2 {
+		return mean, 0, n
+	}
+	var ss float64
+	for _, x := range xs {
+		ss += (x - mean) * (x - mean)
+	}
+	se = math.Sqrt(ss/float64(n-1)) / math.Sqrt(float64(n))
+	return mean, se, n
+}
+
+// seededTotals reads each level's SizeHint on its initial state — the SAME source
+// the Run loops stamp on events (ARCH-DRY; no shape math re-derived here) — and
+// composes the aggregate denominators: configs = outer × per-pass configs,
+// folds = outer × configs × inner k (each sealed pass sweeps the full grid).
+// Flat (1 config): folds = the single pass's inner k.
+func seededTotals(ctx sampler.Ctx, nested bool, runFolds int, configPts []shape.Point, k int) progressTotals {
+	grid := sampler.GridConfigs{Points: configPts}
+	nConfigs, kindConfigs := grid.SizeHint(grid.Init(ctx))
+	foldsSmp := sampler.FixedKFolds{K: k}
+	nFolds, kindFolds := foldsSmp.SizeHint(foldsSmp.Init(ctx))
+	if !nested {
+		return progressTotals{folds: nFolds, foldKind: kindFolds}
+	}
+	cv := sampler.CVDriver{K: runFolds}
+	nOuter, kindOuter := cv.SizeHint(cv.Init(ctx))
+	return progressTotals{
+		nested: true,
+		outer:  nOuter, outerKind: kindOuter,
+		configs: nOuter * nConfigs, configKind: kindConfigs,
+		folds: nOuter * nConfigs * nFolds, foldKind: kindFolds,
+	}
+}
+
+// movingRate retains the latest eligible run completions by event time. rate(now)
+// = (n-1)/(now-oldest) after the confidence gate; `now` in the denominator means
+// a STALL decays live while last-run age remains the sharp freshness signal.
+type movingRate struct {
+	times []time.Time
+}
+
+func (m *movingRate) add(t time.Time) {
+	i := sort.Search(len(m.times), func(i int) bool { return !m.times[i].Before(t) })
+	m.times = append(m.times, time.Time{})
+	copy(m.times[i+1:], m.times[i:])
+	m.times[i] = t
+	if len(m.times) > 64 {
+		m.times = m.times[1:]
+	}
+}
+
+// rate returns eligible runs/minute over the kept event-time window.
+func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
+	if len(m.times) < 16 {
+		return 0, false
+	}
+	oldest := m.times[0]
+	newest := m.times[len(m.times)-1]
+	if newest.Sub(oldest) < 15*time.Second {
+		return 0, false
+	}
+	mins := now.Sub(oldest).Minutes()
+	if mins <= 0 {
+		return 0, false
+	}
+	return float64(len(m.times)-1) / mins, true
+}
+
+// eta = remaining / rate; ok=false when the rate is unavailable or zero.
+func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
+	r, ok := m.rate(now)
+	if !ok || r <= 0 || remaining <= 0 {
+		return 0, false
+	}
+	return time.Duration(float64(remaining) / r * float64(time.Minute)), true
+}
+
+type occupancySample struct {
+	busy, capacity int
+}
+
+type occupancyWindow struct {
+	samples []occupancySample
+}
+
+func (w *occupancyWindow) add(busy, capacity int) {
+	if capacity <= 0 {
+		return
+	}
+	w.samples = append(w.samples, occupancySample{busy: busy, capacity: capacity})
+	if len(w.samples) > 4 {
+		w.samples = w.samples[1:]
+	}
+}
+
+func (w occupancyWindow) mean() (busy, capacity int, ok bool) {
+	if len(w.samples) == 0 {
+		return 0, 0, false
+	}
+	var sum int
+	for _, s := range w.samples {
+		sum += s.busy
+		capacity = s.capacity
+	}
+	return int(math.Round(float64(sum) / float64(len(w.samples)))), capacity, true
+}
+
+// passRow is one outer fold's live board row (metis#38): in-flight counters + the
+// pass's incumbent best (display-only — NOT the 1-SE select rule), collapsing to its
+// held-out score when the driver reports the fold done.
+type passRow struct {
+	configK, foldK int
+	best           float64
+	hasBest        bool
+	done           bool
+	heldOut        float64
+}
+
+// boardState is the pure render input for metis#38's board: the #30 aggregate state
+// plus the per-pass rows and the throughput ring (a mutex'd snapshot — renderers never
+// touch the live sink).
+type boardState struct {
+	st   progressState
+	rows []passRow
+	rate movingRate
+}
+
+// sweepProgress is the mutex'd sink shared by every pass of one shape-run. Events
+// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
+// its own Run's event mutex); health-gated paths use the strict order runControl.mu
+// → sink.mu → boardWriter.mu (never the reverse). Emit policy: fold/config events are throttled
+// to one line per second (injected clock — tests script it, never sleep); a
+// driver-level (outer fold) completion ALWAYS emits; finish() emits the terminal
+// line. A nil *sweepProgress is a no-op everywhere (the non-sweep path is silent).
+type sweepProgress struct {
+	mu        sync.Mutex
+	out       io.Writer
+	now       func() time.Time
+	direction string // the objective direction — orients each pass's display-best (#38)
+	st        progressState
+	rows      []passRow  // metis#38: one row per outer fold (nil on the flat path)
+	rate      movingRate // metis#38: fold-completion throughput window
+	lastEmit  time.Time
+	started   bool
+	// metis#38 board mode (all nil/zero in plain mode): emits paint the rendered frame
+	// to bw instead of printing plain lines. Lock order: sink.mu → bw.mu, ALWAYS — the
+	// ticker enters via tick() (a sink method), never a boardWriter-first path.
+	bw        *boardWriter
+	width     int               // terminal width ($COLUMNS | 80), read once at wiring
+	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
+	occupancy occupancyWindow
+}
+
+func newSweepProgress(out io.Writer, now func() time.Time, direction string, totals progressTotals) *sweepProgress {
+	var rows []passRow
+	if totals.nested && totals.outer > 0 {
+		rows = make([]passRow, totals.outer)
+	}
+	return &sweepProgress{
+		out: out, now: now, direction: direction, rows: rows,
+		st: progressState{
+			nested:     totals.nested,
+			outerTotal: totals.outer, outerKind: totals.outerKind,
+			configTotal: totals.configs, configKind: totals.configKind,
+			foldTotal: totals.folds, foldKind: totals.foldKind,
+		},
+	}
+}
+
+// boardState snapshots the sink for a renderer (rows copied — the caller may hold
+// the snapshot without racing the live fold-in).
+func (sp *sweepProgress) boardState() boardState {
+	sp.mu.Lock()
+	defer sp.mu.Unlock()
+	return sp.snapshotLocked()
+}
+
+// snapshotLocked builds the render snapshot; caller holds sp.mu (shared by
+// boardState() and the board-mode emit — one copy site, close-review DRY note).
+func (sp *sweepProgress) snapshotLocked() boardState {
+	rows := make([]passRow, len(sp.rows))
+	copy(rows, sp.rows)
+	return boardState{st: sp.st, rows: rows, rate: sp.rate}
+}
+
+// passHooks are one pass's typed event targets, closure-bound to its outer-fold
+// index (-1 = the flat path's single pass) — the metis#38 identity seam.
+type passHooks struct {
+	config func(sampler.ProgressEvent[shape.Point, sampler.MeanSE])
+	fold   func(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome])
+}
+
+// forPass hands out a pass's hooks. #30's sink aggregates across passes (the
+// single-line mental model); the per-pass binding exists so #38 can add per-fold
+// board rows behind the same API without touching pkg/sampler.
+func (sp *sweepProgress) forPass(outer int) passHooks {
+	if sp == nil {
+		return passHooks{
+			config: func(sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {},
+			fold:   func(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {},
+		}
+	}
+	return passHooks{
+		config: func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
+			sp.mu.Lock()
+			defer sp.mu.Unlock()
+			sp.st.configK++
+			if outer >= 0 && outer < len(sp.rows) { // #38: this pass's row
+				r := &sp.rows[outer]
+				r.configK++
+				if !r.hasBest || better(sp.direction, ev.Out.Mean, r.best) {
+					r.best, r.hasBest = ev.Out.Mean, true
+				}
+			}
+			sp.maybeEmit(false)
+		},
+		fold: func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
+			sp.mu.Lock()
+			defer sp.mu.Unlock()
+			if !sp.st.nested {
+				sp.st.flatScores = append(sp.st.flatScores, ev.Out.Score)
+			}
+			if outer >= 0 && outer < len(sp.rows) {
+				sp.rows[outer].foldK++
+			}
+			sp.maybeEmit(false)
+		},
+	}
+}
+
+func (sp *sweepProgress) activity(ev activityEvent) {
+	if sp == nil {
+		return
+	}
+	sp.mu.Lock()
+	defer sp.mu.Unlock()
+	switch ev.Kind {
+	case activityStepSuccess:
+		sp.st.stepK++
+		at := ev.At
+		if at.IsZero() {
+			at = sp.now()
+		}
+		if at.After(sp.st.lastStepAt) {
+			sp.st.lastStepAt = at
+		}
+	case activityRunSuccess:
+		if ev.Role != runRoleNestedInnerCV && ev.Role != runRoleFlatCV {
+			return
+		}
+		sp.st.foldK++
+		at := ev.At
+		if at.IsZero() {
+			at = sp.now()
+		}
+		if at.After(sp.st.lastRunAt) {
+			sp.st.lastRunAt = at
+		}
+		sp.rate.add(at)
+	default:
+		return
+	}
+	sp.maybeEmit(false)
+}
+
+// better orients a display-best comparison by the objective direction.
+func better(direction string, candidate, incumbent float64) bool {
+	if direction == "minimize" {
+		return candidate < incumbent
+	}
+	return candidate > incumbent
+}
+
+// driverEvent folds a completed OUTER fold in — always emits (the coarse level is
+// the one the operator watches; its completions are rare and load-bearing).
+func (sp *sweepProgress) driverEvent(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
+	if sp == nil {
+		return
+	}
+	sp.mu.Lock()
+	defer sp.mu.Unlock()
+	sp.st.outerK++
+	sp.st.outerScores = append(sp.st.outerScores, ev.Out)
+	if i := ev.Point.Idx; i >= 0 && i < len(sp.rows) { // #38: collapse this fold's row
+		sp.rows[i].done = true
+		sp.rows[i].heldOut = ev.Out
+	}
+	sp.maybeEmit(true)
+}
+
+// finish emits the terminal state line (always).
+func (sp *sweepProgress) finish() {
+	if sp == nil {
+		return
+	}
+	sp.mu.Lock()
+	defer sp.mu.Unlock()
+	sp.emit()
+}
+
+// tick is the board ticker's entry point (metis#38): repaint with a fresh `now` so
+// the rate decay + ETA move even between events. Sink-first (sp.mu → bw.mu).
+func (sp *sweepProgress) tick() {
+	if sp == nil || sp.bw == nil {
+		return
+	}
+	sp.mu.Lock()
+	if sp.gauge != nil {
+		busy, capacity := sp.gauge()
+		sp.occupancy.add(busy, capacity)
+	}
+	sp.emit() // stores the fresh frame (budget may skip the draw)
+	bw := sp.bw
+	sp.mu.Unlock()
+	bw.forceFlush() // metis#46: the tick is what re-pins the board after a burst window
+}
+
+// abort removes the stored live frame after a sweep failure. Lock order remains
+// progress -> board; the controller is never called while either lock is held.
+func (sp *sweepProgress) abort() {
+	if sp == nil || sp.bw == nil {
+		return
+	}
+	sp.mu.Lock()
+	defer sp.mu.Unlock()
+	sp.bw.discardFrame()
+}
+
+// maybeEmit writes the line if forced (driver/finish) or the throttle elapsed —
+// 1s for plain lines (a log is a record), 100ms for board repaints (a board is a
+// display; the 500ms ticker guarantees freshness anyway). Caller holds sp.mu.
+func (sp *sweepProgress) maybeEmit(force bool) {
+	now := sp.now()
+	throttle := time.Second
+	if sp.bw != nil {
+		throttle = 100 * time.Millisecond
+	}
+	if !force && sp.started && now.Sub(sp.lastEmit) < throttle {
+		return
+	}
+	sp.started = true
+	sp.lastEmit = now
+	sp.emit()
+}
+
+// emit renders the current state: board mode paints the frame (under the fixed
+
+exec
+/bin/zsh -lc 'rg -n "activityEvent|activityExecutor|runRole|occupancyWindow|movingRate|inner-CV runs|CV runs|configs scored|outer folds|~slots|last .* run|no inner-CV|no CV run|runs/min|ETA|leaves|folds/min|folds " cmd/metis' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+cmd/metis/progress.go:3:// metis#30: the sweep progress sink — folds pkg/sampler's per-completion
+cmd/metis/progress.go:33:	configs    int // aggregate across outer folds (outer × per-pass configs)
+cmd/metis/progress.go:35:	folds      int // aggregate leaf count (outer × configs × inner k)
+cmd/metis/progress.go:58:// `outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`
+cmd/metis/progress.go:60:// config): `CV runs 3/5 · score 0.8400` (the running fold mean — nothing to be
+cmd/metis/progress.go:82:		parts = append(parts, "outer folds "+frac(st.outerK, st.outerTotal, st.outerKind))
+cmd/metis/progress.go:83:		parts = append(parts, "configs scored "+frac(st.configK, st.configTotal, st.configKind))
+cmd/metis/progress.go:84:		parts = append(parts, "inner-CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
+cmd/metis/progress.go:95:		parts = append(parts, "CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
+cmd/metis/progress.go:129:// folds = outer × configs × inner k (each sealed pass sweeps the full grid).
+cmd/metis/progress.go:130:// Flat (1 config): folds = the single pass's inner k.
+cmd/metis/progress.go:149:// movingRate retains the latest eligible run completions by event time. rate(now)
+cmd/metis/progress.go:152:type movingRate struct {
+cmd/metis/progress.go:156:func (m *movingRate) add(t time.Time) {
+cmd/metis/progress.go:166:// rate returns eligible runs/minute over the kept event-time window.
+cmd/metis/progress.go:167:func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
+cmd/metis/progress.go:184:func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
+cmd/metis/progress.go:196:type occupancyWindow struct {
+cmd/metis/progress.go:200:func (w *occupancyWindow) add(busy, capacity int) {
+cmd/metis/progress.go:210:func (w occupancyWindow) mean() (busy, capacity int, ok bool) {
+cmd/metis/progress.go:239:	rate movingRate
+cmd/metis/progress.go:243:// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
+cmd/metis/progress.go:256:	rate      movingRate // metis#38: fold-completion throughput window
+cmd/metis/progress.go:264:	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
+cmd/metis/progress.go:265:	occupancy occupancyWindow
+cmd/metis/progress.go:345:func (sp *sweepProgress) activity(ev activityEvent) {
+cmd/metis/progress.go:362:		if ev.Role != runRoleNestedInnerCV && ev.Role != runRoleFlatCV {
+cmd/metis/progress.go:388:// driverEvent folds a completed OUTER fold in — always emits (the coarse level is
+cmd/metis/progress.go:416:// the rate decay + ETA move even between events. Sink-first (sp.mu → bw.mu).
+cmd/metis/blaspins.go:7:// leaf semaphore), not to each leaf's BLAS — NumCPU leaves × multi-threaded BLAS
+cmd/metis/main.go:45:	sampleN := fs.Int("sample", 0, "metis#42: run m of the k outer folds (sparse fold sampling; 0/omitted = all k). k stays the estimand (each fold trains on (k-1)/k of the rows); m only trades precision for cost — use to probe a higher k (e.g. k=10, --sample 3) without the full k× bill. The SE over m<k folds is noisy (m-1 df): probe with it, don't re-select what ships on it. Errors on m>k, on a single-config (flat) run, and combined with --fast.")
+cmd/metis/fingerprints_test.go:17:// folds record metadata (first/last timestamps, latest commit/dirty/status, distinct extra
+cmd/metis/shipe2e_test.go:38:	// Declare this step's read-set D (the code file it reads) so recordMiss folds it into the
+cmd/metis/shipe2e_test.go:59:// inner folds → an honest per-config (mean, SE) leaderboard, argmax-mean winner, driver:single
+cmd/metis/shipe2e_test.go:61:// re-runs the downstream folds through the sweep while the config/fold-invariant data + partition
+cmd/metis/capture_e2e_test.go:19:// call site leaves the direct-helper tests green but must fail THIS one (the Done-when is
+cmd/metis/forkexec_test.go:200:// TestForkServerPerf_LooseBound (metis#44 acceptance): N leaves that each import pandas —
+cmd/metis/forkexec_test.go:250:	// (server start + preload amortize over ~5k leaves in a real sweep, not over n=4).
+cmd/metis/caching.go:55:// (from the sensor's reads.json), folds the transitive-D snapshot, stores the output, and
+cmd/metis/caching.go:112:			// downstream step in the same run still folds an upstream HIT's closure into
+cmd/metis/caching.go:186:	// side (recordMiss folds + stores the SAME closure) — store and HIT-check can't disagree.
+cmd/metis/select_cmd.go:240:			fmt.Fprintf(out, "    %-24s mean %.4f  SE %.4f  (n=%d outer folds)\n", famLabel(f), est[f].Mean, est[f].SE, len(est[f].ToldSet))
+cmd/metis/family.go:10:// reduce the objective metric over the outer folds via the shared sampler.Aggregate →
+cmd/metis/family.go:14:// outer folds (outer-fold 0 → rf md=4, outer-fold 1 → rf md=8), so those rows share the family
+cmd/metis/family.go:16:// them in separate groups and never compute the per-family mean over the outer folds. Only
+cmd/metis/family.go:18:// AggregateView, which pools a config's inner folds across outer folds), not this.
+cmd/metis/caching_test.go:157:	var stepEvents []activityEvent
+cmd/metis/caching_test.go:168:			activity: func(ev activityEvent) {
+cmd/metis/caching_test.go:448:	// guard at the executor level — two folds of one step get distinct entries).
+cmd/metis/run_test.go:195:	var runEvents []activityEvent
+cmd/metis/run_test.go:204:		activity: func(ev activityEvent) {
+cmd/metis/run_test.go:228:	if runEvents[0].Role != runRoleNone {
+cmd/metis/run_test.go:246:	var runEvents []activityEvent
+cmd/metis/run_test.go:254:		activity: func(ev activityEvent) {
+cmd/metis/run_test.go:290:	var runEvents []activityEvent
+cmd/metis/run_test.go:298:		activity: func(ev activityEvent) {
+cmd/metis/run_test.go:343:	// A rejected run writes no runs/ dir and leaves the config .md untouched (#13 — the
+cmd/metis/sweep.go:28:// foldMetric is the per-fold score the resample folds over — the metric the train step
+cmd/metis/sweep.go:35:// fold_score (metis#19): the fitted model's realized complexity (rf mean leaves / logreg
+cmd/metis/sweep.go:74:// shapeSweep is the mutable accumulator the pure nested-Sampler loop folds through the IO
+cmd/metis/sweep.go:87:	manMu         sync.Mutex     // metis#32: guards man.Points — concurrent outer folds (ParExec) each record rows
+cmd/metis/sweep.go:93:// nested run's outer folds run concurrently under ParExec, each recording its inner+outer rows).
+cmd/metis/sweep.go:128:	runRole  runRole              // concrete-run role for every pipeline fold in this pass
+cmd/metis/sweep.go:131:	// resample fans out over folds — all appending to this ONE pass. `mu` guards the
+cmd/metis/sweep.go:213:	// a plain k-fold of the one config on ALL data, not the nested subset-sweep). The outer folds
+cmd/metis/sweep.go:230:			return fmt.Errorf("run: --sample only applies to a nested (multi-config) run — this shape has 1 config, a flat CV with no outer folds to sample")
+cmd/metis/sweep.go:245:			fmt.Fprintf(out, "metis: single-level CV %s — %d config × %d folds (dry run):\n", sh.ID, len(configPts), k)
+cmd/metis/sweep.go:270:	// and a 500ms ticker keeps the rate decay + ETA live between events (sink-first:
+cmd/metis/sweep.go:324:	pass := &sweepPass{ss: ss, splitK: k, stratify: stratify, partRef: ss.partRef, runRole: runRoleFlatCV,
+cmd/metis/sweep.go:361:	// after the folds run.
+cmd/metis/sweep.go:447:	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
+cmd/metis/sweep.go:490:	preOpts.runRole = runRoleNestedPreamble
+cmd/metis/sweep.go:510:	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
+cmd/metis/sweep.go:513:		runRole: runRoleNestedInnerCV,
+cmd/metis/sweep.go:592:	scoreOpts.runRole = runRoleOuterScore
+cmd/metis/sweep.go:611:// reportEstimate prints the honest procedure estimate — mean±SE over the outer folds — and the
+cmd/metis/sweep.go:653:	pointOpts.runRole = p.runRole
+cmd/metis/sweep.go:818:	fmt.Fprintf(ss.out, "metis: sweep %s done — %d configs scored (manifest %s)\n", ss.sh.ID, len(ss.configs), ss.man.ShapeRunID[:12])
+cmd/metis/sweep.go:950:		fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort ?)\n", fmtETA(elapsed), rows, base)
+cmd/metis/sweep.go:957:	fmt.Fprintf(out, "metis: done in %s — %d rows → %s (cohort %s)\n", fmtETA(elapsed), rows, base, fp)
+cmd/metis/trace.go:18:// site-packages (→ the uv.lock digest folds into D per-root, below).
+cmd/metis/forkexec.go:335:			p.noticeOnce("root:"+spec.root, fmt.Sprintf("server for %s died mid-flight (%v) — erroring the step; later leaves use legacy", spec.root, err))
+cmd/metis/progress_test.go:31:			[]string{"outer folds 0/3", "configs scored 84/216", "inner-CV runs 421/1080", "est —"}, []string{"±"}},
+cmd/metis/progress_test.go:33:			[]string{"outer folds 1/3", "est 0.8200"}, []string{"±"}},
+cmd/metis/progress_test.go:35:			[]string{"outer folds 2/3", "est 0.8200 ± 0.0200"}, nil},
+cmd/metis/progress_test.go:39:			[]string{"CV runs 3/5", "score 0.8400"}, []string{"configs", "outer", "folds 3/5"}},
+cmd/metis/progress_test.go:44:			[]string{"outer folds 1/?", "configs scored 3/?"}, nil},
+cmd/metis/progress_test.go:48:			[]string{"outer folds 1/≤8"}, nil},
+cmd/metis/progress_test.go:95:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 200)})
+cmd/metis/progress_test.go:111:	for _, w := range []string{"outer folds 1/2", "inner-CV runs 10/20", "est 0.8300"} {
+cmd/metis/progress_test.go:144:				prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
+cmd/metis/progress_test.go:150:	if !strings.Contains(out.String(), "inner-CV runs 64/64") {
+cmd/metis/progress_test.go:160:// movingRate: keep the latest 64 eligible completion times by event time. It is
+cmd/metis/progress_test.go:163:	var r movingRate
+cmd/metis/progress_test.go:173:	var short movingRate
+cmd/metis/progress_test.go:180:	var ready movingRate
+cmd/metis/progress_test.go:192:	var r2 movingRate
+cmd/metis/progress_test.go:200:	// ETA: remaining/rate.
+cmd/metis/progress_test.go:207:	var w occupancyWindow
+cmd/metis/progress_test.go:241:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
+cmd/metis/progress_test.go:256:// Per-pass rows: each forPass(i) hook folds into ITS row (closure-bound identity);
+cmd/metis/progress_test.go:326:	prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedPreamble, RunID: "pre", At: at(0)})
+cmd/metis/progress_test.go:332:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, RunID: "inner", At: at(i * 1000)})
+cmd/metis/board.go:19:// (for rate/ETA), and the leaf-semaphore gauge (capacity 0 = no gauge → segment
+cmd/metis/board.go:33:// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
+cmd/metis/board.go:64:			lines = append(lines, fmt.Sprintf("  outer fold %d ▸ configs scored %d/%d · inner-CV runs %d/%d%s",
+cmd/metis/board.go:72:	// Leaves / throughput / ETA.
+cmd/metis/board.go:74:	runNoun := "CV runs"
+cmd/metis/board.go:77:		runNoun = "inner-CV runs"
+cmd/metis/board.go:81:		segs = append(segs, fmt.Sprintf("~slots %d/%d", env.busy, env.capacity))
+cmd/metis/board.go:100:		segs = append(segs, fmt.Sprintf("last %s %s ago", runSingular, fmtAge(env.now.Sub(bs.st.lastRunAt))))
+cmd/metis/board.go:109:			segs = append(segs, "~ETA "+fmtETA(eta))
+cmd/metis/board.go:124:	return fmtETA(d)
+cmd/metis/board.go:127:// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
+cmd/metis/board.go:128:func fmtETA(d time.Duration) string {
+cmd/metis/board.go:234:// (this is what re-pins the board after a burst window and keeps ETA/rate moving).
+cmd/metis/activity.go:16:type runRole string
+cmd/metis/activity.go:19:	runRoleNone           runRole = ""
+cmd/metis/activity.go:20:	runRoleNestedInnerCV  runRole = "nested-inner-cv"
+cmd/metis/activity.go:21:	runRoleFlatCV         runRole = "flat-cv"
+cmd/metis/activity.go:22:	runRoleNestedPreamble runRole = "nested-preamble"
+cmd/metis/activity.go:23:	runRoleOuterScore     runRole = "outer-score"
+cmd/metis/activity.go:26:type activityEvent struct {
+cmd/metis/activity.go:31:	Role   runRole
+cmd/metis/activity.go:34:type activityEmitter func(activityEvent)
+cmd/metis/activity.go:36:func (e activityEmitter) emit(ev activityEvent) {
+cmd/metis/activity.go:49:	return func(ev activityEvent) {
+cmd/metis/activity.go:59:	return func(ev activityEvent) {
+cmd/metis/activity.go:64:type activityExecutor struct {
+cmd/metis/activity.go:70:func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+cmd/metis/activity.go:79:	e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
+cmd/metis/run.go:73:	sample   int  // metis#42: nested run does m of the k outer folds (0 = all k). k stays the
+cmd/metis/run.go:83:	runRole     runRole         // role for successful concrete-run activity; zero = ineligible/non-sweep
+cmd/metis/run.go:99:	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
+cmd/metis/run.go:107:// the inner resample folds each — runShapeSweep); a plain `type: experiment` is the
+cmd/metis/run.go:254:	exec = activityExecutor{inner: exec, now: now, emit: o.activity}
+cmd/metis/run.go:304:	o.activity.emit(activityEvent{Kind: activityRunSuccess, At: now(), RunID: runID, Role: o.runRole})
+cmd/metis/runcontrol_test.go:512:	var events []activityEvent
+cmd/metis/runcontrol_test.go:513:	emit := runControlActivityEmitter(control, func(ev activityEvent) {
+cmd/metis/runcontrol_test.go:517:	emit(activityEvent{Kind: activityStepSuccess, StepID: "prep"})
+cmd/metis/runcontrol_test.go:523:	emit(activityEvent{Kind: activityStepSuccess, StepID: "late-step"})
+cmd/metis/runcontrol_test.go:524:	emit(activityEvent{Kind: activityRunSuccess, RunID: "late-run"})
+cmd/metis/shapesweep_test.go:61:// nudge, so distinct models have distinct MEANS (winner selection) and distinct folds give
+cmd/metis/shapesweep_test.go:75:// fixed value per model, fold-independent (like a tree's realized leaves), so a config's
+cmd/metis/shapesweep_test.go:85:// isolation (shipWinner no-ops on an empty ship). 2 configs × 2 folds = 4 per-fold runs.
+cmd/metis/shapesweep_test.go:176:// family) × 2 inner × 2 outer folds → 8 inner + 2 outer rows.
+cmd/metis/shapesweep_test.go:207:		t.Errorf("2 configs × 2 inner × 2 outer folds → 8 inner rows, got %d", nInner)
+cmd/metis/shapesweep_test.go:210:		t.Errorf("1 family × 2 outer folds → 2 outer rows, got %d", nOuter)
+cmd/metis/shapesweep_test.go:213:	// The honest procedure estimate (mean±SE over outer folds) is reported — NOT a single winner
+cmd/metis/shapesweep_test.go:332:		if !strings.Contains(final, "CV runs 2/2") || !strings.Contains(final, "score 0.") {
+cmd/metis/shapesweep_test.go:333:			t.Errorf("the flat final progress line must carry folds k/k + score; got: %q", final)
+cmd/metis/shapesweep_test.go:356:	t.Run("flat CV runs are eligible flat roles", func(t *testing.T) {
+cmd/metis/shapesweep_test.go:359:		counts := map[runRole]int{}
+cmd/metis/shapesweep_test.go:363:			activity: func(ev activityEvent) {
+cmd/metis/shapesweep_test.go:372:		if counts[runRoleFlatCV] != 2 || len(counts) != 1 {
+cmd/metis/shapesweep_test.go:373:			t.Fatalf("flat roles = %v; want exactly 2 flat-CV runs", counts)
+cmd/metis/shapesweep_test.go:380:		counts := map[runRole]int{}
+cmd/metis/shapesweep_test.go:384:			activity: func(ev activityEvent) {
+cmd/metis/shapesweep_test.go:393:		if counts[runRoleNestedPreamble] != 1 {
+cmd/metis/shapesweep_test.go:396:		if counts[runRoleNestedInnerCV] != 8 {
+cmd/metis/shapesweep_test.go:397:			t.Fatalf("nested roles = %v; want 8 inner-CV runs", counts)
+cmd/metis/shapesweep_test.go:399:		if counts[runRoleOuterScore] != 2 {
+cmd/metis/family_test.go:11:// the outer folds — the reduction AggregateView cannot do, because a family's winner differs
+cmd/metis/family_test.go:12:// across outer folds (rf md=4 in fold 0, rf md=8 in fold 1) → distinct free-params, same family.
+cmd/metis/family_test.go:23:	// rf winner differs across outer folds (md4 then md8) — same family, must pool.
+cmd/metis/family_test.go:44:		t.Errorf("rf family mean=%v want 0.79 (pooled two outer folds, inner row ignored)", rf.Mean)
+cmd/metis/family_test.go:47:		t.Errorf("rf family SE should be non-zero over 2 outer folds")
+cmd/metis/exec.go:60:		// driver:single path leaves the var unset (unconfined).
+cmd/metis/identity_e2e_test.go:75:// a real git blob, which is exactly what a traced run leaves behind.
+cmd/metis/identity_e2e_test.go:92:	// record.json (no fingerprint yet) — the state a run leaves for capture to backfill.
+cmd/metis/board_test.go:14:// queued), overflow cap, leaves+throughput line. NO ANSI — escape codes live only in
+cmd/metis/board_test.go:41:		"outer folds 1/3", "configs scored 14/36", "inner-CV runs 47/108", "est 0.7980",
+cmd/metis/board_test.go:43:		"outer fold 1 ▸ configs scored 8/12 · inner-CV runs 25/36 · best 0.8340",
+cmd/metis/board_test.go:45:		"~slots 8/8", "42.5 inner-CV runs/min", "~ETA",
+cmd/metis/board_test.go:51:	if len(lines) != 5 { // aggregate + 3 fold rows + leaves
+cmd/metis/board_test.go:59:	// All-done: every row ✓, no ETA segment (nothing remaining).
+cmd/metis/board_test.go:65:	if strings.Contains(frame, "▸") || strings.Contains(frame, "ETA") {
+cmd/metis/board_test.go:66:		t.Errorf("all-done: no in-flight rows, no ETA:\n%s", frame)
+cmd/metis/board_test.go:72:		t.Errorf("flat board = aggregate + leaves, got %d lines: %v", len(got), got)
+cmd/metis/board_test.go:75:	// Overflow: 14 folds → 12 rows + "… +2 more" + slots/rate + aggregate = 15 lines.
+cmd/metis/board_test.go:101:	if !strings.Contains(frame, "inner-CV runs/min") {
+cmd/metis/board_test.go:105:	// Rate unavailable (fresh ring): "— inner-CV runs/min".
+cmd/metis/board_test.go:107:	st.rate = movingRate{}
+cmd/metis/board_test.go:109:	if !strings.Contains(frame, "— inner-CV runs/min") {
+cmd/metis/board_test.go:122:		"starting", "~slots 8/12", "37 steps completed", "last step 1s ago", "no inner-CV run complete",
+cmd/metis/board_test.go:137:	if strings.Contains(frame, "starting") || strings.Contains(frame, "no inner-CV run complete") {
+cmd/metis/board_test.go:150:		"CV runs 0/20", "starting", "~slots 3/8", "4 steps completed", "last step 2s ago", "no CV run complete",
+cmd/metis/board_test.go:163:	for _, want := range []string{"CV runs 1/20", "last CV run 1s ago", "— CV runs/min"} {
+cmd/metis/board_test.go:168:	if strings.Contains(frame, "starting") || strings.Contains(frame, "ETA") {
+cmd/metis/board_test.go:169:		t.Fatalf("flat pre-confidence frame must be post-startup without ETA:\n%s", frame)
+cmd/metis/board_test.go:175:	var rate movingRate
+cmd/metis/board_test.go:187:	for _, want := range []string{"45.0 inner-CV runs/min", "last inner-CV run 5s ago", "~ETA"} {
+cmd/metis/board_test.go:194:// fmtETA is compact and human: seconds under a minute, m+s under an hour.
+cmd/metis/board_test.go:195:func TestFmtETA(t *testing.T) {
+cmd/metis/board_test.go:203:		if got := fmtETA(d); got != want {
+cmd/metis/board_test.go:204:			t.Errorf("fmtETA(%v) = %q, want %q", d, got, want)
+cmd/metis/board_test.go:308:	bw.paint([]string{"folds 2/8", "31.2 folds/min · ETA 12s"})
+cmd/metis/board_test.go:317:	for _, stale := range []string{"folds 2/8", "folds/min", "ETA"} {
+cmd/metis/board_test.go:361:	if !strings.Contains(s, "outer folds 2/2") || !strings.Contains(s, "outer fold 0 ✓") || !strings.Contains(s, "outer fold 1 ✓") {
+cmd/metis/board_test.go:372:	if finalFrame := strings.LastIndex(s, "outer folds 2/2"); warnIdx > finalFrame {
+cmd/metis/board_test.go:431:	for _, want := range []string{"outer folds 0/2", "outer fold 0 — queued", "no inner-CV run"} {
+cmd/metis/board_test.go:451:		"outer folds 0/2", "outer fold 0 — queued", "configs ", "inner-CV runs ", "inner-CV runs/min", "ETA", "score ", "estimate", "mean ",
+cmd/metis/board_test.go:610:	if !strings.Contains(term2.String(), "outer folds 0/1") {
+cmd/metis/parallel_test.go:30:	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds = 6 per-fold rows
+cmd/metis/parallel_test.go:304:	// 3 configs → nested (outer folds = sweeper.cv.k = 2) × 2 inner folds → deep nesting, ~many leaf calls.
+cmd/metis/parallel_test.go:407:// failureBarrierExec holds the first four admitted inner folds at their train
+cmd/metis/parallel_test.go:517:	for _, forbidden := range []string{"metis: progress", "folds/min", "ETA", "score ", "estimate", "mean "} {
+cmd/metis/record_test.go:65:	// Upstream/D/Deps are metis#2-populated slots — #3 leaves them empty.
+cmd/metis/activity_test.go:30:	var events []activityEvent
+cmd/metis/activity_test.go:32:	got, err := activityExecutor{
+cmd/metis/activity_test.go:35:		emit:  func(ev activityEvent) { events = append(events, ev) },
+cmd/metis/activity_test.go:58:	var events []activityEvent
+cmd/metis/activity_test.go:60:	got, err := activityExecutor{
+cmd/metis/activity_test.go:63:		emit:  func(ev activityEvent) { events = append(events, ev) },
+cmd/metis/nestedcv_e2e_test.go:56:	// The honest estimate is reported as a mean±SE over the outer folds.
+cmd/metis/nestedcv_e2e_test.go:69:	if !strings.Contains(finalProg, "outer folds 2/2") || !strings.Contains(finalProg, "est 0.") {
+cmd/metis/nestedcv_e2e_test.go:86:		t.Errorf("expected 2 outer-fold held-out scores (2 outer folds × 1 family), got %d:\n%s", n, s)
+cmd/metis/nestedcv_e2e_test.go:132:	// 2 configs → nested; outer folds = sweeper.resample.cv.k = 2.
+cmd/metis/nestedcv_e2e_test.go:141:// TestNestedCV_SampleRunsMOfKFolds (metis#42): `--sample m` runs exactly m of the k outer folds of
+cmd/metis/nestedcv_e2e_test.go:144:// Asserts: m held-out scores, m outer ledger rows (folds 0..m-1 of the k-partition), and the
+cmd/metis/nestedcv_e2e_test.go:165:	// 2 sampled outer folds × 1 family (a,b share the scalar `model` knob) = 2 held-out lines.
+cmd/metis/nestedcv_e2e_test.go:182:		t.Errorf("outer rows should cover exactly sampled folds {0,1} of k=3, got %v", outerFolds)
+cmd/metis/nestedcv_e2e_test.go:187:// k folds), --sample on a single-config shape (the flat path has no outer folds to sample), and
+cmd/metis/nestedcv_e2e_test.go:215:		o := base(newShape(t, "[a]")) // 1 config → flat CV, no outer folds
+cmd/metis/caching_soundness_test.go:49:	// format, so recordMiss folds it into the transitive-D closure and isHit re-hashes it.
+
+exec
+/bin/zsh -lc "sed -n '1,320p' cmd/metis/run.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+package main
+
+import (
+	"encoding/json"
+	"fmt"
+	"io"
+	"os"
+	"path/filepath"
+	"strings"
+	"sync"
+	"time"
+
+	"github.com/xianxu/metis/internal/repo"
+	"github.com/xianxu/metis/pkg/cas"
+	"github.com/xianxu/metis/pkg/experiment"
+)
+
+// syncWriter serializes concurrent Write calls — the metis#31 parallel fan-out's
+// progress output. Minimal: it prevents torn lines + the data race on a shared
+// writer; it does NOT reorder or buffer per goroutine (clean per-k/n progress is
+// metis#30's scope). Established in runExperiment when maxParallel>1.
+type syncWriter struct {
+	mu sync.Mutex
+	w  io.Writer
+}
+
+func (s *syncWriter) Write(p []byte) (int, error) {
+	s.mu.Lock()
+	defer s.mu.Unlock()
+	return s.w.Write(p)
+}
+
+// cacheProjectRoot resolves the metis code root (the module dir above steps/) that D
+// paths are relative to and `git hash-object` runs in — the same root metis.trace
+// records in reads.json. Falls back to the experiment dir if step paths don't resolve.
+func cacheProjectRoot(stepPath []string, fallback string) string {
+	for _, p := range stepPath {
+		if root, err := repo.Root(p); err == nil {
+			return root
+		}
+	}
+	return fallback
+}
+
+// ensureCacheGitignore writes .metis-cache/.gitignore so the local, wipeable cache
+// (content-addressed output blobs) is never committed to the experiment's repo — the
+// cache is safe to `rm -rf` and rebuild. Idempotent. (Sharing the git-trackable index
+// across clones is a future enhancement; v1 ignores the whole cache dir.)
+func ensureCacheGitignore(cacheDir string) error {
+	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
+		return err
+	}
+	gi := filepath.Join(cacheDir, ".gitignore")
+	if _, err := os.Stat(gi); err == nil {
+		return nil
+	}
+	body := "# metis#2 step cache — a local, wipeable content-addressed cache (rm -rf is safe).\n" +
+		"# Never commit its output blobs.\n*\n"
+	return os.WriteFile(gi, []byte(body), 0o644)
+}
+
+// runOpts are the inputs to one `metis run`. now/git/out are injected so the e2e
+// test gets a deterministic clock, a fake git probe, and can discard progress output.
+type runOpts struct {
+	expPath  string
+	runID    string
+	stepPath []string
+	now      func() time.Time
+	git      gitProbe
+	cache    bool // enable the metis#2 validating-trace cache (<expDir>/.metis-cache)
+	dryRun   bool // metis#18: list the swept configs without running them
+	fast     bool // metis#32: nested run does ONE outer fold (a ~1/k-cost honest single-point) instead of k
+	sample   int  // metis#42: nested run does m of the k outer folds (0 = all k). k stays the
+	//               estimand knob (train fraction); m is the precision/cost knob — each fold is an
+	//               unbiased sample of the k-fold estimand. --fast ≡ --sample 1 (kept as shorthand).
+	inSweep bool // metis#14: this run is a sweep point — suppress per-point single-run
+	//               capture (the sweep captures once per shape-run in captureSweepCode)
+	out  io.Writer
+	exec experiment.StepExecutor // test seam: an injected fake replaces the subprocess
+	//                              execStep (nil → the production execStep). Composes with
+	//                              cache: the caching decorator still wraps it.
+	activity    activityEmitter // successful step/run facts; nil is a no-op
+	runRole     runRole         // role for successful concrete-run activity; zero = ineligible/non-sweep
+	readRoot    string          // metis#23: when set, the production execStep confines base-dataset reads to this root
+	maxParallel int             // metis#31: >1 ⇒ ParExec batches + a leaf semaphore; sizes leafSem
+	leafSem     chan struct{}   // metis#31: the shared global subprocess budget (nil = serial/cache-only)
+	runControl  *runControl     // one per shape run: global abort + optional 2n admission slots
+	runLabel    string          // config/fold/preamble context captured with the first error
+	forkserver  bool            // metis#44: warm fork-server leaf executor (cmdRun default true;
+	//                           zero-value false keeps direct runOpts callers/tests on legacy exec)
+	forkPool *serverPool // metis#44: the per-root warm-server pool, created once per runExperiment
+	//                      when forkserver is set; threaded through nested runOpts copies.
+	tui bool // metis#38: stdout is a TTY and --no-tui wasn't passed — a SWEEP pins the live board
+	//          (a plain experiment ignores it; non-TTY/piped runs stay on the #30 plain lines)
+	board           *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
+	boardTick       <-chan time.Time  // test seam: nil uses the production 500ms ticker
+	beforeBoardTick func()            // test seam: after tick selection, before health observation
+	afterBoardTick  func()            // test seam: after the health observation returns
+	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
+	leafPins        []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
+	//                             runExperiment (nil = not yet computed; non-nil rides nested runOpts
+	//                             copies like forkPool — an all-suppressed result is empty, not nil)
+}
+
+// runExperiment reads the experiment at o.expPath and dispatches: a `type:
+// experiment-shape` is the metis#18 nested-Sampler SWEEP (the sweeper grids over configs,
+// the inner resample folds each — runShapeSweep); a plain `type: experiment` is the
+// one-point path (runResolvedExperiment). The `.md` is immutable input (#13) — never
+// written back; all side effects live in the shell below, the ordering/validation logic
+// stays in pkg/experiment. Returns the assembled Run (empty for a sweep — the manifest +
+// per-fold records + ledger are its output) and the run error.
+func runExperiment(o runOpts) (experiment.Run, error) {
+	now := o.now
+	if now == nil {
+		now = time.Now
+	}
+	out := o.out
+	if out == nil {
+		out = io.Discard
+	}
+
+	// metis#38: PARSE FIRST (parsing writes nothing) — the board decision needs the file
+	// type, and writer identity is TEMPORAL: everything constructed below (fork-server
+	// pool, execs, sink, capture warnings) captures whatever `out` is at ITS construction,
+	// so the one compositor must exist before any of them or its writes bypass the board.
+	raw, err := os.ReadFile(o.expPath)
+	if err != nil {
+		return experiment.Run{}, err
+	}
+	// Peek the type with the tolerant experiment parser (it ignores the shape-only
+	// data/pipeline/ship/sweeper keys); a shape then re-parses through the STRICT
+	// ParseShape (unknown-key-loud) for the sweep path.
+	exp, err := experiment.Parse(string(raw))
+	if err != nil {
+		return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
+	}
+
+	// metis#31: establish the parallel invariant in ONE home — maxParallel>1 ⇒ a
+	// non-nil SHARED leaf semaphore AND a serialized writer. Doing it here (not in
+	// cmdRun) means no direct-runOpts caller (the tests) can enable maxParallel>1 yet
+	// forget the sem or race the fan-out's progress writes on a bare buffer.
+	if o.maxParallel > 1 && o.leafSem == nil {
+		o.leafSem = make(chan struct{}, o.maxParallel)
+	}
+	if sem := o.leafSem; sem != nil && o.leafGauge == nil {
+		o.leafGauge = func() (int, int) { return len(sem), cap(sem) } // metis#38: occupancy IS the semaphore
+	}
+	// Exactly ONE writer wrap (metis#38): board mode → the pin-bottom compositor (it
+	// serializes internally — no syncWriter stacking); else parallel → syncWriter.
+	if o.tui && exp.Type == "experiment-shape" && !o.dryRun {
+		o.board = newBoardWriter(out, now)
+		out = o.board
+		o.out = out
+		defer o.board.close() // idempotent — an error return must not leak a hidden cursor
+	} else if o.maxParallel > 1 {
+		out = &syncWriter{w: out}
+		o.out = out
+	}
+	// metis#48: default leaf BLAS pins — computed ONCE per top-level run from the ambient
+	// env (an exported operator value wins by exclusion in blasPins), announced loudly,
+	// and injected at both spawn seams (legacy execStep child env; fork-server process
+	// env). Fake-exec runs spawn nothing (no pins, no note); dry-run lists configs (same).
+	// Placed AFTER the writer wrap (the note must route through the board) and BEFORE the
+	// pool creation (the server spawn env needs the pins).
+	if o.exec == nil && !o.dryRun && o.leafPins == nil {
+		o.leafPins = blasPins(os.Environ())
+		if len(o.leafPins) > 0 {
+			fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
+				strings.Join(o.leafPins, " "))
+		}
+	}
+	// metis#44: one warm fork-server pool per top-level run, shut down (EOF-drain) when the
+	// run ends. Only the production executor uses it (an injected test exec bypasses execStep).
+	// Constructed AFTER the writer wrap — its fallback notices must route through the board.
+	if o.forkserver && o.exec == nil && o.forkPool == nil {
+		o.forkPool = newServerPool(out, o.leafPins)
+		defer o.forkPool.shutdown()
+	}
+	if exp.Type == "experiment-shape" {
+		sh, err := experiment.ParseShape(string(raw))
+		if err != nil {
+			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
+		}
+		if err := experiment.ValidateShape(sh); err != nil {
+			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
+		}
+		if o.runControl == nil {
+			o.runControl = newRunControl(o.maxParallel)
+		}
+		return experiment.Run{}, runShapeSweep(o, sh, now, out)
+	}
+	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
+}
+
+// singleRunID names a single run's dir. metis#27: content-address it by the run's
+// point-address (symmetric with a sweep point's dir), so the dir name IS the run identity.
+// An explicit --run overrides; the timestamp form survives only as the no-git fallback
+// (when the shape blob-hash — hence the point-address — can't be computed).
+func singleRunID(o runOpts, exp experiment.Experiment, now func() time.Time) string {
+	if o.runID != "" {
+		return o.runID
+	}
+	sbh, err := shapeBlobHash(o.expPath)
+	if err == nil {
+		if addr, err := pointAddressOf(exp, sbh); err == nil {
+			return addr
+		}
+	}
+	return "run-" + now().UTC().Format("20060102T150405Z")
+}
+
+// runResolvedExperiment runs one already-resolved experiment (a single point) under
+// runID, through the cached runner, and writes its run.json + provenance record (the
+// experiment `.md` is immutable input — not written back, #13). The shared per-point runner
+// both the 1-point path and the sweep loop (metis#7) call — so the run/cache/record wiring
+// lives in ONE place (ARCH-DRY).
+func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
+	if o.runControl == nil {
+		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
+	}
+	o.activity = runControlActivityEmitter(o.runControl, o.activity)
+	return o.runControl.run(o.runLabel, func() (experiment.Run, error) {
+		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
+	})
+}
+
+func runResolvedExperimentAdmitted(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
+	baseDir := filepath.Dir(o.expPath)
+	// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
+	// the child's env, and the child's cwd IS the step dir — a relative path would
+	// resolve $METIS_STEP_DIR/with.json under itself. Absolute paths are correct
+	// from any cwd, so `metis run pipelines/foo.md` (a relative arg) works.
+	runDir, err := filepath.Abs(filepath.Join(baseDir, "runs", runID))
+	if err != nil {
+		return experiment.Run{}, err
+	}
+	expDir, err := filepath.Abs(baseDir)
+	if err != nil {
+		return experiment.Run{}, err
+	}
+
+	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool, pins: o.leafPins}
+	if o.exec != nil {
+		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
+	}
+	if o.cache {
+		cacheDir := filepath.Join(expDir, ".metis-cache")
+		if err := ensureCacheGitignore(cacheDir); err != nil {
+			return experiment.Run{}, err
+		}
+		store := cas.NewFSStore(filepath.Join(cacheDir, "cas"), 0, cas.Clock(now))
+		exec = newCachingExecutor(exec, store, cacheDir, exp.Seed, out)
+	}
+	exec = activityExecutor{inner: exec, now: now, emit: o.activity}
+	runner := experiment.Runner{Exec: exec, Now: now}
+	fmt.Fprintf(out, "metis: run %s of experiment %q\n", runID, exp.ID)
+	run, steps, runErr := runner.Run(exp, runID, runDir)
+
+	// Execution-time enforcement: Runner.Run validates the experiment BEFORE any
+	// step executes, so a semantically-invalid experiment (dangling needs, bad
+	// uses, a cycle) is rejected here — closing the SHAPE-only gap M1 left. Such a
+	// rejection never started a run (run.Started is empty), so surface the error
+	// without writing a bogus record.
+	if run.Started == "" {
+		return run, runErr
+	}
+
+	// Write the ledger even on a mid-run step failure (status=failed) so every
+	// attempt that began is recorded — the record of truth is runs/<id>/run.json.
+	if err := writeRunJSON(runDir, run); err != nil {
+		return run, err
+	}
+	// Assemble + persist the provenance record (metis#3): repo provenance, per-step
+	// output hashes, and the minted point-address. A config that can't be
+	// canonicalized (e.g. a non-finite value) surfaces here as a run error.
+	// The shape's blob-hash content-addresses the intent (metis#27); computed the SAME way
+	// singleRunID/pointAddressOf did, so the record's point_address matches the run dir.
+	// A no-git spec yields "" (a degraded, non-content-addressed run — warned via capture status).
+	sbh, _ := shapeBlobHash(o.expPath)
+	rec, err := assembleRecord(o.git, out, expDir, runDir, exp, run, steps, sbh)
+	if err != nil {
+		return run, err
+	}
+	if err := writeRecordJSON(runDir, rec); err != nil {
+		return run, err
+	}
+	// Capture this run's code closure + run-spec to a git side-ref (metis#14), backfilling
+	// the record with the durable SHA + capture status — so a dirty single run is
+	// reproducible (git checkout the SHA). The sweep loop sets inSweep to capture ONCE
+	// per shape-run instead (captureSweepCode), avoiding redundant per-point capture.
+	// Best-effort (like the sweep path): a backfill hiccup warns, never aborts a finished run.
+	if !o.inSweep {
+		if err := captureSingleRun(o, runID); err != nil {
+			fmt.Fprintf(out, "metis: warning: code-capture backfill failed for run %s: %v\n", runID, err)
+		}
+	}
+	// The experiment .md is IMMUTABLE input (#13): a run writes its output to
+	// runs/<id>/{run,record}.json (+ the .ledger.csv sidecar for sweeps), NEVER to the
+	// config file — so a committed config is a stable content-hash. The human "recent
+	// runs / top-N" view is on-demand via `metis ledger show` over the sidecar.
+	if runErr != nil {
+		return run, runErr
+	}
+	o.activity.emit(activityEvent{Kind: activityRunSuccess, At: now(), RunID: runID, Role: o.runRole})
+	fmt.Fprintf(out, "metis: %s %s\n", run.ID, run.Status)
+	return run, nil
+}
+
+func writeRunJSON(runDir string, run experiment.Run) error {
+	if err := os.MkdirAll(runDir, 0o755); err != nil {
+		return err
+	}
+	b, err := json.MarshalIndent(run, "", "  ")
+	if err != nil {
+		return err
+	}
+	return os.WriteFile(filepath.Join(runDir, "run.json"), append(b, '\n'), 0o644)
+}
+
+codex
+The current tree does include the #49 typed-activity code that was not visible in the pasted excerpt, so I’m switching from “missing implementation” to targeted correctness checks. I’m going to inspect the activity wiring, board rendering, and persistence/error seams for contract drift.
+exec
+/bin/zsh -lc "sed -n '1,180p' cmd/metis/activity.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+package main
+
+import (
+	"time"
+
+	"github.com/xianxu/metis/pkg/experiment"
+)
+
+type activityKind string
+
+const (
+	activityStepSuccess activityKind = "step-success"
+	activityRunSuccess  activityKind = "run-success"
+)
+
+type runRole string
+
+const (
+	runRoleNone           runRole = ""
+	runRoleNestedInnerCV  runRole = "nested-inner-cv"
+	runRoleFlatCV         runRole = "flat-cv"
+	runRoleNestedPreamble runRole = "nested-preamble"
+	runRoleOuterScore     runRole = "outer-score"
+)
+
+type activityEvent struct {
+	Kind   activityKind
+	At     time.Time
+	StepID string
+	RunID  string
+	Role   runRole
+}
+
+type activityEmitter func(activityEvent)
+
+func (e activityEmitter) emit(ev activityEvent) {
+	if e != nil {
+		e(ev)
+	}
+}
+
+func teeActivityEmitter(a, b activityEmitter) activityEmitter {
+	if a == nil {
+		return b
+	}
+	if b == nil {
+		return a
+	}
+	return func(ev activityEvent) {
+		a.emit(ev)
+		b.emit(ev)
+	}
+}
+
+func runControlActivityEmitter(control *runControl, emit activityEmitter) activityEmitter {
+	if control == nil {
+		return emit
+	}
+	return func(ev activityEvent) {
+		control.whileHealthy(func() { emit.emit(ev) })
+	}
+}
+
+type activityExecutor struct {
+	inner experiment.StepExecutor
+	now   func() time.Time
+	emit  activityEmitter
+}
+
+func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+	res, err := e.inner.Execute(step, runDir)
+	if err != nil {
+		return res, err
+	}
+	now := e.now
+	if now == nil {
+		now = time.Now
+	}
+	e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
+	return res, nil
+}
+
+exec
+/bin/zsh -lc "sed -n '1,170p' cmd/metis/board.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+package main
+
+// metis#38: the live progress board — the PURE frame renderer (this half) and the
+// pin-bottom ANSI compositor (boardWriter, below). Presentation only, over the #30
+// sink's boardState snapshot: no pkg/sampler change, no TUI library (the board is
+// output-only — a hand-rolled repaint of N lines; see the plan's no-lib rationale).
+// The paint/content split is deliberate: renderBoard returns plain lines (byte-
+// testable, zero escape codes); ANSI lives ONLY in boardWriter.
+
+import (
+	"fmt"
+	"io"
+	"strings"
+	"sync"
+	"time"
+)
+
+// boardEnv is the render-time environment: terminal width, the current instant
+// (for rate/ETA), and the leaf-semaphore gauge (capacity 0 = no gauge → segment
+// omitted, e.g. a serial run).
+type boardEnv struct {
+	width          int
+	now            time.Time
+	busy, capacity int
+}
+
+// maxFoldRows caps the per-fold section; beyond it the remainder collapses to
+// an "… +N more" line (a 20-fold sweep must not paint a 22-line board).
+const maxFoldRows = 12
+
+// renderBoard renders the frame: the #30 aggregate line, one row per outer fold
+// (✓ done → held-out score · ▸ in-flight → per-pass counters + incumbent · queued),
+// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
+// break the compositor's erase-count bookkeeping).
+func renderBoard(bs boardState, env boardEnv) []string {
+	var lines []string
+	// Row 1: the aggregate — the same core the plain line prints (one source, no
+	// prefix stripping).
+	lines = append(lines, progressCore(bs.st))
+
+	// Per-fold rows (nested only; flat runs have no rows).
+	shown := len(bs.rows)
+	if shown > maxFoldRows {
+		shown = maxFoldRows
+	}
+	// Per-row denominators derive from the seeded aggregate totals (per-pass share).
+	perConfigs, perFolds := 0, 0
+	if n := len(bs.rows); n > 0 {
+		perConfigs = bs.st.configTotal / n
+		perFolds = bs.st.foldTotal / n
+	}
+	for i := 0; i < shown; i++ {
+		r := bs.rows[i]
+		switch {
+		case r.done:
+			lines = append(lines, fmt.Sprintf("  outer fold %d ✓ held-out %.4f", i, r.heldOut))
+		case r.configK == 0 && r.foldK == 0:
+			lines = append(lines, fmt.Sprintf("  outer fold %d — queued", i))
+		default:
+			b := ""
+			if r.hasBest {
+				b = fmt.Sprintf(" · best %.4f", r.best)
+			}
+			lines = append(lines, fmt.Sprintf("  outer fold %d ▸ configs scored %d/%d · inner-CV runs %d/%d%s",
+				i, r.configK, perConfigs, r.foldK, perFolds, b))
+		}
+	}
+	if hidden := len(bs.rows) - shown; hidden > 0 {
+		lines = append(lines, fmt.Sprintf("  … +%d more", hidden))
+	}
+
+	// Leaves / throughput / ETA.
+	var segs []string
+	runNoun := "CV runs"
+	runSingular := "CV run"
+	if bs.st.nested {
+		runNoun = "inner-CV runs"
+		runSingular = "inner-CV run"
+	}
+	if env.capacity > 0 {
+		segs = append(segs, fmt.Sprintf("~slots %d/%d", env.busy, env.capacity))
+	}
+	if bs.st.foldK == 0 {
+		start := []string{"starting"}
+		start = append(start, segs...)
+		if bs.st.stepK > 0 {
+			start = append(start, fmt.Sprintf("%d steps completed", bs.st.stepK))
+			if !bs.st.lastStepAt.IsZero() {
+				start = append(start, "last step "+fmtAge(env.now.Sub(bs.st.lastStepAt))+" ago")
+			}
+		}
+		start = append(start, fmt.Sprintf("no %s complete", runSingular))
+		lines = append(lines, strings.Join(start, " · "))
+		for i, l := range lines {
+			lines[i] = clampLine(l, env.width)
+		}
+		return lines
+	}
+	if !bs.st.lastRunAt.IsZero() {
+		segs = append(segs, fmt.Sprintf("last %s %s ago", runSingular, fmtAge(env.now.Sub(bs.st.lastRunAt))))
+	}
+	if perMin, ok := bs.rate.rate(env.now); ok {
+		segs = append(segs, fmt.Sprintf("%.1f %s/min", perMin, runNoun))
+	} else {
+		segs = append(segs, fmt.Sprintf("— %s/min", runNoun))
+	}
+	if remaining := bs.st.foldTotal - bs.st.foldK; remaining > 0 {
+		if eta, ok := bs.rate.eta(env.now, remaining); ok {
+			segs = append(segs, "~ETA "+fmtETA(eta))
+		}
+	}
+	lines = append(lines, strings.Join(segs, " · "))
+
+	for i, l := range lines {
+		lines[i] = clampLine(l, env.width)
+	}
+	return lines
+}
+
+func fmtAge(d time.Duration) string {
+	if d < 0 {
+		d = 0
+	}
+	return fmtETA(d)
+}
+
+// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
+func fmtETA(d time.Duration) string {
+	d = d.Round(time.Second)
+	if d < time.Minute {
+		return fmt.Sprintf("%ds", int(d.Seconds()))
+	}
+	if d < time.Hour {
+		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
+	}
+	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
+}
+
+// (Height analog of the width limitation: a terminal SHORTER than the board clamps
+// cursor-up at the screen top and desyncs the erase count — the board caps at ~15
+// lines; terminals that small are out of scope, same accepted trade as resize.)
+
+// clampLine truncates to width runes with a trailing … (a wrapped physical line
+// would desync the compositor's cursor-up erase count — width is load-bearing).
+func clampLine(s string, width int) string {
+	if width <= 0 {
+		return s
+	}
+	r := []rune(s)
+	if len(r) <= width {
+		return s
+	}
+	return string(r[:width-1]) + "…"
+}
+
+// ── boardWriter: the pin-bottom ANSI compositor ──────────────────────────────
+
+// boardWriter owns the terminal: the board is pinned to the bottom while every
+// other write (step logs, warnings) scrolls ABOVE it. Paint-only — it stores the
+// last rendered frame and NEVER calls back into the sink (the one global lock
+// order is sink.mu → bw.mu; a callback here would invert it). All output must
+// route through this writer once it exists — a bypassing write corrupts the board
+// (see the plan's writer-plumbing note: writer identity is temporal).
+//
+// metis#46: DOUBLE-BUFFERED with a bounded flush rate. The original design ran a
+// full erase→write→repaint cycle per passthrough write; a warm-cache sweep emits
+// hundreds of lines/second and real terminals — especially mux layers (the
+// operator's ghostty-in-cmux) — paint asynchronously mid-sequence and tear under
+// that flood. Now passthrough COALESCES into `pending` and the terminal sees one
+// atomic erase→dump→repaint per flushBudget (~4Hz) — quiet writes (a cold run's
+
+exec
+/bin/zsh -lc "sed -n '1,220p' cmd/metis/activity_test.go && sed -n '140,330p' cmd/metis/run_test.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+package main
+
+import (
+	"errors"
+	"reflect"
+	"testing"
+	"time"
+
+	"github.com/xianxu/metis/pkg/experiment"
+)
+
+type activityFakeExec struct {
+	result experiment.StepResult
+	err    error
+	calls  int
+}
+
+func (f *activityFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+	f.calls++
+	return f.result, f.err
+}
+
+func TestActivityExecutorEmitsOneStepSuccessAfterSuccessfulInnerExecution(t *testing.T) {
+	at := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
+	wantResult := experiment.StepResult{
+		Metrics:   map[string]float64{"score": 0.91},
+		Artifacts: []string{"train/model.bin"},
+	}
+	inner := &activityFakeExec{result: wantResult}
+	var events []activityEvent
+
+	got, err := activityExecutor{
+		inner: inner,
+		now:   func() time.Time { return at },
+		emit:  func(ev activityEvent) { events = append(events, ev) },
+	}.Execute(experiment.Step{ID: "train"}, "/tmp/run")
+
+	if err != nil {
+		t.Fatalf("Execute returned error: %v", err)
+	}
+	if !reflect.DeepEqual(got, wantResult) {
+		t.Fatalf("Execute result = %+v; want %+v", got, wantResult)
+	}
+	if inner.calls != 1 {
+		t.Fatalf("inner calls = %d; want 1", inner.calls)
+	}
+	if len(events) != 1 {
+		t.Fatalf("events = %d; want 1 (%+v)", len(events), events)
+	}
+	if events[0].Kind != activityStepSuccess || events[0].StepID != "train" || !events[0].At.Equal(at) {
+		t.Fatalf("event = %+v; want one step-success event for train at %s", events[0], at.Format(time.RFC3339))
+	}
+}
+
+func TestActivityExecutorEmitsNothingOnInnerErrorAndPreservesFailure(t *testing.T) {
+	wantErr := errors.New("boom")
+	inner := &activityFakeExec{err: wantErr}
+	var events []activityEvent
+
+	got, err := activityExecutor{
+		inner: inner,
+		now:   func() time.Time { return time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC) },
+		emit:  func(ev activityEvent) { events = append(events, ev) },
+	}.Execute(experiment.Step{ID: "train"}, "/tmp/run")
+
+	if !errors.Is(err, wantErr) {
+		t.Fatalf("Execute error = %v; want %v", err, wantErr)
+	}
+	if !reflect.DeepEqual(got, experiment.StepResult{}) {
+		t.Fatalf("Execute result = %+v; want zero result from inner failure", got)
+	}
+	if inner.calls != 1 {
+		t.Fatalf("inner calls = %d; want 1", inner.calls)
+	}
+	if len(events) != 0 {
+		t.Fatalf("events = %+v; want none on error", events)
+	}
+}
+	}
+
+	run, err := runExperiment(runOpts{
+		expPath:  expPath,
+		runID:    "run-001",
+		stepPath: []string{filepath.Join(root, "testdata", "steps")},
+		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
+		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
+		out:      io.Discard,
+	})
+	if err == nil {
+		t.Fatal("runExperiment: want an error from the failing step, got nil")
+	}
+	if run.Status != "failed" {
+		t.Errorf("returned run status = %q; want failed", run.Status)
+	}
+
+	// runs/run-001/run.json written with status=failed.
+	rb, err := os.ReadFile(filepath.Join(dir, "runs", "run-001", "run.json"))
+	if err != nil {
+		t.Fatalf("read run.json (failed run should still be recorded): %v", err)
+	}
+	var got experiment.Run
+	if err := json.Unmarshal(rb, &got); err != nil {
+		t.Fatalf("parse run.json: %v", err)
+	}
+	if got.ID != "run-001" || got.Experiment != "run-fail" || got.Status != "failed" {
+		t.Errorf("run.json wrong: %+v", got)
+	}
+
+	// #13: a failed run is recorded in run.json/record.json (asserted above) — the config .md
+	// stays byte-for-byte immutable input (the fixture already carries a `## Runs` heading, so
+	// this asserts equality, not absence).
+	updated, err := os.ReadFile(expPath)
+	if err != nil {
+		t.Fatal(err)
+	}
+	if string(updated) != string(b) {
+		t.Errorf("failed run mutated the config .md (must be immutable input):\n%s", updated)
+	}
+}
+
+func TestRunExperimentActivityRunSuccessPublishesAfterRequiredArtifacts(t *testing.T) {
+	root := repoRoot(t)
+	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
+	b, err := os.ReadFile(src)
+	if err != nil {
+		t.Fatal(err)
+	}
+	dir := t.TempDir()
+	expPath := filepath.Join(dir, "run-echo.md")
+	if err := os.WriteFile(expPath, b, 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	var runEvents []activityEvent
+	var publishErrs []error
+	_, err = runExperiment(runOpts{
+		expPath:  expPath,
+		runID:    "run-activity",
+		stepPath: []string{filepath.Join(root, "testdata", "steps")},
+		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
+		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
+		out:      io.Discard,
+		activity: func(ev activityEvent) {
+			if ev.Kind != activityRunSuccess {
+				return
+			}
+			runEvents = append(runEvents, ev)
+			for _, name := range []string{"run.json", "record.json"} {
+				if _, statErr := os.Stat(filepath.Join(dir, "runs", ev.RunID, name)); statErr != nil {
+					publishErrs = append(publishErrs, statErr)
+				}
+			}
+		},
+	})
+	if err != nil {
+		t.Fatalf("runExperiment: %v", err)
+	}
+	if len(publishErrs) > 0 {
+		t.Fatalf("run-success event arrived before required artifacts existed: %v", publishErrs)
+	}
+	if len(runEvents) != 1 {
+		t.Fatalf("run-success events = %d; want 1 (%+v)", len(runEvents), runEvents)
+	}
+	if runEvents[0].RunID != "run-activity" {
+		t.Fatalf("run event = %+v; want runID run-activity", runEvents[0])
+	}
+	if runEvents[0].Role != runRoleNone {
+		t.Fatalf("plain run event role = %q; want no role/ineligible", runEvents[0].Role)
+	}
+}
+
+func TestRunExperimentActivityRunSuccessNotPublishedForFailedExecution(t *testing.T) {
+	root := repoRoot(t)
+	src := filepath.Join(root, "testdata", "experiment", "run-fail.md")
+	b, err := os.ReadFile(src)
+	if err != nil {
+		t.Fatal(err)
+	}
+	dir := t.TempDir()
+	expPath := filepath.Join(dir, "run-fail.md")
+	if err := os.WriteFile(expPath, b, 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	var runEvents []activityEvent
+	_, err = runExperiment(runOpts{
+		expPath:  expPath,
+		runID:    "run-failed-activity",
+		stepPath: []string{filepath.Join(root, "testdata", "steps")},
+		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
+		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
+		out:      io.Discard,
+		activity: func(ev activityEvent) {
+			if ev.Kind == activityRunSuccess {
+				runEvents = append(runEvents, ev)
+			}
+		},
+	})
+	if err == nil {
+		t.Fatal("runExperiment: want failing step error")
+	}
+	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "run.json")); statErr != nil {
+		t.Fatalf("failed run should still write run.json: %v", statErr)
+	}
+	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "record.json")); statErr != nil {
+		t.Fatalf("failed run should still write record.json: %v", statErr)
+	}
+	if len(runEvents) != 0 {
+		t.Fatalf("failed execution emitted run-success events: %+v", runEvents)
+	}
+}
+
+func TestRunExperimentActivityRunSuccessNotPublishedWhenRecordPersistenceFails(t *testing.T) {
+	root := repoRoot(t)
+	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
+	b, err := os.ReadFile(src)
+	if err != nil {
+		t.Fatal(err)
+	}
+	dir := t.TempDir()
+	expPath := filepath.Join(dir, "run-echo.md")
+	if err := os.WriteFile(expPath, b, 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.MkdirAll(filepath.Join(dir, "runs", "run-record-fails", "record.json"), 0o755); err != nil {
+		t.Fatal(err)
+	}
+
+	var runEvents []activityEvent
+	_, err = runExperiment(runOpts{
+		expPath:  expPath,
+		runID:    "run-record-fails",
+		stepPath: []string{filepath.Join(root, "testdata", "steps")},
+		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
+		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
+		out:      io.Discard,
+		activity: func(ev activityEvent) {
+			if ev.Kind == activityRunSuccess {
+				runEvents = append(runEvents, ev)
+			}
+		},
+	})
+	if err == nil {
+		t.Fatal("runExperiment: want record persistence error")
+	}
+	if len(runEvents) != 0 {
+		t.Fatalf("record persistence failure emitted run-success events: %+v", runEvents)
+	}
+}
+
+// TestRunExperiment_RejectsInvalidAtRunTime is the execution-time enforcement
+// test: a semantically-invalid experiment (a cycle — shape-valid, so CUE accepts
+// it) is rejected by `metis run` BEFORE any step runs, closing the SHAPE-only gap
+// M1 deferred. No ledger and no `## Runs` line are written for a rejected run.
+func TestRunExperiment_RejectsInvalidAtRunTime(t *testing.T) {
+	root := repoRoot(t)
+	src := filepath.Join(root, "testdata", "experiment", "invalid-cycle.md")
+	b, err := os.ReadFile(src)
+	if err != nil {
+		t.Fatal(err)
+	}
+	dir := t.TempDir()
+	expPath := filepath.Join(dir, "invalid-cycle.md")
+	if err := os.WriteFile(expPath, b, 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	_, err = runExperiment(runOpts{
+		expPath:  expPath,
+
+exec
+/bin/zsh -lc "sed -n '440,700p' cmd/metis/sweep.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+		return err
+	}
+	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the estimate report
+
+	// metis#32: the nested run now RECORDS (unlike metis#23's estimation-only path) — persist the
+	// inner + per-family outer rows accumulated in ss.man.Points so `metis select` can reduce them
+	// (family from the outer rows, config from the inner rows). Sort to a stable content key first
+	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
+	sortPointRuns(ss.man.Points)
+	if err := writeManifest(ss.o.expPath, ss.man); err != nil {
+		return ss.fail("write nested sweep manifest", err)
+	}
+	cohort, cerr := captureSweepCode(ss.o, ss.man)
+	if cerr != nil {
+		ss.whileHealthy(func() {
+			fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
+		})
+	}
+	if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
+		return ss.fail("write nested sweep ledger", err)
+	}
+	ss.whileHealthy(func() {
+		ss.reportEstimate(est, runFolds)
+		printRunSummary(ss.out, ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
+	})
+	return ss.firstError()
+}
+
+// materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
+// and returns the k analysis_i refs (experiment-relative, so a sealed sweep reading one routes
+// through exp_path → confined). Unconfined (outer-split reads the full dataset to split it).
+func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]string, error) {
+	baseOut, baseID := baseDatasetRef(ss.sh)
+	var needs []string
+	if baseID != "" {
+		needs = []string{baseID}
+	}
+	osStep := experiment.Step{ID: outerSplitStepID, Uses: "metis/outer-split", Needs: needs,
+		With: map[string]any{"dataset": baseOut, "k": outerK, "stratify": stratify}}
+	steps := append(append([]experiment.Step{}, ss.sh.Data...), osStep)
+	exp := experiment.Experiment{Header: ss.sh.Header, Steps: steps}
+	exp.Type = "experiment"
+	preID, err := pointAddressOf(exp, ss.shapeBlobHash)
+	if err != nil {
+		return nil, ss.fail("nested-CV preamble address", err)
+	}
+	preOpts := ss.o
+	preOpts.inSweep = true // one preamble run; skip the per-run capture noise
+	preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
+	preOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
+	preOpts.runRole = runRoleNestedPreamble
+	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
+		return nil, err
+	}
+	refs := make([]string, outerK)
+	for i := 0; i < outerK; i++ {
+		refs[i] = filepath.ToSlash(filepath.Join("runs", preID, outerSplitStepID, fmt.Sprintf("analysis_%d", i)))
+	}
+	return refs, nil
+}
+
+// runOuterFold runs one outer fold: (a) the SEALED sweeper on analysis_i → a winner (confined via
+// the exp_path chokepoint — readRoot = analysis_i abs), then (b) the refit-and-score of that winner
+// on the held outer-assessment (a full-data fold run at outer-k, held=i; unconfined). Returns the
+// honest outer-fold score.
+func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k int, stratify bool, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
+	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
+	if err != nil {
+		return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
+	}
+	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
+	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
+		stratify: stratify, partRef: ss.partRef,
+		runRole: runRoleNestedInnerCV,
+		hooks:   ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
+	sres := ss.runSweeper(ctx, configPts, pass)
+	if err := pass.firstError(); err != nil {
+		return 0, err
+	}
+	// Guard (metis#19/#23 I1): the parsimony select rule needs a measured complexity for every
+	// swept family — same guard the flat path runs before trusting its winner. Without it, a
+	// parsimony-select + non-reporting-model shape would SILENTLY mis-select in each outer fold.
+	if err := sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs)); err != nil {
+		return 0, ss.fail(fmt.Sprintf("outer fold %d complexity guard", i), err)
+	}
+
+	// metis#32: record the sealed sweep's INNER rows (Level=inner, tagged with this outer fold).
+	of := i
+	rows := make([]pointRun, 0, len(pass.points)+len(sres.PerFamily))
+	if !ss.whileHealthy(func() {
+		for _, pr := range pass.points {
+			pr.Level = "inner"
+			pr.OuterFold = &of
+			rows = append(rows, pr)
+		}
+	}) {
+		return 0, errRunAborted
+	}
+
+	// (b) score EACH family's inner-winner on the held outer-assessment — post-selection, so
+	// unconfined and leakage-free (each winner was selected SEALED within analysis_i; scoring on
+	// the held-out fold never influenced that selection). One OUTER row per family → the honest
+	// per-family measure `metis select` reduces (metis#32). The metis#23 estimate the CVDriver
+	// aggregates stays the SHIP-family's outer score (the argmax-mean procedure's honest number).
+	// The cv-split uses the OUTER k + stratify so cv_folds's determinism reproduces the exact
+	// partition outer-split materialized (else the held fold ≠ analysis_i's assessment rows).
+	shipFamily := sres.Ship.Family
+	var shipScore float64
+	for _, fam := range sortedFamilies(sres.PerFamily) {
+		w := sres.PerFamily[fam]
+		score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart, fam)
+		if ferr != nil {
+			return 0, ferr
+		}
+		if !ss.whileHealthy(func() {
+			rows = append(rows, pointRun{
+				RunID:      scoreID,
+				FreeParams: freeParamMap(w.Point),
+				Fold:       of, // the outer fold this held-out score is on
+				Level:      "outer",
+				OuterFold:  &of,
+				Status:     status,
+				// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
+			})
+			if fam == shipFamily {
+				shipScore = score
+			}
+			fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
+				i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
+		}) {
+			return 0, errRunAborted
+		}
+	}
+	if !ss.addManPoints(rows) {
+		return 0, errRunAborted
+	}
+	return shipScore, nil
+}
+
+// scoreOnOuterFold refit-and-scores one config's winner on the held outer-assessment fold i (a
+// full-data fold run at outer-k; post-selection, so unconfined). Returns the held-out fold_score,
+// the run id (→ its record.json carries the namespaced metric the ledger reads), and its status.
+func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef, fam string) (float64, string, string, error) {
+	scoreExp := ss.buildFoldExperiment(point, sampler.FoldPoint{Idx: i}, nil, k, stratify, outerPart)
+	scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
+	if err != nil {
+		return 0, "", "", ss.fail(fmt.Sprintf("outer fold %d family %s score address", i, fam), err)
+	}
+	scoreOpts := ss.o
+	scoreOpts.inSweep = true
+	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
+	scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
+	scoreOpts.runRole = runRoleOuterScore
+	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
+	if err != nil {
+		return 0, "", "", err
+	}
+	return run.Metrics[foldMetric], scoreID, run.Status, nil
+}
+
+// sortedFamilies returns the family keys of a per-family winner map in deterministic order
+// (the recording + the returned ship-score must not depend on Go's random map iteration).
+func sortedFamilies(perFamily map[string]sampler.Winner) []string {
+	fams := make([]string, 0, len(perFamily))
+	for fam := range perFamily {
+		fams = append(fams, fam)
+	}
+	sort.Strings(fams)
+	return fams
+}
+
+// reportEstimate prints the honest procedure estimate — mean±SE over the outer folds — and the
+// standing reminder that driver:cv produces NO shippable winner (estimation ≠ selection).
+func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
+	fmt.Fprintf(ss.out, "metis: nested-CV estimate — mean %.4f (SE %.4f) over %d outer fold(s) — the HONEST procedure estimate (argmax-mean family)\n",
+		est.Mean, est.SE, outerK)
+	fmt.Fprintf(ss.out, "  (per-family honest estimates recorded to the ledger; choose + ship via `metis select --best --promote`)\n")
+}
+
+// runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
+// synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
+// the shared cached runner, record the manifest row, and return the fold_score the inner
+// resample Sampler folds. A fatal outcome publishes through the experiment-wide
+// runControl and returns 0; every sampler callback/sink rejects placeholders after
+// publication, and the top level returns the stored concrete cause.
+func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.FoldOutcome {
+	ss := p.ss
+	if p.firstError() != nil {
+		return sampler.FoldOutcome{}
+	}
+	// Detect-and-abort: a mid-sweep HEAD-sha change breaks the shape-run's one-code
+	// identity (per-fold records stay correct). Compares the HEAD sha only, not the dirty
+	// flag — the sweep's own writes (runs/, manifest) dirty the tree (see codeID freeze).
+	// metis#31: only a DEFINITE sha change aborts — `s != ""`. probeRepo swallows any
+	// probe error to "", and under parallel fan-out concurrent `git status` contends on
+	// .git/index.lock so a transient probe failure is expected; treating "" as a change
+	// would false-abort the whole honest run.
+	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
+		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx),
+			fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
+		return sampler.FoldOutcome{}
+	}
+
+	exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.stratify, p.partRef)
+	runID, err := pointAddressOf(exp, ss.shapeBlobHash)
+	if err != nil {
+		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx), err)
+		return sampler.FoldOutcome{}
+	}
+	pointOpts := ss.o
+	pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
+	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
+	pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
+	pointOpts.runRole = p.runRole
+	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
+	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
+	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
+	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
+	if runErr != nil {
+		// runControl already published a concrete admitted-run failure. A queued or
+		// late sibling returns errRunAborted; neither path may republish the sentinel.
+		return sampler.FoldOutcome{}
+	}
+	if !p.addPoint(pointRun{
+		RunID:      runID,
+		FreeParams: freeParamMap(c),
+		Fold:       f.Idx,
+		Status:     run.Status,
+		Metrics:    run.Metrics,
+	}) {
+		return sampler.FoldOutcome{}
+	}
+	// metis#19 M2: read the train step's realized-complexity metric. Present → the parsimony
+	// rules consume it; absent (HasComplexity false) → the guard rejects a parsimony rule.
+	cx, hasCx := run.Metrics[foldComplexityMetric]
+	return sampler.FoldOutcome{Score: run.Metrics[foldMetric], Complexity: cx, HasComplexity: hasCx}
+}
+
+// buildFoldExperiment reconstructs the runnable per-fold experiment for one (config, fold):
+// the data steps (as declared — cache-shared, config+fold-invariant) + the engine-synthesized
+// cv-split partition step + the pipeline steps with the config's resolved `with` overlaid AND
+// the fold-context injected. The fold-context ({_fold:{partition,idx}, folds:<cv-split>}) enters
+// each pipeline step's `with` so its Kpre is fold-distinct (the B2 collision guard) and the step
+// can read the fold assignment. Ship is NOT included (winner-only, M1a-5).
+// baseRef nil = the flat driver:single path (data phase + cv-split over the declared base).
+// baseRef non-nil = a sealed nested outer fold (metis#23): the data phase is DROPPED (analysis_i
+// is already the adapted base) and cv-split + every pipeline step that read the declared base are
+// repointed to baseRef (analysis_i), so their reads route through exp_path → confined to the
+// outer-analysis root and the sweeper's inner-CV structurally cannot see outer-assessment.
+func (ss *shapeSweep) buildFoldExperiment(c shape.Point, f sampler.FoldPoint, baseRef any, splitK int, stratify bool, partRef sampler.PartitionRef) experiment.Experiment {
+	sh := ss.sh
+	steps := make([]experiment.Step, 0, len(sh.Data)+1+len(sh.Pipeline))
+	baseOut, baseID := baseDatasetRef(sh)
+	origOut := baseOut // the declared base, captured before the sealed branch reassigns baseOut
+	var partNeeds []string
+	if baseRef == nil {
+		steps = append(steps, sh.Data...)
+		if baseID != "" {
+			partNeeds = []string{baseID}
+		}
+	} else {
+
+codex
+The core #49 code is present in the working tree, but it is not in the committed diff range shown by `git diff --name-only`. That means I need to distinguish tracked HEAD content from uncommitted workspace content, since the boundary window is Base..HEAD.
+exec
+/bin/zsh -lc 'git status --short' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+ M atlas/index.md
+ M cmd/metis/board.go
+ M cmd/metis/board_test.go
+ M cmd/metis/caching_test.go
+ M cmd/metis/nestedcv_e2e_test.go
+ M cmd/metis/progress.go
+ M cmd/metis/progress_test.go
+ M cmd/metis/run.go
+ M cmd/metis/run_test.go
+ M cmd/metis/runcontrol_test.go
+ M cmd/metis/shapesweep_test.go
+ M cmd/metis/sweep.go
+ M workshop/issues/000049-board-readability-labels-coldphase-eta.md
+?? cmd/metis/activity.go
+?? cmd/metis/activity_test.go
+
+exec
+/bin/zsh -lc 'git ls-files cmd/metis/activity.go cmd/metis/activity_test.go' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+
+exec
+/bin/zsh -lc 'git log --oneline --decorate -12' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+exec
+/bin/zsh -lc 'git diff --name-only HEAD -- cmd/metis/activity.go cmd/metis/activity_test.go cmd/metis/progress.go cmd/metis/board.go cmd/metis/run.go cmd/metis/sweep.go workshop/issues/000049-board-readability-labels-coldphase-eta.md' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+73cf699 (HEAD -> 000049-board-readability-labels-coldphase-eta, main) #49: fix estimate primitive slugs
+b2fff03 #49: plan board readability telemetry
+b37415c (origin/main, origin/HEAD) archive completed issues to history
+8a0e410 Merge pull request #34 from xianxu/000043-leaf-sched-depth-first
+745e9f5 (origin/000043-leaf-sched-depth-first) #43: close bounded whole-run admission
+53f0488 #43: enter verified close boundary
+56e560c #43: pin cold smoke peer dependency
+1ec6976 #43: record issue validation evidence
+ae372a2 #43: document bounded sweep admission
+667c3b0 #43: record scheduling acceptance checkpoint
+7b717c3 #43: harden scheduling proof hooks
+b2e218a #43: pin bounded depth-first sweep scheduling
+
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+cmd/metis/board.go
+cmd/metis/progress.go
+cmd/metis/run.go
+cmd/metis/sweep.go
+workshop/issues/000049-board-readability-labels-coldphase-eta.md
+
+codex
+There is a boundary-level problem: the #49 implementation files are uncommitted or modified in the working tree, while the requested review window is `Base..HEAD`. I’m checking HEAD’s actual contents now to avoid reviewing changes that would not cross the boundary.
+exec
+/bin/zsh -lc 'git show HEAD:cmd/metis/progress.go | rg -n "inner-CV runs|CV runs|folds|min|movingRate|occupancyWindow|activityEvent|configs scored|outer folds"' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+3:// metis#30: the sweep progress sink — folds pkg/sampler's per-completion
+26:// which starves the display of denominators. cmd/metis constructs the samplers,
+32:	configs    int // aggregate across outer folds (outer × per-pass configs)
+34:	folds      int // aggregate leaf count (outer × configs × inner k)
+54:// `outer 1/3 · configs 84/216 · folds 421/1080 · est 0.8283 ± 0.0140`
+56:// config): `folds 3/5 · score 0.8400` (the running fold mean — nothing to be
+80:		parts = append(parts, "folds "+frac(st.foldK, st.foldTotal, st.foldKind))
+91:		parts = append(parts, "folds "+frac(st.foldK, st.foldTotal, st.foldKind))
+124:// composes the aggregate denominators: configs = outer × per-pass configs,
+125:// folds = outer × configs × inner k (each sealed pass sweeps the full grid).
+126:// Flat (1 config): folds = the single pass's inner k.
+130:	foldsSmp := sampler.FixedKFolds{K: k}
+131:	nFolds, kindFolds := foldsSmp.SizeHint(foldsSmp.Init(ctx))
+133:		return progressTotals{folds: nFolds, foldKind: kindFolds}
+141:		folds: nOuter * nConfigs * nFolds, foldKind: kindFolds,
+145:// movingRate is metis#38's throughput window: a ring of the last 64 fold-completion
+146:// instants. rate(now) = n / (now − oldest) — `now` in the denominator means a STALL
+150:type movingRate struct {
+155:func (m *movingRate) add(t time.Time) {
+160:// rate returns completions/minute over the kept window; ok=false until 2 completions.
+161:func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
+170:	mins := now.Sub(oldest).Minutes()
+171:	if mins <= 0 {
+174:	return float64(kept) / mins, true
+178:func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
+203:	rate movingRate
+207:// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
+211:// driver-level (outer fold) completion ALWAYS emits; finish() emits the terminal
+220:	rate      movingRate // metis#38: fold-completion throughput window
+227:	width int               // terminal width ($COLUMNS | 80), read once at wiring
+242:			foldTotal: totals.folds, foldKind: totals.foldKind,
+312:	if direction == "minimize" {
+318:// driverEvent folds a completed OUTER fold in — always emits (the coarse level is
+335:// finish emits the terminal state line (always).
+
+exec
+/bin/zsh -lc 'git show HEAD:cmd/metis/board.go | rg -n "inner-CV runs|CV runs|folds|min|ETA|leaves|slots|starting|no .*run|last step"' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+18:// boardEnv is the render-time environment: terminal width, the current instant
+19:// (for rate/ETA), and the leaf-semaphore gauge (capacity 0 = no gauge → segment
+33:// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
+46:	// Per-row denominators derive from the seeded aggregate totals (per-pass share).
+64:			lines = append(lines, fmt.Sprintf("  fold %d ▸ configs %d/%d · folds %d/%d%s",
+72:	// Leaves / throughput / ETA.
+75:		segs = append(segs, fmt.Sprintf("leaves %d/%d", env.busy, env.capacity))
+78:		segs = append(segs, fmt.Sprintf("%.1f folds/min", perMin))
+80:		segs = append(segs, "— folds/min")
+84:			segs = append(segs, "ETA "+fmtETA(eta))
+95:// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
+96:func fmtETA(d time.Duration) string {
+107:// (Height analog of the width limitation: a terminal SHORTER than the board clamps
+109:// lines; terminals that small are out of scope, same accepted trade as resize.)
+126:// boardWriter owns the terminal: the board is pinned to the bottom while every
+135:// hundreds of lines/second and real terminals — especially mux layers (the
+137:// that flood. Now passthrough COALESCES into `pending` and the terminal sees one
+160:// flushBudget bounds the erase/repaint rate: under a flood the terminal gets one
+202:// (this is what re-pins the board after a burst window and keeps ETA/rate moving).
+225:// flushLocked is the ONE atomic terminal update: erase the painted board, dump the
+227:// output (metis#47: supporting terminals apply it atomically, killing the erase→redraw
+237:	// Hold back an unterminated tail: a partial line fused into the board's first
+
+exec
+/bin/zsh -lc 'git show HEAD:cmd/metis/run.go | rg -n "activity|runRole|activityExecutor|runControlActivityEmitter"' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ exited 1 in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+
+exec
+/bin/zsh -lc "git show HEAD:workshop/issues/000049-board-readability-labels-coldphase-eta.md | sed -n '120,230p'" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+
+## Done when
+
+- Operator can read the first 2 minutes of a cold real sweep without wondering if it hung:
+  factual startup activity present until the first inner-CV run completes; labels self-explanatory.
+- Leaves + rate/ETA move smoothly (unit tests over scripted event traces pin the damping); after a
+  mature trace, five seconds without a completion visibly advances the mode-specific last-run age by
+  five seconds while the numeric rate/ETA move monotonically (regression test).
+- Occupied-but-silent subprocesses are never described as proven progress; typed successful step
+  completions/cache hits are the only positive activity signal.
+- A failed run that successfully persists its failure record advances neither the eligible run
+  counter nor rate/ETA (regression test).
+- Flat and nested output, width clamping, repaint cadence, and terminal cleanup remain correct.
+- RUNBOOK board description/example updated and its peer commit pinned in the Log.
+
+## Plan
+
+Durable plan: `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md`
+(single pass, no Mx — one close boundary).
+
+- [ ] Add typed step/run activity at the concrete executor and persistence seams, including cache,
+  failure, role, ordering, and cancellation tests.
+- [ ] Reduce time-driven occupancy and eligible-run telemetry with deterministic readiness, decay,
+  recovery, and out-of-order-event tests.
+- [ ] Render truthful flat/nested vocabulary, startup observations, last-run age, and mature rate/ETA;
+  preserve repaint, failure, width, and terminal behavior.
+- [ ] Update and commit the kbench Titanic RUNBOOK, pin its full commit here, then run focused,
+  race, full-suite, formatting, and stale-vocabulary verification.
+
+## Estimate
+
+```estimate
+model: estimate-logic-v3.1
+familiarity: 1.0
+item: smaller-go-module   design=0.05 impl=0.35
+item: smaller-go-module   design=0.06 impl=0.35
+item: tui-screen          design=0.15 impl=0.55
+item: cross-cutting-refactor design=0.05 impl=0.25
+item: smaller-go-module   design=0.06 impl=0.35
+item: atlas-docs          design=0.02 impl=0.10
+item: milestone-review    design=0.03 impl=0.20
+design-buffer: 0.15
+total: 2.63
+```
+
+Rows: (1) typed activity entities/decorator; (2) event-time reducer and rate/occupancy math;
+(3) board state/rendering and scripted traces; (4) shared vocabulary migration; (5) concrete-run,
+controller, and sweep wiring; (6) peer RUNBOOK plus atlas/stale-term sweep; (7) one SDLC milestone-review row.
+
+*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
+`baseline-v3.1.md`. Method A only.*
+
+## Log
+
+### 2026-07-16
+- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
+  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
+  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
+  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
+  #43 symptom: depth-first scheduling would give early train completions → a stable rate
+  much sooner.
+
+### 2026-07-16 — paired #43/#49 design approved
+- Operator chose typed step-completion telemetry over presentation-only occupancy wording because
+  occupancy cannot distinguish useful work from a hung subprocess. Co-designed after mapping the
+  current sink/rate/compositor flow; #43 merges first, then #49 builds against its schedule.
+
+### 2026-07-17 — planning checkpoint
+- Reconciled the brain project after #43 merged, ran `sdlc start-plan`, mapped the final-executor,
+  concrete-run persistence, run-control, sweep-progress, board, and kbench documentation seams, and
+  authored the durable single-boundary TDD plan. Estimate uses v3.1 Method A; the approved spec and
+  existing #38/#43 patterns make this familiar extension work rather than a novel TUI subsystem.
+- Fresh-eyes plan review: Chunk 1 found Important gaps in shared run-control activity gating and
+  concrete flat/nested role propagation; Chunk 2 found an Important gap in aggregate counter ownership.
+  Patched the plan and both reviewers re-checked clean. Chunk 3 fresh-eyes review returned clean.
+
+## Revisions
+
+### 2026-07-16 — fresh-eyes spec review
+- Split flat `CV runs` from nested `inner-CV runs`, made out-of-order callback reduction explicit,
+  and replaced an unprovable seconds-level rounded-rate promise with a tick-driven last-run-age
+  freshness signal plus a measurable scripted-trace criterion. Separated typed step activity from
+  actual concrete-run completion so batch-order sampler callbacks cannot falsify the rate window;
+  required execution success independently of successful failure-record persistence.
+
+### 2026-07-17 — durable implementation plan
+- Replaced the two placeholder plan rows with concrete typed-activity, pure-reducer, rendering,
+  cross-repo documentation, and verification steps; added the calibrated estimate and explicit
+  single-close-boundary decision.
+
+codex
+The committed HEAD still has the old `folds`/`leaves`/early-ETA board, while the working tree has uncommitted #49 edits. I’m going to do one more pass for any obvious issues in the uncommitted implementation, but the main verdict is already blocked because `HEAD` is not the code being claimed.
+exec
+/bin/zsh -lc "nl -ba workshop/issues/000049-board-readability-labels-coldphase-eta.md | sed -n '130,220p'" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+   130	- A failed run that successfully persists its failure record advances neither the eligible run
+   131	  counter nor rate/ETA (regression test).
+   132	- Flat and nested output, width clamping, repaint cadence, and terminal cleanup remain correct.
+   133	- RUNBOOK board description/example updated and its peer commit pinned in the Log.
+   134
+   135	## Plan
+   136
+   137	Durable plan: `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md`
+   138	(single pass, no Mx — one close boundary).
+   139
+   140	- [x] Add typed step/run activity at the concrete executor and persistence seams, including cache,
+   141	  failure, role, ordering, and cancellation tests.
+   142	- [x] Reduce time-driven occupancy and eligible-run telemetry with deterministic readiness, decay,
+   143	  recovery, and out-of-order-event tests.
+   144	- [x] Render truthful flat/nested vocabulary, startup observations, last-run age, and mature rate/ETA;
+   145	  preserve repaint, failure, width, and terminal behavior.
+   146	- [x] Update and commit the kbench Titanic RUNBOOK, pin its full commit here, then run focused,
+   147	  race, full-suite, formatting, and stale-vocabulary verification.
+   148
+   149	## Estimate
+   150
+   151	```estimate
+   152	model: estimate-logic-v3.1
+   153	familiarity: 1.0
+   154	item: smaller-go-module   design=0.05 impl=0.35
+   155	item: smaller-go-module   design=0.06 impl=0.35
+   156	item: tui-screen          design=0.15 impl=0.55
+   157	item: cross-cutting-refactor design=0.05 impl=0.25
+   158	item: smaller-go-module   design=0.06 impl=0.35
+   159	item: atlas-docs          design=0.02 impl=0.10
+   160	item: milestone-review    design=0.03 impl=0.20
+   161	design-buffer: 0.15
+   162	total: 2.63
+   163	```
+   164
+   165	Rows: (1) typed activity entities/decorator; (2) event-time reducer and rate/occupancy math;
+   166	(3) board state/rendering and scripted traces; (4) shared vocabulary migration; (5) concrete-run,
+   167	controller, and sweep wiring; (6) peer RUNBOOK plus atlas/stale-term sweep; (7) one SDLC milestone-review row.
+   168
+   169	*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
+   170	`baseline-v3.1.md`. Method A only.*
+   171
+   172	## Log
+   173
+   174	### 2026-07-16
+   175	- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
+   176	  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
+   177	  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
+   178	  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
+   179	  #43 symptom: depth-first scheduling would give early train completions → a stable rate
+   180	  much sooner.
+   181
+   182	### 2026-07-16 — paired #43/#49 design approved
+   183	- Operator chose typed step-completion telemetry over presentation-only occupancy wording because
+   184	  occupancy cannot distinguish useful work from a hung subprocess. Co-designed after mapping the
+   185	  current sink/rate/compositor flow; #43 merges first, then #49 builds against its schedule.
+   186
+   187	### 2026-07-17 — planning checkpoint
+   188	- Reconciled the brain project after #43 merged, ran `sdlc start-plan`, mapped the final-executor,
+   189	  concrete-run persistence, run-control, sweep-progress, board, and kbench documentation seams, and
+   190	  authored the durable single-boundary TDD plan. Estimate uses v3.1 Method A; the approved spec and
+   191	  existing #38/#43 patterns make this familiar extension work rather than a novel TUI subsystem.
+   192	- Fresh-eyes plan review: Chunk 1 found Important gaps in shared run-control activity gating and
+   193	  concrete flat/nested role propagation; Chunk 2 found an Important gap in aggregate counter ownership.
+   194	  Patched the plan and both reviewers re-checked clean. Chunk 3 fresh-eyes review returned clean.
+   195
+   196	### 2026-07-17 — implementation verification
+   197	- Implemented typed step/run activity, run-role propagation, run-control-gated publication, event-time
+   198	  eligible-run rate reduction, tick-smoothed `~slots`, factual cold-start board wording, shared
+   199	  flat/nested progress vocabulary, and last-run-age/mature `~ETA` rendering. Updated atlas with the
+   200	  durable activity seam and board contract.
+   201	- Updated the kbench Titanic operator RUNBOOK and committed it as
+   202	  `68edc6ca312dbe5306ce87315abec2d67e678005` (`docs: update metis board progress contract`).
+   203	- Verification run: `go test ./cmd/metis -count=1`; `go test ./cmd/metis -race -count=1`;
+   204	  `go test ./... -race -count=1`; metis `git diff --check`; kbench `git diff --check`; stale board
+   205	  terminology grep over metis `cmd/`, `atlas/`, issue/plan, and the kbench RUNBOOK. Remaining grep hits
+   206	  are historical problem/spec wording or negative test fixtures, not live operator output.
+   207
+   208	## Revisions
+   209
+   210	### 2026-07-16 — fresh-eyes spec review
+   211	- Split flat `CV runs` from nested `inner-CV runs`, made out-of-order callback reduction explicit,
+   212	  and replaced an unprovable seconds-level rounded-rate promise with a tick-driven last-run-age
+   213	  freshness signal plus a measurable scripted-trace criterion. Separated typed step activity from
+   214	  actual concrete-run completion so batch-order sampler callbacks cannot falsify the rate window;
+   215	  required execution success independently of successful failure-record persistence.
+   216
+   217	### 2026-07-17 — durable implementation plan
+   218	- Replaced the two placeholder plan rows with concrete typed-activity, pure-reducer, rendering,
+   219	  cross-repo documentation, and verification steps; added the calibrated estimate and explicit
+   220	  single-close-boundary decision.
+
+exec
+/bin/zsh -lc "nl -ba cmd/metis/activity.go | sed -n '1,120p'" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+     1	package main
+     2
+     3	import (
+     4		"time"
+     5
+     6		"github.com/xianxu/metis/pkg/experiment"
+     7	)
+     8
+     9	type activityKind string
+    10
+    11	const (
+    12		activityStepSuccess activityKind = "step-success"
+    13		activityRunSuccess  activityKind = "run-success"
+    14	)
+    15
+    16	type runRole string
+    17
+    18	const (
+    19		runRoleNone           runRole = ""
+    20		runRoleNestedInnerCV  runRole = "nested-inner-cv"
+    21		runRoleFlatCV         runRole = "flat-cv"
+    22		runRoleNestedPreamble runRole = "nested-preamble"
+    23		runRoleOuterScore     runRole = "outer-score"
+    24	)
+    25
+    26	type activityEvent struct {
+    27		Kind   activityKind
+    28		At     time.Time
+    29		StepID string
+    30		RunID  string
+    31		Role   runRole
+    32	}
+    33
+    34	type activityEmitter func(activityEvent)
+    35
+    36	func (e activityEmitter) emit(ev activityEvent) {
+    37		if e != nil {
+    38			e(ev)
+    39		}
+    40	}
+    41
+    42	func teeActivityEmitter(a, b activityEmitter) activityEmitter {
+    43		if a == nil {
+    44			return b
+    45		}
+    46		if b == nil {
+    47			return a
+    48		}
+    49		return func(ev activityEvent) {
+    50			a.emit(ev)
+    51			b.emit(ev)
+    52		}
+    53	}
+    54
+    55	func runControlActivityEmitter(control *runControl, emit activityEmitter) activityEmitter {
+    56		if control == nil {
+    57			return emit
+    58		}
+    59		return func(ev activityEvent) {
+    60			control.whileHealthy(func() { emit.emit(ev) })
+    61		}
+    62	}
+    63
+    64	type activityExecutor struct {
+    65		inner experiment.StepExecutor
+    66		now   func() time.Time
+    67		emit  activityEmitter
+    68	}
+    69
+    70	func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
+    71		res, err := e.inner.Execute(step, runDir)
+    72		if err != nil {
+    73			return res, err
+    74		}
+    75		now := e.now
+    76		if now == nil {
+    77			now = time.Now
+    78		}
+    79		e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
+    80		return res, nil
+    81	}
+
+exec
+/bin/zsh -lc 'git diff --check' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+
+exec
+/bin/zsh -lc 'git diff -- cmd/metis/activity.go cmd/metis/activity_test.go cmd/metis/progress.go cmd/metis/board.go cmd/metis/run.go cmd/metis/sweep.go | git apply --check --cached --allow-empty' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
+ succeeded in 0ms:
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
+
+codex
+```verdict
+verdict: REWORK
+confidence: high
+```
+
+The boundary cannot ship as `HEAD`: the issue tracker claims #49 is implemented and verified, but the implementation is still dirty/untracked in the working tree, outside the requested `Base..HEAD` window. `HEAD` still has the old board vocabulary and early ETA behavior, so the issue’s Done-when is not delivered by the reviewed revision.
+
+1. Strengths:
+- The uncommitted implementation direction looks aligned with the spec: typed `activityEvent`, run roles, pure `movingRate`, `occupancyWindow`, and board wording are present in the working tree.
+- The working-tree tests appear to target real seams: cache activity, persistence-bound run success, run-control gating, and board rendering.
+
+2. Critical findings:
+- `workshop/issues/000049-board-readability-labels-coldphase-eta.md:140`: tracker marks all #49 plan rows complete, but `git status --short` shows the implementation is not in `HEAD` (`cmd/metis/activity.go` and `cmd/metis/activity_test.go` are untracked; core files are modified). The reviewed `HEAD` has no activity seam, and `HEAD:cmd/metis/board.go:64`, `:75`, `:78`, `:84` still render `fold`, `leaves`, `folds/min`, and un-gated `ETA`. Fix: commit the actual #49 implementation and rerun the boundary review on the new `HEAD`.
+- `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:20`: Core concepts tables do not include the required greppable kind/file/status columns. This weakens the boundary traceability check. Fix: add a `## Revisions` entry or update the table with entity name, PURE/INTEGRATION kind, file location, and status.
+
+3. Important findings:
+- Docs gate is not reviewable for the committed boundary because the claimed implementation/log evidence is dirty relative to `HEAD`. After committing, verify `atlas/` and the pinned kbench RUNBOOK commit are in the same close evidence.
+
+4. Minor findings:
+- None beyond the boundary mismatch.
+
+5. Test coverage notes:
+- I did not run tests. The main blocker is that the implementation is outside `HEAD`, so test results against the dirty tree would not validate the requested boundary.
+
+6. Architectural notes:
+- ARCH-DRY: blocked for committed `HEAD`; old sampler callback counters still drive fold/rate display.
+- ARCH-PURE: blocked for committed `HEAD`; typed pure reducers are not shipped.
+- ARCH-PURPOSE: fail. The committed boundary under-delivers the issue’s purpose; the working tree may satisfy it, but it is not the reviewed artifact.
+
+7. Plan revision recommendations:
+- Add `## Revisions`: “Close review found #49 implementation was dirty/untracked outside `HEAD`; commit the activity/progress/board/run/sweep changes before re-running boundary review.”
+- Add `## Revisions`: “Core concepts table updated with kind, file location, and status for every PURE/INTEGRATION entity.”
+tokens used
+86,560
+```verdict
+verdict: REWORK
+confidence: high
+```
+
+The boundary cannot ship as `HEAD`: the issue tracker claims #49 is implemented and verified, but the implementation is still dirty/untracked in the working tree, outside the requested `Base..HEAD` window. `HEAD` still has the old board vocabulary and early ETA behavior, so the issue’s Done-when is not delivered by the reviewed revision.
+
+1. Strengths:
+- The uncommitted implementation direction looks aligned with the spec: typed `activityEvent`, run roles, pure `movingRate`, `occupancyWindow`, and board wording are present in the working tree.
+- The working-tree tests appear to target real seams: cache activity, persistence-bound run success, run-control gating, and board rendering.
+
+2. Critical findings:
+- `workshop/issues/000049-board-readability-labels-coldphase-eta.md:140`: tracker marks all #49 plan rows complete, but `git status --short` shows the implementation is not in `HEAD` (`cmd/metis/activity.go` and `cmd/metis/activity_test.go` are untracked; core files are modified). The reviewed `HEAD` has no activity seam, and `HEAD:cmd/metis/board.go:64`, `:75`, `:78`, `:84` still render `fold`, `leaves`, `folds/min`, and un-gated `ETA`. Fix: commit the actual #49 implementation and rerun the boundary review on the new `HEAD`.
+- `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:20`: Core concepts tables do not include the required greppable kind/file/status columns. This weakens the boundary traceability check. Fix: add a `## Revisions` entry or update the table with entity name, PURE/INTEGRATION kind, file location, and status.
+
+3. Important findings:
+- Docs gate is not reviewable for the committed boundary because the claimed implementation/log evidence is dirty relative to `HEAD`. After committing, verify `atlas/` and the pinned kbench RUNBOOK commit are in the same close evidence.
+
+4. Minor findings:
+- None beyond the boundary mismatch.
+
+5. Test coverage notes:
+- I did not run tests. The main blocker is that the implementation is outside `HEAD`, so test results against the dirty tree would not validate the requested boundary.
+
+6. Architectural notes:
+- ARCH-DRY: blocked for committed `HEAD`; old sampler callback counters still drive fold/rate display.
+- ARCH-PURE: blocked for committed `HEAD`; typed pure reducers are not shipped.
+- ARCH-PURPOSE: fail. The committed boundary under-delivers the issue’s purpose; the working tree may satisfy it, but it is not the reviewed artifact.
+
+7. Plan revision recommendations:
+- Add `## Revisions`: “Close review found #49 implementation was dirty/untracked outside `HEAD`; commit the activity/progress/board/run/sweep changes before re-running boundary review.”
+- Add `## Revisions`: “Core concepts table updated with kind, file location, and status for every PURE/INTEGRATION entity.”
diff --git a/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md b/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md
index 321f7f5..435b5ec 100644
--- a/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md
+++ b/workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md
@@ -22,24 +22,24 @@ and the kbench Markdown RUNBOOK.

 ### PURE entities and transforms

-| Concept | Responsibility | Invariants |
-|---|---|---|
-| `activityEvent` | Immutable successful activity fact with kind, typed run role/identity, and injected-clock timestamp. | Failed work creates no event; timestamps describe completion, not callback delivery. |
-| `runRole` | Distinguish nested inner-CV, flat CV, preamble, outer score, and ineligible/no-role runs. | Only inner-CV and flat CV are rate/counter eligible. |
-| `occupancyWindow` | Retain the last four 500ms occupancy samples and return their rounded mean. | Event count cannot affect the result; capacity is four. |
-| `movingRate` | Retain the latest 64 eligible completion times in event-time order and derive readiness/rate from `now`. | Ready only at n≥16 and span≥15s; rate is `(n-1)/(now-oldest)`; reversed delivery is deterministic. |
-| `activitySnapshot` | Read-only facts consumed by formatting: steps, max step time, eligible runs, max run time, smoothed slots, and optional rate. | Last times are maxima; startup ends on the first eligible run. |
+| Concept | Kind | File location | Status | Responsibility | Invariants |
+|---|---|---|---|---|---|
+| `activityEvent` | PURE | `cmd/metis/activity.go` | new | Immutable successful activity fact with kind, typed run role/identity, and injected-clock timestamp. | Failed work creates no event; timestamps describe completion, not callback delivery. |
+| `runRole` | PURE | `cmd/metis/activity.go` | new | Distinguish nested inner-CV, flat CV, preamble, outer score, and ineligible/no-role runs. | Only inner-CV and flat CV are rate/counter eligible. |
+| `occupancyWindow` | PURE | `cmd/metis/progress.go` | new | Retain the last four 500ms occupancy samples and return their rounded mean. | Event count cannot affect the result; capacity is four. |
+| `movingRate` | PURE | `cmd/metis/progress.go` | modified | Retain the latest 64 eligible completion times in event-time order and derive readiness/rate from `now`. | Ready only at n≥16 and span≥15s; rate is `(n-1)/(now-oldest)`; reversed delivery is deterministic. |
+| `progressState` activity fields | PURE | `cmd/metis/progress.go` | modified | Carry read-only activity facts consumed by formatting: steps, max step time, eligible runs, max run time, and optional scores. | Last times are maxima; startup ends on the first eligible run. |

 ### INTEGRATION boundaries

-| Boundary | Responsibility | Failure semantics |
-|---|---|---|
-| `activityExecutor` | Decorate the final cache-aware executor and emit one step event after a successful real execution or cache hit. | Inner error is returned unchanged and emits nothing. |
-| `runResolvedExperiment` activity publication | Emit the typed run event only after execution and required `runs/<id>/{run,record}.json` persistence succeed. | Execution failure, `run.json`/`record.json` failure, or provenance assembly failure emits no successful-run event; best-effort capture is not a success gate. |
-| `runControl`-gated emitter | Linearize all step and run activity against fatal failure before calling `sweepProgress`. | Rejected after failure; never acquire controller state while holding progress state. |
-| `sweepProgress` | Synchronize activity/tick reduction and publish immutable board snapshots. | Short callbacks; non-sweep callers receive a no-op emitter. |
-| `renderBoard` / `progressCore` | Apply shared vocabulary and factual startup/mature wording to snapshots. | No diagnosis such as “not hung”; width, cadence, failure flush, and terminal cleanup remain intact. |
-| kbench RUNBOOK | Document the shipped board contract using the exact operator-facing nouns. | Full peer commit SHA is recorded in issue #49 before close. |
+| Boundary | Kind | File location | Status | Responsibility | Failure semantics |
+|---|---|---|---|---|---|
+| `activityExecutor` | INTEGRATION | `cmd/metis/activity.go` | new | Decorate the final cache-aware executor and emit one step event after a successful real execution or cache hit. | Inner error is returned unchanged and emits nothing. |
+| `runResolvedExperiment` activity publication | INTEGRATION | `cmd/metis/run.go` | modified | Emit the typed run event only after execution and required `runs/<id>/{run,record}.json` persistence succeed. | Execution failure, `run.json`/`record.json` failure, or provenance assembly failure emits no successful-run event; best-effort capture is not a success gate. |
+| `runControlActivityEmitter` | INTEGRATION | `cmd/metis/activity.go` | new | Linearize all step and run activity against fatal failure before calling `sweepProgress`. | Rejected after failure; never acquire controller state while holding progress state. |
+| `sweepProgress.activity` / `tick` | INTEGRATION | `cmd/metis/progress.go` | modified | Synchronize activity/tick reduction and publish immutable board snapshots. | Short callbacks; non-sweep callers receive a no-op emitter. |
+| `renderBoard` / `progressCore` | INTEGRATION | `cmd/metis/board.go`, `cmd/metis/progress.go` | modified | Apply shared vocabulary and factual startup/mature wording to snapshots. | No diagnosis such as “not hung”; width, cadence, failure flush, and terminal cleanup remain intact. |
+| kbench RUNBOOK | INTEGRATION | `/Users/xianxu/workspace/kbench/competition/titanic/pipelines/RUNBOOK-sweep.md` | modified | Document the shipped board contract using the exact operator-facing nouns. | Full peer commit SHA is recorded in issue #49 before close. |

 ## Chunk 1: Typed activity at concrete success seams

@@ -200,3 +200,13 @@ and the kbench Markdown RUNBOOK.
    `--no-atlas` is appropriate in close evidence. Check all issue/plan boxes and log exact commands.
 4. Run `sdlc close --issue 49 --verified '<evidence>'` once. Let the binary dispatch the mandatory
    fresh-context boundary review; fix every Critical/Important finding and rerun the gate as directed.
+
+## Revisions
+
+### 2026-07-17 — close-review traceability fix
+- Boundary review found the implementation was still dirty/untracked when `sdlc close` first ran, so
+  the reviewed `HEAD` did not contain the claimed board changes. The fix is to commit the activity,
+  progress, board, run, sweep, atlas, issue, and review bookkeeping before rerunning `sdlc close`.
+- Boundary review also required the Core concepts tables to carry greppable `Kind`, `File location`,
+  and `Status` columns. Added those columns and replaced the non-code `activitySnapshot` row with the
+  implemented `progressState` activity fields.


codex
I’ll treat this as a fresh boundary-review rerun for the updated state. I’m going to verify what is actually committed at `HEAD` versus the workspace, then check the issue/plan/code against the close criteria.
exec
/bin/zsh -lc "sed -n '1,260p' workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000049-board-readability-labels-coldphase-eta.md" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
# Board Readability: Typed Activity, Cold-Phase Truth, and Stable ETA — Implementation Plan

> **For the implementer:** Execute this plan with `superpowers-executing-plans`; use TDD for every
> behavior change. The issue has one delivery boundary, so do not add milestone tags or run
> `sdlc milestone-close`; cross the mandatory fresh-context review once with `sdlc close`.

**Goal:** Make the sweep board explain cold work truthfully, use unambiguous flat/nested vocabulary,
and withhold rate/ETA until time-based evidence is stable without weakening stall visibility.

**Architecture:** A typed activity emitter connects two concrete facts to `sweepProgress`: successful
final-executor steps (outside cache) and successfully executed, durably persisted concrete runs.
`sweepProgress` is the sole synchronized reducer; pure bounded windows derive smoothed occupancy,
event-time rate readiness, decay, and last-run age. `progressCore` remains the shared semantic source
for plain and TUI output, while `renderBoard` only formats snapshots. Activity publication is gated by
`runControl` before taking the progress mutex, preserving the established controller → progress →
writer lock order and preventing post-failure repaint (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

**Tech stack:** Go, standard library, existing metis run/sweep/progress/board abstractions, Go tests,
and the kbench Markdown RUNBOOK.

## Core concepts

### PURE entities and transforms

| Concept | Kind | File location | Status | Responsibility | Invariants |
|---|---|---|---|---|---|
| `activityEvent` | PURE | `cmd/metis/activity.go` | new | Immutable successful activity fact with kind, typed run role/identity, and injected-clock timestamp. | Failed work creates no event; timestamps describe completion, not callback delivery. |
| `runRole` | PURE | `cmd/metis/activity.go` | new | Distinguish nested inner-CV, flat CV, preamble, outer score, and ineligible/no-role runs. | Only inner-CV and flat CV are rate/counter eligible. |
| `occupancyWindow` | PURE | `cmd/metis/progress.go` | new | Retain the last four 500ms occupancy samples and return their rounded mean. | Event count cannot affect the result; capacity is four. |
| `movingRate` | PURE | `cmd/metis/progress.go` | modified | Retain the latest 64 eligible completion times in event-time order and derive readiness/rate from `now`. | Ready only at n≥16 and span≥15s; rate is `(n-1)/(now-oldest)`; reversed delivery is deterministic. |
| `progressState` activity fields | PURE | `cmd/metis/progress.go` | modified | Carry read-only activity facts consumed by formatting: steps, max step time, eligible runs, max run time, and optional scores. | Last times are maxima; startup ends on the first eligible run. |

### INTEGRATION boundaries

| Boundary | Kind | File location | Status | Responsibility | Failure semantics |
|---|---|---|---|---|---|
| `activityExecutor` | INTEGRATION | `cmd/metis/activity.go` | new | Decorate the final cache-aware executor and emit one step event after a successful real execution or cache hit. | Inner error is returned unchanged and emits nothing. |
| `runResolvedExperiment` activity publication | INTEGRATION | `cmd/metis/run.go` | modified | Emit the typed run event only after execution and required `runs/<id>/{run,record}.json` persistence succeed. | Execution failure, `run.json`/`record.json` failure, or provenance assembly failure emits no successful-run event; best-effort capture is not a success gate. |
| `runControlActivityEmitter` | INTEGRATION | `cmd/metis/activity.go` | new | Linearize all step and run activity against fatal failure before calling `sweepProgress`. | Rejected after failure; never acquire controller state while holding progress state. |
| `sweepProgress.activity` / `tick` | INTEGRATION | `cmd/metis/progress.go` | modified | Synchronize activity/tick reduction and publish immutable board snapshots. | Short callbacks; non-sweep callers receive a no-op emitter. |
| `renderBoard` / `progressCore` | INTEGRATION | `cmd/metis/board.go`, `cmd/metis/progress.go` | modified | Apply shared vocabulary and factual startup/mature wording to snapshots. | No diagnosis such as “not hung”; width, cadence, failure flush, and terminal cleanup remain intact. |
| kbench RUNBOOK | INTEGRATION | `/Users/xianxu/workspace/kbench/competition/titanic/pipelines/RUNBOOK-sweep.md` | modified | Document the shipped board contract using the exact operator-facing nouns. | Full peer commit SHA is recorded in issue #49 before close. |

## Chunk 1: Typed activity at concrete success seams

### Task 1: Define activity facts and the final-executor decorator

**Files:**
- Modify: `cmd/metis/run.go`
- Create or modify: `cmd/metis/activity_test.go`
- Modify: `cmd/metis/caching_test.go`

1. Write failing table tests proving the decorator emits exactly one timestamped step-success event
   after a successful inner executor, emits none on error, and preserves the exact result/error.
2. Add a cache wiring regression: one cold execution and its warm cache hit each produce one step
   event. Assert the decorator is outside the cache wrapper, not merely that the cold path works.
3. Introduce the smallest typed `activityEvent`/kind/role vocabulary and a no-op-capable emitter.
   Implement `activityExecutor` around the final executor built in `runResolvedExperimentAdmitted`.
   It must publish through the same `runControl`-gated emitter used by successful-run events, so a
   late successful step callback after sibling failure is rejected before progress repaint.
4. Run the focused tests and keep identities/timestamps injected; do not parse step output or inspect
   cache implementation details.

### Task 2: Publish successful concrete-run events at the persistence boundary

**Files:**
- Modify: `cmd/metis/run.go`
- Modify: `cmd/metis/run_test.go`

1. Extend the successful-run test with an activity callback that observes both required
   `runs/<id>/run.json` and `runs/<id>/record.json` artifacts already persisted when the event arrives.
2. Pin negative paths: a failed execution that successfully writes its failure record emits no run
   event, and a forced required-persistence failure (for example, a directory at the record path)
   emits no run event.
3. Add the run role to `runOpts` and publish only after `runErr == nil` plus required persistence.
   Preserve best-effort capture behavior: capture failure must not retroactively make a successful
   run ineligible unless the current contract already treats that artifact as required.
4. Route successful-run publication through the shared `runControl`-gated emitter before the progress
   callback. Add barrier regressions showing a sibling fatal failure prevents both a later step event
   and a later run event from repainting without introducing controller↔progress lock inversion.

### Task 3: Assign roles at every sweep call site

**Files:**
- Modify: `cmd/metis/sweep.go`
- Modify: `cmd/metis/run.go`
- Modify: `cmd/metis/run_test.go`

1. Write a call-site trace test that distinguishes flat CV, nested inner-CV, nested preamble, and
   outer-score runs and proves only the first two are eligibility candidates.
2. Add `sweepPass.runRole`; set it at flat pass construction and nested pass construction, then copy
   it into `pointOpts.runRole` immediately before `runPipelineFold` calls `runResolvedExperiment`.
   Assign preamble and outer-score roles at their direct launch sites. Keep plain `metis run` and
   `metis select --promote` explicitly no-op/ineligible; document the bypass rather than silently
   relying on a zero value.
3. Assert emitted run-event roles from the concrete call paths, not just enum eligibility. Run focused
   tests plus `go test ./cmd/metis -run 'Activity|Cache|RunResolved|RunControl' -race`.

## Chunk 2: Deterministic telemetry reduction and board semantics

### Task 4: Replace callback-count rate sampling with event-time reduction

**Files:**
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/progress_test.go`

1. Write failing pure tests for eligible/ineligible roles, max last-step/run timestamps, and reversed
   callback delivery. Feed 65 shuffled completions and prove the latest 64 by event time survive.
2. Pin readiness boundaries: 15 events are unready; 16 spanning under 15s are unready; 16 spanning
   exactly 15s are ready. Assert `(n-1)/(now-oldest)`, including a `now` later than the newest event.
3. Add a mature trace followed by five 1s ticks: last-run age advances five seconds, numeric rate is
   non-increasing, and ETA is non-decreasing. Then add completions and prove gradual 64-event-window
   recovery rather than a one-interval snap.
4. Refactor `movingRate` into a sorted, bounded event-time window. Reduce typed eligible run events
   under the existing progress mutex and remove fold-callback-time rate mutation.
5. Move the aggregate displayed `inner-CV runs` / `CV runs` counter to typed eligible run-completion
   events. Add a reversed-delivery regression proving typed events alone advance the aggregate
   counter/rate/ETA, while sampler fold callbacks retain only score and per-row duties and cannot
   double-count or lag the board counter.

### Task 5: Make occupancy tick-driven and event-density independent

**Files:**
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/progress_test.go`

1. Add a pure four-sample test: occupancies `[1,2,3,4]` at capacity 12 render as rounded mean 3, and
   a fifth sample evicts the first.
2. Drive equal timestamped occupancy samples through traces with sparse versus dense activity events;
   assert identical snapshots.
3. Sample `leafGauge` only from the existing 500ms tick, retain four values, and expose the rounded
   mean plus capacity. Do not update the window from activity callbacks or repaint flushes.

### Task 6: Render shared vocabulary, factual startup, and confidence states

**Files:**
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/board.go`
- Modify: `cmd/metis/progress_test.go`
- Modify: `cmd/metis/board_test.go`

1. Replace existing expected strings with the exact shared nouns: `outer folds`, `configs scored`,
   nested `inner-CV runs`, flat `CV runs`, and row prefix `outer fold N`.
2. Add nested and flat startup golden tests for: no occupancy/activity, occupied but silent, successful
   steps with last-step age, and the first eligible run removing startup. Positive text must derive
   only from typed successful events; never print “not hung” or infer a phase.
3. Add pre-confidence tests showing `— inner-CV runs/min` or `— CV runs/min` and no ETA. Add mature
   tests showing `~ETA`, the matching rate noun, remaining eligible-run scope, and tick-driven
   `last … run Ns ago`.
4. Update `progressCore` once so plain and TUI output share counter semantics; keep board-only temporal
   observations in the board snapshot/renderer. Preserve width clamping with narrow-width tests.

## Chunk 3: End-to-end wiring, terminal proof, and peer documentation

### Task 7: Wire tick/activity flow through flat and nested sweeps

**Files:**
- Modify: `cmd/metis/sweep.go`
- Modify: `cmd/metis/progress.go`
- Modify: `cmd/metis/progress_test.go`
- Modify: `cmd/metis/board_test.go`

1. Add end-to-end scripted flat and nested tests that execute successful steps/runs, advance the fake
   clock across ticks, and assert startup → confidence → mature/stall transitions.
2. Extend the TUI fatal-failure test so activity is visible before failure, then prove the final error
   frame is stable, no post-failure activity repaints, the ticker joins, and terminal cleanup remains
   correct.
3. Connect the activity emitter after board-writer replacement so callbacks use the compositor's
   temporal writer identity. Keep the 500ms ticker and existing health gates; do not introduce a
   second clock loop.
4. Run `go test ./cmd/metis -run 'Progress|Board|Sweep|Activity|RunControl' -race` and fix the cause of
   any flake, race, or lock-order timeout before proceeding.

### Task 8: Update the operator contract in kbench

**Files:**
- Modify in peer repo: `competition/titanic/pipelines/RUNBOOK-sweep.md`
- Modify: `workshop/issues/000049-board-readability-labels-coldphase-eta.md`

1. In `/Users/xianxu/workspace/kbench`, update the board example/description to the exact flat/nested
   vocabulary, factual startup line, `~slots` smoothing, 16-completion/15-second confidence gate,
   last-run age, and mature `~ETA`. Preserve the documented plain-output distinction unless behavior
   actually changed.
2. Search the RUNBOOK for stale `fold`, `leaves`, and `folds/min` board terminology; inspect each hit
   rather than globally replacing legitimate domain language.
3. Run Markdown/diff checks available in kbench, commit that documentation-only peer change, and add
   its full 40-character SHA to issue #49's Log.

### Task 9: Full verification and close-boundary preparation

**Files:**
- Modify if architecture changed: `atlas/` and `atlas/index.md`
- Modify: `workshop/issues/000049-board-readability-labels-coldphase-eta.md`

1. Run `gofmt` on changed Go files, `go test ./cmd/metis -race`, `go test ./... -race`, and
   `git diff --check` in metis. Run the relevant kbench checks and `git diff --check` there.
2. Grep Go, tests, atlas, and operator docs for obsolete board strings; classify remaining domain
   uses. Confirm flat/nested output, width, repaint cadence, failure cleanup, and no-op non-sweep paths.
3. Update atlas only if the implementation adds a durable architectural seam; otherwise record why
   `--no-atlas` is appropriate in close evidence. Check all issue/plan boxes and log exact commands.
4. Run `sdlc close --issue 49 --verified '<evidence>'` once. Let the binary dispatch the mandatory
   fresh-context boundary review; fix every Critical/Important finding and rerun the gate as directed.

## Revisions

### 2026-07-17 — close-review traceability fix
- Boundary review found the implementation was still dirty/untracked when `sdlc close` first ran, so
  the reviewed `HEAD` did not contain the claimed board changes. The fix is to commit the activity,
  progress, board, run, sweep, atlas, issue, and review bookkeeping before rerunning `sdlc close`.
- Boundary review also required the Core concepts tables to carry greppable `Kind`, `File location`,
  and `Status` columns. Added those columns and replaced the non-code `activitySnapshot` row with the
  implemented `progressState` activity fields.

 succeeded in 0ms:
---
id: 000049
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-17
estimate_hours: 2.63
started: 2026-07-16T12:57:08-07:00
---

# board readability — label semantics, cold-phase "no progress" confusion, jumpy leaves, wild early ETA

## Problem

Operator's first real-sweep board session (titanic-sweep.md, BLAS-pinned, 2026-07-16) surfaced
four readability issues — the board is mechanically correct but hard to READ in exactly the
early phase where the operator most wants reassurance:

1. **Label semantics unclear:** `outer 0/10 · configs 0/720 · folds 0/7200` — operator asked
   "is folds about inner folds?" `folds` = leaf-level (config × inner-fold) RUNS aggregated
   across the whole sweep (10 outer × 72 configs × 10 inner); `configs` = per-outer-pass
   config completions aggregated (72 × 10). Neither is legible from the labels.
2. **Cold-phase "lack of progress":** early in a cold run every fold row shows `queued`, all
   counters sit at 0 for many minutes. Root cause is the metis#43 phase-wave scheduling (all
   cv-splits/features across the grid complete before ANY train chain finishes → zero fold
   completions for ~10 min while heavy upstream work happens). The board renders that
   truthfully but unhelpfully — nothing distinguishes "working through upstream steps" from
   "hung". (#43 fixes the schedule; this issue covers what the board shows MEANWHILE.)
3. **`leaves 2/12` jumps around:** the gauge samples instantaneous `len(leafSem)` at flush
   time — honest, but at 4Hz it reads as noise, and low occupancy during the upstream herd
   phase looks like a problem when it isn't.
4. **ETA changes wildly:** the 64-completion moving window over sparse, phase-heterogeneous
   early completions (fast cache-hit folds vs slow cold trains) swings the rate — the ETA
   flapped across hours on the operator's run. An estimate that volatile is worse than none.

## Spec

### Claim-time directions (retained)

1. Rename/annotate segments (e.g. `inner-folds 421/7200` or a one-line legend on the first
   frame); consider showing per-outer-fold denominators in the fold rows as the source of
   truth for "what is 7200".
2. A PHASE indicator when zero folds have completed but leaves are active (e.g.
   `warming: upstream steps running · 0 folds complete yet`) — distinguish wave-phase from
   hang using signals the sink already has (leaf occupancy > 0, passthrough step lines
   flowing).
3. Smooth the leaves gauge (peak-or-mean over the flush window, or EWMA) — the sink already
   samples at each event; keep a tiny window instead of the instant.
4. ETA damping: suppress until n≥K completions AND the window spans ≥T seconds; consider
   EWMA on the rate; show a range or `~` marker while confidence is low. (The stall-decay
   property — rate → 0 on thrash — must SURVIVE damping; that alarm is the line's purpose.)

### Approved design — typed, time-driven board telemetry

#### Counter vocabulary

Rename the shared `progressCore` labels once (plain output and TUI derive from it; ARCH-DRY), while
letting the already-known `nested` state select truthful mode-specific vocabulary:

- `outer folds k/n` — completed held-out outer-fold evaluations.
- `configs scored k/n` — configs whose full inner resample completed, aggregated across outer folds.
- Nested: `inner-CV runs k/n` — completed `(outer fold × config × inner fold)` pipeline runs. Flat:
  `CV runs k/n` — completed folds in the one single-level CV. These replace the ambiguous `folds`;
  per-outer rows use `outer fold N` and the nested `configs scored` / `inner-CV runs` vocabulary
  with per-pass denominators. Startup, rate, and ETA wording uses the same mode-specific noun.

#### Typed activity seam

- Add a thin `activityExecutor` decorator around the **final** step executor, outside the cache
  decorator. It emits a typed successful-step completion after either a real execution or cache hit;
  no terminal text is parsed, and failed steps continue through the existing fatal error path without
  being counted as reassuring activity (ARCH-PURE).
- The same injected activity sink has a distinct successful-run event emitted by
  `runResolvedExperiment` only when execution has `runErr == nil` **and** required run-record /
  provenance persistence succeeds. A failed execution remains uncounted even when its failure record
  is written successfully. Its typed run role distinguishes nested inner-CV, flat CV, preamble, and
  outer-score runs; only the first two feed the corresponding run counter/rate/ETA. This records
  actual completion time at the concrete-run seam, rather than the later input-ordered `sampler.Run`
  delivery of a completed batch (ARCH-PURPOSE).
- Events carry an injected-clock time and typed identity. `sweepProgress` reduces successful-step
  events to a count/last-step time and eligible run events to the run count/rate window. Concurrent
  callback delivery may differ from event-time order, so each last-time is a max and the latest 64
  eligible run timestamps remain sorted by event time before readiness/rate math. Reversed-delivery
  traces pin both invariants. Callbacks are short and mutex-protected; non-sweep callers use a no-op
  sink.
- Before the first eligible run completes, the board reports observations, never an unprovable
  diagnosis. Nested example: `starting · ~8/12 subprocess slots · 37 steps completed · last step 1s
  ago · no inner-CV run complete`; flat output substitutes `CV run`. With occupancy but no successful
  step yet, it says only that slots are occupied. It never claims “not hung” or infers an upstream
  phase from a start line.
- The startup line disappears on the first eligible inner-CV/CV run. This remains correct both before
  and after #43: #43 shortens startup; #49 truthfully describes it.

#### Time-driven smoothing and ETA confidence

- Sample leaf occupancy on the existing 500ms board tick, not per event. Display the rolling mean of
  the last four samples (two seconds) rounded to a whole slot and prefixed `~`. Equal timestamped
  occupancy traces must render identically regardless of event density.
- Keep the latest 64 eligible run-completion timestamps, but withhold rate and ETA until at least 16
  completions span at least 15 seconds. Compute rate as `(n−1) / (now−oldest)`, removing the current
  early upward bias that counts `n` completions across `n−1` observed intervals.
- Continue using the current time in the denominator on every 500ms tick. With no new completions, the
  numeric rate is non-increasing and ETA is non-decreasing, although display rounding need not change
  within an arbitrary number of seconds after a long mature window. Preserve the fast, truthful stall
  signal separately: after startup, show `last inner-CV run 8s ago` (or `last CV run …` flat), updated
  on every tick. A scripted mature trace followed by five seconds of silence must advance that age by
  five seconds while rate/ETA move monotonically; it makes no diagnosis about why work is quiet
  (ARCH-PURPOSE). After a stall, the 64-completion window recovers gradually rather than snapping to
  one interarrival.
- Render the mature estimate as `~ETA …`; nested mode labels its rate `inner-CV runs/min` and scopes
  ETA to remaining inner-CV runs, while flat mode uses `CV runs/min` and remaining CV runs. Neither
  includes held-out scoring, capture, or final ledger work. Before confidence, show the corresponding
  mode-specific `— … runs/min` and no ETA rather than a volatile fiction.

#### Documentation boundary

Update the kbench Titanic sweep RUNBOOK's board description/example after metis#49 lands. Record the
exact peer commit in this issue's Log before close so the cross-repo requirement is reviewable.

## Done when

- Operator can read the first 2 minutes of a cold real sweep without wondering if it hung:
  factual startup activity present until the first inner-CV run completes; labels self-explanatory.
- Leaves + rate/ETA move smoothly (unit tests over scripted event traces pin the damping); after a
  mature trace, five seconds without a completion visibly advances the mode-specific last-run age by
  five seconds while the numeric rate/ETA move monotonically (regression test).
- Occupied-but-silent subprocesses are never described as proven progress; typed successful step
  completions/cache hits are the only positive activity signal.
- A failed run that successfully persists its failure record advances neither the eligible run
  counter nor rate/ETA (regression test).
- Flat and nested output, width clamping, repaint cadence, and terminal cleanup remain correct.
- RUNBOOK board description/example updated and its peer commit pinned in the Log.

## Plan

Durable plan: `workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md`
(single pass, no Mx — one close boundary).

- [x] Add typed step/run activity at the concrete executor and persistence seams, including cache,
  failure, role, ordering, and cancellation tests.
- [x] Reduce time-driven occupancy and eligible-run telemetry with deterministic readiness, decay,
  recovery, and out-of-order-event tests.
- [x] Render truthful flat/nested vocabulary, startup observations, last-run age, and mature rate/ETA;
  preserve repaint, failure, width, and terminal behavior.
- [x] Update and commit the kbench Titanic RUNBOOK, pin its full commit here, then run focused,
  race, full-suite, formatting, and stale-vocabulary verification.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.05 impl=0.35
item: smaller-go-module   design=0.06 impl=0.35
item: tui-screen          design=0.15 impl=0.55
item: cross-cutting-refactor design=0.05 impl=0.25
item: smaller-go-module   design=0.06 impl=0.35
item: atlas-docs          design=0.02 impl=0.10
item: milestone-review    design=0.03 impl=0.20
design-buffer: 0.15
total: 2.63
```

Rows: (1) typed activity entities/decorator; (2) event-time reducer and rate/occupancy math;
(3) board state/rendering and scripted traces; (4) shared vocabulary migration; (5) concrete-run,
controller, and sweep wiring; (6) peer RUNBOOK plus atlas/stale-term sweep; (7) one SDLC milestone-review row.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

## Log

### 2026-07-16
- Filed from the operator's mid-run feedback on the first real-sweep board session (pins set,
  full 7,200-fold grid). Companion issues from the same session: metis#47 (flash — FIXED,
  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
  #43 symptom: depth-first scheduling would give early train completions → a stable rate
  much sooner.

### 2026-07-16 — paired #43/#49 design approved
- Operator chose typed step-completion telemetry over presentation-only occupancy wording because
  occupancy cannot distinguish useful work from a hung subprocess. Co-designed after mapping the
  current sink/rate/compositor flow; #43 merges first, then #49 builds against its schedule.

### 2026-07-17 — planning checkpoint
- Reconciled the brain project after #43 merged, ran `sdlc start-plan`, mapped the final-executor,
  concrete-run persistence, run-control, sweep-progress, board, and kbench documentation seams, and
  authored the durable single-boundary TDD plan. Estimate uses v3.1 Method A; the approved spec and
  existing #38/#43 patterns make this familiar extension work rather than a novel TUI subsystem.
- Fresh-eyes plan review: Chunk 1 found Important gaps in shared run-control activity gating and
  concrete flat/nested role propagation; Chunk 2 found an Important gap in aggregate counter ownership.
  Patched the plan and both reviewers re-checked clean. Chunk 3 fresh-eyes review returned clean.

### 2026-07-17 — implementation verification
- Implemented typed step/run activity, run-role propagation, run-control-gated publication, event-time
  eligible-run rate reduction, tick-smoothed `~slots`, factual cold-start board wording, shared
  flat/nested progress vocabulary, and last-run-age/mature `~ETA` rendering. Updated atlas with the
  durable activity seam and board contract.
- Updated the kbench Titanic operator RUNBOOK and committed it as
  `68edc6ca312dbe5306ce87315abec2d67e678005` (`docs: update metis board progress contract`).
- Verification run: `go test ./cmd/metis -count=1`; `go test ./cmd/metis -race -count=1`;
  `go test ./... -race -count=1`; metis `git diff --check`; kbench `git diff --check`; stale board
  terminology grep over metis `cmd/`, `atlas/`, issue/plan, and the kbench RUNBOOK. Remaining grep hits
  are historical problem/spec wording or negative test fixtures, not live operator output.

## Revisions

### 2026-07-16 — fresh-eyes spec review
- Split flat `CV runs` from nested `inner-CV runs`, made out-of-order callback reduction explicit,
  and replaced an unprovable seconds-level rounded-rate promise with a tick-driven last-run-age
  freshness signal plus a measurable scripted-trace criterion. Separated typed step activity from
  actual concrete-run completion so batch-order sampler callbacks cannot falsify the rate window;
  required execution success independently of successful failure-record persistence.

### 2026-07-17 — durable implementation plan
- Replaced the two placeholder plan rows with concrete typed-activity, pure-reducer, rendering,
  cross-repo documentation, and verification steps; added the calibrated estimate and explicit
  single-close-boundary decision.

exec
/bin/zsh -lc 'git status --short' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
exec
/bin/zsh -lc 'git log --oneline --decorate -8' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
893c8e0 (HEAD -> 000049-board-readability-labels-coldphase-eta) #49: implement activity-backed board telemetry
73cf699 (main) #49: fix estimate primitive slugs
b2fff03 #49: plan board readability telemetry
b37415c (origin/main, origin/HEAD) archive completed issues to history
8a0e410 Merge pull request #34 from xianxu/000043-leaf-sched-depth-first
745e9f5 (origin/000043-leaf-sched-depth-first) #43: close bounded whole-run admission
53f0488 #43: enter verified close boundary
56e560c #43: pin cold smoke peer dependency

codex
`HEAD` now points at `#49: implement activity-backed board telemetry`, and the worktree appears clean. I’m checking the committed diff against the requested base and then I’ll focus on spec-level behavior rather than the previous “dirty tree” blocker.
exec
/bin/zsh -lc "sed -n '1,220p' cmd/metis/activity.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

import (
	"time"

	"github.com/xianxu/metis/pkg/experiment"
)

type activityKind string

const (
	activityStepSuccess activityKind = "step-success"
	activityRunSuccess  activityKind = "run-success"
)

type runRole string

const (
	runRoleNone           runRole = ""
	runRoleNestedInnerCV  runRole = "nested-inner-cv"
	runRoleFlatCV         runRole = "flat-cv"
	runRoleNestedPreamble runRole = "nested-preamble"
	runRoleOuterScore     runRole = "outer-score"
)

type activityEvent struct {
	Kind   activityKind
	At     time.Time
	StepID string
	RunID  string
	Role   runRole
}

type activityEmitter func(activityEvent)

func (e activityEmitter) emit(ev activityEvent) {
	if e != nil {
		e(ev)
	}
}

func teeActivityEmitter(a, b activityEmitter) activityEmitter {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(ev activityEvent) {
		a.emit(ev)
		b.emit(ev)
	}
}

func runControlActivityEmitter(control *runControl, emit activityEmitter) activityEmitter {
	if control == nil {
		return emit
	}
	return func(ev activityEvent) {
		control.whileHealthy(func() { emit.emit(ev) })
	}
}

type activityExecutor struct {
	inner experiment.StepExecutor
	now   func() time.Time
	emit  activityEmitter
}

func (e activityExecutor) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
	res, err := e.inner.Execute(step, runDir)
	if err != nil {
		return res, err
	}
	now := e.now
	if now == nil {
		now = time.Now
	}
	e.emit.emit(activityEvent{Kind: activityStepSuccess, At: now(), StepID: step.ID})
	return res, nil
}

exec
/bin/zsh -lc 'git diff --name-only 73cf6993cb9cbb87a11696f7adb378b071b39c81..HEAD' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
atlas/index.md
cmd/metis/activity.go
cmd/metis/activity_test.go
cmd/metis/board.go
cmd/metis/board_test.go
cmd/metis/caching_test.go
cmd/metis/nestedcv_e2e_test.go
cmd/metis/progress.go
cmd/metis/progress_test.go
cmd/metis/run.go
cmd/metis/run_test.go
cmd/metis/runcontrol_test.go
cmd/metis/shapesweep_test.go
cmd/metis/sweep.go
workshop/issues/000049-board-readability-labels-coldphase-eta.md
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md
workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md

exec
/bin/zsh -lc "sed -n '1,460p' cmd/metis/progress.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
exec
/bin/zsh -lc 'git diff --stat 73cf6993cb9cbb87a11696f7adb378b071b39c81..HEAD' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
 atlas/index.md                                     |   33 +-
 cmd/metis/activity.go                              |   81 +
 cmd/metis/activity_test.go                         |   78 +
 cmd/metis/board.go                                 |   46 +-
 cmd/metis/board_test.go                            |  129 +-
 cmd/metis/caching_test.go                          |   63 +
 cmd/metis/nestedcv_e2e_test.go                     |    2 +-
 cmd/metis/progress.go                              |  133 +-
 cmd/metis/progress_test.go                         |  157 +-
 cmd/metis/run.go                                   |   17 +-
 cmd/metis/run_test.go                              |  130 +
 cmd/metis/runcontrol_test.go                       |   20 +
 cmd/metis/shapesweep_test.go                       |   55 +-
 cmd/metis/sweep.go                                 |   10 +-
 ...00049-board-readability-labels-coldphase-eta.md |   20 +-
 ...eadability-labels-coldphase-eta-close-review.md | 5866 ++++++++++++++++++++
 ...-board-readability-labels-coldphase-eta-plan.md |   40 +-
 17 files changed, 6743 insertions(+), 137 deletions(-)

 succeeded in 0ms:
package main

// metis#30: the sweep progress sink — folds pkg/sampler's per-completion
// ProgressEvents (typed per level) into ONE throttled aggregated line, so a
// 2,000-fold sweep reports live without a per-fold firehose (single-threaded-
// attention budget). Plain appended lines, no escape codes — non-TTY-safe by
// construction; the TTY board is metis#38, which extends this sink behind the
// same per-pass hooks (outer-fold identity rides the forPass closure binding,
// NEVER an event payload field — pkg/sampler stays coordinate-free).

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/metis/pkg/sampler"
	"github.com/xianxu/metis/pkg/shape"
)

// progressTotals seeds the sink with each level's SizeHint AT WIRING TIME —
// stream-learned totals arrive only with a level's first completion (for the
// driver level: the first COMPLETED outer fold, near the end of a parallel run),
// which starves the display of denominators. cmd/metis constructs the samplers,
// so it reads SizeHint directly; SizeHint stays the single source (ARCH-DRY).
type progressTotals struct {
	nested     bool
	outer      int
	outerKind  sampler.SizeKind
	configs    int // aggregate across outer folds (outer × per-pass configs)
	configKind sampler.SizeKind
	folds      int // aggregate leaf count (outer × configs × inner k)
	foldKind   sampler.SizeKind
}

// progressState is the pure render input: sink-owned aggregate counters per level
// (NEVER ev.K — each concurrent Run instance counts its own 1..k), the completed
// outer scores (→ est mean±SE), and the flat path's running fold scores.
type progressState struct {
	nested               bool
	outerK, outerTotal   int
	outerKind            sampler.SizeKind
	configK, configTotal int
	configKind           sampler.SizeKind
	foldK, foldTotal     int
	foldKind             sampler.SizeKind
	stepK                int
	lastStepAt           time.Time
	lastRunAt            time.Time
	outerScores          []float64 // nested: completed outer-fold held-out scores
	flatScores           []float64 // flat: the one config's completed fold scores
}

// progressLine renders the aggregated line. Nested:
// `outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`
// (est — until an outer fold lands; ± only at n≥2). Flat (since metis#32: iff 1
// config): `CV runs 3/5 · score 0.8400` (the running fold mean — nothing to be
// "best" of). Kinds render k/n (exact), k/≤n (budget), k/? (unknown). Pure.
func progressLine(st progressState) string {
	return "metis: progress " + progressCore(st)
}

// progressCore is the un-prefixed aggregate segment — shared by the plain line and
// the board's first row (extracted so the board never string-strips the prefix;
// a TrimPrefix coupling would silently no-op if the prefix changed — close review).
func progressCore(st progressState) string {
	frac := func(k, total int, kind sampler.SizeKind) string {
		switch kind {
		case sampler.SizeExact:
			return fmt.Sprintf("%d/%d", k, total)
		case sampler.SizeBudget:
			return fmt.Sprintf("%d/≤%d", k, total)
		default:
			return fmt.Sprintf("%d/?", k)
		}
	}
	var parts []string
	if st.nested {
		parts = append(parts, "outer folds "+frac(st.outerK, st.outerTotal, st.outerKind))
		parts = append(parts, "configs scored "+frac(st.configK, st.configTotal, st.configKind))
		parts = append(parts, "inner-CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
		mean, se, n := meanSE(st.outerScores)
		switch {
		case n == 0:
			parts = append(parts, "est —")
		case n == 1:
			parts = append(parts, fmt.Sprintf("est %.4f", mean))
		default:
			parts = append(parts, fmt.Sprintf("est %.4f ± %.4f", mean, se))
		}
	} else {
		parts = append(parts, "CV runs "+frac(st.foldK, st.foldTotal, st.foldKind))
		if mean, _, n := meanSE(st.flatScores); n > 0 {
			parts = append(parts, fmt.Sprintf("score %.4f", mean))
		}
	}
	return strings.Join(parts, " · ")
}

// meanSE is the display-only mean ± standard-error reduce over completed scores.
// Computed locally: the honest estimate stays pkg/sampler's Aggregate/MeanSE —
// this is presentation, not selection (do not export sampler surface for it).
func meanSE(xs []float64) (mean, se float64, n int) {
	n = len(xs)
	if n == 0 {
		return 0, 0, 0
	}
	for _, x := range xs {
		mean += x
	}
	mean /= float64(n)
	if n < 2 {
		return mean, 0, n
	}
	var ss float64
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	se = math.Sqrt(ss/float64(n-1)) / math.Sqrt(float64(n))
	return mean, se, n
}

// seededTotals reads each level's SizeHint on its initial state — the SAME source
// the Run loops stamp on events (ARCH-DRY; no shape math re-derived here) — and
// composes the aggregate denominators: configs = outer × per-pass configs,
// folds = outer × configs × inner k (each sealed pass sweeps the full grid).
// Flat (1 config): folds = the single pass's inner k.
func seededTotals(ctx sampler.Ctx, nested bool, runFolds int, configPts []shape.Point, k int) progressTotals {
	grid := sampler.GridConfigs{Points: configPts}
	nConfigs, kindConfigs := grid.SizeHint(grid.Init(ctx))
	foldsSmp := sampler.FixedKFolds{K: k}
	nFolds, kindFolds := foldsSmp.SizeHint(foldsSmp.Init(ctx))
	if !nested {
		return progressTotals{folds: nFolds, foldKind: kindFolds}
	}
	cv := sampler.CVDriver{K: runFolds}
	nOuter, kindOuter := cv.SizeHint(cv.Init(ctx))
	return progressTotals{
		nested: true,
		outer:  nOuter, outerKind: kindOuter,
		configs: nOuter * nConfigs, configKind: kindConfigs,
		folds: nOuter * nConfigs * nFolds, foldKind: kindFolds,
	}
}

// movingRate retains the latest eligible run completions by event time. rate(now)
// = (n-1)/(now-oldest) after the confidence gate; `now` in the denominator means
// a STALL decays live while last-run age remains the sharp freshness signal.
type movingRate struct {
	times []time.Time
}

func (m *movingRate) add(t time.Time) {
	i := sort.Search(len(m.times), func(i int) bool { return !m.times[i].Before(t) })
	m.times = append(m.times, time.Time{})
	copy(m.times[i+1:], m.times[i:])
	m.times[i] = t
	if len(m.times) > 64 {
		m.times = m.times[1:]
	}
}

// rate returns eligible runs/minute over the kept event-time window.
func (m *movingRate) rate(now time.Time) (perMin float64, ok bool) {
	if len(m.times) < 16 {
		return 0, false
	}
	oldest := m.times[0]
	newest := m.times[len(m.times)-1]
	if newest.Sub(oldest) < 15*time.Second {
		return 0, false
	}
	mins := now.Sub(oldest).Minutes()
	if mins <= 0 {
		return 0, false
	}
	return float64(len(m.times)-1) / mins, true
}

// eta = remaining / rate; ok=false when the rate is unavailable or zero.
func (m *movingRate) eta(now time.Time, remaining int) (time.Duration, bool) {
	r, ok := m.rate(now)
	if !ok || r <= 0 || remaining <= 0 {
		return 0, false
	}
	return time.Duration(float64(remaining) / r * float64(time.Minute)), true
}

type occupancySample struct {
	busy, capacity int
}

type occupancyWindow struct {
	samples []occupancySample
}

func (w *occupancyWindow) add(busy, capacity int) {
	if capacity <= 0 {
		return
	}
	w.samples = append(w.samples, occupancySample{busy: busy, capacity: capacity})
	if len(w.samples) > 4 {
		w.samples = w.samples[1:]
	}
}

func (w occupancyWindow) mean() (busy, capacity int, ok bool) {
	if len(w.samples) == 0 {
		return 0, 0, false
	}
	var sum int
	for _, s := range w.samples {
		sum += s.busy
		capacity = s.capacity
	}
	return int(math.Round(float64(sum) / float64(len(w.samples)))), capacity, true
}

// passRow is one outer fold's live board row (metis#38): in-flight counters + the
// pass's incumbent best (display-only — NOT the 1-SE select rule), collapsing to its
// held-out score when the driver reports the fold done.
type passRow struct {
	configK, foldK int
	best           float64
	hasBest        bool
	done           bool
	heldOut        float64
}

// boardState is the pure render input for metis#38's board: the #30 aggregate state
// plus the per-pass rows and the throughput ring (a mutex'd snapshot — renderers never
// touch the live sink).
type boardState struct {
	st   progressState
	rows []passRow
	rate movingRate
}

// sweepProgress is the mutex'd sink shared by every pass of one shape-run. Events
// arrive concurrently (ParExec goroutines across sibling outer folds, each holding
// its own Run's event mutex); health-gated paths use the strict order runControl.mu
// → sink.mu → boardWriter.mu (never the reverse). Emit policy: fold/config events are throttled
// to one line per second (injected clock — tests script it, never sleep); a
// driver-level (outer fold) completion ALWAYS emits; finish() emits the terminal
// line. A nil *sweepProgress is a no-op everywhere (the non-sweep path is silent).
type sweepProgress struct {
	mu        sync.Mutex
	out       io.Writer
	now       func() time.Time
	direction string // the objective direction — orients each pass's display-best (#38)
	st        progressState
	rows      []passRow  // metis#38: one row per outer fold (nil on the flat path)
	rate      movingRate // metis#38: fold-completion throughput window
	lastEmit  time.Time
	started   bool
	// metis#38 board mode (all nil/zero in plain mode): emits paint the rendered frame
	// to bw instead of printing plain lines. Lock order: sink.mu → bw.mu, ALWAYS — the
	// ticker enters via tick() (a sink method), never a boardWriter-first path.
	bw        *boardWriter
	width     int               // terminal width ($COLUMNS | 80), read once at wiring
	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
	occupancy occupancyWindow
}

func newSweepProgress(out io.Writer, now func() time.Time, direction string, totals progressTotals) *sweepProgress {
	var rows []passRow
	if totals.nested && totals.outer > 0 {
		rows = make([]passRow, totals.outer)
	}
	return &sweepProgress{
		out: out, now: now, direction: direction, rows: rows,
		st: progressState{
			nested:     totals.nested,
			outerTotal: totals.outer, outerKind: totals.outerKind,
			configTotal: totals.configs, configKind: totals.configKind,
			foldTotal: totals.folds, foldKind: totals.foldKind,
		},
	}
}

// boardState snapshots the sink for a renderer (rows copied — the caller may hold
// the snapshot without racing the live fold-in).
func (sp *sweepProgress) boardState() boardState {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.snapshotLocked()
}

// snapshotLocked builds the render snapshot; caller holds sp.mu (shared by
// boardState() and the board-mode emit — one copy site, close-review DRY note).
func (sp *sweepProgress) snapshotLocked() boardState {
	rows := make([]passRow, len(sp.rows))
	copy(rows, sp.rows)
	return boardState{st: sp.st, rows: rows, rate: sp.rate}
}

// passHooks are one pass's typed event targets, closure-bound to its outer-fold
// index (-1 = the flat path's single pass) — the metis#38 identity seam.
type passHooks struct {
	config func(sampler.ProgressEvent[shape.Point, sampler.MeanSE])
	fold   func(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome])
}

// forPass hands out a pass's hooks. #30's sink aggregates across passes (the
// single-line mental model); the per-pass binding exists so #38 can add per-fold
// board rows behind the same API without touching pkg/sampler.
func (sp *sweepProgress) forPass(outer int) passHooks {
	if sp == nil {
		return passHooks{
			config: func(sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {},
			fold:   func(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {},
		}
	}
	return passHooks{
		config: func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
			sp.mu.Lock()
			defer sp.mu.Unlock()
			sp.st.configK++
			if outer >= 0 && outer < len(sp.rows) { // #38: this pass's row
				r := &sp.rows[outer]
				r.configK++
				if !r.hasBest || better(sp.direction, ev.Out.Mean, r.best) {
					r.best, r.hasBest = ev.Out.Mean, true
				}
			}
			sp.maybeEmit(false)
		},
		fold: func(ev sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]) {
			sp.mu.Lock()
			defer sp.mu.Unlock()
			if !sp.st.nested {
				sp.st.flatScores = append(sp.st.flatScores, ev.Out.Score)
			}
			if outer >= 0 && outer < len(sp.rows) {
				sp.rows[outer].foldK++
			}
			sp.maybeEmit(false)
		},
	}
}

func (sp *sweepProgress) activity(ev activityEvent) {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	switch ev.Kind {
	case activityStepSuccess:
		sp.st.stepK++
		at := ev.At
		if at.IsZero() {
			at = sp.now()
		}
		if at.After(sp.st.lastStepAt) {
			sp.st.lastStepAt = at
		}
	case activityRunSuccess:
		if ev.Role != runRoleNestedInnerCV && ev.Role != runRoleFlatCV {
			return
		}
		sp.st.foldK++
		at := ev.At
		if at.IsZero() {
			at = sp.now()
		}
		if at.After(sp.st.lastRunAt) {
			sp.st.lastRunAt = at
		}
		sp.rate.add(at)
	default:
		return
	}
	sp.maybeEmit(false)
}

// better orients a display-best comparison by the objective direction.
func better(direction string, candidate, incumbent float64) bool {
	if direction == "minimize" {
		return candidate < incumbent
	}
	return candidate > incumbent
}

// driverEvent folds a completed OUTER fold in — always emits (the coarse level is
// the one the operator watches; its completions are rare and load-bearing).
func (sp *sweepProgress) driverEvent(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.st.outerK++
	sp.st.outerScores = append(sp.st.outerScores, ev.Out)
	if i := ev.Point.Idx; i >= 0 && i < len(sp.rows) { // #38: collapse this fold's row
		sp.rows[i].done = true
		sp.rows[i].heldOut = ev.Out
	}
	sp.maybeEmit(true)
}

// finish emits the terminal state line (always).
func (sp *sweepProgress) finish() {
	if sp == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.emit()
}

// tick is the board ticker's entry point (metis#38): repaint with a fresh `now` so
// the rate decay + ETA move even between events. Sink-first (sp.mu → bw.mu).
func (sp *sweepProgress) tick() {
	if sp == nil || sp.bw == nil {
		return
	}
	sp.mu.Lock()
	if sp.gauge != nil {
		busy, capacity := sp.gauge()
		sp.occupancy.add(busy, capacity)
	}
	sp.emit() // stores the fresh frame (budget may skip the draw)
	bw := sp.bw
	sp.mu.Unlock()
	bw.forceFlush() // metis#46: the tick is what re-pins the board after a burst window
}

// abort removes the stored live frame after a sweep failure. Lock order remains
// progress -> board; the controller is never called while either lock is held.
func (sp *sweepProgress) abort() {
	if sp == nil || sp.bw == nil {
		return
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.bw.discardFrame()
}

// maybeEmit writes the line if forced (driver/finish) or the throttle elapsed —
// 1s for plain lines (a log is a record), 100ms for board repaints (a board is a
// display; the 500ms ticker guarantees freshness anyway). Caller holds sp.mu.
func (sp *sweepProgress) maybeEmit(force bool) {
	now := sp.now()
	throttle := time.Second
	if sp.bw != nil {
		throttle = 100 * time.Millisecond
	}
	if !force && sp.started && now.Sub(sp.lastEmit) < throttle {
		return
	}
	sp.started = true
	sp.lastEmit = now
	sp.emit()
}

// emit renders the current state: board mode paints the frame (under the fixed

codex
The core reducer now matches the intended architecture at a high level: typed run activity owns aggregate run counts/rate, sampler fold callbacks only maintain score/row state, and occupancy sampling is tick-driven. I’m checking the wiring points and tests next, because that’s where this sort of change usually slips.
exec
/bin/zsh -lc "sed -n '460,560p' cmd/metis/progress.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
// emit renders the current state: board mode paints the frame (under the fixed
// sink.mu → bw.mu order; the snapshot is built inline — boardState() would re-lock);
// plain mode prints the #30 aggregated line. Caller holds sp.mu.
func (sp *sweepProgress) emit() {
	if sp.bw != nil {
		busy, capacity, _ := sp.occupancy.mean()
		sp.bw.paint(renderBoard(sp.snapshotLocked(),
			boardEnv{width: sp.width, now: sp.now(), busy: busy, capacity: capacity}))
		return
	}
	fmt.Fprintln(sp.out, progressLine(sp.st))
}

exec
/bin/zsh -lc "sed -n '200,330p' cmd/metis/run.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
	if o.runID != "" {
		return o.runID
	}
	sbh, err := shapeBlobHash(o.expPath)
	if err == nil {
		if addr, err := pointAddressOf(exp, sbh); err == nil {
			return addr
		}
	}
	return "run-" + now().UTC().Format("20060102T150405Z")
}

// runResolvedExperiment runs one already-resolved experiment (a single point) under
// runID, through the cached runner, and writes its run.json + provenance record (the
// experiment `.md` is immutable input — not written back, #13). The shared per-point runner
// both the 1-point path and the sweep loop (metis#7) call — so the run/cache/record wiring
// lives in ONE place (ARCH-DRY).
func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
	if o.runControl == nil {
		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
	}
	o.activity = runControlActivityEmitter(o.runControl, o.activity)
	return o.runControl.run(o.runLabel, func() (experiment.Run, error) {
		return runResolvedExperimentAdmitted(exp, o, runID, now, out)
	})
}

func runResolvedExperimentAdmitted(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
	baseDir := filepath.Dir(o.expPath)
	// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
	// the child's env, and the child's cwd IS the step dir — a relative path would
	// resolve $METIS_STEP_DIR/with.json under itself. Absolute paths are correct
	// from any cwd, so `metis run pipelines/foo.md` (a relative arg) works.
	runDir, err := filepath.Abs(filepath.Join(baseDir, "runs", runID))
	if err != nil {
		return experiment.Run{}, err
	}
	expDir, err := filepath.Abs(baseDir)
	if err != nil {
		return experiment.Run{}, err
	}

	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool, pins: o.leafPins}
	if o.exec != nil {
		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
	}
	if o.cache {
		cacheDir := filepath.Join(expDir, ".metis-cache")
		if err := ensureCacheGitignore(cacheDir); err != nil {
			return experiment.Run{}, err
		}
		store := cas.NewFSStore(filepath.Join(cacheDir, "cas"), 0, cas.Clock(now))
		exec = newCachingExecutor(exec, store, cacheDir, exp.Seed, out)
	}
	exec = activityExecutor{inner: exec, now: now, emit: o.activity}
	runner := experiment.Runner{Exec: exec, Now: now}
	fmt.Fprintf(out, "metis: run %s of experiment %q\n", runID, exp.ID)
	run, steps, runErr := runner.Run(exp, runID, runDir)

	// Execution-time enforcement: Runner.Run validates the experiment BEFORE any
	// step executes, so a semantically-invalid experiment (dangling needs, bad
	// uses, a cycle) is rejected here — closing the SHAPE-only gap M1 left. Such a
	// rejection never started a run (run.Started is empty), so surface the error
	// without writing a bogus record.
	if run.Started == "" {
		return run, runErr
	}

	// Write the ledger even on a mid-run step failure (status=failed) so every
	// attempt that began is recorded — the record of truth is runs/<id>/run.json.
	if err := writeRunJSON(runDir, run); err != nil {
		return run, err
	}
	// Assemble + persist the provenance record (metis#3): repo provenance, per-step
	// output hashes, and the minted point-address. A config that can't be
	// canonicalized (e.g. a non-finite value) surfaces here as a run error.
	// The shape's blob-hash content-addresses the intent (metis#27); computed the SAME way
	// singleRunID/pointAddressOf did, so the record's point_address matches the run dir.
	// A no-git spec yields "" (a degraded, non-content-addressed run — warned via capture status).
	sbh, _ := shapeBlobHash(o.expPath)
	rec, err := assembleRecord(o.git, out, expDir, runDir, exp, run, steps, sbh)
	if err != nil {
		return run, err
	}
	if err := writeRecordJSON(runDir, rec); err != nil {
		return run, err
	}
	// Capture this run's code closure + run-spec to a git side-ref (metis#14), backfilling
	// the record with the durable SHA + capture status — so a dirty single run is
	// reproducible (git checkout the SHA). The sweep loop sets inSweep to capture ONCE
	// per shape-run instead (captureSweepCode), avoiding redundant per-point capture.
	// Best-effort (like the sweep path): a backfill hiccup warns, never aborts a finished run.
	if !o.inSweep {
		if err := captureSingleRun(o, runID); err != nil {
			fmt.Fprintf(out, "metis: warning: code-capture backfill failed for run %s: %v\n", runID, err)
		}
	}
	// The experiment .md is IMMUTABLE input (#13): a run writes its output to
	// runs/<id>/{run,record}.json (+ the .ledger.csv sidecar for sweeps), NEVER to the
	// config file — so a committed config is a stable content-hash. The human "recent
	// runs / top-N" view is on-demand via `metis ledger show` over the sidecar.
	if runErr != nil {
		return run, runErr
	}
	o.activity.emit(activityEvent{Kind: activityRunSuccess, At: now(), RunID: runID, Role: o.runRole})
	fmt.Fprintf(out, "metis: %s %s\n", run.ID, run.Status)
	return run, nil
}

func writeRunJSON(runDir string, run experiment.Run) error {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "run.json"), append(b, '\n'), 0o644)
}

exec
/bin/zsh -lc "sed -n '1,180p' cmd/metis/board.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

// metis#38: the live progress board — the PURE frame renderer (this half) and the
// pin-bottom ANSI compositor (boardWriter, below). Presentation only, over the #30
// sink's boardState snapshot: no pkg/sampler change, no TUI library (the board is
// output-only — a hand-rolled repaint of N lines; see the plan's no-lib rationale).
// The paint/content split is deliberate: renderBoard returns plain lines (byte-
// testable, zero escape codes); ANSI lives ONLY in boardWriter.

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// boardEnv is the render-time environment: terminal width, the current instant
// (for rate/ETA), and the leaf-semaphore gauge (capacity 0 = no gauge → segment
// omitted, e.g. a serial run).
type boardEnv struct {
	width          int
	now            time.Time
	busy, capacity int
}

// maxFoldRows caps the per-fold section; beyond it the remainder collapses to
// an "… +N more" line (a 20-fold sweep must not paint a 22-line board).
const maxFoldRows = 12

// renderBoard renders the frame: the #30 aggregate line, one row per outer fold
// (✓ done → held-out score · ▸ in-flight → per-pass counters + incumbent · queued),
// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
// break the compositor's erase-count bookkeeping).
func renderBoard(bs boardState, env boardEnv) []string {
	var lines []string
	// Row 1: the aggregate — the same core the plain line prints (one source, no
	// prefix stripping).
	lines = append(lines, progressCore(bs.st))

	// Per-fold rows (nested only; flat runs have no rows).
	shown := len(bs.rows)
	if shown > maxFoldRows {
		shown = maxFoldRows
	}
	// Per-row denominators derive from the seeded aggregate totals (per-pass share).
	perConfigs, perFolds := 0, 0
	if n := len(bs.rows); n > 0 {
		perConfigs = bs.st.configTotal / n
		perFolds = bs.st.foldTotal / n
	}
	for i := 0; i < shown; i++ {
		r := bs.rows[i]
		switch {
		case r.done:
			lines = append(lines, fmt.Sprintf("  outer fold %d ✓ held-out %.4f", i, r.heldOut))
		case r.configK == 0 && r.foldK == 0:
			lines = append(lines, fmt.Sprintf("  outer fold %d — queued", i))
		default:
			b := ""
			if r.hasBest {
				b = fmt.Sprintf(" · best %.4f", r.best)
			}
			lines = append(lines, fmt.Sprintf("  outer fold %d ▸ configs scored %d/%d · inner-CV runs %d/%d%s",
				i, r.configK, perConfigs, r.foldK, perFolds, b))
		}
	}
	if hidden := len(bs.rows) - shown; hidden > 0 {
		lines = append(lines, fmt.Sprintf("  … +%d more", hidden))
	}

	// Leaves / throughput / ETA.
	var segs []string
	runNoun := "CV runs"
	runSingular := "CV run"
	if bs.st.nested {
		runNoun = "inner-CV runs"
		runSingular = "inner-CV run"
	}
	if env.capacity > 0 {
		segs = append(segs, fmt.Sprintf("~slots %d/%d", env.busy, env.capacity))
	}
	if bs.st.foldK == 0 {
		start := []string{"starting"}
		start = append(start, segs...)
		if bs.st.stepK > 0 {
			start = append(start, fmt.Sprintf("%d steps completed", bs.st.stepK))
			if !bs.st.lastStepAt.IsZero() {
				start = append(start, "last step "+fmtAge(env.now.Sub(bs.st.lastStepAt))+" ago")
			}
		}
		start = append(start, fmt.Sprintf("no %s complete", runSingular))
		lines = append(lines, strings.Join(start, " · "))
		for i, l := range lines {
			lines[i] = clampLine(l, env.width)
		}
		return lines
	}
	if !bs.st.lastRunAt.IsZero() {
		segs = append(segs, fmt.Sprintf("last %s %s ago", runSingular, fmtAge(env.now.Sub(bs.st.lastRunAt))))
	}
	if perMin, ok := bs.rate.rate(env.now); ok {
		segs = append(segs, fmt.Sprintf("%.1f %s/min", perMin, runNoun))
	} else {
		segs = append(segs, fmt.Sprintf("— %s/min", runNoun))
	}
	if remaining := bs.st.foldTotal - bs.st.foldK; remaining > 0 {
		if eta, ok := bs.rate.eta(env.now, remaining); ok {
			segs = append(segs, "~ETA "+fmtETA(eta))
		}
	}
	lines = append(lines, strings.Join(segs, " · "))

	for i, l := range lines {
		lines[i] = clampLine(l, env.width)
	}
	return lines
}

func fmtAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return fmtETA(d)
}

// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
func fmtETA(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// (Height analog of the width limitation: a terminal SHORTER than the board clamps
// cursor-up at the screen top and desyncs the erase count — the board caps at ~15
// lines; terminals that small are out of scope, same accepted trade as resize.)

// clampLine truncates to width runes with a trailing … (a wrapped physical line
// would desync the compositor's cursor-up erase count — width is load-bearing).
func clampLine(s string, width int) string {
	if width <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width-1]) + "…"
}

// ── boardWriter: the pin-bottom ANSI compositor ──────────────────────────────

// boardWriter owns the terminal: the board is pinned to the bottom while every
// other write (step logs, warnings) scrolls ABOVE it. Paint-only — it stores the
// last rendered frame and NEVER calls back into the sink (the one global lock
// order is sink.mu → bw.mu; a callback here would invert it). All output must
// route through this writer once it exists — a bypassing write corrupts the board
// (see the plan's writer-plumbing note: writer identity is temporal).
//
// metis#46: DOUBLE-BUFFERED with a bounded flush rate. The original design ran a
// full erase→write→repaint cycle per passthrough write; a warm-cache sweep emits
// hundreds of lines/second and real terminals — especially mux layers (the
// operator's ghostty-in-cmux) — paint asynchronously mid-sequence and tear under
// that flood. Now passthrough COALESCES into `pending` and the terminal sees one
// atomic erase→dump→repaint per flushBudget (~4Hz) — quiet writes (a cold run's
// sparse lines) still flush inline because the budget has long elapsed.
//
//	Write(p)     append to pending; flush inline iff the budget elapsed or the
//	             size cap is hit (bound memory under a frozen-budget flood).
//	paint(lines) store the frame; flush under the same budget.
//	tick/close   force-flush (the 500ms tick restores the board after a burst;
//	             close is idempotent, flushes everything, restores the cursor).
//
// It serializes internally (replacing syncWriter in board mode — one wrap, not two).
type boardWriter struct {

exec
/bin/zsh -lc "sed -n '250,680p' cmd/metis/sweep.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
		return nil
	}

	shapeRunID, err := shapeRunIdentity(sh, sbh)
	if err != nil {
		return err
	}
	ss := &shapeSweep{
		o: o, sh: sh, now: now, out: out, shapeBlobHash: sbh, codeID: sha, start: sweepStart,
		partRef:  partitionRef(sh),
		man:      sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweeper.Sampler, Seed: sh.Seed},
		parallel: o.maxParallel > 1, // metis#31: fan out the sweeper/resample/driver batches
	}
	ctx := sampler.Ctx{Seed: sh.Seed, Partition: ss.partRef}
	// metis#30: seed the sink's denominators AT WIRING TIME from the same SizeHint the
	// levels report (stream-learned totals would arrive only with each level's first
	// completion — for the driver level that's the first COMPLETED outer fold, too late).
	ss.prog = newSweepProgress(out, now, sh.Sweeper.Objective.Direction, seededTotals(ctx, nested, runFolds, configPts, k))
	ss.o.activity = teeActivityEmitter(ss.o.activity, ss.prog.activity)
	// metis#38: board mode — the sink paints the pinned board instead of plain lines,
	// and a 500ms ticker keeps the rate decay + ETA live between events (sink-first:
	// tick() locks sp.mu then hands the frame to bw — the one global lock order).
	if o.board != nil {
		ss.prog.bw = o.board
		ss.prog.width = boardWidth()
		ss.prog.gauge = o.leafGauge
		tickC := o.boardTick
		var ticker *time.Ticker
		if tickC == nil {
			ticker = time.NewTicker(500 * time.Millisecond)
			tickC = ticker.C
		}
		tickDone := make(chan struct{})
		tickStopped := make(chan struct{})
		go func() {
			defer close(tickStopped)
			for {
				select {
				case <-tickC:
					if o.beforeBoardTick != nil {
						o.beforeBoardTick()
					}
					ss.whileHealthy(func() { ss.prog.tick() })
					if o.afterBoardTick != nil {
						o.afterBoardTick()
					}
				case <-tickDone:
					return
				}
			}
		}()
		defer func() {
			if ticker != nil {
				ticker.Stop()
			}
			close(tickDone)
			<-tickStopped
			if result != nil {
				ss.prog.abort()
			}
		}()
	}

	// metis#32: >1 config → nested CV (records inner + per-family outer rows; the honest measure).
	if nested {
		return ss.runNestedCV(ctx, configPts, k, runFolds, stratify, shapeRunID)
	}

	fmt.Fprintf(out, "metis: single-level CV %s (%s) — %d config × %d folds\n", sh.ID, shapeRunID[:12], len(configPts), k)

	// The flat single-level CV path (1 config): the SingleDriver (a runtime sampler node, NOT the
	// deleted shape `driver:`) runs the sweeper once on all data → the sweeper's inner k-fold CV
	// scores the one config → (mean, SE, complexity) recorded to the ledger. metis#32: it MEASURES
	// ONLY — no `shipWinner` (shipping is `metis select --promote`).
	pass := &sweepPass{ss: ss, splitK: k, stratify: stratify, partRef: ss.partRef, runRole: runRoleFlatCV,
		hooks: ss.prog.forPass(-1)} // metis#30: the flat path's single pass
	res := sampler.Run(ctx, sampler.SingleDriver{}, func(sampler.SinglePoint) sampler.SweepResult {
		return ss.runSweeper(ctx, configPts, pass)
	}, sampler.ExecFor[sampler.SinglePoint, sampler.SweepResult](ss.parallel), nil)
	if err := ss.firstError(); err != nil {
		return err
	}
	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the report
	// metis#31: sort the fan-out's completion-order bookkeeping to a stable content key
	// BEFORE persisting, so manifest.json + the ledger are byte-deterministic across
	// serial/parallel runs (the winner/estimate are already deterministic; this makes
	// the on-disk artifacts match metis's content-addressing posture).
	sortPointRuns(pass.points)
	ss.man.Points = pass.points
	ss.configs = pass.configs
	if err := writeManifest(o.expPath, ss.man); err != nil {
		return ss.fail("write sweep manifest", err)
	}
	// Capture the sweep's code closure to a git side ref (metis#8/#14) — BEST-EFFORT: the
	// records + manifest are already valid, so a capture hiccup warns, never aborts.
	cohort, err := captureSweepCode(o, ss.man)
	if err != nil {
		ss.whileHealthy(func() {
			fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
		})
	}
	// Persist the raw per-fold rows to the shape's append-only ledger sidecar (metis#8/#18):
	// AggregateView reduces them read-time to per-config (mean, SE) — so metis#19's 1-SE
	// select re-reduces the same rows without a re-run.
	if err := writeSweepLedger(o.expPath, ss.man); err != nil {
		return ss.fail("write sweep ledger", err)
	}
	// Guard (metis#19): a parsimony rule (one-std-err/pct-loss) needs a measured complexity
	// for every swept family — else the parsimony axis is silently dropped and the winner is
	// quietly wrong. The raw fold rows are already persisted (re-selectable after a fix); only
	// the ship/report is gated. Checked here (post-fold) because HasComplexity is only known
	// after the folds run.
	if err := sampler.GuardComplexity(sh.Sweeper.Objective.Select, configStatsOf(ss.configs)); err != nil {
		return ss.fail("sweep complexity guard", err)
	}
	ss.whileHealthy(func() {
		ss.reportWinner(res)
		printRunSummary(out, o.expPath, now().Sub(sweepStart), len(ss.man.Points), cohort)
	})
	return ss.firstError()
}

// configStatsOf builds the per-config stats (with each config's family) from a completed
// sweep pass — the GuardComplexity input, matching what GridConfigs.Done reduces internally.
// Free over a []configScore so BOTH the flat path (ss.configs) and each driver:cv sealed
// outer fold (pass.configs) guard the same way (ARCH-DRY, metis#23 I1).
func configStatsOf(configs []configScore) []sampler.ConfigStat {
	stats := make([]sampler.ConfigStat, len(configs))
	for i, c := range configs {
		stats[i] = sampler.ConfigStat{Point: c.point, Family: sampler.FamilyOf(c.point), Score: c.meanSE}
	}
	return stats
}

// (metis#32: `shipWinner` was deleted — `metis run` no longer ships; the ship path moved to
// `metis select --promote`, which reconstructs the honest winner via `promotedExperiment` and runs
// it on all data. `shapeConfigToExperiment` (the all-data assembly) is now called from there.)

// runNestedCV drives driver:cv (metis#23): the OUTER resample around the black-box sweeper → the
// honest procedure estimate. A preamble materializes the k outer-analysis subset dirs ONCE; then
// the CVDriver, per outer fold, (a) runs the sweeper SEALED on analysis_i (confined via exp_path so
// its inner-CV cannot see outer-assessment) → a winner, then (b) refits+scores that winner on the
// held outer-assessment — a plain full-data fold run at OUTER k, held=i (post-selection, so
// unconfined and leakage-free; cv_folds's determinism reproduces the exact analysis_i partition).
// Aggregate(outer scores) → mean±SE: the estimate. It ships NO winner (estimation ≠ selection).
//
// PROVENANCE (deliberate, metis#23): the nested path writes NO grouped sweepManifest / ledger and
// does NO captureSweepCode. Each inner run's record.json still exists (via runResolvedExperiment),
// but a driver:cv run is estimation-only — it produces no shippable/reproducible winner — so the
// flat path's manifest+ledger+code-side-ref provenance (which exists to re-select/ship a winner
// without a re-run) has no consumer here. If a durable procedure-estimate provenance is later
// wanted (e.g. to compare estimates across code revisions), wire a thin nested manifest then.
func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, runFolds int, stratify bool, shapeRunID string) error {
	fmt.Fprintf(ss.out, "metis: nested-CV %s (%s) — %d outer fold(s) × (%d configs × %d inner folds)\n",
		ss.sh.ID, shapeRunID[:12], runFolds, len(configPts), k)

	// Preamble: materialize the k outer-analysis subset dirs ONCE (unconfined — outer-split reads
	// the full dataset to split it). Always split into k dirs (a stable partition); --fast just runs
	// fewer of them (runFolds ≤ k). Deterministic run id → the analysis_i refs are locatable.
	analysisRefs, err := ss.materializeOuterAnalysis(k, stratify)
	if err != nil {
		if first := ss.firstError(); first != nil {
			return first
		}
		return ss.fail("nested-CV preamble", err)
	}
	outerPart := sampler.PartitionRef(fmt.Sprintf("outer-cv-k%d-strat%t-seed%d", k, stratify, ss.sh.Seed))

	est := sampler.Run(ctx, sampler.CVDriver{K: runFolds, Stratify: stratify},
		func(p sampler.OuterFoldPoint) float64 {
			if ss.firstError() != nil {
				return 0
			}
			score, ferr := ss.runOuterFold(ctx, configPts, k, stratify, analysisRefs[p.Idx], outerPart, p.Idx)
			if ferr != nil {
				if ss.firstError() == nil {
					ss.fail(fmt.Sprintf("outer fold %d", p.Idx), ferr)
				}
				return 0
			}
			return score
		},
		sampler.ExecFor[sampler.OuterFoldPoint, float64](ss.parallel),
		// metis#30: outer-fold completions always emit. Error-gated: once runControl
		// latches, remaining closures return sentinel zeros — don't fold those into
		// the displayed est (the run is aborting; a fake 0 would tank the line).
		func(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
			ss.whileHealthy(func() { ss.prog.driverEvent(ev) })
		})
	if err := ss.firstError(); err != nil {
		return err
	}
	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the estimate report

	// metis#32: the nested run now RECORDS (unlike metis#23's estimation-only path) — persist the
	// inner + per-family outer rows accumulated in ss.man.Points so `metis select` can reduce them
	// (family from the outer rows, config from the inner rows). Sort to a stable content key first
	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
	sortPointRuns(ss.man.Points)
	if err := writeManifest(ss.o.expPath, ss.man); err != nil {
		return ss.fail("write nested sweep manifest", err)
	}
	cohort, cerr := captureSweepCode(ss.o, ss.man)
	if cerr != nil {
		ss.whileHealthy(func() {
			fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
		})
	}
	if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
		return ss.fail("write nested sweep ledger", err)
	}
	ss.whileHealthy(func() {
		ss.reportEstimate(est, runFolds)
		printRunSummary(ss.out, ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
	})
	return ss.firstError()
}

// materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
// and returns the k analysis_i refs (experiment-relative, so a sealed sweep reading one routes
// through exp_path → confined). Unconfined (outer-split reads the full dataset to split it).
func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]string, error) {
	baseOut, baseID := baseDatasetRef(ss.sh)
	var needs []string
	if baseID != "" {
		needs = []string{baseID}
	}
	osStep := experiment.Step{ID: outerSplitStepID, Uses: "metis/outer-split", Needs: needs,
		With: map[string]any{"dataset": baseOut, "k": outerK, "stratify": stratify}}
	steps := append(append([]experiment.Step{}, ss.sh.Data...), osStep)
	exp := experiment.Experiment{Header: ss.sh.Header, Steps: steps}
	exp.Type = "experiment"
	preID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
		return nil, ss.fail("nested-CV preamble address", err)
	}
	preOpts := ss.o
	preOpts.inSweep = true // one preamble run; skip the per-run capture noise
	preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
	preOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
	preOpts.runRole = runRoleNestedPreamble
	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
		return nil, err
	}
	refs := make([]string, outerK)
	for i := 0; i < outerK; i++ {
		refs[i] = filepath.ToSlash(filepath.Join("runs", preID, outerSplitStepID, fmt.Sprintf("analysis_%d", i)))
	}
	return refs, nil
}

// runOuterFold runs one outer fold: (a) the SEALED sweeper on analysis_i → a winner (confined via
// the exp_path chokepoint — readRoot = analysis_i abs), then (b) the refit-and-score of that winner
// on the held outer-assessment (a full-data fold run at outer-k, held=i; unconfined). Returns the
// honest outer-fold score.
func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k int, stratify bool, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
	if err != nil {
		return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
	}
	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
		stratify: stratify, partRef: ss.partRef,
		runRole: runRoleNestedInnerCV,
		hooks:   ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
	sres := ss.runSweeper(ctx, configPts, pass)
	if err := pass.firstError(); err != nil {
		return 0, err
	}
	// Guard (metis#19/#23 I1): the parsimony select rule needs a measured complexity for every
	// swept family — same guard the flat path runs before trusting its winner. Without it, a
	// parsimony-select + non-reporting-model shape would SILENTLY mis-select in each outer fold.
	if err := sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs)); err != nil {
		return 0, ss.fail(fmt.Sprintf("outer fold %d complexity guard", i), err)
	}

	// metis#32: record the sealed sweep's INNER rows (Level=inner, tagged with this outer fold).
	of := i
	rows := make([]pointRun, 0, len(pass.points)+len(sres.PerFamily))
	if !ss.whileHealthy(func() {
		for _, pr := range pass.points {
			pr.Level = "inner"
			pr.OuterFold = &of
			rows = append(rows, pr)
		}
	}) {
		return 0, errRunAborted
	}

	// (b) score EACH family's inner-winner on the held outer-assessment — post-selection, so
	// unconfined and leakage-free (each winner was selected SEALED within analysis_i; scoring on
	// the held-out fold never influenced that selection). One OUTER row per family → the honest
	// per-family measure `metis select` reduces (metis#32). The metis#23 estimate the CVDriver
	// aggregates stays the SHIP-family's outer score (the argmax-mean procedure's honest number).
	// The cv-split uses the OUTER k + stratify so cv_folds's determinism reproduces the exact
	// partition outer-split materialized (else the held fold ≠ analysis_i's assessment rows).
	shipFamily := sres.Ship.Family
	var shipScore float64
	for _, fam := range sortedFamilies(sres.PerFamily) {
		w := sres.PerFamily[fam]
		score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart, fam)
		if ferr != nil {
			return 0, ferr
		}
		if !ss.whileHealthy(func() {
			rows = append(rows, pointRun{
				RunID:      scoreID,
				FreeParams: freeParamMap(w.Point),
				Fold:       of, // the outer fold this held-out score is on
				Level:      "outer",
				OuterFold:  &of,
				Status:     status,
				// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
			})
			if fam == shipFamily {
				shipScore = score
			}
			fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
				i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
		}) {
			return 0, errRunAborted
		}
	}
	if !ss.addManPoints(rows) {
		return 0, errRunAborted
	}
	return shipScore, nil
}

// scoreOnOuterFold refit-and-scores one config's winner on the held outer-assessment fold i (a
// full-data fold run at outer-k; post-selection, so unconfined). Returns the held-out fold_score,
// the run id (→ its record.json carries the namespaced metric the ledger reads), and its status.
func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef, fam string) (float64, string, string, error) {
	scoreExp := ss.buildFoldExperiment(point, sampler.FoldPoint{Idx: i}, nil, k, stratify, outerPart)
	scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
	if err != nil {
		return 0, "", "", ss.fail(fmt.Sprintf("outer fold %d family %s score address", i, fam), err)
	}
	scoreOpts := ss.o
	scoreOpts.inSweep = true
	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
	scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
	scoreOpts.runRole = runRoleOuterScore
	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
	if err != nil {
		return 0, "", "", err
	}
	return run.Metrics[foldMetric], scoreID, run.Status, nil
}

// sortedFamilies returns the family keys of a per-family winner map in deterministic order
// (the recording + the returned ship-score must not depend on Go's random map iteration).
func sortedFamilies(perFamily map[string]sampler.Winner) []string {
	fams := make([]string, 0, len(perFamily))
	for fam := range perFamily {
		fams = append(fams, fam)
	}
	sort.Strings(fams)
	return fams
}

// reportEstimate prints the honest procedure estimate — mean±SE over the outer folds — and the
// standing reminder that driver:cv produces NO shippable winner (estimation ≠ selection).
func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
	fmt.Fprintf(ss.out, "metis: nested-CV estimate — mean %.4f (SE %.4f) over %d outer fold(s) — the HONEST procedure estimate (argmax-mean family)\n",
		est.Mean, est.SE, outerK)
	fmt.Fprintf(ss.out, "  (per-family honest estimates recorded to the ledger; choose + ship via `metis select --best --promote`)\n")
}

// runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
// synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
// the shared cached runner, record the manifest row, and return the fold_score the inner
// resample Sampler folds. A fatal outcome publishes through the experiment-wide
// runControl and returns 0; every sampler callback/sink rejects placeholders after
// publication, and the top level returns the stored concrete cause.
func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.FoldOutcome {
	ss := p.ss
	if p.firstError() != nil {
		return sampler.FoldOutcome{}
	}
	// Detect-and-abort: a mid-sweep HEAD-sha change breaks the shape-run's one-code
	// identity (per-fold records stay correct). Compares the HEAD sha only, not the dirty
	// flag — the sweep's own writes (runs/, manifest) dirty the tree (see codeID freeze).
	// metis#31: only a DEFINITE sha change aborts — `s != ""`. probeRepo swallows any
	// probe error to "", and under parallel fan-out concurrent `git status` contends on
	// .git/index.lock so a transient probe failure is expected; treating "" as a change
	// would false-abort the whole honest run.
	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx),
			fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
		return sampler.FoldOutcome{}
	}

	exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.stratify, p.partRef)
	runID, err := pointAddressOf(exp, ss.shapeBlobHash)
	if err != nil {
		p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx), err)
		return sampler.FoldOutcome{}
	}
	pointOpts := ss.o
	pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
	pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
	pointOpts.runRole = p.runRole
	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
	if runErr != nil {
		// runControl already published a concrete admitted-run failure. A queued or
		// late sibling returns errRunAborted; neither path may republish the sentinel.
		return sampler.FoldOutcome{}
	}
	if !p.addPoint(pointRun{
		RunID:      runID,
		FreeParams: freeParamMap(c),
		Fold:       f.Idx,
		Status:     run.Status,
		Metrics:    run.Metrics,
	}) {
		return sampler.FoldOutcome{}
	}
	// metis#19 M2: read the train step's realized-complexity metric. Present → the parsimony
	// rules consume it; absent (HasComplexity false) → the guard rejects a parsimony rule.
	cx, hasCx := run.Metrics[foldComplexityMetric]
	return sampler.FoldOutcome{Score: run.Metrics[foldMetric], Complexity: cx, HasComplexity: hasCx}
}

// buildFoldExperiment reconstructs the runnable per-fold experiment for one (config, fold):
// the data steps (as declared — cache-shared, config+fold-invariant) + the engine-synthesized
// cv-split partition step + the pipeline steps with the config's resolved `with` overlaid AND

codex
The run activity event is emitted at the right persistence seam, before the terminal `metis: <id> ok` line, and ineligible roles are assigned at preamble/outer-score sites. One subtle area is ordering: run-success activity can repaint before the sampler fold callback updates per-row state, so I’m checking whether the committed tests pin that transient or leave it as accepted behavior.
exec
/bin/zsh -lc "sed -n '1,380p' cmd/metis/progress_test.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/sampler"
	"github.com/xianxu/metis/pkg/shape"
)

// progressLine renders the aggregated one-line view of a sweep's progressState.
func TestProgressLine(t *testing.T) {
	nested := func(outerK, configK, foldK int, scores []float64) progressState {
		return progressState{
			nested: true,
			outerK: outerK, outerTotal: 3, outerKind: sampler.SizeExact,
			configK: configK, configTotal: 216, configKind: sampler.SizeExact,
			foldK: foldK, foldTotal: 1080, foldKind: sampler.SizeExact,
			outerScores: scores,
		}
	}
	cases := []struct {
		name string
		st   progressState
		want []string
		not  []string
	}{
		{"nested pre-outer", nested(0, 84, 421, nil),
			[]string{"outer folds 0/3", "configs scored 84/216", "inner-CV runs 421/1080", "est —"}, []string{"±"}},
		{"nested one outer", nested(1, 100, 500, []float64{0.82}),
			[]string{"outer folds 1/3", "est 0.8200"}, []string{"±"}},
		{"nested two outer", nested(2, 200, 900, []float64{0.80, 0.84}),
			[]string{"outer folds 2/3", "est 0.8200 ± 0.0200"}, nil},
		{"flat one config", progressState{
			foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact,
			flatScores: []float64{0.80, 0.84, 0.88}},
			[]string{"CV runs 3/5", "score 0.8400"}, []string{"configs", "outer", "folds 3/5"}},
		{"unknown kind", progressState{
			nested: true,
			outerK: 1, outerTotal: 0, outerKind: sampler.SizeUnknown,
			configK: 3, configTotal: 0, configKind: sampler.SizeUnknown},
			[]string{"outer folds 1/?", "configs scored 3/?"}, nil},
		{"budget kind", progressState{
			nested: true,
			outerK: 1, outerTotal: 8, outerKind: sampler.SizeBudget},
			[]string{"outer folds 1/≤8"}, nil},
	}
	for _, c := range cases {
		got := progressLine(c.st)
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: missing %q in %q", c.name, w, got)
			}
		}
		for _, n := range c.not {
			if strings.Contains(got, n) {
				t.Errorf("%s: unwanted %q in %q", c.name, n, got)
			}
		}
	}
}

// scriptedClock returns a now() that steps through the given instants (sticky last).
func scriptedClock(times ...time.Time) func() time.Time {
	i := 0
	return func() time.Time {
		t := times[min(i, len(times)-1)]
		i++
		return t
	}
}

func at(ms int) time.Time {
	return time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC).Add(time.Duration(ms) * time.Millisecond)
}

// The sink throttles fold-level emits to 1/s, ALWAYS emits on an outer (driver)
// completion, and emits a final line at finish().
func TestSweepProgress_ThrottleAndAlwaysEmit(t *testing.T) {
	var out strings.Builder
	// Clock: one reading per event. 10 fold events at 200ms spacing starting at t=0.
	// Emits: event 1 (the FIRST event always emits — started=false) and event 6
	// (t=1000, the first ≥1s after t=0). Events 2-5 and 7-10 are throttled.
	times := []time.Time{at(0)}
	for i := 1; i <= 10; i++ {
		times = append(times, at(i*200))
	}
	times = append(times, at(2100), at(2200)) // driver event, finish
	prog := newSweepProgress(&out, scriptedClock(times...), "maximize",
		progressTotals{nested: true, outer: 2, outerKind: sampler.SizeExact,
			configs: 4, configKind: sampler.SizeExact, folds: 20, foldKind: sampler.SizeExact})
	for i := 1; i <= 10; i++ {
		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 200)})
	}
	throttled := strings.Count(out.String(), "metis: progress")
	if throttled != 2 { // event 1 (first) + event 6 (throttle boundary)
		t.Fatalf("want 2 throttled emits, got %d:\n%s", throttled, out.String())
	}
	// A driver-level completion always emits, regardless of throttle.
	prog.driverEvent(sampler.ProgressEvent[sampler.OuterFoldPoint, float64]{K: 1, Total: 2, Kind: sampler.SizeExact, Out: 0.83})
	if got := strings.Count(out.String(), "metis: progress"); got != 3 {
		t.Fatalf("driver completion must always emit, got %d lines", got)
	}
	prog.finish()
	if got := strings.Count(out.String(), "metis: progress"); got != 4 {
		t.Fatalf("finish must emit the final line, got %d lines", got)
	}
	final := out.String()[strings.LastIndex(out.String(), "metis: progress"):]
	for _, w := range []string{"outer folds 1/2", "inner-CV runs 10/20", "est 0.8300"} {
		if !strings.Contains(final, w) {
			t.Errorf("final line missing %q: %q", w, final)
		}
	}
	if strings.ContainsAny(out.String(), "\x1b\r") {
		t.Error("plain lines must carry no escape codes / carriage returns")
	}
}

// A nil sink is a no-op everywhere (the non-sweep path stays silent).
func TestSweepProgress_NilSafe(t *testing.T) {
	var prog *sweepProgress
	hooks := prog.forPass(0)
	hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{K: 1})
	hooks.config(sampler.ProgressEvent[shape.Point, sampler.MeanSE]{K: 1})
	prog.driverEvent(sampler.ProgressEvent[sampler.OuterFoldPoint, float64]{K: 1})
	prog.finish() // must not panic
}

// Concurrent event fire from many goroutines is race-clean and loses no counts.
func TestSweepProgress_ConcurrentCounts(t *testing.T) {
	var out strings.Builder
	var mu sync.Mutex
	safeOut := writerFunc(func(p []byte) (int, error) { mu.Lock(); defer mu.Unlock(); return out.Write(p) })
	prog := newSweepProgress(safeOut, func() time.Time { return at(0) }, "maximize",
		progressTotals{nested: true, folds: 64, foldKind: sampler.SizeExact})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 8; i++ {
				prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
			}
		}()
	}
	wg.Wait()
	prog.finish()
	if !strings.Contains(out.String(), "inner-CV runs 64/64") {
		t.Errorf("sink-owned counter must see all 64 events (never ev.K):\n%s", out.String())
	}
}

// writerFunc adapts a func to io.Writer for the concurrency test.
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// movingRate: keep the latest 64 eligible completion times by event time. It is
// ready only after enough event-time evidence, and rate = (n-1)/(now-oldest).
func TestMovingRate(t *testing.T) {
	var r movingRate
	if _, ok := r.rate(at(0)); ok {
		t.Error("n=0 must be not-ok")
	}
	for i := 0; i < 15; i++ {
		r.add(at(i * 1000))
	}
	if _, ok := r.rate(at(15000)); ok {
		t.Error("15 completions must be below confidence")
	}
	var short movingRate
	for i := 0; i < 16; i++ {
		short.add(at(i * 900))
	}
	if _, ok := short.rate(at(15000)); ok {
		t.Error("16 completions spanning under 15s must be below confidence")
	}
	var ready movingRate
	for i := 0; i < 16; i++ {
		ready.add(at(i * 1000))
	}
	if got, ok := ready.rate(at(20000)); !ok || got < 44.9 || got > 45.1 {
		t.Errorf("16+ completions spanning ≥15s at now=20s → 45/min, got %v ok=%v", got, ok)
	}
	if got, ok := ready.rate(at(25000)); !ok || got >= 45 {
		t.Errorf("stall must decay the rate using now in the denominator, got %v ok=%v", got, ok)
	}

	// Reversed delivery: 65 completions 1s apart keeps the newest 64 by event time.
	var r2 movingRate
	for i := 64; i >= 0; i-- {
		r2.add(at(i * 1000))
	}
	// newest kept window = t=1s..64s; at now=65s: 63 completions over 64s ≈ 59.06/min.
	if got, ok := r2.rate(at(65000)); !ok || got < 58.9 || got > 59.2 {
		t.Errorf("reversed delivery latest-64 window: want ~59.06/min, got %v ok=%v", got, ok)
	}
	// ETA: remaining/rate.
	if eta, ok := ready.eta(at(20000), 45); !ok || eta != time.Minute {
		t.Errorf("eta 45 remaining at 45/min → 1m, got %v ok=%v", eta, ok)
	}
}

func TestOccupancyWindowRoundedMeanOfLastFourSamples(t *testing.T) {
	var w occupancyWindow
	for _, busy := range []int{1, 2, 3, 4} {
		w.add(busy, 12)
	}
	busy, capacity, ok := w.mean()
	if !ok || busy != 3 || capacity != 12 {
		t.Fatalf("[1,2,3,4]/12 mean = (%d,%d,%v); want (3,12,true)", busy, capacity, ok)
	}
	w.add(9, 12)
	busy, capacity, ok = w.mean()
	if !ok || busy != 5 || capacity != 12 {
		t.Fatalf("[2,3,4,9]/12 rounded mean = (%d,%d,%v); want (5,12,true)", busy, capacity, ok)
	}
}

func TestSweepProgressOccupancySamplesOnlyOnTick(t *testing.T) {
	var out strings.Builder
	busy := 0
	prog := newSweepProgress(&out, func() time.Time { return at(20000) }, "maximize",
		progressTotals{nested: true, folds: 32, foldKind: sampler.SizeExact})
	prog.bw = newBoardWriter(&out, func() time.Time { return at(20000) })
	prog.gauge = func() (int, int) { return busy, 12 }

	for _, v := range []int{1, 2, 3, 4} {
		busy = v
		prog.tick()
	}
	got, cap, ok := prog.occupancy.mean()
	if !ok || got != 3 || cap != 12 {
		t.Fatalf("tick samples [1,2,3,4] mean = (%d,%d,%v); want (3,12,true)", got, cap, ok)
	}

	for i := 0; i < 10; i++ {
		busy = 12
		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
	}
	got, cap, ok = prog.occupancy.mean()
	if !ok || got != 3 || cap != 12 {
		t.Fatalf("activity burst changed occupancy mean to (%d,%d,%v); want unchanged (3,12,true)", got, cap, ok)
	}

	busy = 5
	prog.tick()
	got, cap, ok = prog.occupancy.mean()
	if !ok || got != 4 || cap != 12 {
		t.Fatalf("fifth tick samples [2,3,4,5] mean = (%d,%d,%v); want (4,12,true)", got, cap, ok)
	}
}

// Per-pass rows: each forPass(i) hook folds into ITS row (closure-bound identity);
// driverEvent's Point.Idx collapses the right row to its held-out score.
func TestSweepProgress_PerPassRows(t *testing.T) {
	var out strings.Builder
	prog := newSweepProgress(&out, func() time.Time { return at(0) }, "maximize",
		progressTotals{nested: true, outer: 2, outerKind: sampler.SizeExact,
			configs: 4, configKind: sampler.SizeExact, folds: 12, foldKind: sampler.SizeExact})
	h0, h1 := prog.forPass(0), prog.forPass(1)
	fev := func(score float64) sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome] {
		return sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{Out: sampler.FoldOutcome{Score: score}}
	}
	cev := func(mean float64) sampler.ProgressEvent[shape.Point, sampler.MeanSE] {
		return sampler.ProgressEvent[shape.Point, sampler.MeanSE]{Out: sampler.MeanSE{Mean: mean}}
	}
	h0.fold(fev(0.7))
	h0.fold(fev(0.7))
	h1.fold(fev(0.8))
	h0.config(cev(0.70))
	h0.config(cev(0.75)) // pass 0's best (maximize)
	h1.config(cev(0.85))
	h1.config(cev(0.80))
	st := prog.boardState()
	if len(st.rows) != 2 {
		t.Fatalf("want 2 pass rows, got %+v", st.rows)
	}
	if r := st.rows[0]; r.foldK != 2 || r.configK != 2 || !r.hasBest || r.best != 0.75 || r.done {
		t.Errorf("row 0: %+v", r)
	}
	if r := st.rows[1]; r.foldK != 1 || r.configK != 2 || r.best != 0.85 || r.done {
		t.Errorf("row 1 (maximize keeps 0.85): %+v", r)
	}
	// Minimize direction flips the incumbent.
	prog2 := newSweepProgress(&out, func() time.Time { return at(0) }, "minimize",
		progressTotals{nested: true, outer: 1})
	h := prog2.forPass(0)
	h.config(cev(0.5))
	h.config(cev(0.3))
	h.config(cev(0.4))
	if r := prog2.boardState().rows[0]; r.best != 0.3 {
		t.Errorf("minimize keeps 0.3: %+v", r)
	}
	// driverEvent collapses row 1 by its Point.Idx; row 0 stays in-flight.
	prog.driverEvent(sampler.ProgressEvent[sampler.OuterFoldPoint, float64]{
		K: 1, Total: 2, Kind: sampler.SizeExact, Point: sampler.OuterFoldPoint{Idx: 1}, Out: 0.83})
	st = prog.boardState()
	if r := st.rows[1]; !r.done || r.heldOut != 0.83 {
		t.Errorf("row 1 must collapse to held-out 0.83: %+v", r)
	}
	if st.rows[0].done {
		t.Errorf("row 0 must stay in-flight: %+v", st.rows[0])
	}
}

func TestSweepProgressActivityRunEventsOwnAggregateRunCounterAndRate(t *testing.T) {
	var out strings.Builder
	prog := newSweepProgress(&out, func() time.Time { return at(20000) }, "maximize",
		progressTotals{nested: true, outer: 1, outerKind: sampler.SizeExact,
			configs: 2, configKind: sampler.SizeExact, folds: 16, foldKind: sampler.SizeExact})
	hooks := prog.forPass(0)
	hooks.fold(sampler.ProgressEvent[sampler.FoldPoint, sampler.FoldOutcome]{
		Out: sampler.FoldOutcome{Score: 0.7},
	})
	st := prog.boardState()
	if st.st.foldK != 0 {
		t.Fatalf("sampler fold callback advanced aggregate run counter to %d; want typed events only", st.st.foldK)
	}
	if st.rows[0].foldK != 1 {
		t.Fatalf("sampler fold callback should retain per-row duties; row = %+v", st.rows[0])
	}

	prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedPreamble, RunID: "pre", At: at(0)})
	if got := prog.boardState().st.foldK; got != 0 {
		t.Fatalf("ineligible preamble advanced aggregate run counter to %d", got)
	}

	for i := 15; i >= 0; i-- {
		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, RunID: "inner", At: at(i * 1000)})
	}
	st = prog.boardState()
	if st.st.foldK != 16 {
		t.Fatalf("typed eligible run events advanced foldK to %d; want 16", st.st.foldK)
	}
	if got, ok := st.rate.rate(at(20000)); !ok || got < 44.9 || got > 45.1 {
		t.Fatalf("typed event-time rate = %v ok=%v; want 45/min", got, ok)
	}
}

exec
/bin/zsh -lc 'rg -n "Activity|activity|startup|Mature|first eligible|lag|row|foldK|last-run|monotonic|occupancy|stale|fold callback|typed" cmd/metis/*_test.go' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
cmd/metis/board_test.go:13:// renderBoard is the PURE frame: aggregate line, fold rows (✓ done / ▸ in-flight /
cmd/metis/board_test.go:17:	mkState := func(rows []passRow) boardState {
cmd/metis/board_test.go:23:				foldK: 47, foldTotal: 108, foldKind: sampler.SizeExact,
cmd/metis/board_test.go:26:			rows: rows,
cmd/metis/board_test.go:33:	rows := []passRow{
cmd/metis/board_test.go:35:		{configK: 8, foldK: 25, best: 0.834, hasBest: true},
cmd/metis/board_test.go:38:	lines := renderBoard(mkState(rows), boardEnv{width: 100, now: at(21176), busy: 8, capacity: 8})
cmd/metis/board_test.go:51:	if len(lines) != 5 { // aggregate + 3 fold rows + leaves
cmd/metis/board_test.go:58:	// Per-row denominators derive from the aggregate totals (36 configs / 3 outer = 12).
cmd/metis/board_test.go:59:	// All-done: every row ✓, no ETA segment (nothing remaining).
cmd/metis/board_test.go:62:	st.st.outerK, st.st.foldK, st.st.configK = 3, 108, 36
cmd/metis/board_test.go:66:		t.Errorf("all-done: no in-flight rows, no ETA:\n%s", frame)
cmd/metis/board_test.go:69:	// Flat (no rows): exactly 2 lines — the aggregate + slots/rate.
cmd/metis/board_test.go:70:	flat := boardState{st: progressState{foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact, flatScores: []float64{0.8}}}
cmd/metis/board_test.go:75:	// Overflow: 14 folds → 12 rows + "… +2 more" + slots/rate + aggregate = 15 lines.
cmd/metis/board_test.go:86:	narrow := renderBoard(mkState(rows), boardEnv{width: 30, now: at(2824), busy: 8, capacity: 8})
cmd/metis/board_test.go:87:	for _, l := range narrow {
cmd/metis/board_test.go:92:	if !strings.Contains(strings.Join(narrow, "\n"), "…") {
cmd/metis/board_test.go:97:	frame = strings.Join(renderBoard(mkState(rows), boardEnv{width: 100, now: at(21176)}), "\n")
cmd/metis/board_test.go:106:	st = mkState(rows)
cmd/metis/board_test.go:117:		nested: true, foldTotal: 10, foldKind: sampler.SizeExact,
cmd/metis/board_test.go:125:			t.Fatalf("startup frame missing %q:\n%s", want, frame)
cmd/metis/board_test.go:130:			t.Fatalf("startup frame made diagnosis %q:\n%s", forbidden, frame)
cmd/metis/board_test.go:134:	bs.st.foldK = 1
cmd/metis/board_test.go:138:		t.Fatalf("startup line must disappear after first eligible run:\n%s", frame)
cmd/metis/board_test.go:145:		foldTotal: 20, foldKind: sampler.SizeExact,
cmd/metis/board_test.go:153:			t.Fatalf("flat startup frame missing %q:\n%s", want, frame)
cmd/metis/board_test.go:157:		t.Fatalf("flat startup frame contains nested or diagnostic wording:\n%s", frame)
cmd/metis/board_test.go:160:	bs.st.foldK = 1
cmd/metis/board_test.go:169:		t.Fatalf("flat pre-confidence frame must be post-startup without ETA:\n%s", frame)
cmd/metis/board_test.go:173:func TestRenderBoardMatureShowsLastRunAge(t *testing.T) {
cmd/metis/board_test.go:181:			nested: true, foldK: 16, foldTotal: 32, foldKind: sampler.SizeExact,
cmd/metis/board_test.go:317:	for _, stale := range []string{"folds 2/8", "folds/min", "ETA"} {
cmd/metis/board_test.go:318:		if strings.Contains(suffix, stale) {
cmd/metis/board_test.go:319:			t.Errorf("discard/close redrew stale token %q: %q", stale, suffix)
cmd/metis/board_test.go:332:// Board mode end-to-end over the fixture sweep: frames paint (cursor hide, fold rows),
cmd/metis/board_test.go:454:			t.Errorf("post-publication board output contains stale token %q:\n%q", forbidden, suffix)
cmd/metis/board_test.go:475:// the flag path is what we can pin here; the char-device branch is covered by the
cmd/metis/board_test.go:477:func TestCmdRun_NoTUIFlagParses(t *testing.T) {
cmd/metis/board_test.go:569:// flushes at the size cap rather than growing without limit.
cmd/metis/identity_e2e_test.go:71:// in-closure .py) produce two DISTINCT ledger rows — distinct code_fingerprints, neither
cmd/metis/identity_e2e_test.go:73:// fingerprint over the run-end D closure → rowsFromManifest → the ledger dedups on
cmd/metis/identity_e2e_test.go:134:	// Both runs' rows land in the ledger: SAME point-address, DIFFERENT fingerprint → 2 rows.
cmd/metis/identity_e2e_test.go:136:	led.Append(rowsFromManifest(man, map[string]record.RunRecord{pa: rec1})...)
cmd/metis/identity_e2e_test.go:137:	led.Append(rowsFromManifest(man, map[string]record.RunRecord{pa: rec2})...)
cmd/metis/identity_e2e_test.go:139:		t.Fatalf("same config + changed code must keep BOTH rows (the metis#27 collision fix), got %d", len(led.Rows))
cmd/metis/identity_e2e_test.go:142:		t.Errorf("both rows should share the point-address %q, got %q and %q", pa, led.Rows[0].PointAddr, led.Rows[1].PointAddr)
cmd/metis/identity_e2e_test.go:145:		t.Error("the two rows must carry DISTINCT code_fingerprints")
cmd/metis/activity_test.go:12:type activityFakeExec struct {
cmd/metis/activity_test.go:18:func (f *activityFakeExec) Execute(step experiment.Step, runDir string) (experiment.StepResult, error) {
cmd/metis/activity_test.go:23:func TestActivityExecutorEmitsOneStepSuccessAfterSuccessfulInnerExecution(t *testing.T) {
cmd/metis/activity_test.go:29:	inner := &activityFakeExec{result: wantResult}
cmd/metis/activity_test.go:30:	var events []activityEvent
cmd/metis/activity_test.go:32:	got, err := activityExecutor{
cmd/metis/activity_test.go:35:		emit:  func(ev activityEvent) { events = append(events, ev) },
cmd/metis/activity_test.go:50:	if events[0].Kind != activityStepSuccess || events[0].StepID != "train" || !events[0].At.Equal(at) {
cmd/metis/activity_test.go:55:func TestActivityExecutorEmitsNothingOnInnerErrorAndPreservesFailure(t *testing.T) {
cmd/metis/activity_test.go:57:	inner := &activityFakeExec{err: wantErr}
cmd/metis/activity_test.go:58:	var events []activityEvent
cmd/metis/activity_test.go:60:	got, err := activityExecutor{
cmd/metis/activity_test.go:63:		emit:  func(ev activityEvent) { events = append(events, ev) },
cmd/metis/shapesweep_test.go:89:// runs — the winner is refit on all rows (no _fold) and predicted → a submission artifact.
cmd/metis/shapesweep_test.go:173:// nested CV to the ledger — inner rows (Level=inner) per (outer-fold, config, inner-fold) tagged
cmd/metis/shapesweep_test.go:174:// with their outer fold, + one outer row (Level=outer) per (outer-fold, family) — and reports the
cmd/metis/shapesweep_test.go:176:// family) × 2 inner × 2 outer folds → 8 inner + 2 outer rows.
cmd/metis/shapesweep_test.go:188:		// metis#27: the code_fingerprint must reach the PERSISTED row (capture-before-ledger
cmd/metis/shapesweep_test.go:189:		// ordering) — a reorder would silently yield empty-fingerprint rows.
cmd/metis/shapesweep_test.go:191:			t.Errorf("a swept ledger row must carry a non-empty code_fingerprint; got %+v", r)
cmd/metis/shapesweep_test.go:196:			// metis#32: an inner row is tagged with its outer fold (so select can pool per config).
cmd/metis/shapesweep_test.go:198:				t.Errorf("an inner row must carry its outer-fold coordinate; got %+v", r)
cmd/metis/shapesweep_test.go:203:			t.Errorf("a nested-run ledger row must be Level inner|outer; got %q in %+v", r.Level, r)
cmd/metis/shapesweep_test.go:207:		t.Errorf("2 configs × 2 inner × 2 outer folds → 8 inner rows, got %d", nInner)
cmd/metis/shapesweep_test.go:210:		t.Errorf("1 family × 2 outer folds → 2 outer rows, got %d", nOuter)
cmd/metis/shapesweep_test.go:221:// (FoldOutcome.Complexity → the recorded inner rows). Proves runPipelineFold reads the metric and
cmd/metis/shapesweep_test.go:223:// the metric reaches the recorded inner rows.)
cmd/metis/shapesweep_test.go:230:	// Every recorded INNER row carries the namespaced per-fold complexity (fake emits cx per model).
cmd/metis/shapesweep_test.go:239:			t.Errorf("each inner ledger row must carry train.complexity; got %+v", r.Metrics)
cmd/metis/shapesweep_test.go:243:		t.Fatal("expected recorded inner rows carrying complexity")
cmd/metis/shapesweep_test.go:248:// NOT emit complexity → a hard error (raw rows still persisted; only ship/report is gated).
cmd/metis/shapesweep_test.go:265:	// outer loop, so a guard error aborts before any rows persist here — unlike the old flat path,
cmd/metis/shapesweep_test.go:266:	// raw rows are NOT re-selectable after a nested guard error (a re-run recomputes cheaply from cache).
cmd/metis/shapesweep_test.go:286:// rows, and runs predict → submission). A multi-config shape runs nested; NO submission artifact is
cmd/metis/shapesweep_test.go:309:// records its fold rows, and does NOT ship — the outer selection loop has one candidate, so there is
cmd/metis/shapesweep_test.go:343:	// It RECORDS its fold rows (measure) ...
cmd/metis/shapesweep_test.go:346:		t.Error("the flat single-level CV path must record its fold rows to the ledger")
cmd/metis/shapesweep_test.go:355:func TestShapeSweepActivityRunRolesFromFlatAndNestedCallPaths(t *testing.T) {
cmd/metis/shapesweep_test.go:363:			activity: func(ev activityEvent) {
cmd/metis/shapesweep_test.go:364:				if ev.Kind == activityRunSuccess {
cmd/metis/shapesweep_test.go:384:			activity: func(ev activityEvent) {
cmd/metis/shapesweep_test.go:385:				if ev.Kind == activityRunSuccess {
cmd/metis/shapesweep_test.go:594:	if !strings.Contains(s, "done in 1m30s") || !strings.Contains(s, "42 rows") || !strings.Contains(s, "metis select s.md") {
cmd/metis/family_test.go:10:// FamilyEstimate groups OUTER rows by family (not exact config) and reduces the metric over
cmd/metis/family_test.go:30:	// also drop an INNER row for rf — it must be IGNORED by FamilyEstimate (outer-only).
cmd/metis/family_test.go:43:	if math.Abs(rf.Mean-0.79) > 1e-9 { // (0.80+0.78)/2, NOT contaminated by the 0.99 inner row
cmd/metis/family_test.go:44:		t.Errorf("rf family mean=%v want 0.79 (pooled two outer folds, inner row ignored)", rf.Mean)
cmd/metis/shipe2e_test.go:111:	// metis#32: the nested run RECORDS the honest per-config data (inner rows) + per-family outer
cmd/metis/shipe2e_test.go:112:	// rows to the ledger, and does NOT ship (shipping is `metis select --promote`).
cmd/metis/shipe2e_test.go:124:		t.Errorf("nested run must record inner AND outer rows; got %d inner, %d outer", nInner, nOuter)
cmd/metis/ledger_cmd_test.go:64:// not the raw fold rows — the honest leaderboard.
cmd/metis/ledger_cmd_test.go:73:	// 2 configs (a, b) — not 4 raw fold rows. Config a (mean 0.85) sorts above b (0.71).
cmd/metis/ledger_cmd_test.go:75:	if len(lines) != 1+2 { // header + 2 config rows
cmd/metis/ledger_cmd_test.go:76:		t.Fatalf("expected a header + 2 per-config rows (aggregated), got %d lines:\n%s", len(lines), s)
cmd/metis/ledger_cmd_test.go:80:		t.Errorf("best-first row should be config a (mean 0.85); got: %s", lines[1])
cmd/metis/ledger_cmd_test.go:87:// hoistShapePath pulls the single <shape.md> positional out regardless of flag position.
cmd/metis/ledger_cmd_test.go:89:	// flags before AND after the path both work (the stdlib-flag-stops-at-positional fix).
cmd/metis/ledger_cmd_test.go:94:		p, flags, err := hoistShapePath(args)
cmd/metis/ledger_cmd_test.go:98:		if len(flags) != 2 {
cmd/metis/ledger_cmd_test.go:99:			t.Errorf("hoistShapePath(%v) flags = %v; want the 2 flag tokens", args, flags)
cmd/metis/ledger_cmd_test.go:112:// two fingerprint cohorts and a legacy blank row, plus runs/<id>/record.json for the two
cmd/metis/ledger_cmd_test.go:190:		t.Error("unexpected flag must error, not be ignored")
cmd/metis/ledger_cmd_test.go:195:// (one resolver — the flags must not diverge in matching semantics again).
cmd/metis/ledger_cmd_test.go:205:	// A no-match prefix now errors with the cohort listing (was: silent "(no rows)").
cmd/metis/select_cmd_test.go:70:	// INNER rows (config side): rf md=8 is the inner champion; logreg C=1 is logreg's best.
cmd/metis/select_cmd_test.go:78:		// OUTER rows (family side): logreg holds tight (mean 0.81, SE ~0.01); rf overfit → wide
cmd/metis/select_cmd_test.go:121:// A multi-family ledger with NO outer rows (a flat/`--fast`-less inner-only ledger) is a SHARP
cmd/metis/select_cmd_test.go:126:	shapePath := writeSelectLedger(t, dir, taggedShapeForSelect, false) // inner rows only
cmd/metis/select_cmd_test.go:132:		t.Errorf("the error should point at the missing outer-CV rows; got %v", err)
cmd/metis/select_cmd_test.go:146:	orow := func(cf, addr string, fp map[string]any, ofold int, score float64) ledger.Row {
cmd/metis/select_cmd_test.go:153:		orow("cf1aaaaa", "o-lr-0", lr1, 0, 0.80), orow("cf1aaaaa", "o-rf-0", rf8, 0, 0.74),
cmd/metis/select_cmd_test.go:154:		orow("cf2bbbbb", "o-lr-1", lr1, 1, 0.82), orow("cf2bbbbb", "o-rf-1", rf8, 1, 0.82),
cmd/metis/select_cmd_test.go:286:// ── metis#41: select --point — publish an operator-chosen config by ledger row ──
cmd/metis/select_cmd_test.go:306:// A prefix matching rows of MORE THAN ONE config is ambiguous — loud error listing candidates.
cmd/metis/select_cmd_test.go:325:		t.Fatalf("--point with no matching row must error naming the prefix; got %v", err)
cmd/metis/select_cmd_test.go:330:// guard's filter applies before the row lookup (no silent cross-version ship).
cmd/metis/select_cmd_test.go:337:	irow := func(cf, addr string, fp map[string]any, score float64) ledger.Row {
cmd/metis/select_cmd_test.go:343:	led.Append(irow("cf1aaaaa", "i-lr1-0", lr1, 0.80), irow("cf2bbbbb", "i-rf8-0", rf8, 0.86))
cmd/metis/select_cmd_test.go:367:// --point --promote reconstructs EXACTLY the row's config and ships it as point-{family}-{hash}
cmd/metis/select_cmd_test.go:392:		t.Errorf("the promoted run must carry the row's config (rf max_depth=8); record:\n%s", rec)
cmd/metis/select_cmd_test.go:396:// Close-review finding 2 (metis#41): a config with FAILED fold rows must be VISIBLE at --point —
cmd/metis/select_cmd_test.go:423:	if !strings.Contains(out.String(), "FAILED fold rows") {
cmd/metis/select_cmd_test.go:424:		t.Errorf("--point must warn loudly about failed fold rows; got:\n%s", out.String())
cmd/metis/select_cmd_test.go:504:	orow := func(cf, addr string, fp map[string]any, ofold int, score float64) ledger.Row {
cmd/metis/select_cmd_test.go:511:		orow("cf1aaaaa", "o-lr-0", lr1, 0, 0.80),
cmd/metis/select_cmd_test.go:512:		orow("cf2bbbbb", "o-lr-1", lr1, 1, 0.82),
cmd/metis/select_cmd_test.go:528:	// The inline summary: one line per cohort with its row count (renderCohorts shape).
cmd/metis/select_cmd_test.go:529:	if !strings.Contains(err.Error(), "cf1aaaaa") || !strings.Contains(err.Error(), "rows") {
cmd/metis/select_cmd_test.go:536:func TestSelect_CohortFlag(t *testing.T) {
cmd/metis/select_cmd_test.go:567:	// The logreg winner's handle = the first ledger row of that config (i-lr1-0).
cmd/metis/run_test.go:182:func TestRunExperimentActivityRunSuccessPublishesAfterRequiredArtifacts(t *testing.T) {
cmd/metis/run_test.go:195:	var runEvents []activityEvent
cmd/metis/run_test.go:199:		runID:    "run-activity",
cmd/metis/run_test.go:204:		activity: func(ev activityEvent) {
cmd/metis/run_test.go:205:			if ev.Kind != activityRunSuccess {
cmd/metis/run_test.go:225:	if runEvents[0].RunID != "run-activity" {
cmd/metis/run_test.go:226:		t.Fatalf("run event = %+v; want runID run-activity", runEvents[0])
cmd/metis/run_test.go:233:func TestRunExperimentActivityRunSuccessNotPublishedForFailedExecution(t *testing.T) {
cmd/metis/run_test.go:246:	var runEvents []activityEvent
cmd/metis/run_test.go:249:		runID:    "run-failed-activity",
cmd/metis/run_test.go:254:		activity: func(ev activityEvent) {
cmd/metis/run_test.go:255:			if ev.Kind == activityRunSuccess {
cmd/metis/run_test.go:263:	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "run.json")); statErr != nil {
cmd/metis/run_test.go:266:	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "record.json")); statErr != nil {
cmd/metis/run_test.go:274:func TestRunExperimentActivityRunSuccessNotPublishedWhenRecordPersistenceFails(t *testing.T) {
cmd/metis/run_test.go:290:	var runEvents []activityEvent
cmd/metis/run_test.go:298:		activity: func(ev activityEvent) {
cmd/metis/run_test.go:299:			if ev.Kind == activityRunSuccess {
cmd/metis/runcontrol_test.go:510:func TestRunControlActivityEmitterDropsLateStepAndRunEventsAfterFailure(t *testing.T) {
cmd/metis/runcontrol_test.go:512:	var events []activityEvent
cmd/metis/runcontrol_test.go:513:	emit := runControlActivityEmitter(control, func(ev activityEvent) {
cmd/metis/runcontrol_test.go:517:	emit(activityEvent{Kind: activityStepSuccess, StepID: "prep"})
cmd/metis/runcontrol_test.go:519:		t.Fatalf("healthy activity events = %d; want 1", len(events))
cmd/metis/runcontrol_test.go:523:	emit(activityEvent{Kind: activityStepSuccess, StepID: "late-step"})
cmd/metis/runcontrol_test.go:524:	emit(activityEvent{Kind: activityRunSuccess, RunID: "late-run"})
cmd/metis/runcontrol_test.go:526:		t.Fatalf("late activity after failure was published: %+v", events)
cmd/metis/fingerprints_test.go:16:// cohortSummaries groups by full fingerprint (legacy "" included), counts rows by level,
cmd/metis/fingerprints_test.go:46:		t.Errorf("aaaa row counts: %+v", a)
cmd/metis/fingerprints_test.go:62:// ExtraCommits is DISTINCT-commit cardinality minus one, order-proof: rows arrive
cmd/metis/fingerprints_test.go:64:// rows of the same second commit must not inflate the count (plan-review finding).
cmd/metis/fingerprints_test.go:73:		// Interleaved times: c2 latest, but rows of c1 and c2 alternate out of order.
cmd/metis/fingerprints_test.go:86:	if cs[0].ExtraCommits != 1 { // {c1,c2} → 2 distinct → 1 extra, regardless of row order
cmd/metis/fingerprints_test.go:114:// renderCohorts prints one line per cohort: short fingerprint (or "(legacy)"), row counts
cmd/metis/fingerprints_test.go:137:// dirty flag + capture status — the dirty bit must not vanish behind a missing commit
cmd/metis/parallel_test.go:30:	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds = 6 per-fold rows
cmd/metis/ledger_test.go:12:// rowsFromManifest is pure: it turns a sweep manifest + the per-point records into
cmd/metis/ledger_test.go:13:// ledger rows (namespaced per-step metrics from the records; sweep-SHA + point-address
cmd/metis/ledger_test.go:31:	rows := rowsFromManifest(man, records)
cmd/metis/ledger_test.go:32:	if len(rows) != 2 {
cmd/metis/ledger_test.go:33:		t.Fatalf("want 2 rows, got %d", len(rows))
cmd/metis/ledger_test.go:35:	if rows[0].PointAddr != "addr-a" || rows[0].CodeFingerprint != "cf1" {
cmd/metis/ledger_test.go:36:		t.Errorf("row 0 keys wrong: %+v", rows[0])
cmd/metis/ledger_test.go:38:	if rows[0].Metrics["train.cv_score"] != 0.81 {
cmd/metis/ledger_test.go:39:		t.Errorf("metric should be NAMESPACED train.cv_score=0.81; got %v", rows[0].Metrics)
cmd/metis/ledger_test.go:41:	if rows[1].Status != "failed" {
cmd/metis/ledger_test.go:42:		t.Errorf("failed point should carry failed status; got %+v", rows[1])
cmd/metis/ledger_test.go:46:// promotedExperiment reconstructs the winner's experiment for a row by matching its
cmd/metis/ledger_test.go:94:// The row's free-params round-trip through the CSV as the same values the shape's
cmd/metis/nestedcv_e2e_test.go:42:// scoring execute, the run reports a mean±SE PROCEDURE estimate, RECORDS inner+outer rows to the
cmd/metis/nestedcv_e2e_test.go:72:	// metis#50: the run ends with a paste-ready summary — elapsed, rows → ledger, cohort,
cmd/metis/nestedcv_e2e_test.go:78:	for _, want := range []string{"rows →", ".ledger.csv", "cohort ", "metis select ", "--fingerprint ", "--best --promote", "metis ledger fingerprints "} {
cmd/metis/nestedcv_e2e_test.go:88:	// metis#32: the nested run RECORDS both inner and outer rows (it no longer records nothing):
cmd/metis/nestedcv_e2e_test.go:89:	// inner rows (Level=inner) per (outer-fold, config, inner-fold); one outer row (Level=outer)
cmd/metis/nestedcv_e2e_test.go:102:		t.Errorf("nested run must record inner rows (Level=inner); got none in %d rows", len(led.Rows))
cmd/metis/nestedcv_e2e_test.go:105:		t.Errorf("nested run must record one outer row per (outer-fold, family) = 2; got %d", nOuter)
cmd/metis/nestedcv_e2e_test.go:121:// caveat (records inner/outer rows; ship via `metis select --promote`) is printed.
cmd/metis/nestedcv_e2e_test.go:143:// fold trains on (k-1)/k of the rows; m only sets how many unbiased samples of that estimand run).
cmd/metis/nestedcv_e2e_test.go:144:// Asserts: m held-out scores, m outer ledger rows (folds 0..m-1 of the k-partition), and the
cmd/metis/nestedcv_e2e_test.go:182:		t.Errorf("outer rows should cover exactly sampled folds {0,1} of k=3, got %v", outerFolds)
cmd/metis/caching_test.go:135:func TestCacheActivityEmitsOnColdExecutionAndWarmHit(t *testing.T) {
cmd/metis/caching_test.go:144:id: cache-activity
cmd/metis/caching_test.go:157:	var stepEvents []activityEvent
cmd/metis/caching_test.go:168:			activity: func(ev activityEvent) {
cmd/metis/caching_test.go:169:				if ev.Kind == activityStepSuccess {
cmd/metis/caching_test.go:180:	out1 := run("activity-cold")
cmd/metis/caching_test.go:184:	out2 := run("activity-warm")
cmd/metis/caching_test.go:192:		if ev.Kind != activityStepSuccess || ev.StepID != "prep" {
cmd/metis/caching_test.go:529:	badRefs := []record.CodeRef{{Repo: tmp, Path: "nope.py", BlobHash: "stale"}}
cmd/metis/progress_test.go:15:	nested := func(outerK, configK, foldK int, scores []float64) progressState {
cmd/metis/progress_test.go:20:			foldK: foldK, foldTotal: 1080, foldKind: sampler.SizeExact,
cmd/metis/progress_test.go:37:			foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact,
cmd/metis/progress_test.go:93:			configs: 4, configKind: sampler.SizeExact, folds: 20, foldKind: sampler.SizeExact})
cmd/metis/progress_test.go:95:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 200)})
cmd/metis/progress_test.go:137:		progressTotals{nested: true, folds: 64, foldKind: sampler.SizeExact})
cmd/metis/progress_test.go:144:				prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
cmd/metis/progress_test.go:207:	var w occupancyWindow
cmd/metis/progress_test.go:226:		progressTotals{nested: true, folds: 32, foldKind: sampler.SizeExact})
cmd/metis/progress_test.go:234:	got, cap, ok := prog.occupancy.mean()
cmd/metis/progress_test.go:241:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, At: at(i * 1000)})
cmd/metis/progress_test.go:243:	got, cap, ok = prog.occupancy.mean()
cmd/metis/progress_test.go:245:		t.Fatalf("activity burst changed occupancy mean to (%d,%d,%v); want unchanged (3,12,true)", got, cap, ok)
cmd/metis/progress_test.go:250:	got, cap, ok = prog.occupancy.mean()
cmd/metis/progress_test.go:256:// Per-pass rows: each forPass(i) hook folds into ITS row (closure-bound identity);
cmd/metis/progress_test.go:257:// driverEvent's Point.Idx collapses the right row to its held-out score.
cmd/metis/progress_test.go:262:			configs: 4, configKind: sampler.SizeExact, folds: 12, foldKind: sampler.SizeExact})
cmd/metis/progress_test.go:278:	if len(st.rows) != 2 {
cmd/metis/progress_test.go:279:		t.Fatalf("want 2 pass rows, got %+v", st.rows)
cmd/metis/progress_test.go:281:	if r := st.rows[0]; r.foldK != 2 || r.configK != 2 || !r.hasBest || r.best != 0.75 || r.done {
cmd/metis/progress_test.go:282:		t.Errorf("row 0: %+v", r)
cmd/metis/progress_test.go:284:	if r := st.rows[1]; r.foldK != 1 || r.configK != 2 || r.best != 0.85 || r.done {
cmd/metis/progress_test.go:285:		t.Errorf("row 1 (maximize keeps 0.85): %+v", r)
cmd/metis/progress_test.go:294:	if r := prog2.boardState().rows[0]; r.best != 0.3 {
cmd/metis/progress_test.go:297:	// driverEvent collapses row 1 by its Point.Idx; row 0 stays in-flight.
cmd/metis/progress_test.go:301:	if r := st.rows[1]; !r.done || r.heldOut != 0.83 {
cmd/metis/progress_test.go:302:		t.Errorf("row 1 must collapse to held-out 0.83: %+v", r)
cmd/metis/progress_test.go:304:	if st.rows[0].done {
cmd/metis/progress_test.go:305:		t.Errorf("row 0 must stay in-flight: %+v", st.rows[0])
cmd/metis/progress_test.go:309:func TestSweepProgressActivityRunEventsOwnAggregateRunCounterAndRate(t *testing.T) {
cmd/metis/progress_test.go:313:			configs: 2, configKind: sampler.SizeExact, folds: 16, foldKind: sampler.SizeExact})
cmd/metis/progress_test.go:319:	if st.st.foldK != 0 {
cmd/metis/progress_test.go:320:		t.Fatalf("sampler fold callback advanced aggregate run counter to %d; want typed events only", st.st.foldK)
cmd/metis/progress_test.go:322:	if st.rows[0].foldK != 1 {
cmd/metis/progress_test.go:323:		t.Fatalf("sampler fold callback should retain per-row duties; row = %+v", st.rows[0])
cmd/metis/progress_test.go:326:	prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedPreamble, RunID: "pre", At: at(0)})
cmd/metis/progress_test.go:327:	if got := prog.boardState().st.foldK; got != 0 {
cmd/metis/progress_test.go:332:		prog.activity(activityEvent{Kind: activityRunSuccess, Role: runRoleNestedInnerCV, RunID: "inner", At: at(i * 1000)})
cmd/metis/progress_test.go:335:	if st.st.foldK != 16 {
cmd/metis/progress_test.go:336:		t.Fatalf("typed eligible run events advanced foldK to %d; want 16", st.st.foldK)
cmd/metis/progress_test.go:339:		t.Fatalf("typed event-time rate = %v ok=%v; want 45/min", got, ok)
cmd/metis/trace_test.go:255:t._classify(os.path.join(r, ".venv", "x", "pandas.py"))   # venv/site-packages  -> DROP (flag)

exec
/bin/zsh -lc "sed -n '1,230p' cmd/metis/board_test.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/metis/pkg/sampler"
)

// renderBoard is the PURE frame: aggregate line, fold rows (✓ done / ▸ in-flight /
// queued), overflow cap, leaves+throughput line. NO ANSI — escape codes live only in
// boardWriter (the paint/content split keeps this byte-testable).
func TestRenderBoard(t *testing.T) {
	mkState := func(rows []passRow) boardState {
		st := boardState{
			st: progressState{
				nested: true,
				outerK: 1, outerTotal: 3, outerKind: sampler.SizeExact,
				configK: 14, configTotal: 36, configKind: sampler.SizeExact,
				foldK: 47, foldTotal: 108, foldKind: sampler.SizeExact,
				outerScores: []float64{0.798},
			},
			rows: rows,
		}
		for i := 0; i < 16; i++ {
			st.rate.add(at(i * 1000))
		}
		return st
	}
	rows := []passRow{
		{done: true, heldOut: 0.798},
		{configK: 8, foldK: 25, best: 0.834, hasBest: true},
		{}, // queued: no events yet
	}
	lines := renderBoard(mkState(rows), boardEnv{width: 100, now: at(21176), busy: 8, capacity: 8})
	frame := strings.Join(lines, "\n")
	for _, want := range []string{
		"outer folds 1/3", "configs scored 14/36", "inner-CV runs 47/108", "est 0.7980",
		"outer fold 0 ✓ held-out 0.7980",
		"outer fold 1 ▸ configs scored 8/12 · inner-CV runs 25/36 · best 0.8340",
		"outer fold 2 — queued",
		"~slots 8/8", "42.5 inner-CV runs/min", "~ETA",
	} {
		if !strings.Contains(frame, want) {
			t.Errorf("frame missing %q:\n%s", want, frame)
		}
	}
	if len(lines) != 5 { // aggregate + 3 fold rows + leaves
		t.Errorf("want 5 lines, got %d:\n%s", len(lines), frame)
	}
	if strings.Contains(frame, "\x1b") {
		t.Error("renderer must emit NO escape codes")
	}

	// Per-row denominators derive from the aggregate totals (36 configs / 3 outer = 12).
	// All-done: every row ✓, no ETA segment (nothing remaining).
	allDone := []passRow{{done: true, heldOut: 0.79}, {done: true, heldOut: 0.81}, {done: true, heldOut: 0.82}}
	st := mkState(allDone)
	st.st.outerK, st.st.foldK, st.st.configK = 3, 108, 36
	st.st.outerScores = []float64{0.79, 0.81, 0.82}
	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(21176), busy: 0, capacity: 8}), "\n")
	if strings.Contains(frame, "▸") || strings.Contains(frame, "ETA") {
		t.Errorf("all-done: no in-flight rows, no ETA:\n%s", frame)
	}

	// Flat (no rows): exactly 2 lines — the aggregate + slots/rate.
	flat := boardState{st: progressState{foldK: 3, foldTotal: 5, foldKind: sampler.SizeExact, flatScores: []float64{0.8}}}
	if got := renderBoard(flat, boardEnv{width: 100, now: at(0), busy: 2, capacity: 8}); len(got) != 2 {
		t.Errorf("flat board = aggregate + leaves, got %d lines: %v", len(got), got)
	}

	// Overflow: 14 folds → 12 rows + "… +2 more" + slots/rate + aggregate = 15 lines.
	many := make([]passRow, 14)
	st = mkState(many)
	st.st.outerTotal = 14
	if got := renderBoard(st, boardEnv{width: 100, now: at(21176), busy: 8, capacity: 8}); len(got) != 15 {
		t.Errorf("overflow: want 15 lines (1+12+1+1), got %d", len(got))
	} else if !strings.Contains(strings.Join(got, "\n"), "+2 more") {
		t.Errorf("overflow marker missing:\n%s", strings.Join(got, "\n"))
	}

	// Width clamp: every line ≤ width, truncated with …
	narrow := renderBoard(mkState(rows), boardEnv{width: 30, now: at(2824), busy: 8, capacity: 8})
	for _, l := range narrow {
		if n := len([]rune(l)); n > 30 {
			t.Errorf("line exceeds width 30 (%d runes): %q", n, l)
		}
	}
	if !strings.Contains(strings.Join(narrow, "\n"), "…") {
		t.Error("clamped lines should truncate with …")
	}

	// No gauge (capacity 0): slots segment absent, throughput still present.
	frame = strings.Join(renderBoard(mkState(rows), boardEnv{width: 100, now: at(21176)}), "\n")
	if strings.Contains(frame, "slots") {
		t.Errorf("no gauge → no slots segment:\n%s", frame)
	}
	if !strings.Contains(frame, "inner-CV runs/min") {
		t.Errorf("throughput must survive a missing gauge:\n%s", frame)
	}

	// Rate unavailable (fresh ring): "— inner-CV runs/min".
	st = mkState(rows)
	st.rate = movingRate{}
	frame = strings.Join(renderBoard(st, boardEnv{width: 100, now: at(0), busy: 1, capacity: 8}), "\n")
	if !strings.Contains(frame, "— inner-CV runs/min") {
		t.Errorf("unavailable rate renders —:\n%s", frame)
	}
}

func TestRenderBoardStartupObservationsAreFactual(t *testing.T) {
	now := at(5000)
	bs := boardState{st: progressState{
		nested: true, foldTotal: 10, foldKind: sampler.SizeExact,
		stepK: 37, lastStepAt: at(4000),
	}}
	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 8, capacity: 12}), "\n")
	for _, want := range []string{
		"starting", "~slots 8/12", "37 steps completed", "last step 1s ago", "no inner-CV run complete",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("startup frame missing %q:\n%s", want, frame)
		}
	}
	for _, forbidden := range []string{"not hung", "warming"} {
		if strings.Contains(frame, forbidden) {
			t.Fatalf("startup frame made diagnosis %q:\n%s", forbidden, frame)
		}
	}

	bs.st.foldK = 1
	bs.st.lastRunAt = now
	frame = strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 8, capacity: 12}), "\n")
	if strings.Contains(frame, "starting") || strings.Contains(frame, "no inner-CV run complete") {
		t.Fatalf("startup line must disappear after first eligible run:\n%s", frame)
	}
}

func TestRenderBoardFlatStartupAndConfidenceLabels(t *testing.T) {
	now := at(30000)
	bs := boardState{st: progressState{
		foldTotal: 20, foldKind: sampler.SizeExact,
		stepK: 4, lastStepAt: at(28000),
	}}
	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 3, capacity: 8}), "\n")
	for _, want := range []string{
		"CV runs 0/20", "starting", "~slots 3/8", "4 steps completed", "last step 2s ago", "no CV run complete",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("flat startup frame missing %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "inner-CV") || strings.Contains(frame, "warming") || strings.Contains(frame, "not hung") {
		t.Fatalf("flat startup frame contains nested or diagnostic wording:\n%s", frame)
	}

	bs.st.foldK = 1
	bs.st.lastRunAt = at(29000)
	frame = strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 3, capacity: 8}), "\n")
	for _, want := range []string{"CV runs 1/20", "last CV run 1s ago", "— CV runs/min"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("flat pre-confidence frame missing %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "starting") || strings.Contains(frame, "ETA") {
		t.Fatalf("flat pre-confidence frame must be post-startup without ETA:\n%s", frame)
	}
}

func TestRenderBoardMatureShowsLastRunAge(t *testing.T) {
	now := at(20000)
	var rate movingRate
	for i := 0; i < 16; i++ {
		rate.add(at(i * 1000))
	}
	bs := boardState{
		st: progressState{
			nested: true, foldK: 16, foldTotal: 32, foldKind: sampler.SizeExact,
			lastRunAt: at(15000),
		},
		rate: rate,
	}
	frame := strings.Join(renderBoard(bs, boardEnv{width: 120, now: now, busy: 4, capacity: 8}), "\n")
	for _, want := range []string{"45.0 inner-CV runs/min", "last inner-CV run 5s ago", "~ETA"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("mature frame missing %q:\n%s", want, frame)
		}
	}
}

// fmtETA is compact and human: seconds under a minute, m+s under an hour.
func TestFmtETA(t *testing.T) {
	cases := map[time.Duration]string{
		34 * time.Second:                            "34s",
		190 * time.Second:                           "3m10s",
		2*time.Hour + 5*time.Minute:                 "2h5m",
		2*time.Hour + 5*time.Minute + 9*time.Second: "2h5m",
	}
	for d, want := range cases {
		if got := fmtETA(d); got != want {
			t.Errorf("fmtETA(%v) = %q, want %q", d, got, want)
		}
	}
}

// steppingClock advances a fixed step per reading — every budgeted operation sees
// the flush budget elapsed, preserving the pre-#46 inline-flush semantics in tests
// that assert immediate rendering.
func steppingClock(step time.Duration) func() time.Time {
	var mu sync.Mutex // runOpts.now is called from concurrent ParExec goroutines
	t := at(0)
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		t = t.Add(step)
		return t
	}
}

// boardWriter pins the board to the bottom: passthrough writes scroll above the
// stored frame; erase sequences separate frames; close is idempotent and restores
// the cursor. Driven directly (no ticker) against a bytes.Buffer "terminal".
// (metis#46: a stepping clock keeps each write on the quiet inline-flush path.)
func TestBoardWriter_PinBottom(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond))


exec
/bin/zsh -lc "sed -n '130,230p' cmd/metis/caching_test.go && sed -n '180,330p' cmd/metis/run_test.go && sed -n '340,420p' cmd/metis/shapesweep_test.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
	if hitFor(out3, "train") {
		t.Errorf("train (knob changed) must MISS and re-run; got:\n%s", out3)
	}
}

func TestCacheActivityEmitsOnColdExecutionAndWarmHit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH (cache re-hash uses git hash-object)")
	}
	root := repoRoot(t)
	ws := t.TempDir()
	expPath := filepath.Join(ws, "exp.md")
	md := `---
type: experiment
id: cache-activity
seed: 5
status: active
steps:
  - id: prep
    uses: test/echo
    with: {k: 5}
---
`
	if err := os.WriteFile(expPath, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}

	var stepEvents []activityEvent
	run := func(runID string) string {
		var out strings.Builder
		_, err := runExperiment(runOpts{
			expPath:  expPath,
			runID:    runID,
			stepPath: []string{filepath.Join(root, "testdata", "steps")},
			now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
			git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
			cache:    true,
			out:      &out,
			activity: func(ev activityEvent) {
				if ev.Kind == activityStepSuccess {
					stepEvents = append(stepEvents, ev)
				}
			},
		})
		if err != nil {
			t.Fatalf("run %s: %v", runID, err)
		}
		return out.String()
	}

	out1 := run("activity-cold")
	if strings.Contains(out1, "cache hit") {
		t.Fatalf("cold run should not have cache hits; got:\n%s", out1)
	}
	out2 := run("activity-warm")
	if !hitFor(out2, "prep") {
		t.Fatalf("warm run should hit prep; got:\n%s", out2)
	}
	if len(stepEvents) != 2 {
		t.Fatalf("step events = %d; want one cold execution and one warm cache-hit event: %+v", len(stepEvents), stepEvents)
	}
	for i, ev := range stepEvents {
		if ev.Kind != activityStepSuccess || ev.StepID != "prep" {
			t.Fatalf("event %d = %+v; want prep step success", i, ev)
		}
	}
}

func hitFor(out, step string) bool {
	return strings.Contains(out, "step "+step+" (cache hit)")
}

// TestCache_ToyPipelineHitsOnRerun drives the REAL uv/Python toy pipeline twice with
// caching (uv-gated): the second run must HIT every step (the sensor's real reads.json
// → D re-hashes clean), materialize the parquet outputs from the CAS, and reproduce the
// same cv_score — proving the cache end-to-end with real artifacts, not just test/echo.
func TestCache_ToyPipelineHitsOnRerun(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH; skipping the real-pipeline cache e2e")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := repoRoot(t)
	ws := t.TempDir()
	expDir := filepath.Join(ws, "experiment")
	if err := os.MkdirAll(expDir, 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, filepath.Join(root, "testdata", "experiment", "toy-pipeline.md"),
		filepath.Join(expDir, "toy-pipeline.md"))
	copyDir(t, filepath.Join(root, "testdata", "dataset", "toy"), filepath.Join(ws, "dataset", "toy"))

	run := func(runID string) (string, float64) {
		var out strings.Builder
		r, err := runExperiment(runOpts{
			expPath:  filepath.Join(expDir, "toy-pipeline.md"),
			runID:    runID,
			stepPath: []string{filepath.Join(root, "steps")},
			now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
			git:      fakeGitProbe{name: "metis", sha: "sha", dirty: false},
}

func TestRunExperimentActivityRunSuccessPublishesAfterRequiredArtifacts(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-echo.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	var runEvents []activityEvent
	var publishErrs []error
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-activity",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind != activityRunSuccess {
				return
			}
			runEvents = append(runEvents, ev)
			for _, name := range []string{"run.json", "record.json"} {
				if _, statErr := os.Stat(filepath.Join(dir, "runs", ev.RunID, name)); statErr != nil {
					publishErrs = append(publishErrs, statErr)
				}
			}
		},
	})
	if err != nil {
		t.Fatalf("runExperiment: %v", err)
	}
	if len(publishErrs) > 0 {
		t.Fatalf("run-success event arrived before required artifacts existed: %v", publishErrs)
	}
	if len(runEvents) != 1 {
		t.Fatalf("run-success events = %d; want 1 (%+v)", len(runEvents), runEvents)
	}
	if runEvents[0].RunID != "run-activity" {
		t.Fatalf("run event = %+v; want runID run-activity", runEvents[0])
	}
	if runEvents[0].Role != runRoleNone {
		t.Fatalf("plain run event role = %q; want no role/ineligible", runEvents[0].Role)
	}
}

func TestRunExperimentActivityRunSuccessNotPublishedForFailedExecution(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-fail.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-fail.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	var runEvents []activityEvent
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-failed-activity",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind == activityRunSuccess {
				runEvents = append(runEvents, ev)
			}
		},
	})
	if err == nil {
		t.Fatal("runExperiment: want failing step error")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "run.json")); statErr != nil {
		t.Fatalf("failed run should still write run.json: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "runs", "run-failed-activity", "record.json")); statErr != nil {
		t.Fatalf("failed run should still write record.json: %v", statErr)
	}
	if len(runEvents) != 0 {
		t.Fatalf("failed execution emitted run-success events: %+v", runEvents)
	}
}

func TestRunExperimentActivityRunSuccessNotPublishedWhenRecordPersistenceFails(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "run-echo.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "run-echo.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "runs", "run-record-fails", "record.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	var runEvents []activityEvent
	_, err = runExperiment(runOpts{
		expPath:  expPath,
		runID:    "run-record-fails",
		stepPath: []string{filepath.Join(root, "testdata", "steps")},
		now:      func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC) },
		git:      fakeGitProbe{name: "metis", sha: "testsha", dirty: false},
		out:      io.Discard,
		activity: func(ev activityEvent) {
			if ev.Kind == activityRunSuccess {
				runEvents = append(runEvents, ev)
			}
		},
	})
	if err == nil {
		t.Fatal("runExperiment: want record persistence error")
	}
	if len(runEvents) != 0 {
		t.Fatalf("record persistence failure emitted run-success events: %+v", runEvents)
	}
}

// TestRunExperiment_RejectsInvalidAtRunTime is the execution-time enforcement
// test: a semantically-invalid experiment (a cycle — shape-valid, so CUE accepts
// it) is rejected by `metis run` BEFORE any step runs, closing the SHAPE-only gap
// M1 deferred. No ledger and no `## Runs` line are written for a rejected run.
func TestRunExperiment_RejectsInvalidAtRunTime(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "testdata", "experiment", "invalid-cycle.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	expPath := filepath.Join(dir, "invalid-cycle.md")
	if err := os.WriteFile(expPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = runExperiment(runOpts{
		expPath:  expPath,
	if strings.Contains(out.String(), "nested-CV") {
		t.Errorf("a 1-config shape must NOT run nested-CV; got:\n%s", out.String())
	}
	// It RECORDS its fold rows (measure) ...
	led := loadLedgerOrFatal(t, expPath)
	if len(led.Rows) == 0 {
		t.Error("the flat single-level CV path must record its fold rows to the ledger")
	}
	// ... and does NOT ship (shipping is `metis select --promote`), even though `ship:` is present.
	shipSteps, _ := filepath.Glob(filepath.Join(ws, "runs", "*", "submission"))
	if len(shipSteps) != 0 {
		t.Errorf("the flat path must NOT ship; got %d submission runs", len(shipSteps))
	}
}

func TestShapeSweepActivityRunRolesFromFlatAndNestedCallPaths(t *testing.T) {
	t.Run("flat CV runs are eligible flat roles", func(t *testing.T) {
		ws := t.TempDir()
		expPath := writeShapeFile(t, ws, foldShapeShipMD("[a]"))
		counts := map[runRole]int{}
		_, err := runExperiment(runOpts{
			expPath: expPath, now: fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
			exec: foldFakeExec{}, out: io.Discard,
			activity: func(ev activityEvent) {
				if ev.Kind == activityRunSuccess {
					counts[ev.Role]++
				}
			},
		})
		if err != nil {
			t.Fatalf("flat sweep: %v", err)
		}
		if counts[runRoleFlatCV] != 2 || len(counts) != 1 {
			t.Fatalf("flat roles = %v; want exactly 2 flat-CV runs", counts)
		}
	})

	t.Run("nested emits preamble inner and outer score roles", func(t *testing.T) {
		ws := t.TempDir()
		expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))
		counts := map[runRole]int{}
		_, err := runExperiment(runOpts{
			expPath: expPath, now: fixedNow(), git: fakeGitProbe{name: "metis", sha: "sha"},
			exec: foldFakeExec{}, out: io.Discard,
			activity: func(ev activityEvent) {
				if ev.Kind == activityRunSuccess {
					counts[ev.Role]++
				}
			},
		})
		if err != nil {
			t.Fatalf("nested sweep: %v", err)
		}
		if counts[runRoleNestedPreamble] != 1 {
			t.Fatalf("nested roles = %v; want 1 preamble run", counts)
		}
		if counts[runRoleNestedInnerCV] != 8 {
			t.Fatalf("nested roles = %v; want 8 inner-CV runs", counts)
		}
		if counts[runRoleOuterScore] != 2 {
			t.Fatalf("nested roles = %v; want 2 outer-score runs", counts)
		}
		if len(counts) != 3 {
			t.Fatalf("nested roles = %v; want no ineligible/unexpected run roles", counts)
		}
	})
}

// Fold-distinctness + cache under NESTED (metis#32): each (outer-fold, config, inner-fold) of
// `train` gets a DISTINCT cache entry (the _fold overlay makes Kpre fold-distinct), the
// config/fold-invariant data steps HIT, and a warm re-run HITs everything (0 inner execs).
func TestShapeSweep_CacheFoldDistinctAndReRunHits(t *testing.T) {
	ws := t.TempDir()
	expPath := writeShapeFile(t, ws, foldShapeMD("[a, b]"))

	var cold []string
	if err := runFoldSweep(t, expPath, true, &cold, nil, nil); err != nil {
		t.Fatalf("cold sweep: %v", err)
	}
	// train runs per distinct (outer-fold, config, inner-fold): the sealed inner sweeps
	// (2 outer × 2 configs × 2 inner = 8) + the per-family outer-fold held-out scorings (2) = 10.

codex
The committed tests cover the main seams, including cache hits emitting step activity and failed executions not emitting run success. I’m going to run the focused package tests if the read-only sandbox allows Go’s temporary/cache writes; if it fails for sandbox reasons I’ll report that instead of treating it as code failure.
exec
/bin/zsh -lc "go test ./cmd/metis -run 'Activity|CacheActivity|RunExperimentActivity|ShapeSweepActivity|Progress|Board|RunControlActivity' -count=1" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 1433ms:
ok	github.com/xianxu/metis/cmd/metis	0.965s

codex
The focused activity/progress/board/run-control tests pass. I’m doing a quick static sweep for stale live board/output terminology and then a broader package test pass.
exec
/bin/zsh -lc "rg -n 'leaves |folds/min|ETA |outer [0-9]+/[0-9]+|fold [0-9]+ — queued|fold [0-9]+ ▸|folds [0-9]+/[0-9]+' cmd/metis atlas workshop/issues/000049-board-readability-labels-coldphase-eta.md workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:1:# Board Readability: Typed Activity, Cold-Phase Truth, and Stable ETA — Implementation Plan
workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:8:and withhold rate/ETA until time-based evidence is stable without weakening stall visibility.
workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:112:   non-increasing, and ETA is non-decreasing. Then add completions and prove gradual 64-event-window
workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:184:2. Search the RUNBOOK for stale `fold`, `leaves`, and `folds/min` board terminology; inspect each hit
workshop/issues/000049-board-readability-labels-coldphase-eta.md:20:1. **Label semantics unclear:** `outer 0/10 · configs 0/720 · folds 0/7200` — operator asked
workshop/issues/000049-board-readability-labels-coldphase-eta.md:30:3. **`leaves 2/12` jumps around:** the gauge samples instantaneous `len(leafSem)` at flush
workshop/issues/000049-board-readability-labels-coldphase-eta.md:33:4. **ETA changes wildly:** the 64-completion moving window over sparse, phase-heterogeneous
workshop/issues/000049-board-readability-labels-coldphase-eta.md:41:1. Rename/annotate segments (e.g. `inner-folds 421/7200` or a one-line legend on the first
workshop/issues/000049-board-readability-labels-coldphase-eta.md:44:2. A PHASE indicator when zero folds have completed but leaves are active (e.g.
workshop/issues/000049-board-readability-labels-coldphase-eta.md:48:3. Smooth the leaves gauge (peak-or-mean over the flush window, or EWMA) — the sink already
workshop/issues/000049-board-readability-labels-coldphase-eta.md:50:4. ETA damping: suppress until n≥K completions AND the window spans ≥T seconds; consider
workshop/issues/000049-board-readability-labels-coldphase-eta.md:66:  with per-pass denominators. Startup, rate, and ETA wording uses the same mode-specific noun.
workshop/issues/000049-board-readability-labels-coldphase-eta.md:95:#### Time-driven smoothing and ETA confidence
workshop/issues/000049-board-readability-labels-coldphase-eta.md:100:- Keep the latest 64 eligible run-completion timestamps, but withhold rate and ETA until at least 16
workshop/issues/000049-board-readability-labels-coldphase-eta.md:104:  numeric rate is non-increasing and ETA is non-decreasing, although display rounding need not change
workshop/issues/000049-board-readability-labels-coldphase-eta.md:108:  five seconds while rate/ETA move monotonically; it makes no diagnosis about why work is quiet
workshop/issues/000049-board-readability-labels-coldphase-eta.md:111:- Render the mature estimate as `~ETA …`; nested mode labels its rate `inner-CV runs/min` and scopes
workshop/issues/000049-board-readability-labels-coldphase-eta.md:112:  ETA to remaining inner-CV runs, while flat mode uses `CV runs/min` and remaining CV runs. Neither
workshop/issues/000049-board-readability-labels-coldphase-eta.md:114:  mode-specific `— … runs/min` and no ETA rather than a volatile fiction.
workshop/issues/000049-board-readability-labels-coldphase-eta.md:125:- Leaves + rate/ETA move smoothly (unit tests over scripted event traces pin the damping); after a
workshop/issues/000049-board-readability-labels-coldphase-eta.md:127:  five seconds while the numeric rate/ETA move monotonically (regression test).
workshop/issues/000049-board-readability-labels-coldphase-eta.md:131:  counter nor rate/ETA (regression test).
workshop/issues/000049-board-readability-labels-coldphase-eta.md:177:  DEC 2026 sync output), metis#48 (default BLAS pins — the 3h-ETA root cause when unpinned),
workshop/issues/000049-board-readability-labels-coldphase-eta.md:178:  metis#43 (the phase-wave scheduler, pre-existing). The wild-ETA observation is partly a
atlas/index.md:89:  (`metis: progress outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`;
atlas/index.md:96:  `nested preamble`, `outer score`) keep rate/ETA eligibility at the concrete call sites, and `runControl`
atlas/index.md:105:  `CV runs/min`), and mature `~ETA`. Rate/ETA are withheld until 16 eligible completions span at least
atlas/index.md:138:  (= feature count), hist_gbm **total** leaves summed across boosted trees (metis#21 — sum, not mean:
cmd/metis/exec.go:60:		// driver:single path leaves the var unset (unconfined).
cmd/metis/progress.go:58:// `outer folds 1/3 · configs scored 84/216 · inner-CV runs 421/1080 · est 0.8283 ± 0.0140`
cmd/metis/progress.go:264:	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
cmd/metis/progress.go:416:// the rate decay + ETA move even between events. Sink-first (sp.mu → bw.mu).
cmd/metis/blaspins.go:7:// leaf semaphore), not to each leaf's BLAS — NumCPU leaves × multi-threaded BLAS
cmd/metis/capture_e2e_test.go:19:// call site leaves the direct-helper tests green but must fail THIS one (the Done-when is
cmd/metis/parallel_test.go:517:	for _, forbidden := range []string{"metis: progress", "folds/min", "ETA", "score ", "estimate", "mean "} {
cmd/metis/forkexec_test.go:200:// TestForkServerPerf_LooseBound (metis#44 acceptance): N leaves that each import pandas —
cmd/metis/forkexec_test.go:250:	// (server start + preload amortize over ~5k leaves in a real sweep, not over n=4).
cmd/metis/run.go:99:	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
cmd/metis/nestedcv_e2e_test.go:69:	if !strings.Contains(finalProg, "outer folds 2/2") || !strings.Contains(finalProg, "est 0.") {
cmd/metis/identity_e2e_test.go:75:// a real git blob, which is exactly what a traced run leaves behind.
cmd/metis/identity_e2e_test.go:92:	// record.json (no fingerprint yet) — the state a run leaves for capture to backfill.
cmd/metis/progress_test.go:31:			[]string{"outer folds 0/3", "configs scored 84/216", "inner-CV runs 421/1080", "est —"}, []string{"±"}},
cmd/metis/progress_test.go:33:			[]string{"outer folds 1/3", "est 0.8200"}, []string{"±"}},
cmd/metis/progress_test.go:35:			[]string{"outer folds 2/3", "est 0.8200 ± 0.0200"}, nil},
cmd/metis/progress_test.go:39:			[]string{"CV runs 3/5", "score 0.8400"}, []string{"configs", "outer", "folds 3/5"}},
cmd/metis/progress_test.go:111:	for _, w := range []string{"outer folds 1/2", "inner-CV runs 10/20", "est 0.8300"} {
cmd/metis/board.go:33:// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
cmd/metis/board.go:109:			segs = append(segs, "~ETA "+fmtETA(eta))
cmd/metis/board.go:127:// fmtETA renders a duration compactly: 34s · 3m10s · 2h5m.
cmd/metis/board_test.go:41:		"outer folds 1/3", "configs scored 14/36", "inner-CV runs 47/108", "est 0.7980",
cmd/metis/board_test.go:43:		"outer fold 1 ▸ configs scored 8/12 · inner-CV runs 25/36 · best 0.8340",
cmd/metis/board_test.go:44:		"outer fold 2 — queued",
cmd/metis/board_test.go:59:	// All-done: every row ✓, no ETA segment (nothing remaining).
cmd/metis/board_test.go:194:// fmtETA is compact and human: seconds under a minute, m+s under an hour.
cmd/metis/board_test.go:231:	bw.paint([]string{"AGG line", "fold 0 ▸"})
cmd/metis/board_test.go:236:	if !strings.Contains(first, "AGG line\nfold 0 ▸\n") {
cmd/metis/board_test.go:248:	wantOrder := []string{"\x1b[2A\x1b[J", "⚡ step train\n", "AGG line\nfold 0 ▸\n"}
cmd/metis/board_test.go:308:	bw.paint([]string{"folds 2/8", "31.2 folds/min · ETA 12s"})
cmd/metis/board_test.go:317:	for _, stale := range []string{"folds 2/8", "folds/min", "ETA"} {
cmd/metis/board_test.go:361:	if !strings.Contains(s, "outer folds 2/2") || !strings.Contains(s, "outer fold 0 ✓") || !strings.Contains(s, "outer fold 1 ✓") {
cmd/metis/board_test.go:372:	if finalFrame := strings.LastIndex(s, "outer folds 2/2"); warnIdx > finalFrame {
cmd/metis/board_test.go:431:	for _, want := range []string{"outer folds 0/2", "outer fold 0 — queued", "no inner-CV run"} {
cmd/metis/board_test.go:451:		"outer folds 0/2", "outer fold 0 — queued", "configs ", "inner-CV runs ", "inner-CV runs/min", "ETA", "score ", "estimate", "mean ",
cmd/metis/board_test.go:525:	bw.paint([]string{"AGG", "fold 0 ▸"})
cmd/metis/board_test.go:610:	if !strings.Contains(term2.String(), "outer folds 0/1") {
cmd/metis/board_test.go:621:	bw.paint([]string{"AGG", "fold 0 ▸"})
cmd/metis/record_test.go:65:	// Upstream/D/Deps are metis#2-populated slots — #3 leaves them empty.
cmd/metis/sweep.go:35:// fold_score (metis#19): the fitted model's realized complexity (rf mean leaves / logreg
cmd/metis/sweep.go:270:	// and a 500ms ticker keeps the rate decay + ETA live between events (sink-first:
cmd/metis/run_test.go:343:	// A rejected run writes no runs/ dir and leaves the config .md untouched (#13 — the
cmd/metis/forkexec.go:335:			p.noticeOnce("root:"+spec.root, fmt.Sprintf("server for %s died mid-flight (%v) — erroring the step; later leaves use legacy", spec.root, err))

exec
/bin/zsh -lc 'git diff --check 73cf6993cb9cbb87a11696f7adb378b071b39c81..HEAD' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 exited 2 in 0ms:
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:498: space before tab in indent.
+	b.flushLocked(b.now())
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:500: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:522: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:524: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:556: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:558: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:642: space before tab in indent.
+	var term strings.Builder
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:643: space before tab in indent.
+	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond))
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:644: space before tab in indent.
+	bw.paint([]string{"BOARD"})
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:647: space before tab in indent.
+	pool.noticeOnce("k", "server died; falling back to legacy exec")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:648: space before tab in indent.
+	s := term.String()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:649: space before tab in indent.
+	notice := strings.Index(s, "metis: forkserver: server died")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:660: space before tab in indent.
+	closureByRepo := sweepClosure(o.expPath, man)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:661: space before tab in indent.
+	primary := cacheProjectRoot(o.stepPath, filepath.Dir(o.expPath))
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:662: space before tab in indent.
+	commit, d, status := captureRunCode(closureByRepo, primary, o.expPath, "refs/metis/sweeps/"+man.ShapeRunID)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:664: space before tab in indent.
+	for _, p := range man.Points {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:665: space before tab in indent.
+		pfp, pdirty, err := backfillCodeManifest(o.expPath, p.RunID, d, commit, status)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:666: space before tab in indent.
+		if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:669: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:670: space before tab in indent.
+		if fp == "" && pfp != "" {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:671: space before tab in indent.
+			fp, dirty = pfp, pdirty
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:672: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:673: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:674: space before tab in indent.
+	printFingerprintLine(o.out, fp, commit, dirty)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:678: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:685: space before tab in indent.
+	man := sweepManifest{ShapeRunID: "srun-e2e", Points: []pointRun{{RunID: "pt-0", Status: "failed"}, {RunID: runID, Status: "ok"}}}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:686: space before tab in indent.
+	var out bytes.Buffer
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:687: space before tab in indent.
+	o := runOpts{expPath: expPath, stepPath: []string{filepath.Join(root, "steps")}, out: &out}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:690: space before tab in indent.
+		t.Fatalf("captureSweepCode: %v", err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:691: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:692: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:698: space before tab in indent.
+	pool *serverPool // metis#44: when non-nil, convention-conforming wrappers route through the
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:699: space before tab in indent.
+	//                  warm fork-server (one per project root) instead of a fresh uv/python spawn;
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:700: space before tab in indent.
+	//                  non-conforming wrappers + broken servers fall back to the legacy path below.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:705: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:708: space before tab in indent.
+			base = append(base, kv)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:709: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:710: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:714: space before tab in indent.
+	for _, k := range sortedKeys(env) {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:715: space before tab in indent.
+		base = append(base, k+"="+env[k])
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:716: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:722: space before tab in indent.
+		t.Errorf("artifacts = %v; want %v", arts, want)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:723: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:756: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:766: space before tab in indent.
+	cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.forkserver")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:767: space before tab in indent.
+	cmd.Dir = root
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:769: space before tab in indent.
+	// Own process GROUP: `uv run` spawns python as a child (no exec), and the server forks
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:770: space before tab in indent.
+	// step children — group-kill is the only way to reap the whole tree on a hung shutdown
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:771: space before tab in indent.
+	// (and a test's mid-flight kill). Normal shutdown stays graceful (stdin EOF → drain);
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:773: space before tab in indent.
+	warned  map[string]bool
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:774: space before tab in indent.
+	out     io.Writer
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:775: space before tab in indent.
+	outMu   sync.Mutex
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:778: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:781: space before tab in indent.
+	return &serverPool{
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:782: space before tab in indent.
+		servers: map[string]*forkServer{},
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:783: space before tab in indent.
+		broken:  map[string]bool{},
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:784: space before tab in indent.
+		warned:  map[string]bool{},
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:785: space before tab in indent.
+		out:     out,
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:787: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:789: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:791: space before tab in indent.
+	s := p.servers[spec.root]
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:792: space before tab in indent.
+	if s == nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:793: space before tab in indent.
+		var err error
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:796: space before tab in indent.
+		if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:797: space before tab in indent.
+			p.broken[spec.root] = true
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:798: space before tab in indent.
+			p.mu.Unlock()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:804: space before tab in indent.
+	t.Setenv("PYTHONPATH", mods)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:805: space before tab in indent.
+	t.Setenv("METIS_FORKSERVER_PRELOAD", "") // fast start; preload is a python-side concern
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:806: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:809: space before tab in indent.
+	defer pool.shutdown()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:810: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:811: space before tab in indent.
+	stepDir := filepath.Join(t.TempDir(), "s1")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:813: space before tab in indent.
+		t.Skip("uv not on PATH")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:814: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:815: space before tab in indent.
+	var out strings.Builder
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:818: space before tab in indent.
+	defer pool.shutdown()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:819: space before tab in indent.
+	bogus := t.TempDir() // no pyproject/venv — uv run will fail (or hang-free error)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:820: space before tab in indent.
+	for i := 0; i < 3; i++ {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:822: space before tab in indent.
+	ws := t.TempDir()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:823: space before tab in indent.
+	var out strings.Builder
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:824: space before tab in indent.
+	e := execStep{stepPath: []string{filepath.Join(root, "steps")}, expDir: ws,
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:827: space before tab in indent.
+	defer e.pool.shutdown()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:828: space before tab in indent.
+	runDir := filepath.Join(ws, "runs", "r1")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:829: space before tab in indent.
+	res, err := e.Execute(experiment.Step{ID: "sh", Uses: "test/shellstep"}, runDir)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:831: space before tab in indent.
+	legacy := time.Since(start)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:832: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:833: space before tab in indent.
+	start = time.Now()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:836: space before tab in indent.
+	defer pool.shutdown()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:837: space before tab in indent.
+	forkOne := func() {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:838: space before tab in indent.
+		dir := t.TempDir()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:840: space before tab in indent.
+	t.Setenv("PYTHONPATH", mods)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:841: space before tab in indent.
+	t.Setenv("METIS_FORKSERVER_PRELOAD", "")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:842: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:845: space before tab in indent.
+	defer pool.shutdown()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:846: space before tab in indent.
+	type result struct {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:847: space before tab in indent.
+		ok  bool
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:849: space before tab in indent.
+		t.Fatalf("mid-flight death must be a STEP ERROR (dispatched-and-lost), got ok=%v err=nil", res.ok)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:850: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:900: space before tab in indent.
+		})
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:901: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:902: space before tab in indent.
+	capture := func() record.RunRecord {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:905: space before tab in indent.
+			t.Fatalf("captureSweepCode: %v", err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:906: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:907: space before tab in indent.
+		rb, _ := os.ReadFile(filepath.Join(root, "runs", pa, "record.json"))
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:913: space before tab in indent.
+	sampleN := fs.Int("sample", 0, "metis#42: run m of the k outer folds (sparse fold sampling; 0/omitted = all k). k stays the estimand (each fold trains on (k-1)/k of the rows); m only trades precision for cost — use to probe a higher k (e.g. k=10, --sample 3) without the full k× bill. The SE over m<k folds is noisy (m-1 df): probe with it, don't re-select what ships on it. Errors on m>k, on a single-config (flat) run, and combined with --fast.")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:914: space before tab in indent.
+	forkserver := fs.Bool("forkserver", true, "metis#44: run convention-conforming step wrappers through a warm per-root fork-server (pre-imported pandas/sklearn; ~1s spawn tax removed per leaf). --forkserver=false = legacy per-step uv/python spawn (the escape hatch); non-conforming wrappers and failed servers fall back to legacy automatically (loud, once).")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:915: space before tab in indent.
+	noTUI := fs.Bool("no-tui", false, "metis#38: force the plain progress lines even on a TTY (the live board is default for a sweep when stdout is a terminal; piped/redirected output always gets plain lines)")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:918: space before tab in indent.
+	if err := fs.Parse(args); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:919: space before tab in indent.
+		return err
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:920: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:926: space before tab in indent.
+	if !strings.Contains(finalProg, "outer 2/2") || !strings.Contains(finalProg, "est 0.") {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:927: space before tab in indent.
+		t.Errorf("the final progress line must carry the completed outer count + a numeric est; got: %q", finalProg)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:928: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:940: space before tab in indent.
+	// One held-out score per (outer fold × family): outerK = sweeper.cv.k = 2, and a,b are one
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:941: space before tab in indent.
+	// family → 2 held-out lines.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:942: space before tab in indent.
+	if n := strings.Count(s, "→ held-out "); n != 2 {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:948: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:950: space before tab in indent.
+	"bytes"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:953: space before tab in indent.
+	"fmt"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:954: space before tab in indent.
+	"io"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:955: space before tab in indent.
+	"os"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:956: space before tab in indent.
+	"path/filepath"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:959: space before tab in indent.
+	"sync"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:961: space before tab in indent.
+	"testing"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:962: space before tab in indent.
+	"time"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:963: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:964: space before tab in indent.
+	"github.com/xianxu/metis/pkg/experiment"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:967: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:972: space before tab in indent.
+	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds = 6 per-fold rows
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:975: space before tab in indent.
+		ws := t.TempDir()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:976: space before tab in indent.
+		expPath := writeShapeFile(t, ws, body)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:977: space before tab in indent.
+		if _, err := runExperiment(runOpts{
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:979: space before tab in indent.
+		if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:980: space before tab in indent.
+			t.Fatalf("read manifest: %v", err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:981: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:984: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:989: space before tab in indent.
+	if !bytes.Equal(sL, pL) {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:990: space before tab in indent.
+		t.Errorf("ledger bytes differ serial vs parallel:\n--serial--\n%s\n--parallel--\n%s", sL, pL)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:991: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:992: space before tab in indent.
+	if !bytes.Equal(sM, pM) {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:993: space before tab in indent.
+		t.Errorf("manifest bytes differ serial vs parallel:\n--serial--\n%s\n--parallel--\n%s", sM, pM)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:994: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1192: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1200: space before tab in indent.
+	ws := t.TempDir()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1201: space before tab in indent.
+	// 3 configs → nested (outer folds = sweeper.cv.k = 2) × 2 inner folds → deep nesting, ~many leaf calls.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1202: space before tab in indent.
+	expPath := writeShapeFile(t, ws, foldShapeCVMD("[a, b, c]"))
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1203: space before tab in indent.
+	sem := make(chan struct{}, cap)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1204: space before tab in indent.
+	var mu sync.Mutex
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1205: space before tab in indent.
+	var cur, peak int
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1223: space before tab in indent.
+	pe := peakExec{sem: sem, mu: &mu, cur: &cur, peak: &peak, in: foldFakeExec{}}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1250: space before tab in indent.
+	if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1251: space before tab in indent.
+		t.Fatalf("driver:cv run must complete (no deadlock), got: %v", err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1252: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1253: space before tab in indent.
+	mu.Lock()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1254: space before tab in indent.
+	got := peak
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1259: space before tab in indent.
+	mu.Unlock()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1260: space before tab in indent.
+	if got > cap {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1261: space before tab in indent.
+		t.Fatalf("peak concurrency %d exceeded the global cap %d — the leaf budget leaked across nesting", got, cap)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1263: space before tab in indent.
+	if got < 2 {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1264: space before tab in indent.
+		t.Fatalf("peak concurrency %d — the fan-out never overlapped, so the test can't prove the cap actually holds", got)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1265: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1282: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1285: space before tab in indent.
+	d  time.Duration
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1287: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1435: space before tab in indent.
+	res, err := s.in.Execute(step, runDir)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1436: space before tab in indent.
+	time.Sleep(s.d)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1442: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1453: space before tab in indent.
+	bw.forceFlush() // metis#46: the tick is what re-pins the board after a burst window
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1455: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1475: space before tab in indent.
+	"io"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1476: space before tab in indent.
+	"os"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1477: space before tab in indent.
+	"path/filepath"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1479: space before tab in indent.
+	"sync"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1480: space before tab in indent.
+	"time"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1481: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1483: space before tab in indent.
+	readRoot    string        // metis#23: when set, the production execStep confines base-dataset reads to this root
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1484: space before tab in indent.
+	maxParallel int           // metis#31: >1 ⇒ ParExec batches + a leaf semaphore; sizes leafSem
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1485: space before tab in indent.
+	leafSem     chan struct{} // metis#31: the shared global subprocess budget (nil = serial/cache-only)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1488: space before tab in indent.
+	forkserver  bool          // metis#44: warm fork-server leaf executor (cmdRun default true;
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1489: space before tab in indent.
+	//                           zero-value false keeps direct runOpts callers/tests on legacy exec)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1490: space before tab in indent.
+	forkPool *serverPool // metis#44: the per-root warm-server pool, created once per runExperiment
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1491: space before tab in indent.
+	//                      when forkserver is set; threaded through nested runOpts copies.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1492: space before tab in indent.
+	tui bool // metis#38: stdout is a TTY and --no-tui wasn't passed — a SWEEP pins the live board
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1493: space before tab in indent.
+	//          (a plain experiment ignores it; non-TTY/piped runs stay on the #30 plain lines)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1505: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1508: space before tab in indent.
+		out = &syncWriter{w: out}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1509: space before tab in indent.
+		o.out = out
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1510: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1524: space before tab in indent.
+	// metis#44: one warm fork-server pool per top-level run, shut down (EOF-drain) when the
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1525: space before tab in indent.
+	// run ends. Only the production executor uses it (an injected test exec bypasses execStep).
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1526: space before tab in indent.
+	// Constructed AFTER the writer wrap — its fallback notices must route through the board.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1527: space before tab in indent.
+	if o.forkserver && o.exec == nil && o.forkPool == nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1530: space before tab in indent.
+		defer o.forkPool.shutdown()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1531: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1532: space before tab in indent.
+	if exp.Type == "experiment-shape" {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1534: space before tab in indent.
+		if err := experiment.ValidateShape(sh); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1535: space before tab in indent.
+			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1536: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1540: space before tab in indent.
+		return experiment.Run{}, runShapeSweep(o, sh, now, out)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1541: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1542: space before tab in indent.
+	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1556: space before tab in indent.
+	baseDir := filepath.Dir(o.expPath)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1557: space before tab in indent.
+	// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1558: space before tab in indent.
+	// the child's env, and the child's cwd IS the step dir — a relative path would
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1560: space before tab in indent.
+		return experiment.Run{}, err
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1561: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1562: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1565: space before tab in indent.
+	if o.exec != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1566: space before tab in indent.
+		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:1567: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2197: space before tab in indent.
+	promote := fs.Bool("promote", false, "materialize the selected config(s): reconstruct from the ledger + run on ALL data → runs/best-{family}-{hash}/submission.csv; prints the run id(s)")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2198: space before tab in indent.
+	fingerprint := fs.String("fingerprint", "", "restrict to one code-fingerprint (metis#27)")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2199: space before tab in indent.
+	point := fs.String("point", "", "metis#41: publish an OPERATOR-CHOSEN config by ledger row — a point_addr (git-style prefix ok); ships as point-{family}-{hash}. Mutually exclusive with --best/--best-per-model-class")
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2201: space before tab in indent.
+	shapePath, flags, err := hoistShapePath(args)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2202: space before tab in indent.
+	if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2203: space before tab in indent.
+		return fmt.Errorf("select: %w (usage: metis select <shape.md> [--best | --best-per-model-class | --point ADDR] [--promote] [--fingerprint HASH])", err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2205: space before tab in indent.
+	if err := fs.Parse(flags); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2206: space before tab in indent.
+		return err
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2207: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2213: space before tab in indent.
+	if *point == "" && !*best && !*perClass {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2214: space before tab in indent.
+		*best = true // default view = the single ship recommendation
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2215: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2217: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2218: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2219: space before tab in indent.
+	sort.Slice(picks, func(i, j int) bool { return picks[i].family < picks[j].family })
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2226: space before tab in indent.
+	printSelect(o.out, sh, est, picks, o.perClass)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2227: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2228: space before tab in indent.
+	if o.promote {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2230: space before tab in indent.
+	est    sampler.MeanSE
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2231: space before tab in indent.
+	hasEst bool
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2232: space before tab in indent.
+	caveat string
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2235: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2238: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2239: space before tab in indent.
+	fmt.Fprintf(out, "  %s:\n", head)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2240: space before tab in indent.
+	for _, p := range picks {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2247: space before tab in indent.
+		if p.caveat != "" {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2248: space before tab in indent.
+			fmt.Fprintf(out, "      caveat: %s\n", p.caveat)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2249: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2250: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2252: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2265: space before tab in indent.
+	if f == "" {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2266: space before tab in indent.
+		return "(one family)"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2268: space before tab in indent.
+			return fmt.Errorf("select --promote %s: %w", famLabel(p.family), err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2269: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2270: space before tab in indent.
+		runID := "best-" + familyTag(p.family) + "-" + short(addr)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2273: space before tab in indent.
+		ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2274: space before tab in indent.
+		if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2275: space before tab in indent.
+			return fmt.Errorf("select --promote %s (%s): %w", famLabel(p.family), runID, err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2277: space before tab in indent.
+		return fmt.Errorf("select --point --promote: %w", err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2278: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2279: space before tab in indent.
+	runID := "point-" + familyTag(fam) + "-" + short(addr)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2282: space before tab in indent.
+	ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2283: space before tab in indent.
+	if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2284: space before tab in indent.
+		return fmt.Errorf("select --point --promote (%s): %w", runID, err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2291: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2295: space before tab in indent.
+	"os"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2296: space before tab in indent.
+	"path/filepath"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2297: space before tab in indent.
+	"strings"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2299: space before tab in indent.
+		t.Errorf("guard must inline the per-cohort summary, got: %v", err)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2300: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2362: space before tab in indent.
+		if !strings.Contains(final, "folds 2/2") || !strings.Contains(final, "score 0.") {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2363: space before tab in indent.
+			t.Errorf("the flat final progress line must carry folds k/k + score; got: %q", final)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2364: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2369: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2370: space before tab in indent.
+	if strings.Contains(out.String(), "nested-CV") {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2371: space before tab in indent.
+		t.Errorf("a 1-config shape must NOT run nested-CV; got:\n%s", out.String())
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2373: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2374: space before tab in indent.
+	return led
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2399: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2406: space before tab in indent.
+	o             runOpts
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2407: space before tab in indent.
+	sh            experiment.Shape
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2409: space before tab in indent.
+	parallel      bool           // metis#31: >1 max-parallel ⇒ the sweeper/resample/driver batches run via ParExec
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2410: space before tab in indent.
+	manMu         sync.Mutex     // metis#32: guards man.Points — concurrent outer folds (ParExec) each record rows
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2411: space before tab in indent.
+	prog          *sweepProgress // metis#30: the live-progress sink (nil = silent)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2414: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2440: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2443: space before tab in indent.
+	hooks    passHooks            // metis#30: this pass's progress hooks, closure-bound to its outer fold
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2444: space before tab in indent.
+	// metis#31: under ParExec the sweeper fans out over configs and each config's
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2445: space before tab in indent.
+	// resample fans out over folds — all appending to this ONE pass. `mu` guards the
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2448: space before tab in indent.
+	// sampler's pure Tell/Done, not here.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2449: space before tab in indent.
+	mu      sync.Mutex
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2450: space before tab in indent.
+	configs []configScore
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2451: space before tab in indent.
+	points  []pointRun
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2454: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2466: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2475: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2489: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2501: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2504: space before tab in indent.
+		func(c shape.Point) sampler.MeanSE {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2505: space before tab in indent.
+			ms := sampler.Run(ctx, sampler.FixedKFolds{K: pass.splitK},
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2506: space before tab in indent.
+				func(f sampler.FoldPoint) sampler.FoldOutcome { return pass.runPipelineFold(c, f) },
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2511: space before tab in indent.
+			pass.addConfigScore(configScore{point: c, meanSE: ms})
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2512: space before tab in indent.
+			return ms
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2513: space before tab in indent.
+		},
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2519: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2528: space before tab in indent.
+	// probeRepo's HEAD sha still drives the mid-sweep code-freeze guard (codeID) — NOT the
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2529: space before tab in indent.
+	// identity (metis#27 dropped repo_shas). The shape's blob-hash content-addresses the intent.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2530: space before tab in indent.
+	_, sha, _ := probeRepo(o.git, filepath.Dir(o.expPath))
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2532: space before tab in indent.
+		return err
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2533: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2534: space before tab in indent.
+	ss := &shapeSweep{
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2537: space before tab in indent.
+		partRef:  partitionRef(sh),
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2538: space before tab in indent.
+		man:      sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweeper.Sampler, Seed: sh.Seed},
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2539: space before tab in indent.
+		parallel: o.maxParallel > 1, // metis#31: fan out the sweeper/resample/driver batches
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2541: space before tab in indent.
+		ss.prog.bw = o.board
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2542: space before tab in indent.
+		ss.prog.width = boardWidth()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2543: space before tab in indent.
+		ss.prog.gauge = o.leafGauge
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2551: space before tab in indent.
+		tickDone := make(chan struct{})
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2553: space before tab in indent.
+		go func() {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2555: space before tab in indent.
+			for {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2556: space before tab in indent.
+				select {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2567: space before tab in indent.
+				case <-tickDone:
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2568: space before tab in indent.
+					return
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2569: space before tab in indent.
+				}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2570: space before tab in indent.
+			}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2571: space before tab in indent.
+		}()
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2583: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2584: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2585: space before tab in indent.
+	// metis#32: >1 config → nested CV (records inner + per-family outer rows; the honest measure).
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2587: space before tab in indent.
+	res := sampler.Run(ctx, sampler.SingleDriver{}, func(sampler.SinglePoint) sampler.SweepResult {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2588: space before tab in indent.
+		return ss.runSweeper(ctx, configPts, pass)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2589: space before tab in indent.
+	}, sampler.ExecFor[sampler.SinglePoint, sampler.SweepResult](ss.parallel), nil)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2595: space before tab in indent.
+	// metis#31: sort the fan-out's completion-order bookkeeping to a stable content key
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2596: space before tab in indent.
+	// BEFORE persisting, so manifest.json + the ledger are byte-deterministic across
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2597: space before tab in indent.
+	// serial/parallel runs (the winner/estimate are already deterministic; this makes
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2599: space before tab in indent.
+	sortPointRuns(pass.points)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2600: space before tab in indent.
+	ss.man.Points = pass.points
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2601: space before tab in indent.
+	ss.configs = pass.configs
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2606: space before tab in indent.
+	if err := writeManifest(o.expPath, ss.man); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2609: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2610: space before tab in indent.
+	// Capture the sweep's code closure to a git side ref (metis#8/#14) — BEST-EFFORT: the
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2611: space before tab in indent.
+	// records + manifest are already valid, so a capture hiccup warns, never aborts.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2619: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2620: space before tab in indent.
+	// Persist the raw per-fold rows to the shape's append-only ledger sidecar (metis#8/#18):
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2621: space before tab in indent.
+	// AggregateView reduces them read-time to per-config (mean, SE) — so metis#19's 1-SE
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2622: space before tab in indent.
+	// select re-reduces the same rows without a re-run.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2623: space before tab in indent.
+	if err := writeSweepLedger(o.expPath, ss.man); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2626: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2627: space before tab in indent.
+	// Guard (metis#19): a parsimony rule (one-std-err/pct-loss) needs a measured complexity
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2628: space before tab in indent.
+	// for every swept family — else the parsimony axis is silently dropped and the winner is
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2630: space before tab in indent.
+	// the ship/report is gated. Checked here (post-fold) because HasComplexity is only known
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2631: space before tab in indent.
+	// after the folds run.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2632: space before tab in indent.
+	if err := sampler.GuardComplexity(sh.Sweeper.Objective.Select, configStatsOf(ss.configs)); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2635: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2644: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2647: space before tab in indent.
+	// fewer of them (runFolds ≤ k). Deterministic run id → the analysis_i refs are locatable.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2648: space before tab in indent.
+	analysisRefs, err := ss.materializeOuterAnalysis(k, stratify)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2649: space before tab in indent.
+	if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2655: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2656: space before tab in indent.
+	outerPart := sampler.PartitionRef(fmt.Sprintf("outer-cv-k%d-strat%t-seed%d", k, stratify, ss.sh.Seed))
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2657: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2677: space before tab in indent.
+	est := sampler.Run(ctx, sampler.CVDriver{K: runFolds, Stratify: stratify},
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2678: space before tab in indent.
+		func(p sampler.OuterFoldPoint) float64 {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2681: space before tab in indent.
+				return 0
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2682: space before tab in indent.
+			}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2683: space before tab in indent.
+			score, ferr := ss.runOuterFold(ctx, configPts, k, stratify, analysisRefs[p.Idx], outerPart, p.Idx)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2684: space before tab in indent.
+			if ferr != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2689: space before tab in indent.
+				return 0
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2690: space before tab in indent.
+			}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2691: space before tab in indent.
+			return score
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2692: space before tab in indent.
+		},
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2693: space before tab in indent.
+		sampler.ExecFor[sampler.OuterFoldPoint, float64](ss.parallel),
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2696: space before tab in indent.
+		// latches, remaining closures return sentinel zeros — don't fold those into
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2697: space before tab in indent.
+		// the displayed est (the run is aborting; a fake 0 would tank the line).
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2698: space before tab in indent.
+		func(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2703: space before tab in indent.
+		})
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2706: space before tab in indent.
+		return err
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2707: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2710: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2711: space before tab in indent.
+	// metis#32: the nested run now RECORDS (unlike metis#23's estimation-only path) — persist the
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2712: space before tab in indent.
+	// inner + per-family outer rows accumulated in ss.man.Points so `metis select` can reduce them
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2714: space before tab in indent.
+	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2715: space before tab in indent.
+	sortPointRuns(ss.man.Points)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2716: space before tab in indent.
+	if err := writeManifest(ss.o.expPath, ss.man); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2719: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2727: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2728: space before tab in indent.
+	if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2731: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2740: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2743: space before tab in indent.
+	exp.Type = "experiment"
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2744: space before tab in indent.
+	preID, err := pointAddressOf(exp, ss.shapeBlobHash)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2745: space before tab in indent.
+	if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2748: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2749: space before tab in indent.
+	preOpts := ss.o
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2750: space before tab in indent.
+	preOpts.inSweep = true // one preamble run; skip the per-run capture noise
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2751: space before tab in indent.
+	preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2753: space before tab in indent.
+	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2756: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2757: space before tab in indent.
+	refs := make([]string, outerK)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2758: space before tab in indent.
+	for i := 0; i < outerK; i++ {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2761: space before tab in indent.
+	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2762: space before tab in indent.
+	if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2765: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2766: space before tab in indent.
+	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2767: space before tab in indent.
+	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2769: space before tab in indent.
+		hooks: ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2770: space before tab in indent.
+	sres := ss.runSweeper(ctx, configPts, pass)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2771: space before tab in indent.
+	if err := pass.firstError(); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2774: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2775: space before tab in indent.
+	// Guard (metis#19/#23 I1): the parsimony select rule needs a measured complexity for every
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2776: space before tab in indent.
+	// swept family — same guard the flat path runs before trusting its winner. Without it, a
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2777: space before tab in indent.
+	// parsimony-select + non-reporting-model shape would SILENTLY mis-select in each outer fold.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2778: space before tab in indent.
+	if err := sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs)); err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2781: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2782: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2783: space before tab in indent.
+	// metis#32: record the sealed sweep's INNER rows (Level=inner, tagged with this outer fold).
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2784: space before tab in indent.
+	of := i
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2785: space before tab in indent.
+	rows := make([]pointRun, 0, len(pass.points)+len(sres.PerFamily))
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2798: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2799: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2800: space before tab in indent.
+	// (b) score EACH family's inner-winner on the held outer-assessment — post-selection, so
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2802: space before tab in indent.
+	var shipScore float64
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2803: space before tab in indent.
+	for _, fam := range sortedFamilies(sres.PerFamily) {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2804: space before tab in indent.
+		w := sres.PerFamily[fam]
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2807: space before tab in indent.
+		if ferr != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2810: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2839: space before tab in indent.
+		}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2842: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2847: space before tab in indent.
+	return shipScore, nil
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2849: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2855: space before tab in indent.
+	scoreExp := ss.buildFoldExperiment(point, sampler.FoldPoint{Idx: i}, nil, k, stratify, outerPart)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2856: space before tab in indent.
+	scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2857: space before tab in indent.
+	if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2860: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2861: space before tab in indent.
+	scoreOpts := ss.o
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2862: space before tab in indent.
+	scoreOpts.inSweep = true
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2863: space before tab in indent.
+	scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2865: space before tab in indent.
+	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2866: space before tab in indent.
+	if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2869: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2870: space before tab in indent.
+	return run.Metrics[foldMetric], scoreID, run.Status, nil
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2882: space before tab in indent.
+	ss := p.ss
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2883: space before tab in indent.
+	if p.firstError() != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2885: space before tab in indent.
+	// .git/index.lock so a transient probe failure is expected; treating "" as a change
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2886: space before tab in indent.
+	// would false-abort the whole honest run.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2887: space before tab in indent.
+	if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2891: space before tab in indent.
+		return sampler.FoldOutcome{}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2892: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2893: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2894: space before tab in indent.
+	exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.stratify, p.partRef)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2895: space before tab in indent.
+	runID, err := pointAddressOf(exp, ss.shapeBlobHash)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2896: space before tab in indent.
+	if err != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2899: space before tab in indent.
+		return sampler.FoldOutcome{}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2900: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2901: space before tab in indent.
+	pointOpts := ss.o
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2902: space before tab in indent.
+	pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2903: space before tab in indent.
+	pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2905: space before tab in indent.
+	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2906: space before tab in indent.
+	// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2907: space before tab in indent.
+	// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2908: space before tab in indent.
+	// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2909: space before tab in indent.
+	if runErr != nil {
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2913: space before tab in indent.
+		return sampler.FoldOutcome{}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2914: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2917: space before tab in indent.
+		RunID:      runID,
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2918: space before tab in indent.
+		FreeParams: freeParamMap(c),
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2919: space before tab in indent.
+		Fold:       f.Idx,
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2920: space before tab in indent.
+		Status:     run.Status,
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2921: space before tab in indent.
+		Metrics:    run.Metrics,
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2926: space before tab in indent.
+	// metis#19 M2: read the train step's realized-complexity metric. Present → the parsimony
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2927: space before tab in indent.
+	// rules consume it; absent (HasComplexity false) → the guard rejects a parsimony rule.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2928: space before tab in indent.
+	cx, hasCx := run.Metrics[foldComplexityMetric]
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2930: space before tab in indent.
+	}
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:2931: space before tab in indent.
+	return s
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:3008: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:3021: trailing whitespace.
+
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5616: trailing whitespace.
+   134
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5618: trailing whitespace.
+   136
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5621: trailing whitespace.
+   139
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5630: trailing whitespace.
+   148
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5632: trailing whitespace.
+   150
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5646: trailing whitespace.
+   164
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5650: trailing whitespace.
+   168
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5653: trailing whitespace.
+   171
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5655: trailing whitespace.
+   173
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5663: trailing whitespace.
+   181
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5668: trailing whitespace.
+   186
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5677: trailing whitespace.
+   195
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5689: trailing whitespace.
+   207
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5691: trailing whitespace.
+   209
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5698: trailing whitespace.
+   216
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5708: trailing whitespace.
+     2
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5711: trailing whitespace.
+     5
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5714: trailing whitespace.
+     8
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5716: trailing whitespace.
+    10
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5721: trailing whitespace.
+    15
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5723: trailing whitespace.
+    17
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5731: trailing whitespace.
+    25
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5739: trailing whitespace.
+    33
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5741: trailing whitespace.
+    35
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5747: trailing whitespace.
+    41
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5760: trailing whitespace.
+    54
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5769: trailing whitespace.
+    63
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5775: trailing whitespace.
+    69

exec
/bin/zsh -lc 'go test ./cmd/metis -count=1' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 25006ms:
ok	github.com/xianxu/metis/cmd/metis	24.861s

codex
The broader `cmd/metis` package passes, but `git diff --check` over the review range fails on the newly committed close-review transcript. That matters because the issue log claims `git diff --check` was part of the verification evidence. I’m checking whether there are any behavior blockers beyond that hygiene/verification gap.
exec
/bin/zsh -lc "rg -n 'git diff --check|Verification run|diff --check' workshop/issues/000049-board-readability-labels-coldphase-eta.md workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
exec
/bin/zsh -lc "rg -n 'five|5s|monoton|non-increasing|non-decreasing|last.*age|ETA' cmd/metis/*_test.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
workshop/plans/000049-board-readability-labels-coldphase-eta-plan.md:196:   `git diff --check` in metis. Run the relevant kbench checks and `git diff --check` there.
workshop/issues/000049-board-readability-labels-coldphase-eta.md:203:- Verification run: `go test ./cmd/metis -count=1`; `go test ./cmd/metis -race -count=1`;
workshop/issues/000049-board-readability-labels-coldphase-eta.md:204:  `go test ./... -race -count=1`; metis `git diff --check`; kbench `git diff --check`; stale board
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:3235:+   `git diff --check` in metis. Run the relevant kbench checks and `git diff --check` there.
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:3451:- Verification run: `go test ./cmd/metis -count=1`; `go test ./cmd/metis -race -count=1`;
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:3452:  `go test ./... -race -count=1`; metis `git diff --check`; kbench `git diff --check`; stale board
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5685:   203	- Verification run: `go test ./cmd/metis -count=1`; `go test ./cmd/metis -race -count=1`;
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5686:   204	  `go test ./... -race -count=1`; metis `git diff --check`; kbench `git diff --check`; stale board
workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:5790:/bin/zsh -lc 'git diff --check' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta

 succeeded in 0ms:
cmd/metis/fingerprints_test.go:63:// non-monotone in Finished (ledger order is content-keyed, not temporal), and repeated
cmd/metis/board_test.go:45:		"~slots 8/8", "42.5 inner-CV runs/min", "~ETA",
cmd/metis/board_test.go:59:	// All-done: every row ✓, no ETA segment (nothing remaining).
cmd/metis/board_test.go:65:	if strings.Contains(frame, "▸") || strings.Contains(frame, "ETA") {
cmd/metis/board_test.go:66:		t.Errorf("all-done: no in-flight rows, no ETA:\n%s", frame)
cmd/metis/board_test.go:168:	if strings.Contains(frame, "starting") || strings.Contains(frame, "ETA") {
cmd/metis/board_test.go:169:		t.Fatalf("flat pre-confidence frame must be post-startup without ETA:\n%s", frame)
cmd/metis/board_test.go:187:	for _, want := range []string{"45.0 inner-CV runs/min", "last inner-CV run 5s ago", "~ETA"} {
cmd/metis/board_test.go:194:// fmtETA is compact and human: seconds under a minute, m+s under an hour.
cmd/metis/board_test.go:195:func TestFmtETA(t *testing.T) {
cmd/metis/board_test.go:203:		if got := fmtETA(d); got != want {
cmd/metis/board_test.go:204:			t.Errorf("fmtETA(%v) = %q, want %q", d, got, want)
cmd/metis/board_test.go:308:	bw.paint([]string{"folds 2/8", "31.2 folds/min · ETA 12s"})
cmd/metis/board_test.go:317:	for _, stale := range []string{"folds 2/8", "folds/min", "ETA"} {
cmd/metis/board_test.go:451:		"outer folds 0/2", "outer fold 0 — queued", "configs ", "inner-CV runs ", "inner-CV runs/min", "ETA", "score ", "estimate", "mean ",
cmd/metis/progress_test.go:178:		t.Error("16 completions spanning under 15s must be below confidence")
cmd/metis/progress_test.go:185:		t.Errorf("16+ completions spanning ≥15s at now=20s → 45/min, got %v ok=%v", got, ok)
cmd/metis/progress_test.go:196:	// newest kept window = t=1s..64s; at now=65s: 63 completions over 64s ≈ 59.06/min.
cmd/metis/progress_test.go:200:	// ETA: remaining/rate.
cmd/metis/caching_test.go:420:// executor wiring (the pure Kpre five-term test is agnostic to WHICH map feeds Upstream).
cmd/metis/parallel_test.go:261:		t.Fatalf("cold wave acquired five runs before completing a train: %v", events)
cmd/metis/parallel_test.go:517:	for _, forbidden := range []string{"metis: progress", "folds/min", "ETA", "score ", "estimate", "mean "} {

exec
/bin/zsh -lc "rg -n 'leaves line|leaves segment|fold-completion throughput|fold rows|leaves/throughput' cmd/metis/*.go" in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 0ms:
cmd/metis/ledger.go:110:// writeSweepLedger appends a finished sweep's RAW per-fold rows to the shape's ledger
cmd/metis/sweep.go:29:// emits per (config, fold) run. The ledger keeps the raw per-fold rows under its
cmd/metis/sweep.go:351:	// Persist the raw per-fold rows to the shape's append-only ledger sidecar (metis#8/#18):
cmd/metis/sweep.go:359:	// quietly wrong. The raw fold rows are already persisted (re-selectable after a fix); only
cmd/metis/sweep.go:876:// sortPointRuns orders the per-fold rows by a stable content key (RunID, then Fold)
cmd/metis/progress.go:256:	rate      movingRate // metis#38: fold-completion throughput window
cmd/metis/progress.go:264:	gauge     func() (int, int) // (busy, capacity) leaf occupancy; nil = no leaves segment
cmd/metis/run.go:99:	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
cmd/metis/parallel_test.go:30:	body := foldShapeMD("[a, b, c]") // 3 configs × 2 folds = 6 per-fold rows
cmd/metis/board_test.go:13:// renderBoard is the PURE frame: aggregate line, fold rows (✓ done / ▸ in-flight /
cmd/metis/board_test.go:51:	if len(lines) != 5 { // aggregate + 3 fold rows + leaves
cmd/metis/board_test.go:332:// Board mode end-to-end over the fixture sweep: frames paint (cursor hide, fold rows),
cmd/metis/board.go:33:// and the leaves/throughput/ETA line. Pure; width-clamped (a wrapped line would
cmd/metis/board.go:41:	// Per-fold rows (nested only; flat runs have no rows).
cmd/metis/shapesweep_test.go:309:// records its fold rows, and does NOT ship — the outer selection loop has one candidate, so there is
cmd/metis/shapesweep_test.go:343:	// It RECORDS its fold rows (measure) ...
cmd/metis/shapesweep_test.go:346:		t.Error("the flat single-level CV path must record its fold rows to the ledger")
cmd/metis/select_cmd_test.go:396:// Close-review finding 2 (metis#41): a config with FAILED fold rows must be VISIBLE at --point —
cmd/metis/select_cmd_test.go:423:	if !strings.Contains(out.String(), "FAILED fold rows") {
cmd/metis/select_cmd_test.go:424:		t.Errorf("--point must warn loudly about failed fold rows; got:\n%s", out.String())
cmd/metis/select_cmd.go:387:// the rows of EXACTLY ONE config (any of its fold rows works as a handle — they share the
cmd/metis/select_cmd.go:498:				fmt.Fprintf(o.out, "  pooled inner %s: mean %.4f  SE %.4f  (n=%.0f fold rows)\n",
cmd/metis/select_cmd.go:513:		fmt.Fprintf(o.out, "  warning: this config has FAILED fold rows (pooled estimate covers the emitted metrics only); --best would skip it — promoting is an explicit operator override\n")
cmd/metis/ledger_cmd_test.go:64:// not the raw fold rows — the honest leaderboard.
cmd/metis/ledger_cmd_test.go:73:	// 2 configs (a, b) — not 4 raw fold rows. Config a (mean 0.85) sorts above b (0.71).
cmd/metis/ledger_cmd.go:104:		// metis#18: the sidecar holds RAW per-fold rows — reduce to per-config (mean, SE)

exec
/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta
 succeeded in 26368ms:
ok	github.com/xianxu/metis/cmd/metis	26.148s
ok	github.com/xianxu/metis/internal/repo	0.373s
ok	github.com/xianxu/metis/pkg/cache	0.782s
ok	github.com/xianxu/metis/pkg/cas	0.666s
ok	github.com/xianxu/metis/pkg/experiment	1.050s
ok	github.com/xianxu/metis/pkg/ledger	1.426s
ok	github.com/xianxu/metis/pkg/record	1.226s
ok	github.com/xianxu/metis/pkg/sampler	1.617s
ok	github.com/xianxu/metis/pkg/shape	1.838s

codex
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation now exists in `HEAD` and the main behavior matches the #49 purpose: typed activity owns aggregate run counts/rate, board labels are clearer, startup is factual, and ETA is confidence-gated. I found no Critical correctness blocker, but there are two Important close-boundary issues to fix before shipping.

1. Strengths:
- Typed activity seam is real and injected at the cache-aware executor/run persistence boundary: [activity.go](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/cmd/metis/activity.go:55), [run.go](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/cmd/metis/run.go:254).
- Shared vocabulary is centralized in `progressCore`, so plain output and board agree: [progress.go](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/cmd/metis/progress.go:76).
- Run-role propagation covers flat, nested inner, preamble, and outer score paths: [sweep.go](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/cmd/metis/sweep.go:324).
- Core concepts table now has kind/path/status columns.

2. Critical findings:
- None.

3. Important findings:
- [workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:498): `git diff --check 73cf6993cb9cbb87a11696f7adb378b071b39c81..HEAD` fails with many whitespace errors in the committed close-review transcript, contradicting the issue’s verification log at [issue](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/workshop/issues/000049-board-readability-labels-coldphase-eta.md:203). Fix the transcript whitespace or remove the generated transcript from the committed boundary, then rerun `git diff --check`.
- [issue](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/workshop/issues/000049-board-readability-labels-coldphase-eta.md:125): the Done-when asks for a mature trace followed by five seconds of silence proving last-run age advances and rate/ETA monotonicity. Current tests cover readiness/decay and a single `5s` render, but not that full scripted regression. Add the missing focused test.

4. Minor findings:
- Some comments still say “leaves” / “fold-completion throughput” in board/progress internals; live output is fixed, so this is documentation cleanup only.

5. Test coverage notes:
- Ran `go test ./cmd/metis -run 'Activity|CacheActivity|RunExperimentActivity|ShapeSweepActivity|Progress|Board|RunControlActivity' -count=1`: pass.
- Ran `go test ./cmd/metis -count=1`: pass.
- Ran `go test ./... -count=1`: pass.
- `git diff --check 73cf6993cb9cbb87a11696f7adb378b071b39c81..HEAD`: fails as above.

6. Architectural notes:
- ARCH-DRY: pass for output vocabulary and activity-driven counters.
- ARCH-PURE: pass overall; pure reducers are separately unit-tested.
- ARCH-PURPOSE: mostly pass, but the missing five-second mature-trace regression leaves one Done-when proof incomplete.

7. Plan revision recommendations:
- Empty after the current table fix. Just fix the transcript whitespace and add the missing regression before rerunning close.
tokens used
173,744
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation now exists in `HEAD` and the main behavior matches the #49 purpose: typed activity owns aggregate run counts/rate, board labels are clearer, startup is factual, and ETA is confidence-gated. I found no Critical correctness blocker, but there are two Important close-boundary issues to fix before shipping.

1. Strengths:
- Typed activity seam is real and injected at the cache-aware executor/run persistence boundary: [activity.go](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/cmd/metis/activity.go:55), [run.go](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/cmd/metis/run.go:254).
- Shared vocabulary is centralized in `progressCore`, so plain output and board agree: [progress.go](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/cmd/metis/progress.go:76).
- Run-role propagation covers flat, nested inner, preamble, and outer score paths: [sweep.go](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/cmd/metis/sweep.go:324).
- Core concepts table now has kind/path/status columns.

2. Critical findings:
- None.

3. Important findings:
- [workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/workshop/plans/000049-board-readability-labels-coldphase-eta-close-review.md:498): `git diff --check 73cf6993cb9cbb87a11696f7adb378b071b39c81..HEAD` fails with many whitespace errors in the committed close-review transcript, contradicting the issue’s verification log at [issue](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/workshop/issues/000049-board-readability-labels-coldphase-eta.md:203). Fix the transcript whitespace or remove the generated transcript from the committed boundary, then rerun `git diff --check`.
- [issue](/Users/xianxu/workspace/worktree/metis/000049-board-readability-labels-coldphase-eta/workshop/issues/000049-board-readability-labels-coldphase-eta.md:125): the Done-when asks for a mature trace followed by five seconds of silence proving last-run age advances and rate/ETA monotonicity. Current tests cover readiness/decay and a single `5s` render, but not that full scripted regression. Add the missing focused test.

4. Minor findings:
- Some comments still say “leaves” / “fold-completion throughput” in board/progress internals; live output is fixed, so this is documentation cleanup only.

5. Test coverage notes:
- Ran `go test ./cmd/metis -run 'Activity|CacheActivity|RunExperimentActivity|ShapeSweepActivity|Progress|Board|RunControlActivity' -count=1`: pass.
- Ran `go test ./cmd/metis -count=1`: pass.
- Ran `go test ./... -count=1`: pass.
- `git diff --check 73cf6993cb9cbb87a11696f7adb378b071b39c81..HEAD`: fails as above.

6. Architectural notes:
- ARCH-DRY: pass for output vocabulary and activity-driven counters.
- ARCH-PURE: pass overall; pure reducers are separately unit-tested.
- ARCH-PURPOSE: mostly pass, but the missing five-second mature-trace regression leaves one Done-when proof incomplete.

7. Plan revision recommendations:
- Empty after the current table fix. Just fix the transcript whitespace and add the missing regression before rerunning close.
