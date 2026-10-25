# Boundary Review — metis#72 (whole-issue close)

| field | value |
|-------|-------|
| issue | 72 — Bound step execution with timeout and live logs |
| repo | metis |
| issue file | workshop/issues/000072-step-timeout-live-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99ccccb46a2707de6fbfe1be44c32263f0e636cf..9d1da896717d519493edd1715ac9eab792bc2f7d |
| command | sdlc close --issue 72 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-25T10:43:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Verifying the claimed fixes: reading the atlas section and tests, then running the suites.

Now checking whether the guarded-stage fix has a test that fails without it: reverting forkserver.py to the prior commit and rerunning.

Git-init failures are review-confinement artifacts; running the issue's own lifecycle tests directly.

Wrapping up: BR-5 and BR-8 are confirmed by mutation. One last check: that the BR-19 test now fails, rather than hangs, against the old forkserver.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Metis#72 delivers what its Spec and Plan promise. There is a `with.exec.timeout` policy that `Validate` checks before any step runs. Each step runs in its own process group with TERM, then KILL after a grace period, then a bounded join. Output goes live into a capped `step.log`. Every executed step writes an `outcome.json`, and one function, `classifyOutcome`, decides the outcome on both launch paths. A cancelled run never launches a step still waiting for a budget slot. I re-ran the evidence for the prior findings instead of trusting the commit messages:
- **BR-5 and BR-8:** each fix makes a test fail when I revert it.
- **BR-19:** its test now fails within one second against the round-3 forkserver; before, it hung.
- **BR-9:** the atlas now states the stdio precondition.
- **BR-18:** still open. The latest commit's change to `metis/forkserver.py` has no test that fails without it: reverting the file to `2ab03a3` leaves all 20 forkserver tests passing. A probe also shows one of BR-18's three cases is still live.

Every remaining finding is Minor, so nothing blocks the close.

**Test runs:**
- **Python:** `pytest tests/test_forkserver.py` gives 20 passed.
- **Go, issue tests:** `go test -race` over the lifecycle tests (`TestExecStep_*`, `TestForkedStep*`, classifier, budget, log) is ok.
- **Go, rest of the packages:** `./pkg/...` is ok.
- **Go, full `cmd/metis` failures:** these come from the review sandbox refusing `git init`, which is unrelated to this issue. `TestForkServerPerf_LooseBound` also failed under full-suite load but passed on its own; the issue log already notes it as a flake to watch.
- **Setup:** to build at all, I pointed `go.mod`'s ariadne `replace` at the pinned dependency checkout. I restored the file afterwards, and the checkout is clean.

### 1. Strengths
- **One outcome decider:** `runLegacy` and `runForked` both go through `classifyOutcome` and `writeOutcome` (`cmd/metis/steplifecycle.go:66-72`, `:251-257`). Start failures, a `step.log` that won't open, and cancellation before launch are all recorded through it, never bypassing it.
- **Join confirmed from a pipe metis owns:** `steplifecycle.go:98-116`. This sidesteps Go's `ErrWaitDelay` behaviour, and the escaper fixtures cover exit 0, exit 1 and timeout.
- **Forkserver only signals unreaped children:** it learns of the exit without reaping (`waitid` with `WNOWAIT`, or kqueue on macOS) and reaps only under the lock while removing the child from its table (`forkserver.py:133-157`, `:309-319`). A cancel can therefore never hit a recycled pid.
- **Mutation-tested regressions:**
  - Stamping the end time after the join wait fails `TestExecStep_EscapedGrandchildIsCleanupUnconfirmed`.
  - Removing the post-exit group TERM fails `TestExecStep_InGroupLingererIsStopped`.

### 2. Critical
None.

### 3. Important
None.

### 4. Minor
- **BR-18 (still open)**, `forkserver.py:291-293`: `end_group` escalates only while `watcher.is_alive()`.
  - **Probe:** I patched `Thread.start` to refuse only the watcher thread and used a TERM-ignoring child. The waiter answered (`exit_known=false`, `joined=false`), but the child was never signalled: it was still running, and still in `_children`, after the answer.
  - **Fix:** drop the `watcher.is_alive()` condition. The leader is still unreaped, so `_escalate` can safely signal it; it waits the full grace plus bound, which is bounded.
  - **Test:** add one for a stage failure, e.g. watcher start or a raising `waitpid` in `reap`.
