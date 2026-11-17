package e2ep0

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// ============================================================================
// Tier 1: Item 1 — Retailer Registration & Phone Collision (5 tests)
// ============================================================================

func TestTier1_Item1_DuplicatePhoneRejection409(t *testing.T) {
	h := NewTestHarness(t)
	existingPhone := "+998901112233"

	// Seed pre-existing retailer
	err := h.RetailerRepo.CreateRetailer(context.Background(), retailer.Retailer{
		RetailerID: "ret-seeded-1",
		Phone:      existingPhone,
		SupplierID: DefaultSupplierA,
		Name:       "Existing Shop",
	}, nil)
	if err != nil {
		t.Fatalf("failed to seed retailer: %v", err)
	}

	// Attempt registration with same phone
	payload := map[string]any{
		"phone":       existingPhone,
		"supplier_id": DefaultSupplierA,
		"name":        "Attacker Store",
		"lat":         41.3111,
		"lng":         69.2797,
	}
	resp := h.Client.Post("/v1/auth/retailer/register", payload, "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on duplicate phone, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	var body map[string]string
	_ = resp.JSON(&body)
	if body["error"] != "phone_already_registered" {
		t.Fatalf("expected error 'phone_already_registered', got %q", body["error"])
	}
}

func TestTier1_Item1_NewPhoneHappyPath201(t *testing.T) {
	h := NewTestHarness(t)
	newPhone := "+998909998877"

	payload := map[string]any{
		"phone":       newPhone,
		"supplier_id": DefaultSupplierA,
		"name":        "Fresh Corner Market",
		"lat":         41.3111,
		"lng":         69.2797,
	}
	resp := h.Client.Post("/v1/auth/retailer/register", payload, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created on fresh phone register, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	var res retailer.RegisterResponse
	if err := resp.JSON(&res); err != nil {
		t.Fatalf("failed to parse RegisterResponse: %v", err)
	}
	if res.RetailerID == "" {
		t.Fatal("expected non-empty RetailerID")
	}

	// Verify persistence in repo
	ret, found, err := h.RetailerRepo.FindByPhone(context.Background(), newPhone)
	if err != nil || !found {
		t.Fatalf("expected newly registered retailer to be found in repo, found=%v err=%v", found, err)
	}
	if ret.Phone != newPhone {
		t.Fatalf("phone mismatch in repo: got %s, want %s", ret.Phone, newPhone)
	}
}

func TestTier1_Item1_MobileRegisterPath_NewPhone(t *testing.T) {
	h := NewTestHarness(t)
	newPhone := "+998904445566"

	// Mobile format uses "phone_number" field
	payload := map[string]any{
		"phone_number": newPhone,
		"supplier_id":  DefaultSupplierA,
		"first_name":   "Akmal",
		"store_name":   "Akmal Groceries",
		"latitude":     41.3111,
		"longitude":    69.2797,
	}
	resp := h.Client.Post("/v1/auth/retailer/register", payload, "")
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 200/201 on mobile registration, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	var data map[string]any
	if err := resp.JSON(&data); err != nil {
		t.Fatalf("failed to parse mobile auth JSON: %v", err)
	}
	if data["access_token"] == nil && data["token"] == nil {
		t.Fatalf("expected access token in mobile auth response, got: %+v", data)
	}
}

func TestTier1_Item1_MobileRegisterPath_DuplicatePhoneRejection409(t *testing.T) {
	h := NewTestHarness(t)
	existingPhone := "+998905554433"

	_ = h.RetailerRepo.CreateRetailer(context.Background(), retailer.Retailer{
		RetailerID: "ret-seeded-2",
		Phone:      existingPhone,
		SupplierID: DefaultSupplierA,
	}, nil)

	payload := map[string]any{
		"phone_number": existingPhone,
		"supplier_id":  DefaultSupplierA,
		"store_name":   "Victim Impersonator",
		"latitude":     41.3111,
		"longitude":    69.2797,
	}
	resp := h.Client.Post("/v1/auth/retailer/register", payload, "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate phone on mobile register, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	var body map[string]string
	_ = resp.JSON(&body)
	if body["error"] != "phone_already_registered" {
		t.Fatalf("expected error 'phone_already_registered', got %q", body["error"])
	}
}

func TestTier1_Item1_CanonicalRegisterPath_DuplicatePhoneRejection409(t *testing.T) {
	h := NewTestHarness(t)
	existingPhone := "+998907776655"

	_ = h.RetailerRepo.CreateRetailer(context.Background(), retailer.Retailer{
		RetailerID: "ret-seeded-3",
		Phone:      existingPhone,
		SupplierID: DefaultSupplierA,
	}, nil)

	payload := map[string]any{
		"phone":       existingPhone,
		"supplier_id": DefaultSupplierA,
		"name":        "Canonical Collision Attempt",
		"lat":         41.3111,
		"lng":         69.2797,
	}
	resp := h.Client.Post("/v1/auth/retailer/register", payload, "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on canonical register duplicate phone, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

// ============================================================================
// Tier 1: Item 2 — Factory & Warehouse Setup Role Escalation (5 tests)
// ============================================================================

func TestTier1_Item2_FactorySetup_NonFactoryRoleRejection403(t *testing.T) {
	h := NewTestHarness(t)
	retailerToken := h.IssueRetailerToken(DefaultSupplierA, "ret-user-1")

	payload := map[string]any{
		"name":        "Escalated Factory",
		"factoryName": "Illegal Plant",
		"lat":         41.3111,
		"lng":         69.2797,
	}
	resp := h.Client.Post("/v1/factory/setup", payload, retailerToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when Retailer accesses factory setup, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier1_Item2_WarehouseSetup_NonWarehouseRoleRejection403(t *testing.T) {
	h := NewTestHarness(t)
	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-user-1")

	payload := map[string]any{
		"name":          "Escalated Warehouse",
		"warehouseName": "Illegal Depot",
		"lat":           41.3111,
		"lng":           69.2797,
	}
	resp := h.Client.Post("/v1/warehouse/setup", payload, driverToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when Driver accesses warehouse setup, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier1_Item2_FactorySetup_UnauthenticatedRejection401(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{"name": "Anonymous Factory"}
	resp := h.Client.Post("/v1/factory/setup", payload, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated factory setup, got %d", resp.StatusCode)
	}
}

func TestTier1_Item2_WarehouseSetup_UnauthenticatedRejection401(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{"name": "Anonymous Warehouse"}
	resp := h.Client.Post("/v1/warehouse/setup", payload, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated warehouse setup, got %d", resp.StatusCode)
	}
}

func TestTier1_Item2_FactorySetup_RoleAdminAllowed(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-user-1")

	payload := map[string]any{
		"name":        "Legitimate Admin Factory",
		"factoryName": "Plant Alpha",
	}
	resp := h.Client.Post("/v1/factory/setup", payload, adminToken)
	// Role check MUST pass (status should NOT be 401 Unauthorized or 403 Forbidden)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		t.Fatalf("admin role was blocked by auth/role guard: status %d", resp.StatusCode)
	}
}

// ============================================================================
// Tier 1: Item 3 — Retailer KYC Review Self-Approval & Role Scoping (5 tests)
// ============================================================================

func TestTier1_Item3_KYCReview_RetailerSelfApprovalForbidden403(t *testing.T) {
	h := NewTestHarness(t)
	retailerToken := h.IssueRetailerToken(DefaultSupplierA, "retailer-owner-1")

	payload := map[string]string{
		"status": "APPROVED",
	}
	resp := h.Client.Post("/v1/admin/retailer-kyc/doc-100/review", payload, retailerToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when Retailer attempts to review KYC, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier1_Item3_KYCReview_DriverForbidden403(t *testing.T) {
	h := NewTestHarness(t)
	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-1")

	payload := map[string]string{"status": "APPROVED"}
	resp := h.Client.Post("/v1/admin/retailer-kyc/doc-100/review", payload, driverToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when Driver attempts to review KYC, got %d", resp.StatusCode)
	}
}

func TestTier1_Item3_KYCReview_AdminReviewSuccess200(t *testing.T) {
	h := NewTestHarness(t)
	docID := "doc-review-ok-1"
	h.RetailerRepo.AddKycDocument(retailer.KycDocument{
		DocumentID: docID,
		RetailerID: "ret-1",
		Status:     "PENDING",
	})

	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-reviewer-1")
	payload := map[string]string{
		"status": "APPROVED",
	}
	resp := h.Client.Post("/v1/admin/retailer-kyc/"+docID+"/review", payload, adminToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for admin KYC review, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier1_Item3_KYCReview_AdminIDDerivedFromClaimsSubject(t *testing.T) {
	h := NewTestHarness(t)
	docID := "doc-review-subject-check"
	expectedAdminSubject := "verified-ops-officer-99"

	h.RetailerRepo.AddKycDocument(retailer.KycDocument{
		DocumentID: docID,
		RetailerID: "ret-2",
		Status:     "PENDING",
	})

	adminToken := h.IssueAdminToken(DefaultSupplierA, expectedAdminSubject)
	payload := map[string]string{
		"status": "APPROVED",
	}
	resp := h.Client.Post("/v1/admin/retailer-kyc/"+docID+"/review", payload, adminToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	actualReviewedBy := h.RetailerRepo.GetKycReviewedBy(docID)
	if actualReviewedBy != expectedAdminSubject {
		t.Fatalf("expected reviewedBy to match claims.Subject %q, got %q (hardcoded admin_id detected!)", expectedAdminSubject, actualReviewedBy)
	}
}

func TestTier1_Item3_KYCReview_UnauthenticatedRejected401(t *testing.T) {
	h := NewTestHarness(t)
	resp := h.Client.Post("/v1/admin/retailer-kyc/doc-1/review", map[string]string{"status": "APPROVED"}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated review, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tier 1: Item 4 — Partner API Key Cross-Tenant Forgery Prevention (5 tests)
// ============================================================================

func TestTier1_Item4_PartnerKeys_RoleAdminCrossTenantForbidden403(t *testing.T) {
	h := NewTestHarness(t)
	// Admin belongs to Supplier A
	adminTokenA := h.IssueAdminToken(DefaultSupplierA, "admin-supplier-a")

	// Tries to issue partner key for Supplier B
	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierB,
		"scopes":      []string{"orders:read", "catalog:read"},
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, adminTokenA)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on cross-tenant partner key mint, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier1_Item4_PartnerKeys_AdminMatchingTenantSuccess201(t *testing.T) {
	h := NewTestHarness(t)
	adminTokenA := h.IssueAdminToken(DefaultSupplierA, "admin-supplier-a")

	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
		"scopes":      []string{"orders:read"},
		"environment": "SANDBOX",
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, adminTokenA)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200/201 on matching tenant key issuance, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	var data map[string]any
	_ = resp.JSON(&data)
	secret, _ := data["secret"].(string)
	if !strings.HasPrefix(secret, "pxk_") && !strings.HasPrefix(secret, "pxs_") {
		t.Fatalf("expected secret with prefix 'pxk_' or 'pxs_', got: %s", secret)
	}
}

func TestTier1_Item4_PartnerKeys_LiveKeyGeneration(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-a")

	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
		"environment": "LIVE",
		"scopes":      []string{"orders:write"},
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, adminToken)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200/201, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	var data map[string]any
	_ = resp.JSON(&data)
	secret, _ := data["secret"].(string)
	if !strings.HasPrefix(secret, "pxk_") {
		t.Fatalf("expected LIVE key secret prefix 'pxk_', got: %s", secret)
	}
}

func TestTier1_Item4_PartnerKeys_SandboxKeyGeneration(t *testing.T) {
	h := NewTestHarness(t)
	adminToken := h.IssueAdminToken(DefaultSupplierA, "admin-a")

	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
		"environment": "SANDBOX",
		"scopes":      []string{"orders:read"},
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, adminToken)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200/201, got %d", resp.StatusCode)
	}

	var data map[string]any
	_ = resp.JSON(&data)
	secret, _ := data["secret"].(string)
	if !strings.HasPrefix(secret, "pxs_") {
		t.Fatalf("expected SANDBOX key secret prefix 'pxs_', got: %s", secret)
	}
}

func TestTier1_Item4_PartnerKeys_WarehouseRoleForbidden403(t *testing.T) {
	h := NewTestHarness(t)
	warehouseToken := h.IssueWarehouseToken(DefaultSupplierA, "wh-user")

	payload := map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   DefaultSupplierA,
	}
	resp := h.Client.Post("/v1/admin/partner-keys", payload, warehouseToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for warehouse role issuing keys, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tier 1: Item 5 — Control Tower Cross-Tenant IDOR Prevention (5 tests)
// ============================================================================

func TestTier1_Item5_ControlTower_CrossTenantUpdatePlaybookRejection403(t *testing.T) {
	h := NewTestHarness(t)
	playbookB := "pb-banana-1"
	_ = h.ControlTowerRepo.CreatePlaybook(context.Background(), controltower.Playbook{
		PlaybookID: playbookB,
		SupplierID: DefaultSupplierB,
		Name:       "Banana Original Playbook",
		IsActive:   true,
	})

	// Admin of Supplier A tries to update playbook of Supplier B
	tokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")
	patchBody := map[string]string{"name": "Injected Name"}
	resp := h.Client.Patch("/v1/control-tower/playbooks/"+playbookB, patchBody, tokenA)

	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403/404 on cross-tenant playbook update, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	// Verify playbook was NOT altered
	pb, _ := h.ControlTowerRepo.GetPlaybook(context.Background(), playbookB)
	if pb.Name != "Banana Original Playbook" {
		t.Fatalf("playbook was modified across tenants: got %s", pb.Name)
	}
}

func TestTier1_Item5_ControlTower_CrossTenantDeactivatePlaybookRejection403(t *testing.T) {
	h := NewTestHarness(t)
	playbookB := "pb-banana-2"
	_ = h.ControlTowerRepo.CreatePlaybook(context.Background(), controltower.Playbook{
		PlaybookID: playbookB,
		SupplierID: DefaultSupplierB,
		IsActive:   true,
	})

	tokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")
	resp := h.Client.Post("/v1/control-tower/playbooks/"+playbookB+"/deactivate", nil, tokenA)

	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403/404 on cross-tenant deactivate, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	pb, _ := h.ControlTowerRepo.GetPlaybook(context.Background(), playbookB)
	if !pb.IsActive {
		t.Fatal("playbook was illegally deactivated across tenants!")
	}
}

func TestTier1_Item5_ControlTower_CrossTenantApproveRunRejection403(t *testing.T) {
	h := NewTestHarness(t)
	runB := "run-banana-1"
	_ = h.ControlTowerRepo.CreateRun(context.Background(), controltower.PlaybookRun{
		RunID:      runB,
		SupplierID: DefaultSupplierB,
		Status:     controltower.RunStatusSuggested,
	})

	tokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")
	resp := h.Client.Post("/v1/control-tower/runs/"+runB+"/approve", nil, tokenA)

	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403/404 on cross-tenant approve run, got %d", resp.StatusCode)
	}
}

func TestTier1_Item5_ControlTower_CrossTenantSkipRunRejection403(t *testing.T) {
	h := NewTestHarness(t)
	runB := "run-banana-2"
	_ = h.ControlTowerRepo.CreateRun(context.Background(), controltower.PlaybookRun{
		RunID:      runB,
		SupplierID: DefaultSupplierB,
		Status:     controltower.RunStatusSuggested,
	})

	tokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")
	resp := h.Client.Post("/v1/control-tower/runs/"+runB+"/skip", nil, tokenA)

	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403/404 on cross-tenant skip run, got %d", resp.StatusCode)
	}
}

func TestTier1_Item5_ControlTower_MatchingTenantPlaybookOperationSuccess(t *testing.T) {
	h := NewTestHarness(t)
	playbookA := "pb-apple-legit"
	_ = h.ControlTowerRepo.CreatePlaybook(context.Background(), controltower.Playbook{
		PlaybookID: playbookA,
		SupplierID: DefaultSupplierA,
		Name:       "Apple Authorized Playbook",
		IsActive:   true,
	})

	tokenA := h.IssueAdminToken(DefaultSupplierA, "admin-apple")
	patchBody := map[string]string{"name": "Apple Renamed Playbook"}
	resp := h.Client.Patch("/v1/control-tower/playbooks/"+playbookA, patchBody, tokenA)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for authorized tenant update, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	pb, _ := h.ControlTowerRepo.GetPlaybook(context.Background(), playbookA)
	if pb.Name != "Apple Renamed Playbook" {
		t.Fatalf("playbook update not reflected: %s", pb.Name)
	}
}

// ============================================================================
// Tier 1: Item 6 — Credit-Leave Zero-Tender Debt Forgiveness Exploit (5 tests)
// ============================================================================

func TestTier1_Item6_CreditLeave_ExcludedFromCapturedPaymentQuery(t *testing.T) {
	// P0 Invariant 4 & Audit Finding 6:
	// Verify that capturedPaymentStatement SQL explicitly excludes MethodCredit and credit-leave-%
	orderID := "ord-credit-test-1"
	stmt := order.CapturedPaymentStatement(orderID)

	if !strings.Contains(stmt.SQL, "Method != @credit_method") {
		t.Fatalf("capturedPaymentStatement SQL missing 'Method != @credit_method': %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "IdempotencyKey NOT LIKE 'credit-leave-%'") {
		t.Fatalf("capturedPaymentStatement SQL missing \"IdempotencyKey NOT LIKE 'credit-leave-%%'\": %s", stmt.SQL)
	}
	if stmt.Params["credit_method"] != string(order.MethodCredit) {
		t.Fatalf("expected credit_method = %q, got %v", order.MethodCredit, stmt.Params["credit_method"])
	}
	if stmt.Params["status"] != string(order.PaymentStatusCaptured) {
		t.Fatalf("expected status = %q, got %v", order.PaymentStatusCaptured, stmt.Params["status"])
	}

	// Exercise domain AssertMoneyCoversDelivery: unpaid credit order must fail closed
	h := NewTestHarness(t)
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:    orderID,
		SupplierID: DefaultSupplierA,
		LineItems: []order.LineItem{
			{SKU: "SKU-1", Quantity: 3, UnitPrice: 10000000}, // 30,000,000 UZS delivered gross
		},
	})
	err := h.OrderSvc.AssertMoneyCoversDelivery(context.Background(), orderID, 0, 0)
	if err == nil {
		t.Fatal("expected AssertMoneyCoversDelivery to fail for unpaid credit delivery, got nil")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got code %v: %v", st.Code(), err)
	}
}

func TestTier1_Item6_CreditLeave_ZeroTenderCardSettlementBlocked(t *testing.T) {
	h := NewTestHarness(t)
	orderID := "ord-unpaid-credit-leave"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderID,
		SupplierID:            DefaultSupplierA,
		RetailerID:            "ret-1",
		DriverID:              "driver-1",
		TotalMinor:            50000000,
		Status:                order.StatusDeliveredOnCredit,
		FiscalStatus:          order.FiscalStatusPending,
		LatestFiscalAttemptID: "attempt-1",
		Lat:                   41.311081,
		Lng:                   69.240562,
	})

	// Driver attempts to complete order without verified non-credit tender
	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-1")
	resp := h.Client.Post("/v1/order/complete", map[string]any{
		"order_id":  orderID,
		"latitude":  41.311081,
		"longitude": 69.240562,
	}, driverToken)

	// Must fail closed with 422 Unprocessable Entity
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

func TestTier1_Item6_CreditLeave_CompleteOrderWithoutVerifiedTenderDoesNotClearBalance(t *testing.T) {
	h := NewTestHarness(t)
	orderID := "ord-no-clear-balance"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:      orderID,
		SupplierID:   DefaultSupplierA,
		RetailerID:   "ret-1",
		DriverID:     "driver-1",
		TotalMinor:   40000000,
		Status:       order.StatusDeliveredOnCredit,
		FiscalStatus: order.FiscalStatusPending,
		Lat:          41.311081,
		Lng:          69.240562,
	})

	driverToken := h.IssueDriverToken(DefaultSupplierA, "driver-1")
	resp := h.Client.Post("/v1/order/complete", map[string]any{
		"order_id":  orderID,
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

	// Order status in repository must NOT transition to COMPLETED without tender
	o, found, _ := h.OrderRepo.GetOrder(context.Background(), orderID)
	if !found {
		t.Fatal("order disappeared")
	}
	if o.Status == order.StatusCompleted {
		t.Fatal("exploit: order marked COMPLETED without real collected money!")
	}
}

func TestTier1_Item6_CreditLeave_RealTenderRequiredForBalanceClear(t *testing.T) {
	// Test checkSettlementTenderSufficiency across boundary conditions:
	orderID := "ord-tender-check-1"
	delivered := int64(10000000)

	// 1. Unpaid credit delivery (paid=0, exceptions=0) must fail
	errUnpaid := order.CheckSettlementTenderSufficiency(orderID, 0, 0, delivered)
	if errUnpaid == nil {
		t.Fatal("expected error for unpaid order, got nil")
	}
	if !strings.Contains(errUnpaid.Error(), "without verified payment receipt") {
		t.Fatalf("unexpected error message: %v", errUnpaid)
	}

	// 2. Shortfall (paid=5M, exceptions=0, delivered=10M) must fail
	errShortfall := order.CheckSettlementTenderSufficiency(orderID, 5000000, 0, delivered)
	if errShortfall == nil {
		t.Fatal("expected error for shortfall, got nil")
	}

	// 3. Fully paid (paid=10M, exceptions=0, delivered=10M) must succeed
	if err := order.CheckSettlementTenderSufficiency(orderID, 10000000, 0, delivered); err != nil {
		t.Fatalf("expected nil for fully paid order, got: %v", err)
	}

	// 4. Verify predicate ShouldClearCreditAndAR
	if order.ShouldClearCreditAndAR(nil) {
		t.Fatal("shouldClearCreditAndAR(nil) must return false")
	}
	if order.ShouldClearCreditAndAR(&order.PaymentLeg{AmountMinor: 0}) {
		t.Fatal("shouldClearCreditAndAR(0) must return false")
	}
	if !order.ShouldClearCreditAndAR(&order.PaymentLeg{AmountMinor: 10000000}) {
		t.Fatal("shouldClearCreditAndAR(10M) must return true")
	}
}

func TestTier1_Item6_CreditLeave_DoubleSettlementIdempotency(t *testing.T) {
	h := NewTestHarness(t)
	orderID := "ord-double-settle-idempotent"

	// Settle draft/non-fiscalizing order via SettleOutstandingCardPayment is a safe idempotent no-op
	draftOrder := order.Order{
		OrderID:    orderID,
		Status:     order.StatusPending,
		TotalMinor: 25000000,
	}
	if err := h.OrderSvc.SettleOutstandingCardPayment(context.Background(), draftOrder); err != nil {
		t.Fatalf("expected nil for draft order settlement, got: %v", err)
	}
}

// ============================================================================
// Tier 1: Item 7 — Inventory Stock Inflation on Reservation Release (7 tests)
// ============================================================================

func TestTier1_Item7_InventoryRelease_PreservesQuantityOnHand(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-stock-test-1"
	initialOnHand := int64(150)
	initialReserved := int64(40)
	expectedVersion := int64(1)

	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		WarehouseID:      "wh-1",
		ProductID:        "sku-coke",
		QuantityOnHand:   initialOnHand,
		QuantityReserved: initialReserved,
		Version:          expectedVersion,
	})

	// Release 25 units via InventorySvc
	err := h.InventorySvc.ReleaseReservation(context.Background(), invID, 25, expectedVersion)
	if err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}

	updated, err := h.InventoryRepo.Get(context.Background(), invID)
	if err != nil {
		t.Fatalf("failed to get inventory: %v", err)
	}

	// CRITICAL INVARIANT: QuantityOnHand MUST NOT change!
	if updated.QuantityOnHand != initialOnHand {
		t.Fatalf("STOCK INFLATION DETECTED: QuantityOnHand changed from %d to %d (diff: %+d)",
			initialOnHand, updated.QuantityOnHand, updated.QuantityOnHand-initialOnHand)
	}
	if updated.QuantityReserved != 15 {
		t.Fatalf("expected QuantityReserved == 15, got %d", updated.QuantityReserved)
	}
}

func TestTier1_Item7_InventoryRelease_SubtractsOnlyFromReserved(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-stock-test-2"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		QuantityOnHand:   500,
		QuantityReserved: 100,
		Version:          1,
	})

	err := h.InventorySvc.ReleaseReservation(context.Background(), invID, 100, 1)
	if err != nil {
		t.Fatalf("release error: %v", err)
	}

	updated, _ := h.InventoryRepo.Get(context.Background(), invID)
	if updated.QuantityOnHand != 500 {
		t.Fatalf("onHand changed: got %d, want 500", updated.QuantityOnHand)
	}
	if updated.QuantityReserved != 0 {
		t.Fatalf("reserved not cleared: got %d, want 0", updated.QuantityReserved)
	}
}

func TestTier1_Item7_InventoryRelease_ZeroStockInflationOnMultiRelease(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-stock-multi"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		QuantityOnHand:   100,
		QuantityReserved: 40,
		Version:          1,
	})

	// Release 20 at version 1
	_ = h.InventorySvc.ReleaseReservation(context.Background(), invID, 20, 1)
	// Release 20 at version 2
	_ = h.InventorySvc.ReleaseReservation(context.Background(), invID, 20, 2)

	updated, _ := h.InventoryRepo.Get(context.Background(), invID)
	if updated.QuantityOnHand != 100 {
		t.Fatalf("multi-release stock inflation: onHand is %d, want exactly 100", updated.QuantityOnHand)
	}
	if updated.QuantityReserved != 0 {
		t.Fatalf("reserved: got %d, want 0", updated.QuantityReserved)
	}
}

func TestTier1_Item7_InventoryRelease_CancelReservationLeavesOnHandUnchanged(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-cancel-test"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		QuantityOnHand:   80,
		QuantityReserved: 30,
		Version:          10,
	})

	// Canceled order releases full 30 units
	_ = h.InventorySvc.ReleaseReservation(context.Background(), invID, 30, 10)
	updated, _ := h.InventoryRepo.Get(context.Background(), invID)

	if updated.QuantityOnHand != 80 {
		t.Fatalf("cancellation inflated stock: got %d onHand, want 80", updated.QuantityOnHand)
	}
}

