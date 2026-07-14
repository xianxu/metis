# Boundary Review — metis#32 (milestone M2)

| field | value |
|-------|-------|
| issue | 32 — outer-CV model-family selection — close the loop (nested-CV selects, not just reports) |
| repo | metis |
| issue file | workshop/issues/000032-outer-cv-family-select.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c0e7e13973245bd1731053fa69180cb14e063cb8..HEAD |
| command | sdlc milestone-close --issue 32 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-13T23:53:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** M2 delivers the headline capability cleanly: the honest outer-CV estimate now *steers* family selection rather than just reporting it. The acceptance gate (`TestSelect_PicksGeneralizerNotInnerOverfitter`) genuinely proves the flip — `metis select --best` ships the logreg generalizer over the rf inner-CV overfitter — and the pure reducers (`FamilySelect`, `FamilyEstimate`) are well-separated and unit-tested. The retirement of `metis promote`/`ledger select` and `shipWinner` is complete and the build/tests are green under the relevant packages. What blocks a clean SHIP is a set of cheap, targeted gaps: the **fingerprint join-soundness guard the spec explicitly required is not implemented** (Done-when + plan Task 2.2 both call for it), the datatype authoring doc still advertises the deleted `driver:` field as required, and a documented `-m` flag is a silent no-op. None are crashes or break the core feature — all are "fix-before-boundary-if-cheap."

### 1. Strengths
- **The acceptance gate is real, not a mock reasserting the impl** (`cmd/metis/select_cmd_test.go` `TestSelect_PicksGeneralizerNotInnerOverfitter`): inner rows favor rf md=8, outer rows favor logreg, and the test asserts the *ship* section names logreg and NOT rf — this is the whole point of #32 and it's pinned honestly.
- **`FamilyEstimate` has a genuine pure unit test** (`cmd/metis/family_test.go`) proving cross-outer-fold pooling by `FamilyOf` (rf md4/md8 pool) and that inner rows are ignored — the exact reduction `AggregateView` cannot do. Clean ARCH-PURE separation.
- **The sharp-error contract is delivered and tested** (`TestSelect_MultiFamilyNoOuterRowsErrors`): a multi-family inner-only ledger errors pointing at the missing outer rows, never a silent inner-argmax — the anti-overfit invariant.
- **`FamilySelect` correctly refuses to reuse `SweepResult.Ship`** (`pkg/sampler/select.go:216`), exactly as the spec's I-3 mandates; the `--promote` path funnels through the shared `promotedExperiment` + `runResolvedExperiment` engine (ARCH-DRY satisfied).
- Dead-code removal is thorough (`shipWinner`, `cmdPromote`, `renderPromoted`, `parsePointSelector`, `ledgerParseCell`, `yamlInline` all gone; `freeParamTupleMap` correctly retained for `ledger show`).

### 2. Critical findings
None.

### 3. Important findings

**I-1. Fingerprint join-soundness is not enforced — a mixed-code ledger silently blends versions** (`cmd/metis/select_cmd.go:79`).
`runSelect` only does `led = ledger.Filter(led, o.fingerprint)`; with `--fingerprint` omitted (the default), `Filter` returns the *whole* ledger. But the spec §"Join soundness" says the family↔config reduce **MUST be pinned to one `code_fingerprint`**, Done-when line 180 requires "the family↔config join is fingerprint-scoped," and plan Task 2.2 Step 3 literally says "fingerprint-scoped: pin one `code_fingerprint`, **error sharply on a mixed/absent set**." None of that is implemented. Failure scenario: run a sweep, change a step's code, re-run — the ledger now carries two fingerprint cohorts (append-only, `dedupKey = pointAddr+fingerprint`). `metis select` (no flag) then (a) pools *both* cohorts' outer rows into one family mean in `FamilyEstimate` (family grouping ignores fingerprint), and (b) surfaces the same config twice as two `ConfigStat`s in `configStatsFromLedger` (AggregateView keys on fingerprint) so `SelectConfigs` picks across code versions. This is the documented `workshop/lessons.md:106` footgun and the exact silently-wrong-winner class #32 exists to stop. Fix: after `Filter`, if `o.fingerprint == ""` and the needed rows span >1 distinct `CodeFingerprint`, error with the list (or default to the latest cohort) — ~10 lines; add a mixed-ledger test.

**I-2. The datatype authoring doc still lists the deleted `driver:` field as required** (`construct/datatype/experiment-shape.md:50`, also lines 4, 11, 19, 27, 37, 66).
The doc contradicts *itself*: line 50's required-fields table says `| driver | yes | The outer Sampler — exactly one of single | cv.`, while the new §"The run mode (derived — no `driver:` block; metis#32)" (lines 78–90) says the field was removed. Since `ParseShape` uses `dec.KnownFields(true)` (`pkg/experiment/shape.go:114`), a shape authored per line 50 now **fails to parse**. The M1 revision claimed the run-mode section was rewritten, but the frontmatter description, intro, field table, and `ValidateShape` references were left describing `driver` as live. This is a Docs-gate trap on the primary shape-authoring guide. Fix: purge/reword the six stale `driver` references to match the derived-mode section.

