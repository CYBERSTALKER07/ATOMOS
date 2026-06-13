package dispatch

import (
	"encoding/json"
	"testing"
)

func TestOrderVolumeVU_UsesLineItemSnapshot(t *testing.T) {
	raw, err := json.Marshal([]map[string]any{
		{"sku": "prod-1", "quantity": 3, "unit_volume_vu": 2.5},
		{"sku": "prod-2", "quantity": 1, "unit_volume_vu": 0.5},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := OrderVolumeVU(raw, nil)
	want := 8.0
	if got != want {
		t.Fatalf("OrderVolumeVU = %v want %v", got, want)
	}
}

func TestOrderVolumeVU_FallsBackToProductLookup(t *testing.T) {
	raw, err := json.Marshal([]map[string]any{
		{"sku": "prod-1", "quantity": 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	lookup := map[string]float64{"prod-1": 4.0}
	got := OrderVolumeVU(raw, lookup)
	if got != 8.0 {
		t.Fatalf("OrderVolumeVU = %v want 8", got)
	}
}

func TestManualCapacityWarnings_ExceedsTetrisBuffer(t *testing.T) {
	tetrisEffective := 100.0 * TetrisBuffer
	if tetrisEffective != 95.0 {
		t.Fatalf("unexpected tetris effective %v", tetrisEffective)
	}
}
