# Boundary Review — metis#44 (whole-issue close)

| field | value |
|-------|-------|
| issue | 44 — leaf executor: warm fork-server — kill per-step interpreter+import cost |
| repo | metis |
| issue file | workshop/issues/000044-leaf-forkserver.md |
| boundary | whole-issue close |
| milestone | — |
| window | 138f7220b086bba8ef1f1c3398736d52a4cbc2e1..HEAD |
| command | sdlc close --issue 44 |
| reviewer | claude |
| timestamp | 2026-07-15T13:57:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. One note on process: the Bash tool is broken in this session (harness-level EPERM creating its session-env dir, even with the sandbox override), so I could not re-run `go test`/`pytest`/`gofmt` myself — this review is from close reading of the full diff, the surrounding code, the issue Spec/Plan, and the real wrappers. The findings below reflect that.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

This is a well-designed, well-tested boundary: the fork-server seam preserves the per-step semantics the plan pinned (per-child process isolation, authoritative env, forced `used_site_packages`, same `run_traced` machinery), degrades loudly-never-silently, and the test suite drives real forks against the real server with no mocks. What blocks a clean SHIP is one correctness race in the Python server — forking on the main thread while a waiter thread holds `sys.stdout`'s internal buffer lock leaves that lock permanently held in the child, deadlocking it and hanging the whole run; over a ~5k-leaf sweep at 8-way parallelism this is a real (roughly percent-scale per sweep) production hang, and the fix is one line. Two smaller gaps: mid-flight server death silently re-runs a possibly-still-running step via legacy (the plan said it should error the step), and the issue's Done-when claims a ≥2× perf-test bound plus a real-sweep before/after that the code/Log don't quite deliver as written.

## 1. Strengths

- **The loud-fallback contract is designed, implemented, and tested at both levels** — `serverPool.noticeOnce` (`cmd/metis/forkexec.go:269`) dedups per uses-type and per root, and both `TestExecute_NonConformingWrapperUsesLegacyLoudly` and `TestForkServerPool_BrokenRootFallsBack` assert *exactly one* notice. Speedup degrades, correctness doesn't, and the operator sees it.
- **`stepEnv` + `collectResult` extraction** (`cmd/metis/exec.go:47`, `exec.go:150`) — the two executors genuinely share one env definition and one output-collection path instead of copy-pasting the contract. Clean ARCH-DRY move; the env-authority semantics ("absence is the seal") are stated where they live.
- **`run_traced` extraction** (`metis/trace.py:186`) is a minimal, faithful seam: legacy `main()` delegates byte-identically, and `force_site_packages` closes a genuinely subtle cache-key hole (a warm child never observes site-packages reads) — pinned by a real test (`tests/test_forkserver.py::test_reads_json_written_with_forced_site_packages`).
- **Test quality is high**: real forks, real `uv run` servers, no mocks; the READ_ROOT no-bleed test directly pins the confinement seal; concurrency, SystemExit pass-through, and EOF-drain are all covered; the e2e is parameterized over both executor modes on the real wrappers.
- **The third-party-only preload reasoning** (D-integrity over warm-start greed) survived from plan → module docstring → atlas, with a `METIS_FORKSERVER_PRELOAD` override so tests aren't hostage to it.

## 2. Critical findings

**C1 — fork while a waiter thread holds the stdout lock deadlocks the child → hangs the run.** `metis/forkserver.py:114-131` (`serve`) forks on the main thread while `_wait` threads concurrently `out.write(...)`/`out.flush()` (`forkserver.py:101-103`). Those calls hold `sys.stdout`'s internal `BufferedWriter`/`TextIOWrapper` lock. `os.fork()` copies that lock in its held state into the child, where the owning thread doesn't exist — so the child deadlocks at its first stdout/stderr use, and unconditionally at `_child`'s `finally: sys.stdout.flush()` (`forkserver.py:79`). CPython does not sanitize io-buffer locks at fork (this is exactly why 3.12 warns on fork-in-threaded-process). A deadlocked child never closes its output pipe → its waiter never EOFs → the Go caller blocks in `forkServer.execute` forever → the sweep hangs. Window = every waiter response write; exposure = ~5k forks per sweep; this will bite unattended runs.
**Fix sketch (one line):** in `serve()`, fork under the protocol lock — `with lock: pid = os.fork()` (all writes to `out` happen under `lock`, so holding it across fork guarantees the io-internal locks are free at fork time; the child never touches `lock`). Belt-and-braces: in the child after the `dup2`s, rebind `sys.stdout = io.TextIOWrapper(io.FileIO(1, "wb"))` (and stderr likewise) so no parent buffer/lock state is reachable at all. The plan's fork-safety stance ("waiter threads do IO only") missed that the waiters' IO *shares a locked object with the fork path* — worth a plan `## Revisions` line too.

## 3. Important findings

