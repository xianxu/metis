# Boundary Review — metis#9 (whole-issue close)

| field | value |
|-------|-------|
| issue | 9 — Content-addressed blob store (CAS): put/get by content-hash, size-bounded eviction |
| repo | metis |
| issue file | workshop/issues/000009-cas-blob-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1819d06998c0de9c299017b82043a063f441bd3c..HEAD |
| command | sdlc close --issue 9 |
| reviewer | claude |
| timestamp | 2026-07-05T13:08:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

A clean, well-tested implementation that faithfully delivers issue #9's mechanism-only scope: content-addressed put/get/has, sha256 keys, self-dedup, integrity-verify-on-read, sharded atomic-write FS pool, injected-clock LRU eviction, and a swappable interface with an in-memory fake. All three Done-when items are met and proven by tests (round-trip+integrity+dedup, size-bounded eviction with an explicit evict-then-refetch test, `Store` interface satisfied by both `MemStore` and `FSStore`). Build/vet/tests are green. No Critical findings and nothing blocks the boundary — the two Important items are cheap refinements (one small behavior fix in `Get`, one doc/contract note) that are worth a pass before close but don't gate.

**1. Strengths**
- `runStoreContract` (cas_test.go:38) runs the *same* semantics suite against both `MemStore` and `FSStore` — one source of truth for the interface contract, exactly the right ARCH-DRY move for a swappable seam.
- Pure `selectEvictions` (cas.go:66) extracted from the FS shell and unit-tested with zero filesystem (cas_test.go:117+) — textbook ARCH-PURE. The `keep` protection + oldest-atime-first + hash tie-break gives deterministic, testable eviction.
- Atomic temp+rename write (fs.go:`writeAtomic`) with `.tmp-` prefix skipped by the eviction scan (fs.go:`list`) — a concurrent reader never sees a partial blob.
- `MemStore` defensively copies on both Put and Get (mem.go:37,46) so callers can't mutate stored bytes through an alias.
- Both impls carry `var _ Store` compile-checks; `Clock` mirrors `pkg/experiment`'s convention (verified — pkg/experiment/run.go:28).
- atlas/index.md carries a thorough, accurate pkg/cas entry.

**2. Critical findings**
None.

**3. Important findings**
- **`Get` discards a valid read when the LRU touch fails** (fs.go:115-117). The bytes are already read and integrity-verified at this point; failing the whole Get because `os.Chtimes` (recency stamp) returned an error is over-strict. Worse, the returned error is neither `ErrNotFound` nor `ErrCorrupt` — a wipeable-cache consumer that keys recompute off those two sentinels won't recognize it and may hard-fail instead of recomputing. On a permission-constrained or read-only-metadata mount this turns *every* Get into a hard error. Fix: best-effort the touch — ignore (or log) its error and `return data, nil`. Same pattern in `Put` (touch after a successful write) is lower impact (re-Put is idempotent) but deserves the same treatment.
- **FSStore concurrency contract is undocumented and asymmetric with MemStore.** `MemStore` explicitly documents "Safe for concurrent use" (mem.go:5); `FSStore` says nothing. Under a tight `maxBytes` with concurrent Puts, each Put's `evict` protects only its *own* `keep`, so goroutine A's evict pass can delete the blob goroutine B just wrote — B's next Get then returns `ErrNotFound`. Correctness survives (consumer recomputes), but #2 will consume this and needs the guarantee stated. Recommend a doc comment on `FSStore` making the best-effort/eviction-under-concurrency semantics explicit.

**4. Minor findings**
- `evict` walks the entire pool (`list` = ReadDir every shard + `Info` every file) on *every* Put — O(pool size) per write. Fine now; won't scale to a large pool (future: index/heap).
- `Get` mutates the filesystem (Chtimes) on every read — a read on a read-only mount fails. Inherent to mtime-LRU; note it.
- `entry.atime` actually holds `info.ModTime()` (mtime), and `touch` sets both via `Chtimes(t,t)`. The recency signal is mtime; the field name says atime — mildly confusing.
- "14 tests" in Plan/Log is approximate (~20 leaf assertions incl. contract subtests). Trivial; no plan revision warranted.

**5. Test coverage notes**
Strong overall. Gaps worth a follow-up test (none blocking):
- No empty-blob round-trip (`Put([]byte{})` / `Put(nil)`) — sha256 of empty is valid but untested.
- `MemStore`'s copy-on-Put/Get isolation is load-bearing but unverified — a test that mutates the caller's input slice after Put, and mutates a returned slice, then asserts the stored bytes are intact, would pin it.
- No concurrent-access test for `FSStore` (ties to Important #2).

**6. Architectural notes for upcoming work**
- ARCH-DRY — **pass.** The local `Clock` re-declaration is a deliberate, documented choice (the floor layer must not depend upward on `pkg/experiment`); duplicating a one-line type to preserve dependency direction is correct. Shared contract test and extracted `selectEvictions` avoid the duplication that mattered.
- ARCH-PURE — **pass.** Pure core (`HashOf`, `selectEvictions`) unit-tested without IO; `FSStore` is the thin IO shell; `Store` is the injected seam; `MemStore` the injectable fake. Reference-quality split.
- ARCH-PURPOSE — **pass.** Scope is mechanism-only by design and fully delivered; the shadow-sweep doesn't apply (CAS is a standalone floor, not a single-source being compiled to consumers). #2/#3/#8 consumers are correctly deferred as separable extensions, not the deferred point of this issue. For #2: note that `MemStore` never evicts, so a consumer tested *only* against the fake won't exercise the ErrNotFound-after-Put recompute path that the wipeable-cache contract requires — #2 should cover that path against `FSStore` (or a bounded fake).

**7. Plan revision recommendations**
None — the Plan and Log accurately describe the shipped code (the "14 tests" wording is approximate but not worth a `## Revisions` entry).
