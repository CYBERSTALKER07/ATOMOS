package driver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/staffinvite"
	"golang.org/x/crypto/bcrypt"
)

func TestHandleDriverLogin_IDTokenWithoutVerifierUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/driver/login",
		strings.NewReader(`{"id_token":"tok"}`))
	rr := httptest.NewRecorder()
	svc.HandleDriverLogin(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDriverLogin_NoDefault998(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("DRIVER_DEMO_PHONE", "")
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/driver/login",
		strings.NewReader(`{"phone":"+998901000066","pin":"1234"}`))
	rr := httptest.NewRecorder()
	svc.HandleDriverLogin(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("+998 demo must not login outside ssmr: %s", rr.Body.String())
	}
}

func TestHandleDriverLogin_TableRow(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	hash, err := bcrypt.GenerateFromPassword([]byte("4321"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	svc.SetLoginLookup(func(_ context.Context, phone string) (DriverLoginRecord, bool, error) {
		if phone != "+15550001" {
			return DriverLoginRecord{}, false, nil
		}
		return DriverLoginRecord{
			DriverID: "drv-row", Name: "Row Driver", Phone: phone, PinHash: string(hash),
			SupplierID: "sup-minted", HomeNodeType: "FACTORY", HomeNodeID: "fac-1",
		}, true, nil
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/driver/login",
		strings.NewReader(`{"phone":"+15550001","pin":"4321"}`))
	rr := httptest.NewRecorder()
	svc.HandleDriverLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"drv-row"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "41.311") {
		t.Fatal("must not emit Tashkent demo coords")
	}
}

func TestHandleDriverLogin_SSMRExplicitEnv(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	t.Setenv("DRIVER_DEMO_PHONE", "+15550999")
	t.Setenv("DRIVER_DEMO_PIN", "9999")
	svc := NewService(ServiceConfig{JWTSecret: "t5-secret", SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/driver/login",
		strings.NewReader(`{"phone":"+15550999","pin":"9999"}`))
	rr := httptest.NewRecorder()
	svc.HandleDriverLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDriverLoginSource_NoHardcoded998(t *testing.T) {
	src, err := os.ReadFile("auth_login.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "+998") {
		t.Fatal("driver login must not hardcode +998 demo phones")
	}
	_ = staffinvite.DemoScaffoldAllowed()
}
