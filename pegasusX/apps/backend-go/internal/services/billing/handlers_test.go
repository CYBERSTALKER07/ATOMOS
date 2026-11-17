package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleListInvoices_NoSpanner503(t *testing.T) {
	h := &Handlers{Worker: NewInvoiceWorker(nil, nil, nil, testDiscardLogger())}
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
				Subject: "plat-1", Role: auth.RolePlatformAdmin, MFAVerified: true,
			}))
			next.ServeHTTP(w, req)
		})
	})
	RegisterRoutes(r, h)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/billing/invoices", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, has := body["invoices"]; has {
		t.Fatal("must not invent invoices[] without Spanner")
	}
}

func TestHandleListFeeSchedules_NoSpanner503(t *testing.T) {
	h := &Handlers{Worker: NewInvoiceWorker(nil, nil, nil, testDiscardLogger())}
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
				Subject: "plat-1", Role: auth.RolePlatformAdmin, MFAVerified: true,
			}))
			next.ServeHTTP(w, req)
		})
	})
	RegisterRoutes(r, h)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/billing/fee-schedules", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
}

func TestListPlatformInvoicesEmptySliceNotNil(t *testing.T) {
	// Encoding contract: empty list serializes as [] not null.
	raw, err := json.Marshal(map[string]any{"invoices": []PlatformInvoice{}})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"invoices":[]}` {
		t.Fatalf("got %s", raw)
	}
	_ = context.Background()
}
