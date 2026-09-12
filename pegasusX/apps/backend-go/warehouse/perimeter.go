package warehouse

import (
	"context"
	"fmt"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func (s *Service) PublishSupplierPerimeter(ctx context.Context, supplierID string) error {
	cov := proximity.CoverageStore{Client: s.spannerClient}
	warehouses, err := cov.ListWarehouses(ctx, supplierID)
	if err != nil {
		return err
	}
	
	cells := proximity.PerimeterCells(warehouses)
	
	key := fmt.Sprintf("perimeter:supplier:%s", supplierID)
	
	pipe := s.redisClient.TxPipeline()
	pipe.Del(ctx, key)
	var args []interface{}
	if len(cells) > 0 {
		for _, cell := range cells {
			args = append(args, cell)
		}
	} else {
		// Cache an empty result using a sentinel value to prevent cache stampedes
		// on suppliers with zero coverage.
		args = append(args, "__empty__")
	}
	pipe.SAdd(ctx, key, args...)
	// Expire to prevent permanent stale data if never republished
	pipe.Expire(ctx, key, 7*24*time.Hour)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Service) CheckSupplierPerimeter(ctx context.Context, supplierID string, cell string) (bool, error) {
	key := fmt.Sprintf("perimeter:supplier:%s", supplierID)
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if exists == 0 {
		// If key missing, default to true or perform lazy initialization
		return true, nil
	}
	return s.redisClient.SIsMember(ctx, key, cell).Result()
}
