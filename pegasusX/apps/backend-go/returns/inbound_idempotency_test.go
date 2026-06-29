package returns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestHandleInboundScan_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"barcode":"5901234123457","qty":1,"session_id":"sess-1"}`)
	cached := map[string]any{
		"matched":   true,
		"return_id": "ret_cached",
		"variance":  false,
		"message":   "scanned 1 of 1",
	}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "payload-inbound-scan-5901234123457-sess-1"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256HexBytes(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := &Service{idem: store}
	req := httptest.NewRequest(http.MethodPost, "/v1/returns/inbound/scan", strings.NewReader(string(body)))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleInboundScan(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != string(cachedBytes) {
		t.Fatalf("replay body = %s want %s", rr.Body.String(), string(cachedBytes))
	}
}

func TestHandleInboundConfirm_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"session_id":"sess-1","lines":[{"return_id":"ret-1","disposition":"RESTOCK","qty":1}]}`)
	cached := map[string]any{
		"status":     "confirmed",
		"return_ids": []string{"ret-1"},
	}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "payload-inbound-confirm-RESTOCK-ret-1"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256HexBytes(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := &Service{idem: store}
	req := httptest.NewRequest(http.MethodPost, "/v1/returns/inbound/confirm", strings.NewReader(string(body)))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleInboundConfirm(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != string(cachedBytes) {
		t.Fatalf("replay body = %s want %s", rr.Body.String(), string(cachedBytes))
	}
}
