package supplier

import "testing"

func TestWarehouseDetailTarget(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
		ok   bool
	}{
		{name: "detail path", path: "/v1/supplier/warehouses/wh-1", want: "wh-1", ok: true},
		{name: "coverage path rejected", path: "/v1/supplier/warehouses/wh-1/coverage", ok: false},
		{name: "missing id", path: "/v1/supplier/warehouses/", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := warehouseDetailTarget(tt.path)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("warehouseDetailTarget(%q) = (%q, %t), want (%q, %t)", tt.path, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestWarehouseCoverageTarget(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
		ok   bool
	}{
		{name: "coverage path", path: "/v1/supplier/warehouses/wh-1/coverage", want: "wh-1", ok: true},
		{name: "detail path rejected", path: "/v1/supplier/warehouses/wh-1", ok: false},
		{name: "unknown suffix rejected", path: "/v1/supplier/warehouses/wh-1/metrics", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := warehouseCoverageTarget(tt.path)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("warehouseCoverageTarget(%q) = (%q, %t), want (%q, %t)", tt.path, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestApplyWarehouseDerivedStats(t *testing.T) {
	warehouse := WarehouseResponse{
		H3Indexes: []string{"a", "b", "c"},
	}

	applyWarehouseDerivedStats(&warehouse)

	if warehouse.HexCount != 3 {
		t.Fatalf("applyWarehouseDerivedStats() hex_count = %d, want 3", warehouse.HexCount)
	}
}
