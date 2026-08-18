#!/usr/bin/env bash
# Validates SSMR e2e log output contains all required ecosystem PX_E2E markers.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if [[ -n "${SANDBOX_MARKER_MANIFEST:-}" ]]; then
	MANIFEST="$SANDBOX_MARKER_MANIFEST"
elif [[ -n "${SSMR_MARKER_MANIFEST:-}" ]]; then
	MANIFEST="$SSMR_MARKER_MANIFEST"
elif [[ -f "$ROOT/contracts/sandbox_ecosystem_markers.json" ]]; then
	MANIFEST="$ROOT/contracts/sandbox_ecosystem_markers.json"
else
	MANIFEST="$ROOT/contracts/ssmr_ecosystem_markers.json"
fi
LOG_FILE="${1:-}"

if [[ -z "$LOG_FILE" || ! -f "$LOG_FILE" ]]; then
	echo "usage: $0 <sandbox-e2e.log>" >&2
	echo "sandbox-ecosystem-marker-gate: missing log file" >&2
	exit 1
fi
if [[ ! -f "$MANIFEST" ]]; then
	echo "ssmr-ecosystem-marker-gate: missing manifest $MANIFEST" >&2
	exit 1
fi
if ! command -v python3 >/dev/null 2>&1; then
	echo "ssmr-ecosystem-marker-gate: python3 required" >&2
	exit 1
fi

python3 - "$MANIFEST" "$LOG_FILE" <<'PY'
import json
import sys
from pathlib import Path

manifest_path = Path(sys.argv[1])
log_path = Path(sys.argv[2])
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
log_text = log_path.read_text(encoding="utf-8", errors="replace")

missing = []
for marker in manifest.get("required", []):
    needle = marker.strip()
    if needle and needle not in log_text:
        missing.append(needle)

for group, options in manifest.get("alternatives", {}).items():
    if any(opt in log_text for opt in options):
        continue
    missing.append(f"alternatives:{group} ({' | '.join(options)})")

if missing:
    print("sandbox-ecosystem-marker-gate-FAIL", file=sys.stderr)
    print("missing markers:", file=sys.stderr)
    for item in missing:
        print(f"  - {item}", file=sys.stderr)
    sys.exit(1)

print("sandbox-ecosystem-marker-gate-ok")
PY
