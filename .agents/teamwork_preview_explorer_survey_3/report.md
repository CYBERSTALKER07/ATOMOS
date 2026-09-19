# Architectural Survey & Deep Technical Specification: Middleware, Global Pay, Events & Fleet Hub in Pegasus.X

**Author:** Survey Specialist 3 (teamwork_preview_explorer_survey_3)  
**Target Subsystem:** `pegasus.x/backend` (Sovereign Single-Tenant Core)  
**Date:** 2026-09-16  
**Status:** Read-Only Investigation & Architectural Contract Design  

---

## 1. Executive Summary & Codebase Reality

Following the **Strict Two-System Architectural Boundary**, `pegasus.x` operates as a sovereign, single-tenant distribution operating system powered by:
- **Database:** PostgreSQL 16 (`pgx/v5` connection pool, 25 max conns)
- **Cache & Event Streaming:** Redis 7 (`redis-go/v9`, Streams `XADD`, Geo indexing, Pub/Sub)
- **Monetary Representation:** Strict 64-bit integer minor currency units (tiyins / cents, $1\text{ UZS} = 100\text{ tiyins}$) with zero floating-point math.
- **HTTP Routing:** Chi v5 (`github.com/go-chi/chi/v5`)

### Critical Gaps Uncovered During Survey
1. **Unprotected Supplier Operations:** While `auth.RequireAuthWithKeyManager` validates JWT tokens (`internal/api/router.go:537`), there is **no non-bypassable onboarding gate**. Operational endpoints (`/v1/supplier/*`, `/v1/warehouse/*`, `/v1/catalog/*`) can be invoked by any authenticated supplier regardless of onboarding state.
2. **Mock Repository Fallback & Seed Data:** `internal/supplier/repository.go:90-418` maintains a `MemoryRepository` pre-seeded with Tashkent mock data. In `PostgresRepository` (`repository.go:1024-1060`), any database error or missing record silently falls back to the in-memory mock repository (`if err != nil { return p.memory.GetProfile(ctx, supplierID) }`).
3. **Hardcoded Tenancy Fallbacks:** In `internal/api/handlers_supplier.go:29`, if `supplier_id` is missing in query or JWT claims, it silently defaults to `"sup_pepsico_uz"`. In `handleSupplierLogin` (`handlers_supplier.go:181`), `sid := "sup_pepsico_uz"` is hardcoded.
4. **Missing Phased Wizard Endpoints:** Step-by-step onboarding endpoints (`/v1/supplier/onboarding/products`, `/v1/supplier/onboarding/payment`, `/v1/supplier/onboarding/complete`) do not exist.
5. **No Corporate Card BIN Validation:** Current payment setup (`internal/supplier/service.go:215-225`) merely stores an arbitrary string array of gateways without validating corporate card BIN prefixes for statutory B2B settlements under the 25,000,000 UZS cash ceiling.
6. **In-Memory Payloader Storage:** `internal/onboarding/service.go:98` stores payloaders in `map[string]PayloaderProfile` with no PostgreSQL backing table or migration.
7. **Missing Supplier Warehouse CRUD:** There is no dedicated `/v1/supplier/warehouses` RESTful CRUD with mandatory GPS coordinates, Redis proximity invalidation, and stock/order conflict guards.

---

## 2. Middleware & Routing Architecture

### 2.1 Current Implementation in `pegasus.x/backend`
- **Router Construction:** Constructed in `internal/api/router.go:294-380` using `chi.NewRouter()`.
- **Global Middleware Stack (`router.go:298-311`):**
  ```go
  r.Use(middleware.RequestID)
  r.Use(middleware.RealIP)
  r.Use(middleware.Logger)
  r.Use(middleware.Recoverer)
  r.Use(s.metricsReg.HTTPMiddleware)
  r.Use(s.tracer.HTTPTracingMiddleware)
  r.Use(cors.Handler(cors.Options{...}))
  ```
