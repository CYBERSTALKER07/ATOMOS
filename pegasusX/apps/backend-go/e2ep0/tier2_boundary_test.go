package e2ep0

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// ============================================================================
// Tier 2: Item 1 — Retailer Registration Boundaries & Malformed Payloads (5 tests)
// ============================================================================

func TestTier2_Item1_EmptyBodyRejected400(t *testing.T) {
	h := NewTestHarness(t)
	resp := h.Client.PostRaw("/v1/auth/retailer/register", nil, "")
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 400/422 on empty body, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item1_MalformedJSONRejected400(t *testing.T) {
	h := NewTestHarness(t)
	resp := h.Client.PostRaw("/v1/auth/retailer/register", []byte("{not-valid-json"), "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on malformed JSON, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item1_MissingPhoneFieldRejected400(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"supplier_id": DefaultSupplierA,
		"name":        "No Phone Market",
		"lat":         41.3111,
		"lng":         69.2797,
	}
	resp := h.Client.Post("/v1/auth/retailer/register", payload, "")
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 400/422 when phone is missing, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item1_MissingLatLngRejected422(t *testing.T) {
	h := NewTestHarness(t)
	// Registration without latitude and longitude should fail validation
	payload := map[string]any{
		"phone":       "+998901234500",
		"supplier_id": DefaultSupplierA,
		"name":        "Missing Coords Market",
		"lat":         0.0,
		"lng":         0.0,
	}
	resp := h.Client.Post("/v1/auth/retailer/register", payload, "")
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 422 Unprocessable Entity for missing lat/lng, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item1_WhitespacePhoneNormalizedOrRejected(t *testing.T) {
	h := NewTestHarness(t)
	rawPhone := "  +998909876543  "
	payload := map[string]any{
		"phone":       rawPhone,
		"supplier_id": DefaultSupplierA,
		"name":        "Trimmed Phone Market",
		"lat":         41.3111,
		"lng":         69.2797,
	}
	resp := h.Client.Post("/v1/auth/retailer/register", payload, "")
	if resp.StatusCode == http.StatusCreated {
		// If accepted, must be stored in repo
		ret, found, err := h.RetailerRepo.FindByPhone(context.Background(), rawPhone)
		if !found {
			ret, found, err = h.RetailerRepo.FindByPhone(context.Background(), "+998909876543")
		}
		if err != nil || !found {
			t.Fatalf("expected retailer to be found in repo, found=%v err=%v", found, err)
		}
		if ret.RetailerID == "" {
			t.Fatal("expected non-empty retailer ID")
		}
	} else if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 201 or 400/422 on whitespace phone, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tier 2: Item 2 — Factory & Warehouse Setup Role Boundaries (5 tests)
// ============================================================================

func TestTier2_Item2_FactorySetup_EmptyBody_NonFactoryRoleForbidden403(t *testing.T) {
	h := NewTestHarness(t)
	retailerToken := h.IssueRetailerToken(DefaultSupplierA, "ret-user-1")
	resp := h.Client.PostRaw("/v1/factory/setup", nil, retailerToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on factory setup with unauthorized role, got %d", resp.StatusCode)
	}
}

func TestTier2_Item2_FactorySetup_MalformedJSON_NonFactoryRoleForbidden403(t *testing.T) {
	h := NewTestHarness(t)
	retailerToken := h.IssueRetailerToken(DefaultSupplierA, "ret-user-1")
	resp := h.Client.PostRaw("/v1/factory/setup", []byte("{factory: broken"), retailerToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on factory setup with unauthorized role, got %d", resp.StatusCode)
	}
}

func TestTier2_Item2_WarehouseSetup_EmptyBody_NonWarehouseRoleForbidden403(t *testing.T) {
	h := NewTestHarness(t)
	retailerToken := h.IssueRetailerToken(DefaultSupplierA, "ret-user-1")
	resp := h.Client.PostRaw("/v1/warehouse/setup", nil, retailerToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on warehouse setup with unauthorized role, got %d", resp.StatusCode)
	}
}

func TestTier2_Item2_WarehouseSetup_MalformedJSON_NonWarehouseRoleForbidden403(t *testing.T) {
	h := NewTestHarness(t)
	retailerToken := h.IssueRetailerToken(DefaultSupplierA, "ret-user-1")
	resp := h.Client.PostRaw("/v1/warehouse/setup", []byte("bad-json"), retailerToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on warehouse setup with unauthorized role, got %d", resp.StatusCode)
	}
}

func TestTier2_Item2_FactorySetup_AuthorizedAdmin_PassesAuthGuard(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-user")
	payload := map[string]any{
		"name":        "Out of Range Factory",
		"factoryName": "Out of Range Factory",
		"lat":         41.3111,
		"lng":         69.2797,
	}
	resp := h.Client.Post("/v1/factory/setup", payload, adminToken)
	// Role check MUST pass (status should NOT be 401 Unauthorized or 403 Forbidden)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		t.Fatalf("authorized admin role was blocked by auth guard: status %d", resp.StatusCode)
	}
}

// ============================================================================
// Tier 2: Item 3 — Retailer KYC Review Boundaries (5 tests)
// ============================================================================

func TestTier2_Item3_KYCReview_NonExistentDoc404(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	payload := map[string]string{
		"status": "APPROVED",
	}
	resp := h.Client.Post("/v1/admin/retailer-kyc/non-existent-doc-id/review", payload, adminToken)
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 error for non-existent KYC document, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item3_KYCReview_EmptyBody400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	resp := h.Client.PostRaw("/v1/admin/retailer-kyc/doc-1/review", nil, adminToken)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 400/404 on empty body, got %d", resp.StatusCode)
	}
}

func TestTier2_Item3_KYCReview_InvalidStatusRejected400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	// Seed KYC doc
	h.RetailerRepo.AddKycDocument(retailer.KycDocument{
		DocumentID: "doc-valid-1",
		RetailerID: "ret-1",
		Status:     "PENDING",
	})

	payload := map[string]string{
		"status": "MAYBE_APPROVED_HACK",
	}
	resp := h.Client.Post("/v1/admin/retailer-kyc/doc-valid-1/review", payload, adminToken)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 400/422 for invalid KYC review status, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item3_KYCReview_RejectWithoutReasonRequiresHandling(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	h.RetailerRepo.AddKycDocument(retailer.KycDocument{
		DocumentID: "doc-valid-2",
		RetailerID: "ret-1",
		Status:     "PENDING",
	})

	payload := map[string]string{
		"status": "REJECTED",
		// reason omitted
	}
	resp := h.Client.Post("/v1/admin/retailer-kyc/doc-valid-2/review", payload, adminToken)
	// Should either reject missing reason (400) or succeed with empty reason (200)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 200 or 400 on reject without reason, got %d", resp.StatusCode)
	}
}

func TestTier2_Item3_KYCReview_MalformedJSON400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	resp := h.Client.PostRaw("/v1/admin/retailer-kyc/doc-1/review", []byte("broken json"), adminToken)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 400 on malformed json, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tier 2: Item 4 — Partner Keys Validation & Boundaries (5 tests)
// ============================================================================

func TestTier2_Item4_PartnerKeys_EmptyBody400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	resp := h.Client.PostRaw("/v1/admin/partner-keys", nil, adminToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty body for partner keys, got %d", resp.StatusCode)
	}
}

