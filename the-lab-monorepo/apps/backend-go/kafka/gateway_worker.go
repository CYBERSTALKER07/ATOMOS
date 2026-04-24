package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"backend-go/kafka/workerpool"
	"backend-go/global_paynt"

	"cloud.google.com/go/spanner"
	kafkago "github.com/segmentio/kafka-go"
)

// GatewayWorkerDeps holds dependencies for the gateway capture consumer.
type GatewayWorkerDeps struct {
	Spanner       *spanner.Client
	BrokerAddress string
	Vault         global_paynt.VaultResolver          // Per-supplier credential resolution
	GPDirect      *global_paynt.GlobalPayDirectClient // Global Pay direct API client (may be nil)
}

// StartGatewayWorker begins the partition-parallel async global_paynt-capture consumer.
// Processes GlobalPayntIntentCreated events emitted by the Treasurer. On success it
// updates LedgerEntries Status from PENDING_GATEWAY → SETTLED. On failure it
// marks Status → FAILED and routes the event to the DLQ.
//
// Decoupling the charge from the ledger write eliminates the P0 race condition
// where a gateway charge succeeds but the Spanner commit fails.
// Returns immediately; the pool runs until ctx is cancelled.
func StartGatewayWorker(ctx context.Context, deps GatewayWorkerDeps) {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  []string{deps.BrokerAddress},
		Topic:    TopicMain,
		GroupID:  "lab-gateway-worker-group",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	pool, err := workerpool.New(workerpool.Config{
		Source: reader,
		Name:   "gateway-worker",
		Logger: slog.Default(),
		Handler: func(ctx context.Context, m kafkago.Message) error {
			eventType := EventType(m.Headers, m.Key)
			if eventType != EventGlobalPayntIntentCreated {
				return nil
			}
			var intent GlobalPayntIntentEvent
			if err := json.Unmarshal(m.Value, &intent); err != nil {
				return fmt.Errorf("gateway_worker: unmarshal %s: %w", EventGlobalPayntIntentCreated, err)
			}
			executeGatewayCapture(deps, intent)
			return nil
		},
		OnFailure: func(ctx context.Context, m kafkago.Message, handlerErr error) {
			slog.ErrorContext(ctx, "gateway_worker.handler_error",
				"partition", m.Partition, "offset", m.Offset, "err", handlerErr)
			RouteToDLQ(LogisticsEvent{
				EventName: EventGlobalPayntIntentCreated,
				OrderId:   string(m.Key),
				Timestamp: time.Now().UTC(),
			}, handlerErr.Error())
		},
	})
	if err != nil {
		slog.Error("gateway_worker: pool init failed", "err", err)
		return
	}
	go func() {
		if err := pool.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("gateway_worker: pool exited", "err", err)
		}
	}()
	slog.Info("gateway worker ONLINE", "topic", TopicMain, "group", "lab-gateway-worker-group")
}

// executeGatewayCapture charges the global_paynt provider and updates ledger status.
// For GLOBAL_PAY with an existing authorization, it captures the held amount
// (which may be less than originally authorized after driver edits).
// For GlobalPay/Cash, it performs the legacy full-charge path.
func executeGatewayCapture(deps GatewayWorkerDeps, intent GlobalPayntIntentEvent) {
	// Build a LogisticsEvent for DLQ routing in case of failure.
	dlqEvent := LogisticsEvent{
		EventName: EventGlobalPayntIntentCreated,
		OrderId:   intent.OrderID,
		Amount:    intent.Amount,
		Timestamp: time.Now(),
	}

	// ── Idempotency guard: skip if ledger already settled or failed ──
	currentStatus, guardErr := readLedgerStatus(deps.Spanner, intent.LabTxnId)
	if guardErr != nil {
		slog.Error("gateway_worker.ledger_status_read_failed", "order_id", intent.OrderID, "txn_id", intent.LabTxnId, "err", guardErr)
		return
	}
	if currentStatus != "PENDING_GATEWAY" {
		slog.Info("gateway_worker.idempotency_skip", "order_id", intent.OrderID, "current_status", currentStatus)
		return
	}

	// ── Global Pay auth-capture path: capture an existing hold ──
	if intent.AuthorizationID != "" && intent.GlobalPayntGateway == "GLOBAL_PAY" {
		captureAmount := intent.FinalAmount
		if captureAmount <= 0 {
			captureAmount = intent.Amount
		}

		if deps.GPDirect == nil {
			slog.Error("gateway_worker.gp_client_not_configured", "order_id", intent.OrderID)
			updateLedgerStatus(deps.Spanner, intent, "FAILED", "global_pay_direct_not_configured")
			RouteToDLQ(dlqEvent, "global_pay_direct_not_configured")
			return
		}

		// Resolve per-supplier credentials from vault.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		creds, credErr := resolveGPCredentials(ctx, deps.Vault, intent.OrderID)
		if credErr != nil {
			slog.Error("gateway_worker.credential_resolution_failed", "order_id", intent.OrderID, "err", credErr)
			updateLedgerStatus(deps.Spanner, intent, "FAILED", fmt.Sprintf("credential_resolution_failed: %v", credErr))
			RouteToDLQ(dlqEvent, fmt.Sprintf("credential_resolution_failed: %v", credErr))
			return
		}

		// Capture: amount in tiyins (GP API expects tiyins)
		captureTiyin := captureAmount * 100
		captureResult, captureErr := deps.GPDirect.CaptureGlobalPaynt(ctx, creds, intent.AuthorizationID, captureTiyin)
		if captureErr != nil {
			slog.Error("gateway_worker.capture_failed", "order_id", intent.OrderID, "authorization_id", intent.AuthorizationID, "err", captureErr)
			updateLedgerStatus(deps.Spanner, intent, "FAILED", fmt.Sprintf("capture_failed: %v", captureErr))
			RouteToDLQ(dlqEvent, fmt.Sprintf("capture_failed_%s: %v", intent.GlobalPayntGateway, captureErr))
			return
		}

		slog.Info("gateway_worker.capture_succeeded", "order_id", intent.OrderID, "authorization_id", intent.AuthorizationID, "amount", captureAmount, "captured", captureResult.Captured)
		updateLedgerStatus(deps.Spanner, intent, "SETTLED", "")
		return
	}

	// ── Legacy full-charge path: GlobalPay / Cash / other gateways ──
	gw, gwErr := global_paynt.NewGatewayClient(intent.GlobalPayntGateway)
	if gwErr != nil {
		slog.Error("gateway_worker.gateway_init_failed", "gateway", intent.GlobalPayntGateway, "order_id", intent.OrderID, "err", gwErr)
		updateLedgerStatus(deps.Spanner, intent, "FAILED", fmt.Sprintf("gateway_init_failed: %v", gwErr))
		RouteToDLQ(dlqEvent, fmt.Sprintf("gateway_init_failed: %v", gwErr))
		return
	}

	if chargeErr := gw.Charge(intent.OrderID, intent.Amount); chargeErr != nil {
		slog.Error("gateway_worker.charge_failed", "order_id", intent.OrderID, "gateway", intent.GlobalPayntGateway, "err", chargeErr)
		updateLedgerStatus(deps.Spanner, intent, "FAILED", fmt.Sprintf("charge_failed: %v", chargeErr))
		RouteToDLQ(dlqEvent, fmt.Sprintf("charge_failed_%s: %v", intent.GlobalPayntGateway, chargeErr))
		return
	}

	slog.Info("gateway_worker.charge_succeeded", "order_id", intent.OrderID, "gateway", intent.GlobalPayntGateway, "amount", intent.Amount)
	updateLedgerStatus(deps.Spanner, intent, "SETTLED", "")
}

