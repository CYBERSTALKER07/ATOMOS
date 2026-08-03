#!/usr/bin/env bash
# Run Next static export for Tauri, temporarily excluding app/api (unsupported by output:export).
set -euo pipefail

APP_DIR="${1:-}"
if [[ -z "$APP_DIR" || ! -d "$APP_DIR" ]]; then
  echo "Usage: $0 <app-directory>" >&2
  exit 1
fi

cd "$APP_DIR"
API_DIR="app/api"
STASH=""
cleanup() {
  if [[ -n "$STASH" && -d "$STASH" ]]; then
    rm -rf "$API_DIR"
    mv "$STASH" "$API_DIR"
  fi
}
trap cleanup EXIT

if [[ -d "$API_DIR" ]] && compgen -G "$API_DIR/**/*" >/dev/null 2>&1; then
  STASH="$(mktemp -d)"
  # move contents via renaming the folder
  mv "$API_DIR" "$STASH/api"
  mkdir -p "$API_DIR"
fi

export TAURI_BUILD=1
if command -v pnpm >/dev/null 2>&1; then
  pnpm exec next build
else
  npx --no-install next build
fi