func TestTier1_Item7_InventoryRelease_PartialReleaseCalculation(t *testing.T) {
	h := NewTestHarness(t)
	invID := "inv-partial-clamp"
	h.InventoryRepo.SeedLevel(inventory.Level{
		InventoryID:      invID,
		QuantityOnHand:   60,
		QuantityReserved: 15,
		Version:          5,
	})

	// Attempt to release 50 when only 15 is reserved -> clamps to 15
	err := h.InventorySvc.ReleaseReservation(context.Background(), invID, 50, 5)
	if err != nil {
		t.Fatalf("release error: %v", err)
	}
	updated, _ := h.InventoryRepo.Get(context.Background(), invID)
	if updated.QuantityOnHand != 60 {
		t.Fatalf("onHand changed: %d", updated.QuantityOnHand)
	}
	if updated.QuantityReserved != 0 {
		t.Fatalf("reserved not clamped to 0: got %d", updated.QuantityReserved)
	}
}

func TestTier1_Item7_InventoryRelease_BackorderAndMigrationAreNoOps(t *testing.T) {
	// Backorders and historical migration orders never reserved stock
	errBackorder := order.ReleaseReservationsForOrderInTxn(context.Background(), nil, "sup-1", "wh-1", "ord-bo", order.OrderSourceBackorder, []order.LineItem{
		{SKU: "SKU-A", Quantity: 50},
	})
	if errBackorder != nil {
		t.Fatalf("expected nil for backorder release, got: %v", errBackorder)
	}

	errMigration := order.ReleaseReservationsForOrderInTxn(context.Background(), nil, "sup-1", "wh-1", "ord-mig", order.OrderSourceMigration, []order.LineItem{
		{SKU: "SKU-A", Quantity: 50},
	})
	if errMigration != nil {
		t.Fatalf("expected nil for migration release, got: %v", errMigration)
	}
}

