package plan

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SolvePreview is additive dispatch-preview metadata for portal JSON.
type SolvePreview struct {
	ProposedRoutes    []map[string]any
	OptimizerSource   string
	OptimizerWarnings []string
}

// BuildSolveJob assembles a Job from hydrated dispatch inputs.
func BuildSolveJob(ctx context.Context, supplierID, homeNodeID string, depot dispatch.DepotCoords, orders []dispatch.DispatchableOrder, fleet []dispatch.AvailableDriver) Job {
	return Job{
		TraceID:    outbox.TraceIDFromContext(ctx),
		SupplierID: supplierID,
		HomeNodeID: homeNodeID,
		DepotLat:   depot.Lat,
		DepotLng:   depot.Lng,
		Orders:     orders,
		Fleet:      fleet,
		CellLookup: dispatch.H3CellLookup,
	}
}

// RunSolvePreview executes OptimizeAndValidate when orders and fleet exist.
func RunSolvePreview(
	ctx context.Context,
	client *optimizerclient.Client,
	counters *SourceCounters,
	job Job,
) SolvePreview {
	if len(job.Orders) == 0 || len(job.Fleet) == 0 {
		return SolvePreview{}
	}
	if job.CellLookup == nil {
		job.CellLookup = dispatch.H3CellLookup
	}
	result, source, err := OptimizeAndValidate(ctx, client, job)
	if counters != nil {
		if err != nil {
			counters.RecordError()
		} else {
			counters.Record(source)
		}
	}
	if err != nil {
		RecordPrometheus(SourceFallbackPhase1)
		return SolvePreview{
			OptimizerSource:   SourceFallbackPhase1,
			OptimizerWarnings: []string{err.Error()},
		}
	}
	if result == nil {
		RecordPrometheus(SourceFallbackPhase1)
		return SolvePreview{OptimizerSource: SourceFallbackPhase1}
	}
	RecordPrometheus(source)
	return SolvePreview{
		ProposedRoutes:    RoutesToWire(result),
		OptimizerSource:   source,
		OptimizerWarnings: append([]string(nil), result.Warnings...),
	}
}

// RoutesToWire serialises assignment routes for portal preview JSON.
func RoutesToWire(result *dispatch.AssignmentResult) []map[string]any {
	if result == nil || len(result.Routes) == 0 {
		return []map[string]any{}
	}
	routes := make([]map[string]any, 0, len(result.Routes))
	for _, route := range result.Routes {
		stops := make([]map[string]any, 0, len(route.Orders))
		for _, stop := range route.Orders {
			stops = append(stops, map[string]any{
				"order_id":               stop.OrderID,
				"retailer_id":            stop.RetailerID,
				"retailer_name":          stop.RetailerName,
				"lat":                    stop.Lat,
				"lng":                    stop.Lng,
				"volume_vu":              stop.Volume,
				"receiving_window_open":  stop.ReceivingWindowOpen,
				"receiving_window_close": stop.ReceivingWindowClose,
			})
		}
		orderIDs := make([]string, 0, len(route.Orders))
		for _, stop := range route.Orders {
			if id := stop.OrderID; id != "" {
				orderIDs = append(orderIDs, id)
			}
		}
		utilPct := 0.0
		if route.MaxVolume > 0 {
			utilPct = (route.LoadedVolume / route.MaxVolume) * 100
		}
		routes = append(routes, map[string]any{
			"driver_id":     route.DriverID,
			"loaded_volume": route.LoadedVolume,
			"volume_vu":     route.LoadedVolume,
			"max_volume":    route.MaxVolume,
			"max_volume_vu": route.MaxVolume,
			"util_pct":      utilPct,
			"order_ids":     orderIDs,
			"stops":         stops,
			"stop_count":    len(stops),
		})
	}
	return routes
}
