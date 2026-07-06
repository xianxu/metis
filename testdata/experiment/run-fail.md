---
type: experiment
id: run-fail
seed: 7
status: active
steps:
  - id: boom
    uses: test/echo
    with: {fail: true}
---
# run-fail — M2 failed-run ledger fixture

One `test/echo` step told via `with.fail` to exit non-zero. Structurally and
semantically valid (it runs), so it reaches execution and fails there —
exercising the branch where `metis run` still writes `runs/<id>/run.json` with
status `failed`, then returns an error. The `## Runs` heading below is a
deliberate adversarial leftover: #13 makes the config immutable input, so the
run must NOT append under it (the test asserts the file is byte-unchanged).

## Runs
