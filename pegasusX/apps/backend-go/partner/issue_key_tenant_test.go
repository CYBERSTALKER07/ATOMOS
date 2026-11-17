package partner

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleIssueKey_SupplierAdmin_CannotForgeDifferentSupplier(t *testing.T) {
	repo := NewMemoryKeyRepository()
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}

	body, err := json.Marshal(map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   "sup-victim",
		"scopes":      []string{"*"},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "attacker-user",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-attacker",
	}))

	rr := httptest.NewRecorder()
	h.HandleIssueKey(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleIssueKey_SupplierAdmin_CannotForgeRetailer(t *testing.T) {
	repo := NewMemoryKeyRepository()
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}

	body, err := json.Marshal(map[string]any{
		"tenant_type": "RETAILER",
		"tenant_id":   "ret-victim",
		"scopes":      []string{"*"},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "attacker-user",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-attacker",
	}))

	rr := httptest.NewRecorder()
	h.HandleIssueKey(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleIssueKey_SupplierRoleAdmin_CannotForgeDifferentSupplier(t *testing.T) {
	repo := NewMemoryKeyRepository()
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}

	body, err := json.Marshal(map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   "sup-other",
		"scopes":      []string{"*"},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:      "staff-user",
		Role:         auth.Role("SUPPLIER"),
		SupplierRole: auth.RoleAdmin,
		SupplierID:   "sup-myown",
	}))

	rr := httptest.NewRecorder()
	h.HandleIssueKey(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleIssueKey_SupplierAdmin_HappyPathOwnSupplier(t *testing.T) {
	repo := NewMemoryKeyRepository()
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}

	// Test 1: Explicit matching tenant info
	body, err := json.Marshal(map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   "sup-legit",
		"scopes":      []string{ScopeOrdersRead, ScopeOrdersWrite},
		"environment": "SANDBOX",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "legit-admin",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-legit",
	}))

	rr := httptest.NewRecorder()
	h.HandleIssueKey(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp struct {
		KeyID      string `json:"key_id"`
		Key        string `json:"key"`
		TenantType string `json:"tenant_type"`
		TenantID   string `json:"tenant_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.TenantType != TenantSupplier {
		t.Fatalf("expected TenantType %s, got %s", TenantSupplier, resp.TenantType)
	}
	if resp.TenantID != "sup-legit" {
		t.Fatalf("expected TenantID sup-legit, got %s", resp.TenantID)
	}

	// Test 2: Omitted tenant fields defaulted to supplier admin's own supplier
	body2, _ := json.Marshal(map[string]any{
		"scopes": []string{ScopeOrdersRead},
	})
	req2 := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys", bytes.NewReader(body2))
	req2 = req2.WithContext(auth.WithClaims(req2.Context(), auth.Claims{
		Subject:    "legit-admin",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-legit",
	}))

	rr2 := httptest.NewRecorder()
	h.HandleIssueKey(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d (body: %s)", rr2.Code, rr2.Body.String())
	}
	var resp2 struct {
		TenantType string `json:"tenant_type"`
		TenantID   string `json:"tenant_id"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp2.TenantType != TenantSupplier || resp2.TenantID != "sup-legit" {
		t.Fatalf("expected pinned supplier sup-legit, got %s/%s", resp2.TenantType, resp2.TenantID)
	}
}

func TestHandleIssueKey_InvalidBody(t *testing.T) {
	repo := NewMemoryKeyRepository()
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys", bytes.NewReader([]byte("{invalid-json")))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "admin-user",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	}))

	rr := httptest.NewRecorder()
	h.HandleIssueKey(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", rr.Code)
	}
}

func TestHandleIssueKey_RetailerStaffPrivileges(t *testing.T) {
	repo := NewMemoryKeyRepository()
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}

	body, _ := json.Marshal(map[string]any{
		"tenant_type": "RETAILER",
		"tenant_id":   "ret-100",
		"scopes":      []string{"*"},
	})

	// 1. CASHIER should be rejected (403)
	reqCashier := httptest.NewRequest(http.MethodPost, "/v1/retailer/partner-keys", bytes.NewReader(body))
	reqCashier = reqCashier.WithContext(auth.WithClaims(reqCashier.Context(), auth.Claims{
		Subject:       "cashier-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-100",
		RetailerRole:  "CASHIER",
	}))
	rrCashier := httptest.NewRecorder()
	h.HandleIssueKey(rrCashier, reqCashier)
	if rrCashier.Code != http.StatusForbidden {
		t.Fatalf("expected CASHIER status 403 Forbidden, got %d", rrCashier.Code)
	}

	// 2. VIEWER should be rejected (403)
	reqViewer := httptest.NewRequest(http.MethodPost, "/v1/retailer/partner-keys", bytes.NewReader(body))
	reqViewer = reqViewer.WithContext(auth.WithClaims(reqViewer.Context(), auth.Claims{
		Subject:       "viewer-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-100",
		RetailerRole:  "VIEWER",
	}))
	rrViewer := httptest.NewRecorder()
	h.HandleIssueKey(rrViewer, reqViewer)
	if rrViewer.Code != http.StatusForbidden {
		t.Fatalf("expected VIEWER status 403 Forbidden, got %d", rrViewer.Code)
	}

	// 3. ADMIN (has PermStaffManage and PermCapManage) should succeed (201)
	reqAdmin := httptest.NewRequest(http.MethodPost, "/v1/retailer/partner-keys", bytes.NewReader(body))
	reqAdmin = reqAdmin.WithContext(auth.WithClaims(reqAdmin.Context(), auth.Claims{
		Subject:       "admin-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-100",
		RetailerRole:  "ADMIN",
	}))
	rrAdmin := httptest.NewRecorder()
	h.HandleIssueKey(rrAdmin, reqAdmin)
	if rrAdmin.Code != http.StatusCreated {
		t.Fatalf("expected ADMIN status 201 Created, got %d body=%s", rrAdmin.Code, rrAdmin.Body.String())
	}

	// 4. OWNER should succeed (201)
	reqOwner := httptest.NewRequest(http.MethodPost, "/v1/retailer/partner-keys", bytes.NewReader(body))
	reqOwner = reqOwner.WithContext(auth.WithClaims(reqOwner.Context(), auth.Claims{
		Subject:       "owner-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-100",
		RetailerRole:  "OWNER",
	}))
	rrOwner := httptest.NewRecorder()
	h.HandleIssueKey(rrOwner, reqOwner)
	if rrOwner.Code != http.StatusCreated {
		t.Fatalf("expected OWNER status 201 Created, got %d body=%s", rrOwner.Code, rrOwner.Body.String())
	}

	// 5. Legacy default owner (empty RetailerRole) should succeed (201)
	reqLegacy := httptest.NewRequest(http.MethodPost, "/v1/retailer/partner-keys", bytes.NewReader(body))
	reqLegacy = reqLegacy.WithContext(auth.WithClaims(reqLegacy.Context(), auth.Claims{
		Subject:       "legacy-owner-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-100",
	}))
	rrLegacy := httptest.NewRecorder()
	h.HandleIssueKey(rrLegacy, reqLegacy)
	if rrLegacy.Code != http.StatusCreated {
		t.Fatalf("expected legacy owner status 201 Created, got %d body=%s", rrLegacy.Code, rrLegacy.Body.String())
	}

	// 6. Retailer attempting to forge different retailer tenant should be rejected (403)
	forgedBody, _ := json.Marshal(map[string]any{
		"tenant_type": "RETAILER",
		"tenant_id":   "ret-victim-org",
		"scopes":      []string{"*"},
	})
	reqForge := httptest.NewRequest(http.MethodPost, "/v1/retailer/partner-keys", bytes.NewReader(forgedBody))
	reqForge = reqForge.WithContext(auth.WithClaims(reqForge.Context(), auth.Claims{
		Subject:       "owner-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-100",
		RetailerRole:  "OWNER",
	}))
	rrForge := httptest.NewRecorder()
	h.HandleIssueKey(rrForge, reqForge)
	if rrForge.Code != http.StatusForbidden {
		t.Fatalf("expected forged tenant status 403 Forbidden, got %d", rrForge.Code)
	}

	// 7. ListKeys should reject CASHIER (403)
	reqListCashier := httptest.NewRequest(http.MethodGet, "/v1/retailer/partner-keys", nil)
	reqListCashier = reqListCashier.WithContext(auth.WithClaims(reqListCashier.Context(), auth.Claims{
		Subject:       "cashier-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-100",
		RetailerRole:  "CASHIER",
	}))
	rrListCashier := httptest.NewRecorder()
	h.HandleListKeys(rrListCashier, reqListCashier)
	if rrListCashier.Code != http.StatusForbidden {
		t.Fatalf("expected ListKeys CASHIER status 403 Forbidden, got %d", rrListCashier.Code)
	}

	// 8. RevokeKey should reject CASHIER (403)
	reqRevokeCashier := httptest.NewRequest(http.MethodPost, "/v1/retailer/partner-keys/key-123/revoke", nil)
	reqRevokeCashier = reqRevokeCashier.WithContext(auth.WithClaims(reqRevokeCashier.Context(), auth.Claims{
		Subject:       "cashier-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-100",
		RetailerRole:  "CASHIER",
	}))
	rrRevokeCashier := httptest.NewRecorder()
	h.HandleRevokeKey(rrRevokeCashier, reqRevokeCashier)
	if rrRevokeCashier.Code != http.StatusForbidden {
		t.Fatalf("expected RevokeKey CASHIER status 403 Forbidden, got %d", rrRevokeCashier.Code)
	}
}
