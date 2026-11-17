package p2proutes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2proutes"
)

type routeStubGL struct {
	mu    sync.Mutex
	count int
}

func (g *routeStubGL) PostJournal(_ context.Context, in p2p.GLPostInput) (p2p.GLPostResult, error) {
	var d, c int64
	for _, l := range in.Lines {
		if l.Direction == p2p.DirectionDebit {
			d += l.AmountMinor
		} else {
			c += l.AmountMinor
		}
	}
	if d != c || d <= 0 {
		return p2p.GLPostResult{}, errors.New("unbalanced")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.count++
	return p2p.GLPostResult{EntryID: "gle_rt_" + in.SourceID, PeriodID: in.PeriodID}, nil
}

func issueTestJWT(t *testing.T, secret, supplierID, sub string, role, supRole auth.Role, actorType string, audExp *time.Time, perms ...string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:          sub,
		Role:             role,
		SupplierID:       supplierID,
		SupplierRole:     supRole,
		ActorType:        actorType,
		SupplierPerms:    perms,
		AuditorExpiresAt: audExp,
	}, auth.IssueOptions{
		Secret: secret,
		Issuer: "p2proutes-test",
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return tok
}

func doHTTP(t *testing.T, h http.Handler, method, path, tok string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf []byte
	if body != nil {
		var err error
		buf, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(buf))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestP2PRoutes_RBAC_AuditorReadOnly403_Expiry401_SoD_And_Lifecycle(t *testing.T) {
	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")

	const secret = "p2proutes-test-secret-key-32-bytes!!"
	const supA = "sup-p2p-rt-a"
	const supB = "sup-p2p-rt-b"

	repo := p2p.NewMemoryRepository()
	svc := p2p.NewService(repo, p2p.ServiceConfig{
		GLPoster:                 &routeStubGL{},
		POApprovalThresholdMinor: 200_000,
	})

	r := chi.NewRouter()
	r.Use(auth.SessionAuth(secret))
	r.Use(auth.AttachTenantFromClaims)
	r.Use(auth.RequireTenant(true))
	p2proutes.RegisterRoutes(r, p2proutes.Deps{
		Service:   svc,
		JWTSecret: secret,
	})

	activeAudExp := time.Now().UTC().Add(time.Hour)
	expiredAudExp := time.Now().UTC().Add(-time.Minute)

	makerTok := issueTestJWT(t, secret, supA, "acc-1", auth.RoleAdmin, auth.SupplierRoleAccountant, auth.ActorTypeHuman, nil,
		auth.PermJournalPost, auth.PermAuditRead, auth.PermReconManage)
	checkerTok := issueTestJWT(t, secret, supA, "cfo-1", auth.RoleAdmin, auth.SupplierRoleOwner, auth.ActorTypeHuman, nil)
	whTok := issueTestJWT(t, secret, supA, "wh-1", auth.RoleWarehouseAdmin, auth.SupplierRoleWarehouseManager, auth.ActorTypeHuman, nil)
	auditorTok := issueTestJWT(t, secret, supA, "aud-1", auth.RoleAdmin, auth.SupplierRoleAuditor, auth.ActorTypeHuman, &activeAudExp)
	expiredAudTok := issueTestJWT(t, secret, supA, "aud-exp", auth.RoleAdmin, auth.SupplierRoleAuditor, auth.ActorTypeHuman, &expiredAudExp)
	agentTok := issueTestJWT(t, secret, supA, "bot-1", auth.RoleAdmin, auth.SupplierRoleAgent, auth.ActorTypeAgent, nil)
	otherTenantTok := issueTestJWT(t, secret, supB, "other-owner", auth.RoleAdmin, auth.SupplierRoleOwner, auth.ActorTypeHuman, nil)

	// 1. Expired AUDITOR -> 401 auditor_access_expired on BOTH GET and POST.
	for _, tc := range []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, "/v1/processes/templates", nil},
		{http.MethodGet, "/v1/processes/mining", nil},
		{http.MethodGet, "/v1/p2p/purchase-orders", nil},
		{http.MethodPost, "/v1/p2p/purchase-orders", map[string]any{"vendor_id": "v1", "lines": []map[string]any{{"sku": "S1", "qty_ordered": 1, "unit_cost_minor": 1000}}}},
	} {
		rec := doHTTP(t, r, tc.method, tc.path, expiredAudTok, tc.body)
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "auditor_access_expired") {
			t.Fatalf("expired auditor %s %s -> %d %s, want 401 auditor_access_expired", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}

	// 2. Active AUDITOR and AGENT -> 403 on ALL mutating endpoints.
	mutatingCases := []struct {
		path string
		body map[string]any
	}{
		{"/v1/processes/templates", map[string]any{"code": "X", "name": "X", "domain": "P2P", "steps": []map[string]any{{"code": "S", "name": "S"}}}},
		{"/v1/processes/instances", map[string]any{"object_type": "PurchaseOrder", "object_id": "po-x"}},
		{"/v1/p2p/purchase-orders", map[string]any{"vendor_id": "v1", "lines": []map[string]any{{"sku": "S1", "qty_ordered": 1, "unit_cost_minor": 1000}}}},
		{"/v1/p2p/purchase-orders/po-x/approve", map[string]any{}},
		{"/v1/p2p/receipts", map[string]any{"po_id": "po-x", "warehouse_id": "w1", "lines": []map[string]any{{"sku": "S1", "qty_received": 1}}}},
		{"/v1/p2p/vendor-invoices", map[string]any{"po_id": "po-x", "subtotal_minor": 1000, "vat_minor": 120, "total_minor": 1120}},
		{"/v1/p2p/three-way-match", map[string]any{"vendor_invoice_id": "vinv-x"}},
	}
	for _, tc := range mutatingCases {
		if rec := doHTTP(t, r, http.MethodPost, tc.path, auditorTok, tc.body); rec.Code != http.StatusForbidden {
			t.Fatalf("AUDITOR POST %s -> %d %s, want 403", tc.path, rec.Code, rec.Body.String())
		}
		if rec := doHTTP(t, r, http.MethodPost, tc.path, agentTok, tc.body); rec.Code != http.StatusForbidden {
			t.Fatalf("AGENT POST %s -> %d %s, want 403", tc.path, rec.Code, rec.Body.String())
		}
	}

	// 3. Active AUDITOR allowed 200 on all GETs.
	for _, path := range []string{
		"/v1/processes/templates",
		"/v1/processes/instances",
		"/v1/processes/mining",
		"/v1/p2p/purchase-orders",
		"/v1/p2p/receipts",
		"/v1/p2p/vendor-invoices",
	} {
		if rec := doHTTP(t, r, http.MethodGet, path, auditorTok, nil); rec.Code != http.StatusOK {
			t.Fatalf("active AUDITOR GET %s -> %d %s, want 200", path, rec.Code, rec.Body.String())
		}
	}

	// 4. End-to-end HTTP lifecycle with SoD enforcement.
	poRec := doHTTP(t, r, http.MethodPost, "/v1/p2p/purchase-orders", makerTok, map[string]any{
		"vendor_id": "vnd-http",
		"lines": []map[string]any{
			{"sku": "TEA-1KG", "qty_ordered": 5, "unit_cost_minor": 100_000}, // omits vat_bps -> defaults 1200 -> 560,000 > 200,000
		},
	})
	if poRec.Code != http.StatusAccepted {
		t.Fatalf("create high-value PO -> %d %s, want 202", poRec.Code, poRec.Body.String())
	}
	var poResp p2p.CreatePurchaseOrderResult
	_ = json.Unmarshal(poRec.Body.Bytes(), &poResp)
	poID := poResp.PurchaseOrder.PoID

	// Self-approve blocked 403 SoD.
	if rec := doHTTP(t, r, http.MethodPost, "/v1/p2p/purchase-orders/"+poID+"/approve", makerTok, map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("self-approve PO -> %d %s, want 403", rec.Code, rec.Body.String())
	}
	// Cross-tenant approve blocked 403/404.
	if rec := doHTTP(t, r, http.MethodPost, "/v1/p2p/purchase-orders/"+poID+"/approve", otherTenantTok, map[string]any{}); rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant PO approve -> %d %s, want 404", rec.Code, rec.Body.String())
	}
	// Distinct checker approves PO.
	if rec := doHTTP(t, r, http.MethodPost, "/v1/p2p/purchase-orders/"+poID+"/approve", checkerTok, map[string]any{}); rec.Code != http.StatusOK {
		t.Fatalf("checker approve PO -> %d %s, want 200", rec.Code, rec.Body.String())
	}

	// Warehouse Manager posts Goods Receipt.
	grnRec := doHTTP(t, r, http.MethodPost, "/v1/p2p/receipts", whTok, map[string]any{
		"po_id":        poID,
		"warehouse_id": "wh-rt-1",
		"lines":        []map[string]any{{"sku": "TEA-1KG", "qty_received": 5}},
	})
	if grnRec.Code != http.StatusCreated {
		t.Fatalf("warehouse post receipt -> %d %s, want 201", grnRec.Code, grnRec.Body.String())
	}

	// Accountant logs Vendor Invoice.
	invRec := doHTTP(t, r, http.MethodPost, "/v1/p2p/vendor-invoices", makerTok, map[string]any{
		"po_id":          poID,
		"subtotal_minor": 500_000,
		"vat_minor":      60_000,
		"total_minor":    560_000,
	})
	if invRec.Code != http.StatusCreated {
		t.Fatalf("create vendor invoice -> %d %s, want 201", invRec.Code, invRec.Body.String())
	}
	var invWrap struct {
		VendorInvoice p2p.VendorInvoice `json:"vendor_invoice"`
	}
	_ = json.Unmarshal(invRec.Body.Bytes(), &invWrap)

	// Maker cannot self-verify 3-way match (SoD 403).
	if rec := doHTTP(t, r, http.MethodPost, "/v1/p2p/three-way-match", makerTok, map[string]any{
		"vendor_invoice_id": invWrap.VendorInvoice.VendorInvoiceID,
	}); rec.Code != http.StatusForbidden {
		t.Fatalf("maker self-verify 3-way match -> %d %s, want 403", rec.Code, rec.Body.String())
	}

	// Checker runs 3-way match -> 200 MATCHED_3WAY.
	matchRec := doHTTP(t, r, http.MethodPost, "/v1/p2p/three-way-match", checkerTok, map[string]any{
		"vendor_invoice_id": invWrap.VendorInvoice.VendorInvoiceID,
	})
	if matchRec.Code != http.StatusOK || !strings.Contains(matchRec.Body.String(), p2p.MatchStatusMatched3Way) {
		t.Fatalf("checker 3-way match -> %d %s, want 200 MATCHED_3WAY", matchRec.Code, matchRec.Body.String())
	}

	// Active AUDITOR can inspect PO detail & Process Mining.
	if rec := doHTTP(t, r, http.MethodGet, "/v1/p2p/purchase-orders/"+poID, auditorTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("AUDITOR GET PO -> %d %s, want 200", rec.Code, rec.Body.String())
	}
	if rec := doHTTP(t, r, http.MethodGet, "/v1/processes/mining?domain=P2P", auditorTok, nil); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `"empty":true`) {
		t.Fatalf("AUDITOR GET mining -> %d %s, want 200 non-empty", rec.Code, rec.Body.String())
	}

	// Template & Instance CRUD + Float rejection + 404 + cross-tenant checks.
	if rec := doHTTP(t, r, http.MethodPost, "/v1/processes/templates", checkerTok, map[string]any{
		"code": "P2P-FAST-UZ", "name": "Fast P2P", "domain": "P2P",
		"steps": []map[string]any{{"code": "PO_CREATED", "name": "Created", "sla_seconds": 600}},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("POST template -> %d %s, want 201", rec.Code, rec.Body.String())
	}
	if rec := doHTTP(t, r, http.MethodPost, "/v1/processes/instances", checkerTok, map[string]any{
		"template_code": "P2P-FAST-UZ", "object_type": "CustomDoc", "object_id": "cdoc-1",
	}); rec.Code != http.StatusCreated {
		t.Fatalf("POST instance -> %d %s, want 201", rec.Code, rec.Body.String())
	}
	if rec := doHTTP(t, r, http.MethodGet, "/v1/p2p/purchase-orders/does-not-exist", checkerTok, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("GET missing PO -> %d %s, want 404", rec.Code, rec.Body.String())
	}
	// Float qty/unit_cost rejected cleanly.
	reqFloat := httptest.NewRequest(http.MethodPost, "/v1/p2p/purchase-orders", bytes.NewBufferString(`{"vendor_id":"v1","lines":[{"sku":"S1","qty_ordered":1.5,"unit_cost_minor":1000}]}`))
	reqFloat.Header.Set("Content-Type", "application/json")
	reqFloat.Header.Set("Authorization", "Bearer "+checkerTok)
	recFloat := httptest.NewRecorder()
	r.ServeHTTP(recFloat, reqFloat)
	if recFloat.Code != http.StatusBadRequest {
		t.Fatalf("float qty in PO -> %d %s, want 400", recFloat.Code, recFloat.Body.String())
	}
}
