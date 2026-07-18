# Boundary Review — metis#24 (whole-issue close)

| field | value |
|-------|-------|
| issue | 24 — cache key addressing — input-addressed vs output-hash-chained (interior identity) |
| repo | metis |
| issue file | workshop/issues/000024-cache-key-addressing-input-addressed-vs-output-hash-chained.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2f6b9b0df1132658c4bfc07867fc1a2dc913bd9e^..HEAD |
| command | sdlc close --issue 24 |
| reviewer | claude |
| timestamp | 2026-07-17T17:35:27-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything needed. The #24 close is essentially a decision-and-documentation boundary over machinery that shipped (and was gate-tested) in #18 M1a-3b; I've verified the mechanism, the tests' teeth, and the documentation deliverables directly against the tree.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** metis#24's purpose — decide input-addressed vs output-hash-chained, and make the chosen interior identity *sound* — is fully delivered and provable from the tree. The executor keys on upstream **K_pres** (`cmd/metis/caching.go:142-149`), upstream-code invalidation is restored by the transitive-D snapshot stored in each step's **own** entry (`recordMiss` → `MergeTransitiveD` → `isHit` re-hashes `entry.TransitiveD`), the legacy-entry migration guard (`nil TransitiveD → MISS`) rests on a directly-pinned codec invariant, and the decision is recorded **with the trade-off rationale** in `atlas/index.md:226-232` exactly as the Done-when demands. The one un-delivered Done-when item — the pre-run cache-hit-map printout — is not a silent under-delivery: the issue's own Spec frames this as "a decision issue (not a feature)", and the `## Revisions` section records the deferral with a demand-driven re-entry trigger, mirrored in the atlas ("deliberately UNBUILT — demand-driven, next competition"). Nothing blocks. (Process note: Bash is broken in this review environment — harness-level EPERM on its session-env dir — so I could not re-execute the suite; all verification below is by direct code/test reading. The findings and verdict don't depend on a test run: this close's delta over the already-boundary-reviewed #18 M1a-3b machinery is documentation.)

### 1. Strengths

- **The soundness tests test the actual failure mode, not a proxy.** `cmd/metis/caching_soundness_test.go` drives the REAL topo executor (Runner + cachingExecutor + git blob-hashing) because the heal-before-check ordering — the exact reason the "walk upstream live entries" design was rejected — is structurally invisible to a pure `Validate` unit test. `TestCachingExecutor_HitFeedsDownstreamClosure` is especially good: it targets the HIT-repopulation seam (`caching.go:115`) with a two-edit sequence designed so that reverting that one line fails *this* test while the all-MISS gate stays green — revert-detection built into the test design.
- **The migration guard is pinned at the codec layer, not just via e2e.** `TestEntry_TransitiveDCodec_EmptyIsNonNil_LegacyIsNil` (pkg/cache/transitived_test.go:58) directly asserts the `[] ≠ nil` round-trip that the whole nil-means-legacy guard rests on, plus the explicit-`null` and absent-key legacy blobs — so a re-added `omitempty` fails loudly and locally. The matching lessons.md entry was recorded in the same window.
- **The two-hash divergence is documented at every site it could confuse.** `sortedUpstream` (caching.go:18-25), `buildRecord` (record.go), and the atlas all state that the executor's key is input-addressed while record provenance stays output-addressed — one shared collection primitive, two maps, deliberately divergent meaning (ARCH-DRY done right rather than a copy-paste fork).
- **End-to-end reach:** `TestShapeSweep_HonestE2E` (shipe2e_test.go) proves the #24 gate *through the nested sweep* (upstream code edit re-runs downstream folds while config/fold-invariant data + partition stay cached), and `TestCachingExecutor_KpreUsesUpstreamKpres` pins both input-identity propagation and `_fold` re-keying at the executor level.
- **The close itself is honest bookkeeping:** the Plan row's un-delivered sub-item is struck through *and* explained in a dated `## Revisions` entry, with the atlas cross-referencing it — the deferral is discoverable from both directions.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

