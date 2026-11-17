package order

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

var (
	// ErrInvalidHistoricalStatus rejects non-terminal statuses during historical order import.
	ErrInvalidHistoricalStatus = errors.New("order migration: historical order status must be terminal (COMPLETED, DELIVERED, CANCELLED, RETURNED)")
	// ErrOrderNotMigrationHeld rejects open-order release for orders not in MIGRATION_OPEN or already rolled back.
	ErrOrderNotMigrationHeld = errors.New("order migration: order is not in releasable MIGRATION_OPEN state")
)

var tashkentLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		return time.FixedZone("UZT", 5*3600)
	}
	return loc
}()

// HistoricalOrderRequest supplies one historical legacy order row to import without side effects.
type HistoricalOrderRequest struct {
	SessionID         string
	SupplierID        string
	RetailerID        string
	WarehouseID       string
	ExternalOrderNo   string
	ExternalFiscalRef string
	RawStatus         string
	Currency          string
	TotalMinor        int64
	LineItems         []LineItem
	CreatedAt         time.Time
	Now               time.Time
	Notes             string
	DeliveryFeeMinor  int64
}

// MigratedOpenOrderRequest supplies one unfulfilled legacy open order to hold in MIGRATION_OPEN.
type MigratedOpenOrderRequest struct {
	SessionID             string
	SupplierID            string
	RetailerID            string
	WarehouseID           string
	ExternalOrderNo       string
	ExternalFiscalRef     string
	IntendedLiveStatus    Status
	Currency              string
	TotalMinor            int64
	LineItems             []LineItem
	RequestedDeliveryDate *time.Time
	CreatedAt             time.Time
	Now                   time.Time
	Notes                 string
	DeliveryFeeMinor      int64
}

// ReleaseOpenOrderItem identifies one MIGRATION_OPEN order and its target live status.
type ReleaseOpenOrderItem struct {
	OrderID      string
	TargetStatus Status // PENDING (default) or SCHEDULED
}

// ReleaseOpenOrdersResult summarizes bulk/selective MIGRATION_OPEN release outcomes.
type ReleaseOpenOrdersResult struct {
	ReleasedOrderIDs []string          `json:"released_order_ids"`
	SkippedOrderIDs  []string          `json:"skipped_order_ids,omitempty"`
	Errors           map[string]string `json:"errors,omitempty"`
}

func migOrderShortID(prefix, key string) string {
	sum := sha256.Sum256([]byte(key))
	id := prefix + fmt.Sprintf("%x", sum[:15]) // 6 + 30 = 36 chars <= Orders.OrderId STRING(36)
	if len(id) > 36 {
		return id[:36]
	}
	return id
}

// NormalizeHistoricalOrderStatus enforces terminal-only statuses for historical order migration.
func NormalizeHistoricalOrderStatus(raw string) (Status, string, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "COMPLETED", "DELIVERED", "DONE", "FULFILLED", "ЗАВЕРШЕН", "ДОСТАВЛЕН", "YETKAZILDI", "BAJARILDI":
		return StatusCompleted, "", nil
	case "CANCELLED", "CANCELED", "ОТМЕНЕН", "BEKOR QILINDI":
		return StatusCancelled, "", nil
	case "RETURNED", "ВОЗВРАТ", "QAYTARILDI":
		return StatusCancelled, "legacy_status=RETURNED", nil
	default:
		return "", "", fmt.Errorf("%w: %q", ErrInvalidHistoricalStatus, raw)
	}
}

// ParseMigrationTimestamp parses an ERP timestamp using Asia/Tashkent for naive strings.
func ParseMigrationTimestamp(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"02.01.2006 15:04:05",
		"02.01.2006",
	} {
		if t, err := time.ParseInLocation(layout, s, tashkentLoc); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("order migration: invalid timestamp %q", raw)
}

