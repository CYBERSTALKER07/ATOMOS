package supplier

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

func TestHandlePricingRulesPatchIdempotencyReplay(t *testing.T) {
	idem := idempotency.NewInMemoryStore()
	repo := &pricingTestRepo{profileFound: true, profile: Profile{SupplierID: "sup-1", Currency: "UZS"}}
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Repo:       repo,
		Idem:       idem,
		SupplierID: "sup-1",
		Country:    "UZ",
		Currency:   "UZS",
		Now:        func() time.Time { return now },
	})

	body := `{"base_markup_bps":1200,"retailer_discount_bps":200,"min_margin_bps":100,"currency":"UZS"}`
	cached := []byte(`{"supplier_id":"sup-1","base_markup_bps":1200,"retailer_discount_bps":200,"min_margin_bps":100,"currency":"UZS","rule_version":1,"updated_at":"2026-06-29T12:00:00Z"}`)
	if err := idem.Save(context.Background(), "supplier-pricing-patch:sup-1", idempotency.Record{
		BodyHash:   sha256Hex([]byte(body)),
		StatusCode: http.StatusOK,
		Response:   cached,
		StoredAt:   now,
	}, 24*time.Hour); err != nil {
		t.Fatalf("seed idem: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/pricing/rules", strings.NewReader(body))
	req.Header.Set("Idempotency-Key", "supplier-pricing-patch:sup-1")
	rr := httptest.NewRecorder()
	svc.HandlePricingRules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if rr.Body.String() != string(cached) {
		t.Fatalf("body=%q want=%q", rr.Body.String(), string(cached))
	}
}

func TestHandleUpdateProfileIdempotencyConflict(t *testing.T) {
	idem := idempotency.NewInMemoryStore()
	repo := &pricingTestRepo{
		profileFound: true,
		profile:      Profile{SupplierID: "sup-1", LegalName: "Acme", Country: "UZ", Currency: "UZS"},
	}
	svc := NewService(ServiceConfig{
		Repo:       repo,
		Idem:       idem,
		SupplierID: "sup-1",
		Country:    "UZ",
		Currency:   "UZS",
	})

	if err := idem.Save(context.Background(), "supplier-profile:sup-1", idempotency.Record{
		BodyHash:   sha256Hex([]byte(`{"legal_name":"Acme"}`)),
		StatusCode: http.StatusOK,
		Response:   []byte(`{"supplier_id":"sup-1"}`),
		StoredAt:   time.Now().UTC(),
	}, 24*time.Hour); err != nil {
		t.Fatalf("seed idem: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/v1/supplier/profile", strings.NewReader(`{"legal_name":"Other"}`))
	req.Header.Set("Idempotency-Key", "supplier-profile:sup-1")
	rr := httptest.NewRecorder()
	svc.HandleProfile(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
	}
}

func TestHandleInventoryPatchIdempotencyReplay(t *testing.T) {
	idem := idempotency.NewInMemoryStore()
	svc := NewService(ServiceConfig{
		Repo:       &pricingTestRepo{},
		Idem:       idem,
		SupplierID: "sup-1",
		InventoryService: &inventoryPatchTestServicer{},
	})

	body := `{"sku_id":"sku-1","quantity_delta":5,"quantity":10}`
	cached := []byte(`{"status":"ok"}`)
	if err := idem.Save(context.Background(), "supplier-inventory:sku-1", idempotency.Record{
		BodyHash:   sha256Hex([]byte(body)),
		StatusCode: http.StatusOK,
		Response:   cached,
		StoredAt:   time.Now().UTC(),
	}, 24*time.Hour); err != nil {
		t.Fatalf("seed idem: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/inventory", strings.NewReader(body))
	req.Header.Set("Idempotency-Key", "supplier-inventory:sku-1")
	rr := httptest.NewRecorder()
	svc.HandleInventory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status=%q want ok", payload["status"])
	}
}

type inventoryPatchTestServicer struct{}

func (inventoryPatchTestServicer) ListBySupplier(context.Context, string) ([]InventoryLevelView, error) {
	return nil, nil
}

func (inventoryPatchTestServicer) AdjustStock(context.Context, string, int64, int64) error {
	return nil
}

func (inventoryPatchTestServicer) FindByWarehouseProduct(context.Context, string, string) (string, bool, error) {
	return "", false, nil
}

func (inventoryPatchTestServicer) UpsertLevel(context.Context, InventoryLevelUpsert) error {
	return nil
}
