package kafka

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"backend-go/finance"
	"backend-go/kafka/workerpool"
	"backend-go/outbox"
	"backend-go/settings"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// The exact event shape fired by your delivery endpoint
type LogisticsEvent struct {
	EventName  string    `json:"event_name"`
	OrderId    string    `json:"order_id"`
	RetailerId string    `json:"retailer_id"` // Matches the struct in order/service.go
	Amount     int64     `json:"amount"`
	Timestamp  time.Time `json:"timestamp"`
}

// GenerateTxnId produces a deterministic, content-derived transaction ID.
// Identical (orderID, entryType, amount) always yields the same ID, so the
// UNIQUE index on LedgerEntries(OrderId, EntryType) rejects duplicate inserts
// on DLQ replay instead of silently creating a second ledger row.
func GenerateTxnId(orderID, entryType string, amount int64) string {
	raw := fmt.Sprintf("%s:%s:%d", orderID, entryType, amount)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("TXN-%x", sum[:8])
}

// StartTreasurer begins the partition-parallel financial ledger consumer.
// platformCfg provides the dynamic commission rate from SystemConfig.
// Returns immediately; the worker pool runs until ctx is cancelled.
func StartTreasurer(ctx context.Context, spannerClient *spanner.Client, brokerAddress string, platformCfg *settings.PlatformConfig) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    TopicMain,
		GroupID:  "pegasus-treasurer-group",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	pool, err := workerpool.New(workerpool.Config{
		Source: reader,
		Name:   "treasurer",
		Logger: slog.Default(),
		Handler: func(ctx context.Context, m kafka.Message) error {
			eventType := EventType(m.Headers, m.Key)
			switch eventType {
			case EventOrderCompleted:
				var event LogisticsEvent
				if err := json.Unmarshal(m.Value, &event); err != nil {
					return fmt.Errorf("treasurer: unmarshal %s: %w", EventOrderCompleted, err)
				}
				event.EventName = eventType
				executeLedgerSplit(spannerClient, event, platformCfg)

			case EventOrderModified:
				var modEvt struct {
					OrderID     string `json:"order_id"`
					AmendmentID string `json:"amendment_id"`
					NewAmount   int64  `json:"new_amount"`
					Refunded    int64  `json:"refunded"`
				}
				if err := json.Unmarshal(m.Value, &modEvt); err != nil {
					return fmt.Errorf("treasurer: unmarshal %s: %w", EventOrderModified, err)
				}
				slog.InfoContext(ctx, "treasurer.order_modified",
					"amendment_id", modEvt.AmendmentID, "order_id", modEvt.OrderID,
					"new_amount", modEvt.NewAmount, "refunded", modEvt.Refunded)

			case EventManifestCompleted:
				var event ManifestLifecycleEvent
				if err := json.Unmarshal(m.Value, &event); err != nil {
					return fmt.Errorf("treasurer: unmarshal %s: %w", EventManifestCompleted, err)
				}
				executeManifestSettlement(spannerClient, event, platformCfg)

			case EventOrderCancelledByOrigin:
				var event OrderCancelledByOriginEvent
				if err := json.Unmarshal(m.Value, &event); err != nil {
					return fmt.Errorf("treasurer: unmarshal %s: %w", EventOrderCancelledByOrigin, err)
				}
				voidPendingLedgerEntries(spannerClient, event)
			}
			return nil
		},
		OnFailure: func(ctx context.Context, m kafka.Message, handlerErr error) {
			eventType := EventType(m.Headers, m.Key)
			slog.ErrorContext(ctx, "treasurer.handler_error",
				"event", eventType, "partition", m.Partition, "offset", m.Offset, "err", handlerErr)
			// Re-use RouteToDLQ for financial events so the DLQ retains full
			// event metadata, matching the pre-existing reconciliation behaviour.
			RouteToDLQ(LogisticsEvent{
				EventName: eventType,
				OrderId:   string(m.Key),
				Timestamp: time.Now().UTC(),
			}, handlerErr.Error())
		},
	})
	if err != nil {
		slog.Error("treasurer: pool init failed", "err", err)
		return
	}
	go func() {
		if err := pool.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("treasurer: pool exited", "err", err)
		}
	}()
	slog.Info("treasurer ONLINE", "topic", TopicMain, "group", "pegasus-treasurer-group")
}

