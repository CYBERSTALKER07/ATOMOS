package valuation

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// SpannerRepository implements Repository against Cloud Spanner InventoryCostLayers and InventoryMovements.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed valuation repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// Client returns the underlying Spanner client.
func (r *SpannerRepository) Client() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

func (r *SpannerRepository) RecordReceipt(ctx context.Context, in ReceiptInput) (ReceiptResult, error) {
	if r == nil || r.client == nil {
		return ReceiptResult{}, ErrRepoUnavailable
	}
	var res ReceiptResult
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		out, err := RecordReceiptInTxn(txCtx, txn, nil, in)
		if err != nil {
			return err
		}
		res = out
		return nil
	})
	if err != nil {
		return ReceiptResult{}, err
	}
	return res, nil
}

func (r *SpannerRepository) IssueStockFEFO(ctx context.Context, in IssueInput) (IssueResult, error) {
	if r == nil || r.client == nil {
		return IssueResult{}, ErrRepoUnavailable
	}
	var res IssueResult
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		out, err := IssueStockFEFOInTxn(txCtx, txn, nil, in)
		if err != nil {
			return err
		}
		res = out
		return nil
	})
	if err != nil {
		return IssueResult{}, err
	}
	return res, nil
}

func (r *SpannerRepository) RecordAdjustmentCost(ctx context.Context, in AdjustmentCostInput) (AdjustmentCostResult, error) {
	if r == nil || r.client == nil {
		return AdjustmentCostResult{}, ErrRepoUnavailable
	}
	var res AdjustmentCostResult
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		out, err := RecordAdjustmentCostInTxn(txCtx, txn, nil, in)
		if err != nil {
			return err
		}
		res = out
		return nil
	})
	if err != nil {
		return AdjustmentCostResult{}, err
	}
	return res, nil
}

func (r *SpannerRepository) GetValuationSummary(ctx context.Context, filter ValuationFilter) (ValuationSummaryResponse, error) {
	if r == nil || r.client == nil {
		return ValuationSummaryResponse{}, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(filter.EntityID)
	if entityID == "" {
		return ValuationSummaryResponse{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	winDays := filter.ExpiryRiskDays
	if winDays <= 0 {
		winDays = DefaultExpiryRiskWindowDays
	}

	sql := `SELECT EntityId, WarehouseId, ProductId, LayerId, LotId, ReceiptRefType, ReceiptRefId,
	               QtyRemaining, OriginalQty, UnitCostMinor, Currency, FxRateId, ReceivedAt, ExpiryDate, CreatedAt, UpdatedAt
	        FROM InventoryCostLayers
	        WHERE EntityId = @entityId`
	params := map[string]any{"entityId": entityID}
	if wid := strings.TrimSpace(filter.WarehouseID); wid != "" {
		sql += " AND WarehouseId = @warehouseId"
		params["warehouseId"] = wid
	}
	if pid := strings.TrimSpace(filter.ProductID); pid != "" {
		sql += " AND ProductId = @productId"
		params["productId"] = pid
	}
	if !filter.IncludeZeroQty {
		sql += " AND QtyRemaining > 0"
	}
	sql += " ORDER BY WarehouseId ASC, ProductId ASC, IF(ExpiryDate IS NULL, 1, 0) ASC, ExpiryDate ASC, ReceivedAt ASC"

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	now := time.Now().UTC()
	var layers []InventoryCostLayer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return ValuationSummaryResponse{}, fmt.Errorf("query InventoryCostLayers: %w", err)
		}
		layer, expDate, err := scanCostLayerRow(row)
		if err != nil {
			return ValuationSummaryResponse{}, err
		}
		if expDate != nil {
			days := ComputeExpiryDaysRemaining(expDate, now)
			layer.DaysToExpiry = days
			layer.NearExpiry = days != nil && *days <= winDays
		}
		layers = append(layers, layer)
	}
	return buildValuationResponse(entityID, strings.TrimSpace(filter.WarehouseID), winDays, layers), nil
}

func (r *SpannerRepository) ListMovements(ctx context.Context, filter MovementsFilter) ([]InventoryMovement, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(filter.EntityID)
	if entityID == "" {
		return nil, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	sql := `SELECT EntityId, MovementId, MovementType, ReferenceType, ReferenceId,
	               WarehouseId, ProductId, LotId, LayerId, QtyDelta, UnitCostMinor, TotalCostMinor,
	               Currency, GlEntryId, IdempotencyKey, CreatedAt
	        FROM InventoryMovements
	        WHERE EntityId = @entityId`
	params := map[string]any{
		"entityId": entityID,
		"limit":    int64(limit),
	}
	if wid := strings.TrimSpace(filter.WarehouseID); wid != "" {
		sql += " AND WarehouseId = @warehouseId"
		params["warehouseId"] = wid
	}
	if pid := strings.TrimSpace(filter.ProductID); pid != "" {
		sql += " AND ProductId = @productId"
		params["productId"] = pid
	}
	if mt := strings.ToUpper(strings.TrimSpace(filter.MovementType)); mt != "" {
		sql += " AND UPPER(MovementType) = @movementType"
		params["movementType"] = mt
	}
	sql += " ORDER BY CreatedAt DESC, MovementId DESC LIMIT @limit"

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]InventoryMovement, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query InventoryMovements: %w", err)
		}
		mv, err := scanMovementRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, mv)
	}
	return out, nil
}

func (r *SpannerRepository) ScanExpiryRisks(ctx context.Context, entityID, warehouseID string, withinDays int64, emitEvents bool) ([]ExpiryRiskLayer, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	if withinDays <= 0 {
		withinDays = DefaultExpiryRiskWindowDays
	}
	sum, err := r.GetValuationSummary(ctx, ValuationFilter{
		EntityID:       entityID,
		WarehouseID:    warehouseID,
		ExpiryRiskDays: withinDays,
	})
	if err != nil {
		return nil, err
	}
	var out []ExpiryRiskLayer
	for _, l := range sum.Layers {
		if l.NearExpiry && l.DaysToExpiry != nil && l.QtyRemaining > 0 {
			out = append(out, ExpiryRiskLayer{
				Layer:            l,
				ValueAtRiskMinor: l.TotalValueMinor,
				DaysRemaining:    *l.DaysToExpiry,
			})
		}
	}
	if !emitEvents || len(out) == 0 {
		return out, nil
	}

	now := time.Now().UTC()
	err = spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		localBuf := outbox.NewSpannerTxnBuffer(txn)
		stamped := &entityStampedOutboxBuf{entityID: strings.TrimSpace(entityID), inner: localBuf}
		for _, item := range out {
			if eErr := outbox.EmitJSON(txCtx, stamped, events.AggregateInventoryCostLayer, item.Layer.LayerID, events.TopicDispatch, events.InventoryExpiryRiskEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventInventoryExpiryRisk,
					Version:   1,
					Timestamp: now.Format(time.RFC3339Nano),
				},
				EntityID:         item.Layer.EntityID,
				WarehouseID:      item.Layer.WarehouseID,
				ProductID:        item.Layer.ProductID,
				LotID:            item.Layer.LotID,
				LayerID:          item.Layer.LayerID,
				QtyRemaining:     item.Layer.QtyRemaining,
				UnitCostMinor:    item.Layer.UnitCostMinor,
				ValueAtRiskMinor: item.ValueAtRiskMinor,
				Currency:         item.Layer.Currency,
				ExpiryDate:       item.Layer.ExpiryDate,
				DaysRemaining:    item.DaysRemaining,
				DaysToExpiry:     item.DaysRemaining,
				OccurredAt:       now.Format(time.RFC3339Nano),
			}); eErr != nil {
				return eErr
			}
		}
		return localBuf.Flush(txCtx)
	})
	if err != nil {
		return nil, fmt.Errorf("emit expiry risk outbox: %w", err)
	}
	return out, nil
}

