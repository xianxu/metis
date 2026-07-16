# Boundary Review — metis#52 (whole-issue close)

| field | value |
|-------|-------|
| issue | 52 — select surface ergonomics — --cohort listing + point handles on every concrete config line |
| repo | metis |
| issue file | workshop/issues/000052-select-cohort-flag-point-handles.md |
| boundary | whole-issue close |
| milestone | — |
| window | 58e6923cced2ed71053a0e2b4f5f9dd64bbecab9..HEAD |
| command | sdlc close --issue 52 |
| reviewer | claude |
| timestamp | 2026-07-16T08:43:23-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/metis"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. The code checks out; the gap is the docs bullet. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both operator requests are delivered cleanly: `--cohort` is a genuine one-line delegation to the #39 `showFingerprints` core (no second implementation), and the point-handle attachment is a small pure helper reusing the existing `freeParamsEqual`/`short` vocabulary, with a real round-trip test through `--point`. What keeps this off SHIP is the third Done-when bullet — "RUNBOOK §2 mentions both" — which is undelivered and *undeliverable as written*: no RUNBOOK file exists anywhere in the repo or workspace (the #50 close review already noted this), and no atlas file changed in the window either, leaving `atlas/experiment.md:7`'s select flag list stale. That's a non-blocking docs-gate fix plus an issue-file correction, hence FIX-THEN-SHIP.

**1. Strengths**

- ARCH-DRY exemplary on the `--cohort` door: `select_cmd.go:43-47` returns `showFingerprints(shapePath, os.Stdout)` — one core, two CLI surfaces, exactly what the Spec promised. No copy of the cohort table renderer.
- `pointHandleFor` (`select_cmd.go:262-269`) is pure, uses the *cohort-filtered* ledger (so the handle can't point outside a pinned `--fingerprint`), and reuses `freeParamsEqual` — the same predicate `configStatsFromLedger` and `--point` resolution use, so the printed handle is resolvable by construction.
- The round-trip test (`select_cmd_test.go:557-585`) is the right pin: it doesn't just assert the string appears, it feeds the printed handle back through `runSelect(point: …)` and asserts it lands on the same config. That's the exact bug class this feature could ship (a handle that doesn't resolve).
- `TestSelect_CohortFlag` goes through `run(...)`, the real CLI entrypoint — flag registration, `hoistShapePath` ordering, and delegation are all covered, matching Done-when bullet 1's "real CLI entrypoint" wording.
- Honest degrade: `""` handle → no ` · point` suffix printed (`select_cmd.go:249-253`) rather than an empty or fabricated handle.

**2. Critical findings** — none.

**3. Important findings**

- **Docs gate / ARCH-PURPOSE: Done-when bullet 3 is not delivered, and can't be as written.** No `RUNBOOK*` file exists in this repo or any workspace sibling, and the window contains no atlas change. Meanwhile `atlas/experiment.md:7` enumerates select's surface as `[--best|--best-per-model-class|--point ADDR] [--promote]` — now missing `--cohort` and the ` · point <addr>` handle on pick lines — and `atlas/index.md:47` likewise. This is the same gap the #50 close review flagged (its Important finding: "no RUNBOOK exists in the repo… atlas update missing"). Fix: add `--cohort` + the handle display to `atlas/experiment.md` (and index.md's select line), and correct the issue's Done-when to name the real doc target (see §7). Two sentences of docs; cheap before the boundary.

**4. Minor findings**

- `select_cmd.go:262-268`: `pointHandleFor` returns the *first* matching row's `PointAddr` even when it's `""` (a pre-#41 row without point_addr). In a ledger whose old rows predate #41 but share the code fingerprint (metis-binary upgrade doesn't change step-code fingerprint), a later row of the same config carries a valid addr that would then never be shown. Skip empty-addr rows: `if r.PointAddr != "" && freeParamsEqual(…)`.
- `select_cmd.go:38`: the usage string in the hoist error still reads `[--best | --best-per-model-class | --point ADDR] [--promote] [--fingerprint HASH]` — `--cohort` missing from the documented order.
- `TestSelect_CohortFlag` hijacks global `os.Stdout` via `os.Pipe` with no reader until after the run — fine at fixture scale, but would deadlock past the 64KB pipe buffer and isn't `t.Parallel`-safe; a `defer os.Stdout = orig` would also survive an intervening fatal. Terse note only.

**5. Test coverage notes**

- Both Done-when behaviors that ship code are pinned red→green per the Log: the cohort table content (including the `(legacy)` bucket) and handles on `--best` *and* `--best-per-model-class`, plus the round-trip. Good.
- Untested: the `""`-handle degrade path (no ` · point` printed). It's near-unreachable except via legacy empty-addr rows — which is exactly the Minor above; one fixture row with `PointAddr: ""` would cover both.
- `--cohort` on an empty ledger inherits `showFingerprints`' `(no ledger rows…)` path — already covered by the fingerprints tests; no new gap.

**6. Architectural notes** (explicit pass/flag per marker)

- **ARCH-DRY: pass.** Single cohort-table implementation behind two doors; handle lookup reuses `freeParamsEqual` rather than a parallel comparator; `short` reused for display.
- **ARCH-PURE: pass.** `pointHandleFor` is pure and unit-testable; printing goes through the injected `o.out`. The `--cohort` shortcut writing `os.Stdout` sits in `cmdSelect`, which is already the IO shell (it passes `os.Stdout` into `selectOpts` two lines later) — consistent, not a regression.
- **ARCH-PURPOSE: flag (the Important finding).** The two operator requests are fully delivered — per-class winners carry handles too, not just the cheap `--best` line — but the docs consumer named in Done-when doesn't derive from anything; it doesn't even exist. Shadow-sweep: code surface ✓, tests ✓, atlas ✗, RUNBOOK ✗ (nonexistent target).

**7. Plan revision recommendations**

Add a `## Revisions` entry to `workshop/issues/000052-…md`:

> **2026-07-16 — close review:** Done-when bullet 3 referenced "RUNBOOK §2", but no RUNBOOK exists in this repo (same discovery as #50's close review). Re-target the docs bullet to `atlas/experiment.md` (select surface line) + `atlas/index.md:47`, and deliver it in this window. The estimate's `atlas-docs` row already budgeted this.

Also note for the tracker's hygiene: the Plan's single checkbox covers only the TDD/code work; the docs deliverable should appear as its own unchecked item until the atlas touch lands (per §5, `sdlc close` would otherwise cross with `--no-atlas` on an issue that *does* add architectural surface — the precise-flag would be a misuse here).
