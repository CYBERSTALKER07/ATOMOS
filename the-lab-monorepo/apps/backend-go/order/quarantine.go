package order

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	kafka "github.com/segmentio/kafka-go"
)

// HandleCompleteRoute transitions all undelivered orders on a route to QUARANTINE.
// Called by the driver upon returning to the depot when not all drops were completed.
// Orders already in COMPLETED/AWAITING_PAYMENT/ARRIVED are left untouched.
// VU is intentionally NOT released — physical goods remain on the vehicle.
//
// Route: POST /v1/fleet/route/{routeId}/complete
// Auth:  DRIVER role
func HandleCompleteRoute(db *spanner.Client, producer *kafka.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract routeId from path: /v1/fleet/route/{routeId}/complete
		path := strings.TrimPrefix(r.URL.Path, "/v1/fleet/route/")
		path = strings.TrimSuffix(path, "/complete")
		routeID := strings.TrimSpace(path)
		if routeID == "" {
			http.Error(w, `{"error":"route_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		var quarantinedIDs []string

		_, txnErr := db.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Collect the IDs being quarantined so we can include them in the event payload
			readStmt := spanner.Statement{
				SQL:    `SELECT OrderId FROM Orders WHERE RouteId = @routeId AND State IN ('LOADED','IN_TRANSIT','ARRIVING')`,
				Params: map[string]interface{}{"routeId": routeID},
			}
			iter := txn.Query(ctx, readStmt)
			defer iter.Stop()
			for {
				row, err := iter.Next()
				if err != nil {
					break
				}
				var id string
				if err := row.Columns(&id); err == nil {
					quarantinedIDs = append(quarantinedIDs, id)
				}
			}
			if len(quarantinedIDs) == 0 {
				return nil // nothing undelivered — idempotent
			}

			// Transition undelivered orders to QUARANTINE
			updateStmt := spanner.Statement{
				SQL:    `UPDATE Orders SET State = 'QUARANTINE' WHERE RouteId = @routeId AND State IN ('LOADED','DISPATCHED','IN_TRANSIT','ARRIVING')`,
				Params: map[string]interface{}{"routeId": routeID},
			}
			_, err := txn.Update(ctx, updateStmt)
			return err
		})

		if txnErr != nil {
			log.Printf("[QUARANTINE] route=%s: state transition failed: %v", routeID, txnErr)
			http.Error(w, `{"error":"failed to complete route"}`, http.StatusInternalServerError)
			return
		}

		// Emit ROUTE_COMPLETED event so telemetry and downstream consumers are notified
		if len(quarantinedIDs) > 0 && producer != nil {
			type routeCompletedEvent struct {
				RouteID        string   `json:"route_id"`
				DriverID       string   `json:"driver_id"`
				QuarantinedIDs []string `json:"quarantined_order_ids"`
				Timestamp      int64    `json:"timestamp"`
			}
			payload, _ := json.Marshal(routeCompletedEvent{
				RouteID:        routeID,
				DriverID:       claims.UserID,
				QuarantinedIDs: quarantinedIDs,
				Timestamp:      time.Now().UnixMilli(),
			})
			kafkaCtx, kafkaCancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := producer.WriteMessages(kafkaCtx, kafka.Message{
				Key:   []byte(routeID),
				Value: payload,
			}); err != nil {
				log.Printf("[QUARANTINE] WARNING: ROUTE_COMPLETED Kafka emit failed for route=%s: %v", routeID, err)
			}
			kafkaCancel()
		}

		log.Printf("[QUARANTINE] route=%s | %d orders quarantined | driver=%s",
			routeID, len(quarantinedIDs), claims.UserID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "ROUTE_COMPLETED",
			"quarantined":     len(quarantinedIDs),
			"quarantined_ids": quarantinedIDs,
		})
	}
}
