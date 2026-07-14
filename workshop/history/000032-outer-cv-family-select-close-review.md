# Boundary Review — metis#32 (whole-issue close)

| field | value |
|-------|-------|
| issue | 32 — outer-CV model-family selection — close the loop (nested-CV selects, not just reports) |
| repo | metis |
| issue file | workshop/issues/000032-outer-cv-family-select.md |
| boundary | whole-issue close |
| milestone | — |
| window | 51a7854f767da13982e772317d9c2a82d1e08b9b..HEAD |
| command | sdlc close --issue 32 |
| reviewer | claude |
| timestamp | 2026-07-14T00:05:11-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have a complete picture. All Done-when bullets are delivered and verified, the build/vet/tests pass (including `-race` on `cmd/metis`), and the two prior milestone reviews' Important findings (fingerprint join-soundness guard, stale datatype docs, dead `-m` flag) are all resolved in the final `1261830` commit. Only Minor nits remain.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** metis#32 lands its headline capability cleanly and completely: the honest outer-CV estimate now *steers* model-family selection instead of merely reporting it. `metis run` derives its mode from config-count (`>1`→nested, `1`→flat single-level CV, `--fast`→1 outer fold), records inner+outer rows under a `Level`-keyed ledger schema that provably can't collide, and no longer ships; `metis select` reads that ledger, picks the family on the honest outer estimate (`FamilySelect`, lowest-SE-within-1-SE, *not* `SweepResult.Ship`) and the config on the inner CV, and `--promote` reconstructs+ships on all data. I verified `go build`/`go vet`, `go test ./pkg/{ledger,sampler,experiment}` and `go test -race ./cmd/metis` all green, confirmed the acceptance gate genuinely proves the flip (ships the generalizer over the inner-CV overfitter), and confirmed the join-soundness fingerprint guard the M2 review flagged as missing is now implemented and tested. Every `## Done when` bullet is delivered. Nothing blocks SHIP — the residue is comment/dead-field/determinism nits and a couple of small coverage gaps.

### 1. Strengths
- **The acceptance gate is real, not a mock reasserting the impl** (`cmd/metis/select_cmd_test.go` `TestSelect_PicksGeneralizerNotInnerOverfitter`): inner rows favor the rf deep tree, outer rows favor logreg, and the test asserts the *ship* section names logreg and NOT rf — the exact flip #32 exists to produce, pinned honestly. Traced the numbers: logreg outer mean 0.81/SE 0.01 vs rf 0.78/SE 0.04, band `≥0.80` excludes rf → logreg ships. Correct.
- **The `Level`-keyed collision fix is exact and back-compatible.** `pkg/ledger/ledger.go:255` puts `Level` in the `AggregateView` group key so an inner subset-score never blends with an outer held-out score, while ragged `Encode`/`Decode` (`ledger.go:98-155`) keeps a v1/flat ledger byte-identical (no `level`/`outer_fold` columns until used); decode is header-name-driven so column order is safe. `TestAggregateView_LevelKeyedNoCollision` + `TestEncodeDecode_LevelOuterFoldRoundTrip` pin both.
- **The `FamilySelect` ≠ `SweepResult.Ship` decision is load-bearing and correct** (`pkg/sampler/select.go:216`) — the family choice rides the honest outer estimate, degrading cleanly to argmax-mean with a caveat under one fold; `betterMean` reuse keeps direction handling in one place.
- **Clean ARCH-PURE separation, dead code fully removed.** `FamilyEstimate` (`cmd/metis/family.go`) and `FamilySelect` are pure and unit-tested without IO; `shipWinner`, `cmdPromote`, `renderPromoted`, `parsePointSelector`, `ledgerParseCell`, `yamlInline` are all gone, and `freeParamTupleMap` is correctly retained for `ledger show`'s sibling use (the `lessons.md` package-wide-grep lesson).
- **Concurrency of the nested recording is guarded and *proven*** — `ss.manMu` guards `man.Points`, `sortPointRuns` runs before persist for byte-determinism, and `-race ./cmd/metis` is clean including the peak-concurrency test.

### 2. Critical findings
None.

### 3. Important findings
None. The three Important findings from the M2 boundary review are resolved in this window: the fingerprint join-soundness guard is implemented (`distinctFingerprints` + the `o.fingerprint==""` refusal at `select_cmd.go:105-110`, `TestSelect_MixedFingerprintCohortsError`); the datatype doc's required-fields table no longer lists `driver:` (`construct/datatype/experiment-shape.md:41-55`); the no-op `-m` flag is removed.

