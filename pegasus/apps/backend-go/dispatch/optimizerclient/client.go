// Package optimizerclient is the backend-side HTTP client for the Phase 2
// dispatch optimiser hosted in apps/ai-worker. It is the only call site for
// `POST /v1/optimizer/solve` and the single source of truth for the 2.5 s
// timeout, header convention, and graceful-fallback contract.
//
// On any of: network error, HTTP 5xx, contract.ErrCodeTimeout, malformed JSON,
// or a Go context deadline, Solve returns (nil, error). The caller is
// expected to fall back to the legacy K-Means + binpack pipeline in
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

	contract "optimizercontract"

	"backend-go/dispatch"
)

// Default solver budget. Anything above ~2.5 s starves request handlers; the
// ai-worker side has its own internal cap at 2 s so the overall ceiling is
// honoured even on cold-start latency.
const DefaultTimeout = 2500 * time.Millisecond

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
}

// New returns a Client. endpoint must include the scheme + host, e.g.
// "http://ai-worker:8081". The path contract.SolvePath is appended internally.
func New(endpoint, apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		endpoint:   endpoint,
		apiKey:     apiKey,
	}
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

	req := contract.SolveRequest{
		V:             contract.V,
		TraceID:       in.TraceID,
		SupplierID:    in.SupplierID,
		HomeNodeID:    in.HomeNodeID,
		DepartureTime: in.DepartureTime.UTC().Format(time.RFC3339),
		Stops:         buildStops(in.Orders),
		Vehicles:      buildVehicles(in.Fleet, in.DepotLat, in.DepotLng),
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

// buildStops projects backend GeoOrders into wire-format Stops. Any field the
// solver treats as "no constraint" is left at its zero value (empty string for
// HH:MM, zero for ServiceMinutes which the solver fills in).
func buildStops(orders []dispatch.GeoOrder) []contract.Stop {
	out := make([]contract.Stop, 0, len(orders))
	for _, o := range orders {
		priority := 0
		if o.IsRecovery {
			priority = recoverySavingsBoost
		}
		out = append(out, contract.Stop{
			OrderID:        o.OrderID,
			RetailerID:     o.RetailerID,
			Lat:            o.Lat,
			Lng:            o.Lng,
			VolumeVU:       o.Volume,
			WindowOpen:     o.ReceivingWindowOpen,
			WindowClose:    o.ReceivingWindowClose,
			ServiceMinutes: defaultServiceMinutes,
			Priority:       priority,
		})
	}
	return out
}

// buildVehicles projects AvailableDriver rows into wire-format Vehicles.
// All vehicles share the depot start coordinate — fleet units depart from
// the same home node (warehouse or factory).
func buildVehicles(fleet []dispatch.AvailableDriver, depotLat, depotLng float64) []contract.Vehicle {
	out := make([]contract.Vehicle, 0, len(fleet))
	for _, v := range fleet {
		out = append(out, contract.Vehicle{
			VehicleID:    v.VehicleID,
			DriverID:     v.DriverID,
			MaxVolumeVU:  v.MaxVolumeVU,
			StartLat:     depotLat,
			StartLng:     depotLng,
			AvgSpeedKmph: defaultAvgSpeedKmph,
		})
	}
	return out
}

// mapResponse converts a contract.SolveResponse back into the
// dispatch.AssignmentResult shape the rest of the backend already speaks.
// The reverse-lookup from OrderID → original GeoOrder preserves invoice
// totals, retailer names, and any operator overrides we stripped on the
// outbound projection.
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