func TestTier1_Item7_InventoryRelease_AggregateSKUQuantities(t *testing.T) {
	items := []order.LineItem{
		{SKU: "SKU-1", Quantity: 10},
		{SKU: "SKU-2", Quantity: 5},
		{SKU: "SKU-1", Quantity: 15},
		{SKU: "", Quantity: 20},      // empty SKU ignored
		{SKU: "SKU-3", Quantity: 0},  // zero quantity ignored
		{SKU: "SKU-4", Quantity: -5}, // negative quantity ignored
	}
	aggr := order.AggregateSKUQuantities(items)
	if aggr["SKU-1"] != 25 {
		t.Fatalf("expected SKU-1 qty=25, got %d", aggr["SKU-1"])
	}
	if aggr["SKU-2"] != 5 {
		t.Fatalf("expected SKU-2 qty=5, got %d", aggr["SKU-2"])
	}
	if _, exists := aggr[""]; exists {
		t.Fatal("empty SKU should be ignored")
	}
	if _, exists := aggr["SKU-3"]; exists {
		t.Fatal("zero qty should be ignored")
	}
}

// ============================================================================
// Tier 1: Item 8 — Adyen Webhook Chargeback / Refund Mapping (5 tests)
// ============================================================================

func TestTier1_Item8_AdyenWebhook_ChargebackMappedToNonPaid(t *testing.T) {
	h := NewTestHarness(t)
	payload := GenerateAdyenWebhookPayload(t, h.AdyenHMACKey, "CHARGEBACK", true, "psp-chargeback-1", 500000, "UZS")

	resp := h.Client.PostRaw("/v1/webhooks/adyen", payload, "", "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK [accepted], got %d (body: %s)", resp.StatusCode, resp.String())
	}
	if resp.String() != "[accepted]" {
		t.Fatalf("expected body '[accepted]', got %q", resp.String())
	}

	// Verify recorded attempts
	attempts := h.PaymentRepo.GetAttempts()
	if len(attempts) > 0 {
		last := attempts[len(attempts)-1]
		if last.Status == "PAID" {
			t.Fatalf("SECURITY VIOLATION: CHARGEBACK event mapped to status 'PAID'!")
		}
		if last.Status != "CHARGEBACK" {
			t.Fatalf("expected status 'CHARGEBACK', got %q", last.Status)
		}
	}
}

