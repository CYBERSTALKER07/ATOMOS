#!/usr/bin/env bash
# Writes artifacts/forecast-shadow/acceptance-30d.json for the place-flip gate.
#
# Modes:
#   1) From live API (preferred): RETAILER_BEARER + API_BASE
#        GET /v1/retailer/settings/auto-order/soak-artifact
#   2) From a pre-downloaded soak JSON file: SOAK_ARTIFACT_PATH=...
#   3) Dry schema stub for CI wiring (not evidence): STUB=1
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
OUT_DIR="artifacts/forecast-shadow"
OUT_FILE="$OUT_DIR/acceptance-30d.json"
mkdir -p "$OUT_DIR"

if [[ "${STUB:-}" == "1" ]]; then
  cat >"$OUT_FILE" <<'EOF'
{
  "artifact": "auto-order-soak",
  "version": 1,
  "soak_days": 0,
  "window_days": 30,
  "unmodified_accept_rate": 0,
  "unmodified_acceptance_rate": 0,
  "note": "STUB only — not flip evidence. Replace via soak-artifact API download."
}
EOF
  echo "wrote stub $OUT_FILE (not flip evidence)"
  exit 0
fi

if [[ -n "${SOAK_ARTIFACT_PATH:-}" ]]; then
  python3 - <<PY
import json,sys
from pathlib import Path
src=Path("${SOAK_ARTIFACT_PATH}")
data=json.loads(src.read_text())
stats=(data.get("decision") or {}).get("stats") or {}
rate=float(stats.get("unmodified_accept_rate", data.get("unmodified_accept_rate", 0)))
days=int(data.get("window_days", data.get("soak_days", 30)))
out={
  "artifact": data.get("artifact", "auto-order-soak"),
  "version": data.get("version", 1),
  "retailer_id": data.get("retailer_id"),
  "generated_at": data.get("generated_at"),
  "soak_days": days,
  "window_days": days,
  "unmodified_accept_rate": rate,
  "unmodified_acceptance_rate": rate,
  "decision": data.get("decision"),
  "thresholds": data.get("thresholds"),
}
Path("${OUT_FILE}").write_text(json.dumps(out, indent=2) + "\n")
print(f"wrote ${OUT_FILE} soak_days={days} acceptance={rate}")
PY
  exit 0
fi

API_BASE="${API_BASE:-${NEXT_PUBLIC_API_URL:-http://localhost:8180}}"
API_BASE="${API_BASE%/}"
if [[ -z "${RETAILER_BEARER:-}" ]]; then
  echo "usage: RETAILER_BEARER=… [API_BASE=…] $0" >&2
  echo "   or: SOAK_ARTIFACT_PATH=path/to/soak.json $0" >&2
  echo "   or: STUB=1 $0" >&2
  exit 2
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
curl -fsS -H "Authorization: Bearer ${RETAILER_BEARER}" \
  "${API_BASE}/v1/retailer/settings/auto-order/soak-artifact" >"$tmp"

SOAK_ARTIFACT_PATH="$tmp" "$0"
