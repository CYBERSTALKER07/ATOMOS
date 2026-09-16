# Architectural Survey Report: Supplier Domain, Mock Purge & Authentication

**Date**: 2026-09-16  
**Auditor**: Teamwork Explorer (Survey Specialist 1: Supplier Domain & Mock Purge)  
**Target Repository**: `pegasus.x/backend` (Sovereign Lean Single-Tenant / National Operating Core)  
**Database**: PostgreSQL 16 (`pgx/v5` connection pool) + Redis 7  
**Monetary Invariant**: Strict 64-bit integer minor currency units (tiyins / cents; 1 UZS = 100 tiyins)

---

## 1. Executive Summary

A line-by-line architectural survey of `pegasus.x/backend/internal/supplier/`, `pegasus.x/backend/internal/api/handlers_supplier.go`, and related auth components was performed to prepare for the end-to-end implementation of:
1. Minimal Supplier Sign-Up & Sign-In with STIR deduplication and bcrypt password hashing.
2. Non-bypassable phased onboarding wizard (`/v1/supplier/onboarding/*`).
3. Post-onboarding Warehouse & Fleet Management hub.
4. Total purge of `MemoryRepository`, in-memory fallback maps, and hardcoded mock seeds.

### Key Critical Findings:
1. **Mock Fallback Anti-Pattern**: `internal/supplier/repository.go` contains a 934-line `MemoryRepository` pre-seeded with fake Tashkent data (`sup_pepsico_uz`, fake KYC PDFs, fake topology nodes, fake org members, fake pricing rules, fake Korzinka overrides). Even more dangerously, `PostgresRepository` embeds `memory *MemoryRepository` and executes a **silent error fallback** (`if err != nil { return p.memory... }`) on **every single SQL query**, along with dual-writing to memory. Seven repository methods do not even contain SQL queries at all, delegating 100% to memory!
2. **Dual-Table Architectural Split & Missing Root Writes**: Migration `001_initial_schema.sql` defines the root multi-tenant table `suppliers (supplier_id, name, legal_tax_id)`, which is the foreign key target for `warehouses`, `drivers`, `retailers`, `skus`, and `orders`. However, migration `058_supplier_portal_core_and_operations.sql` created `supplier_profiles`. Currently, `handleSupplierRegister` inserts **only** into `supplier_profiles` (or falls back to memory), never creating a row in `suppliers`. Consequently, any subsequent foreign key reference to `suppliers(supplier_id)` fails in PostgreSQL.
3. **Authentication Theatre**:
   - `handleSupplierRegister` (`handlers_supplier.go:89-162`) accepts a JSON payload, completely ignores `password`, never hashes with bcrypt, does not validate 9-digit Uzbekistan STIR, does not check database uniqueness, and never returns HTTP 409 Conflict.
   - `handleSupplierLogin` (`handlers_supplier.go:169-208`) completely ignores `password`, never calls `bcrypt.CompareHashAndPassword`, hardcodes `sid := "sup_pepsico_uz"`, generates a dummy token, and returns `is_configured` instead of `onboarding_status`.
4. **Missing Schema Elements**: PostgreSQL `suppliers` table currently lacks `phone`, `password_hash`, and `onboarding_status` columns, and lacks a `UNIQUE` constraint on `legal_tax_id`. There are no database tables for supplier payment gateway configuration (`Cash` / `Global Pay`), KYC documents, or payloader dock staff.

---

## 2. Line-by-Line Mock Purge Audit (`internal/supplier/repository.go`)

`pegasus.x/backend/internal/supplier/repository.go` is 1,819 lines long. Over 56% of this file is mock data, in-memory structures, or fallback logic.

### 2.1 In-Memory Data Structures (Lines 90–106)
```go
// Line 90-106
type MemoryRepository struct {
    mu          sync.RWMutex
    profiles    map[string]SupplierProfile
    topology    map[string]map[string]TopologyNode            // supplierID -> nodeID -> Node
    orgMembers  map[string]map[string]OrgMember               // supplierID -> userID -> Member
    pricing     map[string]PricingRule                        // supplierID -> PricingRule
    overrides   map[string]map[string]RetailerPricingOverride // supplierID -> overrideID -> Override
    vetLogs     map[string][]OrderVetLog                      // supplierID -> []OrderVetLog
    policies    map[string]ServicePolicy                      // supplierID -> ServicePolicy
    breaches    map[string][]ServicePromiseBreach             // supplierID -> []ServicePromiseBreach
    aiRecs      map[string]map[string]AIRecommendation        // supplierID -> recID -> Rec
    imports     map[string]map[string]ImportSession           // supplierID -> sessionID -> Session
    crmRetailers map[string]map[string]CRMRetailerSummary     // supplierID -> retailerID -> CRM
    events       map[string][]map[string]interface{}          // supplierID -> []events
    kycDocs      map[string][]KycDocument                     // supplierID -> []KycDocument
}
```

