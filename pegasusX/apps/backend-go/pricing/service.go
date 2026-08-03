package pricing

import (
	"context"
	"time"
)

type Service interface {
	ResolvePrice(ctx context.Context, supplierId, sku string, date time.Time) (int64, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ResolvePrice(ctx context.Context, supplierId, sku string, date time.Time) (int64, error) {
	return s.repo.GetActiveUnitPriceMinor(ctx, supplierId, sku, date)
}
