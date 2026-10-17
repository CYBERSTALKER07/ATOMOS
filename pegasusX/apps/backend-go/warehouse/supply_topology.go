package warehouse

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

type warehouseSupplyContext struct {
	FactoryID    string
	TransferMode string
}

func (s *Service) resolveWarehouseSupplyContext(ctx context.Context, warehouseID string) (warehouseSupplyContext, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return warehouseSupplyContext{}, fmt.Errorf("warehouse_id required")
	}
	if s.spannerClient == nil {
		return warehouseSupplyContext{
			FactoryID:    strings.TrimSpace(s.seedSupplierID),
			TransferMode: supplier.TransferModeTruck,
		}, nil
	}

	row, err := s.spannerClient.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{"SupplierId", "PrimaryFactoryId", "TransferMode", "CoLocateWithFactoryId", "CountryCode", "Lat", "Lng"})
	if err != nil {
		return warehouseSupplyContext{}, err
	}

	var supplierID string
	var primaryFactory, transferMode, coLocate, country spanner.NullString
	var lat, lng spanner.NullFloat64
	if err := row.Columns(&supplierID, &primaryFactory, &transferMode, &coLocate, &country, &lat, &lng); err != nil {
		return warehouseSupplyContext{}, err
	}

	mode := supplier.NormalizeTransferMode(transferMode.StringVal)
	factoryID := ""
	if mode == supplier.TransferModeInternal {
		if co := strings.TrimSpace(coLocate.StringVal); co != "" {
			factoryID = co
		}
	}
	if factoryID == "" {
		factoryID, err = s.engineFactoryID(ctx, supplierID, warehouseID, country.StringVal, lat.Float64, lng.Float64, primaryFactory.StringVal)
		if err != nil {
			return warehouseSupplyContext{}, err
		}
	}
	if factoryID == "" {
		return warehouseSupplyContext{}, proximity.ErrFactoryUnassigned
	}

	return warehouseSupplyContext{
		FactoryID:    factoryID,
		TransferMode: mode,
	}, nil
}

func (s *Service) engineFactoryID(
	ctx context.Context,
	supplierID, warehouseID, country string,
	lat, lng float64,
	primaryFactoryID string,
) (string, error) {
	if s.spannerClient == nil {
		return "", proximity.ErrFactoryUnassigned
	}
	store := &proximity.CoverageStore{Client: s.spannerClient}
	factories, err := store.ListFactories(ctx, supplierID)
	if err != nil {
		return "", err
	}
	lanes, err := store.ListSupplyLanes(ctx, supplierID, warehouseID)
	if err != nil {
		return "", err
	}
	return proximity.ResolveSupplyFactory(country, lat, lng, primaryFactoryID, lanes, factories)
}

// validateSupplyCycle checks if setting targetID as the primary factory (or supply source)
// for sourceID would create a cycle in the supply chain topology.
func (s *Service) validateSupplyCycle(ctx context.Context, sourceID, targetID string) error {
	if sourceID == "" || targetID == "" {
		return nil // nothing to validate
	}
	if sourceID == targetID {
		return errors.New("cyclic_supply_chain: warehouse cannot supply itself")
	}
	if s.spannerClient == nil {
		return nil
	}

	current := targetID
	// Max depth of 20 to prevent unbounded loops in case of corrupt data.
	for i := 0; i < 20; i++ {
		row, err := s.spannerClient.Single().ReadRow(ctx, "Warehouses", spanner.Key{current}, []string{"PrimaryFactoryId", "CoLocateWithFactoryId"})
		if err != nil {
			if errors.Is(err, spanner.ErrRowNotFound) || strings.Contains(err.Error(), "not found") {
				// Target is likely a real Factory (or missing), which is a sink. No cycle.
				return nil
			}
			return err
		}

		var primaryFactory, coLocate spanner.NullString
		if err := row.Columns(&primaryFactory, &coLocate); err != nil {
			return err
		}

		next := strings.TrimSpace(primaryFactory.StringVal)
		if co := strings.TrimSpace(coLocate.StringVal); co != "" {
			next = co
		}

		if next == "" {
			return nil // reached a sink
		}
		if next == sourceID {
			return errors.New("cyclic_supply_chain: transfer chain cycle detected")
		}
		current = next
	}

	return errors.New("cyclic_supply_chain: maximum depth exceeded")
}
