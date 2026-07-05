# Boundary Review — metis#3 (milestone M1)

| field | value |
|-------|-------|
| issue | 3 — Run provenance: snapshot the resolved pipeline config (+ experiment git sha) so ## Runs is knob→score legible |
| repo | metis |
| issue file | workshop/issues/000003-run-provenance.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | cc04241c08f9e55782e2a41cbd72716c9fc6caee^..HEAD |
| command | sdlc milestone-close --issue 3 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-05T14:05:56-07:00 |
| verdict | SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
Confirmed: `pkg/record` depends only on `pkg/cas` (not `pkg/experiment`) — a cleaner decoupling than the plan specified. I have everything I need. Writing up the review.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** metis#3 **M1 (pure record core)** is a tight, well-tested boundary. The delta under review is just the two `#3 M1` commits on top of the branch point `d0dd7c9` — `pkg/record` (structs + `PointAddress` + `OutputHash`), the `pkg/experiment` per-step-retention change, the `#RunRecord`/`#StepRecord` CUE + drift guard, and the one-line `cmd/metis` caller update. (The window base `cc04241^` reaches back before the `#9` CAS merge, so `pkg/cas/*` and the CAS atlas/lessons/history entries also appear in the diff — but that is already-reviewed, closed, merged work with its own SHIP close-review sidecar in `workshop/history/`; I treated it as a pre-reviewed dependency, not a subject of this review.) Build, `vet`, the full suite, and `-race` on `pkg/record` are all green; the CUE drift guard actually runs (0.01s, not skipped — `cue` is on PATH) and exercises every field. Every M1 Plan checklist item is delivered, and there are no Critical or Important findings. Nothing blocks the boundary.

### 1. Strengths

- **`pkg/record` is a clean leaf (ARCH-PURE / ARCH-DRY).** Verified via `go list -deps`: it depends only on `pkg/cas`, **not** `pkg/experiment`. The `StepRun`→`StepRecord` mapping is deferred to the `cmd/metis` assembly site (M2), so the record package carries no orchestration coupling and no IO. This is *better* than the plan, which specified importing `pkg/experiment`.
- **`PointAddress` canonicalization is pinned by the right test.** The map-order footgun is the real risk, and `record_test.go:11` hammers it 25×; sensitivity (`TestPointAddress_Sensitivity`) confirms each determinant moves the address and an identical set reproduces it. `address.go` relies on `encoding/json`'s documented sorted-map-key guarantee at every map level — the correct, minimal choice, with the RFC-8785 upgrade path noted.
- **The drift guard is genuine enforcement, not documentation.** `conformance_test.go` marshals a fully-populated `RunRecord` (including the optional `d`/`deps`/`upstream`/`output_hash`/`metrics` slots) and `cue vet`s it against the closed `#RunRecord`; a renamed/removed/extra Go field fails the build. It correctly `t.Skip`s when `cue` is absent, matching the established `#Run` pattern.
- **Per-step retention is minimal and correct.** `Runner.Run` keeps the flat `Run` (back-compat/ledger) and adds `[]StepRun` in topo order; all three callers (`run_test.go` ×2, `cmd/metis/run.go`) are updated; `TestRunner_Run_ReturnsPerStepResults` asserts order, retained resolved `With`, metrics, and artifacts.
- **Tests pin properties, not the implementation** — determinism, sensitivity, order-independence, nil==empty, JSON round-trip. No mock-reassertion.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

