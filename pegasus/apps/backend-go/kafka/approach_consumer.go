package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"backend-go/kafka/workerpool"
	"backend-go/notifications"
	"backend-go/telemetry"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	goKafka "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ─── DRIVER_APPROACHING Consumer ───────────────────────────────────────────────
// Listens to the logistics topic for DRIVER_APPROACHING events
// fired by the Proximity Engine (Phase 2). On receipt, executes the Push Protocol:
//  1. WebSocket push to connected retailer (instant QR popup)
//  2. FCM data payload fallback (app in background)

// StartApproachConsumer boots the partition-parallel Kafka consumer for
// approach events. Returns immediately; the worker pool runs until ctx is
// cancelled.
func StartApproachConsumer(
	ctx context.Context,
	retailerHub *ws.RetailerHub,
	fcm *notifications.FCMClient,
	spannerClient *spanner.Client,
	brokerAddress string,
) {
	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    TopicMain,
		GroupID:  "pegasus-approach-consumer-group",
		MinBytes: 1,
		MaxBytes: 10 << 20,
	})

	pool, err := workerpool.New(workerpool.Config{
		Source: reader,
		Name:   "approach-consumer",
		Logger: slog.Default(),
		Handler: func(ctx context.Context, m goKafka.Message) error {
			eventType := EventType(m.Headers, m.Key)
			if eventType != "DRIVER_APPROACHING" {
				return nil
			}
			var event approachEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				return fmt.Errorf("unmarshal DRIVER_APPROACHING: %w", err)
			}
			slog.InfoContext(ctx, "approach: event received",
				"order_id", event.OrderID, "retailer_id", event.RetailerID)
			handleApproachEvent(ctx, retailerHub, fcm, spannerClient, &event)
			return nil
		},
	})
	if err != nil {
		slog.Error("approach consumer init failed", "err", err)
		return
	}
	go func() {
		if err := pool.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("approach consumer pool exited", "err", err)
		}
	}()
	slog.Info("approach consumer ONLINE", "topic", TopicMain)
}

// approachEvent mirrors proximity.DriverApproachingEvent but is locally defined
// to avoid circular imports between kafka and proximity packages.
type approachEvent struct {
	OrderID         string    `json:"order_id"`
	SupplierID      string    `json:"supplier_id"`
	SupplierName    string    `json:"supplier_name"`
	RetailerID      string    `json:"retailer_id"`
	DeliveryToken   string    `json:"delivery_token"`
	DriverLatitude  float64   `json:"driver_latitude"`
	DriverLongitude float64   `json:"driver_longitude"`
	Timestamp       time.Time `json:"timestamp"`
}

// handleApproachEvent executes the Push Protocol:
// 1. Try WebSocket (instant, in-app QR popup).
// 2. Fallback to FCM data payload (app backgrounded).
// 3. Mirror to supplier admin portal via telemetry WebSocket.
func handleApproachEvent(
	ctx context.Context,
	retailerHub *ws.RetailerHub,
	fcm *notifications.FCMClient,
	spannerClient *spanner.Client,
	event *approachEvent,
) {
	payload := ws.ApproachPayload{
		Type:            "DRIVER_APPROACHING",
		OrderID:         event.OrderID,
		SupplierID:      event.SupplierID,
		SupplierName:    event.SupplierName,
		RetailerID:      event.RetailerID,
		DeliveryToken:   event.DeliveryToken,
		DriverLatitude:  event.DriverLatitude,
		DriverLongitude: event.DriverLongitude,
	}

	if retailerHub.PushToRetailer(event.RetailerID, payload) {
		slog.InfoContext(ctx, "push protocol: ws delivery success",
			"retailer_id", event.RetailerID, "order_id", event.OrderID)
	} else {
		slog.InfoContext(ctx, "push protocol: ws miss, fcm fallback",
			"retailer_id", event.RetailerID)

		deviceToken, err := lookupRetailerDeviceToken(ctx, spannerClient, event.RetailerID)
		if err != nil || deviceToken == "" {
			slog.WarnContext(ctx, "push protocol: no fcm token; dropped",
				"retailer_id", event.RetailerID, "err", err)
		} else if err := fcm.SendDataMessage(deviceToken, map[string]string{
			"type":             ws.EventDriverApproaching,
			"order_id":         event.OrderID,
			"supplier_id":      event.SupplierID,
			"supplier_name":    event.SupplierName,
			"retailer_id":      event.RetailerID,
			"delivery_token":   event.DeliveryToken,
			"driver_latitude":  fmt.Sprintf("%f", event.DriverLatitude),
			"driver_longitude": fmt.Sprintf("%f", event.DriverLongitude),
		}); err != nil {
			slog.WarnContext(ctx, "push protocol: fcm fallback failed",
				"retailer_id", event.RetailerID, "err", err)
		} else {
			slog.InfoContext(ctx, "push protocol: fcm delivered",
				"retailer_id", event.RetailerID, "order_id", event.OrderID)
		}
	}

	if event.SupplierID != "" {
		telemetry.FleetHub.BroadcastDriverApproaching(
			event.SupplierID, event.OrderID,
			event.DriverLatitude, event.DriverLongitude,
		)
	}
}

// lookupRetailerDeviceToken fetches the FCM push token from the Retailers table.
func lookupRetailerDeviceToken(ctx context.Context, client *spanner.Client, retailerID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	stmt := spanner.Statement{
		SQL:    `SELECT FcmToken FROM Retailers WHERE RetailerId = @id LIMIT 1`,
		Params: map[string]interface{}{"id": retailerID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return "", fmt.Errorf("retailer %s not found", retailerID)
	}
	if err != nil {
		return "", fmt.Errorf("spanner query failed: %w", err)
	}

	var token spanner.NullString
	if err := row.Columns(&token); err != nil {
		return "", fmt.Errorf("column parse failed: %w", err)
	}

	return token.StringVal, nil
}