// RecordReceiptInTxn records a single inventory receipt inside an open Spanner ReadWriteTransaction.
func RecordReceiptInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	in ReceiptInput,
) (ReceiptResult, error) {
	results, err := RecordReceiptsBatchInTxn(ctx, txn, outboxBuf, []ReceiptInput{in})
	if err != nil {
		return ReceiptResult{}, err
	}
	if len(results) == 0 {
		return ReceiptResult{}, nil
	}
	return results[0], nil
}

// RecordReceiptsBatchInTxn atomically persists a batch of receipts, coalesces their GL postings
// via gl.BufferJournalEntriesBatchInTxn (preserving a single hash chain in txn), and emits outbox events.
func RecordReceiptsBatchInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	inputs []ReceiptInput,
) ([]ReceiptResult, error) {
	if txn == nil {
		return nil, fmt.Errorf("valuation: nil spanner read-write transaction")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if len(inputs) == 0 {
		return nil, nil
	}

	now := time.Now().UTC()
	type preparedReceipt struct {
		in             ReceiptInput
		totalCost      int64
		creditAcct     string
		expiryStr      string
		expiryCivil    *civil.Date
		receivedAtIsTS bool
		layerID        string
		movementID     string
	}

	prepped := make([]preparedReceipt, len(inputs))
	keysByEntity := make(map[string][]string)
	for i, raw := range inputs {
		explicitTS := !raw.ReceivedAt.IsZero()
		norm, totalCost, creditAcct, expiryStr, expiryCivil, err := validateAndNormalizeReceipt(raw, now)
		if err != nil {
			return nil, fmt.Errorf("receipt[%d]: %w", i, err)
		}
		prepped[i] = preparedReceipt{
			in:             norm,
			totalCost:      totalCost,
			creditAcct:     creditAcct,
			expiryStr:      expiryStr,
			expiryCivil:    expiryCivil,
			receivedAtIsTS: !explicitTS,
			layerID:        "vcl_" + uuid.NewString(),
			movementID:     "vmv_" + uuid.NewString(),
		}
		keysByEntity[norm.EntityID] = append(keysByEntity[norm.EntityID], norm.IdempotencyKey)
	}

	existingByEntityKey, err := readExistingMovementsByIdemBatchInTxn(ctx, txn, keysByEntity)
	if err != nil {
		return nil, err
	}

	results := make([]ReceiptResult, len(prepped))
	var pendingIdx []int
	seenInBatch := make(map[string]int, len(prepped))
	for i, p := range prepped {
		ek := p.in.EntityID + "\x00" + p.in.IdempotencyKey
		if ex, ok := existingByEntityKey[ek]; ok && len(ex) > 0 {
			mv := ex[0]
			var layer InventoryCostLayer
			if mv.LayerID != "" {
				if l, okL, lErr := readCostLayerByPKInTxn(ctx, txn, p.in.EntityID, mv.WarehouseID, mv.ProductID, mv.LayerID); lErr == nil && okL {
					layer = l
				}
			}
			results[i] = ReceiptResult{
				Layer:            layer,
				Movement:         mv,
				GlEntryID:        mv.GlEntryID,
				TotalCostMinor:   mv.TotalCostMinor,
				VatMinor:         p.in.VatMinor,
				AlreadyProcessed: true,
			}
			continue
		}
		if firstIdx, dup := seenInBatch[ek]; dup {
			seenInBatch["dup:"+strconv.Itoa(i)] = firstIdx
			continue
		}
		seenInBatch[ek] = i
		pendingIdx = append(pendingIdx, i)
	}

	if len(pendingIdx) == 0 {
		return results, nil
	}

	var localFlush *outbox.SpannerTxnBuffer
	targetBuf := outboxBuf
	if targetBuf == nil {
		localFlush = outbox.NewSpannerTxnBuffer(txn)
		targetBuf = localFlush
	}

	glInputs := make([]gl.JournalEntryInput, 0, len(pendingIdx))
	for _, idx := range pendingIdx {
		p := prepped[idx]
		creditTotal := p.totalCost + p.in.VatMinor
		postings := []gl.PostingInput{
			{
				AccountCode: gl.AccountInventory,
				Direction:   gl.DirectionDebit,
				AmountMinor: p.totalCost,
				Currency:    p.in.Currency,
				WarehouseID: p.in.WarehouseID,
				ProductID:   p.in.ProductID,
				LotID:       p.in.LotID,
				Description: fmt.Sprintf("Inventory receipt %s:%s", p.in.ReceiptRefType, p.in.ReceiptRefID),
			},
		}
		if p.in.VatMinor > 0 {
			postings = append(postings, gl.PostingInput{
				AccountCode: gl.AccountVatInput,
				Direction:   gl.DirectionDebit,
				AmountMinor: p.in.VatMinor,
				Currency:    p.in.Currency,
				WarehouseID: p.in.WarehouseID,
				ProductID:   p.in.ProductID,
				LotID:       p.in.LotID,
				Description: "Input VAT on inventory receipt",
			})
		}
		creditPosting := gl.PostingInput{
			AccountCode: p.creditAcct,
			Direction:   gl.DirectionCredit,
			AmountMinor: creditTotal,
			Currency:    p.in.Currency,
			WarehouseID: p.in.WarehouseID,
			ProductID:   p.in.ProductID,
			LotID:       p.in.LotID,
			Description: "Vendor/GRNI accrual for inventory receipt",
		}
		if p.in.VendorID != "" {
			creditPosting.PartyType = "VENDOR"
			creditPosting.PartyID = p.in.VendorID
		}
		postings = append(postings, creditPosting)

		glInputs = append(glInputs, gl.JournalEntryInput{
			EntityID:       p.in.EntityID,
			SourceType:     "INVENTORY_RECEIPT",
			SourceID:       p.in.ReceiptRefID,
			Memo:           fmt.Sprintf("Inventory receipt %s (%s qty=%d)", p.in.ProductID, p.in.WarehouseID, p.in.Quantity),
			Currency:       p.in.Currency,
			CountryPack:    p.in.CountryPack,
			IdempotencyKey: "gl:inv_receipt:" + p.in.IdempotencyKey,
			PostedBy:       p.in.ActorID,
			OccurredAt:     p.in.ReceivedAt,
			Postings:       postings,
		})
	}

	glResults, err := gl.BufferJournalEntriesBatchInTxn(ctx, txn, targetBuf, glInputs)
	if err != nil {
		return nil, fmt.Errorf("post inventory receipt GL: %w", err)
	}

	muts := make([]*spanner.Mutation, 0, len(pendingIdx)*2)
	for j, idx := range pendingIdx {
		p := prepped[idx]
		glRes := glResults[j]

		var recAtVal any = p.in.ReceivedAt
		if p.receivedAtIsTS {
			recAtVal = spanner.CommitTimestamp
		}
		layerMap := map[string]any{
			"EntityId":       p.in.EntityID,
			"WarehouseId":    p.in.WarehouseID,
			"ProductId":      p.in.ProductID,
			"LayerId":        p.layerID,
			"LotId":          nullIfEmpty(p.in.LotID),
			"ReceiptRefType": nullIfEmpty(p.in.ReceiptRefType),
			"ReceiptRefId":   nullIfEmpty(p.in.ReceiptRefID),
			"QtyRemaining":   p.in.Quantity,
			"OriginalQty":    p.in.Quantity,
			"UnitCostMinor":  p.in.UnitCostMinor,
			"Currency":       p.in.Currency,
			"FxRateId":       nullIfEmpty(p.in.FxRateID),
			"ReceivedAt":     recAtVal,
			"CreatedAt":      spanner.CommitTimestamp,
			"UpdatedAt":      spanner.CommitTimestamp,
		}
		if p.expiryCivil != nil {
			layerMap["ExpiryDate"] = *p.expiryCivil
		} else {
			layerMap["ExpiryDate"] = spanner.NullDate{}
		}
		muts = append(muts, spanner.InsertMap("InventoryCostLayers", layerMap))

		muts = append(muts, spanner.InsertMap("InventoryMovements", map[string]any{
			"EntityId":       p.in.EntityID,
			"MovementId":     p.movementID,
			"MovementType":   MovementTypeReceipt,
			"ReferenceType":  p.in.ReceiptRefType,
			"ReferenceId":    p.in.ReceiptRefID,
			"WarehouseId":    p.in.WarehouseID,
			"ProductId":      p.in.ProductID,
			"LotId":          nullIfEmpty(p.in.LotID),
			"LayerId":        p.layerID,
			"QtyDelta":       p.in.Quantity,
			"UnitCostMinor":  p.in.UnitCostMinor,
			"TotalCostMinor": p.totalCost,
			"Currency":       p.in.Currency,
			"GlEntryId":      nullIfEmpty(glRes.EntryID),
			"IdempotencyKey": p.in.IdempotencyKey,
			"CreatedAt":      spanner.CommitTimestamp,
		}))

		daysToExp := ComputeExpiryDaysRemaining(p.expiryCivil, p.in.ReceivedAt)
		nearExpiry := daysToExp != nil && *daysToExp <= p.in.ExpiryRiskWindowDays

		layerDTO := InventoryCostLayer{
			EntityID:        p.in.EntityID,
			WarehouseID:     p.in.WarehouseID,
			ProductID:       p.in.ProductID,
			LayerID:         p.layerID,
			LotID:           p.in.LotID,
			ReceiptRefType:  p.in.ReceiptRefType,
			ReceiptRefID:    p.in.ReceiptRefID,
			QtyRemaining:    p.in.Quantity,
			OriginalQty:     p.in.Quantity,
			UnitCostMinor:   p.in.UnitCostMinor,
			TotalValueMinor: p.totalCost,
			Currency:        p.in.Currency,
			FxRateID:        p.in.FxRateID,
			ReceivedAt:      p.in.ReceivedAt,
			ExpiryDate:      p.expiryStr,
			NearExpiry:      nearExpiry,
			DaysToExpiry:    daysToExp,
			CreatedAt:       p.in.ReceivedAt,
			UpdatedAt:       p.in.ReceivedAt,
		}
		mvDTO := InventoryMovement{
			EntityID:       p.in.EntityID,
			MovementID:     p.movementID,
			MovementType:   MovementTypeReceipt,
			ReferenceType:  p.in.ReceiptRefType,
			ReferenceID:    p.in.ReceiptRefID,
			WarehouseID:    p.in.WarehouseID,
			ProductID:      p.in.ProductID,
			LotID:          p.in.LotID,
			LayerID:        p.layerID,
			QtyDelta:       p.in.Quantity,
			UnitCostMinor:  p.in.UnitCostMinor,
			TotalCostMinor: p.totalCost,
			Currency:       p.in.Currency,
			GlEntryID:      glRes.EntryID,
			IdempotencyKey: p.in.IdempotencyKey,
			CreatedAt:      p.in.ReceivedAt,
		}

		results[idx] = ReceiptResult{
			Layer:             layerDTO,
			Movement:          mvDTO,
			GlEntryID:         glRes.EntryID,
			TotalCostMinor:    p.totalCost,
			VatMinor:          p.in.VatMinor,
			ExpiryRiskEmitted: nearExpiry,
		}

		stampedBuf := &entityStampedOutboxBuf{entityID: p.in.EntityID, inner: targetBuf}
		tsStr := p.in.ReceivedAt.Format(time.RFC3339Nano)
		if eErr := outbox.EmitJSON(ctx, stampedBuf, events.AggregateInventoryCostLayer, p.layerID, events.TopicDispatch, events.InventoryReceivedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventInventoryReceived,
				Version:   1,
				Timestamp: tsStr,
			},
			EntityID:       p.in.EntityID,
			WarehouseID:    p.in.WarehouseID,
			ProductID:      p.in.ProductID,
			LotID:          p.in.LotID,
			LayerID:        p.layerID,
			MovementID:     p.movementID,
			ReferenceType:  p.in.ReceiptRefType,
			ReferenceID:    p.in.ReceiptRefID,
			Quantity:       p.in.Quantity,
			UnitCostMinor:  p.in.UnitCostMinor,
			TotalCostMinor: p.totalCost,
			Currency:       p.in.Currency,
			GlEntryID:      glRes.EntryID,
			ExpiryDate:     p.expiryStr,
			OccurredAt:     tsStr,
		}); eErr != nil {
			return nil, fmt.Errorf("emit inventory.received: %w", eErr)
		}
		if nearExpiry {
			if eErr := outbox.EmitJSON(ctx, stampedBuf, events.AggregateInventoryCostLayer, p.layerID, events.TopicDispatch, events.InventoryExpiryRiskEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventInventoryExpiryRisk,
					Version:   1,
					Timestamp: tsStr,
				},
				EntityID:         p.in.EntityID,
				WarehouseID:      p.in.WarehouseID,
				ProductID:        p.in.ProductID,
				LotID:            p.in.LotID,
				LayerID:          p.layerID,
				QtyRemaining:     p.in.Quantity,
				UnitCostMinor:    p.in.UnitCostMinor,
				ValueAtRiskMinor: p.totalCost,
				Currency:         p.in.Currency,
				ExpiryDate:       p.expiryStr,
				DaysRemaining:    *daysToExp,
				DaysToExpiry:     *daysToExp,
				OccurredAt:       tsStr,
			}); eErr != nil {
				return nil, fmt.Errorf("emit inventory.expiry.risk: %w", eErr)
			}
		}
	}

	for i := range results {
		if firstIdx, dup := seenInBatch["dup:"+strconv.Itoa(i)]; dup {
			cp := results[firstIdx]
			cp.AlreadyProcessed = true
			results[i] = cp
		}
	}

	if err := txn.BufferWrite(muts); err != nil {
		return nil, fmt.Errorf("buffer inventory receipt mutations: %w", err)
	}
	if localFlush != nil {
		if err := localFlush.Flush(ctx); err != nil {
			return nil, fmt.Errorf("flush inventory receipt outbox: %w", err)
		}
	}
	return results, nil
}

