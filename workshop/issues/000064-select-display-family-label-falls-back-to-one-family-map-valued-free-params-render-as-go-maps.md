---
id: 000064
status: codecomplete
deps: []
github_issue:
created: 2026-07-19
updated: 2026-07-19
estimate_hours: 0.5
started: 2026-07-19T00:11:37-07:00
actual_hours: 0.23
---

# select display: family label falls back to (one family); map-valued free-params render as Go maps

## Problem

First sweep with a MAP-VALUED top-level `with` free-param (`decide: {$any: [argmax,
{offsets: {holdout: 0.2}}]}`, kbench M4 cohort a50b6f25) exposed two display defects in
`metis select`:
1. The per-family honest-estimate line printed `(one family)` where `train.model=hist_gbm`
   belonged (rf rendered fine; the underlying outer rows are correct — reading the ledger
   directly shows both families' rows intact). Likely the family-label renderer choking on
   the map-valued free-param somewhere in the label path.
2. Free-param rendering uses Go's map syntax: `train.decide=map[offsets:map[holdout:0.2]]`
   — unreadable in select output and ledger `fp.*` values; should render canonically (e.g.
   `offsets{holdout:0.2}` or compact JSON), stably (map iteration order must not leak).

Display-only (selection itself picked correctly); annoying at exactly the moment the
operator reads results.


## Root cause (diagnosed 2026-07-19; upgrades this from display-only to FUNCTIONAL)

`cell(nil)` encodes a null free-param as an empty CSV cell; the decode loop SKIPS empty
cells (`rec[i] == "" → continue`) — so post-roundtrip the row's FreeParams lacks the key,
while the expanded point carries an explicit nil. `freeParamsEqual` marshals both → byte
mismatch → family key "" → "(one family)" label AND the family's winner silently dropped
from --best-per-model-class listing and promote. First tripped by kbench M4 (cohort
a50b6f25): all three gbm fold winners were cw=null configs (pct-loss parsimony picked them
within tolerance). Operator workaround used: --point promote of a non-null cell.

## Spec

- **Matcher canonicalization:** in `freeParamsEqual`, drop nil-valued entries from the
  point's map before marshal — null ≡ absent, matching what the CSV round-trip does.
  Retroactive: heals existing ledgers (the M4 cohort renders correctly with no re-run).
- **Display:** `freeParamStrFromParams` + `freeParamMapStr` render map/slice values as
  compact JSON (not Go `%v` — `map[offsets:map[holdout:0.2]]`). `famLabel`'s "(one
  family)" fallback stays (legit for genuinely stale rows).
- **Regression tests:** (a) freeParamsEqual: nil-valued point param vs key-absent row →
  equal; distinct configs stay unequal; (b) familyEstimateFromLedger on a shape with a
  null rung + a round-tripped ledger → family label non-empty; (c) freeParamStr JSON
  rendering.


## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.1 impl=0.15
item: milestone-review    design=0.0 impl=0.2
design-buffer: 0.15
total: 0.47
```

(Fix+tests · close review. Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.)

## Done when

- The M4 cohort (a50b6f25) select shows `train.model=hist_gbm` (not "(one family)") and
  BOTH families in --best-per-model-class, with NO re-run (retroactive heal) — verified
  against the live kbench ledger.
- Null-vs-absent equality + display rendering unit-tested; suites green.

## Plan

(Simple work — issue-level plan.)

- [x] matcher nil-drop + display JSON rendering + 3 regression tests; verify against the
  live M4 ledger; pr/merge/close

## Log

### 2026-07-19
- 2026-07-19: closed — pure bugfix, no new architectural surface (matcher canonicalization + display rendering). Regression tests green (null==absent, distinct unequal, JSON rendering); full Go+python suites; RETROACTIVE heal verified on the live kbench M4 ledger without re-run — both families label+list correctly incl. the previously-dropped cw=null gbm winner; review verdict: FIX-THEN-SHIP

### 2026-07-19 (fix)

- Matcher nil-drop in freeParamsEqual (null ≡ absent, canonical with the CSV round-trip's
  empty-cell skip); renderFreeParamValue: composites → compact JSON, nil → `null`; wired
  into freeParamStrFromParams + freeParamMapStr. 3 regression tests. Full Go + python green.
- RETROACTIVE HEAL VERIFIED on the live kbench M4 ledger (a50b6f25), no re-run:
  `train.model=hist_gbm` labels correctly, BOTH families in --best-per-model-class
  (gbm winner = cw=null + offsets — the very row that was silently dropped), decide renders
  as JSON. The operator's --point workaround run (point-hist_gbm-501f3358) remains valid
  blend material.
- Close-review FIX-THEN-SHIP fixes bundled: test (b) delivered (Encode→Decode round-trip
  through the REAL cell/decode seam — found+fixed a test-data bug of its own: fixed params
  are not free params), and the nil-drop made symmetric on `want` (fresh in-memory rows
  with explicit nils now match too).