- `cmd/metis/caching.go:26-35` — `sortedUpstream` silently skips a `need` absent from the map. Correct-by-design for the buildRecord caller (partial failed runs), and unreachable in the executor today (topo order + unconditional `c.kpres` set), but a future execution-reorder would degrade into silently wrong-but-stable keys instead of a loud error. A `missing need → error` wrapper on the executor path would make the invariant explicit.
- `cmd/metis/caching.go:173/115` — an immutable-leaf HIT bypasses the nil-guard, so a surviving *legacy* leaf entry repopulates `c.transitiveD[leaf] = nil` and downstream closures silently omit the leaf's D (under-invalidation), whereas a post-#24 leaf entry's D *is* folded downstream (over-invalidation on a leaf-code edit). Both directions are covered by the leaf policy's declared bet / the safe-MISS direction, but the asymmetry is undocumented; no test covers leaf-closure-feeds-downstream.
- Spec residual: the Spec's constraint that "the reducer's key must incorporate all folds' **manifested** row-content hashes" shipped as a deterministic `partitionRef` id (`cv-k%d-strat%t-seed%d`, sweep.go) in `ToldSet`, with content-hash threading deferred in a code comment. Soundness holds transitively (fold content is a function of input identities + #25's pins), but the Spec claims slightly more than the told-set key literally carries — worth one Revisions line (see §7).

### 5. Test coverage notes

The kind of bug this diff could ship — a stale HIT after an upstream code edit, a spurious re-key on output non-determinism, a vacuous HIT from a legacy entry, an empty-closure step MISSing forever, a torn index read under parallel writers — each has a dedicated test with a documented revert-detection rationale (`caching_soundness_test.go`, `caching_test.go`, `transitived_test.go`, `TestWriteEntry_ReaderNeverSeesTornIndex`). PURE entities (`Kpre`, `MergeTransitiveD`, the codec) are tested without IO; the INTEGRATION layer is tested through the real executor with an injected fake step-exec that writes genuine `reads.json` — the right split. Gaps are the two minors above (leaf-closure downstream feed; executor missing-need hardening).

### 6. Architectural notes

- **ARCH-DRY: pass.** One collection primitive (`sortedUpstream`) for both key and provenance; one fold primitive (`MergeTransitiveD`); one per-repo hasher shape shared by store and validate so they cannot disagree. No duplicated logic introduced.
- **ARCH-PURE: pass.** The pure core (Kpre / Validate / MergeTransitiveD / codec) lives in `pkg/cache` with IO-free tests; all accumulation, git, and FS IO stays in the `cmd/metis` shell. The integration tests use real IO *deliberately* — because the property under test lives in run-time ordering — which is the correct exception, not a purity violation.
- **ARCH-PURPOSE: pass, with the shadow-sweep run.** Consumers of the cache identity: the executor derives from `Kpre` ✓; record provenance deliberately does *not* derive and says so at both sites ✓; atlas restates the model as a map (acceptable — it is documentation, not a deferred consumer). The deferred hit-map is the conditional elaboration of a decision issue, recorded with rationale and a re-entry trigger — it is not "the point" of the issue (the point was the decision + soundness, both shipped and proven), so this is a legitimate deferral, not the follow-up-is-the-purpose anti-pattern.
- Forward note: #25's declared content pins are now load-bearing for data-change propagation (input-addressing dropped the only other carrier); the issue Revisions correctly records that #25 closed the same day. Any future *local-file* ingest root must adopt the same pin rule or the interior identity regains a blind spot.

### 7. Plan revision recommendations

The plan (a single checked row + Revisions) matches the code; no contradiction. One optional tightening: add a one-line `## Revisions` note acknowledging that the Spec's "reducer key incorporates manifested row-content hashes" constraint is satisfied *transitively* (fold content is determined by input identities + #25 pins; the told-set key itself is the deterministic partition id, content-hash threading deferred per the `partitionRef` comment) — so the Spec paragraph can't later be read as claiming a literal content-hash term that the code doesn't carry.
