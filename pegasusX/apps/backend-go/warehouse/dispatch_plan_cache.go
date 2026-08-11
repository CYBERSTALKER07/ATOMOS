package warehouse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
)

type dispatchPlanCacheEntry struct {
	Fingerprint       string           `json:"fingerprint"`
	ProposedRoutes    []map[string]any `json:"proposed_routes,omitempty"`
	OptimizerSource   string           `json:"optimizer_source,omitempty"`
	OptimizerWarnings []string         `json:"optimizer_warnings,omitempty"`
	OrphanOrderIDs    []string         `json:"orphan_order_ids,omitempty"`
	ComputedAt        time.Time        `json:"computed_at"`
}

func dispatchPlanCacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("WAREHOUSE_DISPATCH_PLAN_TTL_SEC"))
	if raw == "" {
		return 60 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 5 {
		return 60 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func dispatchPlanCacheKey(warehouseID string) string {
	return "warehouse:dispatch_plan:" + strings.TrimSpace(warehouseID)
}

func (s *Service) invalidateDispatchPlanCache(ctx context.Context, warehouseID string) {
	if s.cache == nil || strings.TrimSpace(warehouseID) == "" {
		return
	}
	s.cache.Invalidate(ctx, dispatchPlanCacheKey(warehouseID))
}

// InvalidateDispatchPlanCache busts the cached smart-dispatch preview and schedules a background re-warm.
func (s *Service) InvalidateDispatchPlanCache(ctx context.Context, warehouseID string) {
	s.invalidateDispatchPlanCache(ctx, warehouseID)
	s.requestDispatchPlanWarm(ctx, warehouseID)
}

func computeDispatchPlanFingerprint(orders []dispatch.DispatchableOrder, fleetCtx fleetDispatchContext, orderFilter []string) string {
	parts := make([]string, 0, len(orders)+len(fleetCtx.TopOff)+1)
	if len(orderFilter) > 0 {
		filtered := append([]string(nil), orderFilter...)
		sort.Strings(filtered)
		parts = append(parts, "orders:"+strings.Join(filtered, ","))
	} else {
		ids := make([]string, 0, len(orders))
		for _, o := range orders {
			ids = append(ids, o.OrderID)
		}
		sort.Strings(ids)
		parts = append(parts, "orders:"+strings.Join(ids, ","))
	}
	capParts := make([]string, 0, len(fleetCtx.TopOff))
	for driverID, cap := range fleetCtx.TopOff {
		capParts = append(capParts, driverID+":"+strconv.FormatFloat(cap.TotalVolumeVU, 'f', 2, 64))
	}
	sort.Strings(capParts)
	parts = append(parts, "topoff:"+strings.Join(capParts, ","))
	inTransit := make([]string, 0, len(fleetCtx.InTransit))
	for id := range fleetCtx.InTransit {
		inTransit = append(inTransit, id)
	}
	sort.Strings(inTransit)
	parts = append(parts, "intransit:"+strings.Join(inTransit, ","))
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:16])
}

func (s *Service) readDispatchPlanCache(ctx context.Context, warehouseID, fingerprint string) (*dispatchPlanCacheEntry, bool) {
	if s.cache == nil {
		return nil, false
	}
	raw, found, err := s.cache.Get(ctx, dispatchPlanCacheKey(warehouseID))
	if err != nil || !found || len(raw) == 0 {
		return nil, false
	}
	var entry dispatchPlanCacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, false
	}
	if entry.Fingerprint != fingerprint {
		return nil, false
	}
	return &entry, true
}

func (s *Service) writeDispatchPlanCache(ctx context.Context, warehouseID string, entry dispatchPlanCacheEntry) {
	if s.cache == nil {
		return
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, dispatchPlanCacheKey(warehouseID), raw, dispatchPlanCacheTTL())
}

