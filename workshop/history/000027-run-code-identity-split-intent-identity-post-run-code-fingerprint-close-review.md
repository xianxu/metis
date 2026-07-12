# Boundary Review — metis#27 (whole-issue close)

| field | value |
|-------|-------|
| issue | 27 — run/code identity split: intent-identity + post-run code fingerprint |
| repo | metis |
| issue file | workshop/issues/000027-run-code-identity-split-intent-identity-post-run-code-fingerprint.md |
| boundary | whole-issue close |
| milestone | — |
| window | cc1aabbc944a3b4a01cf413915b9cff150659ab7..HEAD |
| command | sdlc close --issue 27 |
| reviewer | claude |
| timestamp | 2026-07-11T21:27:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have a complete picture. Build, vet, all Go tests (9 packages), and Python tests (46) pass; I traced the identity split end-to-end, verified the sweep ordering (capture → ledger), the record↔dir address match, and the shadow-sweep of every consumer. Let me write up the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The whole-issue close for metis#27 is functionally solid and cleanly architected. The diff delivers the identity split exactly as the Spec commits: `PointAddress` recomposes to `(resolved_with, shape_blob_hash, seed)` — repo HEAD dropped — a new pure `CodeFingerprint` over the run-end `D` closure is set post-capture in `backfillCodeManifest` (the one temporally-correct site), the ledger dedups on `(point_addr, code_fingerprint)`, single-run dirs are content-addressed symmetric with sweep points, and `repo_shas`/`sweep_sha` are gone from all code (`--sweep`→`--fingerprint`). I independently verified `go build`/`go vet`/`go test ./...` and `uv run pytest` all green, the CUE drift-guard conforms, and both the acceptance test (two rows on a code change) and the content-addressed-dir test pass and pin real behavior. Nothing blocks SHIP. What holds it back from a clean SHIP is non-blocking: the atlas feature-sketch (`experiment.md`) still documents the **old** `PointAddress(resolvedWith, repoSHAs, seed)` signature and omits `CodeFingerprint`, and the load-bearing sweep ordering (capture-before-ledger) has no end-to-end test.

### 1. Strengths (confirmed-good ground)

- **ARCH-PURE, exemplary.** `record.PointAddress` and `record.CodeFingerprint` (`pkg/record/address.go:36,59`) are pure (`json`/`sort` only), unit-tested with hand-built `[]CodeRef` and zero IO; the git-touching seam (`shapeBlobHash`, `capture.go:275`) is thin and lives in the `cmd/metis` shell. Correct pure-core/thin-shell split.
- **Fingerprint placement is the right one.** `backfillCodeManifest` (`capture.go:344-353`) is the single post-capture site where `D` exists and the record is re-written — the exact temporal-availability lesson `workshop/lessons.md` records from this issue. The `CodeFingerprint` error is surfaced, not swallowed (`capture.go:349-352`).
- **The `Repo`-exclusion portability property is genuinely tested.** `TestCodeFingerprint` (`pkg/record/record_test.go`) pins order-independence, blob-sensitivity, the machine-portable "moved" case, and empty-closure definedness — the fingerprint won't drift across checkouts.
- **Sweep ordering is correct.** `captureSweepCode` (sets the fingerprint) runs at `sweep.go:150` *before* `writeSweepLedger` (`sweep.go:156`), so `rowsFromManifest` reads a populated `record.json`. The dir↔record address can't desync — both `singleRunID`/`pointAddressOf` (`run.go:114`) and `assembleRecord`→`buildRecord` (`run.go:177`) mint from the same `shapeBlobHash(o.expPath)`, and `TestSingleRun_ContentAddressedDir` asserts `rec.PointAddress == dirName`.
- **Composite dedup is minimal and idempotent.** `dedupKey = PointAddr+"\x00"+CodeFingerprint` (`ledger.go:52`); `AggregateView` correctly re-keys its group on `code_fingerprint` (`ledger.go:215`) rather than dropping the code term (which would have merged the exact collision this issue prevents). `TestAppend_DedupByPointAddrAndFingerprint` pins same-addr/different-fingerprint → two rows.

### 2. Critical findings

None. No correctness bug, crash, contract drift, or silent error swallowing found.

### 3. Important findings

