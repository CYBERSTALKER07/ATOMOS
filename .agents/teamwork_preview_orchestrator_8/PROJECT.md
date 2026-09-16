# Project: Supplier Onboarding & Fleet Hub in pegasus.x

## Architecture
- **Ecosystem**: `pegasus.x` Sovereign Lean Single-Tenant Core.
- **Persistence**: PostgreSQL 16 with `pgx/v5` connection pool (`*db.Pool`). Zero mock/in-memory repositories.
- **Cache & Real-time**: Redis 7 (Streams `stream:supplier:events`, Pub/Sub, Geo proximity `warehouses:locations`).
- **WebSockets**: Monotonic sequence tracking (`seq++`) fanout via `internal/ws/hub.go`.
- **Architectural Boundary**: Pure PostgreSQL 16 + Redis 7. ZERO Cloud Spanner, ZERO Apache Kafka.
- **Financial Rule**: Strict 64-bit integer minor units (`tiyin` / `BIGINT`). Zero floating-point arithmetic for prices.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | F1: Migration 069 | `069_supplier_onboarding_and_globalpay.sql` adding STIR uniqueness, auth columns, `products`, `supplier_payment_gateways`, warehouse `status`, `warehouse_trucks`, and `payloaders`. | M1 | Survey 2 |
| 2 | F2: Repository Purge | Purge `MemoryRepository` and 57 silent fallback occurrences from `internal/supplier/repository.go`; wire pure `pgxpool` SQL queries. | M1 | Survey 1 |
| 3 | F3: Supplier Register | `POST /v1/auth/supplier/register` with 9-digit STIR uniqueness (HTTP 409 Conflict), bcrypt hashing, atomic insert into `suppliers` + `supplier_profiles`, `onboarding_status = 'PENDING'`. | M2 | Survey 1, Survey 3 |
| 4 | F4: Supplier Login | `POST /v1/auth/supplier/login` verifying bcrypt password against DB, returning JWT claims, `onboarding_status`, `next_step: "/onboarding/products"`. | M2 | Survey 1, Survey 3 |
| 5 | F5: Onboarding Gate Middleware | `RequireSupplierOnboardingCompleted` blocking operational endpoints with HTTP 428 Precondition Required (`onboarding_incomplete`) when status != `'COMPLETED'`; whitelist auth & onboarding. | M3 | Survey 3 |
| 6 | F6: Onboarding Step 1 (Catalog) | `POST /v1/supplier/onboarding/products` enforcing unique EAN-13, 17-digit MXIK, package code, units_per_case, 64-bit integer tiyin price, 12% VAT, >=1 active product required. | M3 | Survey 3 |
| 7 | F7: Onboarding Step 2 (Payment) | `POST /v1/supplier/onboarding/payment` configuring Cash (default enabled) & Global Pay (`GLOBAL_PAY`) with service ID, secret key, and corporate card BIN validation (Uzcard/Humo KPK). | M3 | Survey 3 |
| 8 | F8: Onboarding Step 3 (Complete) | `POST /v1/supplier/onboarding/complete` transitioning status to `'COMPLETED'`, unblocking the gate, emitting outbox event, Redis stream, and WebSocket hub broadcast. | M3 | Survey 3 |
| 9 | F9: Warehouse Management | `POST/GET/PUT/DELETE /v1/supplier/warehouses` with mandatory `DOUBLE PRECISION` lat/lon, Redis proximity cache invalidation, `warehouse.relocated` event, 409 deletion guard on stock/active orders. | M4 | Survey 2, Survey 3 |
| 10 | F10: Warehouse Trucks | `POST/GET /v1/supplier/warehouses/{id}/trucks` (license plate, capacity kg/m³, fuel type) stored in PostgreSQL. | M4 | Survey 2, Survey 3 |
| 11 | F11: Warehouse Payloaders | `POST/GET /v1/supplier/warehouses/{id}/payloaders` (name, phone, warehouse_id) stored in PostgreSQL. | M4 | Survey 2, Survey 3 |
| 12 | F12: E2E Verification & Hardening | Complete E2E test suite pass (`go test -v -race ./...`) and adversarial hardening. | M5 | E2E Track |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| E2E | E2E Testing Suite Track | Comprehensive opaque-box test suite for R1-R4 covering Tiers 1-4 | none | DONE |
| M1 | PostgreSQL 16 Migration 069 & Repo Purge | `069_supplier_onboarding_and_globalpay.sql`, purge `MemoryRepository`, wire pure `pgxpool` | none | DONE |
| M2 | Supplier Sign-Up & Sign-In | STIR deduplication (409 Conflict), bcrypt hashing, JWT issuance with `onboarding_status` | M1 | DONE |
| M3 | Non-Bypassable Onboarding Gate & Wizard | Middleware HTTP 428, Step 1 Catalog, Step 2 Global Pay, Step 3 Complete + Outbox/WS | M1, M2 | DONE |
| M4 | Warehouse & Fleet Management Hub | Warehouses GPS, Redis invalidation, 409 deletion guard, Trucks & Payloaders endpoints | M1, M3 | IN_PROGRESS |
| M5 | Final Milestone: E2E Pass & Hardening | 100% E2E test pass (`go test -v -race ./...`), adversarial coverage hardening | E2E, M4 | PLANNED |

