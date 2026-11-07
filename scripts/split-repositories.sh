#!/usr/bin/env bash
# Enterprise 3-Way Repository Boundary Guard & Git Subtree Split Orchestrator
# Separates the V.O.I.D monorepo into three standalone, turnkey Git repositories:
#   1. pegasusx   (Priority-1 Production: SSMR B2B Distribution OS — 1 Supplier, 1,000s of Retailers)
#   2. antimatter (Public Marketing, 3D Showcase, xAI Assistant, Supabase Lead Hub + Native CRM Consoles + Remotion Visuals)
#   3. pegasus    (Priority-2 R&D: Multi-Supplier Marketplace + Spatial 3D / Rust Engine)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:---verify-only}"
OUT_DIR="${2:-$(cd "$ROOT/.." && pwd)}"

echo "==> [1/4] Auditing strict 3-way boundary isolation (pegasusX <-> pegasus <-> antimatter)..."

# 1. Ensure pegasusX never imports pegasus Go modules or TS packages
if git -C "$ROOT" grep -nE 'github\.com/pegasus/|@pegasus/' -- 'pegasusX/apps' 'pegasusX/packages'; then
  echo "ERROR: Cross-repository dependency leak detected in pegasusX -> pegasus" >&2
  exit 1
fi

# 2. Ensure pegasus never imports pegasusX Go modules or TS packages
if git -C "$ROOT" grep -nE 'github\.com/pegasusx/|@pegasusx/' -- 'pegasus/apps' 'pegasus/packages'; then
  echo "ERROR: Cross-repository dependency leak detected in pegasus -> pegasusX" >&2
  exit 1
fi

# 3. Ensure antimatter (pegasusX/antimater) never imports internal @pegasusx/* or @pegasus/* workspace packages
if git -C "$ROOT" grep -nE 'from ['\''"]@pegasusx/|from ['\''"]@pegasus/' -- 'pegasusX/antimater/app' 'pegasusX/antimater/components' 'pegasusX/antimater/lib'; then
  echo "ERROR: Cross-repository dependency leak detected in antimatter -> internal monorepo packages" >&2
  exit 1
fi

# 4. Ensure pegasusX operational apps never import from antimatter
if git -C "$ROOT" grep -nE 'antimater/' -- 'pegasusX/apps' 'pegasusX/packages'; then
  echo "ERROR: Cross-repository dependency leak detected in pegasusX operational code -> antimatter" >&2
  exit 1
fi

echo "OK  Zero cross-repository code imports across pegasusX, pegasus, and antimatter."

echo "==> [2/4] Verifying standalone toolchain manifests for all 3 codebases..."
for required_manifest in \
  "$ROOT/pegasusX/go.work" \
  "$ROOT/pegasusX/package.json" \
  "$ROOT/pegasusX/pnpm-workspace.yaml" \
  "$ROOT/pegasusX/turbo.json" \
  "$ROOT/pegasusX/Makefile" \
  "$ROOT/pegasusX/.github/workflows/ci.yml" \
  "$ROOT/pegasusX/antimater/package.json" \
  "$ROOT/pegasusX/antimater/package-lock.json" \
  "$ROOT/pegasusX/antimater/next.config.ts" \
  "$ROOT/pegasusX/antimater/supabase/schema.sql" \
  "$ROOT/pegasus/package.json" \
  "$ROOT/pegasus/pnpm-workspace.yaml" \
  "$ROOT/pegasus/turbo.json"
do
  if [[ ! -f "$required_manifest" ]]; then
    echo "ERROR: Missing standalone manifest: $required_manifest" >&2
    exit 1
  fi
done
echo "OK  All 3 codebases (pegasusX, antimatter, pegasus) have complete standalone manifests."

if [[ "$MODE" == "--verify-only" ]]; then
  echo "==> [3/4] 3-way boundary & split readiness audit complete (run with '--execute [out_dir]' to materialize all 3 git repos)."
  echo "repo-boundary-isolation-ok"
  exit 0
fi

if [[ "$MODE" != "--execute" ]]; then
  echo "Usage: $0 [--verify-only | --execute [output_dir]]" >&2
  exit 1
fi

echo "==> [3/4] Extracting 3 standalone git repositories into $OUT_DIR ..."
mkdir -p "$OUT_DIR"

cd "$ROOT"

# ---------------------------------------------------------------------------
# 1. ANTIMATTER (Marketing Site + Supabase Lead Hub + Native CRM Apps + Visuals)
# ---------------------------------------------------------------------------
echo "  -> [1/3] Splitting antimatter (pegasusX/antimater) history via git subtree..."
AM_SHA="$(git subtree split --prefix=pegasusX/antimater HEAD)"
rm -rf "$OUT_DIR/antimatter"
mkdir -p "$OUT_DIR/antimatter"
git -C "$OUT_DIR/antimatter" init -b main
git -C "$OUT_DIR/antimatter" pull "$ROOT" "$AM_SHA"

