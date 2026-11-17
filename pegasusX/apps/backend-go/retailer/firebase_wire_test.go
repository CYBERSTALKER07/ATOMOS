package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type stubFirebaseVerifier struct {
	phone string
	err   error
}

func (s stubFirebaseVerifier) VerifyIDToken(context.Context, string) (auth.Claims, error) {
	if s.err != nil {
		return auth.Claims{}, s.err
	}
	return auth.Claims{PhoneNumber: s.phone}, nil
}

type phoneRetailerRepo struct {
	testRetailerRepo
	byPhone Retailer
}

func (r *phoneRetailerRepo) FindByPhone(_ context.Context, phone string) (Retailer, bool, error) {
	if strings.TrimSpace(r.byPhone.Phone) == strings.TrimSpace(phone) {
		return r.byPhone, true, nil
	}
	return Retailer{}, false, nil
}

func TestNewService_WiresFirebaseVerifier(t *testing.T) {
	v := stubFirebaseVerifier{phone: "+998901112233"}
	s := NewService(ServiceConfig{FirebaseVerifier: v})
	if s.firebaseVerifier == nil {
		t.Fatal("FirebaseVerifier must copy onto the service")
	}
}

func TestHandleRetailerLogin_IDTokenWithoutVerifierUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{
		JWTSecret: "test-secret",
		Repo:      &testRetailerRepo{},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/login",
		strings.NewReader(`{"id_token":"tok"}`))
	rr := httptest.NewRecorder()
	svc.HandleRetailerLogin(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != auth.FirebaseLoginUnavailable {
		t.Fatalf("error=%q", payload["error"])
	}
}

func TestHandleRetailerLogin_IDTokenVerified(t *testing.T) {
	phone := "+998901112233"
	repo := &phoneRetailerRepo{byPhone: Retailer{
		RetailerID: "ret-fb", Phone: phone, Name: "OTP Shop", SupplierID: "sup-1",
		Lat: 41.3, Lng: 69.2,
	}}
	svc := NewService(ServiceConfig{
		JWTSecret:        "test-secret",
		JWTIssuer:        "pegasusx-test",
		SeedSupplierID:   "sup-1",
		Repo:             repo,
		FirebaseVerifier: stubFirebaseVerifier{phone: phone},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/login",
		strings.NewReader(`{"id_token":"good"}`))
	rr := httptest.NewRecorder()
	svc.HandleRetailerLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if token, _ := payload["token"].(string); token == "" {
		t.Fatal("expected pegasus JWT")
	}
}

func TestHandleRetailerLogin_InvalidIDToken(t *testing.T) {
	svc := NewService(ServiceConfig{
		JWTSecret:        "test-secret",
		Repo:             &testRetailerRepo{},
		FirebaseVerifier: stubFirebaseVerifier{err: auth.ErrFirebaseTokenInvalid},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/login",
		strings.NewReader(`{"id_token":"bad"}`))
	rr := httptest.NewRecorder()
	svc.HandleRetailerLogin(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
