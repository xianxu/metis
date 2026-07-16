# Boundary Review — metis#48 (whole-issue close)

| field | value |
|-------|-------|
| issue | 48 — pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator |
| repo | metis |
| issue file | workshop/issues/000048-default-leaf-blas-pins.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3306041461821c59ce8853b362792135ab66bc7b..HEAD |
| command | sdlc close --issue 48 |
| reviewer | claude |
| timestamp | 2026-07-16T11:56:36-07:00 |
| verdict | unknown |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
API Error: 529 Overloaded. This is a server-side issue, usually temporary — try again in a moment. If it persists, check https://status.claude.com.

---

## Re-review — 2026-07-16T12:01:03-07:00 (unknown)

| field | value |
|-------|-------|
| issue | 48 — pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator |
| repo | metis |
| issue file | workshop/issues/000048-default-leaf-blas-pins.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3306041461821c59ce8853b362792135ab66bc7b..HEAD |
| command | sdlc close --issue 48 |
| reviewer | claude |
| timestamp | 2026-07-16T12:01:03-07:00 |
| verdict | unknown |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
API Error: 529 Overloaded. This is a server-side issue, usually temporary — try again in a moment. If it persists, check https://status.claude.com.

---

## Re-review — 2026-07-16T12:09:49-07:00 (unknown)

| field | value |
|-------|-------|
| issue | 48 — pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator |
| repo | metis |
| issue file | workshop/issues/000048-default-leaf-blas-pins.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3306041461821c59ce8853b362792135ab66bc7b..HEAD |
| command | sdlc close --issue 48 |
| reviewer | claude |
| timestamp | 2026-07-16T12:09:49-07:00 |
| verdict | unknown |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
API Error: 529 Overloaded. This is a server-side issue, usually temporary — try again in a moment. If it persists, check https://status.claude.com.

---

## Re-review — 2026-07-16T12:54:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 48 — pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator |
| repo | metis |
| issue file | workshop/issues/000048-default-leaf-blas-pins.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3306041461821c59ce8853b362792135ab66bc7b..HEAD |
| command | sdlc close --issue 48 |
| reviewer | codex |
| timestamp | 2026-07-16T12:54:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reading additional input from stdin...
OpenAI Codex v0.144.5
--------
workdir: /Users/xianxu/workspace/metis
model: gpt-5.6-sol
provider: openai
approval: never
sandbox: read-only
reasoning effort: none
reasoning summaries: none
session id: 019f6c7d-2cd2-7252-b940-f304a2f9a6ef
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **metis** repository.

- repository: metis   (root: /Users/xianxu/workspace/metis)
- issue:      metis#48   (file: workshop/issues/000048-default-leaf-blas-pins.md)
- window:     Base: 3306041461821c59ce8853b362792135ab66bc7b   Head: HEAD

Review the **metis** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
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
index 553f0af..5f047e0 100644
--- a/atlas/experiment.md
+++ b/atlas/experiment.md
@@ -203,9 +203,17 @@ wrapped by **thin step-executables** honoring the contract above. Hermetic via *
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
diff --git a/cmd/metis/board_test.go b/cmd/metis/board_test.go
index 284b76b..804c462 100644
--- a/cmd/metis/board_test.go
+++ b/cmd/metis/board_test.go
@@ -304,7 +304,7 @@ func TestServerPool_NoticeRoutesThroughBoard(t *testing.T) {
	var term strings.Builder
	bw := newBoardWriter(&term, steppingClock(300*time.Millisecond))
	bw.paint([]string{"BOARD"})
-	pool := newServerPool(bw) // what runExperiment does post-reorder: pool captures the compositor
+	pool := newServerPool(bw, nil) // what runExperiment does post-reorder: pool captures the compositor
	pool.noticeOnce("k", "server died; falling back to legacy exec")
	s := term.String()
	notice := strings.Index(s, "metis: forkserver: server died")
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
diff --git a/cmd/metis/run.go b/cmd/metis/run.go
index 8323bc4..bd26ca2 100644
--- a/cmd/metis/run.go
+++ b/cmd/metis/run.go
@@ -6,6 +6,7 @@ import (
	"io"
	"os"
	"path/filepath"
+	"strings"
	"sync"
	"time"

@@ -89,6 +90,9 @@ type runOpts struct {
	//          (a plain experiment ignores it; non-TTY/piped runs stay on the #30 plain lines)
	board     *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
	leafGauge func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
+	leafPins  []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
+	//                             runExperiment (nil = not yet computed; non-nil rides nested runOpts
+	//                             copies like forkPool — an all-suppressed result is empty, not nil)
 }

 // runExperiment reads the experiment at o.expPath and dispatches: a `type:
@@ -145,11 +149,24 @@ func runExperiment(o runOpts) (experiment.Run, error) {
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
@@ -202,7 +219,7 @@ func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, n
		return experiment.Run{}, err
	}

-	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool}
+	var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool, pins: o.leafPins}
	if o.exec != nil {
		exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
	}
diff --git a/cmd/metis/select_cmd.go b/cmd/metis/select_cmd.go
index c2db20b..50688db 100644
--- a/cmd/metis/select_cmd.go
+++ b/cmd/metis/select_cmd.go
@@ -358,6 +358,8 @@ func promoteSelected(o selectOpts, sh experiment.Shape, picks []familyPick) erro
			return fmt.Errorf("select --promote %s: %w", famLabel(p.family), err)
		}
		runID := "best-" + familyTag(p.family) + "-" + short(addr)
+		// no leafPins: a promoted ship is a SERIAL single all-data fit — multi-threaded
+		// BLAS is wanted here, and one leaf can't oversubscribe (#48's conscious exclusion)
		ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
		if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
			return fmt.Errorf("select --promote %s (%s): %w", famLabel(p.family), runID, err)
@@ -530,6 +532,8 @@ func runPointSelect(o selectOpts, sh experiment.Shape, led ledger.Ledger, metric
		return fmt.Errorf("select --point --promote: %w", err)
	}
	runID := "point-" + familyTag(fam) + "-" + short(addr)
+	// no leafPins: a promoted ship is a SERIAL single all-data fit — multi-threaded
+	// BLAS is wanted here, and one leaf can't oversubscribe (#48's conscious exclusion)
	ro := runOpts{expPath: o.shapePath, runID: runID, stepPath: o.stepPath, cache: true, git: o.git, exec: o.exec, out: o.out}
	if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
		return fmt.Errorf("select --point --promote (%s): %w", runID, err)
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
index fd88044..05f2ad4 100644
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


codex
I’m applying the repository’s SDLC and review workflow skills to verify the boundary against the tracker, actual diff, and tests. I’ll keep the machine-readable verdict as the first content of the final response.
exec
/bin/zsh -lc "sed -n '1,240p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md && sed -n '1,260p' /Users/xianxu/workspace/ariadne/construct/local/sdlc/SKILL.md && sdlc --help && sed -n '1,240p' workshop/lessons.md" in /Users/xianxu/workspace/metis
 succeeded in 0ms:
---
name: superpowers-using-superpowers
description: Use when starting any conversation - establishes how to find and use skills, requiring Skill tool invocation before ANY response including clarifying questions
---

<SUBAGENT-STOP>
If you were dispatched as a subagent to execute a specific task, skip this skill.
</SUBAGENT-STOP>

<EXTREMELY-IMPORTANT>
If you think there is even a 1% chance a skill might apply to what you are doing, you ABSOLUTELY MUST invoke the skill.

IF A SKILL APPLIES TO YOUR TASK, YOU DO NOT HAVE A CHOICE. YOU MUST USE IT.

This is not negotiable. This is not optional. You cannot rationalize your way out of this.
</EXTREMELY-IMPORTANT>

## Instruction Priority

> **Ariadne note:** AGENTS.md Section 3 governs subagent strategy and overrides skills that mandate subagent-driven-development as the default execution path.

Superpowers skills override default system prompt behavior, but **user instructions always take precedence**:

1. **User's explicit instructions** (CLAUDE.md, GEMINI.md, AGENTS.md, direct requests) — highest priority
2. **Superpowers skills** — override default system behavior where they conflict
3. **Default system prompt** — lowest priority

If CLAUDE.md, GEMINI.md, or AGENTS.md says "don't use TDD" and a skill says "always use TDD," follow the user's instructions. The user is in control.

## How to Access Skills

**In Claude Code:** Use the `Skill` tool. When you invoke a skill, its content is loaded and presented to you—follow it directly. Never use the Read tool on skill files.

**In Gemini CLI:** Skills activate via the `activate_skill` tool. Gemini loads skill metadata at session start and activates the full content on demand.

**In other environments:** Check your platform's documentation for how skills are loaded.

## Platform Adaptation

Skills use Claude Code tool names. Non-CC platforms: see `references/codex-tools.md` (Codex) for tool equivalents. Gemini CLI users get the tool mapping loaded automatically via GEMINI.md.

# Using Skills

## The Rule

**Invoke relevant or requested skills BEFORE any response or action.** Even a 1% chance a skill might apply means that you should invoke the skill to check. If an invoked skill turns out to be wrong for the situation, you don't need to use it.

```dot
digraph skill_flow {
    "User message received" [shape=doublecircle];
    "About to EnterPlanMode?" [shape=doublecircle];
    "Already brainstormed?" [shape=diamond];
    "Invoke brainstorming skill" [shape=box];
    "Might any skill apply?" [shape=diamond];
    "Invoke Skill tool" [shape=box];
    "Announce: 'Using [skill] to [purpose]'" [shape=box];
    "Has checklist?" [shape=diamond];
    "Create TodoWrite todo per item" [shape=box];
    "Follow skill exactly" [shape=box];
    "Respond (including clarifications)" [shape=doublecircle];

    "About to EnterPlanMode?" -> "Already brainstormed?";
    "Already brainstormed?" -> "Invoke brainstorming skill" [label="no"];
    "Already brainstormed?" -> "Might any skill apply?" [label="yes"];
    "Invoke brainstorming skill" -> "Might any skill apply?";

    "User message received" -> "Might any skill apply?";
    "Might any skill apply?" -> "Invoke Skill tool" [label="yes, even 1%"];
    "Might any skill apply?" -> "Respond (including clarifications)" [label="definitely not"];
    "Invoke Skill tool" -> "Announce: 'Using [skill] to [purpose]'";
    "Announce: 'Using [skill] to [purpose]'" -> "Has checklist?";
    "Has checklist?" -> "Create TodoWrite todo per item" [label="yes"];
    "Has checklist?" -> "Follow skill exactly" [label="no"];
    "Create TodoWrite todo per item" -> "Follow skill exactly";
}
```

## Red Flags

These thoughts mean STOP—you're rationalizing:

| Thought | Reality |
|---------|---------|
| "This is just a simple question" | Questions are tasks. Check for skills. |
| "I need more context first" | Skill check comes BEFORE clarifying questions. |
| "Let me explore the codebase first" | Skills tell you HOW to explore. Check first. |
| "I can check git/files quickly" | Files lack conversation context. Check for skills. |
| "Let me gather information first" | Skills tell you HOW to gather information. |
| "This doesn't need a formal skill" | If a skill exists, use it. |
| "I remember this skill" | Skills evolve. Read current version. |
| "This doesn't count as a task" | Action = task. Check for skills. |
| "The skill is overkill" | Simple things become complex. Use it. |
| "I'll just do this one thing first" | Check BEFORE doing anything. |
| "This feels productive" | Undisciplined action wastes time. Skills prevent this. |
| "I know what that means" | Knowing the concept ≠ using the skill. Invoke it. |

## Skill Priority

When multiple skills could apply, use this order:

1. **Process skills first** (brainstorming, debugging) - these determine HOW to approach the task
2. **Implementation skills second** (frontend-design, mcp-builder) - these guide execution

"Let's build X" → brainstorming first, then implementation skills.
"Fix this bug" → debugging first, then domain-specific skills.

## Skill Types

**Rigid** (TDD, debugging): Follow exactly. Don't adapt away discipline.

**Flexible** (patterns): Adapt principles to context.

The skill itself tells you which.

## User Instructions

Instructions say WHAT, not HOW. "Add X" or "Fix Y" doesn't mean skip workflows.
---
name: sdlc
description: Use when at an SDLC checkpoint — starting work, closing an issue or milestone, opening/merging a PR, or recovering workflow state after compaction. The `sdlc` binary owns the gates between workflow stages and refuses transitions that lack required evidence.
---

# sdlc — SDLC checkpoint binary

`sdlc` owns the gates between SDLC workflow stages (claim → change-code → pr →
merge, plus close, milestone-close, judge). It requires evidence at each gate,
mutates state, logs the transition, and refuses transitions that lack the
evidence — that is the shape of a "checkpoint guard."

The binary is the single source of truth. This skill is a static pointer and
intentionally carries no copy of the contract, so it can never drift:

- **`sdlc --help`** — the workflow contract: the start-of-work runbook,
  conventions, and the verb list.
- **`sdlc <verb> --help`** — one checkpoint's full contract, flags, and examples.

Read those instead of relying on memory; the binary's help is always current.
sdlc collects ariadne's SDLC checkpoint guards into one binary. Each subcommand
owns one checkpoint: it requires evidence at the gate, mutates state, logs the
transition, and refuses transitions that lack it. We don't model the SDLC as a
state machine — stages stay prose; we codify the gates between them where drift
recurs. `sdlc` manages the development life cycle; prefer it over `git`/`gh`.

