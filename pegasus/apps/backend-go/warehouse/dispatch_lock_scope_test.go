package warehouse

import (
	"testing"

	"backend-go/auth"
)

func TestCanManageDispatchLock(t *testing.T) {
	tests := []struct {
		name            string
		claims          *auth.PegasusClaims
		lockSupplierID  string
		lockWarehouseID string
		lockFactoryID   string
		want            bool
	}{
		{
			name:            "warehouse claims can manage same warehouse lock",
			claims:          &auth.PegasusClaims{Role: "WAREHOUSE", WarehouseID: "wh-1"},
			lockSupplierID:  "sup-1",
			lockWarehouseID: "wh-1",
			want:            true,
		},
		{
			name:            "warehouse claims cannot manage other warehouse lock",
			claims:          &auth.PegasusClaims{Role: "WAREHOUSE", WarehouseID: "wh-1"},
			lockSupplierID:  "sup-1",
			lockWarehouseID: "wh-2",
			want:            false,
		},
		{
			name:           "factory claims can manage same factory lock",
			claims:         &auth.PegasusClaims{Role: "FACTORY", FactoryID: "fac-1"},
			lockSupplierID: "sup-1",
			lockFactoryID:  "fac-1",
			want:           true,
		},
		{
			name:           "factory claims cannot manage other factory lock",
			claims:         &auth.PegasusClaims{Role: "FACTORY", FactoryID: "fac-1"},
			lockSupplierID: "sup-1",
			lockFactoryID:  "fac-9",
			want:           false,
		},
		{
			name:            "supplier claims can manage same supplier lock",
			claims:          &auth.PegasusClaims{Role: "SUPPLIER", SupplierID: "sup-1"},
			lockSupplierID:  "sup-1",
			lockWarehouseID: "wh-1",
			want:            true,
		},
		{
			name:            "supplier claims cannot manage other supplier lock",
			claims:          &auth.PegasusClaims{Role: "SUPPLIER", SupplierID: "sup-1"},
			lockSupplierID:  "sup-2",
			lockWarehouseID: "wh-1",
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canManageDispatchLock(tt.claims, tt.lockSupplierID, tt.lockWarehouseID, tt.lockFactoryID)
			if got != tt.want {
				t.Fatalf("canManageDispatchLock() = %v, want %v", got, tt.want)
			}
		})
	}
}