## Interface Contracts

### Auth API ↔ Client
```http
POST /v1/auth/supplier/register
Request:
{
  "company_name": "OOO Samarkand Logistics",
  "tax_id": "301234567",
  "phone": "+998901234567",
  "password": "SecurePassword123!"
}
Response 201 Created:
{
  "supplier_id": "sup_...",
  "company_name": "OOO Samarkand Logistics",
  "tax_id": "301234567",
  "phone": "+998901234567",
  "onboarding_status": "PENDING",
  "next_step": "/onboarding/products"
}
Response 409 Conflict (duplicate STIR):
{
  "error": "conflict",
  "message": "tax_id already registered"
}

POST /v1/auth/supplier/login
Request:
{
  "tax_id": "301234567",
  "password": "SecurePassword123!"
}
Response 200 OK:
{
  "token": "<jwt_token>",
  "supplier_id": "sup_...",
  "onboarding_status": "PENDING",
  "next_step": "/onboarding/products"
}
```

### Onboarding Gate Middleware ↔ Protected Routes
```http
Any Operational Route (e.g. GET /v1/supplier/warehouses, POST /v1/orders)
Response 428 Precondition Required (when onboarding_status != 'COMPLETED'):
{
  "error": "onboarding_incomplete",
  "onboarding_status": "PENDING",
  "next_step": "/onboarding/products",
  "message": "supplier onboarding must be completed before accessing operational endpoints"
}
```

### Onboarding Wizard API
```http
POST /v1/supplier/onboarding/products
Request:
{
  "name": "Coca-Cola Classic 1.5L",
  "barcode": "4780012345678",
  "mxik_code": "01234567890123456",
  "package_code": "PKG-01",
  "units_per_case": 6,
  "unit_price_tiyin": 1450000,
  "vat_rate": 12.0
}
Response 201 Created

POST /v1/supplier/onboarding/payment
Request:
{
  "cash_enabled": true,
  "global_pay": {
    "enabled": true,
    "service_id": "gp_srv_9921",
    "secret_key": "gp_sec_k4819",
    "corporate_card_bins": ["5614", "9860", "5440", "4073", "5168"]
  }
}
Response 200 OK

POST /v1/supplier/onboarding/complete
Request: {}
Response 200 OK:
{
  "supplier_id": "sup_...",
  "onboarding_status": "COMPLETED",
  "message": "onboarding successfully completed"
}
```

### Warehouse & Fleet API
```http
POST /v1/supplier/warehouses
Request:
{
  "name": "Sergeli Central Distribution Hub",
  "address": "Yangihayot 14, Tashkent",
  "latitude": 41.2215,
  "longitude": 69.2140
}
Response 201 Created

DELETE /v1/supplier/warehouses/{id}
Response 409 Conflict (if on_hand_qty > 0 or active orders exist):
{
  "error": "cannot_delete_warehouse",
  "message": "warehouse has active inventory or non-terminal orders"
}

POST /v1/supplier/warehouses/{id}/trucks
Request:
{
  "license_plate": "01 777 AAA",
  "capacity_kg": 5000.0,
  "capacity_m3": 25.0,
  "fuel_type": "DIESEL"
}

POST /v1/supplier/warehouses/{id}/payloaders
Request:
{
  "name": "Rustam Karimov",
  "phone": "+998909876543"
}
```

## Code Layout
- `pegasus.x/database/migrations/069_supplier_onboarding_and_globalpay.sql`: PostgreSQL 16 DDL
- `pegasus.x/backend/internal/supplier/repository.go`: PostgreSQL repository implementation (purged of MemoryRepository)
- `pegasus.x/backend/internal/supplier/service.go`: Business logic for onboarding, validation, payments, warehouses, trucks, payloaders
- `pegasus.x/backend/internal/supplier/models.go`: Domain models
- `pegasus.x/backend/internal/api/handlers_supplier.go`: Auth and onboarding handlers
- `pegasus.x/backend/internal/api/handlers_warehouse.go` (or `handlers_fleet.go`): Warehouse and fleet handlers
- `pegasus.x/backend/internal/middleware/onboarding.go` (or `internal/auth/middleware.go`): `RequireSupplierOnboardingCompleted`
