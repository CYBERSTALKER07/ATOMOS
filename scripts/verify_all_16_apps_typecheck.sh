#!/usr/bin/env bash
set -e

WORKSPACE_ROOT="/Users/shakhzod/Desktop/V.O.I.D"

echo "================================================================"
echo "STARTING VERIFICATION: 11/11 TYPESCRIPT APPLICATIONS"
echo "================================================================"

# Group 1: pegasusX (6 applications)
echo ""
echo "--- GROUP 1: pegasusX (6 Applications) ---"
PEGASUSX_APPS=("admin-portal" "retailer-app-desktop" "supplier-portal" "warehouse-portal" "factory-portal" "payload-terminal")
for app in "${PEGASUSX_APPS[@]}"; do
  echo "Checking pegasusX/apps/$app..."
  (cd "$WORKSPACE_ROOT/pegasusX/apps/$app" && pnpm exec tsc --noEmit)
  echo "✓ pegasusX/apps/$app: EXIT CODE 0"
done

# Group 2: pegasus (5 applications)
echo ""
echo "--- GROUP 2: pegasus (5 Applications) ---"
PEGASUS_APPS=("admin-portal" "warehouse-portal" "factory-portal" "retailer-app-desktop" "payload-terminal")
for app in "${PEGASUS_APPS[@]}"; do
  echo "Checking pegasus/apps/$app..."
  (cd "$WORKSPACE_ROOT/pegasus/apps/$app" && npx tsc --noEmit)
  echo "✓ pegasus/apps/$app: EXIT CODE 0"
done

echo ""
echo "================================================================"
echo "SUCCESS: ALL 11 APPLICATIONS COMPILED WITH EXIT CODE 0"
echo "================================================================"
