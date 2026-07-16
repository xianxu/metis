# Boundary Review — metis#39 (whole-issue close)

| field | value |
|-------|-------|
| issue | 39 — fingerprint visibility — print cohort at run + a fingerprints inspect command (when/what) |
| repo | metis |
| issue file | workshop/issues/000039-fingerprint-visibility-print-cohort-at-run-a-fingerprints-inspect-command-when-what.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cbbc10cc5b0234f24e040aa0447cd223ef74eb3..HEAD |
| command | sdlc close --issue 39 |
| reviewer | claude |
| timestamp | 2026-07-15T15:25:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I've completed the review. I could not execute the test suite — the session's Bash tool is broken at the harness level (`EPERM` creating `~/.claude/session-env/...` before any command runs, including `echo ok`), so all verification below is by close reading of the diff, the surrounding code, imports, struct fields, and call sites.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

This boundary delivers exactly what metis#39's Spec commits to, with clean architecture: a genuinely pure reducer/resolver/renderer core in `cmd/metis/fingerprints.go`, one IO seam (`loadLedgerRecords`) invoked only on the inspect command and error paths, and both `--fingerprint` flag sites (select, ledger show — I grepped; there are no others) wired through the one shared `pinFingerprint`. All three Done-when items are implemented and pinned by tests that assert real behavior. Nothing blocks the close; the findings are a plan-table misclassification (cheap fix, plan revision), one small test gap, and nits. The main caveat is that I couldn't run `go test`/`go vet` myself — the code reads as compile-correct (imports verified in every touched file, `record.RunRecord` fields `Started`/`Finished`/`Dirty` and `Hash` confirmed at `pkg/record/record.go:78-82`, `runOpts.out` wired at `run.go:102`), but the main agent should re-run the suite before closing since my verification is static only.

**1. Strengths**

- **ARCH-DRY, pass with distinction:** one renderer (`fingerprints.go:132` `renderCohorts`) serves three surfaces (inspect command, cohort-guard error, zero-match error), and one resolver (`resolveFingerprint`) ends the documented `--fingerprint`/`--point` matching-semantics split. `distinctFingerprints` really is deleted, and `ledger.Filter` (`pkg/ledger/ledger.go:385`) stays an exact-match storage primitive as the plan promised.
- **ARCH-PURE, pass:** record-file IO is provably confined to `showFingerprints`, `pinFingerprint`'s zero-match branch, and `cohortGuardErr` — the happy select path (`select_cmd.go:79-93`) does a cheap pure `distinctFingerprintCount` and never touches `record.json`. The reducer takes a plain map; its tests need no fakes.
- The `ExtraCommits` set-cardinality fold (`fingerprints.go:38-41` comment + the order-interleaved test) shows a plan-review finding actually carried into both code and a regression test, not just prose.
- Honest-error discipline: the zero-match "no scored configs" lie is dead, and `TestSelect_FingerprintPrefixAndHonestErrors` explicitly asserts the lie stays dead (`!strings.Contains(err, "no scored configs")`).
- `cmdLedgerFingerprints` rejecting unexpected args (`ledger_cmd.go:40`) rather than silently swallowing a typo'd flag — small, but the right instinct, and tested.

**2. Critical findings** — none.

**3. Important findings**

- **Plan Core-concepts table misclassifies `backfillCodeManifest` (and arguably `printFingerprintLine`) as "Pure entities"** (`workshop/plans/000039-fingerprint-visibility-plan.md:33`). `backfillCodeManifest` does direct `os.ReadFile`/`writeRecordJSON` and its tests ride a real git repo + filesystem — it is INTEGRATION by this repo's own definition. The review contract treats a table/code kind-contradiction as reportable; I'm calibrating it to Important rather than Critical because the *code* is correctly architected (it IS the intended IO seam, and "capture sites" already appear in the integration table at plan line 56) — only the table row is misfiled. Fix: move the row (see plan revisions below). No code change needed.
- **Missing test: `captureSweepCode`'s first-non-empty-fingerprint skip logic** (`capture.go:155-165`). The new `fp == "" && pfp != ""` fold is only exercised with a single point whose record exists (`capture_e2e_test.go:99-113`). The bug class this could ship — point 1's `record.json` missing → sweep prints no cohort line, or prints the wrong point's — is untested. A two-point sweep fixture with the first point's record absent would pin it cheaply.

