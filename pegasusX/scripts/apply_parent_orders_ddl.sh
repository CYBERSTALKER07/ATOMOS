#!/usr/bin/env bash
# Apply 20260817 ParentOrders DDL (Gate 5 Phase 2) + INFORMATION_SCHEMA verify.
#
# Local SSMR emulator (default from .env.ssmr):
#   set -a && source .env.ssmr && set +a
#   bash scripts/apply_parent_orders_ddl.sh
#
# Cloud SSMR Spanner:
#   SPANNER_PROJECT=pegasus-503013 \
#   SPANNER_INSTANCE=pegasusx-ssmr-spanner \
#   SPANNER_DATABASE=pegasusx-ssmr-db \
#   bash scripts/apply_parent_orders_ddl.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ -f "$ROOT/.env.ssmr" && -z "${SPANNER_PROJECT:-}" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env.ssmr"
	set +a
fi

DDL="${DDL_FILE:-$ROOT/apps/backend-go/schema/migrations/20260817_parent_orders.ddl}"
if [[ ! -f "$DDL" ]]; then
	echo "missing DDL: $DDL" >&2
	exit 1
fi

if [[ -n "${SPANNER_EMULATOR_HOST:-}" ]]; then
	echo "==> emulator host=$SPANNER_EMULATOR_HOST"
	echo "==> apply via go run ./cmd/apply-migration"
	(
		cd "$ROOT/apps/backend-go"
		go run ./cmd/apply-migration --ddl "$DDL"
	)
else
	PROJECT="${SPANNER_PROJECT:-pegasus-503013}"
	INSTANCE="${SPANNER_INSTANCE:-pegasusx-ssmr-spanner}"
	DATABASE="${SPANNER_DATABASE:-pegasusx-ssmr-db}"
	echo "==> cloud projects/${PROJECT}/instances/${INSTANCE}/databases/${DATABASE}"
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
fi

echo "==> verify ParentOrders + Orders.ParentOrderId"
(
	cd "$ROOT/apps/backend-go"
	go run ./cmd/apply-migration --ddl "$DDL" --verify 2>/dev/null || true
)

# Prefer Spanner SQL verify when ADC/emulator client is available via apply-migration path above.
# Explicit INFORMATION_SCHEMA check for cloud; emulator path relies on apply-migration success.
if [[ -z "${SPANNER_EMULATOR_HOST:-}" ]]; then
	PROJECT="${SPANNER_PROJECT:-pegasus-503013}"
	INSTANCE="${SPANNER_INSTANCE:-pegasusx-ssmr-spanner}"
	DATABASE="${SPANNER_DATABASE:-pegasusx-ssmr-db}"
	TABLE=$(gcloud spanner databases execute-sql "$DATABASE" \
		--instance="$INSTANCE" --project="$PROJECT" \
		--sql="SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME='ParentOrders'")
	echo "$TABLE"
	if ! grep -q ParentOrders <<<"$TABLE"; then
		echo "schema-drift-FAIL: missing ParentOrders" >&2
		exit 1
	fi
	COL=$(gcloud spanner databases execute-sql "$DATABASE" \
		--instance="$INSTANCE" --project="$PROJECT" \
		--sql="SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='Orders' AND COLUMN_NAME='ParentOrderId'")
	echo "$COL"
	if ! grep -q ParentOrderId <<<"$COL"; then
		echo "schema-drift-FAIL: missing Orders.ParentOrderId" >&2
		exit 1
	fi
fi

echo "parent-orders-ddl-ok"
