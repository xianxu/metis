# Boundary Review — metis#72 (milestone M2)

| field | value |
|-------|-------|
| issue | 72 — Bound step execution with timeout and live logs |
| repo | metis |
| issue file | workshop/issues/000072-step-timeout-live-log.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | d15c25c44686384dbf3b10916473d6ee27a5d825..267d16c72683a4285c302d948985c9301421553b |
| command | sdlc milestone-close --issue 72 --milestone M2 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T09:53:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M2 forkserver diff; now reading the Go lifecycle (steplifecycle.go) and tests.

Checking a pre-existing mid-flight-death test for orphaned children now that forked children lead their own groups.

Running the Go forked-step tests and the Python forkserver tests to confirm the suite is green at head.

Probing the server's exception path empirically: a `log_path` that can't be opened (a directory).

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what the plan asked for. Forked children lead their own process groups. Output streams live into a capped `step.log`. The server is the only process that signals or reaps a child, through a `cancel` message. The ready wait can be cancelled and never falls back to legacy. `runForked` uses the same budget, policy, `classifyOutcome` and `outcome.json` as the legacy path. At head, 30 Go tests pass (`TestForkedStep_*`, `TestForkServerPool_*`, `TestExecStep_*`, `TestClassifyOutcome_*`) and `tests/test_forkserver.py` passes 14/14. I ran the Go tests through a temporary `go.work` pointing at the pinned ariadne substrate, because the `../ariadne` replace directory is missing in this checkout.

Three Important defects stand between this and SHIP. They all sit on the server's error paths and one Go dispatch path:
- When the waiter throws, the forked child is left running and unreaped, and the response invents an exit code. I reproduced this.
- When the reader thread fails, the server still reports the join as confirmed.
- When the request write fails after the server has died, `outcome.json` gets a zero `started` time and a runtime of about 6e10 seconds.

Each is cheap to fix.

## 1. Strengths
- **No recycled-pid kills.** `metis/forkserver.py` signals a group only while its leader is still unreaped. Insert, cancel and reap all happen under the one protocol lock. Since an unreaped leader keeps its pid and group id reserved, a cancel can never hit a recycled pid. This is the right shape for decision 4.
- **Exit detection without reaping.** The waiter sees the exit via `waitid(WNOWAIT)`, or kqueue `NOTE_EXIT` on macOS, and runtime stops at the exit. `test_escaped_holder_is_unjoined_within_bound` pins that the join wait is not counted as runtime.
- **No fallback after dispatch, at every layer.** Each outcome of `serverPool.execute` (`cmd/metis/forkexec.go:431-445`) maps to exactly one of: fall back, error the step, or cancelled before dispatch. `TestForkedStep_CancelDuringReadyWaitNeverLaunches` and `TestForkedStep_ServerDiesDuringCancel` prove it with real processes.
- **One classifier for both paths.** `exitKnown` keeps a late exit 0 on record without letting it decide the outcome. `TestForkedStep_LateExitZeroStaysTimedOut` pins that.
- **Bounded Go wait.** After KILL, Go waits `2·joinBound + grace + 1s`, which covers the server's own post-exit join (`forkexec.go:331`). So "no response" really means the server is unresponsive, not just slow.

## 2. Critical findings
None.

## 3. Important findings

**I-1 (ARCH-ORDER / ARCH-FUNERAL, family `owned-process-left-running`, 2nd finding).** In `metis/forkserver.py` `_wait`, if `_wait_inner` raises (for example `_Log` cannot open `log_path`), the handler answers `exit: -1, joined: false` and returns. It never signals or reaps the child, and never removes it from `_children`.
- I reproduced it: with `log_path` set to a directory, the response arrived in 0.01s and the child was still alive afterwards. It had no reader, so a chatty child would block on a full pipe forever.
- On the Go side, `runForked` sets `exitKnown=true, exitCode=-1` from that response, so `outcome.json` records an exit code that was never observed.
- **Rule:** every exit path of the component that owns a process group must end in TERM → grace → KILL → reap, or in an explicit hand-off. It is never just a return.
- **Fix:** in `_wait`, use try/finally. If the child is still in `_children`, stop its group, reap it under the lock, and delete the entry. Separately, mark the response as "exit unobserved" (for example omit `exit` or add `"exited": false`) and map that to `exitKnown=false`. Also consider opening the log before the fork so a log-create failure becomes "never launched", as `runLegacy` does.