// executeLedgerSplit writes PENDING ledger entries atomically with an outbox event.
// For digital payments (non-CASH), LedgerEntries are created with Status=PENDING_GATEWAY
// and a PaymentIntentCreated event is emitted via transactional outbox. The separate
// Gateway Worker consumer handles the actual charge against the provider, preventing
// the gateway-Spanner race condition where a charge succeeds but the txn rolls back.
// CASH orders are written as SETTLED immediately (no gateway round-trip needed).
func executeLedgerSplit(client *spanner.Client, event LogisticsEvent, platformCfg *settings.PlatformConfig) {
	ctx := context.Background()

	// Read order metadata: Amount and PaymentGateway
	var paymentGateway string
	if event.Amount == 0 {
		row, err := client.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderId}, []string{"Amount", "PaymentGateway"})
		if err != nil {
			slog.Error("treasurer.read_order_failed", "order_id", event.OrderId, "err", err)
			return
		}
		var spannerAmount spanner.NullInt64
		var spannerGateway spanner.NullString
		if err := row.Columns(&spannerAmount, &spannerGateway); err != nil {
			slog.Error("treasurer.parse_order_failed", "order_id", event.OrderId, "err", err)
			return
		}
		event.Amount = spannerAmount.Int64
		paymentGateway = spannerGateway.StringVal
	} else {
		// Event has amount but we still need the gateway
		row, err := client.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderId}, []string{"PaymentGateway"})
		if err != nil {
			slog.Warn("treasurer.read_gateway_failed", "order_id", event.OrderId, "err", err, "fallback", "CASH")
			paymentGateway = "CASH"
		} else {
			var spannerGateway spanner.NullString
			if err := row.Columns(&spannerGateway); err != nil {
				slog.Warn("treasurer.parse_gateway_failed", "order_id", event.OrderId, "err", err)
				paymentGateway = "CASH"
			} else {
				paymentGateway = spannerGateway.StringVal
			}
		}
	}

	// Resolve the SupplierId directly from the Orders table (indexed column).
	var supplierID string
	supRow, supErr := client.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderId}, []string{"SupplierId"})
	if supErr != nil {
		slog.Warn("treasurer.supplier_resolution_failed", "order_id", event.OrderId, "err", supErr)
		RouteToDLQ(event, fmt.Sprintf("supplier_resolution_failed: %v", supErr))
		return
	}
	var nullSupplierID spanner.NullString
	if colErr := supRow.Columns(&nullSupplierID); colErr != nil || !nullSupplierID.Valid || nullSupplierID.StringVal == "" {
		slog.Warn("treasurer.supplier_id_null", "order_id", event.OrderId)
		RouteToDLQ(event, "supplier_id_null")
		return
	}
	supplierID = nullSupplierID.StringVal

	// Strict integer math — dynamic commission from SystemConfig.
	commissionRate := platformCfg.PlatformFeePercent()
	platformCommission := (event.Amount * commissionRate) / 100
	supplierPayout := event.Amount - platformCommission

	txnIdA := GenerateTxnId(event.OrderId, finance.PlatformCreditEntryType, platformCommission)
	txnIdB := GenerateTxnId(event.OrderId, "CREDIT_SUPPLIER", supplierPayout)
	now := time.Now()

	// Determine status: CASH settles immediately; digital payments pend gateway capture.
	isDigital := paymentGateway != "" && paymentGateway != "CASH"
	status := "SETTLED"
	if isDigital {
		status = "PENDING_GATEWAY"
	}
	idempotencyKey := fmt.Sprintf("charge_%s", event.OrderId)

	// For Global Pay auth-capture: read the authorization from PaymentSessions.
	var authorizationID string
	var authorizedAmount int64
	if paymentGateway == "GLOBAL_PAY" {
		authStmt := spanner.Statement{
			SQL:    `SELECT AuthorizationId, AuthorizedAmount FROM PaymentSessions WHERE OrderId = @oid AND Status = 'AUTHORIZED' LIMIT 1`,
			Params: map[string]interface{}{"oid": event.OrderId},
		}
		authIter := client.Single().Query(ctx, authStmt)
		authRow, authErr := authIter.Next()
		authIter.Stop()
		if authErr == nil {
			var nullAuthID spanner.NullString
			var nullAuthAmt spanner.NullInt64
			if colErr := authRow.Columns(&nullAuthID, &nullAuthAmt); colErr == nil {
				authorizationID = nullAuthID.StringVal
				authorizedAmount = nullAuthAmt.Int64
			}
		}
	}

	slog.Info("treasurer.processing_order", "order_id", event.OrderId, "total", event.Amount, "platform_commission", platformCommission, "supplier_id", supplierID, "supplier_payout", supplierPayout, "gateway", paymentGateway, "status", status)

	// Auth-Commit-Capture: write PENDING ledger entries + outbox event inside a single RWTxn.
	// The gateway charge happens asynchronously in the Gateway Worker — never inline.
	txCtx, txCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer txCancel()
	_, txErr := client.ReadWriteTransaction(txCtx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {

		// Entry A: Credit Pegasus (Platform Fee)
		m1 := spanner.Insert("LedgerEntries",
			[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "Status", "IdempotencyKey", "CreatedAt"},
			[]interface{}{txnIdA, event.OrderId, finance.PlatformAccountID, platformCommission, finance.PlatformCreditEntryType, status, idempotencyKey, now},
		)

		// Entry B: Credit the Supplier (Supplier Payout)
		m2 := spanner.Insert("LedgerEntries",
			[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "Status", "IdempotencyKey", "CreatedAt"},
			[]interface{}{txnIdB, event.OrderId, supplierID, supplierPayout, "CREDIT_SUPPLIER", status, idempotencyKey, now},
		)

		if err := txn.BufferWrite([]*spanner.Mutation{m1, m2}); err != nil {
			return err
		}

		// For digital payments, emit PaymentIntentCreated so the Gateway Worker can charge.
		if isDigital {
			return outbox.EmitJSON(txn, "Order", event.OrderId, EventPaymentIntentCreated, TopicMain, PaymentIntentEvent{
				OrderID:            event.OrderId,
				SupplierId:         supplierID,
				Amount:             event.Amount,
				Currency:           "UZS",
				PaymentGateway:     paymentGateway,
				IdempotencyKey:     idempotencyKey,
				PlatformCommission: platformCommission,
				LabCommission:      platformCommission,
				SupplierPayout:     supplierPayout,
				PlatformTxnId:      txnIdA,
				LabTxnId:           txnIdA,
				SupplierTxnId:      txnIdB,
				AuthorizationID:    authorizationID,
				AuthorizedAmount:   authorizedAmount,
				FinalAmount:        event.Amount, // Post-amendment amount (driver's tablet is source of truth)
			}, telemetry.TraceIDFromContext(ctx))
		}
		return nil
	})

	if txErr != nil {
		slog.Error("treasurer.ledger_write_failed", "order_id", event.OrderId, "err", txErr)
		RouteToDLQ(event, txErr.Error())
	} else {
		slog.Info("treasurer.ledger_committed", "order_id", event.OrderId, "status", status)
	}
}

