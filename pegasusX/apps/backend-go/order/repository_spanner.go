package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerRepository persists order rows in Spanner and writes emitted outbox
// events in the same RW transaction.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed order repository.
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

const orderSelectColumns = `OrderId, SupplierId, RetailerId, WarehouseId, DriverId, VehicleId, RouteId, ManifestId, DeliveryToken, Status, OrderSource, ConfirmationStatus, LineItemsJson, TotalMinor, OriginalTotalMinor, Currency, H3Cell, Lat, Lng, RequestedDeliveryDate, DeliverBefore, DeliveryPriority, DeliveryFeeMinor, WarehouseNotes, AutoConfirmAt, DecisionAt, DecisionBy, DerivedFromOrderId, ReceivingWindowOpen, ReceivingWindowClose, PreorderReminderSentAt, NudgeNotifiedAt, ConfirmationNotifiedAt, CancelLockedAt, CancelLockReason, ProposedDeliveryDate, DeliveryProposalAt, DeliveryProposalBy, DeliveryProposalReason, Version, CreatedAt, UpdatedAt`

// CreateOrder writes the Orders row and any emitted outbox events atomically.
func (r *SpannerRepository) CreateOrder(ctx context.Context, o *Order, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner order repository: nil client")
	}
	if o == nil {
		return fmt.Errorf("create order: nil aggregate")
	}

	lineItemsRaw, err := json.Marshal(o.LineItems)
	if err != nil {
		return fmt.Errorf("marshal order line items: %w", err)
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := snapshotReceivingWindowsInTxn(ctx, txn, o); err != nil {
			return err
		}

		// Reserve on-hand stock for fulfillable quantities (all non-backorder orders, including scheduled pre-orders).
		if len(o.LineItems) > 0 && o.WarehouseID != "" && o.Source != OrderSourceBackorder {
			if err := ReserveLineItemsInTxn(ctx, txn, o.SupplierID, o.WarehouseID, o.LineItems); err != nil {
				return err
			}
			if err := insertStockReservationMarkerInTxn(txn, o.OrderID); err != nil {
				return err
			}
		}

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.InsertMap("Orders", map[string]any{
				"OrderId":               o.OrderID,
				"SupplierId":            o.SupplierID,
				"RetailerId":            o.RetailerID,
				"WarehouseId":           o.WarehouseID,
				"DriverId":              nullableString(o.DriverID),
				"VehicleId":             nullableString(o.VehicleID),
				"RouteId":               nullableString(o.RouteID),
				"ManifestId":            nullableString(o.ManifestID),
				"DeliveryToken":         nullableString(o.QRToken),
				"Status":                string(o.Status),
				"OrderSource":           string(o.Source),
				"ConfirmationStatus":    string(o.ConfirmationStatus),
				"LineItemsJson":         lineItemsRaw,
				"TotalMinor":            o.TotalMinor,
				"OriginalTotalMinor":    originalTotalMinorForInsert(o),
				"Currency":              o.Currency,
				"H3Cell":                o.H3Cell,
				"Lat":                   o.Lat,
				"Lng":                   o.Lng,
				"RequestedDeliveryDate": nullableTime(o.RequestedDeliveryDate),
				"DeliverBefore":         nullableTime(o.DeliverBefore),
				"DeliveryPriority":      string(o.DeliveryPriority),
				"DeliveryFeeMinor":       o.DeliveryFeeMinor,
				"WarehouseNotes":        nullableString(o.WarehouseNotes),
				"AutoConfirmAt":         nullableTime(o.AutoConfirmAt),
				"DecisionAt":            nullableTime(o.DecisionAt),
				"DecisionBy":            nullableString(o.DecisionBy),
				"DerivedFromOrderId":    nullableString(o.DerivedFromOrderID),
				"ReceivingWindowOpen":   nullableString(o.ReceivingWindowOpen),
				"ReceivingWindowClose":  nullableString(o.ReceivingWindowClose),
				"PreorderReminderSentAt": nullableTime(o.PreorderReminderSentAt),
				"NudgeNotifiedAt":       nullableTime(o.NudgeNotifiedAt),
				"ConfirmationNotifiedAt": nullableTime(o.ConfirmationNotifiedAt),
				"CancelLockedAt":        nullableTime(o.CancelLockedAt),
				"CancelLockReason":      nullableString(o.CancelLockReason),
				"ProposedDeliveryDate":  nullableTime(o.ProposedDeliveryDate),
				"DeliveryProposalAt":    nullableTime(o.DeliveryProposalAt),
				"DeliveryProposalBy":    nullableString(o.DeliveryProposalBy),
				"DeliveryProposalReason": nullableString(o.DeliveryProposalReason),
				"Version":               o.Version,
				"CreatedAt":             o.CreatedAt.UTC(),
				"UpdatedAt":             o.UpdatedAt.UTC(),
			}),
		}

		for _, e := range buf.events {
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

		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("create order transaction: %w", err)
	}

	return nil
}

