package segment

import (
	"context"
	"fmt"
	"time"
)

// Service resolves segmentation and policies for allocation.
type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// GetRetailerSegment returns the retailer segment letter (defaults to C).
func (s *Service) GetRetailerSegment(ctx context.Context, retailerID string) (string, error) {
	if s == nil || s.repo == nil {
		return SegmentC, nil
	}
	seg, err := s.repo.GetRetailerSegment(ctx, retailerID)
	if err != nil {
		return SegmentC, err
	}
	return seg, nil
}

// ResolveLineContext resolves allocation context for one line.
func (s *Service) ResolveLineContext(ctx context.Context, supplierID, retailerID, sku string) (LineAllocationContext, error) {
	if s == nil || s.repo == nil {
		return LineAllocationContext{}, fmt.Errorf("segment service unavailable")
	}
	segment, err := s.repo.GetRetailerSegment(ctx, retailerID)
	if err != nil {
		return LineAllocationContext{}, err
	}
	skuClass, err := s.repo.GetSkuClass(ctx, supplierID, sku)
	if err != nil {
		return LineAllocationContext{}, err
	}
	riskTier, err := s.repo.GetRiskTier(ctx, retailerID)
	if err != nil {
		return LineAllocationContext{}, err
	}
	policy, err := s.repo.ResolvePolicy(ctx, supplierID, segment, skuClass.VelocityClass)
	if err != nil {
		return LineAllocationContext{}, err
	}
	score := ComputePriorityScore(policy, riskTier, skuClass.StrategicFlag)
	return LineAllocationContext{
		RetailerSegment: segment,
		SkuClass:        skuClass.VelocityClass,
		RiskTier:        riskTier,
		Policy:          policy,
		PriorityScore:   score,
	}, nil
}
