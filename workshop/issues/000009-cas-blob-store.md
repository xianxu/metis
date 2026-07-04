---
id: 000009
status: open
deps: []
github_issue:
created: 2026-07-03
updated: 2026-07-03
estimate_hours:
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

## Done when

- `put` / `get` / `has` round-trip bytes by content hash, with integrity-verify on
  read and dedup of identical content (unit-tested).
- Size-bounded eviction reclaims entries without breaking a consumer (wiped →
  recomputed), proven by an evict-then-refetch test.
- The backend sits behind an interface a second impl (in-memory fake) satisfies, so
  #2 can test against a fake and S3 can slot in later.

## Plan

- [x] Design note settled (2026-07-03, see `## Spec`).
- [ ] (post-design) split implementation milestones.

## Log

### 2026-07-03
- Filed as the storage floor of the metis-v1 cache chain (**CAS ‹ #3 record ‹ #2 cache**), split out of #2 during the caching design so the blob-store *mechanism* stays separate from cache *policy*. Sole MVP consumer is #2 (+ pointer materialization); #3's record only *references* CAS addresses (hashing ≠ storing), so #3 does not depend on it. v1 scope: pure wipeable cache with size-bounded eviction; rooted/durable archival deferred to the #8 promotion hook. Full design: metis-v1 project file + pensive §Caching + metis#2/#3 `## Design`.
