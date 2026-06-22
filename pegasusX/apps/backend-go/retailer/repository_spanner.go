package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/packages/handoff"
	"google.golang.org/api/iterator"
)

// SpannerRepository persists retailer rows in Spanner and writes emitted outbox
// events in the same RW transaction.
type SpannerRepository struct {
	client     *spanner.Client
	supplierID string
}

func retailerPersistenceMap(ret Retailer) map[string]any {
	row := map[string]any{
		"RetailerId":  ret.RetailerID,
		"Phone":       ret.Phone,
		"Name":        ret.Name,
		"CountryCode": ret.CountryCode,
		"Lat":         ret.Lat,
		"Lng":         ret.Lng,
		"H3Cell":      ret.H3Cell,
	}
	if addr := strings.TrimSpace(ret.DeliveryAddress); addr != "" {
		row["DeliveryAddress"] = addr
	}
	if pid := strings.TrimSpace(ret.PlaceID); pid != "" {
		row["PlaceId"] = pid
	}
	if open := strings.TrimSpace(ret.ReceivingWindowOpen); open != "" {
		row["ReceivingWindowOpen"] = open
	} else {
		row["ReceivingWindowOpen"] = nil
	}
	if close := strings.TrimSpace(ret.ReceivingWindowClose); close != "" {
		row["ReceivingWindowClose"] = close
	} else {
		row["ReceivingWindowClose"] = nil
	}
	return row
}

func applyRetailerWindowColumns(ret *Retailer, rwOpen, rwClose spanner.NullString) {
	if rwOpen.Valid {
		ret.ReceivingWindowOpen = rwOpen.StringVal
	}
	if rwClose.Valid {
		ret.ReceivingWindowClose = rwClose.StringVal
	}
}

type orderLineItemJSON struct {
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price_minor"`
}

const (
	receiptPaymentTimelineLimit = 12
	receiptGatewayWebhookLimit  = 6
	receiptDeliveryProofLimit   = 4
	receiptChargebackLimit      = 6
	receiptReversalLimit        = 6
)

type trackingReceiptPaymentRecordSnapshot struct {
	Record     TrackingReceiptPaymentRecord
	OccurredAt time.Time
	CreatedAt  time.Time
}

