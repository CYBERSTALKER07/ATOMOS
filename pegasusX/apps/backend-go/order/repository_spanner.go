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

const orderSelectColumns = `OrderId, SupplierId, RetailerId, WarehouseId, DriverId, VehicleId, RouteId, ManifestId, DeliveryToken, Status, OrderSource, ConfirmationStatus, LineItemsJson, TotalMinor, Currency, H3Cell, Lat, Lng, RequestedDeliveryDate, AutoConfirmAt, DecisionAt, DecisionBy, DerivedFromOrderId, ReceivingWindowOpen, ReceivingWindowClose, Version, CreatedAt, UpdatedAt`

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

		// Phase 2 check: Exhaustion check and increment QuantityReserved for warehouse assigned items
		if len(o.LineItems) > 0 && o.WarehouseID != "" {
			for _, item := range o.LineItems {
				row, err := txn.ReadRow(ctx, "SupplierInventoryV2",
					spanner.Key{o.SupplierID, o.WarehouseID, item.SKU},
					[]string{"QuantityOnHand", "QuantityReserved"})
				if err != nil {
					if spanner.ErrCode(err) == 5 { // NotFound
						return fmt.Errorf("%w: sku %s not found in warehouse %s", ErrInventoryExhausted, item.SKU, o.WarehouseID)
					}
					return fmt.Errorf("read inventory %s: %w", item.SKU, err)
				}
				var qoh, qr int64
				if err := row.Columns(&qoh, &qr); err != nil {
					return fmt.Errorf("decode inventory columns: %w", err)
				}
				if qoh-qr < item.Quantity {
					return fmt.Errorf("%w: sku %s has %d available, requested %d", ErrInventoryExhausted, item.SKU, qoh-qr, item.Quantity)
				}
				mut := spanner.UpdateMap("SupplierInventoryV2", map[string]any{
					"SupplierId":       o.SupplierID,
					"WarehouseId":      o.WarehouseID,
					"ProductId":        item.SKU,
					"QuantityReserved": qr + item.Quantity,
					"UpdatedAt":        spanner.CommitTimestamp,
				})
				if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
					return fmt.Errorf("buffer inventory update: %w", err)
				}
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
				"Currency":              o.Currency,
				"H3Cell":                o.H3Cell,
				"Lat":                   o.Lat,
				"Lng":                   o.Lng,
				"RequestedDeliveryDate": nullableTime(o.RequestedDeliveryDate),
				"AutoConfirmAt":         nullableTime(o.AutoConfirmAt),
				"DecisionAt":            nullableTime(o.DecisionAt),
				"DecisionBy":            nullableString(o.DecisionBy),
				"DerivedFromOrderId":    nullableString(o.DerivedFromOrderID),
				"ReceivingWindowOpen":   nullableString(o.ReceivingWindowOpen),
				"ReceivingWindowClose":  nullableString(o.ReceivingWindowClose),
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
				"Currency":              o.Currency,
				"H3Cell":                o.H3Cell,
				"Lat":                   o.Lat,
				"Lng":                   o.Lng,
				"RequestedDeliveryDate": nullableTime(o.RequestedDeliveryDate),
				"AutoConfirmAt":         nullableTime(o.AutoConfirmAt),
				"DecisionAt":            nullableTime(o.DecisionAt),
				"DecisionBy":            nullableString(o.DecisionBy),
				"DerivedFromOrderId":    nullableString(o.DerivedFromOrderID),
				"Version":               o.Version,
				"UpdatedAt":             o.UpdatedAt,
			}),
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
		statusRaw             string
		sourceRaw             string
		confirmationRaw       string
		lineItemsRaw          []byte
		driverID              spanner.NullString
		vehicleID             spanner.NullString
		routeID               spanner.NullString
		manifestID            spanner.NullString
		deliveryToken         spanner.NullString
		receivingWindowOpen   spanner.NullString
		receivingWindowClose  spanner.NullString
		decisionBy            spanner.NullString
		derivedFromOrderID    spanner.NullString
		requestedDeliveryDate spanner.NullTime
		autoConfirmAt         spanner.NullTime
		decisionAt            spanner.NullTime
		orderRecord           Order
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
		&orderRecord.Currency,
		&orderRecord.H3Cell,
		&orderRecord.Lat,
		&orderRecord.Lng,
		&requestedDeliveryDate,
		&autoConfirmAt,
		&decisionAt,
		&decisionBy,
		&derivedFromOrderID,
		&receivingWindowOpen,
		&receivingWindowClose,
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
	if requestedDeliveryDate.Valid {
		requested := requestedDeliveryDate.Time.UTC()
		orderRecord.RequestedDeliveryDate = &requested
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
