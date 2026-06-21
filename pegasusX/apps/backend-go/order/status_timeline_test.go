package order

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestAssertTimelineAccess(t *testing.T) {
	o := Order{OrderID: "o1", RetailerID: "r1", SupplierID: "s1", WarehouseID: "w1"}
	cases := []struct {
		name   string
		claims auth.Claims
		ok     bool
	}{
		{"retailer own", auth.Claims{Role: auth.RoleRetailer, Subject: "r1"}, true},
		{"retailer other", auth.Claims{Role: auth.RoleRetailer, Subject: "r2"}, false},
		{"warehouse", auth.Claims{Role: auth.RoleWarehouse, HomeNodeID: "w1"}, true},
		{"supplier admin", auth.Claims{Role: auth.RoleAdmin, SupplierID: "s1"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := assertTimelineAccess(tc.claims, o)
			if tc.ok && err != nil {
				t.Fatalf("want access, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("want forbidden")
			}
		})
	}
}