// IssueStockFEFOInTxn consumes active cost layers FEFO and posts DR COGS / CR INVENTORY atomically.
func IssueStockFEFOInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	in IssueInput,
) (IssueResult, error) {
	res, _, err := issueStockFEFOInTxn(ctx, txn, outboxBuf, in, false)
	return res, err
}

// IssueStockFEFOIfPresentInTxn consumes active cost layers if any exist for (EntityId, WarehouseId, ProductId).
// When zero cost layers exist, it returns (IssueResult{}, false, nil) so legacy WMS tests remain intact.
func IssueStockFEFOIfPresentInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	in IssueInput,
) (IssueResult, bool, error) {
	return issueStockFEFOInTxn(ctx, txn, outboxBuf, in, true)
}

func issueStockFEFOInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	in IssueInput,
	skipIfNoLayers bool,
) (IssueResult, bool, error) {
	if txn == nil {
		return IssueResult{}, false, fmt.Errorf("valuation: nil spanner read-write transaction")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	norm, err := validateAndNormalizeIssue(in, time.Now().UTC())
	if err != nil {
		return IssueResult{}, false, err
	}

	existingMvs, err := readExistingIssueMovementsByIdemInTxn(ctx, txn, norm.EntityID, norm.IdempotencyKey)
	if err != nil {
		return IssueResult{}, false, err
	}
	if len(existingMvs) > 0 {
		return buildIssueResultFromMovements(existingMvs, true), true, nil
	}

	layers, err := queryActiveLayersFEFOInTxn(ctx, txn, norm.EntityID, norm.WarehouseID, norm.ProductID, norm.LotID)
	if err != nil {
		return IssueResult{}, false, err
	}
	if len(layers) == 0 {
		if skipIfNoLayers {
			return IssueResult{}, false, nil
		}
		return IssueResult{}, false, fmt.Errorf("%w: no active cost layers for (%s/%s)", ErrInsufficientCostLayers, norm.WarehouseID, norm.ProductID)
	}

	var totalAvail, totalVal int64
	for _, l := range layers {
		totalAvail += l.QtyRemaining
		totalVal += l.QtyRemaining * l.UnitCostMinor
	}
	if totalAvail < norm.Quantity {
		return IssueResult{}, false, fmt.Errorf("%w: have %d need %d for (%s/%s)", ErrInsufficientCostLayers, totalAvail, norm.Quantity, norm.WarehouseID, norm.ProductID)
	}

	var localFlush *outbox.SpannerTxnBuffer
	targetBuf := outboxBuf
	if targetBuf == nil {
		localFlush = outbox.NewSpannerTxnBuffer(txn)
		targetBuf = localFlush
	}

	type layerConsumption struct {
		layer    InventoryCostLayer
		take     int64
		unitCost int64
		cost     int64
	}
	var consumptions []layerConsumption
	var totalCostMinor int64
	remaining := norm.Quantity

	if strings.EqualFold(norm.CostingMethod, CostingMethodMovingAverage) {
		targetTotalCost := computeProportionalCost(totalVal, totalAvail, norm.Quantity)
		avgUnit := ComputeMovingAverageUnitCostMinor(targetTotalCost, norm.Quantity)
		remCost := targetTotalCost
		for i := range layers {
			if remaining == 0 {
				break
			}
			take := layers[i].QtyRemaining
			if take > remaining {
				take = remaining
			}
			remaining -= take
			layers[i].QtyRemaining -= take
			var lc int64
			if remaining == 0 {
				lc = remCost
			} else {
				lc = take * avgUnit
				if lc > remCost {
					lc = remCost
				}
			}
			remCost -= lc
			totalCostMinor += lc
			consumptions = append(consumptions, layerConsumption{
				layer:    layers[i],
				take:     take,
				unitCost: ComputeMovingAverageUnitCostMinor(lc, take),
				cost:     lc,
			})
		}
	} else {
		for i := range layers {
			if remaining == 0 {
				break
			}
			take := layers[i].QtyRemaining
			if take > remaining {
				take = remaining
			}
			remaining -= take
			layers[i].QtyRemaining -= take
			lc := take * layers[i].UnitCostMinor
			totalCostMinor += lc
			consumptions = append(consumptions, layerConsumption{
				layer:    layers[i],
				take:     take,
				unitCost: layers[i].UnitCostMinor,
				cost:     lc,
			})
		}
	}

	var glEntryID string
	if totalCostMinor > 0 {
		glRes, err := gl.BufferJournalEntryInTxn(ctx, txn, targetBuf, gl.JournalEntryInput{
			EntityID:       norm.EntityID,
			SourceType:     "INVENTORY_ISSUE",
			SourceID:       norm.ReferenceID,
			OrderID:        norm.OrderID,
			Memo:           fmt.Sprintf("Inventory COGS issue %s (%s qty=%d)", norm.ProductID, norm.WarehouseID, norm.Quantity),
			Currency:       norm.Currency,
			CountryPack:    norm.CountryPack,
			IdempotencyKey: "gl:inv_issue:" + norm.IdempotencyKey,
			PostedBy:       norm.ActorID,
			OccurredAt:     norm.OccurredAt,
			Postings: []gl.PostingInput{
				{
					AccountCode: gl.AccountCOGS,
					Direction:   gl.DirectionDebit,
					AmountMinor: totalCostMinor,
					Currency:    norm.Currency,
					WarehouseID: norm.WarehouseID,
					ProductID:   norm.ProductID,
					LotID:       norm.LotID,
					Description: "Cost of goods sold (FEFO)",
				},
				{
					AccountCode: gl.AccountInventory,
					Direction:   gl.DirectionCredit,
					AmountMinor: totalCostMinor,
					Currency:    norm.Currency,
					WarehouseID: norm.WarehouseID,
					ProductID:   norm.ProductID,
					LotID:       norm.LotID,
					Description: "Inventory reduction on issue",
				},
			},
		})
		if err != nil {
			return IssueResult{}, false, fmt.Errorf("post inventory issue GL: %w", err)
		}
		glEntryID = glRes.EntryID
	}

	muts := make([]*spanner.Mutation, 0, len(consumptions)*2+2)
	if strings.EqualFold(norm.CostingMethod, CostingMethodMovingAverage) {
		remQty := totalAvail - norm.Quantity
		remVal := totalVal - totalCostMinor
		muts = append(muts, buildMovingAvgLayerMutations(layers, remQty, remVal)...)
	} else {
		for _, c := range consumptions {
			muts = append(muts, spanner.UpdateMap("InventoryCostLayers", map[string]any{
				"EntityId":     c.layer.EntityID,
				"WarehouseId":  c.layer.WarehouseID,
				"ProductId":    c.layer.ProductID,
				"LayerId":      c.layer.LayerID,
				"QtyRemaining": c.layer.QtyRemaining,
				"UpdatedAt":    spanner.CommitTimestamp,
			}))
		}
	}

	created := make([]InventoryMovement, 0, len(consumptions))
	for idx, c := range consumptions {
		mvID := "vmv_" + uuid.NewString()
		mvKey := norm.IdempotencyKey
		if idx > 0 {
			mvKey = norm.IdempotencyKey + "#layer:" + strconv.Itoa(idx)
		}
		muts = append(muts, spanner.InsertMap("InventoryMovements", map[string]any{
			"EntityId":       norm.EntityID,
			"MovementId":     mvID,
			"MovementType":   MovementTypeIssueCOGS,
			"ReferenceType":  norm.ReferenceType,
			"ReferenceId":    norm.ReferenceID,
			"WarehouseId":    norm.WarehouseID,
			"ProductId":      norm.ProductID,
			"LotId":          nullIfEmpty(c.layer.LotID),
			"LayerId":        c.layer.LayerID,
			"QtyDelta":       -c.take,
			"UnitCostMinor":  c.unitCost,
			"TotalCostMinor": c.cost,
			"Currency":       norm.Currency,
			"GlEntryId":      nullIfEmpty(glEntryID),
			"IdempotencyKey": mvKey,
			"CreatedAt":      spanner.CommitTimestamp,
		}))
		created = append(created, InventoryMovement{
			EntityID:       norm.EntityID,
			MovementID:     mvID,
			MovementType:   MovementTypeIssueCOGS,
			ReferenceType:  norm.ReferenceType,
			ReferenceID:    norm.ReferenceID,
			WarehouseID:    norm.WarehouseID,
			ProductID:      norm.ProductID,
			LotID:          c.layer.LotID,
			LayerID:        c.layer.LayerID,
			QtyDelta:       -c.take,
			UnitCostMinor:  c.unitCost,
			TotalCostMinor: c.cost,
			Currency:       norm.Currency,
			GlEntryID:      glEntryID,
			IdempotencyKey: mvKey,
			CreatedAt:      norm.OccurredAt,
		})
	}

	if err := txn.BufferWrite(muts); err != nil {
		return IssueResult{}, false, fmt.Errorf("buffer inventory issue mutations: %w", err)
	}

	if len(created) > 0 {
		stampedBuf := &entityStampedOutboxBuf{entityID: norm.EntityID, inner: targetBuf}
		tsStr := norm.OccurredAt.Format(time.RFC3339Nano)
		if eErr := outbox.EmitJSON(ctx, stampedBuf, events.AggregateInventoryMovement, created[0].MovementID, events.TopicDispatch, events.InventoryIssuedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventInventoryIssued,
				Version:   1,
				Timestamp: tsStr,
			},
			EntityID:       norm.EntityID,
			WarehouseID:    norm.WarehouseID,
			ProductID:      norm.ProductID,
			LotID:          created[0].LotID,
			LayerID:        created[0].LayerID,
			MovementID:     created[0].MovementID,
			ReferenceType:  norm.ReferenceType,
			ReferenceID:    norm.ReferenceID,
			OrderID:        norm.OrderID,
			Quantity:       norm.Quantity,
			UnitCostMinor:  ComputeMovingAverageUnitCostMinor(totalCostMinor, norm.Quantity),
			TotalCostMinor: totalCostMinor,
			Currency:       norm.Currency,
			GlEntryID:      glEntryID,
			OccurredAt:     tsStr,
		}); eErr != nil {
			return IssueResult{}, false, fmt.Errorf("emit inventory.issued: %w", eErr)
		}
	}

	if localFlush != nil {
		if err := localFlush.Flush(ctx); err != nil {
			return IssueResult{}, false, fmt.Errorf("flush inventory issue outbox: %w", err)
		}
	}
	return buildIssueResultFromMovements(created, false), true, nil
}