- **JWT Extraction & Claims Injection (`internal/auth/middleware.go:16-55`):**
  - Context key: `claimsContextKey contextKey = "user_claims"` (`middleware.go:16`).
  - Auth extraction: Reads `Authorization: Bearer <token>` header, validates RS256/HS256 using `KeyManager` or secret string, and injects claims into `r.Context()`:
    ```go
    ctx := context.WithValue(r.Context(), claimsContextKey, claims)
    next.ServeHTTP(w, r.WithContext(ctx))
    ```
  - Retrieval helper: `auth.GetClaims(ctx context.Context) *models.UserClaims` (`middleware.go:108-114`).
- **User Claims Structure (`internal/models/claims.go:20-30`):**
  ```go
  type UserClaims struct {
      UserID       string `json:"user_id"`
      SupplierID   string `json:"supplier_id"`
      WarehouseID  string `json:"warehouse_id,omitempty"`
      FactoryID    string `json:"factory_id,omitempty"`
      DriverID     string `json:"driver_id,omitempty"`
      RetailerID   string `json:"retailer_id,omitempty"`
      Role         Role   `json:"role"`
      IsConfigured bool   `json:"is_configured,omitempty"`
      jwt.RegisteredClaims
  }
  ```

### 2.2 Design of `RequireSupplierOnboardingCompleted` Middleware

#### Behavioral Contract
1. **Target Actors:** Strictly enforces onboarding completion on all requests initiated by actors with `Role == models.RoleSupplier`.
2. **Whitelist Policy:** The following path patterns bypass the gate:
   - `/v1/auth/*` (Login, registration, token refresh)
   - `/v1/supplier/onboarding/*` (The phased onboarding wizard endpoints)
   - `/v1/platform/client-policy`
   - `/health`, `/healthz`, `/ready`, `/.well-known/*`, `/metrics`
3. **Precondition Rejection (HTTP 428):** If the supplier's `onboarding_status != 'COMPLETED'`, the middleware terminates the pipeline and returns HTTP 428 Precondition Required with the current status and next required action.
4. **High-Performance Verification:**
   - Level 1 (In-Memory Claims Check): Inspects `claims.IsConfigured` or `claims.OnboardingStatus`.
   - Level 2 (Redis Fast Cache): Checks `cache:supplier:onboarding:<supplierID>` (TTL: 5 minutes, invalidated immediately upon `/v1/supplier/onboarding/complete`).
   - Level 3 (PostgreSQL Authoritative SoT): Fallback query to `supplier_profiles.onboarding_status`.

#### Go Implementation Blueprint
```go
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/pegasus-x/core/internal/models"
	"github.com/pegasus-x/core/pkg/response"
)

// SupplierStatusProvider abstracts authoritative status resolution
type SupplierStatusProvider interface {
	GetSupplierOnboardingStatus(ctx context.Context, supplierID string) (status string, nextStep string, err error)
}

// RequireSupplierOnboardingCompleted enforces the non-bypassable onboarding gate
func RequireSupplierOnboardingCompleted(provider SupplierStatusProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// 1. Whitelist Bypass
			if isWhitelisted(path) {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Claims Extraction
			claims := GetClaims(r.Context())
			if claims == nil {
				response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication claims missing")
				return
			}

			// Gate applies strictly to Supplier operational roles
			if claims.Role != models.RoleSupplier {
				next.ServeHTTP(w, r)
				return
			}

			supplierID := strings.TrimSpace(claims.SupplierID)
			if supplierID == "" {
				response.Error(w, http.StatusForbidden, "invalid_tenancy", "Supplier ID claim missing")
				return
			}

			// 3. Resolve Authoritative Status
			status, nextStep, err := provider.GetSupplierOnboardingStatus(r.Context(), supplierID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "status_check_failed", "Failed to verify onboarding status")
				return
			}

			// 4. Gate Verification
			if status != "COMPLETED" {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusPreconditionRequired) // HTTP 428
				_ = response.JSON(w, http.StatusPreconditionRequired, map[string]interface{}{
					"error":             "onboarding_incomplete",
					"message":           "Operational access restricted. Supplier must complete onboarding wizard.",
					"onboarding_status": status,
					"next_step":         nextStep,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isWhitelisted(path string) bool {
	if strings.HasPrefix(path, "/v1/auth/") ||
		strings.HasPrefix(path, "/v1/supplier/onboarding/") ||
		path == "/health" || path == "/healthz" || path == "/ready" ||
		path == "/v1/platform/client-policy" ||
		strings.HasPrefix(path, "/.well-known/") ||
		path == "/metrics" {
		return true
	}
	return false
}
```

