package cashrecon

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) SaveReconciliation(ctx context.Context, cr CashReconciliation, eventType string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("cashrecon repository unavailable")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts, err := r.saveReconciliationMutations(cr)
		if err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		if eventType != "" {
			payload := map[string]any{
				"type":                eventType,
				"reconciliation_id":   cr.ReconciliationId,
				"driver_id":           cr.DriverId,
				"route_id":            cr.RouteId,
				"shift_date":          civil.DateOf(cr.ShiftDate.UTC()).String(),
				"expected_cash_minor": cr.ExpectedCashMinor,
				"declared_cash_minor": cr.DeclaredCashMinor,
				"difference_minor":    cr.DifferenceMinor,
				"status":              string(cr.Status),
			}
			if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateDriver, cr.DriverId, events.TopicMain, payload); emitErr != nil {
				return emitErr
			}
		}
		if eventType == EventCashReconciliationCreated && cr.DifferenceMinor != 0 && cr.RouteId != nil && strings.TrimSpace(*cr.RouteId) != "" {
			if exMut, exErr := r.settlementExceptionMutation(ctx, txn, cr); exErr == nil && exMut != nil {
				muts = append(muts, exMut)
			}
		}
		muts = append(muts, outboxMutations(buf.events)...)
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) saveReconciliationMutations(cr CashReconciliation) ([]*spanner.Mutation, error) {
	crRow := map[string]interface{}{
		"ReconciliationId":  cr.ReconciliationId,
		"DriverId":          cr.DriverId,
		"RouteId":           spanner.NullString{StringVal: derefStr(cr.RouteId), Valid: cr.RouteId != nil && strings.TrimSpace(*cr.RouteId) != ""},
		"ShiftDate":         civil.DateOf(cr.ShiftDate.UTC()),
		"ExpectedCashMinor": cr.ExpectedCashMinor,
		"DeclaredCashMinor": cr.DeclaredCashMinor,
		"DifferenceMinor":   cr.DifferenceMinor,
		"Status":            string(cr.Status),
		"DriverNote":        spanner.NullString{StringVal: derefStr(cr.DriverNote), Valid: cr.DriverNote != nil},
		"FinanceNote":       spanner.NullString{StringVal: derefStr(cr.FinanceNote), Valid: cr.FinanceNote != nil},
		"CreatedAt":         cr.CreatedAt,
		"ResolvedAt":        spanner.NullTime{Time: derefTime(cr.ResolvedAt), Valid: cr.ResolvedAt != nil},
		"ResolvedBy":        spanner.NullString{StringVal: derefStr(cr.ResolvedBy), Valid: cr.ResolvedBy != nil},
	}
	return []*spanner.Mutation{spanner.InsertOrUpdateMap("CashReconciliations", crRow)}, nil
}

type settlementExceptionRow struct {
	OrderID     string    `spanner:"OrderId"`
	ExceptionID string    `spanner:"ExceptionId"`
	Type        string    `spanner:"Type"`
	AmountMinor int64     `spanner:"AmountMinor"`
	Status      string    `spanner:"Status"`
	Reason      string    `spanner:"Reason"`
	CreatedBy   string    `spanner:"CreatedBy"`
	CreatedAt   time.Time `spanner:"CreatedAt"`
}

func (r *SpannerRepository) settlementExceptionMutation(ctx context.Context, txn *spanner.ReadWriteTransaction, cr CashReconciliation) (*spanner.Mutation, error) {
	routeID := strings.TrimSpace(*cr.RouteId)
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId FROM Orders
		      WHERE RouteId = @routeId AND DriverId = @driverId
		      ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]interface{}{
			"routeId":  routeID,
			"driverId": cr.DriverId,
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return nil, err
	}
	var orderID string
	if err := row.Column(0, &orderID); err != nil {
		return nil, err
	}
	amount := cr.DifferenceMinor
	if amount < 0 {
		amount = -amount
	}
	ex := settlementExceptionRow{
		OrderID:     orderID,
		ExceptionID: uuid.New().String(),
		Type:        "CASH_DISCREPANCY",
		AmountMinor: amount,
		Status:      "OPEN",
		Reason:      fmt.Sprintf("reconciliation_id=%s", cr.ReconciliationId),
		CreatedBy:   "system:cash-recon",
		CreatedAt:   time.Now().UTC(),
	}
	m, err := spanner.InsertStruct("OrderSettlementExceptions", ex)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *SpannerRepository) GetReconciliation(ctx context.Context, id string) (*CashReconciliation, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("cashrecon repository unavailable")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "CashReconciliations", spanner.Key{id},
		[]string{"ReconciliationId", "DriverId", "RouteId", "ShiftDate", "ExpectedCashMinor", "DeclaredCashMinor",
			"DifferenceMinor", "Status", "DriverNote", "FinanceNote", "CreatedAt", "ResolvedAt", "ResolvedBy"})
	if err != nil {
		if grpcstatus.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return scanReconciliationRow(row)
}