- **atlas docs-gate: `atlas/experiment.md:52-54` documents a signature that no longer exists.** The `pkg/record` sketch still reads `PointAddress(resolvedWith, repoSHAs, seed) … the coarse config+repo+seed content-address`, and lists no `CodeFingerprint`. `atlas/index.md` was correctly updated, so the index-level gate is met, but this feature-sketch now *actively misstates* the base-layer `pkg/record` API (which propagates to dependent repos). Fix: update line 52-54 to `PointAddress(resolvedWith, shapeBlobHash, seed)` (intent identity) and add a `CodeFingerprint([]CodeRef)` bullet (post-run code identity). Non-blocking, doc-only.

- **Test coverage: no end-to-end sweep test asserts the fingerprint reaches the persisted ledger.** The acceptance test (`identity_e2e_test.go`) hand-builds the manifest and calls `captureSweepCode`→`rowsFromManifest`→`Append` directly; the real orchestration (`runShapeSweep`, `sweep.go:150-156`) is never asserted to produce a non-empty `code_fingerprint` column. The capture-before-`writeSweepLedger` ordering is load-bearing — a future reorder would silently yield empty-fingerprint rows (re-introducing the same-config-different-code collision) and **every current test would still pass**. Cheap fix: assert a non-empty `code_fingerprint` in the CSV of an existing `runShapeSweep` e2e (e.g. `TestShapeSweep_NestedLoopWinnerAndLedger`, run in a temp git repo). Current code is correct; this only guards it.

### 4. Minor findings

- `pkg/ledger/ledger.go:1-12` package header is stale: still describes a Row as "(free-param tuple + code SHA + seed)" and "the sweep-SHA (the code-version, git short-SHA human address)". The `Row` struct doc (`:26`) was updated; the package header wasn't. Update to code-fingerprint.
- `shapeBlobHash` (`capture.go:275-296`) duplicates the abs→EvalSymlinks→`--show-toplevel`→`Rel` prologue from `addSpecToClosure` (`capture.go:232-253`) — new duplication (ARCH-DRY). A shared `specRepoRel(specPath) (root, rel, err)` helper would unify both.
- The promote note (`ledger_cmd.go`, `runPromote`) now unconditionally references `metis reproduce (metis#28)` — a command that doesn't exist yet. Harmless, but a not-yet-shipped command in operator-facing output can confuse; consider softening until #28 lands.
- Stale test comments referencing "sweep-SHA": `pkg/ledger/ledger_test.go:229`, `cmd/metis/ledger_test.go:13` (and `TestAggregateView_ReducesPerConfig`'s "grouping by (free-params, sweep-SHA)" comment). Cosmetic.

### 5. Test coverage notes

Strong at the unit level and pins real logic, not mock reasserts: `CodeFingerprint` (order/blob/portability/empty), `PointAddress` sensitivity flipped to `shape-blob`, `Append` composite dedup (the load-bearing collision case), CUE conformance, and the two-rows-on-code-change acceptance through real git blobs + real capture. The single gap is the orchestration-level assertion in §3 (fingerprint in the persisted CSV via `runShapeSweep`). The no-git degraded path (fingerprint = hash of the empty manifest, identical across runs) is exercised only indirectly — acceptable, matching the prior empty-`sweep_sha` behavior.

### 6. Architectural notes for upcoming work

- **metis#28 handoff is clean.** `CodeFingerprint`'s input is the run-end closure; #28 swaps it for a per-step step-time closure + a within-run consistency check without changing the hash function — the seam is where the code comments promise it. The `reproduce`/`verify` verbs the promote note forward-references are #28.
- **metis#26 (shape-blob display keying) can build directly on `shapeBlobHash`** — the helper is now the single pre-run intent term feeding both `PointAddress` and `shapeRunIdentity` (`sweep.go:113,403`).
- Note that `shapeBlobHash` shells real git directly (un-injected), bypassing the `gitProbe` seam that `probeRepo` uses. Consistent with the existing `captureRunCode` capture path (also un-injected), so not a regression — but if a future test needs to drive the intent address with a fake git, this is the seam to reconcile.

### 7. Plan revision recommendations

None. The plan's Core-concepts table matches the shipped code exactly — every entity exists at its stated path with the stated status (`PointAddress` modified, `CodeFingerprint` new, `RunRecord`/`Row`/`Append`/`#RunRecord` modified, `shapeBlobHash` new, single-run minting + `backfillCodeManifest` modified). The plan's `## Revisions` is `(none yet)`, and no drift between plan and code was found, so no entry is required.
