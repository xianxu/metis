# Boundary Review — 000043-leaf-sched-depth-first#43 (whole-issue close)

| field | value |
|-------|-------|
| issue | 43 — leaf scheduler: depth-first run priority so cold-cache sweeps reach trains early |
| repo | 000043-leaf-sched-depth-first |
| issue file | workshop/issues/000043-leaf-sched-depth-first.md |
| boundary | whole-issue close |
| milestone | — |
| window | 17e93ec83aa4cc8a7777e95659b94ad9c9750e2a..HEAD |
| command | sdlc close --issue 43 |
| reviewer | codex |
| timestamp | 2026-07-16T17:41:02-07:00 |
| verdict | SHIP |

## Review

Reading additional input from stdin...
OpenAI Codex v0.144.5
--------
workdir: /Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first
model: gpt-5.6-sol
provider: openai
approval: never
sandbox: read-only
reasoning effort: none
reasoning summaries: none
session id: 019f6d83-bc4d-7ad1-9379-2cbb4babf3b0
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **000043-leaf-sched-depth-first** repository.

- repository: 000043-leaf-sched-depth-first   (root: /Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first)
- issue:      000043-leaf-sched-depth-first#43   (file: workshop/issues/000043-leaf-sched-depth-first.md)
- window:     Base: 17e93ec83aa4cc8a7777e95659b94ad9c9750e2a   Head: HEAD

Review the **000043-leaf-sched-depth-first** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
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
index 8e82dcf..18d8b0a 100644
--- a/atlas/index.md
+++ b/atlas/index.md
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
index 804c462..67ba9e9 100644
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
index bd26ca2..c71ede6 100644
--- a/cmd/metis/run.go
+++ b/cmd/metis/run.go
@@ -82,15 +82,20 @@ type runOpts struct {
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
-	leafPins  []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
+	board           *boardWriter      // metis#38: the pin-bottom compositor (set by runExperiment in board mode)
+	boardTick       <-chan time.Time  // test seam: nil uses the production 500ms ticker
+	beforeBoardTick func()            // test seam: after tick selection, before health observation
+	afterBoardTick  func()            // test seam: after the health observation returns
+	leafGauge       func() (int, int) // metis#38: (busy, capacity) over leafSem — the board's leaves line
+	leafPins        []string          // metis#48: default leaf BLAS pins, computed ONCE per top-level run in
 	//                             runExperiment (nil = not yet computed; non-nil rides nested runOpts
 	//                             copies like forkPool — an all-suppressed result is empty, not nil)
 }
@@ -177,6 +182,9 @@ func runExperiment(o runOpts) (experiment.Run, error) {
 		if err := experiment.ValidateShape(sh); err != nil {
 			return experiment.Run{}, fmt.Errorf("%s: %w", o.expPath, err)
 		}
+		if o.runControl == nil {
+			o.runControl = newRunControl(o.maxParallel)
+		}
 		return experiment.Run{}, runShapeSweep(o, sh, now, out)
 	}
 	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
