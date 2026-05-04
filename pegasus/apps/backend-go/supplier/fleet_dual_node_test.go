package supplier

import (
	"testing"

	"backend-go/auth"
)

func TestEffectiveFleetHomeNode(t *testing.T) {
	tests := []struct {
		name        string
		nodeType    string
		nodeID      string
		warehouseID string
		wantType    string
		wantID      string
	}{
		{
			name:        "explicit home node preserved",
			nodeType:    auth.HomeNodeTypeFactory,
			nodeID:      "fac-1",
			warehouseID: "wh-1",
			wantType:    auth.HomeNodeTypeFactory,
			wantID:      "fac-1",
		},
		{
			name:        "legacy warehouse fallback",
			nodeType:    "",
			nodeID:      "",
			warehouseID: "wh-1",
			wantType:    auth.HomeNodeTypeWarehouse,
			wantID:      "wh-1",
		},
		{
			name:        "empty when no node context",
			nodeType:    "",
			nodeID:      "",
			warehouseID: "",
			wantType:    "",
			wantID:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotID := effectiveFleetHomeNode(tt.nodeType, tt.nodeID, tt.warehouseID)
			if gotType != tt.wantType || gotID != tt.wantID {
				t.Fatalf("effectiveFleetHomeNode() = (%q, %q), want (%q, %q)", gotType, gotID, tt.wantType, tt.wantID)
			}
		})
	}
}

func TestDriverModeForHomeNode(t *testing.T) {
	tests := []struct {
		name     string
		nodeType string
		want     string
	}{
		{name: "factory mode", nodeType: auth.HomeNodeTypeFactory, want: "FACTORY_TRANSFER"},
		{name: "warehouse mode", nodeType: auth.HomeNodeTypeWarehouse, want: "RETAIL_DELIVERY"},
		{name: "empty defaults to retail delivery", nodeType: "", want: "RETAIL_DELIVERY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := driverModeForHomeNode(tt.nodeType); got != tt.want {
				t.Fatalf("driverModeForHomeNode(%q) = %q, want %q", tt.nodeType, got, tt.want)
			}
		})
	}
}
