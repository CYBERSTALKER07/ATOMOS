// Package plan is the orchestration layer for the Phase 2 dispatch optimiser.
// It calls the VRP solver (apps/ai-worker via dispatch/optimizerclient),
// re-validates the response, and falls back to the Phase 1 H3 + BinPack
// pipeline (dispatch.BinPack) on any solver failure, small batch, or invalid plan.
//
// Lives under dispatch/plan/ rather than inside dispatch/ to avoid an import
// cycle: dispatch ← optimizerclient ← plan.
package plan

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
)

const (
	SourceOptimizer          = "optimizer"
	SourceFallbackPhase1     = "fallback_phase1"
	SourceFallbackValidation = "fallback_validation_rejected"
	SourcePureSmallBatch     = "pure_small_batch"

	defaultCapacityBufferPct  = 5.0
	maxAcceptableUtilFraction = 1.0 - defaultCapacityBufferPct/100.0

	// DenseBatchMinStops: AI preferred at/above this order count.
	DenseBatchMinStops = 12

	// fallbackTimeout bounds H3 cell resolution + binpack CPU during peak dispatch.
	fallbackTimeout = 3 * time.Second

	// solverBudget is aligned with optimizerclient.DefaultTimeout plus wire overhead.
	solverBudget = optimizerclient.DefaultTimeout + 250*time.Millisecond
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
	Score      dispatch.ScoreContext
}

// OptimizeAndValidate runs the VRP optimiser for dense batches, re-validates,
// and degrades gracefully to H3 BinPack. Small batches skip the optimizer call.
func OptimizeAndValidate(ctx context.Context, client *optimizerclient.Client, job Job) (*dispatch.AssignmentResult, string, error) {
	n := len(job.Orders)
	threshold := denseBatchThreshold()
	if client != nil && n >= threshold {
		in := optimizerclient.SolveInput{
			TraceID:    job.TraceID,
			SupplierID: job.SupplierID,
			HomeNodeID: job.HomeNodeID,
			DepotLat:   job.DepotLat,
			DepotLng:   job.DepotLng,
			Orders:     geoOrdersFromDispatchable(job.Orders),
			Fleet:      job.Fleet,
		}
		solveCtx, cancel := context.WithTimeout(ctx, solverBudget)
		defer cancel()
		res, err := client.Solve(solveCtx, in)
		if err == nil {
			if rejected := validateAssignment(res, job.Fleet); rejected != "" {
				out := runFallbackWithDeadline(ctx, job)
				out.Warnings = append(out.Warnings,
					fmt.Sprintf("validation rejected: %s — engaged H3 BinPack fallback", rejected))
				return out, SourceFallbackValidation, nil
			}
			return res, SourceOptimizer, nil
		}
		out := runFallbackWithDeadline(ctx, job)
		out.Warnings = append(out.Warnings, fmt.Sprintf("optimizer error → fallback: %v", err))
		return out, SourceFallbackPhase1, nil
	}
	out := runFallbackWithDeadline(ctx, job)
	if client != nil && n < threshold {
		out.Warnings = append(out.Warnings, fmt.Sprintf("pure_small_batch: %d < %d stops — skipped optimizer", n, threshold))
		return out, SourcePureSmallBatch, nil
	}
	return out, SourceFallbackPhase1, nil
}

func denseBatchThreshold() int {
	raw := strings.TrimSpace(os.Getenv("DISPATCH_AI_MIN_STOPS"))
	if raw == "" {
		return DenseBatchMinStops
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return DenseBatchMinStops
	}
	return n
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

// runFallback executes H3 cell grouping + scored BinPack (+ local search).
func runFallback(job Job) *dispatch.AssignmentResult {
	cellLookup := job.CellLookup
	if cellLookup == nil {
		cellLookup = dispatch.H3CellLookup
	}
	score := job.Score
	score.DepotLat = job.DepotLat
	score.DepotLng = job.DepotLng
	if score.CellLookup == nil {
		score.CellLookup = cellLookup
	}

	// Hierarchical pre-cluster when order count exceeds threshold and flag on.
	if dispatch.HierarchicalEnabled() && len(job.Orders) >= dispatch.HierarchicalOrderThreshold() {
		res := dispatch.BinPackHierarchical(job.Orders, job.Fleet, cellLookup, dispatch.BinPackOptions{Score: score})
		if res == nil {
			return &dispatch.AssignmentResult{}
		}
		return res
	}

	res := dispatch.BinPack(job.Orders, job.Fleet, cellLookup, dispatch.BinPackOptions{Score: score})
	if res == nil {
		return &dispatch.AssignmentResult{}
	}
	return res
}

// runFallbackWithDeadline runs Phase 1 fallback with a strict deadline so H3
// geo-batching cannot block request handlers during regional dispatch peaks.
func runFallbackWithDeadline(ctx context.Context, job Job) *dispatch.AssignmentResult {
	fallbackCtx, cancel := context.WithTimeout(ctx, fallbackTimeout)
	defer cancel()

	type result struct {
		res *dispatch.AssignmentResult
	}
	ch := make(chan result, 1)
	go func() {
		ch <- result{res: runFallback(job)}
	}()

	select {
	case out := <-ch:
		return out.res
	case <-fallbackCtx.Done():
		res := &dispatch.AssignmentResult{}
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("fallback deadline exceeded (%s): %v", fallbackTimeout, fallbackCtx.Err()))
		return res
	}
}

func geoOrdersFromDispatchable(in []dispatch.DispatchableOrder) []dispatch.GeoOrder {
	out := make([]dispatch.GeoOrder, 0, len(in))
	for _, o := range in {
		out = append(out, o.ToGeo())
	}
	return out
}