# Copy companion Remotion visuals & Native Executive CRM clients into antimatter repo
mkdir -p "$OUT_DIR/antimatter/clients"
for crm_app in enterprise-app-android enterprise-app-ios enterprise-app-desktop-macos; do
  if [[ -d "$ROOT/pegasusX/apps/$crm_app" ]]; then
    rsync -a --exclude='build' --exclude='.gradle' --exclude='.build' --exclude='DerivedData' \
      "$ROOT/pegasusX/apps/$crm_app/" "$OUT_DIR/antimatter/clients/$crm_app/"
  fi
done
if [[ -d "$ROOT/pegasusX/visuals" ]]; then
  rsync -a --exclude='node_modules' --exclude='out' --exclude='dist' \
    "$ROOT/pegasusX/visuals/" "$OUT_DIR/antimatter/visuals/"
  # Normalize visuals output path from ../antimater/public to ../public
  python3 - "$OUT_DIR/antimatter/visuals" <<'PY'
import pathlib, sys
root = pathlib.Path(sys.argv[1])
for rel in ["package.json", "topics-manifest.json", "scripts/ensure-media-dirs.sh", "src/lib/paths.ts"]:
    p = root / rel
    if p.is_file():
        p.write_text(p.read_text(encoding="utf-8").replace("antimater/public", "public"), encoding="utf-8")
PY
fi

cat > "$OUT_DIR/antimatter/netlify.toml" <<'EOF'
# Standalone Antimatter Marketing & Supabase Lead Hub (Next.js 16)
[build]
  command = "npm ci --legacy-peer-deps && npm run build"
  publish = ".next"

[build.environment]
  NODE_VERSION = "20"

[[plugins]]
  package = "@netlify/plugin-nextjs"
EOF

mkdir -p "$OUT_DIR/antimatter/.github/workflows"
cat > "$OUT_DIR/antimatter/.github/workflows/ci.yml" <<'EOF'
name: Antimatter CI

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  typecheck-and-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
      - name: Install dependencies
        run: npm ci --legacy-peer-deps
      - name: TypeScript check
        run: npx tsc --noEmit
      - name: Build Next.js production bundle
        run: npm run build
EOF

git -C "$OUT_DIR/antimatter" add -A
git -C "$OUT_DIR/antimatter" commit -m "chore(repo): configure standalone antimatter repository with Netlify, CI, visuals, and native CRM clients" >/dev/null

# ---------------------------------------------------------------------------
# 2. PEGASUSX (Priority-1 Production: SSMR B2B Distribution OS)
# ---------------------------------------------------------------------------
echo "  -> [2/3] Splitting pegasusX history via git subtree..."
PX_SHA="$(git subtree split --prefix=pegasusX HEAD)"
rm -rf "$OUT_DIR/pegasusx"
mkdir -p "$OUT_DIR/pegasusx"
git -C "$OUT_DIR/pegasusx" init -b main
git -C "$OUT_DIR/pegasusx" pull "$ROOT" "$PX_SHA"

# Copy current working-tree improvements from pegasusX into standalone repo and strip antimater/visuals/CRM apps
rsync -a \
  --exclude='.git' \
  --exclude='node_modules' \
  --exclude='.next' \
  --exclude='dist' \
  --exclude='.turbo' \
  --exclude='.loki' \
  --exclude='.cgr-*' \
  --exclude='.dynamic_watcher_status.json' \
  --exclude='__pycache__' \
  --exclude='.venv' \
  --exclude='apps/backend-go/backend-go' \
  --exclude='apps/backend-go/main' \
  --exclude='apps/backend-go/ssmr-smokecheck*' \
  --exclude='.env' \
  --exclude='.env.local' \
  --exclude='.env.ssmr' \
  --exclude='.env.k8s*' \
  --exclude='.env.staging.secrets' \
  --exclude='tfplan' \
  --exclude='antimater' \
  --exclude='visuals' \
  --exclude='apps/enterprise-app-android' \
  --exclude='apps/enterprise-app-ios' \
  --exclude='apps/enterprise-app-desktop-macos' \
  "$ROOT/pegasusX/" "$OUT_DIR/pegasusx/"

rm -rf \
  "$OUT_DIR/pegasusx/antimater" \
  "$OUT_DIR/pegasusx/visuals" \
  "$OUT_DIR/pegasusx/apps/enterprise-app-android" \
  "$OUT_DIR/pegasusx/apps/enterprise-app-ios" \
  "$OUT_DIR/pegasusx/apps/enterprise-app-desktop-macos"