**I1 — mid-flight server death re-runs the step via legacy while the forked child may still be running (plan deviation + double-execution hazard).** `serverPool.execute` (`cmd/metis/forkexec.go:301-308`) returns `ok=false` when `forkServer.execute` errors, and `execStep.Execute` (`cmd/metis/exec.go:92-111`) then falls through to the legacy spawn. But a server that dies *after* the request was written may have already forked the child — server death does not kill it (it's reparented and keeps writing metrics.json/artifacts/reads.json into the same stepDir the legacy re-run now writes into concurrently → corrupted step outputs). The plan explicitly pinned the other behavior: "Server crash mid-run: pending requests error that step" (plan §Risks). Fix: split the error surface in `forkServer.execute` — a failure *before* the stdin write (or `s.dead` pre-check) is `errServerUnavailable` → fallback; a death *after* dispatch is a step error → propagate, don't re-run. Add a test that SIGKILLs the server with a request in flight. (ARCH-PURPOSE: the fallback contract is "degrades, never correctness" — this path can currently degrade correctness.)

**I2 — Done-when drift on the perf acceptance (traceability).** The issue's Done-when says "the loose-bound perf test pins ≥2× vs legacy on toy leaves"; `TestForkServerPerf_LooseBound` (`cmd/metis/forkexec_test.go:239-243`) pins only strictly-faster (1×), and the Log's own toy numbers (1.15s vs 0.91s incl. preload) wouldn't pass a 2× bound. Likewise Done-when bullet 1 requires "measured before/after wall-clock in the Log" for a real kbench sweep; the Log has the toy e2e before/after (3.70s→1.89s) and a real smoke *absolute* time (10.1s) but defers the real before/after to "next operator sweep". Either run the titanic smoke once with `--forkserver=false` and record the pair, or add a `## Revisions` entry to the issue re-stating the Done-when to what was actually delivered (loose >1× CI-robust bound + toy before/after + real smoke, headline deferred). Don't close with the contract claiming a bound the test doesn't enforce.

## 4. Minor findings

- `pool.shutdown` has no kill-after-timeout; the plan pinned "close stdin, wait, kill after timeout" (`forkexec.go:242-245` waits unboundedly on `<-s.done`). Interlocks with C1: a wedged child hangs `metis run` at exit. Cheap: `cmd.Process.Kill()` after a deadline.
- Forked children inherit the server's stdin — Go's protocol pipe — as fd 0 (`forkserver.py:121-126`); a stdin-reading step could steal queued requests. `dup2` `/dev/null` onto fd 0 in the child.
- Env divergence: the child scrubs **all** `METIS_*` (`forkserver.py:63`) while legacy strips only `METIS_READ_ROOT` (`exec.go:119`) — an ambient `METIS_FOO` is visible to legacy leaves but not forked ones. The Python comment says "mirroring exec.go's strip", which overstates the mirror; document or align.
- Fork-mode D gains `metis/forkserver.py` (it's in the child's sys.modules snapshot as `__main__`); legacy D doesn't have it. Safe direction (same class as trace.py's documented self-inclusion) but mode-dependent reads.json — worth one docstring/atlas line.
- `_OUTPUT_CAP` trims only after accumulating the full output in memory (`forkserver.py:89-100`); a pathologically chatty step balloons the server. Trim incrementally.
- `parseWrapper` re-reads + re-stats the wrapper file on every leaf (~5k×); negligible vs the fork, but a per-path memo in the pool would be free.
- `metis select --promote` builds `runOpts` without `forkserver` (`select_cmd.go:351`) — promoted re-runs are always legacy. Probably fine (few runs); note it.
- Plan T2 names the pytest file `metis/forkserver_test.py`; it landed (correctly, per repo convention) at `tests/test_forkserver.py` — fold into the plan revision.
- `e2e_test.go:48-56`: the new `forkserver:` field likely breaks gofmt's key-value alignment in that literal — run `gofmt` (I couldn't execute it this session).

## 5. Test coverage notes

Coverage of the shipped surface is genuinely strong (see Strengths). Gaps, in order of value: (a) **no test for mid-flight server death** — the exact I1 path (kill the server process while a request is pending, assert the step errors rather than double-executes); (b) no `execStep.Execute`-level test that a forkserver step failure (exit≠0) surfaces as a run error with the output attached (only the pool level is tested); (c) after fixing C1, a cheap stress guard (fork continuously while responses stream) would pin the fix. I could not re-run `go test ./... -race` or pytest myself (Bash unavailable in this review session); the Log claims green and nothing I read contradicts that, but the gate should rely on the main session's run.

## 6. Architectural notes (ARCH-* pass/flag)

- **ARCH-DRY: pass.** `stepEnv`, `collectResult`, and `run_traced` are exactly the consolidations this diff needed; no duplicated contract definitions remain across the two executors. (Trivial residue: the two broken-root notice branches in `serverPool.execute` are near-twins.)
- **ARCH-PURE: pass, one note.** `parseWrapper` mixes the regex/derivation logic with `ReadFile`/`Stat`; a pure `parseWrapperBytes(b, absPath)` core with the IO at the caller would make the convention tests IO-free. Not blocking — the current tests are cheap tempfile-based and honest.
- **ARCH-PURPOSE: pass with the I2 caveat.** The purpose (kill the spawn tax for real sweeps) is wired end-to-end and default-on: the pool threads through preamble/score/point sweep paths (`sweep.go:418,506,563` all copy `ss.o`), both repos' roots are served, and the real-wrapper e2e runs both modes. The deferred piece is the *acceptance evidence* (real-sweep before/after), not the mechanism — record it or revise the Done-when, per I2. Shadow-sweep of consumers: `metis run` ✓, sweep internals ✓, `select --promote` knowingly legacy (minor above).

## 7. Plan revision recommendations

Add one `## Revisions` entry to `workshop/plans/000044-leaf-forkserver-plan.md` (and reconcile the issue's Done-when) covering: (1) the fork-safety stance must include the C1 fix — "fork under the protocol lock; child rebinds stdio" — since "waiter threads do IO only" was insufficient as stated; (2) shutdown as implemented is close-stdin + unbounded wait (no kill-timeout) — either implement the timeout or revise the contract; (3) mid-flight server death behavior — restate to match whichever side of I1 you land on (plan currently says "error the step", code falls back); (4) T2's test path is `tests/test_forkserver.py`, and the perf test pins strictly-faster, not ≥2× (Done-when wording).
