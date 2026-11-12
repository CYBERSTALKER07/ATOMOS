package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
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

func outboxEventMutation(e outbox.Event) *spanner.Mutation {
	return spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e))
}

const orderSelectColumns = `OrderId, SupplierId, RetailerId, WarehouseId, DriverId, VehicleId, RouteId, ManifestId, DeliveryToken, Status, OrderSource, ConfirmationStatus, LineItemsJson, TotalMinor, OriginalTotalMinor, Currency, H3Cell, Lat, Lng, RequestedDeliveryDate, DeliverBefore, DeliveryPriority, DeliveryFeeMinor, WarehouseNotes, AutoConfirmAt, DecisionAt, DecisionBy, DerivedFromOrderId, ReceivingWindowOpen, ReceivingWindowClose, Timezone, PreorderReminderSentAt, NudgeNotifiedAt, ConfirmationNotifiedAt, CancelLockedAt, CancelLockReason, CancelLockExpiresAt, ProposedDeliveryDate, DeliveryProposalAt, DeliveryProposalBy, DeliveryProposalReason, Version, CreatedAt, UpdatedAt, FiscalStatus, LatestFiscalReceiptId, LatestFiscalAttemptId, FiscalizedAt, ShopClosedAt, ShopClosedReason, ShopClosedGraceEndsAt, ShopClosedResolution, PartialDelivery, ProximityUnlockedAt, ProximityMethod, BuyerAcceptanceStatus, BuyerAcceptanceDeadline, ClaimWindowHours, ClaimWindowEndsAt, ClaimWindowPolicySource`

