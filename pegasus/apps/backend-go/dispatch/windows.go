package dispatch

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// parseHHMM converts "HH:MM" to minutes since midnight. Empty returns ok=false.
func parseHHMM(value string) (minutes int, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	t, err := time.Parse("15:04", value)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

// windowsCompatible returns true when two orders can share a route without
// violating receiving window constraints.
func windowsCompatible(a, b DispatchableOrder) bool {
	aOpen, aHasOpen := parseHHMM(a.ReceivingWindowOpen)
	aClose, aHasClose := parseHHMM(a.ReceivingWindowClose)
	bOpen, bHasOpen := parseHHMM(b.ReceivingWindowOpen)
	bClose, bHasClose := parseHHMM(b.ReceivingWindowClose)

	if !aHasOpen && !aHasClose && !bHasOpen && !bHasClose {
		return true
	}

	// Require overlapping delivery windows when any bound is set.
	openA, closeA := aOpen, aClose
	if !aHasOpen {
		openA = 0
	}
	if !aHasClose {
		closeA = 24*60 - 1
	}
	openB, closeB := bOpen, bClose
	if !bHasOpen {
		openB = 0
	}
	if !bHasClose {
		closeB = 24*60 - 1
	}
	return openA <= closeB && openB <= closeA
}

// routeWindowsCompatible checks whether adding candidate to route orders keeps
// a mutually compatible receiving window cluster.
func routeWindowsCompatible(route []GeoOrder, candidate DispatchableOrder) bool {
	cand := candidate.ToGeo()
	for _, existing := range route {
		if !windowsCompatible(dispatchableFromGeo(existing), dispatchableFromGeo(cand)) {
			return false
		}
	}
	return true
}

func dispatchableFromGeo(g GeoOrder) DispatchableOrder {
	return DispatchableOrder{
		OrderID:              g.OrderID,
		ReceivingWindowOpen:  g.ReceivingWindowOpen,
		ReceivingWindowClose: g.ReceivingWindowClose,
	}
}

// sortByWindowStart orders candidates so earlier receiving windows pack first.
func sortByWindowStart(orders []DispatchableOrder) {
	sort.SliceStable(orders, func(i, j int) bool {
		oi, oiOK := parseHHMM(orders[i].ReceivingWindowOpen)
		oj, ojOK := parseHHMM(orders[j].ReceivingWindowOpen)
		if oiOK && ojOK {
			return oi < oj
		}
		return orders[i].ReceivingWindowOpen < orders[j].ReceivingWindowOpen
	})
}

// windowMismatchWarning builds a human-readable warning when an order cannot
// join an existing route due to incompatible windows.
func windowMismatchWarning(orderID, routeDriver string) string {
	return fmt.Sprintf("ORDER %s: receiving window incompatible with route %s — opening new route", orderID, routeDriver)
}