@@ -205,6 +213,15 @@ func singleRunID(o runOpts, exp experiment.Experiment, now func() time.Time) str
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
diff --git a/cmd/metis/sweep.go b/cmd/metis/sweep.go
index d1e673a..12a8504 100644
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
@@ -93,10 +91,24 @@ type shapeSweep struct {
 
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
@@ -116,44 +128,37 @@ type sweepPass struct {
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
@@ -166,11 +171,15 @@ func (ss *shapeSweep) runSweeper(ctx sampler.Ctx, configPts []shape.Point, pass
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
@@ -180,7 +189,7 @@ func (ss *shapeSweep) runSweeper(ctx sampler.Ctx, configPts []shape.Point, pass
 // the winner by the objective; driver:single ships it (M1a-5). Produces per-config (mean,SE)
 // + the manifest + the raw per-fold ledger. Per-fold failure is fatal to the sweep (surfaced,
 // not swallowed — a partial resample is not an honest estimate).
-func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) error {
+func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.Writer) (result error) {
 	sweepStart := now() // metis#50: the run-end summary reports wall-clock elapsed
 	// probeRepo's HEAD sha still drives the mid-sweep code-freeze guard (codeID) — NOT the
 	// identity (metis#27 dropped repo_shas). The shape's blob-hash content-addresses the intent.
@@ -262,19 +271,41 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
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
@@ -293,7 +324,10 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
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
@@ -301,24 +335,22 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
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
 	cohort, err := captureSweepCode(o, ss.man)
 	if err != nil {
-		fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
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
@@ -326,11 +358,13 @@ func runShapeSweep(o runOpts, sh experiment.Shape, now func() time.Time, out io.
 	// the ship/report is gated. Checked here (post-fold) because HasComplexity is only known
 	// after the folds run.
 	if err := sampler.GuardComplexity(sh.Sweeper.Objective.Select, configStatsOf(ss.configs)); err != nil {
-		return err
+		return ss.fail("sweep complexity guard", err)
 	}
-	ss.reportWinner(res)
-	printRunSummary(out, o.expPath, now().Sub(sweepStart), len(ss.man.Points), cohort)
-	return nil
+	ss.whileHealthy(func() {
+		ss.reportWinner(res)
+		printRunSummary(out, o.expPath, now().Sub(sweepStart), len(ss.man.Points), cohort)
+	})
+	return ss.firstError()
 }
 
 // configStatsOf builds the per-config stats (with each config's family) from a completed
@@ -372,54 +406,38 @@ func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, r
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
@@ -427,18 +445,22 @@ func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, r
 	// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
 	sortPointRuns(ss.man.Points)
 	if err := writeManifest(ss.o.expPath, ss.man); err != nil {
-		return err
+		return ss.fail("write nested sweep manifest", err)
 	}
 	cohort, cerr := captureSweepCode(ss.o, ss.man)
 	if cerr != nil {
-		fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
+		ss.whileHealthy(func() {
+			fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
+		})
 	}
 	if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
-		return err
+		return ss.fail("write nested sweep ledger", err)
 	}
-	ss.reportEstimate(est, runFolds)
-	printRunSummary(ss.out, ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
-	return nil
+	ss.whileHealthy(func() {
+		ss.reportEstimate(est, runFolds)
+		printRunSummary(ss.out, ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
+	})
+	return ss.firstError()
 }
 
 // materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
@@ -457,13 +479,14 @@ func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]str
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
@@ -479,7 +502,7 @@ func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]str
 func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k int, stratify bool, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
 	analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
 	if err != nil {
-		return 0, err
+		return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
 	}
 	// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
 	pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
@@ -487,22 +510,26 @@ func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k i
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
@@ -516,44 +543,51 @@ func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k i
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
@@ -580,8 +614,9 @@ func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
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
@@ -595,34 +630,39 @@ func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.
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
diff --git a/workshop/plans/000043-leaf-sched-depth-first-plan.md b/workshop/plans/000043-leaf-sched-depth-first-plan.md
index f5cb13c..afea65e 100644
--- a/workshop/plans/000043-leaf-sched-depth-first-plan.md
+++ b/workshop/plans/000043-leaf-sched-depth-first-plan.md
@@ -57,7 +57,7 @@
 - Create: `cmd/metis/runcontrol.go`
 - Create: `cmd/metis/runcontrol_test.go`
 
-- [ ] **Step 1: Write failing tests for capacity, first-error authority, and publication-before-release**
+- [x] **Step 1: Write failing tests for capacity, first-error authority, and publication-before-release**
 
 Define bounded helpers in the test file so a broken controller fails locally instead of waiting for
 Go's global test timeout:
@@ -245,13 +245,13 @@ func TestRunControlLateSuccessBecomesAborted(t *testing.T) {
 
 The shared `controlResult` type and bounded helpers keep every blocking assertion finite.
 
-- [ ] **Step 2: Run the focused tests and verify RED**
+- [x] **Step 2: Run the focused tests and verify RED**
 
 Run: `go test ./cmd/metis -run '^TestRunControl' -count=1`
 
 Expected: FAIL to compile because `runControl`, `newRunControl`, and `errRunAborted` do not exist.
 
-- [ ] **Step 3: Implement the minimal controller**
+- [x] **Step 3: Implement the minimal controller**
 
 Create `cmd/metis/runcontrol.go` with this contract:
 
@@ -319,13 +319,13 @@ on the same controller. Orchestration parents remain outside admission, so they
 slot while waiting for child runs. Do not use `context.Context`: there is no interruptible
 subprocess contract in scope, and admitted work is allowed to drain.
 
-- [ ] **Step 4: Run unit tests under the race detector and verify GREEN**
+- [x] **Step 4: Run unit tests under the race detector and verify GREEN**
 
 Run: `go test ./cmd/metis -run '^TestRunControl' -race -count=20`
 
 Expected: PASS on all 20 repetitions with no race report or hang.
 
-- [ ] **Step 5: Commit the primitive**
+- [x] **Step 5: Commit the primitive**
 
 ```bash
 git add cmd/metis/runcontrol.go cmd/metis/runcontrol_test.go
@@ -343,7 +343,7 @@ git commit -m "#43: add whole-run admission control" -m "Bound admitted concrete
 - Modify: `cmd/metis/sweep.go:440-570`
 - Test: `cmd/metis/runcontrol_test.go`
 
-- [ ] **Step 1: Add a failing boundary test**
+- [x] **Step 1: Add a failing boundary test**
 
 Install a controller that already holds an error, call `runResolvedExperiment` with a buffer and a temporary experiment directory, and assert no side effect precedes its cancellation check:
 
@@ -371,7 +371,7 @@ Run: `go test ./cmd/metis -run '^TestRunResolvedExperiment_AbortedBeforeSideEffe
 
 Expected: FAIL because `runOpts` and `runResolvedExperiment` do not consult the controller.
 
-- [ ] **Step 2: Thread control and contextual labels through `runOpts`**
+- [x] **Step 2: Thread control and contextual labels through `runOpts`**
 
 Add:
 
@@ -382,7 +382,7 @@ runLabel   string      // config/fold/preamble context captured with the first e
 
 After strict shape validation and before `runShapeSweep`, assign `o.runControl = newRunControl(o.maxParallel)` only when it is nil. The non-nil path is an injected integration-test seam; production creates exactly one controller here. Leave plain one-point experiments unchanged: they have no fan-out and no controller.
 
-- [ ] **Step 3: Split the wrapper from the admitted body**
+- [x] **Step 3: Split the wrapper from the admitted body**
 
 Keep the call-site name and move the current implementation byte-for-byte into `runResolvedExperimentAdmitted`:
 
@@ -399,7 +399,7 @@ func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, n
 
 The wrapper executes before `filepath.Abs`, cache-directory creation, the run-start line, or executor construction. Its callback contains one concrete run only and never calls another admitted run, satisfying Chunk 1's non-reentrant contract.
 
-- [ ] **Step 4: Give every concrete sweep call a contextual label**
+- [x] **Step 4: Give every concrete sweep call a contextual label**
 
 Set the label on each copied options value immediately before the call:
 
@@ -411,13 +411,13 @@ preambleOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
 
 Pass `fam` into `scoreOnOuterFold` so the label is complete at the boundary. Do not wrap a controller-returned error again at a caller; the stored first error is already contextual.
 
-- [ ] **Step 5: Run boundary and existing run tests**
+- [x] **Step 5: Run boundary and existing run tests**
 
 Run: `go test ./cmd/metis -run 'TestRunResolvedExperiment|TestRunExperiment' -race -count=1`
 
 Expected: PASS; the new abort test proves no output or directory creation before admission.
 
-- [ ] **Step 6: Commit the concrete-run boundary**
+- [x] **Step 6: Commit the concrete-run boundary**
 
 ```bash
 git add cmd/metis/run.go cmd/metis/runcontrol_test.go cmd/metis/sweep.go
@@ -438,7 +438,7 @@ git commit -m "#43: admit sweep runs at the concrete boundary" -m "Acquire befor
 - Test: `cmd/metis/runcontrol_test.go`
 - Test: `cmd/metis/parallel_test.go`
 
-- [ ] **Step 1: Write a failing nested-sibling cancellation test**
+- [x] **Step 1: Write a failing nested-sibling cancellation test**
 
 Add a `failureBarrierExec` for `foldShapeCVMD("[a, b, c]")` with `maxParallel=2` (admission capacity four). The fake records each distinct inner `runDir` on its first step. Once four inner runs have entered, it lets exactly one `train` return `errors.New("injected train failure")`; the other admitted trains wait on `failurePublished`, which the controller's winning-publication hook closes while admission is still held. Every wait is bounded, and the top-level `runExperiment` executes in a goroutine observed through `recvWithin`.
 
@@ -515,7 +515,7 @@ Run: `go test ./cmd/metis -run '^TestNestedCV_FirstFailureStopsQueuedSiblingPass
 
 Expected: FAIL because the current per-pass and outer error latches do not stop sibling passes.
 
-- [ ] **Step 2: Replace pass-local and outer-local error ownership with `runControl`**
+- [x] **Step 2: Replace pass-local and outer-local error ownership with `runControl`**
 
 Give `shapeSweep` helpers:
 
@@ -529,7 +529,7 @@ func (ss *shapeSweep) whileHealthy(fn func()) bool { return ss.o.runControl.whil
 
 Make `sweepPass.setErr` delegate to `ss.fail`; make `sweepPass.firstError` delegate to `ss.firstError`; remove `sweepPass.err`; retain its mutex only for `configs` and `points`. Delete `runNestedCV`'s separate `errMu` / `firstErr` closures and use the shape-wide helpers for orchestration errors.
 
-- [ ] **Step 3: Add an atomic healthy-observation operation**
+- [x] **Step 3: Add an atomic healthy-observation operation**
 
 Add the following controller method and a unit test that holds the observation callback open, starts `fail` concurrently, proves `fail` cannot return until the observation exits, then proves a later observation is rejected:
 
@@ -576,7 +576,7 @@ func TestRunControlObservationLinearizesWithFailure(t *testing.T) {
 }
 ```
 
-- [ ] **Step 4: Gate sampler callbacks and accumulators without changing batch shape**
+- [x] **Step 4: Gate sampler callbacks and accumulators without changing batch shape**
 
 In `runSweeper`, wrap hooks explicitly:
 
@@ -591,7 +591,7 @@ configHook := func(ev sampler.ProgressEvent[shape.Point, sampler.MeanSE]) {
 
 Implement `addConfigScore`, `addPoint`, and `addManPoints` as `whileHealthy` callbacks that acquire their existing mutex only after `control.mu`. Never acquire a pass/progress/manifest mutex and then call the controller. This fixed order makes each append linearize entirely before failure or not occur at all. Zero values may still occupy cancelled batch indices, but no observable sink consumes them.
 
-- [ ] **Step 5: Gate every transition to further work or persistence**
+- [x] **Step 5: Gate every transition to further work or persistence**
 
 Add first-error checks:
 
@@ -640,7 +640,7 @@ Never call the controller while holding `sp.mu` or `bw.mu`; ticker gating acquir
 chosen. This cleanup applies to every error return after board wiring, including non-sampler
 orchestration failures.
 
-- [ ] **Step 6: Prove the TUI cannot repaint stale progress after publication**
+- [x] **Step 6: Prove the TUI cannot repaint stale progress after publication**
 
 First add a direct `boardWriter` unit test: paint a frame containing `folds/min` and `ETA`, call
 `discardFrame`, then call `close`. Assert the suffix beginning at discard contains the erase and
@@ -664,11 +664,11 @@ score, estimate, or stored board row. ANSI erase/synchronized-output/cursor-rest
 pending ordinary error output are permitted. Keep all waits bounded with `recvWithin`; run under
 `-race` so writer snapshots, abort, tick rejection, and close are checked concurrently.
 
-- [ ] **Step 7: Prove orchestration never holds admission while awaiting a child**
+- [x] **Step 7: Prove orchestration never holds admission while awaiting a child**
 
 Extend the nested test with a two-second top-level timeout and `maxParallel=2`. A nested shape has more than four children, so completion proves outer/config sampler closures remain outside `runControl.run`; if any parent acquired before synchronously awaiting a child, the test would exhaust all four slots and time out.
 
-- [ ] **Step 8: Run cancellation, board-abort, and nested tests repeatedly under race**
+- [x] **Step 8: Run cancellation, board-abort, and nested tests repeatedly under race**
 
 Run: `go test ./cmd/metis -run 'TestRunControlObservation|TestBoardWriter_DiscardFrame|TestNestedCV_FirstFailure|TestNestedCV_PeakConcurrency|TestSweep_ProbeFailure' -race -count=20`
 
@@ -678,7 +678,7 @@ This is Chunk 2's focused integration proof; Chunk 3 adds the cold early-complet
 serial/parallel artifact and run-record comparisons, the full `go test ./... -race`, and the
 disposable real-process smoke before close.
 
-- [ ] **Step 9: Commit global abort integration**
+- [x] **Step 9: Commit global abort integration**
 
 ```bash
 git add cmd/metis/run.go cmd/metis/runcontrol.go cmd/metis/runcontrol_test.go cmd/metis/sweep.go cmd/metis/progress.go cmd/metis/board.go cmd/metis/board_test.go cmd/metis/parallel_test.go
@@ -694,7 +694,7 @@ git commit -m "#43: abort queued runs across nested sweep passes" -m "Use one ex
 - Modify: `cmd/metis/runcontrol_test.go`
 - Modify: `cmd/metis/parallel_test.go:16-175`
 
-- [ ] **Step 1: Add acquire/release observation seams for integration tests**
+- [x] **Step 1: Add acquire/release observation seams for integration tests**
 
 Add two nil-by-default hooks to `runControl`:
 
@@ -718,7 +718,7 @@ if c.slots != nil {
 
 These are observation-only test seams: callbacks must not call controller methods or block production. Extend `TestRunControlBoundsAdmission` to count hook acquisitions/releases and assert both equal the attempted run count.
 
-- [ ] **Step 2: Write the deterministic cold-wave regression**
+- [x] **Step 2: Write the deterministic cold-wave regression**
 
 Add a shared trace recorder and executor:
 
@@ -780,7 +780,7 @@ func TestSweep_ColdAdmissionCompletesTrainBeforeFifthRunStarts(t *testing.T) {
 
 Because the fifth acquire hook cannot fire until a token is released, and token release occurs only after the admitted concrete run returns, this test deterministically pins the desired ordering without sleeps.
 
-- [ ] **Step 3: Extend the nested test to inspect both concurrency budgets**
+- [x] **Step 3: Extend the nested test to inspect both concurrency budgets**
 
 Inject a controller into `TestNestedCV_PeakConcurrencyWithinCap`. Its acquire/release hooks update `activeRuns`, `peakRuns`, `acquiredRuns`, and `releasedRuns` under the test mutex while the existing `peakExec` tracks leaves. Run `runExperiment` in a goroutine and use `recvWithin` for a two-second deadlock bound. Assert:
 
@@ -793,7 +793,7 @@ if acquiredRuns != releasedRuns { t.Fatalf("acquired=%d released=%d", acquiredRu
 
 Keep the unit controller test's exact `peak == 2n` assertion; the nested integration assertion is an upper bound because fast fake runs need not saturate all slots.
 
-- [ ] **Step 4: Strengthen serial/parallel determinism to include run semantics**
+- [x] **Step 4: Strengthen serial/parallel determinism to include run semantics**
 
 Extend `TestSweep_ParallelEqualsSerial` so its helper loads every `runs/*/run.json` and `runs/*/record.json` into maps keyed by run ID. Normalize only `Started` and `Finished`, which are timing-bearing:
 
@@ -831,13 +831,13 @@ func semanticRecords(t *testing.T, ws string) map[string]record.RunRecord {
 
 Return both maps beside ledger/manifest bytes and assert `reflect.DeepEqual` for serial versus parallel runs and records. Keep the existing byte comparisons unchanged.
 
-- [ ] **Step 5: Run the complete scheduling acceptance subset**
+- [x] **Step 5: Run the complete scheduling acceptance subset**
 
 Run: `go test ./cmd/metis -run 'TestRunControl|TestSweep_ColdAdmission|TestSweep_ParallelEqualsSerial|TestNestedCV_PeakConcurrency|TestNestedCV_FirstFailure' -race -count=10`
 
 Expected: PASS ten times with no race, timeout, admission leak, post-failure output, or deterministic-output difference.
 
-- [ ] **Step 6: Commit the scheduling regressions**
+- [x] **Step 6: Commit the scheduling regressions**
 
 ```bash
 git add cmd/metis/runcontrol.go cmd/metis/runcontrol_test.go cmd/metis/parallel_test.go
@@ -851,7 +851,7 @@ git commit -m "#43: pin bounded depth-first sweep scheduling" -m "Prove early tr
 - Modify: `atlas/index.md:84-113`
 - Modify: `workshop/issues/000043-leaf-sched-depth-first.md`
 
-- [ ] **Step 1: Add the smallest real subprocess sweep fixture**
+- [x] **Step 1: Add the smallest real subprocess sweep fixture**
 
 Create a three-config, two-fold shape. `test/echo` is a process-level data-phase adapter whose `out` points at the copied toy dataset; the pipeline itself uses the real `metis/train` Python step:
 
@@ -887,11 +887,11 @@ A disposable real-process nested sweep used to verify whole-run admission withou
 
 Run `go run ./cmd/metis run --dry-run testdata/experiment/toy-sweep-smoke.md`; expected output reports three configs and nested-CV mode.
 
-- [ ] **Step 2: Document both concurrency budgets in the atlas**
+- [x] **Step 2: Document both concurrency budgets in the atlas**
 
 In the `metis#31` execution paragraph, add: parallel sampler fan-out remains order-preserving, but every concrete run crosses one sweep-scoped `2n` admission controller before side effects; the existing leaf semaphore remains `n`; the controller also owns the experiment-wide first failure so queued runs stop without producing observable state. Cite `cmd/metis/runcontrol.go`, `runResolvedExperiment`, and the cancellation regressions.
 
-- [ ] **Step 3: Run the disposable cold real-process smoke**
+- [x] **Step 3: Run the disposable cold real-process smoke**
 
 Run entirely inside a temporary no-hardlink clone so code capture can update only the clone's Metis refs. Snapshot the source checkout and refs, redirect writable build/runtime caches, and clean up on both success and failure:
 
@@ -901,10 +901,24 @@ set -euo pipefail
 source_repo=$(pwd -P)
 source_status=$(git status --porcelain=v1 --untracked-files=all)
 source_refs=$(git for-each-ref --format='%(refname) %(objectname)' refs/metis)
+ariadne_repo=$(cd "$source_repo/../ariadne" && pwd -P)
+ariadne_sha=$(git -C "$ariadne_repo" rev-parse HEAD)
 tmpdir=$(mktemp -d /tmp/metis-43-smoke.XXXXXX)
-trap 'rm -rf "$tmpdir"' EXIT
+smoke_log="$tmpdir/smoke.log"
+cleanup() {
+  rc=$?
+  if [ "$rc" -ne 0 ] && [ -f "$smoke_log" ]; then
+    tail -120 "$smoke_log"
+  fi
+  chmod -R u+w "$tmpdir" 2>/dev/null || true
+  rm -rf "$tmpdir"
+  exit "$rc"
+}
+trap cleanup EXIT
+git clone --local --no-hardlinks "$ariadne_repo" "$tmpdir/ariadne"
+git -C "$tmpdir/ariadne" checkout --detach "$ariadne_sha"
+test "$ariadne_sha" = "$(git -C "$tmpdir/ariadne" rev-parse HEAD)"
 git clone --local --no-hardlinks "$source_repo" "$tmpdir/metis-src"
-cp "$source_repo/testdata/experiment/toy-sweep-smoke.md" "$tmpdir/metis-src/testdata/experiment/"
 cd "$tmpdir/metis-src"
 export GOCACHE="$tmpdir/go-cache"
 export GOMODCACHE="$tmpdir/go-mod-cache"
@@ -926,12 +940,13 @@ awk '
       print "cold ordering failed: first_train=" first_train ", fifth_inner=" fifth_inner > "/dev/stderr"
       exit 1
     }
+    printf "cold ordering passed: first_train_line=%d fifth_cv_split_line=%d\n", first_train, fifth_inner
   }
 ' "$tmpdir/smoke.log"
 completed_trains=$(rg -c '✓ step train' "$tmpdir/smoke.log")
 awk -v completed="$completed_trains" -v seconds="$elapsed" 'BEGIN {
   rate=60*completed/seconds
-  if (completed < 7 || seconds > 600 || rate <= 0) {
+  if (completed < 7 || seconds > 600 || rate <= 0 || rate != rate) {
     print "throughput check failed: completed=" completed ", seconds=" seconds ", trains/min=" rate > "/dev/stderr"
     exit 1
   }
@@ -940,6 +955,7 @@ awk -v completed="$completed_trains" -v seconds="$elapsed" 'BEGIN {
 test "$source_status" = "$(git -C "$source_repo" status --porcelain=v1 --untracked-files=all)"
 test "$source_refs" = "$(git -C "$source_repo" for-each-ref --format='%(refname) %(objectname)' refs/metis)"
 cd "$source_repo"
+chmod -R u+w "$tmpdir" 2>/dev/null || true
 rm -rf "$tmpdir"
 trap - EXIT
 )
@@ -947,7 +963,7 @@ trap - EXIT
 
 Expected: exit 0; the first `awk` proves the first completed train precedes the fifth inner run's first `cv-split` step, followed by the nested estimate and row summary. The second requires all six inner trains plus the one outer-score train, completion within ten minutes, and a finite positive measured trains/min rate. The source worktree and `refs/metis/*` snapshots are byte-identical before/after; clone refs, experiment outputs, Go build/module/telemetry caches, uv cache, XDG cache, and Python bytecode all live under the removed temporary directory. Record the relevant ordering/result lines in the issue Log before cleanup.
 
-- [ ] **Step 4: Run the complete automated verification**
+- [x] **Step 4: Run the complete automated verification**
 
 Run:
 
@@ -960,18 +976,18 @@ git diff --check
 
 Expected: every command exits 0; no race report; no whitespace errors.
 
-- [ ] **Step 5: Update the issue record with evidence**
+- [x] **Step 5: Update the issue record with evidence**
 
 Tick the plain issue-plan checkboxes only after their commands pass. Append a dated Log entry naming the focused race repetitions, full race suite, serial/parallel byte comparison, semantic run-record comparison, and temporary real-process smoke result. Do not add an `M1` marker: this issue has one close-review boundary.
 
-- [ ] **Step 6: Commit docs and final verification artifacts**
+- [x] **Step 6: Commit docs and final verification artifacts**
 
 ```bash
 git add testdata/experiment/toy-sweep-smoke.md atlas/index.md workshop/issues/000043-leaf-sched-depth-first.md
 git commit -m "#43: document bounded sweep admission" -m "Record the independent run and leaf budgets and keep a credential-free real-process sweep for cold scheduling smoke checks." -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
 ```
 
-- [ ] **Step 7: Enter the single SDLC close boundary**
+- [x] **Step 7: Enter the single SDLC close boundary**
 
 Run `sdlc close --issue 43 --agent codex --verified '<focused race repetitions; full go test ./... -race; byte/semantic determinism; disposable real-process smoke>'` and address every Critical/Important finding from the gate-owned fresh review before proceeding to PR/merge.
 
@@ -990,3 +1006,12 @@ Run `sdlc close --issue 43 --agent codex --verified '<focused race repetitions;
   assertion to available samples; the direct compositor test retains the explicit stale-ETA proof.
 - Expanded Chunk 2's files, focused race command, and commit boundary to cover `run.go`,
   `progress.go`, `board.go`, and `board_test.go`.
+
+### 2026-07-16 — cold-smoke peer pin correction
+
+- Made the disposable smoke resolve the declared `../ariadne` replacement from the source checkout,
+  snapshot its exact HEAD, and no-hardlink clone Ariadne first into the sibling path expected by the
+  cloned Metis module. The recipe checks out and verifies that detached peer commit before building.
+- Removed the uncommitted-fixture copy assumption now that the fixture is durable, and made cleanup
+  writable-cache-safe while preserving source status/ref, ordering, throughput, and elapsed-time
+  assertions.


codex
I’ll verify the issue/plan, repository instructions, and the actual base-to-HEAD diff, then run targeted read-only tests where feasible.
exec
/bin/zsh -lc "pwd && rg --files -g 'AGENTS.md' -g 'workshop/issues/000043-leaf-sched-depth-first.md' -g 'workshop/plans/000043-leaf-sched-depth-first-plan.md' && git diff --stat 17e93ec83aa4cc8a7777e95659b94ad9c9750e2a..HEAD && git status --short" in /Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first
 succeeded in 0ms:
/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first
workshop/issues/000043-leaf-sched-depth-first.md
workshop/plans/000043-leaf-sched-depth-first-plan.md
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead
 atlas/index.md                                     |   8 +-
 cmd/metis/board.go                                 |  13 +
 cmd/metis/board_test.go                            | 106 +++++
 cmd/metis/parallel_test.go                         | 420 ++++++++++++++++-
 cmd/metis/progress.go                              |  15 +-
 cmd/metis/run.go                                   |  23 +-
 cmd/metis/runcontrol.go                            | 104 +++++
 cmd/metis/runcontrol_test.go                       | 508 +++++++++++++++++++++
 cmd/metis/sweep.go                                 | 276 ++++++-----
 testdata/experiment/toy-sweep-smoke.md             |  27 ++
 workshop/issues/000043-leaf-sched-depth-first.md   |  61 ++-
 .../plans/000043-leaf-sched-depth-first-plan.md    |  97 ++--
 12 files changed, 1480 insertions(+), 178 deletions(-)
git: warning: confstr() failed with code 5: couldn't get path of DARWIN_USER_TEMP_DIR; using /tmp instead

exec
/bin/zsh -lc 'for f in ../AGENTS.md ../../AGENTS.md ../../../AGENTS.md ../../../../AGENTS.md; do if test -f "$f"; then echo "$f"; sed -n '"'1,260p' \""'$f"; fi; done
sed -n '"'1,260p' workshop/issues/000043-leaf-sched-depth-first.md
sed -n '1,180p' workshop/plans/000043-leaf-sched-depth-first-plan.md" in /Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first
 succeeded in 0ms:
---
id: 000043
status: working
deps: []
github_issue:
created: 2026-07-14
updated: 2026-07-16
estimate_hours: 4.74
started: 2026-07-16T12:57:07-07:00
---

# leaf scheduler: depth-first run priority so cold-cache sweeps reach trains early

## Problem

On a COLD cache, a nested sweep's ~2k run closures all start concurrently and the shared leaf
semaphore serves steps roughly FIFO — so a run's `features` queues behind every other run's
`cv-split`, and execution degenerates into phase WAVES: all cv-splits, then all features, then
all trains (observed on the kbench#9 decision run 2026-07-14: 2,160/2,160 cv-splits done,
1,315 features, 0 trains after ~10 min — looked like a hang; zero records complete until wave 3,
so run-level progress reads as frozen). On a warm cache the waves collapse and trains appear
early, which is why this wasn't seen before. Depth-first (finish admitted runs before admitting
new ones) reaches complete records ~immediately, makes progress meaningful, and bounds
in-flight run state; it also pairs with #38's moving-average runs/sec board (wave 1-2 have zero
"runs/sec" by construction).

## Spec

(at claim) Options: bound the number of ADMITTED run closures (e.g. 2×parallel) with a run-level
semaphore above the leaf semaphore (simplest — preserves leaf fairness within admitted runs);
or a priority leaf queue keyed by run progress. Keep byte-determinism of artifacts/ledger
(ordering already normalized by sortPointRuns).

### Approved design — bounded whole-run admission

Use a distinct, shared **run-admission gate** in addition to the existing leaf-subprocess
semaphore:

- `runExperiment` creates the gate once when `maxParallel > 1`, with capacity
  `2 × maxParallel`, and threads it through copied `runOpts`. Serial runs have no gate. There is
  no second CLI/config knob: `--parallel` remains the one operator-facing budget.
- Every concrete `runResolvedExperiment` acquires admission before printing the run-start line,
  creating run directories/records, or executing a step, and releases with `defer` on every return.
  This is the single choke point shared by inner fold runs, outer scoring, preamble work, and future
  concrete runs (ARCH-DRY, ARCH-PURPOSE).
- The gate shares one **runExperiment-scoped abort latch** across every concrete run in the flat or
  nested sweep; it is not scoped to an individual outer-fold pass. Each call supplies its contextual
  run label (config/fold, outer score, or preamble). The admitted wrapper checks the latch before any
  run side effect, executes the concrete run, and publishes its final contextual error to the
  set-once latch **before** releasing admission. Thus a queued sibling cannot acquire in the gap
  between failure and publication; the first contextual error remains authoritative globally.
- A run that acquires after failure returns a typed `errRunAborted` without logs, files, records, or
  activity events. `sampler.Run` may still require a shape-preserving zero value for that batch slot,
  but it is explicitly non-authoritative: fold/config/driver progress hooks and config/point
  accumulation check the shared latch; each sampler level checks it before further scoring,
  persistence, or reporting; and the top-level result is discarded in favor of the original error.
  Tests pin that aborted slots affect no persisted rows, displayed score/estimate, completion count,
  or throughput signal.
- The run gate and leaf semaphore stay independent. No outer/config orchestration closure holds
  admission while synchronously awaiting a child that also needs admission. A concrete admitted run
  may wait for leaf capacity, but a leaf holder never waits for run admission, so the lock order is
  acyclic and nested fan-out cannot semaphore-deadlock (ARCH-PURE).
- Admission order is deliberately not a stable artifact/API. The guarantee is bounded whole-run
  breadth and early completions, not a byte-defined goroutine schedule. `ParExec` continues returning
  outputs by input index; `Run.Tell` stays proposal-ordered; completion-order side records remain
  sorted before persistence. Serial and parallel manifests/ledgers must remain byte-identical.
- Capacity `2n` is the utilization hedge: admitted runs can be between steps or serving cache hits
  while the `n` leaf slots remain available to useful subprocess work. A priority leaf scheduler is
  explicitly out: it leaves all run state admitted and adds starvation/cancellation policy without
  better serving the observed cold-wave problem.

## Done when

- With parallelism `n`, no more than `2n` concrete runs are admitted globally across nested outer
  folds/configs/inner folds, while subprocess occupancy remains capped at `n` by the existing leaf
  semaphore.
- A cold scripted sweep reaches a completed train/inner-CV run before the first step has started for
  every run (the old phase-wave behavior fails this test).
- A first run failure prevents queued-not-admitted runs anywhere in the experiment from creating
  state or executing steps, publishes the contextual error before admission release, and surfaces
  that original error without bogus rows, counters, throughput, score, or estimate.
- In TUI mode, failure stops ticker-driven progress observations and atomically erases the pinned
  board without retaining a frame that deferred close could redraw; only pending ordinary output
  and terminal cleanup escapes after publication.
- Nested execution completes without deadlock. Serial versus admitted-parallel runs persist
  byte-identical deterministic manifests and ledger rows; timing-bearing run records are
  semantically equal apart from their expected timestamps/durations.
- Full `go test ./... -race` and a disposable cold real-sweep smoke show early inner-run progress and
  sane throughput.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: issue-spec design=0.80 impl=0.12
item: greenfield-go-module design=0.30 impl=0.32
item: cross-cutting-refactor design=0.20 impl=0.20
item: cross-cutting-refactor design=0.20 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.10 impl=0.20
item: atlas-docs design=0.06 impl=0.08
item: milestone-review design=0.10 impl=0.20
design-buffer: 0.15
total: 4.74
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The `issue-spec` row includes the paired #43/#49
design and fresh-eyes convergence already performed after claim. The controller is a greenfield
bounded-concurrency concern; the two refactor rows separately cover the concrete-run boundary and
the nested reduction/progress shell; the two smaller-module rows cover deterministic concurrency
regressions and the isolated real-process smoke. Implementation values use v3.1's 40%-of-v2
AI-paired scale, multiplied by 1.5 for novel-but-bounded concurrency. The thorough reviewed plan
earns the 15% design buffer.
The calibration source is currently marked stale by `sdlc estimate-source`, so this derivation
is structurally current but provisionally calibrated.

## Plan

Durable implementation detail: [workshop/plans/000043-leaf-sched-depth-first-plan.md](../plans/000043-leaf-sched-depth-first-plan.md).

- [x] Chunk 1 — build the race-safe `runControl` admission/failure primitive with deterministic handoff tests.
- [x] Chunk 2 — integrate the controller at the concrete-run boundary and linearize every sampler observation/reduction against global failure.
- [x] Chunk 3 — prove early completion, independent nested caps, deterministic records, full race safety, atlas accuracy, and an isolated real-process cold smoke.

## Log

### 2026-07-14

### 2026-07-16 — paired #43/#49 design approved
- Chose bounded whole-run admission at the shared concrete-run choke point over a local fold-only
  gate or priority leaf queue. Co-designed with #49 so the board reports the earlier completions
  truthfully; implementation/review boundaries remain separate (#43 merges first).

### 2026-07-16 — implementation plan approved
- Wrote and independently reviewed all three chunks of the durable #43 plan: controller primitive,
  concrete-run/sampler integration, and scheduling/determinism/real-process verification. Estimate
  derives from the provisional v3.1 calibration; delivery remains one atomic close boundary.

### 2026-07-16 — change-code estimate correction
- The gate's plan-quality judge found the original 1.94h derivation materially under-decomposed the
  concurrency integration and verification surface. Re-derived 4.33h from eight explicit Method A
  work units with a 1.5 novel-but-bounded familiarity multiplier; no scope or architecture changed.

### 2026-07-16 — change-code TUI consumer correction
- The second implementation-entry review found two post-failure display consumers outside the
  sampler reductions: the board ticker and deferred `boardWriter.close`. Added an explicit board
  abort path, controlled-tick race proof, and a ninth Method A unit; the estimate is now 4.74h.

### 2026-07-16 — Chunk 1 controller complete
- Added the optional `2n` admission gate and set-once first-error latch with deterministic tests for
  publication-before-release, serial cancellation, concurrent authority, and late-success discard.
- Focused `-race -count=20`, spec review, and code-quality review passed. Review strengthened the
  suite with mutation-sensitive proofs that admission precedes the first error check and that the
  winner-only publication hook remains inside the controller mutex.

### 2026-07-16 — Task 2 concrete-run boundary complete
- Wrapped the shared concrete runner before path, cache, directory, output, executor, or record side
  effects; all fold, outer-score, and preamble calls now attach complete controller-owned context.
- Focused race tests plus spec and code-quality review passed. A review-driven cache-enabled mutation
  test proves pre-admission cancellation cannot even initialize `.metis-cache`.

### 2026-07-16 — Chunk 2 global abort integration complete
- Replaced pass-local and nested error ownership with the sweep controller, linearized every progress,
  accumulation, scoring, persistence, and report consumer, and preserved the first contextual cause.
- Extended the boundary through the live board: ticker refreshes are gated, joined before cleanup,
  and an error erases and forgets the frame before deferred close. Controlled pre/post-failure tick
  tests and the focused `-race -count=20` suite passed, followed by full spec and quality approval.

### 2026-07-16 — Task 4 scheduling acceptance complete
- Proved a flat cold sweep completes a train before its fifth admission, nested run/leaf peaks remain
  independently capped, every token releases, and serial/parallel manifests, ledgers, runs, and
  records remain byte- or semantically identical as appropriate.
- Review hardened the proof against zero-hook false passes, hook panics, and payload/path identity
  swaps. The scheduling/cancellation subset passed `-race -count=10` and full race verification.

### 2026-07-16 — Task 5 cold real-process acceptance
- Added the credential-free three-config, two-fold `toy-sweep-smoke` fixture; its dry run reported
  nested CV with `2 × (3 configs × 2 inner folds)` and the expected `C=0.5,1,2` grid.
- In a no-hardlink temporary clone with all writable caches redirected below the temporary root, the
  first completed train appeared on log line 25 before the fifth `cv-split` start on line 32. The
  nested estimate appeared on line 60 and the seven-row completion summary on line 62: seven real
  trains finished in 44s (~9.5 trains/min). Source status and `refs/metis/*` were byte-identical
  before and after, and the temporary tree was removed. The isolated build also cloned the declared
  `../ariadne` replacement beside Metis; the first setup attempt had exposed that missing dependency
  before the sweep began.
- Focused scheduling/cancellation tests passed `go test ./cmd/metis -race -run
  'Test(Sweep_ColdAdmissionCompletesTrainBeforeFifthAcquire|NestedCV_PeakConcurrencyWithinCap|NestedCV_FirstFailureStopsAllObservableWork|RunControl)'
  -count=10`; `go test ./cmd/metis -race -count=1`, `go test ./... -race -count=1`, and
  `git diff --check` also passed with no race or whitespace report. `sdlc issue validate --issue 43`
  passed. Task 4's deterministic acceptance covers byte-identical serial/parallel manifests and
  ledgers plus semantically equal run records.
- Quality-review rerun pinned the no-hardlink Ariadne clone at
  `a24643e566eb67ebbc69376126000e469761f09a` before making the no-hardlink Metis clone. The durable
  recipe passed unchanged: first train line 25 preceded fifth `cv-split` line 32; the estimate and
  seven-row summary were on lines 60 and 62; seven trains completed in 45s (~9.3/min). Source status
  and `refs/metis/*` remained byte-identical, and the temporary clone/cache root was removed.

### 2026-07-16 — pre-close verification
- Re-ran the focused scheduling/cancellation/TUI subset under `-race -count=10`, the standalone
  `cmd/metis` race suite, and `go test ./... -race -count=1`; all passed. The smoke dry-run again
  reported `2 × (3 configs × 2 inner folds)`, issue validation conformed, the diff check passed, and
  the worktree was clean before entering the gate-owned close review.

## Revisions

### 2026-07-16 — fresh-eyes spec review
- Made cancellation experiment-wide, required failure publication before admission release, and
  defined how typed aborted slots are excluded from every observable/reduction seam. Narrowed the
  determinism claim for timing-bearing run records and added a barrier-testable failure handoff.

### 2026-07-16 — durable plan review
- Replaced the generic two-line implementation sketch with the reviewed three-chunk plan and added
  the reconciled estimate. Review tightened controller hook semantics, atomically linearized
  observations against failure, and made cold-order/throughput/isolation proof executable.

### 2026-07-16 — estimate-gate revision
- Raised the estimate from 1.94h to 4.33h after the implementation-entry judge identified missing
  primitive decomposition for the controller, two cross-cutting integrations, deterministic race
  suite, and real-process smoke. The approved behavior and one-boundary delivery plan are unchanged.

### 2026-07-16 — TUI failure-path revision
- Extended the plan's global-abort boundary through the live progress board: ticker observations
  linearize through `runControl`, an error return discards and erases the stored frame before the
  top-level close, and a publication-barrier test drives a post-failure tick deterministically.
- Raised the estimate from 4.33h to 4.74h for the added TUI integration and race-test surface.
# Bounded Whole-Run Admission Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bound a parallel sweep to `2 × --parallel` admitted concrete runs so cold sweeps finish useful runs early, while globally aborting queued work after the first failure and preserving deterministic persisted results.

**Architecture:** Add one sweep-scoped `runControl`: its optional token channel bounds whole-run admission, while its mutex-protected error slot is the experiment-wide authoritative failure latch in serial and parallel modes. Wrap the existing `runResolvedExperiment` side-effect body at its single shared boundary, then make the nested sampler shell consult the same control before emitting progress, accumulating results, scoring, or persisting. Keep the existing leaf semaphore independent and unchanged.

**Tech Stack:** Go 1.x, generics-based `pkg/sampler`, channel/mutex coordination, injected `experiment.StepExecutor` fakes, real `uv`/Python process smoke, standard `go test -race`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `errRunAborted` | `cmd/metis/runcontrol.go` | new |

- **`errRunAborted`** — immutable sentinel returned when a concrete run is skipped, or its otherwise-successful result becomes non-authoritative, because another run already published the experiment's first failure.
  - **Relationships:** N:1 from concrete run attempts to the sweep's one authoritative first error; callers use `errors.Is` to distinguish cancellation from the original failure.
  - **DRY rationale:** One typed signal prevents each sampler level from inventing a zero-value/error convention.
  - **Future extensions:** It can later wrap a cancellation reason without changing the admission contract.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runControl` | `cmd/metis/runcontrol.go` | new | goroutine admission and sweep failure state |
| `runResolvedExperiment` admission wrapper | `cmd/metis/run.go` | modified | concrete-run filesystem, cache, executor, record, and capture effects |
| `shapeSweep` abort-aware reduction | `cmd/metis/sweep.go` | modified | nested sampler progress, accumulation, scoring, and persistence |
| `sweepProgress` / `boardWriter` abort path | `cmd/metis/progress.go`, `cmd/metis/board.go` | modified | ticker observations, pinned frame erasure, and deferred board close |

- **`runControl`** — one controller per shape run; its token channel is nil in serial mode and has capacity `2 × maxParallel` in parallel mode, while its first-error latch exists in both modes.
  - **Injected into:** copied `runOpts`, so inner folds, outer scoring, and the outer-analysis preamble share one controller without global state.
  - **Future extensions:** #49 may add a separate typed activity sink beside it; activity is not part of this controller.
- **`runResolvedExperiment` admission wrapper** — acquires before the first concrete-run side effect, checks cancellation, executes the existing body, publishes a contextual failure before token release, and rejects a late success if a sibling failed meanwhile.
  - **Injected into:** every current and future concrete-run call automatically through `runOpts`; individual callers supply only a contextual label.
  - **Future extensions:** a typed run role may replace the label when #49 adds completion telemetry.
- **`shapeSweep` abort-aware reduction** — preserves `sampler.Run`'s fixed output shape while ensuring its cancellation zero values never escape into progress, estimates, manifests, ledgers, or outer scoring.
  - **Injected into:** flat and nested sweep closures through the existing `shapeSweep` / `sweepPass` ownership graph.
  - **Future extensions:** adaptive samplers can consult the same authoritative stop state without changing the concrete-run gate.
- **`sweepProgress` / `boardWriter` abort path** — linearizes ticker repaints through the same
  failure latch, then erases and forgets the pinned frame on any post-wiring error so the top-level
  deferred close can flush ordinary output and restore the cursor without redrawing stale progress.
  - **Injected into:** `runShapeSweep`'s board wiring and named error return; plain output remains
    unchanged and retains no board-specific branch.
  - **Future extensions:** any new live display refresh must enter through the progress sink and the
    controller's healthy-observation operation.

## Chunk 1: Admission and failure primitive

### Task 1: Build `runControl` with race-safe handoff semantics

**Files:**
- Create: `cmd/metis/runcontrol.go`
- Create: `cmd/metis/runcontrol_test.go`

- [x] **Step 1: Write failing tests for capacity, first-error authority, and publication-before-release**

Define bounded helpers in the test file so a broken controller fails locally instead of waiting for
Go's global test timeout:

```go
type controlResult struct { run experiment.Run; err error }

func recvWithin[T any](t *testing.T, ch <-chan T) T {
    t.Helper()
    select {
    case v := <-ch:
        return v
    case <-time.After(2 * time.Second):
        t.Fatal("timed out waiting for run-control event")
        var zero T
        return zero
    }
}

func waitUntil(t *testing.T, pred func() bool) {
    t.Helper()
    deadline := time.Now().Add(2 * time.Second)
    for !pred() {
        if time.Now().After(deadline) { t.Fatal("timed out waiting for condition") }
        runtime.Gosched()
    }
}
```

Then create focused tests with no filesystem or executor mocks:

```go
func TestRunControlBoundsAdmission(t *testing.T) {
    const parallel = 3
    c := newRunControl(parallel)
    release := make(chan struct{})
    var mu sync.Mutex
    active, peak := 0, 0
    var wg sync.WaitGroup
    for i := 0; i < 12; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, _ = c.run("fold", func() (experiment.Run, error) {
                mu.Lock()
                active++
                if active > peak { peak = active }
                mu.Unlock()
                <-release
                mu.Lock(); active--; mu.Unlock()
                return experiment.Run{}, nil
            })
        }()
    }
    waitUntil(t, func() bool { mu.Lock(); defer mu.Unlock(); return peak == 2*parallel })
    close(release)
    done := make(chan struct{})
    go func() { wg.Wait(); close(done) }()
    recvWithin(t, done)
    if peak != 2*parallel { t.Fatalf("peak = %d, want %d", peak, 2*parallel) }
}

func TestRunControlPublishesFailureBeforeAdmissionRelease(t *testing.T) {
    c := &runControl{slots: make(chan struct{}, 1)}
    failingEntered := make(chan struct{})
    letFailureReturn := make(chan struct{})
    published := make(chan struct{})
    letTokenRelease := make(chan struct{})
    secondExecuted := make(chan struct{}, 1)
    c.beforeFailureUnlock = func() {
        close(published)
        <-letTokenRelease
    }

    firstDone := make(chan controlResult, 1)
    go func() {
        run, err := c.run("config a fold 0", func() (experiment.Run, error) {
            close(failingEntered)
            <-letFailureReturn
            return experiment.Run{ID: "must-not-escape"}, errors.New("train failed")
        })
        firstDone <- controlResult{run, err}
    }()
    recvWithin(t, failingEntered)

    secondDone := make(chan controlResult, 1)
    go func() {
        run, err := c.run("config b fold 0", func() (experiment.Run, error) {
            secondExecuted <- struct{}{}
            return experiment.Run{ID: "queued"}, nil
        })
        secondDone <- controlResult{run, err}
    }()
    close(letFailureReturn)

    recvWithin(t, published)
    if got := len(c.slots); got != 1 { t.Fatalf("slot released before publication hook: len=%d", got) }
    close(letTokenRelease)

    first := recvWithin(t, firstDone)
    second := recvWithin(t, secondDone)
    if got := first.err.Error(); !strings.Contains(got, "config a fold 0: train failed") {
        t.Fatalf("first error = %q", got)
    }
    if got := c.firstError(); got == nil || !strings.Contains(got.Error(), "config a fold 0") {
        t.Fatalf("stored failure after hook release = %v", got)
    }
    if !reflect.DeepEqual(first.run, experiment.Run{}) { t.Fatalf("failed run escaped: %+v", first.run) }
    if !errors.Is(second.err, errRunAborted) { t.Fatalf("second error = %v", second.err) }
    if !reflect.DeepEqual(second.run, experiment.Run{}) { t.Fatalf("aborted run escaped: %+v", second.run) }
    select {
    case <-secondExecuted: t.Fatal("queued run executed after failure")
    default:
    }
}
```

Add these complete companion cases:

```go

exec
/bin/zsh -lc 'git diff --check 17e93ec83aa4cc8a7777e95659b94ad9c9750e2a..HEAD
git diff --unified=80 17e93ec83aa4cc8a7777e95659b94ad9c9750e2a..HEAD -- cmd/metis/runcontrol.go cmd/metis/run.go cmd/metis/sweep.go cmd/metis/progress.go cmd/metis/board.go
find .. -name AGENTS.md -print' in /Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first
 succeeded in 113ms:
codex
The core-concepts table matches the introduced entities so far: the sentinel is IO-free, while controller, runner wrapper, sweep reduction, and board abort are correctly classified as integration surfaces. I’m now checking cancellation races and persistence boundaries against the concrete call graph.
exec
/bin/zsh -lc "go test ./cmd/metis -run 'TestRunControl|TestSweep_ColdAdmission|TestNestedCV_FirstFailure|TestRunExperiment_BoardFailure' -count=1" in /Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first
 succeeded in 608ms:
ok  	github.com/xianxu/metis/cmd/metis	0.452s

exec
/bin/zsh -lc "rg -n \"runResolvedExperiment\\(\" cmd/metis --glob '*.go'
rg -n \"whileHealthy|firstError\\(|\\.fail\\(\" cmd/metis/sweep.go cmd/metis/runcontrol.go
nl -ba cmd/metis/runcontrol.go | sed -n '1,150p'
nl -ba cmd/metis/sweep.go | sed -n '250,380p'
nl -ba cmd/metis/sweep.go | sed -n '400,690p'" in /Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first
 succeeded in 0ms:
cmd/metis/select_cmd.go:364:		if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
cmd/metis/select_cmd.go:538:	if _, err := runResolvedExperiment(exp, ro, runID, now, o.out); err != nil {
cmd/metis/sweep.go:488:	if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
cmd/metis/sweep.go:588:	run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
cmd/metis/sweep.go:648:	run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
cmd/metis/run.go:190:	return runResolvedExperiment(exp, o, singleRunID(o, exp, now), now, out)
cmd/metis/run.go:215:func runResolvedExperiment(exp experiment.Experiment, o runOpts, runID string, now func() time.Time, out io.Writer) (experiment.Run, error) {
cmd/metis/runcontrol_test.go:25:	_, err := runResolvedExperiment(exp, runOpts{
cmd/metis/runcontrol.go:36:func (c *runControl) firstError() error {
cmd/metis/runcontrol.go:42:// whileHealthy linearizes an observable transition against first-failure
cmd/metis/runcontrol.go:46:func (c *runControl) whileHealthy(fn func()) bool {
cmd/metis/runcontrol.go:92:	if c.firstError() != nil {
cmd/metis/runcontrol.go:98:		return experiment.Run{}, c.fail(label, err)
cmd/metis/runcontrol.go:100:	if c.firstError() != nil {
cmd/metis/sweep.go:95:	return ss.o.runControl.fail(label, err)
cmd/metis/sweep.go:98:func (ss *shapeSweep) firstError() error {
cmd/metis/sweep.go:99:	return ss.o.runControl.firstError()
cmd/metis/sweep.go:102:func (ss *shapeSweep) whileHealthy(fn func()) bool {
cmd/metis/sweep.go:103:	return ss.o.runControl.whileHealthy(fn)
cmd/metis/sweep.go:107:	return ss.whileHealthy(func() {
cmd/metis/sweep.go:139:	return p.ss.fail(label, err)
cmd/metis/sweep.go:142:func (p *sweepPass) firstError() error {
cmd/metis/sweep.go:143:	return p.ss.firstError()
cmd/metis/sweep.go:149:	return p.ss.whileHealthy(func() {
cmd/metis/sweep.go:157:	return p.ss.whileHealthy(func() {
cmd/metis/sweep.go:175:					ss.whileHealthy(func() { pass.hooks.fold(ev) })
cmd/metis/sweep.go:181:			ss.whileHealthy(func() { pass.hooks.config(ev) })
cmd/metis/sweep.go:290:					ss.whileHealthy(func() { ss.prog.tick() })
cmd/metis/sweep.go:327:	if err := ss.firstError(); err != nil {
cmd/metis/sweep.go:330:	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the report
cmd/metis/sweep.go:339:		return ss.fail("write sweep manifest", err)
cmd/metis/sweep.go:345:		ss.whileHealthy(func() {
cmd/metis/sweep.go:353:		return ss.fail("write sweep ledger", err)
cmd/metis/sweep.go:361:		return ss.fail("sweep complexity guard", err)
cmd/metis/sweep.go:363:	ss.whileHealthy(func() {
cmd/metis/sweep.go:367:	return ss.firstError()
cmd/metis/sweep.go:409:		if first := ss.firstError(); first != nil {
cmd/metis/sweep.go:412:		return ss.fail("nested-CV preamble", err)
cmd/metis/sweep.go:418:			if ss.firstError() != nil {
cmd/metis/sweep.go:423:				if ss.firstError() == nil {
cmd/metis/sweep.go:424:					ss.fail(fmt.Sprintf("outer fold %d", p.Idx), ferr)
cmd/metis/sweep.go:435:			ss.whileHealthy(func() { ss.prog.driverEvent(ev) })
cmd/metis/sweep.go:437:	if err := ss.firstError(); err != nil {
cmd/metis/sweep.go:440:	ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the estimate report
cmd/metis/sweep.go:448:		return ss.fail("write nested sweep manifest", err)
cmd/metis/sweep.go:452:		ss.whileHealthy(func() {
cmd/metis/sweep.go:457:		return ss.fail("write nested sweep ledger", err)
cmd/metis/sweep.go:459:	ss.whileHealthy(func() {
cmd/metis/sweep.go:463:	return ss.firstError()
cmd/metis/sweep.go:482:		return nil, ss.fail("nested-CV preamble address", err)
cmd/metis/sweep.go:505:		return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
cmd/metis/sweep.go:512:	if err := pass.firstError(); err != nil {
cmd/metis/sweep.go:519:		return 0, ss.fail(fmt.Sprintf("outer fold %d complexity guard", i), err)
cmd/metis/sweep.go:525:	if !ss.whileHealthy(func() {
cmd/metis/sweep.go:550:		if !ss.whileHealthy(func() {
cmd/metis/sweep.go:582:		return 0, "", "", ss.fail(fmt.Sprintf("outer fold %d family %s score address", i, fam), err)
cmd/metis/sweep.go:622:	if p.firstError() != nil {
     1	package main
     2	
     3	import (
     4		"errors"
     5		"fmt"
     6		"sync"
     7	
     8		"github.com/xianxu/metis/pkg/experiment"
     9	)
    10	
    11	var errRunAborted = errors.New("run aborted after earlier sweep failure")
    12	
    13	// runControl bounds admitted concrete runs independently of leaf subprocess
    14	// parallelism and latches the first whole-run failure. Observation callbacks
    15	// must not call back into the controller or block production work.
    16	type runControl struct {
    17		slots chan struct{}
    18	
    19		mu  sync.Mutex
    20		err error
    21	
    22		beforeFailureLock   func()
    23		beforeFailureUnlock func()
    24		afterAcquire        func(label string)
    25		beforeRelease       func(label string)
    26	}
    27	
    28	func newRunControl(maxParallel int) *runControl {
    29		control := &runControl{}
    30		if maxParallel > 1 {
    31			control.slots = make(chan struct{}, 2*maxParallel)
    32		}
    33		return control
    34	}
    35	
    36	func (c *runControl) firstError() error {
    37		c.mu.Lock()
    38		defer c.mu.Unlock()
    39		return c.err
    40	}
    41	
    42	// whileHealthy linearizes an observable transition against first-failure
    43	// publication. The callback runs while c.mu is held and therefore must not call
    44	// back into runControl. Downstream locks are acquired only inside fn, preserving
    45	// the global control -> progress/pass/manifest order.
    46	func (c *runControl) whileHealthy(fn func()) bool {
    47		c.mu.Lock()
    48		defer c.mu.Unlock()
    49		if c.err != nil {
    50			return false
    51		}
    52		fn()
    53		return true
    54	}
    55	
    56	func (c *runControl) fail(label string, err error) error {
    57		if err == nil {
    58			return nil
    59		}
    60		contextual := err
    61		if label != "" {
    62			contextual = fmt.Errorf("%s: %w", label, err)
    63		}
    64		if c.beforeFailureLock != nil {
    65			c.beforeFailureLock()
    66		}
    67	
    68		c.mu.Lock()
    69		if c.err == nil {
    70			c.err = contextual
    71			if c.beforeFailureUnlock != nil {
    72				c.beforeFailureUnlock()
    73			}
    74		}
    75		authoritative := c.err
    76		c.mu.Unlock()
    77		return authoritative
    78	}
    79	
    80	func (c *runControl) run(label string, fn func() (experiment.Run, error)) (experiment.Run, error) {
    81		if c.slots != nil {
    82			c.slots <- struct{}{}
    83			defer func() { <-c.slots }()
    84			if c.afterAcquire != nil {
    85				c.afterAcquire(label)
    86			}
    87			if c.beforeRelease != nil {
    88				defer func() { c.beforeRelease(label) }()
    89			}
    90		}
    91	
    92		if c.firstError() != nil {
    93			return experiment.Run{}, errRunAborted
    94		}
    95	
    96		run, err := fn()
    97		if err != nil {
    98			return experiment.Run{}, c.fail(label, err)
    99		}
   100		if c.firstError() != nil {
   101			return experiment.Run{}, errRunAborted
   102		}
   103		return run, nil
   104	}
   250		}
   251	
   252		shapeRunID, err := shapeRunIdentity(sh, sbh)
   253		if err != nil {
   254			return err
   255		}
   256		ss := &shapeSweep{
   257			o: o, sh: sh, now: now, out: out, shapeBlobHash: sbh, codeID: sha, start: sweepStart,
   258			partRef:  partitionRef(sh),
   259			man:      sweepManifest{ShapeRunID: shapeRunID, Shape: sh.ID, Sampler: sh.Sweeper.Sampler, Seed: sh.Seed},
   260			parallel: o.maxParallel > 1, // metis#31: fan out the sweeper/resample/driver batches
   261		}
   262		ctx := sampler.Ctx{Seed: sh.Seed, Partition: ss.partRef}
   263		// metis#30: seed the sink's denominators AT WIRING TIME from the same SizeHint the
   264		// levels report (stream-learned totals would arrive only with each level's first
   265		// completion — for the driver level that's the first COMPLETED outer fold, too late).
   266		ss.prog = newSweepProgress(out, now, sh.Sweeper.Objective.Direction, seededTotals(ctx, nested, runFolds, configPts, k))
   267		// metis#38: board mode — the sink paints the pinned board instead of plain lines,
   268		// and a 500ms ticker keeps the rate decay + ETA live between events (sink-first:
   269		// tick() locks sp.mu then hands the frame to bw — the one global lock order).
   270		if o.board != nil {
   271			ss.prog.bw = o.board
   272			ss.prog.width = boardWidth()
   273			ss.prog.gauge = o.leafGauge
   274			tickC := o.boardTick
   275			var ticker *time.Ticker
   276			if tickC == nil {
   277				ticker = time.NewTicker(500 * time.Millisecond)
   278				tickC = ticker.C
   279			}
   280			tickDone := make(chan struct{})
   281			tickStopped := make(chan struct{})
   282			go func() {
   283				defer close(tickStopped)
   284				for {
   285					select {
   286					case <-tickC:
   287						if o.beforeBoardTick != nil {
   288							o.beforeBoardTick()
   289						}
   290						ss.whileHealthy(func() { ss.prog.tick() })
   291						if o.afterBoardTick != nil {
   292							o.afterBoardTick()
   293						}
   294					case <-tickDone:
   295						return
   296					}
   297				}
   298			}()
   299			defer func() {
   300				if ticker != nil {
   301					ticker.Stop()
   302				}
   303				close(tickDone)
   304				<-tickStopped
   305				if result != nil {
   306					ss.prog.abort()
   307				}
   308			}()
   309		}
   310	
   311		// metis#32: >1 config → nested CV (records inner + per-family outer rows; the honest measure).
   312		if nested {
   313			return ss.runNestedCV(ctx, configPts, k, runFolds, stratify, shapeRunID)
   314		}
   315	
   316		fmt.Fprintf(out, "metis: single-level CV %s (%s) — %d config × %d folds\n", sh.ID, shapeRunID[:12], len(configPts), k)
   317	
   318		// The flat single-level CV path (1 config): the SingleDriver (a runtime sampler node, NOT the
   319		// deleted shape `driver:`) runs the sweeper once on all data → the sweeper's inner k-fold CV
   320		// scores the one config → (mean, SE, complexity) recorded to the ledger. metis#32: it MEASURES
   321		// ONLY — no `shipWinner` (shipping is `metis select --promote`).
   322		pass := &sweepPass{ss: ss, splitK: k, stratify: stratify, partRef: ss.partRef,
   323			hooks: ss.prog.forPass(-1)} // metis#30: the flat path's single pass
   324		res := sampler.Run(ctx, sampler.SingleDriver{}, func(sampler.SinglePoint) sampler.SweepResult {
   325			return ss.runSweeper(ctx, configPts, pass)
   326		}, sampler.ExecFor[sampler.SinglePoint, sampler.SweepResult](ss.parallel), nil)
   327		if err := ss.firstError(); err != nil {
   328			return err
   329		}
   330		ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the report
   331		// metis#31: sort the fan-out's completion-order bookkeeping to a stable content key
   332		// BEFORE persisting, so manifest.json + the ledger are byte-deterministic across
   333		// serial/parallel runs (the winner/estimate are already deterministic; this makes
   334		// the on-disk artifacts match metis's content-addressing posture).
   335		sortPointRuns(pass.points)
   336		ss.man.Points = pass.points
   337		ss.configs = pass.configs
   338		if err := writeManifest(o.expPath, ss.man); err != nil {
   339			return ss.fail("write sweep manifest", err)
   340		}
   341		// Capture the sweep's code closure to a git side ref (metis#8/#14) — BEST-EFFORT: the
   342		// records + manifest are already valid, so a capture hiccup warns, never aborts.
   343		cohort, err := captureSweepCode(o, ss.man)
   344		if err != nil {
   345			ss.whileHealthy(func() {
   346				fmt.Fprintf(out, "metis: warning: code capture failed (%v) — the sweep's records are valid but not committed to a side ref\n", err)
   347			})
   348		}
   349		// Persist the raw per-fold rows to the shape's append-only ledger sidecar (metis#8/#18):
   350		// AggregateView reduces them read-time to per-config (mean, SE) — so metis#19's 1-SE
   351		// select re-reduces the same rows without a re-run.
   352		if err := writeSweepLedger(o.expPath, ss.man); err != nil {
   353			return ss.fail("write sweep ledger", err)
   354		}
   355		// Guard (metis#19): a parsimony rule (one-std-err/pct-loss) needs a measured complexity
   356		// for every swept family — else the parsimony axis is silently dropped and the winner is
   357		// quietly wrong. The raw fold rows are already persisted (re-selectable after a fix); only
   358		// the ship/report is gated. Checked here (post-fold) because HasComplexity is only known
   359		// after the folds run.
   360		if err := sampler.GuardComplexity(sh.Sweeper.Objective.Select, configStatsOf(ss.configs)); err != nil {
   361			return ss.fail("sweep complexity guard", err)
   362		}
   363		ss.whileHealthy(func() {
   364			ss.reportWinner(res)
   365			printRunSummary(out, o.expPath, now().Sub(sweepStart), len(ss.man.Points), cohort)
   366		})
   367		return ss.firstError()
   368	}
   369	
   370	// configStatsOf builds the per-config stats (with each config's family) from a completed
   371	// sweep pass — the GuardComplexity input, matching what GridConfigs.Done reduces internally.
   372	// Free over a []configScore so BOTH the flat path (ss.configs) and each driver:cv sealed
   373	// outer fold (pass.configs) guard the same way (ARCH-DRY, metis#23 I1).
   374	func configStatsOf(configs []configScore) []sampler.ConfigStat {
   375		stats := make([]sampler.ConfigStat, len(configs))
   376		for i, c := range configs {
   377			stats[i] = sampler.ConfigStat{Point: c.point, Family: sampler.FamilyOf(c.point), Score: c.meanSE}
   378		}
   379		return stats
   380	}
   400	func (ss *shapeSweep) runNestedCV(ctx sampler.Ctx, configPts []shape.Point, k, runFolds int, stratify bool, shapeRunID string) error {
   401		fmt.Fprintf(ss.out, "metis: nested-CV %s (%s) — %d outer fold(s) × (%d configs × %d inner folds)\n",
   402			ss.sh.ID, shapeRunID[:12], runFolds, len(configPts), k)
   403	
   404		// Preamble: materialize the k outer-analysis subset dirs ONCE (unconfined — outer-split reads
   405		// the full dataset to split it). Always split into k dirs (a stable partition); --fast just runs
   406		// fewer of them (runFolds ≤ k). Deterministic run id → the analysis_i refs are locatable.
   407		analysisRefs, err := ss.materializeOuterAnalysis(k, stratify)
   408		if err != nil {
   409			if first := ss.firstError(); first != nil {
   410				return first
   411			}
   412			return ss.fail("nested-CV preamble", err)
   413		}
   414		outerPart := sampler.PartitionRef(fmt.Sprintf("outer-cv-k%d-strat%t-seed%d", k, stratify, ss.sh.Seed))
   415	
   416		est := sampler.Run(ctx, sampler.CVDriver{K: runFolds, Stratify: stratify},
   417			func(p sampler.OuterFoldPoint) float64 {
   418				if ss.firstError() != nil {
   419					return 0
   420				}
   421				score, ferr := ss.runOuterFold(ctx, configPts, k, stratify, analysisRefs[p.Idx], outerPart, p.Idx)
   422				if ferr != nil {
   423					if ss.firstError() == nil {
   424						ss.fail(fmt.Sprintf("outer fold %d", p.Idx), ferr)
   425					}
   426					return 0
   427				}
   428				return score
   429			},
   430			sampler.ExecFor[sampler.OuterFoldPoint, float64](ss.parallel),
   431			// metis#30: outer-fold completions always emit. Error-gated: once runControl
   432			// latches, remaining closures return sentinel zeros — don't fold those into
   433			// the displayed est (the run is aborting; a fake 0 would tank the line).
   434			func(ev sampler.ProgressEvent[sampler.OuterFoldPoint, float64]) {
   435				ss.whileHealthy(func() { ss.prog.driverEvent(ev) })
   436			})
   437		if err := ss.firstError(); err != nil {
   438			return err
   439		}
   440		ss.whileHealthy(ss.prog.finish) // metis#30: the terminal progress line, before the estimate report
   441	
   442		// metis#32: the nested run now RECORDS (unlike metis#23's estimation-only path) — persist the
   443		// inner + per-family outer rows accumulated in ss.man.Points so `metis select` can reduce them
   444		// (family from the outer rows, config from the inner rows). Sort to a stable content key first
   445		// (the outer folds appended concurrently under ParExec) for byte-deterministic artifacts.
   446		sortPointRuns(ss.man.Points)
   447		if err := writeManifest(ss.o.expPath, ss.man); err != nil {
   448			return ss.fail("write nested sweep manifest", err)
   449		}
   450		cohort, cerr := captureSweepCode(ss.o, ss.man)
   451		if cerr != nil {
   452			ss.whileHealthy(func() {
   453				fmt.Fprintf(ss.out, "metis: warning: code capture failed (%v) — the nested run's records are valid but not committed to a side ref\n", cerr)
   454			})
   455		}
   456		if err := writeSweepLedger(ss.o.expPath, ss.man); err != nil {
   457			return ss.fail("write nested sweep ledger", err)
   458		}
   459		ss.whileHealthy(func() {
   460			ss.reportEstimate(est, runFolds)
   461			printRunSummary(ss.out, ss.o.expPath, ss.now().Sub(ss.start), len(ss.man.Points), cohort)
   462		})
   463		return ss.firstError()
   464	}
   465	
   466	// materializeOuterAnalysis runs the nested-CV preamble ({data phase + outer-split(k=outerK)}) ONCE
   467	// and returns the k analysis_i refs (experiment-relative, so a sealed sweep reading one routes
   468	// through exp_path → confined). Unconfined (outer-split reads the full dataset to split it).
   469	func (ss *shapeSweep) materializeOuterAnalysis(outerK int, stratify bool) ([]string, error) {
   470		baseOut, baseID := baseDatasetRef(ss.sh)
   471		var needs []string
   472		if baseID != "" {
   473			needs = []string{baseID}
   474		}
   475		osStep := experiment.Step{ID: outerSplitStepID, Uses: "metis/outer-split", Needs: needs,
   476			With: map[string]any{"dataset": baseOut, "k": outerK, "stratify": stratify}}
   477		steps := append(append([]experiment.Step{}, ss.sh.Data...), osStep)
   478		exp := experiment.Experiment{Header: ss.sh.Header, Steps: steps}
   479		exp.Type = "experiment"
   480		preID, err := pointAddressOf(exp, ss.shapeBlobHash)
   481		if err != nil {
   482			return nil, ss.fail("nested-CV preamble address", err)
   483		}
   484		preOpts := ss.o
   485		preOpts.inSweep = true // one preamble run; skip the per-run capture noise
   486		preOpts.readRoot = ""  // outer-split legitimately reads the full dataset
   487		preOpts.runLabel = fmt.Sprintf("outer-analysis preamble (%s)", preID)
   488		if _, err := runResolvedExperiment(exp, preOpts, preID, ss.now, ss.out); err != nil {
   489			return nil, err
   490		}
   491		refs := make([]string, outerK)
   492		for i := 0; i < outerK; i++ {
   493			refs[i] = filepath.ToSlash(filepath.Join("runs", preID, outerSplitStepID, fmt.Sprintf("analysis_%d", i)))
   494		}
   495		return refs, nil
   496	}
   497	
   498	// runOuterFold runs one outer fold: (a) the SEALED sweeper on analysis_i → a winner (confined via
   499	// the exp_path chokepoint — readRoot = analysis_i abs), then (b) the refit-and-score of that winner
   500	// on the held outer-assessment (a full-data fold run at outer-k, held=i; unconfined). Returns the
   501	// honest outer-fold score.
   502	func (ss *shapeSweep) runOuterFold(ctx sampler.Ctx, configPts []shape.Point, k int, stratify bool, analysisRef string, outerPart sampler.PartitionRef, i int) (float64, error) {
   503		analysisAbs, err := filepath.Abs(filepath.Join(filepath.Dir(ss.o.expPath), analysisRef))
   504		if err != nil {
   505			return 0, ss.fail(fmt.Sprintf("outer fold %d analysis path", i), err)
   506		}
   507		// (a) sealed selection: the sweeper's inner-CV runs entirely within analysis_i (inner k/stratify).
   508		pass := &sweepPass{ss: ss, baseRef: analysisRef, readRoot: analysisAbs, splitK: k,
   509			stratify: stratify, partRef: ss.partRef,
   510			hooks: ss.prog.forPass(i)} // metis#30/#38: outer-fold identity via closure binding
   511		sres := ss.runSweeper(ctx, configPts, pass)
   512		if err := pass.firstError(); err != nil {
   513			return 0, err
   514		}
   515		// Guard (metis#19/#23 I1): the parsimony select rule needs a measured complexity for every
   516		// swept family — same guard the flat path runs before trusting its winner. Without it, a
   517		// parsimony-select + non-reporting-model shape would SILENTLY mis-select in each outer fold.
   518		if err := sampler.GuardComplexity(ss.sh.Sweeper.Objective.Select, configStatsOf(pass.configs)); err != nil {
   519			return 0, ss.fail(fmt.Sprintf("outer fold %d complexity guard", i), err)
   520		}
   521	
   522		// metis#32: record the sealed sweep's INNER rows (Level=inner, tagged with this outer fold).
   523		of := i
   524		rows := make([]pointRun, 0, len(pass.points)+len(sres.PerFamily))
   525		if !ss.whileHealthy(func() {
   526			for _, pr := range pass.points {
   527				pr.Level = "inner"
   528				pr.OuterFold = &of
   529				rows = append(rows, pr)
   530			}
   531		}) {
   532			return 0, errRunAborted
   533		}
   534	
   535		// (b) score EACH family's inner-winner on the held outer-assessment — post-selection, so
   536		// unconfined and leakage-free (each winner was selected SEALED within analysis_i; scoring on
   537		// the held-out fold never influenced that selection). One OUTER row per family → the honest
   538		// per-family measure `metis select` reduces (metis#32). The metis#23 estimate the CVDriver
   539		// aggregates stays the SHIP-family's outer score (the argmax-mean procedure's honest number).
   540		// The cv-split uses the OUTER k + stratify so cv_folds's determinism reproduces the exact
   541		// partition outer-split materialized (else the held fold ≠ analysis_i's assessment rows).
   542		shipFamily := sres.Ship.Family
   543		var shipScore float64
   544		for _, fam := range sortedFamilies(sres.PerFamily) {
   545			w := sres.PerFamily[fam]
   546			score, scoreID, status, ferr := ss.scoreOnOuterFold(w.Point, i, k, stratify, outerPart, fam)
   547			if ferr != nil {
   548				return 0, ferr
   549			}
   550			if !ss.whileHealthy(func() {
   551				rows = append(rows, pointRun{
   552					RunID:      scoreID,
   553					FreeParams: freeParamMap(w.Point),
   554					Fold:       of, // the outer fold this held-out score is on
   555					Level:      "outer",
   556					OuterFold:  &of,
   557					Status:     status,
   558					// Metrics filled read-time from the run's record.json (namespaced), like inner rows.
   559				})
   560				if fam == shipFamily {
   561					shipScore = score
   562				}
   563				fmt.Fprintf(ss.out, "  outer fold %d: %s winner %s → held-out %.4f\n",
   564					i, fam, freeParamStrFromParams(w.Point.FreeParams), score)
   565			}) {
   566				return 0, errRunAborted
   567			}
   568		}
   569		if !ss.addManPoints(rows) {
   570			return 0, errRunAborted
   571		}
   572		return shipScore, nil
   573	}
   574	
   575	// scoreOnOuterFold refit-and-scores one config's winner on the held outer-assessment fold i (a
   576	// full-data fold run at outer-k; post-selection, so unconfined). Returns the held-out fold_score,
   577	// the run id (→ its record.json carries the namespaced metric the ledger reads), and its status.
   578	func (ss *shapeSweep) scoreOnOuterFold(point shape.Point, i, k int, stratify bool, outerPart sampler.PartitionRef, fam string) (float64, string, string, error) {
   579		scoreExp := ss.buildFoldExperiment(point, sampler.FoldPoint{Idx: i}, nil, k, stratify, outerPart)
   580		scoreID, err := pointAddressOf(scoreExp, ss.shapeBlobHash)
   581		if err != nil {
   582			return 0, "", "", ss.fail(fmt.Sprintf("outer fold %d family %s score address", i, fam), err)
   583		}
   584		scoreOpts := ss.o
   585		scoreOpts.inSweep = true
   586		scoreOpts.readRoot = "" // the outer-assessment eval reads full data legitimately
   587		scoreOpts.runLabel = fmt.Sprintf("outer fold %d family %s score (%s)", i, fam, scoreID)
   588		run, err := runResolvedExperiment(scoreExp, scoreOpts, scoreID, ss.now, ss.out)
   589		if err != nil {
   590			return 0, "", "", err
   591		}
   592		return run.Metrics[foldMetric], scoreID, run.Status, nil
   593	}
   594	
   595	// sortedFamilies returns the family keys of a per-family winner map in deterministic order
   596	// (the recording + the returned ship-score must not depend on Go's random map iteration).
   597	func sortedFamilies(perFamily map[string]sampler.Winner) []string {
   598		fams := make([]string, 0, len(perFamily))
   599		for fam := range perFamily {
   600			fams = append(fams, fam)
   601		}
   602		sort.Strings(fams)
   603		return fams
   604	}
   605	
   606	// reportEstimate prints the honest procedure estimate — mean±SE over the outer folds — and the
   607	// standing reminder that driver:cv produces NO shippable winner (estimation ≠ selection).
   608	func (ss *shapeSweep) reportEstimate(est sampler.MeanSE, outerK int) {
   609		fmt.Fprintf(ss.out, "metis: nested-CV estimate — mean %.4f (SE %.4f) over %d outer fold(s) — the HONEST procedure estimate (argmax-mean family)\n",
   610			est.Mean, est.SE, outerK)
   611		fmt.Fprintf(ss.out, "  (per-family honest estimates recorded to the ledger; choose + ship via `metis select --best --promote`)\n")
   612	}
   613	
   614	// runPipelineFold runs ONE (config, fold) point: build its per-fold experiment (data +
   615	// synthesized cv-split + pipeline, with the config + fold-context overlaid), run it through
   616	// the shared cached runner, record the manifest row, and return the fold_score the inner
   617	// resample Sampler folds. A fatal outcome publishes through the experiment-wide
   618	// runControl and returns 0; every sampler callback/sink rejects placeholders after
   619	// publication, and the top level returns the stored concrete cause.
   620	func (p *sweepPass) runPipelineFold(c shape.Point, f sampler.FoldPoint) sampler.FoldOutcome {
   621		ss := p.ss
   622		if p.firstError() != nil {
   623			return sampler.FoldOutcome{}
   624		}
   625		// Detect-and-abort: a mid-sweep HEAD-sha change breaks the shape-run's one-code
   626		// identity (per-fold records stay correct). Compares the HEAD sha only, not the dirty
   627		// flag — the sweep's own writes (runs/, manifest) dirty the tree (see codeID freeze).
   628		// metis#31: only a DEFINITE sha change aborts — `s != ""`. probeRepo swallows any
   629		// probe error to "", and under parallel fan-out concurrent `git status` contends on
   630		// .git/index.lock so a transient probe failure is expected; treating "" as a change
   631		// would false-abort the whole honest run.
   632		if _, s, _ := probeRepo(ss.o.git, filepath.Dir(ss.o.expPath)); s != "" && s != ss.codeID {
   633			p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx),
   634				fmt.Errorf("code changed mid-sweep (%s → %s) — re-run to sweep the new revision", ss.codeID, s))
   635			return sampler.FoldOutcome{}
   636		}
   637	
   638		exp := ss.buildFoldExperiment(c, f, p.baseRef, p.splitK, p.stratify, p.partRef)
   639		runID, err := pointAddressOf(exp, ss.shapeBlobHash)
   640		if err != nil {
   641			p.setErr(fmt.Sprintf("config %s fold %d", freeParamStr(c), f.Idx), err)
   642			return sampler.FoldOutcome{}
   643		}
   644		pointOpts := ss.o
   645		pointOpts.inSweep = true        // metis#14: the sweep captures once (captureSweepCode), not per point
   646		pointOpts.readRoot = p.readRoot // metis#23: confine a sealed outer-fold pass to its analysis root
   647		pointOpts.runLabel = fmt.Sprintf("config %s fold %d (%s)", freeParamStr(c), f.Idx, runID)
   648		run, runErr := runResolvedExperiment(exp, pointOpts, runID, ss.now, ss.out)
   649		// A failing fold is FATAL to the sweep, unlike a v1 flat point: a config scored over a
   650		// PARTIAL fold set is not an honest (mean, SE) estimate. Any error (a step failure, a
   651		// validation never-start, a persistence error) aborts — surfaced, never a half-scored config.
   652		if runErr != nil {
   653			// runControl already published a concrete admitted-run failure. A queued or
   654			// late sibling returns errRunAborted; neither path may republish the sentinel.
   655			return sampler.FoldOutcome{}
   656		}
   657		if !p.addPoint(pointRun{
   658			RunID:      runID,
   659			FreeParams: freeParamMap(c),
   660			Fold:       f.Idx,
   661			Status:     run.Status,
   662			Metrics:    run.Metrics,
   663		}) {
   664			return sampler.FoldOutcome{}
   665		}
   666		// metis#19 M2: read the train step's realized-complexity metric. Present → the parsimony
   667		// rules consume it; absent (HasComplexity false) → the guard rejects a parsimony rule.
   668		cx, hasCx := run.Metrics[foldComplexityMetric]
   669		return sampler.FoldOutcome{Score: run.Metrics[foldMetric], Complexity: cx, HasComplexity: hasCx}
   670	}
   671	
   672	// buildFoldExperiment reconstructs the runnable per-fold experiment for one (config, fold):
   673	// the data steps (as declared — cache-shared, config+fold-invariant) + the engine-synthesized
   674	// cv-split partition step + the pipeline steps with the config's resolved `with` overlaid AND
   675	// the fold-context injected. The fold-context ({_fold:{partition,idx}, folds:<cv-split>}) enters
   676	// each pipeline step's `with` so its Kpre is fold-distinct (the B2 collision guard) and the step
   677	// can read the fold assignment. Ship is NOT included (winner-only, M1a-5).
   678	// baseRef nil = the flat driver:single path (data phase + cv-split over the declared base).
   679	// baseRef non-nil = a sealed nested outer fold (metis#23): the data phase is DROPPED (analysis_i
   680	// is already the adapted base) and cv-split + every pipeline step that read the declared base are
   681	// repointed to baseRef (analysis_i), so their reads route through exp_path → confined to the
   682	// outer-analysis root and the sweeper's inner-CV structurally cannot see outer-assessment.
   683	func (ss *shapeSweep) buildFoldExperiment(c shape.Point, f sampler.FoldPoint, baseRef any, splitK int, stratify bool, partRef sampler.PartitionRef) experiment.Experiment {
   684		sh := ss.sh
   685		steps := make([]experiment.Step, 0, len(sh.Data)+1+len(sh.Pipeline))
   686		baseOut, baseID := baseDatasetRef(sh)
   687		origOut := baseOut // the declared base, captured before the sealed branch reassigns baseOut
   688		var partNeeds []string
   689		if baseRef == nil {
   690			steps = append(steps, sh.Data...)

exec
/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first
codex
The targeted controller, cold-admission, nested-failure, and board-failure tests pass. I found no unlabelled sweep concrete-run call or post-failure reduction consumer; the full suite is still running.
 succeeded in 25811ms:
ok  	github.com/xianxu/metis/cmd/metis	25.623s
ok  	github.com/xianxu/metis/internal/repo	0.405s
ok  	github.com/xianxu/metis/pkg/cache	0.623s
ok  	github.com/xianxu/metis/pkg/cas	1.172s
ok  	github.com/xianxu/metis/pkg/experiment	0.899s
ok  	github.com/xianxu/metis/pkg/ledger	1.503s
ok  	github.com/xianxu/metis/pkg/record	1.295s
ok  	github.com/xianxu/metis/pkg/sampler	1.719s
ok  	github.com/xianxu/metis/pkg/shape	1.945s

codex
```verdict
verdict: SHIP
confidence: high
```

The implementation fulfills the issue’s bounded depth-first scheduling and experiment-wide cancellation contract. Concrete runs share one `2n` admission controller, the existing leaf budget remains independent at `n`, first-failure publication precedes admission release, and post-failure sampler, persistence, reporting, and TUI consumers are gated. The plan, core-concepts table, atlas update, and code agree. No Critical, Important, or Minor findings.

1. Strengths

- Admission wraps the shared concrete-run boundary before filesystem, cache, output, executor, or record effects ([run.go](/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first/cmd/metis/run.go:215)).
- `runControl` cleanly separates whole-run admission from leaf concurrency and preserves one authoritative contextual error ([runcontrol.go](/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first/cmd/metis/runcontrol.go:28)).
- Health-gated callbacks establish a consistent controller → progress/pass/manifest lock order ([sweep.go](/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first/cmd/metis/sweep.go:102)).
- Board shutdown joins the ticker and discards its stored frame on every post-wiring error, preventing stale repaint during deferred close ([sweep.go](/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first/cmd/metis/sweep.go:299)).
- Serial/parallel equivalence now checks semantic `run.json` and `record.json` content in addition to byte-identical manifests and ledgers.

2. Critical findings

None.

3. Important findings

None. Atlas documentation covers the new controller and dual budgets. No new user-facing command, flag, or configuration key was introduced, so no README change is required.

4. Minor findings

None.

5. Test coverage notes

- Verified targeted controller, cold-admission, nested-failure, and board-failure tests: pass.
- Verified `go test ./... -count=1`: all packages pass.
- Tests cover capacity, token release, first-error authority, queued-run side-effect rejection, late-success rejection, deadlock resistance, deterministic artifacts, and stale TUI suppression.
- PURE `errRunAborted` semantics are tested without IO; integration behavior uses injected executors and controlled synchronization.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Admission and failure ownership live at one shared concrete-run boundary; no parallel scheduler implementation or duplicated error latch remains.
- `ARCH-PURE`: Pass. The controller is injected through `runOpts`; orchestration stays outside admission, and IO remains within the existing concrete-run shell.
- `ARCH-PURPOSE`: Pass. The shadow sweep found all concrete sweep consumers—preamble, inner folds, and outer scoring—derive from the shared controller. Progress, reductions, persistence, reporting, and board refreshes all consult the same failure authority.

7. Plan revision recommendations

None. The core-concepts classifications, paths, statuses, completed checklist, and documented evidence match the code.
tokens used
66,612
```verdict
verdict: SHIP
confidence: high
```

The implementation fulfills the issue’s bounded depth-first scheduling and experiment-wide cancellation contract. Concrete runs share one `2n` admission controller, the existing leaf budget remains independent at `n`, first-failure publication precedes admission release, and post-failure sampler, persistence, reporting, and TUI consumers are gated. The plan, core-concepts table, atlas update, and code agree. No Critical, Important, or Minor findings.

1. Strengths

- Admission wraps the shared concrete-run boundary before filesystem, cache, output, executor, or record effects ([run.go](/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first/cmd/metis/run.go:215)).
- `runControl` cleanly separates whole-run admission from leaf concurrency and preserves one authoritative contextual error ([runcontrol.go](/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first/cmd/metis/runcontrol.go:28)).
- Health-gated callbacks establish a consistent controller → progress/pass/manifest lock order ([sweep.go](/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first/cmd/metis/sweep.go:102)).
- Board shutdown joins the ticker and discards its stored frame on every post-wiring error, preventing stale repaint during deferred close ([sweep.go](/Users/xianxu/workspace/worktree/metis/000043-leaf-sched-depth-first/cmd/metis/sweep.go:299)).
- Serial/parallel equivalence now checks semantic `run.json` and `record.json` content in addition to byte-identical manifests and ledgers.

2. Critical findings

None.

3. Important findings

None. Atlas documentation covers the new controller and dual budgets. No new user-facing command, flag, or configuration key was introduced, so no README change is required.

4. Minor findings

None.

5. Test coverage notes

- Verified targeted controller, cold-admission, nested-failure, and board-failure tests: pass.
- Verified `go test ./... -count=1`: all packages pass.
- Tests cover capacity, token release, first-error authority, queued-run side-effect rejection, late-success rejection, deadlock resistance, deterministic artifacts, and stale TUI suppression.
- PURE `errRunAborted` semantics are tested without IO; integration behavior uses injected executors and controlled synchronization.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Admission and failure ownership live at one shared concrete-run boundary; no parallel scheduler implementation or duplicated error latch remains.
- `ARCH-PURE`: Pass. The controller is injected through `runOpts`; orchestration stays outside admission, and IO remains within the existing concrete-run shell.
- `ARCH-PURPOSE`: Pass. The shadow sweep found all concrete sweep consumers—preamble, inner folds, and outer scoring—derive from the shared controller. Progress, reductions, persistence, reporting, and board refreshes all consult the same failure authority.

7. Plan revision recommendations

None. The core-concepts classifications, paths, statuses, completed checklist, and documented evidence match the code.
