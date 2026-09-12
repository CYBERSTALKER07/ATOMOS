package factory

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/staffinvite"
)

func TestHandleFactoryLogin_IDTokenWithoutVerifierUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/factory/login",
		strings.NewReader(`{"id_token":"tok"}`))
	rr := httptest.NewRecorder()
	svc.HandleFactoryLogin(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleFactoryRegister_IDTokenWithoutVerifierUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/factory/register",
		strings.NewReader(`{"name":"Bay","phone":"+15551212","id_token":"tok","assigned_factory_id":"fac-1"}`))
	rr := httptest.NewRecorder()
	svc.HandleFactoryRegister(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleFactoryRegister_RequiresInvite(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/factory/register",
		strings.NewReader(`{"name":"Bay","phone":"+15551212","password":"pass12","assigned_factory_id":"fac-1"}`))
	rr := httptest.NewRecorder()
	svc.HandleFactoryRegister(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), staffinvite.ErrInviteRequired.Error()) {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleFactoryRegister_RejectsSeedInviteOutsideSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	tok, _, err := staffinvite.Mint("t5-secret", staffinvite.RoleFactory, "seed-1", "fac-1", 0, svc.now())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/factory/register",
		strings.NewReader(`{"name":"Bay","phone":"+15551212","password":"pass12","invite_token":"`+tok+`"}`))
	rr := httptest.NewRecorder()
	svc.HandleFactoryRegister(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleFactoryRegister_PasswordRequired(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	tok, _, err := staffinvite.Mint("t5-secret", staffinvite.RoleFactory, "sup-minted", "fac-1", 0, svc.now())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/factory/register",
		strings.NewReader(`{"name":"Bay","phone":"+15551212","invite_token":"`+tok+`"}`))
	rr := httptest.NewRecorder()
	svc.HandleFactoryRegister(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), staffinvite.ErrPasswordRequired.Error()) {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestFactoryRegisterSource_NoPreferTenant(t *testing.T) {
	src, err := os.ReadFile("auth_register.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "resolveSupplierScope(") {
		t.Fatal("public factory register must not PreferTenant(seed)")
	}
}

func TestFactoryDemoLogin_NoDefaultPhoneOutsideSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("FACTORY_DEMO_PHONE", "")
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	if svc.verifyFactoryDemoPhone("+998901000099") {
		t.Fatal("+998 must not be a default factory demo phone")
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/factory/login",
		strings.NewReader(`{"phone":"+998901000099","pin":"1234"}`))
	rr := httptest.NewRecorder()
	svc.HandleFactoryLogin(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("demo login must not succeed outside ssmr: %s", rr.Body.String())
	}
}
