---
id: '000025'
status: done
started: 2026-07-17T16:50:53-07:00
created: 2026-07-07
updated: 2026-07-17
estimate_hours: 0.47
actual_hours: 0.9
---

# get-data root cache gap — dataset keyed on path string, not content

## Problem

metis's cache does not hash dataset **bytes** — the dataset enters keys only as the **path string**
(+ code blob-hashes). So if the file behind a stable path **mutates** and the get-data code is unchanged,
get-data takes a **stale cache hit and nothing downstream re-keys** — a silent wrong-answer. A prior-art
survey ranked metis weakest of six: Nix (source content into store) > DVC (dep MD5) > Pachyderm (content
commit) > Nextflow (path+mtime+size) > Make (mtime) > **metis (path only)**.
