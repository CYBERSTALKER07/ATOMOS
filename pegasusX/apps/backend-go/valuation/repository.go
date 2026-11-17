package valuation

import "context"

// Repository defines transactional and query persistence for inventory valuation and movements.
type Repository interface {
	RecordReceipt(ctx context.Context, in ReceiptInput) (ReceiptResult, error)
	IssueStockFEFO(ctx context.Context, in IssueInput) (IssueResult, error)
	RecordAdjustmentCost(ctx context.Context, in AdjustmentCostInput) (AdjustmentCostResult, error)
	GetValuationSummary(ctx context.Context, filter ValuationFilter) (ValuationSummaryResponse, error)
	ListMovements(ctx context.Context, filter MovementsFilter) ([]InventoryMovement, error)
	ScanExpiryRisks(ctx context.Context, entityID, warehouseID string, withinDays int64, emitEvents bool) ([]ExpiryRiskLayer, error)
}