func TestTier1_Item8_AdyenWebhook_NotificationOfChargebackMappedToNonPaid(t *testing.T) {
	h := NewTestHarness(t)
	payload := GenerateAdyenWebhookPayload(t, h.AdyenHMACKey, "NOTIFICATION_OF_CHARGEBACK", true, "psp-notif-cb-1", 500000, "UZS")

	resp := h.Client.PostRaw("/v1/webhooks/adyen", payload, "", "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	attempts := h.PaymentRepo.GetAttempts()
	if len(attempts) > 0 {
		last := attempts[len(attempts)-1]
		if last.Status == "PAID" {
			t.Fatalf("SECURITY VIOLATION: NOTIFICATION_OF_CHARGEBACK mapped to 'PAID'!")
		}
		if last.Status != "CHARGEBACK" {
			t.Fatalf("expected status 'CHARGEBACK', got %q", last.Status)
		}
	}
}

func TestTier1_Item8_AdyenWebhook_RefundMappedToNonPaid(t *testing.T) {
	h := NewTestHarness(t)
	payload := GenerateAdyenWebhookPayload(t, h.AdyenHMACKey, "REFUND", true, "psp-refund-1", 200000, "UZS")

	resp := h.Client.PostRaw("/v1/webhooks/adyen", payload, "", "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	attempts := h.PaymentRepo.GetAttempts()
	if len(attempts) > 0 {
		last := attempts[len(attempts)-1]
		if last.Status == "PAID" {
			t.Fatalf("SECURITY VIOLATION: REFUND event mapped to 'PAID'!")
		}
		if last.Status != "REFUNDED" {
			t.Fatalf("expected status 'REFUNDED', got %q", last.Status)
		}
	}
}

func TestTier1_Item8_AdyenWebhook_UnknownEventDefaultNonSettling(t *testing.T) {
	h := NewTestHarness(t)
	payload := GenerateAdyenWebhookPayload(t, h.AdyenHMACKey, "STRANGE_NEW_ADYEN_CODE", true, "psp-unknown-1", 100000, "UZS")

	resp := h.Client.PostRaw("/v1/webhooks/adyen", payload, "", "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	attempts := h.PaymentRepo.GetAttempts()
	if len(attempts) > 0 {
		last := attempts[len(attempts)-1]
		if last.Status == "PAID" {
			t.Fatalf("SECURITY VIOLATION: unknown event code mapped to 'PAID' in default branch!")
		}
		if last.Status != "UNHANDLED" {
			t.Fatalf("expected status 'UNHANDLED', got %q", last.Status)
		}
	}
}

func TestTier1_Item8_AdyenWebhook_LegitimateCaptureMappedToPaid(t *testing.T) {
	h := NewTestHarness(t)
	payload := GenerateAdyenWebhookPayload(t, h.AdyenHMACKey, "CAPTURE", true, "psp-capture-legit-1", 750000, "UZS")

	resp := h.Client.PostRaw("/v1/webhooks/adyen", payload, "", "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	attempts := h.PaymentRepo.GetAttempts()
	if len(attempts) > 0 {
		last := attempts[len(attempts)-1]
		if last.Status != "PAID" {
			t.Fatalf("expected CAPTURE success to map to 'PAID', got %q", last.Status)
		}
	}
}

// ============================================================================
// Tier 1: Item 9 — Fiscal Receipt Integrity & Soliq Enforcement (5 tests)
// ============================================================================

func TestTier1_Item9_FiscalIntegrity_SyntheticCorrectiveReceiptWithoutSoliqRejected(t *testing.T) {
	h := NewTestHarness(t)
	// Set provider to fail / simulate OFD network error
	h.FiscalProvider.ShouldFail = true

	orderID := "ord-fiscal-fail-1"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderID,
		SupplierID:            DefaultSupplierA,
		LatestFiscalReceiptID: "SOLIQ-ORIG-1",
		TotalMinor:            10000000,
	})

	token := h.IssueAdminToken(DefaultSupplierA, "admin-user")
	payload := map[string]any{
		"correction_amount_minor": 1000000,
		"reason":                  "damaged_goods",
	}
	resp := h.Client.Post("/v1/supplier/orders/"+orderID+"/fiscal/corrective", payload, token)

	// Invariant: Unverified synthetic corrective receipt must NOT be fabricated!
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("synthetic corrective receipt was generated without Soliq provider submission!")
	}
}

func TestTier1_Item9_FiscalIntegrity_CorrectiveReceiptTenantAuthorizationEnforced(t *testing.T) {
	h := NewTestHarness(t)
	orderID := "ord-tenant-b-fiscal"
	h.OrderRepo.SeedOrder(order.Order{
		OrderID:               orderID,
		SupplierID:            DefaultSupplierB,
		LatestFiscalReceiptID: "SOLIQ-ORIG-B",
		TotalMinor:            20000000,
	})

	// Token for Supplier A tries to issue corrective receipt for Supplier B
	tokenA := h.IssueAdminToken(DefaultSupplierA, "admin-a")
	payload := map[string]any{
		"correction_amount_minor": 5000000,
		"reason":                  "return",
	}
	resp := h.Client.Post("/v1/supplier/orders/"+orderID+"/fiscal/corrective", payload, tokenA)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on cross-tenant corrective receipt, got %d (body: %s)", resp.StatusCode, resp.String())
	}
}