// NewSpannerRepository builds a Spanner-backed retailer repository.
func NewSpannerRepository(client *spanner.Client, supplierID string) *SpannerRepository {
	return &SpannerRepository{client: client, supplierID: strings.TrimSpace(supplierID)}
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

// CreateRetailer writes the Retailers row and any emitted outbox events atomically.
func (r *SpannerRepository) CreateRetailer(ctx context.Context, ret Retailer, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner retailer repository: nil client")
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		insertMap := retailerPersistenceMap(ret)
		insertMap["CreatedAt"] = ret.CreatedAt.UTC()
		mutations := []*spanner.Mutation{
			spanner.InsertMap("Retailers", insertMap),
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
		return fmt.Errorf("create retailer transaction: %w", err)
	}

	return nil
}

// FindByPhone looks up a retailer by its unique phone number.
func (r *SpannerRepository) FindByPhone(ctx context.Context, phone string) (Retailer, bool, error) {
	if r == nil || r.client == nil {
		return Retailer{}, false, fmt.Errorf("spanner retailer repository: nil client")
	}

	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Phone, Name, CountryCode, Lat, Lng, H3Cell,
			         ReceivingWindowOpen, ReceivingWindowClose, CreatedAt
			  FROM Retailers
			  WHERE Phone = @Phone
			  LIMIT 1`,
		Params: map[string]interface{}{
			"Phone": phone,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return Retailer{}, false, nil
	}
	if err != nil {
		return Retailer{}, false, fmt.Errorf("query retailer by phone: %w", err)
	}

	var ret Retailer
	var rwOpen, rwClose spanner.NullString
	if err := row.Columns(
		&ret.RetailerID,
		&ret.Phone,
		&ret.Name,
		&ret.CountryCode,
		&ret.Lat,
		&ret.Lng,
		&ret.H3Cell,
		&rwOpen,
		&rwClose,
		&ret.CreatedAt,
	); err != nil {
		return Retailer{}, false, fmt.Errorf("scan retailer by phone: %w", err)
	}
	applyRetailerWindowColumns(&ret, rwOpen, rwClose)
	ret.SupplierID = r.supplierID

	return ret, true, nil
}

// GetRetailer fetches one retailer aggregate by id.
func (r *SpannerRepository) GetRetailer(ctx context.Context, retailerID string) (Retailer, bool, error) {
	if r == nil || r.client == nil {
		return Retailer{}, false, fmt.Errorf("spanner retailer repository: nil client")
	}

	row, err := r.client.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{
		"RetailerId",
		"Phone",
		"Name",
		"CountryCode",
		"Lat",
		"Lng",
		"H3Cell",
		"DeliveryAddress",
		"PlaceId",
		"ReceivingWindowOpen",
		"ReceivingWindowClose",
		"CreatedAt",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return Retailer{}, false, nil
		}
		return Retailer{}, false, fmt.Errorf("read retailer %s: %w", retailerID, err)
	}

	var ret Retailer
	var rwOpen, rwClose spanner.NullString
	var deliveryAddress, placeID spanner.NullString
	if err := row.Columns(
		&ret.RetailerID,
		&ret.Phone,
		&ret.Name,
		&ret.CountryCode,
		&ret.Lat,
		&ret.Lng,
		&ret.H3Cell,
		&deliveryAddress,
		&placeID,
		&rwOpen,
		&rwClose,
		&ret.CreatedAt,
	); err != nil {
		return Retailer{}, false, fmt.Errorf("scan retailer %s: %w", retailerID, err)
	}
	if deliveryAddress.Valid {
		ret.DeliveryAddress = deliveryAddress.StringVal
	}
	if placeID.Valid {
		ret.PlaceID = placeID.StringVal
	}
	applyRetailerWindowColumns(&ret, rwOpen, rwClose)
	ret.SupplierID = r.supplierID

	return ret, true, nil
}

// UpdateRetailer updates the Retailers row and any emitted outbox events atomically.
func (r *SpannerRepository) UpdateRetailer(ctx context.Context, ret Retailer, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner retailer repository: nil client")
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Retailers", retailerPersistenceMap(ret)),
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
		return fmt.Errorf("update retailer transaction: %w", err)
	}

	return nil
}

// ListRetailersBySupplier lists all retailers. (Note: in PegasusX, SupplierId is not saved in Retailers table)
func (r *SpannerRepository) ListRetailersBySupplier(ctx context.Context, supplierID string) ([]Retailer, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner retailer repository: nil client")
	}

	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Phone, Name, CountryCode, Lat, Lng, H3Cell,
			         ReceivingWindowOpen, ReceivingWindowClose, CreatedAt
			  FROM Retailers`,
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	effectiveSupplierID := strings.TrimSpace(supplierID)
	if effectiveSupplierID == "" {
		effectiveSupplierID = r.supplierID
	}
	if r.supplierID != "" && effectiveSupplierID != "" && effectiveSupplierID != r.supplierID {
		return []Retailer{}, nil
	}

	var retailers []Retailer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query all retailers: %w", err)
		}

		var ret Retailer
		var rwOpen, rwClose spanner.NullString
		if err := row.Columns(
			&ret.RetailerID,
			&ret.Phone,
			&ret.Name,
			&ret.CountryCode,
			&ret.Lat,
			&ret.Lng,
			&ret.H3Cell,
			&rwOpen,
			&rwClose,
			&ret.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan list retailers: %w", err)
		}
		applyRetailerWindowColumns(&ret, rwOpen, rwClose)

		// Map supplier ID since the schema does not have it. In single-tenant pegasusX this is safe.
		ret.SupplierID = effectiveSupplierID
		retailers = append(retailers, ret)
	}

	return retailers, nil
}

