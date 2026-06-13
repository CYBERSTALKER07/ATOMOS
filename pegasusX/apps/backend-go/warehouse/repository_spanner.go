package warehouse

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// Repository is the persistence seam for warehouse operations.
type Repository interface {
	ListSupplyRequests(ctx context.Context, warehouseID string, limit int) ([]SupplyRequest, error)
	CreateSupplyRequest(ctx context.Context, req SupplyRequest, emit func(outbox.TxnBuffer) error) error
	UpdateSupplyRequestStatus(ctx context.Context, requestID, status string, emit func(outbox.TxnBuffer) error) error
	Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error

	GetInventoryList(ctx context.Context, warehouseID string) (map[string]InventoryRow, error)
	UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error
	GetAutoDispatch(ctx context.Context, warehouseID string) (bool, error)
	UpdateAutoDispatch(ctx context.Context, warehouseID string, enabled bool, emit func(outbox.TxnBuffer) error) error
	ListAutoDispatchWarehouses(ctx context.Context) ([]AutoDispatchWarehouse, error)
	GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error)
	UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error
	DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error
	CreateTransfer(ctx context.Context, transferID, factoryID, supplierID, warehouseID string, totalVolumeVU float64, emit func(outbox.TxnBuffer) error) error
	UpdateTransferState(ctx context.Context, transferID, supplierID, newState string, emit func(outbox.TxnBuffer) error) error
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
		SQL: `SELECT RequestId, WarehouseId, State, RequestedBy, CoverageStartDate, CoverageDays, ProjectedUnits, CommittedUnits, PendingConfirmationUnits,
		             COALESCE(FactoryId, ''), COALESCE(TransferMode, 'TRUCK'), COALESCE(LinkedTransferId, ''), CreatedAt, UpdatedAt
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
			createdAt                time.Time
			updatedAt                time.Time
		)
		if err := row.Columns(
			&requestID,
			&rowWarehouseID,
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
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("decode warehouse supply request: %w", err)
		}

		requests = append(requests, SupplyRequest{
			RequestID:                requestID,
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
			CreatedAt:                createdAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt:                updatedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	return requests, nil
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
		mutations := []*spanner.Mutation{spanner.InsertOrUpdateMap("WarehouseSupplyRequests", map[string]any{
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
			"CreatedAt":                createdAt,
			"UpdatedAt":                updatedAt,
		})}
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

func (m *inMemoryRepository) GetInventoryList(ctx context.Context, warehouseID string) (map[string]InventoryRow, error) {
	return nil, nil
}
func (m *inMemoryRepository) UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error {
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

func (r *SpannerRepository) GetInventoryList(ctx context.Context, warehouseID string) (map[string]InventoryRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, QuantityOnHand, UpdatedAt
			  FROM InventoryLevels
			  WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make(map[string]InventoryRow)
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var pid string
		var qty int64
		var updated time.Time
		if err := row.Columns(&pid, &qty, &updated); err == nil {
			out[pid] = InventoryRow{
				SKU:         pid,
				ProductName: pid,
				Quantity:    qty,
				UpdatedAt:   updated.Format(time.RFC3339),
			}
		}
	}
	return out, nil
}

func (r *SpannerRepository) UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("InventoryLevels", map[string]any{
				"WarehouseId":    warehouseID,
				"ProductId":      productID,
				"QuantityOnHand": quantity,
				"UpdatedAt":      spanner.CommitTimestamp,
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
		warehouseID := ""
		if warehouseCol.Valid {
			warehouseID = strings.TrimSpace(warehouseCol.StringVal)
		}
		if supplierID != "" && supID != supplierID {
			return fmt.Errorf("transfer_forbidden")
		}

		muts := []*spanner.Mutation{
			spanner.UpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId": transferID,
				"State":      newState,
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
		}
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

func (r *SpannerRepository) GetAutoDispatch(ctx context.Context, warehouseID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AutoDispatchEnabled FROM Warehouses WHERE WarehouseId = @wid`,
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
		WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).
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
				"WarehouseId": warehouseID,
				"AutoDispatchEnabled": enabled,
				"UpdatedAt": spanner.CommitTimestamp,
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