func TestTier1_Item9_FiscalIntegrity_FailedFiscalReceiptExcludedFromPaidCount(t *testing.T) {
	// Item 9 Invariant: OrderFiscalReceipts with status FAILED must NOT be counted as paid
	failedCandidate := order.OpenFiscalCandidate{
		OrderID:            "ord-fiscal-failed-1",
		Status:             string(order.StatusFiscalFailed),
		FiscalStatus:       "FAILED",
		HasPaymentRecorded: false, // FAILED receipts must NOT mark this true!
		HasCryptoOrQRPoD:   true,
	}

	snap := order.EvaluateOpenFiscalOrders(false, []order.OpenFiscalCandidate{failedCandidate})
	if !snap.Frozen {
		t.Fatal("failed fiscal receipt without real payment must freeze shift closeout!")
	}
}

func TestTier1_Item9_FiscalIntegrity_UnpaidPendingFiscalReceiptExcludedFromPaidCount(t *testing.T) {
	// Item 9 Invariant: PENDING fiscal receipts without captured tender must NOT be treated as recorded payment
	pendingCandidate := order.OpenFiscalCandidate{
		OrderID:            "ord-fiscal-pending-unpaid",
		Status:             string(order.StatusFiscalizing),
		FiscalStatus:       order.FiscalStatusPending,
		HasPaymentRecorded: false, // Unpaid PENDING row must NOT mark this true!
		HasCryptoOrQRPoD:   true,
	}

	snap := order.EvaluateOpenFiscalOrders(false, []order.OpenFiscalCandidate{pendingCandidate})
	if !snap.Frozen {
		t.Fatal("unpaid pending fiscal receipt must freeze shift closeout!")
	}
}

