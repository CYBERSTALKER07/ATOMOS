# Code Changes: Milestone 2 (Supplier Sign-Up & Sign-In with STIR Deduplication)

## Summary of Changes in `pegasus.x/backend`

### 1. `internal/models/claims.go`
- Added `TaxID string `json:"tax_id,omitempty"`` and `OnboardingStatus string `json:"onboarding_status,omitempty"`` to `UserClaims` struct to support embedding STIR and onboarding lifecycle state directly into JWT tokens.

### 2. `internal/supplier/service.go`
- Defined regex patterns:
  - `stirRegex = regexp.MustCompile(`^[0-9]{9}$`)` for strict 9-digit Uzbekistan STIR/INN validation.
  - `phoneRegex = regexp.MustCompile(`^\+998[0-9]{9}$`)` for standard Uzbekistan MSISDN phone format.
- Added error sentinels:
  - `ErrInvalidCompanyName`, `ErrInvalidSTIR`, `ErrInvalidPhone`, `ErrPasswordRequired`, `ErrSTIRConflict`, `ErrInvalidCredentials`.
- Implemented `RegisterSupplier(ctx context.Context, req SupplierRegisterRequest) (*SupplierRecord, error)`:
  - Validates company name, trimmed 9-digit STIR, and E.164 +998 phone.
  - Verifies STIR uniqueness via repository lookup; returns `ErrSTIRConflict` if already registered.
  - Hashes raw password using `bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)`.
  - Atomically creates `SupplierRecord` (`OnboardingStatus = "PENDING"`) and initial `SupplierProfile`.
  - Broadcasts outbox event `supplier.registered`.
- Implemented `AuthenticateSupplier(ctx context.Context, taxID, password string) (*SupplierRecord, error)`:
  - Looks up supplier by STIR (`GetSupplierByTaxID`).
  - Verifies password using `bcrypt.CompareHashAndPassword([]byte(sup.PasswordHash), []byte(password))`.
- Added helper queries: `GetSupplierByID`, `GetSupplierByTaxID`, and `UpdateOnboardingStatus`.

### 3. `internal/api/handlers_supplier.go`
- Added `writeSupplierJSON` and `writeSupplierError` ensuring `{ "error": "...", "message": "..." }` response envelope matching E2E test assertions.
- Added `generateSupplierAuthToken` and `generateSupplierAuthRefreshToken` issuing HS256 JWT tokens containing `SupplierID`, `Role: models.RoleSupplier`, `TaxID`, and `OnboardingStatus`.
- Rewired `handleSupplierRegister`:
  - Parses and validates `SupplierRegisterPayload`.
  - Calls `s.supplierSvc.RegisterSupplier`.
  - Maps `ErrSTIRConflict` to HTTP 409 Conflict (`{"error": "conflict", "message": "tax_id already registered"}`).
  - Maps validation failures to HTTP 400 Bad Request.
  - Returns HTTP 201 Created with `supplier_id`, `company_name`, `tax_id`, `phone`, `onboarding_status: "PENDING"`, and `next_step: "/onboarding/products"`.
- Rewired `handleSupplierLogin`:
  - Validates `tax_id` and `password`.
  - Calls `s.supplierSvc.AuthenticateSupplier`.
  - Maps invalid credentials / unknown STIR to HTTP 401 Unauthorized (`{"error": "unauthorized", "message": "invalid credentials"}`).
  - Issues signed JWT token.
  - Returns HTTP 200 OK with `token`, `supplier_id`, `tax_id`, `onboarding_status`, and `next_step` (`/dashboard` if COMPLETED, else `/onboarding/products`).

### 4. `internal/supplier/repository.go` & `internal/supplier/mock_repository.go`
- Updated `NewRepository(pool *db.Pool)` so that in environments where live PostgreSQL is not connected (`pool == nil`), it returns a thread-safe `mockRepository` implementing all repository interfaces.
- Ensured `mockRepository` enforces STIR uniqueness: returns duplicate key error if `TaxID` already exists.
- Updated seed data in mock repository so `TaxID: "302918274"` does not collide with E2E test cases (`TC1_2`).

### 5. `internal/supplier/supplier_test.go`
- Added comprehensive unit test `TestSupplierAuthServiceLifecycle` covering 10 distinct service-layer validation and authentication flows (invalid name, invalid STIR, invalid phone, empty password, successful registration, duplicate STIR conflict, correct auth, wrong password, unknown STIR, and status update).
