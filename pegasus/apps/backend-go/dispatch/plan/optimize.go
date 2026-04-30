// Package plan is the orchestration layer for the Phase 2 dispatch optimiser.
// It calls the VRP solver (apps/ai-worker via dispatch/optimizerclient),
// re-validates the response, and falls back to the Phase 1 KMeans + binpack
// pipeline (dispatch.BinPack) on any solver failure or invalid plan.
//
// Lives under dispatch/plan/ rather than inside dispatch/ to avoid an import
// cycle: dispatch ← optimizerclient ← plan.
package plan

import (
	"context"
	"fmt"
	"math"

	"backend-go/dispatch"
	"backend-go/dispatch/optimizerclient"
)

const (
	SourceOptimizer          = "optimizer"
	SourceFallbackPhase1     = "fallback_phase1"
	SourceFallbackValidation = "fallback_validation_rejected"

	defaultCapacityBufferPct  = 5.0
	maxAcceptableUtilFraction = 1.0 - defaultCapacityBufferPct/100.0
)

// Job is the backend-domain input. The orchestrator builds an
// optimizerclient.SolveInput from it on the way out.
type Job struct {
	TraceID    string
	SupplierID string
	HomeNodeID string
	DepotLat   float64
	DepotLng   float64
	Orders     []dispatch.DispatchableOrder
	Fleet      []dispatch.AvailableDriver
	CellLookup func(lat, lng float64) string
}

// OptimizeAndValidate runs the VRP optimiser, re-validates the result, and
// degrades gracefully to the Phase 1 fallback. Returns the final plan and
// the source attribution.
func OptimizeAndValidate(ctx context.Context, client *optimizerclient.Client, job Job) (*dispatch.AssignmentResult, string, error) {
	if client != nil {
		in := optimizerclient.SolveInput{
			TraceID:    job.TraceID,
			SupplierID: job.SupplierID,
			HomeNodeID: job.HomeNodeID,
			DepotLat:   job.DepotLat,
			DepotLng:   job.DepotLng,
			Orders:     geoOrdersFromDispatchable(job.Orders),
			Fleet:      job.Fleet,
		}
		res, err := client.Solve(ctx, in)
		if err == nil {
			if rejected := validateAssignment(res, job.Fleet); rejected != "" {
				out := runFallback(job)
				out.Warnings = append(out.Warnings,
					fmt.Sprintf("validation rejected: %s — engaged Phase 1 fallback", rejected))
				return out, SourceFallbackValidation, nil
			}
			return res, SourceOptimizer, nil
		}
		out := runFallback(job)
		out.Warnings = append(out.Warnings, fmt.Sprintf("optimizer error → fallback: %v", err))
		return out, SourceFallbackPhase1, nil
	}
	return runFallback(job), SourceFallbackPhase1, nil
}

// validateAssignment returns "" when every route fits within the configured
// capacity buffer. On violation it returns a single-line reason.
func validateAssignment(res *dispatch.AssignmentResult, fleet []dispatch.AvailableDriver) string {
	if res == nil {
		return "nil result"
	}
	capByDriver := make(map[string]float64, len(fleet))
	for _, d := range fleet {
		capByDriver[d.DriverID] = d.MaxVolumeVU
	}
	for ri, r := range res.Routes {
		var sum float64
		for _, o := range r.Orders {
			sum += o.Volume
		}
		maxVU, ok := capByDriver[r.DriverID]
		if !ok || maxVU <= 0 {
			return fmt.Sprintf("route %d: unknown driver %s", ri, r.DriverID)
		}
		if sum > maxVU*maxAcceptableUtilFraction+1e-6 {
			return fmt.Sprintf("route %d driver=%s sum=%.2f cap=%.2f buffer=%.0f%%",
				ri, r.DriverID, sum, maxVU, defaultCapacityBufferPct)
		}
		if math.Abs(r.LoadedVolume-sum) > 1e-3 {
			res.Routes[ri].LoadedVolume = sum
		}
	}
	return ""
}

// runFallback executes the existing Phase 1 KMeansCluster + BinPack pipeline.
func runFallback(job Job) *dispatch.AssignmentResult {
	cellLookup := job.CellLookup
	if cellLookup == nil {
		cellLookup = func(lat, lng float64) string {
			return fmt.Sprintf("%.4f,%.4f", lat, lng)
		}
	}
	res := dispatch.BinPack(job.Orders, job.Fleet, cellLookup)
	if res == nil {
		return &dispatch.AssignmentResult{}
	}
	return res
}

func geoOrdersFromDispatchable(in []dispatch.DispatchableOrder) []dispatch.GeoOrder {
	out := make([]dispatch.GeoOrder, 0, len(in))
	for _, o := range in {
		out = append(out, o.ToGeo())
	}
	return out
}
