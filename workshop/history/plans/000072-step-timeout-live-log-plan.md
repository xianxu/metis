# Step Timeout and Live Log Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Depends on:** metis#71 (landed first: the `with.cache` parser pattern and the `cachingExecutor` changes this threads ctx through). Code starts after #71 lands.

**Goal:** A metis step can run for hours (a Couch session) under a declared timeout, with its output visible in a bounded log file while it runs, and every way it can end (success, failure, timeout, cancellation, unconfirmed cleanup, unretired external resources) is recorded honestly and never retried or published as success.

**Architecture:** One execution owner, `execStep`, gains a `context.Context` (cancellation from SIGINT/SIGTERM) and a per-step timeout from a validated `with.exec` policy. Both launch paths, the legacy subprocess and the forkserver, write child output through one bounded live log writer into `step.log`, and end through one pure outcome classifier that writes `outcome.json`. Termination is TERM, then a grace period, then KILL, then a bounded join. In the forkserver path the server, as the process that reaps the child, is the only one that signals it.

**Tech Stack:** Go (`os/exec`, `syscall`, `context`), Python (`metis/forkserver.py`), CUE vocabulary.

---

## Design decisions (for TL review before code)

1. **Policy surface: `with: {exec: {timeout: "90m"}}`.** It is a closed sub-map next to #71's `with.cache`, parsed by a pure `ParseExecPolicy` that `Validate` calls before any effect. It refuses an unknown key, a non-string value, an unparseable duration, or a duration ≤ 0. There is no default timeout, so ordinary steps behave as today. A namespaced key cannot collide with a step type's own `with.timeout`. Trade-off: `with` is in K_pre, so changing a timeout rekeys a cacheable step. That is sound (a spurious miss, never a stale hit) and rare, and the steps that need timeouts (sessions) are `policy: never` anyway. The alternative, excluding execution-policy keys from K_pre, was rejected because it is a cache-key format change for no correctness gain.
2. **What the timeout bounds: the child's runtime, from launch.** Waiting for a leaf-budget slot is not counted, so a scheduling queue never times out a session. "Deadline expiry before launch does not start the child" is met by the context: if the run is cancelled, or the context's deadline passes, while the step waits for a slot or before it forks, the child is never started and the step ends `cancelled`. This needs a cancellable `leafBudget.acquire(ctx, prio)`. **Alternative for the TL:** count the timeout from step entry, slot wait included. That gives a simpler single wall budget, but contention can kill a session that never ran.
3. **Cancellation source: SIGINT/SIGTERM.** `main` uses `signal.NotifyContext`; a second signal restores the default and exits hard. The board's `q` stop keeps its current meaning: it stops admitting new leaves and does not kill running ones. `runControl.fail` keeps letting in-flight siblings finish. Killing siblings on the first failure is out of scope.
4. **Process ownership.** A legacy child runs in its own process group (`Setpgid`), so the group can be terminated as a unit. Because terminal Ctrl-C then no longer reaches the child directly, item 3's handler forwards it. A forkserver child calls `setpgid(0,0)` after fork for the same reason. Go never signals a forkserver pid itself. It sends a `{"cancel": id, "signal": "TERM"|"KILL"}` request, and the server signals only a child it has not yet reaped, under its protocol lock. This makes PID reuse impossible.
5. **Termination policy (shared, pure parameters).** Send TERM to the group, wait up to `grace` (10s), send KILL to the group, then join with a bound of `joinBound` (5s; the legacy path uses `cmd.WaitDelay`). If the join does not complete, the outcome is `cleanup-unconfirmed`: the child or a grandchild may still hold the log pipe. That is reported, never treated as success, and never re-run. Tests shrink `grace`/`joinBound` through an executor field.
6. **Live, bounded log.** Child stdout and stderr go to `<stepDir>/step.log` as bytes arrive, up to a cap of 64 MiB. Past the cap, the writer appends a single truncation marker, then drains and discards the rest, so the child never blocks on a full pipe. A 64 KiB in-memory tail feeds the error message, replacing today's unbounded `CombinedOutput`. In the forkserver path the server's waiter thread streams the pipe into the same file with the same cap, instead of buffering everything until exit. The request carries `log_path` and `log_cap`, and the response's `output` becomes the tail. `step.log` and `outcome.json` join the reserved step-dir names, so they are never artifacts and never enter the output hash. The step's start line on `out` names the log path, which is the existing progress surface; there is no second runner or UI.
7. **Outcome record: `<stepDir>/outcome.json`.** It is written on every executed (non-cache-hit) step, whatever the result: `{outcome, exit_code?, signal?, started, ended, runtime_seconds, log_truncated, pending_resources?}`. `outcome` is one of `ok | failed | timed-out | cancelled | cleanup-unconfirmed | resources-unretired`. `runtime_seconds` is compute runtime, from launch to the child's exit. #74 adds `wait_seconds` (human wait) as a separate field; it never folds into runtime. The format is additive JSON (ARCH-EVOLVE). Nothing in metis reads it back in this issue; #73/#74 do.
8. **External resources: a domain-free contract.** The env gains `METIS_PENDING_RESOURCES=<stepDir>/pending-resources.json`. A step that starts independently managed resources (Couch actors) writes them there and removes the file once they are retired. On any ending, if the file exists and is non-empty, the outcome is `resources-unretired`, even when exit is 0. The error names the file and its contents for reconciliation, and the artifacts are kept. metis never kills those resources: a timeout terminates only the child's process group.
9. **No duplicate execution.** The forkserver keeps the existing rule: legacy fallback only when nothing was dispatched. A timeout, cancel, lost response or cleanup-unconfirmed ending after dispatch is a step error, never `ok=false`. The rule that settles a late result is that **the first terminal event the step's select observes wins**. If the deadline fires first, the outcome is `timed-out` even if an `exit: 0` response arrives during termination. That late response is logged on `out` and dropped. The `<-s.ready` wait becomes context-bounded. A context cancelled during it returns a `cancelled` step error, never `ok=false`, so it can never route to the legacy path.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ExecPolicy` / `ParseExecPolicy` | `pkg/experiment/execpolicy.go` | new |
| `Outcome` / `classifyOutcome` | `cmd/metis/outcome.go` | new |
| `boundedLog` | `cmd/metis/steplog.go` | new |
| `reservedStepFiles` | `cmd/metis/exec.go` | new (extracted from the inline names at exec.go:239) |

