package proximity

import "cloud.google.com/go/spanner"

// ReadRouter chooses a regional Spanner read client for an H3 cell.
// Implemented by bootstrap/spannerrouter.Router.
type ReadRouter interface {
	For(h3Cell string) *spanner.Client
	Primary() *spanner.Client
}

// ReadClientForCell returns the routed read client for a known H3 cell.
// Falls back to primary when routing context is unavailable.
func ReadClientForCell(primary *spanner.Client, router ReadRouter, h3Cell string) *spanner.Client {
	if primary == nil {
		return nil
	}
	if router == nil {
		return primary
	}
	if h3Cell == "" {
		return primary
	}

	readClient := router.For(h3Cell)
	if readClient == nil {
		return primary
	}
	return readClient
}

// ReadClientForRetailer returns the routed read client for retailer coordinates.
// Falls back to primary when routing context is unavailable.
func ReadClientForRetailer(primary *spanner.Client, router ReadRouter, retailerLat, retailerLng float64) *spanner.Client {
	if primary == nil {
		return nil
	}
	if router == nil {
		return primary
	}

	cell := LookupCell(retailerLat, retailerLng)
	return ReadClientForCell(primary, router, cell)
}

// readClientForRetailer is kept for backward compatibility with existing
// proximity package callsites while the exported helper is adopted repo-wide.
func readClientForRetailer(primary *spanner.Client, router ReadRouter, retailerLat, retailerLng float64) *spanner.Client {
	return ReadClientForRetailer(primary, router, retailerLat, retailerLng)
}
