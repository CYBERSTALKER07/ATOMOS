package segment

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) GetRetailerSegment(ctx context.Context, retailerID string) (string, error) {
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return SegmentC, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "RetailerSegments", spanner.Key{retailerID},
		[]string{"Segment"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return SegmentC, nil
		}
		return SegmentC, err
	}
	var segment string
	if err := row.Column(0, &segment); err != nil {
		return SegmentC, err
	}
	return NormalizeRetailerSegment(segment), nil
}

func (r *SpannerRepository) GetSkuClass(ctx context.Context, supplierID, sku string) (SkuClass, error) {
	supplierID = strings.TrimSpace(supplierID)
	sku = strings.TrimSpace(sku)
	defaults := SkuClass{
		SupplierID:    supplierID,
		Sku:           sku,
		VelocityClass: VelocityB,
		StrategicFlag: false,
	}
	if supplierID == "" || sku == "" {
		return defaults, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "SkuClasses", spanner.Key{supplierID, sku},
		[]string{"VelocityClass", "StrategicFlag", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return defaults, nil
		}
		return defaults, err
	}
	var updatedAt time.Time
	if err := row.Columns(&defaults.VelocityClass, &defaults.StrategicFlag, &updatedAt); err != nil {
		return defaults, err
	}
	defaults.VelocityClass = NormalizeVelocityClass(defaults.VelocityClass)
	defaults.UpdatedAt = updatedAt
	return defaults, nil
}

func (r *SpannerRepository) ResolvePolicy(ctx context.Context, supplierID, retailerSegment, skuClass string) (ServicePolicy, error) {
	supplierID = strings.TrimSpace(supplierID)
	retailerSegment = NormalizeRetailerSegment(retailerSegment)
	skuClass = NormalizeVelocityClass(skuClass)
	if supplierID == "" {
		return DefaultPolicy(supplierID, retailerSegment, skuClass), nil
	}

	policy, found, err := r.queryPolicy(ctx, supplierID, retailerSegment, skuClass)
	if err != nil {
		return ServicePolicy{}, err
	}
	if found {
		return policy, nil
	}
	policy, found, err = r.queryPolicy(ctx, supplierID, retailerSegment, SkuWildcard)
	if err != nil {
		return ServicePolicy{}, err
	}
	if found {
		return policy, nil
	}
	return DefaultPolicy(supplierID, retailerSegment, skuClass), nil
}

func (r *SpannerRepository) queryPolicy(ctx context.Context, supplierID, retailerSegment, skuClass string) (ServicePolicy, bool, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT PolicyId, SupplierId, RetailerSegment, SkuClass, PriorityWeight,
		             TargetServiceLevelBps, MaxFairShareBps, MinFairShareBps, CreditRiskBoost,
		             Enabled, UpdatedAt
		      FROM ServicePolicies
		      WHERE SupplierId = @supplierId
		        AND RetailerSegment = @segment
		        AND SkuClass = @skuClass
		        AND Enabled = true
		      LIMIT 1`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"segment":    retailerSegment,
			"skuClass":   skuClass,
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return ServicePolicy{}, false, nil
	}
	if err != nil {
		return ServicePolicy{}, false, err
	}
	var p ServicePolicy
	var enabled bool
	if err := row.Columns(
		&p.PolicyID, &p.SupplierID, &p.RetailerSegment, &p.SkuClass, &p.PriorityWeight,
		&p.TargetServiceLevelBps, &p.MaxFairShareBps, &p.MinFairShareBps, &p.CreditRiskBoost,
		&enabled, &p.UpdatedAt,
	); err != nil {
		return ServicePolicy{}, false, err
	}
	p.Enabled = enabled
	return p, true, nil
}

func (r *SpannerRepository) GetRiskTier(ctx context.Context, retailerID string) (credit.RiskTier, error) {
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return credit.RiskTierMedium, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "RetailerCreditScores", spanner.Key{retailerID}, []string{"RiskTier"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return credit.RiskTierMedium, nil
		}
		return credit.RiskTierMedium, err
	}
	var tier string
	if err := row.Column(0, &tier); err != nil {
		return credit.RiskTierMedium, err
	}
	rt := credit.RiskTier(strings.TrimSpace(tier))
	if !rt.Valid() {
		return credit.RiskTierMedium, nil
	}
	return rt, nil
}

func (r *SpannerRepository) CountPolicies(ctx context.Context, supplierID string) (int64, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM ServicePolicies WHERE SupplierId = @supplierId`,
		Params: map[string]interface{}{"supplierId": supplierID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Column(0, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *SpannerRepository) UpsertRetailerSegment(ctx context.Context, seg RetailerSegment) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerSegments", map[string]interface{}{
			"RetailerId":    seg.RetailerID,
			"Segment":       NormalizeRetailerSegment(seg.Segment),
			"Reason":        seg.Reason,
			"EffectiveFrom": seg.EffectiveFrom,
			"EffectiveTo":   seg.EffectiveTo,
			"UpdatedBy":     seg.UpdatedBy,
			"UpdatedAt":     seg.UpdatedAt,
		}),
	})
	return err
}

