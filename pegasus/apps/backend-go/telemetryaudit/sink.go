package telemetryaudit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"backend-go/fastjson"
	"backend-go/kafka/workerpool"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	goKafka "github.com/segmentio/kafka-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const consumerGroupID = "pegasus-telemetry-audit-consumer-group"

// StartSink boots the partition-parallel telemetry audit sink.
func StartSink(ctx context.Context, spannerClient *spanner.Client, brokerAddress string) {
	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    TopicRaw,
		GroupID:  consumerGroupID,
		MinBytes: 1,
		MaxBytes: 10 << 20,
	})

	pool, err := workerpool.New(workerpool.Config{
		Source: reader,
		Name:   "telemetry-audit-sink",
		Logger: slog.Default(),
		Handler: func(ctx context.Context, msg goKafka.Message) error {
			var event telemetry.AuditEvent
			if err := fastjson.Unmarshal(msg.Value, &event); err != nil {
				return fmt.Errorf("unmarshal telemetry audit event: %w", err)
			}
			return persistAuditEvent(ctx, spannerClient, event)
		},
	})
	if err != nil {
		slog.Error("telemetry audit sink init failed", "err", err)
		return
	}

	go func() {
		if err := pool.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("telemetry audit sink exited", "err", err)
		}
	}()
	slog.Info("telemetry audit sink ONLINE", "topic", TopicRaw, "group_id", consumerGroupID)
}

func persistAuditEvent(ctx context.Context, spannerClient *spanner.Client, event telemetry.AuditEvent) error {
	_, err := spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("DriverTelemetry",
			[]string{
				"TraceId",
				"DriverId",
				"SupplierId",
				"EventTime",
				"Payload",
				"Lat",
				"Lng",
				"Velocity",
				"Heading",
				"CreatedAt",
			},
			[]interface{}{
				event.TraceID,
				event.DriverID,
				event.SupplierID,
				event.EventTime,
				event.Payload,
				event.Latitude,
				event.Longitude,
				event.Velocity,
				event.Heading,
				spanner.CommitTimestamp,
			},
		),
	})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	if err != nil {
		return fmt.Errorf("insert driver telemetry trace %s: %w", event.TraceID, err)
	}
	return nil
}
