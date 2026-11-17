package proposals

import (
	"context"
	"time"
)

// AuditRecordInput carries immutable AuditEvents metadata for proposal transitions.
type AuditRecordInput struct {
	TenantID   string
	ActorID    string
	ActorType  string
	Role       string
	Action     string
	ObjectType string
	ObjectID   string
	BeforeJSON string
	AfterJSON  string
	ReasonCode string
	ApprovalID string
	TraceID    string
	IP         string
	DeviceID   string
}

// AuditRecorder persists immutable AuditEvents.
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
}

// ReorderCandidate represents a low-cover SKU / pending insight / forecast candidate for REORDER_AGENT.
type ReorderCandidate struct {
	InsightID        string
	SuggestionKey    string
	WarehouseID      string
	ProductID        string
	RetailerID       string
	ProductName      string
	CurrentStock     int64
	ReorderThreshold int64
	SuggestedQty     int64
	UnitPriceMinor   int64
	Currency         string
	TargetFactoryID  string
	ForecastP50Milli int64
	ForecastRunID    string
	ConfidenceBps    int64
	UrgencyLevel     string
	ReasonCode       string
}

// OverdueInvoiceCandidate represents an overdue AR invoice for COLLECTIONS_AGENT.
type OverdueInvoiceCandidate struct {
	InvoiceID    string
	OrderID      string
	RetailerID   string
	RetailerName string
	Phone        string
	Locale       string
	BalanceMinor int64
	Currency     string
	DueAt        time.Time
	DaysOverdue  int64
	DunningStep  int64
}

// ExpiryRiskCandidate represents a near-expiry StockLot / InventoryCostLayer for EXPIRY_REBALANCE_AGENT.
type ExpiryRiskCandidate struct {
	LotID             string
	LayerID           string
	ProductID         string
	ProductName       string
	SourceWarehouseID string
	TargetWarehouseID string
	QtyAvailable      int64
	UnitCostMinor     int64
	ValueAtRiskMinor  int64
	Currency          string
	ExpiryDate        string
	DaysToExpiry      int64
}

// AgentDataReader reads live domain signals for the 3 L1 agents and verifies fresh execution preconditions.
type AgentDataReader interface {
	ListReorderCandidates(ctx context.Context, tenantID string, limit int) ([]ReorderCandidate, error)
	ListOverdueInvoices(ctx context.Context, tenantID string, now time.Time, limit int) ([]OverdueInvoiceCandidate, error)
	ListExpiryRiskCandidates(ctx context.Context, tenantID string, now time.Time, windowDays int, limit int) ([]ExpiryRiskCandidate, error)

	VerifyReorderPrecondition(ctx context.Context, tenantID string, p Proposal) error
	VerifyCollectionsPrecondition(ctx context.Context, tenantID string, p Proposal) error
	VerifyExpiryRebalancePrecondition(ctx context.Context, tenantID string, p Proposal) error
}

// Repository defines durable storage for Proposals.
type Repository interface {
	FindByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (Proposal, bool, error)
	Create(ctx context.Context, p Proposal) error
	Get(ctx context.Context, tenantID, proposalID string) (Proposal, bool, error)
	List(ctx context.Context, f ListFilter) ([]Proposal, error)
	Update(ctx context.Context, p Proposal) error
	HasOverlappingExecuted(ctx context.Context, tenantID, targetType, targetID, proposalType, excludeProposalID string, createdBeforeOrAt time.Time) (bool, error)
	HasActiveForTarget(ctx context.Context, tenantID, targetType, targetID, proposalType string) (bool, error)
	GetL2SoakStats(ctx context.Context, tenantID string, now time.Time) (L2SoakMetrics, error)
}

// L2SoakMetrics captures historical execution & rollback counts for L2 soak gating.
type L2SoakMetrics struct {
	SoakDays          int64
	ExecutedCount     int64
	RollbackCount     int64
	RollbackRateBps   int64
	EarliestExecuteAt *time.Time
}