---

## 3. Phased Onboarding Wizard Endpoints

### 3.1 Step 1: Product Catalog Management (`POST /v1/supplier/onboarding/products`)

#### Business Invariants & Statutory Validations
1. **MXIK (IKPU) Statutory Tax Classification:**
   - Must match exact 17-digit numeric string format (`^\d{17}$`).
   - Validated against `soliq.ValidateMXIK(code)` (`internal/soliq/efactura.go:18-27`).
   - Example valid beverages code: `"01221001001000000"`.
2. **EAN-13 Barcode Validation & Deduplication:**
   - 13 digits long, purely numeric (`^\d{13}$`).
   - Checksum validation:
     $$\text{Sum} = \sum_{i=1}^{12} d_i \times (1 \text{ if } i \text{ odd else } 3)$$
     $$\text{CheckDigit} = (10 - (\text{Sum} \pmod{10})) \pmod{10} == d_{13}$$
   - Uniqueness check in `skus.barcode` across the system.
3. **Monetary Integer Invariant (Tiyins):**
   - `unit_price_minor > 0` (strictly `BIGINT`, minor currency units where $1\text{ UZS} = 100\text{ tiyins}$).
   - No floating-point values accepted.
4. **Statutory Soliq VAT:**
   - Fixed at 12% (`vat_percent = 12`) in compliance with Republic of Uzbekistan Tax Code.
5. **Package Code:** Statutory classification (e.g., `796` for piece, `166` for kg, `778` for pack/box).
6. **Units Per Case / Pack:** Positive integer $\ge 1$.
7. **Wizard Gate Condition:** The supplier must have at least 1 active product in the catalog to satisfy Step 1.

#### REST API Contracts
- `POST /v1/supplier/onboarding/products`:
  - Actions supported via body payload: `ADD`, `EDIT`, `DELETE`.
  - Request Payload Schema:
```json
{
  "action": "ADD",
  "product": {
    "name": "Coca-Cola Classic 1.5L PET",
    "barcode": "4780001234567",
    "mxik_code": "01221001001000000",
    "package_code": "796",
    "unit_price_minor": 1450000,
    "vat_percent": 12,
    "units_per_case": 6,
    "units_per_pack": 1,
    "pack_type": "CASE_6",
    "packaging_material": "PET",
    "size_volume_ml": 1500,
    "size_label": "1.5L"
  }
}
```
  - For `EDIT`:
```json
{
  "action": "EDIT",
  "sku_id": "sku_coke_1500",
  "product": {
    "name": "Coca-Cola Classic 1.5L PET (Updated)",
    "unit_price_minor": 1500000
  }
}
```
  - For `DELETE`:
```json
{
  "action": "DELETE",
  "sku_id": "sku_coke_1500"
}
```
- Response (HTTP 200 / 201):
```json
{
  "status": "success",
  "action": "ADD",
  "sku_id": "sku_coke_1500",
  "active_products_count": 1,
  "step_1_completed": true,
  "next_step": "/v1/supplier/onboarding/payment"
}
```

---

### 3.2 Step 2: Payment Gateway Configuration (`POST /v1/supplier/onboarding/payment`)

#### Business Invariants & Corporate Card BIN Validation
1. **Cash Default Enabled:**
   - `cash_enabled` is true by default.
   - Statutory compliance: Uzbekistan B2B transactions permit cash settlement up to 25,000,000 UZS (`2500000000` tiyins) per invoice leg (`internal/fiscal/calculator_test.go:135`).