// resolveGPCredentials resolves Global Pay credentials from vault for a given order.
func resolveGPCredentials(ctx context.Context, vault global_paynt.VaultResolver, orderID string) (global_paynt.GlobalPayCredentials, error) {
	if vault == nil {
		return global_paynt.ResolveGlobalPayCredentials("", "", "")
	}
	cfg, err := vault.GetDecryptedConfigByOrder(ctx, orderID, "GLOBAL_PAY")
	if err != nil {
		// Fall back to environment-level credentials.
		return global_paynt.ResolveGlobalPayCredentials("", "", "")
	}
	return global_paynt.ResolveGlobalPayCredentials(cfg.MerchantId, cfg.ServiceId, cfg.SecretKey)
}

// readLedgerStatus returns the current Status of a ledger entry by TransactionId.
// Used as the idempotency guard before calling external global_paynt APIs.
func readLedgerStatus(client *spanner.Client, txnID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row, err := client.Single().ReadRow(ctx, "LedgerEntries", spanner.Key{txnID}, []string{"Status"})
	if err != nil {
		return "", fmt.Errorf("read ledger entry %s: %w", txnID, err)
	}
	var status spanner.NullString
	if err := row.Columns(&status); err != nil {
		return "", fmt.Errorf("parse ledger status %s: %w", txnID, err)
	}
	return status.StringVal, nil
}

// updateLedgerStatus transitions LedgerEntries for a given order to the target status.
// Uses a conditional read-then-write inside a ReadWriteTransaction: the write only
// proceeds if the current status is PENDING_GATEWAY, preventing stale-replay overwrites.
func updateLedgerStatus(client *spanner.Client, intent GlobalPayntIntentEvent, targetStatus, failureReason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		now := time.Now()
		mutations := make([]*spanner.Mutation, 0, 2)

		// Update both ledger rows (Lab + Supplier) atomically with conditional guard.
		for _, txnID := range []string{intent.LabTxnId, intent.SupplierTxnId} {
			// Read current status inside the RW txn (serializable isolation).
			row, readErr := txn.ReadRow(ctx, "LedgerEntries", spanner.Key{txnID}, []string{"Status"})
			if readErr != nil {
				return fmt.Errorf("read ledger entry %s: %w", txnID, readErr)
			}
			var currentStatus spanner.NullString
			if colErr := row.Columns(&currentStatus); colErr != nil {
				return fmt.Errorf("parse ledger status %s: %w", txnID, colErr)
			}
			if currentStatus.StringVal != "PENDING_GATEWAY" {
				slog.Info("gateway_worker.conditional_guard_skip", "txn_id", txnID, "current_status", currentStatus.StringVal)
				return nil
			}

			cols := []string{"TransactionId", "Status"}
			vals := []interface{}{txnID, targetStatus}

			if targetStatus == "SETTLED" {
				cols = append(cols, "SettledAt")
				vals = append(vals, now)
			}
			if failureReason != "" {
				cols = append(cols, "FailureReason")
				vals = append(vals, failureReason)
			}

			mutations = append(mutations, spanner.Update("LedgerEntries", cols, vals))
		}

		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})

	if err != nil {
		slog.Error("gateway_worker.ledger_update_failed", "order_id", intent.OrderID, "target_status", targetStatus, "err", err)
	} else {
		slog.Info("gateway_worker.ledger_updated", "order_id", intent.OrderID, "status", targetStatus)
	}
}