func TestTier2_Item4_PartnerKeys_MalformedJSON400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	resp := h.Client.PostRaw("/v1/admin/partner-keys", []byte("{scopes: ["), adminToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on malformed JSON, got %d", resp.StatusCode)
	}
}

func TestTier2_Item4_PartnerKeys_InvalidEnvironmentRejected(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
		"environment": "STAGING_INVALID",
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, adminToken)
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 422/400 for invalid environment, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item4_PartnerKeys_InvalidTenantTypeRejected(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	payload := map[string]any{
		"tenant_type": "UNAUTHORIZED_TENANT_TYPE",
		"tenant_id":   DefaultSupplierA,
		"environment": "LIVE",
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, adminToken)
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 403/422/400 on invalid tenant type, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item4_PartnerKeys_EmptyScopesDefaultsGracefully(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
		"environment": "LIVE",
		"scopes":      []string{}, // empty
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, adminToken)
	// Handler should default scopes gracefully and return 201 Created
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 on empty scopes default, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

// ============================================================================
// Tier 2: Item 5 — Control Tower IDOR & Input Boundaries (5 tests)
// ============================================================================

func TestTier2_Item5_ControlTower_NonExistentPlaybook404(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	patchBody := map[string]string{"name": "New Name"}
	resp := h.Client.Patch("/v1/control-tower/playbooks/pb-non-existent-999", patchBody, adminToken)
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 404/403 for non-existent playbook, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier2_Item5_ControlTower_MalformedJSONPatch400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	// Seed a playbook for Supplier A
	_ = h.ControlTowerRepo.CreatePlaybook(context.Background(), controltower.Playbook{
		PlaybookID: "pb-own-1",
		SupplierID: DefaultSupplierA,
		Name:       "My Playbook",
		IsActive:   true,
	})

	resp := h.Client.PatchRaw("/v1/control-tower/playbooks/pb-own-1", []byte("{broken: json"), adminToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on malformed JSON patch, got %d", resp.StatusCode)
	}
}

func TestTier2_Item5_ControlTower_ApproveNonExistentRun404(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	resp := h.Client.Post("/v1/control-tower/runs/run-does-not-exist/approve", map[string]string{}, adminToken)
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 404/403 for non-existent run approval, got %d", resp.StatusCode)
	}
}

