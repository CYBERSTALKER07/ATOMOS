#!/usr/bin/env bash
# GS-C3: mint a NEW JWT secret for a cell. Never copies the UZ/SSMR secret.
# Writes a gitignored file. Does not upload to GSM (ops).
set -euo pipefail

CELL="${1:-}"
if [[ "$CELL" != "eu" ]]; then
  echo "usage: scripts/mint_cell_jwt.sh eu  (refuses uz — do not rotate the live cell from this catalog)" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/infra/terraform/cells/$CELL/jwt.secret"

if ! command -v openssl >/dev/null 2>&1; then
  echo "FAIL: openssl required to mint a cell JWT" >&2
  exit 1
fi

umask 077
openssl rand -base64 48 >"$OUT"
if [[ ! -s "$OUT" ]]; then
  echo "FAIL: empty JWT file" >&2
  exit 1
fi

# Refuse accidental copy of a value sitting in live staging.tfvars.
if [[ -f "$ROOT/infra/terraform/staging.tfvars" ]]; then
  uz_jwt="$(awk -F= '/^[[:space:]]*jwt_secret[[:space:]]*=/{gsub(/[" ]/, "", $2); print $2; exit}' "$ROOT/infra/terraform/staging.tfvars" || true)"
  if [[ -n "${uz_jwt:-}" && "$(cat "$OUT")" == "$uz_jwt" ]]; then
    echo "FAIL: minted JWT matched staging.tfvars — refusing UZ copy" >&2
    rm -f "$OUT"
    exit 1
  fi
fi

echo "GS-C3: minted new JWT for cell-$CELL at $OUT (gitignored; not uploaded)"