2. **Global Pay Corporate Card Gateway (`GLOBAL_PAY`):**
   - Required merchant configuration:
     - `service_id`: Merchant identifier issued by Global Pay.
     - `secret_key`: HMAC/API secret for payment signature and checkout session generation.
     - `username` / `password`: Credentials for merchant token acquisition (`internal/payment/globalpay.go:133-165`).
3. **Uzbekistan B2B Corporate Card BIN Validation (KPK - Korporativ Plastik Karta):**
   - Statutory rule: Wholesale B2B supply settlements must be executed using corporate business cards (KPK), preventing illegal use of personal salary/consumer cards.
   - Valid Corporate BIN Prefixes:
     - **Uzcard Corporate (KPK):** `5614` (e.g., `561468`, `561400`), `86005`, `86006`.
     - **Humo Corporate (KPK):** `9860` with designated corporate ranges (`986035`, `986060`, `986001`, `986030`).
     - **Uzcard/Mastercard Co-badged Corporate:** `5440`.
     - **Visa Business / Corporate:** `4073`, `4188`, `4232`, `4240`.
     - **Mastercard Business / Corporate:** `5168`, `5218`, `5440`, `5521`.
   - The onboarding setup checks and validates card BIN rules against this dictionary.

#### REST API Contract
- `POST /v1/supplier/onboarding/payment`
- Request Payload:
```json
{
  "cash_enabled": true,
  "gateways": [
    {
      "gateway_type": "GLOBAL_PAY",
      "is_enabled": true,
      "service_id": "gp_srv_992140",
      "secret_key": "live_sec_d83f0192a8b94ce",
      "username": "pepsico_b2b_merchant",
      "password": "SuperSecretB2BPassword123!",
      "enforce_corporate_bin": true,
      "allowed_card_bins": ["5614", "9860", "5440", "4073", "5168"]
    }
  ]
}
```
- Response (HTTP 200 OK):
```json
{
  "status": "configured",
  "cash_enabled": true,
  "active_gateways": ["CASH", "GLOBAL_PAY"],
  "corporate_bin_validation_active": true,
  "step_2_completed": true,
  "next_step": "/v1/supplier/onboarding/complete"
}
```

---

### 3.3 Step 3: Onboarding Completion (`POST /v1/supplier/onboarding/complete`)

#### Execution Sequence & Atomic Mutation
1. **Precondition Audit:**
   - Active SKU check: `SELECT COUNT(*) FROM skus WHERE supplier_id = $1 AND is_active = TRUE`. Count must be $\ge 1$.
   - Payment setup check: Verify cash enabled or valid `GLOBAL_PAY` gateway configured.
2. **Atomic State Transition in PostgreSQL:**
   ```sql
   UPDATE suppliers 
   SET onboarding_status = 'COMPLETED', updated_at = NOW() 
   WHERE supplier_id = $1;

   UPDATE supplier_profiles 
   SET onboarding_status = 'COMPLETED', is_configured = TRUE, updated_at = NOW() 
   WHERE supplier_id = $1;
   ```
3. **Transactional Outbox Event Emission:**
   - Atomic insertion inside active `pgx.Tx`:
     - Aggregate Type: `SUPPLIER`
     - Aggregate ID: `supplier_id`
     - Event Type: `supplier.onboarding_completed`
     - Payload: `{ "supplier_id": sid, "status": "COMPLETED", "completed_at": now }`
4. **Cache Eviction & Real-time Fanout:**
   - Invalidate Redis cache: `DEL cache:supplier:onboarding:<supplierID>`.
   - Publish to Redis 7 Stream: `XADD stream:supplier:events * event_type supplier.onboarding_completed ...`
   - Publish to Redis Pub/Sub: `events:SUPPLIER`.
   - WebSocket Hub Broadcast: `wsHub.BroadcastEnvelope("supplier.onboarding_completed", payload)`.

