package claims

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
