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

---

## Re-review — 2026-07-05T13:18:38-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T13:18:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Both issues are **empirically confirmed**. Now I have everything I need to write the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

A clean, well-structured, well-tested storage floor that delivers issue #9's mechanism-only scope: content-addressed put/get/has, sha256 keys, self-dedup, integrity-verify-on-read, sharded atomic-write FS pool, injected-clock LRU eviction, and a swappable `Store` interface with an in-memory fake. Build/vet/tests are green (50 RUN/PASS). The architecture split (pure `HashOf`/`selectEvictions` core + thin `FSStore` IO shell + injected `Store` seam + `MemStore` fake) is reference-quality, and the shared `runStoreContract` is exactly the right ARCH-DRY move. What blocks a clean SHIP is one confirmed correctness gap: the package's own documented recovery contract — *"a corrupt entry is treated like ErrNotFound and recomputed"* — silently fails, because `Get` leaves the corrupt file on disk and `Put` short-circuits on file-existence without re-verifying, so re-Put **cannot heal a corrupted blob** (verified empirically below). Cheap, localized fix; no current consumers and `rm -rf` remains a safe workaround, so FIX-THEN-SHIP rather than REWORK — but fix it in this pass before recording close.

## 1. Strengths

- **`runStoreContract` (cas_test.go:38)** runs one semantics suite against both `MemStore` and `FSStore` — a single source of truth for the interface contract. Textbook ARCH-DRY for a swappable seam.
- **Pure `selectEvictions` (cas.go:78)** extracted from the FS shell and unit-tested with zero filesystem (cas_test.go:117+). `keep` protection + oldest-mtime-first + hash tie-break give deterministic, FS-free eviction. Textbook ARCH-PURE.
- **Atomic temp+rename write** (fs.go:77) with `.tmp-` prefix skipped by the eviction scan (fs.go:185) — a concurrent reader never observes a partial blob.
- **`touch` is correctly best-effort** now (fs.go:104-107) — a failed `os.Chtimes` no longer poisons an otherwise-valid Get/Put. This is the right resolution of the prior review's Important #1, and it's well-reasoned in the comment.
- **`MemStore` defensively copies on both Put and Get** (mem.go:24-26, 38-39), pinned by a load-bearing test (mem_test.go), so callers can't mutate stored bytes through an alias.
- **`FSStore` concurrency semantics are now documented** (fs.go:20-29) — the best-effort/not-strictly-isolating behavior under tight `maxBytes` is stated for #2 to rely on.
- Both impls carry `var _ Store` compile-checks; `Clock` deliberately kept local to preserve dependency direction (floor takes no upward dep).

## 2. Critical findings

**C1 — Corruption recovery via re-Put silently fails; the documented wipeable-cache contract is broken** (fs.go:121-122 + fs.go:61-64). Confirmed empirically:

```
after re-Put: Get err="cas: blob failed integrity check" got=""
CONFIRMED BUG: re-Put did NOT heal corrupt blob; Get still ErrCorrupt
```

The package doc (cas.go:43-46) and Spec both promise: *"A wipeable-cache consumer treats [ErrCorrupt] like ErrNotFound and recomputes."* But the recompute path is `Get→ErrCorrupt→recompute→Put`, and:
- `Get` returns `ErrCorrupt` **without removing** the corrupt file (fs.go:121).
- `Put` sees `os.Stat(p) == nil` (the corrupt file still exists) → takes the dedup branch (fs.go:61-64) → `touch` + `evict`, **never rewriting the bytes**.

So a corrupt blob is poisoned permanently: every subsequent `Get` returns `ErrCorrupt`, and the documented recovery (re-Put) is a no-op. A consumer that follows the contract loops recomputing without progress (or hard-errors). Only a manual `rm` heals it. Note the safe path is preserved — `Get` never returns wrong bytes — so real-world impact is availability-of-one-blob, and the trigger (on-disk corruption of a file that survived an atomic write) is rare; that's why the verdict is FIX-THEN-SHIP, not REWORK. But it is the package's own stated contract, so fix before close.

**Fix (one clean change resolves it):** on the integrity-mismatch branch of `Get`, best-effort remove the corrupt file before returning `ErrCorrupt`, so corruption degrades to `ErrNotFound` and re-Put restores it exactly like the evict-then-refetch path:
```go
if HashOf(data) != h {
    _ = os.Remove(p) // best-effort self-heal: next Put re-writes; degrades to ErrNotFound
    return nil, ErrCorrupt
}
```
This also fixes the secondary `Has`/`Get` inconsistency (a corrupt blob currently reports `Has→true` but `Get→ErrCorrupt`). Add a regression test (see §5). *Alternative:* have `Put` re-verify integrity when the file exists and overwrite on mismatch — but the Get-removes approach is smaller and mirrors eviction.

