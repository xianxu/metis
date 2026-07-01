---
type: experiment
id: run-echo
seed: 7
status: active
steps:
  - id: first
    uses: test/echo
    with: {msg: hello}
  - id: second
    uses: test/echo
    needs: [first]
    with: {msg: world}
---
# run-echo — M2 end-to-end fixture

Two `test/echo` steps (resolved to `testdata/steps/test/echo`, the process-level
fake step-type) prove `metis run` executes real subprocesses in dependency order
and records a Run. Not a real experiment — the `metis/*` data-plane step-types
arrive in M3; `test/echo` exists only to exercise the real os/exec executor.

## Runs
