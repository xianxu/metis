---
id: 000025
status: open
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
---

# get-data root cache gap — dataset keyed on path string, not content

## Problem

metis's cache does not hash dataset **bytes** — the dataset enters keys only as the **path string**
(+ code blob-hashes). So if the file behind a stable path **mutates** and the get-data code is unchanged,
get-data takes a **stale cache hit and nothing downstream re-keys** — a silent wrong-answer. A prior-art
survey ranked metis weakest of six: Nix (source content into store) > DVC (dep MD5) > Pachyderm (content
commit) > Nextflow (path+mtime+size) > Make (mtime) > **metis (path only)**.

## Spec

Close the root gap so the content-addressed interior becomes end-to-end trustworthy. Incorporate the
ingested dataset's **content hash** (Nix/DVC/Pachyderm model) into get-data's key — or, as a cheap floor,
**size + mtime** (Nextflow model). Everything above the root is already content-addressed via the
output-hash chain, so only the root needs fixing.

Orthogonal to the metis-v2 algebra (a pre-existing soundness bug), but adjacent: nested-CV's per-outer-
fold data-identity relies on partition artifacts re-keying correctly, which this makes trustworthy at the
source. Related: metis#24 (interior addressing).

## Done when

- get-data's cache key incorporates dataset content identity (content-hash, or at minimum size+mtime); a
  same-path data mutation now MISSES (re-runs) instead of stale-hitting.
- A test: cache get-data, mutate the file in place at the same path, re-run → MISS + downstream re-key.

## Plan

- [ ] (spec at claim) add dataset content/stat fingerprint to get-data's key material; same-path-mutation MISS test.

## Log

### 2026-07-07
- Side-find of the metis-v2 caching prior-art survey (pensive). Not part of the algebra; a soundness fix.
