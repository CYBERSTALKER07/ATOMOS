package warehouse

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// DispatchPlanWarmerConfig tunes the background dispatch plan pre-compute loop.
type DispatchPlanWarmerConfig struct {
	Interval time.Duration
}

// StartDispatchPlanWarmer periodically re-warms dispatch plans for active warehouses.
func StartDispatchPlanWarmer(ctx context.Context, svc *Service, cfg DispatchPlanWarmerConfig) {
	if svc == nil || svc.spannerClient == nil {
		return
	}
	if cfg.Interval <= 0 {
		cfg.Interval = dispatchPlanWarmerIntervalFromEnv()
	}
	svc.log.Info("dispatch plan warmer started", "interval", cfg.Interval.String())

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			svc.log.Info("dispatch plan warmer stopped")
			return
		case <-ticker.C:
			svc.runDispatchPlanWarmerTick(ctx)
		}
	}
}

func (s *Service) runDispatchPlanWarmerTick(parent context.Context) {
	ctx := outbox.WithTraceID(parent, newWorkerTraceID())
	warehouseIDs, err := s.listWarehousesWithDispatchableOrders(ctx)
	if err != nil {
		s.log.WarnContext(ctx, "dispatch plan warmer warehouse list failed", "err", err)
		return
	}
	for _, warehouseID := range warehouseIDs {
		s.requestDispatchPlanWarm(ctx, warehouseID)
	}
}

func (s *Service) listWarehousesWithDispatchableOrders(ctx context.Context) ([]string, error) {
	sid := strings.TrimSpace(s.supplierID)
	if sid == "" || s.spannerClient == nil {
		return nil, nil
	}
	repo := dispatch.NewRepository(s.spannerClient)
	rows, err := repo.FetchDispatchable(ctx, dispatch.FetchParams{
		SupplierID: sid,
		Limit:      500,
	})
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, row := range rows {
		wh := strings.TrimSpace(row.WarehouseID)
		if wh == "" {
			continue
		}
		if _, ok := seen[wh]; ok {
			continue
		}
		seen[wh] = struct{}{}
		out = append(out, wh)
	}
	return out, nil
}

func (s *Service) requestDispatchPlanWarm(ctx context.Context, warehouseID string) {
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" || s.spannerClient == nil {
		return
	}
	if s.cache != nil {
		key := "warehouse:dispatch_plan_warm:" + warehouseID
		if _, found, _ := s.cache.Get(ctx, key); found {
			return
		}
		_ = s.cache.Set(ctx, key, []byte("1"), dispatchPlanWarmDebounce())
	}
	go func() {
		warmCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.warmDispatchPlan(warmCtx, warehouseID); err != nil {
			s.log.WarnContext(warmCtx, "dispatch plan warm failed",
				"warehouse_id", warehouseID,
				"err", err,
			)
		}
	}()
}

func (s *Service) warmDispatchPlan(ctx context.Context, warehouseID string) error {
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" || s.spannerClient == nil {
		return nil
	}
	sid := s.resolveDispatchSupplierID(ctx, warehouseID)
	if sid == "" {
		return nil
	}

	repo := dispatch.NewRepository(s.spannerClient)
	rows, err := repo.FetchDispatchable(ctx, dispatch.FetchParams{
		SupplierID:  sid,
		WarehouseID: warehouseID,
		Limit:       500,
	})
	if err != nil {
		return err
	}

	solveDrivers, err := s.loadDispatchDrivers(ctx, warehouseID)
	if err != nil {
		return err
	}
	if len(rows) == 0 || len(solveDrivers) == 0 {
		s.invalidateDispatchPlanCache(ctx, warehouseID)
		return nil
	}

	fleetCtx, err := s.loadFleetDispatchContext(ctx, sid, warehouseID, collectWarehouseDriverIDs(solveDrivers))
	if err != nil {
		return err
	}

	planMeta, _ := s.solveDispatchPreview(ctx, warehouseID, rows, fleetCtx, solveDrivers, nil)
	s.broadcastWarehouseEvent(ctx, warehouseID, map[string]any{
		"type":              "DISPATCH_PLAN_UPDATED",
		"warehouse_id":      warehouseID,
		"order_count":       len(rows),
		"route_count":       lenRoutes(planMeta["proposed_routes"]),
		"plan_fingerprint":  planMeta["plan_fingerprint"],
		"optimizer_source":  planMeta["optimizer_source"],
		"timestamp":         s.now().UTC().Format(time.RFC3339Nano),
	})
	return nil
}

func lenRoutes(raw any) int {
	routes, ok := raw.([]map[string]any)
	if !ok {
		return 0
	}
	return len(routes)
}

func dispatchPlanWarmDebounce() time.Duration {
	raw := strings.TrimSpace(os.Getenv("WAREHOUSE_DISPATCH_PLAN_WARM_DEBOUNCE_SEC"))
	if raw == "" {
		return 8 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 2 {
		return 8 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func dispatchPlanWarmerIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("WAREHOUSE_DISPATCH_PLAN_WARMER_INTERVAL_SEC"))
	if raw == "" {
		return 45 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 15 {
		return 45 * time.Second
	}
	return time.Duration(seconds) * time.Second
}
