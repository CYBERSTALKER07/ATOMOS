package valuation

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Service orchestrates inventory valuation, FEFO/moving-average cost layers, COGS GL postings, and movement queries.
type Service struct {
	repo Repository
}

// NewService constructs a Spanner-backed valuation Service (nil client creates an unbacked Service).
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{}
	}
	return &Service{repo: NewSpannerRepository(client)}
}

// NewServiceWithRepository constructs a valuation Service backed by a custom Repository (e.g., MemoryRepository).
func NewServiceWithRepository(repo Repository) *Service {
	return &Service{repo: repo}
}

// Repo returns the configured Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// RecordReceipt creates a cost layer, movement, GL entry (DR INVENTORY + optional TAX:VAT:INPUT / CR AP:VENDOR|GRNI),
// and emits inventory.received (plus inventory.expiry.risk when near-expiry).
func (s *Service) RecordReceipt(ctx context.Context, in ReceiptInput) (ReceiptResult, error) {
	if s == nil || s.repo == nil {
		return ReceiptResult{}, ErrRepoUnavailable
	}
	return s.repo.RecordReceipt(ctx, in)
}

// IssueStockFEFO depletes active cost layers in FEFO order and posts DR COGS / CR INVENTORY atomically.
func (s *Service) IssueStockFEFO(ctx context.Context, in IssueInput) (IssueResult, error) {
	if s == nil || s.repo == nil {
		return IssueResult{}, ErrRepoUnavailable
	}
	return s.repo.IssueStockFEFO(ctx, in)
}

// RecordAdjustmentCost posts inventory adjustment gains/shrinkage onto cost layers, movements, and GL.
func (s *Service) RecordAdjustmentCost(ctx context.Context, in AdjustmentCostInput) (AdjustmentCostResult, error) {
	if s == nil || s.repo == nil {
		return AdjustmentCostResult{}, ErrRepoUnavailable
	}
	return s.repo.RecordAdjustmentCost(ctx, in)
}

// GetValuationSummary returns SKU moving-average summaries and FEFO layer details.
func (s *Service) GetValuationSummary(ctx context.Context, filter ValuationFilter) (ValuationSummaryResponse, error) {
	if s == nil || s.repo == nil {
		return ValuationSummaryResponse{}, ErrRepoUnavailable
	}
	return s.repo.GetValuationSummary(ctx, filter)
}

// ListMovements queries append-only InventoryMovements for an entity.
func (s *Service) ListMovements(ctx context.Context, filter MovementsFilter) ([]InventoryMovement, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	return s.repo.ListMovements(ctx, filter)
}

// ScanExpiryRisks lists active cost layers expiring within withinDays (and optionally emits inventory.expiry.risk).
func (s *Service) ScanExpiryRisks(ctx context.Context, entityID, warehouseID string, withinDays int64, emitEvents bool) ([]ExpiryRiskLayer, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	return s.repo.ScanExpiryRisks(ctx, entityID, warehouseID, withinDays, emitEvents)
}

// RecordReceiptInTxn records a valued receipt inside an existing Spanner ReadWriteTransaction.
func (s *Service) RecordReceiptInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, in ReceiptInput) (ReceiptResult, error) {
	return RecordReceiptInTxn(ctx, txn, outboxBuf, in)
}

// RecordReceiptsBatchInTxn records multiple valued receipts inside a single Spanner ReadWriteTransaction.
func (s *Service) RecordReceiptsBatchInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, inputs []ReceiptInput) ([]ReceiptResult, error) {
	return RecordReceiptsBatchInTxn(ctx, txn, outboxBuf, inputs)
}

// IssueStockFEFOInTxn depletes cost layers FEFO inside an existing Spanner ReadWriteTransaction.
func (s *Service) IssueStockFEFOInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, in IssueInput) (IssueResult, error) {
	return IssueStockFEFOInTxn(ctx, txn, outboxBuf, in)
}

// IssueStockFEFOIfPresentInTxn depletes cost layers FEFO if layers exist for (EntityId, WarehouseId, ProductId).
func (s *Service) IssueStockFEFOIfPresentInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, in IssueInput) (IssueResult, bool, error) {
	return IssueStockFEFOIfPresentInTxn(ctx, txn, outboxBuf, in)
}

// RecordAdjustmentCostInTxn posts adjustment cost movements & GL inside an existing Spanner ReadWriteTransaction.
func (s *Service) RecordAdjustmentCostInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, in AdjustmentCostInput) (AdjustmentCostResult, error) {
	return RecordAdjustmentCostInTxn(ctx, txn, outboxBuf, in)
}

