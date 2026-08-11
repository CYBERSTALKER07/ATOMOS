#!/usr/bin/env bash
# Blocks AUTO_ORDER_PLACE flip unless evidence artifacts exist.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [[ "${AUTO_ORDER_PLACE_ENABLED:-}" == "true" || "${AUTO_ORDER_PLACE_ENABLED:-}" == "1" ]]; then
  if [[ ! -f artifacts/forecast-shadow/acceptance-30d.json ]]; then
    echo "place-flip-blocked: missing artifacts/forecast-shadow/acceptance-30d.json" >&2
    echo "See docs/AUTO_ORDER_PLACE_FLIP.md" >&2
    exit 1
  fi
  python3 - <<'PY'
import json,sys
from pathlib import Path
p=Path("artifacts/forecast-shadow/acceptance-30d.json")
data=json.loads(p.read_text())
rate=float(data.get("unmodified_acceptance_rate",0))
days=int(data.get("soak_days",0))
if days < 30 or rate < 0.80:
    print(f"place-flip-blocked: soak_days={days} acceptance={rate}", file=sys.stderr)
    sys.exit(1)
print(f"place-flip-ok: soak_days={days} acceptance={rate}")
PY
else
  echo "place-flip-check-skipped: AUTO_ORDER_PLACE_ENABLED not true"
fi