**I-2 (ARCH-ORDER / ARCH-DRY, family `uncertain-outcome-collapsed`, 2nd finding).** In `forkserver.py`, the `read()` thread's `finally` sets `drained` on any exit, so an exception from `log.write` (ENOSPC, EIO) counts as a confirmed join. Then:
- The response says `joined: true`, so the step can be recorded `ok` with a short `step.log` and no reported error.
- The read end is closed, so the child's next write gets SIGPIPE. Disk pressure would kill a long Couch session.
- Go's `boundedLog` does the opposite: it keeps draining, keeps the first sink error, and reports it on `out`.

**Rule:** `joined` may only be asserted from EOF actually observed on the owned pipe; every other way the reader ends is unconfirmed. **Fix:** set a separate `eof` event only on `b == b""`. Make `_Log.write` keep the first sink error and keep draining, mirroring Go. Return the error in the response.

**I-3 (ARCH-SECURE / ARCH-EVOLVE, family `runtime-vs-wait-accounting`, 2nd finding).** When the request write fails, `forkServer.execute` returns an empty `forkRun{}` with `errDispatchedLost` (`cmd/metis/forkexec.go:296-300`), before `run.started` is set. This happens when stdin hits EPIPE because the server died before `readLoop` marked it dead. `runForked` then records `launched=true`, `started=0001-01-01` and `ended=now`, so `runtime_seconds` is about 6.4e10. That is an invented value in a persisted record that #73/#74 will read.

**Rule:** `started`, `ended` and `runtime_seconds` are written only from an observed dispatch or launch and an observed exit. When either is unknown, the field is omitted, never derived from a zero value. **Fix:** set `run.started` before the write, or carry `launchTimeKnown` in `stepEnding` and have `classifyOutcome` omit the times when it is false. Add a classifier table row for it.

## 4. Minor findings
- **Stale comments (family `docs-claim-exceeds-code`, 3rd finding).**
  - **Rule:** when process-group ownership changes, every statement about what a group signal reaches must be re-derived. Sweep `grep -n "group" cmd/metis/forkexec*.go`.
  - `forkexec.go:160-165` and `:364` still say killing the server's group reaps "the whole tree", but forked children now lead their own groups.
  - The comment in `TestForkServerPool_MidFlightDeathErrorsTheStep` relies on the same old claim.
  - The `logMissing` message "a server older than this metis?" can never show for a stale server. Such a server omits `joined`, which decodes to false, and cleanup-unconfirmed takes precedence. Measured prevalence of the family on this issue so far: 3.
- **ARCH-CONSTRAINTS:** the reader uses `select.select` with a 50 ms poll in each child thread. That is 20 wakeups/s per in-flight child, and `select` fails on fd ≥ 1024. `selectors`/`poll`, or a self-pipe to stop the reader, would remove both.
- **ARCH-FUNERAL:** when a cancel goes unanswered, `s.pending[id]` is never deleted, and the broken root's server keeps running until the pool shuts down.
- **ARCH-DRY:** the `record` closure is identical in `runLegacy` and `runForked` (`steplifecycle.go`). It could be a `recordOutcome(stepDir, end)` helper.
- A late result after a timeout or cancel is logged only when it is exit 0. A late non-zero exit is silently recorded. That is acceptable, but it should be stated.

