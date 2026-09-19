package platformadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func testTransitionRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	h := &Handlers{Svc: svc}
	r.Post("/v1/platform-admin/tenants/{tenantType}/{tenantID}/transition", h.HandleTransitionTenant)
	return r
}

func TestHandleTransition_Missing404(t *testing.T) {
	r := testTransitionRouter(NewService(NewMemoryRepository()))
	body, _ := json.Marshal(transitionRequest{Status: StatusApproved, MarketCode: "UZ"})
	req := httptest.NewRequest(http.MethodPost, "/v1/platform-admin/tenants/SUPPLIER/missing/transition", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleTransition_Planned404(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	_ = svc.EnsurePending(context.Background(), TenantSupplier, "sup-1", "Co")
	r := testTransitionRouter(svc)
	body, _ := json.Marshal(transitionRequest{Status: StatusApproved, MarketCode: "EU"})
	req := httptest.NewRequest(http.MethodPost, "/v1/platform-admin/tenants/SUPPLIER/sup-1/transition", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleTransition_FirstApproveStaysPending(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	_ = svc.EnsurePending(context.Background(), TenantSupplier, "sup-1", "Co")
	r := testTransitionRouter(svc)
	body, _ := json.Marshal(transitionRequest{Status: StatusApproved, MarketCode: "UZ", HomeCell: "cell-uz"})
	req := httptest.NewRequest(http.MethodPost, "/v1/platform-admin/tenants/SUPPLIER/sup-1/transition", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var first Tenant
	if err := json.Unmarshal(rr.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusPending {
		t.Fatalf("first step must stay pending: %+v", first)
	}
}
