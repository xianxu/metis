#!/usr/bin/env bash
# ci-setup — the seeded merge-check's post-compile hook (metis#70). Compile builds
# each layer's tools into <layer>/bin and only prints them as "add these layer tool
# directories to your shell PATH"; merge-checks then call them by name (vocabulary).
# Put those directories on PATH for the later workflow steps. Remove this hook once
# the seeded workflow does it itself (ariadne#627).
set -euo pipefail
cd "$(cd "$(dirname "$0")/.." && pwd -P)"
: "${GITHUB_PATH:?ci-setup: runs only inside GitHub Actions}"
for dir in ../ariadne/bin bin; do
  [ -d "$dir" ] || { echo "ci-setup: layer tool directory $dir missing after compile" >&2; exit 1; }
  abs="$(cd "$dir" && pwd -P)"
  echo "ci-setup: PATH += $abs"
  printf '%s\n' "$abs" >> "$GITHUB_PATH"
done