func (r *SpannerRepository) UpsertSkuClass(ctx context.Context, cls SkuClass) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("SkuClasses", map[string]interface{}{
			"SupplierId":    cls.SupplierID,
			"Sku":           cls.Sku,
			"VelocityClass": NormalizeVelocityClass(cls.VelocityClass),
			"StrategicFlag": cls.StrategicFlag,
			"UpdatedAt":     cls.UpdatedAt,
		}),
	})
	return err
}

func (r *SpannerRepository) UpsertServicePolicy(ctx context.Context, policy ServicePolicy) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("ServicePolicies", map[string]interface{}{
			"PolicyId":              policy.PolicyID,
			"SupplierId":            policy.SupplierID,
			"RetailerSegment":       NormalizeRetailerSegment(policy.RetailerSegment),
			"SkuClass":              policy.SkuClass,
			"PriorityWeight":        policy.PriorityWeight,
			"TargetServiceLevelBps": policy.TargetServiceLevelBps,
			"MaxFairShareBps":       policy.MaxFairShareBps,
			"MinFairShareBps":       policy.MinFairShareBps,
			"CreditRiskBoost":       policy.CreditRiskBoost,
			"Enabled":               policy.Enabled,
			"UpdatedAt":             policy.UpdatedAt,
		}),
	})
	return err
}

func (r *SpannerRepository) ListRetailerOrderStats(ctx context.Context, supplierID string) ([]RetailerOrderStats, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT RetailerId, COUNT(*) AS cnt
		      FROM Orders
		      WHERE SupplierId = @supplierId
		      GROUP BY RetailerId`,
		Params: map[string]interface{}{"supplierId": supplierID},
	})
	defer iter.Stop()
	var out []RetailerOrderStats
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var stats RetailerOrderStats
		if err := row.Columns(&stats.RetailerID, &stats.OrderCount); err != nil {
			return nil, err
		}
		out = append(out, stats)
	}
	return out, nil
}

func (r *SpannerRepository) ListRetailerRiskTiers(ctx context.Context, retailerIDs []string) (map[string]credit.RiskTier, error) {
	out := make(map[string]credit.RiskTier, len(retailerIDs))
	if len(retailerIDs) == 0 {
		return out, nil
	}
	iter := r.client.Single().Read(ctx, "RetailerCreditScores", spanner.KeySetFromKeys(keysFromStrings(retailerIDs)...), []string{"RetailerId", "RiskTier"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var id, tier string
		if err := row.Columns(&id, &tier); err != nil {
			return nil, err
		}
		rt := credit.RiskTier(strings.TrimSpace(tier))
		if rt.Valid() {
			out[id] = rt
		}
	}
	return out, nil
}

func (r *SpannerRepository) GetRetailerSegmentRecord(ctx context.Context, retailerID string) (RetailerSegment, bool, error) {
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return RetailerSegment{}, false, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "RetailerSegments", spanner.Key{retailerID},
		[]string{"Segment", "Reason", "EffectiveFrom", "EffectiveTo", "UpdatedBy", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return RetailerSegment{}, false, nil
		}
		return RetailerSegment{}, false, err
	}
	var seg RetailerSegment
	seg.RetailerID = retailerID
	var effTo spanner.NullTime
	if err := row.Columns(&seg.Segment, &seg.Reason, &seg.EffectiveFrom, &effTo, &seg.UpdatedBy, &seg.UpdatedAt); err != nil {
		return RetailerSegment{}, false, err
	}
	if effTo.Valid {
		t := effTo.Time
		seg.EffectiveTo = &t
	}
	return seg, true, nil
}

func (r *SpannerRepository) ListRetailerCreditScores(ctx context.Context, retailerIDs []string) (map[string]int64, error) {
	out := make(map[string]int64, len(retailerIDs))
	if len(retailerIDs) == 0 {
		return out, nil
	}
	iter := r.client.Single().Read(ctx, "RetailerCreditScores", spanner.KeySetFromKeys(keysFromStrings(retailerIDs)...),
		[]string{"RetailerId", "Score"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var id string
		var score int64
		if err := row.Columns(&id, &score); err != nil {
			return nil, err
		}
		out[id] = score
	}
	return out, nil
}

func (r *SpannerRepository) ListRetailerClaimCounts(ctx context.Context, supplierID string) (map[string]int64, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT RetailerId, COUNT(*) AS cnt
		      FROM Claims
		      WHERE SupplierId = @sid
		      GROUP BY RetailerId`,
		Params: map[string]interface{}{"sid": supplierID},
	})
	defer iter.Stop()
	out := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var id string
		var cnt int64
		if err := row.Columns(&id, &cnt); err != nil {
			return nil, err
		}
		out[id] = cnt
	}
	return out, nil
}

