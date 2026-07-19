# Boundary Review — metis#26 (whole-issue close)

| field | value |
|-------|-------|
| issue | 26 — sweep key = shape blob-hash, not repo HEAD sweep_sha |
| repo | metis |
| issue file | workshop/issues/000026-sweep-key-shape-blob-hash-not-repo-head-sweep-sha.md |
| boundary | whole-issue close |
| milestone | — |
| window | be5b652d2993ab425c0529a13df813eca119d9ae^..HEAD |
| command | sdlc close --issue 26 |
| reviewer | claude |
| timestamp | 2026-07-19T16:11:43-07:00 |
| verdict | unknown |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Failed to authenticate. API Error: 401 Invalid authentication credentials

---

## Re-review — 2026-07-19T16:19:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 26 — sweep key = shape blob-hash, not repo HEAD sweep_sha |
| repo | metis |
| issue file | workshop/issues/000026-sweep-key-shape-blob-hash-not-repo-head-sweep-sha.md |
| boundary | whole-issue close |
| milestone | — |
| window | be5b652d2993ab425c0529a13df813eca119d9ae^..HEAD |
| command | sdlc close --issue 26 |
| reviewer | claude |
| timestamp | 2026-07-19T16:19:42-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Verification complete. `captureClosure` (line 34) does `git hash-object -w` on every closure file — including the spec `.md` (added via `addSpecToClosure` at line 93) — so a **dirty** shape's blob reaches the object DB and is GC-pinned by the side-ref. The `git cat-file blob <hash>` recovery chain the close Log claims is sound for both clean and dirty shapes. The subsumption is genuine.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** metis#26 (re-key the sweep identity off repo-HEAD `sweep_sha` onto the shape's git blob-hash) is being closed as subsumed by #27, with only two real changes in-window for #26 itself: the issue-file close log and a `lessons.md` vocabulary reconciliation. Both are correct, and — the load-bearing check for a subsumption close — I independently verified the "already delivered by #27" claim against the current tree rather than the Log: the re-key landed, is correct, and no identity term keys off repo-HEAD anymore. Nothing blocks the boundary. (The review window `be5b652^..HEAD` is enormous because the base anchored to a very old commit, sweeping in all of #27 plus ~30 other issues that each carried their own close-boundary review — the archived #19/#27 review sidecars in the diff confirm this. I scoped my review to #26's actual delta plus the subsumption verification, not a re-review of those closed boundaries.)

### 1. Strengths (confirmed-good ground)
- **The subsumption is real, not asserted.** A full shadow-sweep of every consumer of the old sweep identity confirms each now derives from shape-blob-hash / code-fingerprint: `PointAddress(resolvedWith, shapeBlobHash, seed)` (`pkg/record/address.go:36`), `shapeRunIdentity(sh, sbh)` (`cmd/metis/sweep.go`), `rowsFromManifest` → `rec.CodeFingerprint` (`cmd/metis/ledger.go`), ledger `Filter`/`AggregateView`/dedup keyed on `CodeFingerprint` (`pkg/ledger/ledger.go`), and `RunRecord` with `CodeFingerprint` and no `RepoSHAs` (`pkg/record/record.go`). A `grep` for `SweepSHA|sweep_sha|repoSHAs|RepoSHAs` across all `.go` files returns **only #27-documenting comments** — zero lingering repo-HEAD identity usage (the sole remaining repo-HEAD use is `Code.Commit`, which is durable-side-ref provenance, not identity).
- **Dirty-shape recovery (Done-when 1) actually holds.** `captureClosure` (`capture.go:34`) `hash-object -w`s every closure file including the spec, and pins them via `refs/metis/sweeps/<id>` — so `record.json`'s `Steps[].Code.D` → `git cat-file blob <hash>` recovers the exact bytes even for an uncommitted shape. I traced this rather than take the Log's word.
- **Run-distinction preserved (Done-when 2).** Dropping `sweep_sha` doesn't collapse cohorts: `dedupKey = PointAddr + "\x00" + CodeFingerprint` keeps same-config/different-code runs as two rows, pinned by `TestAppend_DedupByPointAddrAndFingerprint` and the `TestCodeIdentity_TwoRowsOnCodeChange` e2e (both in-window).
- **The Log's traceability references are accurate** — `capture.go:309` (shapeBlobHash), `capture.go:93` (addSpecToClosure), `address.go:36` (PointAddress) all check out. The `lessons.md` reconciliation correctly rewrites the `sweep_sha`/`--sweep` vocabulary to `code_fingerprint`/`--fingerprint` while preserving the pre-#27 note.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- Review-window scope is a context artifact, not a defect: the base anchored far back, so the diff is dominated by already-boundary-reviewed issues (#27, #23, #30–#67). Flagging so the operator knows the SHIP covers #26's delta + the subsumption verification, not a fresh audit of that merged work.
- The issue frontmatter still reads `status: working` (line 3) and `estimate_hours:` is empty (line 8) — both expected mid-close (the close gate finalizes status and measures active-time actuals); noting only so the finalization step isn't skipped.

### 5. Test coverage notes
No new tests are warranted — #26 ships no code. The re-key is covered by #27's in-window suite: `TestPointAddress_Sensitivity` (address changes with the shape-blob term, not repo SHA), `TestCodeFingerprint` (order-independent, blob-sensitive, checkout-portable), `TestShapeSweep_NestedLoopWinnerAndLedger` (asserts every persisted ledger row carries a non-empty `code_fingerprint`, pinning the capture-before-ledger ordering), and the two dedup tests above. Coverage of the re-key's real logic is strong and pins behavior, not mocks.

### 6. Architectural notes for upcoming work
- ARCH-DRY — **pass.** Sweep identity is single-sourced: shape-blob-hash into `PointAddress`, per-file D-closure into `CodeFingerprint`; the dedup key lives in one place (`ledger.go`). No parallel identity computation.
- ARCH-PURE — **pass.** `PointAddress`/`CodeFingerprint` are pure (canonical-hash over sorted structs, no IO); the git probing (`shapeBlobHash`, `gitBlobHashes`, `captureClosure`) is the thin injected IO shell.
- ARCH-PURPOSE — **pass.** The shadow-sweep above is the purpose check: every consumer derives from the content-address, no hand-maintained repo-HEAD restatement survives. The one unbuilt item — a first-class "all variations of this shape = distinct blob-hashes for that path" ledger view — is a separable legibility extension, not #26's purpose (which is the re-key), explicitly deferred to #28 (`metis reproduce`) with operator sign-off. Legitimate deferral, not under-delivery.

### 7. Plan revision recommendations
None. The plan's single `[x]` step and its `RESOLVED: subsumed by #27` note match the code; the Log's two-item Done-when trace matches the current tree.
