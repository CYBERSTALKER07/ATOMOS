package fxrates

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func fxAdminRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("fx-test-secret"))
	RegisterAdminRoutes(r, NewHandlers(NewMemoryRepository()))
	return r
}

func issueFXToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject: "fx-user", Role: role, SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "fx-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestAdminFXPutRequiresPlatformAdmin(t *testing.T) {
	r := fxAdminRouter()
	body := `{"base_currency":"USD","quote_currency":"UZS","rate_scaled":1275000000000}`

	anon := httptest.NewRequest(http.MethodPut, "/v1/admin/fx-rates", strings.NewReader(body))
	anonRec := httptest.NewRecorder()
	r.ServeHTTP(anonRec, anon)
	if anonRec.Code != http.StatusUnauthorized {
		t.Fatalf("anon PUT status=%d want 401", anonRec.Code)
	}

	sup := httptest.NewRequest(http.MethodPut, "/v1/admin/fx-rates", strings.NewReader(body))
	sup.Header.Set("Authorization", "Bearer "+issueFXToken(t, auth.RoleAdmin))
	supRec := httptest.NewRecorder()
	r.ServeHTTP(supRec, sup)
	if supRec.Code != http.StatusForbidden {
		t.Fatalf("supplier ADMIN PUT status=%d want 403", supRec.Code)
	}

	plat := httptest.NewRequest(http.MethodPut, "/v1/admin/fx-rates", strings.NewReader(body))
	plat.Header.Set("Authorization", "Bearer "+issueFXToken(t, auth.RolePlatformAdmin))
	platRec := httptest.NewRecorder()
	r.ServeHTTP(platRec, plat)
	if platRec.Code != http.StatusOK {
		t.Fatalf("platform PUT status=%d body=%s want 200", platRec.Code, platRec.Body.String())
	}
}

func TestAdminFXGetAllowsSupplierAdmin(t *testing.T) {
	r := fxAdminRouter()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/fx-rates", nil)
	req.Header.Set("Authorization", "Bearer "+issueFXToken(t, auth.RoleAdmin))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status=%d want 200", rec.Code)
	}
}