type bootstrapLineItem struct {
	SKU      string `json:"sku"`
	Quantity int64  `json:"quantity"`
}

func (r *SpannerRepository) SumSkuOrderQuantities(ctx context.Context, supplierID string, since time.Time) (map[string]int64, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT LineItemsJson FROM Orders
		      WHERE SupplierId = @sid AND CreatedAt >= @since`,
		Params: map[string]interface{}{
			"sid":  supplierID,
			"since": since,
		},
	})
	defer iter.Stop()
	totals := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var raw []byte
		if err := row.Column(0, &raw); err != nil {
			continue
		}
		var items []bootstrapLineItem
		if err := json.Unmarshal(raw, &items); err != nil {
			continue
		}
		for _, item := range items {
			sku := strings.TrimSpace(item.SKU)
			if sku == "" || item.Quantity <= 0 {
				continue
			}
			totals[sku] += item.Quantity
		}
	}
	return totals, nil
}

func (r *SpannerRepository) ListProductMeta(ctx context.Context, supplierID string) ([]ProductMeta, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, PriceMinor, HandlingClass
		      FROM Products WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]interface{}{"sid": supplierID},
	})
	defer iter.Stop()
	var out []ProductMeta
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var p ProductMeta
		if err := row.Columns(&p.ProductID, &p.PriceMinor, &p.HandlingClass); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *SpannerRepository) ListRetailerSegmentsForSupplier(ctx context.Context, supplierID string) ([]RetailerSegmentView, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT rs.RetailerId, rs.Segment, COALESCE(rs.Reason,''), rs.UpdatedAt
		      FROM RetailerSegments rs
		      INNER JOIN Orders o ON o.RetailerId = rs.RetailerId AND o.SupplierId = @sid
		      GROUP BY rs.RetailerId, rs.Segment, rs.Reason, rs.UpdatedAt
		      ORDER BY rs.UpdatedAt DESC`,
		Params: map[string]interface{}{"sid": supplierID},
	})
	defer iter.Stop()
	var out []RetailerSegmentView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var v RetailerSegmentView
		if err := row.Columns(&v.RetailerID, &v.Segment, &v.Reason, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *SpannerRepository) ListSkuClassesForSupplier(ctx context.Context, supplierID string) ([]SkuClassView, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT Sku, VelocityClass, StrategicFlag, UpdatedAt
		      FROM SkuClasses WHERE SupplierId = @sid ORDER BY Sku`,
		Params: map[string]interface{}{"sid": supplierID},
	})
	defer iter.Stop()
	var out []SkuClassView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var v SkuClassView
		if err := row.Columns(&v.Sku, &v.VelocityClass, &v.StrategicFlag, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *SpannerRepository) SkuClassExists(ctx context.Context, supplierID, sku string) (bool, error) {
	_, err := r.client.Single().ReadRow(ctx, "SkuClasses", spanner.Key{supplierID, sku}, []string{"Sku"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *SpannerRepository) ApplyBootstrapBatch(ctx context.Context, mutations []*spanner.Mutation, buf *segmentTxnBuffer) error {
	if r == nil || r.client == nil {
		return nil
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		all := append([]*spanner.Mutation(nil), mutations...)
		if buf != nil {
			all = append(all, segmentOutboxMutations(buf.events)...)
		}
		if len(all) == 0 {
			return nil
		}
		return txn.BufferWrite(all)
	})
	return err
}

func keysFromStrings(ids []string) []spanner.Key {
	keys := make([]spanner.Key, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, spanner.Key{id})
	}
	return keys
}
