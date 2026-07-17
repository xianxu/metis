---
id: 000025
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-17
estimate_hours: 0.47
started: 2026-07-17T16:50:53-07:00
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

## Spec (at claim, 2026-07-17 — reframed against the current architecture)

The 2026-07-07 premise (local dataset behind a stable path) predates M1a: today the interior is
input-addressed (`Kpre` + transitive-D), upstream artifacts are class-1 keyed via the upstream
`Kpre` chain, and the ONLY external-data root is `kaggle/download` — a REMOTE fetch whose key
material is the competition slug string. Remote content is unknowable at key time and unprobeable
at HIT time without network+creds — so the sound model is **Nix's fixed-output derivation:
declare the expected content identity in config** (ARCH-PURE: identity is data, not IO):

- `kaggle/download`'s `with` gains an optional `sha256: {filename: hex}` pin map. When present:
  after download+unzip, hash the extracted files; any pinned file missing or mismatching **fails
  the step loudly** (a changed remote can never silently propagate). When absent: print the
  computed hashes as a paste-ready pin block + a loud "unpinned ingest" note (escape hatch stays,
  silence doesn't).
- Re-key falls out structurally: `with` is already `Kpre` material, so editing a pin re-keys
  get-data AND everything downstream (the transitive chain) — no metis cache-layer change at all.
- kbench's titanic shapes adopt the pin (computed from the real data files).
- A local-file get-data root, when one exists, uses the same declared-pin contract — recorded in
  atlas as the ingest-identity rule; no speculative local-file machinery now (Simplicity First).

### Done when (reframed — supersedes the original above)

- A pinned download with matching content succeeds; a mutated fake-CLI payload under the same pin
  FAILS loudly (test via the injectable kagglecli fake); an unpinned run prints the paste-ready
  pin block + note.
- Editing a pin produces a different `Kpre` (cite/extend the existing with→Kpre test rather than
  re-prove it).
- kbench titanic get-data is pinned; atlas records the root-identity rule (declared pins for
  external ingest; interior already input-addressed).

*(Original done-when — "same-path data mutation MISSES" — is subsumed: mutation under a pin is a
loud FAIL (better than a silent re-run); mutation without a pin is the documented unpinned mode,
now visible instead of silent.)*

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.08 impl=0.25
item: atlas-docs          design=0.02 impl=0.10
design-buffer: 0.15
total: 0.47
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Rows: (1) pins.go + run() wiring + fake-CLI tests (kaggle repo); (2) kbench pins + metis Kpre
citation/table-case + atlas rule.

## Plan

- [ ] kaggle: `sha256` pin verify + paste-ready block + unpinned note (TDD via kagglecli fake)
- [ ] kbench: pin titanic get-data (real hashes); metis: cite with→Kpre test for re-key
- [ ] atlas: ingest-identity rule; issue Log evidence

## Log

### 2026-07-07
- Side-find of the metis-v2 caching prior-art survey (pensive). Not part of the algebra; a soundness fix.

### 2026-07-17 (claim + reframe)
- Claimed into the platform tranche (operator lane split; #24/#34 queued behind). Recon: the
  issue's local-path premise predates M1a — the live gap is remote ingest identity. Design =
  config-declared content pins (fixed-output derivation); rationale above. ARCH-DRY: no new
  identity mechanism — pins ride the existing `with → Kpre` channel.
