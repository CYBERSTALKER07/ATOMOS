package routing

import "context"

// GeometryBuilder resolves route overlays via Google Routes → OSRM → dense.
type GeometryBuilder struct {
	google *GoogleRoutesClient
	osrm   *OSRMClient
	mode   RoutingProviderMode
}

// NewGeometryBuilder constructs a builder. Either client may be nil.
// mode defaults to auto when empty/unknown.
func NewGeometryBuilder(google *GoogleRoutesClient, osrm *OSRMClient, mode RoutingProviderMode) *GeometryBuilder {
	if mode == "" {
		mode = RoutingProviderAuto
	}
	return &GeometryBuilder{google: google, osrm: osrm, mode: ParseRoutingProviderMode(string(mode))}
}

// Build returns street geometry from the configured provider chain.
func (b *GeometryBuilder) Build(ctx context.Context, routeID string, waypoints []LatLng) RouteGeometry {
	return b.BuildDetail(ctx, routeID, waypoints, false)
}

// BuildDetail optionally includes turn-by-turn steps from the winning provider.
func (b *GeometryBuilder) BuildDetail(ctx context.Context, routeID string, waypoints []LatLng, includeSteps bool) RouteGeometry {
	if len(waypoints) < 2 {
		return BuildDenseRouteGeometry(routeID, waypoints)
	}
	for _, p := range b.providers() {
		if p == nil {
			continue
		}
		geometry, err := p.RouteGeometry(ctx, routeID, waypoints, includeSteps)
		if err == nil && geometry.Source != "" && geometry.Source != "insufficient_waypoints" && len(geometry.Coordinates) >= 2 {
			return geometry
		}
		if err == nil && geometry.Source == "insufficient_waypoints" {
			return geometry
		}
	}
	return BuildDenseRouteGeometry(routeID, waypoints)
}

func (b *GeometryBuilder) providers() []RouteGeometryProvider {
	if b == nil {
		return nil
	}
	switch b.mode {
	case RoutingProviderGoogle:
		if b.google != nil {
			return []RouteGeometryProvider{b.google}
		}
		return nil
	case RoutingProviderOSRM:
		if b.osrm != nil {
			return []RouteGeometryProvider{b.osrm}
		}
		return nil
	default: // auto
		out := make([]RouteGeometryProvider, 0, 2)
		if b.google != nil {
			out = append(out, b.google)
		}
		if b.osrm != nil {
			out = append(out, b.osrm)
		}
		return out
	}
}
