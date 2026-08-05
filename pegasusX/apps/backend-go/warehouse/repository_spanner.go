package warehouse

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// InventoryListOptions controls inventory list reads. Zero value = 15s stale, full scan.
type InventoryListOptions struct {
	Fresh  bool
	Limit  int
	Offset int
}

// Repository is the persistence seam for warehouse operations.
type Repository interface {
	ListSupplyRequests(ctx context.Context, warehouseID string, limit int) ([]SupplyRequest, error)
	CreateSupplyRequest(ctx context.Context, req SupplyRequest, emit func(outbox.TxnBuffer) error) error
	UpdateSupplyRequestStatus(ctx context.Context, requestID, status string, emit func(outbox.TxnBuffer) error) error
	Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error

	GetInventoryList(ctx context.Context, warehouseID string, opts InventoryListOptions) (map[string]InventoryRow, error)
	UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error
	UpdateInventoryPolicy(ctx context.Context, warehouseID, productID, policy string, reorderThreshold *int64, emit func(outbox.TxnBuffer) error) error
	GetAutoDispatch(ctx context.Context, warehouseID string) (bool, error)
	UpdateAutoDispatch(ctx context.Context, warehouseID string, enabled bool, emit func(outbox.TxnBuffer) error) error
	ListAutoDispatchWarehouses(ctx context.Context) ([]AutoDispatchWarehouse, error)
	GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error)
	UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error
	DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error
	CreateTransfer(ctx context.Context, transferID, factoryID, supplierID, warehouseID string, totalVolumeVU float64, emit func(outbox.TxnBuffer) error) error
	UpdateTransferState(ctx context.Context, transferID, supplierID, newState string, emit func(outbox.TxnBuffer) error) error
	CreateWarehouse(ctx context.Context, w Warehouse, emit func(outbox.TxnBuffer) error) error
	GetWarehouse(ctx context.Context, warehouseID string) (Warehouse, error)
	UpdateWarehouse(ctx context.Context, w Warehouse, emit func(outbox.TxnBuffer) error) error
	ListWarehouses(ctx context.Context, supplierID string, limit, offset int) ([]Warehouse, error)
}

// SpannerRepository persists warehouse entities and events through Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository configures a Spanner backend seam.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type spannerTxnBuffer struct {
	events []outbox.Event
	audits []outbox.AuditEntry
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (b *spannerTxnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	b.audits = append(b.audits, e)
	return nil
}

