# Boundary Review — 000053-promote-fingerprint-consistency-guard#53 (whole-issue close)

| field | value |
|-------|-------|
| issue | 53 — promote fingerprint-consistency guard — refuse when the working tree is not the cohort's code |
| repo | 000053-promote-fingerprint-consistency-guard |
| issue file | workshop/issues/000053-promote-fingerprint-consistency-guard.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9ab02c858ad831268c2b922b96c2bec36fc60772..HEAD |
| command | sdlc close --issue 53 |
| reviewer | claude |
| timestamp | 2026-07-17T23:18:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

All the pieces check out structurally — types line up, both promote seams are wired, and the tests exercise the real `runSelect` path. I have what I need for the verdict. (Note: I could not execute the test suite — the Bash tool is broken in this session at the harness level, failing before any command runs — so this review is by close reading, which the read-only review contract expects anyway.)

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The boundary delivers what #53 promised: a pure drift-comparator (`promoteDrift`) over an injected hasher, wired pre-exec into both promote seams (`--best*` and `--point`), with a diff-shaped refusal, a loud `--no-fingerprint-check` override, warn-and-proceed on absent provenance, and the atlas paragraph beside the #32 guard. Every Plan checkbox and Done-when item is traceable to code and tests, and the design refinement (per-path blob compare reusing `gitBlobHashes` instead of a parallel re-mint) is genuinely the right ARCH-DRY call. One Important finding keeps this from a clean SHIP: the production hasher's error semantics diverge from the test fake's, so the deletion/error case renders a misleading refusal message — it fails *safe* (still refuses), which is why this is non-blocking. Confidence is medium only because I could not execute the suite in this session (harness shell failure), not because of doubts about the reading.

### 1. Strengths

- **Pure core, thin seam, honest injection point** — `promoteDrift` (`cmd/metis/promoteguard.go:32`) is pure over the hasher signature and unit-tested with a fake and no IO; `guardPromoteFingerprint` (`promoteguard.go:111`) is the one thin IO caller. Textbook ARCH-PURE.
- **The per-path-compare-over-re-mint refinement held** — reusing `gitBlobHashes` (`cmd/metis/trace.go:62`) means normalization is identical to capture by construction, eliminating the equivalence risk a parallel mint would carry, and it makes the drift *list* fall out of the comparison for free. ARCH-DRY as designed.
- **`cohortFingerprintOf`'s "any row" shortcut is actually sound** — I verified the invariant it leans on: `runSelect` refuses multi-cohort ledgers without a pin (`select_cmd.go:98-102`) and `pinFingerprint` filters to one cohort, so post-pin any row's fingerprint is the cohort's. The comment at `select_cmd.go:348` states this; the code upstream enforces it.
- **The e2e test drives the real thing** — `TestPromoteGuard_RefusesDriftAndRoundTrips` goes through real `runSelect` with real `gitBlobHashes` and a planted `record.json`, covering clean/refuse/override/round-trip in one fixture, plus a separate test proving the `--point` path is guarded. This is exactly the shipped-bug class this diff could carry.
- **Fail-open only where the spec says to** — absent provenance warns-and-proceeds; everything else refuses. The direction of every degradation is correct.

### 2. Critical findings

None.

### 3. Important findings

- **`cmd/metis/promoteguard.go:67-73` — a single missing/unreadable closure file makes the refusal message lie about every other path in that repo, and swallows the underlying error.** `gitBlobHashes` batches one `git hash-object -- <paths>` call; if *any* path is missing, the whole batch errors (`trace.go:66-68`), and `promoteDrift` discards `err`, leaving `now = ""` for **all** paths in that repo — so the refusal renders unchanged files as `captured <h> → working <missing>`. The spec's core deliverable is "which paths changed"; in the realistic delete-one-file drift case the message names every path, not the changed one. A genuine environmental failure (git absent, repo root moved) likewise surfaces as universal `<missing>` drift with the real error silently dropped — the silent-error-swallowing class this checklist flags. Failure scenario: cohort closure = `{train.py, util.py}`; operator deletes `util.py` only; refusal reports both files missing. Fix sketch: on batch error, retry per-path (or stat-first and batch only the existing paths, marking absent ones missing explicitly), and include the hasher error text in the refusal when it isn't a plain missing-file case.
- **`cmd/metis/promoteguard_test.go:16-24` — the fake hasher's missing-path semantics don't match production, so the "missing detected" unit test pins behavior the real hasher can't exhibit.** `fakeHasher` omits unknown paths from the map and returns nil error; real `gitBlobHashes` errors on the whole batch. `TestPromoteDrift_EditAndMissingDetected` therefore asserts a per-path `New: ""` outcome that only exists under the fake, and no e2e test deletes a file. Add an e2e (or real-hasher) case that deletes a closure file — it will demonstrate the finding above and pin whatever fix lands. (This is the ARCH-PURE caveat: the pure core is fine, but the injected seam's fake must be semantics-faithful or the unit tests certify fiction.)

