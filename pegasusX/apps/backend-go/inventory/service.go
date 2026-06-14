package inventory

import (
	"context"
	"fmt"
	"log/slog"

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
