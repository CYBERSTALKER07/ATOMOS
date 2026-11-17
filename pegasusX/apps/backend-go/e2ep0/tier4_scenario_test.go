package e2ep0

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// ============================================================================
// Tier 4: Real-World Operational Lifecycle Scenarios (5 Workflows)
// ============================================================================

// Scenario 1: Complete Order Lifecycle with Anti-Debt-Forgiveness & Fiscal Integrity
func TestTier4_Scenario1_CompleteOrderLifecycle_WithAntiFraudGuards(t *testing.T) {
	h := NewTestHarness(t)
	supplierID := DefaultSupplierA
	retPhone := "+998901239999"
	invID := "inv-scen1-sku-1"
	orderID := "ord-scen1-full-flow"
	docID := "doc-scen1-kyc-1"

	// Step 1: Inventory is stocked in warehouse with reserved quantity
	initialOnHand := int64(200)
	initialReserved := int64(50)
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		WarehouseID:      "wh-central",
		ProductID:        "sku-tea-box",
		SupplierID:       supplierID,
		QuantityOnHand:   initialOnHand,
		QuantityReserved: initialReserved,
		Version:          1,
	})

	// Step 2: Retailer registers on mobile / desktop
	regResp := h.Client.Post("/v1/auth/retailer/register", map[string]any{
		"phone":       retPhone,
		"supplier_id": supplierID,
		"name":        "Chaikhana Oasis",
		"lat":         41.3111,
		"lng":         69.2797,
	}, "")
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("step 2 failed: retailer registration got %d (body: %s)", regResp.StatusCode, regResp.String())
	}
	var regData retailer.RegisterResponse
	_ = regResp.JSON(&regData)
	retailerID := regData.RetailerID

	// Step 3: Retailer submits KYC and Admin approves
	h.RetailerRepo.AddKycDocument(retailer.KycDocument{
		DocumentID: docID,
		RetailerID: retailerID,
		Status:     "PENDING",
	})
	adminToken := h.IssueAdminToken(supplierID, "admin-kyc-officer")
	kycReview := h.Client.Post("/v1/admin/retailer-kyc/"+docID+"/review", map[string]string{
		"status": "APPROVED",
	}, adminToken)
	if kycReview.StatusCode != http.StatusOK {
		t.Fatalf("step 3 failed: kyc review got %d", kycReview.StatusCode)
	}

	// Step 4: Order is placed
	orderGrossMinor := int64(50000000) // 500,000 UZS
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderID,
		SupplierID:            supplierID,
		RetailerID:            retailerID,
		DriverID:              "driver-fast",
		TotalMinor:            orderGrossMinor,
		Status:                order.StatusDeliveredOnCredit,
		FiscalStatus:          order.FiscalStatusPending,
		LatestFiscalReceiptID: "SOLIQ-SCEN1-ORIG",
		Lat:                   41.3111,
		Lng:                   69.2797,
	})

	// Step 5: FRAUD ATTEMPT — Driver attempts zero-tender card settlement
	driverToken := h.IssueDriverToken(supplierID, "driver-fast")
	fraudResp := h.Client.Post("/v1/order/complete", map[string]any{
		"order_id":  orderID,
		"latitude":  41.3111,
		"longitude": 69.2797,
		"tender":    "CARD",
		"amount":    0, // zero tender exploit attempt
	}, driverToken)
	if fraudResp.StatusCode == http.StatusOK {
		t.Fatalf("CRITICAL SECURITY HOLE: zero-tender card settlement succeeded!")
	}
	if fraudResp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d (body: %s)", fraudResp.StatusCode, fraudResp.String())
	}

	// Step 6: Verify balance was NOT cleared and order is NOT completed
	ordAfterFraud, found, _ := h.OrderRepo.GetOrder(context.Background(), orderID)
	if !found || ordAfterFraud.Status == order.StatusCompleted {
		t.Fatalf("order status illegally changed to completed after fraud attempt")
	}

	// Step 7: Release inventory reservation (50 units at version 1)
	err := h.InventorySvc.ReleaseReservation(context.Background(), invID, 50, 1)
	if err != nil {
		t.Fatalf("step 7 failed: reservation release: %v", err)
	}

	// Step 8: Verify Invariants — zero stock inflation
	finalInv, _ := h.InventoryRepo.Get(context.Background(), invID)
	if finalInv.QuantityOnHand != initialOnHand {
		t.Fatalf("STOCK INFLATION: QuantityOnHand mutated from %d to %d", initialOnHand, finalInv.QuantityOnHand)
	}
	if finalInv.QuantityReserved != 0 {
		t.Fatalf("expected reserved stock to be 0, got %d", finalInv.QuantityReserved)
	}
}

