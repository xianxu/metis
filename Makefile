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
# test against a throwaway copy of go.mod (+ go.sum) whose replace points there. A slot
# or clone keeps ../ariadne and the real go.mod.
ifeq ($(wildcard ../ariadne/go.mod),)
ifdef SDLC_DEP_SUBSTRATE
TEST_MODFILE := $(or $(TMPDIR),/tmp)/metis-test.mod
TEST_GOFLAGS := -modfile=$(TEST_MODFILE)
endif
endif

.PHONY: test test-deps
test:
ifdef TEST_MODFILE
	cp go.mod "$(TEST_MODFILE)" && cp go.sum "$(basename $(TEST_MODFILE)).sum"
	go mod edit -modfile="$(TEST_MODFILE)" -replace=github.com/xianxu/ariadne="$(SDLC_DEP_SUBSTRATE)"
endif
	UV_OFFLINE=1 go test $(TEST_GOFLAGS) ./...
	UV_OFFLINE=1 uv run --offline pytest -q tests

test-deps:
	uv sync