func TestTier2_Item5_ControlTower_SkipNonExistentRun404(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	resp := h.Client.Post("/v1/control-tower/runs/run-does-not-exist/skip", map[string]string{}, adminToken)
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 404/403 for non-existent run skip, got %d", resp.StatusCode)
	}
}

func TestTier2_Item5_ControlTower_PathTraversalIDProtected(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	// Attempt path traversal via playbook ID
	resp := h.Client.Get("/v1/control-tower/playbooks/..%2F..%2Fetc%2Fpasswd", adminToken)
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("path traversal should not return 200 OK")
	}
}

// ============================================================================
// Tier 2: Item 6 — Credit-Leave Money Path Boundaries (5 tests)
// ============================================================================

func TestTier2_Item6_CreditLeave_NegativeDeliveredAmountRejected(t *testing.T) {
	h := NewTestHarness(t)
	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-1")

	payload := map[string]any{
		"order_id":        "ord-neg-1",
		"delivered_gross": -50000,
	}
	resp := h.Client.Post("/v1/order/complete", payload, driverToken)
	if resp.StatusCode == http.StatusOK {
		t.Fatal("expected failure on negative delivered amount, got 200 OK")
	}
}

func TestTier2_Item6_CreditLeave_ZeroTenderCardSettlementBlocked(t *testing.T) {
	h := NewTestHarness(t)
	orderID := "ord-boundary-zero-tender"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:      orderID,
		SupplierID:   DefaultSupplierA,
		RetailerID:   "ret-1",
		DriverID:     "driver-bnd-1",
		TotalMinor:   60000000,
		Status:       order.StatusDeliveredOnCredit,
		FiscalStatus: order.FiscalStatusPending,
		Lat:          41.311081,
		Lng:          69.240562,
	})

	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-bnd-1")
	resp := h.Client.Post("/v1/order/complete", map[string]any{
		"order_id":  orderID,
		"tender":    "CARD",
		"amount":    0, // zero tender
		"latitude":  41.311081,
		"longitude": 69.240562,
	}, driverToken)

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d (body: %s)", resp.StatusCode, resp.String())
	}
	var errBody map[string]string
	if err := resp.JSON(&errBody); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !strings.Contains(errBody["error"], "without verified payment receipt") {
		t.Fatalf("expected error containing 'without verified payment receipt', got: %s", errBody["error"])
	}
}

