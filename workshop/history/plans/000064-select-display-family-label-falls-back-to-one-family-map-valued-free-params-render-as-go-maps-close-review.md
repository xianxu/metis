# Boundary Review — metis#64 (whole-issue close)

| field | value |
|-------|-------|
| issue | 64 — select display: family label falls back to (one family); map-valued free-params render as Go maps |
| repo | metis |
| issue file | workshop/issues/000064-select-display-family-label-falls-back-to-one-family-map-valued-free-params-render-as-go-maps.md |
| boundary | whole-issue close |
| milestone | — |
| window | e1b68841be505a968a51e6a35940c575a2d0fead^..HEAD |
| command | sdlc close --issue 64 |
| reviewer | claude |
| timestamp | 2026-07-19T00:21:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All matching consumers (`runSelect`, `runPromote`, `ledger show`) load the ledger from disk via `loadLedger`, so the row side is always CSV-round-tripped (nulls already key-absent) — the one-sided nil-drop is correct for every live call path. I have everything needed; here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is correct, minimal, and lands at the right layer. I independently confirmed the root cause in the code: `cell(nil)` encodes a null free-param as an empty CSV cell (`pkg/ledger/ledger.go:435`) and the decode loop skips empty cells (`pkg/ledger/ledger.go:184`), so a round-tripped row is key-absent where the expanded point carries an explicit nil — exactly as the issue's Root cause section claims. The matcher-side canonicalization (drop nil entries from the point's map before marshal) heals existing ledgers retroactively with no format change, and I verified it is safe: `freeParamMap` builds a fresh map per call (`cmd/metis/sweep.go:978`), so the in-place delete mutates nothing shared, and deleting during range is well-defined Go. I also swept the other matching surfaces for the same hazard: `freeParamMapsEqual` (`select_cmd.go:458`) only ever compares row-vs-row from one decoded ledger (both sides key-absent — consistent), and ledger dedup keys on `(PointAddr, CodeFingerprint)`, not free-params, so the asymmetry can't duplicate rows. What keeps this from a clean SHIP is a traceability gap: the Spec promises three regression tests and test (b) — `familyEstimateFromLedger` on a shape with a null rung against a round-tripped ledger → non-empty family label — was not delivered; the Log's "3 regression tests" overstates (two test functions landed). Note: I could not execute the test suite — Bash is failing at the harness level in this session (session-env mkdir EPERM, persists even unsandboxed) — so "suites green" rests on the Log's claim; the tests were verified by reading only.

**1. Strengths**

- Root cause diagnosis upgraded a "display-only" bug to a functional one (silently dropped family winner in `--best-per-model-class` and promote) and the fix targets that root cause, not the symptom — no `famLabel` patch-over; the "(one family)" fallback correctly stays for genuinely stale rows (`select_cmd.go:273`). Senior-level Root-Cause discipline.
- The matcher-side canonicalization is the right design choice among the alternatives (vs. changing CSV encoding or a decode-side sentinel): zero format change, retroactively heals existing ledgers, and every live consumer funnels through the one predicate `freeParamsEqual` — `promotedExperiment` (`ledger.go:67`), `configStatsFromLedger` (`select_cmd.go:305`), `familyEstimateFromLedger` (`:210`), `pointHandleFor` (`:266`), `runPointSelect` (`:485`) — so one fix covers select, promote, and point-handles at once.
- `renderFreeParamValue` (`sweep.go:993`) consolidates both display call sites (`freeParamStrFromParams` and `freeParamMapStr`) onto one helper rather than patching each `%v` in place, and its output is deterministic: `json.Marshal` sorts map keys and `freeParamMapStr` sorts its keys, satisfying the Spec's "map iteration order must not leak" requirement.
- The negative test cases in `TestFreeParamsEqual_NullEqualsAbsent` (`ledger_test.go:128-138`) pin the two ways this fix could over-reach: distinct configs must stay unequal, and a real non-null param must not be droppable. That last case is exactly the bug a sloppier "ignore missing keys" fix would have shipped.

**2. Critical findings**

None.

