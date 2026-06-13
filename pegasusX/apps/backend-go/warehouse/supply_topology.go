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
			FactoryID:    strings.TrimSpace(s.supplierID),
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
