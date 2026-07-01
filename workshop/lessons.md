# metis — lessons

Rules distilled from work in metis, to prevent repeats (AGENTS.md §4).

## Go / build
- **Offline Go module bring-up.** Before assuming network is needed for a new `go.mod`, check `$(go env GOMODCACHE)/cache/download/...` for the dep **and its transitive `go.mod` graph** (a pre-1.17 dep like `gopkg.in/yaml.v3` pulls `check.v1`'s go.mod into the unpruned graph). If present, `GOPROXY=off go mod tidy` builds `go.sum` with zero network — no sandbox override. (metis#1 M2)

## Testing
- **External-binary drift guards.** To stop Go structs drifting from a CUE/schema single source, add a test that shells the sibling validator (e.g. `vocabulary validate-instance`) on a fixture the structs also parse; `t.Skip` when the binary/toolchain is absent so bare checkouts stay green while the guard runs wherever the tool exists. (metis#1 M2)
- **e2e tests that mutate fixtures copy them into `t.TempDir()` first.** The step-runner appends `## Runs` and writes `runs/`; running against committed `testdata/` would dirty the tree. Verify clean with `git status`. (metis#1 M2)
- **Absolute-path fixtures can mask relative-path bugs.** An e2e that fed the runner an absolute `t.TempDir()` path passed green while the natural `metis run <relative-path>` invocation was broken (relative env paths double-joined into `<dir>/<dir>/…`). Exercise the *natural* invocation (chdir + bare filename), not just the convenient absolute one — the boundary review caught this class of bug; `go test` alone did not. (metis#1 M2)

## Workflow
- **Fresh weave-bootstrapped derivatives need a `construct/base.manifest`.** Without one, the transitive walk stops at the manifest-less repo and a downstream consumer silently under-compiles (only the gitignore action, no error). Author a minimal `internal prose AGENTS.local.md` manifest per new derivative. Tooling fix tracked in `ariadne#155`. (metis#1 M1)