// PrepareHistoricalOrder builds a terminal migrated Order (OrderSource=MIGRATION, FiscalStatus=HISTORICAL_SKIPPED)
// with zero inventory reservation, zero dispatch eligibility, and zero OFD submission.
func PrepareHistoricalOrder(req HistoricalOrderRequest) (Order, map[string]any, error) {
	supplierID := strings.TrimSpace(req.SupplierID)
	retailerID := strings.TrimSpace(req.RetailerID)
	extNo := strings.TrimSpace(req.ExternalOrderNo)
	if supplierID == "" || retailerID == "" || extNo == "" {
		return Order{}, nil, fmt.Errorf("order migration: supplier_id, retailer_id, and external_order_no are required")
	}
	st, noteTag, err := NormalizeHistoricalOrderStatus(req.RawStatus)
	if err != nil {
		return Order{}, nil, err
	}
	if req.TotalMinor < 0 {
		return Order{}, nil, fmt.Errorf("order migration: total_minor cannot be negative: %d", req.TotalMinor)
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = "UZS"
	}
	now := req.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	createdAt := req.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = now
	}
	notes := strings.TrimSpace(req.Notes)
	if noteTag != "" {
		if notes == "" {
			notes = noteTag
		} else {
			notes = notes + "; " + noteTag
		}
	}
	lines := req.LineItems
	if lines == nil {
		lines = []LineItem{}
	}
	linesRaw, err := json.Marshal(lines)
	if err != nil {
		return Order{}, nil, fmt.Errorf("marshal historical line items: %w", err)
	}

	o := Order{
		OrderID:              migOrderShortID("ord_h_", supplierID+"|"+extNo),
		SupplierID:           supplierID,
		RetailerID:           retailerID,
		WarehouseID:          strings.TrimSpace(req.WarehouseID),
		Status:               st,
		Source:               OrderSourceMigration,
		ConfirmationStatus:   ConfirmationStatusConfirmed,
		LineItems:            lines,
		TotalMinor:           req.TotalMinor,
		OriginalTotalMinor:   req.TotalMinor,
		Currency:             cur,
		DeliveryPriority:     DeliveryPriorityStandard,
		DeliveryFeeMinor:     req.DeliveryFeeMinor,
		WarehouseNotes:       notes,
		FiscalStatus:         FiscalStatusHistoricalSkipped,
		ExternalOrderNo:      extNo,
		ExternalFiscalRef:    strings.TrimSpace(req.ExternalFiscalRef),
		SupersededByRollback: false,
		Version:              1,
		CreatedAt:            createdAt,
		UpdatedAt:            now,
	}
	insertMap := buildOrderInsertMap(&o, linesRaw)
	if strings.TrimSpace(o.WarehouseID) == "" {
		insertMap["WarehouseId"] = nil
	}
	return o, insertMap, nil
}

// PrepareMigratedOpenOrder builds a held open Order in Status=MIGRATION_OPEN, OrderSource=MIGRATION
// with ZERO stock reservation until explicitly released post-cutover.
func PrepareMigratedOpenOrder(req MigratedOpenOrderRequest) (Order, map[string]any, error) {
	supplierID := strings.TrimSpace(req.SupplierID)
	retailerID := strings.TrimSpace(req.RetailerID)
	extNo := strings.TrimSpace(req.ExternalOrderNo)
	if supplierID == "" || retailerID == "" || extNo == "" {
		return Order{}, nil, fmt.Errorf("order migration: supplier_id, retailer_id, and external_order_no are required")
	}
	if req.TotalMinor < 0 {
		return Order{}, nil, fmt.Errorf("order migration: total_minor cannot be negative: %d", req.TotalMinor)
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = "UZS"
	}
	now := req.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	createdAt := req.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = now
	}
	lines := req.LineItems
	if lines == nil {
		lines = []LineItem{}
	}
	linesRaw, err := json.Marshal(lines)
	if err != nil {
		return Order{}, nil, fmt.Errorf("marshal open order line items: %w", err)
	}

	notes := strings.TrimSpace(req.Notes)
	if req.IntendedLiveStatus == StatusScheduled {
		if notes == "" {
			notes = "intended_live_status=SCHEDULED"
		} else if !strings.Contains(notes, "intended_live_status=") {
			notes = notes + "; intended_live_status=SCHEDULED"
		}
	}

	fiscalStatus := FiscalStatusNone
	if strings.TrimSpace(req.ExternalFiscalRef) != "" {
		fiscalStatus = FiscalStatusHistoricalSkipped
	}

	o := Order{
		OrderID:               migOrderShortID("ord_o_", supplierID+"|"+extNo),
		SupplierID:            supplierID,
		RetailerID:            retailerID,
		WarehouseID:           strings.TrimSpace(req.WarehouseID),
		Status:                StatusMigrationOpen,
		Source:                OrderSourceMigration,
		ConfirmationStatus:    ConfirmationStatusConfirmed,
		LineItems:             lines,
		TotalMinor:            req.TotalMinor,
		OriginalTotalMinor:    req.TotalMinor,
		Currency:              cur,
		RequestedDeliveryDate: req.RequestedDeliveryDate,
		DeliveryPriority:      DeliveryPriorityStandard,
		DeliveryFeeMinor:      req.DeliveryFeeMinor,
		WarehouseNotes:        notes,
		FiscalStatus:          fiscalStatus,
		ExternalOrderNo:       extNo,
		ExternalFiscalRef:     strings.TrimSpace(req.ExternalFiscalRef),
		SupersededByRollback:  false,
		Version:               1,
		CreatedAt:             createdAt,
		UpdatedAt:             now,
	}
	insertMap := buildOrderInsertMap(&o, linesRaw)
	if strings.TrimSpace(o.WarehouseID) == "" {
		insertMap["WarehouseId"] = nil
	}
	return o, insertMap, nil
}

