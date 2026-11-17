package promotion

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleCheckoutQuoteUsesRetailerOrgNotSubject(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/checkout/quote", strings.NewReader(`{"supplier_id":"other","lines":[{"product_id":"p1","qty":1}]}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "staff-user",
		Role:       auth.RoleRetailer,
		SupplierID: "sup-1",
		RetailerOrgID: "org-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleCheckoutQuote(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d body %s", rr.Code, rr.Body.String())
	}
}
