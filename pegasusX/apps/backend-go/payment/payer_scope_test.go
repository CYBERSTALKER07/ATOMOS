package payment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPayerAccessAllowed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		claims  auth.Claims
		payerID string
		want    bool
	}{
		{"platform any", auth.Claims{Subject: "ops", Role: auth.RolePlatformAdmin}, "payer-x", true},
		{"retailer self", auth.Claims{Subject: "r1", Role: auth.RoleRetailer}, "r1", true},
		{"retailer other", auth.Claims{Subject: "r1", Role: auth.RoleRetailer}, "r2", false},
		{"admin self", auth.Claims{Subject: "a1", Role: auth.RoleAdmin}, "a1", true},
		{"admin other blocked", auth.Claims{Subject: "a1", Role: auth.RoleAdmin}, "a2", false},
		{"driver denied", auth.Claims{Subject: "d1", Role: auth.RoleDriver}, "d1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := payerAccessAllowed(tc.claims, tc.payerID); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func withPayerID(req *http.Request, payerID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("payerId", payerID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestHandleGetPayer_AdminCannotCrossPayer(t *testing.T) {
	t.Parallel()
	svc := &Service{repo: &paymentRepoStub{}}
	req := httptest.NewRequest(http.MethodGet, "/v1/payers/other-payer", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}))
	req = withPayerID(req, "other-payer")
	rr := httptest.NewRecorder()
	svc.HandleGetPayer(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", rr.Code, rr.Body.String())
	}
}

func TestHandleGetPayer_PlatformAdminMayCrossPayer(t *testing.T) {
	t.Parallel()
	svc := &Service{repo: &paymentRepoStub{}}
	req := httptest.NewRequest(http.MethodGet, "/v1/payers/other-payer", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ops", Role: auth.RolePlatformAdmin,
	}))
	req = withPayerID(req, "other-payer")
	rr := httptest.NewRecorder()
	svc.HandleGetPayer(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s want 200", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdatePayer_AdminCannotCrossPayer(t *testing.T) {
	t.Parallel()
	svc := &Service{repo: &paymentRepoStub{}}
	req := httptest.NewRequest(http.MethodPut, "/v1/payers/other-payer", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}))
	req = withPayerID(req, "other-payer")
	rr := httptest.NewRecorder()
	svc.HandleUpdatePayer(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", rr.Code, rr.Body.String())
	}
}
