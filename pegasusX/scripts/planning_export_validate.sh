#!/usr/bin/env bash
# PX-PROD-3: validate a planning training export file (post-export QA).
set -euo pipefail

FILE="${1:?usage: planning_export_validate.sh <export.jsonl|csv> [min_rows]}"
MIN_ROWS="${2:-1}"

if [[ ! -f "$FILE" ]]; then
  echo "file not found: $FILE" >&2
  exit 1
fi

python3 - "$FILE" "$MIN_ROWS" <<'PY'
import json
import csv
import sys
from pathlib import Path

path = Path(sys.argv[1])
min_rows = int(sys.argv[2])
rows = []

if path.suffix.lower() == ".csv":
    with path.open(newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            rows.append(row)
else:
    with path.open(encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rows.append(json.loads(line))

if len(rows) < min_rows:
    print(f"FAIL: row count {len(rows)} < min_rows {min_rows}", file=sys.stderr)
    sys.exit(1)

ml_rows = 0
null_baseline = 0
for row in rows:
    src = (row.get("baseline_source") or "").strip().lower()
    if src == "ml":
        ml_rows += 1
    qty = row.get("baseline_qty")
    try:
        if qty is None or int(float(qty)) <= 0:
            null_baseline += 1
    except (TypeError, ValueError):
        null_baseline += 1

if ml_rows:
    print(f"FAIL: {ml_rows} rows still have baseline_source=ml (math-only contract)", file=sys.stderr)
    sys.exit(1)

null_pct = (null_baseline / len(rows)) * 100 if rows else 0
print(
    f"OK: rows={len(rows)} null_baseline_qty={null_baseline} ({null_pct:.1f}%) ml_rows=0"
)
PY
