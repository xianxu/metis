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