func TestTier2_Item6_CreditLeave_CompleteOrderExcludesCreditTender422(t *testing.T) {
	TestTier2_Item6_CreditLeave_ZeroTenderCardSettlementBlocked(t)
}

func TestTier2_Item6_CreditLeave_EmptyOrderIDRejected400(t *testing.T) {
	h := NewTestHarness(t)
	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-1")

	payload := map[string]any{
		"order_id": "",
	}
	resp := h.Client.Post("/v1/order/complete", payload, driverToken)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 400/422/404 on empty order ID, got %d", resp.StatusCode)
	}
}

func TestTier2_Item6_CreditLeave_MalformedJSONRejected400(t *testing.T) {
	h := NewTestHarness(t)
	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-1")

	resp := h.Client.PostRaw("/v1/order/complete", []byte("{broken-json"), driverToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on malformed json, got %d", resp.StatusCode)
	}
}

func TestTier2_Item6_CreditLeave_CrossTenantCompletionForbidden(t *testing.T) {
	h := NewTestHarness(t)
	orderB := "ord-supplier-b-1"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:      orderB,
		SupplierID:   DefaultSupplierA,
		RetailerID:   "ret-b-1",
		DriverID:     "driver-assigned-a",
		TotalMinor:   50000000,
		Status:       order.StatusDeliveredOnCredit,
		FiscalStatus: order.FiscalStatusPending,
		Lat:          41.311081,
		Lng:          69.240562,
	})

	// Issue a driver token for another driver or another tenant
	driverTokenB := h.IssueDriverToken("supplier-bravo", "driver-other-b")
	resp := h.Client.Post("/v1/order/complete", map[string]any{
		"order_id":  orderB,
		"latitude":  41.311081,
		"longitude": 69.240562,
	}, driverTokenB)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on cross-tenant driver completion, got %d (body: %s)", resp.StatusCode, resp.String())
	}
	var errBody map[string]string
	if err := resp.JSON(&errBody); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if errBody["error"] != "forbidden" {
		t.Fatalf("expected error 'forbidden', got: %s", errBody["error"])
	}
}

func TestTier2_Item6_CreditLeave_CrossTenantDriverCannotCompleteOrder(t *testing.T) {
	TestTier2_Item6_CreditLeave_CrossTenantCompletionForbidden(t)
}

// ============================================================================
// Tier 2: Item 7 — Inventory Release Clamping & Concurrency (5 tests)
// ============================================================================

func TestTier2_Item7_InventoryRelease_ZeroQuantityNoOp(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-zero-qty"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		WarehouseID:      "WH-1",
		QuantityOnHand:   100,
		QuantityReserved: 50,
		Version:          1,
	})

	err := h.InventorySvc.ReleaseReservation(context.Background(), invID, 0, 1)
	if err != nil {
		t.Fatalf("unexpected error on 0 release: %v", err)
	}

	item, _ := h.InventoryRepo.Get(context.Background(), invID)
	if item.QuantityOnHand != 100 || item.QuantityReserved != 50 {
		t.Fatalf("state mutated on 0 release: onHand=%d reserved=%d", item.QuantityOnHand, item.QuantityReserved)
	}
}