#### REST API Contract
- `POST /v1/supplier/onboarding/complete`
- Request Payload: `{}` (or optional `{ "final_notes": "All initial SKUs loaded" }`)
- Response (HTTP 200 OK):
```json
{
  "status": "success",
  "supplier_id": "sup_4a91cf02",
  "onboarding_status": "COMPLETED",
  "gate_unlocked": true,
  "redirect_url": "/dashboard",
  "message": "Supplier onboarding successfully completed. Full logistics and dispatch access granted."
}
```

---

## 4. Warehouse & Fleet Management Hub

### 4.1 Warehouse CRUD & Invariants (`/v1/supplier/warehouses`)

#### Endpoints
- `POST /v1/supplier/warehouses` (Create warehouse)
- `GET /v1/supplier/warehouses` (List supplier warehouses)
- `GET /v1/supplier/warehouses/{id}` (Get warehouse detail)
- `PUT /v1/supplier/warehouses/{id}` (Update warehouse details/coordinates)
- `DELETE /v1/supplier/warehouses/{id}` (Delete warehouse with safety guard)

#### Mandatory Geolocation Coordinates
- `latitude` and `longitude` are required `DOUBLE PRECISION` fields.
- Valid geographic bounding for Uzbekistan operations:
  - Latitude: $37.0 \le \text{lat} \le 45.6$
  - Longitude: $56.0 \le \text{lng} \le 73.2$

#### Redis Proximity Cache Invalidation & Event Emission
When coordinates are updated or warehouse relocated:
1. **Redis Geospatial Index Update:**
   ```go
   rdb.GeoAdd(ctx, "warehouses:locations", &redis.GeoLocation{
       Name: warehouseID,
       Latitude: newLat,
       Longitude: newLng,
   })
   ```
2. **Proximity Cache Invalidation:**
   - Scan & Delete: `DEL cache:warehouse:proximity:*` or keys tracking distance matrices.
3. **Emit `warehouse.relocated`:**
   - Outbox: `outbox.Emit(ctx, tx, "WAREHOUSE", warehouseID, "warehouse.relocated", payload)`
   - Redis Stream: `stream:warehouse:events`
   - WebSocket broadcast to all connected clients.

#### Warehouse Deletion Guard (HTTP 409 Conflict)
Before allowing `DELETE /v1/supplier/warehouses/{id}`, execute two safety checks:
1. **On-Hand Inventory Check:**
   ```sql
   SELECT COALESCE(SUM(quantity_on_hand), 0) 
   FROM stock_balances 
   WHERE warehouse_id = $1;
   ```
   If stock $> 0$, fail with HTTP 409 Conflict.
2. **Active Orders Check:**
   ```sql
   SELECT COUNT(*) 
   FROM orders 
   WHERE warehouse_id = $1 AND status NOT IN ('DELIVERED', 'CANCELLED');
   ```
   If active orders $> 0$, fail with HTTP 409 Conflict.
3. **Error Response (RFC 7807):**
```json
{
  "type": "urn:pegasusx:error:conflict",
  "title": "Conflict",
  "status": 409,
  "detail": "Warehouse cannot be deleted: 1,450 units on-hand stock and 3 active dispatch orders exist."
}
```

---

### 4.2 Fleet Vehicles / Trucks (`/v1/supplier/warehouses/{id}/trucks`)

- **POST `/v1/supplier/warehouses/{id}/trucks`:**
  - Validates Uzbekistan plate format (`license_plate`).
  - Payload capacity in kilograms (`payload_capacity_kg`) and volume capacity in cubic meters / VU (`max_volume_vu` / `max_volume_cbm`).
  - Fuel type enum: `METHANE_CNG`, `PROPANE_LPG`, `DIESEL`, `PETROL`, `ELECTRIC`.
  - Persists into `vehicles` table (`025_fleet_and_driver_lifecycle_management.sql:9-38`).
- **GET `/v1/supplier/warehouses/{id}/trucks`:**
  - Queries `vehicles` filtered by `warehouse_id` and `supplier_id`.
  - Returns vehicle fleet asset cards with operational status (`YARD_STANDBY`, `LOADING_AT_DOCK`, `ACTIVE_ON_ROAD`, `MAINTENANCE`).

