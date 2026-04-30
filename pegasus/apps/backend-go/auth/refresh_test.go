package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestHandleTokenRefresh_PreservesExtendedClaims(t *testing.T) {
	originalToken, err := MintIdentityToken(&LabClaims{
		UserID:        "user-1",
		SupplierID:    "supplier-1",
		Role:          "PAYLOADER",
		WarehouseID:   "warehouse-1",
		WarehouseRole: "PAYLOADER",
	})
	if err != nil {
		t.Fatalf("MintIdentityToken error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+originalToken)
	resp := httptest.NewRecorder()

	HandleTokenRefresh().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	claims := &LabClaims{}
	parsed, err := jwt.ParseWithClaims(body["token"], claims, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("refreshed token invalid: %v", err)
	}
	if claims.SupplierID != "supplier-1" {
		t.Fatalf("supplier_id = %q, want %q", claims.SupplierID, "supplier-1")
	}
	if claims.WarehouseID != "warehouse-1" {
		t.Fatalf("warehouse_id = %q, want %q", claims.WarehouseID, "warehouse-1")
	}
	if claims.WarehouseRole != "PAYLOADER" {
		t.Fatalf("warehouse_role = %q, want %q", claims.WarehouseRole, "PAYLOADER")
	}
}
