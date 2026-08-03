package segment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// BootstrapSegments seeds retailer segments, SKU classes, and default policies.
func (s *Service) BootstrapSegments(ctx context.Context, supplierID, actor string) (BootstrapResult, error) {
	if s == nil || s.repo == nil {
		return BootstrapResult{}, fmt.Errorf("segment service unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	actor = strings.TrimSpace(actor)
	if supplierID == "" {
		return BootstrapResult{}, fmt.Errorf("supplier_id required")
	}
	if actor == "" {
		actor = "system"
	}

	stats, err := s.repo.ListRetailerOrderStats(ctx, supplierID)
	if err != nil {
		return BootstrapResult{}, err
	}
	retailerIDs := make([]string, 0, len(stats))
	for _, st := range stats {
		retailerIDs = append(retailerIDs, st.RetailerID)
	}
	riskTiers, err := s.repo.ListRetailerRiskTiers(ctx, retailerIDs)
	if err != nil {
		return BootstrapResult{}, err
	}
	creditScores, err := s.repo.ListRetailerCreditScores(ctx, retailerIDs)
	if err != nil {
		return BootstrapResult{}, err
	}
	claimCounts, err := s.repo.ListRetailerClaimCounts(ctx, supplierID)
	if err != nil {
		return BootstrapResult{}, err
	}

	now := s.now()
	topVolume := topVolumePercentile(stats, 0.20)
	var mutations []*spanner.Mutation
	buf := &segmentTxnBuffer{}
	var segmentsUpserted int

	for _, st := range stats {
		existing, found, err := s.repo.GetRetailerSegmentRecord(ctx, st.RetailerID)
		if err != nil {
			return BootstrapResult{}, err
		}
		if found && isManualSegment(existing.Reason) {
			continue
		}
		segmentVal := bootstrapSegment(RetailerBootstrapInput{
			RetailerID:  st.RetailerID,
			OrderCount:  st.OrderCount,
			ClaimCount:  claimCounts[st.RetailerID],
			CreditScore: creditScores[st.RetailerID],
			RiskTier:    riskTiers[st.RetailerID],
			InTopVolume: topVolume[st.RetailerID],
		})
		mutations = append(mutations, spanner.InsertOrUpdateMap("RetailerSegments", map[string]interface{}{
			"RetailerId":    st.RetailerID,
			"Segment":       segmentVal,
			"Reason":        bootstrapReason,
			"EffectiveFrom": now,
			"UpdatedBy":     actor,
			"UpdatedAt":     now,
		}))
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, st.RetailerID, events.TopicMain, map[string]interface{}{
			"type":        events.EventRetailerSegmentUpdated,
			"timestamp":   now.Format(time.RFC3339Nano),
			"retailer_id": st.RetailerID,
			"supplier_id": supplierID,
			"segment":     segmentVal,
			"source":      "bootstrap",
		}); err != nil {
			return BootstrapResult{}, err
		}
		segmentsUpserted++
	}

	skuQtys, err := s.repo.SumSkuOrderQuantities(ctx, supplierID, now.Add(-90*24*time.Hour))
	if err != nil {
		return BootstrapResult{}, err
	}
	products, err := s.repo.ListProductMeta(ctx, supplierID)
	if err != nil {
		return BootstrapResult{}, err
	}
	prices := make([]int64, 0, len(products))
	productByID := make(map[string]ProductMeta, len(products))
	for _, p := range products {
		prices = append(prices, p.PriceMinor)
		productByID[p.ProductID] = p
	}
	medPrice := medianPrice(prices)
	ranks := velocityRanks(skuQtys)
	var skuClassesUpserted int

	for sku, qty := range skuQtys {
		meta := productByID[sku]
		cls := bootstrapSkuClass(SkuBootstrapInput{
			Sku:           sku,
			OrderQty:      qty,
			VelocityRank:  ranks[sku],
			PriceMinor:    meta.PriceMinor,
			MedianPrice:   medPrice,
			HandlingClass: meta.HandlingClass,
		})
		cls.SupplierID = supplierID
		cls.UpdatedAt = now
		mutations = append(mutations, spanner.InsertOrUpdateMap("SkuClasses", map[string]interface{}{
			"SupplierId":    supplierID,
			"Sku":           sku,
			"VelocityClass": cls.VelocityClass,
			"StrategicFlag": cls.StrategicFlag,
			"UpdatedAt":     now,
		}))
		if err := outbox.EmitJSON(ctx, buf, events.AggregateProduct, sku, events.TopicMain, map[string]interface{}{
			"type":           events.EventSkuClassUpdated,
			"timestamp":      now.Format(time.RFC3339Nano),
			"supplier_id":    supplierID,
			"sku":            sku,
			"velocity_class": cls.VelocityClass,
			"strategic":      cls.StrategicFlag,
			"source":         "bootstrap",
		}); err != nil {
			return BootstrapResult{}, err
		}
		skuClassesUpserted++
	}

	policiesSeeded, policyMuts, err := s.seedDefaultPoliciesMutations(ctx, supplierID, now, buf)
	if err != nil {
		return BootstrapResult{}, err
	}
	mutations = append(mutations, policyMuts...)
	if err := s.repo.ApplyBootstrapBatch(ctx, mutations, buf); err != nil {
		return BootstrapResult{}, err
	}

	return BootstrapResult{
		SegmentsUpserted:   segmentsUpserted,
		SkuClassesUpserted: skuClassesUpserted,
		PoliciesSeeded:     policiesSeeded,
	}, nil
}

func (s *Service) seedDefaultPoliciesMutations(ctx context.Context, supplierID string, now time.Time, buf *segmentTxnBuffer) (int, []*spanner.Mutation, error) {
	count, err := s.repo.CountPolicies(ctx, supplierID)
	if err != nil {
		return 0, nil, err
	}
	if count > 0 {
		return 0, nil, nil
	}
	defaults := []ServicePolicy{
		{RetailerSegment: SegmentA, SkuClass: VelocityA, PriorityWeight: 100, TargetServiceLevelBps: 9800, MaxFairShareBps: 5000, MinFairShareBps: 1500, CreditRiskBoost: 20},
		{RetailerSegment: SegmentA, SkuClass: VelocityB, PriorityWeight: 90, TargetServiceLevelBps: 9700, MaxFairShareBps: 4500, MinFairShareBps: 1200, CreditRiskBoost: 20},
		{RetailerSegment: SegmentB, SkuClass: VelocityA, PriorityWeight: 75, TargetServiceLevelBps: 9600, MaxFairShareBps: 4000, MinFairShareBps: 1000, CreditRiskBoost: 15},
		{RetailerSegment: SegmentB, SkuClass: VelocityB, PriorityWeight: 70, TargetServiceLevelBps: 9500, MaxFairShareBps: 3500, MinFairShareBps: 800, CreditRiskBoost: 15},
		{RetailerSegment: SegmentC, SkuClass: VelocityA, PriorityWeight: 45, TargetServiceLevelBps: 9200, MaxFairShareBps: 3000, MinFairShareBps: 500, CreditRiskBoost: 10},
		{RetailerSegment: SegmentC, SkuClass: VelocityC, PriorityWeight: 30, TargetServiceLevelBps: 9000, MaxFairShareBps: 2500, MinFairShareBps: 300, CreditRiskBoost: 5},
	}
	muts := make([]*spanner.Mutation, 0, len(defaults))
	seeded := 0
	for i := range defaults {
		p := defaults[i]
		p.PolicyID = uuid.NewString()
		p.SupplierID = supplierID
		p.Enabled = true
		p.UpdatedAt = now
		muts = append(muts, spanner.InsertOrUpdateMap("ServicePolicies", map[string]interface{}{
			"PolicyId":              p.PolicyID,
			"SupplierId":            supplierID,
			"RetailerSegment":       p.RetailerSegment,
			"SkuClass":              p.SkuClass,
			"PriorityWeight":        p.PriorityWeight,
			"TargetServiceLevelBps": p.TargetServiceLevelBps,
			"MaxFairShareBps":       p.MaxFairShareBps,
			"MinFairShareBps":       p.MinFairShareBps,
			"CreditRiskBoost":       p.CreditRiskBoost,
			"Enabled":               p.Enabled,
			"UpdatedAt":             now,
		}))
		if err := outbox.EmitJSON(ctx, buf, events.AggregateSupplier, supplierID, events.TopicMain, map[string]interface{}{
			"type":        events.EventServicePolicyUpdated,
			"timestamp":   now.Format(time.RFC3339Nano),
			"supplier_id": supplierID,
			"policy_id":   p.PolicyID,
			"source":      "bootstrap",
		}); err != nil {
			return 0, nil, err
		}
		seeded++
	}
	return seeded, muts, nil
}

// ListRetailerSegments returns segments for retailers that ordered from this supplier.
func (s *Service) ListRetailerSegments(ctx context.Context, supplierID string) ([]RetailerSegmentView, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("segment service unavailable")
	}
	return s.repo.ListRetailerSegmentsForSupplier(ctx, supplierID)
}

