---
id: '000039'
status: done
started: 2026-07-15T14:49:03-07:00
created: 2026-07-14
updated: 2026-07-15
estimate_hours: 1.55
actual_hours: 0.65
---

# fingerprint visibility — print cohort at run + a fingerprints inspect command (when/what)

## Problem

`code_fingerprint` is load-bearing but invisible. The metis#35 honest-beat run (2026-07-14):
`metis select` refused with *"ledger spans 2 code-fingerprint cohorts [4cc9b742 b7aee3de] — pin one
with `--fingerprint <hash>`"* — correct guard, but the operator has **never seen either hash**:
`metis run` doesn't print the fingerprint it records under, and no command answers "which of these
is which — when did each run, from what code?" Resolving it took reverse-engineering row counts
from the csv (495 = last session's flat run; 2,490 = today's nested). The provenance already
exists on disk — per-run `record.json` CodeManifest (commit, dirty D, capture status, timestamps)
and the capture refs (`refs/metis/sweeps/<shapeRunID>`) — it just has no surface.
