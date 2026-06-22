#!/usr/bin/env bash
# Apply all incremental Spanner migrations for the SSMR sandbox (idempotent).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/apps/backend-go"

GO_BIN="${GO_BIN:-/usr/local/go/bin/go}"
if ! command -v "$GO_BIN" >/dev/null 2>&1; then
	GO_BIN="$(command -v go)"
fi

"$GO_BIN" run ./cmd/setup

BIN="${TMPDIR:-/tmp}/pegasusx-apply-migration"
"$GO_BIN" build -o "$BIN" ./cmd/apply-migration

for ddl in ./schema/migrations/*.ddl; do
	"$BIN" --ddl "$ddl"
done
