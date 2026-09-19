// Package optimizerclient is the backend-side HTTP client for the Phase 2
// dispatch optimiser hosted in services/optimizer-core. It is the only call
// site for `POST /v1/optimizer/solve` and the single source of truth for the
// HTTP timeout, header convention, and graceful-fallback contract.
//
// On any of: network error, HTTP 5xx, contract.ErrCodeTimeout, malformed JSON,
// or a Go context deadline, Solve returns (nil, error). The caller is
// expected to fall back to the legacy H3 + binpack pipeline in
// dispatch.BinPack — the optimiser is an enhancement, never a hard dependency.
package optimizerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
)

// DefaultTimeout is the HTTP soft timeout. Must stay strictly greater than the
// solver default time_limit_ms (5s) so OR-Tools can finish before the wire cuts.
const DefaultTimeout = 8 * time.Second

// DefaultSolverTimeLimitMs is embedded in SolveRequest.Tunables.
const DefaultSolverTimeLimitMs = 5000

// Default per-stop service time when the caller does not specify.
const defaultServiceMinutes = 5

// Default truck cruising speed (km/h) when the vehicle row carries no speed.
const defaultAvgSpeedKmph = 30.0

// Client wraps a configured net/http client + the optimiser endpoint URL +
// the shared internal-API key. Construct once, reuse across requests.
type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	osrm       *routing.OSRMClient
}

// New returns a Client. endpoint must include the scheme + host, e.g.
// "http://optimizer-core:8082". The path contract.SolvePath is appended internally.
func New(endpoint, apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		endpoint:   endpoint,
		apiKey:     apiKey,
	}
}

// WithOSRM attaches an OSRM client used to build distance_matrix_m before Solve.
func (c *Client) WithOSRM(osrm *routing.OSRMClient) *Client {
	if c != nil {
		c.osrm = osrm
	}
	return c
}

// SolveInput is the backend-domain request shape. Solve() converts it into
// contract.SolveRequest under the hood so callers never touch the wire types.
type SolveInput struct {
	TraceID       string
	SupplierID    string
	HomeNodeID    string
	DepotLat      float64
	DepotLng      float64
	DepartureTime time.Time
	Orders        []dispatch.GeoOrder
	Fleet         []dispatch.AvailableDriver
	TetrisBuffer  float64
}

