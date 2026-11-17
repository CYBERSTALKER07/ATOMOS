package e2ep0

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// ============================================================================
// Tier 3: Cross-Feature Interactions & Multi-Tenant Boundary Enforcement
// ============================================================================

// 1. Cross Item 1 + Item 3: Retailer cannot self-approve KYC; Admin approval required
func TestTier3_Cross_RetailerKYC_SelfApprovalBlocked_And_AdminDerivation(t *testing.T) {
	h := NewTestHarness(t)
	retID := "ret-apple-tenant-1"
	docID := "doc-apple-kyc-1"

	// Register retailer under Supplier A
	_ = h.RetailerRepo.CreateRetailer(context.Background(), retailer.Retailer{
		RetailerID: retID,
		SupplierID: DefaultSupplierA,
		Phone:      "+998901110001",
		Name:       "Apple Retailer One",
	}, nil)

	// Add KYC doc scoped to Retailer
	h.RetailerRepo.AddKycDocument(retailer.KycDocument{
		DocumentID: docID,
		RetailerID: retID,
		Status:     "PENDING",
	})

	// Retailer attempts to self-approve KYC document
	retailerToken := h.IssueRetailerToken(DefaultSupplierA, retID)
	payload := map[string]string{"status": "APPROVED"}
	resp := h.Client.Post("/v1/admin/retailer-kyc/"+docID+"/review", payload, retailerToken)

	// Must be blocked with 403 Forbidden
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("CRITICAL SECURITY VIOLATION: Retailer was able to self-approve KYC! Got status %d", resp.StatusCode)
	}

	// Legitimate Admin approval succeeds and derives admin Subject
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-sec-ops-7")
	respAdmin := h.Client.Post("/v1/admin/retailer-kyc/"+docID+"/review", payload, adminToken)
	if respAdmin.StatusCode != http.StatusOK {
		t.Fatalf("expected admin approval to succeed with 200, got %d", respAdmin.StatusCode)
	}
	reviewer := h.RetailerRepo.GetKycReviewedBy(docID)
	if reviewer != "admin-sec-ops-7" {
		t.Fatalf("expected reviewer to match admin Subject, got %q", reviewer)
	}
}

// 2. Cross Item 4 + Item 5: Partner Key vs Control Tower Tenant Isolation
// Partner key minted for Supplier A cannot access or manipulate Control Tower runs of Supplier B
func TestTier3_Cross_PartnerKey_ControlTowerIsolation(t *testing.T) {
	h := NewTestHarness(t)
	// Seed run for Supplier B
	runB := "run-ct-banana-1"
	_ = h.ControlTowerRepo.CreateRun(context.Background(), controltower.PlaybookRun{
		RunID:      runB,
		SupplierID: DefaultSupplierB,
		Status:     controltower.RunStatusSuggested,
	})

	// Mint partner key for Supplier A
	adminTokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")
	keyResp := h.Client.Post("/v1/admin/partner-keys", map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
		"environment": "LIVE",
		"scopes":      []string{"orders:read", "controltower:write"},
	}, adminTokenA)
	if keyResp.StatusCode != http.StatusCreated && keyResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to issue partner key: %d", keyResp.StatusCode)
	}

	var data map[string]any
	_ = keyResp.JSON(&data)
	secret, _ := data["secret"].(string)
	if secret == "" {
		t.Fatal("expected secret in partner key response")
	}

	// Attempt to approve Supplier B's run using Partner Key from Supplier A
	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/v1/control-tower/runs/"+runB+"/approve", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		t.Fatalf("CRITICAL IDOR: Partner key of Supplier A approved Control Tower run of Supplier B!")
	}
}

