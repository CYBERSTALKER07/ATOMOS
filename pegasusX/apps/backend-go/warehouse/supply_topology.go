package warehouse

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
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
		[]string{"PrimaryFactoryId", "TransferMode", "CoLocateWithFactoryId"})
	if err != nil {
		return warehouseSupplyContext{}, err
	}

	var primaryFactory, transferMode, coLocate spanner.NullString
	if err := row.Columns(&primaryFactory, &transferMode, &coLocate); err != nil {
		return warehouseSupplyContext{}, err
	}

	mode := supplier.NormalizeTransferMode(transferMode.StringVal)
	factoryID := strings.TrimSpace(primaryFactory.StringVal)
	if mode == supplier.TransferModeInternal {
		if co := strings.TrimSpace(coLocate.StringVal); co != "" {
			factoryID = co
		}
	}
	if factoryID == "" {
		return warehouseSupplyContext{}, errors.New("primary factory missing")
	}

	return warehouseSupplyContext{
		FactoryID:    factoryID,
		TransferMode: mode,
	}, nil
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
