package routing

import "context"

// GeometryBuilder resolves route overlays, preferring OSRM when configured.
type GeometryBuilder struct {
	osrm *OSRMClient
}

// NewGeometryBuilder constructs a builder. osrm may be nil (dense fallback only).
func NewGeometryBuilder(osrm *OSRMClient) *GeometryBuilder {
	return &GeometryBuilder{osrm: osrm}
}

// Build returns street geometry from OSRM when available, else haversine-densified segments.
func (b *GeometryBuilder) Build(ctx context.Context, routeID string, waypoints []LatLng) RouteGeometry {
	return b.BuildDetail(ctx, routeID, waypoints, false)
}

// BuildDetail optionally includes OSRM turn-by-turn steps.
func (b *GeometryBuilder) BuildDetail(ctx context.Context, routeID string, waypoints []LatLng, includeSteps bool) RouteGeometry {
	if b != nil && b.osrm != nil {
		if geometry, err := b.osrm.RouteGeometry(ctx, routeID, waypoints, includeSteps); err == nil {
			return geometry
		}
	}
	return BuildDenseRouteGeometry(routeID, waypoints)
}
