# Handoff Report — Milestone 4 (Roles 5 & 6 Driver Doorstep & Retailer B2B Wholesale)

**Reviewer Subagent**: `reviewer_m4_1` (`reviewer_critic`)  
**Target Monorepo**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m4_1`  
**Verdict**: **REQUEST_CHANGES**  

---

## 1. Observation

### Observation 1.1: Milestone 4 Package Test Execution (PASS)
Executed command:
```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go test -count=1 -v -race ./internal/doorstep/... ./internal/epod/... ./internal/retailer/...
```
Verbatim test output:
```
=== RUN   TestDoorstep_UrbanCanyonDrift_Success
--- PASS: TestDoorstep_UrbanCanyonDrift_Success (0.00s)
=== RUN   TestDoorstep_UrbanCanyonDrift_MissingPhotoOrSign_FailClosed
--- PASS: TestDoorstep_UrbanCanyonDrift_MissingPhotoOrSign_FailClosed (0.00s)
=== RUN   TestDoorstep_ShopClosed_Countdown_And_Resolution
--- PASS: TestDoorstep_ShopClosed_Countdown_And_Resolution (0.00s)
=== RUN   TestDoorstep_ConcealedDamage_48HourWindow
--- PASS: TestDoorstep_ConcealedDamage_48HourWindow (0.00s)
=== RUN   TestDoorstep_PhysicalVsDigitalConflict
--- PASS: TestDoorstep_PhysicalVsDigitalConflict (0.00s)
=== RUN   TestDoorstep_DynamicTokenGenerationAndVerification
--- PASS: TestDoorstep_DynamicTokenGenerationAndVerification (0.00s)
=== RUN   TestDoorstep_GeofenceBreachAndCameraFallback
2026/09/23 11:24:53 INFO emergency doorstep photo bypass accepted order_id=ord_geofence_breach_02 distance=475.1354294069169
--- PASS: TestDoorstep_GeofenceBreachAndCameraFallback (0.00s)
=== RUN   TestDoorstep_ItemizedOffloadAndCameraLockout
--- PASS: TestDoorstep_ItemizedOffloadAndCameraLockout (0.00s)
=== RUN   TestDoorstep_DualTenderSettlementAndSoliqReceipt
--- PASS: TestDoorstep_DualTenderSettlementAndSoliqReceipt (0.00s)
PASS
ok  	github.com/pegasus-x/core/internal/doorstep	1.364s
=== RUN   TestEPODExecutionWorkflow
2026/09/23 11:24:53 INFO manifest dispatched for delivery run manifest_id=man-01
2026/09/23 11:24:53 INFO driver arrived at delivery stop manifest_id=man-01 order_id=ord-7003 retailer="Havas Uchtepa Discounter"
2026/09/23 11:24:53 INFO electronic proof of delivery confirmed manifest_id=man-01 order_id=ord-7003 recipient="Sherzod Aliyev" cash_minor=210000000 geofence_verified=true
--- PASS: TestEPODExecutionWorkflow (0.00s)
=== RUN   TestOfflineStoreAndForwardSync
--- PASS: TestOfflineStoreAndForwardSync (0.00s)
PASS
ok  	github.com/pegasus-x/core/internal/epod	1.364s
=== RUN   TestRegisterAndShiftLifecycle
--- PASS: TestRegisterAndShiftLifecycle (0.01s)
=== RUN   TestPOSSalesAndInventoryDeduction
--- PASS: TestPOSSalesAndInventoryDeduction (0.01s)
=== RUN   TestParkedCartHolds
--- PASS: TestParkedCartHolds (0.01s)
=== RUN   TestPhysicalCycleCountReconciliation
--- PASS: TestPhysicalCycleCountReconciliation (0.01s)
=== RUN   TestAutoOrderReplenishmentEvaluationAndConfirmation
--- PASS: TestAutoOrderReplenishmentEvaluationAndConfirmation (0.01s)
=== RUN   TestSellThroughAnalytics
--- PASS: TestSellThroughAnalytics (0.01s)
=== RUN   TestRetailer_QuarantineStatus
--- PASS: TestRetailer_QuarantineStatus (0.01s)
=== RUN   TestRetailer_InboundTruckTracking
--- PASS: TestRetailer_InboundTruckTracking (0.01s)
=== RUN   TestRetailer_HandshakeTokenDisplay
--- PASS: TestRetailer_HandshakeTokenDisplay (0.01s)
=== RUN   TestRetailer_DoorstepReviewAndDamagedCartonReconciliation
--- PASS: TestRetailer_DoorstepReviewAndDamagedCartonReconciliation (0.01s)
=== RUN   TestRetailer_SelectPaymentTender
--- PASS: TestRetailer_SelectPaymentTender (0.01s)
=== RUN   TestRetailer_FiscalReceiptDisplay
--- PASS: TestRetailer_FiscalReceiptDisplay (0.01s)
PASS
ok  	github.com/pegasus-x/core/internal/retailer	1.453s
```
Total: 23 test cases executed and passing cleanly with `-race`.

### Observation 1.2: Monorepo Compilation & API Build Failure (FAIL)
Executed command:
```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go test -count=1 ./internal/api/...
```
Verbatim failure:
```
# github.com/pegasus-x/core/internal/fleet
internal/fleet/repository.go:17:2: "github.com/pegasus-x/core/internal/doorstep" imported and not used
FAIL	github.com/pegasus-x/core/internal/api [build failed]
FAIL
```
Executed command:
```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go test ./...
```
Verbatim failure:
```
# github.com/pegasus-x/core/internal/fleet
internal/fleet/repository.go:17:2: "github.com/pegasus-x/core/internal/doorstep" imported and not used
FAIL	github.com/pegasus-x/core/cmd/server [build failed]
FAIL	github.com/pegasus-x/core/cmd/smokecheck [build failed]
FAIL	github.com/pegasus-x/core/internal/api [build failed]
FAIL	github.com/pegasus-x/core/internal/fleet [build failed]
```
Inspection of `backend/internal/fleet/repository.go` at line 17:
```go
14: 	"github.com/google/uuid"
15: 	"github.com/jackc/pgx/v5"
16: 	"github.com/pegasus-x/core/internal/db"
17: 	"github.com/pegasus-x/core/internal/doorstep"
18: 	"github.com/pegasus-x/core/internal/outbox"
```
The package `doorstep` is imported on line 17 but never referenced anywhere in `backend/internal/fleet/repository.go`. In Go, unused imports are fatal compilation errors.

### Observation 1.3: Unmounted Endpoint for Retailer Quarantine Status
In `backend/internal/api/handlers_retailer.go:57-63`:
```go
// handleGetRetailerQuarantineStatus serves GET /v1/retailer/quarantine-status
func (s *Server) handleGetRetailerQuarantineStatus(w http.ResponseWriter, r *http.Request) {
	setQuarantineHeaders(w)
	status := s.retailerSvc.GetQuarantinedFeatureStatus()
	response.JSON(w, http.StatusOK, status)
}
```
Inspection of `backend/internal/api/router.go` (lines 1315–1345):
```go
// B2B Wholesale Inbound Tracking, Doorstep Handshake & Tender Selection
ret.Get("/orders/{orderID}/tracking", s.handleGetRetailerInboundTracking)
ret.Get("/orders/{orderID}/handshake-token", s.handleGetRetailerHandshakeToken)
ret.Get("/orders/{orderID}/doorstep-review", s.handleGetRetailerDoorstepReview)
ret.Post("/orders/{orderID}/payment-tender", s.handleSelectRetailerPaymentTender)
ret.Get("/orders/{orderID}/fiscal-receipt", s.handleGetRetailerFiscalReceipt)
```
Grep for `quarantine-status` across `router.go` returned 0 matches.
`handleGetRetailerQuarantineStatus` is never registered on the router; `GET /v1/retailer/quarantine-status` returns HTTP 404.

### Observation 1.4: Adversarial Audit of Urban Canyon Bypass (`VerifyHandshake`)
In `backend/internal/doorstep/service.go:167-180`:
```go
		} else {
			// Proximity exceeded: check for emergency camera photo fallback
			if req.PhotoEvidenceURL != "" && (req.CaptureSource == "" || req.CaptureSource == CaptureSourceCameraDirect) {
				fallbackApproved = true
				geofenceVerified = true
				slog.Info("emergency doorstep photo bypass accepted", "order_id", req.OrderID, "distance", distanceMeters)
			} else if req.ShopSignText != "" && req.PhotoEvidenceURL != "" {
				fallbackApproved = true
				geofenceVerified = true
			} else {
				return nil, fmt.Errorf("%w: driver is %.1fm away (threshold %.0fm) and missing live photo evidence",
					ErrProximityExceeded, distanceMeters, MaxDoorstepGeofenceMeters)
			}
		}
