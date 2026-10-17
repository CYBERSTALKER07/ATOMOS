package warehouse

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestWarehouseIdempotency_CreateAndReplay(t *testing.T) {
	idem := idempotency.NewInMemoryStore()
	svc := NewService(ServiceConfig{
		Repo: NewInMemoryRepository(),
		Idem: idem,
	})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := auth.WithClaims(req.Context(), auth.Claims{
				Subject:    "test-admin",
				Role:       auth.RoleAdmin,
				SupplierID: "sup-idem-1",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)

	payload := map[string]any{
		"name":        "Idempotent Central Hub",
		"address":     "42 Tashkent Ring Rd",
		"latitude":    41.311,
		"longitude":   69.240,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// 1. Missing Idempotency Key -> 400 Bad Request
	reqMissing := httptest.NewRequest(http.MethodPost, "/v1/warehouses", bytes.NewReader(bodyBytes))
	reqMissing.Header.Set("Content-Type", "application/json")
	wMissing := httptest.NewRecorder()
	r.ServeHTTP(wMissing, reqMissing)
	if wMissing.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when missing idempotency key, got %d body=%s", wMissing.Code, wMissing.Body.String())
	}

	// 2. First call with Idempotency-Key -> 201 Created
	idemKey := "key-create-wh-001"
	req1 := httptest.NewRequest(http.MethodPost, "/v1/warehouses", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idemKey)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d body=%s", w1.Code, w1.Body.String())
	}
	var created1 map[string]any
	if err := json.Unmarshal(w1.Body.Bytes(), &created1); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	whID, ok := created1["warehouse_id"].(string)
	if !ok || whID == "" {
		t.Fatalf("expected warehouse_id in response, got %v", created1)
	}

	// 3. Replay with exact same Idempotency-Key -> Replays cached 201 Created and same warehouse_id
	req2 := httptest.NewRequest(http.MethodPost, "/v1/warehouses", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idemKey)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected replayed 201 Created, got %d body=%s", w2.Code, w2.Body.String())
	}
	var created2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &created2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if created2["warehouse_id"] != whID {
		t.Fatalf("expected replayed warehouse_id %s, got %s", whID, created2["warehouse_id"])
	}

	// 4. Same key with different payload -> 409 Conflict
	payloadDiff := map[string]any{
		"name":        "Totally Different Warehouse",
		"address":     "99 Different St",
	}
	bodyDiff, _ := json.Marshal(payloadDiff)
	req3 := httptest.NewRequest(http.MethodPost, "/v1/warehouses", bytes.NewReader(bodyDiff))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Idempotency-Key", idemKey)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on payload mismatch, got %d body=%s", w3.Code, w3.Body.String())
	}
}

func TestWarehouseIdempotency_UpdateAndReplay(t *testing.T) {
	idem := idempotency.NewInMemoryStore()
	repo := NewInMemoryRepository()
	svc := NewService(ServiceConfig{
		Repo: repo,
		Idem: idem,
	})

	_ = repo.CreateWarehouse(context.Background(), Warehouse{
		WarehouseID: "wh-up-1",
		SupplierID:  "sup-idem-1",
		Name:        "Original Name",
	}, nil)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := auth.WithClaims(req.Context(), auth.Claims{
				Subject:    "test-admin",
				Role:       auth.RoleAdmin,
				SupplierID: "sup-idem-1",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Put("/v1/warehouses/{warehouseId}", svc.HandleUpdateWarehouse)

	payload := map[string]any{
		"name":    "Updated Warehouse Name",
		"address": "New Location Address",
	}
	bodyBytes, _ := json.Marshal(payload)
	idemKey := "key-update-wh-001"

	// 1. Initial Update with key -> 200 OK
	req1 := httptest.NewRequest(http.MethodPut, "/v1/warehouses/wh-up-1", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idemKey)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", w1.Code, w1.Body.String())
	}

	// 2. Replay with exact same key -> 200 OK replayed
	req2 := httptest.NewRequest(http.MethodPut, "/v1/warehouses/wh-up-1", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idemKey)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected replayed 200 OK, got %d body=%s", w2.Code, w2.Body.String())
	}

	// 3. Same key with different payload -> 409 Conflict
	payloadDiff := map[string]any{"name": "Different Name Conflict"}
	bodyDiff, _ := json.Marshal(payloadDiff)
	req3 := httptest.NewRequest(http.MethodPut, "/v1/warehouses/wh-up-1", bytes.NewReader(bodyDiff))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Idempotency-Key", idemKey)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d body=%s", w3.Code, w3.Body.String())
	}
}