**I-3. `-m MSG` is a documented flag that does nothing** (`cmd/metis/select_cmd.go:33,46,56`).
`msg` is parsed and stored in `selectOpts.message`, but `message` is never read anywhere (`runOpts` has no message field; `promoteSelected` never threads it). The `--help` text ("message recorded on the promoted run(s)") promises a behavior that silently doesn't happen — and `-m` isn't in the spec's `select` signature at all (undeclared scope). Fix: remove the flag (YAGNI) or actually record the message on the run.

### 4. Minor findings
- Dead struct fields `familyPick.est` / `familyPick.hasEst` (`select_cmd.go:137-138`) — set but never read (`printSelect` uses the `est` map param; `promoteSelected` uses `winner`). Drop them.
- Stale code comments referencing retired commands: `cmd/metis/ledger.go:102` ("… + `metis promote`"), `pkg/sampler/select.go:261` ("the offline `metis ledger select` path"), `pkg/ledger/ledger.go:233` ("`promote --best` + `ledger select` read over"). Reword to `metis select`.
- `FamilySelect` (`pkg/sampler/select.go:220-249`) scans a `map` for best/pick with strict comparisons → on an exact (mean, SE) tie the winner depends on Go map-iteration order (non-deterministic). Given metis's determinism posture (sortPointRuns everywhere), iterate sorted family keys. Very low real-world risk (float ties).
- Inconsistent handling of a family with an outer estimate but no inner winner: `--best` errors sharply (`select_cmd.go:117-120`) but `--best-per-model-class` silently drops it (`select_cmd.go:108`). Pick one policy.
- `--best-per-model-class` over a multi-family inner-only ledger errors via the `len(est)==0` path (`select_cmd.go:100-102`) even though a per-family report arguably doesn't need a cross-family outer estimate. Defensible, but note it if intentional.

### 5. Test coverage notes
- Coverage of the *delivered* behavior is strong and honest: the flip gate, the pure family reducer, the sharp-error path, and the `--promote` ship-run (`TestSelectPromote_MaterializesShipRunOnAllData` / `_ShipRunIsCodeCaptured` drive the real `runResolvedExperiment` via `foldFakeExec`, correctly re-homing the M1-removed ship-capture invariant).
- The one coverage gap that matters is the flip side of I-1: there is **no test for a mixed-fingerprint ledger** — because the guard doesn't exist. Add the guard *and* its test together. I-3 also has no test (an assertion that `-m` records something would have surfaced the no-op).

### 6. Architectural notes for upcoming work
- **ARCH-DRY — pass with a note.** The correct non-reuse (`FamilySelect` ≠ `SweepResult.Ship`) is the load-bearing decision and it's right. Minor duplication: `familyEstimateFromLedger` and `configStatsFromLedger` each independently `shape.Expand` + `freeParamsEqual`-match rows to points (`select_cmd.go:164-178` vs `233-274`); a shared "row→Point" matcher would consolidate. Not blocking.
- **ARCH-PURE — pass.** Decisions live in pure, directly-tested functions (`FamilySelect`, `FamilyEstimate`, `SelectConfigs`); `runSelect`/`promoteSelected` are the thin IO seam. No business logic buried in IO.
- **ARCH-PURPOSE — pass with a note.** Shadow-sweep of the "honest estimate steers" purpose: `metis select` derives the family from outer rows (enforced + tested ✓), `metis run` no longer ships ✓, the two retired commands are deleted ✓, and the kbench RUNBOOK is a legitimately separable *peer-repo* deferral. The only purpose blemish is the dead `-m` surface (I-3) — a claimed capability that isn't real. For #22 (ensembling), `--best-per-model-class`'s per-family-winner shape is the intended seam and looks ready.

### 7. Plan revision recommendations
- If I-1 is fixed now (recommended), no plan change needed. If deferred, add a `## Revisions` entry recording that **Task 2.2 Step 3's "fingerprint-scoped: pin one `code_fingerprint`, error sharply on a mixed set" was not delivered** — only optional `--fingerprint` filtering exists — and file a follow-up issue, so the plan stops claiming an enforcement the code doesn't provide.
- Minor: the Core-concepts table (plan lines 27, 29) still sites `FamilyEstimate` in `pkg/ledger/ledger.go` and lowercases `familySelect`; the M1 Revisions note already records the move to `cmd/metis/family.go`, and `FamilySelect` is exported in `pkg/sampler/select.go` as the table's location says — a one-line table touch-up would make it exact, but the existing revision note covers the substance.