---

### 4.3 Payloaders Management (`/v1/supplier/warehouses/{id}/payloaders`)

- **Schema Migration (Migration 069):**
  Replaces in-memory map (`onboarding/service.go:98`) with relational PostgreSQL table:
  ```sql
  CREATE TABLE IF NOT EXISTS payloaders (
      payloader_id VARCHAR(64) PRIMARY KEY,
      supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id),
      warehouse_id VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id),
      name VARCHAR(255) NOT NULL,
      phone VARCHAR(32) NOT NULL,
      status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'ON_LEAVE'
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  CREATE INDEX IF NOT EXISTS idx_payloaders_wh ON payloaders (warehouse_id, status);
  CREATE INDEX IF NOT EXISTS idx_payloaders_sup ON payloaders (supplier_id);
  ```
- **Endpoints:**
  - `POST /v1/supplier/warehouses/{id}/payloaders`: Registers a loading bay operator.
  - `GET /v1/supplier/warehouses/{id}/payloaders`: Returns dock loading staff assigned to warehouse.

---

## 5. Realtime & Outbox Architecture

### 5.1 Transactional Outbox Pipeline
```
[HTTP / Domain Mutation]
       │
       ▼
[Active pgx.Tx Transaction]
 ├── INSERT / UPDATE Domain Row (skus, suppliers, warehouses)
 └── outbox.Emit(ctx, tx, "SUPPLIER", sid, "supplier.onboarding_completed", payload)
       │  (INSERT INTO outbox_events, published = FALSE)
       ▼
[Transaction Commit]
       │
       ▼
[Background outbox.RelayWorker] (500ms ticker, FOR UPDATE SKIP LOCKED)
 ├── SELECT * FROM outbox_events WHERE NOT published LIMIT 50
 ├── Redis 7 Streams: XADD stream:supplier:events (MaxLen 100k)
 ├── Redis Pub/Sub: PUBLISH events:SUPPLIER
 └── UPDATE outbox_events SET published = TRUE, published_at = NOW()
       │
       ▼
[WebSocket Hub] (subscribeRedisChannels)
 └── BroadcastEnvelope("supplier.onboarding_completed", payload)
       │ (Monotonic Sequence ++, Ring Buffer 2000 events)
       ▼
[Connected Frontend Clients via /v1/ws]
```

### 5.2 Code Citations
- **Outbox Table Definition:** `database/migrations/002_ump_and_outbox.sql:31-40`
- **Outbox Atomic Emitter:** `internal/outbox/emitter.go:12-28`
- **Outbox Polling Relay Worker:** `internal/outbox/relay.go:17-140`
  - Polling query: `relay.go:60-67` (`FOR UPDATE SKIP LOCKED`)
  - Stream publishing: `relay.go:98-111` (`XADD stream:%s:events`)
  - Pub/Sub notification: `relay.go:121-124` (`PublishEvent(ctx, channel, payload)`)
- **WebSocket Hub Hub Implementation:** `internal/ws/hub.go:40-269`
  - Monotonic sequence numbering: `hub.go:111` (`atomic.AddInt64(&h.seq, 1)`)
  - Redis channel subscriptions: `hub.go:167`
  - Client connection handler: `hub.go:198-214` (`/v1/ws`)

---

## 6. PostgreSQL Migration Specification (069)

To support pure PostgreSQL 16 persistence without mock data or memory repositories, the following DDL script must be placed in `database/migrations/069_supplier_onboarding_and_globalpay.sql`:

