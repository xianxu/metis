---
id: '000013'
status: done
started: 2026-07-06T15:07:51-07:00
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 0.9
actual_hours: 0.41
---

# Config immutability — run output (## Runs / ledger top-N) must leave the experiment .md

## Problem

`metis run` **mutates the experiment `.md` with run output** on every run:
- single run → `appendRunLog(o.expPath, rec)` (`run.go:184`) appends a `- <knobs → score>` line to a
  `## Runs` section (creating it if absent), rewriting the file (`run.go:220`).
- sweep → the ledger top-N summary is regenerated into the body between
  `<!-- metis:ledger:begin -->` / `end` markers (metis#8 `regenLedgerSummary`).

So the config file's content changes even when the *input* didn't → its content-hash churns, and a
committed config is **not** a stable identity. Concretely: this is what forced the repeated
`## Runs`-stripping of `titanic-sweep.md` all through the metis-v1 build, and it makes the
reproducibility model (git rev / blob of the committed config = its identity) unsound. The config
`.md` must be **immutable input**; run output belongs in the record/ledger, not the spec.