// Solve calls the optimiser and returns an AssignmentResult mapped from the
// VRP response. Routes preserve VehicleID + DriverID + per-stop ordering.
// Orphans become AssignmentResult.Orphans with their reason captured in the
// warnings slice for operator visibility.
func (c *Client) Solve(ctx context.Context, in SolveInput) (*dispatch.AssignmentResult, error) {
	if c == nil {
		return nil, errors.New("optimizerclient: nil client")
	}
	if c.endpoint == "" || c.apiKey == "" {
		return nil, errors.New("optimizerclient: endpoint or apiKey not configured")
	}
	if len(in.Orders) == 0 {
		return nil, errors.New("optimizerclient: no orders to solve")
	}
	if len(in.Fleet) == 0 {
		return nil, errors.New("optimizerclient: empty fleet")
	}

	vehicles := buildVehicles(in.Fleet, in.DepotLat, in.DepotLng)
	stops := buildStops(in.Orders)
	req := contract.SolveRequest{
		V:               contract.V,
		TraceID:         in.TraceID,
		SupplierID:      in.SupplierID,
		HomeNodeID:      in.HomeNodeID,
		DepartureTime:   in.DepartureTime.UTC().Format(time.RFC3339),
		Stops:           stops,
		Vehicles:        vehicles,
		DistanceMatrixM: buildDistanceMatrix(ctx, c.osrm, vehicles, stops),
		Tunables: &contract.Tunables{
			TimeLimitMs:  DefaultSolverTimeLimitMs,
			TetrisBuffer: in.TetrisBuffer,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("optimizerclient: encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.endpoint+contract.SolvePath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("optimizerclient: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(contract.AuthHeader, c.apiKey)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("optimizerclient: do request: %w", err)
	}
	defer httpResp.Body.Close()

	// 5xx and 504 (timeout-from-server-side) are fallback triggers.
	if httpResp.StatusCode >= 500 {
		var errResp contract.ErrorResponse
		_ = json.NewDecoder(httpResp.Body).Decode(&errResp)
		return nil, fmt.Errorf("optimizerclient: server status %d code=%s msg=%s",
			httpResp.StatusCode, errResp.Code, errResp.Message)
	}
	// 4xx is non-retryable but we still surface it so the caller logs once.
	if httpResp.StatusCode >= 400 {
		var errResp contract.ErrorResponse
		_ = json.NewDecoder(httpResp.Body).Decode(&errResp)
		return nil, fmt.Errorf("optimizerclient: client status %d code=%s msg=%s",
			httpResp.StatusCode, errResp.Code, errResp.Message)
	}

	var resp contract.SolveResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("optimizerclient: decode response: %w", err)
	}
	if resp.V != contract.V {
		return nil, fmt.Errorf("optimizerclient: response version %q != %q", resp.V, contract.V)
	}
	return mapResponse(resp, in.Orders), nil
}

// recoverySavingsBoost is the savings-rank boost Clarke-Wright adds for
// overflow-bounced orders so they get first dibs on vehicle volume.
const recoverySavingsBoost = 10_000

// buildStops projects backend GeoOrders into wire-format Stops.
func buildStops(orders []dispatch.GeoOrder) []contract.Stop {
	out := make([]contract.Stop, 0, len(orders))
	for _, o := range orders {
		priority := 0
		if o.IsRecovery {
			priority = recoverySavingsBoost
		}
		out = append(out, contract.Stop{
			OrderID:           o.OrderID,
			RetailerID:        o.RetailerID,
			Lat:               o.Lat,
			Lng:               o.Lng,
			VolumeVU:          o.Volume,
			WindowOpen:        o.ReceivingWindowOpen,
			WindowClose:       o.ReceivingWindowClose,
			ServiceMinutes:    defaultServiceMinutes,
			Priority:          priority,
			HandlingClass:     o.HandlingClass,
			RequiresColdChain: o.RequiresColdChain,
			IsHazardous:       o.IsHazardous,
			AccessRestriction: o.AccessRestriction,
		})
	}
	return out
}

// buildVehicles projects AvailableDriver rows into wire-format Vehicles.
// Per-driver StartLat/StartLng win when set; otherwise the warehouse depot.
func buildVehicles(fleet []dispatch.AvailableDriver, depotLat, depotLng float64) []contract.Vehicle {
	out := make([]contract.Vehicle, 0, len(fleet))
	for _, v := range fleet {
		startLat, startLng := depotLat, depotLng
		if v.StartLat != 0 || v.StartLng != 0 {
			startLat, startLng = v.StartLat, v.StartLng
		}
		endLat, endLng := v.EndLat, v.EndLng
		if endLat == 0 && endLng == 0 {
			endLat, endLng = startLat, startLng
		}
		out = append(out, contract.Vehicle{
			VehicleID:        v.VehicleID,
			DriverID:         v.DriverID,
			MaxVolumeVU:      v.MaxVolumeVU,
			StartLat:         startLat,
			StartLng:         startLng,
			EndLat:           endLat,
			EndLng:           endLng,
			AvgSpeedKmph:     defaultAvgSpeedKmph,
			HasRefrigeration: v.HasRefrigeration,
			HazmatCertified:  v.HazmatCertified,
			ShiftStart:       v.ShiftStart,
			ShiftEnd:         v.ShiftEnd,
			MaxRouteMinutes:  v.MaxRouteMinutes,
		})
	}
	return out
}

// buildDistanceMatrix builds the multi-depot node layout matching the Python
// solver: one start node per vehicle, optional distinct end node, then
// customer stops. OSRM /table when available; haversine otherwise.
func buildDistanceMatrix(ctx context.Context, osrm *routing.OSRMClient, vehicles []contract.Vehicle, stops []contract.Stop) [][]int {
	// Strict contract alignment: Exactly 1 node per vehicle (Start), then 1 node per Stop.
	// Distinct End nodes must NOT be interleaved here as they would shift the Stop indices
	// which ai-worker hardcodes to len(Vehicles)+j.
	points := make([]routing.LatLng, 0, len(vehicles)+len(stops))
	for _, v := range vehicles {
		points = append(points, routing.LatLng{Lat: v.StartLat, Lng: v.StartLng})
	}
	for _, s := range stops {
		points = append(points, routing.LatLng{Lat: s.Lat, Lng: s.Lng})
	}
	fallback := routing.HaversineDistanceMatrixM(points)
	if osrm == nil {
		return fallback
	}
	matrix, err := osrm.DistanceMatrix(ctx, points)
	if err != nil || len(matrix) != len(points) {
		return fallback
	}
	return routing.MergeDistanceMatrix(matrix, fallback)
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// mapResponse converts a contract.SolveResponse back into the
// dispatch.AssignmentResult shape the rest of the backend already speaks.
func mapResponse(resp contract.SolveResponse, original []dispatch.GeoOrder) *dispatch.AssignmentResult {
	byOrderID := make(map[string]dispatch.GeoOrder, len(original))
	for _, o := range original {
		byOrderID[o.OrderID] = o
	}

	out := &dispatch.AssignmentResult{
		Routes:   make([]dispatch.DispatchRoute, 0, len(resp.Routes)),
		Orphans:  make([]dispatch.GeoOrder, 0, len(resp.Orphans)),
		Warnings: make([]string, 0),
	}
	for _, r := range resp.Routes {
		route := dispatch.DispatchRoute{
			DriverID:     r.DriverID,
			MaxVolume:    0, // filled below
			LoadedVolume: r.TotalVU,
			Orders:       make([]dispatch.GeoOrder, 0, len(r.Stops)),
		}
		for _, s := range r.Stops {
			if orig, ok := byOrderID[s.OrderID]; ok {
				orig.Assigned = true
				route.Orders = append(route.Orders, orig)
			}
		}
		// Reverse-derive MaxVolume from utilisation if available so the
		// downstream serialiser can render utilPct without re-querying.
		if r.UtilPct > 0 {
			route.MaxVolume = (r.TotalVU / r.UtilPct) * 100
		}
		out.Routes = append(out.Routes, route)
	}
	for _, o := range resp.Orphans {
		if orig, ok := byOrderID[o.OrderID]; ok {
			out.Orphans = append(out.Orphans, orig)
		}
		out.Warnings = append(out.Warnings,
			fmt.Sprintf("orphan %s: %s", o.OrderID, o.Reason))
	}
	out.Warnings = append(out.Warnings,
		fmt.Sprintf("source=%s elapsed_ms=%d util_avg=%.1f%%",
			resp.Source, resp.Stats.ElapsedMs, resp.Stats.AvgUtilisationPct))
	return out
}
