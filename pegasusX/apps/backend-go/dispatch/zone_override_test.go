package dispatch

import "testing"

func TestPointInGeoJSONPolygon(t *testing.T) {
	geo := `{"type":"Polygon","coordinates":[[[69.0,41.0],[69.1,41.0],[69.1,41.1],[69.0,41.1],[69.0,41.0]]]}`
	if !PointInGeoJSONPolygon(geo, 41.05, 69.05) {
		t.Fatal("expected point inside polygon")
	}
	if PointInGeoJSONPolygon(geo, 40.0, 69.0) {
		t.Fatal("expected point outside polygon")
	}
}

func TestApplyZoneOverridesFreeze(t *testing.T) {
	orders := []DispatchableOrder{
		{OrderID: "o1", Lat: 41.05, Lng: 69.05, WarehouseID: "wh1"},
		{OrderID: "o2", Lat: 40.0, Lng: 69.0, WarehouseID: "wh1"},
	}
	overrides := []ZoneOverride{{
		OverrideID: "ov1",
		Action:     "FREEZE_DISPATCH",
		Polygon:    `{"type":"Polygon","coordinates":[[[69.0,41.0],[69.1,41.0],[69.1,41.1],[69.0,41.1],[69.0,41.0]]]}`,
	}}
	filtered, meta := ApplyZoneOverrides(orders, overrides)
	if len(filtered) != 1 || filtered[0].OrderID != "o2" {
		t.Fatalf("unexpected filtered orders: %+v", filtered)
	}
	if len(meta) != 1 {
		t.Fatalf("expected override meta, got %+v", meta)
	}
}
