# Get-Data Root Identity (Content Pins) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close metis#25's root-identity gap with config-declared content pins (Nix fixed-output-derivation model): `kaggle/download` verifies downloaded bytes against an optional `sha256` pin map in `with`, failing loudly on mismatch; pins ride the existing `with → Kpre` channel so a pin edit re-keys the whole downstream.

**Architecture:** All new code lives in the kaggle repo's download step (pure hash/verify helpers + a thin call in `run()`); metis's cache layer is untouched — `with` is already `Kpre` material. kbench adopts the pin in the titanic shapes. Atlas records the ingest-identity rule.

**Tech Stack:** Go (kaggle repo, existing injectable `kagglecli` fake), sha256 stdlib.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `verifyPins` | `kaggle/cmd/kaggle-download/pins.go` | new |
| `pinBlock` | `kaggle/cmd/kaggle-download/pins.go` | new |

- **`verifyPins(dir string, pins map[string]string) (computed map[string]string, err error)`** — hashes every regular file in the step dir (post-unzip), returns the computed `{filename: sha256hex}`; when `pins` is non-empty, errors listing EVERY missing/mismatched pinned file (all failures at once, not first-fail). Pure given a dir listing — tested against real temp dirs (no mocks needed; fs is the input).
  - **DRY rationale:** one hasher/verifier serves verify-mode and print-mode; the paste-ready block derives from the same computed map.
  - **Future extensions:** a local-file get-data root reuses the same pin contract (the atlas rule).
- **`pinBlock(computed map[string]string) string`** — renders the paste-ready `with.sha256` YAML block, sorted by filename. Pure string function.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `downloadWith.Sha256` + `run()` wiring | `kaggle/cmd/kaggle-download/main.go` | modified | step config + stderr notes |
| kbench titanic pins | `kbench/competition/titanic/pipelines/*.md` (get-data `with`) | modified | shape config |

- **`run()` wiring:** after unzip, call `verifyPins`. Pins present → mismatch/missing = step FAILURE (exit non-zero, the runner surfaces it; nothing downstream runs). Pins absent → print the pin block + ONE loud `kaggle/download: UNPINNED ingest — content identity not declared; paste the block below into with.sha256` note to stderr.
- **Re-key is structural:** `with` already feeds `Kpre` (`pkg/cache/cache.go:54`) — cite metis's existing with-sensitivity coverage (`caching_test.go` / `caching_soundness_test.go`; grep for a with-change→MISS case — if none exists explicitly, add ONE table case there, not a new e2e).
- **ARCH-PURPOSE shadow-sweep at close:** every titanic shape whose root is `kaggle/download` carries the pin (grep `uses: kaggle/download` across kbench); atlas ingest rule written; #24's atlas record (next issue) links to it.

## Chunk 1: all tasks

### Task 1: `verifyPins` + `pinBlock` (kaggle repo, TDD)

**Files:** Create `cmd/kaggle-download/pins.go`, `cmd/kaggle-download/pins_test.go`

- [ ] Failing tests: (a) computed map matches known sha256 of fixture files; (b) matching pins → nil error; (c) one mismatched + one missing pinned file → error naming BOTH; (d) empty pins → nil error, computed still returned; (e) `pinBlock` renders sorted, paste-ready YAML.
- [ ] Run red → implement → green.
- [ ] Commit: `#25(metis): pins.go — content verify + paste-ready block` *(kaggle repo; reference metis#25 in body)*

### Task 2: `run()` wiring (kaggle repo)

**Files:** Modify `cmd/kaggle-download/main.go` (`downloadWith` gains `Sha256 map[string]string \`json:"sha256"\``; wire after the unzip loop). Test via the existing fake-CLI test harness in `main_test.go`.

- [ ] Failing tests: (a) fake CLI serves fixture zip, pinned correctly → step succeeds; (b) same pin, MUTATED fake payload → step exits non-zero, stderr names the file; (c) no pins → success + stderr contains `UNPINNED ingest` + the block.
- [ ] Run red → implement → green. Full kaggle suite.
- [ ] Commit: `#25(metis): download verifies declared content pins`

### Task 3: kbench pins + metis Kpre citation

- [ ] Compute real hashes: `shasum -a 256 competition/titanic/data/titanic/*` (kbench) — pin into every shape whose get-data uses `kaggle/download` (grep them). Note: the DOWNLOADED files are what get hashed by the step — confirm the local data dir mirrors what the CLI serves (it was produced by it); if uncertain, run the real download once in a temp workspace to compute pins from the step's own print.
- [ ] metis: locate the with-change→MISS coverage; add one table case if absent.
- [ ] Commit kbench: `titanic: pin get-data content (metis#25)`; metis commit only if a test case was added.

### Task 4: atlas + close

- [ ] metis atlas (experiment.md cache section): the ingest-identity rule — external ingest declares content pins in `with` (fixed-output derivation); interior already input-addressed; unpinned ingest is loud.
- [ ] Issue Log: evidence (test output, the mutated-payload failure line, kbench pin commit sha). Close with cross-repo commit pinned in the Log (the #48 close-review lesson).

**Verification gate:** kaggle suite green; the mutated-payload test red-proofed (revert the verify call, watch (b) fail); kbench dry-run parses the pinned shapes.
