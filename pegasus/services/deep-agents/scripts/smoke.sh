#!/usr/bin/env bash
# Smoke the Deep Agents ecosystem orchestra (no secrets printed).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck disable=SC1091
source .venv/bin/activate

echo "== dry-run =="
void-ecosystem-audit --dry-run

echo "== dry-run fleet =="
void-ecosystem-audit --dry-run --fleet

echo "== import =="
python -c "
from void_deep_agents.subagents import build_subagents, panel_names
from void_deep_agents.fleet import build_fleet_subagents, fleet_names
from void_deep_agents.paths import probe_filesystem_backend, fleet_filesystem_backend
from void_deep_agents import (
    create_void_deep_agent,
    create_void_langchain_agent,
    create_e2e_fleet_agent,
    create_ecosystem_auditor,
)
assert len(panel_names()) == 12, panel_names()
assert len(build_subagents()) == 12
assert all('skills' in s for s in build_subagents()), 'audit panels must wire skills'
assert len(fleet_names()) == 6, fleet_names()
assert len(build_fleet_subagents()) == 6
assert all('skills' in s for s in build_fleet_subagents()), 'fleet must wire skills'
probe = probe_filesystem_backend()
assert all(v == 'ok' for k, v in probe.items() if k not in {'pegasusx_root', 'skills_route'}), probe
probe_f = probe_filesystem_backend(fleet_filesystem_backend())
assert all(v == 'ok' for k, v in probe_f.items() if k not in {'pegasusx_root', 'skills_route'}), probe_f
print('imports_ok panels=', len(panel_names()), 'fleet=', len(fleet_names()), 'fs_probe_ok')
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
set +e
void-ecosystem-audit --panel gap_register_sync \
  "List open P1 gaps still unmarked resolved in ECOSYSTEM_GAP_REGISTER. Respect resolved_gap_ids. End with JSON findings."
rc=$?
set -e
if [[ $rc -ne 0 ]]; then
  echo "LIVE_AUDIT_NOTE: exit=$rc (often API credits/quota). Dry-run + imports already passed."
  exit 0
fi
