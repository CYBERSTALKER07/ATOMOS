#!/usr/bin/env bash
# Fail when new list/dashboard Spanner reads omit stale bounds without allowlist entry.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ALLOWLIST="$ROOT/scripts/spanner_stale_read_allowlist.txt"
BACKEND="$ROOT/apps/backend-go"
VIOLATIONS=()

while IFS= read -r file; do
	[[ -z "$file" ]] && continue
	[[ "$file" == *"_test.go" ]] && continue
	rel="${file#"$BACKEND"/}"
	while IFS= read -r line; do
		lineno="${line%%:*}"
		content="${line#*:}"
		if grep -qxF "$rel:$lineno" "$ALLOWLIST" 2>/dev/null; then
			continue
		fi
		VIOLATIONS+=("$rel:$lineno: $content")
	done < <(grep -n '\.Single()\.Query' "$file" 2>/dev/null || true)
done < <(find "$BACKEND" -name '*.go' -type f -print)

if ((${#VIOLATIONS[@]} > 0)); then
	echo "spanner-stale-read-gate-FAIL — add WithTimestampBound or allowlist entry:" >&2
	for v in "${VIOLATIONS[@]}"; do
		echo "  $v" >&2
	done
	exit 1
fi

echo "spanner-stale-read-gate-ok"
