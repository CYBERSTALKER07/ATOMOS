// Package sync implements the Desert Protocol — the offline batch sync engine.
//
// Architecture: when a driver re-enters LTE coverage, their Expo app fires
// a single POST /v1/sync/batch with every delivery buffered in WatermelonDB.
//
// Conflict-resolution math (why Redis dedup is mandatory):
//   - The POST completes but the 200 OK drops in a tunnel → the app retries.
//   - Without Redis, Spanner sees the same OrderId twice → double-credit.
//   - With Redis SETNX (TTL 24 h), the second attempt for the same
//     (driver_id, order_id) pair is detected and skipped in O(1).
//   - The Kafka emit is gated AFTER Redis locks, so the Treasurer never sees
//     the duplicate either.
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/cache"
	"backend-go/kafka"

	"cloud.google.com/go/spanner"
)

// OfflineDelivery represents a single cryptographic handshake locked in
// WatermelonDB while the driver was in a dead zone.
type OfflineDelivery struct {
	OrderID   string `json:"order_id"`
	Signature string `json:"signature"` // SHA-256 hash verified offline
	Timestamp int64  `json:"timestamp"` // Unix ms of the scan
	Status    string `json:"status"`    // "DELIVERED" | "REJECTED_DAMAGED"
}

// BatchSyncPayload is the full body sent by executeDesertProtocol() in the app.
type BatchSyncPayload struct {
	DriverID   string            `json:"driver_id"`
	Deliveries []OfflineDelivery `json:"deliveries"`
}

// BatchSyncResponse tells the phone exactly which orders were locked so it can
// safely purge them from WatermelonDB without data loss.
type BatchSyncResponse struct {
	Status    string   `json:"status"`
	Processed []string `json:"processed"`
	Skipped   int      `json:"skipped"` // duplicates detected by Redis
}