### 4. Minor findings

- `promoteguard.go:40-55` — `captureCommit` comes from the first fingerprint-matching record in Go map iteration order; cohort records can carry different capture commits (same blobs, different HEAD), so the restore hint is nondeterministic across invocations. Hint-only; sort or pick deterministically if it ever matters.
- `select_cmd.go:349` and `select_cmd.go:534` — the two guard invocations are byte-identical including the `func(m string){...}` closure; a tiny `guardPromote(o, led)` helper would collapse them (ARCH-DRY, borderline at two sites).
- `fingerprints.go:242-243` — `loadLedgerRecords`' doc comment still says it's "never [called] on the happy select path"; the guard now calls it on every `--promote`. Stale comment.
- `promoteguard_test.go:184` — `var _ = fmt.Sprintf // keep fmt if assertions change`: `fmt` is otherwise unused in the file; drop the import instead of pinning it.

### 5. Test coverage notes

- Covered well: clean tree (no false positive), content drift refusal with path + commit hint + override token in the message, loud override, restore round-trip, `--point` parity, legacy warn-and-proceed, wrong-cohort records excluded.
- Gaps: (a) file **deletion** through the real hasher (see Important #2); (b) hasher **error** path (git failure) — currently indistinguishable from all-missing; (c) a multi-path closure where only one path drifts, asserting the refusal names *only* that path — this is the assertion that would have caught Important #1; (d) the "round-trip" restores by rewriting file content rather than an actual `git checkout` of the hint's commit — acceptable given the fixture's commit is synthetic, but worth noting it tests the hint's *effect*, not the literal command.
- The `checked=false` warn conflates true legacy cohorts with cleaned `runs/` dirs (records deleted). Spec's wording ("missing records/D") permits this; no test distinguishes them, which matches the spec.

### 6. Architectural notes

- **ARCH-DRY: pass** (minor consolidation noted above). The big DRY decision — hasher reuse over re-mint — is the diff's best call.
- **ARCH-PURE: pass**, with the seam-fidelity caveat in Important #2: keep injected fakes contract-faithful to the production implementation they stand in for, or the pure core's tests drift from reality.
- **ARCH-PURPOSE: pass.** Shadow-sweep of promote consumers: `promoteSelected` and `runPointSelect` are the only two execution seams under `select --promote` (the retired standalone `metis promote` is gone; `kaggle submit --run` submits an existing run without executing code) — both derive from the one guard. Detection-only scope matches the issue's stated split (restore is #28, a declared separable issue, not this issue's deferred point).
- For #28 (restore): the guard's `driftedPath` list + `captureCommit` is exactly the input a restore verb needs; consider exporting that shape rather than re-deriving it there.
- Edge worth remembering: a mixed ledger (legacy unfingerprinted rows + one cohort) passes the multi-cohort count, and a `--point` at a legacy row is then guarded against the fingerprinted cohort's closure — over-guarding in the safe direction, but the refusal text would attribute the wrong provenance to the promoted config.

### 7. Plan revision recommendations

None — the plan's checklist matches what the code delivers. If Important #1's fix lands, add a `## Log` note (not a revision) that the deletion case routes through per-path fallback hashing, since the "design refinement" paragraph currently implies the batch hasher's semantics are sufficient.

---

## Re-review — 2026-07-17T23:24:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 53 — promote fingerprint-consistency guard — refuse when the working tree is not the cohort's code |
| repo | 000053-promote-fingerprint-consistency-guard |
| issue file | workshop/issues/000053-promote-fingerprint-consistency-guard.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9ab02c858ad831268c2b922b96c2bec36fc60772..HEAD |
| command | sdlc close --issue 53 |
| reviewer | claude |
| timestamp | 2026-07-17T23:24:06-07:00 |
| verdict | SHIP |

## Review

All verification I can do without a shell is complete. The record types, harness helpers, guard wiring, and every folded fix from the prior FIX-THEN-SHIP review check out in the code. Writing up the review now.

```verdict
verdict: SHIP
confidence: medium
```

This boundary delivers exactly what #53's Spec promises, and — the specific thing this second-pass review had to check — every finding from the prior FIX-THEN-SHIP review is verifiably folded into the code, not just claimed in the Log. The batch-poisoning Important is fixed with a per-path retry that keeps the hasher error (`promoteguard.go:77-89`); the fake hasher now mirrors `gitBlobHashes`' real fail-the-whole-batch semantics (`promoteguard_test.go:14-29`); the new deletion e2e asserts the unchanged sibling does NOT read as drifted (`TestPromoteGuard_DeletedFileDoesNotPoisonSiblings`); and all four prior minors landed (shared `guardPromote` helper, sorted-addr deterministic restore hint, refreshed `loadLedgerRecords` comment, no `fmt` pin). Confidence is medium for one reason only: the Bash tool is broken at the harness level in this session (EPERM creating its session-env dir, before any command runs — subagents hit the same wall), so I could not execute the suite; the review is by close reading. Nothing I read blocks a SHIP.