// ListTrackingOrders returns active retailer orders with durable assignment fields.
func (r *SpannerRepository) ListTrackingOrders(ctx context.Context, retailerID string, limit, offset int) ([]TrackingOrder, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner retailer repository: nil client")
	}
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return []TrackingOrder{}, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, RetailerId,
		             COALESCE(WarehouseId, ''), COALESCE(DriverId, ''), COALESCE(VehicleId, ''),
		             COALESCE(RouteId, ''), COALESCE(ManifestId, ''), COALESCE(DeliveryToken, ''), Status, LineItemsJson,
		             TotalMinor, Currency, CreatedAt, UpdatedAt, Lat, Lng
		      FROM Orders@{FORCE_INDEX=Idx_Orders_ByRetailerCreated}
		      WHERE RetailerId = @RetailerId
		        AND Status IN UNNEST(@Statuses)
		      ORDER BY CreatedAt DESC
		      LIMIT @Limit
		      OFFSET @Offset`,
		Params: map[string]interface{}{
			"RetailerId": retailerID,
			"Statuses":   []string{"PENDING", "LOADED", "IN_TRANSIT", "ARRIVED", "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION"},
			"Limit":      int64(limit),
			"Offset":     int64(offset),
		},
	}

	return r.queryTrackingOrders(ctx, stmt, "query retailer tracking orders")
}

// ListRecentReceipts returns recent completed retailer orders as receipt snapshots.
func (r *SpannerRepository) ListRecentReceipts(ctx context.Context, retailerID string, limit int) ([]TrackingOrder, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner retailer repository: nil client")
	}
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return []TrackingOrder{}, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, RetailerId,
		             COALESCE(WarehouseId, ''), COALESCE(DriverId, ''), COALESCE(VehicleId, ''),
		             COALESCE(RouteId, ''), COALESCE(ManifestId, ''), COALESCE(DeliveryToken, ''), Status, LineItemsJson,
		             TotalMinor, Currency, CreatedAt, UpdatedAt, Lat, Lng
		      FROM Orders
		      WHERE RetailerId = @RetailerId
		        AND Status = @Status
		      ORDER BY UpdatedAt DESC
		      LIMIT @Limit`,
		Params: map[string]interface{}{
			"RetailerId": retailerID,
			"Status":     "COMPLETED",
			"Limit":      int64(limit),
		},
	}

	orders, err := r.queryTrackingOrders(ctx, stmt, "query retailer recent receipts")
	if err != nil {
		return nil, err
	}
	if err := r.attachReceiptDossiers(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *SpannerRepository) attachReceiptDossiers(ctx context.Context, orders []TrackingOrder) error {
	if r == nil || r.client == nil || len(orders) == 0 {
		return nil
	}

	orderIDs := uniqueTrackingOrderIDs(orders)
	if len(orderIDs) == 0 {
		return nil
	}

	orderTimelineByOrder, sessionIDsByOrder, err := r.listOrderPaymentTimeline(ctx, orderIDs)
	if err != nil {
		return err
	}
	sessionTimelineBySession, err := r.listSessionPaymentTimeline(ctx, sessionIDsByOrder)
	if err != nil {
		return err
	}
	webhooksByOrder, err := r.listOrderGatewayWebhooks(ctx, orderIDs)
	if err != nil {
		return err
	}
	proofsByOrder, err := r.listOrderDeliveryProofs(ctx, orderIDs)
	if err != nil {
		return err
	}
	chargebacksByOrder, err := r.listOrderChargebacks(ctx, orderIDs)
	if err != nil {
		return err
	}
	reversalsBySession, err := r.listSessionReversals(ctx, sessionIDsByOrder)
	if err != nil {
		return err
	}
	dossiersByOrder := buildTrackingReceiptDossiers(orders, orderTimelineByOrder, sessionTimelineBySession, sessionIDsByOrder, webhooksByOrder, proofsByOrder, chargebacksByOrder, reversalsBySession)

	for index := range orders {
		orderID := strings.TrimSpace(orders[index].OrderID)
		dossier, found := dossiersByOrder[orderID]
		if !found {
			continue
		}
		dossierCopy := dossier
		orders[index].ReceiptDossier = &dossierCopy
		if len(dossierCopy.PaymentTimeline) > 0 {
			evidence := paymentEvidenceFromReceiptRecord(dossierCopy.PaymentTimeline[0])
			orders[index].PaymentEvidence = &evidence
		}
	}

	return nil
}

