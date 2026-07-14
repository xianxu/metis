# Boundary Review — metis#32 (milestone M1)

| field | value |
|-------|-------|
| issue | 32 — outer-CV model-family selection — close the loop (nested-CV selects, not just reports) |
| repo | metis |
| issue file | workshop/issues/000032-outer-cv-family-select.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 615aead73e0b2b3e80a8ba170d8f45fe2df69767^..HEAD |
| command | sdlc milestone-close --issue 32 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-13T21:47:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core M1 deliverable — making a nested `metis run` **record** the full inner+outer CV to the ledger, driven by a config-count dispatch that replaces the deleted `driver:` field — is correct, well-guarded under concurrency, and race-clean. I verified `go build`/`go vet`, the full `go test ./...`, and `-race` on `cmd/metis`, `pkg/{sampler,ledger,experiment}` (all green), and exercised all three dispatch modes via dry-run (1-config→single-level, ≥2→nested, `--fast`→outerK=1 — all print correctly). The honest estimate stays byte-faithful to #23 (the CVDriver still aggregates the ship-family's outer score), and the Level-keyed `AggregateView` collision fix is real and precisely tested. What keeps this from a clean SHIP: the flat single-level / 1-config path and `--fast` have **zero automated coverage** — and the M1 Done-when explicitly committed "a hermetic test covers the 1-config degenerate path" — and the schema-companion docs still describe the now-deleted `driver:` field as current. Both are non-blocking at the gate but should be closed before recording the verdict.

