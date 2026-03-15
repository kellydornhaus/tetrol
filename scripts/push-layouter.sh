#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PREFIX="external/layouter"
REMOTE="layouter"
BRANCH="${1:-main}"

cd "$ROOT_DIR"

if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "[layouter] working tree is dirty; commit or stash before pushing" >&2
    exit 1
fi

echo "[layouter] pushing ${PREFIX} to ${REMOTE}/${BRANCH} via git subtree"
git subtree push --prefix="$PREFIX" "$REMOTE" "$BRANCH"