- **ExecPolicy**: `{Timeout time.Duration}` parsed from `with.exec`. Validate calls it, next to `ParseCachePolicy`. **DRY:** it reuses #71's pattern: one parser per `with.<policy>` sub-map, called by both Validate and the executor. **Extensions:** `grace`, `log_cap` keys, if a real need appears.
- **Outcome / classifyOutcome**: a pure function from `(endEvent, exitStatus, joinConfirmed, pendingResources, logTruncated, times)` to an `Outcome` plus an error or nil. `endEvent` is one of `exited | deadline | cancelled`. It is the single place that decides success, so neither launch path can publish success on its own (ARCH-ORDER: an explicit transition table). The table:

  | end event | join confirmed | pending resources | → outcome | error? |
  |---|---|---|---|---|
  | exited, code 0 | yes | none | ok | nil |
  | exited, code ≠ 0 | yes | none | failed | yes |
  | deadline | yes | none | timed-out | yes |
  | cancelled | yes / n/a (pre-launch) | none | cancelled | yes |
  | any | **no** | any | cleanup-unconfirmed | yes |
  | any, else | yes | **some** | resources-unretired | yes |

  Precedence: cleanup-unconfirmed > resources-unretired > timed-out/cancelled/failed > ok.
- **boundedLog**: an `io.Writer` over an `*os.File`, with a cap and a tail ring. Each `Write` does O(len(p)) work in a single pass. Memory is fixed at the tail size, and the file is at most cap plus the marker.
- **reservedStepFiles**: the set excluded from artifacts gains `step.log`, `outcome.json` and `pending-resources.json`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `execStep.Execute(ctx, …)` | `cmd/metis/exec.go` | modified | `os/exec` child, process group, signals |
| `StepExecutor` | `pkg/experiment/run.go` | modified | adds `ctx` to `Execute` |
| `leafBudget.acquire(ctx, prio)` (interface + `chanSem` + `prioritySem`) | `cmd/metis/prioritysem.go` | modified | slot acquire, now cancellable in every implementation |
| `forkServer.execute(ctx, …)` + cancel request | `cmd/metis/forkexec.go` | modified | JSONL protocol to the server |
| forkserver `cancel` + streaming waiter | `metis/forkserver.py` | modified | `os.fork`, `waitpid`, `killpg` |
| signal context | `cmd/metis/main.go` | modified | SIGINT/SIGTERM |

