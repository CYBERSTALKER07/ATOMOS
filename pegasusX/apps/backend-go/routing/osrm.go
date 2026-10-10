package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

const osrmRouteTimeout = 3 * time.Second
const osrmTableTimeout = 5 * time.Second

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
		Geometry string    `json:"geometry"`
		Distance float64   `json:"distance"`
		Legs     []osrmLeg `json:"legs"`
	} `json:"routes"`
}

type osrmTableResponse struct {
	Code     string      `json:"code"`
	Distances [][]float64 `json:"distances"`
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

// DistanceMatrix calls OSRM /table for pairwise driving distances in meters.
// points must be non-empty; returned matrix is NxN integers (meters).
func (c *OSRMClient) DistanceMatrix(ctx context.Context, points []LatLng) ([][]int, error) {
	if c == nil {
		return nil, fmt.Errorf("osrm: nil client")
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("osrm: empty points")
	}
	if len(points) == 1 {
		return [][]int{{0}}, nil
	}

	var matrix [][]int
	call := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			c.tableURL(points),
			nil,
		)
		if err != nil {
			return fmt.Errorf("osrm table request: %w", err)
		}

		httpClient := c.http
		if httpClient == nil {
			httpClient = &http.Client{Timeout: osrmTableTimeout}
		} else {
			// Table payloads are larger; prefer a dedicated timeout budget.
			clone := *httpClient
			if clone.Timeout == 0 || clone.Timeout < osrmTableTimeout {
				clone.Timeout = osrmTableTimeout
			}
			httpClient = &clone
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("osrm table http: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		if err != nil {
			return fmt.Errorf("osrm table read: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("osrm table status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var payload osrmTableResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return fmt.Errorf("osrm table decode: %w", err)
		}
		if !strings.EqualFold(payload.Code, "Ok") {
			return fmt.Errorf("osrm table code %q", payload.Code)
		}
		if len(payload.Distances) != len(points) {
			return fmt.Errorf("osrm table size %d != %d", len(payload.Distances), len(points))
		}
		out := make([][]int, len(points))
		for i := range points {
			if len(payload.Distances[i]) != len(points) {
				return fmt.Errorf("osrm table row %d size %d", i, len(payload.Distances[i]))
			}
			out[i] = make([]int, len(points))
			for j := range points {
				d := payload.Distances[i][j]
				if d < 0 || math.IsNaN(d) {
					// OSRM uses null/-1 for unreachable; leave 0 and let caller fill haversine.
					out[i][j] = 0
					continue
				}
				out[i][j] = int(d)
			}
		}
		matrix = out
		return nil
	}

	var err error
	if c.breaker != nil {
		err = c.breaker.Do(ctx, call)
	} else {
		err = call(ctx)
	}
	if err != nil {
		return nil, err
	}
	return matrix, nil
}

func (c *OSRMClient) tableURL(points []LatLng) string {
	return c.baseURL + "/table/v1/driving/" + formatOSRMCoordinates(points) +
		"?annotations=distance"
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