- **`OutputHash` sorts by `Path` only** (`pkg/record/address.go:44`). Duplicate-path inputs have unspecified relative order under `sort.Slice` (not stable). Not reachable with well-formed unique-path artifact sets, but a secondary sort key (`Hash`) would make it fully order-independent — mirroring `selectEvictions`' own `(mtime, hash)` tie-break in the sibling CAS package.
- **`PointAddress`/`OutputHash` panic on an un-marshalable value** (`address.go:26`, `address.go:46`). A `.nan`/`.inf`/`-.inf` config value is valid YAML → `float64(NaN/Inf)`, which Go's `json.Marshal` *rejects*. Latent at M1 (pure core, not yet fed real config); **M2 must guard this** when it wires `resolvedWith` from parsed frontmatter — validate config or handle the error rather than crash the run. See architectural note.
- **`repoRoot(t)` test helper is duplicated** across `pkg/record/conformance_test.go:16` and `pkg/experiment/helpers_test.go:15` (~6 lines each, both thin wrappers over `repo.Root`). The meaningful walk is already DRY in `internal/repo`; the last wrapper could hoist to an exported `repo.TestRoot(t)`. Test-only, trivial (ARCH-DRY).
- **`atlas/experiment.md:38`** documents `Runner.Run(exp, runID, runDir)` without the new `[]StepRun` return. Not wrong (it never stated return types), but M2's atlas pass should mention the per-step return.
- **Determinism test doesn't exercise a 3rd-level nested map value** in `resolvedWith` (only the two map levels of `map[string]map[string]any`). `json.Marshal` sorts all levels so deeper nesting is safe; the test just doesn't pin it.

### 5. Test coverage notes

Coverage is strong and targets the exact bug classes this diff could ship (map-order non-determinism, walk-order dependence, CUE↔Go drift, lost/misordered per-step data). Two edge gaps, both Minor above: duplicate-path `OutputHash` ordering and NaN/Inf marshaling — neither reachable with well-formed inputs at M1.

**Docs gate:** `pkg/record` is new architectural terminology, but it introduces **no user-facing or flow surface at M1** — nothing writes a record and there's no CLI/flag change — and the plan explicitly schedules the `atlas/index.md` + record-datatype entry for **M2** (when `record.json` is actually produced and `## Runs` renders). Deferring is defensible here; flagging only so M2 does not skip it. No README surface at M1 → no README finding.

### 6. Architectural notes for upcoming work

- **M2 NaN/Inf guard (from Minor 2):** the point-address derivation is the first place arbitrary user config hits `json.Marshal`; a config with `.inf` currently panics. Decide at M2 whether to reject such config at parse/validate time or make the derivation return an error.
- **CUE-as-source via drift-guard, not codegen (ARCH-PURPOSE).** The Go structs restate `#RunRecord` and a `cue vet` test enforces consistency — the same trade-off already made for `#Run`. This is *enforced* (the build fails on drift), so it satisfies ARCH-PURPOSE's "source is enforced, not just restated" for this codebase's chosen mechanism; it's consistent, not a finding. When **#2** extends the record with the read-set `D`, extend the Go `CodeManifest.D` and CUE `#CodeManifest.d` in the same change — the guard will catch a one-sided edit.
- **Record surface for downstream (#2/#8).** The `Hash` alias, `CodeManifest{D, Deps}` reserved slots, and the `PointAddress`/`OutputHash` signatures are a stable, well-shaped seam for the consumers that will index/derive from the record. No changes needed.
- **ARCH-DRY / ARCH-PURE / ARCH-PURPOSE:** all three **pass** for this boundary (decoupled leaf, pure functions unit-tested with no IO, M1 delivers its own stated scope — the issue-level knob→score purpose is a genuine M2 deliverable, not an under-delivered "follow-up").

### 7. Plan revision recommendations

- **`workshop/plans/000003-run-provenance-plan.md`** — the M1 bullet states *"New pure package `pkg/record` (imports `pkg/experiment` for Step/StepRun, `pkg/cas` for `HashOf`)"*, and Open Decision 1 repeats *"#2/#8 import it without pulling the runner."* The code diverged (better): `pkg/record` imports **only** `pkg/cas`; the `StepRun`→`StepRecord` mapping moved to the `cmd/metis` M2 assembly site. Add a `## Revisions` entry recording that `pkg/record` is a clean leaf over `cas` and does not import `experiment`, so the plan stops claiming an import the code deliberately dropped.
