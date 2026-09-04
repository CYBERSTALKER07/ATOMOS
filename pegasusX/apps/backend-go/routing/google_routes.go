package routing

import (
	"bytes"
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

const (
	googleRoutesTimeout     = 5 * time.Second
	googleRoutesDefaultURL  = "https://routes.googleapis.com"
	googleRoutesComputePath = "/directions/v2:computeRoutes"
	// ComputeRoutes allows origin + destination + up to 25 intermediates.
	googleRoutesMaxWaypoints = 27
	googleRoutesSource       = "google_routes_driving"
)

// GoogleRoutesClient calls Google Routes API ComputeRoutes for driving geometry.
type GoogleRoutesClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
	breaker *circuit.Breaker
}

// NewGoogleRoutesClient returns nil when apiKey is empty.
// baseURL may be empty (defaults to production Routes API) or an httptest URL in tests.
func NewGoogleRoutesClient(apiKey, baseURL string, breaker *circuit.Breaker) *GoogleRoutesClient {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = googleRoutesDefaultURL
	}
	return &GoogleRoutesClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: googleRoutesTimeout},
		breaker: breaker,
	}
}

type googleRoutesRequest struct {
	Origin                   googleWaypoint  `json:"origin"`
	Destination              googleWaypoint  `json:"destination"`
	Intermediates            []googleWaypoint `json:"intermediates,omitempty"`
	TravelMode               string          `json:"travelMode"`
	PolylineQuality          string          `json:"polylineQuality"`
	PolylineEncoding         string          `json:"polylineEncoding"`
	ComputeAlternativeRoutes bool            `json:"computeAlternativeRoutes"`
	LanguageCode             string          `json:"languageCode"`
	Units                    string          `json:"units"`
}

type googleWaypoint struct {
	Location googleLocation `json:"location"`
}

type googleLocation struct {
	LatLng googleLatLng `json:"latLng"`
}

type googleLatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type googleRoutesResponse struct {
	Routes []googleRoute `json:"routes"`
}

type googleRoute struct {
	Polyline googlePolyline `json:"polyline"`
	Legs     []googleLeg    `json:"legs"`
}

type googlePolyline struct {
	EncodedPolyline string `json:"encodedPolyline"`
}

type googleLeg struct {
	Steps []googleStep `json:"steps"`
}

type googleStep struct {
	DistanceMeters         int64                     `json:"distanceMeters"`
	StaticDuration         string                    `json:"staticDuration"`
	NavigationInstruction  *googleNavInstruction     `json:"navigationInstruction"`
	StartLocation          *googleLocationWrapper    `json:"startLocation"`
}

type googleNavInstruction struct {
	Maneuver string `json:"maneuver"`
	Instructions string `json:"instructions"`
}

type googleLocationWrapper struct {
	LatLng googleLatLng `json:"latLng"`
}

// RouteGeometry implements RouteGeometryProvider.
func (c *GoogleRoutesClient) RouteGeometry(ctx context.Context, routeID string, waypoints []LatLng, includeSteps bool) (RouteGeometry, error) {
	if c == nil {
		return RouteGeometry{}, fmt.Errorf("google routes: nil client")
	}
	if len(waypoints) < 2 {
		return RouteGeometry{
			RouteID:     routeID,
			Coordinates: []LatLng{},
			Source:      "insufficient_waypoints",
			StopCount:   len(waypoints),
		}, nil
	}

	trimmed := trimGoogleWaypoints(waypoints)
	body, err := json.Marshal(buildGoogleRoutesRequest(trimmed))
	if err != nil {
		return RouteGeometry{}, fmt.Errorf("google routes encode: %w", err)
	}

	fieldMask := "routes.polyline.encodedPolyline"
	if includeSteps {
		fieldMask = "routes.polyline.encodedPolyline,routes.legs.steps.navigationInstruction,routes.legs.steps.distanceMeters,routes.legs.steps.staticDuration,routes.legs.steps.startLocation"
	}

	var geometry RouteGeometry
	call := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+googleRoutesComputePath, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("google routes request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Goog-Api-Key", c.apiKey)
		req.Header.Set("X-Goog-FieldMask", fieldMask)

		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("google routes http: %w", err)
		}
		defer resp.Body.Close()

		raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			return fmt.Errorf("google routes read: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("google routes status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		}

		var payload googleRoutesResponse
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("google routes decode: %w", err)
		}
		if len(payload.Routes) == 0 || strings.TrimSpace(payload.Routes[0].Polyline.EncodedPolyline) == "" {
			return fmt.Errorf("google routes: empty route")
		}
		route := payload.Routes[0]
		coords, err := DecodePolyline(route.Polyline.EncodedPolyline)
		if err != nil {
			return fmt.Errorf("google routes polyline: %w", err)
		}
		if len(coords) < 2 {
			return fmt.Errorf("google routes polyline too short")
		}

		geometry = RouteGeometry{
			RouteID:         routeID,
			EncodedPolyline: route.Polyline.EncodedPolyline,
			Coordinates:     coords,
			Source:          googleRoutesSource,
			StopCount:       len(waypoints),
		}
		if includeSteps {
			geometry.Steps = parseGoogleRoutesSteps(route.Legs)
		}
		return nil
	}

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

