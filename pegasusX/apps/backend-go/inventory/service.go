package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

// Service implements inventory business logic.
type Service struct {
	repo  Repository
	cache *cache.Cache
	log   *slog.Logger
}

// NewService creates an inventory service with the given dependencies.
func NewService(repo Repository, c *cache.Cache, log *slog.Logger) *Service {
	return &Service{repo: repo, cache: c, log: log}
}

// ListByWarehouse returns all inventory levels for a warehouse.
func (s *Service) ListByWarehouse(ctx context.Context, warehouseID string) ([]Level, error) {
	levels, err := s.repo.ListByWarehouse(ctx, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("list inventory by warehouse: %w", err)
	}
	if levels == nil {
		levels = []Level{}
	}
	return levels, nil
}

// ListBySupplier returns all inventory levels across all warehouses for a supplier.
func (s *Service) ListBySupplier(ctx context.Context, supplierID string) ([]Level, error) {
	levels, err := s.repo.ListBySupplier(ctx, supplierID)
	if err != nil {
		return nil, fmt.Errorf("list inventory by supplier: %w", err)
	}
	if levels == nil {
		levels = []Level{}
	}
	return levels, nil
}

// AdjustStock modifies on-hand stock with optimistic concurrency and cache
// invalidation.
func (s *Service) AdjustStock(ctx context.Context, inventoryID string, delta int64, expectedVersion int64) error {
	if err := s.repo.AdjustStock(ctx, inventoryID, delta, expectedVersion); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "inventory:"+inventoryID)
	}
	return nil
}

// ReserveForOrder holds stock for an incoming order.
func (s *Service) ReserveForOrder(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	if err := s.repo.ReserveForOrder(ctx, inventoryID, quantity, expectedVersion); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "inventory:"+inventoryID)
	}
	return nil
}

// ReserveTxn holds stock within an existing Spanner transaction.
func (s *Service) ReserveTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error {
	return s.repo.ReserveTxn(ctx, txn, warehouseID, productID, quantity)
}

// ReleaseReservation returns reserved stock on order cancellation.
func (s *Service) ReleaseReservation(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	if err := s.repo.ReleaseReservation(ctx, inventoryID, quantity, expectedVersion); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "inventory:"+inventoryID)
	}
	return nil
}

// Upsert creates or replaces an inventory level (used by seed/import).
func (s *Service) Upsert(ctx context.Context, l Level) error {
	if err := s.repo.Upsert(ctx, l); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "inventory:"+l.InventoryID)
	}
	return nil
}

// FindByWarehouseProduct resolves an inventory row for import upserts.
func (s *Service) FindByWarehouseProduct(ctx context.Context, warehouseID, productID string) (*Level, error) {
	level, err := s.repo.GetByWarehouseProduct(ctx, warehouseID, productID)
	if err != nil {
		return nil, fmt.Errorf("find inventory by warehouse product: %w", err)
	}
	return level, nil
}

// ReleaseTxn returns reserved stock within an existing Spanner transaction.
func (s *Service) ReleaseTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error {
	return s.repo.ReleaseTxn(ctx, txn, warehouseID, productID, quantity)
}

// CreateHold initiates a soft hold on stock with an expiration timestamp.
func (s *Service) CreateHold(ctx context.Context, hold Hold) error {
	return s.repo.CreateHold(ctx, hold)
}

// ConfirmHold commits a soft hold to confirmed status.
func (s *Service) ConfirmHold(ctx context.Context, holdID string) error {
	return s.repo.ConfirmHold(ctx, holdID)
}

// ReleaseHold cancels a soft hold and returns held inventory.
func (s *Service) ReleaseHold(ctx context.Context, holdID string) error {
	return s.repo.ReleaseHold(ctx, holdID)
}

// SweepExpiredHolds finds expired holds and releases their inventory.
func (s *Service) SweepExpiredHolds(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	now := time.Now().UTC()
	expired, err := s.repo.ListExpiredHolds(ctx, now, 100)
	if err != nil {
		return 0, fmt.Errorf("list expired holds: %w", err)
	}
	count := 0
	for _, h := range expired {
		if err := s.repo.ReleaseHold(ctx, h.HoldID); err != nil {
			if s.log != nil {
				s.log.Warn("release expired hold failed", "hold_id", h.HoldID, "err", err)
			}
			continue
		}
		count++
	}
	return count, nil
}

// StartHoldSweeper starts a background daemon that periodically sweeps expired holds.
func (s *Service) StartHoldSweeper(ctx context.Context, interval time.Duration) {
	if s == nil || s.repo == nil {
		return
	}
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := s.SweepExpiredHolds(ctx); err != nil {
					if s.log != nil {
						s.log.Warn("hold sweeper tick failed", "err", err)
					}
				} else if n > 0 && s.log != nil {
					s.log.Info("swept expired inventory holds", "count", n)
				}
			}
		}
	}()
}

