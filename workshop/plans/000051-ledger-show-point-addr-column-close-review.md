# Boundary Review — 000051-ledger-show-point-addr-column#51 (whole-issue close)

| field | value |
|-------|-------|
| issue | 51 — ledger show — add a point_addr column (the --point handle has no surface) |
| repo | 000051-ledger-show-point-addr-column |
| issue file | workshop/issues/000051-ledger-show-point-addr-column.md |
| boundary | whole-issue close |
| milestone | — |
| window | 07d97e9d0e92ee61827d73e4b9bb29d5498b9611..HEAD |
| command | sdlc close --issue 51 |
| reviewer | claude |
| timestamp | 2026-07-17T22:59:37-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

**Summary.** The column and the round-trip test are well-intentioned and the test drives the real resolver — but the diff fails its own Done-when on the exact form it names. `metis ledger show <shape> --sort <metric>` routes rows through `ledger.AggregateView` (`cmd/metis/ledger_cmd.go:106`), and `AggregateView` **overwrites `PointAddr` with a synthetic group key** — `CodeFingerprint + "|" + Level + "|" + JSON(FreeParams)` (`pkg/ledger/ledger.go:258-263`). So on a modern per-fold ledger (the metis#18 norm), the new point column under `--sort` renders `short(key)` — e.g. `cf||{"tr` for the existing test fixture — which visually echoes the code column and does **not** resolve through `resolvePointRows` (real point addrs are content hashes; the prefix match at `select_cmd.go:392` will error "no ledger row's point_addr starts with…"). The shipped round-trip test never catches this because it feeds `renderLedger` directly with `Fold: nil` rows, bypassing the aggregation path entirely. metis#52 already solved this exact problem on the select side with `pointHandleFor` (`select_cmd.go:260-269`), which deliberately looks up a **raw** row's addr — confirming the implementor's own prior knowledge that aggregate rows don't carry usable handles. The unsorted view works; the sorted leaderboard — the operator's actual discovery flow — does not. That's the purpose of the issue, so this blocks the close.

## 1. Strengths

- The round-trip test drives the **real** `resolvePointRows` rather than reasserting a prefix match by hand (`ledger_cmd_test.go:234`) — exactly the right pinning instinct, and the Log shows it surfaced the resolver's expand-to-config semantics.
- Fixture rows carry distinct free-params per row (`ledger_cmd_test.go:217-220`), showing correct understanding that the resolver groups by config and would flag ambiguity otherwise.
- Minimal, well-placed diff: column after `code` per Spec, header and doc comment updated together (`ledger_cmd.go:138-159`).
- Scope discipline: the #52 overlap (select board line) is correctly documented as already shipped and excluded.

## 2. Critical findings

1. **`--sort` renders an unresolvable synthetic key as the point handle — Done-when not met** (`cmd/metis/ledger_cmd.go:159` interacting with `pkg/ledger/ledger.go:263`). With any per-fold ledger, `showLedger(--sort)` aggregates first, so `short(r.PointAddr)` prints the first 8 chars of `fingerprint|level|{json}` — misleading (looks like the code fingerprint) and guaranteed to fail `select --point <prefix>`. *Fix sketch (preferred):* in `AggregateView`, set the aggregate row's `PointAddr` to the **first member row's real addr** instead of the group key (keep `key` for the internal map only; update the `ledger.go:258-260` comment). This is sound because `resolvePointRows` documents "any of its fold rows works as a handle" and expands to the whole config; no test pins the aggregate `PointAddr` to the key (checked `ledger_test.go:195-293`), and `SortAll`/`configStatsFromLedger`/`runPointSelect` don't read it. *Alternative (smaller blast radius):* remap `PointAddr` locally in `showLedger` after aggregation via a first-raw-row lookup reusing `freeParamMapsEqual` (mirror of `pointHandleFor`).

## 3. Important findings

1. **The round-trip test doesn't cover the Done-when's named scenario** (`cmd/metis/ledger_cmd_test.go:211-243`). It constructs `Fold: nil` rows and calls `renderLedger` directly, so the aggregation path — the one that's broken — is unexercised; the suite stays green while the contract fails. Rework the fixture test to go through `showLedger(shapePath, "", metric, dir, 0, &buf)` over a per-fold ledger (reuse the existing `writePerFoldLedger` helper at `ledger_cmd_test.go:39` — ARCH-DRY), extract the `point` cell from a rendered row, and resolve it against the **raw** ledger via `resolvePointRows`. That is the true end-to-end round trip the Done-when states.

## 4. Minor findings

- `ledger_cmd_test.go:229`: `r.PointAddr[:8]` hand-slices instead of reusing `short()` — trivially brittle if the width ever changes.
- `ledger_cmd.go:138-140`: the rewritten doc comment has awkward nested parens ("(a header row + …, the metis#51 point handle (short point_addr — …), status, …)"); worth a smoothing pass while fixing the Critical.
- Empty `PointAddr` (legacy rows) renders as an empty cell in the two-space-joined table — consistent with how missing metrics already render; no action needed.

## 5. Test coverage notes

- `TestShowLedger_AggregatesPerConfig` (`ledger_cmd_test.go:65`) now renders the garbage point cell but asserts nothing about it — after the fix, extend it (or the reworked round-trip test) to assert the point cell equals `short()` of a member row's addr, pinning against regression at the aggregation seam.
- The test correctly runs without IO for the pure `renderLedger` path; the reworked version will need `t.TempDir()` via `writePerFoldLedger`, which matches the existing INTEGRATION pattern for `showLedger`.
- The plan has no Core concepts table (reasonable for a 0.17h issue); nothing to cross-check there.

## 6. Architectural notes

- **ARCH-PURPOSE: flag.** The issue's purpose is "the --point handle has a discovery surface." The diff delivers it only for unaggregated views; on the sorted leaderboard — the form the Done-when names and the flow the operator was in when they hit the gap — the column is noise. This is the easy-subset failure mode the marker exists for.
- **ARCH-DRY: pass on the diff itself** (no duplicated logic introduced), with a directive for the fix: `pointHandleFor` (`select_cmd.go:260`) exists *because* aggregate rows lose real addrs. Fixing `AggregateView` to carry a representative real addr makes the source authoritative and would let both consumers (select board line, ledger show) derive from it — the single-source fix. If the local-remap alternative is chosen instead, reuse `freeParamMapsEqual`, don't write a third matcher.
- **ARCH-PURE: pass.** `renderLedger` stays pure and buffer-tested; `showLedger` remains the thin testable core over an injected `io.Writer`.
- **Docs gate: pass.** `--no-atlas` justification holds — atlas documents `ledger show`'s surface at flow level (`atlas/index.md:37`), not per-column, and README doesn't document `ledger show` at all, so no README gap.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/issues/000051-ledger-show-point-addr-column.md`:

- *Revision:* the round-trip plan item ("rendered short handle resolves via the --point prefix path") was delivered against `renderLedger` on raw rows only; under `--sort` (the Done-when's form) `AggregateView` replaces `PointAddr` with its synthetic group key, so the rendered handle neither looks like nor resolves as a point addr. Un-tick both the column and round-trip items until the aggregate rows carry a representative real member addr (fix in `AggregateView`, or a `showLedger`-local remap) and the fixture test round-trips through `showLedger --sort` against the raw ledger.

---

## Re-review — 2026-07-17T23:04:45-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 51 — ledger show — add a point_addr column (the --point handle has no surface) |
| repo | 000051-ledger-show-point-addr-column |
| issue file | workshop/issues/000051-ledger-show-point-addr-column.md |
| boundary | whole-issue close |
| milestone | — |
| window | 07d97e9d0e92ee61827d73e4b9bb29d5498b9611..HEAD |
| command | sdlc close --issue 51 |
| reviewer | claude |
| timestamp | 2026-07-17T23:04:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This window is the rework of the prior REWORK verdict, and it fixes both blocking findings at the right place. `AggregateView` now stamps the first member row's **real** `PointAddr` on each aggregate row while keeping the synthetic `fingerprint|level|json` string as a map-internal grouping key only (`pkg/ledger/ledger.go:258-267`), so the sorted leaderboard — the Done-when's literal flow — renders a resolvable handle. The new `TestLedgerShow_SortedPointHandleRoundTrips` drives the real end-to-end path (`showLedger` with `--sort` over a per-fold ledger on disk → extract the rendered point cell → resolve it via the real `resolvePointRows` against the **raw** ledger → assert the resolved config matches the row). I ran the shadow-sweep over every consumer of aggregate-row `PointAddr` (`SortAll`, `configStatsFromLedger`, `runPointSelect`, `FamilyEstimate`, all pkg/ledger tests) and none depended on the old synthetic key — the semantic change is safe. The fix is sound against the resolver's documented contract ("any of its fold rows works as a handle", `select_cmd.go:386-388`). One caveat on process: the Bash tool cannot initialize in this review environment, so I could not execute the suite; I verified the new tests by tracing their logic against `SortAll`/`betterThan`/`resolvePointRows` and found no failure path. Remaining findings are Minor only.

## 1. Strengths

- **Fixed at the source, not patched at the surface** (`pkg/ledger/ledger.go:267`): the prior review offered a `showLedger`-local remap as the small-blast-radius alternative; the implementor took the preferred single-source fix instead, so any future consumer of `AggregateView` gets a usable handle for free. ARCH-PURPOSE delivered.
- **The end-to-end test is the Done-when, literally** (`ledger_cmd_test.go:217-272`): per-fold ledger on disk → `showLedger --sort` → rendered cell → `resolvePointRows` against the raw ledger → config identity asserted. It also asserts the cell is one of *that config's own* member addrs, which would catch a cross-config mix-up, not just a resolve failure. It reuses the existing `writePerFoldLedger` helper as the prior review directed (ARCH-DRY).
- **Idempotence preserved**: pass-through (`Fold==nil`) rows keep their own addrs untouched, and `TestAggregateView_NonFoldRowPassesThrough` (`ledger_test.go:281-293`) still pins the fixpoint.
- Both prior Minor findings addressed: the test uses `short()` instead of hand-slicing `[:8]`, and the `renderLedger` doc comment was rewritten cleanly (`ledger_cmd.go:138-141`).

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

- `cmd/metis/ledger_cmd_test.go:220`: `showLedger(…, "desc", …)` passes a direction token the CLI never produces (`--dir` documents `maximize | minimize`); it only works because `betterThan` treats anything non-`minimize` as maximize (`ledger.go:399-404`). Use `"maximize"` so the test doesn't read as if `desc` were supported.
- `pkg/ledger/ledger.go:262-263`: the comment claims the aggregate `PointAddr` is "the single source both `ledger show`'s point column **and select's #52 pick-line handles** derive from" — but the pick lines still derive from `pointHandleFor`'s independent raw-ledger scan (`select_cmd.go:262-269`), which was not changed. The values coincide (both take the first append-order member), but the comment overclaims; soften it to "…and a valid handle select's #52 pick lines could derive from" or actually consolidate (see §6).
- No pkg/ledger unit test pins the new contract ("aggregate row carries a real member addr") — it's pinned only via the cmd-level round-trip. One assertion in `TestAggregateView_ReducesPerConfig` (`ledger_test.go:244-250`, e.g. `byModel["a"].PointAddr` ∈ {`a0`,`a1`}) would pin it at the unit seam where the comment makes the promise.

## 5. Test coverage notes

- Coverage is now correct at both seams the prior review flagged: `renderLedger` in isolation (`TestRenderLedger_PointColumnRoundTrips`, pure buffer, no IO — ARCH-PURE pass) and the aggregation path end-to-end. The unit-level `AggregateView` pin above is the one small remaining hole.
- The test's `cols[1]` extraction assumes a non-empty `PointAddr`; true for this fixture. A legacy empty-addr row would shift `strings.Fields` columns — worth remembering if a legacy-row rendering test is ever added, but no action needed here.
- Suite execution was not possible in this environment (Bash tool failure); logic-traced instead as noted in the summary. The main agent should confirm `go test ./...` green before committing the close — the Log claims suite + `-race` green and nothing I read contradicts it.

## 6. Architectural notes

- **ARCH-DRY: pass, with one consolidation left on the table.** Two "representative addr for a config" derivations now coexist: `AggregateView`'s first-member stamp and `pointHandleFor`'s raw scan. They agree today by construction (both first-append-order). A future issue could retire `pointHandleFor` by deriving pick-line handles from the aggregate rows select already computes — then the ledger.go comment's "single source" claim becomes true rather than aspirational. Not blocking; #52 shipped and works.
- **ARCH-PURE: pass.** `AggregateView` stays pure and order-deterministic; `renderLedger` pure over `io.Writer`; `showLedger` remains the thin injected-writer core.
- **ARCH-PURPOSE: pass.** The purpose — the `--point` handle has a discovery surface — is now delivered on the sorted leaderboard, the flow the operator was actually in. Shadow-sweep of consumers found no remaining hand-maintained restatement; the one stale *claim* (the comment) is noted above as Minor.
- **Docs gate: pass.** There is no `README.md` in this repo, so no README gap is possible; atlas documents `ledger show` at flow level (`atlas/index.md:33-39`), not per-column — the `--no-atlas` justification recorded in the plan holds.

## 7. Plan revision recommendations

None — the plan's three ticked items are all actually delivered, and the Log's rework entry honestly records both the original defect and the fix. The issue is ready to close.
