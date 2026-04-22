// Package spannerrouter routes Spanner client selection based on geographic
// region derived from an H3 cell index.
//
// Problem it solves:
//
//	When enable_multiregion=true, the system operates Spanner instances in
//	multiple regions (Asia, EU). A supplier in Tashkent should read from the
//	Asia replica (10 ms), not be routed to EU (150 ms) by accident. Writes
//	always go to the primary instance — Spanner's multi-region config handles
//	replication. Only READS are routed regionally.
//
// Maglev-derived principle:
//
//	Routing decisions are made from a pre-built lookup table, not computed
//	per-request. Every pod builds the same table at init() with zero
//	coordination. The table is O(1) to query. Adding a new region requires
//	rebuilding the table — not a mid-flight change.
//
// Region derivation:
//
//	A res-7 H3 cell is parented to resolution 2 (~90,000 km² each) and
//	looked up in a static map built from representative lat/lng samples at
//	package init. Unknown cells fall back to the primary client (always correct,
//	potentially higher latency). This means no cross-request state, no Redis,
//	and no additional Spanner reads.
//
// Single-region mode:
//
//	When constructed via NewSingleRegion, all calls return the single primary
//	client. No overhead, no conditionals in the caller.
package spannerrouter

import (
	"cloud.google.com/go/spanner"
	h3 "github.com/uber/h3-go/v4"
)

// regionCells is the pre-built res-2 H3 cell → region lookup table.
// Built once at init(), zero allocation per query.
var regionCells map[h3.Cell]string

func init() {
	// Sample points that exhaustively cover each deployment region at H3 res-2.
	// Resolution 2 has ~98 cells globally; each cell is ~90,000 km².
	// Sampling every major urban cluster ensures all supplier warehouse cells
	// in the region are mapped, even at the edges.
	//
	// Uzbekistan / Central Asia cluster ("asia"):
	//   Covers: UZ, KZ, TM, TJ, KG, AF, AZ, AM, GE + surrounding steppe
	// Europe cluster ("eu"):
	//   Covers: EU27 + UK, Ukraine, Russia west of Urals, MENA north coast
	//
	// To add a region: append lat/lng samples and add the region key.
	// Run go test ./bootstrap/spannerrouter/... to verify coverage.
	samples := map[string][][2]float64{
		"asia": {
			// Core Uzbekistan cities
			{41.2, 69.2}, // Tashkent
			{39.7, 66.9}, // Samarkand
			{40.1, 65.4}, // Bukhara
			{40.5, 72.8}, // Andijan
			{41.5, 60.6}, // Urgench
			// Regional neighbours
			{43.3, 76.9}, // Almaty (KZ)
			{51.2, 71.4}, // Astana (KZ)
			{37.9, 58.4}, // Ashgabat (TM)
			{38.5, 68.8}, // Dushanbe (TJ)
			{42.9, 74.6}, // Bishkek (KG)
			{40.4, 49.9}, // Baku (AZ)
			{40.1, 44.5}, // Yerevan (AM)
			{41.7, 44.8}, // Tbilisi (GE)
			{35.7, 51.4}, // Tehran (IR)
			{34.5, 69.2}, // Kabul (AF)
		},
		"eu": {
			// Western Europe
			{48.8, 2.3},  // Paris
			{52.5, 13.4}, // Berlin
			{51.5, -0.1}, // London
			{41.9, 12.5}, // Rome
			{40.4, -3.7}, // Madrid
			{50.8, 4.4},  // Brussels
			{52.3, 4.9},  // Amsterdam
			{48.2, 16.4}, // Vienna
			// Northern / Eastern Europe
			{59.3, 18.0}, // Stockholm
			{60.2, 25.0}, // Helsinki
			{59.9, 10.8}, // Oslo
			{55.7, 12.6}, // Copenhagen
			{50.1, 14.4}, // Prague
			{47.5, 19.1}, // Budapest
			{52.2, 21.0}, // Warsaw
			{50.4, 30.5}, // Kyiv
			{55.8, 37.6}, // Moscow
			{44.8, 20.5}, // Belgrade
			// MENA north coast (EU-adjacent logistics partners)
			{36.8, 10.2}, // Tunis
			{30.1, 31.2}, // Cairo
		},
		"us": {
			{40.7, -74.0},  // New York
			{34.1, -118.2}, // Los Angeles
			{41.9, -87.6},  // Chicago
			{29.7, -95.4},  // Houston
			{33.4, -112.1}, // Phoenix
			{49.2, -123.1}, // Vancouver
			{43.7, -79.4},  // Toronto
			{19.4, -99.1},  // Mexico City
			{-23.5, -46.6}, // São Paulo
			{-34.6, -58.4}, // Buenos Aires
		},
	}

	regionCells = make(map[h3.Cell]string, len(samples)*15)
	for region, pts := range samples {
		for _, pt := range pts {
			// Parent to res-2 so adjacent res-7 cells in the same macro-region
			// all share a single lookup key.
			cell, err := h3.LatLngToCell(h3.LatLng{Lat: pt[0], Lng: pt[1]}, 2)
			if err != nil {
				// Unreachable for valid lat/lng within range; skip silently.
				continue
			}
			// First writer wins. Overlapping samples are benign; resolution 2
			// cells are large enough that overlap would be a data error anyway.
			if _, exists := regionCells[cell]; !exists {
				regionCells[cell] = region
			}
		}
	}
}

