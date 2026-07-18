---
id: 000053
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-17
estimate_hours: 0.48
started: 2026-07-17T23:05:12-07:00
---

# promote fingerprint-consistency guard — refuse when the working tree is not the cohort's code

## Problem

`metis select --fingerprint <fp> --best --promote` selects on ONE code cohort but executes the
promoted run against the CURRENT working tree — with no check that they are the same code.
Same-session promote (tree unchanged) is sound and is the common case. But promote-after-drift
(any edit to a closure file since the sweep) silently ships a submission from code that never
produced the honest estimate the operator selected on — the exact silent-blend class the #32
cohort guard stops at the LEDGER, left open at the PROMOTE seam. Today's only tell is the #39
`recording under code_fingerprint` line on the promote run differing from the pin — visible,
never enforced. (Operator question 2026-07-16: "do we need a way to restore to that state? is
that already happening when we run select --promote?" — answer: no; restore is metis#28,
detection is THIS issue.)

## Spec

Detection only (restore stays metis#28):

- At promote time (both `--best`/`--best-per-model-class` and `--point`), recompute the
  would-be fingerprint of the CURRENT tree over the pinned cohort's D paths (each path's
  working-tree git blob-hash → `record.CodeFingerprint` — the same pure mint) and compare to
  the cohort fingerprint.
- Mismatch → REFUSE with a diff-shaped message: which paths changed (path + old/new short
  blob), the cohort's capture commit (the restore handle: `git checkout <commit>` per repo —
  or metis#28's verb when it lands), and the explicit override `--no-fingerprint-check`
  (escape hatch + loud, per the gate convention).
- No pin (single-cohort ledger): compare against that cohort implicitly — same guard.
- Missing records/D (legacy cohorts) → warn-and-proceed (nothing to compare — never block on
  absent provenance).

## Done when

- A fixture: sweep → edit a closure file → `select --promote` REFUSES naming the changed path;
  `--no-fingerprint-check` proceeds loudly; unchanged tree promotes clean (no false positive).
- The refusal message round-trips: following its checkout hint then re-promoting succeeds.
- Atlas: the promote seam's guard documented beside the #32 cohort guard.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.06 impl=0.35
item: atlas-docs          design=0.01 impl=0.05
design-buffer: 0.15
total: 0.48
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

**Design refinement at claim (same signal, better message):** compare PER-PATH blob hashes
instead of re-minting the run fingerprint — the record's step `Code.D` refs already carry
`{Repo, Path, BlobHash}`; rehash the union (dedup repo+path) of the cohort record's D against
the working tree with the SAME `gitBlobHashes` capture uses (normalization identical by
construction, answering the plan's confirm item) and any differing/missing path IS the drift +
the diff line. Re-minting would require replicating the mint's exact input assembly (dedup/
order) — a needless equivalence risk (ARCH-DRY: reuse the hasher, not a parallel mint).
Records are read via the existing `loadLedgerRecords` (runs/<addr>/record.json); the capture
commit for the restore hint comes from `Steps[].Code.Commit`.

## Plan

- [x] (at claim) hashing-normalization confirm: reuse `gitBlobHashes` — identical by construction
- [x] pure guard core `promoteDrift(records, cohortFP, hasher) ([]driftedPath, hint, ok)` + unit tests (fake hasher: drift / clean / legacy-no-D)
- [x] wire BOTH promote sites pre-exec; `--no-fingerprint-check` flag (loud proceed); warn-and-proceed on absent provenance
- [x] fixture e2e: prepared ledger+record+mini git repo → edit closure file → REFUSE names path+old/new+commit hint; checkout-hint round-trip re-promotes clean; unchanged tree no false positive; override loud
- [x] atlas: promote-seam guard beside the #32 cohort guard; Log evidence

## Log

### 2026-07-16
- Filed from the operator's restore question. Split: metis#28 = reconstruct/re-run a recorded
  state (restore); THIS = cheap detection at the one seam where drift silently invalidates the
  selection (promote). Detection needs no restore machinery — hash the current tree over the
  cohort's D paths and compare.

### 2026-07-17 (built)
- Design refinement held: per-path blob compare (no re-mint) — promoteDrift is pure with an
  injected hasher; guardPromoteFingerprint wires both promote paths pre-exec. 6 tests: 3 unit
  (drift/clean/legacy-unchecked incl. wrong-cohort) + 3 through the REAL runSelect (refuse
  names path+commit-hint; restore-content round-trip re-promotes clean; --point path guarded;
  legacy cohort warns nothing-to-compare and proceeds; override loud). Full -race suite green.
