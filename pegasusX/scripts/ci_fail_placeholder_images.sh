#!/usr/bin/env bash
# Gate-0: refuse rendered overlays that still contain placeholder or :latest images.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OVERLAY="${1:-$ROOT/infra/k8s/overlays/prod}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl required for placeholder image gate" >&2
  exit 1
fi

rendered="$(kubectl kustomize "$OVERLAY" --load-restrictor LoadRestrictionsNone 2>/dev/null || true)"
if [[ -z "$rendered" ]]; then
  echo "kustomize render failed or empty for $OVERLAY" >&2
  exit 1
fi

fail=0
pattern='IMAGE_PLACEHOLDER|:latest|pegasusx-optimizer-core:local|REPLACE_WITH_DIGEST'
if echo "$rendered" | grep -E "$pattern" >/dev/null; then
  echo "FAIL: rendered overlay contains placeholder / :latest / :local / unset digest images:" >&2
  echo "$rendered" | grep -nE "$pattern" | head -40 >&2
  fail=1
fi

# P1-1: optimizer-core must never be remapped onto the backend-go image.
if echo "$rendered" | awk '
  /kind: Deployment/ {d=0}
  /name: optimizer-core/ {d=1}
  d && /image:/ {
    if ($0 ~ /backend-go/) { print; found=1 }
  }
  END { exit found ? 0 : 1 }
' >/dev/null; then
  echo "FAIL: optimizer-core Deployment image remapped to backend-go (crash-loop risk)" >&2
  echo "$rendered" | awk '/kind: Deployment/{d=0} /name: optimizer-core/{d=1} d && /image:/' | head -5 >&2
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi
echo "OK: no placeholder/:latest/:local images in $OVERLAY"
