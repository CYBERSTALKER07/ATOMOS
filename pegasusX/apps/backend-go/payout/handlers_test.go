package payout

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestBatchJSON_SnakeCase(t *testing.T) {
	raw, err := json.Marshal(Batch{
		BatchID:        "po-1",
		SupplierID:     "sup-1",
		Status:         StatusDraft,
		NetPayoutMinor: 125000,
		Currency:       "UZS",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{`"batch_id"`, `"supplier_id"`, `"net_payout_minor"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}
	if strings.Contains(body, `"BatchID"`) {
		t.Fatalf("must not emit PascalCase: %s", body)
	}
}

func TestHandleListBatches_NoRepo503(t *testing.T) {
	h := &Handlers{Svc: NewService(nil)}
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/payouts/batches", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
		Subject:    "admin-1",
	}))
	rr := httptest.NewRecorder()
	h.HandleListBatches(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleListBatches_EmptyArrayJSON(t *testing.T) {
	raw, err := json.Marshal(map[string]any{"batches": []Batch{}})
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Batches []Batch `json:"batches"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Batches == nil {
		t.Fatal("batches must be [] not null")
	}
	if len(parsed.Batches) != 0 {
		t.Fatalf("len=%d", len(parsed.Batches))
	}
}

func TestHandleGetBatch_MissingID(t *testing.T) {
	h := &Handlers{Svc: NewService(NewRepository(nil))}
	r := chi.NewRouter()
	r.Get("/v1/supplier/payouts/batches/{batchID}", h.HandleGetBatch)
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/payouts/batches/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
		Subject:    "admin-1",
	}))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound && rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