### 2.2 Hardcoded Mock Seeds in `NewMemoryRepository()` (Lines 108–416)
- **Lines 127–157**: Seeds fake supplier profiles for `"sup_pepsico_uz"` and `"sup_tashkent_beverage"`:
  - Name: `"PepsiCo International Distribution LLC"`
  - Tax ID: `"302918274"`
  - Bank: `"Ipak Yoli Bank HQ"`, MFO `"00444"`, Account `"20208000900123456001"`
  - Status: `"VERIFIED"`, `IsConfigured: true`
- **Lines 159–220**: Seeds 5 fake KYC documents per supplier pointing to fake URLs:
  - `kyc_doc_guvohnoma_...`: `"https://storage.pegasus.internal/kyc/guvohnoma_pepsico.pdf"`
  - `kyc_doc_tax_...`: `"https://storage.pegasus.internal/kyc/soliq_cert_pepsico.pdf"`
  - `kyc_doc_bank_...`: `"https://storage.pegasus.internal/kyc/bank_letter_ipak_yoli.pdf"`
  - `kyc_doc_director_...`: `"https://storage.pegasus.internal/kyc/director_passport_pinfl.pdf"`
  - `kyc_doc_ses_...`: `"https://storage.pegasus.internal/kyc/ses_sanpin_cert.pdf"`
- **Lines 223–252**: Seeds fake topology nodes:
  - `node_fac_yangiyol`: Yangiyo'l FMCG Bottling Plant (41.1352, 69.0514)
  - `node_wh_sergeli`: Sergeli Central Fulfillment Center (41.2285, 69.2144)
- **Lines 254–280**: Seeds fake org members:
  - `usr_admin_01`: Temur Rustamov (`ADMIN`)
  - `usr_dispatcher_01`: Dilshod Karimov (`DISPATCHER`)