// RecordAdjustmentCostIfPresentInTxn posts adjustment cost when active layers exist (or UnitCostMinor > 0).
func (s *Service) RecordAdjustmentCostIfPresentInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, in AdjustmentCostInput) (AdjustmentCostResult, bool, error) {
	return RecordAdjustmentCostIfPresentInTxn(ctx, txn, outboxBuf, in)
}

func validateAndNormalizeReceipt(in ReceiptInput, defaultNow time.Time) (ReceiptInput, int64, string, string, *civil.Date, error) {
	norm := in
	norm.EntityID = strings.TrimSpace(norm.EntityID)
	norm.WarehouseID = strings.TrimSpace(norm.WarehouseID)
	norm.ProductID = strings.TrimSpace(norm.ProductID)
	norm.LotID = strings.TrimSpace(norm.LotID)
	norm.ReceiptRefType = strings.ToUpper(strings.TrimSpace(norm.ReceiptRefType))
	if norm.ReceiptRefType == "" {
		norm.ReceiptRefType = "MANUAL_RECEIPT"
	}
	norm.ReceiptRefID = strings.TrimSpace(norm.ReceiptRefID)
	norm.VendorID = strings.TrimSpace(norm.VendorID)
	norm.FxRateID = strings.TrimSpace(norm.FxRateID)
	norm.CountryPack = strings.ToUpper(strings.TrimSpace(norm.CountryPack))
	norm.Currency = strings.ToUpper(strings.TrimSpace(norm.Currency))
	if norm.Currency == "" {
		norm.Currency = gl.DefaultCurrencyForPack(norm.CountryPack)
	}
	norm.IdempotencyKey = strings.TrimSpace(norm.IdempotencyKey)
	norm.ActorID = strings.TrimSpace(norm.ActorID)
	if norm.ActorID == "" {
		norm.ActorID = "system"
	}
	if norm.EntityID == "" || norm.WarehouseID == "" || norm.ProductID == "" {
		return norm, 0, "", "", nil, fmt.Errorf("%w: entity_id, warehouse_id, and product_id are required", ErrInvalidInput)
	}
	if norm.ReceiptRefID == "" {
		norm.ReceiptRefID = norm.IdempotencyKey
	}
	if norm.IdempotencyKey == "" {
		return norm, 0, "", "", nil, fmt.Errorf("%w: idempotency_key is required", ErrInvalidInput)
	}
	if norm.Quantity <= 0 {
		return norm, 0, "", "", nil, fmt.Errorf("%w: quantity must be positive", ErrInvalidInput)
	}
	if norm.UnitCostMinor <= 0 {
		return norm, 0, "", "", nil, fmt.Errorf("%w: unit_cost_minor must be positive", ErrInvalidInput)
	}
	if norm.VatMinor < 0 {
		return norm, 0, "", "", nil, fmt.Errorf("%w: vat_minor cannot be negative", ErrInvalidInput)
	}
	if len(norm.Currency) != 3 {
		return norm, 0, "", "", nil, fmt.Errorf("%w: invalid 3-letter currency %q", ErrInvalidInput, norm.Currency)
	}

	creditAcct := strings.ToUpper(strings.TrimSpace(norm.CreditAccount))
	switch creditAcct {
	case "", gl.AccountVendorPayable:
		creditAcct = gl.AccountVendorPayable
	case gl.AccountGRNI:
		creditAcct = gl.AccountGRNI
	default:
		return norm, 0, "", "", nil, fmt.Errorf("%w: %q", ErrUnknownCreditAccount, norm.CreditAccount)
	}

	expiryCivil, err := ParseExpiryDate(norm.ExpiryDate)
	if err != nil {
		return norm, 0, "", "", nil, fmt.Errorf("%w: invalid expiry_date %q", ErrInvalidInput, norm.ExpiryDate)
	}
	var expiryStr string
	if expiryCivil != nil {
		expiryStr = expiryCivil.String()
	}

	if norm.ReceivedAt.IsZero() {
		norm.ReceivedAt = defaultNow.UTC()
	} else {
		norm.ReceivedAt = norm.ReceivedAt.UTC()
	}
	if norm.ExpiryRiskWindowDays <= 0 {
		norm.ExpiryRiskWindowDays = DefaultExpiryRiskWindowDays
	}

	totalCost := norm.Quantity * norm.UnitCostMinor
	return norm, totalCost, creditAcct, expiryStr, expiryCivil, nil
}