BEFORE WORK
  - `sdlc claim --issue N` — the single start-of-work gesture, a CHEAP LOCK.
    Flips an *open* issue to `working` and publishes the claim to origin/main so
    peer agents see it. No estimate demanded (#113) — claim early, the moment an
    idea crystallizes. `--no-start` suppresses the flip.
  - Do NOT hand-edit an issue's `status:` — let `sdlc claim` or `sdlc issue
    set-status` own that transition (it carries the reopen/`→ done` guards).

ENTER IMPLEMENTATION
  - After plan approval, before editing code, run `sdlc change-code`. It owns the
    branching decision (in-place branch by default; `--worktree=yes` for an
    isolated worktree), the plan-quality check, and the `estimate_hours` gate
    (relocated here from claim, #113). Don't start coding without it.

PUBLISH
  - Publishing goes through a PR: `sdlc pr` → `sdlc merge`. Direct `sdlc push`
    if working directly on main.
  - Publish ONCE at issue close, not per milestone — and do NOT reuse a branch
    name that already has a merged PR. `sdlc merge` refuses (#148) when a branch
    has commits not in main despite a merged PR (a reused name would otherwise
    silently strand the new commits); rename to a fresh branch, `sdlc pr`, retry.

RECOVER
  - After a compaction or session resume, run `sdlc state` to recover where you
    are instead of re-inferring from issue files.

LOCAL REPO TRANSACTION LOCK
  - Mutating verbs take an SDLC-owned repo transaction lock at
    `.git/sdlc.lock` before reading/writing issue state, committing, changing
    branches, or pushing. The lock is local to the Git common dir, so linked
    worktrees of the same repo serialize with each other.
  - Wait messages identify the holder pid and command when metadata is
    available. `close` and `milestone-close` release the lock while the external
    boundary-review subprocess runs, then reacquire before finalization; if HEAD
    or the issue/project file state they prepared changed meanwhile, they refuse
    to finalize and tell you to rerun. `change-code`, `merge`, and `push` can still hold the lock during
    long-running review/ship transactions; wait or retry rather than removing
    the lock while that process is alive.
  - A dead same-host holder is reclaimed automatically; initializing metadata
    is waited through. Other stale/timeout errors tell you how to inspect
    `.git/sdlc.lock`. Remote push/ref races are separate: the local lock
    serializes this checkout, not another machine or clone.

WHEN A VERB ERRORS
  Do NOT route around it with hand-rolled `git`/`gh`. Its errors are next-action
  specs. The fix is one of two things:
    (a) satisfy the precondition it names and re-run the same verb (e.g. `sdlc
        merge` saying "no upstream" → run `sdlc pr` first, then `sdlc merge`); or
    (b) if the error is a genuine gap in `sdlc` itself, fix that edge case in the
        source and re-run. We're still ironing out edge cases.
  Only drop to manual when a verb genuinely cannot express the need — say so.

These gates sit inside a wider prose arc the binary does NOT own: ideation
(parley/pensive) → brainstorm → plan → build → milestone review (`sdlc judge`,
auto-dispatched) → close/ship → postmortem.

CONVENTIONS

  --issue vs --github-issue — `--issue N` always means workshop/issues
  (6-digit ID). `--github-issue N` means a GitHub issue number. Bare `--issue`
  never means a GitHub issue.

  Form vs essence — checkpoint guards (close, milestone-close, push, merge)
  defend against *omission* via required-evidence flags; `sdlc judge` defends
  against *theater* via fresh-context review. Form runs first; judge second.

The verb list + per-verb help (`sdlc <verb> --help`) follow below.

Usage:
  sdlc [flags]
  sdlc [command]

Available Commands:
  claim           Start work: flip an open issue to working + broadcast the claim
  start-plan      Enter planning: deliver the architecture principles to design against (#75)
  change-code     Enter implementation after the structural + plan-quality gates
  issue           Create + manage issues (new / set-status / list / show)
  actual          Compute an issue's focused dev-hours via active-time-v3 (#68)
  active-time     Per-issue active-time attribution table (the v3 engine, standalone)
  close           Close an issue or milestone (ACTUAL + VERIFIED + atlas/project sweep)
  milestone-close Close one milestone + auto-dispatch its review
  pr              Open a pull request from a feature branch
  merge           Merge the PR, archive done issues, clean up
  push            Ship from main (clean tree + pre-merge judges + archive)
  state           Inspect workflow state (branch, working issues, drift)
  resolve         Resolve a symbolic artifact ref (ariadne#11, #15 M4) to its current path(s) — read-only
  open            Resolve a ref and open the primary artifact in $EDITOR
  migrate         Move a markdown artifact to a peer repo, rewriting refs (#179)
  judge           Run an LLM-judge check against the diff (fresh-context)
  arch-principles Print the ARCH-* architecture principles (single source; pull for non-gate work)
  estimate-source Name the shared estimate method + the repo-local calibration source (pull)
  process-manual  Unroll every injection source into a linked process manual (#153)
  propagate-base  Re-weave every recursive dependent of this repo (foundation-first)
  help            Help about any command

Flags:
  -h, --help   help for sdlc

Use "sdlc [command] --help" for more information about a command.
# metis — lessons

Rules distilled from work in metis, to prevent repeats (AGENTS.md §4).

## Go / build
- **Offline Go module bring-up.** Before assuming network is needed for a new `go.mod`, check `$(go env GOMODCACHE)/cache/download/...` for the dep **and its transitive `go.mod` graph** (a pre-1.17 dep like `gopkg.in/yaml.v3` pulls `check.v1`'s go.mod into the unpruned graph). If present, `GOPROXY=off go mod tidy` builds `go.sum` with zero network — no sandbox override. (metis#1 M2)

## Provenance / identity
- **A repro-identity must be minted from the INTENDED config, not the config that happened to execute.** metis#3's record derived its point-address from `Runner.Run`'s returned StepRuns — but a failed run returns only the *pre-failure* steps, so a first-step failure gave an *empty* resolved-config → the point-address (a) diverged from the sweep's full-config run-id and (b) **collided** across distinct failed configs (both empty → same hash), silently collapsing distinct failed rows in #8's aggregation. A point's identity is its *intended* config (you'd re-run the same config); thread the full step list into the address, keep per-step provenance from the executed steps. The reviewer reproduced the collision; the happy-path identity test (ok points only) missed it — pin failed-point identity + distinctness explicitly. (metis#7/#3)

- **A "do X once for the batch" optimization gated by a flag turns unsound when a NEW item joins the flow outside that batch.** metis#18 M1a-5's `driver:single` ship set `runOpts.inSweep=true` to ride the sweep's single code-capture — but `captureSweepCode` runs BEFORE the ship exists and its closure is the union of the *fold* runs only, so the ship (the run that produces the actual submission, incl. ship-only steps predict/submission) silently lost its metis#14 durable SHA on a dirty tree. When you suppress per-item work because "the batch already did it," verify the *specific new item* is in that batch — don't trust the comment. The ship is ONE run: let it capture itself (`inSweep=false`), non-redundant (the optimization only avoids N×k redundant per-*fold* captures). The boundary review caught it; the happy-path ship test (asserting only `"shipped"` + a file exists) had no teeth for provenance. (metis#18 M1a-5)

## CLI / test-through-the-entrypoint
- **An e2e that calls the handler directly bypasses the CLI parse — test through the real entrypoint.** metis#8's `promote`/`ledger show` e2es called `runPromote`/`cmdLedger` with pre-ordered args, so two Critical bugs shipped green: (1) `cmdPromote` injected `commit: nil` and no concrete `gitCommitter` existed, so promote wrote the file, **printed "committed"**, and never committed; (2) Go's stdlib `flag` stops at the first positional, so the *documented* `metis promote <shape> --best` order errored (only flags-first worked). Both are in the CLI wiring the e2e skipped. Add a test that drives the actual `cmd*` entrypoint with the **documented arg order**, and one that asserts the side effect the success message claims (here: the file is really committed). Don't let the success print outrun the action. (metis#8)

## Sweep / self-observation
- **A process that writes into its own repo can't use the whole-repo dirty flag as a "did the world change" signal.** metis#7's detect-and-abort froze on `git status --porcelain` (HEAD sha + dirty), but the sweep writes `runs/`/manifest into the tree → after point 1 the tree is dirty → point 2 false-aborted ("sha → sha-dirty"). A change-detector must exclude the actor's *own* outputs: freeze on the HEAD **commit sha only** (catches the realistic commit/branch drift), or scope the dirty check to code paths. The unit tests (constant clean fake probe) all passed — only the **real CLI run** surfaced it. Drive the real flow for anything that observes its own side effects. (metis#7)

## Testing / defensive copies
- **A deep-copy guard test must exercise the dimension the shallow bug lives in.** metis#6's `Expand` deep-cloned only the *current* step's `with` while the outer per-step map stayed shallow — so sibling points spawned by a *later* step's expansion aliased every *earlier* step's map. The guard test used a **single step**, so it structurally couldn't reach the cross-step case → false confidence (it passed while the bug shipped). A copy-isolation test must mutate the *specific* nesting level / index the shallow copy fails to clone (here: a **non-terminal** step in a **≥2-step** shape). Regression-proof it: revert to the shallow copy and confirm the test FAILS. (The boundary review reproduced the alias empirically; the green single-step test did not.) (metis#6)

- **Every NEW run-producing code path must re-assert the project's defended invariants (reproducibility/#14, cache soundness/#24) with a test that has teeth — a surface signal is not proof.** metis#18 M1a-5's ship-path test asserted `"shipped"` in stdout + a `submission.csv` exists; both pass while code-capture is silently skipped. A defended invariant's test must assert the invariant's OWN evidence (here: the ship record's `CaptureStatus=="captured"` + the side-ref resolves — the same shape the invariant's origin tests use), not a downstream artifact that exists regardless. When you add a run path, ask "which invariants does a run promise?" and copy their teeth. (metis#18 M1a-5)

## Caching / soundness testing
- **A load-bearing serialization invariant deserves a DIRECT codec test, not an implicit e2e guard.** metis#18 M1a-3b's migration guard rests entirely on `[] ≠ nil` surviving the JSON round-trip: a genuine #24 empty transitive-`D` closure must decode to a *non-nil* slice (so an empty-closure step still HITs), while a legacy entry with no key decodes to *nil* (→ MISS). Dropping `omitempty` is what makes `[]` survive as non-nil — but that was only exercised *transitively* by an unrelated warm-HIT e2e (`test/echo` steps have empty `D` and must warm-HIT through the on-disk round-trip). A re-added `omitempty` would silently break the guard AND make every empty-closure step MISS forever, while that e2e might still pass for other reasons. When correctness hinges on "empty-non-nil round-trips as non-nil while absent decodes to nil," pin it with an explicit `Encode→Decode→nil-check` test (+ a legacy/`null` blob → nil) so the regression fails THAT test loudly, independent of any e2e fixture's step choices. (Fresh-eyes review caught this as the one revert-risk in an otherwise-SHIP change.)
- **A CAS/wipeable-cache *consumer* must honor the store's recompute contract, not just the store.** metis#9's `pkg/cas` documents that a consumer treats `ErrNotFound`/`ErrCorrupt` as recompute triggers; metis#2's `cachingExecutor.materialize` propagated them verbatim → a wiped/evicted/corrupt output blob hard-failed the run (exit 1), contradicting the design's "`rm -rf cas/` is safe." A "wipeable cache" is only wipeable if every consumer falls through to recompute on a missing blob — check `errors.Is(err, cas.ErrNotFound|cas.ErrCorrupt)` at the consumer and MISS, don't propagate. Test it: store an entry, `rm -rf cas/`, re-run, assert recompute (not error). Latent-until-eviction: harmless with `maxBytes=0`, breaks the moment LRU eviction is enabled. (metis#2)
- **A "passing" cache e2e can be blind to the cache's core soundness.** metis#2's two e2es both HIT on a re-run — but `TestCache_CheapSweeps` used `test/echo` steps that write no `reads.json` (empty read-set D → *vacuous* HIT), and the toy-pipeline test only re-ran *identically* (HITs whether D is real or empty). So a regression silently emptying D would false-HIT with green CI. The soundness claim ("recompute only what changed") needs a test that **changes something and asserts a MISS**: edit a file in a step's D → that step MISSes, an unaffected step HITs. A HIT-only test proves nothing about invalidation. (The boundary review caught this; the e2es did not.) (metis#2)

## Content-addressed storage / durability
- **Content-addressed dedup must verify, not trust existence.** A CAS `Put` that skips the write when the path *exists* silently defeats the wipeable-cache "recompute-into-place" contract: a *corrupt* blob exists yet fails integrity, so `Get→ErrCorrupt→recompute→Put` hits the dedup-skip and never heals. Verify the existing blob hashes to the key before skipping; overwrite (heal) when absent-or-corrupt. The boundary review caught this as Critical; the happy-path tests didn't. (metis#9)
- **A key→path base primitive is a latent path traversal — validate the key at the boundary.** `shardPath(key) = root/<key[:2]>/<key>` made `Has("..")` return `true` (`..` escaped to root's parent). Gate with a strict format check (`isHash`: exactly 64 lowercase hex) so a malformed key reads as absent, keeping every on-disk path inside `root` — even though today's only callers pass internal sha256 keys (downstream consumers won't). (metis#9)
- **Best-effort maintenance must be consistent across sibling ops.** If a failed recency-stamp (`touch`) is swallowed so it never fails a valid Get/Put, then a failed eviction (`evict`'s `os.Remove`) must be too — otherwise a maintenance hiccup returns `(validHash, err)`, which a consumer reads as "not stored." Make all post-success cache-maintenance best-effort (return nothing, swallow/log). (metis#9)
- **Inject the clock for filesystem recency.** mtime-LRU eviction needs a deterministic recency signal — stamp the file mtime from an injected `Clock` (`os.Chtimes(clock())`) on put/get, don't read wall-clock. Keep the eviction victim-math a *pure* function fed a directory listing (`selectEvictions(entries, maxBytes, keep)`), unit-tested with no filesystem. Re-declare a one-line `Clock` type locally rather than importing upward from a higher layer (a storage *floor* must not depend on `pkg/experiment`). (metis#9)

## Testing
- **A documented concurrency contract needs a concurrent `-race` test.** Running `-race` over a single-goroutine suite proves nothing about a "safe for concurrent use" claim — the race detector only flags races on paths that actually run concurrently. If you write the guarantee in a doc comment (esp. one a downstream consumer is told to rely on), pin it with a `t.Parallel`/goroutine stress test. The boundary review flagged the untested claim as Important. (metis#9)
- **External-binary drift guards.** To stop Go structs drifting from a CUE/schema single source, add a test that shells the sibling validator (e.g. `vocabulary validate-instance`) on a fixture the structs also parse; `t.Skip` when the binary/toolchain is absent so bare checkouts stay green while the guard runs wherever the tool exists. (metis#1 M2)
- **e2e tests that run against fixtures copy them into `t.TempDir()` first.** The step-runner writes `runs/` (+ `record.json`) next to the experiment; running against committed `testdata/` would dirty the tree. (Since #13 the experiment `.md` itself is immutable input — never written back — but the `runs/` artifacts still justify the TempDir rule.) Verify clean with `git status`. (metis#1 M2)
- **Absolute-path fixtures can mask relative-path bugs.** An e2e that fed the runner an absolute `t.TempDir()` path passed green while the natural `metis run <relative-path>` invocation was broken (relative env paths double-joined into `<dir>/<dir>/…`). Exercise the *natural* invocation (chdir + bare filename), not just the convenient absolute one — the boundary review caught this class of bug; `go test` alone did not. (metis#1 M2)

## Python / uv
- **Pin `requires-python`; let uv provision the interpreter — don't build on the system one.** The machine's `python3` was 3.14, which had no binary wheels for pandas/scikit-learn/pyarrow. Setting `requires-python = ">=3.12,<3.13"` made `uv sync` fetch a managed CPython 3.12 and resolve wheels cleanly (no source builds). Check `uv run python -c "import pandas, sklearn, pyarrow"` succeeds *before* writing code on top. (metis#1 M3)
- **Reproducibility is a runner concern, not a per-step one.** Inject the experiment's single `seed` (and any stable anchor like the experiment dir) via env from the runner, rather than duplicating `seed` into every step's `with`. One source (the `#Experiment.seed`), all steps derive — a re-run then reproduces identical outputs (verified: same cv_score + predictions across runs). (metis#1 M3)
- **Step wrappers run with cwd = the step dir, so resolve the project root from `$0`, not cwd.** `steps/metis/*` do `ROOT="$(cd "$(dirname "$0")/../.." && pwd)"; exec uv run --project "$ROOT" …`; a cwd-relative resolution would break because the runner chdirs into `runs/<id>/<step>/`. (metis#1 M3)

## Testing
- **A `uv run` e2e inside `go test` needs a pre-synced venv + a skip guard.** `TestToyPipeline_EndToEnd` shells the real wrappers (`uv run`); run `uv sync` first so per-step invocations are fast/offline, and `t.Skip` when `uv` isn't on PATH so bare checkouts stay green. Confirm `uv run` doesn't dirty the repo (`git status` — the lockfile must be current). (metis#1 M3)
- **A CUE drift-guard for a record with no `type:` discriminator uses `cue vet -d`.** `#Experiment` has a markdown fixture + `validate-instance`, but `#Run` is emitted as `run.json` and carries no `type` field, so guard it by marshaling a Go `Run` to JSON and `cue vet -d '#Run' run.json experiment.cue` (closed schema → renamed/removed/extra field fails). Skip when `cue` is absent. (metis#1 M3)

## Workflow
- **Fresh weave-bootstrapped derivatives need a `construct/base.manifest`.** Without one, the transitive walk stops at the manifest-less repo and a downstream consumer silently under-compiles (only the gitignore action, no error). Author a minimal `internal prose AGENTS.local.md` manifest per new derivative. Tooling fix tracked in `ariadne#155`. (metis#1 M1)

## Integration / verification
- **The acceptance/integration demo IS the invocation-path test — build it before claiming a substrate "done".** metis#7's sweep was only ever exercised with the `test/echo` step, so the sweep→train **hyperparam path was never run** — metis/train did `kind = w["model"]` (a string) and `make_model` ignored C/n_estimators/max_depth. Every piece was contract-correct + unit-green, yet the first real sweep (kbench#4) failed 42/42 points on the `$oneof` model dict (metis#12; `$oneof` was unified into the `$any` map form in metis#17). Corollary: **unit tests with fixed-value fakes mask real-CLI behavior.** metis#8's promote round-trip test used a `fakeGitProbe{sha:"sha"}`, so it "reproduced the point-address" — but in the real CLI `promote` commits the winner (advancing HEAD), and the point-address is code-version-sensitive, so a real round-trip reproduces the *result* (cv_score), not the address. Run the real end-to-end early; a fake that pins the varying input hides exactly the seam the demo exists to test.

## Cross-repo / caching
- **A "walk up to the git repo root" heuristic mis-roots the stdlib when HOME is a git repo.** metis#11's multi-root sensor classified `~/.local/share/uv/python/.../lib/python3.12/*` as first-party because an ancestor (`~`) had a `.git`. The old single-`_PROJECT_ROOT` code was implicitly protected; multi-root wasn't. Exclude the Python install/stdlib prefixes (`sys.base_prefix`, `os.path.dirname(os.__file__)`, `sysconfig` stdlib paths), not just `site-packages`/`.venv`.
- **A persisted cache key's store and validate sides must group/format identically — or you get a false HIT/MISS.** When D went repo-qualified (metis#11), `recordMiss` (store) and `isHit` (validate) both had to group by repo via one shared helper (`hashDByRepo`); asymmetry is a silent soundness bug. And a format bump (v1→v2 `reads.json`) must fail LOUD if an old file is read, never parse to an empty D → a vacuous K_pre-only HIT (worse than the bug being fixed).

## Verification / CLI-invocation
- **A manual-verification command must put flags before positionals AND exercise the code path it claims to.** Go's stdlib `flag` stops parsing at the first non-flag, so a trailing `--dry-run <file>` becomes a second positional and trips arity checks (`metis run` wants exactly one experiment). Worse, a mode flag only affects the branch that reads it: `--dry-run` is sweep-only, so "dry-running" a plain experiment either errors or silently does a full live run — proving nothing about step resolution. Write the proof command against the branch you're actually asserting (here: a hermetic full run that must resolve all three layers), flags first. And never seed a plan's test snippet with speculative imports ("if not already imported") — Go hard-errors on an unused import. (metis#16, plan review)

## Capture / git paths
- **Mixing `filepath.Abs` (keeps the symlink) with git's `--show-toplevel` (realpath) breaks `Rel`.** On macOS a temp path is `/var/folders/…` (a symlink to `/private/var/…`); `git rev-parse --show-toplevel` returns the realpath. `filepath.Rel(realpathRoot, symlinkAbs)` then yields a broken `../../` path that `git hash-object` rejects — silently aborting the whole repo's capture. Fix: `filepath.EvalSymlinks` the path before `Rel`, and symlink-resolve repo-root map keys for a stable identity. (metis#14)
- **Best-effort capture must skip a missing/uncapturable file, not abort the closure.** Adding the run-spec `.md` to the closure hashed a file that didn't exist on disk (a test fixture) → one `git hash-object` failure killed the whole repo's capture → empty D. Guard each optional closure member (spec) with an existence check; a per-file failure should degrade, not zero out. (metis#14)

## Deleting a language/algebra construct (metis#17 plan review)
- **Grep for committed testdata/fixtures the removal breaks, and migrate them in the SAME commit as the engine change — and run `go test ./...`, not a scoped `go test ./pkg/X/`.** Deleting the `$oneof` case from `pkg/shape` broke `cmd/metis/shape_e2e_test.go` (it reads a committed `.md` shape still using `$oneof`), but a shape-package-scoped test would report false-green because the broken consumer lives in `cmd/`. Any commit that removes a grammar/algebra construct must migrate every in-tree consumer (testdata, fixtures, AND the engine file's own doc comments — which describe the deleted semantics) atomically, gated by a whole-module `go test ./...`.

## Design / prior-art (metis-v2 experiment-algebra design)
- **For a load-bearing design decision, run a parallel prior-art survey that MAPS findings to your model — and treat cross-survey convergence as a strong signal.** metis-v2's driver/sweeper/pipeline design dispatched 3 orthogonal researchers (ML frameworks · config/sweep/adaptive · reproducible-caching), each briefed on OUR model + specific open questions and asked to map findings back (not describe tools). Two independent surveys (ML-frameworks *and* caching) arrived at the *same* missing element (a per-step purity/target-safety property) — that convergence was the sharpest evidence in the whole design. It also *validated* the operator's own layering (mlr3 was the exact structural twin) and named the differentiator (1-SE selection is uncontested across all six systems). Cheaper and higher-signal than reasoning from first principles alone.
- **When a design "knob" is really a footgun, the mature-framework move is to REMOVE the degree of freedom, not tune it — prefer a structural/emergent mechanism over a hand-declared marker.** Two of my elaborations got *simpler* under operator pushback: a movable resample cut (`over:`) → one structural `data│pipeline` cut (no mature framework exposes a movable cut — it's a leakage footgun); a per-step `fit_scope` marker → cross-fold safety emergent from the DAG + target-safety owned by the feature step (sklearn `TargetEncoder`). Bias toward "derive/enforce structurally," and if you must know a per-step property, DERIVE it from a trace (metis already traces code reads → extend to column reads), never a hand-typed tag.

## Plan authoring / cache-key changes (metis#18 M1a plan review)
- **Swapping a cache-key term? Trace the FULL invalidation-propagation graph, not the local swap — a term can transitively carry a soundness property that physically lives in a DIFFERENT subsystem.** The M1a plan proposed making the interior input-addressed by replacing `Kpre`'s upstream-*output-hash* term with the upstream's `Kpre`. But metis's read-set `D` deliberately EXCLUDES data + upstream artifacts (`trace.py`: they're "class-1", keyed via upstream output-hashes in `Kpre`), so the output-hash-chain is the *only* carrier of upstream-**code-edit** propagation to downstream steps. Deleting it makes an edit to `features.py` re-run `features` (its own `D` catches it) but NOT re-key `train` (whose `Kpre` uses upstream's code-invariant `Kpre`, and whose `D` excludes upstream output) → `train` serves a stale output. Input-addressing is only sound if paired with a replacement propagation (validate the transitive-`D` closure at HIT-check). The plan's own test asserted the *desired* nondeterminism-suppression — the *identical* mechanism silently suppresses legitimate code-edit propagation, and the key can't tell them apart. Always add a test: "edit an upstream step's code → the downstream step MISSes."
- **Cross-check a plan's canonical example artifact against EVERY task that names the same entity.** Task 5's reshaped shape had no `cv-split` step (resample declared in `sweeper.resample.cv`), but Task 12 assumed a `cv-split` data-phase step — a contradiction visible only when the two tasks are read together, never within either alone. A config-invariant artifact (the partition) must materialize ONCE above the sweeper, and single-sourced config (`k`/`stratify` in one place) must not be duplicated into a step's `with`. When a plan has a worked example, diff it against each task's assumptions.
- **A "validate against upstream state" cache scheme is UNSOUND in a topo executor that heals upstream entries on re-run.** The M1a-3 fix (round 1) was "at a step's hit-check, re-hash its transitive-upstream entries' `D`." But the topo executor runs upstream first: an edited upstream MISSes → re-runs → `recordMiss` OVERWRITES its `Entry.D` with the *new* code hash — *before* the downstream is checked. So the downstream's walk re-hashes the already-healed entry → clean → HIT → serves a stale output (the exact bug the fix targeted, one level up). Sound version: store the **transitive-`D` snapshot in the DOWNSTREAM's own `Entry`** (a topo-fold `transitiveD[id] = ownD ∪ ⋃_{d∈needs} transitiveD[d]`), and validate the current tree against *that* snapshot — store and validate then key on the same bytes (symmetry), and it needs no upstream-entry lookup at validate time (eviction-robust). When a key is deliberately insensitive to an upstream edit (input-addressing), the downstream must carry its own snapshot of what it depended on.
- **A soundness property whose correctness depends on run-time ORDERING is inert under a pure unit test — drive the real executor.** The round-1 `Validate`-level unit test ("move a file in the root's `D`, assert the leaf MISSes") used hand-built entries and went green, blind to the recordMiss-heal ordering that made the mechanism inert. Only a real-executor e2e (warm run → edit `features.py` → assert `train` MISSes) exposes it. If a mechanism's correctness rests on heal-before-check / topo accumulation, the faithful test drives the actual edit→re-run sequence.
- **When a run-time coordinate (fold idx, partition) distinguishes otherwise-identical step invocations, verify it actually enters `Kpre` — don't assume side-channel injection carries it.** `Kpre` has no fold term; if the fold-context is injected like `seed` (env side-channel), all k folds of a step share one `Kpre` → collide (first-runner-wins, wrong scores) and the reducer told-set collapses to one identity. Overlay the coordinate into the step's `with` (or add a `Kpre` arg), and add a "two variants → two distinct cache entries" test (revert the overlay → it FAILS).
- **A `"X→Y"` rename shorthand in a plan is unsafe when `X` has a *second* consumer or its replacement-justification is unverified.** M1a-3's "`upstreamHashes→upstreamKpres`, `c.outputs→c.kpres`" read as clean renames, but `upstreamHashes` the *function* must SURVIVE (the record-provenance path re-uses it independently), while `c.outputs`/`recordOutput` become genuinely DEAD once `Kpre` stops reading them — and the plan's "keep the output-hash for provenance" was a phantom justification (no consumer reads it). Spell out **survive / delete / dead** for each touch-point, and grep the sole-reader before asserting "kept for X." A shorthand rename also silently staled the doc comments claiming the executor `Kpre` and the record derive an identical upstream term (post-change they diverge by design).
- **When a cache soundness mechanism has an asymmetric HIT-vs-MISS population path, the boundary test must exercise the HIT-feeds-downstream arm specifically.** M1a-3's `c.transitiveD[id]` is built from a fresh fold on MISS but *repopulated from the stored entry* on HIT. Every all-MISS soundness test (edit upstream → everything downstream misses) is BLIND to a dropped HIT-repopulation — the bug surfaces only one edit later: edit the *downstream's own* code so the upstream HITS + the downstream re-stores an **empty** upstream closure, THEN edit the upstream → it no longer invalidates. The faithful test edits the downstream first (forcing an upstream HIT into a downstream re-store), then edits the upstream; reverting the repopulation line must fail exactly that test.

## Boundary decomposition (metis#18 M1a-1 impl)
- **A schema-struct change at a foundational boundary breaks its downstream consumers' build; when the decomposition defers the consumer-rewire, scope that boundary's "green" to the changed package + flag the deferred consumer — don't force whole-module green by pulling later-boundary work forward.** M1a-1 removed `Shape.Steps/Sweep/Experiment`, which breaks the 4 `cmd/metis` files (`run.go`/`sweep.go`/`ledger*.go`) that M1a-4 rewires into the nested Sampler loop. Correct move: verify the breakage is *confined* (`go build ./...` names ONLY `cmd/metis`), scope M1a-1's green to `pkg/experiment`+CUE, and flag "`cmd/metis` red until M1a-4" so intermediate milestone-closes review against a known-non-building main package rather than chasing a false whole-module-green bar.
- **A drift-guard / self-test FIXTURE is a hidden consumer of a schema change.** M1a-1's `TestShapeConformsToCUE` reads `testdata/experiment/titanic-baseline-shape.md` — a fixture that had to be reshaped to v2 alongside the live kbench shape the plan named. Grep for the testdata fixtures a schema validates and reshape them in the same commit as the struct/CUE change. **Corollary (M1a-1 review):** a *reference/authoring doc* is a hidden consumer too — the milestone-review's Docs gate caught `construct/datatype/experiment-shape.md` + `atlas/index.md` still describing the deleted v1 vocabulary (an author following them produces a now-rejected shape). Sweep docs, not just fixtures.
- **`yaml.v3` `,inline` on an *exported* embedded struct DOES coexist with `KnownFields(true)`** (promoted fields count as known) — but every existing composite literal of the outer struct that named a now-promoted field breaks and must move to `Outer{Embedded: Embedded{…}}`. Let the compiler enumerate the breakages rather than grepping. (M1a-1 header-DRY refactor.)
- **Size a merged boundary by its VERIFICATION surface, not just its logical cohesion.** M1a-3 (cache) + M1a-4 (IO) were merged so the cache soundness gate could run (it needs `cmd/metis` green). But un-redding `cmd/metis` IS a ~1000-line sweep-driver re-architecture (37 removed-`Shape`-field refs across `run.go`/`sweep.go`/`ledger*.go`) — so "merge so the test can run" silently pulled the whole rewire into one boundary, blowing past its ~1h estimate + one fork's runway. When a boundary's gate is coupled to a large rewire, the boundary inherits the rewire's size. **Corollary — the dependency order was IO-first:** `cmd/metis` red blocks *testing* the cache, and the IO rewire is what un-reds it, so IO must land BEFORE the cache #24 change (reorder, don't just merge). The clean primitives (`Entry.TransitiveD`/`MergeTransitiveD`/`fold_score`) can land first regardless — they're pure + testable without `cmd/metis`.

## Spec design / select-rule + measured complexity (metis#19 design + 2 spec reviews)
- **Trace a selection/scoring rule over the REAL data before a spec claims it "recovers" an empirical result.** #19's v1 spec said the parsimony rule recovers the md=4/6-feature config that scored public 0.782; a reviewer ran it over the cached ledger and it shipped a *different*, unvalidated config (md=4/1-feature) — multi-axis Pareto drove to the joint corner, dominating the 6-feature config on the feature axis. When a spec cites an empirical artifact as the thing a mechanism recovers, verify the mechanism actually *selects that artifact* over the real ledger, not merely something in its neighborhood; word the Done-when as "verified, not asserted."
- **"Minimize X, tie-break Y" only lands on the Y-preferred config when X ties EXACTLY.** #19 relies on realized rf leaf-count being ~feature-independent so the mean tie-break recovers the more-feature config — but `minimize` is primary and unforgiving: one extra leaf re-selects the sparse corner. If a design's correctness rests on an empirical near-equality, (a) gate it in Done-when AND (b) pre-commit a fallback — here, bin the scalar with a tolerance ε so near-equal values tie, then the tie-break decides. "Verified later" with no plan-B re-invites the original failure.
- **A single summary scalar can silently re-encode an axis you declared neutral.** #19's rf complexity as *total* leaf count folds `n_estimators` (declared capacity-neutral per Breiman's LLN) back into the number, wrongly ranking 200 trees "simpler" than 500. Commit to *mean* per-tree. Verify a proposed scalar against the *rationale that justified it*, not just its name.
- **"The reducer" (singular) hides a dual-path ripple; and storage-plumbing ≠ reduction-plumbing.** A metric that must reach a decision flows through EVERY reduction/selection surface — for metis that's `pkg/sampler` (in-memory → shipped `Winner`) AND `pkg/ledger`/`promote` (offline CSV leaderboard, no family grouping). A spec saying "the reducer" under-scopes the plan. Separately: an arbitrary per-metric may already round-trip through record→cache→ledger (`map[string]float64`) while the *typed fold output* (`float64`) and the *reduction into a per-config quantity* do not — "captured + cached" being true does not make "reduced + consumed" true.
- **"Mirrors X / reuses the existing idiom (ARCH-DRY)" must be checked at EVERY representation layer.** The `select` union mirrors `driver` in Go (optional pointers + exactly-one count check) but a first draft claimed a CUE *closed disjunction* — which driver does NOT use (driver is optional CUE fields, Go-only exactly-one). Read the referenced code at each layer (CUE, Go, Python); an idiom is rarely uniform across them.
- **"Property P is recoverable straight off struct S" needs its failure modes enumerated.** #19 first claimed the model family reads off `shape.FreeParam`; but `FreeParam` doesn't record the `$any` FORM, so a tagged branch-label and an untagged bare-string alternative are indistinguishable there. The robust signal lives in a *different* struct — `Point.With`'s single-key-map `{label: sub}` bundling. When a design routes a load-bearing decision through a struct, confirm the struct genuinely carries the distinguishing bit.
- **Check the literature before inventing a capacity/complexity model — and measure the realized artifact, don't predict it from config.** `2^depth` and `min(features, depth)` were both plausible first-principles guesses; `min()` empirically *inverts* (ranks the overfitter simpler). The literature (cost-complexity `|T|` = realized leaves; Breiman LLN → n_estimators-neutral; RF non-parametric → cross-family param-count unsound; tidymodels declares-not-computes) settled it and pivoted the whole design: complexity is a property of the *fitted* model (each model class reports `complexity(fitted)`, e.g. realized leaf count), measured + cached + reduced — not estimated from hyperparameters (static bounds like `2^depth` overstate). One measured scalar collapsed three layers of declared-schema machinery.

## Plan authoring — type-param ripple & tolerance test arithmetic (metis#19 plan review)
- **Widening a Go GENERIC type parameter (a `Sampler[…,O,R]`'s `O`/`R`) is a signature change, not an additive struct change — grep EVERY file in the package, especially internal `_test.go` composition proofs.** #19's plan widened the fold output `float64`→`FoldOutcome` and named the obvious consumers (`folds_test.go`, `configs_test.go`, `driver_test.go`) but missed `pkg/sampler/run_test.go` — an *internal* test that composes all three nested `Run` calls. Because it shares the package, its non-compilation makes `go test ./pkg/sampler/` red, so the plan's per-task "→ PASS" checkpoints were false. Adding a struct FIELD is backward-compatible; changing a type PARAM is not — the "adding fields is safe" intuition does not transfer. Grep the type name across the whole package (incl. `_test.go`) before claiming a package stays green.
- **When a plan pins a tolerance/threshold constant AND hand-writes unit-test numbers around its boundary, arithmetic-check each datum falls on the intended side.** #19 pinned `complexityBinRelTol = 0.05` then wrote a "within-ε" test case of cx 15 vs 16 — a 6.7% gap that is *outside* 5%, so the test would have selected the opposite config and been unwritable as specified. The ε-binning was the exact mechanism the corner-fix rests on, so the silent off-by-a-boundary would have surfaced only when the load-bearing regression test refused to pass. Compute `min·(1+ε)` (or the analogous boundary) against each test datum when authoring the plan.

## Schema migration — Go test-helper fixtures (metis#19 M1)
- **Inline-schema YAML inside Go *test helpers* (not just committed `.md` fixtures) are hidden consumers of a struct/schema change.** Migrating `objective.select` scalar→union broke `cmd/metis/shapesweep_test.go`'s shared `foldShape` helper (fed multiple e2e tests) and `ledger_cmd_test.go`'s inline shape — neither was in the plan's migration list (which named only the `.md` shapes). Grep the WHOLE `cmd/` + `pkg/` tree for the changed field/scalar (`select: argmax-mean`), not just the named shape files; a shared test helper multiplies one stale literal into several red tests. (Extends the drift-guard-fixture note above.)

## Ledger analysis — re-run appends a new sweep cohort (metis#19 M2)
- **Re-running a sweep after a step's code changes APPENDS a new `sweep_sha` cohort to the ledger — it does not replace the old rows.** `AggregateView` groups by `(free-params, sweep-SHA)`, so old and new configs are *distinct* groups; an unscoped `metis ledger select`/`show` over the mixed ledger silently blends two code-versions (in #19 M2: argmax-mean picked the OLD pre-complexity md=8 with cx 0.0, and the parsimony guard tripped on the old complexity-less rows). Scope offline analysis to the fresh `--sweep <full-SHA>` after any re-fit (Filter matches the FULL sha, not a prefix). This is by-design (code-version-addressed provenance = the point of the sweep-SHA), but a real analysis footgun — the mixed ledger looks like one run.

## Plan authoring — temporal availability of runtime-discovered values (metis#27)
- **When a value is runtime-discovered, its compute+write site must be where that input actually exists in the control flow — not a function that runs before it.** #27's `code_fingerprint` = hash of the post-run `D` closure; the first plan drafted it into `buildRecord`, which runs BEFORE `captureRunCode` produces `D` (temporally impossible — would hash an empty closure). The correct site is `backfillCodeManifest` (post-capture, already re-writes the whole record). A plan can cite correct file:line ranges yet place a computation at a point where its input isn't available. When reviewing a plan, trace the *temporal availability* of every input at its claimed compute site, not just that the line numbers are real.

## Ensemble complexity measure follows combination semantics (metis#21 — GBM branch)
- **The reduction over an ensemble's sub-models must match how the ensemble COMBINES them — mean for averaging, SUM for additive.** #19 set rf complexity = *mean* leaves/tree (bagging averages independent trees → count-neutral, Breiman's LLN). GBM (metis#21) is the inverse: boosting is *additive* (F=Σ trees, sequential; ESL §10.2, Friedman 2001), so complexity = *total* leaves *summed* across all boosted trees — count-SENSITIVE, because iteration count is boosting's primary overfitting knob (ESL §10.12; "unlike RF, GBT can overfit"). A mean-per-tree GBM measure would be max_iter-blind → the parsimony rule couldn't prefer fewer rounds → affirmatively wrong. Don't copy a sibling model's reduction; derive it from the combination rule. (XGBoost's own Ω=γT sums leaves across the ensemble — the production precedent.) Measure the REALIZED artifact on the fitted object (`n_iter_ < max_iter` under early stopping), not the configured cap.
- **A regularization/shrinkage knob can decouple a structural complexity proxy from effective capacity ACROSS its values — contain by fixing the knob in the sweep (a stratum), not by inventing an unvalidated correction.** GBM's `learning_rate` (shrinkage) means a low-ν/many-tree config has more leaves yet often regularizes better (Bühlmann–Hothorn: shrinkage slows per-step DoF growth) — so total-leaves misranks across ν. The fix is NOT to bolt on a ν-weighted measure (an untested modeling assumption) but to fix `learning_rate` in the baseline shape (fixed-ν stratum, where the proxy is a clean monotone DoF measure) and defer the correction until a real sweep exposes the misranking (measure-before-rebuild). The model branch stays ν-general; only the demo shape fixes it.

## Nested-CV + read-confinement (metis#23 design + M1/M2 reviews)
- **A "reuse the existing trace" plan must confirm the trace captures the reads you actually need.** #23 planned to reuse metis's read-trace for leakage confinement, but recon found it's a *code* closure (`.py`+`uv.lock` allowlist + `METIS_RUN_DIR` exclusion + parquet's C-extension bypass of the audit hook) — the *data* reads to confine were never in it. The enforcement had to move to the `metis.io` **data chokepoint** (`exp_path`), which covers parquet because it asserts at path-resolution, not via the audit hook. Verify a to-be-reused mechanism's *content*, not just its existence.
- **A confinement chokepoint's placement is decided by the resolver's branch structure, not by "where data is loaded."** The assert belongs in `exp_path` ONLY — `load_dataset`/`dataset_dir`-upstream also serve run-dir handoff reads (a sibling of the analysis root) and confining there crashes every legit `features→train` handoff. And this is **invisible to every offline test** (base-dataset tests never exercise a confined handoff; fake-exec e2e bypasses `metis.io`) → a "handoff-read-PASSES" regression test is mandatory. Corollary (M2): a run-dir artifact (`analysis_i`) read as a *handoff* takes the upstream branch and **bypasses the chokepoint** — to be confined it must be referenced **exp-relative** so the read routes through `exp_path`.
- **A selection-correctness guard is NOT a ship-only step — it belongs on every path that trusts a winner.** M2 forked `GuardComplexity` out of the nested path along with the genuinely ship-only tail (writeManifest/reportWinner/shipWinner), so `driver:cv` + a parsimony rule + a non-reporting model would **silently mis-select per outer fold** while `driver:single` loudly rejected the same shape. When forking a path's tail, classify each step as *ship-only* vs *correctness* — the latter runs on both. (Refactor the guard's input builder to a free function so both paths share it — ARCH-DRY.)
- **When a full real e2e is blocked, prove the mechanism through the real chain in composable pieces + record the residual as a tracked deferral.** #23's confinement: a real-subprocess test drives `execStep → uv cv-split → exp_path` (enforcement), the driver wiring is code-confirmed, and the missing orchestration-level e2e (blocked on a toy data-step) is filed as a follow-up issue — not left as an implied-but-absent guarantee. A composition of real-chain proofs beats one mock e2e; a named deferral beats a silent gap.
- **A full-context fork that stops at a genuine design blocker (rather than guess-building) is working as intended.** The M2 fork built the one unambiguous piece (`CVDriver`), hit the confinement-routing decision, and reported it instead of guessing ~400 lines — exactly right. Resolve the decision in the main session (warm context), then either re-fork the now-unambiguous remainder or build it in-session.

## Leakage-safe target features — prove at the feature level, two-level safety (metis#20 plan review)
- **A target-encoding self-leak is proven at the FEATURE level, not the CV level.** The crisp, non-flaky proof is `corr(encoding, own_label)` on synthetic *no-signal small-group* data: naive-incl-self ≈ 0.7, cross-fit ≈ 0 (huge margins, seed-robust) — plus a real-signal counter-test (`enc.std() > 0` + per-group enc ≈ true rate) to kill a "return prior/constant" cheat. This isolates the leak from model + CV noise; measuring downstream cv-inflation is noisier and harder to make discriminating. A feature that correlates with its own label is the *cause* of cv inflation, so proving it at the encoding level is a strictly superior operationalization of a "naive inflates cv" Done-when.
- **"LOO leaks more than K-fold" is NOT visible in `corr(enc, own_label)` on random data — both are ≈ 0.** LOO's leak is *within-group invertibility*: in a realized group, raw-LOO `enc_i = (S − y_i)/(n−1)` is a deterministic function of (group, own label) — all survivors collapse to one value, all non-survivors to another, separated by exactly `1/(n−1)`, which a flexible model that isolates the group inverts. Test that *structure* directly on a constructed group (deterministic, crisp); never assert a marginal-correlation inequality between two near-zero noise quantities (it passes/fails by seed — a fragile test that looks meaningful).
- **The shrinkage prior conventionally includes the encoded row** (global `y.mean()` blended via `m·prior`), matching sklearn `TargetEncoder` — an O(1/N) residual self-dependence that is accepted, not a bug. Don't write an absolute "own label never used" claim in a docstring; qualify it "never via the group aggregate."
- **Two-level leakage safety for a target feature is separable in code, and both levels are needed.** The *fit-mask* (only analysis rows in `fit_idx`) gives cross-*fold* safety even when assessment rows carry real labels (they're excluded from every aggregate); the *internal cross-fit* gives *within*-fold safety (a fit row's own label never enters its own OOF encoding). The engine gives the first for free (features live in the `pipeline` phase); the feature step owns the second (no marker — the step's own responsibility). Keeping them separate makes each independently testable.

## Parallelizing an injected-seam / content-addressed system (metis#31 plan review)
- **Parallelizing a loop turns every serialized side-effect into a stampede — audit ALL of them, not just the obvious shared writes.** A per-iteration `git status`/`git rev-parse` (or any lock-taking helper) that was harmless serially becomes concurrent contention (`.git/index.lock`) once the loop fans out; combined with error-swallowing that maps "probe failed" onto a domain signal ("code changed mid-sweep"), it becomes a spurious whole-run abort. When adding concurrency, grep the leaf for EVERY subprocess/lock/file-write, not only the data you're protecting — and make a swallowed probe error never masquerade as a definite state change (`if s != "" && s != frozen { abort }`).
- **A concurrency fix needs a reader-vs-writer test, not writer-vs-writer.** Concurrent identical-content `os.WriteFile`s whose only read is after `wg.Wait()` can't observe a torn write, and `-race` treats filesystem ops as non-races — so the test passes even against the non-atomic code it exists to condemn. To prove a temp+rename atomicity fix, race a `lookup`/read loop against the writers (or vary payload length) and assert the reader never sees a partial/parse-failing file.
- **When a test seam injects a fake at the enforcement point, add ONE test that exercises the real enforcement.** If the semaphore/lock lives in the production leaf (`execStep`) but every e2e injects a fake exec that re-implements the guard, the suite validates the PATTERN, never the WIRING — a forgotten acquire or a mis-threaded `runOpts.leafSem → execStep.sem` passes green. A thin test hitting the real leaf (a trivial resolvable subprocess, sem cap 1, two concurrent Execute, assert serialization) closes the gap.
- **Order-preserving fan-out gives bit-identical float reductions — but only for the reduce, not for append-order side-records.** Index-addressed result writes + fixed-order `Tell` make `Aggregate` bit-identical to serial (the honest `Done`); but separate bookkeeping slices (`pass.points`, ledger rows) appended INSIDE the concurrent `runPoint` land in completion order → non-deterministic `manifest.json`/`.ledger.csv` bytes. Sort append-order side-records by a content key before persisting if reproducible bytes matter (content-addressing posture).
- **"Wrap X in a mutex" written in a design's prose is not done until a task builds it.** A promised synchronization affordance (a locked `out` writer) that never appears in the task list is an aspiration; the `-race` gate (or a torn progress line) bills you for it. Track it as a task or delete the promise and make `-race` tests pass `io.Discard`.

## Testing order-preservation without deadlocking the serial baseline (metis#31 impl)
- **A completion-reversal barrier proves a parallel exec keeps INPUT order — but it DEADLOCKS a serial exec.** To prove `ParExec` writes results by index (not completion order), the strongest runPoint makes point `i` block until point `i+1` finishes (so completion order is the reverse of input). But a serial `SeqExec` runs point 0 first, which waits for point 1, which never runs → hang. Fix: use the reversal barrier ONLY on the parallel run; give the serial baseline a plain (no-barrier) runPoint that computes the same scores. Equal results then prove order-preservation. (Corollary: any "wait-for-another-point" test harness is parallel-only by construction.)

## Milestone-splitting a content-address-feeding schema field (metis#32 M1)
- **A schema field that feeds the content-address can't be milestone-split from its downstream consumers.** metis#32's plan split "delete `driver:`" (M1) from "the no-auto-ship behavior + retire the old flow" (M2). But `driver:` is a hashed shape term, and deleting it forces the run-mode to be *derived* (config-count dispatch), which *mechanically* flips every multi-config shape from flat-ship to nested-no-ship AND breaks ~9 inline-`driver:` test fixtures — so the behavior change + the in-repo test migration are pulled into the SAME milestone as the field deletion, not the next one. When a plan splits "delete field X" from "handle X's downstream behavior," check whether X is load-bearing for a derived dispatch or the content-address; if so, they're one boundary. (Only genuinely-decoupled consumers — a peer-repo RUNBOOK, retiring sibling commands, atlas docs — stay in the later milestone.)

## Deleting a "self-contained" command's helpers (metis#32 M2)
- **Before deleting a retired command's helpers, grep the WHOLE package (every file), not just the command's own file.** Retiring `metis promote` (`ledger_cmd.go`), I confirmed its helpers had "no non-promote refs" — but scoped the grep to exclude `ledger_cmd.go` and missed that `freeParamTupleMap` was ALSO called by `ledger show`'s `freeParamTuple` in the SIBLING `ledger.go`. The build caught it (an undefined ref), but a package-wide `grep -rn <helper> pkg cmd` (no per-file exclusion) would have caught it before the delete. A "self-contained" block often has one helper a sibling quietly reuses.

## Deferring PART of a migration surface to yourself (metis#32 kbench migration)
- **When you scope impl forks to "this repo only, I'll do the peer write myself," enumerate EVERY peer file the plan listed — not just the doc.** metis#32 removed the shape `driver:` field + `metis run` auto-ship (a breaking CLI change). The plan's migration surface named the kbench RUNBOOK **and** the kbench sweep shapes **and** the kbench e2e test. I told the forks "metis-repo only, defer kbench to me," then migrated only the RUNBOOK — leaving `titanic-sweep.md`/`-smoke.md` still carrying `driver:` (so `metis run` hard-failed to parse) and `e2e/thread_test.py` asserting the old flat+auto-ship model. The operator hit the parse error immediately after "merged + done." A breaking change isn't done until its DOWNSTREAM CONSUMERS parse/run/pass — grep the peer repo for the removed field/command/behavior (`driver:`, `ledger select`, `metis promote`, "shipped winner") before declaring the migration complete, and treat the peer-repo consumers as part of the issue's close-verification, not an afterthought.

## A substitution seal needs a sole-road check (metis#35)
- **A seal that substitutes a derived artifact and deletes its producers is sound ONLY if that artifact is the sole road from the deleted producers into the consumers — check for bypass edges before deleting.** metis#23 substituted `analysis_i` for the base dataset and dropped the data phase; `features`' `raw: get-data` was a second road (an upstream step-id ref, resolved by `upstream_path` which the L2 confinement deliberately exempts) → dangling read the crafted fixtures never hit. Before deleting a producer, enumerate EVERY `with` leaf of every downstream step for references to it — and note that the ADDRESSING SCHEME decides which guard even sees the read (exp-relative path → confined `exp_path`; step-id → exempt `upstream_path`).
- **A shadow-sweep grep must match every serialization of the reference, not just the canonical one.** The same `raw: get-data` edge existed as YAML in shapes and as JSON (`"raw":"get-data"`) in generated winner experiments; a pattern built from the hand-authored form under-enumerates. Derive the pattern from the KEY NAME, run it from the repo root (so `atlas/`, `workshop/`, generated artifacts are in the net).
- **A two-repo fix must enumerate BOTH repos' atlases** — the close gate only checks the repo you close in; the peer's atlas goes stale silently by default (kbench's atlas documented the two-road boundary verbatim).
- **An assertion written for a path that has never executed is untested code — expect it to be wrong the first time the path runs.** The nested smoke e2e's `0.0 < score` bound (authored in the #32 migration while the test was xfailed) rejected a legitimate 0.0 accuracy on ~3-row fixture folds the first time the test actually ran.

## Default `--parallel NumCPU` thrashes on real sklearn leaves — pin BLAS threads

**metis#42 probe (2026-07-14).** First launch of the k10 sweep with default `--parallel` (NumCPU=12)
drove load-avg to 83: ~22 Python leaves × multi-threaded BLAS each. 885 trains STARTED, zero
finished in ~4 min — the run looked alive while making no progress. The `--parallel` help text
documents exactly this caveat; it still shipped as the default behavior on a real sweep. Relaunch
with `OMP_NUM_THREADS=1 …=1 --parallel 8` → load ~21, ~107 trains/min, done in ~28 min.

**Rule:** for a real (subprocess-leaf) sweep, the leaf's thread env must be pinned and
`--parallel` capped below core count. **RESOLVED BY DEFAULT since metis#48** — bare `metis run`
now injects the four pins at both spawn seams (export a `*_NUM_THREADS` value to override); the
rule survives for non-metis contexts and as the WHY behind the default. Diagnostic signature of
the thrash: starts ≫ completions with the process alive (throughput ≈ 0) — which is also why the
#38 progress board needs a moving-average runs/sec line, not just liveness.

## Plan-sketch folds: set-cardinality, not incremental counts (metis#39 plan review)
- **A "keep the latest, count the others" fold specified incrementally silently overcounts under non-monotone input — specify it as set-cardinality (`len(set)-1`) with the latest tracked separately.** The #39 plan's ExtraCommits sketch counted displacement transitions per ROW; interleaved-timestamp records (two concurrent sweeps, same fingerprint) would have inflated it row-for-row, and the plan's own happy-path fixture (2 records, monotone) structurally couldn't catch it — add an out-of-order fixture whenever a fold's correctness depends on input order. metis-specific ground truth: **ledger rows are NOT time-ordered** (`sortPointRuns` orders by content key; append order is sweep-completion order).
- **When a plan states the same helper in two places (Core-concepts table + task sketch), the signatures WILL drift — single-source the signature in one place and make the other reference it.** The #39 plan declared `printFingerprintLine` with and without a `status` param in two sections; an implementer following either writes a different function, and output-fragment tests wouldn't catch the divergence.

## Plan-review lessons (metis#30 plan)
- **A plan that says "extend the existing e2es" must be verified against what those tests actually exercise and capture.** The #30 plan named a "flat e2e" that never enters the instrumented code path (it runs a plain single experiment, not a sweep) and passes `out: io.Discard` — plans inherit stale mental models of the test suite; grep the named test before approving the task. (There is NO real-uv sweep e2e in this repo; nested e2es drive foldFakeExec.)
- **Event-carried metadata is only as fresh as the first event.** For hierarchical progress, the outer level's first completion can be the END of the run — static facts (totals) must be seeded at wiring time from the same single source, not learned from the stream.
- **For a fan-in event seam shared across concurrent contexts, the payload OR the closure binding must carry the context identity — and beware lookalike identities.** `FoldPoint.Partition` looks like an outer-fold discriminator but is byte-identical across all outer folds; the real identity comes from per-pass closure binding (`runOuterFold` knows `i` at closure creation).
- **Grep-verify counting claims in plans ("all N call sites").** A wrong count signals the enumeration was estimated, not run — the same plan also missed that `countSampler` hardcodes its point-set (a required test-fake refactor left unstated).

## Plan-review lessons (metis#38 plan)
- **Writer identity is temporal, not call-graph.** When a plan claims "all output routes through one wrapper," audit every construction-time capture of the underlying writer (pools, closures, structs built earlier in the call chain) — a component that grabbed the writer BEFORE the wrap exists is an invisible bypass. Grep the writer variable at every `new*(out)` site, not just Fprintf sites. (The forkserver pool + captureSweepCode's o.out both captured pre-board writers.)
- **"Ticker calls repaint()" is a deadlock-or-staleness smell.** In state-owner + painter designs, fix ONE global lock order (state.mu → painter.mu) and route timers through the state owner; a painter-first timer either inverts locks via a state callback or repaints a frame that can't refresh time-derived values (ETA/rate decay).
- **A stdlib-only TUI plan must name its terminal-size mechanism explicitly.** Width detection is the one capability ANSI pin-bottom genuinely needs beyond stdlib's comfortable surface, and a wrong width isn't cosmetic — a wrapped line breaks the cursor-up erase-count bookkeeping the whole repaint scheme rests on.

## Plan-review lessons (metis#48 plan)
- **A constructor-grep is not a coverage proof — also grep direct callers of the layer BELOW the wiring point.** Wiring computed in an entry function (`runExperiment`) silently misses call paths that enter beneath it (select_cmd.go builds fresh `runOpts` and calls `runResolvedExperiment` directly). When a plan claims "every production construction is threaded," check the constructor sites AND downstream-function callers; every bypass found is either threaded or documented as a conscious exclusion.
- **Doc-consistency sweeps must include Go sources, not just `*.md`** — operator guidance lives in flag `--help` strings and load-bearing comments (main.go's `--parallel` help told operators to hand-pin BLAS).
- **A plan's inline test code must be written against the VERIFIED fixture/format; promote any fixture gap to an explicit numbered step.** A parenthetical "check the fixture" hedge next to code that contradicts it breaks the TDD red-green sequence and invites wrong-reason debugging (env-dump dumps METIS_* only; experiment steps live in YAML frontmatter, not a fenced block).

exec
/bin/zsh -lc "sed -n '1,280p' workshop/issues/000048-default-leaf-blas-pins.md; git diff --stat 3306041461821c59ce8853b362792135ab66bc7b..HEAD; git diff --check 3306041461821c59ce8853b362792135ab66bc7b..HEAD; rg -n \"runResolvedExperiment\\(|runExperiment\\(|newServerPool\\(|startForkServer\\(\" cmd/metis --glob '*.go'; rg -n \"OMP_NUM_THREADS|OPENBLAS_NUM_THREADS|VECLIB_MAXIMUM_THREADS|MKL_NUM_THREADS|BLAS pinned|leaf BLAS\" --glob '"'!workshop/history/**'"' ." in /Users/xianxu/workspace/metis
 succeeded in 0ms:
---
id: 000048
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 0.96
started: 2026-07-16T11:10:34-07:00
---

# pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator

## Problem

Running `metis run titanic-sweep.md` bare (no env pins, default `--parallel`=NumCPU) puts the
sweep into BLAS-oversubscription thrash: NumCPU Python leaves × multi-threaded BLAS each →
load-avg ~7× cores, throughput ≈ 0. Observed on the metis#42 k10 probe (load 83, 885 trains
started / 0 finishing) and AGAIN by the operator on 2026-07-16 — the #38 board's rate line
showed the collapse as a ~3h ETA (the display did its job; the default remains a footgun).
The RUNBOOK's "ALWAYS pin OMP/OPENBLAS/VECLIB/MKL=1" is documentation doing a default's job
— the parallelism budget belongs to the ORCHESTRATOR (the #31 leaf semaphore), not to each
leaf's BLAS. (Deeper-fix candidate already flagged in workshop/lessons.md under the #42
entry; promoted to an issue by the operator's hit.)

## Spec

metis sets single-thread BLAS env for its LEAF subprocesses by default:

- At the leaf spawn seams — legacy `execStep` and the #44 fork-server spawn (the server
  process env; forked children inherit) — inject `OMP_NUM_THREADS=1 OPENBLAS_NUM_THREADS=1
  VECLIB_MAXIMUM_THREADS=1 MKL_NUM_THREADS=1` UNLESS the variable is already set in the
  parent env (an explicit operator choice always wins — escape hatch by construction).
- Loud once-per-run note when injecting (visibility: the run's env story is knowable).
- Cache-key question at design: env vars are not in the read-set D — confirm injection does
  not perturb `Kpre`/fingerprint identity (it should not; document why).
- RUNBOOK simplifies: the pins move from "ALWAYS type this" to "defaulted; override by
  exporting your own values".

## Done when

- A bare `metis run` on a real sweep spawns leaves with the four pins set (asserted at the
  spawn seam with a fake exec / env capture); operator-set values pass through untouched.
- Fork-server path covered too (server env at spawn; a test pins it).
- One loud injection note per run; RUNBOOK updated; a bare real-sweep smoke shows sane
  throughput (no thrash) — rate line evidence in the close.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.05 impl=0.25
item: smaller-go-module   design=0.05 impl=0.30
item: smaller-go-module   design=0.02 impl=0.15
item: atlas-docs          design=0.02 impl=0.10
design-buffer: 0.15
total: 0.96
```

Rows: (1) blasPins pure core + unit tests; (2) two spawn seams (execStep, fork-server) +
seam tests; (3) runExperiment wiring + note + full-chain e2e; (4) RUNBOOK rewrite + atlas +
stale-pin grep-sweep.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Cache-identity question resolved at design (no cache work): `Kpre` hashes
`{step_id, uses, with, seed, upstream}`, HIT-validation re-hashes read-set D, fingerprint is
git state — env is in none of them, so injection perturbs nothing (documented in
`blaspins.go`'s doc comment).

## Plan

Durable plan: `workshop/plans/000048-default-leaf-blas-pins-plan.md` (single pass, no Mx —
one close boundary).

- [x] blasPins pure core (ambient-wins rule) + unit tests
- [x] legacy execStep seam: pins field → child env, env-dump seam test
- [x] fork-server seam: pins on server env at spawn (children inherit), real-uv test
- [x] runExperiment once-per-run wiring + loud note + full-chain e2e (note + passthrough)
- [x] docs: RUNBOOK §1 simplification (kbench), atlas + main.go --parallel help + lessons rule, stale-pin grep-sweep (Go+md+py)
- [x] bare real-sweep smoke — rate-line evidence in the Log below

## Log

### 2026-07-16
- Filed from the operator's UX pass (issue 3 of 3): bare `metis run titanic-sweep.md` → board
  ETA ~3h = the #42 thrash signature at default NumCPU without pins. Workaround today:
  the RUNBOOK §1 pinned invocation (`--sample 3 --parallel 8` + env pins) — the full 7,200-fold
  grid bare is the worst case. This issue makes the safe thing the default thing.

### 2026-07-16 (built + smoke)
- Full SDLC single pass: plan fresh-eyes-reviewed (3 Important + hidden-trap sweep — the
  select-path bypass made an explicit decision, env-dump fixture gap promoted to a step, fixture
  syntax corrected against the real frontmatter convention; all folded), change-code judges
  plan-quality CLEAN / estimate-quality INFO. TDD red-green per task; full `go test ./... -race`
  green.
- **Cache-identity confirmed in code** (not just reasoned): `Kpre` = {step_id, uses, with, seed,
  upstream}; validation re-hashes read-set D; fingerprint is git state — env in none of them.
  Documented in blaspins.go.
- **`select --promote` deliberately unpinned** (serial single all-data fit — multi-threaded BLAS
  wanted; one leaf can't oversubscribe): decision comments at both select_cmd.go sites (plan
  review finding).
- **Bare real-sweep smoke (Done-when evidence):** disposable kbench workspace copy (kbench#10
  pattern: rsync minus .metis-cache/runs/.git, .venv symlink + UV_NO_SYNC=1, sibling symlinks for
  the metis#16 deps walk), COLD cache, real 891-row data, `metis run --fast titanic-sweep.md`
  BARE (no env pins, default --parallel, fork-server on) — the operator's exact footgun
  invocation: **`done in 1m23s — 722 rows → ledger (cohort e901889f)`, 720/720 inner folds ≈
  520 folds/min** (pinned reference ~107 trains/min; the pre-#48 bare run was a ~3h-ETA thrash
  with throughput ≈ 0). Exactly ONE note line: `metis: leaf BLAS pinned single-thread
  (MKL... OMP... OPENBLAS... VECLIB...) — the parallelism budget is --parallel; export a value
  yourself to override`. Smoke scale per the estimate-judge advisory: `--fast` (1 outer fold),
  not the full grid — sufficient to exercise both seams under real BLAS load.
- Smoke setup dead ends worth keeping: the deps-chain walk needs the ../kaggle → ../metis
  SIBLINGS present next to a workspace copy (symlinks suffice); a first attempt filtered the
  run's output through `grep -E "...error..."` and swallowed the real failure line ("no
  step-type executable") — capture full logs, filter at read time.
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
 atlas/experiment.md                              | 14 ++++-
 cmd/metis/blaspins.go                            | 45 ++++++++++++++
 cmd/metis/blaspins_e2e_test.go                   | 74 ++++++++++++++++++++++++
 cmd/metis/blaspins_test.go                       | 61 +++++++++++++++++++
 cmd/metis/board_test.go                          |  2 +-
 cmd/metis/exec.go                                |  6 ++
 cmd/metis/exec_test.go                           | 25 ++++++++
 cmd/metis/forkexec.go                            | 15 +++--
 cmd/metis/forkexec_test.go                       | 53 +++++++++++++++--
 cmd/metis/main.go                                |  2 +-
 cmd/metis/run.go                                 | 21 ++++++-
 cmd/metis/select_cmd.go                          |  4 ++
 testdata/steps/test/env-dump                     |  5 ++
 workshop/issues/000048-default-leaf-blas-pins.md | 39 +++++++++++--
 workshop/lessons.md                              | 12 ++--
 15 files changed, 349 insertions(+), 29 deletions(-)
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
cmd/metis/main.go:58:	_, err := runExperiment(runOpts{
cmd/metis/shipe2e_test.go:85:		_, err := runExperiment(runOpts{
cmd/metis/capture_e2e_test.go:39:	run, err := runExperiment(runOpts{
cmd/metis/forkexec_test.go:106:	pool := newServerPool(io.Discard, nil)
cmd/metis/forkexec_test.go:148:	pool := newServerPool(&syncWriter{w: &out}, nil)
cmd/metis/forkexec_test.go:178:		out: &out, pool: newServerPool(&out, nil)}
cmd/metis/forkexec_test.go:233:	pool := newServerPool(io.Discard, nil)
cmd/metis/forkexec_test.go:282:	pool := newServerPool(io.Discard, nil)
cmd/metis/forkexec_test.go:344:	pool := newServerPool(io.Discard, []string{"OMP_NUM_THREADS=1"})
cmd/metis/select_cmd.go:364:		if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
cmd/metis/select_cmd.go:538:	if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
cmd/metis/caching_test.go:95:		_, err := runExperiment(runOpts{
cmd/metis/caching_test.go:162:		r, err := runExperiment(runOpts{
cmd/metis/caching_test.go:250:	if _, err := runExperiment(opts); err != nil {
cmd/metis/caching_test.go:261:	if _, err := runExperiment(opts); err != nil {
cmd/metis/run_test.go:35:	run, err := runExperiment(runOpts{
cmd/metis/run_test.go:104:	run, err := runExperiment(runOpts{
cmd/metis/run_test.go:142:	run, err := runExperiment(runOpts{
cmd/metis/run_test.go:199:	_, err = runExperiment(runOpts{
cmd/metis/sweep.go:465:	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
cmd/metis/sweep.go:554:	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
cmd/metis/sweep.go:611:	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
cmd/metis/forkexec.go:115:func startForkServer(root string, pins []string) (*forkServer, error) {
cmd/metis/forkexec.go:282:func newServerPool(out io.Writer, pins []string) *serverPool {
cmd/metis/forkexec.go:318:		s, err = startForkServer(spec.root, p.pins)
cmd/metis/shapesweep_test.go:161:	_, err := runExperiment(runOpts{
cmd/metis/shapesweep_test.go:252:	_, err := runExperiment(runOpts{
cmd/metis/shapesweep_test.go:426:	_, err := runExperiment(runOpts{
cmd/metis/blaspins_e2e_test.go:47:	if _, err := runExperiment(runOpts{
cmd/metis/caching_soundness_test.go:102:	_, err := runExperiment(runOpts{
cmd/metis/e2e_test.go:48:	run, err := runExperiment(runOpts{
cmd/metis/board_test.go:232:	_, err := runExperiment(runOpts{
cmd/metis/board_test.go:271:	if _, err := runExperiment(runOpts{
cmd/metis/board_test.go:307:	pool := newServerPool(bw, nil) // what runExperiment does post-reorder: pool captures the compositor
cmd/metis/run.go:105:func runExperiment(o runOpts) (experiment.Run, error) {
cmd/metis/run.go:169:		o.forkPool = newServerPool(out, o.leafPins)
cmd/metis/run.go:182:	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
cmd/metis/run.go:207:func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
cmd/metis/nestedcv_e2e_test.go:20:	_, err := runExperiment(runOpts{
cmd/metis/nestedcv_e2e_test.go:127:	if _, err := runExperiment(runOpts{expPath: expPath, now: fixedNow(),
cmd/metis/nestedcv_e2e_test.go:152:	_, err := runExperiment(runOpts{
cmd/metis/nestedcv_e2e_test.go:203:		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
cmd/metis/nestedcv_e2e_test.go:210:		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
cmd/metis/nestedcv_e2e_test.go:217:		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
cmd/metis/nestedcv_e2e_test.go:224:		if _, err := runExperiment(o); err == nil || !strings.Contains(err.Error(), "sample") {
cmd/metis/record_e2e_test.go:69:	run, err := runExperiment(opts)
cmd/metis/record_e2e_test.go:125:	if _, err := runExperiment(opts); err != nil {
cmd/metis/record_e2e_test.go:164:	run, err := runExperiment(runOpts{
cmd/metis/parallel_test.go:28:		if _, err := runExperiment(runOpts{
cmd/metis/parallel_test.go:108:	_, err := runExperiment(runOpts{
cmd/metis/parallel_test.go:156:		if _, err := runExperiment(runOpts{
cmd/metis/parallel_test.go:202:	_, err := runExperiment(runOpts{
cmd/metis/identity_e2e_test.go:46:	if _, err := runExperiment(opts); err != nil {
./atlas/experiment.md:208:- **Default leaf BLAS pins (metis#48) — `cmd/metis/blaspins.go`:** the parallelism budget belongs
./atlas/experiment.md:210:  (`OMP/OPENBLAS/VECLIB/MKL_NUM_THREADS=1`) ONCE per top-level run — minus any name the operator
./cmd/metis/blaspins.go:14:// manual `OMP_NUM_THREADS=1 metis run` never did.
./cmd/metis/blaspins.go:16:	"MKL_NUM_THREADS=1",
./cmd/metis/blaspins.go:17:	"OMP_NUM_THREADS=1",
./cmd/metis/blaspins.go:18:	"OPENBLAS_NUM_THREADS=1",
./cmd/metis/blaspins.go:19:	"VECLIB_MAXIMUM_THREADS=1",
./cmd/metis/blaspins.go:23:// value always wins (escape hatch by construction: `export OMP_NUM_THREADS=8`
./cmd/metis/forkexec_test.go:333:			"json.dump({\"omp\": os.environ.get(\"OMP_NUM_THREADS\", \"\")}, open(\"envcap.json\", \"w\"))\n"+
./cmd/metis/forkexec_test.go:339:	// exactness: an ambient OMP_NUM_THREADS would DUPLICATE the appended pin (CPython's
./cmd/metis/forkexec_test.go:341:	t.Setenv("OMP_NUM_THREADS", "sentinel") // registers restore...
./cmd/metis/forkexec_test.go:342:	os.Unsetenv("OMP_NUM_THREADS")          // ...then genuinely absent for the spawn
./cmd/metis/forkexec_test.go:344:	pool := newServerPool(io.Discard, []string{"OMP_NUM_THREADS=1"})
./workshop/plans/000048-default-leaf-blas-pins-plan.md:5:**Goal:** `metis run` pins leaf BLAS to single-thread by default (OMP/OPENBLAS/VECLIB/MKL=1) at both leaf-spawn seams, unless the operator already exported a value — making the safe thing the default thing (metis#48).
./workshop/plans/000048-default-leaf-blas-pins-plan.md:22:Injecting pins therefore cannot perturb cache keys or fingerprints — exactly as the RUNBOOK's manual `OMP_NUM_THREADS=1 metis run` never did. This is *by design*, not accident: BLAS thread count changes wall-clock, not outputs (same trained model, modulo float nondeterminism that already exists across machines). Documented in `blaspins.go`'s doc comment (the code home of the fact) — no cache change in this issue.
./workshop/plans/000048-default-leaf-blas-pins-plan.md:74:		"MKL_NUM_THREADS=1",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:75:		"OMP_NUM_THREADS=1",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:76:		"OPENBLAS_NUM_THREADS=1",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:77:		"VECLIB_MAXIMUM_THREADS=1",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:87:	got := blasPins([]string{"OMP_NUM_THREADS=8", "PATH=/usr/bin"})
./workshop/plans/000048-default-leaf-blas-pins-plan.md:89:		if kv == "OMP_NUM_THREADS=1" {
./workshop/plans/000048-default-leaf-blas-pins-plan.md:90:			t.Fatalf("ambient OMP_NUM_THREADS=8 must suppress the pin; got %v", got)
./workshop/plans/000048-default-leaf-blas-pins-plan.md:102:		"OMP_NUM_THREADS=4", "OPENBLAS_NUM_THREADS=4",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:103:		"VECLIB_MAXIMUM_THREADS=4", "MKL_NUM_THREADS=4",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:110:// TestBlasPins_PrefixNotName: OMP_NUM_THREADS_X=9 is a DIFFERENT var — must not
./workshop/plans/000048-default-leaf-blas-pins-plan.md:111:// suppress the OMP_NUM_THREADS pin (name match is exact, up to '=').
./workshop/plans/000048-default-leaf-blas-pins-plan.md:113:	got := blasPins([]string{"OMP_NUM_THREADS_X=9"})
./workshop/plans/000048-default-leaf-blas-pins-plan.md:116:		if kv == "OMP_NUM_THREADS=1" {
./workshop/plans/000048-default-leaf-blas-pins-plan.md:147:// manual `OMP_NUM_THREADS=1 metis run` never did.
./workshop/plans/000048-default-leaf-blas-pins-plan.md:149:	"MKL_NUM_THREADS=1",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:150:	"OMP_NUM_THREADS=1",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:151:	"OPENBLAS_NUM_THREADS=1",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:152:	"VECLIB_MAXIMUM_THREADS=1",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:156:// value always wins (escape hatch by construction: `export OMP_NUM_THREADS=8`
./workshop/plans/000048-default-leaf-blas-pins-plan.md:214:		pins:     []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"},
./workshop/plans/000048-default-leaf-blas-pins-plan.md:223:	for _, want := range []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"} {
./workshop/plans/000048-default-leaf-blas-pins-plan.md:238:	pins []string // metis#48: leaf BLAS pins (computed once per run by runExperiment;
./workshop/plans/000048-default-leaf-blas-pins-plan.md:245:	base = append(base, e.pins...) // metis#48: default leaf BLAS pins (operator values already won in blasPins)
./workshop/plans/000048-default-leaf-blas-pins-plan.md:273:			"json.dump({\"omp\": os.environ.get(\"OMP_NUM_THREADS\", \"\")}, open(\"env.json\", \"w\"))\n"+
./workshop/plans/000048-default-leaf-blas-pins-plan.md:279:	// exactness: ambient OMP_NUM_THREADS would DUPLICATE the appended pin (CPython os.environ
./workshop/plans/000048-default-leaf-blas-pins-plan.md:281:	t.Setenv("OMP_NUM_THREADS", "sentinel") // registers restore...
./workshop/plans/000048-default-leaf-blas-pins-plan.md:282:	os.Unsetenv("OMP_NUM_THREADS")          // ...then genuinely absent for the spawn
./workshop/plans/000048-default-leaf-blas-pins-plan.md:284:	pool := newServerPool(&syncWriter{w: &out}, []string{"OMP_NUM_THREADS=1"})
./workshop/plans/000048-default-leaf-blas-pins-plan.md:340:	for _, k := range []string{"OPENBLAS_NUM_THREADS", "VECLIB_MAXIMUM_THREADS", "MKL_NUM_THREADS"} {
./workshop/plans/000048-default-leaf-blas-pins-plan.md:344:	t.Setenv("OMP_NUM_THREADS", "7")
./workshop/plans/000048-default-leaf-blas-pins-plan.md:372:	if n := strings.Count(out.String(), "metis: leaf BLAS pinned"); n != 1 {
./workshop/plans/000048-default-leaf-blas-pins-plan.md:375:	if strings.Contains(out.String(), "OMP_NUM_THREADS") {
./workshop/plans/000048-default-leaf-blas-pins-plan.md:384:	for _, want := range []string{"OPENBLAS_NUM_THREADS=1", "VECLIB_MAXIMUM_THREADS=1", "MKL_NUM_THREADS=1", "OMP_NUM_THREADS=7"} {
./workshop/plans/000048-default-leaf-blas-pins-plan.md:400:	leafPins []string // metis#48: leaf BLAS pins, computed ONCE per top-level run in
./workshop/plans/000048-default-leaf-blas-pins-plan.md:407:	// metis#48: default leaf BLAS pins — computed ONCE from the ambient env (an exported
./workshop/plans/000048-default-leaf-blas-pins-plan.md:413:			fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
./workshop/plans/000048-default-leaf-blas-pins-plan.md:431:- Modify: `kbench/competition/titanic/pipelines/RUNBOOK-sweep.md` (§1, lines ~37–45): the pinned invocation drops the four `*_NUM_THREADS=1` prefixes; reword to "metis pins leaf BLAS single-thread by default (metis#48) — export a value yourself to override; keep `--parallel` as the one knob."
./workshop/plans/000048-default-leaf-blas-pins-plan.md:434:- [ ] **Step 1: RUNBOOK edit** (kbench repo — separate commit there; cite metis#48). Side-note from review: the current RUNBOOK invocation omits `MKL_NUM_THREADS=1` — the default fixes that inconsistency for free; say so in the commit body.
./workshop/plans/000048-default-leaf-blas-pins-plan.md:435:- [ ] **Step 2: `--parallel` flag help text** — main.go:48 says "pin OMP_NUM_THREADS=1 or set n below NumCPU": now contradicts the default; reword ("leaf BLAS is pinned single-thread by default (#48); n is the one knob").
./workshop/plans/000048-default-leaf-blas-pins-plan.md:436:- [ ] **Step 3: atlas edit** + `git add atlas/ && git commit -m "#48: atlas — default leaf BLAS pins"` (metis)
./cmd/metis/blaspins_test.go:12:		"MKL_NUM_THREADS=1",
./cmd/metis/blaspins_test.go:13:		"OMP_NUM_THREADS=1",
./cmd/metis/blaspins_test.go:14:		"OPENBLAS_NUM_THREADS=1",
./cmd/metis/blaspins_test.go:15:		"VECLIB_MAXIMUM_THREADS=1",
./cmd/metis/blaspins_test.go:25:	got := blasPins([]string{"OMP_NUM_THREADS=8", "PATH=/usr/bin"})
./cmd/metis/blaspins_test.go:27:		if kv == "OMP_NUM_THREADS=1" {
./cmd/metis/blaspins_test.go:28:			t.Fatalf("ambient OMP_NUM_THREADS=8 must suppress the pin; got %v", got)
./cmd/metis/blaspins_test.go:40:		"OMP_NUM_THREADS=4", "OPENBLAS_NUM_THREADS=4",
./cmd/metis/blaspins_test.go:41:		"VECLIB_MAXIMUM_THREADS=4", "MKL_NUM_THREADS=4",
./cmd/metis/blaspins_test.go:48:// TestBlasPins_PrefixNotName: OMP_NUM_THREADS_X=9 is a DIFFERENT var — must not
./cmd/metis/blaspins_test.go:49:// suppress the OMP_NUM_THREADS pin (name match is exact, up to '=').
./cmd/metis/blaspins_test.go:51:	got := blasPins([]string{"OMP_NUM_THREADS_X=9"})
./cmd/metis/blaspins_test.go:54:		if kv == "OMP_NUM_THREADS=1" {
./cmd/metis/forkexec.go:279:	pins    []string // metis#48: default leaf BLAS pins, applied to every server's spawn env
./cmd/metis/exec.go:41:	pins []string // metis#48: default leaf BLAS pins (computed once per run by runExperiment;
./cmd/metis/exec.go:131:	// metis#48: default leaf BLAS pins (operator-exported values already won in blasPins,
./workshop/plans/000048-default-leaf-blas-pins-close-review.md:5:| issue | 48 — pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator |
./workshop/plans/000048-default-leaf-blas-pins-close-review.md:27:| issue | 48 — pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator |
./workshop/plans/000048-default-leaf-blas-pins-close-review.md:49:| issue | 48 — pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator |
./cmd/metis/blaspins_e2e_test.go:23:	for _, k := range []string{"OPENBLAS_NUM_THREADS", "VECLIB_MAXIMUM_THREADS", "MKL_NUM_THREADS"} {
./cmd/metis/blaspins_e2e_test.go:27:	t.Setenv("OMP_NUM_THREADS", "7")
./cmd/metis/blaspins_e2e_test.go:57:	if n := strings.Count(out.String(), "metis: leaf BLAS pinned"); n != 1 {
./cmd/metis/blaspins_e2e_test.go:60:	if strings.Contains(out.String(), "OMP_NUM_THREADS") {
./cmd/metis/blaspins_e2e_test.go:69:	for _, want := range []string{"OPENBLAS_NUM_THREADS=1", "VECLIB_MAXIMUM_THREADS=1", "MKL_NUM_THREADS=1", "OMP_NUM_THREADS=7"} {
./workshop/issues/000048-default-leaf-blas-pins.md:12:# pin leaf BLAS threads by default — the parallelism budget belongs to the orchestrator
./workshop/issues/000048-default-leaf-blas-pins.md:31:  process env; forked children inherit) — inject `OMP_NUM_THREADS=1 OPENBLAS_NUM_THREADS=1
./workshop/issues/000048-default-leaf-blas-pins.md:32:  VECLIB_MAXIMUM_THREADS=1 MKL_NUM_THREADS=1` UNLESS the variable is already set in the
./workshop/issues/000048-default-leaf-blas-pins.md:110:  with throughput ≈ 0). Exactly ONE note line: `metis: leaf BLAS pinned single-thread
./cmd/metis/run.go:93:	leafPins  []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
./cmd/metis/run.go:152:	// metis#48: default leaf BLAS pins — computed ONCE per top-level run from the ambient
./cmd/metis/run.go:161:			fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
./workshop/lessons.md:159:with `OMP_NUM_THREADS=1 …=1 --parallel 8` → load ~21, ~107 trains/min, done in ~28 min.
./cmd/metis/exec_test.go:244:		pins:     []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"},
./cmd/metis/exec_test.go:253:	for _, want := range []string{"OMP_NUM_THREADS=1", "MKL_NUM_THREADS=1"} {
./testdata/steps/test/env-dump:14:  # metis#48: the four leaf BLAS pins (full names — asserted verbatim by the seam test)
./testdata/steps/test/env-dump:15:  echo "OMP_NUM_THREADS=${OMP_NUM_THREADS:-<unset>}"
./testdata/steps/test/env-dump:16:  echo "OPENBLAS_NUM_THREADS=${OPENBLAS_NUM_THREADS:-<unset>}"
./testdata/steps/test/env-dump:17:  echo "VECLIB_MAXIMUM_THREADS=${VECLIB_MAXIMUM_THREADS:-<unset>}"
./testdata/steps/test/env-dump:18:  echo "MKL_NUM_THREADS=${MKL_NUM_THREADS:-<unset>}"

exec
/bin/zsh -lc "rg -n \"metis run|parallel|forkserver|NUM_THREADS|RUNBOOK\" README.md atlas cmd/metis/main.go; nl -ba cmd/metis/run.go | sed -n '135,230p'; nl -ba cmd/metis/exec.go | sed -n '105,150p'; nl -ba cmd/metis/forkexec.go | sed -n '105,130p;268,325p'; nl -ba cmd/metis/blaspins.go; git diff 3306041461821c59ce8853b362792135ab66bc7b..HEAD -- README.md workshop/plans/000048-default-leaf-blas-pins-plan.md" in /Users/xianxu/workspace/metis
 succeeded in 0ms:
rg: README.md: No such file or directory (os error 2)
cmd/metis/main.go:1:// Command metis is the Go step-runner control plane: `metis run <experiment.md>`
cmd/metis/main.go:25:		return fmt.Errorf("usage: metis run [--run <id>] <experiment.md>")
cmd/metis/main.go:46:	forkserver := fs.Bool("forkserver", true, "metis#44: run convention-conforming step wrappers through a warm per-root fork-server (pre-imported pandas/sklearn; ~1s spawn tax removed per leaf). --forkserver=false = legacy per-step uv/python spawn (the escape hatch); non-conforming wrappers and failed servers fall back to legacy automatically (loud, once).")
cmd/metis/main.go:48:	parallel := fs.Int("parallel", defaultParallel(), "metis#31: max concurrent step subprocesses across ALL sweep levels (driver×sweeper×resample share one global cap); <=1 = serial (exact pre-#31 behavior). Default runtime.NumCPU(), overridable by METIS_MAX_PARALLEL. Leaf BLAS is pinned single-thread by default (metis#48; export a *_NUM_THREADS value yourself to override), so n is the ONE parallelism knob. On a COLD cache the first batch's ≤n points may each recompute the shared upstream (a bounded thundering herd).")
cmd/metis/main.go:56:	// cmdRun just passes maxParallel; runExperiment establishes the parallel invariant
cmd/metis/main.go:66:		forkserver:  *forkserver,
cmd/metis/main.go:69:		maxParallel: *parallel,
atlas/index.md:44:  command (record IO on inspect/error paths only); each `metis run` prints
atlas/index.md:82:  metis#32: `metis run` now DERIVES the mode by config-count + **records** inner+outer ledger rows +
atlas/index.md:139:  rebuild the exact run DIRECTLY, not by re-expanding the grid. The **driver** is `cmd/metis`: `metis run` on an experiment-shape
atlas/index.md:171:  winner — the Sampler-fold-node algebra driving it is the `pkg/sampler` bullet. `metis run` on a shape
atlas/index.md:196:  writes the index entry. `metis run --cache` (default on).
atlas/index.md:223:`cmd/metis run` + pure `pkg/experiment` `Parse`/`Validate`/`TopoSort`, semantics enforced on
atlas/index.md:226:entrypoints + uv env; `metis run` walks a toy pipeline to a real CV score). The end-to-end
atlas/experiment.md:9:(`metis run <id>`, M2) executes it with **no agent in the loop**, unifying data
atlas/experiment.md:30:`metis run [--run <id>] [--cache] <experiment.md>` reads + validates an experiment, executes
atlas/experiment.md:80:   take each layer's `steps/` dir, **nearest (leaf) first**. So `metis run` in kbench
atlas/experiment.md:153:- **Nested-CV (metis#23; derived, records — metis#32) — `cmd/metis/sweep.go`:** `metis run` on a
atlas/experiment.md:168:  (`Level`-keyed) — the signal `metis select` reduces to pick the family. `metis run` **measures only,
atlas/experiment.md:180:  submit` uploads:** `metis run` on a `>1`-config sweep records the whole nested CV to the ledger — a
atlas/experiment.md:189:  ledger with no `outer` rows is a sharp error (never a silent inner-argmax). `metis run --fast` = one outer
atlas/experiment.md:195:  **in batch order** (`SeqExec` serial default · `ParExec` goroutine fan-out · `ExecFor(parallel)`
atlas/experiment.md:197:  byte-identical `Done` either way — parallelism is a pure speedup, not a semantic change. The ONE
atlas/experiment.md:199:  `--parallel`, default `NumCPU`, `METIS_MAX_PARALLEL`) acquired around `cmd.CombinedOutput()` in
atlas/experiment.md:202:  driver×sweeper×resample nesting, deadlock-free**. `runExperiment` establishes the parallel invariant
atlas/experiment.md:208:- **Default leaf BLAS pins (metis#48) — `cmd/metis/blaspins.go`:** the parallelism budget belongs
atlas/experiment.md:210:  (`OMP/OPENBLAS/VECLIB/MKL_NUM_THREADS=1`) ONCE per top-level run — minus any name the operator
atlas/experiment.md:217:- **Warm fork-server leaf executor (metis#44) — `metis/forkserver.py` + `cmd/metis/forkexec.go`:**
atlas/experiment.md:220:  lazily as `uv run --project <root> python -m metis.forkserver`; it preloads **third-party only**
atlas/experiment.md:232:  root. `metis run --forkserver=false` is the escape hatch. Step authoring is UNCHANGED (a step is
atlas/experiment.md:251:  `testdata/dataset/toy/`) runs end-to-end via `metis run` to a real CV score — the metis#1
atlas/experiment.md:285:`metis run` invokes it **on read** — a cyclic or dangling-`needs` experiment is rejected
   135		if o.maxParallel > 1 && o.leafSem == nil {
   136			o.leafSem = make(chan struct{}, o.maxParallel)
   137		}
   138		if sem := o.leafSem; sem != nil && o.leafGauge == nil {
   139			o.leafGauge = func() (int, int) { return len(sem), cap(sem) } // metis#38: occupancy IS the semaphore
   140		}
   141		// Exactly ONE writer wrap (metis#38): board mode → the pin-bottom compositor (it
   142		// serializes internally — no syncWriter stacking); else parallel → syncWriter.
   143		if o.tui && exp.Type == "experiment-shape" && !o.dryRun {
   144			o.board = newBoardWriter(out, now)
   145			out = o.board
   146			o.out = out
   147			defer o.board.close() // idempotent — an error return must not leak a hidden cursor
   148		} else if o.maxParallel > 1 {
   149			out = &syncWriter{w: out}
   150			o.out = out
   151		}
   152		// metis#48: default leaf BLAS pins — computed ONCE per top-level run from the ambient
   153		// env (an exported operator value wins by exclusion in blasPins), announced loudly,
   154		// and injected at both spawn seams (legacy execStep child env; fork-server process
   155		// env). Fake-exec runs spawn nothing (no pins, no note); dry-run lists configs (same).
   156		// Placed AFTER the writer wrap (the note must route through the board) and BEFORE the
   157		// pool creation (the server spawn env needs the pins).
   158		if o.exec == nil && !o.dryRun && o.leafPins == nil {
   159			o.leafPins = blasPins(os.Environ())
   160			if len(o.leafPins) > 0 {
   161				fmt.Fprintf(out, "metis: leaf BLAS pinned single-thread (%s) — the parallelism budget is --parallel; export a value yourself to override\n",
   162					strings.Join(o.leafPins, " "))
   163			}
   164		}
   165		// metis#44: one warm fork-server pool per top-level run, shut down (EOF-drain) when the
   166		// run ends. Only the production executor uses it (an injected test exec bypasses execStep).
   167		// Constructed AFTER the writer wrap — its fallback notices must route through the board.
   168		if o.forkserver && o.exec == nil && o.forkPool == nil {
   169			o.forkPool = newServerPool(out, o.leafPins)
   170			defer o.forkPool.shutdown()
   171		}
   172		if exp.Type == "experiment-shape" {
   173			sh, err := experiment.ParseShape(string(raw))
   174			if err != nil {
   175				return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
   176			}
   177			if err := experiment.ValidateShape(sh); err != nil {
   178				return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
   179			}
   180			return experiment.Run{}, runShapeSweep(o, sh, now, out)
   181		}
   182		return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
   183	}
   184
   185	// singleRunID names a single run's dir. metis#27: content-address it by the run's
   186	// point-address (symmetric with a sweep point's dir), so the dir name IS the run identity.
   187	// An explicit --run overrides; the timestamp form survives only as the no-git fallback
   188	// (when the shape blob-hash — hence the point-address — can't be computed).
   189	func singleRunID(o runOpts, exp experiment.Experiment, now func() time.Time) string {
   190		if o.runID != "" {
   191			return o.runID
   192		}
   193		sbh, err := shapeBlobHash(o.expPath)
   194		if err == nil {
   195			if addr, err := pointAddressOf(exp, sbh); err == nil {
   196				return addr
   197			}
   198		}
   199		return "run-" + now().UTC().Format("20060102T150405Z")
   200	}
   201
   202	// runResolvedExperiment runs one already-resolved experiment (a single point) under
   203	// runID, through the cached runner, and writes its run.json + provenance record (the
   204	// experiment `.md` is immutable input — not written back, #13). The shared per-point runner
   205	// both the 1-point path and the sweep loop (metis#7) call — so the run/cache/record wiring
   206	// lives in ONE place (ARCH-DRY).
   207	func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
   208		baseDir := filepath.Dir(o.expPath)
   209		// Absolutize at the runner boundary: execStep injects runDir/stepDir/expDir into
   210		// the child's env, and the child's cwd IS the step dir — a relative path would
   211		// resolve $METIS_STEP_DIR/with.json under itself. Absolute paths are correct
   212		// from any cwd, so `metis run pipelines/foo.md` (a relative arg) works.
   213		runDir, err := filepath.Abs(filepath.Join(baseDir, "runs", runID))
   214		if err != nil {
   215			return experiment.Run{}, err
   216		}
   217		expDir, err := filepath.Abs(baseDir)
   218		if err != nil {
   219			return experiment.Run{}, err
   220		}
   221
   222		var exec experiment.StepExecutor = execStep{stepPath: o.stepPath, expDir: expDir, seed: exp.Seed, readRoot: o.readRoot, out: out, sem: o.leafSem, pool: o.forkPool, pins: o.leafPins}
   223		if o.exec != nil {
   224			exec = o.exec // test seam: drive the loop/cache with a fake, no subprocess
   225		}
   226		if o.cache {
   227			cacheDir := filepath.Join(expDir, ".metis-cache")
   228			if err := ensureCacheGitignore(cacheDir); err != nil {
   229				return experiment.Run{}, err
   230			}
   105					// I1: dispatched-and-lost — the forked child may still be running in this
   106					// stepDir; a legacy re-run would double-execute. Error the step instead.
   107					return experiment.StepResult{}, fmt.Errorf("exec %s (forkserver): %v", exe, ferr)
   108				}
   109				if ok {
   110					if resp.Exit != 0 {
   111						return experiment.StepResult{}, fmt.Errorf("exec %s (forkserver): exit status %d\n%s", exe, resp.Exit, resp.Output)
   112					}
   113					return e.collectResult(step, stepDir, runDir)
   114				}
   115			} else {
   116				e.pool.noticeOnce("uses:"+step.Uses,
   117					fmt.Sprintf("step %q wrapper doesn't match the uv/metis.trace convention — legacy exec (no warm-start)", step.Uses))
   118			}
   119		}
   120
   121		cmd := exec.Command(exe)
   122		cmd.Dir = stepDir
   123		// metis#23: strip any inherited METIS_READ_ROOT so an ambient shell value can never
   124		// confine the flat (driver:single) path — we set it ourselves below only when sealing.
   125		base := make([]string, 0, len(os.Environ()))
   126		for _, kv := range os.Environ() {
   127			if !strings.HasPrefix(kv, "METIS_READ_ROOT=") {
   128				base = append(base, kv)
   129			}
   130		}
   131		// metis#48: default leaf BLAS pins (operator-exported values already won in blasPins,
   132		// so no duplicate names reach the child).
   133		base = append(base, e.pins...)
   134		for _, k := range sortedKeys(env) {
   135			base = append(base, k+"="+env[k])
   136		}
   137		cmd.Env = base
   138		// metis#31: acquire the global leaf budget around the ONLY real subprocess spawn
   139		// (resolve/mkdir/with.json above are cheap, non-subprocess — they draw no budget).
   140		// Release immediately after the process exits, before the cheap metrics/artifact
   141		// reads, so a slot is held only while a subprocess is actually running. An
   142		// orchestration goroutine never reaches here holding another slot ⇒ deadlock-free.
   143		if e.sem != nil {
   144			e.sem <- struct{}{}
   145		}
   146		combined, cmdErr := cmd.CombinedOutput()
   147		if e.sem != nil {
   148			<-e.sem
   149		}
   150		if cmdErr != nil {
   105		Module string            `json:"module"`
   106		Cwd    string            `json:"cwd"`
   107		Env    map[string]string `json:"env"`
   108	}
   109
   110	// startForkServer launches `uv run --project <root> python -m metis.forkserver` and wires
   111	// the reader goroutines. The server env = ambient + the default single-thread BLAS pins
   112	// (metis#48; names the operator exported are already excluded by blasPins — an explicit
   113	// choice wins). Forked step children inherit this env; per-step METIS_* vars travel in
   114	// requests, never here.
   115	func startForkServer(root string, pins []string) (*forkServer, error) {
   116		cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.forkserver")
   117		cmd.Dir = root
   118		cmd.Env = append(os.Environ(), pins...)
   119		// Own process GROUP: `uv run` spawns python as a child (no exec), and the server forks
   120		// step children — group-kill is the only way to reap the whole tree on a hung shutdown
   121		// (and a test's mid-flight kill). Normal shutdown stays graceful (stdin EOF → drain);
   122		// Ctrl-C on `metis run` closes the stdin pipe, so detached servers still self-exit.
   123		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
   124		stdin, err := cmd.StdinPipe()
   125		if err != nil {
   126			return nil, err
   127		}
   128		stdout, err := cmd.StdoutPipe()
   129		if err != nil {
   130			return nil, err
   268	}
   269
   270	// serverPool lazily starts one forkServer per project root and remembers roots whose start
   271	// failed (so a broken layer degrades to legacy once, loudly, not per-leaf).
   272	type serverPool struct {
   273		mu      sync.Mutex
   274		servers map[string]*forkServer
   275		broken  map[string]bool
   276		warned  map[string]bool
   277		out     io.Writer
   278		outMu   sync.Mutex
   279		pins    []string // metis#48: default leaf BLAS pins, applied to every server's spawn env
   280	}
   281
   282	func newServerPool(out io.Writer, pins []string) *serverPool {
   283		return &serverPool{
   284			servers: map[string]*forkServer{},
   285			broken:  map[string]bool{},
   286			warned:  map[string]bool{},
   287			out:     out,
   288			pins:    pins,
   289		}
   290	}
   291
   292	// noticeOnce prints one loud line per key — the escape-hatch visibility contract: falling
   293	// back to legacy exec is fine, doing it silently is not.
   294	func (p *serverPool) noticeOnce(key, msg string) {
   295		p.outMu.Lock()
   296		defer p.outMu.Unlock()
   297		if p.warned[key] {
   298			return
   299		}
   300		p.warned[key] = true
   301		fmt.Fprintf(p.out, "metis: forkserver: %s\n", msg)
   302	}
   303
   304	// execute routes a parsed step through the root's warm server. Outcomes:
   305	//   - (resp, true, nil)  — a real step outcome (even exit != 0);
   306	//   - (_, false, nil)    — nothing dispatched (broken/unstartable server) → legacy is SAFE;
   307	//   - (_, false, err)    — dispatched-and-lost → the caller must ERROR the step (I1: the
   308	//     forked child may still be running; a legacy re-run would double-execute into stepDir).
   309	func (p *serverPool) execute(spec wrapperSpec, cwd string, env map[string]string) (forkResp, bool, error) {
   310		p.mu.Lock()
   311		if p.broken[spec.root] {
   312			p.mu.Unlock()
   313			return forkResp{}, false, nil
   314		}
   315		s := p.servers[spec.root]
   316		if s == nil {
   317			var err error
   318			s, err = startForkServer(spec.root, p.pins)
   319			if err != nil {
   320				p.broken[spec.root] = true
   321				p.mu.Unlock()
   322				p.noticeOnce("root:"+spec.root, fmt.Sprintf("start failed for %s (%v) — legacy exec for this root", spec.root, err))
   323				return forkResp{}, false, nil
   324			}
   325			p.servers[spec.root] = s
     1	package main
     2
     3	import "strings"
     4
     5	// blasPinDefaults are the single-thread pins metis injects into LEAF subprocesses by
     6	// default (metis#48): the parallelism budget belongs to the ORCHESTRATOR (the metis#31
     7	// leaf semaphore), not to each leaf's BLAS — NumCPU leaves × multi-threaded BLAS
     8	// oversubscribes ~NumCPU× (observed: load-avg 83 on 12 cores, throughput ≈ 0).
     9	//
    10	// Cache identity: env is deliberately OUTSIDE run identity — Kpre hashes
    11	// {step_id, uses, with, seed, upstream} (pkg/cache), HIT-validation re-hashes the
    12	// read-set D (file blob hashes), and the code fingerprint is git state. Injecting
    13	// pins perturbs neither cache keys nor fingerprints — exactly as the RUNBOOK's
    14	// manual `OMP_NUM_THREADS=1 metis run` never did.
    15	var blasPinDefaults = []string{
    16		"MKL_NUM_THREADS=1",
    17		"OMP_NUM_THREADS=1",
    18		"OPENBLAS_NUM_THREADS=1",
    19		"VECLIB_MAXIMUM_THREADS=1",
    20	}
    21
    22	// blasPins returns the defaults NOT already set in environ — an explicit operator
    23	// value always wins (escape hatch by construction: `export OMP_NUM_THREADS=8`
    24	// passes through untouched). Pure. Always non-nil: an all-suppressed result is
    25	// empty, distinguishable from runOpts' nil "not yet computed" sentinel.
    26	func blasPins(environ []string) []string {
    27		pins := make([]string, 0, len(blasPinDefaults))
    28		for _, def := range blasPinDefaults {
    29			name := def[:strings.IndexByte(def, '=')]
    30			if !envHasName(environ, name) {
    31				pins = append(pins, def)
    32			}
    33		}
    34		return pins
    35	}
    36
    37	// envHasName reports whether environ sets exactly `name` (match up to '=').
    38	func envHasName(environ []string, name string) bool {
    39		for _, kv := range environ {
    40			if strings.HasPrefix(kv, name) && len(kv) > len(name) && kv[len(name)] == '=' {
    41				return true
    42			}
    43		}
    44		return false
    45	}
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead

exec
/bin/zsh -lc "rg -n \"fork|environ|Env|req.env|env.update|os.exec\" metis/forkserver.py cmd/metis/forkexec.go; nl -ba metis/forkserver.py | sed -n '1,240p'; git show 3306041461821c59ce8853b362792135ab66bc7b:cmd/metis/run.go | sed -n '95,190p'; git diff --name-only 3306041461821c59ce8853b362792135ab66bc7b..HEAD | rg 'README|atlas|RUNBOOK' || true" in /Users/xianxu/workspace/metis
 succeeded in 0ms:
metis/forkserver.py:1:"""metis.forkserver — the warm leaf fork-server (metis#44).
metis/forkserver.py:3:`python -m metis.forkserver` pre-imports the heavy third-party libraries once, then serves
metis/forkserver.py:4:step-execution requests as JSON lines on stdin. Per request it **forks** (main thread only):
metis/forkserver.py:24:  - Only the MAIN thread forks; waiter threads do IO only. The server never executes a
metis/forkserver.py:28:POSIX-only by construction (os.fork).
metis/forkserver.py:49:    names = os.environ.get("METIS_FORKSERVER_PRELOAD", _DEFAULT_PRELOAD)
metis/forkserver.py:55:            print(f"forkserver: preload {name!r} not importable (skipped)", file=sys.stderr)
metis/forkserver.py:64:        for k in [k for k in os.environ if k.startswith("METIS_")]:
metis/forkserver.py:65:            del os.environ[k]
metis/forkserver.py:66:        os.environ.update(req.get("env") or {})
metis/forkserver.py:89:    response line. IO only — never forks, never touches os.environ."""
metis/forkserver.py:121:        # C1 (close review): fork UNDER the protocol lock. Waiter threads write responses to
metis/forkserver.py:122:        # `out` only while holding `lock`, so holding it across fork() guarantees stdout's
metis/forkserver.py:124:        # fork landing mid-write copies a held io lock into the child (whose owner thread
metis/forkserver.py:128:            pid = os.fork()
metis/forkserver.py:138:            # D symmetry: the child inherits metis.forkserver in sys.modules (first-party →
metis/forkserver.py:139:            # it would land in every forkserver-run step's read-set D, making cache keys
metis/forkserver.py:142:            sys.modules.pop("metis.forkserver", None)
cmd/metis/forkexec.go:3:// forkexec.go — the metis#44 warm fork-server executor seam. The step-AUTHORING contract is
cmd/metis/forkexec.go:6:// `python -m metis.forkserver` (one per project root — metis's and kbench's venvs differ)
cmd/metis/forkexec.go:19:	"os/exec"
cmd/metis/forkexec.go:42:// parseWrapper decides forkability: a wrapper is forkable iff it follows the convention
cmd/metis/forkexec.go:67:// errServerUnavailable marks "the fork-server for this root can't serve" (start failure or
cmd/metis/forkexec.go:69:var errServerUnavailable = errors.New("fork-server unavailable")
cmd/metis/forkexec.go:71:type forkResp struct {
cmd/metis/forkexec.go:76:// forkServer is one warm server process (one uv project root). Response routing is by
cmd/metis/forkexec.go:78:type forkServer struct {
cmd/metis/forkexec.go:83:	pending map[int]chan forkResp
cmd/metis/forkexec.go:103:type forkReq struct {
cmd/metis/forkexec.go:107:	Env    map[string]string `json:"env"`
cmd/metis/forkexec.go:110:// startForkServer launches `uv run --project <root> python -m metis.forkserver` and wires
cmd/metis/forkexec.go:115:func startForkServer(root string, pins []string) (*forkServer, error) {
cmd/metis/forkexec.go:116:	cmd := exec.Command("uv", "run", "--project", root, "python", "-m", "metis.forkserver")
cmd/metis/forkexec.go:118:	cmd.Env = append(os.Environ(), pins...)
cmd/metis/forkexec.go:119:	// Own process GROUP: `uv run` spawns python as a child (no exec), and the server forks
cmd/metis/forkexec.go:139:	s := &forkServer{
cmd/metis/forkexec.go:141:		pending: map[int]chan forkResp{},
cmd/metis/forkexec.go:150:func (s *forkServer) drainStderr(r io.Reader) {
cmd/metis/forkexec.go:169:func (s *forkServer) readLoop(stdout io.Reader) {
cmd/metis/forkexec.go:195:			ch <- forkResp{Exit: pl.Exit, Output: pl.Output}
cmd/metis/forkexec.go:205:		err = errors.New("fork-server exited")
cmd/metis/forkexec.go:220:// the forked child may STILL BE RUNNING and writing into stepDir, so the caller must ERROR
cmd/metis/forkexec.go:223:var errDispatchedLost = errors.New("fork-server died with the request in flight")
cmd/metis/forkexec.go:228:func (s *forkServer) execute(module, cwd string, env map[string]string) (forkResp, error) {
cmd/metis/forkexec.go:233:		return forkResp{}, s.dead
cmd/metis/forkexec.go:237:	ch := make(chan forkResp, 1)
cmd/metis/forkexec.go:239:	b, err := json.Marshal(forkReq{ID: id, Module: module, Cwd: cwd, Env: env})
cmd/metis/forkexec.go:245:		// dispatched (conservative: no legacy re-run against a possibly-forked child).
cmd/metis/forkexec.go:248:		return forkResp{}, fmt.Errorf("%w: write request: %v", errDispatchedLost, err)
cmd/metis/forkexec.go:255:		return forkResp{}, fmt.Errorf("%w: %v", errDispatchedLost, s.dead)
cmd/metis/forkexec.go:260:func (s *forkServer) shutdown() {
cmd/metis/forkexec.go:270:// serverPool lazily starts one forkServer per project root and remembers roots whose start
cmd/metis/forkexec.go:274:	servers map[string]*forkServer
cmd/metis/forkexec.go:284:		servers: map[string]*forkServer{},
cmd/metis/forkexec.go:301:	fmt.Fprintf(p.out, "metis: forkserver: %s\n", msg)
cmd/metis/forkexec.go:308://     forked child may still be running; a legacy re-run would double-execute into stepDir).
cmd/metis/forkexec.go:309:func (p *serverPool) execute(spec wrapperSpec, cwd string, env map[string]string) (forkResp, bool, error) {
cmd/metis/forkexec.go:313:		return forkResp{}, false, nil
cmd/metis/forkexec.go:323:			return forkResp{}, false, nil
cmd/metis/forkexec.go:336:			return forkResp{}, false, err
cmd/metis/forkexec.go:339:		return forkResp{}, false, nil
cmd/metis/forkexec.go:347:	servers := make([]*forkServer, 0, len(p.servers))
cmd/metis/forkexec.go:351:	p.servers = map[string]*forkServer{}
     1	"""metis.forkserver — the warm leaf fork-server (metis#44).
     2
     3	`python -m metis.forkserver` pre-imports the heavy third-party libraries once, then serves
     4	step-execution requests as JSON lines on stdin. Per request it **forks** (main thread only):
     5	the child scrubs `METIS_*` from its env, applies the request's env verbatim, chdirs to the
     6	step dir, redirects stdout+stderr onto a per-request pipe, and runs the step through the SAME
     7	`metis.trace.run_traced` machinery as the legacy `python -m metis.trace <mod>` subprocess —
     8	then `os._exit`s. Every step therefore keeps today's per-process semantics (crash isolation,
     9	its own reads.json, an authoritative env) while skipping the ~1s interpreter+import tax that
    10	metis#44 measured on every one of a sweep's ~5k leaf spawns.
    11
    12	Protocol (JSONL over the server's stdin/stdout; stderr is free-form diagnostics):
    13	  →  {"id": 7, "module": "kbench.titanic.features", "cwd": "<stepDir>", "env": {"METIS_*": …}}
    14	  ←  {"ready": true}                        (once, after preload — requests may be sent before)
    15	  ←  {"id": 7, "exit": 0, "output": "<combined child stdout+stderr, tail-capped>"}
    16	stdin EOF ⇒ drain in-flight children, exit 0.
    17
    18	Design constraints (the metis#44 plan pins these):
    19	  - Preload THIRD-PARTY ONLY. The cache read-set D is derived from the child's sys.modules
    20	    snapshot (first-party filter): preloaded first-party would widen every step's D
    21	    (over-invalidation), a delta-rule would under-capture it (stale cache hits). Third-party
    22	    is excluded from D by design, so warming it is free. Override the preload list with
    23	    `METIS_FORKSERVER_PRELOAD` (comma-separated; empty string = no preload — tests).
    24	  - Only the MAIN thread forks; waiter threads do IO only. The server never executes a
    25	    BLAS/OpenMP region (imports only), so each child initializes its own threadpools.
    26	  - The child forces `used_site_packages` (see run_traced) — it inherits the imports and
    27	    would never observe the site-packages reads that normally set the flag.
    28	POSIX-only by construction (os.fork).
    29	"""
    30
    31	from __future__ import annotations
    32
    33	import importlib
    34	import io
    35	import json
    36	import os
    37	import sys
    38	import threading
    39	import traceback
    40
    41	_DEFAULT_PRELOAD = "numpy,pandas,sklearn,pyarrow"
    42	_OUTPUT_CAP = 200_000  # keep the tail; matches CombinedOutput's role (error context only)
    43
    44
    45	def _preload() -> list[str]:
    46	    """Import the heavy libraries the leaves share. A missing one is fine (a layer's venv
    47	    may not carry it) — the child pays its own import then, exactly like today."""
    48	    loaded = []
    49	    names = os.environ.get("METIS_FORKSERVER_PRELOAD", _DEFAULT_PRELOAD)
    50	    for name in filter(None, (n.strip() for n in names.split(","))):
    51	        try:
    52	            importlib.import_module(name)
    53	            loaded.append(name)
    54	        except ImportError:
    55	            print(f"forkserver: preload {name!r} not importable (skipped)", file=sys.stderr)
    56	    return loaded
    57
    58
    59	def _child(req: dict) -> "None":  # never returns — os._exit
    60	    code = 1
    61	    try:
    62	        # The request env is AUTHORITATIVE for METIS_*: scrub first so an absent key (e.g.
    63	        # READ_ROOT on an unconfined run) is genuinely absent, mirroring exec.go's strip.
    64	        for k in [k for k in os.environ if k.startswith("METIS_")]:
    65	            del os.environ[k]
    66	        os.environ.update(req.get("env") or {})
    67	        os.chdir(req["cwd"])
    68	        from metis.trace import run_traced  # already imported in the server; cheap here
    69
    70	        run_traced(req["module"], force_site_packages=True)
    71	        code = 0
    72	    except SystemExit as e:  # a step's deliberate exit code passes through
    73	        c = e.code
    74	        code = c if isinstance(c, int) else (0 if c is None else 1)
    75	    except BaseException:
    76	        traceback.print_exc()
    77	        code = 1
    78	    finally:
    79	        try:
    80	            sys.stdout.flush()
    81	            sys.stderr.flush()
    82	        except Exception:
    83	            pass
    84	        os._exit(code)
    85
    86
    87	def _wait(req_id, pid: int, rfd: int, lock: threading.Lock, out) -> None:
    88	    """Waiter thread: drain the child's combined-output pipe to EOF, reap it, emit the
    89	    response line. IO only — never forks, never touches os.environ."""
    90	    chunks = []
    91	    with os.fdopen(rfd, "rb") as fh:
    92	        while True:
    93	            b = fh.read(65536)
    94	            if not b:
    95	                break
    96	            chunks.append(b)
    97	    _, status = os.waitpid(pid, 0)
    98	    exit_code = os.waitstatus_to_exitcode(status)
    99	    output = b"".join(chunks).decode("utf-8", "replace")
   100	    if len(output) > _OUTPUT_CAP:
   101	        output = output[-_OUTPUT_CAP:]
   102	    with lock:
   103	        out.write(json.dumps({"id": req_id, "exit": exit_code, "output": output}) + "\n")
   104	        out.flush()
   105
   106
   107	def serve() -> None:
   108	    out = sys.stdout
   109	    lock = threading.Lock()
   110	    loaded = _preload()
   111	    with lock:
   112	        out.write(json.dumps({"ready": True, "preloaded": loaded}) + "\n")
   113	        out.flush()
   114	    waiters: list[threading.Thread] = []
   115	    for line in sys.stdin:
   116	        line = line.strip()
   117	        if not line:
   118	            continue
   119	        req = json.loads(line)  # a malformed line is a protocol bug — fail loud, not skip
   120	        r, w = os.pipe()
   121	        # C1 (close review): fork UNDER the protocol lock. Waiter threads write responses to
   122	        # `out` only while holding `lock`, so holding it across fork() guarantees stdout's
   123	        # internal BufferedWriter/TextIOWrapper locks are FREE in the child — otherwise a
   124	        # fork landing mid-write copies a held io lock into the child (whose owner thread
   125	        # doesn't exist there) and the child deadlocks at its first stdout use, hanging the
   126	        # whole run. The child never touches `lock` itself.
   127	        with lock:
   128	            pid = os.fork()
   129	        if pid == 0:  # ---- child
   130	            os.close(r)
   131	            os.dup2(w, 1)
   132	            os.dup2(w, 2)
   133	            os.close(w)
   134	            # Belt-and-braces for C1: rebind std streams onto fresh objects over the duped
   135	            # fds so NO parent buffer/lock state is reachable in the child at all.
   136	            sys.stdout = io.TextIOWrapper(io.FileIO(1, "wb", closefd=False), line_buffering=True)
   137	            sys.stderr = io.TextIOWrapper(io.FileIO(2, "wb", closefd=False), line_buffering=True)
   138	            # D symmetry: the child inherits metis.forkserver in sys.modules (first-party →
   139	            # it would land in every forkserver-run step's read-set D, making cache keys
   140	            # differ between executors for the same step). The step never imports it — drop
   141	            # it before the trace snapshot.
   142	            sys.modules.pop("metis.forkserver", None)
   143	            _child(req)  # never returns
   144	        # ---- parent
   145	        os.close(w)
   146	        t = threading.Thread(target=_wait, args=(req["id"], pid, r, lock, out), daemon=True)
   147	        t.start()
   148	        waiters.append(t)
   149	    for t in waiters:  # stdin EOF: drain in-flight children before exiting
   150	        t.join()
   151
   152
   153	if __name__ == "__main__":
   154	    serve()
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
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
	// metis#44: one warm fork-server pool per top-level run, shut down (EOF-drain) when the
	// run ends. Only the production executor uses it (an injected test exec bypasses execStep).
	// Constructed AFTER the writer wrap — its fallback notices must route through the board.
	if o.forkserver && o.exec == nil && o.forkPool == nil {
		o.forkPool = newServerPool(out)
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
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
atlas/experiment.md

codex
```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The implementation cleanly centralizes the ambient-wins rule, threads its result through both production spawn paths, and documents the new behavior in the atlas and CLI help. No correctness defect is apparent from static review. Before closing, the tracker needs verifiable evidence for the claimed external RUNBOOK update; it is required by the Spec/Plan but absent from this review window. Test execution was blocked by the read-only environment, so confidence is medium.

1. Strengths

- `blasPins` is a genuinely pure, deterministic core with exact-name matching, stable ordering, and correct empty/non-nil sentinel behavior ([cmd/metis/blaspins.go:15](/Users/xianxu/workspace/metis/cmd/metis/blaspins.go:15)).
- Both execution seams consume the same computed pins: legacy child environment at [cmd/metis/exec.go:131](/Users/xianxu/workspace/metis/cmd/metis/exec.go:131), and fork-server process environment at [cmd/metis/forkexec.go:115](/Users/xianxu/workspace/metis/cmd/metis/forkexec.go:115).
- Once-per-run computation is correctly placed after writer wrapping and before pool creation, preserving board routing and fork-server startup semantics ([cmd/metis/run.go:152](/Users/xianxu/workspace/metis/cmd/metis/run.go:152)).
- The default executor and deliberate `select --promote` bypass were enumerated and documented, avoiding an accidental unpinned production path.
- The atlas and `--parallel` help accurately describe the new default and override mechanism.

2. Critical findings

None.

3. Important findings

- Requirements traceability: [workshop/issues/000048-default-leaf-blas-pins.md:37](/Users/xianxu/workspace/metis/workshop/issues/000048-default-leaf-blas-pins.md:37) and the durable plan require the kbench RUNBOOK rewrite, but no RUNBOOK change or exact peer commit reference appears in the reviewed metis window. Do not assume the external edit occurred from the checked issue box. Before closing, verify that peer change and record its repository/commit in the issue Log; otherwise reopen that checklist item.

4. Minor findings

None.

5. Test coverage notes

- Unit tests cover bare ambient state, operator override, exact-name matching, and empty/non-nil behavior.
- Integration tests exercise the real legacy subprocess seam and a skip-guarded real `uv` fork-server child.
- The full-chain test verifies note cardinality, note contents, injected defaults, and operator passthrough.
- The full-chain fixture is a plain experiment rather than a sweep, but the sweep propagation is structurally shared through copied `runOpts`; the recorded real-sweep smoke provides complementary operational evidence.
- I could not execute tests: Go failed creating its build directory because this review environment is read-only. The reported `go test ./... -race` result therefore remains independently unverified here.

6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** one defaults list and one ambient-filtering function feed both spawn seams and the note.
- **ARCH-PURE — pass:** environment inspection and subprocess mutation remain in thin integration code; `blasPins` is directly unit-tested without IO.
- **ARCH-PURPOSE — pass:** both `metis run` executor paths are covered. The serial promotion exclusion is consistent with the issue’s oversubscription purpose and is explicitly documented.

7. Plan revision recommendations

None. The plan’s technical claims match the implementation; only the external RUNBOOK completion evidence needs verification and logging.
tokens used
60,228
```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The implementation cleanly centralizes the ambient-wins rule, threads its result through both production spawn paths, and documents the new behavior in the atlas and CLI help. No correctness defect is apparent from static review. Before closing, the tracker needs verifiable evidence for the claimed external RUNBOOK update; it is required by the Spec/Plan but absent from this review window. Test execution was blocked by the read-only environment, so confidence is medium.

1. Strengths

- `blasPins` is a genuinely pure, deterministic core with exact-name matching, stable ordering, and correct empty/non-nil sentinel behavior ([cmd/metis/blaspins.go:15](/Users/xianxu/workspace/metis/cmd/metis/blaspins.go:15)).
- Both execution seams consume the same computed pins: legacy child environment at [cmd/metis/exec.go:131](/Users/xianxu/workspace/metis/cmd/metis/exec.go:131), and fork-server process environment at [cmd/metis/forkexec.go:115](/Users/xianxu/workspace/metis/cmd/metis/forkexec.go:115).
- Once-per-run computation is correctly placed after writer wrapping and before pool creation, preserving board routing and fork-server startup semantics ([cmd/metis/run.go:152](/Users/xianxu/workspace/metis/cmd/metis/run.go:152)).
- The default executor and deliberate `select --promote` bypass were enumerated and documented, avoiding an accidental unpinned production path.
- The atlas and `--parallel` help accurately describe the new default and override mechanism.

2. Critical findings

None.

3. Important findings

- Requirements traceability: [workshop/issues/000048-default-leaf-blas-pins.md:37](/Users/xianxu/workspace/metis/workshop/issues/000048-default-leaf-blas-pins.md:37) and the durable plan require the kbench RUNBOOK rewrite, but no RUNBOOK change or exact peer commit reference appears in the reviewed metis window. Do not assume the external edit occurred from the checked issue box. Before closing, verify that peer change and record its repository/commit in the issue Log; otherwise reopen that checklist item.

4. Minor findings

None.

5. Test coverage notes

- Unit tests cover bare ambient state, operator override, exact-name matching, and empty/non-nil behavior.
- Integration tests exercise the real legacy subprocess seam and a skip-guarded real `uv` fork-server child.
- The full-chain test verifies note cardinality, note contents, injected defaults, and operator passthrough.
- The full-chain fixture is a plain experiment rather than a sweep, but the sweep propagation is structurally shared through copied `runOpts`; the recorded real-sweep smoke provides complementary operational evidence.
- I could not execute tests: Go failed creating its build directory because this review environment is read-only. The reported `go test ./... -race` result therefore remains independently unverified here.

6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** one defaults list and one ambient-filtering function feed both spawn seams and the note.
- **ARCH-PURE — pass:** environment inspection and subprocess mutation remain in thin integration code; `blasPins` is directly unit-tested without IO.
- **ARCH-PURPOSE — pass:** both `metis run` executor paths are covered. The serial promotion exclusion is consistent with the issue’s oversubscription purpose and is explicitly documented.

7. Plan revision recommendations

None. The plan’s technical claims match the implementation; only the external RUNBOOK completion evidence needs verification and logging.
