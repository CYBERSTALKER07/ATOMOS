#!/usr/bin/env bash
# Smoke the Deep Agents ecosystem orchestra (no secrets printed).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck disable=SC1091
source .venv/bin/activate

echo "== dry-run =="
void-ecosystem-audit --dry-run

echo "== import =="
python -c "
from void_deep_agents.subagents import build_subagents, panel_names
from void_deep_agents.paths import probe_filesystem_backend
assert len(panel_names()) == 12, panel_names()
assert len(build_subagents()) == 12
assert len(build_subagents(['money_fiscal','role_parity'])) == 2
probe = probe_filesystem_backend()
assert all(v == 'ok' for k, v in probe.items() if k != 'root'), probe
print('imports_ok panels=', len(panel_names()), 'fs_probe_ok')
"

# Load .env without exporting placeholder blindly into live call
if [[ -f .env ]]; then
  # shellcheck disable=SC1091
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

KEY="${XAI_API_KEY:-}"
if [[ -z "$KEY" || "$KEY" == "xai-your-key-here" ]]; then
  echo "== live audit =="
  echo "SKIPPED: set XAI_API_KEY in $ROOT/.env (see .env.example), then re-run:"
  echo "  void-ecosystem-audit --panel gap_register_sync \"Sync open P1 gaps\""
  echo "  void-ecosystem-audit --full --json-out /tmp/pegasusx-audit.json"
  exit 0
fi

echo "== live audit (gap_register_sync panel) =="
void-ecosystem-audit --panel gap_register_sync \
  "List open P1 gaps still unmarked resolved in ECOSYSTEM_GAP_REGISTER. Respect resolved_gap_ids. End with JSON findings."
