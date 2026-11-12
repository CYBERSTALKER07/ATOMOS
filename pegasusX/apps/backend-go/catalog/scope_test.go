package catalog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestListCategoriesRejectsForeignSupplier(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodGet, "/v1/catalog/categories?supplier_id=other", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleListCategories(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d body %s", rr.Code, rr.Body.String())
	}
}
