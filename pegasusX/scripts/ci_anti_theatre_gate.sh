#!/usr/bin/env bash
# Mechanical enforcement of the Anti-Reward-Hacking Laws in agents.md.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

fail=0
die() {
  echo "anti-theatre-gate-error: $*" >&2
  fail=1
}

# Law 0.1: Gate scripts must never point at legacy/ Terraform or dead directories
if grep -rn 'legacy/' scripts/phase*_gate.sh >/dev/null 2>&1; then
  die "scripts/phase*_gate.sh must never reference dead legacy/ directories"
fi

# Law 0.1: Active Terraform monitoring module must contain all 6 domain SLO alerts
for alert in \
  outbox_lag_high \
  fiscal_success_low \
  capture_success_low \
  outbox_relay_restarts_high \
  outbox_dlq_depth_nonzero \
  partner_webhook_success_low
do
  if ! grep -q "$alert" infra/terraform/modules/monitoring/main.tf; then
    die "infra/terraform/modules/monitoring/main.tf missing SLO alert policy: $alert"
  fi
done

# Law 0.4: packages/mobile-ios-design/*.swift must be symlinks to canonical packages/mobile-ios-core/Sources
for f in packages/mobile-ios-design/*.swift; do
  if [[ ! -L "$f" ]]; then
    die "$f is a duplicate regular file instead of a symlink to packages/mobile-ios-core/Sources"
  fi
done

# Law 0.5: Zero tracked build caches, vendor gems, object files, or Terraform plans in git
tracked_junk="$(git ls-files | grep -E '(\.gem/|/target/|\.tsbuildinfo$|\.o$|\.tfplan$|\.tfstate$|ai-worker/ai-worker$)' || true)"
if [[ -n "$tracked_junk" ]]; then
  die "tracked build/vendor artifacts found in git: $(echo "$tracked_junk" | head -n 5 | tr '\n' ' ')"
fi

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "anti-theatre-gate-ok"
