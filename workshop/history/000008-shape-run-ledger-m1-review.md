# Boundary Review — metis#8 (milestone M1)

| field | value |
|-------|-------|
| issue | 8 — Shape run-ledger: CSV sidecar keyed by free-param tuple + promotion to an experiment |
| repo | metis |
| issue file | workshop/issues/000008-shape-run-ledger.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | bda6e9edfae6ff8484fbd06b64e5f21880201bfd..HEAD |
| command | sdlc milestone-close --issue 8 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-05T20:52:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
All three probes confirmed empirically. I have what I need. The M1 subject (`pkg/ledger`) builds, vets, and its tests pass; the surrounding diff is already-merged #2/#6/#7 work the window reaches back over. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** metis#8 **M1 (pure ledger core)** is a clean, well-structured pure package that delivers every M1 Plan checklist item — `Row`/`Ledger`, append-only dedup-by-point-address, ragged union columns, an `encoding/csv`-backed codec, objective-driven `Best`/`TopN`, and `Filter` — all unit-tested with zero IO (ARCH-PURE holds cleanly). The genuine review subject is exactly the `pkg/ledger` two-file addition plus the issue/plan (commit `33ecb6c`); the rest of the diff is pre-reviewed, merged #2/#6/#7 work the window base (`bda6e9e`, the #3 close) reaches back over, treated as a dependency per the #3-review precedent. Build/vet/tests are green. The one thing that keeps this from a clean SHIP: the CSV codec silently **corrupts a list-valued free-param** (`features: [title, family]` → the string `"[title family]"` on round-trip) — and `features` is the keystone swept coordinate in the plan's own titanic example. It's untested (the ragged round-trip test uses only scalars), and it will break M2's `promote` round-trip Done-when when a decoded row is reconstructed into a singleton experiment. Non-blocking at this gate (Important, no runtime consumer at M1), but it should be fixed or explicitly scoped before M2 builds on it.

### 1. Strengths

- **Ragged union-column codec is the right shape and deterministic** (`ledger.go:69`, `:207`). Header = fixed keys + sorted `fp.*`/`metric.*` union; blank cells for absent keys. Sorted columns + append-order rows make the sidecar git-diff-stable — exactly what the batched-commit story needs. I verified decode→encode is **byte-idempotent** even where Go types drift (the string form is stable), so the persisted CSV won't churn in git across a load/save cycle.
- **Namespace prefixing genuinely prevents collisions** (`fp.`/`metric.`). A metric `train.cv_score` and a hypothetical free-param `train.cv_score` land in distinct columns, and the decode switch checks `fpPrefix` before `metricPrefix` so an `fp.metric.foo` key routes correctly. This is the v0 flat last-write-wins fix, done properly.
- **Append dedup lazy-init is correct against pre-seeded rows** (`ledger.go:43`). A `Ledger{Rows: …}` built directly (or via `Decode`) rebuilds `seen` from existing rows on first `Append`, so idempotent re-append works whether the ledger came from a struct literal or a decode. The dedup test (`ledger_test.go:14`) pins both the idempotent-re-run and new-code-version-appends cases.
- **`Best`/`TopN` skip-failed + skip-missing-metric, with deterministic tie-breaking** (`ledger.go:146`, `:165`). `sort.SliceStable` + strict `betterThan` keeps insertion order on ties; `Best` reading `best.Metrics[metric]` on the zero row is safe (nil-map read) because `!found` short-circuits. Both directions and the empty-ledger `!ok` case are tested.
- **ARCH-PURE / ARCH-DRY both pass.** Pure functions, no IO, unit-tested directly; the doc comment names the IO seam (manifest/record.json read + sidecar commit) as cmd/metis. The codec reuses stdlib `encoding/csv` rather than hand-rolling. No duplicated logic worth consolidating.

### 2. Critical findings

None. (No runtime consumer ships at M1; the pure core compiles and its tests pass.)

### 3. Important findings

- **List/slice free-param values round-trip lossily to a string** — `cell` (`ledger.go:231`) renders any non-scalar with `fmt.Sprintf("%v", …)` and `parseCell` (`ledger.go:238`) has no inverse, so a list collapses to an unparseable string. Empirically confirmed: `features: []any{"title","family"}` encodes to the cell `[title family]` and decodes back to the **string** `"[title family]"`, not a list.
  - *Failure scenario:* the plan's keystone `titanic-baseline-shape` sweeps `features: {$any: [[], [title], [title, family]]}`. #7's manifest carries that list as a free-param value; M2's `rowsFromManifest` → `Row.FreeParams["adapt.features"] = []any{…}`. `promote --best` (M2) decodes the persisted CSV, reconstructs the singleton experiment overlaying `features` = the corrupted string, and the reconstructed point mints a **different point-address** → M2's Done-when "the promoted experiment re-runs and reproduces the row's point-address" fails for any shape with a list free-param.
  - *Why it slipped:* `TestCSV_RaggedRoundTrip` (`ledger_test.go:37`) exercises only scalar free-params (`model`, `C`, `n_estimators`) — the exact "green round-trip test blind to the dimension the bug lives in" pattern metis#6/#2 already logged in `lessons.md`.
  - *Fix sketch:* JSON-encode any non-scalar cell in `cell` and try `json.Unmarshal` first in `parseCell` (falling back to the current scalar heuristics), or — if lists are deliberately out of M1 scope — make `cell` error/flag on a non-scalar and add a `## Revisions` note scoping the codec to scalars, so M2 knows to handle `features` before promote. Either way, add a list-valued free-param to the round-trip test and regression-proof it (revert → test fails).

