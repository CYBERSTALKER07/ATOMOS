package warehouse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/staffinvite"
)

func TestHandleWarehouseLogin_IDTokenWithoutVerifierUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/warehouse/login",
		strings.NewReader(`{"id_token":"tok"}`))
	rr := httptest.NewRecorder()
	svc.HandleWarehouseLogin(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleWarehouseRegister_IDTokenWithoutVerifierUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/warehouse/register",
		strings.NewReader(`{"name":"Ops","phone":"+15551212","id_token":"tok","assigned_warehouse_id":"wh-1"}`))
	rr := httptest.NewRecorder()
	svc.HandleWarehouseRegister(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleWarehouseRegister_RequiresInvite(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/warehouse/register",
		strings.NewReader(`{"name":"Ops","phone":"+15551212","password":"pass12","assigned_warehouse_id":"wh-1"}`))
	rr := httptest.NewRecorder()
	svc.HandleWarehouseRegister(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), staffinvite.ErrInviteRequired.Error()) {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestWarehouseRegisterSource_NoPreferTenant(t *testing.T) {
	src, err := os.ReadFile("auth_register.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "resolveSupplierScope(") {
		t.Fatal("public warehouse register must not PreferTenant(seed)")
	}
}

func TestVerifyWarehouseStaffSecret_RejectsPlaintext(t *testing.T) {
	if verifyWarehouseStaffSecret("plaintext", "plaintext") {
		t.Fatal("warehouse login must be bcrypt only")
	}
	hash, err := staffinvite.HashPassword("pass12")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyWarehouseStaffSecret(hash, "pass12") {
		t.Fatal("bcrypt must verify")
	}
}

func TestHandleOpsStaff_CreateHashesPIN(t *testing.T) {
	svc := NewService(ServiceConfig{SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/staff",
		strings.NewReader(`{"name":"Picker","phone":"+15550001","role":"WAREHOUSE_STAFF"}`))
	rr := httptest.NewRecorder()
	svc.HandleOpsStaff(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	pin, _ := body["pin"].(string)
	if pin == "" || pin == "5678" || pin == "password_hash" {
		t.Fatalf("expected one-time generated pin, got %q", pin)
	}
}
