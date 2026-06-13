package supplier

import "testing"

func TestSupplierDriverTruckStatus(t *testing.T) {
	tests := []struct {
		name             string
		isActive         bool
		onActiveManifest bool
		wantStatus       string
		wantUnavailable  bool
	}{
		{name: "inactive", isActive: false, wantStatus: "UNAVAILABLE", wantUnavailable: true},
		{name: "active manifest", isActive: true, onActiveManifest: true, wantStatus: "IN_TRANSIT", wantUnavailable: true},
		{name: "available", isActive: true, wantStatus: "AVAILABLE", wantUnavailable: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, unavailable := supplierDriverTruckStatus(tt.isActive, tt.onActiveManifest)
			if status != tt.wantStatus || unavailable != tt.wantUnavailable {
				t.Fatalf("supplierDriverTruckStatus() = (%q, %v) want (%q, %v)", status, unavailable, tt.wantStatus, tt.wantUnavailable)
			}
		})
	}
}