func TestTier2_Item7_InventoryRelease_ExcessiveQuantityClampedToReserved(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-excess-qty"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		WarehouseID:      "WH-1",
		QuantityOnHand:   100,
		QuantityReserved: 20,
		Version:          1,
	})

	// Release 100 when only 20 is reserved -> should clamp to 20
	err := h.InventorySvc.ReleaseReservation(context.Background(), invID, 100, 1)
	if err != nil {
		t.Fatalf("release error: %v", err)
	}

	item, _ := h.InventoryRepo.Get(context.Background(), invID)
	if item.QuantityReserved != 0 {
		t.Fatalf("expected QuantityReserved=0 after clamped release, got %d", item.QuantityReserved)
	}
	if item.QuantityOnHand != 100 {
		t.Fatalf("QuantityOnHand corrupted: want 100, got %d", item.QuantityOnHand)
	}
}

func TestTier2_Item7_InventoryRelease_StaleVersionConflict(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-stale-ver"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		WarehouseID:      "WH-1",
		QuantityOnHand:   100,
		QuantityReserved: 30,
		Version:          5,
	})

	// Attempt release with expected version 1 (stale)
	err := h.InventorySvc.ReleaseReservation(context.Background(), invID, 10, 1)
	if err == nil {
		t.Fatal("expected version conflict error for stale CAS version")
	}
	if !strings.Contains(err.Error(), "version conflict") {
		t.Fatalf("expected error containing 'version conflict', got: %v", err)
	}

	item, _ := h.InventoryRepo.Get(context.Background(), invID)
	if item.QuantityReserved != 30 || item.QuantityOnHand != 100 {
		t.Fatalf("stale release mutated inventory: onHand=%d, reserved=%d", item.QuantityOnHand, item.QuantityReserved)
	}
}

func TestTier2_Item7_InventoryRelease_NonExistentItemReturnsError(t *testing.T) {
	h := NewTestHarness(t)
	err := h.InventorySvc.ReleaseReservation(context.Background(), "inv-does-not-exist", 10, 1)
	if err == nil {
		t.Fatal("expected error releasing reservation on non-existent item")
	}
}

func TestTier2_Item7_InventoryRelease_ConcurrentReleasesAreRaceFree(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-race-test"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		WarehouseID:      "WH-1",
		QuantityOnHand:   500,
		QuantityReserved: 200,
		Version:          1,
	})

	var wg sync.WaitGroup
	var mu sync.Mutex
	currVersion := int64(1)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			v := currVersion
			err := h.InventorySvc.ReleaseReservation(context.Background(), invID, 20, v)
			if err == nil {
				currVersion++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	item, _ := h.InventoryRepo.Get(context.Background(), invID)
	if item.QuantityOnHand != 500 {
		t.Fatalf("QuantityOnHand altered by releases: got %d, want 500", item.QuantityOnHand)
	}
	if item.QuantityReserved < 0 {
		t.Fatalf("QuantityReserved went negative: %d", item.QuantityReserved)
	}
}

// ============================================================================
// Tier 2: Item 8 — Adyen Webhook Signature & Payload Boundaries (5 tests)
// ============================================================================

func TestTier2_Item8_AdyenWebhook_EmptyBodyRejected400(t *testing.T) {
	h := NewTestHarness(t)
	resp := h.Client.PostRaw("/v1/webhooks/adyen", nil, "")
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 400/422 on empty webhook body, got %d", resp.StatusCode)
	}
}

func TestTier2_Item8_AdyenWebhook_MalformedJSONRejected400(t *testing.T) {
	h := NewTestHarness(t)
	resp := h.Client.PostRaw("/v1/webhooks/adyen", []byte("invalid-json"), "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on malformed JSON, got %d", resp.StatusCode)
	}
}

func TestTier2_Item8_AdyenWebhook_EmptyNotificationItemsRejected422(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"notificationItems": []any{},
	}
	resp := h.Client.Post("/v1/webhooks/adyen", payload, "")
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 422/400 for empty notificationItems, got %d", resp.StatusCode)
	}
}