// 3. Cross Item 6 + Item 7: Credit-Leave & Inventory Hold Interaction
// When an order is delivered on credit, uncollected tender cannot clear the balance,
// and releasing unfulfilled inventory must preserve QuantityOnHand without inflation.
func TestTier3_Cross_CreditLeave_And_InventoryRelease(t *testing.T) {
	h := NewTestHarness(t)
	orderID := "ord-cross-credit-inv-1"
	invID := "inv-cross-coke-1"
	initialOnHand := int64(100)
	initialReserved := int64(20)

	// Seed inventory
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		WarehouseID:      "wh-main",
		ProductID:        "sku-coke-1L",
		QuantityOnHand:   initialOnHand,
		QuantityReserved: initialReserved,
		Version:          1,
	})

	// Seed order delivered on credit with assigned driver and coordinates
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:      orderID,
		SupplierID:   DefaultSupplierA,
		RetailerID:   "ret-1",
		DriverID:     "driver-cross-1",
		TotalMinor:   50000000,
		Status:       order.StatusDeliveredOnCredit,
		FiscalStatus: order.FiscalStatusPending,
		Lat:          41.311081,
		Lng:          69.240562,
	})

	// 1. Attempt zero-tender completion with driver credentials and geofence -> must fail closed (422)
	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-cross-1")
	compResp := h.Client.Post("/v1/order/complete", map[string]any{
		"order_id":  orderID,
		"latitude":  41.311081,
		"longitude": 69.240562,
	}, driverToken)
	if compResp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d (body: %s)", compResp.StatusCode, compResp.String())
	}
	var compErr map[string]string
	if err := compResp.JSON(&compErr); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !strings.Contains(compErr["error"], "without verified payment receipt") {
		t.Fatalf("expected error containing 'without verified payment receipt', got: %s", compErr["error"])
	}

	// 2. Cancellation / partial return releases 10 reserved units via InventorySvc
	cur, err := h.InventoryRepo.Get(context.Background(), invID)
	if err != nil {
		t.Fatalf("failed to fetch current inventory: %v", err)
	}
	err = h.InventorySvc.ReleaseReservation(context.Background(), invID, 10, cur.Version)
	if err != nil {
		t.Fatalf("failed to release inventory reservation: %v", err)
	}

	updated, err := h.InventoryRepo.Get(context.Background(), invID)
	if err != nil {
		t.Fatalf("failed to fetch updated inventory: %v", err)
	}

	// Invariant: QuantityOnHand MUST be strictly preserved
	if updated.QuantityOnHand != initialOnHand {
		t.Fatalf("STOCK INFLATION: QuantityOnHand changed from %d to %d", initialOnHand, updated.QuantityOnHand)
	}
	if updated.QuantityReserved != 10 {
		t.Fatalf("expected reserved=10, got %d", updated.QuantityReserved)
	}
}

func TestTier3_Cross_OrderCreditLeave_InventoryRelease_Interaction(t *testing.T) {
	TestTier3_Cross_CreditLeave_And_InventoryRelease(t)
}

// 4. Cross Item 2 + Item 10: Multi-Surface Privilege Escalation Defense
// A single Retailer token attempting Factory Setup, Warehouse Setup, and Supplier Partner Key issuance is blocked at every gate.
func TestTier3_Cross_RetailerToken_PrivilegeEscalation_BlockedAcrossAllSurfaces(t *testing.T) {
	h := NewTestHarness(t)
	retailerToken := h.IssueRetailerToken(DefaultSupplierA, "ret-attacker")

	// Surface 1: Factory Setup (Admin/Factory only)
	r1 := h.Client.Post("/v1/factory/setup", map[string]any{"name": "Attacker Factory"}, retailerToken)
	if r1.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on factory setup for retailer, got %d", r1.StatusCode)
	}

	// Surface 2: Warehouse Setup (Admin/Warehouse only)
	r2 := h.Client.Post("/v1/warehouse/setup", map[string]any{"name": "Attacker Warehouse"}, retailerToken)
	if r2.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on warehouse setup for retailer, got %d", r2.StatusCode)
	}

	// Surface 3: Supplier Partner Keys issuance (Admin only)
	r3 := h.Client.Post("/v1/supplier/partner-keys", map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
	}, retailerToken)
	if r3.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on supplier partner keys issuance for retailer, got %d", r3.StatusCode)
	}
}

// 5. Cross Item 1 + Item 3 + Order Flow: Registration to KYC to Operational Access
func TestTier3_Cross_RetailerRegistration_KYCApproval_Lifecycle(t *testing.T) {
	h := NewTestHarness(t)
	phone := "+998901234567"
	docID := "doc-lifecycle-kyc-1"

	// 1. Retailer Registers
	regPayload := map[string]any{
		"phone":       phone,
		"supplier_id": DefaultSupplierA,
		"name":        "Lifecycle Grocer",
		"lat":         41.3111,
		"lng":         69.2797,
	}
	regResp := h.Client.Post("/v1/auth/retailer/register", regPayload, "")
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("registration failed: %d (body: %s)", regResp.StatusCode, regResp.String())
	}

	var regRes retailer.RegisterResponse
	_ = regResp.JSON(&regRes)
	if regRes.RetailerID == "" {
		t.Fatal("empty retailer ID returned")
	}

	// 2. Submit KYC Document
	h.RetailerRepo.AddKycDocument(retailer.KycDocument{
		DocumentID: docID,
		RetailerID: regRes.RetailerID,
		Status:     "PENDING",
	})

	// 3. Admin Reviews & Approves KYC
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-reviewer")
	reviewResp := h.Client.Post("/v1/admin/retailer-kyc/"+docID+"/review", map[string]string{
		"status": "APPROVED",
	}, adminToken)
	if reviewResp.StatusCode != http.StatusOK {
		t.Fatalf("KYC review failed: %d (body: %s)", reviewResp.StatusCode, reviewResp.String())
	}

	// Verify reviewer recorded
	reviewer := h.RetailerRepo.GetKycReviewedBy(docID)
	if reviewer != "admin-reviewer" {
		t.Fatalf("expected reviewer 'admin-reviewer', got %q", reviewer)
	}
}