// ListSkuClasses returns SKU classes for a supplier.
func (s *Service) ListSkuClasses(ctx context.Context, supplierID string) ([]SkuClassView, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("segment service unavailable")
	}
	return s.repo.ListSkuClassesForSupplier(ctx, supplierID)
}

// SetRetailerSegment manual override (never overwritten by bootstrap).
func (s *Service) SetRetailerSegment(ctx context.Context, retailerID, actor string, in SetRetailerSegmentInput) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("segment service unavailable")
	}
	seg := NormalizeRetailerSegment(in.Segment)
	if seg == "" {
		return fmt.Errorf("invalid_segment")
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "manual override"
	}
	now := s.now()
	return s.repo.UpsertRetailerSegment(ctx, RetailerSegment{
		RetailerID:    retailerID,
		Segment:       seg,
		Reason:        manualReasonPrefix + reason,
		EffectiveFrom: now,
		UpdatedBy:     actor,
		UpdatedAt:     now,
	})
}

// SetSkuClass manual override for velocity class.
func (s *Service) SetSkuClass(ctx context.Context, supplierID, sku, actor string, in SetSkuClassInput) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("segment service unavailable")
	}
	velocity := NormalizeVelocityClass(in.VelocityClass)
	strategic := false
	if in.StrategicFlag != nil {
		strategic = *in.StrategicFlag
	}
	return s.repo.UpsertSkuClass(ctx, SkuClass{
		SupplierID:    supplierID,
		Sku:           sku,
		VelocityClass: velocity,
		StrategicFlag: strategic,
		UpdatedAt:     s.now(),
	})
}
