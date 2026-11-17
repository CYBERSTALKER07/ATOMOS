package claims

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestHandleFileOrderClaim_IdempotentReplay(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Idem:   idempotency.NewInMemoryStore(),
		Now:    func() time.Time { return now },
		NewID:  func() string { return "idem1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	body := []byte(`{"claim_type":"MISSING","description":"dup","line_items":[{"sku":"sku-1","quantity":1,"reason":"MISSING"}],"evidences":[]}`)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", "ord-1")

	doPost := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/orders/ord-1/claims", bytes.NewReader(body))
		req.Header.Set("Idempotency-Key", "claim-file:ord-1:test")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject: "ret-1", Role: auth.RoleRetailer,
		}))
		rr := httptest.NewRecorder()
		svc.HandleFileOrderClaim(rr, req)
		return rr
	}

	rr1 := doPost()
	if rr1.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", rr1.Code, rr1.Body.String())
	}
	var c1 Claim
	if err := json.Unmarshal(rr1.Body.Bytes(), &c1); err != nil || c1.ClaimID == "" {
		t.Fatalf("first decode: %v body=%s", err, rr1.Body.String())
	}

	rr2 := doPost()
	if rr2.Code != http.StatusCreated {
		t.Fatalf("replay status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	var c2 Claim
	if err := json.Unmarshal(rr2.Body.Bytes(), &c2); err != nil {
		t.Fatal(err)
	}
	if c2.ClaimID != c1.ClaimID {
		t.Fatalf("replay claim_id=%s want %s", c2.ClaimID, c1.ClaimID)
	}

	// Same key, different body → conflict
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/ord-1/claims", bytes.NewReader([]byte(
		`{"claim_type":"MISSING","description":"other","line_items":[{"sku":"sku-1","quantity":1}],"evidences":[]}`,
	)))
	req.Header.Set("Idempotency-Key", "claim-file:ord-1:test")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	rr3 := httptest.NewRecorder()
	svc.HandleFileOrderClaim(rr3, req)
	if rr3.Code != http.StatusConflict {
		t.Fatalf("mismatch status=%d want 409 body=%s", rr3.Code, rr3.Body.String())
	}
}

func TestHandleFileOrderClaim_RequiresIdempotencyKey(t *testing.T) {
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: completedOrder(time.Now().UTC())},
		Idem:   idempotency.NewInMemoryStore(),
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", "ord-1")
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/ord-1/claims", strings.NewReader(`{"claim_type":"MISSING"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	rr := httptest.NewRecorder()
	svc.HandleFileOrderClaim(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRejectClaim_RequiresIdempotencyKey(t *testing.T) {
	svc := NewService(Config{
		Repo: NewMemoryRepository(),
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("claimID", "clm-1")
	req := httptest.NewRequest(http.MethodPost, "/v1/claims/clm-1/reject", strings.NewReader(`{"reason":"Invalid"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}))
	rr := httptest.NewRecorder()
	svc.HandleRejectClaim(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "idempotency_key_required") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleFileOrderClaim_CrossTenantIdempotencyIsolated(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: completedOrder(now)}, // owned by RetailerID ret-1
		Idem:   idempotency.NewInMemoryStore(),
		Now:    func() time.Time { return now },
		NewID:  func() string { return "idem-iso-1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	body := []byte(`{"claim_type":"MISSING","description":"dup","line_items":[{"sku":"sku-1","quantity":1,"reason":"MISSING"}],"evidences":[]}`)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", "ord-1")

	postAs := func(orgID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/orders/ord-1/claims", bytes.NewReader(body))
		req.Header.Set("Idempotency-Key", "shared-tenant-claim-key")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject:       "multi-org-user",
			Role:          auth.RoleRetailer,
			RetailerOrgID: orgID,
		}))
		rr := httptest.NewRecorder()
		svc.HandleFileOrderClaim(rr, req)
		return rr
	}

	rr1 := postAs("ret-1")
	if rr1.Code != http.StatusCreated {
		t.Fatalf("ret-1 status=%d body=%s", rr1.Code, rr1.Body.String())
	}

	// Same Subject + same Idempotency-Key + same body, but acting under ret-2:
	// must NOT replay ret-1's created claim and must hit authz -> 403 Forbidden.
	rr2 := postAs("ret-2")
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("ret-2 status=%d want 403 Forbidden (cross-tenant replay leak!) body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestHandleListSupplierClaims_CursorHTTP(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	for i := 1; i <= 3; i++ {
		if err := repo.CreateClaim(context.Background(), Claim{
			ClaimID:    "clm_" + string(rune('0'+i)),
			OrderID:    "ord-1",
			SupplierID: "sup-1",
			Status:     StatusOpen,
			CreatedAt:  now.Add(time.Duration(i) * time.Minute),
		}, nil); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(Config{
		Repo: repo,
		Now:  func() time.Time { return now },
		Log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	getPage := func(q string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/supplier/claims"+q, nil)
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1",
		}))
		rr := httptest.NewRecorder()
		svc.HandleListSupplierClaims(rr, req)
		return rr
	}

	rr1 := getPage("?limit=2")
	if rr1.Code != http.StatusOK {
		t.Fatalf("page1 status=%d body=%s", rr1.Code, rr1.Body.String())
	}
	var p1 struct {
		Claims     []Claim `json:"claims"`
		NextCursor string  `json:"next_cursor"`
	}
	if err := json.Unmarshal(rr1.Body.Bytes(), &p1); err != nil {
		t.Fatal(err)
	}
	if len(p1.Claims) != 2 || p1.NextCursor == "" {
		t.Fatalf("want 2 claims and next_cursor, got %+v", p1)
	}

	rr2 := getPage("?limit=2&cursor=" + p1.NextCursor)
	if rr2.Code != http.StatusOK {
		t.Fatalf("page2 status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	var p2 struct {
		Claims     []Claim `json:"claims"`
		NextCursor string  `json:"next_cursor"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &p2); err != nil {
		t.Fatal(err)
	}
	if len(p2.Claims) != 1 || p2.NextCursor != "" || p2.Claims[0].ClaimID != "clm_1" {
		t.Fatalf("unexpected page2: %+v", p2)
	}

	rrBad := getPage("?cursor=not-valid-cursor")
	if rrBad.Code != http.StatusBadRequest {
		t.Fatalf("bad cursor status=%d want 400 body=%s", rrBad.Code, rrBad.Body.String())
	}
}