// ImportHistoricalOrder writes one idempotent historical order to Spanner without reserving stock or emitting fiscal/dispatch jobs.
func ImportHistoricalOrder(ctx context.Context, client *spanner.Client, req HistoricalOrderRequest) (Order, bool, error) {
	if client == nil {
		return Order{}, false, fmt.Errorf("order migration: spanner client is required")
	}
	var out Order
	var existed bool
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		o, dup, err := ImportHistoricalOrderInTxn(ctx, txn, req)
		if err != nil {
			return err
		}
		out = o
		existed = dup
		return nil
	})
	return out, existed, err
}

// ImportHistoricalOrderInTxn writes one historical order inside an active Spanner RW transaction, skipping duplicates on (SupplierId, ExternalOrderNo).
func ImportHistoricalOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, req HistoricalOrderRequest) (Order, bool, error) {
	if txn == nil {
		return Order{}, false, fmt.Errorf("order migration: nil spanner transaction")
	}
	o, insertMap, err := PrepareHistoricalOrder(req)
	if err != nil {
		return Order{}, false, err
	}
	existing, found, err := lookupOrderByExternalNoInTxn(ctx, txn, o.SupplierID, o.ExternalOrderNo)
	if err != nil {
		return Order{}, false, err
	}
	if found {
		return existing, true, nil
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("Orders", insertMap)}); err != nil {
		return Order{}, false, err
	}
	return o, false, nil
}

// ImportMigratedOpenOrder writes one held MIGRATION_OPEN order to Spanner with zero stock reservation.
func ImportMigratedOpenOrder(ctx context.Context, client *spanner.Client, req MigratedOpenOrderRequest) (Order, bool, error) {
	if client == nil {
		return Order{}, false, fmt.Errorf("order migration: spanner client is required")
	}
	var out Order
	var existed bool
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		o, dup, err := ImportMigratedOpenOrderInTxn(ctx, txn, req)
		if err != nil {
			return err
		}
		out = o
		existed = dup
		return nil
	})
	return out, existed, err
}

// ImportMigratedOpenOrderInTxn writes one held MIGRATION_OPEN order inside an active Spanner RW transaction with zero inventory reservation.
func ImportMigratedOpenOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, req MigratedOpenOrderRequest) (Order, bool, error) {
	if txn == nil {
		return Order{}, false, fmt.Errorf("order migration: nil spanner transaction")
	}
	o, insertMap, err := PrepareMigratedOpenOrder(req)
	if err != nil {
		return Order{}, false, err
	}
	existing, found, err := lookupOrderByExternalNoInTxn(ctx, txn, o.SupplierID, o.ExternalOrderNo)
	if err != nil {
		return Order{}, false, err
	}
	if found {
		return existing, true, nil
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("Orders", insertMap)}); err != nil {
		return Order{}, false, err
	}
	return o, false, nil
}

// ResolveReleaseTargetStatus validates and returns the live transition target for a MIGRATION_OPEN order.
func ResolveReleaseTargetStatus(o Order, requested Status) (Status, error) {
	if o.SupersededByRollback || o.Status != StatusMigrationOpen {
		return "", fmt.Errorf("%w: order=%s status=%s superseded=%v", ErrOrderNotMigrationHeld, o.OrderID, o.Status, o.SupersededByRollback)
	}
	target := requested
	if target == "" {
		if strings.Contains(o.WarehouseNotes, "intended_live_status=SCHEDULED") {
			target = StatusScheduled
		} else {
			target = StatusPending
		}
	}
	if target != StatusPending && target != StatusScheduled {
		return "", fmt.Errorf("%w: release target must be PENDING or SCHEDULED, got %q", ErrInvalidStatusTransition, target)
	}
	if err := ValidateStatusTransition(string(StatusMigrationOpen), string(target), TransitionOpts{}); err != nil {
		return "", err
	}
	return target, nil
}

