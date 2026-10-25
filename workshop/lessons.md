# Lessons

Compact guidance for Metis experiments, ledgers, schemas, and reproducible data
work. Keep numerical and incident detail in the issue or experiment record.

## Reproducibility and provenance

- Every result carries code fingerprint, data fingerprint, configuration, seed,
  dependency versions, and the cohort/ledger epoch that produced it. A display
  label or cache filename is not provenance.
- A rerun appends a new fingerprint cohort; it must not overwrite evidence from a
  prior code or data state. Content-addressed storage writes atomically and
  verifies the object it reads.
- Cache keys include every input that changes semantics, including schema fields,
  model options, feature policy, fold definition, and runtime-discovered values.
  Never reuse a compatibility cache after an authority downgrade.
- A durable ledger is append-only and exposes commit outcome. Test partial writes,
  duplicate appends, restart, and a reader that sees mixed-format history.
- A substitution or migration is complete only when the old path is impossible
  or explicitly supported; prove the sole road with an executable guard.

## Data and statistical validity

- Separate selection, fitting, and evaluation. Nested CV and read confinement
  must prevent target, fold, time, and future-data leakage at the feature level.
- A feature is predict-time safe only under an invariance test: scramble or drop
  the target and verify the prediction path does not change for the forbidden
  reason.
- Measure an evaluation noise floor before optimizing a leaderboard or threshold.
  A small score delta without paired folds, repeated seeds, or a two-scheme diff
  is not evidence of an improvement.
- Complexity follows combination semantics. Count set cardinality and actual
  combinations, not incremental loops or a proxy selected for convenience.
- A domain-informed baseline and residual target are hypotheses. Compare paired
  against the same folds and the explicit residual-zero baseline.
- A result that survives multiple independent washes is a structural finding;
  stop tuning and report the limitation rather than inventing a lever.
- Public test rows, hidden duplicates, cluster anchors, spatial buffers, and
  regional trends need a stated estimand. Do not call a learned structure a leak
  without quantifying the alternative scheme.

## Architecture and boundaries

- Reuse the domain reducer; do not hand-roll a biased twin. A helper is pure only
  when its dependencies are pure, not because its expression looks deterministic.
- A content-addressed or injected seam has a production scheduling owner. Tests
  must cross that owner and join every worker before asserting order or cleanup.
- Parallelization changes ordering, cache pressure, and failure timing. Pin
  order-preservation without deadlocking the serial baseline, and bound BLAS or
  NumPy threads when `--parallel NumCPU` fans out real leaves.
- A schema field that feeds content identity must be migrated in its own bounded
  stage. Partial migrations need old/new readers and an explicit overlap epoch.
- A command's helper is not self-contained until its callers, CLI flags, and
  integration fixtures no longer rely on it. Delete the full production chain.

- Canonicalize a path where paths meet, not at one producer (metis#76). Two paths
  that are compared (an exp-relative read against a read root) must share one
  form; fixing only the runner made the experiment dir physical while the sealed
  sweep's root stayed logical, and every within-root read tripped confinement.
  The one builder every caller passes through (`execStep.stepEnv`) owns it.
- A check that hides a tool's stderr must refuse a missing tool before judging
  (metis#70): command-not-found once read as "valid fixture rejected".

- A process-group owner that also controls reaping (the forkserver) signals a group only while
  its leader is unreaped (the pid, hence the group id, stays reserved); one that cannot (Go's
  exec reaps in Wait) relies on POSIX reserving the id while any member lives and says so at the
  call site. Either way it escalates on the observed exit rather than on the output pipe, and
  never waits under a lock another actor needs to signal (metis#72). Go's exec reports
  ErrWaitDelay only after a successful exit, so join evidence comes from EOF on a pipe you own.
## Testing and CLI

- Test through the CLI entry point for path, environment, and serialization
  behavior. A unit test that hand-builds argv proves the helper, not the wiring.
- Fakes must model the state the fix reads, including absence, cache miss, failed
  IO, and partial ledger. A fake with the same value shape as production masks
  format mismatches.
- Run the named suite in the named environment (`uv`, Go, or the project runner)
  and verify exit status. A filtered or skipped run is not evidence.
- A test property must disagree with the implementation under the mutation it
  claims to pin. Equal sets, disabled features, and a baseline that shares the
  same bug produce false greens.
- Plans state the estimand, tolerance arithmetic, fold cardinality, cache key,
  and failure behavior. Close only after the measured evidence exists.
- A test harness's scratch file (a temporary go.mod, generated config) lives in
  a private per-run `mktemp -d` that a trap removes, never at a fixed shared path:
  concurrent suites on one host collide there and nothing cleans it up (metis#75
  BR-1).
- A cache key that addresses inputs cannot see an upstream's realized output. A
  nondeterministic producer therefore makes every descendant uncached; it never
  makes them "reuse if output unchanged" unless the declared artifacts provably
  capture everything read (metis#71).

- A test that forks or spawns a long-lived child installs every patch before the fork, sends
  the child's stdout and stderr away from the runner, and kills it in `finally`. A child left
  holding the runner's stderr hangs the whole run; a mutation that breaks process-group code
  leaks its fixtures (metis#72, twice).
- A test fixture that makes its child a group leader sets the group in the parent too
  (`setpgid(pid, pid)` after the fork), as production code does: otherwise a signal sent before
  the child is scheduled misses with ESRCH, and on a loaded machine TERM and KILL both miss. A
  test that checks a reaper's work probes with `kill(pid, 0)`, never a second `waitpid` that
  races the reaper (metis#72, failed the merge-time suite under load).

## Working rule

For every experiment, write down what is held constant, what is allowed to vary,
what data can be seen at prediction time, and which artifact proves the answer.
Prefer paired comparisons and explicit provenance over a plausible single number.