### 4. Minor findings
- **Dead struct fields** `familyPick.est` / `hasEst` (`cmd/metis/select_cmd.go:168-169`) are set (lines 140, 152) but never read — `printSelect` uses the `est` map param, `promoteSelected` uses `winner`. Drop them.
- **Stale comments referencing retired commands** (the M2 Minor, not yet swept): `pkg/ledger/ledger.go:233` ("`promote --best` + `ledger select`"), `pkg/sampler/select.go:261` ("the offline `metis ledger select` path"), `cmd/metis/ledger.go:102` (header "… + `metis promote`"). Reword to `metis select`.
- **Residual "driver" prose in the datatype doc** at `experiment-shape.md:30,125,127` ("no sweeper/driver", "the driver honestly evaluates") — the field is gone; reword to "outer evaluator" for consistency with the rewritten §"The run mode".
- **`FamilySelect` map-iteration nondeterminism on exact ties** (`pkg/sampler/select.go:222-225,241-248`): `bestFam` and `pick` are chosen by scanning a `map` with strict comparisons, so an exact (mean, SE) tie resolves by Go map order. Real-world risk is ~nil (float ties), but given metis's sort-everything determinism posture, iterating sorted family keys would close it.
- **Cross-issue drift:** issue #29 (`workshop/issues/000029-driver-cv-confinement-e2e.md`, status `open`) still specifies an e2e test over a shape-authored `driver: cv` block, which can no longer be authored. Not in #32's committed migration surface, but the operator should re-scope #29 since its premise is now invalidated.

### 5. Test coverage notes
- Coverage of the *delivered* behavior is strong and honest: the flip gate, the pure family reducer (`family_test.go` proves cross-outer-fold pooling by `FamilyOf` and that inner rows are ignored), the sharp-error paths (multi-family-no-outer, mixed-fingerprint, empty-ship), the collision test, encode/decode round-trip, the 1-config degenerate path, and the re-homed ship-capture invariant (`TestSelectPromote_ShipRunIsCodeCaptured`).
- Small gaps (non-blocking): no end-to-end `runSelect` test over a `--fast` (single-outer-fold) ledger — the pure `FamilySelect` single-fold path is unit-tested (`TestFamilySelect_SingleFoldArgmaxMean`), but the command wiring over such a ledger isn't; and the asymmetric "family has an outer estimate but no inner winner" handling (`--best` errors at `select_cmd.go:149`, `--best-per-model-class` silently drops at `:139`) is untested. Consider one policy + a test.

### 6. Architecture
- **ARCH-DRY — pass with a note.** The correct non-reuse (`FamilySelect` ≠ `SweepResult.Ship`) is right, and `--promote` funnels through the shared `promotedExperiment`+`runResolvedExperiment` engine. Mild residual duplication: `familyEstimateFromLedger` (`select_cmd.go:195-209`) and `configStatsFromLedger` (`:264-305`) each independently `shape.Expand(sh.Pipeline,0)` + `freeParamsEqual`-match rows→points; a shared `row→Point` matcher would consolidate. Not blocking.
- **ARCH-PURE — pass.** Decisions live in pure, directly-tested functions (`FamilyEstimate`, `FamilySelect`, `SelectConfigs`, `Aggregate`); `runSelect`/`promoteSelected`/`runNestedCV` are the thin IO seam that reads records and writes the ledger. No business logic buried in IO.
- **ARCH-PURPOSE — pass.** Shadow-sweep of "the honest estimate steers selection": `metis run` records inner+outer from source (enforced by config-count dispatch, tested), `metis run` no longer ships, `metis select` derives the family from outer rows and the config from inner rows, `ledger select`+`promote` are deleted, and the acceptance gate proves the flip. The one deferred consumer — the kbench `RUNBOOK-sweep.md` — is a legitimately separable *peer-repo* write, correctly out of scope for the metis close. The `--best-per-model-class` per-family shape is the ready metis#22 ensembling seam.

### 7. Plan revision recommendations
None required — the plan already carries the M1 and M2 `## Revisions` entries reconciling `runModeFor` (inlined), `FamilyEstimate`'s siting (`cmd/metis/family.go`), and the now-delivered fingerprint guard. The plan matches the code.