func (r *SpannerRepository) listOrderPaymentTimeline(ctx context.Context, orderIDs []string) (map[string][]trackingReceiptPaymentRecordSnapshot, map[string]string, error) {
	if r == nil || r.client == nil || len(orderIDs) == 0 {
		return map[string][]trackingReceiptPaymentRecordSnapshot{}, map[string]string{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, LedgerEntryId, COALESCE(SessionId, ''), Gateway, EntryType, AmountMinor, Currency,
		             COALESCE(ReferenceId, ''), Source, OccurredAt, CreatedAt
		      FROM PaymentLedgerEntries
		      WHERE OrderId IN UNNEST(@OrderIds)
		      ORDER BY OrderId ASC, OccurredAt DESC`,
		Params: map[string]interface{}{
			"OrderIds": orderIDs,
		},
	}

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	timelineByOrder := make(map[string][]trackingReceiptPaymentRecordSnapshot, len(orderIDs))
	sessionIDsByOrder := make(map[string]string, len(orderIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("query retailer receipt payment timeline: %w", err)
		}

		var (
			orderID     string
			sessionID   string
			record      TrackingReceiptPaymentRecord
			referenceID string
			occurredAt  time.Time
			createdAt   time.Time
		)
		if err := row.Columns(
			&orderID,
			&record.LedgerEntryID,
			&sessionID,
			&record.Gateway,
			&record.EntryType,
			&record.AmountMinor,
			&record.Currency,
			&referenceID,
			&record.Source,
			&occurredAt,
			&createdAt,
		); err != nil {
			return nil, nil, fmt.Errorf("scan retailer receipt payment timeline: %w", err)
		}
		orderID = strings.TrimSpace(orderID)
		sessionID = strings.TrimSpace(sessionID)
		if sessionID != "" && sessionIDsByOrder[orderID] == "" {
			sessionIDsByOrder[orderID] = sessionID
		}
		if len(timelineByOrder[orderID]) >= receiptPaymentTimelineLimit {
			continue
		}
		record.OrderID = orderID
		record.SessionID = sessionID
		record.ReferenceID = strings.TrimSpace(referenceID)
		record.OccurredAt = occurredAt.UTC().Format(time.RFC3339Nano)
		record.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		timelineByOrder[orderID] = append(timelineByOrder[orderID], trackingReceiptPaymentRecordSnapshot{
			Record:     record,
			OccurredAt: occurredAt.UTC(),
			CreatedAt:  createdAt.UTC(),
		})
	}

	return timelineByOrder, sessionIDsByOrder, nil
}

func (r *SpannerRepository) listSessionPaymentTimeline(ctx context.Context, sessionIDsByOrder map[string]string) (map[string][]trackingReceiptPaymentRecordSnapshot, error) {
	sessionIDs := uniqueTrackingSessionIDs(sessionIDsByOrder)
	if r == nil || r.client == nil || len(sessionIDs) == 0 {
		return map[string][]trackingReceiptPaymentRecordSnapshot{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SessionId, ''), LedgerEntryId, COALESCE(OrderId, ''), Gateway, EntryType, AmountMinor, Currency,
		             COALESCE(ReferenceId, ''), Source, OccurredAt, CreatedAt
		      FROM PaymentLedgerEntries
		      WHERE SessionId IN UNNEST(@SessionIds)
		      ORDER BY SessionId ASC, OccurredAt DESC`,
		Params: map[string]interface{}{
			"SessionIds": sessionIDs,
		},
	}

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	timelineBySession := make(map[string][]trackingReceiptPaymentRecordSnapshot, len(sessionIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query retailer receipt session timeline: %w", err)
		}

		var (
			sessionID   string
			orderID     string
			record      TrackingReceiptPaymentRecord
			referenceID string
			occurredAt  time.Time
			createdAt   time.Time
		)
		if err := row.Columns(
			&sessionID,
			&record.LedgerEntryID,
			&orderID,
			&record.Gateway,
			&record.EntryType,
			&record.AmountMinor,
			&record.Currency,
			&referenceID,
			&record.Source,
			&occurredAt,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan retailer receipt session timeline: %w", err)
		}
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			continue
		}
		if len(timelineBySession[sessionID]) >= receiptPaymentTimelineLimit {
			continue
		}
		record.SessionID = sessionID
		record.OrderID = strings.TrimSpace(orderID)
		record.ReferenceID = strings.TrimSpace(referenceID)
		record.OccurredAt = occurredAt.UTC().Format(time.RFC3339Nano)
		record.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		timelineBySession[sessionID] = append(timelineBySession[sessionID], trackingReceiptPaymentRecordSnapshot{
			Record:     record,
			OccurredAt: occurredAt.UTC(),
			CreatedAt:  createdAt.UTC(),
		})
	}

	return timelineBySession, nil
}