func validateAndNormalizeIssue(in IssueInput, defaultNow time.Time) (IssueInput, error) {
	norm := in
	norm.EntityID = strings.TrimSpace(norm.EntityID)
	norm.WarehouseID = strings.TrimSpace(norm.WarehouseID)
	norm.ProductID = strings.TrimSpace(norm.ProductID)
	norm.LotID = strings.TrimSpace(norm.LotID)
	norm.ReferenceType = strings.ToUpper(strings.TrimSpace(norm.ReferenceType))
	if norm.ReferenceType == "" {
		norm.ReferenceType = "PICK_TASK"
	}
	norm.ReferenceID = strings.TrimSpace(norm.ReferenceID)
	norm.OrderID = strings.TrimSpace(norm.OrderID)
	norm.CostingMethod = strings.ToUpper(strings.TrimSpace(norm.CostingMethod))
	if norm.CostingMethod == "" {
		norm.CostingMethod = CostingMethodFEFO
	}
	if norm.CostingMethod != CostingMethodFEFO && norm.CostingMethod != CostingMethodMovingAverage {
		return norm, fmt.Errorf("%w: unsupported costing_method %q", ErrInvalidInput, norm.CostingMethod)
	}
	norm.CountryPack = strings.ToUpper(strings.TrimSpace(norm.CountryPack))
	norm.Currency = strings.ToUpper(strings.TrimSpace(norm.Currency))
	if norm.Currency == "" {
		norm.Currency = gl.DefaultCurrencyForPack(norm.CountryPack)
	}
	norm.IdempotencyKey = strings.TrimSpace(norm.IdempotencyKey)
	norm.ActorID = strings.TrimSpace(norm.ActorID)
	if norm.ActorID == "" {
		norm.ActorID = "system"
	}
	if norm.EntityID == "" || norm.WarehouseID == "" || norm.ProductID == "" {
		return norm, fmt.Errorf("%w: entity_id, warehouse_id, and product_id are required", ErrInvalidInput)
	}
	if norm.ReferenceID == "" {
		norm.ReferenceID = norm.IdempotencyKey
	}
	if norm.IdempotencyKey == "" {
		return norm, fmt.Errorf("%w: idempotency_key is required", ErrInvalidInput)
	}
	if norm.Quantity <= 0 {
		return norm, fmt.Errorf("%w: quantity must be positive", ErrInvalidInput)
	}
	if norm.OccurredAt.IsZero() {
		norm.OccurredAt = defaultNow.UTC()
	} else {
		norm.OccurredAt = norm.OccurredAt.UTC()
	}
	return norm, nil
}

func validateAndNormalizeAdjustment(in AdjustmentCostInput, defaultNow time.Time) (AdjustmentCostInput, error) {
	norm := in
	norm.EntityID = strings.TrimSpace(norm.EntityID)
	norm.WarehouseID = strings.TrimSpace(norm.WarehouseID)
	norm.ProductID = strings.TrimSpace(norm.ProductID)
	norm.LotID = strings.TrimSpace(norm.LotID)
	norm.AdjustmentID = strings.TrimSpace(norm.AdjustmentID)
	norm.ReasonCode = strings.TrimSpace(norm.ReasonCode)
	norm.CountryPack = strings.ToUpper(strings.TrimSpace(norm.CountryPack))
	norm.Currency = strings.ToUpper(strings.TrimSpace(norm.Currency))
	if norm.Currency == "" {
		norm.Currency = gl.DefaultCurrencyForPack(norm.CountryPack)
	}
	norm.IdempotencyKey = strings.TrimSpace(norm.IdempotencyKey)
	norm.ActorID = strings.TrimSpace(norm.ActorID)
	if norm.ActorID == "" {
		norm.ActorID = "system"
	}
	if norm.EntityID == "" || norm.WarehouseID == "" || norm.ProductID == "" || norm.AdjustmentID == "" {
		return norm, fmt.Errorf("%w: entity_id, warehouse_id, product_id, and adjustment_id are required", ErrInvalidInput)
	}
	if norm.DeltaQty == 0 {
		return norm, fmt.Errorf("%w: delta_qty must be non-zero", ErrInvalidInput)
	}
	if norm.IdempotencyKey == "" {
		norm.IdempotencyKey = "adj:" + norm.AdjustmentID
	}
	if norm.OccurredAt.IsZero() {
		norm.OccurredAt = defaultNow.UTC()
	} else {
		norm.OccurredAt = norm.OccurredAt.UTC()
	}
	return norm, nil
}