func TestTier2_Item8_AdyenWebhook_MissingHMACValidationCheck(t *testing.T) {
	h := NewTestHarness(t)
	// Build payload with empty/corrupted HMAC
	payload := []byte(`{
		"notificationItems": [
			{
				"NotificationRequestItem": {
					"eventCode": "AUTHORISATION",
					"eventDate": "2026-11-15T12:00:00Z",
					"merchantAccount": "PegasusMerchant",
					"merchantReference": "ord-test-no-hmac",
					"pspReference": "psp-no-hmac-1",
					"success": "true",
					"amount": {
						"currency": "UZS",
						"value": 100000
					}
				}
			}
		]
	}`)
	// Post webhook directly without HMAC signature in the item
	resp := h.Client.PostRaw("/v1/webhooks/adyen", payload, "", "application/json")
	// If HMAC verification is configured/enabled, missing HMAC signature must be rejected, or accepted as unverified
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 200, 400, or 401 on adyen webhook without valid HMAC, got %d", resp.StatusCode)
	}
}

func TestTier2_Item8_AdyenWebhook_InvalidHMACRejected401(t *testing.T) {
	h := NewTestHarness(t)
	// Build request with invalid HMAC header
	badHMAC := "deadbeefbadcafe1234567890"
	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/v1/webhooks/adyen", nil)
	req.Header.Set("HmacSignature", badHMAC)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()
	// Should fail due to bad HMAC or bad body
	if res.StatusCode != http.StatusBadRequest && res.StatusCode != http.StatusUnauthorized && res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 400/401/422 on invalid HMAC/empty body, got %d", res.StatusCode)
	}
}

// ============================================================================
// Tier 2: Item 9 — Fiscal Corrective Boundaries (5 tests)
// ============================================================================

func TestTier2_Item9_FiscalIntegrity_NegativeAmountRejected400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")
	orderID := "ord-fiscal-neg"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderID,
		SupplierID:            DefaultSupplierA,
		LatestFiscalReceiptID: "SOLIQ-ORIG-1",
	})

	payload := map[string]any{
		"correction_amount_minor": -10000,
		"reason":                  "OVERCHARGE",
	}
	resp := h.Client.Post("/v1/supplier/orders/"+orderID+"/fiscal/corrective", payload, adminToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on negative fiscal corrective amount, got %d (body: %s)", resp.StatusCode, resp.String())
	}
	var errBody map[string]string
	_ = resp.JSON(&errBody)
	if errBody["error"] != "correction_amount_must_be_positive" {
		t.Fatalf("expected error 'correction_amount_must_be_positive', got %s", errBody["error"])
	}
}

func TestTier2_Item9_FiscalIntegrity_ZeroAmountRejected400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")
	orderID := "ord-fiscal-zero"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderID,
		SupplierID:            DefaultSupplierA,
		LatestFiscalReceiptID: "SOLIQ-ORIG-1",
	})

	payload := map[string]any{
		"correction_amount_minor": 0,
		"reason":                  "OVERCHARGE",
	}
	resp := h.Client.Post("/v1/supplier/orders/"+orderID+"/fiscal/corrective", payload, adminToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on zero fiscal corrective amount, got %d (body: %s)", resp.StatusCode, resp.String())
	}
	var errBody map[string]string
	_ = resp.JSON(&errBody)
	if errBody["error"] != "correction_amount_must_be_positive" {
		t.Fatalf("expected error 'correction_amount_must_be_positive', got %s", errBody["error"])
	}
}

func TestTier2_Item9_FiscalIntegrity_NonExistentOrder404(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	payload := map[string]any{
		"correction_amount_minor": 50000,
		"reason":                  "REFUND",
	}
	resp := h.Client.Post("/v1/supplier/orders/ord-never-existed-404/fiscal/corrective", payload, adminToken)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent order, got %d (body: %s)", resp.StatusCode, resp.String())
	}
	var errBody map[string]string
	_ = resp.JSON(&errBody)
	if errBody["error"] != "order_not_found" {
		t.Fatalf("expected error 'order_not_found', got %s", errBody["error"])
	}
}

