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

## Working rule

For every experiment, write down what is held constant, what is allowed to vary,
what data can be seen at prediction time, and which artifact proves the answer.
Prefer paired comparisons and explicit provenance over a plausible single number.