func (s *Service) solveDispatchPreview(
	ctx context.Context,
	warehouseID string,
	orders []dispatch.DispatchableOrder,
	fleetCtx fleetDispatchContext,
	solveDrivers []PortalDriver,
	orderFilter []string,
) (map[string]any, string) {
	out := map[string]any{}
	if len(orders) == 0 || len(solveDrivers) == 0 {
		return out, ""
	}

	fullFingerprint := computeDispatchPlanFingerprint(orders, fleetCtx, nil)
	subsetFingerprint := computeDispatchPlanFingerprint(orders, fleetCtx, orderFilter)
	useSubset := len(orderFilter) > 0

	if useSubset {
		if cached, ok := s.readDispatchPlanCache(ctx, warehouseID, fullFingerprint); ok {
			filtered := filterProposedRoutesByOrderIDs(cached.ProposedRoutes, orderFilter)
			return s.planMetaFromCacheEntry(cached, filtered, subsetFingerprint), subsetFingerprint
		}
	}

	fingerprint := subsetFingerprint
	if !useSubset {
		fingerprint = fullFingerprint
	}
	if cached, ok := s.readDispatchPlanCache(ctx, warehouseID, fingerprint); ok {
		routes := cached.ProposedRoutes
		if useSubset {
			routes = filterProposedRoutesByOrderIDs(cached.ProposedRoutes, orderFilter)
		}
		return s.planMetaFromCacheEntry(cached, routes, fingerprint), fingerprint
	}

	driverInputs := buildFleetDriverInputs(solveDrivers, fleetCtx, warehouseID)
	fleet := dispatch.BuildAvailableFleet(driverInputs, nil)
	if len(fleet) == 0 {
		return out, fingerprint
	}
	depot := dispatch.ResolveDepot(ctx, s.spannerClient, warehouseID, dispatch.DepotCoords{
		Lat: s.fallbackDepotLat,
		Lng: s.fallbackDepotLng,
	})
	sid := strings.TrimSpace(s.seedSupplierID)
	job := plan.BuildSolveJob(ctx, sid, warehouseID, depot, orders, fleet)
	solve := plan.RunSolvePreview(ctx, s.optimizerClient, s.planCounters, job)

	proposedRoutes := solve.ProposedRoutes
	if len(proposedRoutes) > 0 {
		routingAttach := s.routeGeometryBuilder
		if routingAttach != nil {
			routing.AttachRouteGeometryToProposedRoutes(ctx, routingAttach, routing.LatLng{
				Lat: depot.Lat,
				Lng: depot.Lng,
			}, proposedRoutes)
		}
		out["proposed_routes"] = proposedRoutes
	}
	if solve.OptimizerSource != "" {
		out["optimizer_source"] = solve.OptimizerSource
	}
	if len(solve.OptimizerWarnings) > 0 {
		out["optimizer_warnings"] = solve.OptimizerWarnings
	}
	computedAt := s.now().UTC()
	cacheFingerprint := fullFingerprint
	s.writeDispatchPlanCache(ctx, warehouseID, dispatchPlanCacheEntry{
		Fingerprint:       cacheFingerprint,
		ProposedRoutes:    proposedRoutes,
		OptimizerSource:   solve.OptimizerSource,
		OptimizerWarnings: solve.OptimizerWarnings,
		ComputedAt:        computedAt,
	})

	responseRoutes := proposedRoutes
	if useSubset {
		responseRoutes = filterProposedRoutesByOrderIDs(proposedRoutes, orderFilter)
	}
	if len(responseRoutes) > 0 {
		out["proposed_routes"] = responseRoutes
	}
	out["plan_fingerprint"] = fingerprint
	out["plan_computed_at"] = computedAt.Format(time.RFC3339Nano)
	out["plan_stale"] = false
	return out, fingerprint
}

func (s *Service) planMetaFromCacheEntry(entry *dispatchPlanCacheEntry, routes []map[string]any, fingerprint string) map[string]any {
	out := map[string]any{}
	if len(routes) > 0 {
		out["proposed_routes"] = routes
	}
	if entry.OptimizerSource != "" {
		out["optimizer_source"] = entry.OptimizerSource
	}
	if len(entry.OptimizerWarnings) > 0 {
		out["optimizer_warnings"] = entry.OptimizerWarnings
	}
	out["plan_fingerprint"] = fingerprint
	out["plan_computed_at"] = entry.ComputedAt.UTC().Format(time.RFC3339Nano)
	out["plan_stale"] = false
	return out
}
