package warehouse

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// Repository is the persistence seam for warehouse operations.
type Repository interface {
	ListSupplyRequests(ctx context.Context, warehouseID string, limit int) ([]SupplyRequest, error)
	CreateSupplyRequest(ctx context.Context, req SupplyRequest, emit func(outbox.TxnBuffer) error) error
	Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error
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
		SQL: `SELECT RequestId, WarehouseId, State, RequestedBy, CoverageStartDate, CoverageDays, ProjectedUnits, CommittedUnits, PendingConfirmationUnits, CreatedAt, UpdatedAt
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
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("decode warehouse supply request: %w", err)
		}

		requests = append(requests, SupplyRequest{
			RequestID:                requestID,
			WarehouseID:              rowWarehouseID,
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
		state = "OPEN"
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
