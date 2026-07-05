---
id: 000009
status: working
deps: []
github_issue:
created: 2026-07-03
updated: 2026-07-05
estimate_hours: 1
started: 2026-07-05T12:50:01-07:00
---

# Content-addressed blob store (CAS): put/get by content-hash, size-bounded eviction

## Problem

The v1 cache (#2) and the pointer-materialization it relies on both need a place to
put and fetch bytes **by content hash** — step outputs, plus any retained input
bytes. There is none today: `metis run` writes step outputs to fixed working-tree
paths, which clobber across configs in a sweep and can be neither deduplicated nor
reused. Before #2 can skip/recompute or materialize cached outputs, it needs a dumb,
generic content-addressed blob store underneath it.

## Spec

A **generic content-addressed blob store** — mechanism only; no cache or provenance
semantics (those live in #2 / #3):

- **Interface:** `put(bytes) → hash`, `get(hash) → bytes`, `has(hash) → bool`,
  integrity-verify on read (re-hash, detect corruption). The key *is* the content
  hash (e.g. sha256).
- **Immutable + self-deduplicating:** a hash always maps to the same bytes;
  identical content is stored once. (This is what git's object store is.)
- **Local filesystem** pool at `cas/<hash>` (sharded dirs), behind a **swappable
  interface** so an S3 backend can slot in later (the pensive's "same interface, S3
  later"; S3 itself is `explicitly_out` of v1).
- **Pure cache in v1 — a wipeable store, not durable.** So it carries
  **size-bounded / LRU eviction** and may evict any entry with *no correctness
  impact*: a wiped entry is recomputed (the #2/#3 durability contract guarantees
  every result reconstructs from durable roots — git + external refetch).
  `rm -rf cas/` must always be safe.

**Explicitly not** (keep the primitive dumb): cache keying / skip policy (→ #2);
provenance records / the per-step record (→ #3); rooted / durable retention (post-v1
archival, which hangs off #8's promotion hook — the "rooting" event).

### Revision (2026-07-05): the CAS holds large *output* bytes only — code lives in git

The #8 durability design narrows this primitive's scope:
- **Code + config are NOT CAS bytes.** They're stored in **git** (blobs on a side ref) and
  referenced from #3's records by a `(path, git-blob-hash, commit)` pointer-manifest — git's
  blob-hash *is* the content-hash. So the CAS holds only **large *output* bytes** (and any pinned
  external download), as a wipeable `content-hash → bytes` map.
- This *strengthens* the wipeable-cache guarantee: `rm -rf cas/` loses only recomputable output
  bytes; nothing irreplaceable (code manifest, metrics, git blobs) was ever in the CAS. The
  Problem's "retained input bytes" is superseded — inputs are either git (code/config) or
  refetchable (external data), never CAS-durable.

## Done when

- `put` / `get` / `has` round-trip bytes by content hash, with integrity-verify on
  read and dedup of identical content (unit-tested).
- Size-bounded eviction reclaims entries without breaking a consumer (wiped →
  recomputed), proven by an evict-then-refetch test.
- The backend sits behind an interface a second impl (in-memory fake) satisfies, so
  #2 can test against a fake and S3 can slot in later.

## Plan

Atomic mechanism (one `pkg/cas/` package, single-pass — plain checkboxes, closes in one
`sdlc close`). Design settled in `## Spec` + Revision; TDD (tests lead each step).

- [x] Design note settled (2026-07-03, see `## Spec`); impl plan split (2026-07-05).
- [x] `pkg/cas`: `Hash` (hex sha256) + `HashOf(data)`; the `Store` interface (`Put(data)→Hash`, `Get(h)→data` integrity-verified, `Has(h)→bool`) + `ErrNotFound`, `ErrCorrupt`.
- [x] `MemStore` — in-memory fake satisfying `Store` (for #2 to test against). `var _ Store` compile-check.
- [x] `FSStore` — sharded pool `root/<h[:2]>/<h>`, atomic write (temp+rename), integrity-verify + touch-on-access via an **injected clock** (no wall-clock — controllable-time; matches `pkg/experiment` `Clock`). Dedup: re-Put of existing content is a no-op touch. Eviction victim-math is a pure `selectEvictions` (ARCH-PURE, FS-free unit tests).
- [x] Size-bounded LRU eviction: `maxBytes` budget, evict oldest-by-mtime until under budget on Put (0 = unbounded); never evicts the just-written entry. `rm -rf cas/` always safe.
- [x] Tests: round-trip, dedup (stored once), Has, integrity (corrupt file → `ErrCorrupt`), eviction (deterministic via fake clock) + **evict-then-refetch** (re-Put restores), touch-on-Get recency (true LRU), interface conformance — 14 tests green.
- [x] atlas: add `pkg/cas` to `atlas/index.md` + a CAS note (storage floor of the cache chain; wipeable-cache contract).

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.5 impl=0.3
item: atlas-docs             design=0.05 impl=0.05
design-buffer: 0.15
total: 0.98
```

Design held at the greenfield-go-module **floor** (0.5) as a design-recurrence hedge —
*instead of* the ×0.2 pre-settled discount (which would drive design to ~0.1); v3.1 keeps design
hours uncompressed since the operator-loop design dialogue still happens, and under-estimating
already-efficient work is the live calibration risk. Plus the +15% thorough-plan buffer. Impl at
40%-of-v2 (v3.1) for the FS+Mem stores, LRU eviction, integrity-verify, atomic writes, and the
full test set; a small atlas note. (Estimate-quality judge flagged the earlier prose for claiming
both floor and discount; reconciled to floor-only.)

## Log

### 2026-07-03
- Filed as the storage floor of the metis-v1 cache chain (**CAS ‹ #3 record ‹ #2 cache**), split out of #2 during the caching design so the blob-store *mechanism* stays separate from cache *policy*. Sole MVP consumer is #2 (+ pointer materialization); #3's record only *references* CAS addresses (hashing ≠ storing), so #3 does not depend on it. v1 scope: pure wipeable cache with size-bounded eviction; rooted/durable archival deferred to the #8 promotion hook. Full design: metis-v1 project file + pensive §Caching + metis#2/#3 `## Design`.

### 2026-07-05
- **Built `pkg/cas`** (TDD, 14 tests green; build+vet+full-suite clean). `cas.go` (`Hash`/`HashOf` sha256, `Store`, `ErrNotFound`/`ErrCorrupt`, pure `selectEvictions`), `mem.go` (`MemStore` fake), `fs.go` (`FSStore`: sharded `root/<h[:2]>/<h>`, atomic temp+rename, integrity-verify on read, injected-clock mtime-LRU eviction). **Adopted all three `change-code` plan-judge findings** (INFO): (1) ARCH-DRY — matched `pkg/experiment`'s `Clock` convention (local type, no cross-layer dep — CAS is the floor); (2) determinism — `Put`/`Get` stamp mtime via `os.Chtimes(clock())` so eviction ordering is wall-clock-independent (test uses a fake clock, no flakiness); (3) ARCH-PURE — eviction victim-selection is the pure `selectEvictions(entries, maxBytes, keep)`, unit-tested with no filesystem. Estimate-quality judge's advisory (floor-vs-×0.2-discount double-justification) reconciled in `## Estimate` (floor-only, as a design-recurrence hedge).
