# metis#44 — warm fork-server leaf executor

**Goal:** kill the per-step `uv run → fresh python → import pandas/sklearn` tax (~1.0s measured
per spawn, ~5k spawns ≈ 10–15 min of a 30-min sweep) by forking each step from a warm server
that pre-imported the heavy third-party libraries — while preserving every per-step semantic:
process isolation, the `metis.trace` read-sensor (`reads.json`), the env contract, crash
isolation, and per-step caching/content-addressing.

**Non-goals:** depth-first scheduling (metis#43), engine-side cv-split, any change to the step
AUTHORING contract (a step stays a bash wrapper; discovery/resolve untouched).

## Core concepts

| Name | Lives in | Status |
|------|----------|--------|
| `run_traced(module)` | `metis/trace.py` | modified (extracted from `main`) |
| `forkserver` server loop | `metis/forkserver.py` | new |
| `wrapperSpec` (parse) | `cmd/metis/forkexec.go` | new |
| `serverPool` | `cmd/metis/forkexec.go` | new |
| `execStep.Execute` routing | `cmd/metis/exec.go` | modified |

- **`run_traced(module)`** — the existing trace `main()` body (install audithook → runpy the
  step module as `__main__` → write `reads.json` on the way out) extracted into a function both
  entrypoints call: the legacy `python -m metis.trace <mod>` path and a forked child. Pure
  refactor; byte-identical `reads.json` on the legacy path. Takes a `force_site_packages: bool`
  (see the flag note below).
- **`forkserver` server loop** — `python -m metis.forkserver`: pre-imports third-party heavies
  (`pandas`, `numpy`, `sklearn`, `pyarrow`) then serves JSONL requests on stdin. Per request
  `{id, module, cwd, env}`: **fork on the main thread**; the child scrubs `METIS_*` from its
  env, applies the request env, chdirs, dup2s stdout+stderr onto a per-request pipe, calls
  `run_traced(module, force_site_packages=True)`, and `os._exit(code)`. The parent hands the
  pid+pipe to a waiter thread (drain pipe → `waitpid` → write `{id, exit, output}` to stdout
  under a lock). stdin EOF = drain in-flight children and exit.
  - **Preload third-party ONLY.** D (the cache read-set) is the *first-party* sys.modules
    snapshot at exit; preloaded first-party step modules would land in every child's snapshot
    and widen every step's D (over-invalidation). Third-party is excluded from D by design, so
    preloading it changes nothing. First-party imports stay in the child (~50ms, pure Python).
  - **`used_site_packages` forced true in forkserver children.** Today the flag is set by
    observing a site-packages read; children inherit the imports and would observe none,
    silently dropping the uv.lock dep from cache keys. Forcing it is conservative and correct.
  - **Fork-safety stance:** only the main thread forks; waiter threads do IO only; the server
    never executes a BLAS/OpenMP region (imports only), so children initialize their own
    threadpools. We already pin `OMP_NUM_THREADS=1` for real sweeps. Contingency if macOS
    ObjC guards fire in practice: export `OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES` on the
    server; ultimate escape hatch `--forkserver=false`.
- **`wrapperSpec`** — pure function: wrapper file bytes + path → `{root, module}` or
  `not-forkable`. Matches the exact two-repo convention (`exec uv run --project "$ROOT" python
  -m metis.trace <module>`; `root = dir(dir(dir(exe)))`, verified by `root/pyproject.toml`).
  Non-matching wrappers fall back to legacy exec with ONE loud notice per uses-type — the
  authoring contract is unchanged and exotic steps keep working.
- **`serverPool`** — `map[root]*server`, lazily started (`uv run --project <root> python -m
  metis.forkserver`; kbench's venv has metis as a path dep, so the module resolves in both
  envs), request ids matched to in-flight calls, mutex-guarded writes, shutdown (close stdin,
  wait, kill after timeout) at run end. Server start failure → loud notice → legacy exec.
- **`execStep.Execute` routing** — if forkserver enabled AND wrapper parses: acquire the SAME
  leaf semaphore, send request, await response (response.output plays CombinedOutput's role in
  errors). Else: today's `exec.Command` path, untouched. The caching decorator wraps Execute
  and is oblivious. `--forkserver` (default true) on `metis run`.

## Integration points

- Go↔Python protocol: JSONL over the server's stdin/stdout (request `{id, module, cwd, env}`,
  response `{id, exit, output}`). Child output goes to a per-fork pipe, never the protocol
  stream and never a file in stepDir (a stepDir log would pollute `collectArtifacts`).
- Env: request `env` carries exactly what `exec.go` injects today (`METIS_STEP_DIR`, `RUN_DIR`,
  `STEP_ID`, `EXP_DIR`, `SEED`, optional `READ_ROOT`); the child scrubs ALL `METIS_*` first so
  an absent `READ_ROOT` is genuinely absent (mirrors exec.go's strip).
- Semaphore: unchanged — one in-flight fork = one leaf slot; acquired around request→response.

## Tasks (single close; TDD per task; commit per task)

- [ ] **T1** `metis/trace.py`: extract `run_traced(module, force_site_packages=False)` from
      `main()`; legacy entry delegates. Tests: existing trace behavior unchanged (run the
      existing suite; the Go `trace_test.go`/`capture_test.go` are the regression net).
- [ ] **T2** `metis/forkserver.py` + `metis/forkserver_test.py` (pytest, real forks, no mocks):
      (a) round-trip — toy module writes metrics.json, asserts request cwd+env applied;
      (b) env isolation — request 2 must NOT see request 1's `METIS_READ_ROOT`;
      (c) failure — raising module → nonzero exit, traceback in `output`;
      (d) concurrency — 4 overlapping slow requests, ids matched;
      (e) `reads.json` written per child with `used_site_packages: true`.
- [ ] **T3** `cmd/metis/forkexec.go`: `wrapperSpec` parse (TDD: the exact convention, a
      non-matching wrapper, missing pyproject.toml → not-forkable).
- [ ] **T4** `serverPool` + `Execute` routing + `--forkserver` flag. Tests: pool unit tests with
      a stub server script; ONE hermetic integration test through the real
      `uv run --project <metis> python -m metis.forkserver` with a toy step wrapper in tmp
      (metrics + artifacts collected, error path). Full `go test ./... -race` green.
- [ ] **T5** Acceptance + docs: loose-bound perf test (N sequential toy leaves, forkserver ≥2×
      legacy; skippable `-short`); REAL kbench smoke (`metis run --fast` on titanic-sweep — the
      invocation-verified lesson) with before/after wall-clock in the Log; atlas
      (`experiment.md` executor section) + RUNBOOK note; close.

## Risks

- macOS fork + preloaded native libs (ObjC guard, Accelerate). Mitigated: import-only server,
  pinned BLAS threads, contingency env var, `--forkserver=false` hatch, and T4/T5's real-fork
  tests run on this exact platform before any real sweep depends on it.
- Wrapper drift: a future step wrapper deviating from the convention silently loses the speedup
  (never correctness) and says so loudly once.
- Server crash mid-run: pending requests error that step (run aborts as a step failure would
  today); rerun with `--forkserver=false` if it recurs.