## 5. Test coverage notes
- No Go test cancels the context while a forked step is running; only cancellation during the ready wait is covered.
- No `runForked` test feeds a `joined:false` response (escaped holder) or a pending-resources file through the forked path. Python pins the server side only.
- Nothing exercises the server's error paths (I-1, I-2) or the write-failure dispatch-lost path (I-3). Each fix needs a test that fails without it. A directory as `log_path` reproduces I-1 cheaply.
- `TestForkServerPerf_LooseBound` was seen to fail once under load (according to the Log). It should be registered as a flake or have its bound relaxed.

## 6. Architectural notes
- **ARCH-DRY:** pass, apart from the `record` nit. `_Log` mirrors `boundedLog` across languages by necessity. I-2 shows the risk of that mirroring, so a shared conformance test (same input → same file and tail, including a sink failure) would help.
- **ARCH-PURE:** pass. The classifier stays pure, and `runForked` is glue.
- **ARCH-PURPOSE:** pass. Both launch paths honor `with.exec`, the live log, `outcome.json`, and the no-duplicate rule.
- **ARCH-MOCK:** pass. Tests use the real uv server and readiness-controlled toy modules.
- **ARCH-CONSTRAINTS:** flag (Minor, polling and `select` above).
- **ARCH-SECURE:** flag (I-3). The response's `runtime` field is also used without validation.
- **ARCH-ORDER:** flag (I-1, I-2). The transition table is respected in Go. The server's waiter breaks the "exit, then confirmed join" ordering on its error paths.
- **ARCH-FUNERAL:** flag (I-1, plus the Minor pending-map leak). `step.log` and `outcome.json` residue is as M1 already documented.
- **ARCH-EVOLVE:** pass with a note. The protocol change is additive, but fields missing from an older server decode as `false`, so the stale-server diagnosis cannot be reached (Minor above).

## 7. Plan revision recommendations
- Add a `## Revisions` entry: a forked step's ending whose exit was not observed (waiter failure, or the request write failing after dispatch) records no `exit_code` and no `started`/`runtime_seconds`. It is cleanup-unconfirmed.
- Add a `## Revisions` entry: the state table's stale-server row records cleanup-unconfirmed, because a missing `joined` means the join is unconfirmed. The separate logMissing path applies only to a current server that wrote no log.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: owned-process-left-running
    title: |
      forkserver waiter exception path leaves the forked child running and unreaped, and fabricates exit -1
    detail: |
      metis/forkserver.py _wait answers exit -1 joined false on any _wait_inner exception but never signals, reaps, or removes the child from _children; reproduced with log_path set to a directory (response in 0.01s, child still alive after). Go runForked then sets exitKnown=true exitCode=-1, so outcome.json records an unobserved exit code. Rule: every exit path of a process-group owner ends in TERM/grace/KILL plus reap or an explicit hand-off; fix with try/finally in _wait, an exit-unobserved response mapped to exitKnown=false, and ideally opening the log before fork.
  - id: new
    severity: Important
    kind: defect
    family: uncertain-outcome-collapsed
    title: |
      forkserver reader failure (log write error) is reported as a confirmed join
    detail: |
      read() sets drained in its finally, so an exception from _Log.write (ENOSPC/EIO) yields joined=true, closes the pipe (the child then gets SIGPIPE), and the error is never reported, unlike Go boundedLog, which keeps draining and reports the sink error. Rule: joined only from observed EOF on the owned pipe; set a separate eof event, keep the first sink error and keep draining, and return the error in the response.
  - id: new
    severity: Important
    kind: defect
    family: runtime-vs-wait-accounting
    title: |
      request-write failure after dispatch records started=zero time and runtime of about 6e10s in outcome.json
    detail: |
      forkServer.execute returns forkRun{} with errDispatchedLost on a stdin write error (forkexec.go:296-300) before run.started is set; runForked marks launched=true with started zero and ended=now. Rule: times are written only from an observed dispatch/launch and exit; unknown times are omitted (carry launchTimeKnown, or set started before the write), with a classifier table row and a test.
  - id: new
    severity: Minor
    kind: defect
    family: docs-claim-exceeds-code
    title: |
      stale claims that killing the server's group reaps the whole tree, and an unreachable stale-server diagnosis
    detail: |
      forkexec.go:160-165 and :364 and the MidFlightDeath test comment predate forked children leading their own groups; the logMissing message is unreachable for a stale server because missing joined decodes false and cleanup-unconfirmed wins. Rule: when process-group ownership changes, re-derive every statement of what a group signal reaches (sweep the group comments in forkexec*.go). Measured prevalence on this issue: 3 findings in this family.
  - id: new
    severity: Minor
    kind: defect
    family: concurrent-extent-unbounded
    title: |
      per-child reader polls select() at 50ms and select fails for fd 1024 or higher
    detail: |
      Use selectors/poll or a self-pipe wakeup to stop the reader; 20 wakeups/s per in-flight child otherwise.
  - id: new
    severity: Minor
    kind: defect
    family: artifact-removal-path
    title: |
      unanswered cancel leaves s.pending[id] and the broken root's server alive until pool shutdown
    detail: |
      Delete the pending entry when execute returns without a response; consider shutting down a root's server when it is marked broken.
  - id: new
    severity: Minor
    kind: defect
    family: forked-path-test-gaps
    title: |
      no Go test for ctx cancel while a forked step runs, for a joined=false response, or for pending resources via runForked
    detail: |
      Python pins the server side only; the Go mapping of these endings through runForked is untested.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a7764c92-257d-4642-a552-627a1863ee2d","repository":"github.com/xianxu/metis","issue_id":"000072","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/1b21b96ca693ad6f","reviewed_head":"267d16c72683a4285c302d948985c9301421553b","slot":"metis:1","requested_at":"2026-10-25T16:47:44.708739Z"},"phase":"finished","pid":54327,"started_at":"2026-10-25T16:47:44.808048Z","finished_at":"2026-10-25T16:52:57.895018Z","session_id":"9c94df14-3b5e-4fed-900d-5ace567da90b","outcome":"returned"}] -->