// RecordAdjustmentCostInTxn updates cost layers for approved stock adjustments and posts DR EXP:SHRINK / CR INVENTORY (or inverse).
func RecordAdjustmentCostInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	in AdjustmentCostInput,
) (AdjustmentCostResult, error) {
	res, _, err := recordAdjustmentCostInTxn(ctx, txn, outboxBuf, in, false)
	return res, err
}

// RecordAdjustmentCostIfPresentInTxn posts adjustment cost when active layers exist (or positive unit cost is supplied).
func RecordAdjustmentCostIfPresentInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	in AdjustmentCostInput,
) (AdjustmentCostResult, bool, error) {
	return recordAdjustmentCostInTxn(ctx, txn, outboxBuf, in, true)
}

func recordAdjustmentCostInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	in AdjustmentCostInput,
	skipIfNoLayers bool,
) (AdjustmentCostResult, bool, error) {
	if txn == nil {
		return AdjustmentCostResult{}, false, fmt.Errorf("valuation: nil spanner read-write transaction")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	norm, err := validateAndNormalizeAdjustment(in, time.Now().UTC())
	if err != nil {
		return AdjustmentCostResult{}, false, err
	}

	existingMvs, err := readExistingIssueMovementsByIdemInTxn(ctx, txn, norm.EntityID, norm.IdempotencyKey)
	if err != nil {
		return AdjustmentCostResult{}, false, err
	}
	if len(existingMvs) > 0 {
		var total int64
		var glID string
		for _, mv := range existingMvs {
			total += mv.TotalCostMinor
			if glID == "" {
				glID = mv.GlEntryID
			}
		}
		return AdjustmentCostResult{
			Movements:        existingMvs,
			TotalCostMinor:   total,
			GlEntryID:        glID,
			AlreadyProcessed: true,
		}, true, nil
	}

	layers, err := queryActiveLayersFEFOInTxn(ctx, txn, norm.EntityID, norm.WarehouseID, norm.ProductID, norm.LotID)
	if err != nil {
		return AdjustmentCostResult{}, false, err
	}

	var localFlush *outbox.SpannerTxnBuffer
	targetBuf := outboxBuf
	if targetBuf == nil {
		localFlush = outbox.NewSpannerTxnBuffer(txn)
		targetBuf = localFlush
	}

	if norm.DeltaQty < 0 {
		need := -norm.DeltaQty
		if len(layers) == 0 {
			if skipIfNoLayers {
				return AdjustmentCostResult{}, false, nil
			}
			return AdjustmentCostResult{}, false, fmt.Errorf("%w: no cost layers for adjustment %s", ErrInsufficientCostLayers, norm.AdjustmentID)
		}
		var avail int64
		for _, l := range layers {
			avail += l.QtyRemaining
		}
		if avail < need {
			return AdjustmentCostResult{}, false, fmt.Errorf("%w: shrink %d exceeds active qty %d", ErrInsufficientCostLayers, need, avail)
		}

		remaining := need
		var totalCost int64
		type adjConsume struct {
			layer InventoryCostLayer
			take  int64
			cost  int64
		}
		var consumed []adjConsume
		for i := range layers {
			if remaining == 0 {
				break
			}
			take := layers[i].QtyRemaining
			if take > remaining {
				take = remaining
			}
			remaining -= take
			layers[i].QtyRemaining -= take
			c := take * layers[i].UnitCostMinor
			totalCost += c
			consumed = append(consumed, adjConsume{layer: layers[i], take: take, cost: c})
		}

		var glEntryID string
		if totalCost > 0 {
			glRes, gErr := gl.BufferJournalEntryInTxn(ctx, txn, targetBuf, gl.JournalEntryInput{
				EntityID:       norm.EntityID,
				SourceType:     "INVENTORY_ADJUSTMENT",
				SourceID:       norm.AdjustmentID,
				Memo:           fmt.Sprintf("Inventory shrink adjustment %s (%s)", norm.ProductID, norm.ReasonCode),
				Currency:       norm.Currency,
				CountryPack:    norm.CountryPack,
				IdempotencyKey: "gl:inv_adj:" + norm.IdempotencyKey,
				PostedBy:       norm.ActorID,
				OccurredAt:     norm.OccurredAt,
				Postings: []gl.PostingInput{
					{
						AccountCode: gl.AccountShrinkExpense,
						Direction:   gl.DirectionDebit,
						AmountMinor: totalCost,
						Currency:    norm.Currency,
						WarehouseID: norm.WarehouseID,
						ProductID:   norm.ProductID,
						LotID:       norm.LotID,
						Description: "Inventory shrinkage adjustment",
					},
					{
						AccountCode: gl.AccountInventory,
						Direction:   gl.DirectionCredit,
						AmountMinor: totalCost,
						Currency:    norm.Currency,
						WarehouseID: norm.WarehouseID,
						ProductID:   norm.ProductID,
						LotID:       norm.LotID,
						Description: "Inventory reduction from adjustment",
					},
				},
			})
			if gErr != nil {
				return AdjustmentCostResult{}, false, fmt.Errorf("post inventory shrink GL: %w", gErr)
			}
			glEntryID = glRes.EntryID
		}

		muts := make([]*spanner.Mutation, 0, len(consumed)*2)
		created := make([]InventoryMovement, 0, len(consumed))
		for idx, c := range consumed {
			muts = append(muts, spanner.UpdateMap("InventoryCostLayers", map[string]any{
				"EntityId":     c.layer.EntityID,
				"WarehouseId":  c.layer.WarehouseID,
				"ProductId":    c.layer.ProductID,
				"LayerId":      c.layer.LayerID,
				"QtyRemaining": c.layer.QtyRemaining,
				"UpdatedAt":    spanner.CommitTimestamp,
			}))
			mvID := "vmv_" + uuid.NewString()
			mvKey := norm.IdempotencyKey
			if idx > 0 {
				mvKey = norm.IdempotencyKey + "#layer:" + strconv.Itoa(idx)
			}
			muts = append(muts, spanner.InsertMap("InventoryMovements", map[string]any{
				"EntityId":       norm.EntityID,
				"MovementId":     mvID,
				"MovementType":   MovementTypeAdjustmentShrink,
				"ReferenceType":  "INVENTORY_ADJUSTMENT",
				"ReferenceId":    norm.AdjustmentID,
				"WarehouseId":    norm.WarehouseID,
				"ProductId":      norm.ProductID,
				"LotId":          nullIfEmpty(c.layer.LotID),
				"LayerId":        c.layer.LayerID,
				"QtyDelta":       -c.take,
				"UnitCostMinor":  c.layer.UnitCostMinor,
				"TotalCostMinor": c.cost,
				"Currency":       norm.Currency,
				"GlEntryId":      nullIfEmpty(glEntryID),
				"IdempotencyKey": mvKey,
				"CreatedAt":      spanner.CommitTimestamp,
			}))
			created = append(created, InventoryMovement{
				EntityID:       norm.EntityID,
				MovementID:     mvID,
				MovementType:   MovementTypeAdjustmentShrink,
				ReferenceType:  "INVENTORY_ADJUSTMENT",
				ReferenceID:    norm.AdjustmentID,
				WarehouseID:    norm.WarehouseID,
				ProductID:      norm.ProductID,
				LotID:          c.layer.LotID,
				LayerID:        c.layer.LayerID,
				QtyDelta:       -c.take,
				UnitCostMinor:  c.layer.UnitCostMinor,
				TotalCostMinor: c.cost,
				Currency:       norm.Currency,
				GlEntryID:      glEntryID,
				IdempotencyKey: mvKey,
				CreatedAt:      norm.OccurredAt,
			})
		}

		if err := txn.BufferWrite(muts); err != nil {
			return AdjustmentCostResult{}, false, fmt.Errorf("buffer inventory shrink mutations: %w", err)
		}
		if len(created) > 0 {
			stampedBuf := &entityStampedOutboxBuf{entityID: norm.EntityID, inner: targetBuf}
			tsStr := norm.OccurredAt.Format(time.RFC3339Nano)
			if eErr := outbox.EmitJSON(ctx, stampedBuf, events.AggregateInventoryMovement, created[0].MovementID, events.TopicDispatch, events.InventoryIssuedEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventInventoryIssued,
					Version:   1,
					Timestamp: tsStr,
				},
				EntityID:       norm.EntityID,
				WarehouseID:    norm.WarehouseID,
				ProductID:      norm.ProductID,
				LotID:          created[0].LotID,
				LayerID:        created[0].LayerID,
				MovementID:     created[0].MovementID,
				ReferenceType:  "INVENTORY_ADJUSTMENT",
				ReferenceID:    norm.AdjustmentID,
				Quantity:       need,
				UnitCostMinor:  ComputeMovingAverageUnitCostMinor(totalCost, need),
				TotalCostMinor: totalCost,
				Currency:       norm.Currency,
				GlEntryID:      glEntryID,
				OccurredAt:     tsStr,
			}); eErr != nil {
				return AdjustmentCostResult{}, false, fmt.Errorf("emit inventory.issued for shrink: %w", eErr)
			}
		}
		if localFlush != nil {
			if err := localFlush.Flush(ctx); err != nil {
				return AdjustmentCostResult{}, false, err
			}
		}
		return AdjustmentCostResult{
			Movements:      created,
			TotalCostMinor: totalCost,
			GlEntryID:      glEntryID,
		}, true, nil
	}

	// Positive delta (surplus gain).
	unitCost := norm.UnitCostMinor
	if unitCost <= 0 {
		var totalQ, totalV int64
		for _, l := range layers {
			totalQ += l.QtyRemaining
			totalV += l.QtyRemaining * l.UnitCostMinor
		}
		unitCost = ComputeMovingAverageUnitCostMinor(totalV, totalQ)
	}
	if unitCost <= 0 {
		if skipIfNoLayers {
			return AdjustmentCostResult{}, false, nil
		}
		return AdjustmentCostResult{}, false, fmt.Errorf("%w: unit_cost_minor required for positive adjustment %s", ErrInvalidInput, norm.AdjustmentID)
	}

	totalCost := norm.DeltaQty * unitCost
	glRes, gErr := gl.BufferJournalEntryInTxn(ctx, txn, targetBuf, gl.JournalEntryInput{
		EntityID:       norm.EntityID,
		SourceType:     "INVENTORY_ADJUSTMENT",
		SourceID:       norm.AdjustmentID,
		Memo:           fmt.Sprintf("Inventory gain adjustment %s (%s)", norm.ProductID, norm.ReasonCode),
		Currency:       norm.Currency,
		CountryPack:    norm.CountryPack,
		IdempotencyKey: "gl:inv_adj:" + norm.IdempotencyKey,
		PostedBy:       norm.ActorID,
		OccurredAt:     norm.OccurredAt,
		Postings: []gl.PostingInput{
			{
				AccountCode: gl.AccountInventory,
				Direction:   gl.DirectionDebit,
				AmountMinor: totalCost,
				Currency:    norm.Currency,
				WarehouseID: norm.WarehouseID,
				ProductID:   norm.ProductID,
				LotID:       norm.LotID,
				Description: "Inventory gain adjustment",
			},
			{
				AccountCode: gl.AccountShrinkExpense,
				Direction:   gl.DirectionCredit,
				AmountMinor: totalCost,
				Currency:    norm.Currency,
				WarehouseID: norm.WarehouseID,
				ProductID:   norm.ProductID,
				LotID:       norm.LotID,
				Description: "Inventory surplus offset",
			},
		},
	})
	if gErr != nil {
		return AdjustmentCostResult{}, false, fmt.Errorf("post inventory gain GL: %w", gErr)
	}

	layerID := "vcl_" + uuid.NewString()
	mvID := "vmv_" + uuid.NewString()
	muts := []*spanner.Mutation{
		spanner.InsertMap("InventoryCostLayers", map[string]any{
			"EntityId":       norm.EntityID,
			"WarehouseId":    norm.WarehouseID,
			"ProductId":      norm.ProductID,
			"LayerId":        layerID,
			"LotId":          nullIfEmpty(norm.LotID),
			"ReceiptRefType": "INVENTORY_ADJUSTMENT",
			"ReceiptRefId":   norm.AdjustmentID,
			"QtyRemaining":   norm.DeltaQty,
			"OriginalQty":    norm.DeltaQty,
			"UnitCostMinor":  unitCost,
			"Currency":       norm.Currency,
			"FxRateId":       spanner.NullString{},
			"ReceivedAt":     spanner.CommitTimestamp,
			"ExpiryDate":     spanner.NullDate{},
			"CreatedAt":      spanner.CommitTimestamp,
			"UpdatedAt":      spanner.CommitTimestamp,
		}),
		spanner.InsertMap("InventoryMovements", map[string]any{
			"EntityId":       norm.EntityID,
			"MovementId":     mvID,
			"MovementType":   MovementTypeAdjustmentGain,
			"ReferenceType":  "INVENTORY_ADJUSTMENT",
			"ReferenceId":    norm.AdjustmentID,
			"WarehouseId":    norm.WarehouseID,
			"ProductId":      norm.ProductID,
			"LotId":          nullIfEmpty(norm.LotID),
			"LayerId":        layerID,
			"QtyDelta":       norm.DeltaQty,
			"UnitCostMinor":  unitCost,
			"TotalCostMinor": totalCost,
			"Currency":       norm.Currency,
			"GlEntryId":      nullIfEmpty(glRes.EntryID),
			"IdempotencyKey": norm.IdempotencyKey,
			"CreatedAt":      spanner.CommitTimestamp,
		}),
	}
	if err := txn.BufferWrite(muts); err != nil {
		return AdjustmentCostResult{}, false, fmt.Errorf("buffer inventory gain mutations: %w", err)
	}

	stampedBuf := &entityStampedOutboxBuf{entityID: norm.EntityID, inner: targetBuf}
	tsStr := norm.OccurredAt.Format(time.RFC3339Nano)
	if eErr := outbox.EmitJSON(ctx, stampedBuf, events.AggregateInventoryCostLayer, layerID, events.TopicDispatch, events.InventoryReceivedEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventInventoryReceived,
			Version:   1,
			Timestamp: tsStr,
		},
		EntityID:       norm.EntityID,
		WarehouseID:    norm.WarehouseID,
		ProductID:      norm.ProductID,
		LotID:          norm.LotID,
		LayerID:        layerID,
		MovementID:     mvID,
		ReferenceType:  "INVENTORY_ADJUSTMENT",
		ReferenceID:    norm.AdjustmentID,
		Quantity:       norm.DeltaQty,
		UnitCostMinor:  unitCost,
		TotalCostMinor: totalCost,
		Currency:       norm.Currency,
		GlEntryID:      glRes.EntryID,
		OccurredAt:     tsStr,
	}); eErr != nil {
		return AdjustmentCostResult{}, false, fmt.Errorf("emit inventory.received for gain: %w", eErr)
	}
	if localFlush != nil {
		if err := localFlush.Flush(ctx); err != nil {
			return AdjustmentCostResult{}, false, err
		}
	}

	layer := InventoryCostLayer{
		EntityID:        norm.EntityID,
		WarehouseID:     norm.WarehouseID,
		ProductID:       norm.ProductID,
		LayerID:         layerID,
		LotID:           norm.LotID,
		ReceiptRefType:  "INVENTORY_ADJUSTMENT",
		ReceiptRefID:    norm.AdjustmentID,
		QtyRemaining:    norm.DeltaQty,
		OriginalQty:     norm.DeltaQty,
		UnitCostMinor:   unitCost,
		TotalValueMinor: totalCost,
		Currency:        norm.Currency,
		ReceivedAt:      norm.OccurredAt,
		CreatedAt:       norm.OccurredAt,
		UpdatedAt:       norm.OccurredAt,
	}
	mv := InventoryMovement{
		EntityID:       norm.EntityID,
		MovementID:     mvID,
		MovementType:   MovementTypeAdjustmentGain,
		ReferenceType:  "INVENTORY_ADJUSTMENT",
		ReferenceID:    norm.AdjustmentID,
		WarehouseID:    norm.WarehouseID,
		ProductID:      norm.ProductID,
		LotID:          norm.LotID,
		LayerID:        layerID,
		QtyDelta:       norm.DeltaQty,
		UnitCostMinor:  unitCost,
		TotalCostMinor: totalCost,
		Currency:       norm.Currency,
		GlEntryID:      glRes.EntryID,
		IdempotencyKey: norm.IdempotencyKey,
		CreatedAt:      norm.OccurredAt,
	}
	return AdjustmentCostResult{
		Movements:      []InventoryMovement{mv},
		Layer:          &layer,
		TotalCostMinor: totalCost,
		GlEntryID:      glRes.EntryID,
	}, true, nil
}

