#!/usr/bin/env bash
# Smoke the Deep Agents ecosystem harness (no secrets printed).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck disable=SC1091
source .venv/bin/activate

echo "== dry-run =="
void-ecosystem-audit --dry-run

echo "== import =="
python -c "from void_deep_agents import create_ecosystem_auditor, create_void_deep_agent; print('imports_ok')"

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
  echo "  void-ecosystem-audit \"List open P1 gaps from ECOSYSTEM_GAP_REGISTER and classify by surface\""
  exit 0
fi

echo "== live audit =="
void-ecosystem-audit "List open P1 gaps still unmarked resolved in docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md. Classify remaining by surface (backend, money, data-flow, clients, cloud). Do not invent statuses; cite paths."