- **New (Minor)**, `steplifecycle.go:174-183` and `:195-208`: `workshop/lessons.md` (added in this range) states the rule "signal a group only while its leader is unreaped". The legacy path breaks it:
  - `awaitDrain` signals `-pid` after `cmd.Wait` has reaped the leader.
  - `terminate` can do the same when the exit and the deadline arrive together.

  It is safe while any group member is alive, because the group id stays reserved. It only goes wrong if the pid wraps around within the grace-plus-bound window. Either document that precondition at the call site and narrow the lesson to the forkserver, or note that the legacy path can't uphold the rule without `waitid(WNOWAIT)`.

### 5. Test coverage notes
- The latest commit restructures the waiter into guarded stages but adds no test that fails without that change (see BR-18).
- Otherwise, the forked path's cancel, `joined=false`, pending-resources and pre-fork-refusal cases are covered against the real `uv` server.

### 6. Architectural notes
- **ARCH-DRY:** pass. `_escalate` is the single TERM/grace/KILL helper in Python. The Go side has `terminate` and `awaitDrain`, which are similar but have different jobs (before vs after the exit).
- **ARCH-PURE:** pass. `classifyOutcome` and `ParseExecPolicy` are pure and table-tested.
- **ARCH-PURPOSE:** pass. Both launch paths honour the policy, the log and the outcome record.
- **ARCH-MOCK:** pass. Tests use fixture steps and the real forkserver.
- **ARCH-CONSTRAINTS:** pass. Log cap, grace and join bound are explicit.
- **ARCH-SECURE:** pass. `pending-resources.json` is read with a size limit, and an unreadable file counts as pending.
- **ARCH-ORDER:** flag (Minor, BR-18). One waiter stage-failure ordering leaves the child running, and no test exercises stage failures.
- **ARCH-FUNERAL:** pass. `step.log` is capped per step and removed with its run dir, as the atlas documents.
- **ARCH-EVOLVE:** pass. `outcome.json` is new; the protocol version bump has a stated reason, refusing a stale server before dispatch.

### 7. Plan revision recommendations
None. The plan's Revisions section already records the dispatched-then-lost change and the protocol-version refusal.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      steplifecycle.go:140 uses the Wait time; mutation stamping ended after awaitDrain fails TestExecStep_EscapedGrandchildIsCleanupUnconfirmed (runtime >= joinBound).
  - id: BR-8
    disposition: addressed
    note: |
      awaitDrain TERM/grace/KILLs the group after a normal exit; disabling it fails TestExecStep_InGroupLingererIsStopped.
  - id: BR-9
    disposition: addressed
    note: |
      atlas/experiment.md METIS_PENDING_RESOURCES entry states leave-the-group and detach-from-stdio, and the EPIPE consequence.
  - id: BR-18
    disposition: not-addressed
    note: |
      No regression test (reverting forkserver.py to 2ab03a3 keeps 20/20 green); probe: watcher.start failure answers but never signals a TERM-ignoring child (end_group gated on watcher.is_alive), child left running.
  - id: BR-19
    disposition: addressed
    note: |
      Run against fd69993 forkserver the test now fails at the lock.acquire(timeout=1) assertion instead of hanging.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: group-signal-after-reap
    title: |
      Legacy path signals -pid after cmd.Wait reaped the leader, contradicting the lessons.md rule added in this range
    detail: |
      steplifecycle.go:174-183 (awaitDrain) and 195-208 (terminate, when exit and deadline coincide) signal the group after its leader is reaped. Safe while any member lives (pgid reserved), unsafe only on pid wraparound within grace plus bound. Document the precondition at the call site and scope the lesson to the forkserver.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"41c48e30-d036-40e9-8983-cf43d38ea81e","repository":"github.com/xianxu/metis","issue_id":"000072","recipe":"milestone-review","round":6,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/d04ee47d1cd0a24b","reviewed_head":"9d1da896717d519493edd1715ac9eab792bc2f7d","slot":"metis:1","requested_at":"2026-10-25T17:37:19.770709Z"},"phase":"finished","pid":87555,"started_at":"2026-10-25T17:37:19.890254Z","finished_at":"2026-10-25T17:43:47.558501Z","session_id":"7c48f5ff-98c4-4a2f-82ad-0fe2c3b01514","outcome":"returned"}] -->

---

## Re-review — 2026-10-25T11:49:24-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 72 — Bound step execution with timeout and live logs |
| repo | metis |
| issue file | workshop/issues/000072-step-timeout-live-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | 911c1058e60f6128bc94321769d6dd948673be22..32d4d6362227534974e7f4e6ec7a18630e470e88 |
| command | sdlc close --issue 72 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-25T11:49:24-07:00 |
| verdict | SHIP |

## Review

I've read the interdiff. Next I'll run the forkserver tests, then put the old `watcher.is_alive()` gate back to check that the new test catches it.

```verdict
verdict: SHIP
confidence: high
```