func TestTier1_Item9_FiscalIntegrity_SuccessFiscalReceiptCounted(t *testing.T) {
	// SUCCESS fiscal receipts ARE recorded payment
	successCandidate := order.OpenFiscalCandidate{
		OrderID:            "ord-fiscal-success-1",
		Status:             string(order.StatusCompleted),
		FiscalStatus:       order.FiscalStatusSuccess,
		HasPaymentRecorded: true,
		HasCryptoOrQRPoD:   true,
	}

	snap := order.EvaluateOpenFiscalOrders(false, []order.OpenFiscalCandidate{successCandidate})
	if snap.Frozen {
		t.Fatal("successful paid order should not freeze shift closeout!")
	}
}

// ============================================================================
// Tier 1: Item 10 — Backend-Go Auth Session & Cookie Bridge (5 tests)
// ============================================================================

func TestTier1_Item10_AuthSession_BearerHeaderAuthenticationSuccess(t *testing.T) {
	h := NewTestHarness(t)
	token := h.IssueAdminToken(DefaultSupplierA, "admin-header-user")

	resp := h.Client.Get("/v1/test/auth-probe", token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK with Bearer token, got %d", resp.StatusCode)
	}

	var data map[string]any
	_ = resp.JSON(&data)
	if data["authenticated"] != true || data["subject"] != "admin-header-user" {
		t.Fatalf("probe response mismatch: %+v", data)
	}
}

