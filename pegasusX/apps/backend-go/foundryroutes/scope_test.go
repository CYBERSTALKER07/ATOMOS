package foundryroutes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundryroutes"
)

func TestFoundryRejectsAnonymousAndForeignSupplier(t *testing.T) {
	r := chi.NewRouter()
	foundryroutes.RegisterRoutes(r, foundryroutes.Deps{Engine: foundry.NewEngine()})

	anon := httptest.NewRequest(http.MethodPost, "/v1/foundry/guards/aml-check", bytes.NewReader([]byte(`{"supplier_id":"sup_1","legal_entity_tin":"TIN","calendar_date":"2026-10-29","cash_minor":1}`)))
	anonRR := httptest.NewRecorder()
	r.ServeHTTP(anonRR, anon)
	if anonRR.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d", anonRR.Code)
	}

	body, _ := json.Marshal(map[string]any{
		"legal_entity_tin": "TIN-555111",
		"supplier_id":      "someone-else",
		"calendar_date":    "2026-10-29",
		"cash_minor":       int64(1),
	})
	spoof := httptest.NewRequest(http.MethodPost, "/v1/foundry/guards/aml-check", bytes.NewReader(body))
	spoof = spoof.WithContext(auth.WithClaims(spoof.Context(), auth.Claims{
		Subject:    "cashier-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup_1",
	}))
	spoofRR := httptest.NewRecorder()
	r.ServeHTTP(spoofRR, spoof)
	if spoofRR.Code != http.StatusForbidden {
		t.Fatalf("foreign supplier status = %d body %s", spoofRR.Code, spoofRR.Body.String())
	}
}
