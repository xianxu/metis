# Boundary Review — metis#50 (whole-issue close)

| field | value |
|-------|-------|
| issue | 50 — run-end summary — elapsed time, fingerprint, rows, and paste-ready next commands |
| repo | metis |
| issue file | workshop/issues/000050-run-end-summary-elapsed-fingerprint-next-commands.md |
| boundary | whole-issue close |
| milestone | — |
| window | fa2b60411cb70ecf5760654de6d0e1d2adf08659..HEAD |
| command | sdlc close --issue 50 |
| reviewer | claude |
| timestamp | 2026-07-16T08:23:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All checks are done — I've verified the diff against the Spec/Plan, the helpers it reuses, the control flow (no double summary), the hint commands' existence, test coverage, and the docs gate. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Tight, well-shaped diff that delivers the issue's purpose on both sweep paths: `printRunSummary` runs at the end of the flat path (`sweep.go:332`) and the nested path (`sweep.go:440`), which are mutually exclusive (nested returns at `sweep.go:282`, so no double print). The fingerprint is returned from the existing single mint site in `captureSweepCode` rather than re-derived, the clock is the injected `now`, and the hinted commands (`metis select … --fingerprint`, `--best --promote`, `metis ledger fingerprints`) all exist as real surface. What keeps this from a clean SHIP is two cheap gaps: the degraded `(cohort ?)` branch — an explicit Done-when bullet — has zero test coverage, and the atlas touch the issue's own estimate budgeted (`atlas-docs` row) never happened.

**1. Strengths**

- ARCH-DRY done right: `captureSweepCode` returning the already-minted fingerprint (`capture.go:167`) instead of re-deriving it is exactly the "one mint site" the Spec called for; `short`, `ledgerPath`, and `fmtETA` are reused, not re-implemented.
- ARCH-PURE: elapsed comes from the injected `now` captured at `sweep.go:184`, and `printRunSummary` writes to an injected `io.Writer` — testable without IO.
- The degraded path degrades honestly in code: `cohort ?` plus un-pinned hints (`sweep.go:903-909`) matches the Spec's "no lying pin" intent.
- TDD evidence is real: the nested test asserts the full block contents (`nestedcv_e2e_test.go:72-82`), the flat test asserts the block appears on the degenerate path too (`shapesweep_test.go:335-339`).

**2. Critical findings** — none.

**3. Important findings**

- **Atlas update missing for the run-end handoff (docs gate).** The estimate block explicitly budgets `item: atlas-docs … docs row = RUNBOOK/atlas touch + smoke evidence`, no RUNBOOK exists in the repo, and neither `atlas/experiment.md` nor `atlas/index.md` changed in the window. `atlas/experiment.md` already documents the run→select flow in detail (lines 168-185: "run measures / select chooses"); the run-end summary is the new operator-facing seam completing that loop (#39's fingerprint visibility → paste-ready select) and should get a sentence there. Fix: one or two lines in `atlas/experiment.md`'s select/honest-selection section noting the run-end summary hands the operator the pinned `select` commands.
- **The degraded `(cohort ?)` branch is untested.** Done-when bullet 2 says "Degraded/absent fingerprint degrades the block gracefully (no lying pin)" — nothing asserts it. Both e2e fixtures exercise a successful capture, so `cohort == ""` never runs under test; a regression that prints an empty `--fingerprint ` pin would ship silently. `printRunSummary` is pure — a three-line unit test calling it with `cohort=""` and asserting `(cohort ?)` appears and `--fingerprint` does not would close this.

**4. Minor findings**

- `nestedcv_e2e_test.go:76`: `s[strings.Index(s, "metis: done in "):]` panics with a slice-bounds error if the summary is absent, because the preceding check is `t.Errorf` (non-fatal). Use `t.Fatalf` or guard the index.
- On a mid-loop backfill error, `captureSweepCode` returns a possibly non-empty `fp` (`capture.go:160`) while skipping the `printFingerprintLine` — so the #50 summary can print a cohort pin that the #39 "recording under" line never announced, and later points lack the backfilled fingerprint. Consider returning `""` on the error path so failure degrades to the honest `cohort ?`.
- Spec suggested `fmtETA → fmtDuration` rename ("or share" — share was chosen, fine), but an "ETA" formatter now formatting elapsed time is a small naming lie; cheap rename when next touching `board.go:96`.
- Spec's example shows grouped digits ("2,160 rows"); `%d` prints `2160`. Cosmetic drift from the illustrative block.
- The fixture clock is frozen (`fixedNow`), so the tests could pin the deterministic `done in 0s` instead of only the `"metis: done in "` prefix — the Spec promised "tests assert a scripted duration."

**5. Test coverage notes**

Nested and flat paths both assert the summary on captured output — the right level (behavior, not mocks). The gaps are the degraded branch (above) and the un-pinned duration value. The three updated `captureSweepCode` call sites in tests correctly discard the new return where irrelevant.

**6. Architectural notes**

- ARCH-DRY: **pass** (see Strengths). One in-the-small note: the two branches of `printRunSummary` duplicate the three-command block; acceptable at this size, but a third variant would warrant folding the pin into a variable.
- ARCH-PURE: **pass** — injected clock and writer; the helper is directly unit-testable (which is also why the missing degraded-branch test is cheap to add).
- ARCH-PURPOSE: **pass** — both sweep paths, both cohort states, delivered; no "follow-up" deferral of the point. The only under-delivery is the docs row, flagged above.

**7. Plan revision recommendations**

None — the single Plan checkbox matches what shipped, and there is no Core concepts table to cross-check. If the atlas touch is intentionally dropped rather than done, add a `## Revisions` entry saying so (and close with `--no-atlas` acknowledged), since the estimate currently claims a docs row that has no corresponding change.

One process note: Bash was unusable this session (the sandbox denies the harness's own `~/.claude/session-env` directory — worth a look via `/sandbox`), so window verification was done against the provided diff plus the file tools; the "real smoke run" Done-when item is accepted on the Log's recorded evidence, not re-run.
