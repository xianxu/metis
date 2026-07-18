---
id: 000053
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-17
estimate_hours:
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

## Plan

- [ ] (at claim) Confirm D-path blob hashing against the working tree matches capture's
  hashing (same normalization); TDD the compare + refusal; wire both promote paths.

## Log

### 2026-07-16
- Filed from the operator's restore question. Split: metis#28 = reconstruct/re-run a recorded
  state (restore); THIS = cheap detection at the one seam where drift silently invalidates the
  selection (promote). Detection needs no restore machinery — hash the current tree over the
  cohort's D paths and compare.
