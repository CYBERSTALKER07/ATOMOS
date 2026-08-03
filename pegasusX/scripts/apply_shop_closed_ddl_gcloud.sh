#!/usr/bin/env bash
# Apply 20260729 shop-closed DDL via gcloud (ADC-friendly) + INFORMATION_SCHEMA verify.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT="${SPANNER_PROJECT:-pegasus-503013}"
INSTANCE="${SPANNER_INSTANCE:-pegasusx-ssmr-spanner}"
DATABASE="${SPANNER_DATABASE:-pegasusx-ssmr-db}"
DDL="${DDL_FILE:-$ROOT/apps/backend-go/schema/migrations/20260729_shop_closed_proximity_partial.ddl}"

if [[ ! -f "$DDL" ]]; then
	echo "missing DDL: $DDL" >&2
	exit 1
fi

echo "==> target projects/${PROJECT}/instances/${INSTANCE}/databases/${DATABASE}"
echo "==> ddl $DDL"

while IFS= read -r stmt; do
	[[ -z "$stmt" ]] && continue
	echo "==> $(echo "$stmt" | cut -c1-100)…"
	if out=$(gcloud spanner databases ddl update "$DATABASE" \
		--instance="$INSTANCE" \
		--project="$PROJECT" \
		--ddl="$stmt" 2>&1); then
		echo "  OK"
	else
		if grep -qiE 'Duplicate|Already exists|already exists' <<<"$out"; then
			echo "  SKIP (already applied)"
		else
			echo "$out" >&2
			echo "  ERR" >&2
			exit 1
		fi
	fi
done < <(python3 - "$DDL" <<'PY'
import sys
path = sys.argv[1]
lines = []
for line in open(path, encoding="utf-8"):
    s = line.strip()
    if not s or s.startswith("--"):
        continue
    if "--" in line:
        line = line.split("--", 1)[0]
    lines.append(line)
body = "\n".join(lines)
for part in body.split(";"):
    one = " ".join(part.split())
    if one:
        print(one)
PY
)

echo "==> verify Orders columns"
VERIFY=$(gcloud spanner databases execute-sql "$DATABASE" \
	--instance="$INSTANCE" --project="$PROJECT" \
	--sql="SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='Orders' AND COLUMN_NAME IN UNNEST(['ShopClosedAt','ShopClosedGraceEndsAt','PartialDelivery','ProximityUnlockedAt','ProximityMethod','ShopClosedReason','ShopClosedResolution']) ORDER BY COLUMN_NAME")
echo "$VERIFY"

for need in ShopClosedAt ShopClosedGraceEndsAt PartialDelivery ProximityUnlockedAt ProximityMethod ShopClosedReason ShopClosedResolution; do
	if ! grep -q "$need" <<<"$VERIFY"; then
		echo "schema-drift-FAIL: missing Orders.$need" >&2
		exit 1
	fi
done

TABLE=$(gcloud spanner databases execute-sql "$DATABASE" \
	--instance="$INSTANCE" --project="$PROJECT" \
	--sql="SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME='OrderShopClosedLog'")
echo "$TABLE"
if ! grep -q OrderShopClosedLog <<<"$TABLE"; then
	echo "schema-drift-FAIL: missing OrderShopClosedLog" >&2
	exit 1
fi

echo "shop-closed-ddl-gcloud-ok projects/${PROJECT}/instances/${INSTANCE}/databases/${DATABASE}"
