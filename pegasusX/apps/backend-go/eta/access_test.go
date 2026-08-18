package eta

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestAllowRouteAccess(t *testing.T) {
	sc := RouteScope{RouteID: "r1", SupplierID: "sup-1", WarehouseID: "wh-1", DriverID: "drv-1", RetailerIDs: []string{"ret-1"}}
	cases := []struct {
		name   string
		c      auth.Claims
		tenant string
		ok     bool
	}{
		{"admin same", auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-1"}, "sup-1", true},
		{"admin other", auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-2"}, "sup-2", false},
		{"warehouse home", auth.Claims{Role: auth.RoleWarehouse, HomeNodeType: auth.HomeNodeWarehouse, HomeNodeID: "wh-1"}, "", true},
		{"warehouse other node", auth.Claims{Role: auth.RoleWarehouse, HomeNodeType: auth.HomeNodeWarehouse, HomeNodeID: "wh-x"}, "", false},
		{"driver subject", auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"}, "", true},
		{"driver other", auth.Claims{Role: auth.RoleDriver, Subject: "drv-x"}, "", false},
		{"retailer org", auth.Claims{Role: auth.RoleRetailer, RetailerOrgID: "ret-1"}, "", true},
		{"retailer other", auth.Claims{Role: auth.RoleRetailer, RetailerOrgID: "ret-x"}, "", false},
		{"empty scope", auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-1"}, "sup-1", false},
	}
	for _, tc := range cases {
		scope := sc
		if tc.name == "empty scope" {
			scope = RouteScope{}
		}
		if got := AllowRouteAccess(tc.c, tc.tenant, scope); got != tc.ok {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.ok)
		}
	}
}
