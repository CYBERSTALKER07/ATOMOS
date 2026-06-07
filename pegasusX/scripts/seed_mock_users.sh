#!/bin/bash
set -e

# Make sure the backend is running before executing this script!
# Usage: ./scripts/seed_mock_users.sh

BASE_URL="http://localhost:8080"

echo "=========================================="
echo "Seeding PegasusX Mock Credentials..."
echo "=========================================="

echo "[1/4] Registering Supplier Admin..."
SUPPLIER_JSON=$(cat <<EOF
{
  "account": {
    "legalName": "Mock Supplier",
    "contactName": "Admin",
    "email": "admin@supplier.com",
    "country": "UZ",
    "phone": "+998901111111",
    "password": "password123"
  },
  "location": {
    "warehouse": { "name": "Main WH", "address": "123 Main St", "lat": 41.2995, "lng": 69.2401 },
    "sameAsWarehouse": true,
    "billing": { "address": "", "lat": 0, "lng": 0 }
  },
  "business": {
    "taxId": "TAX123",
    "companyRegNumber": "REG123",
    "fleetVehicleCount": 5,
    "fleetMaxVU": 100,
    "factoryCount": 1
  },
  "categories": ["electronics"],
  "phone": "+998901111111"
}
EOF
)

SUPPLIER_RESP=$(curl -s -X POST "$BASE_URL/v1/auth/supplier/register" \
  -H "Content-Type: application/json" \
  -d "$SUPPLIER_JSON")

TOKEN=$(echo $SUPPLIER_RESP | grep -o '"token":"[^"]*' | grep -o '[^"]*$')
if [ -z "$TOKEN" ]; then
  echo "Failed to register supplier. (Maybe already registered?)"
  # Try to login instead to get the token
  LOGIN_RESP=$(curl -s -X POST "$BASE_URL/v1/auth/supplier/login" -H "Content-Type: application/json" -d '{"phone": "+998901111111", "password": "password123"}')
  TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*' | grep -o '[^"]*$')
  if [ -z "$TOKEN" ]; then
    echo "Failed to login as supplier too. Is backend running?"
    exit 1
  fi
fi

echo "  Supplier registered! Token obtained."

echo "[2/4] Creating Warehouse Admin..."
curl -s -X POST "$BASE_URL/v1/supplier/org/members" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Warehouse Admin",
    "phone": "+998902222222",
    "role": "WAREHOUSE_ADMIN",
    "password": "password123",
    "pin": "1234"
  }' > /dev/null
echo "  Warehouse Admin created."

echo "[3/4] Creating Fleet Driver..."
# Need to use WAREHOUSE as home_node_type to assign them to a hub.
curl -s -X POST "$BASE_URL/v1/supplier/fleet/drivers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Mock Driver",
    "phone": "+998903333333",
    "pin": "1234",
    "home_node_type": "WAREHOUSE",
    "home_node_id": "seed-warehouse-1"
  }' > /dev/null
echo "  Driver created."

echo "[4/4] Registering Retailer..."
curl -s -X POST "$BASE_URL/v1/auth/retailer/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Mock Retailer",
    "phone": "+998904444444",
    "password": "password123",
    "lat": 41.3000,
    "lng": 69.3000
  }' > /dev/null
echo "  Retailer registered."

echo "=========================================="
echo "Done! You can now log into the UIs using these mock credentials:"
echo "------------------------------------------"
echo "  Supplier Portal   : +998901111111 / password123"
echo "  Warehouse Portal  : +998902222222 / password123 (or pin 1234 if mobile)"
echo "  Driver App        : +998903333333 / pin 1234"
echo "  Retailer App      : +998904444444 / password123"
echo "=========================================="