func buildMovingAvgLayerMutations(layers []InventoryCostLayer, remQty, remVal int64) []*spanner.Mutation {
	var muts []*spanner.Mutation
	if remQty <= 0 {
		for _, l := range layers {
			muts = append(muts, spanner.UpdateMap("InventoryCostLayers", map[string]any{
				"EntityId":     l.EntityID,
				"WarehouseId":  l.WarehouseID,
				"ProductId":    l.ProductID,
				"LayerId":      l.LayerID,
				"QtyRemaining": int64(0),
				"UpdatedAt":    spanner.CommitTimestamp,
			}))
		}
		return muts
	}
	avg := remVal / remQty
	rem := remVal % remQty
	lastActive := -1
	for i, l := range layers {
		if l.QtyRemaining > 0 {
			lastActive = i
		}
	}
	for i, l := range layers {
		if l.QtyRemaining <= 0 {
			muts = append(muts, spanner.UpdateMap("InventoryCostLayers", map[string]any{
				"EntityId":     l.EntityID,
				"WarehouseId":  l.WarehouseID,
				"ProductId":    l.ProductID,
				"LayerId":      l.LayerID,
				"QtyRemaining": int64(0),
				"UpdatedAt":    spanner.CommitTimestamp,
			}))
			continue
		}
		if i == lastActive && rem != 0 {
			if l.QtyRemaining == 1 {
				muts = append(muts, spanner.UpdateMap("InventoryCostLayers", map[string]any{
					"EntityId":      l.EntityID,
					"WarehouseId":   l.WarehouseID,
					"ProductId":     l.ProductID,
					"LayerId":       l.LayerID,
					"QtyRemaining":  l.QtyRemaining,
					"UnitCostMinor": avg + rem,
					"UpdatedAt":     spanner.CommitTimestamp,
				}))
			} else {
				muts = append(muts, spanner.UpdateMap("InventoryCostLayers", map[string]any{
					"EntityId":      l.EntityID,
					"WarehouseId":   l.WarehouseID,
					"ProductId":     l.ProductID,
					"LayerId":       l.LayerID,
					"QtyRemaining":  l.QtyRemaining - 1,
					"UnitCostMinor": avg,
					"UpdatedAt":     spanner.CommitTimestamp,
				}))
				remLayer := map[string]any{
					"EntityId":       l.EntityID,
					"WarehouseId":    l.WarehouseID,
					"ProductId":      l.ProductID,
					"LayerId":        "vcl_" + uuid.NewString(),
					"LotId":          nullIfEmpty(l.LotID),
					"ReceiptRefType": nullIfEmpty(l.ReceiptRefType),
					"ReceiptRefId":   nullIfEmpty(l.ReceiptRefID),
					"QtyRemaining":   int64(1),
					"OriginalQty":    int64(1),
					"UnitCostMinor":  avg + rem,
					"Currency":       l.Currency,
					"FxRateId":       nullIfEmpty(l.FxRateID),
					"ReceivedAt":     l.ReceivedAt,
					"CreatedAt":      spanner.CommitTimestamp,
					"UpdatedAt":      spanner.CommitTimestamp,
				}
				if d, dErr := ParseExpiryDate(l.ExpiryDate); dErr == nil && d != nil {
					remLayer["ExpiryDate"] = *d
				} else {
					remLayer["ExpiryDate"] = spanner.NullDate{}
				}
				muts = append(muts, spanner.InsertMap("InventoryCostLayers", remLayer))
			}
			continue
		}
		muts = append(muts, spanner.UpdateMap("InventoryCostLayers", map[string]any{
			"EntityId":      l.EntityID,
			"WarehouseId":   l.WarehouseID,
			"ProductId":     l.ProductID,
			"LayerId":       l.LayerID,
			"QtyRemaining":  l.QtyRemaining,
			"UnitCostMinor": avg,
			"UpdatedAt":     spanner.CommitTimestamp,
		}))
	}
	return muts
}

func queryActiveLayersFEFOInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	entityID, warehouseID, productID, preferredLotID string,
) ([]InventoryCostLayer, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT EntityId, WarehouseId, ProductId, LayerId, LotId, ReceiptRefType, ReceiptRefId,
		             QtyRemaining, OriginalQty, UnitCostMinor, Currency, FxRateId, ReceivedAt, ExpiryDate, CreatedAt, UpdatedAt
		      FROM InventoryCostLayers
		      WHERE EntityId = @entityId
		        AND WarehouseId = @warehouseId
		        AND ProductId = @productId
		        AND QtyRemaining > 0
		      ORDER BY IF(ExpiryDate IS NULL, 1, 0) ASC, ExpiryDate ASC, ReceivedAt ASC, LayerId ASC`,
		Params: map[string]any{
			"entityId":    entityID,
			"warehouseId": warehouseID,
			"productId":   productID,
		},
	})
	defer iter.Stop()

	var preferred []InventoryCostLayer
	var rest []InventoryCostLayer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query active InventoryCostLayers: %w", err)
		}
		l, _, err := scanCostLayerRow(row)
		if err != nil {
			return nil, err
		}
		if preferredLotID != "" && l.LotID == preferredLotID {
			preferred = append(preferred, l)
		} else {
			rest = append(rest, l)
		}
	}
	if len(preferred) > 0 {
		return append(preferred, rest...), nil
	}
	return rest, nil
}

func readCostLayerByPKInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	entityID, warehouseID, productID, layerID string,
) (InventoryCostLayer, bool, error) {
	row, err := txn.ReadRow(ctx, "InventoryCostLayers",
		spanner.Key{entityID, warehouseID, productID, layerID},
		[]string{
			"EntityId", "WarehouseId", "ProductId", "LayerId", "LotId", "ReceiptRefType", "ReceiptRefId",
			"QtyRemaining", "OriginalQty", "UnitCostMinor", "Currency", "FxRateId", "ReceivedAt", "ExpiryDate", "CreatedAt", "UpdatedAt",
		})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return InventoryCostLayer{}, false, nil
		}
		return InventoryCostLayer{}, false, err
	}
	l, _, err := scanCostLayerRow(row)
	if err != nil {
		return InventoryCostLayer{}, false, err
	}
	return l, true, nil
}

func readExistingMovementsByIdemBatchInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	keysByEntity map[string][]string,
) (map[string][]InventoryMovement, error) {
	out := make(map[string][]InventoryMovement)
	for entityID, keys := range keysByEntity {
		uniq := uniqueStrings(keys)
		if len(uniq) == 0 {
			continue
		}
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT EntityId, MovementId, MovementType, ReferenceType, ReferenceId,
			             WarehouseId, ProductId, LotId, LayerId, QtyDelta, UnitCostMinor, TotalCostMinor,
			             Currency, GlEntryId, IdempotencyKey, CreatedAt
			      FROM InventoryMovements@{FORCE_INDEX=UQ_InventoryMovements_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
			      WHERE EntityId = @entityId AND IdempotencyKey IN UNNEST(@keys)`,
			Params: map[string]any{
				"entityId": entityID,
				"keys":     uniq,
			},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return nil, fmt.Errorf("read InventoryMovements idempotency: %w", err)
			}
			mv, err := scanMovementRow(row)
			if err != nil {
				iter.Stop()
				return nil, err
			}
			k := mv.EntityID + "\x00" + mv.IdempotencyKey
			out[k] = append(out[k], mv)
		}
		iter.Stop()
	}
	return out, nil
}

func readExistingIssueMovementsByIdemInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	entityID, baseIdemKey string,
) ([]InventoryMovement, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT EntityId, MovementId, MovementType, ReferenceType, ReferenceId,
		             WarehouseId, ProductId, LotId, LayerId, QtyDelta, UnitCostMinor, TotalCostMinor,
		             Currency, GlEntryId, IdempotencyKey, CreatedAt
		      FROM InventoryMovements
		      WHERE EntityId = @entityId
		        AND (IdempotencyKey = @baseKey OR STARTS_WITH(IdempotencyKey, @prefix))
		      ORDER BY IdempotencyKey ASC`,
		Params: map[string]any{
			"entityId": entityID,
			"baseKey":  baseIdemKey,
			"prefix":   baseIdemKey + "#layer:",
		},
	})
	defer iter.Stop()

	var out []InventoryMovement
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read existing issue movements: %w", err)
		}
		mv, err := scanMovementRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, mv)
	}
	return out, nil
}

func scanCostLayerRow(row *spanner.Row) (InventoryCostLayer, *civil.Date, error) {
	var l InventoryCostLayer
	var lotID, refType, refID, fxRate spanner.NullString
	var expDate spanner.NullDate
	if err := row.Columns(
		&l.EntityID, &l.WarehouseID, &l.ProductID, &l.LayerID,
		&lotID, &refType, &refID,
		&l.QtyRemaining, &l.OriginalQty, &l.UnitCostMinor, &l.Currency, &fxRate,
		&l.ReceivedAt, &expDate, &l.CreatedAt, &l.UpdatedAt,
	); err != nil {
		return InventoryCostLayer{}, nil, fmt.Errorf("scan InventoryCostLayer: %w", err)
	}
	if lotID.Valid {
		l.LotID = lotID.StringVal
	}
	if refType.Valid {
		l.ReceiptRefType = refType.StringVal
	}
	if refID.Valid {
		l.ReceiptRefID = refID.StringVal
	}
	if fxRate.Valid {
		l.FxRateID = fxRate.StringVal
	}
	l.TotalValueMinor = l.QtyRemaining * l.UnitCostMinor
	var expPtr *civil.Date
	if expDate.Valid {
		d := expDate.Date
		expPtr = &d
		l.ExpiryDate = d.String()
	}
	return l, expPtr, nil
}

func scanMovementRow(row *spanner.Row) (InventoryMovement, error) {
	var mv InventoryMovement
	var lotID, layerID, glEntryID, idem spanner.NullString
	if err := row.Columns(
		&mv.EntityID, &mv.MovementID, &mv.MovementType, &mv.ReferenceType, &mv.ReferenceID,
		&mv.WarehouseID, &mv.ProductID, &lotID, &layerID,
		&mv.QtyDelta, &mv.UnitCostMinor, &mv.TotalCostMinor,
		&mv.Currency, &glEntryID, &idem, &mv.CreatedAt,
	); err != nil {
		return InventoryMovement{}, fmt.Errorf("scan InventoryMovement: %w", err)
	}
	if lotID.Valid {
		mv.LotID = lotID.StringVal
	}
	if layerID.Valid {
		mv.LayerID = layerID.StringVal
	}
	if glEntryID.Valid {
		mv.GlEntryID = glEntryID.StringVal
	}
	if idem.Valid {
		mv.IdempotencyKey = idem.StringVal
	}
	return mv, nil
}

type entityStampedOutboxBuf struct {
	inner    outbox.TxnBuffer
	entityID string
}

func (b *entityStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.entityID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.entityID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

func nullIfEmpty(s string) spanner.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: s, Valid: true}
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
