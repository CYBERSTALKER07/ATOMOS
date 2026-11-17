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
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
)

const (
	SourceOptimizer          = "optimizer"
	SourceFallbackPhase1     = "fallback_phase1"
	SourceFallbackValidation = "fallback_validation_rejected"
	SourcePureSmallBatch     = "pure_small_batch"

	// Product honesty classes (G4.C) — additive to raw optimizer_source.
	ClassOptimal   = "OPTIMAL"   // OR-Tools / optimizer-core accepted
	ClassHeuristic = "HEURISTIC" // phase-1 binpack / fallback / validation reject

	defaultCapacityBufferPct  = 5.0
	maxAcceptableUtilFraction = 1.0 - defaultCapacityBufferPct/100.0

	// DenseBatchMinStops: AI preferred at/above this order count.
	DenseBatchMinStops = 12

	// fallbackTimeout bounds H3 cell resolution + binpack CPU during peak dispatch.
	fallbackTimeout = 10 * time.Second

	// solverBudget bounds parallel subproblem wave completion within the <= 60s budget.
	solverBudget = 55 * time.Second

	// defaultPartitionWorkers caps concurrent HTTP calls to optimizer-core across subproblems.
	defaultPartitionWorkers = 8
)

// OptimizerClass maps raw optimizer_source to HEURISTIC|OPTIMAL product labels.
func OptimizerClass(source string) string {
	switch strings.TrimSpace(source) {
	case SourceOptimizer:
		return ClassOptimal
	case "", SourceFallbackPhase1, SourceFallbackValidation, SourcePureSmallBatch, "manual":
		return ClassHeuristic
	default:
		// Unknown sources fail-honest as HEURISTIC (never claim OPTIMAL without OR-Tools).
		return ClassHeuristic
	}
}

// Job is the backend-domain input. The orchestrator builds an
// optimizerclient.SolveInput from it on the way out.
type Job struct {
	TraceID              string
	SupplierID           string
	HomeNodeID           string
	DepotLat             float64
	DepotLng             float64
	Orders               []dispatch.DispatchableOrder
	Fleet                []dispatch.AvailableDriver
	CellLookup           func(lat, lng float64) string
	Score                dispatch.ScoreContext
	MaxStopsPerRoute     int
	MaxStopsPerPartition int
	TimeLimitMs          int
}