### 4. Minor findings

- **`TopN` panics on a negative `n`** (`ledger.go:178`): `qualified[:n]` with `n=-1` → `slice bounds out of range [:-1]` (confirmed). No caller at M1, but M2's `--top N` flag reaches it; clamp with `if n < 0 { n = 0 }` (or `n = max(0, n)`).
- **Float/int drift on round-trip** (`ledger.go:245`): `parseCell` tries `Atoi` before `ParseFloat`, so a Go `float64(1.0)` decodes to `int(1)` (confirmed). Harmless for the point-address (JSON-equal for whole numbers) and real YAML whole-number knobs are already `int`; only a genuinely fractional-looking whole float drifts. Cosmetic, but a `.` -preserving check (or the JSON approach above) would make it total.
- **Decode silently drops a malformed metric cell** (`ledger.go:128`): a non-numeric `metric.*` value is skipped with no error. Encode never emits one, so only a hand-edited/corrupt sidecar hits it; lenient-but-silent is acceptable for a metric column — note only.
- **`Filter(l, "")` returns the input by value sharing `Rows`/`seen`** (`ledger.go:187`): a subsequent `Append` to the "filtered" ledger mutates the original's `seen` map. Filter is a read-view in practice, but returning a shallow copy that aliases the dedup map is a latent footgun if a caller ever appends to a filtered view.
- **Mixed method/free-function surface**: `Append` is a method; `Encode`/`Decode`/`Best`/`TopN`/`Filter` are free functions taking `Ledger` by value. Consistent internally enough, but worth settling before M2/M3 consume the API.

### 5. Test coverage notes

Tests pin real properties, not the implementation: dedup idempotence + new-address-appends, ragged union columns ($oneof logreg-blanks-`n_estimators` / rf-blanks-`C`), scalar round-trip with blanks + namespaced metrics + failed status, `Best` both directions + skip-failed/skip-missing + empty, `TopN` ordering, `Filter` by sweep-SHA. The gaps that would catch a shipped bug: (a) **list-valued free-param round-trip** — the Important finding, currently uncovered; (b) `TopN` negative-`n`; (c) a decode→encode byte-idempotence assertion (I verified it holds manually — worth pinning so a future codec change can't silently start churning the git sidecar). Empty-ledger encode/decode is implicitly fine (verified: header-only CSV round-trips to an empty ledger).

### 6. Architectural notes for upcoming work

- **The codec's value model is the seam M2/M3 inherit.** Free-param values in this system are `map[string]any` sourced from yaml.v3 (`int`, `float64`, `string`, `bool`, **and `[]any`**). Whatever the fix for the list gap, settle the canonical cell encoding now, because M2's `promotedExperiment(shape, row)` reconstruction and `promote --point 'model=rf,…'` matching both depend on a row's free-param values surviving a CSV round-trip with their type intact enough to reproduce the point-address.
- **Point-address is the dedup identity and the repo-basename caveat rides along.** The issue's Done-when already scopes the checkout-basename `repo_shas` deferral honestly (issue:162); nothing in M1 disturbs it. Keep the ledger's "global dedup" claims scoped to within-a-checkout until that's pinned, as the issue states.
- **ARCH-PURPOSE partial:** M1 legitimately delivers the pure subset and defers M2/M3; the "lift unification" is explicitly conceptual per the plan. The one purpose gap is the list codec (above) — the ledger can't yet faithfully represent the headline free-param of its own keystone example.

### 7. Plan revision recommendations

- **`workshop/plans/000008-shape-run-ledger-plan.md`** — add a `## Revisions` entry recording that the M1 CSV codec round-trips **scalar** free-params only; list/map-valued free-params (e.g. the titanic `features` list) are not yet faithfully round-tripped, and M2 must either extend the codec (JSON-encode non-scalar cells) or handle the reconstruction before `promote` can round-trip a list-bearing shape. This keeps the M1 "CSV codec (round-trip…)" claim honest about what it covers.
- **Atlas (note, not a plan-revision blocker):** `pkg/ledger` is new architectural surface with no `atlas/` entry in this range (the #8 M1 commit touched no atlas file). The plan explicitly schedules the atlas entry for M2 and there is no user-facing/flow surface at M1 (no subcommand/flag yet — those are M2), so deferring is defensible per the #3 M1 precedent — flagging only so M2's `atlas/index.md` pass does not skip the `pkg/ledger` entry. No README exists in the repo, so there is no README surface to update.