## 3. Important findings

**I1 — Path traversal via a non-hash `Hash` key on `Get`/`Has`** (fs.go:48-53). `shardPath` joins the raw `Hash` string into a filesystem path with no format validation. Confirmed empirically:
```
shardPath(Hash("../../../../../../etc/hosts")) = "/etc/hosts" ok=true
Has(Hash("../../../../../../etc/hosts")) = true   // existence oracle for arbitrary paths
```
`Get`'s integrity check (`HashOf(data) != h`) blocks *content* exfiltration (it returns `ErrCorrupt`, never the foreign bytes), but: (a) `Has` is an unguarded existence oracle for any path, and (b) `Get` will `os.ReadFile` an arbitrary attacker-named file into memory before rejecting it. Today all keys are internally-generated 64-char hex sha256, so there's no live exploit — but this is the ariadne **base layer**, downstream #2/#3 will pass hashes read from records/manifests, and a base primitive that turns an arbitrary key string into a filesystem path is a latent traversal. **Fix (cheap):** validate the key format in `shardPath` — treat anything that isn't a well-formed hash (length ≠ 64, or non-hex, or contains a path separator / `.`) as absent (`return "", false`). That makes `Get`/`Has` reject malformed keys as `ErrNotFound`/`false` and keeps every on-disk path inside `root`.

**I2 — `evict` failure fails an otherwise-successful `Put`; inconsistent with the best-effort `touch`** (fs.go:64, 72, 145-163). After `writeAtomic` succeeds the blob **is stored**, but if `evict`'s `os.Remove` returns a non-`NotExist` error, `Put` returns `(h, err)` — a valid hash paired with a non-nil error. A consumer that reads the error as "not stored" is wrong (the blob is on disk). Eviction is best-effort cache maintenance, exactly like `touch` (which the same diff correctly swallows). Recommend making `evict` best-effort too — log/swallow its errors so a maintenance hiccup never fails a Put whose write succeeded. This is the "inconsistent error handling across the diff" class.

## 4. Minor findings

- Orphaned `.tmp-*` files (from a crashed/interrupted `Put`) are skipped by `list` (fs.go:185) so they're never counted toward the budget nor reclaimed by eviction — they accumulate until `rm -rf`. Acceptable under the wipeable contract; worth a cleanup-on-startup or age-based sweep later.
- `evict` walks the entire pool (`ReadDir` every shard + `Info` every file) on *every* bounded `Put` — O(pool) per write. Fine now; won't scale to a large pool (future: index/heap).
- `Get` mutates the filesystem (`Chtimes`) on every read — inherent to mtime-LRU; the best-effort `touch` already keeps it from failing reads, so this is just a note.
- No `fsync` on the write/rename — a post-write power loss can lose a just-Put blob. Correct-by-design for a wipeable cache (recompute covers it); note only.

## 5. Test coverage notes

Coverage is strong for the happy paths, dedup, eviction ordering, LRU-recency, and evict-then-refetch. Gaps that map directly to the findings above:
- **No re-Put-after-corruption test** — this is precisely why C1 shipped. `TestFSStore_GetDetectsCorruption` asserts `ErrCorrupt` but never tries to recover. Add: corrupt on disk → `Get`→`ErrCorrupt` → re-`Put`(correct bytes) → `Get` succeeds. (Fails today; passes after the C1 fix.)
- **No malformed-key / path-traversal test** for `Get`/`Has` (I1). Add: `Has("../../etc/hosts")` → `false`, and a `..`-bearing key → `ErrNotFound`.
- **No concurrent-access test** for `FSStore` — the concurrency contract is now documented (fs.go:20-29) but unexercised; a `t.Parallel` Put/Get stress test would pin the "safe per-op" half of the claim.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** Shared `runStoreContract`, extracted `selectEvictions`, and the deliberately-local `Clock` (duplicating a one-line type to preserve dependency direction — floor must not depend upward on `pkg/experiment`) are the right calls. No copy-paste to consolidate.
- **ARCH-PURE — pass.** Pure core (`HashOf`, `selectEvictions`) unit-tested with no IO; `FSStore` is the thin IO shell; `Store` is the injected seam; `MemStore` the injectable fake. Reference-quality split.
- **ARCH-PURPOSE — pass, with one caveat.** Mechanism-only scope is fully delivered and the shadow-sweep doesn't apply (CAS is a standalone floor, not a single-source compiled to consumers; #2/#3/#8 are correctly deferred *separable* extensions, not the deferred point of this issue). The caveat is C1: the issue's stated purpose includes *"may evict/lose any entry with no correctness impact — it's recomputed"*, and the **corruption** sub-case doesn't recompute-into-place. Fixing C1 makes the purpose whole. For **#2**: `MemStore` never evicts and can't corrupt, so a consumer tested *only* against the fake won't exercise the `ErrNotFound`/`ErrCorrupt`-then-recompute path the wipeable-cache contract requires — #2 should cover that path against `FSStore` (or a bounded/corruptible fake).

