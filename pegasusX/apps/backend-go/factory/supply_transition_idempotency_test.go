package factory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestHandleSupplyRequestTransition_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"action":"ACKNOWLEDGE"}`)
	cached := map[string]string{
		"request_id": "sr-1",
		"state":      "ACKNOWLEDGED",
	}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "factory-supply-transition:sr-1:ACKNOWLEDGE"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256HexBytes(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := &Service{idem: store}
	req := httptest.NewRequest(http.MethodPatch, "/v1/factory/supply-requests/sr-1", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "sr-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleSupplyRequestTransition(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var got map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["request_id"] != "sr-1" || got["state"] != "ACKNOWLEDGED" {
		t.Fatalf("replay response = %+v want %+v", got, cached)
	}
}
