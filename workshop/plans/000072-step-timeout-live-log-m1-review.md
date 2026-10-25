# Boundary Review — metis#72 (milestone M1)

| field | value |
|-------|-------|
| issue | 72 — Bound step execution with timeout and live logs |
| repo | metis |
| issue file | workshop/issues/000072-step-timeout-live-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 99ccccb46a2707de6fbfe1be44c32263f0e636cf..77863b0613972bcff58aaf52a829748b17ae41e8 |
| command | sdlc milestone-close --issue 72 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T09:21:36-07:00 |
| verdict | REWORK |

## Review

Reviewing the M1 diff now; checking how Go's exec.Cmd.Wait reports a WaitDelay expiry, since that decides cleanup-unconfirmed.

Confirmed one bug. Now probing the timeout path, where an escaped grandchild holds the pipe.

Probe confirms the timeout path also misclassifies. Removing the probes and running the full suite.

```verdict
verdict: REWORK
confidence: high
```

M1 delivers most of what it set out to do. The policy is parsed and validated before any step runs, with a CUE drift guard. `boundedLog` is capped and counted. `classifyOutcome` is one pure table. Cancellation reaches every executor and decorator, the leaf-budget acquire can be cancelled without losing a slot, and the legacy launch path has a process group, a live log and `outcome.json`. The suite passes, apart from 15 failures that all come from the review sandbox: `git init` is refused outside the reviewer's directories. The #72 tests also pass under `-race -count=3`.

One correctness bug blocks SHIP. The legacy path decides "cleanup-unconfirmed" from `errors.Is(werr, exec.ErrWaitDelay)`. But Go's `Cmd.Wait` (go1.27 `os/exec/exec.go:945,962`) only returns `ErrWaitDelay` when the child exited successfully. In every other case the `ExitError` wins and the pipe-drain expiry is silently dropped. So an escaped grandchild still holding the log pipe is reported as `failed`, `timed-out` or `cancelled`, never `cleanup-unconfirmed`. That contradicts plan decision 5 and the state table row "KILLed but join exceeds the bound → cleanup-unconfirmed". I reproduced it in the disposable checkout, then removed the probes:
- An escaper variant that exits 1 gave `outcome=failed`.
- An escaper variant with `with.exec.timeout: 300ms` gave `outcome=timed-out signal=KILL`.

The only fixture, `escaper`, exits 0, so the one case where the check works is the only case tested.

### 1. Strengths
- `cmd/metis/outcome.go:62`: `classifyOutcome` is a real pure transition table. Its precedence is pinned by an independent test (`outcome_test.go:11`) that covers every row plus the cross-precedence cases.
- `cmd/metis/prioritysem.go:88-122`: the cancellable acquire tracks each waiter's heap index, and on the popped path it passes the slot on. `TestPrioritySem_HandoffRacesCancel` uses the `onCancelObserved` hook to force the handoff-races-cancel ordering deterministically, so it is not one random sample. A seeded stress test checks slot conservation on top.
- `cmd/metis/steplog.go`: the writer always consumes its whole input and never returns an error, so a full disk or a dead file never blocks the child on a full pipe. Sink errors are kept and surfaced through `Err`. The counted test runs at N and 2N.
- The fixtures are driven by a ready-marker file and a FIFO instead of sleeps. Pulling `reservedStepFiles` (`exec.go:179`) out of the inline list keeps artifact exclusion in one place.

### 2. Critical findings
- **The cleanup-unconfirmed check misses every non-zero exit** (`cmd/metis/legacyexec.go:132-134`, and `terminate` at :152-166).
  - **Why:** `ErrWaitDelay` is hidden behind `ExitError` whenever the child exits non-zero or is killed by a signal. That covers every timeout and cancel ending.
  - **Fix sketch:** track whether the log copy finished separately from Wait's error. One option is to own the pipe with `os.Pipe` plus your own copier goroutine and a done channel: after exit (or KILL), `joined` means the copier finished within `joinBound`; otherwise close the read end and set `joined=false`. A cheaper alternative is to detect a pipe-drain timeout from the time between the process exiting and `Wait` returning, which is less clean.
  - **Tests to add:** an escaper fixture that exits non-zero, and a timeout case that leaves an escaper behind. Both must give `cleanup-unconfirmed`.
  - **ARCH-ORDER:** an unconfirmed outcome is being collapsed into a confirmed failure.

