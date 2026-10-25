# Sourced (not executable, so run-merge-checks skips it) by the experiment checks.
# Resolves the vocabulary tool or stops the check: a missing tool must never read
# as a schema verdict (metis#70, where CI's missing PATH entry posed as a rejection).
VOCAB="${VOCAB:-vocabulary}"
if ! command -v "$VOCAB" >/dev/null 2>&1; then
  echo "$(basename "$0"): vocabulary tool '$VOCAB' not found; put the layer tool dirs on PATH or set VOCAB" >&2
  exit 2
fi