// CreateOrder writes the Orders row and any emitted outbox events atomically.
func (r *SpannerRepository) CreateOrder(ctx context.Context, o *Order, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error {
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

	err = spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := snapshotReceivingWindowsInTxn(ctx, txn, o); err != nil {
			return err
		}
		if err := snapshotWarehousePolicyInTxn(ctx, txn, o); err != nil {
			return err
		}

		// Reserve on-hand stock for fulfillable quantities (all non-backorder orders, including scheduled pre-orders).
		needReservationMarker := false
		if !stockOpts.Skip && len(o.LineItems) > 0 && o.WarehouseID != "" && o.Source != OrderSourceBackorder {
			expected := time.Time{}
			if o.RequestedDeliveryDate != nil {
				expected = o.RequestedDeliveryDate.UTC()
			} else if o.DeliverBefore != nil {
				expected = o.DeliverBefore.UTC()
			}
			if err := ReserveLineItemsForOrderInTxn(ctx, txn, o.SupplierID, o.WarehouseID, o.OrderID, o.RetailerID, expected, o.LineItems); err != nil {
				return err
			}
			needReservationMarker = true
		}

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		var mutations []*spanner.Mutation

		orderInsert := map[string]any{
			"OrderId":                o.OrderID,
			"SupplierId":             o.SupplierID,
			"RetailerId":             o.RetailerID,
			"WarehouseId":            o.WarehouseID,
			"DriverId":               nullableString(o.DriverID),
			"VehicleId":              nullableString(o.VehicleID),
			"RouteId":                nullableString(o.RouteID),
			"ManifestId":             nullableString(o.ManifestID),
			"DeliveryToken":          nullableString(o.QRToken),
			"Status":                 string(o.Status),
			"OrderSource":            string(o.Source),
			"ConfirmationStatus":     string(o.ConfirmationStatus),
			"LineItemsJson":          lineItemsRaw,
			"TotalMinor":             o.TotalMinor,
			"OriginalTotalMinor":     originalTotalMinorForInsert(o),
			"Currency":               o.Currency,
			"H3Cell":                 o.H3Cell,
			"Lat":                    o.Lat,
			"Lng":                    o.Lng,
			"RequestedDeliveryDate":  nullableTime(o.RequestedDeliveryDate),
			"DeliverBefore":          nullableTime(o.DeliverBefore),
			"DeliveryPriority":       string(o.DeliveryPriority),
			"DeliveryFeeMinor":       o.DeliveryFeeMinor,
			"WarehouseNotes":         nullableString(o.WarehouseNotes),
			"AutoConfirmAt":          nullableTime(o.AutoConfirmAt),
			"DecisionAt":             nullableTime(o.DecisionAt),
			"DecisionBy":             nullableString(o.DecisionBy),
			"DerivedFromOrderId":     nullableString(o.DerivedFromOrderID),
			"ReceivingWindowOpen":    nullableString(o.ReceivingWindowOpen),
			"ReceivingWindowClose":   nullableString(o.ReceivingWindowClose),
			"Timezone":               nullableString(o.Timezone),
			"PreorderReminderSentAt": nullableTime(o.PreorderReminderSentAt),
			"NudgeNotifiedAt":        nullableTime(o.NudgeNotifiedAt),
			"ConfirmationNotifiedAt": nullableTime(o.ConfirmationNotifiedAt),
			"CancelLockedAt":         nullableTime(o.CancelLockedAt),
			"CancelLockReason":       nullableString(o.CancelLockReason),
			"CancelLockExpiresAt":    nullableTime(o.CancelLockExpiresAt),
			"ProposedDeliveryDate":   nullableTime(o.ProposedDeliveryDate),
			"DeliveryProposalAt":     nullableTime(o.DeliveryProposalAt),
			"DeliveryProposalBy":     nullableString(o.DeliveryProposalBy),
			"DeliveryProposalReason": nullableString(o.DeliveryProposalReason),
			"Version":                o.Version,
			"CreatedAt":              o.CreatedAt.UTC(),
			"UpdatedAt":              o.UpdatedAt.UTC(),
		}
		if pid := strings.TrimSpace(o.ParentOrderID); pid != "" {
			orderInsert["ParentOrderId"] = pid
		}
		// In Cloud Spanner, parent row Orders must precede interleaved child tables
		// (OrderStockReservationMarkers, OrderServicePromiseSnapshots) in the mutation buffer.
		mutations = append(mutations, spanner.InsertMap("Orders", orderInsert))

		if needReservationMarker {
			if m := stockReservationMarkerMutation(o.OrderID); m != nil {
				mutations = append(mutations, m)
			}
		}

		if promiseMut, err := snapshotServicePromiseInTxn(ctx, txn, o); err == nil && promiseMut != nil {
			mutations = append(mutations, promiseMut)
		}

		// Deduplicate and atomically redeem promotions.
		promoMap := make(map[string]bool)
		for _, line := range o.LineItems {
			if line.PromotionID != "" && !promoMap[line.PromotionID] {
				promoMap[line.PromotionID] = true
				row, readErr := txn.ReadRow(ctx, "SupplierPromotions", spanner.Key{line.PromotionID}, []string{"MaxRedemptions", "CurrentRedemptions"})
				if readErr == nil {
					var maxR, currR spanner.NullInt64
					if err := row.Columns(&maxR, &currR); err == nil {
						if maxR.Valid && maxR.Int64 > 0 && currR.Int64 >= maxR.Int64 {
							return fmt.Errorf("promotion redemption limit reached for %s", line.PromotionID)
						}
						mutations = append(mutations, spanner.UpdateMap("SupplierPromotions", map[string]any{
							"PromotionId":        line.PromotionID,
							"CurrentRedemptions": currR.Int64 + 1,
							"UpdatedAt":          spanner.CommitTimestamp,
						}))
					}
				}
			}
		}

		for _, e := range buf.events {
			mutations = append(mutations, outboxEventMutation(e))
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

	// The Spanner client re-invokes this closure after an aborted commit, so the
	// CAS must compare against the caller's version captured once — not o.Version,
	// which a prior invocation of this same closure may already have incremented.
	callerVersion := o.Version
	err = spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{o.OrderID}, []string{
			"Version", "Status", "OrderSource", "LineItemsJson", "SupplierId", "WarehouseId",
		})
		if err != nil {
			return err
		}
		var (
			version     int64
			prevStatus  string
			orderSource string
			prevLineRaw []byte
			supplierID  string
			warehouseID string
		)
		if err := row.Columns(&version, &prevStatus, &orderSource, &prevLineRaw, &supplierID, &warehouseID); err != nil {
			return err
		}
		if version != callerVersion {
			return fmt.Errorf("optimistic concurrency conflict: expected %d, got %d", callerVersion, version)
		}

		if o.Status == StatusCancelled && !strings.EqualFold(strings.TrimSpace(prevStatus), string(StatusCancelled)) {
			var prevLineItems []LineItem
			if len(prevLineRaw) > 0 {
				if err := json.Unmarshal(prevLineRaw, &prevLineItems); err != nil {
					return fmt.Errorf("parse line items for release %s: %w", o.OrderID, err)
				}
			}
			wh := strings.TrimSpace(o.WarehouseID)
			if wh == "" {
				wh = strings.TrimSpace(warehouseID)
			}
			sid := strings.TrimSpace(o.SupplierID)
			if sid == "" {
				sid = strings.TrimSpace(supplierID)
			}
			if err := ReleaseReservationsForOrderInTxn(ctx, txn, sid, wh, o.OrderID, OrderSource(orderSource), prevLineItems); err != nil {
				return fmt.Errorf("release reservations in txn %s: %w", o.OrderID, err)
			}
		}

		// Derive from the version read in this transaction, not the caller struct:
		// Spanner may abort and re-run this closure, and `o.Version++` would then
		// compare an already-incremented copy against the still-old row.
		o.Version = version + 1
		o.UpdatedAt = time.Now().UTC()

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		orderMap := map[string]any{
			"OrderId":                o.OrderID,
			"WarehouseId":            o.WarehouseID,
			"DriverId":               nullableString(o.DriverID),
			"VehicleId":              nullableString(o.VehicleID),
			"RouteId":                nullableString(o.RouteID),
			"ManifestId":             nullableString(o.ManifestID),
			"DeliveryToken":          nullableString(o.QRToken),
			"Status":                 string(o.Status),
			"OrderSource":            string(o.Source),
			"ConfirmationStatus":     string(o.ConfirmationStatus),
			"LineItemsJson":          lineItemsRaw,
			"TotalMinor":             o.TotalMinor,
			"OriginalTotalMinor":     originalTotalMinorForUpdate(o),
			"Currency":               o.Currency,
			"H3Cell":                 o.H3Cell,
			"Lat":                    o.Lat,
			"Lng":                    o.Lng,
			"RequestedDeliveryDate":  nullableTime(o.RequestedDeliveryDate),
			"DeliverBefore":          nullableTime(o.DeliverBefore),
			"DeliveryPriority":       string(o.DeliveryPriority),
			"DeliveryFeeMinor":       o.DeliveryFeeMinor,
			"WarehouseNotes":         nullableString(o.WarehouseNotes),
			"AutoConfirmAt":          nullableTime(o.AutoConfirmAt),
			"DecisionAt":             nullableTime(o.DecisionAt),
			"DecisionBy":             nullableString(o.DecisionBy),
			"DerivedFromOrderId":     nullableString(o.DerivedFromOrderID),
			"PreorderReminderSentAt": nullableTime(o.PreorderReminderSentAt),
			"NudgeNotifiedAt":        nullableTime(o.NudgeNotifiedAt),
			"ConfirmationNotifiedAt": nullableTime(o.ConfirmationNotifiedAt),
			"CancelLockedAt":         nullableTime(o.CancelLockedAt),
			"CancelLockReason":       nullableString(o.CancelLockReason),
			"CancelLockExpiresAt":    nullableTime(o.CancelLockExpiresAt),
			"ProposedDeliveryDate":   nullableTime(o.ProposedDeliveryDate),
			"DeliveryProposalAt":     nullableTime(o.DeliveryProposalAt),
			"DeliveryProposalBy":     nullableString(o.DeliveryProposalBy),
			"DeliveryProposalReason": nullableString(o.DeliveryProposalReason),
			"Version":                o.Version,
			"UpdatedAt":              o.UpdatedAt,
		}
		// ADR-009 denorm fiscal rollup (columns additive; tolerate missing until migrate).
		if strings.TrimSpace(o.FiscalStatus) != "" {
			orderMap["FiscalStatus"] = o.FiscalStatus
		}
		if strings.TrimSpace(o.LatestFiscalReceiptID) != "" {
			orderMap["LatestFiscalReceiptId"] = o.LatestFiscalReceiptID
		}
		if strings.TrimSpace(o.LatestFiscalAttemptID) != "" {
			orderMap["LatestFiscalAttemptId"] = o.LatestFiscalAttemptID
		}
		if o.FiscalizedAt != nil {
			orderMap["FiscalizedAt"] = *o.FiscalizedAt
		}
		if strings.TrimSpace(o.BuyerAcceptanceStatus) != "" {
			orderMap["BuyerAcceptanceStatus"] = o.BuyerAcceptanceStatus
		}
		if o.BuyerAcceptanceDeadline != nil {
			orderMap["BuyerAcceptanceDeadline"] = *o.BuyerAcceptanceDeadline
		}
		if o.ClaimWindowHours > 0 {
			orderMap["ClaimWindowHours"] = o.ClaimWindowHours
		}
		if o.ClaimWindowEndsAt != nil {
			orderMap["ClaimWindowEndsAt"] = *o.ClaimWindowEndsAt
		}
		if strings.TrimSpace(o.ClaimWindowPolicySource) != "" {
			orderMap["ClaimWindowPolicySource"] = o.ClaimWindowPolicySource
		}
		// Shop-closed / proximity / partial (2026-07-29 additive columns).
		if strings.TrimSpace(o.ShopClosedResolution) != "" {
			orderMap["ShopClosedResolution"] = o.ShopClosedResolution
		}
		if o.ShopClosedAt != nil {
			orderMap["ShopClosedAt"] = *o.ShopClosedAt
		}
		if strings.TrimSpace(o.ShopClosedReason) != "" {
			orderMap["ShopClosedReason"] = o.ShopClosedReason
		}
		if o.ShopClosedGraceEndsAt != nil {
			orderMap["ShopClosedGraceEndsAt"] = *o.ShopClosedGraceEndsAt
		}
		if o.PartialDelivery {
			orderMap["PartialDelivery"] = true
		}
		if o.ProximityUnlockedAt != nil {
			orderMap["ProximityUnlockedAt"] = *o.ProximityUnlockedAt
		}
		if strings.TrimSpace(o.ProximityMethod) != "" {
			orderMap["ProximityMethod"] = o.ProximityMethod
		}
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", orderMap),
		}

		if string(o.Status) != prevStatus {
			metaRaw, _ := json.Marshal(o.TransitionMetadata)
			eventKind := strings.TrimSpace(o.TransitionEventKind)
			if eventKind == "" {
				eventKind = DefaultTransitionEventKind(o.Status)
			}
			transitionID := uuid.NewString()
			mutations = append(mutations, spanner.InsertMap("OrderStatusTransitions", map[string]any{
				"OrderId":        o.OrderID,
				"TransitionId":   transitionID,
				"PreviousStatus": prevStatus,
				"NewStatus":      string(o.Status),
				"Reason":         strings.TrimSpace(o.TransitionReason),
				"ActorRole":      strings.TrimSpace(o.TransitionActorRole),
				"ActorId":        strings.TrimSpace(o.TransitionActorID),
				"EventKind":      eventKind,
				"MetadataJson":   metaRaw,
				"CreatedAt":      o.UpdatedAt,
			}))
		}

		for _, fr := range o.PendingFiscalReceipts {
			if strings.TrimSpace(fr.AttemptID) == "" || strings.TrimSpace(fr.OrderID) == "" {
				continue
			}
			createdAt := fr.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			updatedAt := fr.UpdatedAt.UTC()
			if updatedAt.IsZero() {
				updatedAt = createdAt
			}
			mutations = append(mutations, spanner.InsertMap("OrderFiscalReceipts", map[string]any{
				"OrderId":             fr.OrderID,
				"AttemptId":           fr.AttemptID,
				"SupplierId":          fr.SupplierID,
				"RetailerId":          nullableString(fr.RetailerID),
				"Provider":            fr.Provider,
				"Status":              fr.Status,
				"FiscalReceiptId":     nullableString(fr.FiscalReceiptID),
				"FiscalQR":            nullableString(fr.FiscalQR),
				"AmountMinor":         fr.AmountMinor,
				"Currency":            fr.Currency,
				"PaymentMethod":       nullableString(fr.PaymentMethod),
				"ProviderPayloadJSON": fr.ProviderPayload,
				"ErrorCode":           nullableString(fr.ErrorCode),
				"ErrorMessage":        nullableString(fr.ErrorMessage),
				"ReasonCode":          nullableString(fr.ReasonCode),
				"ActorId":             nullableString(fr.ActorID),
				"TraceId":             nullableString(fr.TraceID),
				"CreatedAt":           createdAt,
				"UpdatedAt":           updatedAt,
			}))
		}
		if o.FiscalReceiptUpdate != nil {
			fr := *o.FiscalReceiptUpdate
			updatedAt := fr.UpdatedAt.UTC()
			if updatedAt.IsZero() {
				updatedAt = time.Now().UTC()
			}
			mutations = append(mutations, spanner.UpdateMap("OrderFiscalReceipts", map[string]any{
				"OrderId":             fr.OrderID,
				"AttemptId":           fr.AttemptID,
				"Status":              fr.Status,
				"FiscalReceiptId":     nullableString(fr.FiscalReceiptID),
				"FiscalQR":            nullableString(fr.FiscalQR),
				"ProviderPayloadJSON": fr.ProviderPayload,
				"ErrorCode":           nullableString(fr.ErrorCode),
				"ErrorMessage":        nullableString(fr.ErrorMessage),
				"ReasonCode":          nullableString(fr.ReasonCode),
				"ActorId":             nullableString(fr.ActorID),
				"UpdatedAt":           updatedAt,
			}))
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

		if o.PendingExceptionTicket != nil {
			t := o.PendingExceptionTicket
			createdAt := t.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			mutations = append(mutations, spanner.InsertMap("ExceptionTickets", map[string]any{
				"TicketId":     t.TicketID,
				"Type":         t.Type,
				"OrderId":      t.OrderID,
				"EhfId":        nullableString(t.EhfID),
				"Severity":     t.Severity,
				"Status":       t.Status,
				"Title":        t.Title,
				"Description":  nullableString(t.Description),
				"AssignedRole": nullableString(t.AssignedRole),
				"CreatedAt":    createdAt,
				"CreatedBy":    nullableString(t.CreatedBy),
				"Payload":      spanner.NullJSON{Value: t.Payload, Valid: t.Payload != nil},
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

		for _, cr := range o.ConditionReports {
			if strings.TrimSpace(cr.ReportID) == "" {
				continue
			}
			photoURLs, err := json.Marshal(cr.PhotoURLs)
			if err != nil {
				return fmt.Errorf("marshal condition report photo urls: %w", err)
			}
			proofIDs, err := json.Marshal(cr.ProofIDs)
			if err != nil {
				return fmt.Errorf("marshal condition report proof ids: %w", err)
			}
			mutations = append(mutations, spanner.InsertMap("OrderConditionReports", map[string]any{
				"ReportId":         cr.ReportID,
				"OrderId":          cr.OrderID,
				"SupplierId":       cr.SupplierID,
				"RetailerId":       cr.RetailerID,
				"LineItemIndex":    nullableInt64(cr.LineItemIndex),
				"SKU":              nullableString(cr.SKU),
				"ConditionType":    string(cr.ConditionType),
				"Severity":         string(cr.Severity),
				"Description":      nullableString(cr.Description),
				"PhotoURLsJson":    photoURLs,
				"ProofIdsJson":     proofIDs,
				"ReportedBy":       cr.ReportedBy,
				"ReportedByRole":   cr.ReportedByRole,
				"ResolutionStatus": string(cr.ResolutionStatus),
				"ResolvedBy":       nullableString(cr.ResolvedBy),
				"ResolvedAt":       nullableTime(cr.ResolvedAt),
				"ResolutionNotes":  nullableString(cr.ResolutionNotes),
				"CreatedAt":        cr.CreatedAt.UTC(),
			}))
		}

		for _, e := range buf.events {
			mutations = append(mutations, outboxEventMutation(e))
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

// GetOrderTxn fetches one order aggregate by id within a transaction.
func (r *SpannerRepository) GetOrderTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (Order, bool, error) {
	if r == nil || r.client == nil {
		return Order{}, false, fmt.Errorf("spanner order repository: nil client")
	}

	row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, strings.Split(orderSelectColumns, ", "))
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return Order{}, false, nil
		}
		return Order{}, false, fmt.Errorf("read order txn %s: %w", orderID, err)
	}

	o, err := scanOrderRowRow(row)
	if err != nil {
		return Order{}, false, fmt.Errorf("scan order txn %s: %w", orderID, err)
	}
	return o, true, nil
}

// GetFiscalAttempt loads one OrderFiscalReceipts row for worker idempotency (ADR-009).
func (r *SpannerRepository) GetFiscalAttempt(ctx context.Context, orderID, attemptID string) (FiscalReceiptRow, bool, error) {
	if r == nil || r.client == nil {
		return FiscalReceiptRow{}, false, fmt.Errorf("spanner order repository: nil client")
	}
	orderID = strings.TrimSpace(orderID)
	attemptID = strings.TrimSpace(attemptID)
	if orderID == "" || attemptID == "" {
		return FiscalReceiptRow{}, false, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "OrderFiscalReceipts", spanner.Key{orderID, attemptID}, []string{
		"OrderId", "AttemptId", "SupplierId", "RetailerId", "Provider", "Status",
		"FiscalReceiptId", "FiscalQR", "AmountMinor", "Currency", "PaymentMethod",
		"ErrorCode", "ErrorMessage", "ReasonCode", "ActorId", "TraceId", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return FiscalReceiptRow{}, false, nil
		}
		// Table may not exist pre-migration.
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "OrderFiscalReceipts") {
			return FiscalReceiptRow{}, false, nil
		}
		return FiscalReceiptRow{}, false, fmt.Errorf("read fiscal attempt %s/%s: %w", orderID, attemptID, err)
	}
	var fr FiscalReceiptRow
	var retailer, fiscalID, fiscalQR, payMethod, errCode, errMsg, reason, actor, trace spanner.NullString
	var created, updated time.Time
	if err := row.Columns(
		&fr.OrderID, &fr.AttemptID, &fr.SupplierID, &retailer, &fr.Provider, &fr.Status,
		&fiscalID, &fiscalQR, &fr.AmountMinor, &fr.Currency, &payMethod,
		&errCode, &errMsg, &reason, &actor, &trace, &created, &updated,
	); err != nil {
		return FiscalReceiptRow{}, false, fmt.Errorf("scan fiscal attempt: %w", err)
	}
	if retailer.Valid {
		fr.RetailerID = retailer.StringVal
	}
	if fiscalID.Valid {
		fr.FiscalReceiptID = fiscalID.StringVal
	}
	if fiscalQR.Valid {
		fr.FiscalQR = fiscalQR.StringVal
	}
	if payMethod.Valid {
		fr.PaymentMethod = payMethod.StringVal
	}
	if errCode.Valid {
		fr.ErrorCode = errCode.StringVal
	}
	if errMsg.Valid {
		fr.ErrorMessage = errMsg.StringVal
	}
	if reason.Valid {
		fr.ReasonCode = reason.StringVal
	}
	if actor.Valid {
		fr.ActorID = actor.StringVal
	}
	if trace.Valid {
		fr.TraceID = trace.StringVal
	}
	fr.CreatedAt = created.UTC()
	fr.UpdatedAt = updated.UTC()
	return fr, true, nil
}

// GetFiscalByReceiptID loads a fiscal attempt by public receipt id (Idx_OrderFiscalReceipts_ByReceiptId).
func (r *SpannerRepository) GetFiscalByReceiptID(ctx context.Context, receiptID string) (FiscalReceiptRow, bool, error) {
	if r == nil || r.client == nil {
		return FiscalReceiptRow{}, false, fmt.Errorf("spanner order repository: nil client")
	}
	receiptID = strings.TrimSpace(receiptID)
	if receiptID == "" {
		return FiscalReceiptRow{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, AttemptId, SupplierId, RetailerId, Provider, Status,
			FiscalReceiptId, FiscalQR, AmountMinor, Currency, PaymentMethod,
			ErrorCode, ErrorMessage, ReasonCode, ActorId, TraceId, CreatedAt, UpdatedAt
			FROM OrderFiscalReceipts@{FORCE_INDEX=Idx_OrderFiscalReceipts_ByReceiptId}
			WHERE FiscalReceiptId = @rid
			LIMIT 1`,
		Params: map[string]any{"rid": receiptID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done || errors.Is(err, spanner.ErrRowNotFound) {
			return FiscalReceiptRow{}, false, nil
		}
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "OrderFiscalReceipts") {
			return FiscalReceiptRow{}, false, nil
		}
		return FiscalReceiptRow{}, false, fmt.Errorf("read fiscal by receipt_id %s: %w", receiptID, err)
	}
	var fr FiscalReceiptRow
	var retailer, fiscalID, fiscalQR, payMethod, errCode, errMsg, reason, actor, trace spanner.NullString
	var created, updated time.Time
	if err := row.Columns(
		&fr.OrderID, &fr.AttemptID, &fr.SupplierID, &retailer, &fr.Provider, &fr.Status,
		&fiscalID, &fiscalQR, &fr.AmountMinor, &fr.Currency, &payMethod,
		&errCode, &errMsg, &reason, &actor, &trace, &created, &updated,
	); err != nil {
		return FiscalReceiptRow{}, false, fmt.Errorf("scan fiscal by receipt_id: %w", err)
	}
	if retailer.Valid {
		fr.RetailerID = retailer.StringVal
	}
	if fiscalID.Valid {
		fr.FiscalReceiptID = fiscalID.StringVal
	}
	if fiscalQR.Valid {
		fr.FiscalQR = fiscalQR.StringVal
	}
	if payMethod.Valid {
		fr.PaymentMethod = payMethod.StringVal
	}
	if errCode.Valid {
		fr.ErrorCode = errCode.StringVal
	}
	if errMsg.Valid {
		fr.ErrorMessage = errMsg.StringVal
	}
	if reason.Valid {
		fr.ReasonCode = reason.StringVal
	}
	if actor.Valid {
		fr.ActorID = actor.StringVal
	}
	if trace.Valid {
		fr.TraceID = trace.StringVal
	}
	fr.CreatedAt = created.UTC()
	fr.UpdatedAt = updated.UTC()
	return fr, true, nil
}

// CountFiscalAttemptsByStatus counts OrderFiscalReceipts rows for an order/status.
func (r *SpannerRepository) CountFiscalAttemptsByStatus(ctx context.Context, orderID, status string) (int64, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("spanner order repository: nil client")
	}
	orderID = strings.TrimSpace(orderID)
	status = strings.TrimSpace(status)
	if orderID == "" || status == "" {
		return 0, nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OrderFiscalReceipts WHERE OrderId = @oid AND Status = @st`,
		Params: map[string]any{"oid": orderID, "st": status},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "OrderFiscalReceipts") {
			return 0, nil
		}
		return 0, fmt.Errorf("count fiscal attempts: %w", err)
	}
	var n int64
	if err := row.Columns(&n); err != nil {
		return 0, err
	}
	return n, nil
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
	iter := r.client.Single().WithTimestampBound(spanner.MaxStaleness(15*time.Second)).Query(ctx, stmt)
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
	iter := r.client.Single().WithTimestampBound(spanner.MaxStaleness(15*time.Second)).Query(ctx, stmt)
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
			WHERE SupplierId IS NOT NULL
			AND ConfirmationStatus = @confirmation_status
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
		timezone               spanner.NullString
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
		cancelLockExpiresAt    spanner.NullTime
		proposedDeliveryDate   spanner.NullTime
		deliveryProposalAt     spanner.NullTime
		deliveryProposalBy     spanner.NullString
		deliveryProposalReason spanner.NullString
		deliveryFeeMinor       int64
		orderRecord            Order

		fiscalStatus            spanner.NullString
		latestFiscalReceiptID   spanner.NullString
		latestFiscalAttemptID   spanner.NullString
		fiscalizedAt            spanner.NullTime
		shopClosedAt            spanner.NullTime
		shopClosedReason        spanner.NullString
		shopClosedGraceEndsAt   spanner.NullTime
		shopClosedResolution    spanner.NullString
		partialDelivery         spanner.NullBool
		proximityUnlockedAt     spanner.NullTime
		proximityMethod         spanner.NullString
		buyerAcceptanceStatus   spanner.NullString
		buyerAcceptanceDeadline spanner.NullTime
		claimWindowHours        spanner.NullInt64
		claimWindowEndsAt       spanner.NullTime
		claimWindowPolicySource spanner.NullString
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
		&timezone,
		&preorderReminderSentAt,
		&nudgeNotifiedAt,
		&confirmationNotifiedAt,
		&cancelLockedAt,
		&cancelLockReason,
		&cancelLockExpiresAt,
		&proposedDeliveryDate,
		&deliveryProposalAt,
		&deliveryProposalBy,
		&deliveryProposalReason,
		&orderRecord.Version,
		&orderRecord.CreatedAt,
		&orderRecord.UpdatedAt,
		&fiscalStatus,
		&latestFiscalReceiptID,
		&latestFiscalAttemptID,
		&fiscalizedAt,
		&shopClosedAt,
		&shopClosedReason,
		&shopClosedGraceEndsAt,
		&shopClosedResolution,
		&partialDelivery,
		&proximityUnlockedAt,
		&proximityMethod,
		&buyerAcceptanceStatus,
		&buyerAcceptanceDeadline,
		&claimWindowHours,
		&claimWindowEndsAt,
		&claimWindowPolicySource,
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
	orderRecord.Timezone = timezone.StringVal
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
	if cancelLockExpiresAt.Valid {
		t := cancelLockExpiresAt.Time.UTC()
		orderRecord.CancelLockExpiresAt = &t
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

	orderRecord.FiscalStatus = fiscalStatus.StringVal
	orderRecord.LatestFiscalReceiptID = latestFiscalReceiptID.StringVal
	orderRecord.LatestFiscalAttemptID = latestFiscalAttemptID.StringVal
	if fiscalizedAt.Valid {
		t := fiscalizedAt.Time.UTC()
		orderRecord.FiscalizedAt = &t
	}
	if shopClosedAt.Valid {
		t := shopClosedAt.Time.UTC()
		orderRecord.ShopClosedAt = &t
	}
	orderRecord.ShopClosedReason = shopClosedReason.StringVal
	if shopClosedGraceEndsAt.Valid {
		t := shopClosedGraceEndsAt.Time.UTC()
		orderRecord.ShopClosedGraceEndsAt = &t
	}
	orderRecord.ShopClosedResolution = shopClosedResolution.StringVal
	orderRecord.PartialDelivery = partialDelivery.Valid && partialDelivery.Bool
	if proximityUnlockedAt.Valid {
		t := proximityUnlockedAt.Time.UTC()
		orderRecord.ProximityUnlockedAt = &t
	}
	orderRecord.ProximityMethod = proximityMethod.StringVal
	orderRecord.BuyerAcceptanceStatus = buyerAcceptanceStatus.StringVal
	if buyerAcceptanceDeadline.Valid {
		t := buyerAcceptanceDeadline.Time.UTC()
		orderRecord.BuyerAcceptanceDeadline = &t
	}
	if claimWindowHours.Valid {
		orderRecord.ClaimWindowHours = claimWindowHours.Int64
	}
	if claimWindowEndsAt.Valid {
		t := claimWindowEndsAt.Time.UTC()
		orderRecord.ClaimWindowEndsAt = &t
	}
	orderRecord.ClaimWindowPolicySource = claimWindowPolicySource.StringVal

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

func nullableInt64(value *int64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

// ListManifestOrders returns all orders linked to the specified manifest.
