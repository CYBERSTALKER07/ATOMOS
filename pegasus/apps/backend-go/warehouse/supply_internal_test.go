package warehouse

import "testing"

func TestIsInternalSupplyLane(t *testing.T) {
	tests := []struct {
		name      string
		cfg       warehouseNearbyConfig
		factoryID string
		want      bool
	}{
		{
			name:      "nearby with matching factory",
			cfg:       warehouseNearbyConfig{IsNearby: true, PrimaryFactory: "FAC-001"},
			factoryID: "FAC-001",
			want:      true,
		},
		{
			name:      "nearby with different factory",
			cfg:       warehouseNearbyConfig{IsNearby: true, PrimaryFactory: "FAC-001"},
			factoryID: "FAC-002",
			want:      false,
		},
		{
			name:      "not nearby",
			cfg:       warehouseNearbyConfig{IsNearby: false, PrimaryFactory: "FAC-001"},
			factoryID: "FAC-001",
			want:      false,
		},
		{
			name:      "nearby without primary factory",
			cfg:       warehouseNearbyConfig{IsNearby: true, PrimaryFactory: ""},
			factoryID: "FAC-001",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInternalSupplyLane(tt.cfg, tt.factoryID); got != tt.want {
				t.Fatalf("isInternalSupplyLane() = %v, want %v", got, tt.want)
			}
		})
	}
}