// executeManifestSettlement performs manifest-level financial reconciliation.
// Triggered by EventManifestCompleted: verifies every order on the manifest
// has paired ledger entries, flags anomalies for missing entries, auto-advances
// CASH invoices to PENDING_DEPOSIT, and emits EventManifestSettled.
func executeManifestSettlement(client *spanner.Client, event ManifestLifecycleEvent, platformCfg *settings.PlatformConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	manifestID := event.ManifestID
	if manifestID == "" {
		slog.Warn("treasurer.manifest_completed_empty_id")
		return
	}

	// Read all orders on this manifest
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.Amount, o.Currency, o.PaymentGateway, o.SupplierId
		      FROM Orders o
		      WHERE o.ManifestId = @mid AND o.State = 'COMPLETED'`,
		Params: map[string]interface{}{"mid": manifestID},
	}

	type manifestOrder struct {
		OrderID  string
		Amount   int64
		Currency string
		Gateway  string
		Supplier string
	}

	var orders []manifestOrder
	var supplierID string

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("treasurer.manifest_orders_read_failed", "manifest_id", manifestID, "err", err)
			return
		}
		var oid string
		var amount spanner.NullInt64
		var currency, gateway, sid spanner.NullString
		if err := row.Columns(&oid, &amount, &currency, &gateway, &sid); err != nil {
			slog.Error("treasurer.manifest_row_parse_error", "manifest_id", manifestID, "err", err)
			continue
		}
		cur := "UZS"
		if currency.Valid && currency.StringVal != "" {
			cur = currency.StringVal
		}
		if sid.Valid {
			supplierID = sid.StringVal
		}
		orders = append(orders, manifestOrder{
			OrderID:  oid,
			Amount:   amount.Int64,
			Currency: cur,
			Gateway:  gateway.StringVal,
			Supplier: sid.StringVal,
		})
	}

	if len(orders) == 0 {
		slog.Info("treasurer.manifest_no_orders", "manifest_id", manifestID)
		return
	}

	// Reconcile: verify each order has ledger entries
	var totalAmount, cashAmount, digitalAmount int64
	var anomalyCount, settledCount int
	now := time.Now().UTC()
	commissionRate := platformCfg.PlatformFeePercent()

	var anomalyMutations []*spanner.Mutation
	var cashInvoiceMutations []*spanner.Mutation

	for _, o := range orders {
		totalAmount += o.Amount
		if o.Gateway == "" || o.Gateway == "CASH" {
			cashAmount += o.Amount
		} else {
			digitalAmount += o.Amount
		}

		// Check for ledger entry existence
		ledgerStmt := spanner.Statement{
			SQL:    `SELECT COUNT(1) FROM LedgerEntries WHERE OrderId = @oid`,
			Params: map[string]interface{}{"oid": o.OrderID},
		}
		ledgerIter := client.Single().Query(ctx, ledgerStmt)
		ledgerRow, ledgerErr := ledgerIter.Next()
		ledgerIter.Stop()

		var entryCount int64
		if ledgerErr == nil {
			ledgerRow.Columns(&entryCount)
		}

		if entryCount >= 2 {
			settledCount++
			continue
		}

		// Missing ledger entries — create anomaly
		anomalyCount++
		anomalyMutations = append(anomalyMutations, spanner.Insert("LedgerAnomalies",
			[]string{"OrderId", "RetailerId", "SpannerAmount", "GatewayAmount", "Currency", "GatewayProvider", "Status", "DetectedAt"},
			[]interface{}{o.OrderID, "", o.Amount, int64(0), o.Currency, o.Gateway, "DELTA", now},
		))
		slog.Warn("treasurer.ledger_anomaly", "order_id", o.OrderID, "manifest_id", manifestID, "entry_count", entryCount, "expected", 2)
	}

	// Auto-advance CASH invoices to PENDING_DEPOSIT on manifest completion
	cashInvoiceStmt := spanner.Statement{
		SQL: `SELECT InvoiceId FROM MasterInvoices
		      WHERE OrderId IN (SELECT OrderId FROM Orders WHERE ManifestId = @mid AND State = 'COMPLETED')
		        AND PaymentMode = 'CASH'
		        AND CustodyStatus = 'HELD_BY_DRIVER'`,
		Params: map[string]interface{}{"mid": manifestID},
	}
	cashIter := client.Single().Query(ctx, cashInvoiceStmt)
	for {
		cashRow, cashErr := cashIter.Next()
		if cashErr == iterator.Done {
			break
		}
		if cashErr != nil {
			break
		}
		var invoiceID string
		if colErr := cashRow.Columns(&invoiceID); colErr == nil {
			cashInvoiceMutations = append(cashInvoiceMutations, spanner.Update("MasterInvoices",
				[]string{"InvoiceId", "CustodyStatus"},
				[]interface{}{invoiceID, "PENDING_DEPOSIT"},
			))
		}
	}
	cashIter.Stop()

	// Commit anomalies + cash custody advancement + outbox event in one txn
	platformFee := (totalAmount * commissionRate) / 100
	supplierPayout := totalAmount - platformFee

	settlementEvent := ManifestSettlementEvent{
		ManifestID:     manifestID,
		SupplierId:     supplierID,
		TotalOrders:    len(orders),
		SettledOrders:  settledCount,
		TotalAmount:    totalAmount,
		CashAmount:     cashAmount,
		DigitalAmount:  digitalAmount,
		PlatformFee:    platformFee,
		SupplierPayout: supplierPayout,
		AnomalyCount:   anomalyCount,
		Currency:       "UZS",
		Timestamp:      now,
	}

	_, txErr := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation
		mutations = append(mutations, anomalyMutations...)
		mutations = append(mutations, cashInvoiceMutations...)

		if len(mutations) > 0 {
			if err := txn.BufferWrite(mutations); err != nil {
				return fmt.Errorf("settlement mutations: %w", err)
			}
		}

		// Emit MANIFEST_SETTLED via outbox — downstream surfaces (portal, analytics) react to this
		return outbox.EmitJSON(txn, "Manifest", manifestID, EventManifestSettled, TopicMain, settlementEvent, telemetry.TraceIDFromContext(ctx))
	})

	if txErr != nil {
		slog.Error("treasurer.settlement_failed", "manifest_id", manifestID, "err", txErr)
		RouteToDLQ(LogisticsEvent{EventName: EventManifestSettled, OrderId: manifestID, Timestamp: time.Now().UTC()}, fmt.Sprintf("manifest_settlement_failed: %v", txErr))
		return
	}

	slog.Info("treasurer.manifest_settled", "manifest_id", manifestID, "orders", len(orders), "settled", settledCount, "anomalies", anomalyCount, "total", totalAmount, "fee", platformFee, "payout", supplierPayout, "cash_advanced", len(cashInvoiceMutations))
}

// voidPendingLedgerEntries marks all PENDING_GATEWAY ledger entries for a cancelled order as VOIDED.
// Called when ORDER_CANCELLED_BY_ORIGIN fires — hard kill path. No gateway charge is attempted.
func voidPendingLedgerEntries(client *spanner.Client, event OrderCancelledByOriginEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orderID := event.OrderID
	if orderID == "" {
		slog.Warn("treasurer.cancel_empty_order_id")
		return
	}

	// Read existing ledger entries for this order that are still pending.
	stmt := spanner.Statement{
		SQL:    `SELECT TransactionId, Status FROM LedgerEntries WHERE OrderId = @oid AND Status IN ('PENDING_GATEWAY', 'PENDING_DEPOSIT')`,
		Params: map[string]interface{}{"oid": orderID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var txnIDs []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("treasurer.void_read_failed", "order_id", orderID, "err", err)
			return
		}
		var txnID, status string
		if err := row.Columns(&txnID, &status); err != nil {
			slog.Error("treasurer.void_parse_failed", "order_id", orderID, "err", err)
			continue
		}
		txnIDs = append(txnIDs, txnID)
	}

	if len(txnIDs) == 0 {
		slog.Info("treasurer.void_no_pending", "order_id", orderID)
		return
	}

	now := time.Now()
	_, txErr := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation
		for _, txnID := range txnIDs {
			m := spanner.Update("LedgerEntries",
				[]string{"TransactionId", "OrderId", "Status", "FailureReason", "SettledAt"},
				[]interface{}{txnID, orderID, "VOIDED", fmt.Sprintf("cancelled_by_origin: %s", event.Reason), now},
			)
			mutations = append(mutations, m)
		}
		return txn.BufferWrite(mutations)
	})

	if txErr != nil {
		slog.Error("treasurer.void_failed", "order_id", orderID, "entries", len(txnIDs), "err", txErr)
		RouteToDLQ(LogisticsEvent{EventName: EventOrderCancelledByOrigin, OrderId: orderID, Timestamp: time.Now().UTC()},
			fmt.Sprintf("void_ledger_failed: %v", txErr))
		return
	}

	slog.Info("treasurer.void_complete", "order_id", orderID, "entries", len(txnIDs), "reason", event.Reason)
}