// Scenario 2: Multi-Tenant Supplier Operational Isolation Lifecycle
func TestTier4_Scenario2_MultiTenantSupplierIsolation(t *testing.T) {
	h := NewTestHarness(t)
	supplierA := DefaultSupplierA
	supplierB := DefaultSupplierB

	// Step 1: Supplier A creates Control Tower Playbook
	playbookA := "pb-tenant-a-ops"
	_ = h.ControlTowerRepo.CreatePlaybook(context.Background(), controltower.Playbook{
		PlaybookID: playbookA,
		SupplierID: supplierA,
		Name:       "Supplier A Secret Playbook",
		IsActive:   true,
	})

	// Step 2: Supplier B admin attempts IDOR to modify Supplier A's playbook
	adminTokenB := h.IssueAdminToken(supplierB, "admin-b")
	patchResp := h.Client.Patch("/v1/control-tower/playbooks/"+playbookA, map[string]string{
		"name": "Hacked Playbook Name",
	}, adminTokenB)
	if patchResp.StatusCode != http.StatusForbidden && patchResp.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-tenant playbook patch succeeded: status %d", patchResp.StatusCode)
	}

	// Step 3: Supplier B admin attempts to issue partner key for Supplier A
	forgeResp := h.Client.Post("/v1/admin/partner-keys", map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   supplierA,
		"environment": "LIVE",
	}, adminTokenB)
	if forgeResp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-tenant partner key issue succeeded: status %d", forgeResp.StatusCode)
	}

	// Step 4: Legitimate Supplier A admin operates their own playbook
	adminTokenA := h.IssueAdminToken(supplierA, "admin-a")
	validPatch := h.Client.Patch("/v1/control-tower/playbooks/"+playbookA, map[string]string{
		"name": "Supplier A Updated Playbook",
	}, adminTokenA)
	if validPatch.StatusCode != http.StatusOK {
		t.Fatalf("legitimate admin playbook patch failed: status %d", validPatch.StatusCode)
	}

	// Verify persistence
	pbA, _ := h.ControlTowerRepo.GetPlaybook(context.Background(), playbookA)
	if pbA.Name != "Supplier A Updated Playbook" {
		t.Fatalf("unexpected playbook name: %s", pbA.Name)
	}
}