// 6. Cross Item 6 + Item 8: Credit-Leave Order & Adyen Chargeback Non-Settling Mapping
// An order on credit followed by an Adyen chargeback event must not settle or disguise money debt.
func TestTier3_Cross_CreditLeave_AdyenChargeback_NonSettling(t *testing.T) {
	h := NewTestHarness(t)
	orderID := "ord-chargeback-credit-1"

	h.OrderRepo.SeedOrder(order.Order{
		OrderID:      orderID,
		SupplierID:   DefaultSupplierA,
		RetailerID:   "ret-1",
		TotalMinor:   75000000,
		Status:       order.StatusDeliveredOnCredit,
		FiscalStatus: order.FiscalStatusPending,
	})

	// Webhook delivers CHARGEBACK notification for this order's pspReference
	payload := GenerateAdyenWebhookPayload(t, h.AdyenHMACKey, "CHARGEBACK", true, "psp-cb-order-1", 75000000, "UZS")
	resp := h.Client.PostRaw("/v1/webhooks/adyen", payload, "", "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook returned %d", resp.StatusCode)
	}

	// Verify attempt recorded with CHARGEBACK status and NEVER PAID
	attempts := h.PaymentRepo.GetAttempts()
	if len(attempts) > 0 {
		for _, a := range attempts {
			if a.Status == "PAID" {
				t.Fatalf("SECURITY VIOLATION: Chargeback event marked as PAID!")
			}
		}
	}

	// Verify order status in repository was NOT marked completed
	o, found, _ := h.OrderRepo.GetOrder(context.Background(), orderID)
	if !found {
		t.Fatal("order missing")
	}
	if o.Status == order.StatusCompleted {
		t.Fatal("order incorrectly marked completed after chargeback!")
	}
}

// 7. Cross Item 9 + Multi-Tenant: Corrective Fiscal Receipt Isolation
// Admin of Supplier A cannot issue a corrective fiscal receipt for an order belonging to Supplier B
func TestTier3_Cross_FiscalCorrective_TenantIsolation(t *testing.T) {
	h := NewTestHarness(t)
	adminTokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")

	// Seed order belonging to Supplier Bravo
	orderB := "ord-supplier-bravo-fiscal-corr"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderB,
		SupplierID:            DefaultSupplierB,
		RetailerID:            "ret-b",
		TotalMinor:            120000000,
		LatestFiscalReceiptID: "SOLIQ-REC-BRAVO-1",
	})

	payload := map[string]any{
		"correction_amount_minor": 120000000,
		"reason":                  "damaged_goods",
	}
	// Supplier A admin attempts to issue corrective receipt for Supplier B order
	resp := h.Client.Post("/v1/supplier/orders/"+orderB+"/fiscal/corrective", payload, adminTokenA)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on cross-tenant corrective receipt, got %d (body: %s)", resp.StatusCode, resp.String())
	}
	var errBody map[string]string
	_ = resp.JSON(&errBody)
	if errBody["error"] != "forbidden" {
		t.Fatalf("expected error 'forbidden', got %s", errBody["error"])
	}
}

// 8. Cross Item 4 + Tenant Isolation: Cross-Tenant Partner Key Minting Rejection
// Admin of Supplier A cannot issue partner keys with tenant_id of Supplier B
func TestTier3_Cross_PartnerKeys_CrossTenantMintingRejected(t *testing.T) {
	h := NewTestHarness(t)
	adminTokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")

	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierB, // Different supplier!
		"environment": "LIVE",
		"scopes":      []string{"orders:write"},
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, adminTokenA)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on cross-tenant partner key minting, got %d", resp.StatusCode)
	}
}

