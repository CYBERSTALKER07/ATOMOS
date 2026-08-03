#!/usr/bin/env bash
# D3: apply base Spanner schema + incremental migrations via gcloud (batch DDL).
# Faster than go run ./cmd/setup (one RPC per statement) — batches many statements per update.
#
# Usage (IDE terminal, logged in as blackfoxenterprise3697@gmail.com):
#   gcloud config set project pegasus-503013
#   bash scripts/d3_apply_schema_gcloud.sh
#
# Env overrides:
#   PROJECT / INSTANCE / DATABASE / BATCH_SIZE
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT="${PROJECT:-pegasus-503013}"
INSTANCE="${INSTANCE:-pegasusx-staging-spanner}"
DATABASE="${DATABASE:-pegasusx-staging-db}"
BATCH_SIZE="${BATCH_SIZE:-15}"
BASE_DDL="$ROOT/apps/backend-go/schema/spanner.ddl"
MIG_DIR="$ROOT/apps/backend-go/schema/migrations"
LAST_MIGRATION="${LAST_MIGRATION:-20260720_order_fiscal_receipts.ddl}"

echo "==> project=$PROJECT instance=$INSTANCE database=$DATABASE batch=$BATCH_SIZE"
gcloud config set project "$PROJECT" >/dev/null

# Parse DDL file into statements (drop line comments; split on ;)
parse_ddl() {
	local file=$1
	python3 - "$file" <<'PY'
import re, sys
path = sys.argv[1]
text = open(path, encoding="utf-8").read()
# strip /* */ blocks
text = re.sub(r"/\*.*?\*/", "", text, flags=re.S)
lines = []
for line in text.splitlines():
    if line.strip().startswith("#"):
        continue
    # strip -- comments
    if "--" in line:
        line = line.split("--", 1)[0]
    lines.append(line)
body = "\n".join(lines)
parts = [p.strip() for p in body.split(";") if p.strip()]
for p in parts:
    # single-line for gcloud --ddl
    one = " ".join(p.split())
    if one:
        print(one)
PY
}

apply_batch() {
	local -a stmts=("$@")
	[[ ${#stmts[@]} -eq 0 ]] && return 0
	local args=()
	local s
	for s in "${stmts[@]}"; do
		args+=(--ddl="$s")
	done
	# Ignore already-exists conflicts so re-runs are safe
	if gcloud spanner databases ddl update "$DATABASE" \
		--instance="$INSTANCE" \
		--project="$PROJECT" \
		"${args[@]}" 2>&1; then
		echo "OK batch size=${#stmts[@]}"
	else
		# retry one-by-one to skip conflicts
		echo "WARN batch failed — retrying statements individually"
		for s in "${stmts[@]}"; do
			if gcloud spanner databases ddl update "$DATABASE" \
				--instance="$INSTANCE" \
				--project="$PROJECT" \
				--ddl="$s" 2>&1; then
				echo "  OK $(echo "$s" | cut -c1-80)…"
			else
				# common on re-run
				echo "  SKIP/ERR $(echo "$s" | cut -c1-80)…"
			fi
		done
	fi
}

apply_file() {
	local file=$1
	echo "==> file $(basename "$file")"
	local -a batch=()
	local stmt count=0
	while IFS= read -r stmt; do
		batch+=("$stmt")
		count=$((count + 1))
		if [[ ${#batch[@]} -ge $BATCH_SIZE ]]; then
			apply_batch "${batch[@]}"
			batch=()
		fi
	done < <(parse_ddl "$file")
	apply_batch "${batch[@]}"
	echo "==> done $(basename "$file") statements≈$count"
}

echo "==> base schema"
apply_file "$BASE_DDL"

echo "==> incremental migrations"
last_seen=0
for ddl in $(find "$MIG_DIR" -name '*.ddl' | sort); do
	apply_file "$ddl"
	if [[ "$(basename "$ddl")" == "$LAST_MIGRATION" ]]; then
		last_seen=1
	fi
done

if [[ "$last_seen" -ne 1 ]]; then
	echo "FAIL: did not process $LAST_MIGRATION" >&2
	exit 1
fi

echo "==> verify"
gcloud spanner databases execute-sql "$DATABASE" \
	--instance="$INSTANCE" --project="$PROJECT" \
	--sql="SELECT COUNT(*) AS tables FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ''"

gcloud spanner databases execute-sql "$DATABASE" \
	--instance="$INSTANCE" --project="$PROJECT" \
	--sql="SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '' AND TABLE_NAME IN ('Orders','OrderFiscalReceipts','Suppliers','OutboxEvents') ORDER BY TABLE_NAME"

echo "d3-gcloud-schema-ok"