func (r *SpannerRepository) listOrderGatewayWebhooks(ctx context.Context, orderIDs []string) (map[string][]TrackingReceiptGatewayWebhook, error) {
	if r == nil || r.client == nil || len(orderIDs) == 0 {
		return map[string][]TrackingReceiptGatewayWebhook{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, WebhookId, COALESCE(SessionId, ''), Gateway, TransactionId, Status,
		             AmountMinor, Currency, SignatureValid, ReceivedAt
		      FROM PaymentWebhooks
		      WHERE OrderId IN UNNEST(@OrderIds)
		      ORDER BY OrderId ASC, ReceivedAt DESC`,
		Params: map[string]interface{}{
			"OrderIds": orderIDs,
		},
	}

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	webhooksByOrder := make(map[string][]TrackingReceiptGatewayWebhook, len(orderIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query retailer receipt gateway webhooks: %w", err)
		}

		var (
			orderID    string
			sessionID  string
			webhook    TrackingReceiptGatewayWebhook
			receivedAt time.Time
		)
		if err := row.Columns(
			&orderID,
			&webhook.WebhookID,
			&sessionID,
			&webhook.Gateway,
			&webhook.TransactionID,
			&webhook.Status,
			&webhook.AmountMinor,
			&webhook.Currency,
			&webhook.SignatureValid,
			&receivedAt,
		); err != nil {
			return nil, fmt.Errorf("scan retailer receipt gateway webhooks: %w", err)
		}
		orderID = strings.TrimSpace(orderID)
		if orderID == "" || len(webhooksByOrder[orderID]) >= receiptGatewayWebhookLimit {
			continue
		}
		webhook.SessionID = strings.TrimSpace(sessionID)
		webhook.ReceivedAt = receivedAt.UTC().Format(time.RFC3339Nano)
		webhooksByOrder[orderID] = append(webhooksByOrder[orderID], webhook)
	}

	return webhooksByOrder, nil
}

func (r *SpannerRepository) listOrderDeliveryProofs(ctx context.Context, orderIDs []string) (map[string][]TrackingReceiptDeliveryProof, error) {
	if r == nil || r.client == nil || len(orderIDs) == 0 {
		return map[string][]TrackingReceiptDeliveryProof{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, ProofId, ProofType, QRTokenHash, ScannedTokenHash, Latitude, Longitude, DistanceM, CapturedAt
		      FROM OrderDeliveryProofs
		      WHERE OrderId IN UNNEST(@OrderIds)
		      ORDER BY OrderId ASC, CapturedAt DESC`,
		Params: map[string]interface{}{
			"OrderIds": orderIDs,
		},
	}

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	proofsByOrder := make(map[string][]TrackingReceiptDeliveryProof, len(orderIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query retailer receipt delivery proofs: %w", err)
		}

		var (
			orderID          string
			proof            TrackingReceiptDeliveryProof
			qrTokenHash      spanner.NullString
			scannedTokenHash spanner.NullString
			latitude         spanner.NullFloat64
			longitude        spanner.NullFloat64
			distanceM        spanner.NullFloat64
			capturedAt       time.Time
		)
		if err := row.Columns(
			&orderID,
			&proof.ProofID,
			&proof.ProofType,
			&qrTokenHash,
			&scannedTokenHash,
			&latitude,
			&longitude,
			&distanceM,
			&capturedAt,
		); err != nil {
			return nil, fmt.Errorf("scan retailer receipt delivery proofs: %w", err)
		}
		orderID = strings.TrimSpace(orderID)
		if orderID == "" || len(proofsByOrder[orderID]) >= receiptDeliveryProofLimit {
			continue
		}
		proof.QRTokenHashPresent = qrTokenHash.Valid && strings.TrimSpace(qrTokenHash.StringVal) != ""
		proof.ScannedTokenHashPresent = scannedTokenHash.Valid && strings.TrimSpace(scannedTokenHash.StringVal) != ""
		if latitude.Valid {
			value := latitude.Float64
			proof.Latitude = &value
		}
		if longitude.Valid {
			value := longitude.Float64
			proof.Longitude = &value
		}
		if distanceM.Valid {
			value := distanceM.Float64
			proof.DistanceM = &value
		}
		proof.CapturedAt = capturedAt.UTC().Format(time.RFC3339Nano)
		proofsByOrder[orderID] = append(proofsByOrder[orderID], proof)
	}

	return proofsByOrder, nil
}

func (r *SpannerRepository) listOrderChargebacks(ctx context.Context, orderIDs []string) (map[string][]TrackingReceiptChargebackRecord, error) {
	if r == nil || r.client == nil || len(orderIDs) == 0 {
		return map[string][]TrackingReceiptChargebackRecord{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, ChargebackId, Gateway, AmountMinor, Currency, CreatedAt
		      FROM PaymentChargebacks
		      WHERE OrderId IN UNNEST(@OrderIds)
		      ORDER BY OrderId ASC, CreatedAt DESC`,
		Params: map[string]interface{}{
			"OrderIds": orderIDs,
		},
	}

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	chargebacksByOrder := make(map[string][]TrackingReceiptChargebackRecord, len(orderIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query retailer receipt chargebacks: %w", err)
		}

		var (
			orderID   string
			record    TrackingReceiptChargebackRecord
			createdAt time.Time
		)
		if err := row.Columns(&orderID, &record.ChargebackID, &record.Gateway, &record.AmountMinor, &record.Currency, &createdAt); err != nil {
			return nil, fmt.Errorf("scan retailer receipt chargebacks: %w", err)
		}
		orderID = strings.TrimSpace(orderID)
		if orderID == "" || len(chargebacksByOrder[orderID]) >= receiptChargebackLimit {
			continue
		}
		record.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		chargebacksByOrder[orderID] = append(chargebacksByOrder[orderID], record)
	}

	return chargebacksByOrder, nil
}

func (r *SpannerRepository) listSessionReversals(ctx context.Context, sessionIDsByOrder map[string]string) (map[string][]TrackingReceiptReversalRecord, error) {
	sessionIDs := uniqueTrackingSessionIDs(sessionIDsByOrder)
	if r == nil || r.client == nil || len(sessionIDs) == 0 {
		return map[string][]TrackingReceiptReversalRecord{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT SessionId, ReversalId, CreatedAt
		      FROM PaymentReversals
		      WHERE SessionId IN UNNEST(@SessionIds)
		      ORDER BY SessionId ASC, CreatedAt DESC`,
		Params: map[string]interface{}{
			"SessionIds": sessionIDs,
		},
	}

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	reversalsBySession := make(map[string][]TrackingReceiptReversalRecord, len(sessionIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query retailer receipt reversals: %w", err)
		}

		var (
			sessionID string
			record    TrackingReceiptReversalRecord
			createdAt time.Time
		)
		if err := row.Columns(&sessionID, &record.ReversalID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan retailer receipt reversals: %w", err)
		}
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" || len(reversalsBySession[sessionID]) >= receiptReversalLimit {
			continue
		}
		record.SessionID = sessionID
		record.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		reversalsBySession[sessionID] = append(reversalsBySession[sessionID], record)
	}

	return reversalsBySession, nil
}

func buildTrackingReceiptDossiers(orders []TrackingOrder, orderTimelineByOrder map[string][]trackingReceiptPaymentRecordSnapshot, sessionTimelineBySession map[string][]trackingReceiptPaymentRecordSnapshot, sessionIDsByOrder map[string]string, webhooksByOrder map[string][]TrackingReceiptGatewayWebhook, proofsByOrder map[string][]TrackingReceiptDeliveryProof, chargebacksByOrder map[string][]TrackingReceiptChargebackRecord, reversalsBySession map[string][]TrackingReceiptReversalRecord) map[string]TrackingReceiptDossier {
	if len(orders) == 0 {
		return map[string]TrackingReceiptDossier{}
	}

	dossiers := make(map[string]TrackingReceiptDossier, len(orders))
	for _, order := range orders {
		orderID := strings.TrimSpace(order.OrderID)
		if orderID == "" {
			continue
		}
		sessionID := strings.TrimSpace(sessionIDsByOrder[orderID])
		paymentTimeline := mergeTrackingReceiptTimeline(orderTimelineByOrder[orderID], sessionTimelineBySession[sessionID])
		if sessionID == "" && len(paymentTimeline) > 0 {
			sessionID = strings.TrimSpace(paymentTimeline[0].SessionID)
		}
		dossier := newTrackingReceiptDossier(sessionID)
		dossier.PaymentTimeline = paymentTimeline
		if webhooks := webhooksByOrder[orderID]; len(webhooks) > 0 {
			dossier.GatewayWebhooks = append(dossier.GatewayWebhooks, webhooks...)
		}
		if proofs := proofsByOrder[orderID]; len(proofs) > 0 {
			dossier.DeliveryProofs = append(dossier.DeliveryProofs, proofs...)
			dossier.ProofStatus.DeliveryProofAvailable = true
			dossier.ProofStatus.MissingArtifacts = []string{}
		}
		if chargebacks := chargebacksByOrder[orderID]; len(chargebacks) > 0 {
			dossier.Chargebacks = append(dossier.Chargebacks, chargebacks...)
		}
		if reversals := reversalsBySession[sessionID]; len(reversals) > 0 {
			dossier.Reversals = enrichTrackingReceiptReversals(reversals, sessionTimelineBySession[sessionID], order.Currency)
		}
		dossier.ProofStatus.PaymentTimelineAvailable = len(dossier.PaymentTimeline) > 0
		dossier.ProofStatus.GatewayWebhooksAvailable = len(dossier.GatewayWebhooks) > 0
		dossiers[orderID] = dossier
	}

	return dossiers
}

func mergeTrackingReceiptTimeline(orderTimeline []trackingReceiptPaymentRecordSnapshot, sessionTimeline []trackingReceiptPaymentRecordSnapshot) []TrackingReceiptPaymentRecord {
	if len(orderTimeline) == 0 && len(sessionTimeline) == 0 {
		return []TrackingReceiptPaymentRecord{}
	}

	combined := make([]trackingReceiptPaymentRecordSnapshot, 0, len(orderTimeline)+len(sessionTimeline))
	seen := make(map[string]struct{}, len(orderTimeline)+len(sessionTimeline))
	appendUnique := func(items []trackingReceiptPaymentRecordSnapshot) {
		for _, item := range items {
			ledgerEntryID := strings.TrimSpace(item.Record.LedgerEntryID)
			if ledgerEntryID == "" {
				continue
			}
			if _, exists := seen[ledgerEntryID]; exists {
				continue
			}
			seen[ledgerEntryID] = struct{}{}
			combined = append(combined, item)
		}
	}
	appendUnique(orderTimeline)
	appendUnique(sessionTimeline)

	sort.SliceStable(combined, func(i, j int) bool {
		if combined[i].OccurredAt.Equal(combined[j].OccurredAt) {
			return combined[i].CreatedAt.After(combined[j].CreatedAt)
		}
		return combined[i].OccurredAt.After(combined[j].OccurredAt)
	})
	if len(combined) > receiptPaymentTimelineLimit {
		combined = combined[:receiptPaymentTimelineLimit]
	}

	merged := make([]TrackingReceiptPaymentRecord, 0, len(combined))
	for _, item := range combined {
		merged = append(merged, item.Record)
	}
	return merged
}

func enrichTrackingReceiptReversals(reversals []TrackingReceiptReversalRecord, sessionTimeline []trackingReceiptPaymentRecordSnapshot, fallbackCurrency string) []TrackingReceiptReversalRecord {
	if len(reversals) == 0 {
		return []TrackingReceiptReversalRecord{}
	}

	timelineByReference := make(map[string]TrackingReceiptPaymentRecord, len(sessionTimeline))
	for _, item := range sessionTimeline {
		if item.Record.EntryType != "CHARGEBACK_REVERSAL_RECORDED" {
			continue
		}
		referenceID := strings.TrimSpace(item.Record.ReferenceID)
		if referenceID == "" {
			continue
		}
		timelineByReference[referenceID] = item.Record
	}

	enriched := make([]TrackingReceiptReversalRecord, 0, len(reversals))
	for _, reversal := range reversals {
		updated := reversal
		if matched, found := timelineByReference[strings.TrimSpace(reversal.ReversalID)]; found {
			updated.SessionID = matched.SessionID
			updated.Gateway = matched.Gateway
			updated.AmountMinor = matched.AmountMinor
			updated.Currency = matched.Currency
			updated.LedgerEntryID = matched.LedgerEntryID
		}
		if strings.TrimSpace(updated.Gateway) == "" {
			updated.Gateway = "UNKNOWN"
		}
		if strings.TrimSpace(updated.Currency) == "" {
			updated.Currency = strings.TrimSpace(fallbackCurrency)
		}
		enriched = append(enriched, updated)
	}

	return enriched
}

func newTrackingReceiptDossier(sessionID string) TrackingReceiptDossier {
	return TrackingReceiptDossier{
		SessionID:       strings.TrimSpace(sessionID),
		PaymentTimeline: []TrackingReceiptPaymentRecord{},
		GatewayWebhooks: []TrackingReceiptGatewayWebhook{},
		DeliveryProofs:  []TrackingReceiptDeliveryProof{},
		Chargebacks:     []TrackingReceiptChargebackRecord{},
		Reversals:       []TrackingReceiptReversalRecord{},
		ProofStatus: TrackingReceiptProofStatus{
			DeliveryProofAvailable: false,
			MissingArtifacts:       []string{trackingMissingDeliveryHandoffProof},
		},
	}
}

func paymentEvidenceFromReceiptRecord(record TrackingReceiptPaymentRecord) TrackingPaymentEvidence {
	return TrackingPaymentEvidence{
		EntryType:   record.EntryType,
		Gateway:     record.Gateway,
		AmountMinor: record.AmountMinor,
		Currency:    record.Currency,
		ReferenceID: record.ReferenceID,
		OccurredAt:  record.OccurredAt,
	}
}

func uniqueTrackingOrderIDs(orders []TrackingOrder) []string {
	if len(orders) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(orders))
	ids := make([]string, 0, len(orders))
	for _, order := range orders {
		orderID := strings.TrimSpace(order.OrderID)
		if orderID == "" {
			continue
		}
		if _, exists := seen[orderID]; exists {
			continue
		}
		seen[orderID] = struct{}{}
		ids = append(ids, orderID)
	}
	return ids
}