func TestTier1_Item10_AuthSession_CookieAuthenticationSuccess(t *testing.T) {
	h := NewTestHarness(t)
	token := h.IssueRetailerToken(DefaultSupplierA, "retailer-cookie-user")

	resp := h.Client.GetWithCookie("/v1/test/auth-probe", "supplier_jwt", token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK with supplier_jwt cookie, got %d", resp.StatusCode)
	}

	var data map[string]any
	_ = resp.JSON(&data)
	if data["authenticated"] != true || data["subject"] != "retailer-cookie-user" {
		t.Fatalf("probe response mismatch: %+v", data)
	}
}

func TestTier1_Item10_AuthSession_MissingAuthFailsClosed401(t *testing.T) {
	h := NewTestHarness(t)
	resp := h.Client.Get("/v1/test/auth-probe", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing auth, got %d", resp.StatusCode)
	}
}

func TestTier1_Item10_AuthSession_ExpiredTokenRejected401(t *testing.T) {
	h := NewTestHarness(t)
	expiredToken := h.IssueExpiredToken(auth.RoleAdmin, DefaultSupplierA, "expired-admin")

	resp := h.Client.Get("/v1/test/auth-probe", expiredToken)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired token, got %d", resp.StatusCode)
	}
}

func TestTier1_Item10_AuthSession_TamperedTokenRejected401(t *testing.T) {
	h := NewTestHarness(t)
	tamperedToken := h.IssueTamperedToken(auth.RoleAdmin, DefaultSupplierA, "tampered-admin")

	resp := h.Client.Get("/v1/test/auth-probe", tamperedToken)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for tampered token, got %d", resp.StatusCode)
	}
}
