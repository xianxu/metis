# Unify `$oneof` into `$any` — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse the two choice primitives into one: `$any` dispatches on its argument shape — **list = untagged sum** (bare value), **map = tagged sum** (bundled `{label: sub}`) — both recursive. Delete `$oneof`.

**Architecture:** The change is localized to `expandDescriptor`'s `$any` case in `pkg/shape/shape.go` (a type-switch on the argument) plus deleting the `$oneof` case; everything else is a doc/test/testdata sweep of `$oneof` references. The **map branch is `$oneof`'s existing logic verbatim** (same `{label: sub}` output → zero consumer change); the **list branch gains recursion** (`expandValue` per element), which is a no-op for today's scalar/list alternatives (backward-compatible).

**Tech Stack:** Go 1.26 (`pkg/shape`); the existing `expandValue`/`expandProduct`/`FreeParam` recursion; CUE only in comments (the value-algebra is untyped, so no schema enum to change).

---

## Core concepts

### The dispatch (the whole engine change)

`expandValue` already routes a `{$key: arg}` map to `expandDescriptor(path, key, arg)`. Today `$any` expects `arg.([]any)` verbatim and `$oneof` expects `arg.(map[string]any)` recursive. **Unify under `$any`** by type-switching on `arg`:

- **`[]any` (list) → untagged sum.** For each element, `expandValue(path, elem)` (recurse), value placed **bare**. Free-param rule (avoids double-recording): if the element's own expansion produced coords (`s.free` non-empty — it was a nested descriptor at this path), **use `s.free`**; else the element is a leaf → record **`{Path: path, Value: s.value}`**. Empty list → error (kept).
- **`map[string]any` (map) → tagged sum.** Exactly today's `$oneof` body: for each `sortedKeys` label, `expandValue(join(path,label), branch)`, bundle `{label: r.value}`, free = `concat({Path: path, Value: label}, r.free)`. Empty map → error.
- **default → error** `"$any takes a list of alternatives or a map of labeled branches"`.

Then **delete the `$oneof` case**. A stale `$oneof:` key then falls to `expandDescriptor`'s `default` → `unknown space-descriptor "$oneof"` — the correct migration signal.

