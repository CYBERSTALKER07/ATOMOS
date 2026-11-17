package warehouseops

import (
	"reflect"
	"sort"
	"testing"
)

func TestResolveLatestUpdates(t *testing.T) {
	tests := []struct {
		name    string
		updates []LWWInventorySet
		want    map[string]LWWInventorySet
	}{
		{
			name:    "empty updates",
			updates: nil,
			want:    map[string]LWWInventorySet{},
		},
		{
			name: "single update",
			updates: []LWWInventorySet{
				{ProductID: "sku-1", Quantity: 10, WriteClock: 100, WorkerNodeID: "node-a"},
			},
			want: map[string]LWWInventorySet{
				"sku-1": {ProductID: "sku-1", Quantity: 10, WriteClock: 100, WorkerNodeID: "node-a"},
			},
		},
		{
			name: "duplicate product keeps strictly highest clock",
			updates: []LWWInventorySet{
				{ProductID: "sku-1", Quantity: 10, WriteClock: 100, WorkerNodeID: "node-a"},
				{ProductID: "sku-1", Quantity: 15, WriteClock: 150, WorkerNodeID: "node-b"},
				{ProductID: "sku-1", Quantity: 8, WriteClock: 120, WorkerNodeID: "node-c"},
			},
			want: map[string]LWWInventorySet{
				"sku-1": {ProductID: "sku-1", Quantity: 15, WriteClock: 150, WorkerNodeID: "node-b"},
			},
		},
		{
			name: "trims whitespace and ignores empty product IDs",
			updates: []LWWInventorySet{
				{ProductID: "   ", Quantity: 10, WriteClock: 100, WorkerNodeID: "node-a"},
				{ProductID: "  sku-2  ", Quantity: 20, WriteClock: 200, WorkerNodeID: "node-b"},
			},
			want: map[string]LWWInventorySet{
				"sku-2": {ProductID: "sku-2", Quantity: 20, WriteClock: 200, WorkerNodeID: "node-b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveLatestUpdates(tt.updates)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveLatestUpdates() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFilterStrictlyNewer(t *testing.T) {
	existingClocks := map[string]int64{
		"sku-older": 100,
		"sku-same":  200,
		"sku-newer": 300,
	}

	updates := map[string]LWWInventorySet{
		"sku-older": {ProductID: "sku-older", Quantity: 50, WriteClock: 150, WorkerNodeID: "node-1"}, // 150 > 100 -> INCLUDE
		"sku-same":  {ProductID: "sku-same", Quantity: 60, WriteClock: 200, WorkerNodeID: "node-2"},  // 200 == 200 -> EXCLUDE
		"sku-newer": {ProductID: "sku-newer", Quantity: 70, WriteClock: 250, WorkerNodeID: "node-3"}, // 250 < 300 -> EXCLUDE
		"sku-new":   {ProductID: "sku-new", Quantity: 80, WriteClock: 10, WorkerNodeID: "node-4"},    // 10 > 0 (absent) -> INCLUDE
	}

	got := FilterStrictlyNewer(existingClocks, updates)
	sort.Slice(got, func(i, j int) bool {
		return got[i].ProductID < got[j].ProductID
	})

	want := []LWWInventorySet{
		{ProductID: "sku-new", Quantity: 80, WriteClock: 10, WorkerNodeID: "node-4"},
		{ProductID: "sku-older", Quantity: 50, WriteClock: 150, WorkerNodeID: "node-1"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterStrictlyNewer() = %+v, want %+v", got, want)
	}
}