**3. Important findings**

- **Spec test (b) not delivered** (`cmd/metis/ledger_test.go`, Spec line 49-52). The Spec commits to three regression tests; only (a) `freeParamsEqual` and (c) JSON rendering landed. Missing: `familyEstimateFromLedger` on a shape with a null rung + a round-tripped (Encode→Decode) ledger → family label non-empty. Test (a) pins the predicate, but (b) is the only test that would pin the end-to-end symptom through the real CSV round-trip — the integration seam (`cell`/decode-skip ↔ matcher) is exactly where this bug lived, and a future change to `cell`'s null encoding would slip past test (a). Cheap to add: build a 2-rung shape with a nil-valued param, Encode/Decode a small ledger, assert `familyOf` resolves. Also fixes the Log's "3 regression tests" overstatement. (ARCH-PURPOSE: the committed test scope is the one under-delivered item; everything else in Spec/Done-when is delivered.)

**4. Minor findings**

- The nil-drop is one-sided: if a future caller ever passes fresh in-memory rows (whose `FreeParams` from a JSON manifest do carry explicit nils) as `want`, the match fails the other way. All current callers go through `loadLedger`, so this is latent only — a symmetric drop on `want` (or a comment stating the round-tripped-only precondition) would harden it. `cmd/metis/ledger.go:101`.
- Same-class latent gap, pre-existing: an empty-*string* free-param (`cell("")` → `""` → decode-skip → absent) has the identical absent-vs-present mismatch and is not canonicalized. Not this issue's scope; worth a note in the issue Log or a follow-up if `""` rungs ever appear.
- Comment drift at `ledger.go:101`: `// reuses the sweep driver's renderer` — `freeParamMap` is a map builder, not a renderer (the comment described the old inline-marshal line).
- `promotedExperiment`'s error message still renders free-params with `%v` (`ledger.go:71`) — Go map syntax in the one user-facing string not migrated to `freeParamMapStr`. Cosmetic.

**5. Test coverage notes**

The predicate and rendering are pinned pure, no IO — good. Gaps: (i) Spec test (b) above; (ii) `renderFreeParamValue`'s `nil → "null"` branch and `[]any → JSON` branch have no direct assertion (only the nested-map case is tested); one more `want` string in `TestFreeParamStr_CompositeValuesRenderAsJSON` covers both cheaply. Caveat repeated for the record: I could not run the suites (harness Bash failure); the main agent should re-run `go test ./...` before close.

**6. Architectural notes**

- **ARCH-DRY: pass, with one deliberate non-reuse to affirm.** `renderFreeParamValue` structurally mirrors `pkg/ledger.cell` (nil / composite-JSON / `%v` switch), but the divergence is load-bearing: `cell`'s `nil → ""` is the storage contract this very bug hinged on, while display needs `nil → "null"`. Sharing them would couple human rendering to CSV encoding — the non-reuse is correct. Within the display layer, both call sites derive from the one helper. Pass.
- **ARCH-PURE: pass.** Both touched functions are pure and tested without IO; the fix adds no IO-layer logic. Pass.
- **ARCH-PURPOSE: pass with the one finding.** Shadow-sweep of consumers of the null≡absent canonicalization: all five matcher call sites funnel through `freeParamsEqual`; `freeParamMapsEqual` is self-consistent by construction; dedup/aggregate keys are unaffected or self-consistent per source. The purpose (correct family labels + no silently dropped winners, retroactively) is delivered; the deferred item is spec test (b), flagged above — it's a pinning gap, not a deferred purpose.
- Docs gate: no new user-facing surface, subcommand, or flag; atlas/README updates not required for this bugfix. Pass (a `--no-atlas` close is legitimate here per §5).

**7. Plan revision recommendations**

- Either deliver test (b) and keep the Log line, or append a `## Revisions` entry to the issue noting test (b) was dropped in favor of the live-ledger manual verification, and correct "3 regression tests" → "2 regression tests (spec test (b) waived: live M4 ledger verification covers the end-to-end path)". The issue file must not claim tests the diff doesn't contain.
