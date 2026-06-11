#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKEND_DIR="$ROOT/apps/backend-go"
SCHEMA_PATH="$ROOT/contracts/events.schema.json"
TMP_SCHEMA="$(mktemp)"

cd "$BACKEND_DIR"
go run ./cmd/gen-contracts \
  -source events/events.go \
  -mode json-schema \
  -schema-out "$TMP_SCHEMA" \
  -pretty=true

if ! diff -q "$TMP_SCHEMA" "$SCHEMA_PATH" >/dev/null 2>&1; then
  echo "gen-contracts gate failed: $SCHEMA_PATH is out of date" >&2
  echo "Run: cd apps/backend-go && go run ./cmd/gen-contracts -source events/events.go -mode json-schema -schema-out ../../contracts/events.schema.json -pretty=true" >&2
  diff -u "$SCHEMA_PATH" "$TMP_SCHEMA" || true
  rm -f "$TMP_SCHEMA"
  exit 1
fi

rm -f "$TMP_SCHEMA"
echo "gen-contracts gate: OK"
