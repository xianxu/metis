---
id: 000025
status: done
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-17
estimate_hours: 0.47
started: 2026-07-17T16:50:53-07:00
actual_hours: 0.9
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

- [x] kaggle: `sha256` pin verify + paste-ready block + unpinned note (TDD via kagglecli fake)
- [x] kbench: pin titanic get-data (real hashes); metis: cite with→Kpre test for re-key
- [x] atlas: ingest-identity rule; issue Log evidence

## Log

### 2026-07-07
- Side-find of the metis-v2 caching prior-art survey (pensive). Not part of the algebra; a soundness fix.

### 2026-07-17 (claim + reframe)
- 2026-07-17: closed — kaggle suite+e2e green, 8 new tests, mutation red-proofed (neutered verify fails the test); kbench e2e 3 passed with the new stderr note; pinned titanic-sweep.md parses via -dry-run (10x72x10); cross-repo commits pinned in Log (kaggle 0960f34+a9aadcf, kbench 742238c); re-key cited to existing with->Kpre tests. actual 0.9h = LABELED JUDGMENT (transcripts under brain project dir - engine cannot attribute; not a measured value); review verdict: SHIP
- Claimed into the platform tranche (operator lane split; #24/#34 queued behind). Recon: the
  issue's local-path premise predates M1a — the live gap is remote ingest identity. Design =
  config-declared content pins (fixed-output derivation); rationale above. ARCH-DRY: no new
  identity mechanism — pins ride the existing `with → Kpre` channel.

### 2026-07-17 (built — evidence)
- **kaggle repo** (commits `0960f34` + `a9aadcf`, pushed): `pins.go` (recursive slash-relative
  sha256, contract files excluded mirroring metis `collectArtifacts`; all failures in ONE error
  incl. unpinned extras — declared identity is complete) + `run()` wiring (pins → verify, loud
  fail; no pins → `UNPINNED ingest` note + paste-ready block). 8 new tests; **mutation test
  red-proofed** (neutered the verify call → test FAILS; restored → passes). Full kaggle suite +
  its e2e green.
- **kbench** (commit `742238c`, pushed): `titanic-sweep.md` get-data pinned (3 files, hashes
  from live run `best-rf-6dde4f89`'s get-data artifacts); RUNBOOK note (pin edit ⇒ cold run +
  new cohort); baseline/features/sweep-smoke deliberately UNPINNED (e2e dual-use — plan
  Revisions). kbench e2e suite green (3 passed, 60s) — the new stderr note breaks no assertion;
  pinned shape parses (`metis run -dry-run`: 10×72×10 listed).
- **Re-key citation (no new metis code):** `with` → `Kpre` sensitivity already pinned by
  `caching_test.go:123-132` (knob change → selective MISS), `:447-452` (nested-map re-key) and
  map-order canonicalization by `record_test.go:12-22` — a `sha256:` map is deterministic Kpre
  material; a pin edit re-keys get-data + the transitive chain structurally.
- Plan checkboxes all ticked; atlas ingest-identity rule added (experiment.md cache section).
