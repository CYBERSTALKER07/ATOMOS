#!/usr/bin/env bash
# Apply 20260818 GlobalProducts DDL (Gate 5 / §8.10 Phase 3) + verify.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ -f "$ROOT/.env.ssmr" && -z "${SPANNER_PROJECT:-}" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env.ssmr"
	set +a
fi

DDL="${DDL_FILE:-$ROOT/apps/backend-go/schema/migrations/20260818_global_products.ddl}"
if [[ ! -f "$DDL" ]]; then
	echo "missing DDL: $DDL" >&2
	exit 1
fi

if [[ -n "${SPANNER_EMULATOR_HOST:-}" ]]; then
	echo "==> emulator host=$SPANNER_EMULATOR_HOST"
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

echo "==> verify GlobalProducts tables"
grep -q 'CREATE TABLE GlobalProducts' "$DDL"
grep -q 'CREATE TABLE SupplierProductOffers' "$DDL"
grep -q 'CREATE TABLE ProductMatchQueue' "$DDL"
grep -q 'CREATE TABLE UnitsOfMeasure' "$DDL"
echo "global-products-ddl-ok"