func (r *SpannerRepository) ListReconciliationsByStatus(ctx context.Context, status ReconciliationStatus) ([]CashReconciliation, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("cashrecon repository unavailable")
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ReconciliationId, DriverId, RouteId, ShiftDate, ExpectedCashMinor, DeclaredCashMinor,
		             DifferenceMinor, Status, DriverNote, FinanceNote, CreatedAt, ResolvedAt, ResolvedBy
		      FROM CashReconciliations WHERE Status = @status ORDER BY CreatedAt DESC LIMIT 200`,
		Params: map[string]interface{}{"status": string(status)},
	})
	defer iter.Stop()
	return collectReconciliations(iter)
}

func (r *SpannerRepository) ListByDriver(ctx context.Context, driverID string, shiftDate time.Time) ([]CashReconciliation, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("cashrecon repository unavailable")
	}
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return nil, fmt.Errorf("driver_id required")
	}
	d := civil.DateOf(shiftDate.UTC())
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ReconciliationId, DriverId, RouteId, ShiftDate, ExpectedCashMinor, DeclaredCashMinor,
		             DifferenceMinor, Status, DriverNote, FinanceNote, CreatedAt, ResolvedAt, ResolvedBy
		      FROM CashReconciliations
		      WHERE DriverId = @driverId AND ShiftDate = @shiftDate
		      ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{
			"driverId":  driverID,
			"shiftDate": d,
		},
	})
	defer iter.Stop()
	return collectReconciliations(iter)
}

func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID string, status ReconciliationStatus, limit int) ([]CashReconciliation, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("cashrecon repository unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, fmt.Errorf("supplier_id required")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	params := map[string]interface{}{
		"supplierId": supplierID,
		"limit":      limit,
	}
	sql := `SELECT cr.ReconciliationId, cr.DriverId, cr.RouteId, cr.ShiftDate, cr.ExpectedCashMinor, cr.DeclaredCashMinor,
	               cr.DifferenceMinor, cr.Status, cr.DriverNote, cr.FinanceNote, cr.CreatedAt, cr.ResolvedAt, cr.ResolvedBy
	        FROM CashReconciliations cr
	        JOIN Drivers d ON cr.DriverId = d.DriverId
	        WHERE d.SupplierId = @supplierId`
	if status != "" {
		sql += ` AND cr.Status = @status`
		params["status"] = string(status)
	}
	sql += ` ORDER BY cr.CreatedAt DESC LIMIT @limit`
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	return collectReconciliations(iter)
}

func collectReconciliations(iter *spanner.RowIterator) ([]CashReconciliation, error) {
	var out []CashReconciliation
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		cr, err := scanReconciliationRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *cr)
	}
	return out, nil
}

func scanReconciliationRow(row *spanner.Row) (*CashReconciliation, error) {
	var cr CashReconciliation
	var routeID spanner.NullString
	var driverNote, financeNote, resolvedBy spanner.NullString
	var resolvedAt spanner.NullTime
	var shiftDate civil.Date
	var status string
	if err := row.Columns(
		&cr.ReconciliationId, &cr.DriverId, &routeID, &shiftDate,
		&cr.ExpectedCashMinor, &cr.DeclaredCashMinor, &cr.DifferenceMinor,
		&status, &driverNote, &financeNote, &cr.CreatedAt, &resolvedAt, &resolvedBy,
	); err != nil {
		return nil, err
	}
	cr.ShiftDate = shiftDate.In(time.UTC)
	cr.Status = ReconciliationStatus(status)
	if routeID.Valid && strings.TrimSpace(routeID.StringVal) != "" {
		v := strings.TrimSpace(routeID.StringVal)
		cr.RouteId = &v
	}
	if driverNote.Valid {
		v := driverNote.StringVal
		cr.DriverNote = &v
	}
	if financeNote.Valid {
		v := financeNote.StringVal
		cr.FinanceNote = &v
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		cr.ResolvedAt = &t
	}
	if resolvedBy.Valid {
		v := resolvedBy.StringVal
		cr.ResolvedBy = &v
	}
	return &cr, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
