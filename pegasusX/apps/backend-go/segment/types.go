package segment

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

const (
	SegmentA = "A"
	SegmentB = "B"
	SegmentC = "C"

	VelocityA = "A"
	VelocityB = "B"
	VelocityC = "C"

	SkuWildcard = "*"

	AllocationModeFirstFit = "FIRST_FIT"
	AllocationModePolicy   = "POLICY"
)

// RetailerSegment is a commercial segment tag for a retailer.
type RetailerSegment struct {
	RetailerID    string
	Segment       string
	Reason        string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	UpdatedBy     string
	UpdatedAt     time.Time
}

// SkuClass classifies SKU velocity / strategic importance for a supplier.
type SkuClass struct {
	SupplierID    string
	Sku           string
	VelocityClass string
	StrategicFlag bool
	UpdatedAt     time.Time
}

// ServicePolicy maps segment × SKU class to allocation weights.
type ServicePolicy struct {
	PolicyID              string
	SupplierID            string
	RetailerSegment       string
	SkuClass              string
	PriorityWeight        int64
	TargetServiceLevelBps int64
	MaxFairShareBps       int64
	MinFairShareBps       int64
	CreditRiskBoost       int64
	Enabled               bool
	UpdatedAt             time.Time
}

// LineAllocationContext is resolved policy context for one order line.
type LineAllocationContext struct {
	RetailerSegment string
	SkuClass        string
	RiskTier        credit.RiskTier
	Policy          ServicePolicy
	PriorityScore   int64
}

// Repository reads segmentation and policy data.
type Repository interface {
	GetRetailerSegment(ctx context.Context, retailerID string) (string, error)
	GetSkuClass(ctx context.Context, supplierID, sku string) (SkuClass, error)
	ResolvePolicy(ctx context.Context, supplierID, retailerSegment, skuClass string) (ServicePolicy, error)
	GetRiskTier(ctx context.Context, retailerID string) (credit.RiskTier, error)
	CountPolicies(ctx context.Context, supplierID string) (int64, error)
	UpsertRetailerSegment(ctx context.Context, seg RetailerSegment) error
	UpsertSkuClass(ctx context.Context, cls SkuClass) error
	UpsertServicePolicy(ctx context.Context, policy ServicePolicy) error
	ListRetailerOrderStats(ctx context.Context, supplierID string) ([]RetailerOrderStats, error)
	ListRetailerRiskTiers(ctx context.Context, retailerIDs []string) (map[string]credit.RiskTier, error)
	GetRetailerSegmentRecord(ctx context.Context, retailerID string) (RetailerSegment, bool, error)
	ListRetailerCreditScores(ctx context.Context, retailerIDs []string) (map[string]int64, error)
	ListRetailerClaimCounts(ctx context.Context, supplierID string) (map[string]int64, error)
	SumSkuOrderQuantities(ctx context.Context, supplierID string, since time.Time) (map[string]int64, error)
	ListProductMeta(ctx context.Context, supplierID string) ([]ProductMeta, error)
	ListRetailerSegmentsForSupplier(ctx context.Context, supplierID string) ([]RetailerSegmentView, error)
	ListSkuClassesForSupplier(ctx context.Context, supplierID string) ([]SkuClassView, error)
	SkuClassExists(ctx context.Context, supplierID, sku string) (bool, error)
	ApplyBootstrapBatch(ctx context.Context, mutations []*spanner.Mutation, buf *segmentTxnBuffer) error
}

// RetailerOrderStats aggregates order volume for bootstrap heuristics.
type RetailerOrderStats struct {
	RetailerID string
	OrderCount int64
}

// BootstrapResult summarizes a bootstrap run.
type BootstrapResult struct {
	SegmentsUpserted   int `json:"segments_upserted"`
	SkuClassesUpserted int `json:"sku_classes_upserted"`
	PoliciesSeeded     int `json:"policies_seeded"`
}

// RetailerSegmentView is API response for retailer segment list.
type RetailerSegmentView struct {
	RetailerID string    `json:"retailer_id"`
	Segment    string    `json:"segment"`
	Reason     string    `json:"reason,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SkuClassView is API response for SKU class list.
type SkuClassView struct {
	Sku           string    `json:"sku"`
	VelocityClass string    `json:"velocity_class"`
	StrategicFlag bool      `json:"strategic_flag"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SetRetailerSegmentInput is manual segment override body.
type SetRetailerSegmentInput struct {
	Segment string `json:"segment"`
	Reason  string `json:"reason,omitempty"`
}

// SetSkuClassInput is manual SKU class override body.
type SetSkuClassInput struct {
	VelocityClass string `json:"velocity_class"`
	StrategicFlag *bool  `json:"strategic_flag,omitempty"`
}

// ProductMeta holds catalog fields used in SKU class bootstrap.
type ProductMeta struct {
	ProductID     string
	PriceMinor    int64
	HandlingClass string
}
