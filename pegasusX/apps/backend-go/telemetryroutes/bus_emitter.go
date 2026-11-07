package telemetryroutes

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// DirectKafkaLocationBusEmitter streams throttled driver location events directly
// to Kafka TopicRealtime via outbox.Publisher, bypassing Cloud Spanner OutboxEvents
// commits for high-frequency ephemeral telemetry.
type DirectKafkaLocationBusEmitter struct {
	publisher outbox.Publisher
	log       *slog.Logger
}

func NewDirectKafkaLocationBusEmitter(publisher outbox.Publisher, log *slog.Logger) *DirectKafkaLocationBusEmitter {
	if log == nil {
		log = slog.Default()
	}
	return &DirectKafkaLocationBusEmitter{
		publisher: publisher,
		log:       log,
	}
}

func (e *DirectKafkaLocationBusEmitter) EmitDriverLocation(ctx context.Context, supplierID, driverID, routeID string, payload []byte) error {
	if e == nil || e.publisher == nil {
		return nil
	}
	if driverID == "" {
		return nil
	}
	body := payload
	if routeID != "" {
		body = injectRouteID(payload, routeID)
	}

	if hp, ok := e.publisher.(outbox.HeaderPublisher); ok {
		headers := map[string][]byte{
			"driver_id":      []byte(driverID),
			"supplier_id":    []byte(supplierID),
			"aggregate_type": []byte(events.AggregateDriver),
		}
		if routeID != "" {
			headers["route_id"] = []byte(routeID)
		}
		return hp.PublishWithHeaders(ctx, events.TopicRealtime, []byte(driverID), body, headers)
	}

	return e.publisher.Publish(ctx, events.TopicRealtime, []byte(driverID), body)
}

// SpannerLocationBusEmitter is the production LocationBusEmitter: it writes a
// DRIVER_LOCATION_UPDATED row to OutboxEvents (TopicRealtime) so the relay
// publishes it to Kafka, lighting up the notification dispatcher and the
// digital-twin consumer. It writes a standalone mutation (no surrounding domain
// txn — the driver location itself lives in Redis/WS at full fidelity; this is
// the throttled bus copy). route_id is injected into the payload when known so
// twin consumers (keyed on route) can consume it.
type SpannerLocationBusEmitter struct {
	client *spanner.Client
}

func NewSpannerLocationBusEmitter(client *spanner.Client) *SpannerLocationBusEmitter {
	return &SpannerLocationBusEmitter{client: client}
}

func (e *SpannerLocationBusEmitter) EmitDriverLocation(ctx context.Context, supplierID, driverID, routeID string, payload []byte) error {
	if e == nil || e.client == nil {
		return nil
	}
	if driverID == "" {
		return nil
	}
	body := payload
	if routeID != "" {
		body = injectRouteID(payload, routeID)
	}
	event := outbox.Event{
		EventID:       newLocationEventID(driverID),
		AggregateType: events.AggregateDriver,
		AggregateID:   driverID,
		TopicName:     events.TopicRealtime,
		Payload:       body,
		CreatedAt:     time.Now().UTC(),
		SupplierID:    supplierID,
	}
	_, err := e.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(event)),
	})
	return err
}

func newLocationEventID(driverID string) string {
	return outbox.ClampEventID("loc_" + driverID + "_" + time.Now().UTC().Format("20060102150405.000000000"))
}

// injectRouteID adds route_id (and data.route_id) to the location envelope so
// route-keyed consumers (digital twin) can attribute the ping to a route.
func injectRouteID(payload []byte, routeID string) []byte {
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		return payload
	}
	object["route_id"] = routeID
	if data, ok := object["data"].(map[string]any); ok {
		data["route_id"] = routeID
	}
	out, err := json.Marshal(object)
	if err != nil {
		return payload
	}
	return out
}
