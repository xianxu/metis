---
id: '000027'
status: done
started: 2026-07-11T20:25:34-07:00
created: 2026-07-11
updated: 2026-07-11
estimate_hours: 3.47
actual_hours: N/A
---

# run/code identity split: intent-identity + post-run code fingerprint

## Problem

A run's identity is minted **before** the run (to name the run-dir and to dedup the ledger), but its
true **code identity** (the read-set `D`) is only known **after** the run (D is discovered by tracing
execution). Today metis papers over this tension by folding the **workspace repo HEAD** (`repo_shas`)
into `point_address` = `CanonicalHash(resolved_with, repo_shas, seed)` (`pkg/record/address.go:33`).
That repo-HEAD term is a *pre-run code proxy* — it's what makes two runs of the **same config but
different code** get distinct `point_address`es (why the two Titanic sweep cohorts don't collide).

Consequences:
- **Coarse + commit-forcing:** repo HEAD moves on any repo change; a *dirty* run can't be
  content-identified (two dirty variants share HEAD → same `point_address` → the ledger dedups → one
  **silently overwrites** the other, losing "which variations we swept").
- **Blocks metis#26:** #26 wants to drop the repo-HEAD `sweep_sha`. But dropping it *without a
  replacement for the code-identity role* re-introduces exactly the same-config-different-code
  collision. So #26 depends on this issue.

The deeper fact (surfaced in the walkthrough): **you cannot put precise code identity in the pre-run
key** — it's runtime-discovered. And per-step `D` can even differ *within* a run (each step is a
separate process; a file edited between steps yields two blobs for one path), so a run's code identity
is only well-defined when code is *consistent across its steps* (see metis#28 for the guard).