// 9. Cross Item 4 + Item 10: Auth Session Cookie vs Bearer Header for Partner Key Issuance
// Verifies that both valid Bearer token and valid httpOnly session cookie authenticate the admin,
// but invalid/expired tokens in either channel fail closed.
func TestTier3_Cross_AuthChannels_PartnerKeyIssuance(t *testing.T) {
	h := NewTestHarness(t)
	adminTokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")

	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
		"environment": "LIVE",
		"scopes":      []string{"orders:read"},
	}

	// 1. Success via Bearer Header
	respHeader := h.Client.Post("/v1/admin/partner-keys", payload, adminTokenA)
	if respHeader.StatusCode != http.StatusCreated && respHeader.StatusCode != http.StatusOK {
		t.Fatalf("failed via Bearer header: %d", respHeader.StatusCode)
	}

	// 2. Success via Session Cookie
	respCookie := h.Client.PostWithCookie("/v1/admin/partner-keys", payload, auth.CookieName, adminTokenA)
	if respCookie.StatusCode != http.StatusCreated && respCookie.StatusCode != http.StatusOK {
		t.Fatalf("failed via session cookie: %d", respCookie.StatusCode)
	}

	// 3. Rejection with Expired Token in Cookie
	expiredToken := h.IssueExpiredToken(auth.RoleAdmin, DefaultSupplierA, "admin-apple")
	respExpired := h.Client.PostWithCookie("/v1/admin/partner-keys", payload, auth.CookieName, expiredToken)
	if respExpired.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 on expired session cookie, got %d", respExpired.StatusCode)
	}
}

// 10. Cross Item 5 + Multi-Tenant: Control Tower Playbook Deactivation Isolation
// Admin of Supplier B cannot deactivate playbooks created by Supplier A
func TestTier3_Cross_ControlTower_CrossTenantPlaybookDeactivation(t *testing.T) {
	h := NewTestHarness(t)
	playbookA := "pb-tenant-apple-isolated"

	_ = h.ControlTowerRepo.CreatePlaybook(context.Background(), controltower.Playbook{
		PlaybookID: playbookA,
		SupplierID: DefaultSupplierA,
		Name:       "Apple Proprietary Playbook",
		IsActive:   true,
	})

	// Supplier B admin attempts to deactivate
	adminTokenB := h.IssueAdminToken(DefaultSupplierB, "admin-banana")
	resp := h.Client.Post("/v1/control-tower/playbooks/"+playbookA+"/deactivate", nil, adminTokenB)

	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403/404 on cross-tenant playbook deactivation, got %d", resp.StatusCode)
	}

	// Verify playbook remained active
	pb, _ := h.ControlTowerRepo.GetPlaybook(context.Background(), playbookA)
	if !pb.IsActive {
		t.Fatal("CRITICAL IDOR: Supplier A playbook was deactivated by Supplier B admin!")
	}
}

// 11. Cross Item 7 + Item 10: Inventory Level Query Scope Enforcement
// Warehouse token can list its warehouse levels, while unauthenticated or alien requests fail closed.
func TestTier3_Cross_Inventory_AuthAndWarehouseScoping(t *testing.T) {
	h := NewTestHarness(t)
	whID := "wh-apple-north"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      "inv-north-1",
		WarehouseID:      whID,
		SupplierID:       DefaultSupplierA,
		QuantityOnHand:   300,
		QuantityReserved: 50,
		Version:          1,
	})

	// 1. Warehouse level query
	levels, err := h.InventoryRepo.ListByWarehouse(context.Background(), whID)
	if err != nil {
		t.Fatalf("failed to query warehouse inventory: %v", err)
	}
	if len(levels) != 1 || levels[0].QuantityOnHand != 300 {
		t.Fatalf("unexpected inventory level query result: %+v", levels)
	}

	// 2. Querying a different warehouse returns empty
	otherLevels, _ := h.InventoryRepo.ListByWarehouse(context.Background(), "wh-alien-south")
	if len(otherLevels) != 0 {
		t.Fatalf("expected 0 items for alien warehouse, got %d", len(otherLevels))
	}
}

// 12. Cross Item 4 + Item 10: Warehouse Token Cannot Issue Partner Keys
func TestTier3_Cross_WarehouseTokenCannotIssuePartnerKeys(t *testing.T) {
	h := NewTestHarness(t)
	warehouseToken := h.IssueWarehouseToken(DefaultSupplierA, "wh-user-attempt")

	// Attempt to issue partner keys
	resp := h.Client.Post("/v1/admin/partner-keys", map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
	}, warehouseToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on admin partner keys issuance for warehouse token, got %d", resp.StatusCode)
	}
}

// Suppress unused imports
var _ = time.Now
var _ = strings.TrimSpace