// UpdateOrder overrides an order state, persists immutable delivery proof rows,
// and emits outbox events atomically.
func (r *SpannerRepository) UpdateOrder(ctx context.Context, o Order, proofs []DeliveryProofArtifact, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner order repository: nil client")
	}

	lineItemsRaw, err := json.Marshal(o.LineItems)
	if err != nil {
		return fmt.Errorf("marshal order line items: %w", err)
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{o.OrderID}, []string{"Version"})
		if err != nil {
			return err
		}
		var version int64
		if err := row.Columns(&version); err != nil {
			return err
		}
		if version != o.Version {
			return fmt.Errorf("optimistic concurrency conflict: expected %d, got %d", o.Version, version)
		}

		o.Version++
		o.UpdatedAt = time.Now().UTC()

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":               o.OrderID,
				"WarehouseId":           o.WarehouseID,
				"DriverId":              nullableString(o.DriverID),
				"VehicleId":             nullableString(o.VehicleID),
				"RouteId":               nullableString(o.RouteID),
				"ManifestId":            nullableString(o.ManifestID),
				"DeliveryToken":         nullableString(o.QRToken),
				"Status":                string(o.Status),
				"OrderSource":           string(o.Source),
				"ConfirmationStatus":    string(o.ConfirmationStatus),
				"LineItemsJson":         lineItemsRaw,
				"TotalMinor":            o.TotalMinor,
				"OriginalTotalMinor":    originalTotalMinorForUpdate(o),
				"Currency":              o.Currency,
				"H3Cell":                o.H3Cell,
				"Lat":                   o.Lat,
				"Lng":                   o.Lng,
				"RequestedDeliveryDate": nullableTime(o.RequestedDeliveryDate),
				"DeliverBefore":         nullableTime(o.DeliverBefore),
				"DeliveryPriority":      string(o.DeliveryPriority),
				"DeliveryFeeMinor":      o.DeliveryFeeMinor,
				"WarehouseNotes":        nullableString(o.WarehouseNotes),
				"AutoConfirmAt":         nullableTime(o.AutoConfirmAt),
				"DecisionAt":            nullableTime(o.DecisionAt),
				"DecisionBy":            nullableString(o.DecisionBy),
				"DerivedFromOrderId":    nullableString(o.DerivedFromOrderID),
				"PreorderReminderSentAt": nullableTime(o.PreorderReminderSentAt),
				"NudgeNotifiedAt":       nullableTime(o.NudgeNotifiedAt),
				"ConfirmationNotifiedAt": nullableTime(o.ConfirmationNotifiedAt),
				"CancelLockedAt":        nullableTime(o.CancelLockedAt),
				"CancelLockReason":      nullableString(o.CancelLockReason),
				"ProposedDeliveryDate":  nullableTime(o.ProposedDeliveryDate),
				"DeliveryProposalAt":    nullableTime(o.DeliveryProposalAt),
				"DeliveryProposalBy":    nullableString(o.DeliveryProposalBy),
				"DeliveryProposalReason": nullableString(o.DeliveryProposalReason),
				"Version":               o.Version,
				"UpdatedAt":             o.UpdatedAt,
			}),
		}

		for _, ret := range o.PendingSupplierReturns {
			returnID := strings.TrimSpace(ret.ReturnID)
			if returnID == "" {
				continue
			}
			mutations = append(mutations, spanner.InsertMap("SupplierReturns", map[string]any{
				"ReturnId":       returnID,
				"OrderId":        o.OrderID,
				"SkuId":          strings.TrimSpace(ret.SKU),
				"RejectedQty":    ret.RejectedQty,
				"Reason":         strings.TrimSpace(ret.Reason),
				"DriverNotes":    nullableString(ret.DriverNotes),
				"Status":         "PENDING",
				"ManifestId":     nullableString(ret.ManifestID),
				"DriverId":       nullableString(ret.DriverID),
				"WarehouseId":    nullableString(ret.WarehouseID),
				"ExpectedQty":    ret.RejectedQty,
				"ReceivedQty":    int64(0),
				"PhysicalStatus": "PENDING",
				"CreatedAt":      spanner.CommitTimestamp,
			}))
		}

		for _, proof := range proofs {
			proofID := strings.TrimSpace(proof.ProofID)
			if proofID == "" {
				continue
			}
			capturedAt := proof.CapturedAt.UTC()
			if capturedAt.IsZero() {
				capturedAt = time.Now().UTC()
			}
			mutations = append(mutations, spanner.InsertMap("OrderDeliveryProofs", map[string]any{
				"ProofId":          proofID,
				"OrderId":          o.OrderID,
				"SupplierId":       o.SupplierID,
				"RetailerId":       o.RetailerID,
				"DriverId":         strings.TrimSpace(proof.DriverID),
				"ProofType":        string(proof.ProofType),
				"QRTokenHash":      nullableString(proof.QRTokenHash),
				"ScannedTokenHash": nullableString(proof.ScannedTokenHash),
				"Latitude":         nullableFloat64(proof.Latitude),
				"Longitude":        nullableFloat64(proof.Longitude),
				"DistanceM":        nullableFloat64(proof.DistanceM),
				"CapturedAt":       capturedAt,
			}))
		}

		for _, e := range buf.events {
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

		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("update order transaction: %w", err)
	}

	return nil
}