### 3. Important findings
None.

### 4. Minor findings
- **Start failure bypasses the classifier** (`legacyexec.go:96-99`): it writes a bare `Outcome{failed}`, sets `exitCode` and then never uses it. An `os.Create` failure on `step.log` (:82-85) writes no `outcome.json` at all, which contradicts "written on every executed step". ARCH-ORDER.
- **`readPending` treats an unreadable pending-resources file as "none"** (`legacyexec.go:170-178`). If `pending-resources.json` exists but can't be read (permissions, or it's a directory), the step can be published as `ok`. It should fail visibly instead. ARCH-SECURE.
- **The forkserver path ignores `with.exec.timeout` at this head and writes no `outcome.json`** (`exec.go:113-130`). Meanwhile the atlas (`atlas/experiment.md`) states the timeout and process-group behaviour without saying it is legacy-only. Either qualify the atlas line until M2 lands, or have M1 refuse a timeout on forkable steps. ARCH-CONSTRAINTS.
- **`runtime_seconds` includes termination time**: `ended` is stamped after `terminate`, but plan decision 7 says runtime runs from launch to the child's exit.
- **Possible data races when the join is unconfirmed**: if the child itself survives KILL, the Wait goroutine is still running while `cmd.ProcessState`, `log.Tail()` and `log.Truncated()` are read. This is rare, because the escaped-holder case closes the pipes first.
- **`step.log` adds up to 64 MiB of disk per executed step per run**, and the only removal path is the operator deleting run directories by hand. The plan names "the existing runs/ lifecycle", but nothing removes them automatically. ARCH-FUNERAL.

### 5. Test coverage notes
- Every plan-listed M1 fixture exists and passes: live log, TERM, KILL, escaper, pending resources, queued cancel, and ordinary success. Running the #72 tests under `-race` was clean.
- The gap is the escaper fixture. It only pins the exit-0 case, which is the single case where the `ErrWaitDelay` check works. That is exactly the kind of bug this diff shipped.
- `pkg/experiment` passes in full, including `TestExecPolicyConformsToCUE` and the Validate refusals.

### 6. Architecture pass (each principle)
- **ARCH-DRY:** pass. `reservedStepFiles` is extracted, and the parser follows #71's pattern.
- **ARCH-PURE:** pass. `ParseExecPolicy`, `classifyOutcome` and `boundedLog` are tested without IO; `runLegacy` is glue.
- **ARCH-PURPOSE:** pass. The forkserver is M2 by plan, not a deferred core.
- **ARCH-MOCK:** pass. Tests use real processes with readiness control, as the plan declared.
- **ARCH-CONSTRAINTS:** flag (Minor). The forkserver silently runs without the timeout in M1.
- **ARCH-SECURE:** flag (Minor). An unreadable pending-resources file degrades to "none".
- **ARCH-ORDER:** flag (Critical). The join is reported confirmed without evidence. The start-failure bypass is Minor.
- **ARCH-FUNERAL:** flag (Minor). Per-step log growth with only operator cleanup.
- **ARCH-EVOLVE:** pass. `outcome.json` is new, additive JSON and nothing reads it yet.

For M2: the forkserver's join check must not repeat this mistake. The cancel/response protocol should report "pipe drained" separately from the exit status.