func uniqueTrackingSessionIDs(sessionIDsByOrder map[string]string) []string {
	if len(sessionIDsByOrder) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(sessionIDsByOrder))
	ids := make([]string, 0, len(sessionIDsByOrder))
	for _, sessionID := range sessionIDsByOrder {
		trimmed := strings.TrimSpace(sessionID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		ids = append(ids, trimmed)
	}
	sort.Strings(ids)

	return ids
}

func (r *SpannerRepository) queryTrackingOrders(ctx context.Context, stmt spanner.Statement, op string) ([]TrackingOrder, error) {

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	orders := make([]TrackingOrder, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return orders, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		current, err := decodeTrackingOrder(row)
		if err != nil {
			return nil, err
		}
		orders = append(orders, current)
	}
}

func decodeTrackingOrder(row *spanner.Row) (TrackingOrder, error) {
	var (
		order               TrackingOrder
		storedDeliveryToken string
		lineItems           []byte
		createdAt           time.Time
		updatedAt           time.Time
		lat                 spanner.NullFloat64
		lng                 spanner.NullFloat64
	)
	if err := row.Columns(
		&order.OrderID,
		&order.SupplierID,
		&order.RetailerID,
		&order.WarehouseID,
		&order.DriverID,
		&order.VehicleID,
		&order.RouteID,
		&order.ManifestID,
		&storedDeliveryToken,
		&order.Status,
		&lineItems,
		&order.TotalMinor,
		&order.Currency,
		&createdAt,
		&updatedAt,
		&lat,
		&lng,
	); err != nil {
		return TrackingOrder{}, fmt.Errorf("scan retailer tracking order: %w", err)
	}

	order.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	order.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	order.TrackingStatus = "unassigned"
	if strings.TrimSpace(order.DriverID) != "" && strings.TrimSpace(order.RouteID) != "" {
		order.TrackingStatus = "assigned"
	}
	order.LiveLocationAvailable = false
	order.Items = decodeTrackingLineItems(lineItems)
	if lat.Valid {
		order.DeliveryLat = lat.Float64
	}
	if lng.Valid {
		order.DeliveryLng = lng.Float64
	}
	order.DeliveryToken = trackingDeliveryToken(order.OrderID, storedDeliveryToken, order.Status)
	order.PaymentStatus = trackingPaymentStatus(order.Status)
	return order, nil
}

