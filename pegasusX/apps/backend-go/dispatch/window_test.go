package dispatch

import "testing"

func TestSortByWindowUrgency(t *testing.T) {
	orders := []DispatchableOrder{
		{OrderID: "flexible", ReceivingWindowClose: "", TotalMinor: 100},
		{OrderID: "early", ReceivingWindowClose: "12:00", TotalMinor: 50},
		{OrderID: "late", ReceivingWindowClose: "18:00", TotalMinor: 200},
	}
	SortByWindowUrgency(orders)
	if orders[0].OrderID != "early" || orders[1].OrderID != "late" || orders[2].OrderID != "flexible" {
		t.Fatalf("unexpected order: %#v", orders)
	}
}

func TestBuildPreviewIncludesWindows(t *testing.T) {
	preview := BuildPreview([]DispatchableOrder{{
		OrderID:              "o-1",
		RetailerID:           "r-1",
		RetailerName:         "Corner Shop",
		TotalMinor:           2500,
		Currency:             "UZS",
		ReceivingWindowOpen:  "09:00",
		ReceivingWindowClose: "17:00",
		VolumeVU:             3,
	}})
	if len(preview.UndispatchedOrders) != 1 {
		t.Fatalf("expected one wire row, got %d", len(preview.UndispatchedOrders))
	}
	row := preview.UndispatchedOrders[0]
	if row["receiving_window_open"] != "09:00" || row["receiving_window_close"] != "17:00" {
		t.Fatalf("window fields missing: %#v", row)
	}
	if preview.WindowConstrained != 1 {
		t.Fatalf("expected constrained count 1, got %d", preview.WindowConstrained)
	}
	if preview.GeoOrders[0].ReceivingWindowClose != "17:00" {
		t.Fatalf("geo order missing window: %#v", preview.GeoOrders[0])
	}
}