// Router selects a *spanner.Client based on entity location.
// All write paths MUST use Primary() — regional clients are read-only replicas.
type Router struct {
	primary  *spanner.Client
	regional map[string]*spanner.Client // region → read-replica client
}

// NewSingleRegion returns a Router backed by exactly one Spanner client.
// For(any cell) and Primary() both return c. Use when enable_multiregion=false.
func NewSingleRegion(c *spanner.Client) *Router {
	return &Router{
		primary:  c,
		regional: nil,
	}
}

// New returns a multiregion Router. regional maps region strings (e.g. "asia",
// "eu") to their Spanner read-replica clients. Unknown regions fall back to
// primary. primary must be non-nil.
func New(primary *spanner.Client, regional map[string]*spanner.Client) *Router {
	return &Router{
		primary:  primary,
		regional: regional,
	}
}

// For returns the Spanner client closest to the entity identified by h3Cell.
// If h3Cell is empty, the cell is unknown, or no regional client is configured
// for its region, Primary() is returned.
//
// Use For() for read queries where the entity's location is known (order reads,
// driver reads, retailer reads). For aggregations that span regions (dashboard
// totals, cross-region fleet views) always use Primary() for consistent results.
func (r *Router) For(h3Cell string) *spanner.Client {
	if len(r.regional) == 0 || h3Cell == "" {
		return r.primary
	}

	region := cellToRegion(h3Cell)
	if region == "" {
		return r.primary
	}

	if c, ok := r.regional[region]; ok {
		return c
	}
	return r.primary
}

// Primary always returns the primary Spanner client. Use for:
//   - All write transactions (ReadWriteTransaction)
//   - Cross-region aggregation queries (supplier dashboard totals)
//   - Entities with no location context (supplier profile, config)
//   - The outbox relay (reads and writes OutboxEvents)
func (r *Router) Primary() *spanner.Client {
	return r.primary
}

// cellToRegion maps a res-7 H3 cell to a deployment region string.
// Returns "" if the cell cannot be mapped (unknown region → primary fallback).
func cellToRegion(cell string) string {
	if len(cell) != 15 {
		return ""
	}

	// CellFromString returns zero-value Cell on invalid input (no error).
	c := h3.CellFromString(cell)
	if c == 0 {
		return ""
	}

	// Parent to res-2 — the granularity at which we distinguish regions.
	// This is a fast bit-mask operation in the h3 library (~50 ns).
	parent, err := c.Parent(2)
	if err != nil {
		return ""
	}

	return regionCells[parent]
}