```
In the `else if req.ShopSignText != "" && req.PhotoEvidenceURL != ""` branch, `req.CaptureSource` is not validated. If a caller supplies `req.CaptureSource = "GALLERY"` alongside `req.ShopSignText`, the condition evaluates to `true`, approving the geofence bypass with a gallery upload.

### Observation 1.5: Adversarial Audit of Damaged Offload Camera Lockout
In `backend/internal/doorstep/service.go:259-267`:
```go
			capSource := item.CaptureSource
			if capSource == "" {
				capSource = req.CaptureSource
			}
			// Block gallery uploads strictly
			if capSource == CaptureSourceGallery || capSource == CaptureSourceGalleryUpload || capSource == CaptureSourceDeviceStorage {
				return nil, ErrCameraLockout
			}
```
The check is a blacklist rather than an affirmative whitelist. If `capSource` is omitted or set to an alternative string, it bypasses the explicit `CaptureSourceCameraDirect` requirement.

### Observation 1.6: Boundary Audit (Spanner & Kafka) (PASS)
Executed grep search for `(spanner|kafka)` across `internal/doorstep`, `internal/epod`, `internal/retailer`, `internal/fleet`:
Zero references found in production code. PostgreSQL 16 (`pgx/v5`) and Redis 7 Streams are strictly utilized.

---

## 2. Logic Chain

1. **Premise 1 (Compiler & Build Health)**: The root canonical doctrine (AGENTS.md) and Truth Protocol state that repository source code and passing automated tests are the sole Source of Truth. If any compilation or regression error exists, code cannot be certified for release.
2. **Premise 2 (Unused Import Regression)**: From Observation 1.2, line 17 of `backend/internal/fleet/repository.go` introduces an unused import `"github.com/pegasus-x/core/internal/doorstep"`.
3. **Premise 3 (Blast Radius of Build Failure)**: Because `fleet` is imported by `api`, `server`, and `smokecheck`, compiling `api` or running monorepo tests `go test ./...` fails with `[build failed]`.
4. **Premise 4 (Worker Handoff Discrepancy)**: Worker M4's handoff report claimed that `internal/api` had clean compilation and passing tests. This claim was refuted by live observation.
5. **Premise 5 (Quarantine Route Omission)**: From Observation 1.3, `handleGetRetailerQuarantineStatus` is implemented but omitted from `router.go`, leaving an incomplete public interface contract for Retailer B2B Wholesale status verification.
6. **Premise 6 (Security Lockout Hole)**: From Observation 1.4, `VerifyHandshake` allows gallery uploads when `ShopSignText` is non-empty, creating a security hole in the doorstep camera lockout for geofence overrides.
7. **Deductive Conclusion**: While domain models, 64-bit integer tiyin calculations, dynamic OTP HMAC verification, and local package tests passed, the monorepo compilation failure, unmounted route, and camera bypass loophole require remediation before approval. Therefore, the verdict is `REQUEST_CHANGES`.

---

## 3. Caveats

- **Active PostgreSQL Instance**: Verification tests ran using the standard unit test harness without a live PostgreSQL 16 socket. When `r.pool == nil`, repositories fall back to internal maps. SQL syntax and migration table definitions were inspected directly in `074_ecosystem_hardening_and_parity.sql`.
- **Consumer POS Isolation**: In-store POS functions still exist in `internal/retailer` and emit deprecation headers (`Deprecation: true`, `X-Quarantined-Scope`) rather than returning hard HTTP 410 Gone errors, to preserve legacy unit test compatibility.

---

## 4. Conclusion & Required Changes

**Verdict**: **`REQUEST_CHANGES`**

### Required Action Items for Worker M4:
1. **[CRITICAL] Fix Unused Import in `internal/fleet/repository.go`**:
   - File: `pegasus.x/backend/internal/fleet/repository.go:17`
   - Remove `"github.com/pegasus-x/core/internal/doorstep"`.
   - Verify `go test -count=1 ./internal/fleet/...`, `go test -count=1 ./internal/api/...`, and `go test ./...` compile and pass cleanly.

2. **[MAJOR] Mount `/v1/retailer/quarantine-status` in `router.go`**:
   - File: `pegasus.x/backend/internal/api/router.go`
   - Add `ret.Get("/quarantine-status", s.handleGetRetailerQuarantineStatus)` inside the retailer subrouter block.

3. **[MAJOR] Plug Camera Lockout Bypass in `VerifyHandshake`**:
   - File: `pegasus.x/backend/internal/doorstep/service.go:169-178`
   - Ensure that whenever `distanceMeters > MaxDoorstepGeofenceMeters`, gallery uploads (`GALLERY`, `GALLERY_UPLOAD`, `DEVICE_STORAGE`) are strictly rejected with `ErrCameraLockout`, and `CaptureSource == CaptureSourceCameraDirect` is affirmatively required for both photo bypass paths.

4. **[DEFENSIVE] Whitelist `CaptureSourceCameraDirect` in `ProcessPartialOffload`**:
   - File: `pegasus.x/backend/internal/doorstep/service.go:264-267`
   - Enforce `if capSource != CaptureSourceCameraDirect { return nil, ErrCameraLockout }` when `item.RejectedQty > 0`.

---

## 5. Verification Method

To independently reproduce observations and verify fixes:

1. **Verify Unused Import Failure**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -count=1 ./internal/fleet/...
   go test -count=1 ./internal/api/...
   ```
   *Expected Current Output*: `# github.com/pegasus-x/core/internal/fleet ... imported and not used: "github.com/pegasus-x/core/internal/doorstep"`.

2. **Verify Doorstep, EPOD, and Retailer Package Tests**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -count=1 -v -race ./internal/doorstep/... ./internal/epod/... ./internal/retailer/...
   ```
   *Expected Current Output*: PASS across all 23 tests.

3. **Verify Route Absence in Router**:
   ```bash
   grep -rn "quarantine-status" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/router.go
   ```
   *Expected Current Output*: Empty (0 matches).