if [[ -f "$ROOT/.github/workflows/ci-unit-tests-go.yml" ]]; then
  cp "$ROOT/.github/workflows/ci-unit-tests-go.yml" "$OUT_DIR/pegasusx/.github/workflows/ci-unit-tests-go.yml"
  python3 - "$OUT_DIR/pegasusx/.github/workflows/ci-unit-tests-go.yml" <<'PY'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
if p.is_file():
    p.write_text(p.read_text(encoding="utf-8").replace("default: pegasusX/apps/backend-go", "default: apps/backend-go"), encoding="utf-8")
PY
fi

# Normalize standalone pegasusx/.github/workflows/ci.yml root paths
python3 - "$OUT_DIR/pegasusx/.github/workflows/ci.yml" <<'PY'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
if p.is_file():
    text = p.read_text(encoding="utf-8")
    text = text.replace("working-directory: pegasusX/apps/", "working-directory: apps/")
    text = text.replace("working-directory: pegasusX/${{ matrix.xcodegen_dir }}", "working-directory: ${{ matrix.xcodegen_dir }}")
    text = text.replace("working-directory: pegasusX\n", "working-directory: .\n")
    text = text.replace("${{ github.workspace }}/pegasusX/", "${{ github.workspace }}/")
    text = text.replace("GITLEAKS_CONFIG: pegasusX/.gitleaks.toml", "GITLEAKS_CONFIG: .gitleaks.toml")
    text = text.replace("scan-ref: 'pegasusX/", "scan-ref: '")
    p.write_text(text, encoding="utf-8")
PY

git -C "$OUT_DIR/pegasusx" add -A
git -C "$OUT_DIR/pegasusx" commit -m "chore(repo): finalize standalone pegasusX production repository and normalize root CI" >/dev/null || true

# ---------------------------------------------------------------------------
# 3. PEGASUS (Priority-2 R&D: Multi-Supplier Marketplace + Spatial 3D / Rust Engine)
# ---------------------------------------------------------------------------
echo "  -> [3/3] Splitting pegasus history via git subtree..."
PG_SHA="$(git subtree split --prefix=pegasus HEAD)"
rm -rf "$OUT_DIR/pegasus"
mkdir -p "$OUT_DIR/pegasus"
git -C "$OUT_DIR/pegasus" init -b main
git -C "$OUT_DIR/pegasus" pull "$ROOT" "$PG_SHA"

cp "$ROOT/pegasus/package.json" "$OUT_DIR/pegasus/package.json"
cp "$ROOT/pegasus/scripts/guard_utils.py" "$OUT_DIR/pegasus/scripts/guard_utils.py"
cp "$ROOT/pegasus/pnpm-workspace.yaml" "$OUT_DIR/pegasus/pnpm-workspace.yaml"
cp "$ROOT/pegasus/turbo.json" "$OUT_DIR/pegasus/turbo.json"
mkdir -p "$OUT_DIR/pegasus/.github/workflows"
cat > "$OUT_DIR/pegasus/.github/workflows/ci.yml" <<'EOF'
name: Pegasus CI

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  backend-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: Build backend-go
        working-directory: apps/backend-go
        run: go build ./...
EOF

git -C "$OUT_DIR/pegasus" add -A
git -C "$OUT_DIR/pegasus" commit -m "chore(repo): add standalone pnpm-workspace.yaml, turbo.json, guard fixes, and root CI workflow" >/dev/null || true

# Copy untracked local .env files AFTER commits so local dev works out of the box without tracking secrets
for env_file in "$ROOT/pegasusX/antimater"/.env*; do
  if [[ -f "$env_file" && "$(basename "$env_file")" != ".env.example" ]]; then
    cp "$env_file" "$OUT_DIR/antimatter/$(basename "$env_file")"
  fi
done
for env_file in "$ROOT/pegasusX"/.env*; do
  if [[ -f "$env_file" && "$(basename "$env_file")" != *.example ]]; then
    cp "$env_file" "$OUT_DIR/pegasusx/$(basename "$env_file")"
  fi
done
for env_file in "$ROOT/pegasus"/.env*; do
  if [[ -f "$env_file" && "$(basename "$env_file")" != *.example ]]; then
    cp "$env_file" "$OUT_DIR/pegasus/$(basename "$env_file")"
  fi
done

echo "==> [4/4] All 3 standalone repositories materialized:"
echo "    1. pegasusX   (SSMR B2B Distribution OS):     $OUT_DIR/pegasusx"
echo "    2. antimatter (Marketing & Supabase Lead Hub): $OUT_DIR/antimatter"
echo "    3. pegasus    (Multi-Supplier & 3D Engine):    $OUT_DIR/pegasus"
