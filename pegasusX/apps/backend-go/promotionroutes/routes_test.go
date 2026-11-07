package promotionroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
)

func promotionRouter(svc *promotion.Service) http.Handler {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: "promotion-test-secret"})
	return r
}

func issuePromoCookie(t *testing.T, role auth.Role) *http.Cookie {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "promo-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "promotion-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: tok}
}

func TestPromotionRoutes_NilService(t *testing.T) {
	r := promotionRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/promotions", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestPromotionRoutes_Unauthenticated(t *testing.T) {
	svc := &promotion.Service{}
	r := promotionRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/supplier/promotions"},
		{http.MethodPost, "/v1/supplier/promotions"},
		{http.MethodPost, "/v1/retailer/promotions/enroll"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without cookie returned %d, want 401", ep.method, ep.path, rr.Code)
		}
	}
}

func TestPromotionRoutes_RoleGuard(t *testing.T) {
	svc := &promotion.Service{}
	r := promotionRouter(svc)

	cookie := issuePromoCookie(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/promotions", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer cookie accessing admin route returned %d, want 403 Forbidden", rr.Code)
	}
}
