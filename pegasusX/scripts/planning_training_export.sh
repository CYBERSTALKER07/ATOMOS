#!/usr/bin/env bash
# PX-PROD-3: export DemandForecastBaseline rows for offline ML training (collect-only).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/apps/backend-go"

OUT="${1:-}"
ARGS=(./cmd/planning-training-export -days "${PLANNING_EXPORT_DAYS:-30}" -format "${PLANNING_EXPORT_FORMAT:-jsonl}")
if [[ -n "${PLANNING_EXPORT_SUPPLIER_ID:-}" ]]; then
  ARGS+=(-supplier-id "$PLANNING_EXPORT_SUPPLIER_ID")
fi
if [[ -n "$OUT" ]]; then
  ARGS+=(-out "$OUT")
fi

go run "${ARGS[@]}"
