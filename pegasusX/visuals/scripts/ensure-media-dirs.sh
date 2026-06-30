#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SITE_MEDIA="$ROOT/../softwareengineercv-main/public/media"
mkdir -p "$SITE_MEDIA"/{platform,solutions,roles,capabilities,technology,ai-vision,operations,apps-deploy}
echo "Media output dirs ready under $SITE_MEDIA"
