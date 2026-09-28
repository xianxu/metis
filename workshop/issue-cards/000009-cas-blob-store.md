---
id: '000009'
status: done
started: 2026-07-05T12:50:01-07:00
created: 2026-07-03
updated: 2026-07-05
estimate_hours: 1
actual_hours: 1.00
---

# Content-addressed blob store (CAS): put/get by content-hash, size-bounded eviction

## Problem

The v1 cache (#2) and the pointer-materialization it relies on both need a place to
put and fetch bytes **by content hash** — step outputs, plus any retained input
bytes. There is none today: `metis run` writes step outputs to fixed working-tree
paths, which clobber across configs in a sweep and can be neither deduplicated nor
reused. Before #2 can skip/recompute or materialize cached outputs, it needs a dumb,
generic content-addressed blob store underneath it.