// computeProportionalCost computes integer half-up cost for issueQty out of totalAvailQty with totalValMinor.
// When issueQty == totalAvailQty, it guarantees exact full depletion (totalValMinor) with zero residue.
func computeProportionalCost(totalValMinor, totalAvailQty, issueQty int64) int64 {
	if issueQty <= 0 || totalAvailQty <= 0 || totalValMinor <= 0 {
		return 0
	}
	if issueQty >= totalAvailQty {
		return totalValMinor
	}
	num := new(big.Int).Mul(big.NewInt(totalValMinor), big.NewInt(issueQty))
	half := new(big.Int).Quo(big.NewInt(totalAvailQty), big.NewInt(2))
	num.Add(num, half)
	num.Quo(num, big.NewInt(totalAvailQty))
	return num.Int64()
}

func buildIssueResultFromMovements(mvs []InventoryMovement, already bool) IssueResult {
	var totalQty, totalCost int64
	var glID string
	for _, mv := range mvs {
		if mv.QtyDelta < 0 {
			totalQty += -mv.QtyDelta
		} else {
			totalQty += mv.QtyDelta
		}
		totalCost += mv.TotalCostMinor
		if glID == "" {
			glID = mv.GlEntryID
		}
	}
	return IssueResult{
		Movements:        append([]InventoryMovement(nil), mvs...),
		TotalQtyIssued:   totalQty,
		TotalCostMinor:   totalCost,
		UnitCostAvgMinor: ComputeMovingAverageUnitCostMinor(totalCost, totalQty),
		GlEntryID:        glID,
		AlreadyProcessed: already,
	}
}

func sortLayersFEFO(layers []InventoryCostLayer) {
	sort.Slice(layers, func(i, j int) bool {
		a, b := layers[i], layers[j]
		if a.WarehouseID != b.WarehouseID {
			return a.WarehouseID < b.WarehouseID
		}
		if a.ProductID != b.ProductID {
			return a.ProductID < b.ProductID
		}
		if a.ExpiryDate != b.ExpiryDate {
			if a.ExpiryDate == "" {
				return false // NULLS LAST
			}
			if b.ExpiryDate == "" {
				return true
			}
			return a.ExpiryDate < b.ExpiryDate
		}
		if !a.ReceivedAt.Equal(b.ReceivedAt) {
			return a.ReceivedAt.Before(b.ReceivedAt)
		}
		return a.LayerID < b.LayerID
	})
}

func buildValuationResponse(entityID, warehouseID string, expiryRiskDays int64, layers []InventoryCostLayer) ValuationSummaryResponse {
	if expiryRiskDays <= 0 {
		expiryRiskDays = DefaultExpiryRiskWindowDays
	}
	type skuKey struct {
		warehouseID string
		productID   string
		currency    string
	}
	skuMap := make(map[skuKey]*SKUValuationSummary)
	var keys []skuKey
	var totalQty, totalVal int64
	currency := "UZS"

	for _, l := range layers {
		if l.Currency != "" && currency == "UZS" {
			currency = l.Currency
		}
		totalQty += l.QtyRemaining
		totalVal += l.TotalValueMinor

		k := skuKey{warehouseID: l.WarehouseID, productID: l.ProductID, currency: l.Currency}
		s, ok := skuMap[k]
		if !ok {
			s = &SKUValuationSummary{
				EntityID:    l.EntityID,
				WarehouseID: l.WarehouseID,
				ProductID:   l.ProductID,
				Currency:    l.Currency,
			}
			skuMap[k] = s
			keys = append(keys, k)
		}
		s.TotalQtyRemaining += l.QtyRemaining
		s.TotalValueMinor += l.TotalValueMinor
		s.LayerCount++
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].warehouseID != keys[j].warehouseID {
			return keys[i].warehouseID < keys[j].warehouseID
		}
		if keys[i].productID != keys[j].productID {
			return keys[i].productID < keys[j].productID
		}
		return keys[i].currency < keys[j].currency
	})

	skus := make([]SKUValuationSummary, 0, len(keys))
	for _, k := range keys {
		s := skuMap[k]
		s.MovingAvgUnitCostMinor = ComputeMovingAverageUnitCostMinor(s.TotalValueMinor, s.TotalQtyRemaining)
		skus = append(skus, *s)
	}
	if layers == nil {
		layers = []InventoryCostLayer{}
	}

	return ValuationSummaryResponse{
		EntityID:          entityID,
		WarehouseID:       warehouseID,
		Currency:          currency,
		TotalQtyRemaining: totalQty,
		TotalValueMinor:   totalVal,
		ExpiryRiskDays:    expiryRiskDays,
		SKUs:              skus,
		Layers:            layers,
	}
}
