#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="$ROOT_DIR/build/web"
PORT="${PORT:-8080}"
HOST="${HOST:-127.0.0.1}"

if [[ ! -d "$BUILD_DIR" ]]; then
  echo "[serve-web] missing build output at $BUILD_DIR" >&2
  echo "[serve-web] run scripts/build-web.sh first" >&2
  exit 1
fi

echo "[serve-web] serving $BUILD_DIR at http://$HOST:$PORT/"

cd "$BUILD_DIR"
python3 -m http.server "$PORT" --bind "$HOST" &
SERVER_PID=$!

cleanup() {
  echo "[serve-web] stopping server"
  kill "$SERVER_PID" 2>/dev/null || true
}
trap cleanup EXIT

sleep 1
URL="http://$HOST:$PORT/"
if command -v open >/dev/null 2>&1; then
  open "$URL" >/dev/null 2>&1 || true
elif command -v xdg-open >/dev/null 2>&1; then
  xdg-open "$URL" >/dev/null 2>&1 || true
elif command -v start >/dev/null 2>&1; then
  start "$URL" >/dev/null 2>&1 || true
else
  echo "[serve-web] open $URL in your browser"
fi

wait "$SERVER_PID"
