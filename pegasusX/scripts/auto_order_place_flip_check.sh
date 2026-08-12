#!/usr/bin/env bash
# Blocks AUTO_ORDER_PLACE flip unless evidence artifacts exist.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

MIN_RATE="${AUTO_ORDER_SOAK_MIN_UNMODIFIED:-0.80}"
MIN_DAYS="${AUTO_ORDER_SOAK_MIN_DAYS:-30}"

if [[ "${AUTO_ORDER_PLACE_ENABLED:-}" == "true" || "${AUTO_ORDER_PLACE_ENABLED:-}" == "1" ]]; then
  if [[ ! -f artifacts/forecast-shadow/acceptance-30d.json ]]; then
    echo "place-flip-blocked: missing artifacts/forecast-shadow/acceptance-30d.json" >&2
    echo "Generate via: bash scripts/generate_auto_order_soak_artifact.sh" >&2
    echo "See docs/AUTO_ORDER_PLACE_FLIP.md" >&2
    exit 1
  fi
  python3 - <<PY
import json,sys,os
from pathlib import Path
p=Path("artifacts/forecast-shadow/acceptance-30d.json")
data=json.loads(p.read_text())
# Accept either schema key (runtime vs flip-check historical naming).
rate=float(data.get("unmodified_acceptance_rate",
            data.get("unmodified_accept_rate", 0)))
days=int(data.get("soak_days", data.get("window_days", 0)))
min_rate=float(os.environ.get("AUTO_ORDER_SOAK_MIN_UNMODIFIED", "${MIN_RATE}"))
min_days=int(os.environ.get("AUTO_ORDER_SOAK_MIN_DAYS", "${MIN_DAYS}"))
if days < min_days or rate < min_rate:
    print(f"place-flip-blocked: soak_days={days} acceptance={rate} (need days>={min_days} rate>={min_rate})", file=sys.stderr)
    sys.exit(1)
print(f"place-flip-ok: soak_days={days} acceptance={rate}")
PY
else
  echo "place-flip-check-skipped: AUTO_ORDER_PLACE_ENABLED not true"
fi