- **Lines 282–297**: Seeds fake pricing rule: `"Standard FMCG Wholesale Schedule"`, min order 500,000 UZS, 14.5% markup.
- **Lines 299–321**: Seeds fake retailer pricing override: `ovr_korzinka` for `ret_korzinka_chain` (Korzinka), fixed price 6,800,000 tiyins, 5.0% discount.
- **Lines 323–333**: Seeds fake service policy: `pol_standard_...`, target SLA 24h, 98.5% fill rate, 150 BPS penalty.
- **Lines 335–370**: Seeds fake AI recommendations: `rec_01` (Rebalance Pepsi 1.5L from Yangiyo'l to Sergeli), `rec_02` (4% volume tier for Yunusobod).
- **Lines 372–405**: Seeds fake CRM retailer summaries: `ret_oqtepa_01` (Oq-Tepa Savdo Markazi, 485M UZS volume), `ret_yunusobod_02` (Yunusobod Fayz Supermarket).
- **Lines 407–412**: Seeds fake live audit events (`e1`–`e4`): auto-dispatch, delivery OTP, Soliq e-invoice, Korzinka discount.

### 2.3 Pure In-Memory Method Implementations (Lines 418–1022)
- Lines 418–1022 implement all 36 methods of `Repository` on `*MemoryRepository` using mutex read/write locks and map operations.

### 2.4 PostgresRepository Dual-State & Silent Fallback Anti-Pattern (Lines 1024–1790)
```go
// Lines 1024-1038
type PostgresRepository struct {
    pool   *db.Pool
    memory *MemoryRepository
}

func NewRepository(pool *db.Pool) Repository {
    mem := NewMemoryRepository()
    if pool == nil {
        return mem
    }
    return &PostgresRepository{
        pool:   pool,
        memory: mem,
    }
}
```
Every single query has a silent fallback to `p.memory`, and every write operation dual-writes to `p.memory`. Exact citations:
- `GetProfile`: line 1056 (`if err != nil { return p.memory.GetProfile(ctx, supplierID) }`)
- `SaveProfile`: lines 1093, 1095 (`if err != nil { return p.memory.SaveProfile(ctx, profile) }`, `_, _ = p.memory.SaveProfile(ctx, profile)`)
- `ListTopologyNodes`: lines 1109, 1121, 1126
- `GetTopologyNode`: line 1145
- `SaveTopologyNode`: lines 1175, 1177
- `DeleteTopologyNode`: lines 1185, 1187
- `ListOrgMembers`: lines 1200, 1211, 1216
- `GetOrgMember`: line 1233
- `SaveOrgMember`: lines 1259, 1261
- `DeleteOrgMember`: lines 1269, 1271
- `GetPricingRule`: line 1292
- `SavePricingRule`: lines 1325, 1327
- `ListRetailerPricingOverrides`: lines 1342, 1354, 1359
- `GetRetailerPricingOverride`: line 1379
- `SaveRetailerPricingOverride`: lines 1410, 1412
- `DeleteRetailerPricingOverride`: lines 1420, 1422
- `SaveOrderVetLog`: lines 1441, 1443
- `ListOrderVetLogs`: lines 1461, 1473, 1481
- `GetServicePolicy`: line 1501
- `SaveServicePolicy`: lines 1527, 1529
- `SavePromiseBreach`: lines 1548, 1550
- `ListPromiseBreaches`: lines 1568, 1579, 1584
- `ListAIRecommendations`: lines 1599, 1612, 1620
- `GetAIRecommendation`: line 1640
- `SaveAIRecommendation`: lines 1671, 1673
- `UpdateAIRecommendationStatus`: lines 1693, 1698
- `SaveImportSession`: lines 1724, 1726
- `GetImportSession`: line 1744
- `ListImportSessions`: lines 1766, 1778, 1786

### 2.5 Methods with ZERO PostgreSQL Queries (Pure Mock Delegation) (Lines 1791–1818)
The following 7 methods in `PostgresRepository` do not execute any SQL at all:
```go
// Line 1791-1794
func (p *PostgresRepository) ListCRMRetailers(ctx context.Context, supplierID string) ([]CRMRetailerSummary, error) {
    return p.memory.ListCRMRetailers(ctx, supplierID)
}

// Line 1796-1798
func (p *PostgresRepository) GetCRMRetailerDetail(ctx context.Context, supplierID, retailerID string) (*CRMRetailerSummary, error) {
    return p.memory.GetCRMRetailerDetail(ctx, supplierID, retailerID)
}

// Line 1800-1802
func (p *PostgresRepository) AddEvent(ctx context.Context, supplierID string, event map[string]interface{}) error {
    return p.memory.AddEvent(ctx, supplierID, event)
}

// Line 1804-1806
func (p *PostgresRepository) ListEvents(ctx context.Context, supplierID string, limit int) ([]map[string]interface{}, error) {
    return p.memory.ListEvents(ctx, supplierID, limit)
}

// Line 1808-1810
func (p *PostgresRepository) ListKycDocuments(ctx context.Context, supplierID string) ([]KycDocument, error) {
    return p.memory.ListKycDocuments(ctx, supplierID)
}

// Line 1812-1814
func (p *PostgresRepository) SubmitKycDocument(ctx context.Context, doc KycDocument) (*KycDocument, error) {
    return p.memory.SubmitKycDocument(ctx, doc)
}

// Line 1816-1818
func (p *PostgresRepository) ReviewKycDocument(ctx context.Context, supplierID, docID, status, reviewedBy, rejectionReason string) (*KycDocument, error) {
    return p.memory.ReviewKycDocument(ctx, supplierID, docID, status, reviewedBy, rejectionReason)
}
```

### 2.6 Mock Seeds in Tests (`internal/supplier/supplier_test.go`)
- Lines 10, 58, 106, 154, 217, 256, 300, 349, 417: All 9 test functions instantiate `NewMemoryRepository()` directly and rely on pre-seeded `"sup_pepsico_uz"`.

---

## 3. Existing Models, Domain Representations & Gap Analysis

| Concept | Existing Location | Database Table | Status & Deficiencies |
| :--- | :--- | :--- | :--- |
| **Supplier (Tenancy Root)** | `internal/models/domain.go:28-35` (`models.Supplier`) | `suppliers` (`001_initial_schema.sql`) | Missing `phone`, `password_hash`, `onboarding_status`, `updated_at`. Lacks `UNIQUE(legal_tax_id)`. Bypassed during registration! |
| **Supplier Profile (Portal)** | `internal/supplier/models.go:22-52` (`SupplierProfile`) | `supplier_profiles` (`058_...sql`) | Contains business address, bank accounts, and `kyc_status`. Not linked transactionally to `suppliers`. |
| **SKU / Catalog Product** | `internal/models/domain.go:75-120` (`models.SKU`) | `skus` (`001`, `059`, `063`, `064`) | Full schema exists for tiyins, MXIK, barcode, pack types. Currently created via `handleCreateProduct`, but no dedicated `/v1/supplier/onboarding/products` wizard route. |
| **Packaging Hierarchy** | `internal/models/domain.go:122-135` (`ProductPackagingUnit`) | `product_packaging_units` (`064_...sql`) | Schema exists for flexible units (PCE, PACK, CASE, PALLET). |
| **Payment Configs** | None | None | No schema or models for Cash + Global Pay setup with corporate card BIN limits. |
| **KYC Documents** | `internal/supplier/models.go:7-20` (`KycDocument`) | None | No database table exists in PostgreSQL migrations 001–068. Pure in-memory mock. |
| **Payloaders (Dock Staff)** | `internal/onboarding/service.go:78-86` (`PayloaderProfile`) | None | In-memory map in `onboarding.Service`. No PostgreSQL table in migrations 001–068. |
| **Fleet / Vehicles** | `internal/fleet/models.go` (`Vehicle`) | `vehicles` (`025_...sql`) | Exists in PostgreSQL. Has `license_plate`, `payload_capacity_kg`, `max_volume_vu`, `fuel_type`, `warehouse_id`. |
| **Onboarding Lifecycle** | `internal/onboarding/lifecycle.go` | None | In-memory `LifecycleManager` with CRO step percentages. Not enforced at API gateway. |
| **JWT User Claims** | `internal/models/claims.go:19-30` (`UserClaims`) | N/A | Has `UserID`, `SupplierID`, `Role`, `IsConfigured`. Missing `OnboardingStatus`. |

---

## 4. Current Authentication Audit

### 4.1 Route Declarations (`internal/api/router.go:371–373`)
```go
r.Post("/v1/auth/supplier/login", s.handleSupplierLogin)
r.Post("/v1/auth/supplier/register", s.handleSupplierRegister)
r.Post("/v1/auth/supplier/refresh", s.handleSupplierRefresh)
```

### 4.2 `handleSupplierRegister` Analysis (`internal/api/handlers_supplier.go:89–162`)
1. **Payload Deficiencies**:
   ```go
   type SupplierRegisterPayload struct {
       Account struct {
           LegalName   string `json:"legalName"`
           ContactName string `json:"contactName"`
           Email       string `json:"email"`
           Country     string `json:"country"`
           Phone       string `json:"phone"`
       } `json:"account"`
       LegalName string `json:"legal_name"`
       Phone     string `json:"phone"`
       Email     string `json:"email"`
       IDToken   string `json:"id_token"`
       Password  string `json:"password"`
   }
   ```
   - Does not parse `company_name` or `tax_id` (STIR) from top-level payload.
   - Defaults empty fields: if `legalName == ""` it defaults to `"New Uzbekistan Supplier LLC"`, if `phone == ""` it defaults to `"+998901234567"`.
2. **Missing Validation & STIR Deduplication**:
   - Zero STIR format validation (requires 9 numeric digits).
   - Zero PostgreSQL deduplication check. Duplicate tax IDs are silently accepted.
   - Never returns HTTP 409 Conflict.
3. **Password Hashing Omission**:
   - `req.Password` is received but completely ignored.
   - `bcrypt.GenerateFromPassword` is NEVER called.
   - No password hash is stored anywhere.
4. **Broken Foreign Key Invariant**:
   - Executes `s.supplierSvc.UpdateProfile(prof)`, which only writes to `supplier_profiles` (or falls back to `MemoryRepository`).
   - Does NOT insert a record into the root `suppliers` table (`001_initial_schema.sql`).
   - If any warehouse or product is subsequently inserted referencing this `supplier_id`, PostgreSQL throws: `foreign key constraint "warehouses_supplier_id_fkey" violated`.
5. **Onboarding State Deficiencies**:
   - Sets `KycStatus: "PENDING_SUBMISSION"`.
   - Returns `next_step: "/setup/business"` instead of `"/onboarding/products"`.
   - Returns `is_registered: true, is_configured: false`, omitting `onboarding_status: "PENDING"`.

### 4.3 `handleSupplierLogin` Analysis (`internal/api/handlers_supplier.go:169–208`)
1. **Hardcoded Tenant Mock**:
   ```go
   // Line 181
   sid := "sup_pepsico_uz"
   prof, err := s.supplierSvc.GetProfile(r.Context(), sid)
   ```
   Regardless of what phone or password is sent in the request, it always logs in as `"sup_pepsico_uz"`!
2. **Password Verification Omission**:
   - Completely ignores `req.Password`.
   - Never calls `bcrypt.CompareHashAndPassword`.
   - Anyone providing any password (or empty password) succeeds.
3. **Missing Database Lookup**:
   - Does not query `suppliers` or `supplier_profiles` by phone or tax ID.
4. **Missing Onboarding Status**:
   - Returns `next_step: "/dashboard"` or `"/setup/business"`.
   - Does not return `onboarding_status`.

---

## 5. Concrete Implementation Blueprint for Worker

To achieve full compliance with requirements R1, R2, R3, R4 and ensure zero regressions, the worker must implement the following architectural components:

### 5.1 Step 1: PostgreSQL Migration `069_supplier_onboarding_and_globalpay.sql`
File location: `pegasus.x/database/migrations/069_supplier_onboarding_and_globalpay.sql`

```sql
-- ==============================================================================
-- PEGASUS.X MIGRATION 069: SUPPLIER ONBOARDING, DEDUPLICATION & GLOBALPAY GATEWAY
-- Storage Engine: PostgreSQL 16
-- Monetary Invariant: Strict 64-bit integer minor currency units (tiyins / cents)
-- ==============================================================================

-- 1. ENHANCE ROOT SUPPLIERS TABLE FOR AUTH & ONBOARDING STATUS
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS phone VARCHAR(32);
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS onboarding_status VARCHAR(32) NOT NULL DEFAULT 'PENDING';
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Enforce strict STIR uniqueness in PostgreSQL
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_suppliers_legal_tax_id'
    ) THEN
        ALTER TABLE suppliers ADD CONSTRAINT uq_suppliers_legal_tax_id UNIQUE (legal_tax_id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_suppliers_phone ON suppliers (phone);
CREATE INDEX IF NOT EXISTS idx_suppliers_onboarding_status ON suppliers (onboarding_status);

-- 2. SUPPLIER PAYMENT CONFIGURATIONS (CASH + GLOBAL PAY B2B GATEWAY)
CREATE TABLE IF NOT EXISTS supplier_payment_configs (
    config_id VARCHAR(64) PRIMARY KEY,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    cash_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    global_pay_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    global_pay_service_id VARCHAR(128),
    global_pay_secret_key VARCHAR(255),
    allowed_card_bins TEXT[] DEFAULT ARRAY['8600', '9860', '5614'], -- UZCARD / HUMO B2B corporate card BINs
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_supplier_payment_configs_sup UNIQUE (supplier_id)
);

-- 3. SUPPLIER KYC DOCUMENTS TABLE (PURGING IN-MEMORY MOCKS)
CREATE TABLE IF NOT EXISTS supplier_kyc_documents (
    document_id VARCHAR(64) PRIMARY KEY,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    document_type VARCHAR(64) NOT NULL, -- GUVOHNOMA, TAX_CERTIFICATE, BANK_LETTER, DIRECTOR_ID, SANITARY_SES
    document_number VARCHAR(128) NOT NULL,
    document_url TEXT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING', -- PENDING, VERIFIED, REJECTED
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by VARCHAR(128),
    rejection_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_supplier_kyc_sup ON supplier_kyc_documents (supplier_id, status);

-- 4. SUPPLIER AUDIT & TELEMETRY EVENTS TABLE (PURGING IN-MEMORY EVENTS)
CREATE TABLE IF NOT EXISTS supplier_audit_events (
    event_id VARCHAR(64) PRIMARY KEY,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    description TEXT NOT NULL,
    color VARCHAR(64) DEFAULT 'bg-desk-accent text-white',
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_events_sup ON supplier_audit_events (supplier_id, created_at DESC);

-- 5. WAREHOUSE LOADING DOCK PAYLOADERS TABLE
CREATE TABLE IF NOT EXISTS payloaders (
    payloader_id VARCHAR(64) PRIMARY KEY,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    warehouse_id VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, ON_LEAVE, SUSPENDED
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payloaders_wh ON payloaders (warehouse_id, status);
CREATE INDEX IF NOT EXISTS idx_payloaders_sup ON payloaders (supplier_id);
```

### 5.2 Step 2: Pure PostgreSQL 16 Persistence in `internal/supplier/repository.go`
1. Delete `type MemoryRepository struct` and its entire implementation (lines 90–1022).
2. Refactor `PostgresRepository`:
   ```go
   type PostgresRepository struct {
       pool *db.Pool
   }

   func NewRepository(pool *db.Pool) Repository {
       if pool == nil {
           panic("db.Pool is required for supplier.PostgresRepository")
       }
       return &PostgresRepository{pool: pool}
   }
   ```
3. Remove all `p.memory` calls and dual-writes. Return true errors:
   ```go
   if err != nil {
       return nil, fmt.Errorf("failed to query supplier profile: %w", err)
   }
   ```
4. Implement pure PostgreSQL queries for the 7 unwritten methods:
   - `ListCRMRetailers`:
     ```sql
     SELECT r.retailer_id, r.name, r.name, r.legal_tax_id, r.phone, 'Tashkent' AS district,
            'GOLD' AS tier, COALESCE(SUM(o.effective_total_minor), 0) AS lifetime_volume_minor,
            COUNT(o.order_id)::INT AS total_orders, 2500000000::BIGINT AS credit_limit_minor,
            0::BIGINT AS current_debt_minor, 'ACTIVE' AS status, MAX(o.created_at) AS last_order_at
     FROM retailers r
     LEFT JOIN orders o ON r.retailer_id = o.retailer_id AND o.supplier_id = r.supplier_id
     WHERE r.supplier_id = $1
     GROUP BY r.retailer_id, r.name, r.legal_tax_id, r.phone
     ```
   - `ListKycDocuments`, `SubmitKycDocument`, `ReviewKycDocument`: Target `supplier_kyc_documents`.
   - `AddEvent`, `ListEvents`: Target `supplier_audit_events`.

### 5.3 Step 3: Minimal Supplier Sign-Up & Sign-In Implementation
Update `internal/api/handlers_supplier.go`:

1. **`handleSupplierRegister`**:
   - Request Body:
     ```json
     {
       "company_name": "ООО Toshkent Distribyutsiya",
       "tax_id": "301234567",
       "phone": "+998901234567",
       "password": "SecurePassword123!"
     }
     ```
   - Validation:
     - `company_name`: `strings.TrimSpace(req.CompanyName) != ""` (HTTP 400).
     - `tax_id`: `regexp.MatchString(`^\d{9}$`, req.TaxID)` (HTTP 400: `"tax_id must be exactly 9 digits"`).
     - `phone`: `strings.HasPrefix(phone, "+998") && len(phone) == 13` (HTTP 400).
     - `password`: `len(req.Password) >= 8` (HTTP 400: `"password must be at least 8 characters"`).
   - STIR Uniqueness Check:
     - Query: `SELECT COUNT(*) FROM suppliers WHERE legal_tax_id = $1`
     - If count > 0: return HTTP 409 Conflict (`{"error": "conflict", "message": "Supplier with STIR already registered"}`).
   - Password Hashing:
     - `hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)`
   - Atomic Persistence (`pool.RunInTx`):
     - Insert into `suppliers`: `(supplier_id, name, legal_tax_id, phone, password_hash, onboarding_status, currency, created_at, updated_at)` values `(sid, req.CompanyName, req.TaxID, req.Phone, string(hash), 'PENDING', 'UZS', now, now)`.
     - Insert into `supplier_profiles`: `(supplier_id, company_name, legal_entity_name, tax_id_inn, primary_phone, primary_email, headquarters_address, region, is_active, is_registered, is_configured, kyc_status, created_at, updated_at)` values `(sid, req.CompanyName, req.CompanyName, req.TaxID, req.Phone, req.Email, '', 'Tashkent City', true, true, false, 'PENDING_SUBMISSION', now, now)`.
     - Insert default payment config into `supplier_payment_configs`: `(config_id, supplier_id, cash_enabled, global_pay_enabled, allowed_card_bins, created_at, updated_at)` values `(uuid, sid, true, false, ARRAY['8600', '9860', '5614'], now, now)`.
   - JWT Generation:
     - Mint token with claims: `UserID: "usr_" + sid`, `SupplierID: sid`, `Role: RoleSupplier`, `OnboardingStatus: "PENDING"`.
   - Response (HTTP 201 Created):
     ```json
     {
       "supplier_id": "sup_12345678",
       "company_name": "ООО Toshkent Distribyutsiya",
       "tax_id": "301234567",
       "phone": "+998901234567",
       "onboarding_status": "PENDING",
       "token": "<jwt_token>",
       "refresh_token": "<refresh_token>",
       "next_step": "/onboarding/products"
     }
     ```

2. **`handleSupplierLogin`**:
   - Request Body:
     ```json
     {
       "phone": "+998901234567",
       "password": "SecurePassword123!"
     }
     ```
   - Credential Lookup in PostgreSQL:
     - Query: `SELECT supplier_id, name, legal_tax_id, phone, password_hash, onboarding_status FROM suppliers WHERE phone = $1 OR legal_tax_id = $1`
     - If `pgx.ErrNoRows`: return HTTP 401 Unauthorized (`{"error": "invalid_credentials", "message": "Invalid phone number or password"}`).
   - Bcrypt Hash Comparison:
     - `err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))`
     - If err != nil: return HTTP 401 Unauthorized (`{"error": "invalid_credentials", "message": "Invalid phone number or password"}`).
   - Dynamic Next Step Resolution:
     - If `onboarding_status == "PENDING"`: `next_step = "/onboarding/products"`
     - If `onboarding_status == "IN_PROGRESS"`: check active SKU count; if 0 `next_step = "/onboarding/products"`, else `next_step = "/onboarding/payment"`.
     - If `onboarding_status == "COMPLETED"`: `next_step = "/dashboard"`.
   - Response (HTTP 200 OK):
     ```json
     {
       "supplier_id": "sup_12345678",
       "company_name": "ООО Toshkent Distribyutsiya",
       "tax_id": "301234567",
       "phone": "+998901234567",
       "onboarding_status": "PENDING",
       "token": "<jwt_token>",
       "refresh_token": "<refresh_token>",
       "next_step": "/onboarding/products"
     }
     ```

### 5.4 Step 4: Non-Bypassable Onboarding Gate Middleware
In `internal/auth/middleware.go` or `internal/api/middleware.go`:

```go
func (s *Server) RequireSupplierOnboardingCompleted(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := auth.GetClaims(r.Context())
        if claims == nil || claims.Role != models.RoleSupplier {
            next.ServeHTTP(w, r)
            return
        }

        path := r.URL.Path
        // Whitelist authentication and onboarding wizard routes
        if strings.HasPrefix(path, "/v1/auth/") ||
           strings.HasPrefix(path, "/v1/supplier/onboarding/") ||
           strings.HasPrefix(path, "/v1/onboarding/") {
            next.ServeHTTP(w, r)
            return
        }

        // Query current supplier onboarding status from PostgreSQL
        var status string
        err := s.pool.QueryRow(r.Context(), "SELECT onboarding_status FROM suppliers WHERE supplier_id = $1", claims.SupplierID).Scan(&status)
        if err != nil || status != "COMPLETED" {
            response.JSON(w, http.StatusPreconditionRequired, map[string]interface{}{
                "error":             "onboarding_incomplete",
                "message":           "Supplier operational endpoints are locked until onboarding is completed",
                "onboarding_status": status,
                "next_step":         "/onboarding/products",
            })
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

Apply this middleware to all protected supplier operational route groups in `internal/api/router.go`.

### 5.5 Step 5: Phased Onboarding Wizard Endpoints
Wire the following routes in `internal/api/router.go`:
```go
r.Route("/v1/supplier/onboarding", func(ob chi.Router) {
    ob.Use(auth.RequireAuthWithKeyManager(cfg.JWTSecret, km))
    ob.Use(auth.RequireRole(models.RoleSupplier))

    ob.Post("/products", s.handleSupplierOnboardingProducts)
    ob.Get("/products", s.handleSupplierOnboardingListProducts)
    ob.Delete("/products/{skuID}", s.handleSupplierOnboardingDeleteProduct)

    ob.Get("/payment", s.handleSupplierOnboardingGetPayment)
    ob.Post("/payment", s.handleSupplierOnboardingConfigurePayment)

    ob.Post("/complete", s.handleSupplierOnboardingComplete)
})
```

1. **Step 1: Product Catalog (`POST /v1/supplier/onboarding/products`)**:
   - Validates:
     - Name non-empty.
     - Unique EAN-13 barcode (13 digits, valid EAN checksum).
     - 17-digit statutory MXIK tax code: `^\d{17}$`.
     - Package code non-empty.
     - Units per case: `>= 1`.
     - Unit price in 64-bit integer tiyins: `> 0`.
     - VAT percent: standard 12%.
   - Inserts directly into PostgreSQL table `skus`.
   - Transitions `suppliers.onboarding_status` to `"IN_PROGRESS"`.
2. **Step 2: Payment Gateway (`POST /v1/supplier/onboarding/payment`)**:
   - Validates:
     - Cash enabled by default (`cash_enabled: true`).
     - Global Pay (`GLOBAL_PAY`) gateway parameters (`service_id`, `secret_key`).
     - Allowed B2B corporate card BINs (e.g. `8600`, `9860`, `5614`).
   - Inserts or updates PostgreSQL table `supplier_payment_configs`.
3. **Step 3: Complete Onboarding (`POST /v1/supplier/onboarding/complete`)**:
   - Verification Gate:
     - Counts active SKUs for supplier: `SELECT COUNT(*) FROM skus WHERE supplier_id = $1 AND is_active = TRUE`. Requires `>= 1`.
     - Checks payment config: `SELECT cash_enabled, global_pay_enabled FROM supplier_payment_configs WHERE supplier_id = $1`. Requires at least one payment method enabled.
   - Atomic Transition:
     - Updates `suppliers SET onboarding_status = 'COMPLETED', updated_at = NOW() WHERE supplier_id = $1`.
     - Updates `supplier_profiles SET is_configured = TRUE, updated_at = NOW() WHERE supplier_id = $1`.
     - Emits outbox event `supplier.onboarding.completed` and broadcasts via WebSocket Hub.
     - Returns HTTP 200 OK with `onboarding_status: "COMPLETED", next_step: "/dashboard"`.

### 5.6 Step 6: Post-Onboarding Warehouse & Fleet Management Hub
Wire and enhance the following endpoints:
1. `POST/GET/PUT/DELETE /v1/supplier/warehouses`:
   - Mandatory `latitude` and `longitude` (`DOUBLE PRECISION`, valid ranges `[-90, 90]` and `[-180, 180]`).
   - On coordinate updates: invalidates Redis proximity cache keys (`geo:warehouses:*`) and emits WebSocket event `warehouse.relocated`.
   - On deletion:
     - Checks `stock_balances`: `SELECT COALESCE(SUM(on_hand_qty), 0) FROM stock_balances WHERE warehouse_id = $1`. If > 0, returns HTTP 409 Conflict (`"Cannot delete warehouse with active on-hand inventory"`).
     - Checks `orders`: `SELECT COUNT(*) FROM orders WHERE warehouse_id = $1 AND status NOT IN ('DELIVERED', 'CANCELLED')`. If > 0, returns HTTP 409 Conflict (`"Cannot delete warehouse with active orders"`).
2. Fleet & Dock Logistics:
   - `POST/GET /v1/supplier/warehouses/{id}/trucks`: Target `vehicles` table (`license_plate`, `payload_capacity_kg`, `max_volume_vu`, `fuel_type`, `operational_status`).
   - `POST/GET /v1/supplier/warehouses/{id}/payloaders`: Target `payloaders` table (`name`, `phone`, `warehouse_id`, `status`).

---

## 6. Verification and Risk Analysis

### Risks & Mitigations:
1. **Existing Unit Tests Breakage**:
   - `supplier_test.go` has 9 tests instantiating `NewMemoryRepository()`.
   - *Mitigation*: When removing `MemoryRepository` from production `repository.go`, create an isolated test suite in `supplier_test.go` using a mock `Repository` interface or an in-memory test implementation specifically inside `supplier_test.go` (or `internal/supplier/testing.go`), ensuring production code remains 100% pure PostgreSQL.
2. **Existing Mobile Wiring Test Dependencies**:
   - `supplier_mobile_live_wiring_test.go` queries `GET /v1/products?supplier_id=sup_pepsico_uz`.
   - *Mitigation*: Ensure migration 069 or seed scripts in test setup create a baseline supplier and sample SKU if running in test mode without persistent DB.
3. **Database Migration Cleanliness**:
   - Migration `069_supplier_onboarding_and_globalpay.sql` uses idempotent DDL (`ADD COLUMN IF NOT EXISTS`, `CREATE TABLE IF NOT EXISTS`, `DO $$ BEGIN ... END $$`) so it executes without error on both clean and pre-existing databases.

---

## 7. Next Actions for Implementation Agent
1. Create and apply `069_supplier_onboarding_and_globalpay.sql`.
2. Update `internal/models/domain.go` and `internal/models/claims.go` to include `OnboardingStatus` and new supplier fields.
3. Purge `MemoryRepository` and silent fallbacks from `internal/supplier/repository.go`.
4. Implement `handleSupplierRegister` and `handleSupplierLogin` with STIR deduplication and bcrypt in `internal/api/handlers_supplier.go`.
5. Implement `RequireSupplierOnboardingCompleted` middleware and `/v1/supplier/onboarding/*` routes in `internal/api/router.go`.
6. Implement warehouse coordinate validation, deletion conflict guards, and trucks/payloaders endpoints.
7. Run and verify `go test -v -race ./internal/supplier/...` and `./internal/api/...`.
