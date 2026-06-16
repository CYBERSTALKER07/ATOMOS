package warehouse

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// AutoDispatchWarehouse is one warehouse node opted into background smart dispatch.
type AutoDispatchWarehouse struct {
	WarehouseID string
	SupplierID  string
}

// AutoDispatchWorkerConfig tunes the warehouse auto-dispatch background loop.
type AutoDispatchWorkerConfig struct {
	Interval time.Duration
}

// StartAutoDispatchWorker ticks warehouses with AutoDispatchEnabled and commits optimizer routes.
func StartAutoDispatchWorker(ctx context.Context, svc *Service, cfg AutoDispatchWorkerConfig) {
	if svc == nil || svc.spannerClient == nil {
		return
	}
	if cfg.Interval <= 0 {
		cfg.Interval = autoDispatchIntervalFromEnv()
	}
	svc.log.Info("warehouse auto-dispatch worker started", "interval", cfg.Interval.String())

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			svc.log.Info("warehouse auto-dispatch worker stopped")
			return
		case <-ticker.C:
			svc.runAutoDispatchTick(ctx, cfg.Interval)
		}
	}
}

func (s *Service) runAutoDispatchTick(parent context.Context, debounce time.Duration) {
	ctx := outbox.WithTraceID(parent, newWorkerTraceID())
	warehouses, err := s.repo.ListAutoDispatchWarehouses(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "auto-dispatch warehouse list failed", "err", err)
		return
	}
	for _, wh := range warehouses {
		if err := s.runAutoDispatchForWarehouse(ctx, wh, debounce); err != nil {
			s.log.WarnContext(ctx, "auto-dispatch tick skipped",
				"warehouse_id", wh.WarehouseID,
				"supplier_id", wh.SupplierID,
				"err", err,
			)
		}
	}
}

func (s *Service) runAutoDispatchForWarehouse(ctx context.Context, wh AutoDispatchWarehouse, debounce time.Duration) error {
	warehouseID := strings.TrimSpace(wh.WarehouseID)
	supplierID := strings.TrimSpace(wh.SupplierID)
	if warehouseID == "" || supplierID == "" {
		return fmt.Errorf("invalid warehouse scope")
	}
	if s.cache != nil && debounce > 0 {
		key := "warehouse:auto_dispatch:" + warehouseID
		if _, found, _ := s.cache.Get(ctx, key); found {
			return nil
		}
		if err := s.cache.Set(ctx, key, []byte("1"), debounce); err != nil {
			s.log.WarnContext(ctx, "auto-dispatch debounce set failed", "warehouse_id", warehouseID, "err", err)
		}
	}

	result, err := s.ExecuteDispatch(ctx, DispatchExecuteRequest{
		WarehouseID:   warehouseID,
		SupplierID:    supplierID,
		Mode:          "AUTO",
		AcceptPartial: true,
	})
	if err != nil {
		return err
	}
	if result.Status != "dispatched" {
		return nil
	}

	s.log.InfoContext(ctx, "auto-dispatch committed",
		"warehouse_id", warehouseID,
		"supplier_id", supplierID,
		"manifests_created", result.ManifestsCreated,
		"orders_assigned", result.OrdersAssigned,
		"optimizer_source", result.OptimizerSource,
	)
	s.broadcastWarehouseEvent(ctx, warehouseID, map[string]any{
		"type":              "DISPATCH_COMMITTED",
		"warehouse_id":      warehouseID,
		"manifests_created": result.ManifestsCreated,
		"orders_assigned":   result.OrdersAssigned,
		"optimizer_source":  result.OptimizerSource,
		"auto_dispatch":     true,
		"timestamp":         s.now().UTC().Format(time.RFC3339Nano),
	})
	return nil
}

func autoDispatchIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("WAREHOUSE_AUTO_DISPATCH_INTERVAL_SEC"))
	if raw == "" {
		return 60 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 15 {
		return 60 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func newWorkerTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "warehouse-auto-dispatch"
	}
	return hex.EncodeToString(buf)
}