---

## Re-review — 2026-10-25T10:09:11-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 72 — Bound step execution with timeout and live logs |
| repo | metis |
| issue file | workshop/issues/000072-step-timeout-live-log.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | d15c25c44686384dbf3b10916473d6ee27a5d825..295193018d799d0af518256547a5f0dec04e852a |
| command | sdlc milestone-close --issue 72 --milestone M2 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T10:09:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing: checking the legacy path's sink-error handling, then I'll run the tests.

Reproducing a suspected hang in the forkserver's error path; fixing my test script and re-running.

Confirmed the hang. Now checking whether the BR-12 fix has a test that fails without it.

The BR-12 revert passes every test, so nothing pins that fix. I'm also checking an earlier one-off failure for flakiness.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Most of the round-3 fixes hold up. The log is now opened before fork, so a `step.log` that won't open is refused with `launched=false` and nothing runs. A join is confirmed only by EOF on the child's pipe, a log write error is kept and reported while draining continues (the same as Go's `boundedLog`), and `classifyOutcome` writes `ended`, runtime and exit code only when the exit was actually observed. The forkserver tests pass (18), and so does every fork, lifecycle and classify test in Go. Two Important findings stay open:

- **BR-10:** the waiter's error path still does not always end its child. I reproduced it: after an injected waiter failure, a child that ignores TERM survives, and the server then blocks while holding its protocol lock.
- **BR-12:** the zero-start-time fix has no test. I moved the fix back to its old position and every test still passed.

Both are cheap to fix.

1. **Strengths**
   - `_pump` (`metis/forkserver.py:116-130`): one selector plus a wake pipe. It returns True only on EOF, which removes the 50ms polling and the fd-1024 limit. It is tested directly for both the EOF and the woken case.
   - The ready line now carries a protocol version (`checkForkProtocol`, `cmd/metis/forkexec.go:84-94`). A stale server is refused before anything is dispatched. This replaces the `logMissing` check, which could never fire.
   - `classifyOutcome` keeps the "times only from observed events" rule in one pure place (`cmd/metis/outcome.go:64-75`), with a pure table test.
   - The `pending` entry is removed in a `defer` on every return after dispatch (`forkexec.go:334-338`).
   - New Go tests go through the real `runForked`: cancel mid-run, a `joined=false` answer, pending resources, and a refusal before fork.

