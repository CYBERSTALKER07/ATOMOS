package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"backend-go/cache"
	"backend-go/finance"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// Refund status constants matching the Refunds table CHECK constraint.
const (
	RefundPending        = "PENDING"
	RefundSettled        = "SETTLED"
	RefundFailed         = "FAILED"
	RefundManualRequired = "MANUAL_REQUIRED"
)

// RefundRequest is the input for initiating a supplier-triggered refund.
type RefundRequest struct {
	OrderID   string `json:"order_id"`
	Reason    string `json:"reason"`
	Amount    int64  `json:"amount,omitempty"`
	Currency  string `json:"currency,omitempty"`
	AmountUZS int64  `json:"amount_uzs"` // 0 = full refund (uses session LockedAmount)
}

// RefundResult is the output after a refund attempt.
type RefundResult struct {
	RefundID         string `json:"refund_id"`
	Status           string `json:"status"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	AmountUZS        int64  `json:"amount_uzs"`
	Gateway          string `json:"gateway"`
	ProviderRefundID string `json:"provider_refund_id,omitempty"`
}

// RefundService handles the refund lifecycle: validation, gateway call,
// ledger reversal, and Kafka event emission.
type RefundService struct {
	spanner       *spanner.Client
	feeBP         int64 // Platform fee in basis points (0 = zero-fee era)
	vaultResolver VaultResolver
	execution     *ProviderExecutionRouter
}

// NewRefundService creates a refund service. feeBP is the platform commission
// in basis points (e.g., 500 = 5%). Pass 0 for zero-fee era.
func NewRefundService(sc *spanner.Client, feeBP int64, vaultResolver VaultResolver, execution *ProviderExecutionRouter) *RefundService {
	return &RefundService{spanner: sc, feeBP: feeBP, vaultResolver: vaultResolver, execution: execution}
}

// InitiateRefund validates the order state, calls the gateway, creates the
// Refunds row, writes reversal ledger entries, and emits PAYMENT_REFUNDED.
func (rs *RefundService) InitiateRefund(ctx context.Context, req RefundRequest, initiatedBy string) (*RefundResult, error) {
	// 1. Read order + session in single snapshot
	row, err := rs.spanner.Single().ReadRow(ctx, "Orders",
		spanner.Key{req.OrderID},
		[]string{"OrderId", "SupplierId", "RetailerId", "State", "PaymentGateway", "Amount"})
	if err != nil {
		return nil, fmt.Errorf("read order: %w", err)
	}

	var orderID, supplierID, retailerID, state, gateway string
	var orderAmount int64
	if err := row.Columns(&orderID, &supplierID, &retailerID, &state, &gateway, &orderAmount); err != nil {
		return nil, fmt.Errorf("parse order: %w", err)
	}

	// Only COMPLETED or CANCELLED orders can be refunded
	if state != "COMPLETED" && state != "CANCELLED" {
		return nil, fmt.Errorf("order %s is in state %s — only COMPLETED or CANCELLED orders can be refunded", orderID, state)
	}

	// 2. Find settled payment session for this order
	stmt := spanner.Statement{
		SQL:    `SELECT SessionId, Gateway, LockedAmount, PaidAmount, Currency, ProviderReference FROM PaymentSessions WHERE OrderId = @orderID AND Status = 'SETTLED' LIMIT 1`,
		Params: map[string]interface{}{"orderID": orderID},
	}
	iter := rs.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	sessRow, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("no settled payment session for order %s: %w", orderID, err)
	}

	var sessionID, sessGateway string
	var lockedAmount spanner.NullInt64
	var paidAmount spanner.NullInt64
	var sessionCurrency spanner.NullString
	var providerReference spanner.NullString
	if err := sessRow.Columns(&sessionID, &sessGateway, &lockedAmount, &paidAmount, &sessionCurrency, &providerReference); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	resolvedCurrency := normalizeCurrencyCode("")
	if sessionCurrency.Valid {
		resolvedCurrency = normalizeCurrencyCode(sessionCurrency.StringVal)
	}

	if req.Currency != "" && normalizeCurrencyCode(req.Currency) != resolvedCurrency {
		return nil, fmt.Errorf("refund currency %s does not match settled session currency %s", normalizeCurrencyCode(req.Currency), resolvedCurrency)
	}

	// Determine refund amount
	refundAmount := req.Amount
	if refundAmount <= 0 {
		refundAmount = req.AmountUZS
	}
	if refundAmount <= 0 {
		if paidAmount.Valid && paidAmount.Int64 > 0 {
			refundAmount = paidAmount.Int64
		} else {
			refundAmount = lockedAmount.Int64
		}
	}

	// Guard: refund cannot exceed paid amount
	maxRefundable := lockedAmount.Int64
	if paidAmount.Valid && paidAmount.Int64 > 0 {
		maxRefundable = paidAmount.Int64
	}
	if refundAmount > maxRefundable {
		return nil, fmt.Errorf("refund amount %d exceeds maximum refundable %d", refundAmount, maxRefundable)
	}

	// Snapshot-authoritative fee split for reversal ledger math.
	snapshot, hasSnapshot, snapshotErr := LoadSupplierSettlementSnapshot(ctx, rs.spanner, orderID, supplierID)
	if snapshotErr != nil {
		log.Printf("[REFUND] Settlement snapshot lookup failed for order %s supplier %s: %v", orderID, supplierID, snapshotErr)
	}

	// 3. Call payment gateway for refund
	refundID := uuid.New().String()
	providerRefundID := ""
	refundStatus := RefundPending

	if sessGateway == "CASH" {
		// Cash refunds are manual — mark for manual processing
		refundStatus = RefundManualRequired
		log.Printf("[REFUND] Cash refund for order %s — requires manual processing", orderID)
	} else {
		executionClient, executionErr := rs.resolveExecutionClient(ctx, orderID, sessGateway)
		if executionErr != nil {
			refundStatus = RefundManualRequired
			log.Printf("[REFUND] Provider execution unavailable for refund on order %s via %s: %v", orderID, sessGateway, executionErr)
		} else {
			providerPaymentID, refLookupErr := rs.lookupProviderPaymentID(ctx, sessionID, providerReference)
			if refLookupErr != nil {
				refundStatus = RefundManualRequired
				log.Printf("[REFUND] Provider payment reference lookup failed for order %s via %s: %v", orderID, sessGateway, refLookupErr)
			} else if providerPaymentID == "" {
				refundStatus = RefundManualRequired
				log.Printf("[REFUND] Provider payment reference missing for order %s via %s; refund requires manual handling", orderID, sessGateway)
			} else {
				refundResult, refErr := executionClient.RefundPayment(ctx, ProviderRefundRequest{
					OrderID:   orderID,
					PaymentID: providerPaymentID,
					Amount:    refundAmount,
					Currency:  resolvedCurrency,
				})
				if refErr != nil {
					if errors.Is(refErr, ErrAdyenDirectOperationUnsupported) || errors.Is(refErr, ErrAirwallexDirectOperationUnsupported) {
						refundStatus = RefundManualRequired
						log.Printf("[REFUND] Gateway refund requires manual handling for order %s via %s: %v", orderID, sessGateway, refErr)
					} else {
						refundStatus = RefundFailed
						log.Printf("[REFUND] Gateway refund failed for order %s via %s: %v", orderID, sessGateway, refErr)
					}
				} else {
					refundStatus = RefundSettled
					providerRefundID = strings.TrimSpace(refundResult.ProviderRefundID)
					if providerRefundID == "" {
						providerRefundID = strings.TrimSpace(refundResult.ProviderReference)
					}
					if providerRefundID == "" {
						providerRefundID = providerPaymentID
					}
					log.Printf("[REFUND] Gateway refund succeeded for order %s via %s: %d %s", orderID, sessGateway, refundAmount, resolvedCurrency)
				}
			}
		}
	}

	// 4. Persist refund + reversal ledger entries in single transaction
	now := time.Now()
	_, txErr := rs.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Insert Refunds row
		refundMut := spanner.Insert("Refunds",
			[]string{"RefundId", "OrderId", "SessionId", "Gateway", "AmountUZS", "Reason", "Status", "ProviderRefundId", "InitiatedBy", "CreatedAt"},
			[]interface{}{refundID, orderID, sessionID, sessGateway, refundAmount, req.Reason, refundStatus, providerRefundID, initiatedBy, now},
		)

		mutations := []*spanner.Mutation{refundMut}

		// Only create reversal ledger entries if gateway actually refunded
		if refundStatus == RefundSettled {
			// Reversal: debit payout owner account + debit platform commission account.
			totalTiyin := refundAmount * 100
			platformReversal := int64(0)
			payoutAccountID := supplierID

			if hasSnapshot {
				effectiveGross := snapshot.GrossAmount
				if effectiveGross <= 0 {
					effectiveGross = maxRefundable
				}
				if effectiveGross > 0 {
					platformReversal = totalTiyin * snapshot.FeeAmount / effectiveGross
				}
				if snapshot.PayoutOwnerID != "" {
					payoutAccountID = snapshot.PayoutOwnerID
				}
			} else {
				platformReversal = totalTiyin * rs.feeBP / 10000
			}
			if platformReversal < 0 {
				platformReversal = 0
			}
			if platformReversal > totalTiyin {
				platformReversal = totalTiyin
			}
			supplierReversal := totalTiyin - platformReversal

			platformTxnID := fmt.Sprintf("TXN-REFUND-PEGASUS-%s", refundID[:8])
			supTxnID := fmt.Sprintf("TXN-REFUND-SUP-%s", refundID[:8])

			// Debit Pegasus (reverse the commission)
			mutations = append(mutations, spanner.Insert("LedgerEntries",
				[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
				[]interface{}{platformTxnID, orderID, finance.PlatformAccountID, -platformReversal, "DEBIT_REFUND", now},
			))

			// Debit payout owner (supplier or warehouse-local owner)
			mutations = append(mutations, spanner.Insert("LedgerEntries",
				[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
				[]interface{}{supTxnID, orderID, payoutAccountID, -supplierReversal, "DEBIT_REFUND", now},
			))

			// Update session status to reflect refund
			mutations = append(mutations, spanner.Update("PaymentSessions",
				[]string{"SessionId", "Status", "UpdatedAt"},
				[]interface{}{sessionID, "REFUNDED", spanner.CommitTimestamp},
			))
		}

		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		// 5. Emit OUTBOX event atomically
		payload := map[string]interface{}{
			"order_id":    orderID,
			"retailer_id": retailerID,
			"supplier_id": supplierID,
			"refund_id":   refundID,
			"amount":      refundAmount,
			"currency":    resolvedCurrency,
			"status":      refundStatus,
			"timestamp":   now.Format(time.RFC3339),
		}

		// NOTE: cannot import backend-go/kafka here — kafka/gateway_worker.go
		// imports backend-go/payment, which would create an import cycle. The
		// versionscan literal-name resolver counts this string-literal use as
		// a producer reference for kafka.EventPaymentRefunded.
		return outbox.EmitJSON(txn, "Refund", refundID, "PAYMENT_REFUNDED", "pegasus-logistics-events", payload, telemetry.TraceIDFromContext(ctx))
	})

	if txErr != nil {
		return nil, fmt.Errorf("refund transaction failed: %w", txErr)
	}

	// Cache invalidate
	cache.Invalidate(ctx, cache.PrefixActiveOrders+retailerID, cache.SupplierProfile(supplierID))

	log.Printf("[REFUND] Refund %s for order %s: %d %s → %s", refundID, orderID, refundAmount, resolvedCurrency, refundStatus)

	return &RefundResult{
		RefundID:         refundID,
		Status:           refundStatus,
		Amount:           refundAmount,
		Currency:         resolvedCurrency,
		AmountUZS:        refundAmount,
		Gateway:          sessGateway,
		ProviderRefundID: providerRefundID,
	}, nil
}

func (rs *RefundService) resolveExecutionClient(ctx context.Context, orderID, gateway string) (ProviderExecutionClient, error) {
	if rs.execution == nil {
		return nil, fmt.Errorf("provider execution router not configured")
	}

	switch normalizeGateway(gateway) {
	case "GLOBAL_PAY":
		if rs.vaultResolver == nil {
			return nil, fmt.Errorf("vault resolver not configured for gateway %s", gateway)
		}
		cfg, err := rs.vaultResolver.GetDecryptedConfigByOrder(ctx, orderID, gateway)
		if err != nil {
			return nil, fmt.Errorf("resolve %s credentials for order %s: %w", gateway, orderID, err)
		}
		creds, err := ResolveGlobalPayCredentials(cfg.MerchantId, cfg.ServiceId, cfg.SecretKey)
		if err != nil {
			return nil, err
		}
		return rs.execution.Resolve(gateway, NewGlobalPayExecutionCredentials(creds))
	case "ADYEN":
		if rs.vaultResolver == nil {
			return nil, fmt.Errorf("vault resolver not configured for gateway %s", gateway)
		}
		cfg, err := rs.vaultResolver.GetDecryptedConfigByOrder(ctx, orderID, gateway)
		if err != nil {
			return nil, fmt.Errorf("resolve %s credentials for order %s: %w", gateway, orderID, err)
		}
		creds, err := ResolveAdyenCredentials(cfg.MerchantId, cfg.ServiceId, cfg.SecretKey)
		if err != nil {
			return nil, err
		}
		return rs.execution.Resolve(gateway, NewAdyenExecutionCredentials(creds))
	case "AIRWALLEX":
		if rs.vaultResolver == nil {
			return nil, fmt.Errorf("vault resolver not configured for gateway %s", gateway)
		}
		cfg, err := rs.vaultResolver.GetDecryptedConfigByOrder(ctx, orderID, gateway)
		if err != nil {
			return nil, fmt.Errorf("resolve %s credentials for order %s: %w", gateway, orderID, err)
		}
		creds, err := ResolveAirwallexCredentials(cfg.MerchantId, cfg.ServiceId, cfg.SecretKey)
		if err != nil {
			return nil, err
		}
		return rs.execution.Resolve(gateway, NewAirwallexExecutionCredentials(creds))
	default:
		return rs.execution.Resolve(gateway, ProviderExecutionCredentials{})
	}
}

func (rs *RefundService) lookupProviderPaymentID(ctx context.Context, sessionID string, providerReference spanner.NullString) (string, error) {
	resolvedReference := strings.TrimSpace(providerReference.StringVal)
	if resolvedReference != "" {
		return resolvedReference, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT ProviderTransactionId
		      FROM PaymentAttempts
		      WHERE SessionId = @sessionID AND ProviderTransactionId IS NOT NULL
		      ORDER BY FinishedAt DESC, StartedAt DESC
		      LIMIT 1`,
		Params: map[string]interface{}{"sessionID": sessionID},
	}
	iter := rs.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query provider transaction for session %s: %w", sessionID, err)
	}

	var providerTxn spanner.NullString
	if err := row.Columns(&providerTxn); err != nil {
		return "", fmt.Errorf("parse provider transaction for session %s: %w", sessionID, err)
	}
	return strings.TrimSpace(providerTxn.StringVal), nil
}