// OptimizeAndValidate runs the VRP optimiser for dense batches, re-validates,
// and degrades gracefully to balanced spatial fallback. Small batches skip the optimizer call.
// Batches larger than the partition target (default 200, max 250 stops) are split into
// depot/capacity-balanced spatial subproblems solved concurrently by optimizerclient workers.
func OptimizeAndValidate(ctx context.Context, client *optimizerclient.Client, job Job) (*dispatch.AssignmentResult, string, error) {
	n := len(job.Orders)
	threshold := denseBatchThreshold()
	maxRouteStops := dispatch.EffectiveMaxStopsPerRoute(job.MaxStopsPerRoute, n, len(job.Fleet))
	partitionLimit := resolvePartitionLimit(job.MaxStopsPerPartition)

	if client != nil && n >= threshold {
		if n > partitionLimit {
			return solvePartitionedWave(ctx, client, job, maxRouteStops, partitionLimit)
		}
		in := optimizerclient.SolveInput{
			TraceID:          job.TraceID,
			SupplierID:       job.SupplierID,
			HomeNodeID:       job.HomeNodeID,
			DepotLat:         job.DepotLat,
			DepotLng:         job.DepotLng,
			Orders:           geoOrdersFromDispatchable(job.Orders),
			Fleet:            job.Fleet,
			TetrisBuffer:     maxAcceptableUtilFraction,
			MaxStopsPerRoute: maxRouteStops,
			TimeLimitMs:      job.TimeLimitMs,
		}
		solveCtx, cancel := context.WithTimeout(ctx, solverBudget)
		defer cancel()
		res, err := client.Solve(solveCtx, in)
		if err == nil {
			if res == nil || (len(res.Routes) == 0 && len(job.Orders) > 0) {
				out := runFallbackWithDeadline(ctx, job)
				out.Warnings = append(out.Warnings,
					"optimizer returned zero routes — engaged spatial BinPack fallback")
				return out, SourceFallbackValidation, nil
			}
			if rejected := validateAssignmentWithCap(res, job.Fleet, maxRouteStops); rejected != "" {
				out := runFallbackWithDeadline(ctx, job)
				out.Warnings = append(out.Warnings,
					fmt.Sprintf("validation rejected: %s — engaged spatial BinPack fallback", rejected))
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

func resolvePartitionLimit(explicit int) int {
	if explicit > 0 && explicit <= dispatch.MaxPartitionStops {
		return explicit
	}
	return dispatch.DefaultPartitionTargetStops
}

func solvePartitionedWave(ctx context.Context, client *optimizerclient.Client, job Job, maxRouteStops, partitionLimit int) (*dispatch.AssignmentResult, string, error) {
	partPlan := dispatch.BuildSpatialSubproblems(job.Orders, job.Fleet, dispatch.SpatialPartitionOptions{
		DepotLat:             job.DepotLat,
		DepotLng:             job.DepotLng,
		MaxStopsPerPartition: partitionLimit,
		MaxStopsPerRoute:     maxRouteStops,
		TetrisBuffer:         maxAcceptableUtilFraction,
	})
	if len(partPlan.Subproblems) == 0 {
		out := &dispatch.AssignmentResult{
			Routes:           []dispatch.DispatchRoute{},
			Splits:           []dispatch.SplitOrder{},
			Orphans:          partPlan.DeferredOverflow,
			OverflowWarnings: partPlan.OverflowWarnings,
			Warnings:         append([]string(nil), partPlan.Warnings...),
		}
		return out, SourceFallbackPhase1, nil
	}

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

	waveCtx, cancel := context.WithTimeout(ctx, solverBudget)
	defer cancel()

	subResults := make([]*dispatch.AssignmentResult, len(partPlan.Subproblems))
	type partOutcome struct {
		err      error
		rejected string
	}
	outcomes := make([]partOutcome, len(partPlan.Subproblems))

	workers := defaultPartitionWorkers
	if len(partPlan.Subproblems) < workers {
		workers = len(partPlan.Subproblems)
	}
	jobs := make(chan int, len(partPlan.Subproblems))
	for i := range partPlan.Subproblems {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				sub := partPlan.Subproblems[idx]
				subCtx, subCancel := context.WithTimeout(waveCtx, optimizerclient.DefaultTimeout)
				res, err := client.Solve(subCtx, optimizerclient.SolveInput{
					TraceID:          fmt.Sprintf("%s-p%d", job.TraceID, idx),
					SupplierID:       job.SupplierID,
					HomeNodeID:       job.HomeNodeID,
					DepotLat:         sub.DepotLat,
					DepotLng:         sub.DepotLng,
					Orders:           geoOrdersFromDispatchable(sub.Orders),
					Fleet:            sub.Fleet,
					TetrisBuffer:     maxAcceptableUtilFraction,
					MaxStopsPerRoute: maxRouteStops,
					TimeLimitMs:      job.TimeLimitMs,
				})
				subCancel()
				if err != nil {
					outcomes[idx] = partOutcome{err: err}
					subResults[idx] = dispatch.BinPack(sub.Orders, sub.Fleet, cellLookup, dispatch.BinPackOptions{
						Score:            score,
						MaxStopsPerRoute: maxRouteStops,
					})
					continue
				}
				if res == nil || (len(res.Routes) == 0 && len(sub.Orders) > 0) {
					outcomes[idx] = partOutcome{rejected: "empty routes"}
					subResults[idx] = dispatch.BinPack(sub.Orders, sub.Fleet, cellLookup, dispatch.BinPackOptions{
						Score:            score,
						MaxStopsPerRoute: maxRouteStops,
					})
					continue
				}
				if reason := validateAssignmentWithCap(res, sub.Fleet, maxRouteStops); reason != "" {
					outcomes[idx] = partOutcome{rejected: reason}
					subResults[idx] = dispatch.BinPack(sub.Orders, sub.Fleet, cellLookup, dispatch.BinPackOptions{
						Score:            score,
						MaxStopsPerRoute: maxRouteStops,
					})
					continue
				}
				subResults[idx] = res
			}
		}()
	}
	wg.Wait()

	merged := dispatch.MergePartitionResults(subResults, partPlan.DeferredOverflow)
	merged.OverflowWarnings = append(merged.OverflowWarnings, partPlan.OverflowWarnings...)
	merged.Warnings = append(merged.Warnings, partPlan.Warnings...)

	anyErr := false
	anyRejected := false
	for i, o := range outcomes {
		if o.err != nil {
			anyErr = true
			merged.Warnings = append(merged.Warnings, fmt.Sprintf("subproblem_%d_fallback_err: %v", i, o.err))
		} else if o.rejected != "" {
			anyRejected = true
			merged.Warnings = append(merged.Warnings, fmt.Sprintf("subproblem_%d_fallback_validation: %s", i, o.rejected))
		}
	}

	if finalReason := validateAssignmentWithCap(merged, job.Fleet, maxRouteStops); finalReason != "" {
		out := runFallbackWithDeadline(ctx, job)
		out.Warnings = append(out.Warnings, fmt.Sprintf("merged validation rejected: %s", finalReason))
		return out, SourceFallbackValidation, nil
	}
	if anyRejected {
		return merged, SourceFallbackValidation, nil
	}
	if anyErr {
		return merged, SourceFallbackPhase1, nil
	}
	return merged, SourceOptimizer, nil
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

// validateAssignment returns "" when every route fits within the configured capacity buffer.
func validateAssignment(res *dispatch.AssignmentResult, fleet []dispatch.AvailableDriver) string {
	return validateAssignmentWithCap(res, fleet, 0)
}

// validateAssignmentWithCap returns "" when every route fits within the configured
// capacity buffer and explicit MaxStopsPerRoute cap. On violation it returns a single-line reason.
func validateAssignmentWithCap(res *dispatch.AssignmentResult, fleet []dispatch.AvailableDriver, maxStopsPerRoute int) string {
	if res == nil {
		return "nil result"
	}
	capByDriver := make(map[string]float64, len(fleet))
	for _, d := range fleet {
		capByDriver[d.DriverID] = d.MaxVolumeVU
	}
	for ri, r := range res.Routes {
		if maxStopsPerRoute > 0 && len(r.Orders) > maxStopsPerRoute {
			return fmt.Sprintf("route %d driver=%s stops=%d exceeds max_stops_per_route=%d",
				ri, r.DriverID, len(r.Orders), maxStopsPerRoute)
		}
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

// runFallback executes spatial balanced partitioning (for large batches) or H3 BinPack.
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
	maxRouteStops := dispatch.EffectiveMaxStopsPerRoute(job.MaxStopsPerRoute, len(job.Orders), len(job.Fleet))

	// Partition large batches across spatial subproblems when order count exceeds single-subproblem size.
	if len(job.Orders) > resolvePartitionLimit(job.MaxStopsPerPartition) ||
		(dispatch.HierarchicalEnabled() && len(job.Orders) >= dispatch.HierarchicalOrderThreshold()) {
		res := dispatch.BinPackHierarchical(job.Orders, job.Fleet, cellLookup, dispatch.BinPackOptions{
			Score:            score,
			MaxStopsPerRoute: maxRouteStops,
		})
		if res == nil {
			return &dispatch.AssignmentResult{}
		}
		return res
	}

	res := dispatch.BinPack(job.Orders, job.Fleet, cellLookup, dispatch.BinPackOptions{
		Score:            score,
		MaxStopsPerRoute: maxRouteStops,
	})
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
		defer func() {
			if r := recover(); r != nil {
				ch <- result{res: &dispatch.AssignmentResult{
					Warnings: []string{fmt.Sprintf("fallback panicked: %v", r)},
				}}
			}
		}()
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