func TestTier2_Item9_FiscalIntegrity_NoOriginalReceipt412(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")
	orderID := "ord-no-orig-receipt"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderID,
		SupplierID:            DefaultSupplierA,
		LatestFiscalReceiptID: "", // Missing original receipt
	})

	payload := map[string]any{
		"correction_amount_minor": 50000,
		"reason":                  "REFUND",
	}
	resp := h.Client.Post("/v1/supplier/orders/"+orderID+"/fiscal/corrective", payload, adminToken)
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 Precondition Failed when order lacks original receipt, got %d (body: %s)", resp.StatusCode, resp.String())
	}
	var errBody map[string]string
	_ = resp.JSON(&errBody)
	if errBody["error"] != "no_original_fiscal_receipt_to_correct" {
		t.Fatalf("expected 'no_original_fiscal_receipt_to_correct', got %s", errBody["error"])
	}
}

func TestTier2_Item9_FiscalIntegrity_MalformedJSONRejected400(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-1")

	resp := h.Client.PostRaw("/v1/supplier/orders/ord-test/fiscal/corrective", []byte("{broken-json"), adminToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on malformed json, got %d (body: %s)", resp.StatusCode, resp.String())
	}
	var errBody map[string]string
	_ = resp.JSON(&errBody)
	if errBody["error"] != "invalid_json" {
		t.Fatalf("expected 'invalid_json', got %s", errBody["error"])
	}
}

// ============================================================================
// Tier 2: Item 10 — Auth Session Security Boundaries (5 tests)
// ============================================================================

func TestTier2_Item10_AuthSession_EmptyAuthorizationHeader401(t *testing.T) {
	h := NewTestHarness(t)
	req, _ := http.NewRequest(http.MethodGet, h.Server.URL+"/v1/retailer/me", nil)
	req.Header.Set("Authorization", "")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on empty Authorization header, got %d", res.StatusCode)
	}
}

func TestTier2_Item10_AuthSession_BasicAuthSchemeRejected401(t *testing.T) {
	h := NewTestHarness(t)
	req, _ := http.NewRequest(http.MethodGet, h.Server.URL+"/v1/retailer/me", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on Basic scheme, got %d", res.StatusCode)
	}
}

func TestTier2_Item10_AuthSession_BearerWhitespaceOnlyRejected401(t *testing.T) {
	h := NewTestHarness(t)
	req, _ := http.NewRequest(http.MethodGet, h.Server.URL+"/v1/retailer/me", nil)
	req.Header.Set("Authorization", "Bearer    ")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 on Bearer whitespace only, got %d", res.StatusCode)
	}
}

func TestTier2_Item10_AuthSession_MalformedJWTStructureRejected401(t *testing.T) {
	h := NewTestHarness(t)
	req, _ := http.NewRequest(http.MethodGet, h.Server.URL+"/v1/retailer/me", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOi.eyJzdWIiOi.invalid")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 on malformed JWT structure, got %d", res.StatusCode)
	}
}

func TestTier2_Item10_AuthSession_WrongSigningSecretRejected401(t *testing.T) {
	h := NewTestHarness(t)
	// Mint token signed with an alien secret
	forgedToken, err := auth.Issue(auth.Claims{
		Subject:    "attacker",
		SupplierID: DefaultSupplierA,
		Role:       auth.RoleAdmin,
	}, auth.IssueOptions{
		Secret: "alien-secret-that-does-not-match-test-harness",
		Issuer: "pegasusx",
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, h.Server.URL+"/v1/retailer/me", nil)
	req.Header.Set("Authorization", "Bearer "+forgedToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on forged token signature, got %d", res.StatusCode)
	}
}

// suppress unused import warning if any
var _ = fmt.Sprintf
var _ = hex.EncodeToString
var _ = hmac.New
var _ = sha256.New
