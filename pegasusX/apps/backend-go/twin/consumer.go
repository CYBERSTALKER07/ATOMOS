package twin

import (
	"context"
	"encoding/json"
	"log/slog"

	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	pegasuskafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	kafka "github.com/segmentio/kafka-go"
)

var (
	twinUpdateSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "twin_update_success_total",
			Help: "Total successful twin updates",
		},
		[]string{"event_type"},
	)
	twinUpdateLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "twin_update_latency_seconds",
			Help:    "Latency of twin updates in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"event_type"},
	)
)

type EventConsumer struct {
	service *Service
	log     *slog.Logger
}

func NewEventConsumer(service *Service, log *slog.Logger) *EventConsumer {
	return &EventConsumer{
		service: service,
		log:     log,
	}
}

func (c *EventConsumer) HandleEvent(ctx context.Context, msg kafka.Message) error {
	envelope, err := pegasuskafka.ParseEnvelope(msg.Value)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to parse event envelope", "err", err)
		return nil
	}

	start := time.Now()
	var handleErr error

	switch envelope.Type {
	case events.EventRouteCreated:
		var payload events.RouteEvent
		if err := json.Unmarshal(msg.Value, &payload); err == nil && payload.RouteID != "" {
			handleErr = c.service.HandleRouteStarted(ctx, payload.RouteID, payload.DriverID, int64(payload.OrderCount))
		}
	case events.EventDriverLocationUpdated:
		var payload events.OrderEvent
		if err := json.Unmarshal(msg.Value, &payload); err == nil && payload.RouteID != "" {
			handleErr = c.service.HandleLocationUpdate(ctx, payload.RouteID, payload.GPSLat, payload.GPSLng, payload.H3Cell)
		}
	case events.EventOrderStatusChanged:
		var payload events.OrderEvent
		if err := json.Unmarshal(msg.Value, &payload); err == nil && payload.RouteID != "" && payload.OrderID != "" {
			handleErr = c.service.HandleStopStatusChanged(ctx, payload.RouteID, payload.OrderID, payload.Status)
		}
	case "route.eta.updated":
		var payload struct {
			RouteID string `json:"routeId"`
			Stops   []struct {
				StopID           string     `json:"stopId"` // wait, in ETA service it was StopId in RouteETA, but json might be stopId
				PredictedArrival *time.Time `json:"predictedArrival"`
				WindowStart      *time.Time `json:"windowStart"`
				WindowEnd        *time.Time `json:"windowEnd"`
			} `json:"stops"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err == nil && payload.RouteID != "" {
			var stops []StopETAUpdate
			for _, s := range payload.Stops {
				stops = append(stops, StopETAUpdate{
					StopID:           s.StopID,
					PredictedArrival: s.PredictedArrival,
					WindowStart:      s.WindowStart,
					WindowEnd:        s.WindowEnd,
				})
			}
			handleErr = c.service.HandleETAUpdate(ctx, payload.RouteID, stops)
		}
	}

	if handleErr == nil && envelope.Type != "" {
		twinUpdateSuccess.WithLabelValues(envelope.Type).Inc()
		twinUpdateLatency.WithLabelValues(envelope.Type).Observe(time.Since(start).Seconds())
	}
	return handleErr
}