```sql
-- ==============================================================================
-- PEGASUS.X MIGRATION 069: SUPPLIER ONBOARDING, GLOBAL PAY & WAREHOUSE FLEET HUB
-- Storage Engine: PostgreSQL 16
-- Monetary Invariant: Strict 64-bit integer minor currency units (tiyins / cents)
-- ==============================================================================

-- 1. ENHANCE SUPPLIERS TABLE WITH LEGAL UNIQUENESS & ONBOARDING STATUS
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS phone VARCHAR(32);
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS onboarding_status VARCHAR(32) NOT NULL DEFAULT 'PENDING';
-- States: 'PENDING', 'PRODUCTS_PENDING', 'PAYMENT_PENDING', 'COMPLETED'
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Enforce strict Uzbekistan STIR/INN uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_suppliers_legal_tax_id ON suppliers (legal_tax_id);

-- 2. ENHANCE SUPPLIER PROFILES FOR RUNTIME PARITY
ALTER TABLE supplier_profiles ADD COLUMN IF NOT EXISTS onboarding_status VARCHAR(32) NOT NULL DEFAULT 'PENDING';
ALTER TABLE supplier_profiles ADD COLUMN IF NOT EXISTS is_configured BOOLEAN NOT NULL DEFAULT FALSE;

-- 3. SUPPLIER PAYMENT GATEWAYS TABLE
CREATE TABLE IF NOT EXISTS supplier_payment_gateways (
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    gateway_type VARCHAR(32) NOT NULL, -- 'CASH', 'GLOBAL_PAY'
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    service_id VARCHAR(128),
    secret_key_enc TEXT,
    merchant_username VARCHAR(128),
    merchant_password_enc TEXT,
    allowed_card_bins TEXT[] DEFAULT ARRAY['5614', '9860', '5440', '4073', '5168'],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (supplier_id, gateway_type)
);

CREATE INDEX IF NOT EXISTS idx_supplier_payment_gateways_sup 
    ON supplier_payment_gateways (supplier_id, is_enabled);

-- 4. RELATIONAL PAYLOADERS TABLE (REPLACES IN-MEMORY MAP)
CREATE TABLE IF NOT EXISTS payloaders (
    payloader_id VARCHAR(64) PRIMARY KEY,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    warehouse_id VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'ON_LEAVE'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payloaders_wh_status ON payloaders (warehouse_id, status);
CREATE INDEX IF NOT EXISTS idx_payloaders_sup ON payloaders (supplier_id);
```

---

## 7. Synthesis & Implementation Checklist for Teamwork Implementers

| Component | Target File | Action Required |
|---|---|---|
| Middleware | `internal/auth/middleware.go` | Implement `RequireSupplierOnboardingCompleted` returning HTTP 428. |
| Router Wiring | `internal/api/router.go` | Mount middleware on protected supplier route group; add `/v1/supplier/onboarding/*` routes. |
| Products Onboarding | `internal/api/handlers_onboarding_products.go` | Validate 17-digit MXIK, EAN-13 modulo 10 checksum, 64-bit tiyin price, 12% VAT. |
| Payment Onboarding | `internal/api/handlers_onboarding_payment.go` | Configure Cash + Global Pay with Uzbekistan B2B corporate card BIN checks. |
| Complete Onboarding | `internal/api/handlers_onboarding_complete.go` | Transition status to `'COMPLETED'`, emit `outbox.Emit` + Redis Stream + WS envelope. |
| Warehouse CRUD | `internal/api/handlers_supplier_warehouses.go` | Full CRUD with mandatory lat/lon, Redis GeoAdd & proximity cache DEL, deletion stock guard (HTTP 409). |
| Trucks / Fleet | `internal/api/handlers_supplier_warehouses.go` | Wire `POST/GET /v1/supplier/warehouses/{id}/trucks` directly to `fleet.Service`. |
| Payloaders | `internal/api/handlers_supplier_warehouses.go` | Wire `POST/GET /v1/supplier/warehouses/{id}/payloaders` to PostgreSQL `payloaders` table. |
| DB Migration | `database/migrations/069_supplier_onboarding_and_globalpay.sql` | Create migration DDL with unique STIR, `supplier_payment_gateways`, and `payloaders`. |
| Purge Mock Seeds | `internal/supplier/repository.go` | Remove `MemoryRepository` fallback from `PostgresRepository`. |

All components are strictly architected to preserve the zero-cross-contamination rule between `pegasusX` (Spanner/Kafka) and `pegasus.x` (PostgreSQL 16/Redis 7).