**4. Minor findings**

- If a cohort's latest record has an empty `Commit` (degraded capture) but `Dirty: true`, `codeStr` renders `commit ?` and drops the dirty marker entirely (`fingerprints.go:118-127` headline fields only set inside `if commit != ""`; `codeStr` early-returns on empty commit) — the dirty bit is arguably the more important half.
- Two cohorts sharing an 8-char hex prefix would render identical `short()` values in both the ambiguity error and the inspect table, leaving the operator no way to type a disambiguating prefix from what's shown (~2⁻³² chance; note for a future `--full` flag).
- `levelStr` silently omits any `Level` value outside `{"", "inner", "outer"}` from the per-level breakdown while still counting it in `Rows` — fine while the enum is closed, invisible drift if it ever grows.
- `distinctFingerprintCount` and `resolveFingerprint` are near-identical distinct-fingerprint scans (ARCH-DRY nit — both tiny and pure; not worth consolidating unless a third appears).
- `TestLedgerFingerprints_CLI` swaps `os.Stdout` via a pipe and drains only after `run()` returns — deadlocks if output ever exceeds the pipe buffer (~64KB). Fine at fixture scale; a `t.Cleanup` restore would also protect against a panic mid-swap leaving stdout hijacked.

**5. Test coverage notes**

Coverage maps cleanly onto the Done-when table: reducer (grouping, legacy, ordering, set-cardinality), resolver (unique/ambiguous/zero/empty/exact), renderer, CLI through the real `run()` entrypoint with the bogus-flag rejection, select prefix/zero-match/ambiguous/guard-names-command, ledger-show shared resolution including the intentional exit-0→error behavior change, and both capture sites' output (sweep asserts exactly-once). The one real gap is the sweep missing-record skip above. Reminder: **I could not execute the suite** (harness Bash failure, not a code problem) — re-run `go test ./cmd/metis/ && go vet ./...` before `sdlc close`.

**6. Architectural notes for upcoming work**

- Pre-existing, preserved deliberately, worth an issue someday: with no `--fingerprint`, a ledger with **one** real cohort **plus legacy blank rows** passes the guard (`distinctFingerprintCount` ignores `""`) and the reduce blends legacy rows into the estimates — the exact blend class #32 guards against, just with the legacy cohort. Not this diff's scope (it faithfully preserved `distinctFingerprints`' semantics), but now that `(legacy)` is a visible first-class cohort in the inspect table, the asymmetry is more noticeable.
- The plan's own future-extension note (`--json` cohort output for agents) is well-positioned: the reducer already returns structured data.

**7. Plan revision recommendations**

Append to `workshop/plans/000039-fingerprint-visibility-plan.md`:

> **## Revisions**
> *2026-07-15 — boundary review (metis#39 close):* Core-concepts table corrections: (1) move `backfillCodeManifest` from "Pure entities" to "Integration points" (direct `record.json` read/write; tests ride real fs+git — it is the mint-site IO seam, not a pure entity); `printFingerprintLine` is a pure formatter over an injected writer and may stay, noted as such. (2) Add rows for entities the implementation introduced that the table omits: `pinFingerprint` (fingerprints.go, integration — record IO on error paths), `cohortGuardErr` (fingerprints.go, integration), `distinctFingerprintCount` (fingerprints.go, pure — replaces deleted `distinctFingerprints`), `showFingerprints`/`cmdLedgerFingerprints`/`cmdLedgerShow` (ledger_cmd.go, CLI shell).
