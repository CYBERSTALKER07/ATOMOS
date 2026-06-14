package dispatch

import (
	"reflect"
	"testing"
)

func TestSuggestOrdersToUnselect_MinimalSet(t *testing.T) {
	route := DispatchRoute{
		DriverID:     "drv-1",
		MaxVolume:    100,
		LoadedVolume: 190,
		Orders: []GeoOrder{
			{OrderID: "o-small", Volume: 20},
			{OrderID: "o-large", Volume: 90},
			{OrderID: "o-mid", Volume: 80},
		},
	}
	selected, excess := SuggestOrdersToUnselect(route)
	if excess <= 0 {
		t.Fatalf("expected positive excess, got %v", excess)
	}
	if !reflect.DeepEqual(selected, []string{"o-large", "o-mid"}) {
		t.Fatalf("unexpected selection: %#v", selected)
	}
}

func TestSuggestOrdersToUnselect_WithinCapacity(t *testing.T) {
	route := DispatchRoute{
		MaxVolume:    100,
		LoadedVolume: 90,
		Orders:       []GeoOrder{{OrderID: "o-1", Volume: 90}},
	}
	selected, excess := SuggestOrdersToUnselect(route)
	if len(selected) != 0 || excess != 0 {
		t.Fatalf("expected no suggestions, got ids=%v excess=%v", selected, excess)
	}
}
