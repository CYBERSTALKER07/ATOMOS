package dispatch

import "fmt"
import "testing"

func TestSplitManifestNamesOverflowChunks(t *testing.T) {
	orders := make([]GeoOrder, 26)
	for i := range orders {
		orders[i] = GeoOrder{OrderID: fmt.Sprintf("ord-%d", i), Volume: 1}
	}
	group := SplitManifestAutoRoute("driver-abc12345", "trk-1", orders, 25, 1_700_000)
	if len(group.Chunks) != 2 {
		t.Fatalf("chunks=%d want 2", len(group.Chunks))
	}
	if group.Chunks[0].Suffix != "A" || group.Chunks[1].Suffix != "B" {
		t.Fatalf("suffixes=%q %q", group.Chunks[0].Suffix, group.Chunks[1].Suffix)
	}
	if len(group.Chunks[0].Orders) != 25 || len(group.Chunks[1].Orders) != 1 {
		t.Fatalf("sizes=%d %d", len(group.Chunks[0].Orders), len(group.Chunks[1].Orders))
	}
	if group.Chunks[0].RouteID == group.Chunks[1].RouteID {
		t.Fatal("chunk route ids must differ")
	}
}

func TestExpandOversizeRoutesLeavesSmallRoutes(t *testing.T) {
	in := []DispatchRoute{{
		DriverID:     "d1",
		MaxVolume:    150,
		LoadedVolume: 3,
		Orders:       []GeoOrder{{OrderID: "o1", Volume: 1}, {OrderID: "o2", Volume: 2}},
	}}
	got := ExpandOversizeRoutes(in, 99)
	if len(got) != 1 || got[0].RouteID != "" {
		t.Fatalf("small route must stay unsplit: %+v", got)
	}
}

func TestAlphaIndex(t *testing.T) {
	if AlphaIndex(0) != "A" || AlphaIndex(25) != "Z" || AlphaIndex(26) != "AA" {
		t.Fatalf("A=%s Z=%s AA=%s", AlphaIndex(0), AlphaIndex(25), AlphaIndex(26))
	}
}
