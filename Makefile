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
.PHONY: test test-deps
test:
	UV_OFFLINE=1 go test ./...
	UV_OFFLINE=1 uv run --offline pytest -q tests

test-deps:
	uv sync
