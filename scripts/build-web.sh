#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUILD="$ROOT/build/web"

echo "==> Cleaning build/web/"
rm -rf "$BUILD"
mkdir -p "$BUILD"

echo "==> Compiling WASM binary"
GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w" -o "$BUILD/tetrol.wasm" "$ROOT"

# Strip WASM symbols if wasm-strip is available.
if command -v wasm-strip >/dev/null 2>&1; then
  echo "==> Stripping WASM binary"
  wasm-strip "$BUILD/tetrol.wasm"
else
  echo "    (wasm-strip not found — skipping; install wabt for smaller builds)"
fi

echo "==> Copying wasm_exec.js from Go toolchain"
GOROOT="$(go env GOROOT)"
cp "$GOROOT/lib/wasm/wasm_exec.js" "$BUILD/"

echo "==> Copying web/ static files"
cp "$ROOT/web/"* "$BUILD/"


WASM_SIZE=$(ls -lh "$BUILD/tetrol.wasm" | awk '{print $5}')
echo "==> Done!  tetrol.wasm is $WASM_SIZE"
echo "    Output: $BUILD/"
