#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PREFIX="external/layouter"
REMOTE="layouter"
BRANCH="${1:-main}"

cd "$ROOT_DIR"

if ! git diff --quiet -- "$PREFIX" || ! git diff --cached --quiet -- "$PREFIX"; then
    echo "[layouter] working tree has layouter changes; commit or stash before pulling" >&2
    exit 1
fi

echo "[layouter] fetching ${REMOTE}/${BRANCH}"
git fetch "$REMOTE" "$BRANCH"

echo "[layouter] merging ${REMOTE}/${BRANCH} into ${PREFIX}"
git merge --allow-unrelated-histories -Xsubtree="$PREFIX" "$REMOTE/$BRANCH"
