#!/usr/bin/env bash
# Clears a corrupted Remotion Chrome Headless Shell download (Z_BUF_ERROR / unexpected end of file).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CACHE="$ROOT/node_modules/.remotion/chrome-headless-shell"

echo "Removing Remotion browser cache: $CACHE"
rm -rf "$CACHE"

# Optional stale puppeteer zip (unrelated version, can confuse debugging)
PUPPETEER_ZIP="$HOME/.cache/puppeteer/chrome-headless-shell"
if [[ -d "$PUPPETEER_ZIP" ]]; then
  echo "Removing puppeteer headless-shell cache: $PUPPETEER_ZIP"
  rm -rf "$PUPPETEER_ZIP"
fi

echo "Done. Re-run: npm run render:order-lifecycle"
echo "Chrome will re-download on next render (~150MB)."