// DeliveryDeltaRefundRequest is the input for InitiateDeliveryDeltaRefund.
// SupplierID/RetailerID are optional context (best-effort populated for the
// outbox event); resolution falls back to the Orders row.
type DeliveryDeltaRefundRequest struct {
	OrderID           string
	DeliverySessionID string
	OriginalAmount    int64
	AdjustedAmount    int64
	SupplierID        string
	RetailerID        string
}

// InitiateDeliveryDeltaRefund issues a partial refund for the price delta
// (OriginalAmount - AdjustedAmount) on an already-SETTLED direct (non-hosted)
// saved-card session. Skipped for hosted/AUTH-CAPTURE flows because those
// capture against PaymentSessions.FinalAmount instead.
//
// Idempotency: RefundId is derived deterministically from DeliverySessionID,
// so replays return the cached result via Spanner PK conflict semantics.
//
// State guard: skipped (returns nil, nil) for orders whose payment session is
// not SETTLED, for CASH gateway, or for non-positive deltas.
func (rs *RefundService) InitiateDeliveryDeltaRefund(ctx context.Context, req DeliveryDeltaRefundRequest) (*RefundResult, error) {
	delta := req.OriginalAmount - req.AdjustedAmount
	if delta <= 0 {
		return nil, nil
	}
	dsid := strings.TrimSpace(req.DeliverySessionID)
	if dsid == "" {
		return nil, fmt.Errorf("delivery session id required")
	}
	if len(dsid) > 32 {
		dsid = dsid[:32]
	}
	deltaRefundID := "DDR-" + dsid

	// Idempotency replay: if a Refunds row already exists for this delivery
	// session, return the cached outcome without retrying the provider.
	if row, err := rs.spanner.Single().ReadRow(ctx, "Refunds",
		spanner.Key{deltaRefundID},
		[]string{"RefundId", "Status", "AmountUZS", "Gateway", "ProviderRefundId"},
	); err == nil {
		var rid, status, gateway string
		var amt int64
		var providerRef spanner.NullString
		if scanErr := row.Columns(&rid, &status, &amt, &gateway, &providerRef); scanErr == nil {
			return &RefundResult{
				RefundID:         rid,
				Status:           status,
				Amount:           amt,
				AmountUZS:        amt,
				Gateway:          gateway,
				ProviderRefundID: providerRef.StringVal,
			}, nil
		}
	} else if spanner.ErrCode(err) != 5 {
		return nil, fmt.Errorf("delivery delta refund idempotency lookup: %w", err)
	}

	// Read order context (supplier/retailer for ledger + event payload).
	supplierID := strings.TrimSpace(req.SupplierID)
	retailerID := strings.TrimSpace(req.RetailerID)
	if supplierID == "" || retailerID == "" {
		if orderRow, err := rs.spanner.Single().ReadRow(ctx, "Orders",
			spanner.Key{req.OrderID},
			[]string{"SupplierId", "RetailerId"},
		); err == nil {
			var s, r string
			if scanErr := orderRow.Columns(&s, &r); scanErr == nil {
				if supplierID == "" {
					supplierID = s
				}
				if retailerID == "" {
					retailerID = r
				}
			}
		}
	}

	// Find SETTLED session. If none, this is a hosted/AUTH-CAPTURE flow and
	// the gateway worker will capture FinalAmount instead — no-op.
	stmt := spanner.Statement{
		SQL:    `SELECT SessionId, Gateway, LockedAmount, PaidAmount, Currency, ProviderReference FROM PaymentSessions WHERE OrderId = @orderID AND Status = 'SETTLED' LIMIT 1`,
		Params: map[string]interface{}{"orderID": req.OrderID},
	}
	iter := rs.spanner.Single().Query(ctx, stmt)
	sessRow, err := iter.Next()
	iter.Stop()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, nil
		}
		return nil, fmt.Errorf("delivery delta refund settled session lookup: %w", err)
	}

	var sessionID, sessGateway string
	var lockedAmount spanner.NullInt64
	var paidAmount spanner.NullInt64
	var sessionCurrency spanner.NullString
	var providerReference spanner.NullString
	if err := sessRow.Columns(&sessionID, &sessGateway, &lockedAmount, &paidAmount, &sessionCurrency, &providerReference); err != nil {
		return nil, fmt.Errorf("delivery delta refund parse session: %w", err)
	}

	// Cash gateway has no programmatic refund path.
	if sessGateway == "CASH" {
		return nil, nil
	}

	resolvedCurrency := normalizeCurrencyCode("")
	if sessionCurrency.Valid {
		resolvedCurrency = normalizeCurrencyCode(sessionCurrency.StringVal)
	}

	// Guard: refund cannot exceed paid amount.
	maxRefundable := lockedAmount.Int64
	if paidAmount.Valid && paidAmount.Int64 > 0 {
		maxRefundable = paidAmount.Int64
	}
	if delta > maxRefundable {
		return nil, fmt.Errorf("delivery delta refund amount %d exceeds maximum refundable %d", delta, maxRefundable)
	}

	// Snapshot-authoritative fee split for reversal ledger math.
	snapshot, hasSnapshot, snapshotErr := LoadSupplierSettlementSnapshot(ctx, rs.spanner, req.OrderID, supplierID)
	if snapshotErr != nil {
		log.Printf("[REFUND.DELTA] settlement snapshot lookup failed for order %s supplier %s: %v", req.OrderID, supplierID, snapshotErr)
	}

	// Provider call. Failures fall back to MANUAL_REQUIRED so operators can
	// reconcile out-of-band; we still persist the Refunds row to anchor
	// idempotency and downstream visibility.
	providerRefundID := ""
	refundStatus := RefundPending
	executionClient, executionErr := rs.resolveExecutionClient(ctx, req.OrderID, sessGateway)
	if executionErr != nil {
		refundStatus = RefundManualRequired
		log.Printf("[REFUND.DELTA] provider execution unavailable order=%s gateway=%s: %v", req.OrderID, sessGateway, executionErr)
	} else {
		providerPaymentID, refLookupErr := rs.lookupProviderPaymentID(ctx, sessionID, providerReference)
		switch {
		case refLookupErr != nil:
			refundStatus = RefundManualRequired
			log.Printf("[REFUND.DELTA] provider payment ref lookup failed order=%s: %v", req.OrderID, refLookupErr)
		case providerPaymentID == "":
			refundStatus = RefundManualRequired
			log.Printf("[REFUND.DELTA] provider payment ref missing order=%s", req.OrderID)
		default:
			refundResult, refErr := executionClient.RefundPayment(ctx, ProviderRefundRequest{
				OrderID:   req.OrderID,
				PaymentID: providerPaymentID,
				Amount:    delta,
				Currency:  resolvedCurrency,
			})
			if refErr != nil {
				if errors.Is(refErr, ErrAdyenDirectOperationUnsupported) || errors.Is(refErr, ErrAirwallexDirectOperationUnsupported) {
					refundStatus = RefundManualRequired
				} else {
					refundStatus = RefundFailed
				}
				log.Printf("[REFUND.DELTA] provider refund failed order=%s gateway=%s: %v", req.OrderID, sessGateway, refErr)
			} else {
				refundStatus = RefundSettled
				providerRefundID = strings.TrimSpace(refundResult.ProviderRefundID)
				if providerRefundID == "" {
					providerRefundID = strings.TrimSpace(refundResult.ProviderReference)
				}
				if providerRefundID == "" {
					providerRefundID = providerPaymentID
				}
			}
		}
	}

	now := time.Now()
	_, txErr := rs.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.Insert("Refunds",
				[]string{"RefundId", "OrderId", "SessionId", "Gateway", "AmountUZS", "Reason", "Status", "ProviderRefundId", "InitiatedBy", "CreatedAt"},
				[]interface{}{deltaRefundID, req.OrderID, sessionID, sessGateway, delta, "DELIVERY_DELTA_REFUND", refundStatus, providerRefundID, "DELIVERY_RECONCILIATION", now},
			),
		}

		if refundStatus == RefundSettled {
			totalTiyin := delta * 100
			platformReversal := int64(0)
			payoutAccountID := supplierID

			if hasSnapshot {
				effectiveGross := snapshot.GrossAmount
				if effectiveGross <= 0 {
					effectiveGross = maxRefundable
				}
				if effectiveGross > 0 {
					platformReversal = totalTiyin * snapshot.FeeAmount / effectiveGross
				}
				if snapshot.PayoutOwnerID != "" {
					payoutAccountID = snapshot.PayoutOwnerID
				}
			} else {
				platformReversal = totalTiyin * rs.feeBP / 10000
			}
			if platformReversal < 0 {
				platformReversal = 0
			}
			if platformReversal > totalTiyin {
				platformReversal = totalTiyin
			}
			supplierReversal := totalTiyin - platformReversal

			platformTxnID := fmt.Sprintf("TXN-DDR-PEGASUS-%s", deltaRefundID[len(deltaRefundID)-8:])
			supTxnID := fmt.Sprintf("TXN-DDR-SUP-%s", deltaRefundID[len(deltaRefundID)-8:])

			mutations = append(mutations, spanner.Insert("LedgerEntries",
				[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
				[]interface{}{platformTxnID, req.OrderID, finance.PlatformAccountID, -platformReversal, "DEBIT_REFUND", now},
			))
			mutations = append(mutations, spanner.Insert("LedgerEntries",
				[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
				[]interface{}{supTxnID, req.OrderID, payoutAccountID, -supplierReversal, "DEBIT_REFUND", now},
			))
		}

		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		// Outbox emit. NOTE: kafka import would create a cycle (kafka imports
		// payment), so we emit by literal event name. notification_dispatcher
		// resolves DELIVERY_DELTA_REFUNDED via its switch table.
		payload := map[string]interface{}{
			"order_id":            req.OrderID,
			"session_id":          sessionID,
			"delivery_session_id": req.DeliverySessionID,
			"refund_id":           deltaRefundID,
			"original_amount":     req.OriginalAmount,
			"adjusted_amount":     req.AdjustedAmount,
			"delta_amount":        delta,
			"currency":            resolvedCurrency,
			"gateway":             sessGateway,
			"status":              refundStatus,
			"provider_refund_id":  providerRefundID,
			"supplier_id":         supplierID,
			"retailer_id":         retailerID,
			"timestamp":           now.Format(time.RFC3339),
		}
		return outbox.EmitJSON(txn, "Refund", deltaRefundID, "DELIVERY_DELTA_REFUNDED", "pegasus-logistics-events", payload, telemetry.TraceIDFromContext(ctx))
	})

	if txErr != nil {
		return nil, fmt.Errorf("delivery delta refund persistence: %w", txErr)
	}

	cache.Invalidate(ctx, cache.PrefixActiveOrders+retailerID, cache.SupplierProfile(supplierID))

	log.Printf("[REFUND.DELTA] order=%s session=%s delta=%d %s status=%s", req.OrderID, sessionID, delta, resolvedCurrency, refundStatus)

	return &RefundResult{
		RefundID:         deltaRefundID,
		Status:           refundStatus,
		Amount:           delta,
		AmountUZS:        delta,
		Currency:         resolvedCurrency,
		Gateway:          sessGateway,
		ProviderRefundID: providerRefundID,
	}, nil
}
func (rs *RefundService) GetRefundsByOrder(ctx context.Context, orderID string) ([]RefundResult, error) {
	stmt := spanner.Statement{
		SQL: `SELECT r.RefundId, r.Status, r.AmountUZS, r.Gateway, r.ProviderRefundId, ps.Currency
		      FROM Refunds r
		      LEFT JOIN PaymentSessions ps ON ps.SessionId = r.SessionId
		      WHERE r.OrderId = @orderID
		      ORDER BY r.CreatedAt DESC`,
		Params: map[string]interface{}{"orderID": orderID},
	}
	iter := rs.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []RefundResult
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query refunds for order %s: %w", orderID, err)
		}
		var r RefundResult
		var providerRef spanner.NullString
		var currency spanner.NullString
		if err := row.Columns(&r.RefundID, &r.Status, &r.AmountUZS, &r.Gateway, &providerRef, &currency); err != nil {
			return nil, fmt.Errorf("parse refund for order %s: %w", orderID, err)
		}
		r.ProviderRefundID = providerRef.StringVal
		r.Amount = r.AmountUZS
		r.Currency = normalizeCurrencyCode("")
		if currency.Valid {
			r.Currency = normalizeCurrencyCode(currency.StringVal)
		}
		results = append(results, r)
	}

	return results, nil
}
