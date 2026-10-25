# Canonical repo name from git remote (portable across worktrees and containers)
REPO_NAME := $(shell git remote get-url origin 2>/dev/null | sed 's|.*/||; s|\.git$$||')

# This project nests issues and history under workshop/
WF_ISSUES_DIR = workshop/issues
WF_HISTORY_DIR = workshop/history

.DEFAULT_GOAL := help
-include Makefile.workflow
-include Makefile.local

.PHONY: help
help: $(WF_HELP_TARGETS)
	@true

# This layer's tools, built by `weave compile` (foundation first) and put on
# dependents' PATH: kaggle/kbench run `metis` from here.
.PHONY: tools metis-build
tools: metis-build

metis-build:
	@mkdir -p bin
	go build -o bin/metis ./cmd/metis

# Full suite (metis#75): Go + Python, what `sdlc local-check run --name suite` runs.
# Offline by contract: UV_OFFLINE reaches uv here and in the step subprocesses the Go
# tests spawn, so a cold checkout builds its venv from the warm uv cache and a missing
# package fails loudly instead of waiting on the network. Warm the cache once, online,
# with `make test-deps`.
# go.mod replaces ariadne with ../ariadne, which a local-check checkout (sdlc-local-*)
# lacks; sdlc exports SDLC_DEP_SUBSTRATE (the caller's ariadne, from construct/deps), so
# there the Go suite runs against a private, per-run copy of go.mod (+ go.sum) whose
# replace points at it, removed on exit. A slot or clone keeps ../ariadne and the real go.mod.
.PHONY: test test-deps
test:
	@set -e; modflag=; \
	if [ ! -f ../ariadne/go.mod ] && [ -n "$$SDLC_DEP_SUBSTRATE" ]; then \
	  d=$$(mktemp -d "$${TMPDIR:-/tmp}/metis-test.XXXXXX"); trap 'rm -rf "$$d"' EXIT; \
	  cp go.mod "$$d/test.mod"; cp go.sum "$$d/test.sum"; \
	  go mod edit -modfile="$$d/test.mod" -replace=github.com/xianxu/ariadne="$$SDLC_DEP_SUBSTRATE"; \
	  modflag="-modfile=$$d/test.mod"; echo "make test: ariadne => $$SDLC_DEP_SUBSTRATE"; \
	fi; \
	echo "UV_OFFLINE=1 go test $$modflag ./..."; UV_OFFLINE=1 go test $${modflag:+"$$modflag"} ./...
	UV_OFFLINE=1 uv run --offline pytest -q tests

test-deps:
	uv sync