### 1. Strengths

- **The per-path retry fix is the right shape** — `promoteguard.go:77-89` retries individually only when the batch fails, so the common clean/edited cases stay one git call per repo, unchanged siblings still verify with their real hashes, and the failing path carries the first line of the actual hasher error into the refusal as `<unhashable: …>` (`promoteguard.go:117-118`). No silent swallowing anywhere on the path.
- **Fake-hasher fidelity restored** — `fakeHasher` (`promoteguard_test.go:16-29`) now fails the whole call on any unknown path, matching `gitBlobHashes` (`trace.go:66-68`), and `TestPromoteDrift_EditAndMissingDetected` pins precisely the recovery contract: edited sibling gets its real new hash, missing path carries the error. The unit tests no longer certify fiction.
- **Both promote seams derive from one guard** — `guardPromote(o, led)` at `select_cmd.go:349` and `select_cmd.go:534` is a single shared call; the prior byte-identical duplication is gone.
- **Deterministic restore hint** — sorted record pick (`promoteguard.go:46`) replaces map-iteration-order nondeterminism.
- **Fail-open only where the spec says** — absent/legacy provenance warns loudly and proceeds (`promoteguard.go:139-142`); no cohort identity (`cohortFP == ""`) is a no-op; everything else refuses. Every degradation points the safe direction.

### 2. Critical findings

None.

### 3. Important findings

None. (I specifically re-checked the two prior Importants against the head: both fixed, both now covered by tests that would catch a regression.)

### 4. Minor findings

- `promoteguard.go:53-54` — `captureCommit` can be taken from an earlier same-cohort record that carries steps but no D refs, while the closure comes from a later record; commits within a cohort can differ (same blobs, different HEAD). Deterministic now, but still hint-only imprecision; not worth code today.
- `select_cmd.go:39` — the usage string in the `hoistShapePath` error doesn't mention `--no-fingerprint-check` (or `--cohort`); pre-existing abbreviation pattern, flag help covers it.
- `promoteguard.go:119-120` — the `<missing>` default branch is unreachable with the real hasher (a missing file always errors per-path, so `Err` is set → `<unhashable>`); harmless defensive rendering.

### 5. Test coverage notes

- Covered: clean tree (no false positive), content-drift refusal asserting path + capture-commit hint + override token + headline, loud override, restore round-trip, `--point` parity, legacy warn-and-proceed, wrong-cohort exclusion, per-path retry semantics at the unit level, and — new this window — real-hasher deletion without sibling poisoning (the exact assertion the prior review said would have caught the batch bug).
- Remaining small gaps (fine to leave): the refusal's `captured <old> → working <new>` blob rendering and the `<unhashable: …>` branch aren't string-asserted anywhere (only path names and headline are); the round-trip restores by rewriting content rather than a literal `git checkout` (acceptable — the fixture commit `cafe1234` is synthetic; the test pins the hint's effect).
- I could not run the suite (harness shell failure); the issue Log claims full `-race` green at close, and nothing in the code contradicts compilability or the claimed behavior, but the main agent should re-run `go test ./...` in a working session before recording the close verdict.

### 6. Architectural notes

- **ARCH-DRY: pass.** The design's central DRY call — reuse `gitBlobHashes` instead of re-minting the fingerprint — held through the fix; the guard also reuses `loadLedgerRecords`, `short`, and the prior review's duplication minor is consolidated into `guardPromote`. The duplicated empty-`ship:` check in the two promote paths predates this diff.
- **ARCH-PURE: pass.** `promoteDrift` is pure over the injected hasher and unit-tested with zero IO; `guardPromoteFingerprint` is the one thin IO caller. The seam-fidelity caveat from the prior review is resolved — the fake now honors the production contract.
- **ARCH-PURPOSE: pass.** Shadow-sweep of promote consumers: `promoteSelected` (`select_cmd.go:349`) and `runPointSelect` (`select_cmd.go:534`) are the only two seams that execute a promoted run, and both derive from the single guard; `kaggle submit --run` ships an existing run without executing code. Detection-only scope matches the issue's declared split (restore is #28, a genuinely separable issue, not a deferred core).
- Carry-forward for #28: `driftedPath` + `captureCommit` is exactly a restore verb's input; and the mixed-ledger edge (legacy unfingerprinted rows alongside one cohort → a `--point` at a legacy row is guarded against the cohort's closure, over-guarding in the safe direction) remains worth a conscious decision when restore lands.

### 7. Plan revision recommendations

None — the plan checklist, Log (including the close-review fold entry), and code agree. The only follow-through this review asks of the main agent is operational, not documentary: re-run the full suite in a session with a working shell, since neither this review nor the prior one could execute it.