func buildGoogleRoutesRequest(waypoints []LatLng) googleRoutesRequest {
	req := googleRoutesRequest{
		Origin:                   toGoogleWaypoint(waypoints[0]),
		Destination:              toGoogleWaypoint(waypoints[len(waypoints)-1]),
		TravelMode:               "DRIVE",
		PolylineQuality:          "HIGH_QUALITY",
		PolylineEncoding:         "ENCODED_POLYLINE",
		ComputeAlternativeRoutes: false,
		LanguageCode:             "en-US",
		Units:                    "METRIC",
	}
	if len(waypoints) > 2 {
		mids := waypoints[1 : len(waypoints)-1]
		req.Intermediates = make([]googleWaypoint, 0, len(mids))
		for _, p := range mids {
			req.Intermediates = append(req.Intermediates, toGoogleWaypoint(p))
		}
	}
	return req
}

func toGoogleWaypoint(p LatLng) googleWaypoint {
	return googleWaypoint{Location: googleLocation{LatLng: googleLatLng{Latitude: p.Lat, Longitude: p.Lng}}}
}

func trimGoogleWaypoints(waypoints []LatLng) []LatLng {
	if len(waypoints) <= googleRoutesMaxWaypoints {
		return waypoints
	}
	// Keep origin + destination; evenly sample intermediates into the remaining slots.
	maxMids := googleRoutesMaxWaypoints - 2
	mids := waypoints[1 : len(waypoints)-1]
	out := make([]LatLng, 0, googleRoutesMaxWaypoints)
	out = append(out, waypoints[0])
	if maxMids > 0 && len(mids) > 0 {
		if len(mids) <= maxMids {
			out = append(out, mids...)
		} else if maxMids == 1 {
			out = append(out, mids[0])
		} else {
			for i := 0; i < maxMids; i++ {
				idx := i * (len(mids) - 1) / (maxMids - 1)
				out = append(out, mids[idx])
			}
		}
	}
	out = append(out, waypoints[len(waypoints)-1])
	return out
}

func parseGoogleRoutesSteps(legs []googleLeg) []RouteStep {
	if len(legs) == 0 {
		return nil
	}
	steps := make([]RouteStep, 0, 16)
	for _, leg := range legs {
		for _, step := range leg.Steps {
			instruction := ""
			maneuver := ""
			if step.NavigationInstruction != nil {
				instruction = strings.TrimSpace(step.NavigationInstruction.Instructions)
				maneuver = strings.TrimSpace(step.NavigationInstruction.Maneuver)
			}
			lat, lng := 0.0, 0.0
			if step.StartLocation != nil {
				lat = step.StartLocation.LatLng.Latitude
				lng = step.StartLocation.LatLng.Longitude
			}
			steps = append(steps, RouteStep{
				Instruction: instruction,
				DistanceM:   float64(step.DistanceMeters),
				DurationS:   parseGoogleDurationSeconds(step.StaticDuration),
				Maneuver:    maneuver,
				Lat:         lat,
				Lng:         lng,
			})
		}
	}
	return steps
}

func parseGoogleDurationSeconds(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if strings.HasSuffix(raw, "s") {
		raw = strings.TrimSuffix(raw, "s")
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}