// Scenario 3: Driver Shift Completion & Fiscal Reconciliation Audit
func TestTier4_Scenario3_DriverShift_FiscalReconciliationAudit(t *testing.T) {
	h := NewTestHarness(t)
	supplierID := DefaultSupplierA

	orderSuccess := "ord-shift-paid-1"
	orderFailed := "ord-shift-failed-1"
	orderPending := "ord-shift-pending-1"

	// Seed fiscal attempts in repository
	h.OrderRepo.SeedFiscalReceipt(order.FiscalReceiptRow{
		AttemptID:       "att-paid-1",
		OrderID:         orderSuccess,
		SupplierID:      supplierID,
		Status:          order.FiscalStatusSuccess,
		FiscalReceiptID: "SOLIQ-REC-PAID-1",
		AmountMinor:     25000000,
		CreatedAt:       time.Now().UTC(),
	})
	h.OrderRepo.SeedFiscalReceipt(order.FiscalReceiptRow{
		AttemptID:       "att-fail-1",
		OrderID:         orderFailed,
		SupplierID:      supplierID,
		Status:          order.FiscalStatusFailed,
		FiscalReceiptID: "",
		AmountMinor:     30000000,
		CreatedAt:       time.Now().UTC(),
	})
	h.OrderRepo.SeedFiscalReceipt(order.FiscalReceiptRow{
		AttemptID:       "att-pend-1",
		OrderID:         orderPending,
		SupplierID:      supplierID,
		Status:          order.FiscalStatusPending,
		FiscalReceiptID: "",
		AmountMinor:     15000000,
		CreatedAt:       time.Now().UTC(),
	})

	// Verify repository retrieval
	rowSuccess, found, err := h.OrderRepo.GetFiscalReceipt(context.Background(), orderSuccess)
	if !found || err != nil {
		t.Fatalf("failed to get success fiscal receipt: %v", err)
	}
	if rowSuccess.Status != order.FiscalStatusSuccess || rowSuccess.FiscalReceiptID != "SOLIQ-REC-PAID-1" {
		t.Fatalf("unexpected success row: %+v", rowSuccess)
	}

	// Exercise production EvaluateOpenFiscalOrders:
	// Success: captured + PoD -> not blocking
	// Failed: unrecorded PoD -> blocking (Frozen=true, Count=1)
	// Pending: retryable OFD -> retry queued (RetryQueuedCount=1)
	candidates := []order.OpenFiscalCandidate{
		{
			OrderID:            orderSuccess,
			Status:             string(order.StatusCompleted),
			FiscalStatus:       order.FiscalStatusSuccess,
			HasPaymentRecorded: true,
			HasCryptoOrQRPoD:   true,
		},
		{
			OrderID:            orderFailed,
			Status:             string(order.StatusFiscalFailed),
			FiscalStatus:       order.FiscalStatusFailed,
			HasPaymentRecorded: true,
			HasCryptoOrQRPoD:   false, // Missing PoD freezes shift end
		},
		{
			OrderID:            orderPending,
			Status:             string(order.StatusFiscalizing),
			FiscalStatus:       order.FiscalStatusPending,
			HasPaymentRecorded: true,
			HasCryptoOrQRPoD:   true, // Retryable OFD outage
		},
	}

	snap := order.EvaluateOpenFiscalOrders(false, candidates)
	if !snap.Frozen || snap.Count != 1 {
		t.Fatalf("expected 1 blocking order with frozen cash bag, got Count=%d Frozen=%v", snap.Count, snap.Frozen)
	}
	if len(snap.OrderIDs) != 1 || snap.OrderIDs[0] != orderFailed {
		t.Fatalf("expected orderFailed to block shift, got %v", snap.OrderIDs)
	}
	if snap.RetryQueuedCount != 1 || len(snap.RetryOrderIDs) != 1 || snap.RetryOrderIDs[0] != orderPending {
		t.Fatalf("expected orderPending to be retry-queued, got %v", snap.RetryOrderIDs)
	}
	if snap.FiscalIssueCount != 2 {
		t.Fatalf("expected 2 fiscal issues, got %d", snap.FiscalIssueCount)
	}

	// When driver obtains PoD for failed order, evaluate non-blocking transition
	candidates[1].HasCryptoOrQRPoD = true
	snapClean := order.EvaluateOpenFiscalOrders(false, candidates)
	if snapClean.Frozen || snapClean.Count != 0 {
		t.Fatalf("expected shift-end unblocked once PoD recorded, got Frozen=%v Count=%d", snapClean.Frozen, snapClean.Count)
	}
	if snapClean.RetryQueuedCount != 2 {
		t.Fatalf("expected 2 retry queued orders, got %d", snapClean.RetryQueuedCount)
	}

	// Attempting corrective receipt with failing provider fails closed
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderSuccess,
		SupplierID:            supplierID,
		LatestFiscalReceiptID: "SOLIQ-REC-PAID-1",
	})
	h.FiscalProvider.ShouldFail = true
	adminToken := h.IssueAdminToken(supplierID, "admin-audit")
	corrResp := h.Client.Post("/v1/supplier/orders/"+orderSuccess+"/fiscal/corrective", map[string]any{
		"correction_amount_minor": 10000000,
		"reason":                  "damaged_goods",
	}, adminToken)
	if corrResp.StatusCode == http.StatusOK {
		t.Fatal("corrective receipt succeeded when fiscal provider failed!")
	}

	// When provider is operational, corrective receipt succeeds and returns 200 OK
	h.FiscalProvider.ShouldFail = false
	corrOk := h.Client.Post("/v1/supplier/orders/"+orderSuccess+"/fiscal/corrective", map[string]any{
		"correction_amount_minor": 10000000,
		"reason":                  "damaged_goods",
	}, adminToken)
	if corrOk.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on valid corrective receipt, got %d (body: %s)", corrOk.StatusCode, corrOk.String())
	}
}

