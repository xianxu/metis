# Boundary Review — 000025-get-data-root-cache-gap-dataset-keyed-on-path-not-content#25 (whole-issue close)

| field | value |
|-------|-------|
| issue | 25 — get-data root cache gap — dataset keyed on path string, not content |
| repo | 000025-get-data-root-cache-gap-dataset-keyed-on-path-not-content |
| issue file | workshop/issues/000025-get-data-root-cache-gap-dataset-keyed-on-path-not-content.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7076f4bba58a87cdde11a80ec0f95a5f5f067a06..HEAD |
| command | sdlc close --issue 25 |
| reviewer | claude |
| timestamp | 2026-07-17T17:11:25-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This boundary closes metis#25 as reframed: content identity for the only external-ingest root (`kaggle/download`) is declared in config as a `with.sha256` pin map (Nix fixed-output model), verified post-download in the kaggle repo, riding the existing `with → Kpre` channel so a pin edit structurally re-keys the whole downstream — zero metis cache-layer change. I independently verified every Log claim against the code: `pins.go` and its 5 tests plus the 3 `run()`-wiring tests exist and pin real behavior (8 new tests as claimed); the mutated-payload test exercises the real verify path via the injectable fake; `titanic-sweep.md` carries 3 real pins; the RUNBOOK cold-run/cohort note is present; all three cited metis re-key tests exist at the stated lines and say what the Log says they say; the atlas entry is accurate; and the plan's own ARCH-PURPOSE shadow-sweep checks out (4 shapes use `kaggle/download`: 1 pinned, 3 in the documented dual-use exclusion set from plan Revisions). Only minor findings — nothing blocks the close. (Note: Bash was unavailable in this session, so this is a static review — I could not re-run the suites; the Log's green-suite claims are corroborated by code inspection only.)

**Strengths**

1. `verifyPins` (kaggle `cmd/kaggle-download/pins.go:23`) is a genuinely clean core: all failures collected into one error (mismatch + missing + unpinned-extra), sorted deterministic output, and the "extra file = changed content" completeness rule is both implemented and tested — a declared identity that couldn't be partial.
2. The self-reference trap was anticipated and tested from both sides: top-level-only contract-file exclusion mirrors metis `collectArtifacts` (`cmd/metis/exec.go:217-234`), and `pins_test.go:42` pins the nasty edge (nested `sub/reads.json` is data, not excluded) exactly as `exec_test.go:203-229` does on the metis side.
3. ARCH-DRY at its best: no new identity mechanism at all — the re-key guarantee is a *citation* of existing tests (`cmd/metis/caching_test.go:123-132`, `:447-452`, `pkg/record/record_test.go:12-22`, all verified real and on-point) rather than a redundant re-proof or a parallel cache channel.
4. The plan-review Revisions (pin-scope, exclusion set, hash-source) were all actually honored in the code, and the hard scope call — leaving the three dual-use e2e shapes unpinned because one static pin block can't satisfy live + fixture data — is documented in three places (plan Revisions, atlas, RUNBOOK) instead of being silently skipped.
5. `run()` wiring (kaggle `main.go:78-90`) is a textbook thin shell: verify → loud fail, or paste-ready block on stderr; the escape hatch exists but can't be quiet.

**Critical findings** — none.

**Important findings** — none.

**Minor findings**

1. Atlas wording overstates slightly: "Unpinned ingest is LOUD … never silent" is true only when the step *executes* — a warm-cache HIT of an unpinned root skips execution and prints nothing, so the original silent-stale-hit mode persists for the unpinned dual-use shapes until they're pinned. The pinned decision shape is safe; suggest one clause in `atlas/experiment.md` ("loud at execution; a warm HIT of an unpinned root stays silent — the pin is the fix").
2. Ambiguous citation: issue Log line 121 says `record_test.go:12-22` — that's `pkg/record/record_test.go` (CanonicalHash map-order), but `cmd/metis/record_test.go` also exists and its lines 12-22 are an unrelated fixture. Worth the fuller path.
3. Cross-repo duplication of the contract-file trio (`with.json`/`metrics.json`/`reads.json`) between metis `exec.go:234` and kaggle `pins.go:33` — unavoidable across repos and both comments cross-reference, but if metis ever adds a fourth contract channel, kaggle's exclusion drifts silently (ARCH-DRY, accepted).
4. Bookkeeping: the plan doc's task checkboxes (`workshop/plans/000025-get-data-root-cache-gap-plan.md:44-67`) are all unticked while the issue's `## Plan` is fully ticked — harmless, but a fresh reader can't tell the plan doc is done.
5. Kaggle test nit: `TestRun_UnpinnedPrintsPasteReadyBlock` captures stderr via `os.Pipe` without a concurrent reader — fine at current output size, would deadlock past the 64KB pipe buffer if the pin block ever grew huge.

**Test coverage notes**

Coverage matches the failure modes this diff could ship: match-succeeds, mutation-fails-naming-the-file (Log claims red-proofing; the test does call the real path, so a neutered verify would indeed pass silently — the claim is plausible and the test structure supports it), unpinned-prints-block, all-failures-in-one-error, contract-file exclusion incl. the nested edge, plus pre-existing zip-slip and empty-slug guards. PURE entities (`verifyPins`, `pinBlock`) are tested against real temp dirs with no mocks — fs-as-input, consistent with the repo's ARCH-PURE reading. INTEGRATION (`run()`) goes through the injectable fake CLI. The one untested combination is pinned-run-with-extra-file through `run()` (covered at the `verifyPins` level only) — not worth adding.

**Architectural notes for upcoming work**

- ARCH-DRY: **pass** (see Strengths 2–3; minor 3 is the only residue).
- ARCH-PURE: **pass** — identity is data (declared in config), verification is a deterministic function of the dir, IO shell stays thin.
- ARCH-PURPOSE: **pass** — shadow-sweep run and clean; the deferred "local-file get-data root uses the same rule" is a genuine non-existent consumer (no such step type exists today), so it's a recorded rule, not an under-delivered purpose.
- Residual for a future issue: pins verify at *ingest* time only — a cached artifact mutated on disk after the fact is interior artifact-store integrity, out of this issue's scope and pre-existing; and the unpinned dual-use shapes (minor 1) would be fully closed by per-mode pins or a fixture-aware pin source if that ever becomes worth the complexity.

**Plan revision recommendations**

None required — the plan's Revisions section already matches what shipped. Optional tidy-ups only: tick the plan doc's checkboxes (minor 4) and disambiguate the `record_test.go` citation (minor 2).