// ListSupplyRequests returns warehouse-scoped supply requests ordered by newest update.
func (r *SpannerRepository) ListSupplyRequests(ctx context.Context, warehouseID string, limit int) ([]SupplyRequest, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner warehouse repository: nil client")
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return nil, fmt.Errorf("warehouse_id required")
	}
	if limit <= 0 {
		limit = 100
	}

	stmt := spanner.Statement{
		SQL: `SELECT RequestId, WarehouseId, SupplierId, State, RequestedBy, CoverageStartDate, CoverageDays, ProjectedUnits, CommittedUnits, PendingConfirmationUnits,
		             COALESCE(FactoryId, ''), COALESCE(TransferMode, 'TRUCK'), COALESCE(LinkedTransferId, ''),
		             COALESCE(Priority, 'NORMAL'), COALESCE(Notes, ''), COALESCE(RegionId, ''),
		             RequestedDeliveryDate, COALESCE(TotalVolumeVU, 0), CreatedAt, UpdatedAt
			FROM WarehouseSupplyRequests@{FORCE_INDEX=Idx_WarehouseSupplyRequests_ByWarehouseUpdated}
			WHERE WarehouseId = @warehouseId
			ORDER BY UpdatedAt DESC
			LIMIT @limit`,
		Params: map[string]any{
			"warehouseId": warehouseID,
			"limit":       limit,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	requests := make([]SupplyRequest, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list warehouse supply requests: %w", err)
		}

		var (
			requestID                string
			rowWarehouseID           string
			supplierID               string
			state                    string
			requestedBy              spanner.NullString
			coverageStartDate        string
			coverageDays             int64
			projectedUnits           int64
			committedUnits           int64
			pendingConfirmationUnits int64
			factoryID                string
			transferMode             string
			linkedTransferID         string
			priority                 string
			notes                    string
			regionID                 string
			requestedDeliveryDate    spanner.NullTime
			totalVolumeVU            float64
			createdAt                time.Time
			updatedAt                time.Time
		)
		if err := row.Columns(
			&requestID,
			&rowWarehouseID,
			&supplierID,
			&state,
			&requestedBy,
			&coverageStartDate,
			&coverageDays,
			&projectedUnits,
			&committedUnits,
			&pendingConfirmationUnits,
			&factoryID,
			&transferMode,
			&linkedTransferID,
			&priority,
			&notes,
			&regionID,
			&requestedDeliveryDate,
			&totalVolumeVU,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("decode warehouse supply request: %w", err)
		}

		deliveryDate := ""
		if requestedDeliveryDate.Valid {
			deliveryDate = requestedDeliveryDate.Time.UTC().Format(time.RFC3339Nano)
		}
		vu := totalVolumeVU
		if vu <= 0 && projectedUnits > 0 {
			vu = float64(projectedUnits)
		}
		requests = append(requests, SupplyRequest{
			RequestID:                requestID,
			SupplierID:               supplierID,
			WarehouseID:              rowWarehouseID,
			FactoryID:                factoryID,
			TransferMode:             transferMode,
			LinkedTransferID:         linkedTransferID,
			State:                    state,
			Status:                   state,
			RequestedBy:              strings.TrimSpace(requestedBy.StringVal),
			CoverageStartDate:        coverageStartDate,
			CoverageDays:             int(coverageDays),
			ProjectedUnits:           projectedUnits,
			CommittedUnits:           committedUnits,
			PendingConfirmationUnits: pendingConfirmationUnits,
			Priority:                 priority,
			Notes:                    notes,
			RegionID:                 regionID,
			RequestedDeliveryDate:    deliveryDate,
			TotalVolumeVU:            vu,
			CreatedAt:                createdAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt:                updatedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	if len(requests) > 0 {
		requestIDs := make([]string, 0, len(requests))
		for _, req := range requests {
			requestIDs = append(requestIDs, req.RequestID)
		}
		itemsByRequest, err := r.loadSupplyRequestItems(ctx, requestIDs)
		if err != nil {
			return nil, err
		}
		for i := range requests {
			requests[i].Items = itemsByRequest[requests[i].RequestID]
		}
	}

	return requests, nil
}

func (r *SpannerRepository) loadSupplyRequestItems(ctx context.Context, requestIDs []string) (map[string][]SupplyRequestItem, error) {
	if len(requestIDs) == 0 {
		return map[string][]SupplyRequestItem{}, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT RequestId, ItemId, ProductId, RequestedQuantity, RecommendedQuantity, UnitVolumeVU,
		             COALESCE(ShippedQuantity, 0), COALESCE(ReceivedQuantity, 0)
		      FROM WarehouseSupplyRequestItems
		      WHERE RequestId IN UNNEST(@ids)
		      ORDER BY RequestId, ProductId`,
		Params: map[string]any{"ids": requestIDs},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make(map[string][]SupplyRequestItem)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("load warehouse supply request items: %w", err)
		}
		var requestID, itemID, productID string
		var requested, recommended, shipped, received int64
		var unitVU float64
		if err := row.Columns(&requestID, &itemID, &productID, &requested, &recommended, &unitVU, &shipped, &received); err != nil {
			continue
		}
		out[requestID] = append(out[requestID], SupplyRequestItem{
			ItemID:            itemID,
			ProductID:         productID,
			RequestedQuantity: requested,
			ShippedQuantity:   shipped,
			ReceivedQuantity:  received,
			RecommendedQty:    recommended,
			UnitVolumeVU:      unitVU,
		})
	}
	return out, nil
}

// CreateSupplyRequest persists a supply request row atomically with outbox events.
func (r *SpannerRepository) CreateSupplyRequest(ctx context.Context, req SupplyRequest, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner warehouse repository: nil client")
	}

	createdAt, err := parseSupplyRequestTimestamp(req.CreatedAt)
	if err != nil {
		return err
	}
	updatedAt, err := parseSupplyRequestTimestamp(req.UpdatedAt)
	if err != nil {
		return err
	}

	state := strings.TrimSpace(req.State)
	if state == "" {
		state = strings.TrimSpace(req.Status)
	}
	if state == "" {
		state = "SUBMITTED"
	}

	buf := &spannerTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"RequestId":                req.RequestID,
			"SupplierId":               req.SupplierID,
			"WarehouseId":              req.WarehouseID,
			"State":                    state,
			"FactoryId":                nullableWarehouseString(req.FactoryID),
			"TransferMode":             nullableWarehouseString(req.TransferMode),
			"RequestedBy":              nullableWarehouseString(req.RequestedBy),
			"CoverageStartDate":        req.CoverageStartDate,
			"CoverageDays":             int64(req.CoverageDays),
			"ProjectedUnits":           req.ProjectedUnits,
			"CommittedUnits":           req.CommittedUnits,
			"PendingConfirmationUnits": req.PendingConfirmationUnits,
			"Priority":                 nullableWarehouseString(req.Priority),
			"Notes":                    nullableWarehouseString(req.Notes),
			"RegionId":                 nullableWarehouseString(req.RegionID),
			"CreatedAt":                createdAt,
			"UpdatedAt":                updatedAt,
		}
		if strings.TrimSpace(req.RequestedDeliveryDate) != "" {
			if t, err := time.Parse(time.RFC3339Nano, req.RequestedDeliveryDate); err == nil {
				row["RequestedDeliveryDate"] = t
			}
		}
		totalVU := req.TotalVolumeVU
		if totalVU <= 0 && req.ProjectedUnits > 0 {
			totalVU = float64(req.ProjectedUnits)
		}
		if totalVU > 0 {
			row["TotalVolumeVU"] = totalVU
		}
		mutations := []*spanner.Mutation{spanner.InsertOrUpdateMap("WarehouseSupplyRequests", row)}
		for _, item := range req.Items {
			itemID := strings.TrimSpace(item.ItemID)
			if itemID == "" {
				itemID = uuid.NewString()
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("WarehouseSupplyRequestItems", map[string]any{
				"RequestId":           req.RequestID,
				"ItemId":              itemID,
				"ProductId":           item.ProductID,
				"RequestedQuantity":   item.RequestedQuantity,
				"RecommendedQuantity": item.RecommendedQty,
				"UnitVolumeVU":        item.UnitVolumeVU,
				"CreatedAt":           createdAt,
			}))
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("warehouse create supply request: %w", err)
	}

	return nil
}

func (r *SpannerRepository) UpdateSupplyRequestStatus(ctx context.Context, requestID, status string, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner warehouse repository: nil client")
	}

	buf := &spannerTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
			"RequestId": requestID,
			"State":     status,
			"UpdatedAt": time.Now().UTC(),
		})}
		mutations = append(mutations, outboxMutations(buf.events)...)
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("warehouse update supply request status: %w", err)
	}

	return nil
}

// Apply executes the in-memory mutation and durably persists any emitted outbox events.
func (r *SpannerRepository) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner warehouse repository: nil client")
	}

	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}

	buf := &spannerTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
	}

	if len(buf.events) == 0 && len(buf.audits) == 0 {
		return nil
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := outboxMutations(buf.events)
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("warehouse outbox persist: %w", err)
	}

	return nil
}

func outboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	mutations := make([]*spanner.Mutation, 0, len(eventsList))
	for _, e := range eventsList {
		createdAt := e.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		row := map[string]any{
			"EventId":       e.EventID,
			"AggregateType": e.AggregateType,
			"AggregateId":   e.AggregateID,
			"TopicName":     e.TopicName,
			"Payload":       e.Payload,
			"CreatedAt":     createdAt,
			"PublishedAt":   nil,
		}
		if e.PublishedAt != nil {
			row["PublishedAt"] = e.PublishedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	return mutations
}

func parseSupplyRequestTimestamp(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse supply request timestamp %q: %w", raw, err)
	}
	return parsed.UTC(), nil
}

func nullableWarehouseString(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// inMemoryRepository provides scaffold compatibility.
type inMemoryRepository struct {
	mu       sync.RWMutex
	requests []SupplyRequest
}

// NewInMemoryRepository configures local fallback.
func NewInMemoryRepository() Repository {
	return &inMemoryRepository{requests: make([]SupplyRequest, 0)}
}

type inMemoryTxnBuffer struct {
	events []outbox.Event
}

func (b *inMemoryTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (r *inMemoryRepository) ListSupplyRequests(_ context.Context, warehouseID string, limit int) ([]SupplyRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows := make([]SupplyRequest, 0, len(r.requests))
	for _, req := range r.requests {
		if warehouseID == "" || req.WarehouseID == warehouseID {
			rows = append(rows, req)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (r *inMemoryRepository) CreateSupplyRequest(_ context.Context, req SupplyRequest, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	r.requests = append(r.requests, req)
	r.mu.Unlock()

	buf := &inMemoryTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
	}
	return nil
}

func (r *inMemoryRepository) UpdateSupplyRequestStatus(_ context.Context, requestID, status string, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	for i := range r.requests {
		if r.requests[i].RequestID == requestID {
			r.requests[i].Status = status
			r.requests[i].State = status
			r.requests[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			break
		}
	}
	r.mu.Unlock()

	buf := &inMemoryTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
	}
	return nil
}

func (r *inMemoryRepository) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}
	buf := &inMemoryTxnBuffer{}
	if emit != nil {
		_ = emit(buf)
	}
	return nil
}

func (m *inMemoryRepository) CreateWarehouse(ctx context.Context, w Warehouse, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) GetWarehouse(ctx context.Context, warehouseID string) (Warehouse, error) {
	return Warehouse{}, nil
}
func (m *inMemoryRepository) UpdateWarehouse(ctx context.Context, w Warehouse, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) ListWarehouses(ctx context.Context, supplierID string, limit, offset int) ([]Warehouse, error) {
	return nil, nil
}

func (m *inMemoryRepository) GetInventoryList(ctx context.Context, warehouseID string, opts InventoryListOptions) (map[string]InventoryRow, error) {
	return nil, nil
}
func (m *inMemoryRepository) UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) UpdateInventoryPolicy(ctx context.Context, warehouseID, productID, policy string, reorderThreshold *int64, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error) {
	return nil, nil
}
func (m *inMemoryRepository) UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) CreateTransfer(ctx context.Context, transferID, factoryID, supplierID, warehouseID string, totalVolumeVU float64, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) UpdateTransferState(ctx context.Context, transferID, supplierID, newState string, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) GetAutoDispatch(ctx context.Context, warehouseID string) (bool, error) {
	return false, nil
}
func (m *inMemoryRepository) UpdateAutoDispatch(ctx context.Context, warehouseID string, enabled bool, emit func(outbox.TxnBuffer) error) error {
	return nil
}

func (m *inMemoryRepository) ListAutoDispatchWarehouses(_ context.Context) ([]AutoDispatchWarehouse, error) {
	return []AutoDispatchWarehouse{}, nil
}

func (r *SpannerRepository) GetInventoryList(ctx context.Context, warehouseID string, opts InventoryListOptions) (map[string]InventoryRow, error) {
	sql := `SELECT si.ProductId, si.QuantityOnHand, si.QuantityReserved, si.UpdatedAt,
	             COALESCE(si.OutOfStockPolicy, ''), COALESCE(si.ReorderThreshold, 0),
	             COALESCE(w.DefaultOutOfStockPolicy, 'REJECT')
	      FROM SupplierInventoryV2 si
	      INNER JOIN Warehouses w ON si.WarehouseId = w.WarehouseId AND si.SupplierId = w.SupplierId
	      WHERE si.WarehouseId = @wid
	      ORDER BY si.ProductId`
	params := map[string]any{"wid": warehouseID}
	if opts.Limit > 0 {
		sql += ` LIMIT @limit OFFSET @offset`
		params["limit"] = opts.Limit
		params["offset"] = opts.Offset
	}
	stmt := spanner.Statement{SQL: sql, Params: params}
	txn := r.client.Single()
	if !opts.Fresh {
		txn = txn.WithTimestampBound(spanner.ExactStaleness(15 * time.Second))
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	out := make(map[string]InventoryRow)
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var pid, productPolicy, warehousePolicy string
		var qoh, qr, reorder int64
		var updated time.Time
		if err := row.Columns(&pid, &qoh, &qr, &updated, &productPolicy, &reorder, &warehousePolicy); err == nil {
			avail := qoh - qr
			if avail < 0 {
				avail = 0
			}
			out[pid] = InventoryRow{
				SKU:              pid,
				ProductName:      pid,
				Quantity:         avail,
				QuantityOnHand:   qoh,
				ReorderThreshold: reorder,
				OutOfStockPolicy: strings.ToUpper(strings.TrimSpace(productPolicy)),
				EffectivePolicy:  ResolveOutOfStockPolicy(warehousePolicy, productPolicy),
				UpdatedAt:        updated.Format(time.RFC3339),
			}
		}
	}
	return out, nil
}

func (r *SpannerRepository) loadWarehouseSupplierID(ctx context.Context, warehouseID string) (string, error) {
	row, err := r.client.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"SupplierId"})
	if err != nil {
		return "", err
	}
	var supplierID string
	if err := row.Columns(&supplierID); err != nil {
		return "", err
	}
	return supplierID, nil
}

func (r *SpannerRepository) UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error {
	supplierID, err := r.loadWarehouseSupplierID(ctx, warehouseID)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
				"SupplierId":       supplierID,
				"WarehouseId":      warehouseID,
				"ProductId":        productID,
				"QuantityOnHand":   quantity,
				"QuantityReserved": int64(0),
				"UpdatedAt":        spanner.CommitTimestamp,
			}),
		}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) UpdateInventoryPolicy(ctx context.Context, warehouseID, productID, policy string, reorderThreshold *int64, emit func(outbox.TxnBuffer) error) error {
	supplierID, err := r.loadWarehouseSupplierID(ctx, warehouseID)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		update := map[string]any{
			"SupplierId":  supplierID,
			"WarehouseId": warehouseID,
			"ProductId":   productID,
			"UpdatedAt":   spanner.CommitTimestamp,
		}
		if strings.TrimSpace(policy) != "" {
			update["OutOfStockPolicy"] = policy
		}
		if reorderThreshold != nil {
			update["ReorderThreshold"] = *reorderThreshold
		}
		muts := []*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierInventoryV2", update)}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error) {
	stmt := spanner.Statement{
		SQL: `SELECT LockId, EntityType, EntityId, Reason, CreatedAt
			  FROM WarehouseDispatchLocks
			  WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make(map[string]DispatchLock)
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var lockID, eType, eID, reason string
		var created time.Time
		if err := row.Columns(&lockID, &eType, &eID, &reason, &created); err == nil {
			out[lockID] = DispatchLock{
				LockID:     lockID,
				EntityType: eType,
				EntityID:   eID,
				Reason:     reason,
				CreatedAt:  created.Format(time.RFC3339),
			}
		}
	}
	return out, nil
}

func (r *SpannerRepository) UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("WarehouseDispatchLocks", map[string]any{
				"WarehouseId": warehouseID,
				"LockId":      lock.LockID,
				"EntityType":  lock.EntityType,
				"EntityId":    lock.EntityID,
				"Reason":      lock.Reason,
				"CreatedAt":   spanner.CommitTimestamp,
				"UpdatedAt":   spanner.CommitTimestamp,
			}),
		}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error {
	_ = warehouseID
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.Delete("WarehouseDispatchLocks", spanner.Key{lockID}),
		}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) CreateTransfer(ctx context.Context, transferID, factoryID, supplierID, warehouseID string, totalVolumeVU float64, emit func(outbox.TxnBuffer) error) error {
	row := map[string]any{
		"TransferId":    transferID,
		"FactoryId":     factoryID,
		"SupplierId":    supplierID,
		"State":         "APPROVED",
		"TotalVolumeVU": totalVolumeVU,
		"CreatedAt":     spanner.CommitTimestamp,
		"UpdatedAt":     spanner.CommitTimestamp,
	}
	if wh := strings.TrimSpace(warehouseID); wh != "" {
		row["WarehouseId"] = wh
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{spanner.InsertOrUpdateMap("FactoryInternalTransfers", row)}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) UpdateTransferState(ctx context.Context, transferID, supplierID, newState string, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "FactoryInternalTransfers", spanner.Key{transferID},
			[]string{"TransferId", "SupplierId", "State", "WarehouseId", "TotalVolumeVU"})
		if err != nil {
			return err
		}
		var id, supID, state string
		var warehouseCol spanner.NullString
		var totalVolume float64
		if err := row.Columns(&id, &supID, &state, &warehouseCol, &totalVolume); err != nil {
			return err
		}
		prevState := strings.ToUpper(strings.TrimSpace(state))
		nextState := strings.ToUpper(strings.TrimSpace(newState))
		if prevState == nextState {
			return nil
		}
		if !isWarehouseTransferTransitionAllowed(prevState, nextState) {
			return fmt.Errorf("invalid_transfer_state: %s -> %s", prevState, nextState)
		}
		warehouseID := ""
		if warehouseCol.Valid {
			warehouseID = strings.TrimSpace(warehouseCol.StringVal)
		}
		if supplierID != "" && supID != supplierID {
			return fmt.Errorf("transfer_forbidden")
		}

		transferUpdate := map[string]any{
			"TransferId": transferID,
			"State":      newState,
			"UpdatedAt":  spanner.CommitTimestamp,
		}
		if strings.EqualFold(newState, "RECEIVED") {
			transferUpdate["ReceivedAt"] = spanner.CommitTimestamp
		}
		muts := []*spanner.Mutation{spanner.UpdateMap("FactoryInternalTransfers", transferUpdate)}
		if strings.EqualFold(newState, "RECEIVED") && !strings.EqualFold(state, "RECEIVED") && strings.TrimSpace(warehouseID) != "" {
			units := int64(totalVolume)
			if units <= 0 {
				units = 1
			}
			if err := inventory.CreditBulkVUInTxn(ctx, txn, warehouseID, supID, units); err != nil {
				return err
			}
		}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

var warehouseTransferTransitions = map[string]map[string]struct{}{
	"APPROVED":   {"IN_TRANSIT": {}, "RECEIVED": {}},
	"IN_TRANSIT": {"ARRIVED": {}, "RECEIVED": {}},
	"DISPATCHED": {"ARRIVED": {}, "RECEIVED": {}},
	"ARRIVED":    {"RECEIVED": {}},
}

func isWarehouseTransferTransitionAllowed(from, to string) bool {
	allowed, ok := warehouseTransferTransitions[from]
	if !ok {
		return to == "RECEIVED" && (from == "IN_TRANSIT" || from == "ARRIVED" || from == "APPROVED")
	}
	_, ok = allowed[to]
	return ok
}

func (r *SpannerRepository) GetAutoDispatch(ctx context.Context, warehouseID string) (bool, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT AutoDispatchEnabled FROM Warehouses WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return false, nil
		}
		return false, err
	}
	var enabled bool
	if err := row.Columns(&enabled); err != nil {
		return false, err
	}
	return enabled, nil
}

func (r *SpannerRepository) ListAutoDispatchWarehouses(ctx context.Context) ([]AutoDispatchWarehouse, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner warehouse repository: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, SupplierId
		      FROM Warehouses@{FORCE_INDEX=Idx_Warehouses_ByAutoDispatch}
		      WHERE AutoDispatchEnabled = TRUE
		        AND IsActive = TRUE`,
	}
	iter := r.client.Single().
		WithTimestampBound(spanner.ExactStaleness(15*time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()

	out := make([]AutoDispatchWarehouse, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("list auto-dispatch warehouses: %w", err)
		}
		var wh AutoDispatchWarehouse
		if err := row.Columns(&wh.WarehouseID, &wh.SupplierID); err != nil {
			return nil, fmt.Errorf("scan auto-dispatch warehouse: %w", err)
		}
		out = append(out, wh)
	}
}

func (r *SpannerRepository) UpdateAutoDispatch(ctx context.Context, warehouseID string, enabled bool, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.UpdateMap("Warehouses", map[string]any{
				"WarehouseId":         warehouseID,
				"AutoDispatchEnabled": enabled,
				"UpdatedAt":           spanner.CommitTimestamp,
			}),
		}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}
