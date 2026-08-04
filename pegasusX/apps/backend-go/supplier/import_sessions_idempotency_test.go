package supplier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestImportApproveIdempotencyReplay(t *testing.T) {
	idem := idempotency.NewInMemoryStore()
	svc := &Service{
		idem: idem,
		now:  func() time.Time { return time.Unix(0, 0).UTC() },
	}
	body := []byte(`{}`)
	cached := []byte(`{"session_id":"sess-1","status":"APPROVED","next_phase":"apply"}`)
	if err := idem.Save(context.Background(), "sup-1:supplier-import-approve:sess-1", idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: http.StatusAccepted,
		Response:   cached,
		StoredAt:   svc.now(),
	}, 24*3600*1e9); err != nil {
		t.Fatalf("seed idem: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/imports/sess-1/approve", strings.NewReader(string(body)))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{SupplierID: "sup-1", Role: auth.RoleAdmin}))
	req.Header.Set("Idempotency-Key", "supplier-import-approve:sess-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "sess-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handlePostImportApprove(&ImportRepository{}, svc)(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d want %d body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if rec.Body.String() != string(cached) {
		t.Fatalf("body = %q want %q", rec.Body.String(), string(cached))
	}
}

func TestImportApproveIdempotencyConflict(t *testing.T) {
	idem := idempotency.NewInMemoryStore()
	svc := &Service{
		idem: idem,
		now:  func() time.Time { return time.Unix(0, 0).UTC() },
	}
	if err := idem.Save(context.Background(), "sup-1:supplier-import-approve:sess-1", idempotency.Record{
		BodyHash:   sha256Hex([]byte(`{}`)),
		StatusCode: http.StatusAccepted,
		Response:   []byte(`{"ok":true}`),
		StoredAt:   svc.now(),
	}, 24*3600*1e9); err != nil {
		t.Fatalf("seed idem: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/imports/sess-1/approve", strings.NewReader(`{"probe":true}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{SupplierID: "sup-1", Role: auth.RoleAdmin}))
	req.Header.Set("Idempotency-Key", "supplier-import-approve:sess-1")
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "sess-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handlePostImportApprove(&ImportRepository{}, svc)(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d want %d", rec.Code, http.StatusConflict)
	}
}
