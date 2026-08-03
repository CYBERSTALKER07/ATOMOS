#!/usr/bin/env bash
# Discover GCP/staging wiring — names and health only (no secret values).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# gke-gcloud-auth-plugin lives in the Cloud SDK bin (not always on PATH).
if [[ -d "/opt/homebrew/share/google-cloud-sdk/bin" ]]; then
	export PATH="/opt/homebrew/share/google-cloud-sdk/bin:$PATH"
fi

echo "==> discover-staging-wiring"
echo "date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "repo: $ROOT"

if command -v gcloud >/dev/null 2>&1; then
	echo "gcloud project: $(gcloud config get-value project 2>/dev/null || echo unknown)"
	echo "gcloud account: $(gcloud config get-value account 2>/dev/null || echo unknown)"
	if gcloud secrets list --project=pegasus-503013 --filter="name:pegasusx" --format="value(name)" 2>/dev/null | head -5; then
		echo "gsm: pegasusx secrets listed (first 5 above)"
	else
		echo "gsm: list failed (check IAM)"
	fi
else
	echo "gcloud: not installed"
fi

BASE="${PUBLIC_BASE_URL:-https://api-ssmr.pegasusx.app}"
BASE="${BASE%/}"
echo "PUBLIC_BASE_URL probe: $BASE"
if curl -fsS --max-time 15 "${BASE}/healthz" >/dev/null 2>&1; then
	echo "healthz: ok"
else
	echo "healthz: FAIL or unreachable"
fi

if command -v kubectl >/dev/null 2>&1; then
	echo "kubectl context: $(kubectl config current-context 2>/dev/null || echo none)"
	kubectl get deploy -n pegasusx-ssmr 2>/dev/null | head -10 || echo "kubectl: cannot list pegasusx-ssmr (plugin/auth?)"
else
	echo "kubectl: not installed"
fi

echo "discover-staging-wiring-done"
echo "See docs/gap-closure/STAGING_WIRING_MATRIX.md"
