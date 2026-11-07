package routing

import (
	"context"
	"strings"
)

// RouteGeometryProvider resolves driving geometry for a waypoint sequence.
type RouteGeometryProvider interface {
	RouteGeometry(ctx context.Context, routeID string, waypoints []LatLng, includeSteps bool) (RouteGeometry, error)
}

// RoutingProviderMode selects which street-geometry backends to attempt.
// Values: "auto" (Google → OSRM → dense), "google", "osrm".
type RoutingProviderMode string

const (
	RoutingProviderAuto   RoutingProviderMode = "auto"
	RoutingProviderGoogle RoutingProviderMode = "google"
	RoutingProviderOSRM   RoutingProviderMode = "osrm"
)

// ParseRoutingProviderMode normalizes ROUTING_PROVIDER env values.
func ParseRoutingProviderMode(raw string) RoutingProviderMode {
	switch RoutingProviderMode(strings.ToLower(strings.TrimSpace(raw))) {
	case RoutingProviderGoogle:
		return RoutingProviderGoogle
	case RoutingProviderOSRM:
		return RoutingProviderOSRM
	default:
		return RoutingProviderAuto
	}
}
