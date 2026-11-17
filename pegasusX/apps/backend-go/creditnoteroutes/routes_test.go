package creditnoteroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
)

func creditNoteRouter(h *creditnote.Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("creditnote-test-secret"))
	RegisterRoutes(r, Deps{Handlers: h})
	return r
}

func issueCreditNoteToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "creditnote-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "creditnote-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestCreditNoteRoutes_NilHandlers(t *testing.T) {
	r := creditNoteRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/credit-notes", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestCreditNoteRoutes_Unauthenticated(t *testing.T) {
	r := creditNoteRouter(&creditnote.Handlers{})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/supplier/credit-notes"},
		{http.MethodPost, "/v1/supplier/credit-notes"},
		{http.MethodGet, "/v1/warehouse/reverse-logistics"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without auth returned %d, want 401", ep.method, ep.path, rr.Code)
		}
	}
}

func TestCreditNoteRoutes_RoleGuard(t *testing.T) {
	r := creditNoteRouter(&creditnote.Handlers{})

	driverToken := issueCreditNoteToken(t, auth.RoleDriver)

	// Driver should not be allowed to access supplier credit notes
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/credit-notes", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("driver accessing supplier credit-notes returned %d, want 403 Forbidden", rr.Code)
	}
}
