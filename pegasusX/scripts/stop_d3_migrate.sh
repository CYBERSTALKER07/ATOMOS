#!/usr/bin/env bash
# Stop stuck D3 Spanner schema jobs (setup / apply-migration / phase0 migrate).
set -euo pipefail
killed=0
while read -r pid cmd; do
  case "$cmd" in
    *phase0_apply_spanner*|*cmd/setup*|*apply-migration*|*go-build*setup*)
      echo "kill $pid $cmd"
      kill "$pid" 2>/dev/null || true
      killed=$((killed + 1))
      ;;
  esac
done < <(ps -axo pid=,command=)
sleep 1
echo "stopped_count=$killed"
ps -axo pid=,command= | grep -E 'phase0_apply_spanner|cmd/setup|apply-migration' | grep -v grep || echo "none remaining"
