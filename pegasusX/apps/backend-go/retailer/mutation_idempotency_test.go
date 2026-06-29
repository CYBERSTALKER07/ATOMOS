package retailer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func TestHandleUpdateProfile_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"name":"Corner Store","receiving_window_open":"09:00","receiving_window_close":"17:00"}`)
	cached := map[string]any{"retailer_id": "ret-1", "name": "Corner Store"}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "retailer-profile-update:ret-1:test"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := NewService(ServiceConfig{
		Repo: &testRetailerRepo{
			found: true,
			retailer: Retailer{
				RetailerID: "ret-1",
				Name:       "Old Name",
			},
		},
		SupplierID: "sup-1",
		Idem:       store,
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/retailer/profile", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != string(cachedBytes) {
		t.Fatalf("replay body = %s want %s", rr.Body.String(), string(cachedBytes))
	}
}

func TestHandleEditPreorder_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"order_id":"ord-1","requested_delivery_date":"2026-01-03T12:00:00Z","line_items":[{"sku":"sku-1","quantity":1,"unit_price_minor":500}]}`)
	cached := order.RetailerOrderLifecycleResponse{OrderID: "ord-1", ConfirmationStatus: order.ConfirmationStatusPending}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "retailer-edit-preorder:ord-1"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := NewService(ServiceConfig{
		Repo:       &testRetailerRepo{},
		SupplierID: "sup-1",
		Orders:     &testOrderLifecycle{},
		Idem:       store,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/edit-preorder", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleEditPreorder(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var got order.RetailerOrderLifecycleResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.OrderID != "ord-1" {
		t.Fatalf("order_id = %s want ord-1", got.OrderID)
	}
}

func TestHandleSupplierAdd_IdempotencyReplay(t *testing.T) {
	body := []byte{}
	cached := []supplierPreference{{
		SupplierID: "sup-1",
		Name:       "pegasusX Supplier",
		IsFavorite: true,
	}}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "retailer-supplier-add:sup-2"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := NewService(ServiceConfig{
		Repo:       &testRetailerRepo{},
		SupplierID: "sup-1",
		Idem:       store,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/suppliers/sup-2/add", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("supplierID", "sup-2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleSupplierAdd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != string(cachedBytes) {
		t.Fatalf("replay body = %s want %s", rr.Body.String(), string(cachedBytes))
	}
}

func TestHandleRetailerSetup_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"name":"Setup Store","lat":41.3,"lng":69.2}`)
	cached := map[string]any{"is_configured": true, "user": map[string]any{"id": "ret-1"}}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "retailer-setup:ret-1"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := NewService(ServiceConfig{
		Repo: &testRetailerRepo{
			found: true,
			retailer: Retailer{
				RetailerID: "ret-1",
				Name:       "Existing",
			},
		},
		SupplierID: "sup-1",
		JWTSecret:  "test-secret-key-for-retailer-setup",
		Idem:       store,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/setup", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleRetailerSetup(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != string(cachedBytes) {
		t.Fatalf("replay body = %s want %s", rr.Body.String(), string(cachedBytes))
	}
}