### 7. Plan revision recommendations
None, provided the fix lands. The plan's state table is correct and the code has to match it.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: uncertain-outcome-collapsed
    title: |
      Legacy cleanup-unconfirmed relies on ErrWaitDelay, which Go hides behind ExitError on any non-zero or signal exit
    detail: |
      legacyexec.go:132 sets joined=false only when errors.Is(werr, exec.ErrWaitDelay), but Cmd.Wait (go1.27 exec.go:945,962) returns ErrWaitDelay only when the child exited successfully. An escaped grandchild holding the pipe after a non-zero exit, a timeout or a cancel is therefore classified failed/timed-out/cancelled, not cleanup-unconfirmed (reproduced: an escaper that exits 1 gave outcome=failed; an escaper under timeout 300ms gave timed-out/KILL). Track the pipe drain separately from Wait's error (own os.Pipe plus a copier done channel bounded by joinBound) and add escaper fixtures for a non-zero exit and for a timeout.
  - id: new
    severity: Minor
    kind: defect
    family: single-outcome-decider-bypassed
    title: |
      Start failure writes a bare failed Outcome outside classifyOutcome; a step.log create failure writes no outcome.json
    detail: |
      legacyexec.go:96-99 and :82-85. Route both through classifyOutcome/record so every executed step gets a consistent outcome.json.
  - id: new
    severity: Minor
    kind: defect
    family: untrusted-input-degrades-visibly
    title: |
      readPending treats an unreadable pending-resources.json as no pending resources
    detail: |
      legacyexec.go:170-178 discards non-ENOENT open/read errors (permissions, a directory), so a step can publish ok while resources may be live. Treat any error other than not-exist as resources-unretired or a step error.
  - id: new
    severity: Minor
    kind: defect
    family: docs-claim-exceeds-code
    title: |
      The forkserver path ignores with.exec.timeout and writes no outcome.json until M2, but the atlas states the timeout without that limit
    detail: |
      exec.go:113-130 ignores the policy. Qualify the atlas/experiment.md execution-policy bullet as legacy-only until M2, or refuse a timeout on forkable steps in the meantime.
  - id: new
    severity: Minor
    kind: defect
    family: runtime-vs-wait-accounting
    title: |
      runtime_seconds includes termination time, not launch-to-exit as plan decision 7 states
    detail: |
      end.ended is stamped after terminate() returns, so grace and join time count as compute runtime.
  - id: new
    severity: Minor
    kind: defect
    family: concurrent-extent-unbounded
    title: |
      Unconfirmed join reads cmd.ProcessState and boundedLog while the Wait goroutine may still be writing them
    detail: |
      When terminate returns joined=false, the goroutine running cmd.Wait is still live; the later reads are a data race (rare, only when the child itself survives KILL).
  - id: new
    severity: Minor
    kind: defect
    family: artifact-removal-path
    title: |
      step.log adds up to 64 MiB per executed step per run, with only operator-triggered run-dir removal
    detail: |
      The plan names the existing runs/ lifecycle, but nothing removes runs automatically; per-launch disk growth is new with this diff.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"97213f35-12e5-4dd9-b34d-7308bd7d2b51","repository":"github.com/xianxu/metis","issue_id":"000072","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/847240254a2afa50","reviewed_head":"77863b0613972bcff58aaf52a829748b17ae41e8","slot":"metis:1","requested_at":"2026-10-25T16:15:41.396739Z"},"phase":"finished","pid":14913,"started_at":"2026-10-25T16:15:41.605229Z","finished_at":"2026-10-25T16:21:34.024474Z","session_id":"bc82caaa-01b2-493f-8689-185193b4858f","outcome":"returned"}] -->

---

## Re-review — 2026-10-25T09:29:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 72 — Bound step execution with timeout and live logs |
| repo | metis |
| issue file | workshop/issues/000072-step-timeout-live-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 99ccccb46a2707de6fbfe1be44c32263f0e636cf..b1765c8b68774d63ea4a8f0ac172d6e453a158aa |
| command | sdlc milestone-close --issue 72 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T09:29:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Read the core change and tests; now running the suite and a revert check on the BR-1 fix.

Now the revert checks: mutate the drain-based join (BR-1), readPending (BR-3), runtime end (BR-5).

