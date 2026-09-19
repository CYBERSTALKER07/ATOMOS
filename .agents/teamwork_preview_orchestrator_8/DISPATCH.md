## 2026-09-16T13:15:19Z

Mission:
End-to-end implementation of Supplier Sign-Up, Sign-In, Non-Bypassable Phased Onboarding (Product Catalog with MXIK/Tiyins, Cash & Global Pay Corporate Card Gateway), and Post-Onboarding Warehouse/Fleet Management in `pegasus.x`.

Working directory: /Users/shakhzod/Desktop/V.O.I.D
Integrity mode: development

Requirements:
### R1. Minimal Supplier Sign-Up & Sign-In with STIR Legal Deduplication
- Implement `POST /v1/auth/supplier/register` accepting:
  - `company_name` (Legal company name)
  - `tax_id` (Uzbekistan 9-digit STIR/INN)
  - `phone` (`+998XXXXXXXXX`)
  - `password` (bcrypt hashed)
- Enforce STIR uniqueness in PostgreSQL; return HTTP 409 Conflict if already registered.
- Set initial state: `onboarding_status = 'PENDING'`.
- Implement `POST /v1/auth/supplier/login`:
  - Returns JWT claims and current `onboarding_status` with `next_step: "/onboarding/products"`.

### R2. Non-Bypassable Onboarding Gate & Phased Wizard
- Build middleware `RequireSupplierOnboardingCompleted`:
  - If `onboarding_status != 'COMPLETED'`, block all operational supplier endpoints with HTTP 428 Precondition Required (`onboarding_incomplete`).
  - Whitelist `/v1/auth/*` and `/v1/supplier/onboarding/*`.
- Step 1: Product Catalog Management:
  - `POST /v1/supplier/onboarding/products` (Add, Edit, Delete).
  - Enforce: Name, unique EAN-13 barcode, 17-digit statutory MXIK code, package code, units_per_case, unit_price_tiyin (64-bit integer), 12% VAT.
  - Gate requirement: At least 1 active product required to complete step.
- Step 2: Payment Gateway Configuration:
  - `POST /v1/supplier/onboarding/payment`.
  - Cash enabled by default.
  - Global Pay (`GLOBAL_PAY`) corporate card gateway setup with service ID, secret key, and corporate card BIN validation (B2B corporate cards only).
- Step 3: Complete Onboarding:
  - `POST /v1/supplier/onboarding/complete`: Transitions `onboarding_status` to `'COMPLETED'`, unblocks the gate, and emits outbox/WebSocket event.

### R3. Warehouse & Fleet Management Hub
- `POST/GET/PUT/DELETE /v1/supplier/warehouses`:
  - Mandatory `latitude` and `longitude` (`DOUBLE PRECISION`) for retailer proximity and driver dispatch routing.
  - Invalidate Redis proximity cache and emit `warehouse.relocated` on coordinate updates.
  - Block warehouse deletion with HTTP 409 Conflict if on-hand stock > 0 or active orders exist.
- Fleet & Dock Logistics:
  - `POST/GET /v1/supplier/warehouses/{id}/trucks` (license plate, capacity kg/m³, fuel type).
  - `POST/GET /v1/supplier/warehouses/{id}/payloaders` (name, phone, warehouse_id).

### R4. Zero Mock Data & Pure PostgreSQL 16 Persistence
- Purge `MemoryRepository` and hardcoded seeds from `pegasus.x/backend/internal/supplier/repository.go`.
- Create PostgreSQL migration `069_supplier_onboarding_and_globalpay.sql` for all new columns and tables.
- All endpoints query and write directly to PostgreSQL 16 via `pgxpool`.
- Full test suite passing with `go test -v -race ./...`.