### 1. Strengths
- **Concurrency of the new nested recording is genuinely well-guarded and *proven*, not asserted.** `ss.manMu` for `man.Points`, per-outer-fold isolated `sweepPass` with `pass.mu`, the `errMu` latch, and `syncWriter` over `out` — and `TestSweep_ParallelEqualsSerial` (now exercising the **nested** path) asserts byte-identical ledger CSV + manifest serial-vs-parallel, backed by `sortPointRuns` before persist (`sweep.go:398`). `-race` clean including `TestNestedCV_PeakConcurrencyWithinCap`.
- **The Level-keyed collision fix is exact and back-compatible.** `ledger.go:258` puts `Level` in the group key (inner subset-score never blends with the outer held-out score), while ragged `Encode`/`Decode` (`ledger.go:100-124`) keeps a v1/flat ledger byte-identical (no `level`/`outer_fold` columns unless used). Decode is by header name, so column reordering is safe.
- **Estimate faithfully preserved while extending recording.** `runOuterFold` now scores every family but returns the ship-family's score (`sweep.go:499-501`), and `Ship == PerFamily[Ship.Family]` (`select.go:88-92`), so #23's number is unchanged — a careful, verified preservation.
- **Clean ARCH-PURE separation.** `FamilyEstimate` (`family.go`) and the `SeqExec`/`ParExec`/`ExecFor` combinators are pure and unit-tested without IO; recording IO stays in the shell.
- **Thorough `driver:` removal** (shape.go struct+validator, CUE, ~10 test fixtures) and a sharp `lessons.md` capture (a content-address-feeding field can't be milestone-split from its derived-dispatch consumers).

### 2. Critical findings
None.

### 3. Important findings
- **Flat single-level / 1-config path and `--fast` have no test coverage; the M1 Done-when's "hermetic test covers the 1-config degenerate path" is unsatisfied.** Every `runShapeSweep` test uses ≥2 configs → all now route to the nested branch, so the entire `nested==false` branch (`sweep.go:240-284`: `SingleDriver` → `reportWinner` → flat `GuardComplexity` → `writeSweepLedger` with `Level=""` → **no ship**) and the `o.fast`/`runFolds=1` logic are unexercised. I confirmed dispatch is correct via dry-run, but nothing pins the record-and-no-ship behavior. *Fix:* add a hermetic 1-config fake-exec test (asserts `Level=""` rows recorded, zero submission artifacts) + a `--fast` test on a multi-config shape (asserts one outer fold, one held-out line per family). `cmd/metis/nestedcv_e2e_test.go` / `shapesweep_test.go`.
- **Docs now contradict the M1 schema change.** `driver:` was deleted from CUE but is still documented as current in `construct/datatype/experiment-shape.md:78-86` ("### The driver block"), `atlas/experiment.md:150-159`, and `atlas/index.md:70-77` — and the atlas claim that `driver:cv` "writes **NO** grouped manifest/ledger" is now *inverted* by M1 (the nested path records). Per AGENTS.md §8 and the docs gate, this is stale-describing-removed-surface, not just missing docs. The plan's boundary revision defers the full atlas/RUNBOOK rewrite to M2 (defensible — it couples to `metis select`), but the schema-companion **datatype doc** especially should track the field removal at M1. *Fix:* at minimum a one-line stale-marker + correct the `driver:cv` "records nothing" claim now.

### 4. Minor findings
- Set-once error-latch is duplicated: `runNestedCV` inlines `errMu`/`firstErr`/`setFirst`/`getFirst` (`sweep.go:363-376`), replicating `sweepPass.setErr`/`firstError` (`sweep.go:126-140`). Extract a tiny shared `errLatch` type (ARCH-DRY).
- `reportWinner` inlines the family-key sort (`sweep.go:746-750`) that `sortedFamilies` (`sweep.go:530`) now provides — reuse it (ARCH-DRY).
- The nested-CV preamble (`outer-split`) run is not in `man.Points`, so `captureSweepCode` omits `outer_split.py`'s closure; on a dirty tree the nested run's provenance no longer pins the outer-split code version. Pre-existing #23 property, but #32 now writes a ledger for this path, so it's newly relevant.
- The 1-config flat path records `Level=""` rows, but the spec table calls them "inner rows" — a naming mismatch, harmless in M1.

### 5. Test coverage notes
- **Nested path: excellent.** Inner/outer counts, `OuterFold` on inner rows, capture-before-ledger fingerprint ordering, no-ship, byte-identical serial/parallel, peak-concurrency-within-cap. The reader-vs-writer atomic-index test and the real-`execStep` semaphore test (from #31, in-window) are strong.
- **Gaps:** flat/1-config/`--fast` (Important #1). `FamilyEstimate` is unit-tested but has no production caller in M1 (dead until M2) — acceptable per plan.

### 6. Architectural notes for upcoming work (M2)
- `FamilyEstimate`'s production `familyOf` must reconstruct a `shape.Point` from a `ledger.Row` and call `sampler.FamilyOf` — verify that round-trip is faithful for tagged-union model configs.
- A 1-config ledger has **no outer rows** → `FamilyEstimate` returns empty; decide `select`'s behavior (trivial-promote vs the "lacks required rows" sharp error).
- `AggregateView`'s aggregate-row `PointAddr` changed from `cf|fpb` to `cf||fpb` (`ledger.go:258`); confirm no M2 consumer parses `PointAddr` positionally (`promotedExperiment` matches on free-params, so it's fine — but check when retiring `promote`).

### 7. Plan revision recommendations
- **`## Revisions` entry (Core concepts drift):** the table lists `runModeFor(shape)` as a *new pure function* in `sweep.go`, but the dispatch was inlined as `nested := len(configPts) > 1` (`sweep.go:204`) — no such function exists. Record that it was inlined (a one-liner didn't warrant extraction), or extract it to match the table.
- **`## Revisions` entry (siting):** the table sites `FamilyEstimate` at `pkg/ledger/ledger.go (or cmd/metis/select_cmd.go)`; it landed in a new `cmd/metis/family.go` (consistent with Task 1.2's allowance) — update the location line.
- **`## Revisions` entry (Done-when):** note the M1 Done-when's "hermetic test covers the 1-config degenerate path" is not yet satisfied — either add it before close or record the deferral rationale.