This fix review covers only the interdiff since the close review read `9d1da89`. It is a narrow change: a gate removed in `metis/forkserver.py`, a new `_start_watcher` seam, one new test plus a fixture fix, two Go doc comments, and two `lessons.md` entries. The issue file changes only its status, `actual_hours` and the `## Log`. `## Spec`, `## Revisions` and `## Done when` are untouched, so this is not a scope change. Both open Minor findings are addressed. For BR-18, the new test fails when the fix is reverted. BR-20 is a documentation correction, and I checked it against the code. The interdiff introduces no new defects, and there were no conflict resolutions to judge.

**Test runs:**
- `pytest tests/test_forkserver.py`: 21 passed.
- **Mutation:** I put back the old condition that skipped the escalation in `end_group` (`forkserver.py:290`). `test_watcher_start_failure_still_ends_the_child` then failed. With the file restored, the test passed 5 out of 5 runs, and the checkout is clean.

### 1. Strengths
- **One rule for when to escalate:** `end_group` (`forkserver.py:290-292`) now escalates whenever the child's exit was never observed, including when the watcher never ran. Because the leader is unreaped there, signalling its group is safe. The wait is bounded by grace plus the join bound, and then `_late_reap` takes over.
- **A test seam that catches the bug:** `_start_watcher` (`forkserver.py:337-339`) lets a test make the watcher fail to start. The test fails without the fix.
- **The test only checks, it doesn't reap:** it watches for the late reaper's work with `kill(pid, 0)` and never calls a second `waitpid` that would race the reaper (`tests/test_forkserver.py:549-556`).
- **Fixture fixed at the root cause:** `_fork_quiet_child` now calls `setpgid(pid, pid)` in the parent too, as the server already does (`tests/test_forkserver.py:441-447`). This fixes the failure under machine load seen at merge time, where TERM and KILL missed with ESRCH before the child was scheduled.

### 2. Critical
None.

### 3. Important
None.

### 4. Minor
- `cmd/metis/steplifecycle.go:164` and `:197`: the new comment lines run well past the surrounding wrap width. Cosmetic only; gofmt doesn't reflow comments.

### 5. Test coverage notes
BR-18 has three cases:
- **Watcher fails to start:** pinned by the new test, and the mutation confirms it.
- **An exception in one cleanup stage:** covered by the guarded stages and by `test_waiter_failure_after_exit_answers_unjoined_and_reaps`, which the close review already saw.
- **The fallback when the watcher can't observe the exit without reaping (`forkserver.py:254-269`):** this one cannot follow the rule. When the no-reap observation itself fails, the only portable way left to learn of the exit is a non-blocking reap. That limit is stated in `watch()`, and the response reports the error with `joined=false`, so the uncertain outcome is still visible.

### 6. Architectural notes
These cover the interdiff only.
- **ARCH-DRY:** pass. The new path reuses `_escalate`.
- **ARCH-PURE:** pass. `_start_watcher` is a thin IO seam and holds no logic.
- **ARCH-PURPOSE:** pass. BR-18 was fixed as a class, not just the cited site:
  - The rule "escalate unless the exit was observed" now covers every path where the exit is never observed.
  - The fallback that can't follow the rule is documented as a limit.
  - The BR-20 lesson now says which kind of process-group owner each rule applies to.
- **ARCH-EVOLVE:** pass. No persisted format changes.

### 7. Plan revision recommendations
None.

```findings
dispose:
  - id: BR-18
    disposition: addressed
    note: |
      end_group now escalates whenever no exit was observed (forkserver.py:290-292); reverting to the watcher.is_alive gate fails test_watcher_start_failure_still_ends_the_child; stage exceptions covered by guarded stages; the reap-first fallback is documented as the unavoidable case and answers joined=false with the error.
  - id: BR-20
    disposition: addressed
    note: |
      Precondition stated at awaitDrain (steplifecycle.go:160-165) and terminate (196-197); lessons.md:64-69 scopes the unreaped-leader rule to owners that control reaping and has the Go exec path state the POSIX pgid-reservation precondition. Prose-only, matches the code.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3c28b0d8-3206-4c46-a0f7-96e1033bc23e","repository":"github.com/xianxu/metis","issue_id":"000072","recipe":"fix-review","round":7,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/d7474e0f343e6255","reviewed_head":"32d4d6362227534974e7f4e6ec7a18630e470e88","slot":"metis:1","requested_at":"2026-10-25T18:47:51.291845Z"},"phase":"finished","pid":18982,"started_at":"2026-10-25T18:47:51.426983Z","finished_at":"2026-10-25T18:49:22.578308Z","session_id":"01ac85fa-24ae-4648-baab-062ce01ab253","outcome":"returned"}] -->