## 7. Plan revision recommendations

None required **if C1 is fixed** — the Plan/Log accurately describe the intended shipped code, and the Log already supersedes the "14 tests" wording with "25 leaf-passes." Only if the team chooses *not* to fix C1: add a `## Revisions` entry (and amend the `ErrCorrupt` doc at cas.go:43-46 + the Spec's wipeable-cache clause) stating that a *corrupt* blob is not healed by re-Put and requires a manual `rm` — because as written the doc claims a recovery the code doesn't deliver.

---

## Re-review — 2026-07-05T13:28:53-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-07-05T13:28:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

A reference-quality storage floor that fully delivers issue #9's mechanism-only scope: content-addressed put/get/has, sha256 keys, self-dedup, integrity-verify-on-read, sharded atomic-write FS pool, injected-clock LRU eviction, and a swappable `Store` interface with an in-memory fake. All three Done-when items are met and proven by tests. I independently verified build + vet + full `go test ./...` + `go test -race ./pkg/cas/` are all green (27 leaf passes). I also empirically re-verified the three prior close-review fixes — corrupt-heal on re-Put (`TestFSStore_ReputHealsCorruption` passes), malformed-key/path-traversal rejection (`TestFSStore_RejectsMalformedKeys` passes), and best-effort `evict`/`touch` — and confirmed each is present in the current code and not merely claimed in commit messages. **No Critical findings; nothing blocks the boundary.** The lone Important is a cheap test-coverage add (the documented concurrency guarantee has zero test); the rest are minors and forward-looking notes for #2.

## 1. Strengths
- **`runStoreContract` (cas_test.go:38)** runs one semantics suite against both `MemStore` and `FSStore` — a single source of truth for the interface contract. Textbook **ARCH-DRY** for a swappable seam.
- **Pure `selectEvictions` (cas.go:78)** extracted from the FS shell, unit-tested with zero filesystem (cas_test.go:117+). `keep`-protection + oldest-mtime-first + hash tie-break give deterministic, FS-free eviction. Textbook **ARCH-PURE**.
- **The C1 corrupt-heal fix is well-designed** (fs.go:79–89): `Put` re-reads and re-hashes the existing blob before dedup-skipping, so an absent *or corrupt* blob falls through to the atomic overwrite and heals in place — the strictest reading of the wipeable-cache contract (it even heals a corrupt blob nobody has `Get`-ed yet, which the alternative "Get removes corrupt" fix would have left poisoned until first read).
- **Path-traversal hardening** (fs.go:51–71): `isHash` gates `shardPath`, so `Get`/`Has` answer `ErrNotFound`/`false` for any key that isn't 64 lowercase hex — no separator or `..` can escape `root`. Confirmed by `TestFSStore_RejectsMalformedKeys`.
- **Best-effort `touch`/`evict`** (fs.go:127, 171) — a failed `Chtimes` or eviction never fails an otherwise-valid Put/Get; eviction returns nothing and swallows scan/remove errors. Consistent error handling, well-reasoned in comments.
- **`MemStore` defensively copies on Put and Get** (mem.go:24, 38), pinned by a load-bearing test (mem_test.go).
- **`Clock` is deliberately local** — verified `pkg/experiment/run.go:28` has the same `type Clock func() time.Time`; re-declaring a one-line type to keep the floor from depending upward is the correct ARCH-DRY call, not accidental duplication.
- atlas/index.md carries an accurate, thorough `pkg/cas` entry; package is standalone (no external Go references), correct for mechanism-only.

## 2. Critical findings
None.

## 3. Important findings
- **The documented concurrency guarantee is entirely untested** (fs.go:20–29). `FSStore` advertises "individual Put/Get/Has are safe to call from multiple goroutines" plus the best-effort-under-tight-`maxBytes` caveat that #2 is told to rely on — but every existing test is single-goroutine, so the `-race` run I did exercised no concurrent paths and proves nothing about this claim. A small `t.Parallel` stress test (N goroutines Put/Get overlapping content, plus concurrent Put under a tight `maxBytes` asserting no panic / no `ErrCorrupt` / re-Put recovers) would pin the "safe per-op" half and demonstrate the documented residency caveat. Cheap; worth adding in this pass since the guarantee is load-bearing for the next consumer.

## 4. Minor findings
- `Put` re-reads the **entire** existing blob into memory and re-hashes it on every dedup (fs.go:83) — O(size) IO + a 2×-blob memory spike on duplicate Puts. Deliberate (it's what makes heal-on-re-Put correct) and not hot after an expensive recompute, but worth a note for large blobs.
- `Get` returns `ErrCorrupt` without removing the corrupt file (fs.go:144), so a corrupt blob reports `Has→true` but `Get→ErrCorrupt` — a mild inconsistency. Acceptable: `Put` heals it and the un-touched corrupt blob ages out as an eviction victim; `Has` only claims presence, not integrity.
- No `fsync` before/after rename (fs.go:115) — a power loss can lose a just-Put blob. Correct-by-design for a wipeable cache (recompute covers it); note only.
- Orphaned `.tmp-*` files from a crashed Put are skipped by `list` (fs.go:206) so they neither count toward budget nor get reclaimed — they accumulate until `rm -rf`. Fine under the wipeable contract; a future age-based sweep would tidy.
- `evict` walks the whole pool (`ReadDir` per shard + `Info` per file) on every bounded Put (fs.go:188) — O(pool) per write. Fine now; won't scale to a large pool (future: index/heap).
- Plan checkbox (issue line 78) still says "14 tests green"; actual is 27 leaf passes. The Log already supersedes it ("25 leaf-passes"); numbers drift but no `## Revisions` entry warranted.

## 5. Test coverage notes
Strong for happy paths, dedup, eviction ordering, LRU-recency, evict-then-refetch, corruption-detect, corruption-heal, malformed-key rejection, empty-blob, and MemStore copy-isolation. The one real gap maps to Important above: **no concurrent-access test** despite an advertised concurrency contract. (`-race` passed, but only over single-goroutine code, so it validated nothing about concurrency here.)

## 6. Architectural notes for upcoming work
- **ARCH-DRY — pass.** Shared `runStoreContract`, extracted `selectEvictions`, deliberately-local `Clock`. No copy-paste to consolidate.
- **ARCH-PURE — pass.** Pure core (`HashOf`, `selectEvictions`) unit-tested with no IO; `FSStore` is the thin IO shell; `Store` is the injected seam; `MemStore` the injectable fake. Reference-quality split.
- **ARCH-PURPOSE — pass.** Mechanism-only scope fully delivered; shadow-sweep N/A (standalone floor, not a single-source compiled to consumers; #2/#3/#8 are correctly deferred *separable* extensions). The prior C1 gap — the corruption sub-case of "may lose any entry, it's recomputed" — is now genuinely whole (heal-on-re-Put, tested).
- **API-shape decision for #2 (not a defect, a deliberate call to make):** the `Put([]byte)` / `Get() []byte` interface precludes streaming, so "large output bytes" is capped by RAM and Put holds 2× blob size on the dedup path. The simple `[]byte` API is defensible for v1 (Simplicity-First / YAGNI — most step outputs fit in memory, and `bytes.NewReader` wraps trivially for the future S3 backend), so I am **not** flagging it as a finding — but #2's design should consciously confirm blob sizes stay memory-bound before locking consumers against this shape.
- **For #2 (carried from prior review, still valid):** `MemStore` never evicts and cannot corrupt, so a consumer tested *only* against the fake will never exercise the `ErrNotFound`/`ErrCorrupt`→recompute path the wipeable-cache contract requires. #2 should cover that path against `FSStore` (or a bounded/corruptible fake).

## 7. Plan revision recommendations
None. The Plan and Log accurately describe the shipped code (the "14 tests" wording is stale but already superseded in the Log; not worth a `## Revisions` entry).