2. **Critical findings:** none.

3. **Important findings (not-addressed)**
   - **BR-10** (`metis/forkserver.py` `_wait`, the `except` and `finally` blocks). In the error path, KILL is skipped when `done` is set, but `done` only means the pipe hit EOF. It does not mean the child exited. The `finally` block then calls `os.waitpid(pid, 0)` while holding `lock`.
     - Reproduced in-process: I made `_wait_exit_noreap` raise, used a child that ignores TERM and has its stdout pointed at /dev/null, and set grace=0.5. After 3s the waiter was still alive with the lock held, and it answered only when the child exited on its own after 8s.
     - While that child lives, every other response and every cancel is blocked.
     - If anything inside `finally` raises (`waitpid`, `os.close`), no response is sent at all. Without a step timeout, Go then waits forever. The old blanket `except` always answered.
     - Fix: in the error path, escalate based on the child's exit, not the pipe (KILL after grace unless the exit was observed). Wait for the exit outside the lock and take the lock only for the reap. Guard the reap-and-respond step so an answer is always sent.
     - Add a test that injects a waiter failure, like my reproduction.
     - Third finding in family `owned-process-left-running`. The rule: a process-group owner escalates TERM, then grace, then KILL based on its child's observed exit, and never blocks under a shared lock.
   - **BR-12:** the code fix is in place (`forkexec.go:322`), but no test fails without it.
     - Fix: add a Go test that closes `srv.stdin` while the reader is still alive, calls `execute`/`runForked`, and asserts `errDispatchedLost`, a non-zero `run.started`, and an `outcome.json` whose `started` is recent and has no `ended`.

4. **Minor findings**
   - `_wait` writes out the TERM, grace, KILL sequence twice, and the two copies already differ (ARCH-DRY). One shared helper would also fix BR-10.
   - `TestForkServerPerf_LooseBound` failed once in four runs here (13s). The issue log already says it's watched; it should be registered as a flake or have its bound loosened.

5. **Test coverage notes**
   - Go test environment: `go.mod` points at `../ariadne`, which isn't in the checkout. I ran the tests with a temporary modfile pointing at the pinned dependency clone.
   - Four tests that call `git init` fail because this review environment refuses `git init` outside its directories. That is the environment, not the code.
   - Not covered by any test:
     - the error path in `_wait` (BR-10)
     - a request-write failure after dispatch (BR-12)
     - the `defer` that removes the `pending` entry (BR-15; I accept it on inspection)

6. **Architectural notes**
   - Every principle was checked:

     | Principle | Result | Note |
     |---|---|---|
     | ARCH-DRY | flag | Minor: the duplicated sequence above |
     | ARCH-PURE | pass | |
     | ARCH-PURPOSE | pass | |
     | ARCH-MOCK | pass | Real uv server, no external service |
     | ARCH-CONSTRAINTS | pass | The polling is gone |
     | ARCH-SECURE | pass | `runtime` is a pointer and its absence is handled visibly |
     | ARCH-ORDER | flag | BR-10 |
     | ARCH-FUNERAL | pass | The wake-pipe fds are closed. A broken root's server lives until pool shutdown; that is documented and bounded |
     | ARCH-EVOLVE | pass | The protocol field is additive, and refusing an older server has a concrete reason: it would silently drop `log_path` and `cancel` |

   - A server refused for an old protocol stays in the pool, and every step on that root checks again and falls back. Consider marking the root as falling back to legacy, so it isn't restarted or kept warm for nothing.

7. **Plan revision recommendations:** none beyond what's already logged. The round-3 revision already replaces the plan's EVOLVE wording ("no version negotiation") with the protocol check.

