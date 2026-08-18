package payload

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHandlePayloaderLogin_IDTokenWithoutVerifierUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: NewInMemoryRepository(), JWTSecret: "t5-secret", SeedSupplierID: "seed-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login",
		strings.NewReader(`{"id_token":"tok"}`))
	rr := httptest.NewRecorder()
	svc.HandlePayloaderLogin(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandlePayloaderLogin_NoDefault998(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("PAYLOAD_DEMO_PHONE", "")
	svc := NewService(ServiceConfig{
		Repo: NewInMemoryRepository(), JWTSecret: "t5-secret", SeedSupplierID: "seed-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login",
		strings.NewReader(`{"phone":"+998901110022","pin":"33333333"}`))
	rr := httptest.NewRecorder()
	svc.HandlePayloaderLogin(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("+998 demo must not login outside ssmr: %s", rr.Body.String())
	}
}

func TestHandlePayloaderLogin_StaffRow(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	hash, err := bcrypt.GenerateFromPassword([]byte("3333"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(ServiceConfig{
		Repo: NewInMemoryRepository(), JWTSecret: "t5-secret", SeedSupplierID: "seed-1",
	})
	svc.SetStaffLookup(func(_ context.Context, phone string) (PayloadStaffRecord, bool, error) {
		if phone != "+15550002" {
			return PayloadStaffRecord{}, false, nil
		}
		return PayloadStaffRecord{
			UserID: "pay-row", Name: "Row Loader", Phone: phone, PasswordHash: string(hash),
			SupplierID: "sup-minted", WarehouseID: "wh-1",
		}, true, nil
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login",
		strings.NewReader(`{"phone":"+15550002","pin":"3333"}`))
	rr := httptest.NewRecorder()
	svc.HandlePayloaderLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "41.311") {
		t.Fatal("must not emit Tashkent demo coords")
	}
}

func TestPayloaderLoginSource_NoHardcoded998(t *testing.T) {
	src, err := os.ReadFile("auth_login.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "+998") {
		t.Fatal("payloader login must not hardcode +998 demo phones")
	}
}
