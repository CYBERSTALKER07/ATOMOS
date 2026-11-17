package fxrates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSeedBootstrapRatesIdentity(t *testing.T) {
	repo := NewMemoryRepository()
	if err := SeedBootstrapRates(context.Background(), repo, SeedOptions{OperatingCurrency: "uzs"}); err != nil {
		t.Fatal(err)
	}
	rate, ok, err := repo.GetAsOf(context.Background(), "UZS", "UZS", time.Now())
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if rate.RateScaled != DefaultScale {
		t.Fatalf("scaled=%d", rate.RateScaled)
	}
}

func TestHandlersListAndUpsert(t *testing.T) {
	repo := NewMemoryRepository()
	_ = SeedBootstrapRates(context.Background(), repo, SeedOptions{
		OperatingCurrency: "UZS",
		USDToUZSScaled:    12_500 * DefaultScale,
	})
	h := NewHandlers(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/fx-rates", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d", rec.Code)
	}
	var list struct {
		Rates []rateDTO `json:"rates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Rates) < 1 {
		t.Fatal("expected seeded rates")
	}

	body := `{"base_currency":"EUR","quote_currency":"UZS","rate_scaled":13500000000000,"source":"MANUAL"}`
	put := httptest.NewRequest(http.MethodPut, "/v1/admin/fx-rates", strings.NewReader(body))
	putRec := httptest.NewRecorder()
	h.HandleUpsert(putRec, put)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", putRec.Code, putRec.Body.String())
	}
	_, ok, err := repo.GetAsOf(context.Background(), "EUR", "UZS", time.Now().Add(time.Minute))
	if err != nil || !ok {
		t.Fatalf("eur rate missing ok=%v err=%v", ok, err)
	}
}