```findings
dispose:
  - id: BR-10
    disposition: not-addressed
    note: |
      Refusing a bad log before fork is fixed and tested. Reproduced: an injected waiter failure with a TERM-ignoring child whose stdout is /dev/null skips KILL because the pipe EOF sets done, then waitpid blocks for the child's whole life while holding the protocol lock (alive and locked at 3s, answered at 8s). If anything in finally raises, no answer is sent. No test covers the error path. Rule: TERM/grace/KILL escalates on the observed exit, never blocks under the lock, and always answers.
  - id: BR-11
    disposition: addressed
    note: |
      _pump returns True only on EOF and _Log._put keeps the first error while consumption continues. test_log_sink_error_keeps_draining_and_reports and test_pump_stopped_by_wake_is_not_eof pin it, log_error is returned and printed as on the legacy path, and joined is eof and not error.
  - id: BR-12
    disposition: not-addressed
    note: |
      The code fix is present: forkRun.started is set before the write (forkexec.go:322) and classifyOutcome omits end times without an observed exit (tested). Moving started back after the write leaves every Fork/Classify/Lifecycle test green, so nothing pins the write-error path. Needed: a test that closes srv.stdin with the reader alive and asserts errDispatchedLost plus a recent started and no ended in outcome.json.
  - id: BR-13
    disposition: addressed
    note: |
      The group comments at forkexec.go:183-189 and :399-400 and the MidFlightDeath test comment now say forked children lead their own groups. logMissing and its table row were removed in favour of checkForkProtocol, which TestCheckForkProtocol and test_ready_line_declares_protocol pin.
  - id: BR-14
    disposition: addressed
    note: |
      The reader is a selectors.DefaultSelector with a wake pipe (forkserver.py:116-130): no 50ms timeout, no select() fd limit. test_pump_stopped_by_wake_is_not_eof covers the wake.
  - id: BR-15
    disposition: addressed
    note: |
      A deferred delete of s.pending[id] runs on every return after dispatch (forkexec.go:334-338). The broken root's server is kept until pool shutdown, documented in the issue log as a deliberate choice.
  - id: BR-16
    disposition: addressed
    note: |
      Added TestForkedStep_CancelWhileRunning, TestForkedStep_UnjoinedResponseIsCleanupUnconfirmed, TestForkedStep_PendingResourcesRefuseSuccess and TestForkedStep_ServerRefusalBeforeForkNeverLaunches, all through the real runForked and uv server; all pass.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: owned-process-left-running
    title: |
      _wait spells out the TERM/grace/KILL sequence twice and the copies already differ
    detail: |
      The normal path escalates after an observed exit; the error copy gates KILL on pipe EOF, which is the BR-10 defect. One shared helper (ARCH-DRY) that escalates on the observed exit would fix both. Count this with BR-10, not as a separate instance.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4e4545cb-2aad-4bf9-bcba-12dd3c1c4cfb","repository":"github.com/xianxu/metis","issue_id":"000072","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/908d57c9daed13a0","reviewed_head":"295193018d799d0af518256547a5f0dec04e852a","slot":"metis:1","requested_at":"2026-10-25T17:00:56.035888Z"},"phase":"finished","pid":16313,"started_at":"2026-10-25T17:00:56.215856Z","finished_at":"2026-10-25T17:09:08.812809Z","session_id":"47a2fd0e-3ad0-40a3-bfe7-8782c01d6cb3","outcome":"returned"}] -->

---

## Re-review — 2026-10-25T10:33:55-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 72 — Bound step execution with timeout and live logs |
| repo | metis |
| issue file | workshop/issues/000072-step-timeout-live-log.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | d15c25c44686384dbf3b10916473d6ee27a5d825..e4eabd172e5c12568306341ea9994de46ce58211 |
| command | sdlc milestone-close --issue 72 --milestone M2 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T10:33:55-07:00 |
| verdict | SHIP |

## Review

Checking prior findings BR-10/12/17 against the head code; next, running the Go and Python test suites and reverting fixes to confirm the regression tests catch them.

Lifecycle tests pass; the 15 failures are git-init refusals from the review environment. Now reverting fixes to check regression tests go red.

Head file restored. Now confirming the new BR-10 tests go red against the round-3 forkserver, this time with a hard time limit.

BR-10, BR-12 and BR-17 all check out. Next: the plan's M2 checklist and the atlas diff.

```verdict
verdict: SHIP
confidence: medium
```

All three open findings are fixed. For each behavior fix I reverted it in this checkout and confirmed its regression test then fails. BR-12: if `started` moves back after the request write, `TestForkedStep_RequestWriteFailureRecordsOnlyObservedTimes` fails and the outcome shows `Started:0001-01-01`. BR-10: if the reap is skipped on the error path, `test_waiter_failure_after_exit_answers_unjoined_and_reaps` fails with `DID NOT RAISE ChildProcessError`. Run against the round-3 `forkserver.py` (fd69993), `test_exit_watcher_failure_falls_back_and_still_reaps` hangs, which is the BR-10/BR-17 lock-held defect showing up. BR-17 is gone: one `_escalate` helper (`metis/forkserver.py:208`) now drives both waiter paths and acts on the observed exit.

At head, `tests/test_forkserver.py` passes (20 of 20). Every forkserver, legacy, outcome and lifecycle Go test passes too. 15 Go tests in `./cmd/metis` fail, all at `git init`, because the review environment refuses it outside its directories (sdlc #339). They touch capture and caching, not this diff.

Nothing blocks SHIP. Three Minor items remain, all on error paths that report themselves as degraded, with `joined=false` and an error shown.

1. **Strengths**
   - `metis/forkserver.py:251-267`: exit watching moved to its own thread, so the waiter never blocks while holding the protocol lock. A child that survives KILL is handed to `_late_reap` and reported `exit_known=false`, not given an invented exit code.
   - `cmd/metis/forkexec.go:322`: the dispatch time is recorded before the request write. Together with `classifyOutcome` (`outcome.go:67-75`) writing end, runtime and exit code only for an observed exit, this enforces "times only from observed events" in one place.
   - `serve()` opens the log before forking (`forkserver.py:339-344`), so an unopenable `step.log` is refused with `launched=false` and nothing runs.
   - The new in-process tests patch before forking, send the child's output to `/dev/null`, and always kill the child in `finally`. Leaked children were the hazard here.
   - The protocol version on the ready line (`checkForkProtocol`) refuses a stale server before anything is dispatched, so falling back to legacy stays safe.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **4th finding in family `owned-process-left-running`.** Rule: *a step's process group is signalled only while its leader is unreaped, so the leader must not be reaped until the group has been escalated, and the reap must be on a path no exception can skip.*
     - Instance A: when exit watching fails, the fallback (`forkserver.py:258-264`) reaps the leader and removes it from `_children` before `_drain_join` runs. `_escalate` then signals nothing, so a lingering process still in the group is never stopped.
     - Instance B: an exception in the second `try` (`forkserver.py:284-289`) before the reap leaves a zombie and a stale `_children` entry.
     - Instance C: if `watcher.start()` fails (`forkserver.py:270-272`, outside any `try`), no answer is ever sent.
     - Fix the rule, not the instances: reap in an outer `finally` that runs after escalation, and in the fallback signal the group before the non-blocking reap.
   - **2nd finding in family `forked-path-test-gaps`.** Rule: *every forkserver test that takes the protocol lock or waits on a child does so with a bound, so a regression fails instead of hanging the suite.* `test_exit_watcher_failure_falls_back_and_still_reaps` uses an unbounded `with lock:` (`tests/test_forkserver.py:~516`). Against round-3 code it hung the run and I had to kill it.
   - The `_escalate` branch for an unobserved exit (`forkserver.py:282-283`) is effectively unreachable: `exited.wait()` doesn't raise in a worker thread. Either delete it or note that it is defensive.

5. **Test coverage**
   - Go covers the forked path well: the request-write failure, a `joined=false` answer, a cancel mid-run, pending resources and a refusal before fork.
   - Python has no test of an in-group lingering process combined with a failed exit watcher (instance A above).
   - The gofmt drift `gofmt -l` reports in `cmd/metis/exec.go` already exists at the base commit and comes from the toolchain version.

6. **Architecture**
   - **ARCH-DRY: pass.** The waiter has a single escalation helper. Go's `terminate`/`awaitDrain` and its TERM/KILL loop over the protocol are a separate implementation for a separate process owner, which is acceptable.
   - **ARCH-PURE: pass.** `classifyOutcome` is pure and table-tested; the forkserver is IO glue.
   - **ARCH-PURPOSE: pass.** Both launch paths honor the timeout, live log and outcome record.
   - **ARCH-MOCK: pass.** Tests use real processes and the real uv server, as the plan says.
   - **ARCH-CONSTRAINTS: pass.** The Go wait after KILL (2·join + grace + 1s) covers the server's own join after the exit (join + grace + join).
   - **ARCH-SECURE: pass.** `pending-resources.json` is treated as opaque and bounded, and an unreadable file counts as pending.
   - **ARCH-ORDER: pass, with the Minor above.** In the fallback, the order "reap only after group escalation" is not enforced.
   - **ARCH-FUNERAL: pass.** The new files live in the run dir and are capped per step.
   - **ARCH-EVOLVE: pass.** `outcome.json` and the protocol fields are additive, and the version gate refuses an older server for a stated reason.

7. **Plan revisions:** none needed. The M2 Tasks 7 and 8, their fixtures and the atlas forkserver section are delivered, and the existing Revisions entries match the code.

```findings
dispose:
  - id: BR-10
    disposition: addressed
    note: |
      Reverting the reap-on-error branch fails test_waiter_failure_after_exit_answers_unjoined_and_reaps (DID NOT RAISE ChildProcessError); the round-3 forkserver hangs test_exit_watcher_failure_falls_back_and_still_reaps; log opens before fork; exit_known=false on survival.
  - id: BR-12
    disposition: addressed
    note: |
      Mutation checked: moving forkRun.started after the write turns TestForkedStep_RequestWriteFailureRecordsOnlyObservedTimes red (Started zero); classifyOutcome omits end/runtime/exit when unobserved.
  - id: BR-17
    disposition: addressed
    note: |
      One _escalate helper (forkserver.py:208) used by both _drain_join and the unobserved-exit path, gated on the observed exit; behavior covered by the BR-10 tests.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: owned-process-left-running
    title: |
      Forkserver waiter can reap the leader before group escalation, or skip the reap, on rare error paths
    detail: |
      Rule for all 4 instances in this family: a group is signalled only while its leader is unreaped, so reap only after group escalation, in a finally no exception can skip. The watcher fallback (forkserver.py:258-264) reaps and pops _children before _drain_join, so _escalate signals nothing and an in-group holder lives on; an exception at 284-289 leaves a zombie and a stale _children entry; a watcher.start failure at 270-272 sends no answer.
  - id: new
    severity: Minor
    kind: defect
    family: forked-path-test-gaps
    title: |
      test_exit_watcher_failure_falls_back_and_still_reaps takes the protocol lock without a bound, so a regression hangs the suite
    detail: |
      Rule: every forkserver test that takes the lock or waits on a child is bounded (lock.acquire(timeout=...), join timeouts). Run against the round-3 forkserver.py (fd69993), this test hung until killed instead of failing.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"93926904-35ab-4446-9deb-5f6149d2caec","repository":"github.com/xianxu/metis","issue_id":"000072","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/0185610a5f76773e","reviewed_head":"e4eabd172e5c12568306341ea9994de46ce58211","slot":"metis:1","requested_at":"2026-10-25T17:20:41.010999Z"},"phase":"finished","pid":36401,"started_at":"2026-10-25T17:20:41.114284Z","finished_at":"2026-10-25T17:33:53.567565Z","session_id":"9cf19862-52a4-4876-981a-caae0c5a9eab","outcome":"returned"}] -->