// HandleBatchSync processes the offline queue when LTE is restored.
//
// Route: POST /v1/sync/batch
// Auth:  JWT with role=DRIVER enforced by auth.RequireRole middleware upstream.
func HandleBatchSync(db *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload BatchSyncPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.DriverID == "" {
			http.Error(w, `{"error":"malformed sync payload or missing driver_id"}`, http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		var processed []string
		skipped := 0

		for _, delivery := range payload.Deliveries {
			if delivery.OrderID == "" || delivery.Status == "" {
				log.Printf("[DESERT] Skipping malformed delivery in batch from %s", payload.DriverID)
				continue
			}

			// ── Step 0: Order Reassignment Guard (Phase 5.5) ───────────────────────
			// Verify this order is still assigned to this driver. If the order was
			// reassigned while the driver was offline, reject it to prevent
			// double-delivery or stale state transitions.
			if db != nil {
				assignRow, assignErr := db.Single().ReadRow(ctx, "Orders",
					spanner.Key{delivery.OrderID}, []string{"DriverId"})
				if assignErr != nil {
					log.Printf("[DESERT] Cannot verify assignment for %s: %v — skipping", delivery.OrderID, assignErr)
					continue
				}
				var currentDriverID spanner.NullString
				if err := assignRow.Columns(&currentDriverID); err != nil {
					log.Printf("[DESERT] Cannot parse DriverId for %s: %v — skipping", delivery.OrderID, err)
					continue
				}
				if !currentDriverID.Valid || currentDriverID.StringVal != payload.DriverID {
					log.Printf("[DESERT] Order %s reassigned (current=%s, sync=%s) — rejecting",
						delivery.OrderID, currentDriverID.StringVal, payload.DriverID)
					skipped++
					continue
				}
			}

			// ── Step 1: Redis Idempotency Gate ─────────────────────────────────────
			// SETNX (SET if Not eXists) with a 24-hour TTL.
			// If the key already exists, this batch was previously received (possibly
			// from a retry after a dropped ACK). We skip safely.
			dedupKey := fmt.Sprintf("%s%s:%s", cache.PrefixDesertSync, payload.DriverID, delivery.OrderID)
			locked, err := cache.Client.SetNX(ctx, dedupKey, "sealed", cache.TTLDesertSync).Result()
			if err != nil {
				// Redis is unavailable — fail CLOSED to prevent double-credit.
				// The delivery stays in the phone's local queue for the next sync pulse.
				log.Printf("[DESERT] Redis dedup unavailable for %s/%s: %v — skipping to prevent double-credit",
					payload.DriverID, delivery.OrderID, err)
				continue
			} else if !locked {
				// Already processed. Safe to skip — the phone will delete on the
				// processed list, but this order just won't be in it.
				log.Printf("[DESERT] Duplicate detected (Redis): %s for driver %s — skipped",
					delivery.OrderID, payload.DriverID)
				skipped++
				// We still add it to processed so the phone cleans up its local DB.
				// If we returned it as "not processed", the phone would keep retrying
				// forever. The Kafka side is already idempotent on order_id anyway.
				processed = append(processed, delivery.OrderID)
				continue
			}

			// ── Step 2: Zero-Trust Timestamp Guard ─────────────────────────────────
			// Reject deliveries older than 48 hours (prevents replaying stale events).
			scanTime := time.UnixMilli(delivery.Timestamp)
			if time.Since(scanTime) > 48*time.Hour {
				log.Printf("[DESERT] Stale delivery rejected: %s (scanned %v ago)",
					delivery.OrderID, time.Since(scanTime).Round(time.Minute))
				// Unlock Redis so the driver doesn't get silently stuck
				cache.Client.Del(ctx, dedupKey)
				continue
			}

			// ── Step 3a: QUARANTINE commit — Spanner BEFORE Kafka emit ─────────────
			// For REJECTED_DAMAGED deliveries: persist the state change first.
			// Emitting an event for a state that hasn't been committed creates ghost
			// events in downstream consumers. If Spanner fails, unlock Redis so the
			// phone retries on the next sync pulse.
			if delivery.Status == "REJECTED_DAMAGED" && db != nil {
				quarantineCtx, qCancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, quarantineErr := db.ReadWriteTransaction(quarantineCtx, func(qCtx context.Context, txn *spanner.ReadWriteTransaction) error {
					stmt := spanner.Statement{
						SQL:    `UPDATE Orders SET State = 'QUARANTINE' WHERE OrderId = @orderId AND State IN ('LOADED','DISPATCHED','IN_TRANSIT','ARRIVING','ARRIVED')`,
						Params: map[string]interface{}{"orderId": delivery.OrderID},
					}
					_, err := txn.Update(qCtx, stmt)
					return err
				})
				qCancel()
				if quarantineErr != nil {
					log.Printf("[DESERT] QUARANTINE transition FAILED for %s: %v — unlocking dedup, skipping emit",
						delivery.OrderID, quarantineErr)
					cache.Client.Del(ctx, dedupKey)
					continue
				}
				log.Printf("[DESERT] Order %s → QUARANTINE committed, proceeding to emit", delivery.OrderID)
			}

			// ── Step 3b: Emit Immutable Event to Kafka ─────────────────────────────
			// Spanner state is committed above (if REJECTED_DAMAGED). Safe to emit.
			// If Kafka fails for REJECTED_DAMAGED: dedup key removed → phone retries →
			// Spanner UPDATE is a no-op (already QUARANTINE) → Kafka retried cleanly.
			event := kafka.OrderSyncEvent{
				OrderID:   delivery.OrderID,
				DriverID:  payload.DriverID,
				NewStatus: delivery.Status,
				Signature: delivery.Signature,
				Timestamp: delivery.Timestamp,
			}

			if err := kafka.EmitOrderSyncEvent(event); err != nil {
				// Kafka is down. Unlock Redis so the next sync pulse can retry.
				log.Printf("[DESERT] Kafka emit failed for %s: %v — unlocking dedup key",
					delivery.OrderID, err)
				cache.Client.Del(ctx, dedupKey)
				// Do NOT add to processed — the phone keeps this in its local queue.
				continue
			}

			log.Printf("[DESERT] Locked & emitted: %s | %s | driver=%s",
				delivery.OrderID, delivery.Status, payload.DriverID)
			processed = append(processed, delivery.OrderID)
		}

		// ── Step 4: Return the lock manifest ───────────────────────────────────────
		// The phone ONLY deletes WatermelonDB records whose order_id appears in
		// processed[]. Everything else stays buffered for the next sync pulse.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BatchSyncResponse{
			Status:    "SYNC_COMPLETE",
			Processed: processed,
			Skipped:   skipped,
		})
	}
}
