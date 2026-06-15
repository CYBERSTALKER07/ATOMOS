package retailer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteMobileAuthResponse_IncludesConfiguredClaim(t *testing.T) {
	t.Parallel()

	svc := &Service{
		supplierID: "sup-1",
		jwtSecret:  "test-secret",
		jwtIssuer:  "pegasusx-test",
	}

	rec := httptest.NewRecorder()
	svc.writeMobileAuthResponse(rec, Retailer{
		RetailerID: "ret-1",
		SupplierID: "sup-1",
		Name:       "Demo Store",
		Lat:        41.3,
		Lng:        69.2,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["is_configured"] != true {
		t.Fatalf("is_configured=%v want true", payload["is_configured"])
	}
	if token, _ := payload["token"].(string); token == "" {
		t.Fatal("expected token in login response")
	}
}

func TestRetailerProfileConfigured_RequiresNameAndCoordinates(t *testing.T) {
	t.Parallel()

	if retailerProfileConfigured(Retailer{Name: "Store"}) {
		t.Fatal("expected incomplete profile without coordinates")
	}
	if !retailerProfileConfigured(Retailer{Name: "Store", Lat: 41.0}) {
		t.Fatal("expected configured profile with name and lat")
	}
}