func trackingDeliveryToken(orderID, storedToken, status string) string {
	return handoff.FromEnv().PublicToken(orderID, storedToken, status)
}

func trackingPaymentStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION":
		return "pending"
	case "COMPLETED":
		return "paid"
	default:
		return ""
	}
}

func decodeTrackingLineItems(raw []byte) []TrackingLineItem {
	if len(raw) == 0 {
		return []TrackingLineItem{}
	}
	var source []orderLineItemJSON
	if err := json.Unmarshal(raw, &source); err != nil {
		return []TrackingLineItem{}
	}
	items := make([]TrackingLineItem, 0, len(source))
	for _, item := range source {
		items = append(items, TrackingLineItem{
			ProductID:   item.SKU,
			ProductName: item.Name,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.UnitPrice * item.Quantity,
		})
	}
	return items
}

// GetSupplierPricingRule returns the effective supplier pricing authority row
// for retailer display and checkout surfaces.
func (r *SpannerRepository) GetSupplierPricingRule(ctx context.Context, supplierID string) (SupplierPricingRule, bool, error) {
	if r == nil || r.client == nil {
		return SupplierPricingRule{}, false, fmt.Errorf("spanner retailer repository: nil client")
	}

	effectiveSupplierID := strings.TrimSpace(supplierID)
	if effectiveSupplierID == "" {
		effectiveSupplierID = r.supplierID
	}
	if effectiveSupplierID == "" {
		return SupplierPricingRule{}, false, nil
	}
	if r.supplierID != "" && effectiveSupplierID != r.supplierID {
		return SupplierPricingRule{}, false, nil
	}

	row, err := r.client.Single().ReadRow(ctx, "SupplierPricingRules", spanner.Key{effectiveSupplierID}, []string{
		"SupplierId",
		"BaseMarkupBps",
		"RetailerDiscountBps",
		"MinMarginBps",
		"Currency",
		"RuleVersion",
		"UpdatedBy",
		"UpdatedAt",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return SupplierPricingRule{}, false, nil
		}
		return SupplierPricingRule{}, false, fmt.Errorf("read supplier pricing rule %s: %w", effectiveSupplierID, err)
	}

	var (
		rule      SupplierPricingRule
		updatedBy spanner.NullString
	)
	if err := row.Columns(
		&rule.SupplierID,
		&rule.BaseMarkupBps,
		&rule.RetailerDiscountBps,
		&rule.MinMarginBps,
		&rule.Currency,
		&rule.RuleVersion,
		&updatedBy,
		&rule.UpdatedAt,
	); err != nil {
		return SupplierPricingRule{}, false, fmt.Errorf("scan supplier pricing rule %s: %w", effectiveSupplierID, err)
	}
	if updatedBy.Valid {
		rule.UpdatedBy = updatedBy.StringVal
	}

	return rule, true, nil
}