// ReleaseMigratedOpenOrders transitions selected (or all held) MIGRATION_OPEN orders for supplierID
// to live PENDING/SCHEDULED with OrderSource=MANUAL, reserving standard inventory/FEFO lots in Spanner.
func ReleaseMigratedOpenOrders(
	ctx context.Context,
	client *spanner.Client,
	supplierID string,
	items []ReleaseOpenOrderItem,
	actorID string,
	now time.Time,
) (ReleaseOpenOrdersResult, error) {
	if client == nil {
		return ReleaseOpenOrdersResult{}, fmt.Errorf("order migration: spanner client is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return ReleaseOpenOrdersResult{}, fmt.Errorf("order migration: supplier_id is required")
	}

	if len(items) == 0 {
		listed, err := listMigrationOpenOrderIDs(ctx, client, supplierID)
		if err != nil {
			return ReleaseOpenOrdersResult{}, err
		}
		for _, oid := range listed {
			items = append(items, ReleaseOpenOrderItem{OrderID: oid})
		}
	}

	res := ReleaseOpenOrdersResult{
		ReleasedOrderIDs: make([]string, 0, len(items)),
		Errors:           map[string]string{},
	}
	for _, it := range items {
		err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			iter := txn.Query(ctx, spanner.Statement{
				SQL:    "SELECT " + orderSelectColumns + " FROM Orders WHERE OrderId = @oid AND SupplierId = @sid LIMIT 1",
				Params: map[string]any{"oid": it.OrderID, "sid": supplierID},
			})
			row, qErr := iter.Next()
			iter.Stop()
			if qErr == iterator.Done {
				return fmt.Errorf("%w: order %s not found", ErrOrderNotMigrationHeld, it.OrderID)
			}
			if qErr != nil {
				return qErr
			}
			o, sErr := scanOrderRowRow(row)
			if sErr != nil {
				return sErr
			}
			target, tErr := ResolveReleaseTargetStatus(o, it.TargetStatus)
			if tErr != nil {
				return tErr
			}

			if len(o.LineItems) > 0 && strings.TrimSpace(o.WarehouseID) != "" {
				var expected time.Time
				if o.RequestedDeliveryDate != nil {
					expected = o.RequestedDeliveryDate.UTC()
				}
				if err := ReserveLineItemsForOrderInTxn(ctx, txn, o.SupplierID, o.WarehouseID, o.OrderID, o.RetailerID, expected, o.LineItems); err != nil {
					return err
				}
				if err := insertStockReservationMarkerInTxn(txn, o.OrderID); err != nil {
					return err
				}
			}

			updateMap := map[string]any{
				"OrderId":     o.OrderID,
				"Status":      string(target),
				"OrderSource": string(OrderSourceManual),
				"Version":     o.Version + 1,
				"UpdatedAt":   now.UTC(),
			}
			if strings.TrimSpace(o.ExternalFiscalRef) != "" {
				updateMap["FiscalStatus"] = FiscalStatusHistoricalSkipped
			} else if strings.TrimSpace(o.FiscalStatus) == "" {
				updateMap["FiscalStatus"] = FiscalStatusNone
			}

			transRow := spanner.InsertMap("OrderStatusTransitions", map[string]any{
				"OrderId":        o.OrderID,
				"TransitionId":   uuid.NewString(),
				"PreviousStatus": string(StatusMigrationOpen),
				"NewStatus":      string(target),
				"Reason":         "migration_open_order_released",
				"ActorRole":      "ADMIN",
				"ActorId":        strings.TrimSpace(actorID),
				"EventKind":      "MIGRATION_OPEN_ORDER_RELEASED",
				"CreatedAt":      now.UTC(),
			})

			return txn.BufferWrite([]*spanner.Mutation{
				spanner.UpdateMap("Orders", updateMap),
				transRow,
			})
		})
		if err != nil {
			res.Errors[it.OrderID] = err.Error()
			res.SkippedOrderIDs = append(res.SkippedOrderIDs, it.OrderID)
			continue
		}
		res.ReleasedOrderIDs = append(res.ReleasedOrderIDs, it.OrderID)
	}
	if len(res.Errors) == 0 {
		res.Errors = nil
	}
	return res, nil
}

func lookupOrderByExternalNoInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, extNo string) (Order, bool, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    "SELECT " + orderSelectColumns + " FROM Orders WHERE SupplierId = @sid AND ExternalOrderNo = @eno LIMIT 1",
		Params: map[string]any{"sid": strings.TrimSpace(supplierID), "eno": strings.TrimSpace(extNo)},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return Order{}, false, nil
	}
	if err != nil {
		return Order{}, false, err
	}
	o, err := scanOrderRowRow(row)
	return o, err == nil, err
}

func listMigrationOpenOrderIDs(ctx context.Context, client *spanner.Client, supplierID string) ([]string, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId FROM Orders
			WHERE SupplierId = @sid AND Status = 'MIGRATION_OPEN' AND COALESCE(SupersededByRollback, FALSE) = FALSE
			ORDER BY CreatedAt ASC`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	var ids []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return ids, nil
		}
		if err != nil {
			return nil, err
		}
		var oid string
		if err := row.Columns(&oid); err != nil {
			return nil, err
		}
		ids = append(ids, oid)
	}
}
