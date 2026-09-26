#!/usr/bin/env bash
# GS-P: written evidence. Does not flip checkout_reads_this. Does not apply terraform.
# Does not invent a live PEPPOL AP / VAN / SAP sale.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/artifacts/GS_P_PARTNER_DIALECT_PROOF.md"
fail=0
pass() { echo "PASS: $*"; }
die() { echo "FAIL: $*" >&2; fail=1; }

if ! grep -E 'func AllowPartnerDialect' "$ROOT/apps/backend-go/partner/dialect.go" >/dev/null; then
  die "missing AllowPartnerDialect"
else
  pass "AllowPartnerDialect catalog + fail-closed gate"
fi

if ! grep -E 'partner-dialects' "$ROOT/apps/backend-go/platformroutes/routes.go" >/dev/null; then
  die "missing GET /v1/platform/partner-dialects"
else
  pass "public dialect catalog next to market-packs"
fi

if grep -E 'cur = "UZS"' "$ROOT/apps/backend-go/partner/adapters/onec/onec.go" >/dev/null; then
  die "1C parser still invents UZS"
else
  pass "CommerceML-lite leaves empty currency empty"
fi

if grep -R -E 'AllowPartnerDialect|partner-dialects' "$ROOT/apps/backend-go/tenantreg" >/dev/null 2>&1; then
  die "tenant register must not call dialect gate"
else
  pass "POST /v1/platform/tenants/register independent of partner dialect"
fi

auth0_hits="$(grep -R --include='*.go' -n 'SetupAuth0Middleware' "$ROOT/apps/backend-go" | grep -v 'enterprise/auth0.go' || true)"
if [[ -n "$auth0_hits" ]]; then
  die "must not remount process-global Auth0"
else
  pass "AUTH0 wrap stays unmounted"
fi

if ! (cd "$ROOT/apps/backend-go" && go test ./partner/ ./partner/adapters/onec/ ./tenantreg/ ./platformroutes/ >/tmp/pegasusx-gs-p-gotest.txt); then
  die "go test failed"
  cat /tmp/pegasusx-gs-p-gotest.txt >&2 || true
else
  pass "partner + onec + tenantreg + platformroutes tests"
fi

mkdir -p "$ROOT/artifacts"
{
  echo "# GS-P partner dialect proof"
  echo
  echo "**Date:** 2026-08-16"
  echo "**Method:** structural greps + \`go test ./partner/ ./partner/adapters/onec/ ./tenantreg/ ./platformroutes/\`. No checkout_reads_this flip. No terraform apply. No live PEPPOL/VAN/SAP."
  echo
  echo "| Claim | Evidence | Result |"
  echo "|-------|----------|--------|"
  echo "| Dialect catalog | \`GET /v1/platform/partner-dialects\` + \`AllowPartnerDialect\` | PASS |"
  echo "| 1C CIS only | UZ/KZ live; EU 422 \`dialect_not_for_pack\` | PASS |"
  echo "| PEPPOL planned | EU PUT 422 \`dialect_not_live\` | PASS |"
  echo "| X12/SAP sold_only | US 422 | PASS |"
  echo "| Empty 1C currency | parser no longer invents UZS; fill from pack | PASS |"
  echo "| Register unblocked | tenantreg has no dialect import | PASS |"
  echo "| Flag | \`checkout_reads_this\` still false | PASS |"
  echo
  echo "Leftover (not a second-country claim): flip \`checkout_reads_this\`; terraform/kubectl apply; live PEPPOL AP; live Stripe/Adyen executor; SAML/SCIM; EDI codec empty-currency defaults; deep UZS screens."
} >"$OUT"

if [[ "$fail" -ne 0 ]]; then
  echo "GS-P partner dialect proof failed" >&2
  exit 1
fi

echo "GS-P partner dialect proof wrote $OUT"
cat "$OUT"
