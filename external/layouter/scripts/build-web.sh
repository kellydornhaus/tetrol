#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="$ROOT_DIR/build/web"
STATIC_DIR="$ROOT_DIR/web"
CODE_DIR="$BUILD_DIR/code"

echo "[web] rebuilding $BUILD_DIR"
if [[ -d "$BUILD_DIR" ]]; then
  chmod -R +w "$BUILD_DIR" || true
fi
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

echo "[web] copying static site"
rsync -a --exclude '.DS_Store' "$STATIC_DIR/" "$BUILD_DIR/"

echo "[web] copying code snippets"
mkdir -p "$CODE_DIR/examples/zoo"
rsync -a --prune-empty-dirs \
  --include '*/' \
  --include '*.go' \
  --include '*.xml' \
  --include '*.css' \
  --include '*.md' \
  --exclude '*' \
  "$ROOT_DIR/examples/zoo/" "$CODE_DIR/examples/zoo/"
rm -f "$CODE_DIR/examples/zoo/go.mod" "$CODE_DIR/examples/zoo/go.sum"

echo "[web] compiling wasm bundle"
cd "$ROOT_DIR/examples/zoo"
GOOS=js GOARCH=wasm go build -o "$BUILD_DIR/layouter.wasm" .

echo "[web] copying wasm_exec.js"
GO_ROOT="$(go env GOROOT)"
if [[ -z "$GO_ROOT" ]]; then
  echo "GOROOT is not set; install Go before building the web target" >&2
  exit 1
fi
if [[ -f "$GO_ROOT/misc/wasm/wasm_exec.js" ]]; then
  cp "$GO_ROOT/misc/wasm/wasm_exec.js" "$BUILD_DIR/wasm_exec.js"
elif [[ -f "$GO_ROOT/lib/wasm/wasm_exec.js" ]]; then
  cp "$GO_ROOT/lib/wasm/wasm_exec.js" "$BUILD_DIR/wasm_exec.js"
else
  echo "Unable to locate wasm_exec.js under $GO_ROOT" >&2
  exit 1
fi

echo "[web] adding static site helpers"
touch "$BUILD_DIR/.nojekyll"
cp "$BUILD_DIR/index.html" "$BUILD_DIR/404.html"

echo "[web] build complete -> $BUILD_DIR"