**Why the free-param rule matters:** a list element resolves *at `path`* (no path segment added, unlike a map branch which adds `.label`). So a nested descriptor inside a list element writes coords at `path` too — blindly adding `{path, value}` would duplicate. The "use `s.free` if present, else `{path, value}`" rule keeps leaves clean (identical to today) and structured elements non-duplicated. (Structured list elements have inherently murky "which alternative" coords — that's *why* the map/tagged form exists; the list form is for simple values.)

### Entities

| Name | Lives in | Status |
|------|----------|--------|
| `expandDescriptor` (`$any` case) | `pkg/shape/shape.go` | modified |
| `expandDescriptor` (`$oneof` case) | `pkg/shape/shape.go` | deleted |

- **expandDescriptor `$any`** — gains the list/map type-switch + list recursion; absorbs `$oneof`'s map logic.
  - **DRY rationale:** one choice primitive instead of two; the tagged-sum logic lives once (moved, not copied).
  - **Test surface:** `pkg/shape/shape_test.go` — colocated, no IO.

No new files, no integration points (pure algebra change).

### Test surface (RED-first)

1. **Golden equivalence** — `$any:{map}` produces byte-identical points (values + free-params) to what `$oneof:{map}` produced. Migrate the existing `$oneof` tests (`shape_test.go:27`, `:107`, the empty-error `:233`) to `$any` map form; they must still pass with the same assertions (the 36-point ADD test, the bundling, the empty→error).
2. **List recursion (new capability)** — `$any:[{$linear-range:[0,1,3]}, 9]` expands the nested range → 4 points (0, 0.5, 1, 9); free-params decompose (no duplicate coord at `path`).
3. **List backward-compat** — `$any:[logreg, rf]` (scalars) and `$any:[[], [title], [title,family]]` (lists) unchanged: bare values, coord `{path, value}`.
4. **Bad arg** — `$any: 5` (scalar) → the "list or map" error.
5. **`$oneof` removed** — a shape using `$oneof:` errors with `unknown space-descriptor "$oneof"`.

---

## Task 1: Engine — `$any` dispatches on list/map; delete `$oneof`

**Files:** Modify `pkg/shape/shape.go` (the `expandDescriptor` switch). Test `pkg/shape/shape_test.go`.

- [ ] **Step 1: Migrate the existing `$oneof` tests → `$any` map form (RED via rename).** In `shape_test.go`, replace the three `"$oneof"` literals (`:27`, `:107`, `:233`) with `"$any"`. Run `go test ./pkg/shape/` → the **two value-asserting** ones fail (`:27` 36-point ADD, `:107` ragged → `$any` doesn't handle maps yet → "$any takes a list of alternatives"). *(The `:233` empty-map test stays green — it only asserts `err != nil`, and `$any` on a map still errors pre-GREEN; that's fine, it's still a valid regression.)*

- [ ] **Step 2: Add the new-capability + bad-arg tests (RED).** Append to `shape_test.go`:

```go
// $any over a LIST recurses into each alternative (the new uniform recursion): a
// nested descriptor inside an element expands, and coords decompose without a
// duplicate at the element's own path.
func TestExpandAnyList_RecursesIntoElements(t *testing.T) {
	steps := []experiment.Step{{ID: "s", With: map[string]any{
		"lr": map[string]any{"$any": []any{
			map[string]any{"$linear-range": []any{0.0, 1.0, 3}}, // → 0, 0.5, 1
			9.0,
		}},
	}}}
	pts, err := Expand(steps, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 4 { // 3 from the range + 1 scalar
		t.Fatalf("got %d points, want 4 (range(3)+scalar)", len(pts))
	}
	for _, p := range pts { // exactly one free-param coord at "s.lr", never duplicated
		n := 0
		for _, fp := range p.FreeParams {
			if fp.Path == "s.lr" {
				n++
			}
		}
		if n != 1 {
			t.Errorf("point %v has %d s.lr coords, want exactly 1", p.With, n)
		}
	}
}

// $any rejects a non-list, non-map argument.
func TestExpandAny_BadArg(t *testing.T) {
	_, err := Expand([]experiment.Step{{ID: "s", With: map[string]any{
		"x": map[string]any{"$any": 5},
	}}}, 5)
	if err == nil {
		t.Fatal("$any with a scalar arg must error (want: list or map)")
	}
}

// A stale $oneof is a clear error after the merge (migration signal).
func TestOneofRemoved(t *testing.T) {
	_, err := Expand([]experiment.Step{{ID: "s", With: map[string]any{
		"x": map[string]any{"$oneof": map[string]any{"a": 1}},
	}}}, 5)
	if err == nil || !strings.Contains(err.Error(), "$oneof") {
		t.Fatalf("stale $oneof should error as unknown descriptor; got %v", err)
	}
}
```

(Add `strings` to the test imports if absent.)

- [ ] **Step 3: Run to confirm RED** — `go test ./pkg/shape/ -run 'TestExpandAny|TestOneof|TestExpand' -v` → the migrated map tests + new tests fail; `$oneof` test (if any remain) still passes (not yet deleted).

- [ ] **Step 4: GREEN — rewrite the `$any` case + delete `$oneof`.** In `expandDescriptor` (`shape.go:133`):

```go
case "$any":
	switch a := arg.(type) {
	case []any: // untagged sum — recurse per element, value bare
		if len(a) == 0 {
			return nil, fmt.Errorf("%s: $any is empty — an empty set would collapse the whole sweep to zero points", path)
		}
		var out []resolved
		for _, alt := range a {
			subs, err := expandValue(path, alt, rangeSteps)
			if err != nil {
				return nil, err
			}
			for _, s := range subs {
				free := s.free
				if len(free) == 0 { // leaf alternative → its value is the coord
					free = []FreeParam{{Path: path, Value: s.value}}
				}
				out = append(out, resolved{value: s.value, free: free})
			}
		}
		return out, nil
	case map[string]any: // tagged sum — the former $oneof: recurse per branch, bundle {label: sub}
		if len(a) == 0 {
			return nil, fmt.Errorf("%s: $any map has no branches — would collapse the whole sweep to zero points", path)
		}
		var out []resolved
		for _, label := range sortedKeys(a) {
			sub, err := expandValue(join(path, label), a[label], rangeSteps)
			if err != nil {
				return nil, err
			}
			for _, r := range sub {
				bundled := map[string]any{label: r.value}
				free := concat([]FreeParam{{Path: path, Value: label}}, r.free)
				out = append(out, resolved{value: bundled, free: free})
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s: $any takes a list of alternatives or a map of labeled branches", path)
	}
```

Delete the entire `case "$oneof":` block. Keep `$linear-range`/`$log-range` and the `default`.

- [ ] **Step 4b: Update `shape.go`'s OWN doc comments** (they describe the deleted semantics). Rewrite the package-doc algebra block (lines ~8-16): drop the separate `$any` "VERBATIM … unlike $oneof" wording + the `$oneof` bullet; state the unified model — `$any:[…]` = untagged sum, each alternative **recursively expanded**, value bare; `$any:{L:sub}` = tagged sum, each branch recursively expanded, bundled `{label: sub}` (counts ADD). Fix the `FreeParam` doc (line ~33): "a `$oneof` branch label" → "a `$any`-map branch label".

- [ ] **Step 4c: Migrate the committed testdata shape IN THIS COMMIT** (a `cmd/` consumer breaks otherwise — see below). `testdata/experiment/titanic-baseline-shape.md` (~:27): `$oneof:` → `$any:` (map form, byte-identical → still 21 points). Also clean the non-literal `$oneof` text in `shape_test.go`: the comments at `:17` ("Proves $oneof ADDs" → "$any-map ADDs") and `:225`, and the error-message string at `:235` ("empty $oneof must error" → "empty $any map must error").

- [ ] **Step 5: GREEN — run the WHOLE module** `go test ./...` → all pass. **Not just `./pkg/shape/`**: `cmd/metis/shape_e2e_test.go` reads `titanic-baseline-shape.md` and asserts 21 points, so it goes red the moment `$oneof` is deleted unless 4c migrated it — a scoped shape-only test would false-green. `go vet ./...` clean.

- [ ] **Step 6: Commit** — `#17: $any subsumes $oneof — list=untagged / map=tagged, both recursive` (engine + shape.go docs + testdata + shape_test together = one green commit; never commit a `$oneof`-deleting change with a red `go test ./...`).

---

## Task 2: Sweep every `$oneof` reference in metis (tests, testdata, docs, CUE comments)

**Files (from the grep):** `pkg/ledger/ledger_test.go`, `construct/datatype/experiment-shape.md`, `atlas/experiment.md`, `atlas/index.md`, `construct/vocabulary/experiment.cue`, `workshop/lessons.md`. *(testdata shape + shape.go/shape_test.go comments are in Task 1 — they gate the build.)*

- [ ] **Step 1:** *(`testdata/experiment/titanic-baseline-shape.md` + `shape.go`/`shape_test.go` comments already migrated in Task 1 — they gate the build, so they can't be deferred here.)*

- [ ] **Step 2: `pkg/ledger/ledger_test.go`** — the `:36` `$oneof` reference: if it's a code fixture, rename `$oneof`→`$any` (map); if a comment, update wording. Run `go test ./pkg/ledger/`.

- [ ] **Step 3: `construct/datatype/experiment-shape.md`** — rewrite the grammar prose (lines 4, 57-65): one primitive `$any`; **list = untagged (bare), each alternative recursively expanded** (fix the `:57` "each value taken **verbatim**" wording — it's recursive now), **map = tagged (bundled `{label: sub}`, conditional/ADD)**, both recursive. Drop the separate `$oneof` bullet; keep the 36-point ADD-vs-multiply example but under the `$any` map form.

- [ ] **Step 4: `atlas/experiment.md` + `atlas/index.md`** — reconcile the shape-algebra description to the one-primitive model (list=untagged/bare, map=tagged/bundled, recursive); remove `$oneof` as a distinct construct. Include the "why keep the tagged form" one-liner (readability + adaptive-sampler legibility, cf. metis#7).

- [ ] **Step 5: `construct/vocabulary/experiment.cue`** — update the two comments (`:47`, `:56`) `$any/$oneof` → `$any` (note list/map forms).

- [ ] **Step 6: `workshop/lessons.md`** — the metis#12 lesson references `$oneof`; add a parenthetical "(now `$any` map form, metis#17)" rather than rewrite history.

- [ ] **Step 6b: The Python data-plane references (comments / error-string / test names — NOT config keys, so no behavior change; the Python reads the resolved `{rf:{…}}` bundle, unchanged).** Sweep `$oneof`→`$any` (map form):
  - `metis/model.py` — comments `:24`/`:44` **and the error-message string `:37`** (`'$oneof bundle (…)'` → `'$any-map bundle (…)'`).
  - `metis/steps/train.py` — comments `:10`/`:34`.
  - `tests/test_model.py:69` — docstring.
  - `tests/test_steps.py` — the test **name** `test_train_step_accepts_oneof_model_config` → `…accepts_any_map_model_config` + its docstring (`:61-64`). Run `uv run pytest tests/test_model.py tests/test_steps.py` (skip-guard if uv absent).
  - *(kbench's `e2e/thread_test.py` `$oneof` docstrings are **kbench#7**'s scope — cross-repo.)*

- [ ] **Step 7: Grep-confirm + full suite** — `grep -rn 'oneof' --include=*.go --include=*.cue --include=*.md --include=*.py . | grep -v workshop/history` (**note `--include=*.py`** — the completeness grep must be able to catch a Python occurrence) shows only intentional historical/"was $oneof" mentions. `go test ./...` all green; `uv run pytest` green (or skip if no uv); `go vet ./...` clean.

- [ ] **Step 8: Commit** — `#17: sweep $oneof → $any across tests, testdata, datatype, atlas, cue`.

---

## Task 3: Cross-repo rollout + close

- [x] **Step 1: Verified kbench against this branch BEFORE merging.** Implemented **kbench#7** (migrated `$oneof:`→`$any:` in `titanic-sweep.md` + `titanic-sweep-smoke.md`) and ran `pytest e2e/thread_test.py::test_sweep_smoke_composes_and_trains` against this metis branch checkout → **1 passed** (both-forms anchor: `features` list + `model` map → metis/train). Integrated state proven before either merge — no broken window.

- [x] **Step 2: Close metis#17** — closed (Review-Verdict SHIP, actual 0.71). Then merge metis, then merge kbench#7 (back-to-back).

- [x] **Step 3:** atlas + index reconciled in Task 2; the kbench-side atlas/prose is kbench#7's scope (done in kbench#7).

---

## Decisions

- **One keyword, dispatch on arg shape (list=untagged / map=tagged).** The syntax already signals tagged-vs-untagged (list literal vs map literal); a second keyword is redundant. Matches the type-theory (untagged vs tagged sum) and the ecosystem split (sklearn/hyperopt list-of-bags vs Hydra/Ax tagged).
- **Keep the tagged (map) form — don't flatten to list-of-bags-with-discriminator.** External tagging reads better AND is legible to adaptive samplers (conditional/hierarchical BO), which metis#7's `Sampler` seam exists to accept. So the tag is structure, not sugar.
- **List form gains recursion (uniform), but it's a no-op for existing shapes.** All current `$any` elements are scalars/plain-lists (no nested `$`-descriptor), so recursion changes nothing today; it only adds capability. `$`-keys are reserved, so a real feature list can't collide with a descriptor.
- **Free-param for a recursive list element:** use the element's own coords if it produced any (structured element), else `{path, value}` (leaf). Avoids double-recording at `path`; structured-list coords are inherently murky (that's what the map form is for).
- **CUE needs no schema change** — the value-algebra is untyped (`with` bag, value-level), so `$oneof` lived only in comments, not a closed enum.
