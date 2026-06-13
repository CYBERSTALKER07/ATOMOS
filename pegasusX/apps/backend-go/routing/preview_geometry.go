package routing

import (
	"context"
	"fmt"
)

// RouteGeometryWire is the portal-facing route overlay subset.
type RouteGeometryWire struct {
	RouteID         string   `json:"route_id,omitempty"`
	EncodedPolyline string   `json:"encoded_polyline,omitempty"`
	Coordinates     []LatLng `json:"coordinates"`
	Source          string   `json:"source"`
	StopCount       int      `json:"stop_count,omitempty"`
}

// AttachRouteGeometryToProposedRoutes enriches optimizer preview routes with map overlays.
func AttachRouteGeometryToProposedRoutes(
	ctx context.Context,
	builder *GeometryBuilder,
	depot LatLng,
	routes []map[string]any,
) {
	if builder == nil || len(routes) == 0 {
		return
	}
	for index, route := range routes {
		if route == nil {
			continue
		}
		waypoints := waypointsFromProposedRoute(depot, route)
		if len(waypoints) < 2 {
			continue
		}
		routeID := fmt.Sprintf("preview-%d", index)
		if driverID, ok := route["driver_id"].(string); ok && driverID != "" {
			routeID = "preview-" + driverID
		}
		geometry := builder.Build(ctx, routeID, waypoints)
		route["route_geometry"] = ToRouteGeometryWire(geometry)
	}
}

// ToRouteGeometryWire maps the driver contract to the portal preview subset.
func ToRouteGeometryWire(geometry RouteGeometry) RouteGeometryWire {
	return RouteGeometryWire{
		RouteID:         geometry.RouteID,
		EncodedPolyline: geometry.EncodedPolyline,
		Coordinates:     geometry.Coordinates,
		Source:          geometry.Source,
		StopCount:       geometry.StopCount,
	}
}

func waypointsFromProposedRoute(depot LatLng, route map[string]any) []LatLng {
	waypoints := []LatLng{depot}
	stops, ok := route["stops"].([]map[string]any)
	if !ok {
		return waypoints
	}
	for _, stop := range stops {
		lat, latOK := floatFromAny(stop["lat"])
		lng, lngOK := floatFromAny(stop["lng"])
		if !latOK || !lngOK {
			continue
		}
		waypoints = append(waypoints, LatLng{Lat: lat, Lng: lng})
	}
	return waypoints
}

func floatFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}
