#!/usr/bin/env bash
set -e

WORKSPACE_ROOT="/Users/shakhzod/Desktop/V.O.I.D"
TSC_SYSTEM="$WORKSPACE_ROOT/pegasus.x/node_modules/.bin/tsc"

echo "================================================================"
echo "STARTING VERIFICATION: 16/16 TYPESCRIPT APPLICATIONS"
echo "================================================================"

# Group 1: pegasus.x (5 applications)
echo ""
echo "--- GROUP 1: pegasus.x (5 Applications + Shared Packages) ---"
cd "$WORKSPACE_ROOT/pegasus.x"
pnpm check-types --force

# Group 2: pegasusX (6 applications)
echo ""
echo "--- GROUP 2: pegasusX (6 Applications) ---"
PEGASUSX_APPS=("admin-portal" "retailer-app-desktop" "supplier-portal" "warehouse-portal" "factory-portal" "payload-terminal")
for app in "${PEGASUSX_APPS[@]}"; do
  echo "Checking pegasusX/apps/$app..."
  (cd "$WORKSPACE_ROOT/pegasusX/apps/$app" && pnpm exec tsc --noEmit)
  echo "✓ pegasusX/apps/$app: EXIT CODE 0"
done

# Group 3: pegasus (5 applications)
echo ""
echo "--- GROUP 3: pegasus (5 Applications) ---"
PEGASUS_APPS=("admin-portal" "warehouse-portal" "factory-portal" "retailer-app-desktop" "payload-terminal")
for app in "${PEGASUS_APPS[@]}"; do
  echo "Checking pegasus/apps/$app..."
  (cd "$WORKSPACE_ROOT/pegasus/apps/$app" && "$TSC_SYSTEM" --noEmit)
  echo "✓ pegasus/apps/$app: EXIT CODE 0"
done

echo ""
echo "================================================================"
echo "SUCCESS: ALL 16 APPLICATIONS COMPILED WITH EXIT CODE 0"
echo "================================================================"
