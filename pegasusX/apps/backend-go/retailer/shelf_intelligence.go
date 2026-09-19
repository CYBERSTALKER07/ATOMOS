package retailer

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// RetailerShelfAlert represents an out-of-stock or low-stock incident.
type RetailerShelfAlert struct {
	AlertID           string    `json:"alert_id"`
	RetailerID        string    `json:"retailer_id"`
	LocationID        string    `json:"location_id"`
	GlobalProductID   string    `json:"global_product_id"`
	Status            string    `json:"status"` // OPEN or RESOLVED
	CurrentStock      int64     `json:"current_stock"`
	CapacityThreshold int64     `json:"capacity_threshold"`
	CreatedAt         time.Time `json:"created_at"`
	ResolvedAt        time.Time `json:"resolved_at,omitempty"`
}

const (
	ShelfAlertStatusOpen     = "OPEN"
	ShelfAlertStatusResolved = "RESOLVED"
)

// CheckAndGenerateOOSAlerts compares CurrentStock in StoreStock vs an arbitrary capacity threshold (stub for planning logic).
// Emits an alert if the stock is below the threshold and an OPEN alert does not already exist.
func (s *Service) CheckAndGenerateOOSAlerts(ctx context.Context, retailerID, locationID, productID string, currentStock, threshold int64) error {
	if currentStock >= threshold || s.spannerClient == nil {
		// Nothing to alert or no DB
		return nil
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Check if open alert exists
		stmt := spanner.Statement{
			SQL: `SELECT AlertId FROM RetailerShelfAlerts 
				WHERE RetailerId = @ret AND LocationId = @loc AND GlobalProductId = @prod AND Status = 'OPEN'`,
			Params: map[string]interface{}{
				"ret":  retailerID,
				"loc":  locationID,
				"prod": productID,
			},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()
		_, err := iter.Next()
		if err == nil {
			// Alert already open
			return nil
		}

		alertID := uuid.New().String()
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("RetailerShelfAlerts",
				[]string{"AlertId", "RetailerId", "LocationId", "GlobalProductId", "Status", "CurrentStock", "CapacityThreshold", "CreatedAt"},
				[]interface{}{alertID, retailerID, locationID, productID, ShelfAlertStatusOpen, currentStock, threshold, spanner.CommitTimestamp},
			),
		})

		// [Phase 2] Real-time Shelf Intelligence WebSocket requires outbox event fanout.
		// Deferring this for now.

		return nil
	})
	return err
}

// ResolveShelfAlert marks an alert as resolved once stock is replenished.
func ResolveShelfAlert(ctx context.Context, client *spanner.Client, retailerID, locationID, alertID string) error {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.Update("RetailerShelfAlerts",
			[]string{"AlertId", "RetailerId", "LocationId", "Status", "ResolvedAt"},
			[]interface{}{alertID, retailerID, locationID, ShelfAlertStatusResolved, spanner.CommitTimestamp},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}