- **StepExecutor ctx**: every decorator (`cachingExecutor`, `activityExecutor`) and test fake passes it through. The cache's uncached and miss paths pass ctx to inner. A cancelled run records `failed` through the existing Runner path.
- **Cancellable acquire**: a waiter blocked in `prioritySem` must leave the heap when ctx is done. If a handoff raced in first, the slot is released back. This is a sequence-tested invariant: the slot count is conserved across cancel/handoff interleavings, driven by a deterministic test that controls ordering through channels.
- **No new fake**: the external dependency is real OS processes and the real uv forkserver. Tests use real children with **readiness control**: a fixture writes a ready marker, then blocks reading a FIFO the test controls, so no test depends on sleep timing to know the child has started.

## State design table

Actions available to a user or automation: Ctrl-C/SIGTERM (cancel the run), the board's `q` (stop admitting), rerun the experiment, and read `step.log`/`outcome.json`.

| Internal state / failure mode | Visible state (`outcome.json`, run.json) | Recovery action | Possible loss |
|---|---|---|---|
| child exits 0, no pending resources | ok; step recorded | none | none |
| child exits ≠ 0 | failed; run failed; log kept | read step.log, fix, rerun | none (artifacts kept) |
| deadline: TERM, then exit within grace | timed-out; signal=TERM | raise the timeout or fix the step, rerun | partial work in the step dir (kept, not published) |
| deadline: ignores TERM, KILLed | timed-out; signal=KILL | same | the child had no chance to clean up |
| KILLed but join exceeds the bound (escaped grandchild holds the pipe) | cleanup-unconfirmed | find and kill the orphan using the log; rerun | orphan process may still run |
| SIGINT/SIGTERM while running | cancelled; run failed | rerun | partial step work |
| cancelled while waiting for a slot or before fork | cancelled, with no started/runtime | rerun | none (never launched) |
| exit 0 but pending-resources.json is non-empty | resources-unretired; run failed | retire the listed resources (Couch), then rerun or accept manually | none; the resources are still live, by design |
| timeout or cancel with pending resources | resources-unretired (takes precedence over timed-out) | same | same |
| forkserver: dispatched, then server dies or response is lost | failed ("dispatched-lost"); root marked broken; no fallback | rerun (the next run restarts the server) | the step may or may not have run (unconfirmed, so not retried) |
| forkserver: cancel sent, response never arrives within the join bound | cleanup-unconfirmed | as above | as above |
| forkserver: exit-0 response arrives after the deadline fired | timed-out (the first event wins); late result logged and dropped | rerun | the possibly complete result is not published |
| metis itself SIGKILLed | no outcome.json; run.json status `started` without `finished` | treat as interrupted (#73 owns interrupted publication); orphans possible | the step's outcome is unknown |
| log exceeds the cap | outcome as usual; log_truncated=true; marker in the log | read the tail in the error and the head in the file | output past the cap |

Covered: every ending of a launched child in both paths, cancellation before launch, and external-resource reconciliation. **Not covered:** metis's own SIGKILL (no handler is possible; #73 handles interrupted runs) and killing escaped or setsid'd descendants (reported, never chased, because they are exactly the "independently managed" actors metis must not touch).

## Hot-path cost

`boundedLog.Write` runs per chunk of child output. The growth driver is output bytes B. The work is O(B), a single pass: each byte is copied into the file (until the cap) and into the ring. Memory is O(tail) regardless of B. **Counted test:** feed N = cap and 2N = 2·cap bytes in random-sized chunks. Assert bytes written to the file ≤ cap + len(marker) in both cases, ring bytes ≤ tail, and total bytes consumed == input. That rejects re-copying the ring per write, by counting copies through an injected counter: copies ≤ 2·B.

## ARCH notes

- **PURE:** policy parsing, outcome classification and log bounding are pure. Exec and forkexec are glue around them.
- **ORDER:** the state machine is `classifyOutcome`'s table. The events are exit, deadline and cancel, plus a late response, which is ignored once the step's terminal event is chosen. Concurrency extent: when `Execute` returns, no goroutine it started is alive. The log copier is joined via Wait/WaitDelay, and the timer is stopped. The event most likely to be mishandled is an exit-0 response racing the deadline; the test makes that race deterministic by releasing the fixture's FIFO after the deadline has been observed.
- **CONSTRAINTS:** this is a batch workload. Steps run for minutes to hours. Log cap 64 MiB per step, tail 64 KiB, grace 10s, join bound 5s. Termination is bounded at grace + joinBound (about 15s) after the deadline.
- **FUNERAL:** `step.log`, `outcome.json` and `pending-resources.json` live in the run dir and are removed with it by the existing runs/ lifecycle. They are bounded per step (log ≤ cap; the others are tiny). Nothing is added outside the run dir.
- **SECURE:** `with.exec` is author input, parsed into a typed value at Validate. `pending-resources.json` is child-written: it is read as opaque bytes (at most 64 KiB, shown verbatim, never interpreted). `outcome.json` is metis-written and nothing reads it back yet.
- **EVOLVE:** `outcome.json` is new and additive. The forkserver protocol gains optional request fields and a `cancel` message type. Server and client ship together in one repo and run from the same checkout, so there is no version negotiation. A stale server would ignore `log_path` and `cancel`; the Go side treats a missing log as an error at the end of the step, not silently.
- **MOCK:** real processes and the real uv server, with readiness-controlled fixtures. There is no external service.
- **PURPOSE:** both launch paths, both cancellation sources, and the resource contract couch-bench needs are covered. The #74 wait field is left to #74 by design (a separate field in the same record).

## Milestones

- [ ] **M1 — policy, outcome, log, legacy path, cancellation.**
  - Task 1: `ExecPolicy` with its parser and table tests. Validate refuses each malformed case. CUE gets `#ExecPolicy` (`timeout?: string`), and #71's drift-guard pattern is extended to it.
  - Task 2: `boundedLog`, with the counted test at N and 2N.
  - Task 3: `classifyOutcome`, with table tests covering every row and the precedence order.
  - Task 4: thread ctx through `StepExecutor`, Runner, the cache, activity and the test fakes. Signal context in main.
  - Task 5: cancellable `leafBudget.acquire` with a deterministic interleaving test (cancel before handoff, handoff then cancel, slot conservation).
  - Task 6: the legacy path: Setpgid, boundedLog into `step.log`, timer, TERM/grace/KILL/WaitDelay, `outcome.json`, pending resources, reserved names.
  - Task 6 fixtures (real shell, readiness-controlled):
    1. Live-log: the test reads new bytes in `step.log` while the child still blocks on its FIFO.
    2. A timeout that TERM ends.
    3. A child that ignores TERM (`trap '' TERM`) and is KILLed.
    4. An escaped grandchild (`python3 -c 'os.setsid()…'` holding stdout), giving cleanup-unconfirmed.
    5. Exit 0 with a pending-resources file, giving resources-unretired.
    6. Cancel while blocked on a budget slot: the child never starts, shown by a launch marker file that is absent.
    7. An ordinary success, whose output is unchanged.
  - Atlas: `atlas/experiment.md`, for the env contract, `step.log`, `outcome.json` and `with.exec`.
- [ ] **M2 — forkserver path.**
  - Task 7: Python: setpgid in the child, a streaming capped waiter writing `log_path`, and a `cancel` message that signals only unreaped children under the lock. Tests in `tests/test_forkserver.py`: log bytes visible before exit, cancel TERM, cancel KILL, cancel after reap is a no-op, and the response's output is the tail.
  - Task 8: Go: request fields, a context-bounded ready wait, and a deadline/cancel that sends `cancel` and joins on the response within the bound. A late response is dropped and logged. The lost-after-dispatch rule is unchanged, and no fallback happens after dispatch.
  - Task 8 fixtures: toy modules gated on a FIFO for readiness, covering timeout, KILL escalation, a late exit-0 after the deadline (stays timed-out), and the server dying during a cancel (cleanup-unconfirmed, root broken, no legacy re-run).
  - Atlas: the forkserver section.

Each task is TDD: failing test, minimal code, green, commit. Concrete code lands per task. This document fixes the contracts, tables and fixtures the tests assert.
