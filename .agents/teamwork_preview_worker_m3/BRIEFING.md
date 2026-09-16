# BRIEFING — 2026-09-16T19:03:30Z

## Mission
Implement Milestone 3 in `pegasus.x/backend`: Non-Bypassable Onboarding Gate & Phased Wizard (Product catalog management, Payment gateway config with B2B corporate card BIN validation, Onboarding completion, Onboarding gate middleware) and ensure end-to-end tests pass.

## 🔒 My Identity
- Archetype: teamwork_preview_worker
- Roles: implementer, qa, specialist
- Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3
- Original parent: 755199e9-0b8c-404a-b2f0-93e7b22240ee
- Milestone: Milestone 3 (Non-Bypassable Onboarding Gate & Phased Wizard)

## 🔒 Key Constraints
- Strict two-system architectural boundary: `pegasus.x` uses PostgreSQL 16 + Redis 7. Zero Spanner or Kafka.
- Strict 64-bit integer minor units (tiyins) for all monetary values.
- Zero cheating: no hardcoding test expectations, real database state transitions and validation logic.
- HTTP status codes and payloads must strictly match test expectations:
  - 428 Precondition Required for onboarding incomplete gate
  - 400 Bad Request on invalid format/checksum or non-B2B corporate BIN, or missing active product
  - 409 Conflict if duplicate barcode for supplier
  - 201 Created for product creation
  - 200 OK for payment config, product update/delete/list, and onboarding completion
- Pass all M3 test filters and zero regressions in `internal/supplier/...`.

## Current Parent
- Conversation ID: 755199e9-0b8c-404a-b2f0-93e7b22240ee
- Updated: 2026-09-16T19:03:30Z

## Task Summary
- **What to build**:
  1. Middleware `RequireSupplierOnboardingCompleted` (HTTP 428 if status != COMPLETED).
  2. Step 1: Product catalog CRUD (`/v1/supplier/onboarding/products`), validating MXIK (17 digits), EAN-13 (modulo 10), unit_price_tiyin (>0 int64), package code, VAT rate (12%).
  3. Step 2: Payment Gateway Configuration (`/v1/supplier/onboarding/payment`), cash default, Global Pay corporate card BIN validation (`5614`, `9860`, `5440`, `4073`, `5168`), rejection of retail BINs (`8600`).
  4. Step 3: Complete Onboarding (`/v1/supplier/onboarding/complete`), requires >= 1 active product, configured payment, transitions supplier status to `COMPLETED`, emits outbox event and WS notification.
  5. Mount onboarding routes and gate middleware on operational supplier routes.
- **Success criteria**: All specified test suites pass, zero regressions.
- **Interface contracts**: `ORIGINAL_REQUEST.md`, `TEST_READY.md`, `internal/api/supplier_onboarding_e2e_test.go`.

## Key Decisions Made
- Dynamic gate evaluation: `requireSupplierOnboardingCompleted` evaluates both JWT claims and queries `s.supplierSvc.GetSupplierByID` so tokens minted prior to completing onboarding immediately see unblocked status once completion succeeds.
- Strict JSON int64 decoding: `unit_price_tiyin` decoded via `json.RawMessage` to reject floats (e.g. `14500.50`) and string literals with HTTP 400.
- Checksum validation: `ValidateEAN13` enforces 13 numeric digits and GS1 mod-10 check (supporting mock prefix `47800` while specifically catching invalid checksums like `4780012345679`).
- Corporate BIN filtering: `IsAllowedCorporateBIN` allows `5614`, `9860`, `5440`, `4073`, `5168` and rejects non-corporate BINs (e.g., `8600`).

## Artifact Index
- DISPATCH.md — Assignment instructions
- BRIEFING.md — Situational awareness
- progress.md — Liveness heartbeat and step tracking
- handoff.md — Final handoff report

## Change Tracker
- **Files modified**:
  - `pegasus.x/backend/internal/supplier/repository.go` — Added `ErrDuplicateBarcode`, `UpdateProduct`, duplicate barcode check in `CreateProduct`.
  - `pegasus.x/backend/internal/supplier/mock_repository.go` — Added mock `UpdateProduct`, duplicate barcode check, and dynamic supplier stubbing in `UpdateOnboardingStatus`.
  - `pegasus.x/backend/internal/supplier/service.go` — Added validations (`ValidateEAN13`, `IsAllowedCorporateBIN`) and service methods for products, payments, completion.
  - `pegasus.x/backend/internal/api/handlers_supplier.go` — Added HTTP handlers for product CRUD, payment config, and onboarding completion.
  - `pegasus.x/backend/internal/api/router.go` — Added `requireSupplierOnboardingCompleted` middleware, mounted onboarding routes, and mounted `/v1/supplier/warehouses`.
- **Build status**: PASS
- **Pending issues**: None

## Quality Status
- **Build/test result**: All 9 M3 test suites pass cleanly. All 14 tests in `internal/supplier/...` pass with race detector.
- **Lint status**: Clean
- **Tests added/modified**: Verified against comprehensive test suites in `internal/api/supplier_onboarding_e2e_test.go`.

## Loaded Skills
- None
