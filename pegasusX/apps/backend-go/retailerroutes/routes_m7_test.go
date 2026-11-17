package retailerroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

func TestRetailerRoutes_M7AuthAndRoleGuards(t *testing.T) {
	r := retailerRouter(&retailer.Service{})
	driverToken := issueRetailerToken(t, auth.RoleDriver)
	retailerToken := issueRetailerToken(t, auth.RoleRetailer)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPatch, "/v1/retailer/tax-identity", `{"informal_shop":true}`},
		{http.MethodGet, "/v1/retailer/join-requests", ""},
		{http.MethodPost, "/v1/retailer/join-requests/jr-1/approve", `{}`},
		{http.MethodPost, "/v1/retailer/join-requests/jr-1/reject", `{}`},
		{http.MethodGet, "/v1/retailer/nasiya/opening-balances", ""},
		{http.MethodPost, "/v1/retailer/ar-invoices/inv-1/acknowledge", `{}`},
		{http.MethodPost, "/v1/retailer/ar-invoices/inv-1/dispute", `{"reason":"mismatch"}`},
	}

	for _, ep := range endpoints {
		reqUnauth := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
		rrUnauth := httptest.NewRecorder()
		r.ServeHTTP(rrUnauth, reqUnauth)
		if rrUnauth.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without auth=%d want=401", ep.method, ep.path, rrUnauth.Code)
		}

		reqDriver := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
		reqDriver.Header.Set("Authorization", "Bearer "+driverToken)
		rrDriver := httptest.NewRecorder()
		r.ServeHTTP(rrDriver, reqDriver)
		if rrDriver.Code != http.StatusForbidden {
			t.Fatalf("%s %s as driver=%d want=403", ep.method, ep.path, rrDriver.Code)
		}

		reqRet := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
		reqRet.Header.Set("Authorization", "Bearer "+retailerToken)
		rrRet := httptest.NewRecorder()
		r.ServeHTTP(rrRet, reqRet)
		if rrRet.Code == http.StatusUnauthorized || rrRet.Code == http.StatusForbidden || rrRet.Code == http.StatusNotFound {
			t.Fatalf("%s %s as retailer unexpectedly rejected/missing: status=%d", ep.method, ep.path, rrRet.Code)
		}
	}
}