Revert checks done (BR-1, BR-3 go red; BR-5 doesn't). Checking the cancellable semaphore and remaining callers.

Wrapping up: checking the CUE drift guard claim (Task 1), then writing the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Critical finding from round 1 (BR-1) is fixed, and a revert check proves it. metis now owns the output pipe and runs its own copier, so it decides the join itself instead of relying on `exec.ErrWaitDelay`. If I put back the old `ErrWaitDelay` check, all three escaper subtests fail (exit 0, exit 1, timeout); with the fix they pass. BR-2, BR-3, BR-4 and BR-7 are also resolved. BR-6 is fixed by inspection, but no fixture can make it fail. BR-5 is fixed in the code, but its regression test is circular: it still passes if `ended` is stamped after the join, so I'm marking BR-5 not-addressed (Minor). I'm raising two new Minor findings. Nothing blocks the gate.

How I tested: I ran the targeted tests and `pkg/experiment` with `-race` (all green), using a temporary `go.mod` replace pointing at the pinned ariadne substrate. I didn't run `go vet` or the full `cmd/metis` suite (only the `-run` subset). The checkout was restored afterwards (`go.mod` and `legacyexec.go` reverted; `git status` clean).

**1. Strengths**
- `cmd/metis/legacyexec.go:110-160`: the pipe and copier belong to metis, and the copier is joined (`<-drained`) before the log is read. The join is now based on evidence, and the read is race-safe on every path.
- `cmd/metis/outcome.go:62`: `classifyOutcome` is the only place that decides success. It now also handles failures before launch (`launchErr`), so start failures and log-create failures go through the same table.
- `cmd/metis/prioritysem.go:85-122`: cancellation while queued gives the slot back through `release()` if a handoff raced in first. A forced-ordering hook (`onCancelObserved`) makes that interleaving reproducible.
- The test fixtures are readiness-controlled (marker file plus FIFO), so tests don't depend on sleep timing.
- `pkg/experiment/execpolicy_test.go:17`: one case table drives both the Go parser tests and the CUE drift guard (`cue` is installed here and the guard ran).

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- **Owned processes left running (`legacyexec.go:139-152`).** If a step exits on its own but leaves a backgrounded process that is still in its process group (for example `cmd &` without `setsid`) holding stdout, metis never signals the group. After `joinBound` it reports cleanup-unconfirmed and leaves a process it owns running. **Fix:** if the drain doesn't finish, TERM/KILL the group (`-pid`) before giving up. This is safe: a group's ID isn't reused while any member is alive.
- **Atlas guarantee needs a precondition (`atlas/experiment.md`).** The atlas says "metis never kills" independently managed resources. But if such a resource inherited the step's stdout or stderr, every step ends cleanup-unconfirmed, and `pr.Close()` makes the resource's next write fail with EPIPE/SIGPIPE. **This is the 2nd finding in family `docs-claim-exceeds-code`.** The rule: every behavioral promise in the #72 atlas section states the conditions it depends on (launch path, stdio). Here that means telling step authors that resources must detach from the step's stdio. Sweep the whole section against that rule.
- **BR-5 test is circular.** `TestExecStep_RuntimeExcludesTermination` checks `ended - started == runtime`, but `classifyOutcome` computes runtime from exactly those two fields. A meaningful check: for the exit-0 escaper, runtime is well under `joinBound`.
- `logFile.Close()` is deferred and its error is dropped. The `fs.ErrNotExist`-vs-other branch in `readPending` for the open error (as opposed to the read error) has no fixture.

**5. Test coverage notes**
- Revert checks:
  - BR-1: reverting goes red (3 of 3 subtests).
  - BR-3, read-error branch: reverting goes red.
  - BR-3, open-error branch: no test covers it.
  - BR-5: stamping `ended` after the join stays green, so the test doesn't catch it.
  - BR-6: removing `<-drained` stays green under `-race`, because no fixture leaves the copier racing.
- In the legacy path, the race between the child's exit and the deadline is decided by Go's random `select` choice, and no test forces the order. The plan schedules that deterministic test for M2's forkserver path, so it's acceptable for M1.

**6. Architecture**
- ARCH-DRY: pass. `reservedStepFiles` was extracted; there is one classifier and one log writer.
- ARCH-PURE: pass. Parse, classify and bound are pure; `runLegacy` is glue.
- ARCH-PURPOSE: pass. M1 delivers the legacy path, cancellation and the resource contract; the forkserver is a separate milestone and the atlas says so.
- ARCH-MOCK: pass. Real processes with readiness control, as the plan justifies.
- ARCH-CONSTRAINTS: pass. Cap, tail, grace and join bound are enforced.
- ARCH-SECURE: pass. The pending-resources file is read with a size bound and treated as opaque; an unreadable file shows up visibly.
- ARCH-ORDER: pass, with the deterministic exit-vs-deadline test deferred to M2. The Wait goroutine outliving `Execute` when the child survives KILL is documented and reported as cleanup-unconfirmed.
- ARCH-FUNERAL: flag. See the first Minor finding (owned processes left running after a normal exit). The disk residue is documented.
- ARCH-EVOLVE: pass. `outcome.json` is new and additive.

**7. Plan revisions to record**
- Add a "## Revisions" entry saying a normal exit with an in-group descendant still holding stdout ends cleanup-unconfirmed, or KILLs the group once the fix above is applied. The state design table currently covers only the escaped-descendant case.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Owned os.Pipe + copier drain within joinBound; reverting to the ErrWaitDelay check fails all three escaper subtests (exit 0, exit 1, timeout).
  - id: BR-2
    disposition: addressed
    note: |
      Start and log-create failures route via notLaunched -> classifyOutcome -> writeOutcome (legacyexec.go:72-75); TestExecStep_StartFailureRecordsOutcome covers start.
  - id: BR-3
    disposition: addressed
    note: |
      readPending reports non-ENOENT errors as pending; reverting the read-error branch fails TestExecStep_UnreadablePendingResourcesRefuseSuccess (directory fixture).
  - id: BR-4
    disposition: addressed
    note: |
      atlas/experiment.md now states that only the legacy path honors with.exec until M2 (forkserver: no timeout, step.log or outcome.json), which matches exec.go:113-138.
  - id: BR-5
    disposition: not-addressed
    note: |
      Code fix is real (ended = Wait return time), but TestExecStep_RuntimeExcludesTermination stays green when ended is stamped after the join; add an exit-0 escaper assertion runtime < joinBound.
  - id: BR-6
    disposition: addressed
    note: |
      Copier joined unconditionally before log reads; ProcessState read only when procDone; -race clean. Revert not red because no fixture survives KILL, but the fix is structural.
  - id: BR-7
    disposition: addressed
    note: |
      Atlas now states the explicit bound (steps x 64 MiB per run) and removal with the run dir, which ARCH-FUNERAL accepts as a stated growth bound.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: owned-process-left-running
    title: |
      Normal exit with an in-group descendant holding stdout leaves that owned process running
    detail: |
      legacyexec.go:139-152 signals the process group only on deadline or cancel; after a self-exit whose backgrounded in-group child holds the pipe, metis reports cleanup-unconfirmed after joinBound and never TERM/KILLs a group it created. Signal -pid before giving up on the drain.
  - id: new
    severity: Minor
    kind: defect
    family: docs-claim-exceeds-code
    title: |
      Atlas promise that metis never kills external resources omits the stdio precondition
    detail: |
      2nd finding in this family. Rule: every behavioral promise in the metis#72 atlas section states its preconditions (launch path, stdio). A resource that inherited the step's stdout or stderr makes every step cleanup-unconfirmed, and pr.Close() gives it EPIPE/SIGPIPE on its next write; say that resources must detach from the step's stdio, and sweep the section.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"cdaa240d-4b88-4083-b34b-ae9eb0c7cd65","repository":"github.com/xianxu/metis","issue_id":"000072","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/2a0cf0c1f9c7b837","reviewed_head":"b1765c8b68774d63ea4a8f0ac172d6e453a158aa","slot":"metis:1","requested_at":"2026-10-25T16:25:20.553215Z"},"phase":"finished","pid":37611,"started_at":"2026-10-25T16:25:20.763411Z","finished_at":"2026-10-25T16:29:38.585147Z","session_id":"f0c33243-3c64-48a8-8929-4663ce506afe","outcome":"returned"}] -->