// GetOrder fetches one order aggregate by id.
func (r *SpannerRepository) GetOrder(ctx context.Context, orderID string) (Order, bool, error) {
	if r == nil || r.client == nil {
		return Order{}, false, fmt.Errorf("spanner order repository: nil client")
	}

	row, err := r.client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID}, strings.Split(orderSelectColumns, ", "))
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return Order{}, false, nil
		}
		return Order{}, false, fmt.Errorf("read order %s: %w", orderID, err)
	}

	o, err := scanOrderRowRow(row)
	if err != nil {
		return Order{}, false, fmt.Errorf("scan order %s: %w", orderID, err)
	}
	return o, true, nil
}

// ListRetailerOrders returns recent retailer-scoped orders newest first.
func (r *SpannerRepository) ListRetailerOrders(ctx context.Context, retailerID string, limit int) ([]Order, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner order repository: nil client")
	}
	if limit <= 0 {
		limit = 25
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + `
			FROM Orders@{FORCE_INDEX=Idx_Orders_ByRetailerCreated}
			WHERE RetailerId = @retailer_id
			ORDER BY CreatedAt DESC
			LIMIT @limit`,
		Params: map[string]any{
			"retailer_id": strings.TrimSpace(retailerID),
			"limit":       int64(limit),
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	return collectOrders(iter)
}

// ListWarehouseOrdersByDeliveryWindow returns future-dated orders for planning.
func (r *SpannerRepository) ListWarehouseOrdersByDeliveryWindow(ctx context.Context, warehouseID string, from, to time.Time, limit int) ([]Order, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner order repository: nil client")
	}
	if limit <= 0 {
		limit = 200
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + `
			FROM Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseRequestedDelivery}
			WHERE WarehouseId = @warehouse_id
			AND RequestedDeliveryDate >= @from
			AND RequestedDeliveryDate < @to
			ORDER BY RequestedDeliveryDate DESC, UpdatedAt DESC
			LIMIT @limit`,
		Params: map[string]any{
			"warehouse_id": strings.TrimSpace(warehouseID),
			"from":         from.UTC(),
			"to":           to.UTC(),
			"limit":        int64(limit),
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	return collectOrders(iter)
}

// ListDueAutoConfirmOrders returns AI preorders whose auto-confirm deadline has elapsed.
func (r *SpannerRepository) ListDueAutoConfirmOrders(ctx context.Context, before time.Time, limit int) ([]Order, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner order repository: nil client")
	}
	if limit <= 0 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + `
			FROM Orders@{FORCE_INDEX=Idx_Orders_ByConfirmationAutoConfirm}
			WHERE ConfirmationStatus = @confirmation_status
			AND AutoConfirmAt IS NOT NULL
			AND AutoConfirmAt <= @before
			ORDER BY AutoConfirmAt ASC, UpdatedAt DESC
			LIMIT @limit`,
		Params: map[string]any{
			"confirmation_status": string(ConfirmationStatusPending),
			"before":              before.UTC(),
			"limit":               int64(limit),
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	return collectOrders(iter)
}

func collectOrders(iter *spanner.RowIterator) ([]Order, error) {
	orders := make([]Order, 0)
	err := iter.Do(func(row *spanner.Row) error {
		orderRecord, err := scanOrderRowRow(row)
		if err != nil {
			return err
		}
		orders = append(orders, orderRecord)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func scanOrderRowRow(row *spanner.Row) (Order, error) {
	var (
		statusRaw              string
		sourceRaw              string
		confirmationRaw        string
		lineItemsRaw           []byte
		driverID               spanner.NullString
		vehicleID              spanner.NullString
		routeID                spanner.NullString
		manifestID             spanner.NullString
		deliveryToken          spanner.NullString
		receivingWindowOpen    spanner.NullString
		receivingWindowClose   spanner.NullString
		decisionBy             spanner.NullString
		derivedFromOrderID     spanner.NullString
		warehouseNotes         spanner.NullString
		cancelLockReason       spanner.NullString
		deliveryPriorityRaw    spanner.NullString
		requestedDeliveryDate  spanner.NullTime
		deliverBefore          spanner.NullTime
		autoConfirmAt          spanner.NullTime
		decisionAt             spanner.NullTime
		preorderReminderSentAt spanner.NullTime
		nudgeNotifiedAt        spanner.NullTime
		confirmationNotifiedAt spanner.NullTime
		cancelLockedAt         spanner.NullTime
		proposedDeliveryDate   spanner.NullTime
		deliveryProposalAt     spanner.NullTime
		deliveryProposalBy     spanner.NullString
		deliveryProposalReason spanner.NullString
		deliveryFeeMinor       int64
		orderRecord            Order
	)
	if err := row.Columns(
		&orderRecord.OrderID,
		&orderRecord.SupplierID,
		&orderRecord.RetailerID,
		&orderRecord.WarehouseID,
		&driverID,
		&vehicleID,
		&routeID,
		&manifestID,
		&deliveryToken,
		&statusRaw,
		&sourceRaw,
		&confirmationRaw,
		&lineItemsRaw,
		&orderRecord.TotalMinor,
		&orderRecord.OriginalTotalMinor,
		&orderRecord.Currency,
		&orderRecord.H3Cell,
		&orderRecord.Lat,
		&orderRecord.Lng,
		&requestedDeliveryDate,
		&deliverBefore,
		&deliveryPriorityRaw,
		&deliveryFeeMinor,
		&warehouseNotes,
		&autoConfirmAt,
		&decisionAt,
		&decisionBy,
		&derivedFromOrderID,
		&receivingWindowOpen,
		&receivingWindowClose,
		&preorderReminderSentAt,
		&nudgeNotifiedAt,
		&confirmationNotifiedAt,
		&cancelLockedAt,
		&cancelLockReason,
		&proposedDeliveryDate,
		&deliveryProposalAt,
		&deliveryProposalBy,
		&deliveryProposalReason,
		&orderRecord.Version,
		&orderRecord.CreatedAt,
		&orderRecord.UpdatedAt,
	); err != nil {
		return Order{}, err
	}
	orderRecord.DriverID = driverID.StringVal
	orderRecord.VehicleID = vehicleID.StringVal
	orderRecord.RouteID = routeID.StringVal
	orderRecord.ManifestID = manifestID.StringVal
	orderRecord.QRToken = deliveryToken.StringVal
	orderRecord.Status = Status(statusRaw)
	orderRecord.Source = OrderSource(sourceRaw)
	orderRecord.ConfirmationStatus = ConfirmationStatus(confirmationRaw)
	if deliveryPriorityRaw.Valid {
		orderRecord.DeliveryPriority = DeliveryPriority(deliveryPriorityRaw.StringVal)
	}
	orderRecord.DeliveryFeeMinor = deliveryFeeMinor
	orderRecord.WarehouseNotes = warehouseNotes.StringVal
	orderRecord.CancelLockReason = cancelLockReason.StringVal
	if requestedDeliveryDate.Valid {
		requested := requestedDeliveryDate.Time.UTC()
		orderRecord.RequestedDeliveryDate = &requested
	}
	if deliverBefore.Valid {
		deliver := deliverBefore.Time.UTC()
		orderRecord.DeliverBefore = &deliver
	}
	if autoConfirmAt.Valid {
		autoConfirm := autoConfirmAt.Time.UTC()
		orderRecord.AutoConfirmAt = &autoConfirm
	}
	if decisionAt.Valid {
		decision := decisionAt.Time.UTC()
		orderRecord.DecisionAt = &decision
	}
	orderRecord.DecisionBy = decisionBy.StringVal
	orderRecord.DerivedFromOrderID = derivedFromOrderID.StringVal
	orderRecord.ReceivingWindowOpen = receivingWindowOpen.StringVal
	orderRecord.ReceivingWindowClose = receivingWindowClose.StringVal
	if preorderReminderSentAt.Valid {
		t := preorderReminderSentAt.Time.UTC()
		orderRecord.PreorderReminderSentAt = &t
	}
	if nudgeNotifiedAt.Valid {
		t := nudgeNotifiedAt.Time.UTC()
		orderRecord.NudgeNotifiedAt = &t
	}
	if confirmationNotifiedAt.Valid {
		t := confirmationNotifiedAt.Time.UTC()
		orderRecord.ConfirmationNotifiedAt = &t
	}
	if cancelLockedAt.Valid {
		t := cancelLockedAt.Time.UTC()
		orderRecord.CancelLockedAt = &t
	}
	if proposedDeliveryDate.Valid {
		t := proposedDeliveryDate.Time.UTC()
		orderRecord.ProposedDeliveryDate = &t
	}
	if deliveryProposalAt.Valid {
		t := deliveryProposalAt.Time.UTC()
		orderRecord.DeliveryProposalAt = &t
	}
	orderRecord.DeliveryProposalBy = deliveryProposalBy.StringVal
	orderRecord.DeliveryProposalReason = deliveryProposalReason.StringVal
	if orderRecord.OriginalTotalMinor == 0 {
		orderRecord.OriginalTotalMinor = orderRecord.TotalMinor
	}
	if len(lineItemsRaw) > 0 {
		if err := json.Unmarshal(lineItemsRaw, &orderRecord.LineItems); err != nil {
			return Order{}, err
		}
	}
	return orderRecord, nil
}

func nullableString(value string) spanner.NullString {
	trimmed := strings.TrimSpace(value)
	return spanner.NullString{StringVal: trimmed, Valid: trimmed != ""}
}

func nullableTime(value *time.Time) interface{} {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC()
}

func nullableFloat64(value *float64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

// ListManifestOrders returns all orders linked to the specified manifest.
func (r *SpannerRepository) ListManifestOrders(ctx context.Context, manifestID string) ([]Order, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT " + orderSelectColumns + " FROM Orders WHERE ManifestId = @mid ORDER BY CreatedAt ASC",
		Params: map[string]any{"mid": manifestID},
	}
	var res []Order
	err := r.client.Single().Query(ctx, stmt).Do(func(row *spanner.Row) error {
		o, err := scanOrderRowRow(row)
		if err != nil {
			return err
		}
		res = append(res, o)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list manifest orders %s: %w", manifestID, err)
	}
	return res, nil
}

// ListWarehousePreorders returns scheduled/auto-accepted pre-orders for a warehouse node.
func (r *SpannerRepository) ListWarehousePreorders(ctx context.Context, warehouseID string, limit, offset int) ([]Order, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + ` FROM Orders
		      WHERE WarehouseId = @wid
		        AND OrderSource = @src
		        AND Status IN ('SCHEDULED', 'AUTO_ACCEPTED')
		      ORDER BY RequestedDeliveryDate ASC, UpdatedAt DESC
		      LIMIT @lim OFFSET @off`,
		Params: map[string]any{
			"wid": warehouseID,
			"src": string(OrderSourceManualPreorder),
			"lim": limit,
			"off": offset,
		},
	}
	return r.queryOrders(ctx, stmt)
}

// ListOrdersForStockCommitment returns active orders that consume warehouse stock projections.
func (r *SpannerRepository) ListOrdersForStockCommitment(ctx context.Context, warehouseID string, limit int) ([]Order, error) {
	if limit <= 0 {
		limit = 500
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + ` FROM Orders
		      WHERE WarehouseId = @wid
		        AND Status IN ('PENDING', 'SCHEDULED', 'AUTO_ACCEPTED', 'LOADED', 'IN_TRANSIT', 'DELAYED')
		      ORDER BY UpdatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"wid": warehouseID, "lim": limit},
	}
	return r.queryOrders(ctx, stmt)
}

func (r *SpannerRepository) queryOrders(ctx context.Context, stmt spanner.Statement) ([]Order, error) {
	var res []Order
	err := r.client.Single().Query(ctx, stmt).Do(func(row *spanner.Row) error {
		o, err := scanOrderRowRow(row)
		if err != nil {
			return err
		}
		res = append(res, o)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func originalTotalMinorForInsert(o *Order) int64 {
	if o == nil {
		return 0
	}
	if o.OriginalTotalMinor > 0 {
		return o.OriginalTotalMinor
	}
	return o.TotalMinor
}

func originalTotalMinorForUpdate(o Order) int64 {
	if o.OriginalTotalMinor > 0 {
		return o.OriginalTotalMinor
	}
	return o.TotalMinor
}
