package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

const osrmRouteTimeout = 3 * time.Second

// OSRMClient calls the public OSRM route service for driving geometry.
type OSRMClient struct {
	baseURL string
	http    *http.Client
	breaker *circuit.Breaker
}

// NewOSRMClient returns nil when baseURL is empty.
func NewOSRMClient(baseURL string, breaker *circuit.Breaker) *OSRMClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	return &OSRMClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: osrmRouteTimeout},
		breaker: breaker,
	}
}

type osrmRouteResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Geometry string  `json:"geometry"`
		Distance float64 `json:"distance"`
		Legs     []osrmLeg `json:"legs"`
	} `json:"routes"`
}

// RouteGeometry requests a driving route through OSRM for the given waypoints.
func (c *OSRMClient) RouteGeometry(ctx context.Context, routeID string, waypoints []LatLng, includeSteps bool) (RouteGeometry, error) {
	if c == nil {
		return RouteGeometry{}, fmt.Errorf("osrm: nil client")
	}
	if len(waypoints) < 2 {
		return RouteGeometry{
			RouteID:     routeID,
			Coordinates: []LatLng{},
			Source:      "insufficient_waypoints",
			StopCount:   len(waypoints),
		}, nil
	}

	var geometry RouteGeometry
	call := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			c.routeURL(waypoints, includeSteps),
			nil,
		)
		if err != nil {
			return fmt.Errorf("osrm request: %w", err)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("osrm http: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return fmt.Errorf("osrm read body: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("osrm status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var payload osrmRouteResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return fmt.Errorf("osrm decode: %w", err)
		}
		if !strings.EqualFold(payload.Code, "Ok") || len(payload.Routes) == 0 {
			return fmt.Errorf("osrm code %q", payload.Code)
		}
		route := payload.Routes[0]
		coords, err := DecodePolyline(route.Geometry)
		if err != nil {
			return fmt.Errorf("osrm polyline: %w", err)
		}
		if len(coords) < 2 {
			return fmt.Errorf("osrm polyline too short")
		}

		geometry = RouteGeometry{
			RouteID:         routeID,
			EncodedPolyline: route.Geometry,
			Coordinates:     coords,
			Source:          "osrm_driving",
			StopCount:       len(waypoints),
		}
		if includeSteps {
			geometry.Steps = parseOSRMSteps(route.Legs)
		}
		return nil
	}

	var err error
	if c.breaker != nil {
		err = c.breaker.Do(ctx, call)
	} else {
		err = call(ctx)
	}
	if err != nil {
		return RouteGeometry{}, err
	}
	return geometry, nil
}

func (c *OSRMClient) routeURL(waypoints []LatLng, includeSteps bool) string {
	steps := "false"
	if includeSteps {
		steps = "true"
	}
	return c.baseURL + "/route/v1/driving/" + formatOSRMCoordinates(waypoints) +
		"?overview=full&geometries=polyline&steps=" + steps
}

func formatOSRMCoordinates(waypoints []LatLng) string {
	var b strings.Builder
	for i, point := range waypoints {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(strconv.FormatFloat(point.Lng, 'f', 6, 64))
		b.WriteByte(',')
		b.WriteString(strconv.FormatFloat(point.Lat, 'f', 6, 64))
	}
	return b.String()
}
