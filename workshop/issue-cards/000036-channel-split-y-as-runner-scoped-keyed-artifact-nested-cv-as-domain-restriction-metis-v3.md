---
id: 000036
status: working
started: 2026-07-19T16:22:10-07:00
created: 2026-07-14
updated: 2026-07-19
estimate_hours: 17
github_issue:
---

# channel split: y as runner-scoped keyed artifact — nested CV as domain restriction (metis-v3)

## Problem

metis#35's root cause is architectural: the nested-CV seal (metis#23) substitutes a derived
artifact (`analysis_i`) and deletes its producers — sound only when that artifact is the sole road
from raw data to the pipeline, an invariant nothing enforces (the `raw: get-data` bypass proved
it). Row-cloning also costs O(k·N) storage, forces `test=None` shape mismatches, and hides
*features* when only *labels* need hiding — under transductive (Kaggle) semantics, hiding held
rows' features actively mismatches the deployment (both-frames features like ticket_size see
train+test at ship time). The protection we actually need is: **no step's fitted parameter may
depend on a held row's label.**

**Research notes (read first):**
`workshop/pensive/2026-07-14-01-pensive-feature-engineering-algebra-under-cv.md` — the full
model (two-channel data, fit∘apply scope signatures, aggregate classes), the verified literature
map (two deep-research passes + a 27-agent adversarial verify), and the framing: the fit boundary
is a **declassification point**; cross-fitting is the declassification policy for the y channel.