// Scenario 4: Role Onboarding, Boundary Enforcement & Privilege Escalation Defense
func TestTier4_Scenario4_RoleOnboarding_PrivilegeEscalationDefense(t *testing.T) {
	h := NewTestHarness(t)
	supplierID := DefaultSupplierA

	// Step 1: User initially has Retailer role
	retailerToken := h.IssueRetailerToken(supplierID, "retailer-promoted-user")

	// Retailer attempts to access Factory setup
	fResp := h.Client.Post("/v1/factory/setup", map[string]any{"name": "Factory Hack"}, retailerToken)
	if fResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on factory setup, got %d", fResp.StatusCode)
	}

	// Retailer attempts to access Warehouse setup
	wResp := h.Client.Post("/v1/warehouse/setup", map[string]any{"name": "Warehouse Hack"}, retailerToken)
	if wResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on warehouse setup, got %d", wResp.StatusCode)
	}

	// Retailer attempts to issue supplier partner keys
	kResp := h.Client.Post("/v1/supplier/partner-keys", map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   supplierID,
	}, retailerToken)
	if kResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on supplier partner keys, got %d", kResp.StatusCode)
	}

	// Step 2: User is assigned Warehouse role
	warehouseToken := h.IssueWarehouseToken(supplierID, "warehouse-promoted-user")

	// Warehouse user can manage inventory
	whID := "wh-scenario-4"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      "inv-scen4-1",
		WarehouseID:      whID,
		SupplierID:       supplierID,
		QuantityOnHand:   100,
		QuantityReserved: 10,
		Version:          1,
	})
	levels, err := h.InventoryRepo.ListByWarehouse(context.Background(), whID)
	if err != nil || len(levels) == 0 {
		t.Fatalf("warehouse user unable to list levels: %v", err)
	}

	// But Warehouse user STILL cannot issue supplier partner keys
	whKeyResp := h.Client.Post("/v1/admin/partner-keys", map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   supplierID,
	}, warehouseToken)
	if whKeyResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for warehouse role on admin partner keys, got %d", whKeyResp.StatusCode)
	}
}

// Scenario 5: End-to-End Partner API Lifecycle
func TestTier4_Scenario5_PartnerAPILifecycle(t *testing.T) {
	h := NewTestHarness(t)
	supplierID := DefaultSupplierA
	adminToken := h.IssueAdminToken(supplierID, "admin-partner-mgr")

	// Step 1: Issue SANDBOX key
	sbResp := h.Client.Post("/v1/admin/partner-keys", map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   supplierID,
		"environment": "SANDBOX",
		"scopes":      []string{"orders:read", "orders:write"},
	}, adminToken)
	if sbResp.StatusCode != http.StatusCreated && sbResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to issue sandbox key: %d", sbResp.StatusCode)
	}
	var sbData map[string]any
	_ = sbResp.JSON(&sbData)
	sbSecret, _ := sbData["secret"].(string)
	if sbSecret == "" || sbSecret[:4] != "pxs_" {
		t.Fatalf("expected SANDBOX key prefix 'pxs_', got %q", sbSecret)
	}

	// Step 2: Issue LIVE key
	liveResp := h.Client.Post("/v1/admin/partner-keys", map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   supplierID,
		"environment": "LIVE",
		"scopes":      []string{"orders:read", "orders:write"},
	}, adminToken)
	if liveResp.StatusCode != http.StatusCreated && liveResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to issue live key: %d", liveResp.StatusCode)
	}
	var liveData map[string]any
	_ = liveResp.JSON(&liveData)
	liveSecret, _ := liveData["secret"].(string)
	liveKeyID, _ := liveData["key_id"].(string)
	if liveSecret == "" || liveSecret[:4] != "pxk_" {
		t.Fatalf("expected LIVE key prefix 'pxk_', got %q", liveSecret)
	}

	// Step 3: Revoke key
	if liveKeyID != "" {
		revResp := h.Client.Post("/v1/admin/partner-keys/"+liveKeyID+"/revoke", nil, adminToken)
		if revResp.StatusCode != http.StatusOK && revResp.StatusCode != http.StatusNoContent {
			t.Logf("key revoke returned %d", revResp.StatusCode)
		}
	}
}

// Suppress unused imports
var _ = auth.RoleAdmin
